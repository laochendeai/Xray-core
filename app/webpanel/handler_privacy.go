package webpanel

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type PrivacyDiagnosticsContextResponse struct {
	Supported   bool                 `json:"supported"`
	Unsupported string               `json:"unsupportedReason,omitempty"`
	TunStatus   *TunStatus           `json:"tunStatus,omitempty"`
	TunSettings *TunEditableSettings `json:"tunSettings,omitempty"`
}

type PrivacyHardeningStatusResponse struct {
	Platform                   string                         `json:"platform"`
	BrowserPolicy              PrivacyBrowserPolicyStatus     `json:"browserPolicy"`
	ControlledBrowser          PrivacyControlledBrowserStatus `json:"controlledBrowser"`
	DailyBrowserFingerprint    PrivacyFingerprintHardening    `json:"dailyBrowserFingerprint"`
	CurrentPageCanHardenSystem bool                           `json:"currentPageCanHardenSystem"`
}

type PrivacyFingerprintHardening struct {
	CanHardenDailyBrowser       bool   `json:"canHardenDailyBrowser"`
	RequiresControlledBrowser   bool   `json:"requiresControlledBrowser"`
	Reason                      string `json:"reason"`
	ControlledBrowserActionName string `json:"controlledBrowserActionName"`
}

type PrivacyBrowserPolicyStatus struct {
	Supported             bool                               `json:"supported"`
	Installed             bool                               `json:"installed"`
	Configured            bool                               `json:"configured"`
	Installable           bool                               `json:"installable"`
	CanInstall            bool                               `json:"canInstall"`
	RestartRequired       bool                               `json:"restartRequired"`
	UnsupportedReason     string                             `json:"unsupportedReason,omitempty"`
	InstallUnavailable    string                             `json:"installUnavailable,omitempty"`
	PolicyFileName        string                             `json:"policyFileName"`
	InstallCommand        string                             `json:"installCommand"`
	RemoveCommand         string                             `json:"removeCommand"`
	Expected              map[string]string                  `json:"expected"`
	DetectedBrowsers      int                                `json:"detectedBrowsers"`
	ConfiguredBrowsers    int                                `json:"configuredBrowsers"`
	ConfiguredPolicyFiles int                                `json:"configuredPolicyFiles"`
	Targets               []PrivacyBrowserPolicyTargetStatus `json:"targets"`
}

type PrivacyBrowserPolicyTargetStatus struct {
	Browser    string                           `json:"browser"`
	Detected   bool                             `json:"detected"`
	Configured bool                             `json:"configured"`
	Paths      []PrivacyBrowserPolicyPathStatus `json:"paths"`
}

type PrivacyBrowserPolicyPathStatus struct {
	Path     string `json:"path"`
	Exists   bool   `json:"exists"`
	Matching bool   `json:"matching"`
	Error    string `json:"error,omitempty"`
}

type PrivacyControlledBrowserStatus struct {
	Supported              bool   `json:"supported"`
	Available              bool   `json:"available"`
	NodeAvailable          bool   `json:"nodeAvailable"`
	PlaywrightAvailable    bool   `json:"playwrightAvailable"`
	DisplayAvailable       bool   `json:"displayAvailable"`
	RequiresVisibleSession bool   `json:"requiresVisibleSession"`
	UnsupportedReason      string `json:"unsupportedReason,omitempty"`
	ScriptPath             string `json:"scriptPath,omitempty"`
	Command                string `json:"command"`
	OutputDir              string `json:"outputDir"`
	LogFile                string `json:"logFile"`
}

type PrivacyHardeningActionResponse struct {
	OK      bool                            `json:"ok"`
	Message string                          `json:"message"`
	Output  string                          `json:"output,omitempty"`
	PID     int                             `json:"pid,omitempty"`
	LogFile string                          `json:"logFile,omitempty"`
	Status  *PrivacyHardeningStatusResponse `json:"status,omitempty"`
}

type privacyBrowserPolicyTarget struct {
	Browser          string
	PolicyDirs       []string
	BinaryCandidates []string
}

const privacyBrowserPolicyFileName = "xray-privacy-policy.json"

var privacyBrowserPolicyExpected = map[string]string{
	"WebRtcIPHandling": "disable_non_proxied_udp",
	"DnsOverHttpsMode": "off",
}

var defaultPrivacyBrowserPolicyTargets = []privacyBrowserPolicyTarget{
	{
		Browser:          "Google Chrome",
		PolicyDirs:       []string{"/etc/opt/chrome/policies/managed"},
		BinaryCandidates: []string{"google-chrome", "google-chrome-stable"},
	},
	{
		Browser:          "Chromium",
		PolicyDirs:       []string{"/etc/chromium/policies/managed"},
		BinaryCandidates: []string{"chromium", "chromium-browser"},
	},
	{
		Browser:          "Microsoft Edge",
		PolicyDirs:       []string{"/etc/opt/edge/policies/managed"},
		BinaryCandidates: []string{"microsoft-edge", "microsoft-edge-stable"},
	},
	{
		Browser:          "Brave",
		PolicyDirs:       []string{"/etc/brave/policies/managed", "/etc/opt/brave.com/brave/policies/managed"},
		BinaryCandidates: []string{"brave-browser", "brave"},
	},
	{
		Browser:          "Vivaldi",
		PolicyDirs:       []string{"/etc/opt/vivaldi/policies/managed"},
		BinaryCandidates: []string{"vivaldi", "vivaldi-stable"},
	},
}

func (wp *WebPanel) handlePrivacyDiagnosticsContext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if wp.tunManager == nil {
		writeJSON(w, http.StatusOK, PrivacyDiagnosticsContextResponse{
			Supported:   false,
			Unsupported: "TUN manager is not configured",
		})
		return
	}

	if ok := wp.enforceTunAccess(w, r); !ok {
		return
	}

	settings, err := wp.tunManager.EditableSettings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load privacy diagnostics context: "+err.Error())
		return
	}

	status := wp.tunStatusSnapshot()
	writeJSON(w, http.StatusOK, PrivacyDiagnosticsContextResponse{
		Supported:   true,
		TunStatus:   status,
		TunSettings: settings,
	})
}

func (wp *WebPanel) handlePrivacyHardeningStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if ok := wp.enforcePrivacyLocalAccess(w, r); !ok {
		return
	}

	writeJSON(w, http.StatusOK, wp.privacyHardeningStatus())
}

func (wp *WebPanel) handlePrivacyInstallBrowserPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if ok := wp.enforcePrivacyLocalAccess(w, r); !ok {
		return
	}

	output, err := wp.runPrivacyBrowserPolicyInstaller()
	status := wp.privacyHardeningStatus()
	if err != nil {
		writeJSON(w, http.StatusConflict, PrivacyHardeningActionResponse{
			OK:      false,
			Message: err.Error(),
			Output:  output,
			Status:  &status,
		})
		return
	}

	writeJSON(w, http.StatusOK, PrivacyHardeningActionResponse{
		OK:      true,
		Message: "Browser privacy policy installed or repaired. Fully restart Chrome/Chromium-family browsers before verification.",
		Output:  output,
		Status:  &status,
	})
}

func (wp *WebPanel) handlePrivacyOpenControlledBrowser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if ok := wp.enforcePrivacyLocalAccess(w, r); !ok {
		return
	}

	pid, logFile, err := wp.startPrivacyControlledBrowser()
	status := wp.privacyHardeningStatus()
	if err != nil {
		writeJSON(w, http.StatusConflict, PrivacyHardeningActionResponse{
			OK:      false,
			Message: err.Error(),
			LogFile: logFile,
			Status:  &status,
		})
		return
	}

	writeJSON(w, http.StatusOK, PrivacyHardeningActionResponse{
		OK:      true,
		Message: "Controlled IPPure browser started with randomized fingerprint and WebRTC hardening.",
		PID:     pid,
		LogFile: logFile,
		Status:  &status,
	})
}

func (wp *WebPanel) enforcePrivacyLocalAccess(w http.ResponseWriter, r *http.Request) bool {
	if wp.tunManager != nil {
		return wp.enforceTunAccess(w, r)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(strings.TrimSpace(host))
	if ip != nil && ip.IsLoopback() {
		return true
	}

	writeError(w, http.StatusForbidden, "Privacy hardening is limited to local browser requests for safety")
	return false
}

func (wp *WebPanel) privacyHardeningStatus() PrivacyHardeningStatusResponse {
	return PrivacyHardeningStatusResponse{
		Platform:                   runtime.GOOS,
		BrowserPolicy:              wp.privacyBrowserPolicyStatus(),
		ControlledBrowser:          wp.privacyControlledBrowserStatus(),
		CurrentPageCanHardenSystem: false,
		DailyBrowserFingerprint: PrivacyFingerprintHardening{
			CanHardenDailyBrowser:       false,
			RequiresControlledBrowser:   true,
			Reason:                      "A normal web page can measure high-entropy browser surfaces but cannot globally rewrite the daily browser fingerprint for all sites.",
			ControlledBrowserActionName: "openControlledBrowser",
		},
	}
}

func (wp *WebPanel) privacyBrowserPolicyStatus() PrivacyBrowserPolicyStatus {
	status := inspectPrivacyBrowserPolicy(runtime.GOOS, privacyBrowserPolicyTargets(), privacyBrowserPolicyFileName)
	status.Expected = copyStringMap(privacyBrowserPolicyExpected)
	status.InstallCommand = "sudo ./scripts/install-browser-privacy-policy.sh"
	status.RemoveCommand = "sudo ./scripts/install-browser-privacy-policy.sh --remove"

	if !status.Supported {
		return status
	}

	scriptPath, scriptErr := wp.resolvePrivacyScriptPath("install-browser-privacy-policy.sh")
	status.Installable = scriptErr == nil
	if scriptErr != nil {
		status.InstallUnavailable = scriptErr.Error()
		return status
	}

	askpassScriptPath, _ := wp.resolvePrivacyScriptPath("webpanel-sudo-askpass.sh")
	status.CanInstall = os.Geteuid() == 0 || graphicalSudoAvailable(askpassScriptPath) || commandExists("pkexec")
	if !status.CanInstall {
		status.InstallUnavailable = "installing managed browser policies requires root, graphical sudo, or pkexec"
	}
	status.InstallCommand = fmt.Sprintf("sudo %s", shellQuote(scriptPath))
	return status
}

func inspectPrivacyBrowserPolicy(platform string, targets []privacyBrowserPolicyTarget, policyFileName string) PrivacyBrowserPolicyStatus {
	status := PrivacyBrowserPolicyStatus{
		Supported:      platform == "linux",
		PolicyFileName: policyFileName,
		Expected:       copyStringMap(privacyBrowserPolicyExpected),
		Targets:        []PrivacyBrowserPolicyTargetStatus{},
	}
	if !status.Supported {
		status.UnsupportedReason = "Managed Chromium browser policy automation is currently implemented for Linux only. Use the controlled browser action for fingerprint spoofing on other platforms."
		return status
	}

	for _, target := range targets {
		targetStatus := PrivacyBrowserPolicyTargetStatus{
			Browser:  target.Browser,
			Detected: browserPolicyTargetDetected(target),
			Paths:    []PrivacyBrowserPolicyPathStatus{},
		}

		for _, dir := range target.PolicyDirs {
			pathStatus := inspectPrivacyBrowserPolicyPath(filepath.Join(dir, policyFileName))
			if pathStatus.Matching {
				targetStatus.Configured = true
				status.ConfiguredPolicyFiles++
			}
			targetStatus.Paths = append(targetStatus.Paths, pathStatus)
		}

		if targetStatus.Detected {
			status.DetectedBrowsers++
			if targetStatus.Configured {
				status.ConfiguredBrowsers++
			}
		}
		if targetStatus.Configured {
			status.Configured = true
		}

		status.Targets = append(status.Targets, targetStatus)
	}

	if status.DetectedBrowsers > 0 {
		status.Installed = status.ConfiguredBrowsers == status.DetectedBrowsers
	} else {
		status.Installed = status.Configured
	}
	status.RestartRequired = status.Configured
	return status
}

func inspectPrivacyBrowserPolicyPath(path string) PrivacyBrowserPolicyPathStatus {
	status := PrivacyBrowserPolicyPathStatus{Path: path}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return status
		}
		status.Error = err.Error()
		return status
	}

	status.Exists = true
	var policy map[string]interface{}
	if err := json.Unmarshal(raw, &policy); err != nil {
		status.Error = "invalid policy JSON: " + err.Error()
		return status
	}

	for key, expected := range privacyBrowserPolicyExpected {
		if strings.TrimSpace(fmt.Sprint(policy[key])) != expected {
			return status
		}
	}
	status.Matching = true
	return status
}

func browserPolicyTargetDetected(target privacyBrowserPolicyTarget) bool {
	for _, candidate := range target.BinaryCandidates {
		if commandExists(candidate) {
			return true
		}
	}
	for _, dir := range target.PolicyDirs {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return true
		}
	}
	return false
}

func privacyBrowserPolicyTargets() []privacyBrowserPolicyTarget {
	override := strings.TrimSpace(os.Getenv("XRAY_BROWSER_POLICY_DIRS"))
	if override == "" {
		return defaultPrivacyBrowserPolicyTargets
	}

	parts := strings.FieldsFunc(override, func(r rune) bool {
		return r == ':' || r == ',' || r == '\n'
	})
	targets := make([]privacyBrowserPolicyTarget, 0, len(parts))
	for index, part := range parts {
		dir := strings.TrimSpace(part)
		if dir == "" {
			continue
		}
		targets = append(targets, privacyBrowserPolicyTarget{
			Browser:    fmt.Sprintf("Policy override %d", index+1),
			PolicyDirs: []string{dir},
		})
	}
	if len(targets) == 0 {
		return defaultPrivacyBrowserPolicyTargets
	}
	return targets
}

func (wp *WebPanel) runPrivacyBrowserPolicyInstaller() (string, error) {
	if runtime.GOOS != "linux" {
		return "", fmt.Errorf("browser managed policy installer is currently supported on Linux only")
	}

	scriptPath, err := wp.resolvePrivacyScriptPath("install-browser-privacy-policy.sh")
	if err != nil {
		return "", fmt.Errorf("browser policy installer script is missing: %w", err)
	}

	cmdArgs := []string{scriptPath, "--install"}
	var cmd *exec.Cmd
	switch {
	case os.Geteuid() == 0:
		cmd = execCommandCompat(scriptPath, "--install")
	case graphicalSudoAvailable(mustPrivacyScriptPath(wp, "webpanel-sudo-askpass.sh")):
		askpassScriptPath := mustPrivacyScriptPath(wp, "webpanel-sudo-askpass.sh")
		cmd = execCommandCompat("sudo", append([]string{"-A"}, cmdArgs...)...)
		cmd.Env = append(os.Environ(),
			"SUDO_ASKPASS="+askpassScriptPath,
			"SUDO_ASKPASS_PROMPT=WebPanel needs your password to install or repair browser privacy policies.",
		)
	default:
		if !commandExists("pkexec") {
			return "", fmt.Errorf("installing managed browser policies requires root, graphical sudo, or pkexec")
		}
		cmd = execCommandCompat("pkexec", cmdArgs...)
		detachFromControllingTTY(cmd)
	}

	if cmd.Env == nil {
		cmd.Env = os.Environ()
	}
	cmd.Dir = repoRootFromScript(scriptPath)
	output, err := cmd.CombinedOutput()
	trimmed := strings.TrimSpace(string(output))
	if err != nil {
		if trimmed == "" {
			return "", fmt.Errorf("run browser policy installer: %w", err)
		}
		return trimmed, fmt.Errorf("run browser policy installer: %w", err)
	}
	return trimmed, nil
}

func (wp *WebPanel) privacyControlledBrowserStatus() PrivacyControlledBrowserStatus {
	status := PrivacyControlledBrowserStatus{
		RequiresVisibleSession: true,
		Command:                "IPPURE_HEADLESS=0 IPPURE_KEEP_OPEN=1 node scripts/verify-ippure.mjs",
		OutputDir:              filepath.Join("runtime", "ippure-webpanel"),
		LogFile:                filepath.Join("runtime", "privacy-hardening", "controlled-browser.log"),
	}

	scriptPath, err := wp.resolvePrivacyScriptPath("verify-ippure.mjs")
	if err != nil {
		status.UnsupportedReason = "IPPure verification script is missing: " + err.Error()
		return status
	}
	status.ScriptPath = scriptPath

	nodePath, nodeOK := resolveCommandPath("node")
	status.NodeAvailable = nodeOK
	if !nodeOK {
		status.UnsupportedReason = "node is not available; install Node.js to launch the controlled browser from WebPanel"
		return status
	}

	repoRoot := repoRootFromScript(scriptPath)
	status.PlaywrightAvailable = playwrightImportAvailable(repoRoot, nodePath)
	if !status.PlaywrightAvailable {
		status.UnsupportedReason = "the Node.js playwright package is not available from the repository root"
		return status
	}

	status.DisplayAvailable = visibleDesktopSessionAvailable()
	if !status.DisplayAvailable {
		status.UnsupportedReason = "a visible desktop session is required to open the controlled browser; run the command manually in a desktop terminal or set IPPURE_HEADLESS=1"
		return status
	}

	status.Supported = true
	status.Available = true
	return status
}

func (wp *WebPanel) startPrivacyControlledBrowser() (int, string, error) {
	status := wp.privacyControlledBrowserStatus()
	if !status.Available {
		return 0, status.LogFile, fmt.Errorf("%s", status.UnsupportedReason)
	}

	nodePath, _ := resolveCommandPath("node")
	scriptPath := status.ScriptPath
	repoRoot := repoRootFromScript(scriptPath)
	outputDir := filepath.Join(repoRoot, "runtime", "ippure-webpanel")
	logDir := filepath.Join(repoRoot, "runtime", "privacy-hardening")
	logPath := filepath.Join(logDir, "controlled-browser.log")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return 0, logPath, fmt.Errorf("create IPPure output directory: %w", err)
	}
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return 0, logPath, fmt.Errorf("create privacy hardening log directory: %w", err)
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return 0, logPath, fmt.Errorf("open controlled browser log: %w", err)
	}
	defer logFile.Close()

	cmd := execCommandCompat(nodePath, scriptPath)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(),
		"IPPURE_HEADLESS=0",
		"IPPURE_KEEP_OPEN=1",
		"IPPURE_OUTPUT_DIR="+outputDir,
	)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	detachFromControllingTTY(cmd)

	if err := cmd.Start(); err != nil {
		return 0, logPath, fmt.Errorf("start controlled IPPure browser: %w", err)
	}
	pid := cmd.Process.Pid
	if err := cmd.Process.Release(); err != nil {
		return pid, logPath, fmt.Errorf("release controlled browser process: %w", err)
	}
	return pid, logPath, nil
}

func (wp *WebPanel) resolvePrivacyScriptPath(name string) (string, error) {
	if wp.tunManager != nil {
		if path, err := wp.tunManager.resolveRepoScriptPath(name); err == nil {
			return path, nil
		}
	}

	candidates := []string{}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "scripts", name))
	}
	if wp.config != nil && strings.TrimSpace(wp.config.ConfigPath) != "" {
		candidates = append(candidates, filepath.Join(filepath.Dir(wp.config.ConfigPath), "scripts", name))
	}
	if executable, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(executable), "scripts", name))
	}

	seen := map[string]struct{}{}
	checked := []string{}
	for _, candidate := range candidates {
		candidate = filepath.Clean(candidate)
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		checked = append(checked, candidate)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("%s not found in %s", name, strings.Join(checked, ", "))
}

func mustPrivacyScriptPath(wp *WebPanel, name string) string {
	path, _ := wp.resolvePrivacyScriptPath(name)
	return path
}

func repoRootFromScript(scriptPath string) string {
	return filepath.Dir(filepath.Dir(scriptPath))
}

func commandExists(name string) bool {
	_, ok := resolveCommandPath(name)
	return ok
}

func visibleDesktopSessionAvailable() bool {
	switch runtime.GOOS {
	case "linux":
		return strings.TrimSpace(os.Getenv("DISPLAY")) != "" || strings.TrimSpace(os.Getenv("WAYLAND_DISPLAY")) != ""
	case "darwin", "windows":
		return true
	default:
		return false
	}
}

func playwrightImportAvailable(repoRoot string, nodePath string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmdName, cmdArgs := prepareCommandInvocation(
		nodePath,
		[]string{
			"--input-type=module",
			"-e",
			"import('playwright').then(() => process.exit(0)).catch(() => process.exit(1))",
		},
	)
	cmd := exec.CommandContext(ctx, cmdName, cmdArgs...)
	cmd.Dir = repoRoot
	return cmd.Run() == nil
}

func copyStringMap(input map[string]string) map[string]string {
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

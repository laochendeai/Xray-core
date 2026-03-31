package webpanel

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	stdnet "net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	defaultTunSelectionPolicy = TunSelectionPolicyFastest
	defaultTunRouteMode       = TunRouteModeStrictProxy
	defaultTunCIDRs           = []string{
		"127.0.0.0/8",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16",
		"100.64.0.0/10",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	}
	defaultTunDomains = []string{
		"full:localhost",
	}
	defaultTunDNS = []string{
		"1.1.1.1",
		"8.8.8.8",
	}
	defaultTunChinaDNS = []string{
		"223.5.5.5",
		"119.29.29.29",
	}
)

type TunSelectionPolicy string
type TunRouteMode string

const (
	TunSelectionPolicyFastest        TunSelectionPolicy = "fastest"
	TunSelectionPolicyLowestLatency  TunSelectionPolicy = "lowest_latency"
	TunSelectionPolicyLowestFailRate TunSelectionPolicy = "lowest_fail_rate"
)

const (
	TunRouteModeStrictProxy TunRouteMode = "strict_proxy"
	TunRouteModeAutoTested  TunRouteMode = "auto_tested"
)

const (
	tunLowestFailRateSubsetSize = 3
	tunDirectProbeTimeout       = 1200 * time.Millisecond
	tunDirectProbeCacheTTL      = 6 * time.Hour
	tunDirectProbeConcurrency   = 12
	tunDirectProbeCacheVersion  = 1
	tunDirectProbeCacheFileName = "route-probe-cache.json"
)

type TunManager struct {
	configPath string
	xrayBin    string
	mu         sync.Mutex
}

type TunFeatureSettings struct {
	BinaryPath        string   `json:"binaryPath"`
	HelperPath        string   `json:"helperPath"`
	StateDir          string   `json:"stateDir"`
	RuntimeConfigPath string   `json:"runtimeConfigPath"`
	InterfaceName     string   `json:"interfaceName"`
	MTU               uint32   `json:"mtu"`
	RemoteDNS         []string `json:"remoteDns"`
	UseSudo           *bool    `json:"useSudo"`
	AllowRemote       bool     `json:"allowRemote"`
	SelectionPolicy   string   `json:"selectionPolicy"`
	RouteMode         string   `json:"routeMode"`
	ProtectCIDRs      []string `json:"protectCidrs"`
	ProtectDomains    []string `json:"protectDomains"`
}

type TunEditableSettings struct {
	SelectionPolicy string   `json:"selectionPolicy"`
	RouteMode       string   `json:"routeMode"`
	ProtectCIDRs    []string `json:"protectCidrs"`
	ProtectDomains  []string `json:"protectDomains"`
}

type tunDirectProbeCache struct {
	Version int                                 `json:"version"`
	Entries map[string]tunDirectProbeCacheEntry `json:"entries"`
}

type tunDirectProbeCacheEntry struct {
	Decision  bool      `json:"decision"`
	CheckedAt time.Time `json:"checkedAt"`
}

type tunDirectProbeRequest struct {
	Key   string
	Host  string
	Ports []int
}

type tunConfigEnvelope struct {
	WebPanel struct {
		Tun *TunFeatureSettings `json:"tun"`
	} `json:"webpanel"`
}

type TunStatus struct {
	Status                      string             `json:"status"`
	Running                     bool               `json:"running"`
	Available                   bool               `json:"available"`
	AllowRemote                 bool               `json:"allowRemote"`
	UseSudo                     bool               `json:"useSudo"`
	HelperExists                bool               `json:"helperExists"`
	ElevationReady              bool               `json:"elevationReady"`
	HelperCurrent               bool               `json:"helperCurrent"`
	BinaryCurrent               bool               `json:"binaryCurrent"`
	PrivilegeInstallRecommended bool               `json:"privilegeInstallRecommended"`
	BinaryPath                  string             `json:"binaryPath"`
	HelperPath                  string             `json:"helperPath"`
	StateDir                    string             `json:"stateDir"`
	RuntimeConfigPath           string             `json:"runtimeConfigPath"`
	InterfaceName               string             `json:"interfaceName"`
	MTU                         uint32             `json:"mtu"`
	RemoteDNS                   []string           `json:"remoteDns"`
	ConfigPath                  string             `json:"configPath"`
	XrayBinary                  string             `json:"xrayBinary"`
	Message                     string             `json:"message"`
	LastOutput                  string             `json:"lastOutput,omitempty"`
	Diagnostics                 []string           `json:"diagnostics,omitempty"`
	MachineState                MachineState       `json:"machineState,omitempty"`
	LastStateReason             MachineStateReason `json:"lastStateReason,omitempty"`
	LastStateChangedAt          *time.Time         `json:"lastStateChangedAt,omitempty"`
	RecentMachineEvents         []MachineEvent     `json:"recentMachineEvents,omitempty"`
}

func NewTunManager(configPath string) (*TunManager, error) {
	xrayBin, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve current xray binary: %w", err)
	}

	return &TunManager{
		configPath: configPath,
		xrayBin:    xrayBin,
	}, nil
}

func (m *TunManager) Status() *TunStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	settings, err := m.loadSettings()
	if err != nil {
		return &TunStatus{
			Status:     "unavailable",
			Available:  false,
			ConfigPath: m.configPath,
			XrayBinary: m.xrayBin,
			Message:    err.Error(),
		}
	}

	return m.inspectLocked(settings)
}

func (m *TunManager) Start(activeNodes []NodeRecord) *TunStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	settings, err := m.loadSettings()
	if err != nil {
		return &TunStatus{
			Status:     "unavailable",
			Available:  false,
			ConfigPath: m.configPath,
			XrayBinary: m.xrayBin,
			Message:    err.Error(),
		}
	}

	preflight := m.inspectLocked(settings)
	if preflight.PrivilegeInstallRecommended {
		preflight.Status = "blocked"
		preflight.Message = "Install or repair the privilege helper before enabling transparent TUN mode"
		return preflight
	}

	if err := m.generateRuntimeConfigLocked(settings, activeNodes); err != nil {
		preflight.Status = "error"
		preflight.Available = false
		preflight.Message = "Failed to prepare TUN runtime config"
		preflight.LastOutput = err.Error()
		preflight.Diagnostics = append(preflight.Diagnostics, "Regenerate the runtime config after fixing the base config.")
		return preflight
	}

	output, execErr := m.runHelperLocked(settings, "start", true)
	status := m.inspectLocked(settings)
	status.LastOutput = output
	if execErr != nil {
		status.Status = "error"
		status.Message = "Failed to enable transparent TUN mode"
		status.Diagnostics = append(status.Diagnostics, execErr.Error())
		return status
	}

	status.Message = "Transparent TUN mode is enabled"
	return status
}

func (m *TunManager) Stop() *TunStatus {
	return m.RestoreClean()
}

func (m *TunManager) RestoreClean() *TunStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	settings, err := m.loadSettings()
	if err != nil {
		return &TunStatus{
			Status:     "unavailable",
			Available:  false,
			ConfigPath: m.configPath,
			XrayBinary: m.xrayBin,
			Message:    err.Error(),
		}
	}

	output, execErr := m.runHelperLocked(settings, "stop", true)
	status := m.inspectLocked(settings)
	status.LastOutput = output
	if execErr != nil {
		status.Status = "error"
		status.Message = "Failed to restore a clean network state"
		status.Diagnostics = append(status.Diagnostics, execErr.Error())
		return status
	}

	status.Message = "Transparent TUN mode is disabled"
	return status
}

func (m *TunManager) Toggle(activeNodes []NodeRecord) *TunStatus {
	current := m.Status()
	if current.Running {
		return m.RestoreClean()
	}
	return m.Start(activeNodes)
}

func (m *TunManager) InstallPrivilege() *TunStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	settings, err := m.loadSettings()
	if err != nil {
		return &TunStatus{
			Status:     "unavailable",
			Available:  false,
			ConfigPath: m.configPath,
			XrayBinary: m.xrayBin,
			Message:    err.Error(),
		}
	}

	output, installErr := m.installPrivilegeLocked(settings)
	reloadedSettings, reloadErr := m.loadSettings()

	var status *TunStatus
	switch {
	case reloadErr == nil:
		status = m.inspectLocked(reloadedSettings)
	case settings != nil:
		status = m.inspectLocked(settings)
	default:
		status = &TunStatus{
			Status:     "unavailable",
			Available:  false,
			ConfigPath: m.configPath,
			XrayBinary: m.xrayBin,
			Message:    reloadErr.Error(),
		}
	}

	status.LastOutput = output
	if installErr != nil {
		status.Status = "error"
		status.Message = "Failed to install privilege helper"
		status.Diagnostics = append(status.Diagnostics, installErr.Error())
		if reloadErr != nil {
			status.Diagnostics = append(status.Diagnostics, "Reload updated TUN settings after fixing the installer failure.")
		}
		return status
	}

	if reloadErr != nil {
		status.Status = "error"
		status.Message = "Privilege helper install finished, but the updated TUN settings could not be reloaded"
		status.Diagnostics = append(status.Diagnostics, reloadErr.Error())
		return status
	}

	if !status.HelperExists || (status.UseSudo && !status.ElevationReady) || status.PrivilegeInstallRecommended {
		status.Status = "error"
		status.Message = "Privilege helper install finished, but readiness verification failed"
		status.Diagnostics = append(status.Diagnostics, "Verify the installed helper path and sudo -n readiness on this machine.")
		return status
	}

	status.Message = "Privilege helper is installed"
	return status
}

func (m *TunManager) IsRequestAllowed(remoteAddr string) (bool, *TunFeatureSettings, error) {
	settings, err := m.loadSettings()
	if err != nil {
		return false, nil, err
	}
	if settings.AllowRemote {
		return true, settings, nil
	}

	host, _, err := stdnet.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := stdnet.ParseIP(strings.TrimSpace(host))
	if ip == nil {
		return false, settings, fmt.Errorf("unable to parse remote address %q", remoteAddr)
	}

	return ip.IsLoopback(), settings, nil
}

func (m *TunManager) installPrivilegeLocked(settings *TunFeatureSettings) (string, error) {
	if settings == nil {
		return "", fmt.Errorf("tun settings are not configured")
	}

	if _, err := exec.LookPath("pkexec"); err != nil {
		return "", fmt.Errorf("pkexec is not available: %w", err)
	}

	configPath, err := filepath.Abs(m.configPath)
	if err != nil {
		return "", fmt.Errorf("resolve config path: %w", err)
	}
	xraySourcePath, err := filepath.Abs(m.xrayBin)
	if err != nil {
		return "", fmt.Errorf("resolve current xray binary: %w", err)
	}

	installScriptPath := filepath.Join(filepath.Dir(configPath), "scripts", "install-webpanel-tun-sudoers.sh")
	if _, err := os.Stat(installScriptPath); err != nil {
		return "", fmt.Errorf("install script is missing: %w", err)
	}
	askpassScriptPath := filepath.Join(filepath.Dir(configPath), "scripts", "webpanel-sudo-askpass.sh")

	targetUser := strings.TrimSpace(currentUserName())
	if targetUser == "" {
		return "", fmt.Errorf("unable to determine the current non-root user for sudoers installation")
	}

	installArgs := []string{
		installScriptPath,
		"--config", configPath,
		"--user", targetUser,
		"--xray-src", xraySourcePath,
	}

	if graphicalSudoAvailable(askpassScriptPath) {
		cmd := exec.Command("sudo", append([]string{"-A"}, installArgs...)...)
		cmd.Env = append(os.Environ(),
			"SUDO_ASKPASS="+askpassScriptPath,
			"SUDO_ASKPASS_PROMPT=WebPanel needs your password to install or repair the transparent proxy helper.",
		)
		output, err := cmd.CombinedOutput()
		trimmed := strings.TrimSpace(string(output))
		if err != nil {
			if trimmed == "" {
				return "", fmt.Errorf("run privilege installer with graphical sudo: %w", err)
			}
			return trimmed, fmt.Errorf("run privilege installer with graphical sudo: %w", err)
		}

		return trimmed, nil
	}

	if _, err := exec.LookPath("pkexec"); err != nil {
		return "", fmt.Errorf("pkexec is not available: %w", err)
	}

	cmd := exec.Command("pkexec", installArgs...)
	// Detach from the launch terminal so pkexec uses the desktop polkit agent
	// instead of prompting on the hidden controlling TTY of the web panel process.
	detachFromControllingTTY(cmd)
	output, err := cmd.CombinedOutput()
	trimmed := strings.TrimSpace(string(output))
	if err != nil {
		if trimmed == "" {
			return "", fmt.Errorf("run privilege installer: %w", err)
		}
		return trimmed, fmt.Errorf("run privilege installer: %w", err)
	}

	return trimmed, nil
}

func graphicalSudoAvailable(askpassScriptPath string) bool {
	if strings.TrimSpace(os.Getenv("DISPLAY")) == "" {
		return false
	}
	if _, err := exec.LookPath("sudo"); err != nil {
		return false
	}
	if _, err := os.Stat(askpassScriptPath); err != nil {
		return false
	}
	for _, candidate := range []string{"zenity", "kdialog"} {
		if _, err := exec.LookPath(candidate); err == nil {
			return true
		}
	}
	return false
}

func currentUserName() string {
	if u, err := user.Current(); err == nil {
		if username := strings.TrimSpace(u.Username); username != "" {
			return username
		}
	}

	for _, key := range []string{"SUDO_USER", "USER", "LOGNAME"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}

	return ""
}

func (m *TunManager) loadSettings() (*TunFeatureSettings, error) {
	if m.configPath == "" {
		return nil, fmt.Errorf("config path is not configured for the web panel")
	}

	raw, err := os.ReadFile(m.configPath)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var envelope tunConfigEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	settings := &TunFeatureSettings{}
	if envelope.WebPanel.Tun != nil {
		*settings = *envelope.WebPanel.Tun
	}

	baseDir := filepath.Dir(m.configPath)
	if settings.BinaryPath == "" {
		settings.BinaryPath = m.xrayBin
	}
	settings.BinaryPath = resolvePath(baseDir, settings.BinaryPath)

	if settings.HelperPath == "" {
		settings.HelperPath = filepath.Join(baseDir, "scripts", "webpanel-tun-helper.sh")
	}
	settings.HelperPath = resolvePath(baseDir, settings.HelperPath)

	if settings.StateDir == "" {
		settings.StateDir = filepath.Join(baseDir, "runtime", "tun")
	}
	settings.StateDir = resolvePath(baseDir, settings.StateDir)

	if settings.RuntimeConfigPath == "" {
		settings.RuntimeConfigPath = filepath.Join(settings.StateDir, "config.json")
	}
	settings.RuntimeConfigPath = resolvePath(baseDir, settings.RuntimeConfigPath)

	if settings.InterfaceName == "" {
		settings.InterfaceName = "xray0"
	}
	if settings.MTU == 0 {
		settings.MTU = 1500
	}
	if len(settings.RemoteDNS) == 0 {
		settings.RemoteDNS = append([]string{}, defaultTunDNS...)
	} else {
		settings.RemoteDNS = uniqStrings(settings.RemoteDNS)
	}

	useSudo := os.Geteuid() != 0
	if settings.UseSudo != nil {
		useSudo = *settings.UseSudo
	}
	settings.UseSudo = &useSudo
	settings.SelectionPolicy = string(normalizeTunSelectionPolicy(settings.SelectionPolicy))
	settings.RouteMode = string(normalizeTunRouteMode(settings.RouteMode))

	settings.ProtectCIDRs = uniqStrings(append(append([]string{}, defaultTunCIDRs...), settings.ProtectCIDRs...))
	settings.ProtectDomains = normalizeTunDomainRules(append(append([]string{}, defaultTunDomains...), settings.ProtectDomains...))

	return settings, nil
}

func (m *TunManager) EditableSettings() (*TunEditableSettings, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.loadEditableSettingsLocked()
}

func (m *TunManager) UpdateEditableSettings(next TunEditableSettings) (*TunEditableSettings, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	config, err := m.readConfigMapLocked()
	if err != nil {
		return nil, err
	}

	webpanel, _ := config["webpanel"].(map[string]interface{})
	if webpanel == nil {
		webpanel = map[string]interface{}{}
	}

	tun, _ := webpanel["tun"].(map[string]interface{})
	if tun == nil {
		tun = map[string]interface{}{}
	}

	selectionPolicy := string(normalizeTunSelectionPolicy(next.SelectionPolicy))
	tun["selectionPolicy"] = selectionPolicy

	routeMode := string(normalizeTunRouteMode(next.RouteMode))
	tun["routeMode"] = routeMode

	protectDomains := normalizeTunDomainRules(next.ProtectDomains)
	if len(protectDomains) > 0 {
		tun["protectDomains"] = protectDomains
	} else {
		delete(tun, "protectDomains")
	}

	protectCIDRs := uniqStrings(next.ProtectCIDRs)
	if len(protectCIDRs) > 0 {
		tun["protectCidrs"] = protectCIDRs
	} else {
		delete(tun, "protectCidrs")
	}

	webpanel["tun"] = tun
	config["webpanel"] = webpanel

	encoded, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("encode updated TUN settings: %w", err)
	}

	if err := NewConfigFileManager(m.configPath).WriteConfig(encoded); err != nil {
		return nil, err
	}

	return m.loadEditableSettingsLocked()
}

func (m *TunManager) loadEditableSettingsLocked() (*TunEditableSettings, error) {
	config, err := m.readConfigMapLocked()
	if err != nil {
		return nil, err
	}

	webpanel, _ := config["webpanel"].(map[string]interface{})
	tun, _ := webpanel["tun"].(map[string]interface{})

	settings := &TunEditableSettings{
		SelectionPolicy: string(defaultTunSelectionPolicy),
		RouteMode:       string(defaultTunRouteMode),
	}
	if tun == nil {
		return settings, nil
	}

	if value, ok := tun["selectionPolicy"].(string); ok {
		settings.SelectionPolicy = string(normalizeTunSelectionPolicy(value))
	}
	if value, ok := tun["routeMode"].(string); ok {
		settings.RouteMode = string(normalizeTunRouteMode(value))
	}
	settings.ProtectDomains = normalizeTunDomainRules(stringSliceFromAny(tun["protectDomains"]))
	settings.ProtectCIDRs = uniqStrings(stringSliceFromAny(tun["protectCidrs"]))

	return settings, nil
}

func (m *TunManager) readConfigMapLocked() (map[string]interface{}, error) {
	if m.configPath == "" {
		return nil, fmt.Errorf("config path is not configured for the web panel")
	}

	raw, err := os.ReadFile(m.configPath)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	return config, nil
}

func (m *TunManager) inspectLocked(settings *TunFeatureSettings) *TunStatus {
	status := &TunStatus{
		Status:            "stopped",
		Available:         true,
		AllowRemote:       settings.AllowRemote,
		UseSudo:           settings.UseSudo != nil && *settings.UseSudo,
		HelperCurrent:     true,
		BinaryCurrent:     true,
		BinaryPath:        settings.BinaryPath,
		HelperPath:        settings.HelperPath,
		StateDir:          settings.StateDir,
		RuntimeConfigPath: settings.RuntimeConfigPath,
		InterfaceName:     settings.InterfaceName,
		MTU:               settings.MTU,
		RemoteDNS:         append([]string{}, settings.RemoteDNS...),
		ConfigPath:        m.configPath,
		XrayBinary:        settings.BinaryPath,
	}
	if normalizeTunRouteMode(settings.RouteMode) == TunRouteModeAutoTested {
		status.Diagnostics = append(status.Diagnostics, "Auto-tested split routing will probe base direct rules before enabling transparent mode; the first start or stale cache refresh can take longer.")
	}

	if _, err := os.Stat(settings.HelperPath); err == nil {
		status.HelperExists = true
	} else {
		status.Available = false
		status.Status = "unavailable"
		status.Message = "TUN helper script is missing"
		status.PrivilegeInstallRecommended = true
		status.Diagnostics = append(status.Diagnostics, "Expected helper: "+settings.HelperPath)
		return status
	}

	if settings.UseSudo != nil && *settings.UseSudo && os.Geteuid() != 0 {
		status.ElevationReady = checkSudoReady(settings)
		if !status.ElevationReady {
			status.PrivilegeInstallRecommended = true
			status.Diagnostics = append(status.Diagnostics, "Configure passwordless sudo for the helper or run the panel as root.")
		}
	} else {
		status.ElevationReady = true
	}

	repoHelperPath := filepath.Join(filepath.Dir(m.configPath), "scripts", "webpanel-tun-helper.sh")
	helperCurrent, helperCompareErr := filesMatch(settings.HelperPath, repoHelperPath)
	if helperCompareErr == nil {
		status.HelperCurrent = helperCurrent
	} else if !os.IsNotExist(helperCompareErr) {
		status.Diagnostics = append(status.Diagnostics, "Unable to compare the installed helper with the repo helper: "+helperCompareErr.Error())
	}
	if !status.HelperCurrent {
		status.PrivilegeInstallRecommended = true
		status.Diagnostics = append(status.Diagnostics, "The installed helper is older than the repo version. Repair it before enabling transparent mode so clean restore can recover DNS correctly.")
	}

	binaryCurrent, binaryCompareErr := filesMatch(settings.BinaryPath, m.xrayBin)
	if binaryCompareErr == nil {
		status.BinaryCurrent = binaryCurrent
	} else if !os.IsNotExist(binaryCompareErr) {
		status.Diagnostics = append(status.Diagnostics, "Unable to compare the installed TUN binary with the current WebPanel binary: "+binaryCompareErr.Error())
	}
	if !status.BinaryCurrent {
		status.PrivilegeInstallRecommended = true
		status.Diagnostics = append(status.Diagnostics, "The installed TUN xray binary is older than the current WebPanel binary. Repair it before enabling transparent mode.")
	}

	output, err := m.runHelperLocked(settings, "status", false)
	status.LastOutput = output
	if err != nil {
		status.Available = false
		status.Status = "error"
		status.Message = "Failed to query transparent mode status"
		status.Diagnostics = append(status.Diagnostics, err.Error())
		return status
	}

	if strings.Contains(output, "ACTION=status:running") {
		status.Running = true
		status.Status = "running"
		status.Message = "Transparent TUN mode is enabled"
		return status
	}

	status.Message = "Transparent TUN mode is disabled"
	return status
}

func (m *TunManager) generateRuntimeConfigLocked(settings *TunFeatureSettings, activeNodes []NodeRecord) error {
	if err := os.MkdirAll(settings.StateDir, 0755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	raw, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("read base config: %w", err)
	}

	runtimeConfig, err := buildTunRuntimeConfig(raw, settings, activeNodes)
	if err != nil {
		return err
	}

	if err := os.WriteFile(settings.RuntimeConfigPath, runtimeConfig, 0644); err != nil {
		return fmt.Errorf("write runtime config: %w", err)
	}

	return nil
}

func (m *TunManager) runHelperLocked(settings *TunFeatureSettings, action string, allowElevation bool) (string, error) {
	if settings.HelperPath == "" {
		return "", fmt.Errorf("tun helper path is empty")
	}

	helperArgs := []string{
		action,
		settings.BinaryPath,
		settings.RuntimeConfigPath,
		settings.StateDir,
		settings.InterfaceName,
	}
	helperArgs = append(helperArgs, settings.RemoteDNS...)

	cmdName := settings.HelperPath
	cmdArgs := helperArgs
	if allowElevation && settings.UseSudo != nil && *settings.UseSudo && os.Geteuid() != 0 {
		cmdName = "sudo"
		cmdArgs = append([]string{"-n", settings.HelperPath}, helperArgs...)
	}

	cmd := exec.Command(cmdName, cmdArgs...)
	cmd.Env = append(os.Environ(),
		"XRAY_BIN="+settings.BinaryPath,
		"XRAY_CONFIG="+settings.RuntimeConfigPath,
		"STATE_DIR="+settings.StateDir,
		"TUN_NAME="+settings.InterfaceName,
		"REMOTE_DNS="+strings.Join(settings.RemoteDNS, " "),
	)
	output, err := cmd.CombinedOutput()
	trimmed := strings.TrimSpace(string(output))
	if err != nil {
		if trimmed == "" {
			return "", fmt.Errorf("run helper %q: %w", action, err)
		}
		return trimmed, fmt.Errorf("run helper %q: %w", action, err)
	}
	return trimmed, nil
}

func buildTunRuntimeConfig(raw []byte, settings *TunFeatureSettings, activeNodes []NodeRecord) ([]byte, error) {
	return buildTunRuntimeConfigWithDirectProbeResults(raw, settings, activeNodes, nil)
}

func buildTunRuntimeConfigWithDirectProbeResults(raw []byte, settings *TunFeatureSettings, activeNodes []NodeRecord, directProbeResults map[string]bool) ([]byte, error) {
	if len(activeNodes) == 0 {
		return nil, fmt.Errorf("no active nodes available for transparent mode")
	}

	var config map[string]interface{}
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, fmt.Errorf("parse base config: %w", err)
	}

	outbounds, ok := config["outbounds"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("base config has no outbounds")
	}

	poolOutbounds, err := buildActivePoolOutbounds(activeNodes)
	if err != nil {
		return nil, err
	}
	balancerConfig := buildTunBalancerConfig(settings, activeNodes)

	delete(config, "api")
	delete(config, "webpanel")

	config["inbounds"] = []interface{}{
		map[string]interface{}{
			"tag":      "tun-in",
			"port":     0,
			"protocol": "tun",
			"settings": map[string]interface{}{
				"name": settings.InterfaceName,
				"MTU":  settings.MTU,
			},
			"sniffing": map[string]interface{}{
				"enabled":      true,
				"destOverride": []string{"http", "tls", "quic"},
			},
		},
	}

	dnsConfig := buildTunDNSConfig(settings)
	if len(dnsConfig) > 0 {
		config["dns"] = dnsConfig
	}

	routing, _ := config["routing"].(map[string]interface{})
	if routing == nil {
		routing = map[string]interface{}{}
	}
	existingRules, _ := routing["rules"].([]interface{})
	resolvedProbeResults := resolveTunDirectProbeResults(settings, existingRules, directProbeResults)
	priorityRules, _ := splitTunRoutingRules(existingRules)
	priorityRules = filterTunPriorityRules(priorityRules, settings, resolvedProbeResults)
	prependRules := make([]interface{}, 0, 6)
	prependRules = append(prependRules, map[string]interface{}{
		"type":        "field",
		"inboundTag":  []string{"tun-in"},
		"port":        "53",
		"outboundTag": "dns-out",
	})
	if len(settings.ProtectDomains) > 0 {
		prependRules = append(prependRules, map[string]interface{}{
			"type":        "field",
			"domain":      settings.ProtectDomains,
			"outboundTag": "direct",
		})
	}
	if len(settings.ProtectCIDRs) > 0 {
		prependRules = append(prependRules, map[string]interface{}{
			"type":        "field",
			"ip":          settings.ProtectCIDRs,
			"outboundTag": "direct",
		})
	}
	prependRules = append(prependRules, map[string]interface{}{
		"type":        "field",
		"inboundTag":  []string{"tun-in"},
		"network":     "udp",
		"port":        "443",
		"outboundTag": "block",
	})
	if runtimeAssetExists(settings, "geosite.dat") {
		prependRules = append(prependRules, map[string]interface{}{
			"type":        "field",
			"inboundTag":  []string{"dns-cn"},
			"outboundTag": "direct",
		})
	}
	prependRules = append(prependRules, map[string]interface{}{
		"type":        "field",
		"inboundTag":  []string{"dns-remote"},
		"balancerTag": "node-pool-active",
	})

	tunDirectRules := make([]interface{}, 0, 2)
	if runtimeAssetExists(settings, "geosite.dat") {
		tunDirectRules = append(tunDirectRules, map[string]interface{}{
			"type":        "field",
			"domain":      []string{"geosite:cn"},
			"outboundTag": "direct",
		})
	}
	if runtimeAssetExists(settings, "geoip.dat") {
		tunDirectRules = append(tunDirectRules, map[string]interface{}{
			"type":        "field",
			"ip":          []string{"geoip:cn"},
			"outboundTag": "direct",
		})
	}
	tunCatchAllRule := map[string]interface{}{
		"type":        "field",
		"inboundTag":  []string{"tun-in"},
		"balancerTag": "node-pool-active",
	}

	rules := append(prependRules, priorityRules...)
	rules = append(rules, tunDirectRules...)
	rules = append(rules, tunCatchAllRule)
	routing["rules"] = rules

	routing["balancers"] = []interface{}{
		map[string]interface{}{
			"tag":         "node-pool-active",
			"selector":    balancerConfig.Selectors,
			"strategy":    balancerConfig.Strategy,
			"fallbackTag": balancerConfig.FallbackTag,
		},
	}
	config["routing"] = routing
	injectTunBurstObservatory(config, balancerConfig.Selectors)

	outbounds = filterOutboundsByTag(outbounds, "dns-out")
	outbounds = ensureTunUtilityOutbounds(outbounds)
	outbounds = append(outbounds, map[string]interface{}{
		"tag":      "dns-out",
		"protocol": "dns",
		"settings": map[string]interface{}{
			"nonIPQuery": "skip",
		},
	})
	config["outbounds"] = append(outbounds, poolOutbounds...)

	formatted, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode runtime config: %w", err)
	}

	return append(bytes.TrimRight(formatted, "\n"), '\n'), nil
}

type tunBalancerConfig struct {
	Selectors   []string
	Strategy    map[string]interface{}
	FallbackTag string
}

func buildTunBalancerConfig(settings *TunFeatureSettings, activeNodes []NodeRecord) tunBalancerConfig {
	policy := defaultTunSelectionPolicy
	if settings != nil {
		policy = normalizeTunSelectionPolicy(settings.SelectionPolicy)
	}

	allSelectors := buildTunOutboundSelectors(activeNodes)
	if len(allSelectors) == 0 {
		return tunBalancerConfig{
			Selectors: allSelectors,
			Strategy: map[string]interface{}{
				"type": "roundrobin",
			},
		}
	}

	switch policy {
	case TunSelectionPolicyLowestLatency:
		return tunBalancerConfig{
			Selectors:   allSelectors,
			Strategy:    map[string]interface{}{"type": "leastping"},
			FallbackTag: buildTunOutboundTag(bestNodeByLowestLatency(activeNodes).ID),
		}
	case TunSelectionPolicyLowestFailRate:
		candidates := pickLowestFailRateNodes(activeNodes, tunLowestFailRateSubsetSize)
		selectors := buildTunOutboundSelectors(candidates)
		if len(selectors) == 0 {
			selectors = allSelectors
		}
		return tunBalancerConfig{
			Selectors:   selectors,
			Strategy:    map[string]interface{}{"type": "leastping"},
			FallbackTag: buildTunOutboundTag(bestNodeByLowestFailRate(activeNodes).ID),
		}
	default:
		return tunBalancerConfig{
			Selectors:   allSelectors,
			Strategy:    map[string]interface{}{"type": "leastload"},
			FallbackTag: buildTunOutboundTag(bestNodeByFastestPriority(activeNodes).ID),
		}
	}
}

func injectTunBurstObservatory(config map[string]interface{}, selectors []string) {
	if len(selectors) == 0 {
		return
	}

	burstConfig := map[string]interface{}{
		"subjectSelector": selectors,
		"pingConfig": map[string]interface{}{
			"destination": "https://www.gstatic.com/generate_204",
			"interval":    "15s",
			"sampling":    4,
			"timeout":     "5s",
			"httpMethod":  "HEAD",
		},
	}

	if existingBurst, ok := config["burstObservatory"].(map[string]interface{}); ok {
		if pingConfig, ok := existingBurst["pingConfig"].(map[string]interface{}); ok {
			mergedPingConfig := map[string]interface{}{}
			for key, value := range pingConfig {
				mergedPingConfig[key] = value
			}
			if _, ok := mergedPingConfig["destination"]; !ok || strings.TrimSpace(fmt.Sprint(mergedPingConfig["destination"])) == "" {
				mergedPingConfig["destination"] = "https://www.gstatic.com/generate_204"
			}
			if _, ok := mergedPingConfig["interval"]; !ok || strings.TrimSpace(fmt.Sprint(mergedPingConfig["interval"])) == "" {
				mergedPingConfig["interval"] = "15s"
			}
			if _, ok := mergedPingConfig["sampling"]; !ok {
				mergedPingConfig["sampling"] = 4
			}
			if _, ok := mergedPingConfig["timeout"]; !ok || strings.TrimSpace(fmt.Sprint(mergedPingConfig["timeout"])) == "" {
				mergedPingConfig["timeout"] = "5s"
			}
			if _, ok := mergedPingConfig["httpMethod"]; !ok || strings.TrimSpace(fmt.Sprint(mergedPingConfig["httpMethod"])) == "" {
				mergedPingConfig["httpMethod"] = "HEAD"
			}
			burstConfig["pingConfig"] = mergedPingConfig
		}
	} else if observatoryConfig, ok := config["observatory"].(map[string]interface{}); ok {
		pingConfig := burstConfig["pingConfig"].(map[string]interface{})
		if value, ok := observatoryConfig["probeURL"]; ok && strings.TrimSpace(fmt.Sprint(value)) != "" {
			pingConfig["destination"] = value
		}
		if value, ok := observatoryConfig["probeInterval"]; ok && strings.TrimSpace(fmt.Sprint(value)) != "" {
			pingConfig["interval"] = value
		}
	}

	config["burstObservatory"] = burstConfig
	delete(config, "observatory")
}

func buildTunOutboundSelectors(activeNodes []NodeRecord) []string {
	selectors := make([]string, 0, len(activeNodes))
	for _, node := range activeNodes {
		selectors = append(selectors, buildTunOutboundTag(node.ID))
	}
	return selectors
}

func buildTunOutboundTag(nodeID string) string {
	return "pool-active-" + nodeID
}

func buildActivePoolOutbounds(activeNodes []NodeRecord) ([]interface{}, error) {
	result := make([]interface{}, 0, len(activeNodes))
	for _, node := range activeNodes {
		link, err := ParseShareLinkURI(node.URI)
		if err != nil {
			return nil, fmt.Errorf("parse active node %s: %w", node.ID, err)
		}
		outboundJSON, err := BuildOutboundJSON(link, buildTunOutboundTag(node.ID))
		if err != nil {
			return nil, fmt.Errorf("build active node %s: %w", node.ID, err)
		}

		var outbound map[string]interface{}
		if err := json.Unmarshal(outboundJSON, &outbound); err != nil {
			return nil, fmt.Errorf("decode active outbound %s: %w", node.ID, err)
		}
		if err := resolveTunOutboundAddresses(outbound); err != nil {
			return nil, fmt.Errorf("normalize active outbound %s: %w", node.ID, err)
		}
		result = append(result, outbound)
	}
	return result, nil
}

func buildTunDNSConfig(settings *TunFeatureSettings) map[string]interface{} {
	servers := make([]interface{}, 0, len(defaultTunChinaDNS)+len(settings.RemoteDNS))
	hasGeosite := runtimeAssetExists(settings, "geosite.dat")
	hasGeoip := runtimeAssetExists(settings, "geoip.dat")

	if hasGeosite {
		for _, address := range defaultTunChinaDNS {
			server := map[string]interface{}{
				"address":      normalizeTunResolverAddress(address),
				"domains":      []string{"geosite:cn"},
				"skipFallback": true,
				"tag":          "dns-cn",
			}
			if hasGeoip {
				server["expectIPs"] = []string{"geoip:cn"}
			}
			servers = append(servers, server)
		}
	}

	for _, address := range uniqStrings(settings.RemoteDNS) {
		servers = append(servers, map[string]interface{}{
			"address": normalizeTunResolverAddress(address),
			"tag":     "dns-remote",
		})
	}

	if len(servers) == 0 {
		return nil
	}

	return map[string]interface{}{
		"servers":                servers,
		"queryStrategy":          "UseIP",
		"disableFallbackIfMatch": hasGeosite,
	}
}

func normalizeTunSelectionPolicy(value string) TunSelectionPolicy {
	switch TunSelectionPolicy(strings.ToLower(strings.TrimSpace(value))) {
	case TunSelectionPolicyLowestLatency:
		return TunSelectionPolicyLowestLatency
	case TunSelectionPolicyLowestFailRate:
		return TunSelectionPolicyLowestFailRate
	default:
		return TunSelectionPolicyFastest
	}
}

func normalizeTunRouteMode(value string) TunRouteMode {
	switch TunRouteMode(strings.ToLower(strings.TrimSpace(value))) {
	case TunRouteModeAutoTested:
		return TunRouteModeAutoTested
	default:
		return TunRouteModeStrictProxy
	}
}

func normalizeTunDomainRules(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		next := normalizeTunDomainRule(value)
		if next == "" {
			continue
		}
		normalized = append(normalized, next)
	}
	return uniqStrings(normalized)
}

func normalizeTunDomainRule(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	switch {
	case strings.HasPrefix(trimmed, "*."):
		host := strings.Trim(strings.TrimPrefix(trimmed, "*."), ".")
		if host == "" {
			return ""
		}
		return "domain:" + host
	case strings.HasPrefix(trimmed, "."):
		host := strings.Trim(strings.TrimPrefix(trimmed, "."), ".")
		if host == "" {
			return ""
		}
		return "domain:" + host
	default:
		return trimmed
	}
}

func stringSliceFromAny(raw interface{}) []string {
	switch typed := raw.(type) {
	case []string:
		return append([]string{}, typed...)
	case []interface{}:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				values = append(values, text)
			}
		}
		return values
	default:
		return nil
	}
}

func tunNodeFailRate(node NodeRecord) float64 {
	if node.TotalPings <= 0 {
		return 1
	}
	return float64(node.FailedPings) / float64(node.TotalPings)
}

func tunNodeDelay(node NodeRecord) int64 {
	if node.AvgDelayMs <= 0 {
		return int64(^uint64(0) >> 1)
	}
	return node.AvgDelayMs
}

func tunNodeCheckedAt(node NodeRecord) int64 {
	if node.LastCheckedAt != nil {
		return node.LastCheckedAt.UnixNano()
	}
	if node.StatusUpdatedAt != nil {
		return node.StatusUpdatedAt.UnixNano()
	}
	return node.AddedAt.UnixNano()
}

func compareNodesByLowestLatency(a, b NodeRecord) bool {
	if tunNodeDelay(a) != tunNodeDelay(b) {
		return tunNodeDelay(a) < tunNodeDelay(b)
	}
	if tunNodeFailRate(a) != tunNodeFailRate(b) {
		return tunNodeFailRate(a) < tunNodeFailRate(b)
	}
	if a.ConsecutiveFails != b.ConsecutiveFails {
		return a.ConsecutiveFails < b.ConsecutiveFails
	}
	if a.TotalPings != b.TotalPings {
		return a.TotalPings > b.TotalPings
	}
	return tunNodeCheckedAt(a) > tunNodeCheckedAt(b)
}

func compareNodesByLowestFailRate(a, b NodeRecord) bool {
	if tunNodeFailRate(a) != tunNodeFailRate(b) {
		return tunNodeFailRate(a) < tunNodeFailRate(b)
	}
	if a.ConsecutiveFails != b.ConsecutiveFails {
		return a.ConsecutiveFails < b.ConsecutiveFails
	}
	if tunNodeDelay(a) != tunNodeDelay(b) {
		return tunNodeDelay(a) < tunNodeDelay(b)
	}
	if a.TotalPings != b.TotalPings {
		return a.TotalPings > b.TotalPings
	}
	return tunNodeCheckedAt(a) > tunNodeCheckedAt(b)
}

func compareNodesByFastestPriority(a, b NodeRecord) bool {
	if tunNodeDelay(a) != tunNodeDelay(b) {
		return tunNodeDelay(a) < tunNodeDelay(b)
	}
	if tunNodeFailRate(a) != tunNodeFailRate(b) {
		return tunNodeFailRate(a) < tunNodeFailRate(b)
	}
	if a.ConsecutiveFails != b.ConsecutiveFails {
		return a.ConsecutiveFails < b.ConsecutiveFails
	}
	if a.TotalPings != b.TotalPings {
		return a.TotalPings > b.TotalPings
	}
	return tunNodeCheckedAt(a) > tunNodeCheckedAt(b)
}

func bestNodeByLowestLatency(nodes []NodeRecord) NodeRecord {
	sorted := append([]NodeRecord(nil), nodes...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return compareNodesByLowestLatency(sorted[i], sorted[j])
	})
	return sorted[0]
}

func bestNodeByLowestFailRate(nodes []NodeRecord) NodeRecord {
	sorted := append([]NodeRecord(nil), nodes...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return compareNodesByLowestFailRate(sorted[i], sorted[j])
	})
	return sorted[0]
}

func bestNodeByFastestPriority(nodes []NodeRecord) NodeRecord {
	sorted := append([]NodeRecord(nil), nodes...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return compareNodesByFastestPriority(sorted[i], sorted[j])
	})
	return sorted[0]
}

func pickLowestFailRateNodes(nodes []NodeRecord, limit int) []NodeRecord {
	sorted := append([]NodeRecord(nil), nodes...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return compareNodesByLowestFailRate(sorted[i], sorted[j])
	})
	if limit <= 0 || limit >= len(sorted) {
		return sorted
	}
	return sorted[:limit]
}

func normalizeTunResolverAddress(address string) string {
	trimmed := strings.TrimSpace(address)
	if trimmed == "" {
		return trimmed
	}
	if strings.Contains(trimmed, "://") || strings.EqualFold(trimmed, "localhost") {
		return trimmed
	}
	return "tcp://" + trimmed
}

func resolveTunOutboundAddresses(outbound map[string]interface{}) error {
	if len(outbound) == 0 {
		return nil
	}

	settings, _ := outbound["settings"].(map[string]interface{})
	if len(settings) == 0 {
		return nil
	}

	tag, _ := outbound["tag"].(string)
	for _, key := range []string{"servers", "vnext"} {
		rawEntries, _ := settings[key].([]interface{})
		for _, rawEntry := range rawEntries {
			entry, ok := rawEntry.(map[string]interface{})
			if !ok {
				continue
			}
			address, _ := entry["address"].(string)
			resolved, err := resolveTunOutboundAddress(address)
			if err != nil {
				return fmt.Errorf("resolve %s address %q: %w", tag, address, err)
			}
			entry["address"] = resolved
		}
	}

	return nil
}

func resolveTunOutboundAddress(address string) (string, error) {
	trimmed := strings.TrimSpace(address)
	if trimmed == "" {
		return "", fmt.Errorf("empty address")
	}
	if ip := stdnet.ParseIP(trimmed); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			return ip4.String(), nil
		}
		return "", fmt.Errorf("ipv6 address %q is not supported by the TUN helper route guard", trimmed)
	}

	ips, err := stdnet.LookupIP(trimmed)
	if err != nil {
		return "", err
	}
	for _, ip := range ips {
		if ip4 := ip.To4(); ip4 != nil {
			return ip4.String(), nil
		}
	}
	return "", fmt.Errorf("no IPv4 address found")
}

func filterOutboundsByTag(outbounds []interface{}, tag string) []interface{} {
	if tag == "" {
		return append([]interface{}{}, outbounds...)
	}

	filtered := make([]interface{}, 0, len(outbounds))
	for _, rawOutbound := range outbounds {
		outbound, ok := rawOutbound.(map[string]interface{})
		if ok && outbound["tag"] == tag {
			continue
		}
		filtered = append(filtered, rawOutbound)
	}
	return filtered
}

func ensureTunUtilityOutbounds(outbounds []interface{}) []interface{} {
	result := append([]interface{}{}, outbounds...)
	if !hasOutboundTag(result, "direct") {
		result = append(result, map[string]interface{}{
			"tag":      "direct",
			"protocol": "freedom",
			"settings": map[string]interface{}{},
		})
	}
	if !hasOutboundTag(result, "block") {
		result = append(result, map[string]interface{}{
			"tag":      "block",
			"protocol": "blackhole",
			"settings": map[string]interface{}{},
		})
	}
	return result
}

func hasOutboundTag(outbounds []interface{}, tag string) bool {
	for _, rawOutbound := range outbounds {
		outbound, ok := rawOutbound.(map[string]interface{})
		if ok && outbound["tag"] == tag {
			return true
		}
	}
	return false
}

func runtimeAssetExists(settings *TunFeatureSettings, assetName string) bool {
	if strings.TrimSpace(assetName) == "" {
		return false
	}

	candidates := make([]string, 0, 4)
	if settings != nil && strings.TrimSpace(settings.BinaryPath) != "" {
		candidates = append(candidates, filepath.Join(filepath.Dir(settings.BinaryPath), assetName))
	}
	candidates = append(candidates,
		filepath.Join("/usr/local/share/xray", assetName),
		filepath.Join("/usr/share/xray", assetName),
		filepath.Join("/opt/share/xray", assetName),
	)

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return true
		}
	}

	return false
}

func splitTunRoutingRules(rules []interface{}) ([]interface{}, []interface{}) {
	priorityRules := make([]interface{}, 0, len(rules))
	fallbackRules := make([]interface{}, 0, len(rules))

	for _, rule := range rules {
		ruleMap, ok := rule.(map[string]interface{})
		if ok && isGenericTunFallbackRule(ruleMap) {
			fallbackRules = append(fallbackRules, rule)
			continue
		}
		priorityRules = append(priorityRules, rule)
	}

	return priorityRules, fallbackRules
}

func filterTunPriorityRules(rules []interface{}, settings *TunFeatureSettings, directProbeResults map[string]bool) []interface{} {
	filtered := make([]interface{}, 0, len(rules))
	routeMode := normalizeTunRouteMode("")
	if settings != nil {
		routeMode = normalizeTunRouteMode(settings.RouteMode)
	}
	for _, rule := range rules {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok {
			filtered = append(filtered, rule)
			continue
		}

		nextRule, keep := filterTunDirectRule(ruleMap, settings, routeMode, directProbeResults)
		if keep {
			filtered = append(filtered, nextRule)
		}
	}
	return filtered
}

func filterTunDirectRule(rule map[string]interface{}, settings *TunFeatureSettings, routeMode TunRouteMode, directProbeResults map[string]bool) (map[string]interface{}, bool) {
	if len(rule) == 0 {
		return rule, true
	}
	outboundTag, _ := rule["outboundTag"].(string)
	if outboundTag != "direct" {
		return rule, true
	}
	if _, hasInboundTag := rule["inboundTag"]; hasInboundTag {
		return rule, true
	}

	filteredRule := make(map[string]interface{}, len(rule))
	for key, value := range rule {
		filteredRule[key] = value
	}

	probePorts := tunDirectProbePorts(rule)

	if rawDomains, ok := rule["domain"]; ok {
		domains := normalizeTunDomainRules(filterTunStringList(rawDomains, func(string) bool {
			return true
		}))
		domains = filterTunStringSlice(domains, func(value string) bool {
			return shouldKeepTunDirectDomainRule(value, settings, routeMode, directProbeResults, probePorts)
		})
		if len(domains) > 0 {
			filteredRule["domain"] = domains
		} else {
			delete(filteredRule, "domain")
		}
	}

	if rawIPs, ok := rule["ip"]; ok {
		ips := filterTunStringList(rawIPs, func(value string) bool {
			return shouldKeepTunDirectIPRule(value, settings, routeMode, directProbeResults, probePorts)
		})
		if len(ips) > 0 {
			filteredRule["ip"] = ips
		} else {
			delete(filteredRule, "ip")
		}
	}

	_, hasDomain := filteredRule["domain"]
	_, hasIP := filteredRule["ip"]
	if !hasDomain && !hasIP {
		return nil, false
	}

	return filteredRule, true
}

func shouldKeepTunDirectDomainRule(value string, settings *TunFeatureSettings, routeMode TunRouteMode, directProbeResults map[string]bool, probePorts []int) bool {
	if isProtectedTunDomainRule(value, settings) {
		return true
	}
	if routeMode != TunRouteModeAutoTested {
		return false
	}

	request, ok := buildTunDirectProbeRequestForDomain(value, probePorts)
	if !ok {
		return false
	}
	return directProbeResults[request.Key]
}

func shouldKeepTunDirectIPRule(value string, settings *TunFeatureSettings, routeMode TunRouteMode, directProbeResults map[string]bool, probePorts []int) bool {
	if isProtectedTunCIDRRule(value, settings) {
		return true
	}
	if routeMode != TunRouteModeAutoTested {
		return false
	}

	request, ok := buildTunDirectProbeRequestForIP(value, probePorts)
	if !ok {
		return false
	}
	return directProbeResults[request.Key]
}

func isGenericTunFallbackRule(rule map[string]interface{}) bool {
	if len(rule) == 0 {
		return false
	}

	for key := range rule {
		switch key {
		case "type", "network", "outboundTag", "balancerTag":
			continue
		default:
			return false
		}
	}

	_, hasOutbound := rule["outboundTag"]
	_, hasBalancer := rule["balancerTag"]
	return hasOutbound || hasBalancer
}

func resolveTunDirectProbeResults(settings *TunFeatureSettings, rules []interface{}, overrides map[string]bool) map[string]bool {
	results := make(map[string]bool)
	for key, value := range overrides {
		results[key] = value
	}

	routeMode := normalizeTunRouteMode("")
	if settings != nil {
		routeMode = normalizeTunRouteMode(settings.RouteMode)
	}
	if len(overrides) > 0 || routeMode != TunRouteModeAutoTested {
		return results
	}

	requests := collectTunDirectProbeRequests(rules, settings)
	if len(requests) == 0 {
		return results
	}

	cache := loadTunDirectProbeCache(settings)
	now := time.Now()
	pending := make([]tunDirectProbeRequest, 0, len(requests))

	for _, request := range requests {
		entry, ok := cache.Entries[request.Key]
		if ok && !entry.CheckedAt.IsZero() && now.Sub(entry.CheckedAt) <= tunDirectProbeCacheTTL {
			results[request.Key] = entry.Decision
			continue
		}
		pending = append(pending, request)
	}

	for key, decision := range runTunDirectProbeRequests(pending) {
		results[key] = decision
		cache.Entries[key] = tunDirectProbeCacheEntry{
			Decision:  decision,
			CheckedAt: now,
		}
	}

	saveTunDirectProbeCache(settings, cache)
	return results
}

func collectTunDirectProbeRequests(rules []interface{}, settings *TunFeatureSettings) []tunDirectProbeRequest {
	routeMode := normalizeTunRouteMode("")
	if settings != nil {
		routeMode = normalizeTunRouteMode(settings.RouteMode)
	}
	if routeMode != TunRouteModeAutoTested {
		return nil
	}

	seen := make(map[string]struct{})
	requests := make([]tunDirectProbeRequest, 0)
	for _, rawRule := range rules {
		rule, ok := rawRule.(map[string]interface{})
		if !ok {
			continue
		}

		outboundTag, _ := rule["outboundTag"].(string)
		if outboundTag != "direct" {
			continue
		}
		if _, hasInboundTag := rule["inboundTag"]; hasInboundTag {
			continue
		}

		probePorts := tunDirectProbePorts(rule)

		if rawDomains, ok := rule["domain"]; ok {
			for _, value := range filterTunStringSlice(normalizeTunDomainRules(filterTunStringList(rawDomains, func(string) bool {
				return true
			})), func(value string) bool {
				return !isProtectedTunDomainRule(value, settings)
			}) {
				request, ok := buildTunDirectProbeRequestForDomain(value, probePorts)
				if !ok {
					continue
				}
				if _, exists := seen[request.Key]; exists {
					continue
				}
				seen[request.Key] = struct{}{}
				requests = append(requests, request)
			}
		}

		if rawIPs, ok := rule["ip"]; ok {
			for _, value := range filterTunStringList(rawIPs, func(value string) bool {
				return !isProtectedTunCIDRRule(value, settings)
			}) {
				request, ok := buildTunDirectProbeRequestForIP(value, probePorts)
				if !ok {
					continue
				}
				if _, exists := seen[request.Key]; exists {
					continue
				}
				seen[request.Key] = struct{}{}
				requests = append(requests, request)
			}
		}
	}

	return requests
}

func tunDirectProbePorts(rule map[string]interface{}) []int {
	if len(rule) == 0 {
		return []int{443, 80}
	}

	if network, ok := rule["network"].(string); ok {
		normalized := strings.ToLower(strings.TrimSpace(network))
		if normalized != "" && !strings.Contains(normalized, "tcp") {
			return nil
		}
	}

	ports := make([]int, 0, 3)
	addPort := func(port int) {
		if port < 1 || port > 65535 {
			return
		}
		for _, existing := range ports {
			if existing == port {
				return
			}
		}
		ports = append(ports, port)
	}

	switch typed := rule["port"].(type) {
	case float64:
		addPort(int(typed))
	case int:
		addPort(typed)
	case string:
		for _, segment := range strings.Split(typed, ",") {
			part := strings.TrimSpace(segment)
			if part == "" {
				continue
			}
			if strings.Contains(part, "-") {
				bounds := strings.SplitN(part, "-", 2)
				if port, err := strconv.Atoi(strings.TrimSpace(bounds[0])); err == nil {
					addPort(port)
				}
				continue
			}
			if port, err := strconv.Atoi(part); err == nil {
				addPort(port)
			}
		}
	}

	if len(ports) == 0 {
		return []int{443, 80}
	}
	return ports
}

func buildTunDirectProbeRequestForDomain(value string, ports []int) (tunDirectProbeRequest, bool) {
	normalized := strings.ToLower(normalizeTunDomainRule(value))
	if normalized == "" {
		return tunDirectProbeRequest{}, false
	}

	host := ""
	switch {
	case strings.HasPrefix(normalized, "full:"):
		host = strings.TrimPrefix(normalized, "full:")
	case strings.HasPrefix(normalized, "domain:"):
		host = strings.TrimPrefix(normalized, "domain:")
	default:
		return tunDirectProbeRequest{}, false
	}

	host = strings.Trim(host, ".")
	if host == "" || stdnet.ParseIP(host) != nil || len(ports) == 0 {
		return tunDirectProbeRequest{}, false
	}

	return tunDirectProbeRequest{
		Key:   tunDirectProbeKey("domain", host, ports),
		Host:  host,
		Ports: append([]int{}, ports...),
	}, true
}

func buildTunDirectProbeRequestForIP(value string, ports []int) (tunDirectProbeRequest, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || len(ports) == 0 {
		return tunDirectProbeRequest{}, false
	}

	host := ""
	switch {
	case strings.Contains(trimmed, "/"):
		ip, network, err := stdnet.ParseCIDR(trimmed)
		if err != nil {
			return tunDirectProbeRequest{}, false
		}
		ones, bits := network.Mask.Size()
		if ones != bits {
			return tunDirectProbeRequest{}, false
		}
		host = ip.String()
	default:
		host = trimmed
	}

	ip := stdnet.ParseIP(host)
	if ip == nil || ip.To4() == nil {
		return tunDirectProbeRequest{}, false
	}

	host = ip.String()
	return tunDirectProbeRequest{
		Key:   tunDirectProbeKey("ip", host, ports),
		Host:  host,
		Ports: append([]int{}, ports...),
	}, true
}

func tunDirectProbeKey(kind, host string, ports []int) string {
	portText := make([]string, 0, len(ports))
	for _, port := range ports {
		portText = append(portText, strconv.Itoa(port))
	}
	return kind + "|" + strings.ToLower(strings.TrimSpace(host)) + "|" + strings.Join(portText, ",")
}

func loadTunDirectProbeCache(settings *TunFeatureSettings) tunDirectProbeCache {
	cache := tunDirectProbeCache{
		Version: tunDirectProbeCacheVersion,
		Entries: make(map[string]tunDirectProbeCacheEntry),
	}
	if settings == nil || strings.TrimSpace(settings.StateDir) == "" {
		return cache
	}

	raw, err := os.ReadFile(filepath.Join(settings.StateDir, tunDirectProbeCacheFileName))
	if err != nil {
		return cache
	}
	if err := json.Unmarshal(raw, &cache); err != nil {
		return tunDirectProbeCache{
			Version: tunDirectProbeCacheVersion,
			Entries: make(map[string]tunDirectProbeCacheEntry),
		}
	}
	if cache.Version != tunDirectProbeCacheVersion || cache.Entries == nil {
		cache.Version = tunDirectProbeCacheVersion
		cache.Entries = make(map[string]tunDirectProbeCacheEntry)
	}
	return cache
}

func saveTunDirectProbeCache(settings *TunFeatureSettings, cache tunDirectProbeCache) {
	if settings == nil || strings.TrimSpace(settings.StateDir) == "" {
		return
	}
	if cache.Entries == nil {
		cache.Entries = make(map[string]tunDirectProbeCacheEntry)
	}
	cache.Version = tunDirectProbeCacheVersion

	if err := os.MkdirAll(settings.StateDir, 0755); err != nil {
		return
	}

	encoded, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(settings.StateDir, tunDirectProbeCacheFileName), append(bytes.TrimRight(encoded, "\n"), '\n'), 0644)
}

func runTunDirectProbeRequests(requests []tunDirectProbeRequest) map[string]bool {
	results := make(map[string]bool, len(requests))
	if len(requests) == 0 {
		return results
	}

	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)
	sem := make(chan struct{}, tunDirectProbeConcurrency)

	for _, request := range requests {
		request := request
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			decision := probeTunDirectRequest(request)
			<-sem

			mu.Lock()
			results[request.Key] = decision
			mu.Unlock()
		}()
	}

	wg.Wait()
	return results
}

func probeTunDirectRequest(request tunDirectProbeRequest) bool {
	if strings.TrimSpace(request.Host) == "" || len(request.Ports) == 0 {
		return false
	}

	dialer := stdnet.Dialer{Timeout: tunDirectProbeTimeout}
	for _, port := range request.Ports {
		conn, err := dialer.Dial("tcp", stdnet.JoinHostPort(request.Host, strconv.Itoa(port)))
		if err == nil {
			_ = conn.Close()
			return true
		}
	}
	return false
}

func filterTunStringList(raw interface{}, keep func(string) bool) []string {
	values := make([]string, 0)
	switch typed := raw.(type) {
	case []interface{}:
		for _, item := range typed {
			text, _ := item.(string)
			if keep(text) {
				values = append(values, text)
			}
		}
	case []string:
		for _, item := range typed {
			if keep(item) {
				values = append(values, item)
			}
		}
	}
	return values
}

func filterTunStringSlice(values []string, keep func(string) bool) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if keep(value) {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func isProtectedTunCIDRRule(value string, settings *TunFeatureSettings) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || settings == nil {
		return false
	}
	for _, protected := range settings.ProtectCIDRs {
		if trimmed == protected {
			return true
		}
	}
	return false
}

func isProtectedTunDomainRule(value string, settings *TunFeatureSettings) bool {
	normalized := strings.ToLower(normalizeTunDomainRule(value))
	if normalized == "" {
		return false
	}
	if settings != nil {
		for _, protected := range settings.ProtectDomains {
			if normalized == strings.ToLower(normalizeTunDomainRule(protected)) {
				return true
			}
		}
	}
	if normalized == "full:localhost" || normalized == "domain:localhost" {
		return true
	}

	host := ""
	switch {
	case strings.HasPrefix(normalized, "full:"):
		host = strings.TrimPrefix(normalized, "full:")
	case strings.HasPrefix(normalized, "domain:"):
		host = strings.TrimPrefix(normalized, "domain:")
	default:
		return false
	}
	host = strings.Trim(host, ".")
	if host == "" {
		return false
	}

	for _, suffix := range []string{"ts.net", "local", "localdomain", "lan", "home", "internal", "test", "arpa"} {
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			return true
		}
	}
	return false
}

func resolvePath(baseDir, value string) string {
	if value == "" {
		return value
	}
	if filepath.IsAbs(value) {
		return filepath.Clean(value)
	}
	return filepath.Clean(filepath.Join(baseDir, value))
}

func uniqStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func checkSudoReady(settings *TunFeatureSettings) bool {
	if _, err := exec.LookPath("sudo"); err != nil {
		return false
	}

	cmd := exec.Command("sudo", "-n", "-l")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	listing := string(output)
	if settings == nil {
		return strings.Contains(listing, "NOPASSWD")
	}

	if settings.HelperPath != "" && strings.Contains(listing, settings.HelperPath) {
		return true
	}

	if settings.BinaryPath != "" && strings.Contains(listing, settings.BinaryPath) {
		return true
	}

	return false
}

func filesMatch(leftPath, rightPath string) (bool, error) {
	if leftPath == "" || rightPath == "" {
		return false, fmt.Errorf("compare files: empty path")
	}

	leftInfo, err := os.Stat(leftPath)
	if err != nil {
		return false, err
	}
	rightInfo, err := os.Stat(rightPath)
	if err != nil {
		return false, err
	}

	if os.SameFile(leftInfo, rightInfo) {
		return true, nil
	}
	if leftInfo.Size() != rightInfo.Size() {
		return false, nil
	}

	leftSum, err := fileSHA256(leftPath)
	if err != nil {
		return false, err
	}
	rightSum, err := fileSHA256(rightPath)
	if err != nil {
		return false, err
	}
	return bytes.Equal(leftSum, rightSum), nil
}

func fileSHA256(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, err
	}
	return hash.Sum(nil), nil
}

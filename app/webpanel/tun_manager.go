package webpanel

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	stdnet "net"
	"net/http"
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
	defaultTunAggregation     = TunAggregationSettings{
		Enabled:            false,
		Mode:               string(TunAggregationModeSingleBest),
		MaxPathsPerSession: 2,
		SchedulerPolicy:    string(TunAggregationSchedulerPolicyWeightedSplit),
		Health: TunAggregationHealthSettings{
			MaxSessionLossPct:             5,
			MaxPathJitterMs:               120,
			RollbackOnConsecutiveFailures: 3,
		},
	}
	defaultTunCIDRs = []string{
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
type TunDestinationBindingPreset string
type TunDestinationBindingSelectionMode string
type TunAggregationMode string
type TunAggregationSchedulerPolicy string
type TunAggregationRuntimePath string
type TunAggregationStatusCode string

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
	TunDestinationBindingPresetOpenAI        TunDestinationBindingPreset = "openai"
	TunDestinationBindingPresetChatGPT       TunDestinationBindingPreset = "chatgpt"
	TunDestinationBindingPresetClaude        TunDestinationBindingPreset = "claude"
	TunDestinationBindingPresetGemini        TunDestinationBindingPreset = "gemini"
	TunDestinationBindingPresetGitHub        TunDestinationBindingPreset = "github"
	TunDestinationBindingPresetGitHubCopilot TunDestinationBindingPreset = "github_copilot"
	TunDestinationBindingPresetOpenRouter    TunDestinationBindingPreset = "openrouter"
	TunDestinationBindingPresetCursor        TunDestinationBindingPreset = "cursor"
	TunDestinationBindingPresetQwen          TunDestinationBindingPreset = "qwen"
	TunDestinationBindingPresetPerplexity    TunDestinationBindingPreset = "perplexity"
	TunDestinationBindingPresetDeepSeek      TunDestinationBindingPreset = "deepseek"
	TunDestinationBindingPresetCustom        TunDestinationBindingPreset = "custom"
)

const (
	TunDestinationBindingSelectionModePrimaryOnly     TunDestinationBindingSelectionMode = "primary_only"
	TunDestinationBindingSelectionModeFailoverOrdered TunDestinationBindingSelectionMode = "failover_ordered"
	TunDestinationBindingSelectionModeFailoverFastest TunDestinationBindingSelectionMode = "failover_fastest"
)

const (
	TunAggregationModeSingleBest    TunAggregationMode = "single_best"
	TunAggregationModeRedundant2    TunAggregationMode = "redundant_2"
	TunAggregationModeWeightedSplit TunAggregationMode = "weighted_split"
)

const (
	TunAggregationSchedulerPolicySingleBest    TunAggregationSchedulerPolicy = "single_best"
	TunAggregationSchedulerPolicyRedundant2    TunAggregationSchedulerPolicy = "redundant_2"
	TunAggregationSchedulerPolicyWeightedSplit TunAggregationSchedulerPolicy = "weighted_split"
)

const (
	TunAggregationPathStableSinglePath               TunAggregationRuntimePath = "stable_single_path"
	TunAggregationPathExperimentalUDPQUICAggregation TunAggregationRuntimePath = "experimental_udp_quic_aggregation"
)

const (
	TunAggregationStatusDisabled       TunAggregationStatusCode = "disabled"
	TunAggregationStatusRequested      TunAggregationStatusCode = "requested"
	TunAggregationStatusFallbackStable TunAggregationStatusCode = "fallback_stable"
)

const (
	tunLowestFailRateSubsetSize = 3
	tunDirectProbeTimeout       = 1200 * time.Millisecond
	tunDirectProbeCacheTTL      = 6 * time.Hour
	tunDirectProbeConcurrency   = 12
	tunDirectProbeCacheVersion  = 1
	tunDirectProbeCacheFileName = "route-probe-cache.json"
	tunPublicEgressProbeTimeout = 1800 * time.Millisecond
	tunPublicEgressCacheTTL     = 6 * time.Hour
	tunPublicEgressCacheVersion = 1
	tunPublicEgressCacheFile    = "egress-probe-cache.json"
	tunAggregationRuntimeFile   = "aggregation-runtime.json"
)

var tunDestinationBindingPresetDomains = map[TunDestinationBindingPreset][]string{
	TunDestinationBindingPresetOpenAI: {
		"domain:openai.com",
		"domain:api.openai.com",
		"domain:auth.openai.com",
		"domain:chatgpt.com",
		"domain:chat.openai.com",
		"domain:oaistatic.com",
		"domain:oaiusercontent.com",
	},
	TunDestinationBindingPresetChatGPT: {
		"domain:chatgpt.com",
		"domain:chat.openai.com",
		"domain:oaistatic.com",
		"domain:oaiusercontent.com",
	},
	TunDestinationBindingPresetClaude: {
		"domain:claude.ai",
		"domain:anthropic.com",
	},
	TunDestinationBindingPresetGemini: {
		"domain:gemini.google.com",
		"domain:ai.google.dev",
		"domain:aistudio.google.com",
		"full:generativelanguage.googleapis.com",
	},
	TunDestinationBindingPresetGitHub: {
		"full:api.github.com",
		"domain:github.com",
		"domain:githubusercontent.com",
		"domain:githubassets.com",
		"domain:github.io",
	},
	TunDestinationBindingPresetGitHubCopilot: {
		"full:github.com",
		"full:api.github.com",
		"full:copilot.github.com",
	},
	TunDestinationBindingPresetOpenRouter: {
		"domain:openrouter.ai",
	},
	TunDestinationBindingPresetCursor: {
		"domain:cursor.com",
	},
	TunDestinationBindingPresetQwen: {
		"domain:qwen.ai",
		"full:dashscope.aliyuncs.com",
	},
	TunDestinationBindingPresetPerplexity: {
		"domain:perplexity.ai",
	},
	TunDestinationBindingPresetDeepSeek: {
		"domain:deepseek.com",
	},
}

type TunManager struct {
	configPath         string
	xrayBin            string
	mu                 sync.Mutex
	directEgressProber func(*TunFeatureSettings) tunPublicEgressProbeResult
}

type TunDestinationBinding struct {
	Preset          string   `json:"preset"`
	Domains         []string `json:"domains,omitempty"`
	NodeID          string   `json:"nodeId"`
	FallbackNodeIDs []string `json:"fallbackNodeIds,omitempty"`
	SelectionMode   string   `json:"selectionMode,omitempty"`
}

type TunAggregationHealthSettings struct {
	MaxSessionLossPct             int `json:"maxSessionLossPct"`
	MaxPathJitterMs               int `json:"maxPathJitterMs"`
	RollbackOnConsecutiveFailures int `json:"rollbackOnConsecutiveFailures"`
}

type TunAggregationSettings struct {
	Enabled            bool                         `json:"enabled"`
	Mode               string                       `json:"mode"`
	MaxPathsPerSession int                          `json:"maxPathsPerSession"`
	SchedulerPolicy    string                       `json:"schedulerPolicy"`
	RelayEndpoint      string                       `json:"relayEndpoint"`
	Health             TunAggregationHealthSettings `json:"health"`
}

type TunAggregationStatus struct {
	Enabled            bool                           `json:"enabled"`
	Status             string                         `json:"status"`
	RequestedPath      string                         `json:"requestedPath"`
	EffectivePath      string                         `json:"effectivePath"`
	Ready              bool                           `json:"ready"`
	RelayConfigured    bool                           `json:"relayConfigured"`
	Mode               string                         `json:"mode"`
	MaxPathsPerSession int                            `json:"maxPathsPerSession"`
	SchedulerPolicy    string                         `json:"schedulerPolicy"`
	RelayEndpoint      string                         `json:"relayEndpoint,omitempty"`
	Reason             string                         `json:"reason"`
	Prototype          *TunAggregationPrototypeStatus `json:"prototype,omitempty"`
	Relay              *TunAggregationRelayStatus     `json:"relay,omitempty"`
	Benchmark          *TunAggregationBenchmarkStatus `json:"benchmark,omitempty"`
}

type TunFeatureSettings struct {
	BinaryPath          string                  `json:"binaryPath"`
	HelperPath          string                  `json:"helperPath"`
	StateDir            string                  `json:"stateDir"`
	RuntimeConfigPath   string                  `json:"runtimeConfigPath"`
	InterfaceName       string                  `json:"interfaceName"`
	MTU                 uint32                  `json:"mtu"`
	RemoteDNS           []string                `json:"remoteDns"`
	UseSudo             *bool                   `json:"useSudo"`
	AllowRemote         bool                    `json:"allowRemote"`
	SelectionPolicy     string                  `json:"selectionPolicy"`
	RouteMode           string                  `json:"routeMode"`
	ProtectCIDRs        []string                `json:"protectCidrs"`
	ProtectDomains      []string                `json:"protectDomains"`
	DestinationBindings []TunDestinationBinding `json:"destinationBindings,omitempty"`
	Aggregation         TunAggregationSettings  `json:"aggregation,omitempty"`
}

type TunEditableSettings struct {
	SelectionPolicy     string                  `json:"selectionPolicy"`
	RouteMode           string                  `json:"routeMode"`
	RemoteDNS           []string                `json:"remoteDns"`
	ProtectCIDRs        []string                `json:"protectCidrs"`
	ProtectDomains      []string                `json:"protectDomains"`
	DestinationBindings []TunDestinationBinding `json:"destinationBindings,omitempty"`
	Aggregation         TunAggregationSettings  `json:"aggregation,omitempty"`
}

type TunRoutingDiagnostic struct {
	Category string   `json:"category"`
	DNSPath  string   `json:"dnsPath"`
	Resolver string   `json:"resolver"`
	Route    string   `json:"route"`
	Reason   string   `json:"reason"`
	Domains  []string `json:"domains,omitempty"`
}

type tunDirectProbeCache struct {
	Version int                                 `json:"version"`
	Entries map[string]tunDirectProbeCacheEntry `json:"entries"`
}

type tunDirectProbeCacheEntry struct {
	Decision  bool      `json:"decision"`
	CheckedAt time.Time `json:"checkedAt"`
}

type tunPublicEgressCache struct {
	Version int                        `json:"version"`
	Direct  *tunPublicEgressCacheEntry `json:"direct,omitempty"`
}

type tunPublicEgressCacheEntry struct {
	IP        string    `json:"ip"`
	CheckedAt time.Time `json:"checkedAt"`
}

type tunPublicEgressProbeResult struct {
	IP        string
	Endpoint  string
	CheckedAt time.Time
	Error     string
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
	Status                      string                 `json:"status"`
	Running                     bool                   `json:"running"`
	Available                   bool                   `json:"available"`
	AllowRemote                 bool                   `json:"allowRemote"`
	UseSudo                     bool                   `json:"useSudo"`
	HelperExists                bool                   `json:"helperExists"`
	ElevationReady              bool                   `json:"elevationReady"`
	HelperCurrent               bool                   `json:"helperCurrent"`
	BinaryCurrent               bool                   `json:"binaryCurrent"`
	PrivilegeInstallRecommended bool                   `json:"privilegeInstallRecommended"`
	BinaryPath                  string                 `json:"binaryPath"`
	HelperPath                  string                 `json:"helperPath"`
	StateDir                    string                 `json:"stateDir"`
	RuntimeConfigPath           string                 `json:"runtimeConfigPath"`
	InterfaceName               string                 `json:"interfaceName"`
	MTU                         uint32                 `json:"mtu"`
	RemoteDNS                   []string               `json:"remoteDns"`
	ConfigPath                  string                 `json:"configPath"`
	XrayBinary                  string                 `json:"xrayBinary"`
	Message                     string                 `json:"message"`
	LastOutput                  string                 `json:"lastOutput,omitempty"`
	Diagnostics                 []string               `json:"diagnostics,omitempty"`
	DirectEgress                *TunEgressObservation  `json:"directEgress,omitempty"`
	ProxyEgress                 *TunEgressObservation  `json:"proxyEgress,omitempty"`
	Aggregation                 *TunAggregationStatus  `json:"aggregation,omitempty"`
	RoutingDiagnostics          []TunRoutingDiagnostic `json:"routingDiagnostics,omitempty"`
	MachineState                MachineState           `json:"machineState,omitempty"`
	LastStateReason             MachineStateReason     `json:"lastStateReason,omitempty"`
	LastStateChangedAt          *time.Time             `json:"lastStateChangedAt,omitempty"`
	RecentMachineEvents         []MachineEvent         `json:"recentMachineEvents,omitempty"`
}

type TunEgressObservation struct {
	Status    string     `json:"status"`
	Route     string     `json:"route"`
	IP        string     `json:"ip,omitempty"`
	CheckedAt *time.Time `json:"checkedAt,omitempty"`
	Source    string     `json:"source,omitempty"`
	Stale     bool       `json:"stale,omitempty"`
	Note      string     `json:"note,omitempty"`
	Error     string     `json:"error,omitempty"`
}

func NewTunManager(configPath string) (*TunManager, error) {
	xrayBin, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve current xray binary: %w", err)
	}

	return &TunManager{
		configPath:         configPath,
		xrayBin:            xrayBin,
		directEgressProber: defaultTunDirectEgressProber,
	}, nil
}

func (m *TunManager) Status() *TunStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.statusLocked(true)
}

func (m *TunManager) StatusWithoutEgressProbe() *TunStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.statusLocked(false)
}

func (m *TunManager) statusLocked(includeEgressProbe bool) *TunStatus {
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

	return m.inspectLocked(settings, includeEgressProbe)
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

	preflight := m.inspectLocked(settings, true)
	if preflight.Running {
		return preflight
	}
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
	status := m.inspectLocked(settings, true)
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
	status := m.inspectLocked(settings, true)
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
		status = m.inspectLocked(reloadedSettings, true)
	case settings != nil:
		status = m.inspectLocked(settings, true)
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

	if _, ok := resolveCommandPath("pkexec"); !ok {
		return "", fmt.Errorf("pkexec is not available")
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
		cmd := execCommandCompat("sudo", append([]string{"-A"}, installArgs...)...)
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

	if _, ok := resolveCommandPath("pkexec"); !ok {
		return "", fmt.Errorf("pkexec is not available")
	}

	cmd := execCommandCompat("pkexec", installArgs...)
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
	if _, ok := resolveCommandPath("sudo"); !ok {
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

// Some CI environments expose sudo/pkexec/test helpers as shebang scripts
// that are not directly spawnable on Windows. Use the declared interpreter when needed.
func prepareCommandInvocation(name string, args []string) (string, []string) {
	resolvedName, ok := resolveCommandPath(name)
	if !ok {
		return name, args
	}

	interpreter, interpreterArgs, ok := detectScriptInterpreter(resolvedName)
	if !ok {
		return resolvedName, args
	}

	invocationArgs := make([]string, 0, len(interpreterArgs)+1+len(args))
	invocationArgs = append(invocationArgs, interpreterArgs...)
	invocationArgs = append(invocationArgs, resolvedName)
	invocationArgs = append(invocationArgs, args...)
	return interpreter, invocationArgs
}

func resolveCommandPath(name string) (string, bool) {
	if candidate, err := exec.LookPath(name); err == nil {
		return candidate, true
	}

	if strings.ContainsAny(name, `/\`) {
		if info, err := os.Stat(name); err == nil && !info.IsDir() {
			return name, true
		}
		return name, false
	}

	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if strings.TrimSpace(dir) == "" {
			continue
		}
		candidate := filepath.Join(dir, name)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true
		}
	}

	return name, false
}

func detectScriptInterpreter(path string) (string, []string, bool) {
	content, err := os.ReadFile(path)
	if err != nil || !bytes.HasPrefix(content, []byte("#!")) {
		return "", nil, false
	}

	firstLine := content
	if newline := bytes.IndexByte(firstLine, '\n'); newline >= 0 {
		firstLine = firstLine[:newline]
	}
	firstLine = bytes.TrimSpace(bytes.TrimPrefix(bytes.TrimRight(firstLine, "\r"), []byte("#!")))
	fields := strings.Fields(string(firstLine))
	if len(fields) == 0 {
		return "", nil, false
	}

	interpreterName := fields[0]
	interpreterArgs := fields[1:]
	if filepath.Base(interpreterName) == "env" && len(fields) > 1 {
		interpreterName = fields[1]
		interpreterArgs = fields[2:]
	}

	interpreter, err := exec.LookPath(filepath.Base(interpreterName))
	if err != nil {
		return "", nil, false
	}
	return interpreter, interpreterArgs, true
}

func execCommandCompat(name string, args ...string) *exec.Cmd {
	cmdName, cmdArgs := prepareCommandInvocation(name, args)
	return exec.Command(cmdName, cmdArgs...)
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
		settings.RemoteDNS = normalizeTunRemoteDNS(settings.RemoteDNS)
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
	settings.DestinationBindings = normalizeTunDestinationBindings(settings.DestinationBindings)
	settings.Aggregation = normalizeTunAggregationSettings(settings.Aggregation)

	return settings, nil
}

func (m *TunManager) EditableSettings() (*TunEditableSettings, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.loadEditableSettingsLocked()
}

func (m *TunManager) SettingsSnapshot() (*TunFeatureSettings, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.loadSettings()
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

	remoteDNS := normalizeTunRemoteDNS(next.RemoteDNS)
	if len(remoteDNS) > 0 {
		tun["remoteDns"] = remoteDNS
	} else {
		delete(tun, "remoteDns")
	}
	delete(tun, "remoteDnsAutoPick")
	delete(tun, "remoteDnsMaxCount")

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

	destinationBindings := normalizeTunDestinationBindings(next.DestinationBindings)
	if len(destinationBindings) > 0 {
		tun["destinationBindings"] = destinationBindings
	} else {
		delete(tun, "destinationBindings")
	}

	tun["aggregation"] = tunAggregationSettingsToAny(normalizeTunAggregationSettings(next.Aggregation))

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
		RemoteDNS:       append([]string{}, defaultTunDNS...),
		Aggregation:     normalizeTunAggregationSettings(TunAggregationSettings{}),
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
	if remoteDNS := normalizeTunRemoteDNS(stringSliceFromAny(tun["remoteDns"])); len(remoteDNS) > 0 {
		settings.RemoteDNS = remoteDNS
	}
	settings.ProtectDomains = normalizeTunDomainRules(stringSliceFromAny(tun["protectDomains"]))
	settings.ProtectCIDRs = uniqStrings(stringSliceFromAny(tun["protectCidrs"]))
	settings.DestinationBindings = normalizeTunDestinationBindings(tunDestinationBindingsFromAny(tun["destinationBindings"]))
	settings.Aggregation = tunAggregationSettingsFromAny(tun["aggregation"])

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

func (m *TunManager) inspectLocked(settings *TunFeatureSettings, includeEgressProbe bool) *TunStatus {
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
	if includeEgressProbe {
		defer m.populateEgressObservationsLocked(settings, status)
	}

	if normalizeTunRouteMode(settings.RouteMode) == TunRouteModeAutoTested {
		status.Diagnostics = append(status.Diagnostics, "Auto-tested split routing will probe base direct rules before enabling transparent mode; the first start or stale cache refresh can take longer.")
	}
	status.Aggregation = buildTunAggregationStatus(settings)
	appendUniqueTunDiagnostic(status, formatTunAggregationDiagnostic(status.Aggregation))

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

func (m *TunManager) populateEgressObservationsLocked(settings *TunFeatureSettings, status *TunStatus) {
	if settings == nil || status == nil {
		return
	}

	status.DirectEgress = m.directEgressObservationLocked(settings, status.Running)
	status.ProxyEgress = buildTunProxyEgressObservation(status.Running)
	appendUniqueTunDiagnostic(status, formatTunEgressDiagnostic("Direct", status.DirectEgress))
	appendUniqueTunDiagnostic(status, formatTunEgressDiagnostic("Proxy", status.ProxyEgress))
}

func (m *TunManager) directEgressObservationLocked(settings *TunFeatureSettings, tunRunning bool) *TunEgressObservation {
	cache := loadTunPublicEgressCache(settings)

	if tunRunning {
		if tunDirectEgressProbeAllowedWhileRunning() {
			live := m.runDirectEgressProbeLocked(settings)
			if live.Error == "" {
				saveTunPublicEgressCache(settings, tunPublicEgressCache{
					Version: tunPublicEgressCacheVersion,
					Direct: &tunPublicEgressCacheEntry{
						IP:        live.IP,
						CheckedAt: live.CheckedAt,
					},
				})
				return buildTunLiveDirectEgressObservation(
					live,
					"Transparent TUN is active, but the current process is UID-bypassed, so this direct probe stayed off the proxy path.",
				)
			}
			if cached := cachedTunDirectEgressObservation(cache.Direct, "The live direct probe failed while transparent TUN was active, so the last successful independent direct probe was reused.", live.Error); cached != nil {
				return cached
			}
			return &TunEgressObservation{
				Status: "error",
				Route:  "system-direct",
				Source: "system-http-probe",
				Error:  live.Error,
				Note:   "Transparent TUN is active and the direct probe failed even though the current process is UID-bypassed.",
			}
		}

		if cache.Direct != nil && cache.Direct.IP != "" && !cache.Direct.CheckedAt.IsZero() {
			checkedAt := cache.Direct.CheckedAt
			observation := &TunEgressObservation{
				Status:    "cached",
				Route:     "system-direct",
				IP:        cache.Direct.IP,
				CheckedAt: &checkedAt,
				Source:    "cache",
				Note:      "Transparent TUN is active, so the panel reused the last independent direct probe instead of probing through the proxy path.",
			}
			if time.Since(cache.Direct.CheckedAt) > tunPublicEgressCacheTTL {
				observation.Status = "stale"
				observation.Stale = true
				observation.Note = "Transparent TUN is active, and the last independent direct probe is stale because the panel process is not UID-bypassed."
			}
			return observation
		}

		return &TunEgressObservation{
			Status: "blocked",
			Route:  "system-direct",
			Source: "independent-direct-probe-required",
			Note:   "Transparent TUN is active and the panel process is not UID-bypassed, so no independent direct egress probe is available yet.",
		}
	}

	live := m.runDirectEgressProbeLocked(settings)
	if live.Error == "" {
		saveTunPublicEgressCache(settings, tunPublicEgressCache{
			Version: tunPublicEgressCacheVersion,
			Direct: &tunPublicEgressCacheEntry{
				IP:        live.IP,
				CheckedAt: live.CheckedAt,
			},
		})
		return buildTunLiveDirectEgressObservation(live, "Measured with transparent TUN disabled.")
	}

	if cached := cachedTunDirectEgressObservation(cache.Direct, "The live direct probe failed while transparent TUN was disabled, so the last successful independent direct probe was reused.", live.Error); cached != nil {
		return cached
	}

	return &TunEgressObservation{
		Status: "error",
		Route:  "system-direct",
		Source: "system-http-probe",
		Error:  live.Error,
		Note:   "Transparent TUN is disabled, but the independent direct egress probe failed and no cached value was available.",
	}
}

func buildTunLiveDirectEgressObservation(result tunPublicEgressProbeResult, note string) *TunEgressObservation {
	checkedAt := result.CheckedAt
	return &TunEgressObservation{
		Status:    "live",
		Route:     "system-direct",
		IP:        result.IP,
		CheckedAt: &checkedAt,
		Source:    coalesceTunEgressSource(result.Endpoint, "system-http-probe"),
		Note:      note,
	}
}

func cachedTunDirectEgressObservation(entry *tunPublicEgressCacheEntry, note string, probeError string) *TunEgressObservation {
	if entry == nil || entry.IP == "" || entry.CheckedAt.IsZero() {
		return nil
	}

	checkedAt := entry.CheckedAt
	observation := &TunEgressObservation{
		Status:    "cached",
		Route:     "system-direct",
		IP:        entry.IP,
		CheckedAt: &checkedAt,
		Source:    "cache",
		Note:      note,
		Error:     probeError,
	}
	if time.Since(entry.CheckedAt) > tunPublicEgressCacheTTL {
		observation.Status = "stale"
		observation.Stale = true
	}
	return observation
}

func buildTunProxyEgressObservation(tunRunning bool) *TunEgressObservation {
	observation := &TunEgressObservation{
		Route:  "proxy(node-pool-active)",
		Source: "node-pool-active-balancer",
	}
	if tunRunning {
		observation.Status = "dynamic"
		observation.Note = "Proxy-eligible traffic currently uses the active node-pool balancer and is reported separately from the machine's direct egress."
		return observation
	}

	observation.Status = "inactive"
	observation.Note = "Transparent TUN is disabled, so proxy egress is inactive."
	return observation
}

func formatTunEgressDiagnostic(scope string, observation *TunEgressObservation) string {
	if observation == nil {
		return ""
	}

	parts := []string{fmt.Sprintf("%s egress [%s]", scope, observation.Status)}
	if observation.Route != "" {
		parts = append(parts, "route="+observation.Route)
	}
	if observation.IP != "" {
		parts = append(parts, "ip="+observation.IP)
	}
	if observation.CheckedAt != nil && !observation.CheckedAt.IsZero() {
		parts = append(parts, "checkedAt="+observation.CheckedAt.UTC().Format(time.RFC3339))
	}
	if observation.Source != "" {
		parts = append(parts, "source="+observation.Source)
	}
	if observation.Stale {
		parts = append(parts, "stale=true")
	}
	if observation.Error != "" {
		parts = append(parts, "error="+observation.Error)
	}
	if observation.Note != "" {
		parts = append(parts, "note="+observation.Note)
	}
	return strings.Join(parts, " ")
}

func tunDirectEgressProbeAllowedWhileRunning() bool {
	return os.Geteuid() == 0
}

func (m *TunManager) runDirectEgressProbeLocked(settings *TunFeatureSettings) tunPublicEgressProbeResult {
	if m != nil && m.directEgressProber != nil {
		return m.directEgressProber(settings)
	}
	return defaultTunDirectEgressProber(settings)
}

func defaultTunDirectEgressProber(_ *TunFeatureSettings) tunPublicEgressProbeResult {
	client := &http.Client{Timeout: tunPublicEgressProbeTimeout}
	endpoints := []string{
		"https://api.ipify.org",
		"https://ipv4.icanhazip.com",
		"https://ifconfig.me/ip",
	}

	lastError := ""
	for _, endpoint := range endpoints {
		req, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			lastError = err.Error()
			continue
		}
		req.Header.Set("User-Agent", "xray-webpanel-tun-egress-probe/1")

		resp, err := client.Do(req)
		if err != nil {
			lastError = err.Error()
			continue
		}

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 256))
		resp.Body.Close()
		if readErr != nil {
			lastError = readErr.Error()
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastError = fmt.Sprintf("%s returned HTTP %d", endpoint, resp.StatusCode)
			continue
		}

		candidate := strings.TrimSpace(string(body))
		ip := stdnet.ParseIP(candidate)
		if ip == nil {
			lastError = fmt.Sprintf("%s returned a non-IP body", endpoint)
			continue
		}
		if ip4 := ip.To4(); ip4 != nil {
			candidate = ip4.String()
		} else {
			candidate = ip.String()
		}

		return tunPublicEgressProbeResult{
			IP:        candidate,
			Endpoint:  endpoint,
			CheckedAt: time.Now().UTC(),
		}
	}

	if lastError == "" {
		lastError = "all public-IP probe endpoints failed"
	}
	return tunPublicEgressProbeResult{Error: lastError}
}

func coalesceTunEgressSource(value string, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
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
	if err := writeTunAggregationRuntimeState(settings, activeNodes); err != nil {
		return err
	}

	return nil
}

func writeTunAggregationRuntimeState(settings *TunFeatureSettings, activeNodes []NodeRecord) error {
	if settings == nil {
		return nil
	}

	runtimeState := buildTunAggregationStatus(settings)
	if runtimeState == nil {
		return nil
	}
	attachTunAggregationPrototype(runtimeState, settings, activeNodes, time.Now())
	attachTunAggregationRelayDiagnostics(runtimeState, settings, time.Now())

	raw, err := json.MarshalIndent(runtimeState, "", "  ")
	if err != nil {
		return fmt.Errorf("encode aggregation runtime state: %w", err)
	}

	outputPath := filepath.Join(settings.StateDir, tunAggregationRuntimeFile)
	if err := os.WriteFile(outputPath, raw, 0o644); err != nil {
		return fmt.Errorf("write aggregation runtime state: %w", err)
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

	cmd := execCommandCompat(cmdName, cmdArgs...)
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
	prependRules := make([]interface{}, 0, 5)
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
		prependRules = append(prependRules, map[string]interface{}{
			"type":        "field",
			"inboundTag":  []string{"dns-direct-local"},
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
	bindingRules := buildTunDestinationBindingRules(settings, activeNodes)

	rules := append(prependRules, priorityRules...)
	rules = append(rules, bindingRules...)
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
	servers := make([]interface{}, 0, 1+len(defaultTunChinaDNS)+len(settings.RemoteDNS))
	hasGeosite := runtimeAssetExists(settings, "geosite.dat")
	hasGeoip := runtimeAssetExists(settings, "geoip.dat")
	protectedDomains := uniqStrings(append([]string{}, settings.ProtectDomains...))

	if len(protectedDomains) > 0 {
		servers = append(servers, map[string]interface{}{
			"address":      "localhost",
			"domains":      protectedDomains,
			"skipFallback": true,
			"tag":          "dns-direct-local",
		})
	}

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

func buildTunDestinationBindingRules(settings *TunFeatureSettings, activeNodes []NodeRecord) []interface{} {
	if settings == nil || len(settings.DestinationBindings) == 0 || len(activeNodes) == 0 {
		return nil
	}

	activeNodeMap := make(map[string]NodeRecord, len(activeNodes))
	for _, node := range activeNodes {
		if strings.TrimSpace(node.ID) == "" {
			continue
		}
		activeNodeMap[node.ID] = node
	}

	rules := make([]interface{}, 0, len(settings.DestinationBindings))
	for _, binding := range normalizeTunDestinationBindings(settings.DestinationBindings) {
		targetID := selectTunDestinationBindingNode(binding, activeNodeMap)
		if targetID == "" {
			continue
		}

		domains := tunDestinationBindingDomains(binding)
		if len(domains) == 0 {
			continue
		}

		rules = append(rules, map[string]interface{}{
			"type":        "field",
			"domain":      domains,
			"outboundTag": buildTunOutboundTag(targetID),
		})
	}
	return rules
}

func selectTunDestinationBindingNode(binding TunDestinationBinding, activeNodeMap map[string]NodeRecord) string {
	selectionMode := normalizeTunDestinationBindingSelectionMode(binding.SelectionMode)
	candidateIDs := make([]string, 0, 1+len(binding.FallbackNodeIDs))
	candidateIDs = append(candidateIDs, binding.NodeID)
	candidateIDs = append(candidateIDs, binding.FallbackNodeIDs...)

	switch selectionMode {
	case TunDestinationBindingSelectionModeFailoverFastest:
		candidates := make([]NodeRecord, 0, len(candidateIDs))
		for _, candidateID := range candidateIDs {
			if node, ok := activeNodeMap[candidateID]; ok {
				candidates = append(candidates, node)
			}
		}
		if len(candidates) == 0 {
			return ""
		}
		return bestNodeByFastestPriority(candidates).ID
	case TunDestinationBindingSelectionModeFailoverOrdered:
		fallthrough
	case TunDestinationBindingSelectionModePrimaryOnly:
		for _, candidateID := range candidateIDs {
			if _, ok := activeNodeMap[candidateID]; ok {
				return candidateID
			}
			if selectionMode == TunDestinationBindingSelectionModePrimaryOnly {
				break
			}
		}
		return ""
	default:
		return ""
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

func normalizeTunAggregationMode(value string) TunAggregationMode {
	switch TunAggregationMode(strings.ToLower(strings.TrimSpace(value))) {
	case TunAggregationModeRedundant2:
		return TunAggregationModeRedundant2
	case TunAggregationModeWeightedSplit:
		return TunAggregationModeWeightedSplit
	default:
		return TunAggregationModeSingleBest
	}
}

func normalizeTunAggregationSchedulerPolicy(value string) TunAggregationSchedulerPolicy {
	switch TunAggregationSchedulerPolicy(strings.ToLower(strings.TrimSpace(value))) {
	case TunAggregationSchedulerPolicySingleBest:
		return TunAggregationSchedulerPolicySingleBest
	case TunAggregationSchedulerPolicyRedundant2:
		return TunAggregationSchedulerPolicyRedundant2
	default:
		return TunAggregationSchedulerPolicyWeightedSplit
	}
}

func normalizeTunAggregationHealthSettings(settings TunAggregationHealthSettings) TunAggregationHealthSettings {
	normalized := settings
	if normalized.MaxSessionLossPct <= 0 {
		normalized.MaxSessionLossPct = defaultTunAggregation.Health.MaxSessionLossPct
	}
	if normalized.MaxPathJitterMs <= 0 {
		normalized.MaxPathJitterMs = defaultTunAggregation.Health.MaxPathJitterMs
	}
	if normalized.RollbackOnConsecutiveFailures <= 0 {
		normalized.RollbackOnConsecutiveFailures = defaultTunAggregation.Health.RollbackOnConsecutiveFailures
	}
	return normalized
}

func normalizeTunAggregationSettings(settings TunAggregationSettings) TunAggregationSettings {
	normalized := defaultTunAggregation
	normalized.Enabled = settings.Enabled
	if strings.TrimSpace(settings.Mode) != "" {
		normalized.Mode = string(normalizeTunAggregationMode(settings.Mode))
	}
	if settings.MaxPathsPerSession > 0 {
		normalized.MaxPathsPerSession = settings.MaxPathsPerSession
	}
	if normalized.MaxPathsPerSession < 1 {
		normalized.MaxPathsPerSession = 1
	}
	if normalized.MaxPathsPerSession > 8 {
		normalized.MaxPathsPerSession = 8
	}
	if strings.TrimSpace(settings.SchedulerPolicy) != "" {
		normalized.SchedulerPolicy = string(normalizeTunAggregationSchedulerPolicy(settings.SchedulerPolicy))
	}
	normalized.RelayEndpoint = strings.TrimSpace(settings.RelayEndpoint)
	normalized.Health = normalizeTunAggregationHealthSettings(settings.Health)
	return normalized
}

func tunAggregationSettingsFromAny(raw interface{}) TunAggregationSettings {
	if raw == nil {
		return normalizeTunAggregationSettings(TunAggregationSettings{})
	}

	switch typed := raw.(type) {
	case TunAggregationSettings:
		return normalizeTunAggregationSettings(typed)
	default:
		payload, err := json.Marshal(raw)
		if err != nil {
			return normalizeTunAggregationSettings(TunAggregationSettings{})
		}
		var settings TunAggregationSettings
		if err := json.Unmarshal(payload, &settings); err != nil {
			return normalizeTunAggregationSettings(TunAggregationSettings{})
		}
		return normalizeTunAggregationSettings(settings)
	}
}

func tunAggregationSettingsToAny(settings TunAggregationSettings) map[string]interface{} {
	normalized := normalizeTunAggregationSettings(settings)
	return map[string]interface{}{
		"enabled":            normalized.Enabled,
		"mode":               normalized.Mode,
		"maxPathsPerSession": normalized.MaxPathsPerSession,
		"schedulerPolicy":    normalized.SchedulerPolicy,
		"relayEndpoint":      normalized.RelayEndpoint,
		"health": map[string]interface{}{
			"maxSessionLossPct":             normalized.Health.MaxSessionLossPct,
			"maxPathJitterMs":               normalized.Health.MaxPathJitterMs,
			"rollbackOnConsecutiveFailures": normalized.Health.RollbackOnConsecutiveFailures,
		},
	}
}

func buildTunAggregationStatus(settings *TunFeatureSettings) *TunAggregationStatus {
	if settings == nil {
		return nil
	}

	aggregation := normalizeTunAggregationSettings(settings.Aggregation)
	requestedPath := string(TunAggregationPathStableSinglePath)
	if aggregation.Enabled {
		requestedPath = string(TunAggregationPathExperimentalUDPQUICAggregation)
	}

	status := &TunAggregationStatus{
		Enabled:            aggregation.Enabled,
		Status:             string(TunAggregationStatusDisabled),
		RequestedPath:      requestedPath,
		EffectivePath:      string(TunAggregationPathStableSinglePath),
		Ready:              false,
		RelayConfigured:    false,
		Mode:               aggregation.Mode,
		MaxPathsPerSession: aggregation.MaxPathsPerSession,
		SchedulerPolicy:    aggregation.SchedulerPolicy,
		RelayEndpoint:      aggregation.RelayEndpoint,
		Reason:             "Experimental UDP/QUIC aggregation is disabled, so transparent mode stays on the stable single-path balancer.",
	}

	if !aggregation.Enabled {
		return status
	}

	if aggregation.RelayEndpoint == "" {
		status.Status = string(TunAggregationStatusFallbackStable)
		status.Reason = "Experimental UDP/QUIC aggregation is enabled, but relayEndpoint is empty, so the runtime falls back to stable single-path mode."
		return status
	}

	status.Status = string(TunAggregationStatusRequested)
	status.Ready = true
	status.RelayConfigured = true
	status.Reason = "Experimental UDP/QUIC aggregation is configured behind the feature flag. Local scheduler diagnostics, the relay-side assembler preview, and the synthetic benchmark harness are available, but the effective transparent path stays on stable single-path mode until live packet steering lands."
	return status
}

func formatTunAggregationDiagnostic(status *TunAggregationStatus) string {
	if status == nil {
		return ""
	}

	return fmt.Sprintf(
		"Aggregation mode [%s]: requested=%s effective=%s mode=%s scheduler=%s maxPaths=%d relayConfigured=%t reason=%s",
		status.Status,
		status.RequestedPath,
		status.EffectivePath,
		status.Mode,
		status.SchedulerPolicy,
		status.MaxPathsPerSession,
		status.RelayConfigured,
		status.Reason,
	)
}

func normalizeTunDestinationBindingPreset(value string) TunDestinationBindingPreset {
	switch TunDestinationBindingPreset(strings.ToLower(strings.TrimSpace(value))) {
	case TunDestinationBindingPresetOpenAI:
		return TunDestinationBindingPresetOpenAI
	case TunDestinationBindingPresetChatGPT:
		return TunDestinationBindingPresetChatGPT
	case TunDestinationBindingPresetClaude:
		return TunDestinationBindingPresetClaude
	case TunDestinationBindingPresetGemini:
		return TunDestinationBindingPresetGemini
	case TunDestinationBindingPresetGitHub:
		return TunDestinationBindingPresetGitHub
	case TunDestinationBindingPresetGitHubCopilot:
		return TunDestinationBindingPresetGitHubCopilot
	case TunDestinationBindingPresetOpenRouter:
		return TunDestinationBindingPresetOpenRouter
	case TunDestinationBindingPresetCursor:
		return TunDestinationBindingPresetCursor
	case TunDestinationBindingPresetQwen:
		return TunDestinationBindingPresetQwen
	case TunDestinationBindingPresetPerplexity:
		return TunDestinationBindingPresetPerplexity
	case TunDestinationBindingPresetDeepSeek:
		return TunDestinationBindingPresetDeepSeek
	default:
		return TunDestinationBindingPresetCustom
	}
}

func normalizeTunDestinationBindingSelectionMode(value string) TunDestinationBindingSelectionMode {
	switch TunDestinationBindingSelectionMode(strings.ToLower(strings.TrimSpace(value))) {
	case TunDestinationBindingSelectionModeFailoverOrdered:
		return TunDestinationBindingSelectionModeFailoverOrdered
	case TunDestinationBindingSelectionModeFailoverFastest:
		return TunDestinationBindingSelectionModeFailoverFastest
	default:
		return TunDestinationBindingSelectionModePrimaryOnly
	}
}

func normalizeTunDestinationBindings(bindings []TunDestinationBinding) []TunDestinationBinding {
	result := make([]TunDestinationBinding, 0, len(bindings))
	seen := make(map[string]struct{}, len(bindings))
	for _, binding := range bindings {
		next, ok := normalizeTunDestinationBinding(binding)
		if !ok {
			continue
		}
		key := next.Preset + "|" + next.NodeID + "|" + next.SelectionMode + "|" + strings.Join(next.FallbackNodeIDs, ",") + "|" + strings.Join(next.Domains, ",")
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, next)
	}
	return result
}

func normalizeTunDestinationBinding(binding TunDestinationBinding) (TunDestinationBinding, bool) {
	nodeID := strings.TrimSpace(binding.NodeID)
	if nodeID == "" {
		return TunDestinationBinding{}, false
	}

	preset := normalizeTunDestinationBindingPreset(binding.Preset)
	selectionMode := normalizeTunDestinationBindingSelectionMode(binding.SelectionMode)
	fallbackNodeIDs := make([]string, 0, len(binding.FallbackNodeIDs))
	seenFallbacks := make(map[string]struct{}, len(binding.FallbackNodeIDs))
	for _, candidate := range binding.FallbackNodeIDs {
		next := strings.TrimSpace(candidate)
		if next == "" || next == nodeID {
			continue
		}
		if _, exists := seenFallbacks[next]; exists {
			continue
		}
		seenFallbacks[next] = struct{}{}
		fallbackNodeIDs = append(fallbackNodeIDs, next)
	}
	if selectionMode == TunDestinationBindingSelectionModePrimaryOnly && len(fallbackNodeIDs) > 0 {
		selectionMode = TunDestinationBindingSelectionModeFailoverOrdered
	}

	normalized := TunDestinationBinding{
		Preset:          string(preset),
		NodeID:          nodeID,
		FallbackNodeIDs: fallbackNodeIDs,
		SelectionMode:   string(selectionMode),
	}
	if preset == TunDestinationBindingPresetCustom {
		normalized.Domains = normalizeTunDomainRules(binding.Domains)
		if len(normalized.Domains) == 0 {
			return TunDestinationBinding{}, false
		}
	}

	return normalized, true
}

func tunDestinationBindingDomains(binding TunDestinationBinding) []string {
	preset := normalizeTunDestinationBindingPreset(binding.Preset)
	if preset == TunDestinationBindingPresetCustom {
		return normalizeTunDomainRules(binding.Domains)
	}
	return append([]string{}, tunDestinationBindingPresetDomains[preset]...)
}

func tunDestinationBindingsFromAny(raw interface{}) []TunDestinationBinding {
	switch typed := raw.(type) {
	case []TunDestinationBinding:
		return append([]TunDestinationBinding{}, typed...)
	case []interface{}:
		result := make([]TunDestinationBinding, 0, len(typed))
		for _, item := range typed {
			entry, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			preset, _ := entry["preset"].(string)
			nodeID, _ := entry["nodeId"].(string)
			selectionMode, _ := entry["selectionMode"].(string)
			result = append(result, TunDestinationBinding{
				Preset:          strings.TrimSpace(preset),
				Domains:         stringSliceFromAny(entry["domains"]),
				NodeID:          strings.TrimSpace(nodeID),
				FallbackNodeIDs: stringSliceFromAny(entry["fallbackNodeIds"]),
				SelectionMode:   strings.TrimSpace(selectionMode),
			})
		}
		return result
	default:
		return nil
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

func normalizeTunRemoteDNS(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	return uniqStrings(normalized)
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

func loadTunPublicEgressCache(settings *TunFeatureSettings) tunPublicEgressCache {
	cache := tunPublicEgressCache{Version: tunPublicEgressCacheVersion}
	if settings == nil || strings.TrimSpace(settings.StateDir) == "" {
		return cache
	}

	raw, err := os.ReadFile(filepath.Join(settings.StateDir, tunPublicEgressCacheFile))
	if err != nil {
		return cache
	}
	if err := json.Unmarshal(raw, &cache); err != nil {
		return tunPublicEgressCache{Version: tunPublicEgressCacheVersion}
	}
	if cache.Version != tunPublicEgressCacheVersion {
		cache.Version = tunPublicEgressCacheVersion
		cache.Direct = nil
	}
	return cache
}

func saveTunPublicEgressCache(settings *TunFeatureSettings, cache tunPublicEgressCache) {
	if settings == nil || strings.TrimSpace(settings.StateDir) == "" {
		return
	}
	cache.Version = tunPublicEgressCacheVersion

	if err := os.MkdirAll(settings.StateDir, 0755); err != nil {
		return
	}

	encoded, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(settings.StateDir, tunPublicEgressCacheFile), append(bytes.TrimRight(encoded, "\n"), '\n'), 0644)
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
	if _, ok := resolveCommandPath("sudo"); !ok {
		return false
	}

	cmd := execCommandCompat("sudo", "-n", "-l")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	return sudoListingAllowsTunActionsWithoutPassword(settings, string(output))
}

func sudoListingAllowsTunActionsWithoutPassword(settings *TunFeatureSettings, listing string) bool {
	if settings == nil {
		return strings.Contains(listing, "NOPASSWD:")
	}
	if settings.HelperPath == "" || settings.BinaryPath == "" || settings.RuntimeConfigPath == "" || settings.StateDir == "" {
		return false
	}

	interfaceName := strings.TrimSpace(settings.InterfaceName)
	if interfaceName == "" {
		interfaceName = "xray0"
	}
	remoteDNS := settings.RemoteDNS
	if len(remoteDNS) == 0 {
		remoteDNS = defaultTunDNS
	}

	required := make([]string, 0, 2)
	for _, action := range []string{"start", "stop"} {
		args := []string{settings.HelperPath, action, settings.BinaryPath, settings.RuntimeConfigPath, settings.StateDir, interfaceName}
		args = append(args, remoteDNS...)
		required = append(required, strings.Join(args, " "))
	}

	for _, command := range required {
		if !sudoListingHasNoPasswordCommand(listing, command) {
			return false
		}
	}
	return true
}

func sudoListingHasNoPasswordCommand(listing, expectedCommand string) bool {
	expectedCommand = strings.TrimSpace(expectedCommand)
	if expectedCommand == "" {
		return false
	}

	for _, line := range strings.Split(listing, "\n") {
		for _, section := range strings.Split(line, "NOPASSWD:")[1:] {
			if passwdIndex := strings.Index(section, "PASSWD:"); passwdIndex >= 0 {
				section = section[:passwdIndex]
			}
			for _, command := range strings.Split(section, ",") {
				command = strings.TrimSpace(command)
				if command == "ALL" || command == expectedCommand {
					return true
				}
			}
		}
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

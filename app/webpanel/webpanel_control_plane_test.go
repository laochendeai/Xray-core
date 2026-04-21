package webpanel

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWebPanelStartTransparentModeBlocksWhenPoolBelowMinimum(t *testing.T) {
	t.Parallel()

	wp, paths := newTestControlPlaneWebPanel(t)
	defer wp.subManager.Stop()

	wp.subManager.mu.Lock()
	wp.subManager.state.ValidationConfig.MinActiveNodes = 2
	wp.subManager.mu.Unlock()

	status := wp.startTransparentMode()
	if status == nil {
		t.Fatal("expected tun status")
	}
	if status.Status != "blocked" {
		t.Fatalf("expected blocked status, got %q", status.Status)
	}
	if !strings.Contains(status.Message, "blocked") {
		t.Fatalf("expected blocked message, got %q", status.Message)
	}

	snapshot := wp.controlPlane.Snapshot()
	if snapshot.MachineState != MachineStateClean {
		t.Fatalf("expected machine to remain clean, got %q", snapshot.MachineState)
	}
	if snapshot.LastStateReason != MachineReasonEnableBlockedMinActiveNotMet {
		t.Fatalf("expected enable-block reason, got %q", snapshot.LastStateReason)
	}

	runtimeConfig, err := os.ReadFile(paths.runtimeConfigPath)
	if err == nil && len(runtimeConfig) > 0 {
		t.Fatalf("did not expect runtime config to be written when enablement is blocked: %s", paths.runtimeConfigPath)
	}
}

func TestWebPanelHandlePoolHealthChangeRestoresCleanWhenProxied(t *testing.T) {
	t.Parallel()

	wp, paths := newTestControlPlaneWebPanel(t)
	defer wp.subManager.Stop()

	if err := os.WriteFile(paths.helperStatePath, []byte("running\n"), 0o644); err != nil {
		t.Fatalf("write helper state: %v", err)
	}
	wp.controlPlane.Transition(MachineStateProxied, MachineReasonOperatorEnabled, EventActorOperator, "enabled for test")

	wp.handlePoolHealthChange(PoolHealthSummary{
		ActiveNodes:    0,
		MinActiveNodes: 1,
		Healthy:        false,
	})

	snapshot := wp.controlPlane.Snapshot()
	if snapshot.MachineState != MachineStateClean {
		t.Fatalf("expected clean state after fallback, got %q", snapshot.MachineState)
	}
	if snapshot.LastStateReason != MachineReasonAutomaticFallbackMinActive {
		t.Fatalf("expected automatic fallback reason, got %q", snapshot.LastStateReason)
	}

	helperState, err := os.ReadFile(paths.helperStatePath)
	if err != nil {
		t.Fatalf("read helper state: %v", err)
	}
	if strings.TrimSpace(string(helperState)) != "stopped" {
		t.Fatalf("expected helper state to be stopped, got %q", strings.TrimSpace(string(helperState)))
	}

	status := wp.tunStatusSnapshot()
	if status.Running {
		t.Fatal("expected tun status to be stopped after fallback")
	}
	if status.MachineState != MachineStateClean {
		t.Fatalf("expected decorated machine state clean, got %q", status.MachineState)
	}
}

func TestWebPanelHandlePoolHealthChangeLeavesMachineDegradedWhenFallbackFails(t *testing.T) {
	t.Parallel()

	wp, paths := newTestControlPlaneWebPanel(t)
	defer wp.subManager.Stop()

	if err := os.WriteFile(paths.helperStatePath, []byte("running\n"), 0o644); err != nil {
		t.Fatalf("write helper state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(paths.stateDir, "fail-stop"), []byte("1"), 0o644); err != nil {
		t.Fatalf("write fail-stop marker: %v", err)
	}
	wp.controlPlane.Transition(MachineStateProxied, MachineReasonOperatorEnabled, EventActorOperator, "enabled for test")

	wp.handlePoolHealthChange(PoolHealthSummary{
		ActiveNodes:    0,
		MinActiveNodes: 1,
		Healthy:        false,
	})

	snapshot := wp.controlPlane.Snapshot()
	if snapshot.MachineState != MachineStateDegraded {
		t.Fatalf("expected degraded state after failed fallback, got %q", snapshot.MachineState)
	}
	if snapshot.LastStateReason != MachineReasonFallbackFailed {
		t.Fatalf("expected fallback failed reason, got %q", snapshot.LastStateReason)
	}

	helperState, err := os.ReadFile(paths.helperStatePath)
	if err != nil {
		t.Fatalf("read helper state: %v", err)
	}
	if strings.TrimSpace(string(helperState)) != "running" {
		t.Fatalf("expected helper state to remain running, got %q", strings.TrimSpace(string(helperState)))
	}
}

func TestWebPanelStartTransparentModeBlocksWhenEligiblePoolBelowMinimumEvenIfActivePoolMeetsIt(t *testing.T) {
	t.Parallel()

	wp, paths := newTestControlPlaneWebPanel(t)
	defer wp.subManager.Stop()

	now := time.Now()
	wp.subManager.mu.Lock()
	wp.subManager.state.ValidationConfig.MinActiveNodes = 2
	wp.subManager.state.Nodes = []NodeRecord{
		testTransparentNodeRecord(t, "node-good", "vmess", &now, 120, 0),
		testTransparentNodeRecord(t, "node-hy2", "hysteria2", &now, 80, 0),
	}
	wp.subManager.mu.Unlock()

	status := wp.startTransparentMode()
	if status == nil {
		t.Fatal("expected tun status")
	}
	if status.Status != "blocked" {
		t.Fatalf("expected blocked status, got %q", status.Status)
	}
	if !strings.Contains(status.Message, "stable eligible pool") {
		t.Fatalf("expected stable-pool message, got %q", status.Message)
	}

	diagnostics := strings.Join(status.Diagnostics, "\n")
	for _, token := range []string{
		"Stable mode captures TCP plus UDP/53 and UDP/443",
		"unmanaged WebRTC/STUN is not guaranteed",
		"Transparent-mode eligible nodes: 1 / active 2 / minimum required 2.",
		"Excluded active nodes: hysteria2=1",
		"Transparent mode only starts when the stable eligible pool meets the minimum size.",
	} {
		if !strings.Contains(diagnostics, token) {
			t.Fatalf("expected diagnostics to contain %q\n%s", token, diagnostics)
		}
	}

	snapshot := wp.controlPlane.Snapshot()
	if snapshot.MachineState != MachineStateClean {
		t.Fatalf("expected machine to remain clean, got %q", snapshot.MachineState)
	}
	if snapshot.LastStateReason != MachineReasonEnableBlockedMinActiveNotMet {
		t.Fatalf("expected enable-block reason, got %q", snapshot.LastStateReason)
	}

	runtimeConfig, err := os.ReadFile(paths.runtimeConfigPath)
	if err == nil && len(runtimeConfig) > 0 {
		t.Fatalf("did not expect runtime config to be written when eligible enablement is blocked: %s", paths.runtimeConfigPath)
	}
}

func TestWebPanelStartTransparentModeTreatsAlreadyRunningTunAsSuccess(t *testing.T) {
	t.Parallel()

	wp, paths := newTestControlPlaneWebPanel(t)
	defer wp.subManager.Stop()

	if err := os.WriteFile(paths.helperStatePath, []byte("running\n"), 0o644); err != nil {
		t.Fatalf("write helper state: %v", err)
	}

	now := time.Now()
	wp.subManager.mu.Lock()
	wp.subManager.state.ValidationConfig.MinActiveNodes = 1
	wp.subManager.state.Nodes = []NodeRecord{
		testTransparentNodeRecord(t, "node-good", "vmess", &now, 120, 0),
	}
	wp.subManager.mu.Unlock()

	status := wp.startTransparentMode()
	if status == nil {
		t.Fatal("expected tun status")
	}
	if status.Status != "running" {
		t.Fatalf("expected running status, got %q", status.Status)
	}
	if !status.Running {
		t.Fatal("expected running=true when transparent TUN is already active")
	}
	if status.Message != "Transparent TUN mode is enabled" {
		t.Fatalf("unexpected message: %q", status.Message)
	}

	snapshot := wp.controlPlane.Snapshot()
	if snapshot.MachineState != MachineStateProxied {
		t.Fatalf("expected proxied state after start request, got %q", snapshot.MachineState)
	}
	if snapshot.LastStateReason != MachineReasonOperatorEnabled {
		t.Fatalf("expected operator enabled reason, got %q", snapshot.LastStateReason)
	}

	if _, err := os.Stat(paths.runtimeConfigPath); !os.IsNotExist(err) {
		t.Fatalf("did not expect runtime config rewrite when TUN is already running, got err=%v", err)
	}
}

func TestWebPanelStartTransparentModeKeepsMachineCleanWhenPrivilegeRepairIsRequired(t *testing.T) {
	t.Parallel()

	wp, paths := newTestControlPlaneWebPanel(t)
	defer wp.subManager.Stop()

	staleBinaryPath := filepath.Join(t.TempDir(), "stale-xray")
	if err := os.WriteFile(staleBinaryPath, []byte("stale-xray-binary"), 0o755); err != nil {
		t.Fatalf("write stale binary: %v", err)
	}

	rawConfig, err := os.ReadFile(wp.tunManager.configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	var config map[string]any
	if err := json.Unmarshal(rawConfig, &config); err != nil {
		t.Fatalf("parse config: %v", err)
	}

	webpanelConfig, _ := config["webpanel"].(map[string]any)
	tunConfig, _ := webpanelConfig["tun"].(map[string]any)
	tunConfig["binaryPath"] = staleBinaryPath

	rawConfig, err = json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(wp.tunManager.configPath, rawConfig, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	now := time.Now()
	wp.subManager.mu.Lock()
	wp.subManager.state.ValidationConfig.MinActiveNodes = 1
	wp.subManager.state.Nodes = []NodeRecord{
		testTransparentNodeRecord(t, "node-good", "vmess", &now, 120, 0),
	}
	wp.subManager.mu.Unlock()

	status := wp.startTransparentMode()
	if status == nil {
		t.Fatal("expected tun status")
	}
	if status.Status != "blocked" {
		t.Fatalf("expected blocked status, got %q", status.Status)
	}
	if !strings.Contains(status.Message, "Install or repair") {
		t.Fatalf("unexpected blocked message: %q", status.Message)
	}
	if status.Running {
		t.Fatal("did not expect running=true when privilege repair is still required")
	}

	snapshot := wp.controlPlane.Snapshot()
	if snapshot.MachineState != MachineStateClean {
		t.Fatalf("expected machine to remain clean, got %q", snapshot.MachineState)
	}
	if snapshot.LastStateReason != MachineReasonStartupDefaultClean {
		t.Fatalf("expected startup clean reason to remain, got %q", snapshot.LastStateReason)
	}

	if _, err := os.Stat(paths.runtimeConfigPath); !os.IsNotExist(err) {
		t.Fatalf("did not expect runtime config write when privilege repair blocks startup, got err=%v", err)
	}
}

func TestWebPanelEligibleTransparentNodesExcludesUnstableActiveNodes(t *testing.T) {
	t.Parallel()

	wp, _ := newTestControlPlaneWebPanel(t)
	defer wp.subManager.Stop()

	now := time.Now()
	stale := now.Add(-11 * time.Minute)

	wp.subManager.mu.Lock()
	wp.subManager.state.Nodes = []NodeRecord{
		testTransparentNodeRecord(t, "good", "vmess", &now, 80, 0),
		testTransparentNodeRecord(t, "hy2", "hysteria2", &now, 70, 0),
		testTransparentNodeRecord(t, "failing", "vmess", &now, 90, 1),
		testTransparentNodeRecord(t, "missing-delay", "vmess", &now, 0, 0),
		testTransparentNodeRecord(t, "stale", "vmess", &stale, 110, 0),
		testTransparentNodeRecord(t, "unchecked", "vmess", nil, 130, 0),
	}
	wp.subManager.mu.Unlock()

	eligible, summary := wp.eligibleTransparentNodes()
	if len(eligible) != 1 {
		t.Fatalf("expected exactly one eligible node, got %d", len(eligible))
	}
	if eligible[0].ID != "good" {
		t.Fatalf("expected good node to remain eligible, got %q", eligible[0].ID)
	}

	if summary.ActiveNodes != 6 {
		t.Fatalf("expected 6 active nodes in summary, got %d", summary.ActiveNodes)
	}
	if summary.EligibleNodes != 1 {
		t.Fatalf("expected 1 eligible node in summary, got %d", summary.EligibleNodes)
	}
	if summary.ExcludedProtocol != 1 {
		t.Fatalf("expected one protocol exclusion, got %d", summary.ExcludedProtocol)
	}
	if summary.ExcludedConsecutiveFails != 1 {
		t.Fatalf("expected one consecutive-fails exclusion, got %d", summary.ExcludedConsecutiveFails)
	}
	if summary.ExcludedMissingDelay != 1 {
		t.Fatalf("expected one missing-delay exclusion, got %d", summary.ExcludedMissingDelay)
	}
	if summary.ExcludedUncheckedOrStale != 2 {
		t.Fatalf("expected two unchecked-or-stale exclusions, got %d", summary.ExcludedUncheckedOrStale)
	}
}

func TestWebPanelHandlePoolHealthChangeRestoresCleanWhenEligiblePoolDropsBelowMinimum(t *testing.T) {
	t.Parallel()

	wp, paths := newTestControlPlaneWebPanel(t)
	defer wp.subManager.Stop()

	if err := os.WriteFile(paths.helperStatePath, []byte("running\n"), 0o644); err != nil {
		t.Fatalf("write helper state: %v", err)
	}

	now := time.Now()
	wp.subManager.mu.Lock()
	wp.subManager.state.ValidationConfig.MinActiveNodes = 2
	wp.subManager.state.Nodes = []NodeRecord{
		testTransparentNodeRecord(t, "node-good", "vmess", &now, 120, 0),
		testTransparentNodeRecord(t, "node-hy2", "hysteria2", &now, 80, 0),
	}
	wp.subManager.mu.Unlock()

	wp.controlPlane.Transition(MachineStateProxied, MachineReasonOperatorEnabled, EventActorOperator, "enabled for test")

	wp.handlePoolHealthChange(PoolHealthSummary{
		ActiveNodes:    2,
		MinActiveNodes: 2,
		Healthy:        true,
	})

	snapshot := wp.controlPlane.Snapshot()
	if snapshot.MachineState != MachineStateClean {
		t.Fatalf("expected clean state after eligible-pool fallback, got %q", snapshot.MachineState)
	}
	if snapshot.LastStateReason != MachineReasonAutomaticFallbackMinActive {
		t.Fatalf("expected automatic fallback reason, got %q", snapshot.LastStateReason)
	}

	helperState, err := os.ReadFile(paths.helperStatePath)
	if err != nil {
		t.Fatalf("read helper state: %v", err)
	}
	if strings.TrimSpace(string(helperState)) != "stopped" {
		t.Fatalf("expected helper state to be stopped, got %q", strings.TrimSpace(string(helperState)))
	}
}

func TestWebPanelEnsureCleanStartupStateStopsRunningTun(t *testing.T) {
	t.Parallel()

	wp, paths := newTestControlPlaneWebPanel(t)
	defer wp.subManager.Stop()

	if err := os.WriteFile(paths.helperStatePath, []byte("running\n"), 0o644); err != nil {
		t.Fatalf("write helper state: %v", err)
	}

	wp.ensureCleanStartupState()

	snapshot := wp.controlPlane.Snapshot()
	if snapshot.MachineState != MachineStateClean {
		t.Fatalf("expected clean machine state after startup enforcement, got %q", snapshot.MachineState)
	}
	if snapshot.LastStateReason != MachineReasonStartupDefaultClean {
		t.Fatalf("expected startup clean reason, got %q", snapshot.LastStateReason)
	}

	helperState, err := os.ReadFile(paths.helperStatePath)
	if err != nil {
		t.Fatalf("read helper state: %v", err)
	}
	if strings.TrimSpace(string(helperState)) != "stopped" {
		t.Fatalf("expected helper state to be stopped after startup enforcement, got %q", strings.TrimSpace(string(helperState)))
	}
}

func TestTunStatusSnapshotIncludesRoutingDiagnostics(t *testing.T) {
	t.Parallel()

	wp, paths := newTestControlPlaneWebPanel(t)
	defer wp.subManager.Stop()

	configRaw, err := os.ReadFile(wp.config.ConfigPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var config map[string]any
	if err := json.Unmarshal(configRaw, &config); err != nil {
		t.Fatalf("parse config: %v", err)
	}
	webpanel := config["webpanel"].(map[string]any)
	tun := webpanel["tun"].(map[string]any)
	binPath := filepath.Join(filepath.Dir(wp.config.ConfigPath), "xray-test")
	tun["binaryPath"] = binPath
	updatedRaw, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(wp.config.ConfigPath, updatedRaw, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(binPath, []byte("stub"), 0o755); err != nil {
		t.Fatalf("write binary stub: %v", err)
	}

	// Ensure geosite-based CN direct diagnostics are included for the active binary-path lookup.
	assetPath := filepath.Join(filepath.Dir(binPath), "geosite.dat")
	if err := os.WriteFile(assetPath, []byte("stub"), 0o644); err != nil {
		t.Fatalf("write geosite asset: %v", err)
	}

	status := wp.tunStatusSnapshot()
	if status == nil {
		t.Fatal("expected tun status")
	}

	var protected *TunRoutingDiagnostic
	var cn *TunRoutingDiagnostic
	var remote *TunRoutingDiagnostic

	for i := range status.RoutingDiagnostics {
		item := &status.RoutingDiagnostics[i]
		switch item.Category {
		case "protected_direct_domains":
			protected = item
		case "cn_direct_domains":
			cn = item
		case "default_proxy_domains":
			remote = item
		}
	}

	if protected == nil {
		t.Fatal("expected protected_direct_domains routing diagnostic")
	}
	if protected.DNSPath != "dns-direct-local" || protected.Resolver != "localhost" || protected.Route != "direct" {
		t.Fatalf("unexpected protected direct diagnostic: %#v", protected)
	}
	if len(protected.Domains) == 0 || protected.Domains[0] != "full:localhost" {
		t.Fatalf("expected protected domains to include full:localhost, got %#v", protected.Domains)
	}

	if cn == nil {
		t.Fatal("expected cn_direct_domains routing diagnostic")
	}
	if cn.DNSPath != "dns-cn" || cn.Route != "direct" {
		t.Fatalf("unexpected cn direct diagnostic: %#v", cn)
	}
	if !strings.Contains(cn.Resolver, "223.5.5.5") {
		t.Fatalf("expected cn resolver list, got %q", cn.Resolver)
	}

	if remote == nil {
		t.Fatal("expected default_proxy_domains routing diagnostic")
	}
	if remote.DNSPath != "dns-remote" || !strings.Contains(remote.Route, "node-pool-active") {
		t.Fatalf("unexpected remote routing diagnostic: %#v", remote)
	}
	if remote.Resolver != "1.1.1.1" {
		t.Fatalf("expected remote resolver from tun settings, got %q", remote.Resolver)
	}

	diagnostics := strings.Join(status.Diagnostics, "\n")
	for _, token := range []string{
		"DNS/routing decision [protected_direct_domains]",
		"dns=dns-direct-local",
		"DNS/routing decision [cn_direct_domains]",
		"dns=dns-cn",
		"DNS/routing decision [default_proxy_domains]",
		"dns=dns-remote",
	} {
		if !strings.Contains(diagnostics, token) {
			t.Fatalf("expected diagnostics to contain %q\n%s", token, diagnostics)
		}
	}

	if _, err := os.Stat(paths.runtimeConfigPath); err == nil {
		// This test only inspects status diagnostics, no runtime config write is expected.
		t.Fatalf("did not expect runtime config write for diagnostic snapshot test: %s", paths.runtimeConfigPath)
	}
}

func TestTunStatusSnapshotIncludesAggregationPrototype(t *testing.T) {
	t.Parallel()

	wp, _ := newTestControlPlaneWebPanel(t)
	defer wp.subManager.Stop()

	checkedAt := time.Now().Add(-1 * time.Minute)
	wp.subManager.mu.Lock()
	wp.subManager.state.Nodes = []NodeRecord{{
		ID:            "node-1",
		URI:           mustGenerateTunTestURI(t, "203.0.113.81"),
		Remark:        "node-1",
		Protocol:      "vmess",
		Address:       "203.0.113.81",
		Port:          443,
		Status:        NodeStatusActive,
		AddedAt:       time.Now(),
		AvgDelayMs:    32,
		TotalPings:    10,
		LastCheckedAt: &checkedAt,
	}}
	wp.subManager.mu.Unlock()

	config := map[string]any{}
	raw, err := os.ReadFile(wp.config.ConfigPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatalf("decode config: %v", err)
	}
	webpanel := config["webpanel"].(map[string]any)
	tun := webpanel["tun"].(map[string]any)
	tun["aggregation"] = map[string]any{
		"enabled":            true,
		"mode":               "single_best",
		"maxPathsPerSession": 2,
		"schedulerPolicy":    "single_best",
		"relayEndpoint":      "https://relay.example/ingress",
	}
	updatedRaw, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("encode config: %v", err)
	}
	if err := os.WriteFile(wp.config.ConfigPath, updatedRaw, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	status := wp.tunStatusSnapshot()
	if status == nil || status.Aggregation == nil || status.Aggregation.Prototype == nil {
		t.Fatalf("expected aggregation prototype in status snapshot, got %#v", status)
	}
	if status.Aggregation.Prototype.CandidatePathCount != 1 {
		t.Fatalf("expected one candidate path, got %#v", status.Aggregation.Prototype)
	}
	if status.Aggregation.Prototype.SelectedPathCount != 1 {
		t.Fatalf("expected one selected path, got %#v", status.Aggregation.Prototype)
	}
	if status.Aggregation.Relay == nil || !status.Aggregation.Relay.Ready {
		t.Fatalf("expected aggregation relay preview in status snapshot, got %#v", status.Aggregation)
	}
	if status.Aggregation.Benchmark == nil || !status.Aggregation.Benchmark.Ready {
		t.Fatalf("expected aggregation benchmark preview in status snapshot, got %#v", status.Aggregation)
	}

	diagnostics := strings.Join(status.Diagnostics, "\n")
	if !strings.Contains(diagnostics, "Aggregation prototype: candidates=1 selected=1 sessions=1") {
		t.Fatalf("expected prototype diagnostic in status output\n%s", diagnostics)
	}
	if !strings.Contains(diagnostics, "Aggregation relay prototype: sessions=1") {
		t.Fatalf("expected relay diagnostic in status output\n%s", diagnostics)
	}
	if !strings.Contains(diagnostics, "Aggregation benchmark [degraded_primary]") {
		t.Fatalf("expected benchmark diagnostic in status output\n%s", diagnostics)
	}
}

type controlPlaneTestPaths struct {
	stateDir          string
	helperStatePath   string
	runtimeConfigPath string
}

func newTestControlPlaneWebPanel(t *testing.T) (*WebPanel, controlPlaneTestPaths) {
	t.Helper()

	tempDir := t.TempDir()
	stateDir := filepath.Join(tempDir, "runtime", "tun")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("create state dir: %v", err)
	}

	helperPath := filepath.Join(tempDir, "webpanel-tun-helper.sh")
	writeTestTunHelper(t, helperPath)

	configPath := filepath.Join(tempDir, "config.json")
	runtimeConfigPath := filepath.Join(stateDir, "config.json")
	config := map[string]any{
		"outbounds": []map[string]any{
			{
				"tag":      "direct",
				"protocol": "freedom",
			},
		},
		"webpanel": map[string]any{
			"tun": map[string]any{
				"binaryPath":        "/bin/true",
				"helperPath":        helperPath,
				"stateDir":          stateDir,
				"runtimeConfigPath": runtimeConfigPath,
				"interfaceName":     "xray0",
				"remoteDns":         []string{"1.1.1.1"},
				"useSudo":           false,
			},
		},
	}
	raw, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, raw, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	tunManager, err := NewTunManager(configPath)
	if err != nil {
		t.Fatalf("new tun manager: %v", err)
	}

	subManager := NewSubscriptionManager(configPath, nil, nil, nil)
	subManager.SetPoolHealthCallback(nil)

	wp := &WebPanel{
		config:       &Config{ConfigPath: configPath},
		subManager:   subManager,
		tunManager:   tunManager,
		controlPlane: NewControlPlaneStateStore(configPath),
	}

	return wp, controlPlaneTestPaths{
		stateDir:          stateDir,
		helperStatePath:   filepath.Join(stateDir, "helper.state"),
		runtimeConfigPath: runtimeConfigPath,
	}
}

func writeTestTunHelper(t *testing.T, helperPath string) {
	t.Helper()

	const script = `#!/bin/sh
set -eu

action="${1:-}"
state_dir="${4:-}"
state_file="${state_dir}/helper.state"
fail_stop="${state_dir}/fail-stop"

mkdir -p "${state_dir}"

case "${action}" in
  status)
    if [ -f "${state_file}" ] && [ "$(cat "${state_file}")" = "running" ]; then
      echo "ACTION=status:running"
    else
      echo "ACTION=status:stopped"
    fi
    ;;
  start)
    echo "running" > "${state_file}"
    echo "ACTION=start:running"
    ;;
  stop)
    if [ -f "${fail_stop}" ]; then
      echo "ACTION=stop:failed"
      exit 1
    fi
    echo "stopped" > "${state_file}"
    echo "ACTION=stop:stopped"
    ;;
  *)
    echo "ACTION=${action}:unsupported"
    exit 1
    ;;
esac
`

	if err := os.WriteFile(helperPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write helper script: %v", err)
	}
}

func testTransparentNodeRecord(t *testing.T, id, protocol string, lastCheckedAt *time.Time, avgDelay int64, consecutiveFails int) NodeRecord {
	t.Helper()

	req := ShareLinkRequest{
		Protocol: protocol,
		Address:  "203.0.113.31",
		Port:     443,
		Remark:   id,
		SNI:      "example.com",
	}

	switch protocol {
	case "hysteria2":
		req.Password = "secret"
		req.ALPN = "h3"
		req.AllowInsecure = true
	default:
		req.UUID = "11111111-1111-1111-1111-111111111111"
		req.TLS = "tls"
	}

	uri, err := GenerateShareLink(req)
	if err != nil {
		t.Fatalf("generate %s share link: %v", protocol, err)
	}

	return NodeRecord{
		ID:               id,
		URI:              uri,
		Remark:           id,
		Protocol:         protocol,
		Address:          req.Address,
		Port:             req.Port,
		Status:           NodeStatusActive,
		AddedAt:          time.Now(),
		AvgDelayMs:       avgDelay,
		ConsecutiveFails: consecutiveFails,
		LastCheckedAt:    lastCheckedAt,
	}
}

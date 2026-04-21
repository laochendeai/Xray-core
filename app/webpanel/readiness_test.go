package webpanel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestWebPanelReadinessSnapshotBlocksWhenConfigPathMissing(t *testing.T) {
	t.Parallel()

	wp := &WebPanel{
		config:         &Config{},
		releaseChecker: cachedReleaseChecker(UpdateStatusResponse{CurrentVersion: "1.0.0", Source: "test", Status: "ok"}),
	}

	response := wp.readinessSnapshot(context.Background())
	if response.Healthy {
		t.Fatal("expected readiness to be unhealthy when config path is missing")
	}
	if response.BlockingCount == 0 {
		t.Fatal("expected at least one blocking check")
	}

	check, ok := readinessCheckByKey(response.Checks, "config_path")
	if !ok {
		t.Fatal("expected config_path readiness check")
	}
	if check.Severity != ReadinessSeverityBlocking {
		t.Fatalf("expected config_path to be blocking, got %q", check.Severity)
	}
	if status, _ := check.Facts["status"].(string); status != "missing" {
		t.Fatalf("expected missing config-path status, got %#v", check.Facts["status"])
	}
}

func TestWebPanelReadinessSnapshotSummarizesConfiguredState(t *testing.T) {
	t.Parallel()

	wp, _ := newTestControlPlaneWebPanel(t)
	defer wp.subManager.Stop()

	configPath := wp.config.ConfigPath
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatalf("parse config: %v", err)
	}
	config["api"] = map[string]interface{}{
		"tag":      "api",
		"listen":   "127.0.0.1:10085",
		"services": []string{"HandlerService", "StatsService", "LoggerService", "RoutingService", "ObservatoryService"},
	}
	config["stats"] = map[string]interface{}{}
	config["policy"] = map[string]interface{}{
		"system": map[string]interface{}{
			"statsInboundUplink":    true,
			"statsInboundDownlink":  true,
			"statsOutboundUplink":   true,
			"statsOutboundDownlink": true,
		},
	}

	updatedRaw, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, updatedRaw, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	now := time.Now()
	wp.subManager.mu.Lock()
	wp.subManager.started = true
	wp.subManager.dispatcherAvailable = true
	wp.subManager.state.Subscriptions = []SubscriptionRecord{
		{
			ID:         "sub-1",
			SourceType: SubscriptionSourceURL,
			URL:        "https://example.com/sub",
			Remark:     "test",
		},
	}
	wp.subManager.state.ValidationConfig.MinActiveNodes = 1
	wp.subManager.state.Nodes = []NodeRecord{
		testTransparentNodeRecord(t, "ready-node", "vmess", &now, 90, 0),
	}
	wp.subManager.prober = &NodeProber{
		probeURL: "https://www.gstatic.com/generate_204",
		interval: 45 * time.Second,
		tags: map[string]struct{}{
			"pool_ready-node": {},
		},
		running: true,
	}
	wp.subManager.mu.Unlock()
	wp.releaseChecker = cachedReleaseChecker(UpdateStatusResponse{
		CurrentVersion:  "1.0.0",
		LatestVersion:   "1.0.0",
		Source:          "test",
		Status:          "ok",
		UpdateAvailable: false,
	})

	response := wp.readinessSnapshot(context.Background())
	if response.BlockingCount != 0 {
		t.Fatalf("expected no blocking checks, got %d", response.BlockingCount)
	}

	configSections, ok := readinessCheckByKey(response.Checks, "config_sections")
	if !ok {
		t.Fatal("expected config_sections check")
	}
	if configSections.Severity != ReadinessSeverityOK {
		t.Fatalf("expected config sections to be ok, got %q", configSections.Severity)
	}

	probing, ok := readinessCheckByKey(response.Checks, "probing")
	if !ok {
		t.Fatal("expected probing check")
	}
	if probing.Severity != ReadinessSeverityOK {
		t.Fatalf("expected probing check to be ok, got %q", probing.Severity)
	}
	if running, _ := probing.Facts["running"].(bool); !running {
		t.Fatalf("expected probing to be running, got %#v", probing.Facts["running"])
	}

	subscriptions, ok := readinessCheckByKey(response.Checks, "subscriptions")
	if !ok {
		t.Fatal("expected subscriptions check")
	}
	if subscriptions.Severity != ReadinessSeverityOK {
		t.Fatalf("expected subscriptions check to be ok, got %q", subscriptions.Severity)
	}
}

func TestWebPanelReadinessSnapshotSkipsDirectEgressProbe(t *testing.T) {
	t.Parallel()

	wp, _ := newTestControlPlaneWebPanel(t)
	defer wp.subManager.Stop()

	called := false
	wp.tunManager.directEgressProber = func(*TunFeatureSettings) tunPublicEgressProbeResult {
		called = true
		return tunPublicEgressProbeResult{Error: "readiness should not trigger direct egress probe"}
	}
	wp.releaseChecker = cachedReleaseChecker(UpdateStatusResponse{
		CurrentVersion: "1.0.0",
		Source:         "test",
		Status:         "ok",
	})

	response := wp.readinessSnapshot(context.Background())
	if called {
		t.Fatal("expected readiness snapshot to skip direct egress probing")
	}

	tunCheck, ok := readinessCheckByKey(response.Checks, "tun")
	if !ok {
		t.Fatal("expected tun readiness check")
	}
	if _, ok := tunCheck.Facts["status"].(string); !ok {
		t.Fatalf("expected tun readiness facts to include status, got %#v", tunCheck.Facts)
	}
}

func TestWebPanelReadinessUpdatesCheckUsesCacheOnly(t *testing.T) {
	transport := &countingRoundTripper{}
	wp := &WebPanel{
		config: &Config{},
		releaseChecker: newReleaseChecker(
			&http.Client{Transport: transport},
			"https://example.invalid/releases.atom",
			"test/source",
			time.Hour,
		),
	}

	check := wp.readinessUpdatesCheck(context.Background())
	if transport.count != 0 {
		t.Fatalf("expected readiness update check to avoid outbound release fetches, got %d request(s)", transport.count)
	}
	if check.Severity != ReadinessSeverityWarning {
		t.Fatalf("expected unavailable update cache to be warning, got %q", check.Severity)
	}
	if status, _ := check.Facts["status"].(string); status != "unavailable" {
		t.Fatalf("expected unavailable update status, got %#v", check.Facts["status"])
	}
	if source, _ := check.Facts["source"].(string); source != "test/source" {
		t.Fatalf("expected source to round-trip, got %#v", check.Facts["source"])
	}
}

func cachedReleaseChecker(status UpdateStatusResponse) *releaseChecker {
	now := time.Date(2026, 4, 4, 4, 0, 0, 0, time.UTC)
	return &releaseChecker{
		source:   status.Source,
		cacheTTL: time.Hour,
		now:      func() time.Time { return now },
		cached:   &status,
		cachedAt: now,
	}
}

type countingRoundTripper struct {
	count int
}

func (rt *countingRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	rt.count++
	return nil, fmt.Errorf("unexpected release fetch")
}

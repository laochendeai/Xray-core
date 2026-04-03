package webpanel

import (
	"context"
	"encoding/json"
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

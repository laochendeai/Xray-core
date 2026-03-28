package webpanel

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSubscriptionManagerMigratesLegacyDemotedNodes(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	statePath := filepath.Join(tempDir, "node_pool_state.json")
	uri, err := GenerateShareLink(ShareLinkRequest{
		Protocol: "vmess",
		Address:  "example.com",
		Port:     443,
		UUID:     "11111111-1111-1111-1111-111111111111",
		Remark:   "legacy",
		TLS:      "tls",
		SNI:      "example.com",
	})
	if err != nil {
		t.Fatalf("generate share link: %v", err)
	}

	legacy := map[string]any{
		"nodes": []map[string]any{
			{
				"id":             "abc123",
				"uri":            uri,
				"remark":         "legacy node",
				"protocol":       "vmess",
				"address":        "example.com",
				"port":           443,
				"outboundTag":    "active_abc123",
				"status":         "demoted",
				"subscriptionId": "sub-1",
				"addedAt":        time.Now().UTC().Format(time.RFC3339),
			},
		},
		"validationConfig": map[string]any{
			"minSamples":       10,
			"maxFailRate":      0.3,
			"maxAvgDelayMs":    1000,
			"probeIntervalSec": 60,
			"probeUrl":         "https://www.gstatic.com/generate_204",
			"demoteAfterFails": 5,
		},
	}

	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("marshal legacy state: %v", err)
	}
	if err := os.WriteFile(statePath, data, 0o644); err != nil {
		t.Fatalf("write legacy state: %v", err)
	}

	sm := NewSubscriptionManager(configPath, nil, nil, nil)
	defer sm.Stop()

	nodes := sm.ListNodes("")
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node after migration, got %d", len(nodes))
	}
	if nodes[0].Status != NodeStatusQuarantine {
		t.Fatalf("expected legacy demoted node to migrate to quarantine, got %q", nodes[0].Status)
	}
	if nodes[0].StatusReason != TransitionReasonMigrationLegacyDemoted {
		t.Fatalf("expected migration reason, got %q", nodes[0].StatusReason)
	}
	if nodes[0].OutboundTag != probeOutboundTag(nodes[0].ID) {
		t.Fatalf("expected migrated probe tag %q, got %q", probeOutboundTag(nodes[0].ID), nodes[0].OutboundTag)
	}
}

func TestSubscriptionManagerHandleProbeResultsUpdatesLifecycle(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	sm := NewSubscriptionManager(configPath, nil, nil, nil)
	defer sm.Stop()

	stagingID := "staging-1"
	activeID := "active-1"
	sm.mu.Lock()
	sm.state.Nodes = []NodeRecord{
		{
			ID:              stagingID,
			Remark:          "staging",
			Address:         "staging.example.com",
			Port:            443,
			Status:          NodeStatusStaging,
			StatusReason:    TransitionReasonSubscriptionNodeDiscovered,
			OutboundTag:     probeOutboundTag(stagingID),
			AddedAt:         time.Now().Add(-time.Minute),
			StatusUpdatedAt: timePtr(time.Now().Add(-time.Minute)),
			Cleanliness:     CleanlinessUnknown,
			BandwidthTier:   BandwidthTierUnknown,
		},
		{
			ID:              activeID,
			Remark:          "active",
			Address:         "active.example.com",
			Port:            443,
			Status:          NodeStatusActive,
			StatusReason:    TransitionReasonManualPromote,
			OutboundTag:     probeOutboundTag(activeID),
			AddedAt:         time.Now().Add(-time.Minute),
			StatusUpdatedAt: timePtr(time.Now().Add(-time.Minute)),
			Cleanliness:     CleanlinessUnknown,
			BandwidthTier:   BandwidthTierUnknown,
		},
	}
	sm.state.ValidationConfig = ValidationConfig{
		MinSamples:       1,
		MaxFailRate:      0.5,
		MaxAvgDelayMs:    500,
		ProbeIntervalSec: 60,
		ProbeURL:         "https://www.gstatic.com/generate_204",
		DemoteAfterFails: 1,
		MinActiveNodes:   1,
	}
	sm.mu.Unlock()

	sm.handleProbeResults([]ProbeResult{
		{Tag: probeOutboundTag(stagingID), Success: true, DelayMs: 120},
		{Tag: probeOutboundTag(activeID), Success: false, DelayMs: 0},
	})

	nodes := sm.ListNodes("")
	statusByID := map[string]NodeStatus{}
	for _, node := range nodes {
		statusByID[node.ID] = node.Status
	}

	if statusByID[stagingID] != NodeStatusActive {
		t.Fatalf("expected staging node to become active, got %q", statusByID[stagingID])
	}
	if statusByID[activeID] != NodeStatusQuarantine {
		t.Fatalf("expected active node to become quarantine, got %q", statusByID[activeID])
	}

	summary := sm.GetPoolSummary()
	if summary.ActiveCount != 1 {
		t.Fatalf("expected 1 active node in summary, got %d", summary.ActiveCount)
	}
	if summary.QuarantineCount != 1 {
		t.Fatalf("expected 1 quarantined node in summary, got %d", summary.QuarantineCount)
	}
}

func TestSubscriptionManagerDoesNotRequalifyQuarantineNodeOnFailedProbe(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	sm := NewSubscriptionManager(configPath, nil, nil, nil)
	defer sm.Stop()

	quarantineID := "quarantine-1"
	sm.mu.Lock()
	sm.state.Nodes = []NodeRecord{
		{
			ID:               quarantineID,
			Remark:           "quarantine",
			Address:          "quarantine.example.com",
			Port:             443,
			Status:           NodeStatusQuarantine,
			StatusReason:     TransitionReasonProbeFailuresExceeded,
			OutboundTag:      probeOutboundTag(quarantineID),
			AddedAt:          time.Now().Add(-time.Minute),
			StatusUpdatedAt:  timePtr(time.Now().Add(-time.Minute)),
			TotalPings:       10,
			FailedPings:      4,
			AvgDelayMs:       120,
			ConsecutiveFails: 7,
			Cleanliness:      CleanlinessUnknown,
			BandwidthTier:    BandwidthTierUnknown,
		},
	}
	sm.state.ValidationConfig = ValidationConfig{
		MinSamples:       1,
		MaxFailRate:      1,
		MaxAvgDelayMs:    500,
		ProbeIntervalSec: 60,
		ProbeURL:         "https://www.gstatic.com/generate_204",
		DemoteAfterFails: 2,
		MinActiveNodes:   1,
	}
	sm.mu.Unlock()

	sm.handleProbeResults([]ProbeResult{
		{Tag: probeOutboundTag(quarantineID), Success: false, DelayMs: 0},
	})

	nodes := sm.ListNodes("")
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Status != NodeStatusQuarantine {
		t.Fatalf("expected quarantined node to stay quarantined after failed probe, got %q", nodes[0].Status)
	}
	if nodes[0].ConsecutiveFails != 8 {
		t.Fatalf("expected consecutive fails to increment, got %d", nodes[0].ConsecutiveFails)
	}
}

func TestSubscriptionManagerNormalizeProbeStateOnStart(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	sm := NewSubscriptionManager(configPath, nil, nil, nil)
	defer sm.Stop()

	activeID := "active-dirty"
	sm.mu.Lock()
	sm.state.Nodes = []NodeRecord{
		{
			ID:               activeID,
			Remark:           "dirty active",
			Address:          "dirty.example.com",
			Port:             443,
			Status:           NodeStatusActive,
			StatusReason:     TransitionReasonProbeRequalified,
			OutboundTag:      probeOutboundTag(activeID),
			AddedAt:          time.Now().Add(-time.Minute),
			StatusUpdatedAt:  timePtr(time.Now().Add(-time.Minute)),
			TotalPings:       100,
			FailedPings:      40,
			AvgDelayMs:       180,
			ConsecutiveFails: 6,
			Cleanliness:      CleanlinessUnknown,
			BandwidthTier:    BandwidthTierUnknown,
		},
	}
	sm.state.ValidationConfig = ValidationConfig{
		MinSamples:       1,
		MaxFailRate:      1,
		MaxAvgDelayMs:    500,
		ProbeIntervalSec: 60,
		ProbeURL:         "https://www.gstatic.com/generate_204",
		DemoteAfterFails: 2,
		MinActiveNodes:   1,
	}
	sm.mu.Unlock()

	if err := sm.Start(); err != nil {
		t.Fatalf("start subscription manager: %v", err)
	}

	nodes := sm.ListNodes("")
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Status != NodeStatusQuarantine {
		t.Fatalf("expected startup normalization to quarantine invalid active node, got %q", nodes[0].Status)
	}
	if nodes[0].StatusReason != TransitionReasonProbeFailuresExceeded {
		t.Fatalf("expected startup normalization reason, got %q", nodes[0].StatusReason)
	}
}

func timePtr(value time.Time) *time.Time {
	return &value
}

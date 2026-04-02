package webpanel

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	handlerservice "github.com/xtls/xray-core/app/proxyman/command"
	"google.golang.org/grpc"
)

type stubHandlerServiceClient struct {
	addedTags   []string
	removedTags []string
}

func (s *stubHandlerServiceClient) AddInbound(context.Context, *handlerservice.AddInboundRequest, ...grpc.CallOption) (*handlerservice.AddInboundResponse, error) {
	return &handlerservice.AddInboundResponse{}, nil
}

func (s *stubHandlerServiceClient) RemoveInbound(context.Context, *handlerservice.RemoveInboundRequest, ...grpc.CallOption) (*handlerservice.RemoveInboundResponse, error) {
	return &handlerservice.RemoveInboundResponse{}, nil
}

func (s *stubHandlerServiceClient) AlterInbound(context.Context, *handlerservice.AlterInboundRequest, ...grpc.CallOption) (*handlerservice.AlterInboundResponse, error) {
	return &handlerservice.AlterInboundResponse{}, nil
}

func (s *stubHandlerServiceClient) ListInbounds(context.Context, *handlerservice.ListInboundsRequest, ...grpc.CallOption) (*handlerservice.ListInboundsResponse, error) {
	return &handlerservice.ListInboundsResponse{}, nil
}

func (s *stubHandlerServiceClient) GetInboundUsers(context.Context, *handlerservice.GetInboundUserRequest, ...grpc.CallOption) (*handlerservice.GetInboundUserResponse, error) {
	return &handlerservice.GetInboundUserResponse{}, nil
}

func (s *stubHandlerServiceClient) GetInboundUsersCount(context.Context, *handlerservice.GetInboundUserRequest, ...grpc.CallOption) (*handlerservice.GetInboundUsersCountResponse, error) {
	return &handlerservice.GetInboundUsersCountResponse{}, nil
}

func (s *stubHandlerServiceClient) AddOutbound(_ context.Context, in *handlerservice.AddOutboundRequest, _ ...grpc.CallOption) (*handlerservice.AddOutboundResponse, error) {
	if in != nil && in.Outbound != nil {
		s.addedTags = append(s.addedTags, in.Outbound.Tag)
	}
	return &handlerservice.AddOutboundResponse{}, nil
}

func (s *stubHandlerServiceClient) RemoveOutbound(_ context.Context, in *handlerservice.RemoveOutboundRequest, _ ...grpc.CallOption) (*handlerservice.RemoveOutboundResponse, error) {
	if in != nil {
		s.removedTags = append(s.removedTags, in.Tag)
	}
	return &handlerservice.RemoveOutboundResponse{}, nil
}

func (s *stubHandlerServiceClient) AlterOutbound(context.Context, *handlerservice.AlterOutboundRequest, ...grpc.CallOption) (*handlerservice.AlterOutboundResponse, error) {
	return &handlerservice.AlterOutboundResponse{}, nil
}

func (s *stubHandlerServiceClient) ListOutbounds(context.Context, *handlerservice.ListOutboundsRequest, ...grpc.CallOption) (*handlerservice.ListOutboundsResponse, error) {
	return &handlerservice.ListOutboundsResponse{}, nil
}

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

func TestSubscriptionManagerMovesStagingNodeToQuarantineAfterRepeatedFailures(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	sm := NewSubscriptionManager(configPath, nil, nil, nil)
	defer sm.Stop()

	stagingID := "staging-fails"
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
	}
	sm.state.ValidationConfig = ValidationConfig{
		MinSamples:       2,
		MaxFailRate:      0.5,
		MaxAvgDelayMs:    500,
		ProbeIntervalSec: 60,
		ProbeURL:         "https://www.gstatic.com/generate_204",
		DemoteAfterFails: 2,
		MinActiveNodes:   1,
	}
	sm.mu.Unlock()

	sm.handleProbeResults([]ProbeResult{
		{Tag: probeOutboundTag(stagingID), Success: false, DelayMs: 0},
		{Tag: probeOutboundTag(stagingID), Success: false, DelayMs: 0},
	})

	nodes := sm.ListNodes("")
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Status != NodeStatusQuarantine {
		t.Fatalf("expected staging node to move to quarantine, got %q", nodes[0].Status)
	}
	if nodes[0].StatusReason != TransitionReasonProbeFailuresExceeded {
		t.Fatalf("expected probe failure reason, got %q", nodes[0].StatusReason)
	}
}

func TestSubscriptionManagerBulkPromotePromotesSelectedNodes(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	sm := NewSubscriptionManager(configPath, nil, nil, nil)
	defer sm.Stop()

	stagingID := "staging-bulk"
	quarantineID := "quarantine-bulk"
	activeID := "active-bulk"
	removedID := "removed-bulk"

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
			ID:              quarantineID,
			Remark:          "quarantine",
			Address:         "quarantine.example.com",
			Port:            443,
			Status:          NodeStatusQuarantine,
			StatusReason:    TransitionReasonProbeFailuresExceeded,
			OutboundTag:     probeOutboundTag(quarantineID),
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
		{
			ID:              removedID,
			Remark:          "removed",
			Address:         "removed.example.com",
			Port:            443,
			Status:          NodeStatusRemoved,
			StatusReason:    TransitionReasonManualRemove,
			OutboundTag:     probeOutboundTag(removedID),
			AddedAt:         time.Now().Add(-time.Minute),
			StatusUpdatedAt: timePtr(time.Now().Add(-time.Minute)),
			Cleanliness:     CleanlinessUnknown,
			BandwidthTier:   BandwidthTierUnknown,
		},
	}
	sm.mu.Unlock()

	promoted, err := sm.BulkPromote([]string{stagingID, quarantineID, activeID, removedID, "missing"})
	if err != nil {
		t.Fatalf("bulk promote: %v", err)
	}
	if promoted != 2 {
		t.Fatalf("expected 2 promoted nodes, got %d", promoted)
	}

	nodes := sm.ListNodes("")
	statusByID := make(map[string]NodeStatus, len(nodes))
	reasonByID := make(map[string]TransitionReason, len(nodes))
	for _, node := range nodes {
		statusByID[node.ID] = node.Status
		reasonByID[node.ID] = node.StatusReason
	}

	if statusByID[stagingID] != NodeStatusActive {
		t.Fatalf("expected staging node to become active, got %q", statusByID[stagingID])
	}
	if statusByID[quarantineID] != NodeStatusActive {
		t.Fatalf("expected quarantine node to become active, got %q", statusByID[quarantineID])
	}
	if reasonByID[stagingID] != TransitionReasonManualPromote {
		t.Fatalf("expected staging reason to become manual promote, got %q", reasonByID[stagingID])
	}
	if reasonByID[quarantineID] != TransitionReasonManualPromote {
		t.Fatalf("expected quarantine reason to become manual promote, got %q", reasonByID[quarantineID])
	}
	if statusByID[activeID] != NodeStatusActive {
		t.Fatalf("expected active node to stay active, got %q", statusByID[activeID])
	}
	if statusByID[removedID] != NodeStatusRemoved {
		t.Fatalf("expected removed node to stay removed, got %q", statusByID[removedID])
	}

	summary := sm.GetPoolSummary()
	if summary.ActiveCount != 3 {
		t.Fatalf("expected 3 active nodes in summary, got %d", summary.ActiveCount)
	}
}

func TestSubscriptionManagerBulkValidateMovesCandidateNodesToStaging(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	handlerStub := &stubHandlerServiceClient{}
	sm := NewSubscriptionManager(configPath, &GRPCClient{handlerClient: handlerStub}, nil, nil)
	defer sm.Stop()

	uri, err := GenerateShareLink(ShareLinkRequest{
		Protocol: "vmess",
		Address:  "candidate.example.com",
		Port:     443,
		UUID:     "22222222-2222-2222-2222-222222222222",
		Remark:   "candidate",
		TLS:      "tls",
		SNI:      "candidate.example.com",
	})
	if err != nil {
		t.Fatalf("generate share link: %v", err)
	}

	candidateID := "candidate-bulk"
	activeID := "active-bulk"
	sm.mu.Lock()
	sm.state.Nodes = []NodeRecord{
		{
			ID:              candidateID,
			URI:             uri,
			Remark:          "candidate",
			Protocol:        "vmess",
			Address:         "candidate.example.com",
			Port:            443,
			Status:          NodeStatusCandidate,
			StatusReason:    TransitionReasonManualRestore,
			AddedAt:         time.Now().Add(-time.Minute),
			StatusUpdatedAt: timePtr(time.Now().Add(-time.Minute)),
			Cleanliness:     CleanlinessUnknown,
			BandwidthTier:   BandwidthTierUnknown,
		},
		{
			ID:              activeID,
			URI:             uri,
			Remark:          "active",
			Protocol:        "vmess",
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
	sm.mu.Unlock()

	validated, err := sm.BulkValidate([]string{candidateID, activeID, "missing"})
	if err != nil {
		t.Fatalf("bulk validate: %v", err)
	}
	if validated != 1 {
		t.Fatalf("expected 1 validated node, got %d", validated)
	}

	nodes := sm.ListNodes("")
	statusByID := make(map[string]NodeStatus, len(nodes))
	reasonByID := make(map[string]TransitionReason, len(nodes))
	tagByID := make(map[string]string, len(nodes))
	for _, node := range nodes {
		statusByID[node.ID] = node.Status
		reasonByID[node.ID] = node.StatusReason
		tagByID[node.ID] = node.OutboundTag
	}

	if statusByID[candidateID] != NodeStatusStaging {
		t.Fatalf("expected candidate node to become staging, got %q", statusByID[candidateID])
	}
	if reasonByID[candidateID] != TransitionReasonManualValidate {
		t.Fatalf("expected candidate node reason to become manual validate, got %q", reasonByID[candidateID])
	}
	if tagByID[candidateID] != probeOutboundTag(candidateID) {
		t.Fatalf("expected candidate outbound tag %q, got %q", probeOutboundTag(candidateID), tagByID[candidateID])
	}
	if len(handlerStub.addedTags) != 1 || handlerStub.addedTags[0] != probeOutboundTag(candidateID) {
		t.Fatalf("expected outbound %q to be added once, got %v", probeOutboundTag(candidateID), handlerStub.addedTags)
	}
	if statusByID[activeID] != NodeStatusActive {
		t.Fatalf("expected active node to stay active, got %q", statusByID[activeID])
	}

	summary := sm.GetPoolSummary()
	if summary.CandidateCount != 0 {
		t.Fatalf("expected 0 candidate nodes in summary, got %d", summary.CandidateCount)
	}
	if summary.StagingCount != 1 {
		t.Fatalf("expected 1 staging node in summary, got %d", summary.StagingCount)
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

func TestSubscriptionManagerAddManualSubscriptionStoresSanitizedContent(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	sm := NewSubscriptionManager(configPath, nil, nil, nil)
	defer sm.Stop()

	content := "\ufeffvless://11111111-1111-1111-1111-111111111111@example.com:443?security=tls#manual-node"

	sub, err := sm.AddSubscription(SubscriptionInput{
		SourceType: SubscriptionSourceManual,
		Content:    content,
		Remark:     "manual import",
	})
	if err != nil {
		t.Fatalf("add manual subscription: %v", err)
	}

	if sub.SourceType != SubscriptionSourceManual {
		t.Fatalf("expected manual source type, got %q", sub.SourceType)
	}
	if sub.AutoRefresh {
		t.Fatal("expected manual import auto-refresh to be disabled")
	}
	if sub.Content != "" {
		t.Fatal("expected returned subscription content to be redacted")
	}

	if got := sm.state.Subscriptions[0].Content; got == "" {
		t.Fatal("expected stored subscription content to be preserved")
	}

	nodes := sm.ListNodes("")
	if len(nodes) != 1 {
		t.Fatalf("expected 1 imported node, got %d", len(nodes))
	}
	if nodes[0].Remark != "manual-node" {
		t.Fatalf("expected imported remark manual-node, got %q", nodes[0].Remark)
	}

	listed := sm.ListSubscriptions()
	if len(listed) != 1 {
		t.Fatalf("expected 1 listed subscription, got %d", len(listed))
	}
	if listed[0].Content != "" {
		t.Fatal("expected listed subscription content to be redacted")
	}
}

func TestSubscriptionManagerRestoreNodeMovesRemovedNodeBackToCandidate(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	sm := NewSubscriptionManager(configPath, nil, nil, nil)
	defer sm.Stop()

	uri, err := GenerateShareLink(ShareLinkRequest{
		Protocol: "vless",
		Address:  "restore.example.com",
		Port:     443,
		UUID:     "22222222-2222-2222-2222-222222222222",
		Remark:   "restore-me",
		TLS:      "tls",
		SNI:      "restore.example.com",
	})
	if err != nil {
		t.Fatalf("generate share link: %v", err)
	}

	removedAt := time.Now().Add(-time.Minute)
	sm.mu.Lock()
	sm.state.Nodes = []NodeRecord{
		{
			ID:              "removed-restore",
			URI:             uri,
			Remark:          "restore-me",
			Protocol:        "vless",
			Address:         "restore.example.com",
			Port:            443,
			Status:          NodeStatusRemoved,
			StatusReason:    TransitionReasonManualRemove,
			AddedAt:         removedAt.Add(-time.Hour),
			StatusUpdatedAt: timePtr(removedAt),
			LastEventAt:     timePtr(removedAt),
			Cleanliness:     CleanlinessUnknown,
			BandwidthTier:   BandwidthTierUnknown,
		},
	}
	sm.mu.Unlock()

	if err := sm.RestoreNode("removed-restore"); err != nil {
		t.Fatalf("restore node: %v", err)
	}

	nodes := sm.ListNodes("")
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node after restore, got %d", len(nodes))
	}
	if nodes[0].Status != NodeStatusCandidate {
		t.Fatalf("expected restored node to move to candidate, got %q", nodes[0].Status)
	}
	if nodes[0].StatusReason != TransitionReasonManualRestore {
		t.Fatalf("expected restore reason, got %q", nodes[0].StatusReason)
	}
	if nodes[0].OutboundTag != "" {
		t.Fatalf("expected restored candidate node to have no outbound tag, got %q", nodes[0].OutboundTag)
	}
}

func TestSubscriptionManagerBulkRestoreMovesSelectedRemovedNodesToCandidate(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	sm := NewSubscriptionManager(configPath, nil, nil, nil)
	defer sm.Stop()

	removedID := "removed-candidate"
	otherRemovedID := "removed-keep"
	activeID := "active-keep"

	sm.mu.Lock()
	sm.state.Nodes = []NodeRecord{
		{
			ID:              removedID,
			Remark:          "restore this",
			Address:         "restore.example.com",
			Port:            443,
			Status:          NodeStatusRemoved,
			StatusReason:    TransitionReasonManualRemove,
			AddedAt:         time.Now().Add(-2 * time.Minute),
			StatusUpdatedAt: timePtr(time.Now().Add(-time.Minute)),
			Cleanliness:     CleanlinessUnknown,
			BandwidthTier:   BandwidthTierUnknown,
		},
		{
			ID:              otherRemovedID,
			Remark:          "stay removed",
			Address:         "stay.example.com",
			Port:            443,
			Status:          NodeStatusRemoved,
			StatusReason:    TransitionReasonManualRemove,
			AddedAt:         time.Now().Add(-3 * time.Minute),
			StatusUpdatedAt: timePtr(time.Now().Add(-2 * time.Minute)),
			Cleanliness:     CleanlinessUnknown,
			BandwidthTier:   BandwidthTierUnknown,
		},
		{
			ID:              activeID,
			Remark:          "still active",
			Address:         "active.example.com",
			Port:            443,
			Status:          NodeStatusActive,
			StatusReason:    TransitionReasonManualPromote,
			AddedAt:         time.Now().Add(-4 * time.Minute),
			StatusUpdatedAt: timePtr(time.Now().Add(-3 * time.Minute)),
			Cleanliness:     CleanlinessUnknown,
			BandwidthTier:   BandwidthTierUnknown,
		},
	}
	sm.mu.Unlock()

	restored, err := sm.BulkRestore([]string{removedID, activeID, "missing"})
	if err != nil {
		t.Fatalf("bulk restore: %v", err)
	}
	if restored != 1 {
		t.Fatalf("expected 1 restored node, got %d", restored)
	}

	nodes := sm.ListNodes("")
	statusByID := make(map[string]NodeStatus, len(nodes))
	for _, node := range nodes {
		statusByID[node.ID] = node.Status
	}

	if statusByID[removedID] != NodeStatusCandidate {
		t.Fatalf("expected selected removed node to become candidate, got %q", statusByID[removedID])
	}
	if statusByID[otherRemovedID] != NodeStatusRemoved {
		t.Fatalf("expected unselected removed node to stay removed, got %q", statusByID[otherRemovedID])
	}
	if statusByID[activeID] != NodeStatusActive {
		t.Fatalf("expected active node to stay active, got %q", statusByID[activeID])
	}
}

func TestSubscriptionManagerBulkPurgeRemovedDeletesOnlySelectedRemovedNodes(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	sm := NewSubscriptionManager(configPath, nil, nil, nil)
	defer sm.Stop()

	removedID := "removed-purge"
	otherRemovedID := "removed-keep"
	activeID := "active-keep"

	sm.mu.Lock()
	sm.state.Nodes = []NodeRecord{
		{
			ID:              removedID,
			Remark:          "purge this",
			Address:         "purge.example.com",
			Port:            443,
			Status:          NodeStatusRemoved,
			StatusReason:    TransitionReasonManualRemove,
			AddedAt:         time.Now().Add(-3 * time.Minute),
			StatusUpdatedAt: timePtr(time.Now().Add(-2 * time.Minute)),
			Cleanliness:     CleanlinessUnknown,
			BandwidthTier:   BandwidthTierUnknown,
		},
		{
			ID:              otherRemovedID,
			Remark:          "keep this",
			Address:         "keep.example.com",
			Port:            443,
			Status:          NodeStatusRemoved,
			StatusReason:    TransitionReasonManualRemove,
			AddedAt:         time.Now().Add(-2 * time.Minute),
			StatusUpdatedAt: timePtr(time.Now().Add(-time.Minute)),
			Cleanliness:     CleanlinessUnknown,
			BandwidthTier:   BandwidthTierUnknown,
		},
		{
			ID:              activeID,
			Remark:          "stay active",
			Address:         "active.example.com",
			Port:            443,
			Status:          NodeStatusActive,
			StatusReason:    TransitionReasonManualPromote,
			AddedAt:         time.Now().Add(-4 * time.Minute),
			StatusUpdatedAt: timePtr(time.Now().Add(-3 * time.Minute)),
			Cleanliness:     CleanlinessUnknown,
			BandwidthTier:   BandwidthTierUnknown,
		},
	}
	sm.mu.Unlock()

	purged, err := sm.BulkPurgeRemoved([]string{removedID, activeID, "missing"})
	if err != nil {
		t.Fatalf("bulk purge removed: %v", err)
	}
	if purged != 1 {
		t.Fatalf("expected 1 purged removed node, got %d", purged)
	}

	nodes := sm.ListNodes("")
	statusByID := make(map[string]NodeStatus, len(nodes))
	for _, node := range nodes {
		statusByID[node.ID] = node.Status
	}

	if _, ok := statusByID[removedID]; ok {
		t.Fatalf("expected selected removed node to be deleted from state, still found %q", statusByID[removedID])
	}
	if statusByID[otherRemovedID] != NodeStatusRemoved {
		t.Fatalf("expected unselected removed node to stay removed, got %q", statusByID[otherRemovedID])
	}
	if statusByID[activeID] != NodeStatusActive {
		t.Fatalf("expected active node to stay active, got %q", statusByID[activeID])
	}

	summary := sm.GetPoolSummary()
	if summary.RemovedCount != 1 {
		t.Fatalf("expected 1 removed node in summary after purge, got %d", summary.RemovedCount)
	}
}

func TestSubscriptionManagerRefreshSubscriptionParksMissingNodesInCandidate(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	handlerStub := &stubHandlerServiceClient{}
	sm := NewSubscriptionManager(configPath, &GRPCClient{handlerClient: handlerStub}, nil, nil)
	defer sm.Stop()

	missingURI, err := GenerateShareLink(ShareLinkRequest{
		Protocol: "vless",
		Address:  "missing.example.com",
		Port:     443,
		UUID:     "33333333-3333-3333-3333-333333333333",
		Remark:   "missing-node",
		TLS:      "tls",
		SNI:      "missing.example.com",
	})
	if err != nil {
		t.Fatalf("generate missing share link: %v", err)
	}

	liveURI, err := GenerateShareLink(ShareLinkRequest{
		Protocol: "vless",
		Address:  "live.example.com",
		Port:     443,
		UUID:     "44444444-4444-4444-4444-444444444444",
		Remark:   "live-node",
		TLS:      "tls",
		SNI:      "live.example.com",
	})
	if err != nil {
		t.Fatalf("generate live share link: %v", err)
	}

	subID := "sub-missing"
	sm.mu.Lock()
	sm.state.Subscriptions = []SubscriptionRecord{
		{
			ID:         subID,
			SourceType: SubscriptionSourceManual,
			Content:    missingURI + "\n" + liveURI,
			Remark:     "manual sub",
		},
	}
	sm.mu.Unlock()

	if err := sm.refreshSubscription(subID); err != nil {
		t.Fatalf("initial refresh subscription: %v", err)
	}

	missingID := ""
	liveID := ""
	for _, node := range sm.ListNodes("") {
		switch node.Address {
		case "missing.example.com":
			missingID = node.ID
		case "live.example.com":
			liveID = node.ID
		}
	}
	if missingID == "" || liveID == "" {
		t.Fatalf("expected both imported node IDs, got missing=%q live=%q", missingID, liveID)
	}

	sm.mu.Lock()
	sm.state.Subscriptions[0].Content = liveURI
	sm.mu.Unlock()

	if err := sm.refreshSubscription(subID); err != nil {
		t.Fatalf("second refresh subscription: %v", err)
	}

	nodes := sm.ListNodes("")
	statusByID := make(map[string]NodeStatus, len(nodes))
	reasonByID := make(map[string]TransitionReason, len(nodes))
	tagByID := make(map[string]string, len(nodes))
	for _, node := range nodes {
		statusByID[node.ID] = node.Status
		reasonByID[node.ID] = node.StatusReason
		tagByID[node.ID] = node.OutboundTag
	}

	if statusByID[missingID] != NodeStatusCandidate {
		t.Fatalf("expected missing node to move to candidate, got %q", statusByID[missingID])
	}
	if reasonByID[missingID] != TransitionReasonSubscriptionMissing {
		t.Fatalf("expected missing node reason to become subscription_missing, got %q", reasonByID[missingID])
	}
	if tagByID[missingID] != "" {
		t.Fatalf("expected missing node outbound tag to be cleared, got %q", tagByID[missingID])
	}
	if statusByID[liveID] != NodeStatusStaging {
		t.Fatalf("expected live node to be staged, got %q", statusByID[liveID])
	}

	listed := sm.ListSubscriptions()
	if len(listed) != 1 {
		t.Fatalf("expected 1 subscription, got %d", len(listed))
	}
	if listed[0].NodeCount != 1 {
		t.Fatalf("expected subscription node count 1 after parking missing node, got %d", listed[0].NodeCount)
	}
}

func TestSubscriptionManagerRefreshSubscriptionReintroducesCandidateMissingNode(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	handlerStub := &stubHandlerServiceClient{}
	sm := NewSubscriptionManager(configPath, &GRPCClient{handlerClient: handlerStub}, nil, nil)
	defer sm.Stop()

	uri, err := GenerateShareLink(ShareLinkRequest{
		Protocol: "vless",
		Address:  "return.example.com",
		Port:     443,
		UUID:     "55555555-5555-5555-5555-555555555555",
		Remark:   "return-node",
		TLS:      "tls",
		SNI:      "return.example.com",
	})
	if err != nil {
		t.Fatalf("generate share link: %v", err)
	}

	subID := "sub-return"
	sm.mu.Lock()
	sm.state.Subscriptions = []SubscriptionRecord{
		{
			ID:         subID,
			SourceType: SubscriptionSourceManual,
			Content:    uri,
			Remark:     "manual sub",
		},
	}
	sm.mu.Unlock()

	if err := sm.refreshSubscription(subID); err != nil {
		t.Fatalf("initial refresh subscription: %v", err)
	}

	nodes := sm.ListNodes("")
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node after initial refresh, got %d", len(nodes))
	}
	nodeID := nodes[0].ID
	handlerStub.addedTags = nil

	sm.mu.Lock()
	sm.state.Nodes[0].Status = NodeStatusCandidate
	sm.state.Nodes[0].StatusReason = TransitionReasonSubscriptionMissing
	sm.state.Nodes[0].OutboundTag = ""
	now := time.Now().Add(-time.Minute)
	sm.state.Nodes[0].StatusUpdatedAt = timePtr(now)
	sm.state.Nodes[0].LastEventAt = timePtr(now)
	sm.mu.Unlock()

	if err := sm.refreshSubscription(subID); err != nil {
		t.Fatalf("second refresh subscription: %v", err)
	}

	nodes = sm.ListNodes("")
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Status != NodeStatusStaging {
		t.Fatalf("expected reintroduced node to move to staging, got %q", nodes[0].Status)
	}
	if nodes[0].StatusReason != TransitionReasonSubscriptionReintroduced {
		t.Fatalf("expected reintroduced node reason, got %q", nodes[0].StatusReason)
	}
	if nodes[0].OutboundTag != probeOutboundTag(nodeID) {
		t.Fatalf("expected reintroduced node outbound tag %q, got %q", probeOutboundTag(nodeID), nodes[0].OutboundTag)
	}
	if len(handlerStub.addedTags) != 1 || handlerStub.addedTags[0] != probeOutboundTag(nodeID) {
		t.Fatalf("expected outbound %q to be added once, got %v", probeOutboundTag(nodeID), handlerStub.addedTags)
	}
}

func timePtr(value time.Time) *time.Time {
	return &value
}

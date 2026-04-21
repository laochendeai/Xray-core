package webpanel

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	handlerservice "github.com/xtls/xray-core/app/proxyman/command"
	xnet "github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/features/routing"
	"github.com/xtls/xray-core/transport"
	"google.golang.org/grpc"
)

type stubHandlerServiceClient struct {
	addedTags   []string
	removedTags []string
}

type stubRoutingDispatcher struct{}

func (s *stubRoutingDispatcher) Type() interface{} { return routing.DispatcherType() }
func (s *stubRoutingDispatcher) Start() error      { return nil }
func (s *stubRoutingDispatcher) Close() error      { return nil }
func (s *stubRoutingDispatcher) Dispatch(context.Context, xnet.Destination) (*transport.Link, error) {
	return nil, nil
}
func (s *stubRoutingDispatcher) DispatchLink(context.Context, xnet.Destination, *transport.Link) error {
	return nil
}

func waitFor(t *testing.T, timeout time.Duration, condition func() bool, description string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", description)
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

func TestSubscriptionManagerHandleProbeResultsStoresNodeExitIP(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	statePath := filepath.Join(tempDir, "node_pool_state.json")
	sm := NewSubscriptionManager(configPath, nil, nil, nil)
	sm.saveDelay = 5 * time.Millisecond
	sm.dispatcher = &stubRoutingDispatcher{}
	sm.nodeExitIPProber = func(context.Context, routing.Dispatcher, string) nodeExitIPProbeResult {
		return nodeExitIPProbeResult{
			IP:        "203.0.113.44",
			Source:    "https://api.ipify.org",
			CheckedAt: time.Now().UTC(),
		}
	}
	defer sm.Stop()

	sm.mu.Lock()
	sm.state.Nodes = []NodeRecord{
		{
			ID:               "node-exit-ip-success",
			URI:              "vmess://example",
			Remark:           "probeable",
			Status:           NodeStatusActive,
			StatusReason:     TransitionReasonManualPromote,
			SubscriptionID:   "sub-1",
			AddedAt:          time.Now().Add(-time.Minute),
			OutboundTag:      probeOutboundTag("node-exit-ip-success"),
			Cleanliness:      CleanlinessUnknown,
			BandwidthTier:    BandwidthTierUnknown,
			ExitIPStatus:     NodeExitIPStatusUnknown,
			TotalPings:       4,
			FailedPings:      0,
			AvgDelayMs:       120,
			ConsecutiveFails: 0,
		},
	}
	sm.mu.Unlock()

	sm.handleProbeResults([]ProbeResult{{
		Tag:     probeOutboundTag("node-exit-ip-success"),
		Success: true,
		DelayMs: 88,
	}})

	waitFor(t, time.Second, func() bool {
		nodes := sm.ListNodes("")
		return len(nodes) == 1 && nodes[0].ExitIPStatus == NodeExitIPStatusAvailable && nodes[0].ExitIP == "203.0.113.44"
	}, "node exit IP probe success")

	sm.mu.Lock()
	sm.writeStateLocked()
	sm.mu.Unlock()

	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read state file: %v", err)
	}
	var persisted NodePoolState
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("decode persisted state: %v", err)
	}
	if len(persisted.Nodes) != 1 || persisted.Nodes[0].ExitIP != "203.0.113.44" || persisted.Nodes[0].ExitIPStatus != NodeExitIPStatusAvailable {
		t.Fatalf("unexpected persisted exit IP state: %#v", persisted.Nodes)
	}
}

func TestSubscriptionManagerHandleProbeResultsCachesFreshNodeExitIP(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	sm := NewSubscriptionManager(configPath, nil, nil, nil)
	sm.saveDelay = 5 * time.Millisecond
	sm.dispatcher = &stubRoutingDispatcher{}
	sm.nodeExitIPProbeTTL = time.Hour

	callCount := 0
	sm.nodeExitIPProber = func(context.Context, routing.Dispatcher, string) nodeExitIPProbeResult {
		callCount++
		return nodeExitIPProbeResult{
			IP:        "203.0.113.45",
			Source:    "https://api.ipify.org",
			CheckedAt: time.Now().UTC(),
		}
	}
	defer sm.Stop()

	checkedAt := time.Now().Add(-5 * time.Minute).UTC()
	sm.mu.Lock()
	sm.state.Nodes = []NodeRecord{
		{
			ID:              "node-exit-ip-cache",
			URI:             "vmess://example",
			Remark:          "fresh-cache",
			Status:          NodeStatusActive,
			StatusReason:    TransitionReasonManualPromote,
			SubscriptionID:  "sub-1",
			AddedAt:         time.Now().Add(-time.Minute),
			OutboundTag:     probeOutboundTag("node-exit-ip-cache"),
			Cleanliness:     CleanlinessUnknown,
			BandwidthTier:   BandwidthTierUnknown,
			ExitIPStatus:    NodeExitIPStatusAvailable,
			ExitIP:          "203.0.113.45",
			ExitIPSource:    "https://api.ipify.org",
			ExitIPCheckedAt: &checkedAt,
		},
	}
	sm.mu.Unlock()

	sm.handleProbeResults([]ProbeResult{{
		Tag:     probeOutboundTag("node-exit-ip-cache"),
		Success: true,
		DelayMs: 52,
	}})

	time.Sleep(100 * time.Millisecond)
	if callCount != 0 {
		t.Fatalf("expected no exit-IP reprobe for fresh cache, got %d calls", callCount)
	}
}

func TestSubscriptionManagerHandleProbeResultsStoresNodeExitIPError(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	sm := NewSubscriptionManager(configPath, nil, nil, nil)
	sm.saveDelay = 5 * time.Millisecond
	sm.dispatcher = &stubRoutingDispatcher{}
	sm.nodeExitIPProber = func(context.Context, routing.Dispatcher, string) nodeExitIPProbeResult {
		return nodeExitIPProbeResult{
			Source:    "https://api.ipify.org",
			CheckedAt: time.Now().UTC(),
			Error:     "probe timeout",
		}
	}
	defer sm.Stop()

	sm.mu.Lock()
	sm.state.Nodes = []NodeRecord{
		{
			ID:             "node-exit-ip-error",
			URI:            "vmess://example",
			Remark:         "probe-error",
			Status:         NodeStatusStaging,
			StatusReason:   TransitionReasonSubscriptionNodeDiscovered,
			SubscriptionID: "sub-1",
			AddedAt:        time.Now().Add(-time.Minute),
			OutboundTag:    probeOutboundTag("node-exit-ip-error"),
			Cleanliness:    CleanlinessUnknown,
			BandwidthTier:  BandwidthTierUnknown,
			ExitIPStatus:   NodeExitIPStatusUnknown,
		},
	}
	sm.mu.Unlock()

	sm.handleProbeResults([]ProbeResult{{
		Tag:     probeOutboundTag("node-exit-ip-error"),
		Success: true,
		DelayMs: 44,
	}})

	waitFor(t, time.Second, func() bool {
		nodes := sm.ListNodes("")
		return len(nodes) == 1 && nodes[0].ExitIPStatus == NodeExitIPStatusError && nodes[0].ExitIPError == "probe timeout"
	}, "node exit IP probe error")
}

func TestSubscriptionManagerRunNodeIntelligenceRefreshStoresVerdicts(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	statePath := filepath.Join(tempDir, "node_pool_state.json")
	sm := NewSubscriptionManager(configPath, nil, nil, nil)
	sm.nodeIntelligenceLookup = func(context.Context, string) nodeIPConnectionLookupResult {
		return nodeIPConnectionLookupResult{
			ASN:       7922,
			Org:       "Comcast Cable Communications, LLC",
			ISP:       "Comcast Cable Communications, LLC",
			Domain:    "comcast.net",
			CheckedAt: time.Now().UTC(),
		}
	}
	defer sm.Stop()

	sm.mu.Lock()
	sm.state.ValidationConfig.MinSamples = 5
	sm.state.Nodes = []NodeRecord{
		{
			ID:                    "node-intel-success",
			URI:                   "vmess://example",
			Remark:                "intel",
			Status:                NodeStatusActive,
			StatusReason:          TransitionReasonManualPromote,
			SubscriptionID:        "sub-1",
			AddedAt:               time.Now().Add(-time.Minute),
			OutboundTag:           probeOutboundTag("node-intel-success"),
			Cleanliness:           CleanlinessUnknown,
			CleanlinessConfidence: NodeIntelligenceConfidenceUnknown,
			BandwidthTier:         BandwidthTierUnknown,
			ExitIPStatus:          NodeExitIPStatusAvailable,
			ExitIP:                "198.51.100.40",
			ExitIPCheckedAt:       timePtr(time.Now().UTC()),
			NetworkType:           NodeNetworkTypeUnknown,
			NetworkTypeConfidence: NodeIntelligenceConfidenceUnknown,
			TotalPings:            12,
			FailedPings:           0,
			AvgDelayMs:            180,
			ConsecutiveFails:      0,
		},
	}
	sm.mu.Unlock()

	sm.runNodeIntelligenceRefresh(probeOutboundTag("node-intel-success"), "198.51.100.40")

	nodes := sm.ListNodes("")
	if len(nodes) != 1 {
		t.Fatalf("expected one node, got %d", len(nodes))
	}
	if nodes[0].NetworkType != NodeNetworkTypeResidentialLikely {
		t.Fatalf("expected residential-likely network type, got %q", nodes[0].NetworkType)
	}
	if nodes[0].Cleanliness != CleanlinessTrusted {
		t.Fatalf("expected trusted cleanliness, got %q", nodes[0].Cleanliness)
	}
	if nodes[0].CleanlinessReason == "" || nodes[0].NetworkTypeReason == "" {
		t.Fatalf("expected persisted intelligence reasons, got %#v", nodes[0])
	}

	sm.mu.Lock()
	sm.writeStateLocked()
	sm.mu.Unlock()

	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read state file: %v", err)
	}
	var persisted NodePoolState
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("decode persisted state: %v", err)
	}
	if len(persisted.Nodes) != 1 || persisted.Nodes[0].Cleanliness != CleanlinessTrusted || persisted.Nodes[0].NetworkType != NodeNetworkTypeResidentialLikely {
		t.Fatalf("unexpected persisted intelligence state: %#v", persisted.Nodes)
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

func TestSubscriptionManagerAddManualSubscriptionImportsAnyTLS(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	sm := NewSubscriptionManager(configPath, nil, nil, nil)
	defer sm.Stop()

	content := strings.Join([]string{
		"anytls://dongtaiwang.com@45.221.98.14:59901?security=none&type=tcp&allowInsecure=1&insecure=1#US_17",
		"anytls://dongtaiwang.com@45.221.98.14:59901?security=none&type=tcp&alpn=h2&allowInsecure=1&sni=45.221.98.14&insecure=1#US_21",
	}, "\n")

	sub, err := sm.AddSubscription(SubscriptionInput{
		SourceType: SubscriptionSourceManual,
		Content:    content,
		Remark:     "manual anytls import",
	})
	if err != nil {
		t.Fatalf("add manual anytls subscription: %v", err)
	}
	if sub.SourceType != SubscriptionSourceManual {
		t.Fatalf("expected manual source type, got %q", sub.SourceType)
	}

	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if len(sm.state.Nodes) != 2 {
		t.Fatalf("expected 2 imported anytls nodes, got %d", len(sm.state.Nodes))
	}

	foundALPN := false
	for _, node := range sm.state.Nodes {
		if node.Protocol != "anytls" {
			t.Fatalf("expected anytls protocol, got %q", node.Protocol)
		}
		if !strings.HasPrefix(node.URI, "anytls://dongtaiwang.com@45.221.98.14:59901") {
			t.Fatalf("unexpected canonical anytls uri: %q", node.URI)
		}
		if strings.Contains(node.URI, "alpn=h2") {
			foundALPN = true
		}
	}
	if !foundALPN {
		t.Fatal("expected one canonical AnyTLS URI to preserve alpn=h2")
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

func TestSubscriptionManagerRefreshSubscriptionKeepsMissingNodesInPlace(t *testing.T) {
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

	if err := sm.RefreshSubscription(subID); err != nil {
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

	if err := sm.RefreshSubscription(subID); err != nil {
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

	if statusByID[missingID] != NodeStatusStaging {
		t.Fatalf("expected missing node to stay staging, got %q", statusByID[missingID])
	}
	if reasonByID[missingID] != TransitionReasonSubscriptionNodeDiscovered {
		t.Fatalf("expected missing node reason to stay subscription_node_discovered, got %q", reasonByID[missingID])
	}
	if tagByID[missingID] == "" {
		t.Fatalf("expected missing node outbound tag to stay registered")
	}
	nodeByID := make(map[string]NodeRecord, len(nodes))
	for _, node := range nodes {
		nodeByID[node.ID] = node
	}
	if !nodeByID[missingID].SubscriptionMissing {
		t.Fatalf("expected missing node to be marked subscriptionMissing")
	}
	if statusByID[liveID] != NodeStatusStaging {
		t.Fatalf("expected live node to be staged, got %q", statusByID[liveID])
	}

	listed := sm.ListSubscriptions()
	if len(listed) != 1 {
		t.Fatalf("expected 1 subscription, got %d", len(listed))
	}
	if listed[0].NodeCount != 1 {
		t.Fatalf("expected subscription node count 1 after marking missing node, got %d", listed[0].NodeCount)
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

	if err := sm.RefreshSubscription(subID); err != nil {
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
	sm.state.Nodes[0].SubscriptionMissing = true
	sm.state.Nodes[0].OutboundTag = ""
	now := time.Now().Add(-time.Minute)
	sm.state.Nodes[0].StatusUpdatedAt = timePtr(now)
	sm.state.Nodes[0].LastEventAt = timePtr(now)
	sm.mu.Unlock()

	if err := sm.RefreshSubscription(subID); err != nil {
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
	if nodes[0].SubscriptionMissing {
		t.Fatalf("expected reintroduced node subscriptionMissing to be cleared")
	}
	if nodes[0].OutboundTag != probeOutboundTag(nodeID) {
		t.Fatalf("expected reintroduced node outbound tag %q, got %q", probeOutboundTag(nodeID), nodes[0].OutboundTag)
	}
	if len(handlerStub.addedTags) != 1 || handlerStub.addedTags[0] != probeOutboundTag(nodeID) {
		t.Fatalf("expected outbound %q to be added once, got %v", probeOutboundTag(nodeID), handlerStub.addedTags)
	}
}

func TestSubscriptionManagerRefreshSubscriptionDedupesSameBatchLinks(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	handlerStub := &stubHandlerServiceClient{}
	sm := NewSubscriptionManager(configPath, &GRPCClient{handlerClient: handlerStub}, nil, nil)
	defer sm.Stop()

	uri, err := GenerateShareLink(ShareLinkRequest{
		Protocol: "vless",
		Address:  "dedupe.example.com",
		Port:     443,
		UUID:     "66666666-6666-6666-6666-666666666666",
		Remark:   "dedupe-node",
		TLS:      "tls",
		SNI:      "dedupe.example.com",
	})
	if err != nil {
		t.Fatalf("generate share link: %v", err)
	}

	subID := "sub-dedupe"
	sm.mu.Lock()
	sm.state.Subscriptions = []SubscriptionRecord{
		{
			ID:         subID,
			SourceType: SubscriptionSourceManual,
			Content:    uri + "\n" + uri + "\n",
			Remark:     "manual duplicate sub",
		},
	}
	sm.mu.Unlock()

	if err := sm.RefreshSubscription(subID); err != nil {
		t.Fatalf("refresh subscription: %v", err)
	}

	nodes := sm.ListNodes("")
	if len(nodes) != 1 {
		t.Fatalf("expected duplicate links in one refresh to create 1 node, got %d: %#v", len(nodes), nodes)
	}
	if nodes[0].Address != "dedupe.example.com" {
		t.Fatalf("expected deduped node address, got %q", nodes[0].Address)
	}
	if len(handlerStub.addedTags) != 1 {
		t.Fatalf("expected one outbound registration for duplicate links, got %v", handlerStub.addedTags)
	}

	listed := sm.ListSubscriptions()
	if len(listed) != 1 {
		t.Fatalf("expected one subscription, got %d", len(listed))
	}
	if listed[0].NodeCount != 1 {
		t.Fatalf("expected node count 1 for duplicate subscription content, got %d", listed[0].NodeCount)
	}
}

func TestSubscriptionManagerUpdateSubscriptionMetadataKeepsNodeHistory(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	handlerStub := &stubHandlerServiceClient{}
	sm := NewSubscriptionManager(configPath, &GRPCClient{handlerClient: handlerStub}, nil, nil)
	defer sm.Stop()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(
			"vless://11111111-1111-1111-1111-111111111111@meta.example.com:443?security=tls&sni=meta.example.com#meta-node\n",
		))
	}))
	defer server.Close()

	sub, err := sm.AddSubscription(SubscriptionInput{
		SourceType:      SubscriptionSourceURL,
		URL:             server.URL,
		Remark:          "url sub",
		AutoRefresh:     true,
		RefreshInterval: 60,
	})
	if err != nil {
		t.Fatalf("add URL subscription: %v", err)
	}

	nodes := sm.ListNodes("")
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	originalNode := nodes[0]

	updated, err := sm.UpdateSubscription(sub.ID, SubscriptionUpdateInput{
		SourceType:      sourceTypePtr(SubscriptionSourceURL),
		Remark:          stringPtr("updated remark"),
		AutoRefresh:     boolPtr(false),
		RefreshInterval: intPtr(120),
	})
	if err != nil {
		t.Fatalf("update subscription metadata: %v", err)
	}

	if updated.ID != sub.ID {
		t.Fatalf("expected subscription ID to remain %q, got %q", sub.ID, updated.ID)
	}
	if updated.Remark != "updated remark" {
		t.Fatalf("expected updated remark, got %q", updated.Remark)
	}
	if updated.AutoRefresh {
		t.Fatal("expected auto-refresh to be disabled after pause")
	}
	if updated.RefreshInterval != 120 {
		t.Fatalf("expected refresh interval 120, got %d", updated.RefreshInterval)
	}

	nodes = sm.ListNodes("")
	if len(nodes) != 1 {
		t.Fatalf("expected node count to remain 1, got %d", len(nodes))
	}
	if nodes[0].ID != originalNode.ID {
		t.Fatalf("expected node ID %q to remain unchanged, got %q", originalNode.ID, nodes[0].ID)
	}
	if len(handlerStub.addedTags) != 1 {
		t.Fatalf("expected no extra outbound registrations during metadata-only update, got %v", handlerStub.addedTags)
	}
	if len(handlerStub.removedTags) != 0 {
		t.Fatalf("expected no outbound removals during metadata-only update, got %v", handlerStub.removedTags)
	}

	listed := sm.ListSubscriptions()
	if len(listed) != 1 {
		t.Fatalf("expected 1 subscription, got %d", len(listed))
	}
	if listed[0].ID != sub.ID {
		t.Fatalf("expected listed subscription ID %q, got %q", sub.ID, listed[0].ID)
	}
	if listed[0].AutoRefresh {
		t.Fatal("expected listed subscription auto-refresh to stay disabled")
	}
}

func TestSubscriptionManagerRefreshSubscriptionKeepsActiveMissingNodeInPlace(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	handlerStub := &stubHandlerServiceClient{}
	sm := NewSubscriptionManager(configPath, &GRPCClient{handlerClient: handlerStub}, nil, nil)
	defer sm.Stop()

	missingURI, err := GenerateShareLink(ShareLinkRequest{
		Protocol: "vless",
		Address:  "active-missing.example.com",
		Port:     443,
		UUID:     "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Remark:   "active-missing",
		TLS:      "tls",
		SNI:      "active-missing.example.com",
	})
	if err != nil {
		t.Fatalf("generate missing share link: %v", err)
	}
	liveURI, err := GenerateShareLink(ShareLinkRequest{
		Protocol: "vless",
		Address:  "active-live.example.com",
		Port:     443,
		UUID:     "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		Remark:   "active-live",
		TLS:      "tls",
		SNI:      "active-live.example.com",
	})
	if err != nil {
		t.Fatalf("generate live share link: %v", err)
	}

	subID := "sub-active-missing"
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

	if err := sm.RefreshSubscription(subID); err != nil {
		t.Fatalf("initial refresh subscription: %v", err)
	}

	nodes := sm.ListNodes("")
	missingID := ""
	for _, node := range nodes {
		if node.Address == "active-missing.example.com" {
			missingID = node.ID
			break
		}
	}
	if missingID == "" {
		t.Fatalf("expected missing node id after initial refresh")
	}

	sm.mu.Lock()
	for i := range sm.state.Nodes {
		if sm.state.Nodes[i].ID != missingID {
			continue
		}
		sm.state.Nodes[i].Status = NodeStatusActive
		sm.state.Nodes[i].StatusReason = TransitionReasonManualPromote
		sm.state.Nodes[i].OutboundTag = probeOutboundTag(missingID)
		break
	}
	sm.state.Subscriptions[0].Content = liveURI
	sm.mu.Unlock()

	if err := sm.RefreshSubscription(subID); err != nil {
		t.Fatalf("second refresh subscription: %v", err)
	}

	nodes = sm.ListNodes("")
	for _, node := range nodes {
		if node.ID != missingID {
			continue
		}
		if node.Status != NodeStatusActive {
			t.Fatalf("expected missing active node to stay active, got %q", node.Status)
		}
		if node.StatusReason != TransitionReasonManualPromote {
			t.Fatalf("expected missing active node reason to stay manual_promote, got %q", node.StatusReason)
		}
		if !node.SubscriptionMissing {
			t.Fatalf("expected missing active node to be marked subscriptionMissing")
		}
		if node.OutboundTag != probeOutboundTag(missingID) {
			t.Fatalf("expected missing active node outbound tag %q, got %q", probeOutboundTag(missingID), node.OutboundTag)
		}
		return
	}

	t.Fatalf("missing active node %q not found after refresh", missingID)
}

func TestSubscriptionManagerUpdateSubscriptionURLReconcilesNodesInPlace(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	handlerStub := &stubHandlerServiceClient{}
	sm := NewSubscriptionManager(configPath, &GRPCClient{handlerClient: handlerStub}, nil, nil)
	defer sm.Stop()

	serverA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(
			"vless://22222222-2222-2222-2222-222222222222@update-a.example.com:443?security=tls&sni=update-a.example.com#update-a\n",
		))
	}))
	defer serverA.Close()

	serverB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(
			"vless://33333333-3333-3333-3333-333333333333@update-b.example.com:443?security=tls&sni=update-b.example.com#update-b\n",
		))
	}))
	defer serverB.Close()

	sub, err := sm.AddSubscription(SubscriptionInput{
		SourceType:      SubscriptionSourceURL,
		URL:             serverA.URL,
		Remark:          "url sub",
		AutoRefresh:     false,
		RefreshInterval: 60,
	})
	if err != nil {
		t.Fatalf("add URL subscription: %v", err)
	}

	nodes := sm.ListNodes("")
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	originalNodeID := nodes[0].ID

	updated, err := sm.UpdateSubscription(sub.ID, SubscriptionUpdateInput{
		URL: stringPtr(serverB.URL),
	})
	if err != nil {
		t.Fatalf("update subscription URL: %v", err)
	}

	if updated.ID != sub.ID {
		t.Fatalf("expected subscription ID to remain %q, got %q", sub.ID, updated.ID)
	}
	if updated.URL != serverB.URL {
		t.Fatalf("expected updated URL %q, got %q", serverB.URL, updated.URL)
	}

	nodes = sm.ListNodes("")
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes after reconciliation, got %d", len(nodes))
	}

	statusByID := make(map[string]NodeStatus, len(nodes))
	reasonByID := make(map[string]TransitionReason, len(nodes))
	subscriptionByID := make(map[string]string, len(nodes))
	addressByID := make(map[string]string, len(nodes))
	for _, node := range nodes {
		statusByID[node.ID] = node.Status
		reasonByID[node.ID] = node.StatusReason
		subscriptionByID[node.ID] = node.SubscriptionID
		addressByID[node.ID] = node.Address
	}

	if statusByID[originalNodeID] != NodeStatusStaging {
		t.Fatalf("expected original node to stay staging, got %q", statusByID[originalNodeID])
	}
	if reasonByID[originalNodeID] != TransitionReasonSubscriptionNodeDiscovered {
		t.Fatalf("expected original node reason subscription_node_discovered, got %q", reasonByID[originalNodeID])
	}
	if subscriptionByID[originalNodeID] != sub.ID {
		t.Fatalf("expected original node subscription ID %q, got %q", sub.ID, subscriptionByID[originalNodeID])
	}
	for _, node := range nodes {
		if node.ID == originalNodeID && !node.SubscriptionMissing {
			t.Fatalf("expected original node to be marked subscriptionMissing")
		}
	}

	replacementCount := 0
	for nodeID, address := range addressByID {
		if address != "update-b.example.com" {
			continue
		}
		replacementCount++
		if statusByID[nodeID] != NodeStatusStaging {
			t.Fatalf("expected replacement node to be staging, got %q", statusByID[nodeID])
		}
		if reasonByID[nodeID] != TransitionReasonSubscriptionNodeDiscovered {
			t.Fatalf("expected replacement node reason subscription_node_discovered, got %q", reasonByID[nodeID])
		}
		if subscriptionByID[nodeID] != sub.ID {
			t.Fatalf("expected replacement node subscription ID %q, got %q", sub.ID, subscriptionByID[nodeID])
		}
	}
	if replacementCount != 1 {
		t.Fatalf("expected 1 replacement node, got %d", replacementCount)
	}

	if len(handlerStub.addedTags) != 2 {
		t.Fatalf("expected two outbound registrations after URL update, got %v", handlerStub.addedTags)
	}
}

func TestSubscriptionManagerUpdateSubscriptionRejectsDuplicateURL(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	handlerStub := &stubHandlerServiceClient{}
	sm := NewSubscriptionManager(configPath, &GRPCClient{handlerClient: handlerStub}, nil, nil)
	defer sm.Stop()

	serverA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(
			"vless://44444444-4444-4444-4444-444444444444@dup-a.example.com:443?security=tls&sni=dup-a.example.com#dup-a\n",
		))
	}))
	defer serverA.Close()

	serverB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(
			"vless://55555555-5555-5555-5555-555555555555@dup-b.example.com:443?security=tls&sni=dup-b.example.com#dup-b\n",
		))
	}))
	defer serverB.Close()

	first, err := sm.AddSubscription(SubscriptionInput{
		SourceType:      SubscriptionSourceURL,
		URL:             serverA.URL,
		Remark:          "first",
		AutoRefresh:     false,
		RefreshInterval: 60,
	})
	if err != nil {
		t.Fatalf("add first subscription: %v", err)
	}
	second, err := sm.AddSubscription(SubscriptionInput{
		SourceType:      SubscriptionSourceURL,
		URL:             serverB.URL,
		Remark:          "second",
		AutoRefresh:     false,
		RefreshInterval: 60,
	})
	if err != nil {
		t.Fatalf("add second subscription: %v", err)
	}

	if _, err := sm.UpdateSubscription(first.ID, SubscriptionUpdateInput{URL: stringPtr(serverB.URL)}); err == nil {
		t.Fatal("expected duplicate URL update to fail")
	}

	listed := sm.ListSubscriptions()
	if len(listed) != 2 {
		t.Fatalf("expected 2 subscriptions, got %d", len(listed))
	}
	urlByID := map[string]string{}
	for _, sub := range listed {
		urlByID[sub.ID] = sub.URL
	}
	if urlByID[first.ID] != serverA.URL {
		t.Fatalf("expected first subscription URL to remain %q, got %q", serverA.URL, urlByID[first.ID])
	}
	if urlByID[second.ID] != serverB.URL {
		t.Fatalf("expected second subscription URL to remain %q, got %q", serverB.URL, urlByID[second.ID])
	}
}

func TestSubscriptionManagerUpdateManualSubscriptionOnlyAllowsRemark(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	sm := NewSubscriptionManager(configPath, nil, nil, nil)
	defer sm.Stop()

	sub, err := sm.AddSubscription(SubscriptionInput{
		SourceType: SubscriptionSourceManual,
		Content:    "vless://88888888-8888-8888-8888-888888888888@manual.example.com:443?security=tls&sni=manual.example.com#manual-node",
		Remark:     "manual source",
	})
	if err != nil {
		t.Fatalf("add manual subscription: %v", err)
	}

	updated, err := sm.UpdateSubscription(sub.ID, SubscriptionUpdateInput{
		SourceType: sourceTypePtr(SubscriptionSourceManual),
		Remark:     stringPtr("manual source updated"),
	})
	if err != nil {
		t.Fatalf("update manual remark: %v", err)
	}
	if updated.ID != sub.ID {
		t.Fatalf("expected ID %q to remain unchanged, got %q", sub.ID, updated.ID)
	}
	if updated.Remark != "manual source updated" {
		t.Fatalf("expected updated remark, got %q", updated.Remark)
	}

	if _, err := sm.UpdateSubscription(sub.ID, SubscriptionUpdateInput{
		URL: stringPtr("https://example.com/sub.txt"),
	}); err == nil {
		t.Fatal("expected manual source URL update to fail")
	}
	if _, err := sm.UpdateSubscription(sub.ID, SubscriptionUpdateInput{
		AutoRefresh: boolPtr(true),
	}); err == nil {
		t.Fatal("expected manual source autoRefresh update to fail")
	}
}

func boolPtr(value bool) *bool {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func stringPtr(value string) *string {
	return &value
}

func sourceTypePtr(value SubscriptionSourceType) *SubscriptionSourceType {
	return &value
}

func timePtr(value time.Time) *time.Time {
	return &value
}

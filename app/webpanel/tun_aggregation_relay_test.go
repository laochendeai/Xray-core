package webpanel

import (
	"testing"
	"time"
)

func TestTunAggregationRelayAssemblerDedupesAndFlushesReorderedPackets(t *testing.T) {
	t.Parallel()

	startedAt := time.Unix(1700000000, 0).UTC()
	assembler := newTunAggregationRelayAssembler()

	assembler.ingest(tunAggregationRelayEnvelope{
		SessionID:    "session-1",
		Sequence:     1,
		PathID:       "path-b",
		SentAt:       startedAt,
		ArrivedAt:    startedAt.Add(22 * time.Millisecond),
		PayloadBytes: 1200,
	})
	assembler.ingest(tunAggregationRelayEnvelope{
		SessionID:    "session-1",
		Sequence:     0,
		PathID:       "path-a",
		SentAt:       startedAt,
		ArrivedAt:    startedAt.Add(11 * time.Millisecond),
		PayloadBytes: 1200,
	})
	assembler.ingest(tunAggregationRelayEnvelope{
		SessionID:    "session-1",
		Sequence:     1,
		PathID:       "path-a",
		SentAt:       startedAt,
		ArrivedAt:    startedAt.Add(24 * time.Millisecond),
		PayloadBytes: 1200,
	})

	result := tunAggregationSummarizeSimulation(assembler, startedAt, 2)
	if result.DeliveredPacketCount != 2 {
		t.Fatalf("expected 2 delivered packets, got %#v", result)
	}
	if result.DuplicateDrops != 1 {
		t.Fatalf("expected 1 duplicate drop, got %#v", result)
	}
	if result.ReorderedPackets != 1 {
		t.Fatalf("expected 1 reordered packet, got %#v", result)
	}
	if result.MaxReorderBufferDepth != 1 {
		t.Fatalf("expected max reorder depth 1, got %#v", result)
	}
	if result.StartupLatencyMs != 11 {
		t.Fatalf("expected startup latency 11ms, got %#v", result)
	}
}

func TestBuildTunAggregationRelayStatusUsesPrototypeContract(t *testing.T) {
	t.Parallel()

	now := time.Unix(1700000100, 0).UTC()
	prototype := buildTunAggregationPrototype(TunAggregationSettings{
		Enabled:            true,
		MaxPathsPerSession: 2,
		SchedulerPolicy:    string(TunAggregationSchedulerPolicyWeightedSplit),
		RelayEndpoint:      "https://relay.example/ingress",
	}, []NodeRecord{
		{
			ID:            "node-fast",
			Remark:        "node-fast",
			Protocol:      "vmess",
			AvgDelayMs:    28,
			TotalPings:    10,
			FailedPings:   0,
			LastCheckedAt: &now,
		},
		{
			ID:            "node-mid",
			Remark:        "node-mid",
			Protocol:      "vmess",
			AvgDelayMs:    52,
			TotalPings:    10,
			FailedPings:   0,
			LastCheckedAt: &now,
		},
	}, now)
	if prototype == nil {
		t.Fatal("expected prototype")
	}

	status := buildTunAggregationRelayStatus(TunAggregationSettings{
		Enabled:            true,
		MaxPathsPerSession: 2,
		SchedulerPolicy:    string(TunAggregationSchedulerPolicyWeightedSplit),
		RelayEndpoint:      "https://relay.example/ingress",
	}, prototype, now)
	if status == nil {
		t.Fatal("expected relay status")
	}
	if !status.Ready {
		t.Fatalf("expected ready relay status, got %#v", status)
	}
	if status.ContractVersion != tunAggregationRelayContractVersion {
		t.Fatalf("expected contract version %q, got %#v", tunAggregationRelayContractVersion, status)
	}
	if status.SessionCount != 1 || len(status.Sessions) != 1 {
		t.Fatalf("expected one relay session summary, got %#v", status)
	}
	if status.PacketCount != tunAggregationBenchmarkPacketCount {
		t.Fatalf("expected %d simulated packets, got %#v", tunAggregationBenchmarkPacketCount, status)
	}
	if status.DeliveredPacketCount == 0 {
		t.Fatalf("expected relay status to deliver packets, got %#v", status)
	}
	if len(status.Sessions[0].PathIDs) != 2 {
		t.Fatalf("expected two selected paths in relay status, got %#v", status.Sessions[0])
	}
}

func TestBuildTunAggregationBenchmarkStatusShowsDegradedPathBenefit(t *testing.T) {
	t.Parallel()

	now := time.Unix(1700000200, 0).UTC()
	prototype := buildTunAggregationPrototype(TunAggregationSettings{
		Enabled:            true,
		MaxPathsPerSession: 2,
		SchedulerPolicy:    string(TunAggregationSchedulerPolicyRedundant2),
		RelayEndpoint:      "https://relay.example/ingress",
	}, []NodeRecord{
		{
			ID:            "node-fast",
			Remark:        "node-fast",
			Protocol:      "vmess",
			AvgDelayMs:    24,
			TotalPings:    12,
			FailedPings:   0,
			LastCheckedAt: &now,
		},
		{
			ID:            "node-backup",
			Remark:        "node-backup",
			Protocol:      "vmess",
			AvgDelayMs:    46,
			TotalPings:    12,
			FailedPings:   0,
			LastCheckedAt: &now,
		},
	}, now)
	if prototype == nil {
		t.Fatal("expected prototype")
	}

	benchmark := buildTunAggregationBenchmarkStatus(TunAggregationSettings{
		Enabled:            true,
		MaxPathsPerSession: 2,
		SchedulerPolicy:    string(TunAggregationSchedulerPolicyRedundant2),
		RelayEndpoint:      "https://relay.example/ingress",
	}, prototype, now)
	if benchmark == nil || !benchmark.Ready {
		t.Fatalf("expected ready benchmark status, got %#v", benchmark)
	}
	if len(benchmark.Scenarios) != 2 {
		t.Fatalf("expected two benchmark scenarios, got %#v", benchmark)
	}

	var degraded *TunAggregationBenchmarkScenarioStatus
	for i := range benchmark.Scenarios {
		if benchmark.Scenarios[i].Name == string(TunAggregationBenchmarkScenarioDegradedPrimary) {
			degraded = &benchmark.Scenarios[i]
			break
		}
	}
	if degraded == nil {
		t.Fatalf("expected degraded scenario, got %#v", benchmark.Scenarios)
	}
	t.Logf(
		"degraded_primary baseline(startup=%dms stalls=%d goodput=%.1fkbps loss=%.1f%% stability=%.1f%%) aggregated(startup=%dms stalls=%d goodput=%.1fkbps loss=%.1f%% stability=%.1f%%)",
		degraded.Baseline.StartupLatencyMs,
		degraded.Baseline.StallCount,
		degraded.Baseline.GoodputKbps,
		degraded.Baseline.LossPct,
		degraded.Baseline.StabilityPct,
		degraded.Aggregated.StartupLatencyMs,
		degraded.Aggregated.StallCount,
		degraded.Aggregated.GoodputKbps,
		degraded.Aggregated.LossPct,
		degraded.Aggregated.StabilityPct,
	)
	if degraded.GoodputGainKbps <= 0 && degraded.LossReductionPct <= 0 && degraded.StallReduction <= 0 {
		t.Fatalf("expected degraded scenario to show at least one improvement, got %#v", degraded)
	}
}

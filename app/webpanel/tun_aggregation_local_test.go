package webpanel

import (
	"testing"
	"time"
)

func TestBuildTunAggregationPrototypeWeightedSplitUsesEligiblePaths(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 9, 12, 0, 0, 0, time.UTC)
	checkedAt := now.Add(-2 * time.Minute)
	staleAt := now.Add(-30 * time.Minute)

	prototype := buildTunAggregationPrototype(TunAggregationSettings{
		Enabled:            true,
		SchedulerPolicy:    string(TunAggregationSchedulerPolicyWeightedSplit),
		MaxPathsPerSession: 2,
	}, []NodeRecord{
		{
			ID:            "node-fast",
			URI:           mustGenerateTunTestURI(t, "203.0.113.11"),
			Remark:        "node-fast",
			Protocol:      "vmess",
			AvgDelayMs:    20,
			TotalPings:    10,
			LastCheckedAt: &checkedAt,
		},
		{
			ID:            "node-mid",
			URI:           mustGenerateTunTestURI(t, "203.0.113.12"),
			Remark:        "node-mid",
			Protocol:      "vmess",
			AvgDelayMs:    45,
			TotalPings:    10,
			FailedPings:   1,
			LastCheckedAt: &checkedAt,
		},
		{
			ID:            "node-slow",
			URI:           mustGenerateTunTestURI(t, "203.0.113.13"),
			Remark:        "node-slow",
			Protocol:      "vmess",
			AvgDelayMs:    80,
			TotalPings:    10,
			LastCheckedAt: &checkedAt,
		},
		{
			ID:            "node-stale",
			URI:           mustGenerateTunTestURI(t, "203.0.113.14"),
			Remark:        "node-stale",
			Protocol:      "vmess",
			AvgDelayMs:    10,
			TotalPings:    10,
			LastCheckedAt: &staleAt,
		},
	}, now)
	if prototype == nil {
		t.Fatal("expected aggregation prototype")
	}
	if !prototype.Ready {
		t.Fatalf("expected ready prototype, got %#v", prototype)
	}
	if prototype.CandidatePathCount != 3 {
		t.Fatalf("expected 3 candidate paths, got %#v", prototype)
	}
	if prototype.SelectedPathCount != 2 {
		t.Fatalf("expected 2 selected paths, got %#v", prototype)
	}
	if prototype.SessionCount != 1 || len(prototype.Sessions) != 1 {
		t.Fatalf("expected one preview session, got %#v", prototype.Sessions)
	}
	if len(prototype.Paths) != 4 {
		t.Fatalf("expected 4 path snapshots, got %#v", prototype.Paths)
	}

	if prototype.Paths[0].NodeID != "node-fast" || prototype.Paths[0].State != "selected" {
		t.Fatalf("expected fastest node to be selected first, got %#v", prototype.Paths[0])
	}
	if prototype.Paths[1].NodeID != "node-mid" || prototype.Paths[1].State != "selected" {
		t.Fatalf("expected second path to be selected, got %#v", prototype.Paths[1])
	}
	if prototype.Paths[2].NodeID != "node-slow" || prototype.Paths[2].State != "standby" {
		t.Fatalf("expected third eligible path to stay standby, got %#v", prototype.Paths[2])
	}
	if prototype.Paths[3].NodeID != "node-stale" || prototype.Paths[3].State != "excluded" {
		t.Fatalf("expected stale path to be excluded, got %#v", prototype.Paths[3])
	}
}

func TestTunAggregationSessionStoreExpiresSessions(t *testing.T) {
	t.Parallel()

	store := newTunAggregationSessionStore(10 * time.Second)
	now := time.Date(2026, time.April, 9, 12, 5, 0, 0, time.UTC)

	store.UpsertPreviewSession(
		tunAggregationPreviewFlowKey{Protocol: "quic", Target: "preview.local", Port: 443},
		"QUIC preview session",
		string(TunAggregationSchedulerPolicySingleBest),
		[]string{"node-1", "node-2"},
		[]string{"node-1"},
		now,
		"single_best chose 1 of 2 eligible path(s).",
	)

	snapshots := store.Snapshot(now.Add(5 * time.Second))
	if len(snapshots) != 1 {
		t.Fatalf("expected live session snapshot, got %#v", snapshots)
	}

	expired := store.Snapshot(now.Add(11 * time.Second))
	if len(expired) != 0 {
		t.Fatalf("expected expired sessions to be removed, got %#v", expired)
	}
}

package webpanel

import (
	"path/filepath"
	"testing"
)

func TestControlPlaneStateStorePersistsTransitions(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.json")
	store := NewControlPlaneStateStore(configPath)

	initial := store.Snapshot()
	if initial.MachineState != MachineStateClean {
		t.Fatalf("expected clean default state, got %q", initial.MachineState)
	}
	if initial.LastStateReason != MachineReasonStartupDefaultClean {
		t.Fatalf("expected default startup reason, got %q", initial.LastStateReason)
	}

	store.Transition(MachineStateProxied, MachineReasonOperatorEnabled, EventActorOperator, "test enable")

	reloaded := NewControlPlaneStateStore(configPath)
	snapshot := reloaded.Snapshot()
	if snapshot.MachineState != MachineStateProxied {
		t.Fatalf("expected proxied state after reload, got %q", snapshot.MachineState)
	}
	if snapshot.LastStateReason != MachineReasonOperatorEnabled {
		t.Fatalf("expected operator enabled reason after reload, got %q", snapshot.LastStateReason)
	}
	if len(snapshot.RecentMachineEvents) != 1 {
		t.Fatalf("expected 1 machine event, got %d", len(snapshot.RecentMachineEvents))
	}
	if snapshot.RecentMachineEvents[0].Details != "test enable" {
		t.Fatalf("expected persisted event details, got %q", snapshot.RecentMachineEvents[0].Details)
	}
}

package webpanel

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/xtls/xray-core/common/errors"
)

type MachineState string

const (
	MachineStateClean      MachineState = "clean"
	MachineStateProxied    MachineState = "proxied"
	MachineStateDegraded   MachineState = "degraded"
	MachineStateRecovering MachineState = "recovering"
)

type MachineStateReason string

const (
	MachineReasonStartupDefaultClean          MachineStateReason = "startup_default_clean"
	MachineReasonStartupStatusUnavailable     MachineStateReason = "startup_status_unavailable"
	MachineReasonStartupCleanupFailed         MachineStateReason = "startup_cleanup_failed"
	MachineReasonOperatorEnabled              MachineStateReason = "operator_enabled"
	MachineReasonTunStartFailed               MachineStateReason = "tun_start_failed"
	MachineReasonOperatorRestoreClean         MachineStateReason = "operator_restore_clean"
	MachineReasonEnableBlockedMinActiveNotMet MachineStateReason = "enable_blocked_min_active_not_met"
	MachineReasonPoolBelowMinActiveNodes      MachineStateReason = "pool_below_min_active_nodes"
	MachineReasonAutomaticFallbackMinActive   MachineStateReason = "automatic_fallback_min_active_not_met"
	MachineReasonFallbackFailed               MachineStateReason = "fallback_failed"
	MachineReasonStateLoadDefaulted           MachineStateReason = "state_load_defaulted"
)

type MachineEvent struct {
	State   MachineState       `json:"state"`
	Reason  MachineStateReason `json:"reason"`
	Actor   EventActor         `json:"actor"`
	At      time.Time          `json:"at"`
	Details string             `json:"details,omitempty"`
}

type ControlPlaneState struct {
	MachineState        MachineState       `json:"machineState"`
	LastStateReason     MachineStateReason `json:"lastStateReason"`
	LastStateChangedAt  time.Time          `json:"lastStateChangedAt"`
	RecentMachineEvents []MachineEvent     `json:"recentMachineEvents,omitempty"`
}

type ControlPlaneStateStore struct {
	mu    sync.RWMutex
	path  string
	state *ControlPlaneState
}

const machineEventLimit = 25

func NewControlPlaneStateStore(configPath string) *ControlPlaneStateStore {
	store := &ControlPlaneStateStore{
		path: filepath.Join(filepath.Dir(configPath), "control_plane_state.json"),
	}
	store.state = store.load()
	return store
}

func (s *ControlPlaneStateStore) Snapshot() ControlPlaneState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return copyControlPlaneState(*s.state)
}

func (s *ControlPlaneStateStore) Transition(nextState MachineState, reason MachineStateReason, actor EventActor, details string) ControlPlaneState {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.state.MachineState = nextState
	s.state.LastStateReason = reason
	s.state.LastStateChangedAt = now
	s.state.RecentMachineEvents = append(s.state.RecentMachineEvents, MachineEvent{
		State:   nextState,
		Reason:  reason,
		Actor:   actor,
		At:      now,
		Details: details,
	})
	if len(s.state.RecentMachineEvents) > machineEventLimit {
		s.state.RecentMachineEvents = append([]MachineEvent(nil), s.state.RecentMachineEvents[len(s.state.RecentMachineEvents)-machineEventLimit:]...)
	}
	s.writeLocked()
	return copyControlPlaneState(*s.state)
}

func (s *ControlPlaneStateStore) load() *ControlPlaneState {
	defaultState := defaultControlPlaneState()
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return defaultState
	}

	state := &ControlPlaneState{}
	if err := json.Unmarshal(raw, state); err != nil {
		errors.LogWarning(context.Background(), "control plane: failed to parse state file: ", err.Error())
		defaultState.RecentMachineEvents = append(defaultState.RecentMachineEvents, MachineEvent{
			State:   defaultState.MachineState,
			Reason:  MachineReasonStateLoadDefaulted,
			Actor:   EventActorSystem,
			At:      time.Now(),
			Details: err.Error(),
		})
		return defaultState
	}

	if state.MachineState == "" {
		state.MachineState = MachineStateClean
	}
	if state.LastStateReason == "" {
		state.LastStateReason = MachineReasonStartupDefaultClean
	}
	if state.LastStateChangedAt.IsZero() {
		state.LastStateChangedAt = time.Now()
	}
	if len(state.RecentMachineEvents) > machineEventLimit {
		state.RecentMachineEvents = append([]MachineEvent(nil), state.RecentMachineEvents[len(state.RecentMachineEvents)-machineEventLimit:]...)
	}
	return state
}

func (s *ControlPlaneStateStore) writeLocked() {
	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		errors.LogWarning(context.Background(), "control plane: failed to marshal state: ", err.Error())
		return
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		errors.LogWarning(context.Background(), "control plane: failed to create state dir: ", err.Error())
		return
	}

	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		errors.LogWarning(context.Background(), "control plane: failed to write state: ", err.Error())
	}
}

func defaultControlPlaneState() *ControlPlaneState {
	now := time.Now()
	return &ControlPlaneState{
		MachineState:       MachineStateClean,
		LastStateReason:    MachineReasonStartupDefaultClean,
		LastStateChangedAt: now,
	}
}

func copyControlPlaneState(state ControlPlaneState) ControlPlaneState {
	cloned := state
	if len(state.RecentMachineEvents) > 0 {
		cloned.RecentMachineEvents = append([]MachineEvent(nil), state.RecentMachineEvents...)
	}
	return cloned
}

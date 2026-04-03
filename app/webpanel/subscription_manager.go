package webpanel

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	handlerservice "github.com/xtls/xray-core/app/proxyman/command"
	"github.com/xtls/xray-core/common/errors"
	core "github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/features/routing"
)

type NodeStatus string

const (
	NodeStatusCandidate  NodeStatus = "candidate"
	NodeStatusStaging    NodeStatus = "staging"
	NodeStatusActive     NodeStatus = "active"
	NodeStatusQuarantine NodeStatus = "quarantine"
	NodeStatusRemoved    NodeStatus = "removed"
)

type TransitionReason string

const (
	TransitionReasonSubscriptionNodeDiscovered TransitionReason = "subscription_node_discovered"
	TransitionReasonOutboundRegistrationFailed TransitionReason = "outbound_registration_failed"
	TransitionReasonProbeQualified             TransitionReason = "probe_qualified"
	TransitionReasonProbeRequalified           TransitionReason = "probe_requalified"
	TransitionReasonProbeFailuresExceeded      TransitionReason = "probe_failures_exceeded"
	TransitionReasonManualValidate             TransitionReason = "manual_validate"
	TransitionReasonManualPromote              TransitionReason = "manual_promote"
	TransitionReasonManualRestore              TransitionReason = "manual_restore"
	TransitionReasonManualQuarantine           TransitionReason = "manual_quarantine"
	TransitionReasonManualRemove               TransitionReason = "manual_remove"
	TransitionReasonSubscriptionMissing        TransitionReason = "subscription_missing"
	TransitionReasonSubscriptionDeleted        TransitionReason = "subscription_deleted"
	TransitionReasonSubscriptionReintroduced   TransitionReason = "subscription_reintroduced"
	TransitionReasonMigrationLegacyDemoted     TransitionReason = "migration_legacy_demoted"
)

type EventActor string

const (
	EventActorSystem    EventActor = "system"
	EventActorOperator  EventActor = "operator"
	EventActorMigration EventActor = "migration"
)

type CleanlinessStatus string

const (
	CleanlinessUnknown    CleanlinessStatus = "unknown"
	CleanlinessTrusted    CleanlinessStatus = "trusted"
	CleanlinessSuspicious CleanlinessStatus = "suspicious"
)

type BandwidthTier string

const (
	BandwidthTierUnknown BandwidthTier = "unknown"
)

type NodeEvent struct {
	NodeID   string           `json:"nodeId"`
	Remark   string           `json:"remark,omitempty"`
	Status   NodeStatus       `json:"status"`
	Reason   TransitionReason `json:"reason"`
	Actor    EventActor       `json:"actor"`
	At       time.Time        `json:"at"`
	Details  string           `json:"details,omitempty"`
	NodeAddr string           `json:"nodeAddress,omitempty"`
}

type PoolHealthSummary struct {
	ActiveNodes     int       `json:"activeNodes"`
	MinActiveNodes  int       `json:"minActiveNodes"`
	Healthy         bool      `json:"healthy"`
	LastEvaluatedAt time.Time `json:"lastEvaluatedAt"`
}

type NodePoolSummary struct {
	CandidateCount      int              `json:"candidateCount"`
	StagingCount        int              `json:"stagingCount"`
	ActiveCount         int              `json:"activeCount"`
	QuarantineCount     int              `json:"quarantineCount"`
	RemovedCount        int              `json:"removedCount"`
	TrustedCount        int              `json:"trustedCount"`
	SuspiciousCount     int              `json:"suspiciousCount"`
	UnknownCleanCount   int              `json:"unknownCleanCount"`
	ActiveNodes         int              `json:"activeNodes"`
	MinActiveNodes      int              `json:"minActiveNodes"`
	Healthy             bool             `json:"healthy"`
	LastEvaluatedAt     time.Time        `json:"lastEvaluatedAt"`
	LatestEventAt       *time.Time       `json:"latestEventAt,omitempty"`
	LatestEventReason   TransitionReason `json:"latestEventReason,omitempty"`
	LatestEventStatus   NodeStatus       `json:"latestEventStatus,omitempty"`
	LatestEventActor    EventActor       `json:"latestEventActor,omitempty"`
	LatestEventNodeID   string           `json:"latestEventNodeId,omitempty"`
	LatestEventNodeAddr string           `json:"latestEventNodeAddress,omitempty"`
}

type BulkRemoveFilter struct {
	IDs          []string            `json:"ids,omitempty"`
	Statuses     []NodeStatus        `json:"statuses,omitempty"`
	Cleanliness  []CleanlinessStatus `json:"cleanliness,omitempty"`
	OnlyUnstable bool                `json:"onlyUnstable,omitempty"`
}

type BulkPromoteRequest struct {
	IDs []string `json:"ids,omitempty"`
}

type BulkValidateRequest struct {
	IDs []string `json:"ids,omitempty"`
}

type BulkPurgeRemovedRequest struct {
	IDs []string `json:"ids,omitempty"`
}

type BulkRestoreRequest struct {
	IDs []string `json:"ids,omitempty"`
}

type SubscriptionSourceType string

const (
	SubscriptionSourceURL    SubscriptionSourceType = "url"
	SubscriptionSourceManual SubscriptionSourceType = "manual"
	SubscriptionSourceFile   SubscriptionSourceType = "file"
)

// NodePoolState is the top-level persisted state.
type NodePoolState struct {
	Subscriptions    []SubscriptionRecord `json:"subscriptions"`
	Nodes            []NodeRecord         `json:"nodes"`
	ValidationConfig ValidationConfig     `json:"validationConfig"`
	RecentNodeEvents []NodeEvent          `json:"recentNodeEvents,omitempty"`
}

// SubscriptionRecord represents a subscription source.
type SubscriptionRecord struct {
	ID              string                 `json:"id"`
	SourceType      SubscriptionSourceType `json:"sourceType,omitempty"`
	URL             string                 `json:"url,omitempty"`
	Content         string                 `json:"content,omitempty"`
	SourceName      string                 `json:"sourceName,omitempty"`
	Remark          string                 `json:"remark"`
	AutoRefresh     bool                   `json:"autoRefresh"`
	RefreshInterval int                    `json:"refreshIntervalMin"`
	LastRefresh     *time.Time             `json:"lastRefresh,omitempty"`
	NodeCount       int                    `json:"nodeCount"`
}

type SubscriptionInput struct {
	URL             string
	Content         string
	SourceName      string
	SourceType      SubscriptionSourceType
	Remark          string
	AutoRefresh     bool
	RefreshInterval int
}

type SubscriptionUpdateInput struct {
	SourceType      *SubscriptionSourceType
	URL             *string
	Remark          *string
	AutoRefresh     *bool
	RefreshInterval *int
}

// NodeRecord represents a node in the pool.
type NodeRecord struct {
	ID               string            `json:"id"`
	URI              string            `json:"uri"`
	Remark           string            `json:"remark"`
	Protocol         string            `json:"protocol"`
	Address          string            `json:"address"`
	Port             int               `json:"port"`
	OutboundTag      string            `json:"outboundTag"`
	Status           NodeStatus        `json:"status"`
	StatusReason     TransitionReason  `json:"statusReason"`
	SubscriptionID   string            `json:"subscriptionId"`
	AddedAt          time.Time         `json:"addedAt"`
	PromotedAt       *time.Time        `json:"promotedAt,omitempty"`
	StatusUpdatedAt  *time.Time        `json:"statusUpdatedAt,omitempty"`
	LastEventAt      *time.Time        `json:"lastEventAt,omitempty"`
	TotalPings       int               `json:"totalPings"`
	FailedPings      int               `json:"failedPings"`
	AvgDelayMs       int64             `json:"avgDelayMs"`
	ConsecutiveFails int               `json:"consecutiveFails"`
	LastCheckedAt    *time.Time        `json:"lastCheckedAt,omitempty"`
	Cleanliness      CleanlinessStatus `json:"cleanliness"`
	BandwidthTier    BandwidthTier     `json:"bandwidthTier"`
}

// ValidationConfig holds the criteria for promoting/quarantining nodes.
type ValidationConfig struct {
	MinSamples        int     `json:"minSamples"`
	MaxFailRate       float64 `json:"maxFailRate"`
	MaxAvgDelayMs     int64   `json:"maxAvgDelayMs"`
	ProbeIntervalSec  int     `json:"probeIntervalSec"`
	ProbeURL          string  `json:"probeUrl"`
	DemoteAfterFails  int     `json:"demoteAfterFails"`
	AutoRemoveDemoted bool    `json:"autoRemoveDemoted"`
	MinActiveNodes    int     `json:"minActiveNodes"`
	MinBandwidthKbps  int     `json:"minBandwidthKbps"`
}

// SubscriptionManager manages subscriptions and the node pool lifecycle.
type SubscriptionManager struct {
	mu                  sync.RWMutex
	state               *NodePoolState
	statePath           string
	grpcClient          *GRPCClient
	prober              *NodeProber
	instance            *core.Instance
	runtimeCtx          context.Context
	stopCh              chan struct{}
	refreshMu           sync.Mutex
	saveCh              chan struct{}
	saveDelay           time.Duration
	onPoolHealthChange  func(PoolHealthSummary)
	bgWG                sync.WaitGroup
	started             bool
	dispatcherAvailable bool
}

const (
	nodeEventLimit         = 25
	scheduledPersistDelay  = 350 * time.Millisecond
	nodeProbeTagPrefix     = "pool_"
	legacyDemotedStatus    = "demoted"
	legacyStagingTagPrefix = "staging_"
	legacyActiveTagPrefix  = "active_"
)

// NewSubscriptionManager creates a new SubscriptionManager.
func NewSubscriptionManager(configPath string, grpcClient *GRPCClient, instance *core.Instance, runtimeCtx context.Context) *SubscriptionManager {
	statePath := filepath.Join(filepath.Dir(configPath), "node_pool_state.json")

	sm := &SubscriptionManager{
		statePath:  statePath,
		grpcClient: grpcClient,
		instance:   instance,
		runtimeCtx: runtimeCtx,
		stopCh:     make(chan struct{}),
		saveCh:     make(chan struct{}, 1),
		saveDelay:  scheduledPersistDelay,
	}

	state, changed := sm.loadState()
	sm.state = state
	if changed {
		sm.mu.Lock()
		sm.writeStateLocked()
		sm.mu.Unlock()
	}

	sm.bgWG.Add(1)
	go func() {
		defer sm.bgWG.Done()
		sm.persistLoop()
	}()
	return sm
}

// Start initializes the prober and begins auto-refresh loops.
func (sm *SubscriptionManager) Start() error {
	var dispatcher routing.Dispatcher
	if sm.instance != nil {
		if f := sm.instance.GetFeature(routing.DispatcherType()); f != nil {
			dispatcher = f.(routing.Dispatcher)
		}
	}

	sm.mu.Lock()
	normalized := sm.normalizeProbeStateLocked()
	cfg := sm.state.ValidationConfig
	tags := make([]string, 0, len(sm.state.Nodes))
	for _, n := range sm.state.Nodes {
		if isProbeableStatus(n.Status) && n.OutboundTag != "" {
			tags = append(tags, n.OutboundTag)
		}
	}
	sm.started = true
	sm.dispatcherAvailable = dispatcher != nil
	sm.mu.Unlock()

	if normalized {
		sm.requestScheduledSave()
		go sm.emitPoolHealth()
	}

	if dispatcher == nil {
		errors.LogWarning(context.Background(), "node pool: routing dispatcher not available, probing disabled")
	} else {
		probeCtx := context.Background()
		if sm.runtimeCtx != nil && core.FromContext(sm.runtimeCtx) != nil {
			probeCtx = core.ToBackgroundDetachedContext(sm.runtimeCtx)
		}

		sm.prober = NewNodeProber(probeCtx, dispatcher, cfg.ProbeURL, cfg.ProbeIntervalSec, sm.handleProbeResults)
		for _, tag := range tags {
			sm.prober.AddTag(tag)
		}
		sm.prober.Start()
	}

	sm.bgWG.Add(1)
	go func() {
		defer sm.bgWG.Done()
		sm.autoRefreshLoop()
	}()

	sm.bgWG.Add(1)
	go func() {
		defer sm.bgWG.Done()
		sm.reregisterOutbounds()
	}()
	sm.emitPoolHealth()
	return nil
}

// Stop shuts down the manager.
func (sm *SubscriptionManager) Stop() {
	select {
	case <-sm.stopCh:
	default:
		close(sm.stopCh)
	}
	sm.mu.Lock()
	sm.started = false
	sm.dispatcherAvailable = false
	sm.mu.Unlock()
	if sm.prober != nil {
		sm.prober.Stop()
	}
	sm.bgWG.Wait()
}

func (sm *SubscriptionManager) SetPoolHealthCallback(callback func(PoolHealthSummary)) {
	sm.mu.Lock()
	sm.onPoolHealthChange = callback
	sm.mu.Unlock()
	sm.emitPoolHealth()
}

// ListSubscriptions returns all subscriptions.
func (sm *SubscriptionManager) ListSubscriptions() []SubscriptionRecord {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make([]SubscriptionRecord, len(sm.state.Subscriptions))
	copy(result, sm.state.Subscriptions)
	for i := range result {
		result[i] = copySubscriptionRecordForAPI(result[i])
		count := 0
		for _, n := range sm.state.Nodes {
			if n.SubscriptionID == result[i].ID && isCountedSubscriptionNode(n) {
				count++
			}
		}
		result[i].NodeCount = count
	}
	return result
}

// AddSubscription adds a new subscription and immediately fetches it.
func (sm *SubscriptionManager) AddSubscription(input SubscriptionInput) (*SubscriptionRecord, error) {
	rec, err := buildSubscriptionRecord(input)
	if err != nil {
		return nil, err
	}

	sm.mu.Lock()
	if sm.subscriptionSourceExistsLocked(rec, "") {
		sm.mu.Unlock()
		return nil, fmt.Errorf("subscription already exists")
	}
	sm.state.Subscriptions = append(sm.state.Subscriptions, rec)
	sm.writeStateLocked()
	sm.mu.Unlock()

	if err := sm.RefreshSubscription(rec.ID); err != nil {
		sm.mu.Lock()
		sm.removeSubscriptionLocked(rec.ID)
		sm.writeStateLocked()
		sm.mu.Unlock()
		return nil, fmt.Errorf("failed to fetch subscription: %w", err)
	}

	sm.mu.RLock()
	defer sm.mu.RUnlock()
	for _, s := range sm.state.Subscriptions {
		if s.ID == rec.ID {
			copyRec := copySubscriptionRecordForAPI(s)
			return &copyRec, nil
		}
	}
	return nil, fmt.Errorf("subscription disappeared after creation")
}

func (sm *SubscriptionManager) UpdateSubscription(id string, input SubscriptionUpdateInput) (*SubscriptionRecord, error) {
	sm.refreshMu.Lock()
	defer sm.refreshMu.Unlock()

	sm.mu.Lock()
	idx := sm.findSubscriptionIndexLocked(id)
	if idx < 0 {
		sm.mu.Unlock()
		return nil, fmt.Errorf("subscription not found")
	}

	current := sm.state.Subscriptions[idx]
	current.SourceType = normalizeSubscriptionSourceType(current.SourceType)
	updated := current

	if input.SourceType != nil {
		requestedType := normalizeSubscriptionSourceType(*input.SourceType)
		if requestedType != current.SourceType {
			sm.mu.Unlock()
			return nil, fmt.Errorf("subscription source type cannot be changed")
		}
	}
	if input.Remark != nil {
		updated.Remark = strings.TrimSpace(*input.Remark)
	}

	requiresRefresh := false
	switch current.SourceType {
	case SubscriptionSourceURL:
		if input.URL != nil {
			updatedURL := strings.TrimSpace(*input.URL)
			if updatedURL == "" {
				sm.mu.Unlock()
				return nil, fmt.Errorf("subscription URL is required")
			}
			if updated.URL != updatedURL {
				updated.URL = updatedURL
				requiresRefresh = true
			}
		}
		if input.AutoRefresh != nil {
			updated.AutoRefresh = *input.AutoRefresh
		}
		if input.RefreshInterval != nil {
			updated.RefreshInterval = *input.RefreshInterval
		}
	case SubscriptionSourceManual, SubscriptionSourceFile:
		if input.URL != nil {
			sm.mu.Unlock()
			return nil, fmt.Errorf("only URL subscriptions support URL updates")
		}
		if input.AutoRefresh != nil || input.RefreshInterval != nil {
			sm.mu.Unlock()
			return nil, fmt.Errorf("only URL subscriptions support refresh policy updates")
		}
	default:
		sm.mu.Unlock()
		return nil, fmt.Errorf("unsupported subscription source type: %s", current.SourceType)
	}

	normalizeSubscriptionRecord(&updated)
	if sm.subscriptionSourceExistsLocked(updated, id) {
		sm.mu.Unlock()
		return nil, fmt.Errorf("subscription already exists")
	}

	sm.state.Subscriptions[idx] = updated
	sm.writeStateLocked()
	sm.mu.Unlock()

	if requiresRefresh {
		if err := sm.refreshSubscriptionLocked(id); err != nil {
			sm.mu.Lock()
			if rollbackIdx := sm.findSubscriptionIndexLocked(id); rollbackIdx >= 0 {
				sm.state.Subscriptions[rollbackIdx] = current
				sm.writeStateLocked()
			}
			sm.mu.Unlock()
			return nil, fmt.Errorf("failed to refresh updated subscription: %w", err)
		}
	}

	sm.mu.RLock()
	defer sm.mu.RUnlock()
	idx = sm.findSubscriptionIndexLocked(id)
	if idx < 0 {
		return nil, fmt.Errorf("subscription disappeared after update")
	}
	copyRec := copySubscriptionRecordForAPI(sm.state.Subscriptions[idx])
	return &copyRec, nil
}

// DeleteSubscription removes a subscription and marks its nodes removed.
func (sm *SubscriptionManager) DeleteSubscription(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	found := false
	for _, s := range sm.state.Subscriptions {
		if s.ID == id {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("subscription not found")
	}

	for i := range sm.state.Nodes {
		if sm.state.Nodes[i].SubscriptionID != id {
			continue
		}
		if sm.state.Nodes[i].Status == NodeStatusRemoved {
			continue
		}
		if err := sm.applyTransitionLocked(i, NodeStatusRemoved, TransitionReasonSubscriptionDeleted, EventActorSystem, "subscription deleted"); err != nil {
			return err
		}
	}
	sm.removeSubscriptionLocked(id)
	sm.writeStateLocked()
	go sm.emitPoolHealth()
	return nil
}

// RefreshSubscription triggers a refresh of a specific subscription.
func (sm *SubscriptionManager) RefreshSubscription(id string) error {
	sm.refreshMu.Lock()
	defer sm.refreshMu.Unlock()
	return sm.refreshSubscriptionLocked(id)
}

// ListNodes returns nodes filtered by status (empty string = all).
func (sm *SubscriptionManager) ListNodes(status string) []NodeRecord {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if status == legacyDemotedStatus {
		status = string(NodeStatusQuarantine)
	}

	result := make([]NodeRecord, 0, len(sm.state.Nodes))
	for _, n := range sm.state.Nodes {
		if status == "" || string(n.Status) == status {
			result = append(result, n)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		return nodeTime(result[i]).After(nodeTime(result[j]))
	})
	return result
}

func (sm *SubscriptionManager) ListNodesByStatuses(statuses ...NodeStatus) []NodeRecord {
	if len(statuses) == 0 {
		return sm.ListNodes("")
	}
	allowed := make(map[NodeStatus]struct{}, len(statuses))
	for _, status := range statuses {
		allowed[status] = struct{}{}
	}

	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make([]NodeRecord, 0, len(sm.state.Nodes))
	for _, n := range sm.state.Nodes {
		if _, ok := allowed[n.Status]; ok {
			result = append(result, n)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		return nodeTime(result[i]).After(nodeTime(result[j]))
	})
	return result
}

func (sm *SubscriptionManager) ListRecentNodeEvents(limit int) []NodeEvent {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return copyRecentNodeEvents(sm.state.RecentNodeEvents, limit)
}

func (sm *SubscriptionManager) GetPoolSummary() NodePoolSummary {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.poolSummaryLocked()
}

func (sm *SubscriptionManager) GetValidationConfig() ValidationConfig {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state.ValidationConfig
}

type SubscriptionManagerReadinessSnapshot struct {
	Started             bool               `json:"started"`
	DispatcherAvailable bool               `json:"dispatcherAvailable"`
	SubscriptionCount   int                `json:"subscriptionCount"`
	NodeCount           int                `json:"nodeCount"`
	ProbeURL            string             `json:"probeUrl"`
	ProbeIntervalSec    int                `json:"probeIntervalSec"`
	PoolSummary         NodePoolSummary    `json:"poolSummary"`
	Prober              NodeProberSnapshot `json:"prober"`
}

func (sm *SubscriptionManager) ReadinessSnapshot() SubscriptionManagerReadinessSnapshot {
	sm.mu.RLock()
	cfg := sm.state.ValidationConfig
	prober := sm.prober
	snapshot := SubscriptionManagerReadinessSnapshot{
		Started:             sm.started,
		DispatcherAvailable: sm.dispatcherAvailable,
		SubscriptionCount:   len(sm.state.Subscriptions),
		NodeCount:           len(sm.state.Nodes),
		ProbeURL:            cfg.ProbeURL,
		ProbeIntervalSec:    cfg.ProbeIntervalSec,
		PoolSummary:         sm.poolSummaryLocked(),
	}
	sm.mu.RUnlock()

	if prober != nil {
		snapshot.Prober = prober.Snapshot()
	}

	return snapshot
}

func (sm *SubscriptionManager) UpdateValidationConfig(cfg ValidationConfig) {
	applyValidationDefaults(&cfg)

	sm.mu.Lock()
	sm.state.ValidationConfig = cfg
	sm.writeStateLocked()
	sm.mu.Unlock()

	if sm.prober != nil {
		sm.prober.UpdateConfig(cfg.ProbeURL, cfg.ProbeIntervalSec)
	}
	sm.emitPoolHealth()
}

// PromoteNode manually promotes a staging/quarantine node into active.
func (sm *SubscriptionManager) PromoteNode(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i, n := range sm.state.Nodes {
		if n.ID != id {
			continue
		}
		if n.Status != NodeStatusStaging && n.Status != NodeStatusQuarantine {
			return fmt.Errorf("node is not in a promotable status")
		}
		if err := sm.applyTransitionLocked(i, NodeStatusActive, TransitionReasonManualPromote, EventActorOperator, "manual promote"); err != nil {
			return err
		}
		sm.writeStateLocked()
		go sm.emitPoolHealth()
		return nil
	}
	return fmt.Errorf("node not found")
}

func (sm *SubscriptionManager) ValidateNode(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i, n := range sm.state.Nodes {
		if n.ID != id {
			continue
		}
		if n.Status != NodeStatusCandidate {
			return fmt.Errorf("node is not in candidate status")
		}
		if err := sm.applyTransitionLocked(i, NodeStatusStaging, TransitionReasonManualValidate, EventActorOperator, "manual move to validation"); err != nil {
			return err
		}
		sm.writeStateLocked()
		go sm.emitPoolHealth()
		return nil
	}
	return fmt.Errorf("node not found")
}

func (sm *SubscriptionManager) BulkPromote(ids []string) (int, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		idSet[id] = struct{}{}
	}

	promoted := 0
	for i := range sm.state.Nodes {
		node := sm.state.Nodes[i]
		if _, ok := idSet[node.ID]; !ok {
			continue
		}
		if node.Status != NodeStatusStaging && node.Status != NodeStatusQuarantine {
			continue
		}
		if err := sm.applyTransitionLocked(i, NodeStatusActive, TransitionReasonManualPromote, EventActorOperator, "bulk promote"); err != nil {
			return promoted, err
		}
		promoted++
	}

	if promoted == 0 {
		return 0, nil
	}

	sm.writeStateLocked()
	go sm.emitPoolHealth()
	return promoted, nil
}

func (sm *SubscriptionManager) BulkValidate(ids []string) (int, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		idSet[id] = struct{}{}
	}

	validated := 0
	for i := range sm.state.Nodes {
		node := sm.state.Nodes[i]
		if _, ok := idSet[node.ID]; !ok {
			continue
		}
		if node.Status != NodeStatusCandidate {
			continue
		}
		if err := sm.applyTransitionLocked(i, NodeStatusStaging, TransitionReasonManualValidate, EventActorOperator, "bulk move to validation"); err != nil {
			return validated, err
		}
		validated++
	}

	if validated == 0 {
		return 0, nil
	}

	sm.writeStateLocked()
	go sm.emitPoolHealth()
	return validated, nil
}

// DemoteNode keeps backward compatibility with the old API and now quarantines the node.
func (sm *SubscriptionManager) DemoteNode(id string) error {
	return sm.QuarantineNode(id)
}

// QuarantineNode manually moves a node out of the active pool without deleting it.
func (sm *SubscriptionManager) QuarantineNode(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i, n := range sm.state.Nodes {
		if n.ID != id {
			continue
		}
		if n.Status != NodeStatusActive && n.Status != NodeStatusStaging {
			return fmt.Errorf("node is not in an active or staging status")
		}
		if err := sm.applyTransitionLocked(i, NodeStatusQuarantine, TransitionReasonManualQuarantine, EventActorOperator, "manual quarantine"); err != nil {
			return err
		}
		sm.writeStateLocked()
		go sm.emitPoolHealth()
		return nil
	}
	return fmt.Errorf("node not found")
}

// DeleteNode removes a single node from the working system while preserving it in removed state.
func (sm *SubscriptionManager) DeleteNode(id string) error {
	return sm.RemoveNode(id)
}

func (sm *SubscriptionManager) RemoveNode(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i, n := range sm.state.Nodes {
		if n.ID != id {
			continue
		}
		if n.Status == NodeStatusRemoved {
			return nil
		}
		if err := sm.applyTransitionLocked(i, NodeStatusRemoved, TransitionReasonManualRemove, EventActorOperator, "manual remove"); err != nil {
			return err
		}
		sm.writeStateLocked()
		go sm.emitPoolHealth()
		return nil
	}
	return fmt.Errorf("node not found")
}

func (sm *SubscriptionManager) RestoreNode(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i, n := range sm.state.Nodes {
		if n.ID != id {
			continue
		}
		if n.Status != NodeStatusRemoved {
			return fmt.Errorf("node is not in removed status")
		}
		if err := sm.applyTransitionLocked(i, NodeStatusCandidate, TransitionReasonManualRestore, EventActorOperator, "manual restore to candidate"); err != nil {
			return err
		}
		sm.writeStateLocked()
		go sm.emitPoolHealth()
		return nil
	}
	return fmt.Errorf("node not found")
}

func (sm *SubscriptionManager) BulkRestore(ids []string) (int, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		idSet[id] = struct{}{}
	}

	restored := 0
	for i := range sm.state.Nodes {
		node := sm.state.Nodes[i]
		if _, ok := idSet[node.ID]; !ok {
			continue
		}
		if node.Status != NodeStatusRemoved {
			continue
		}
		if err := sm.applyTransitionLocked(i, NodeStatusCandidate, TransitionReasonManualRestore, EventActorOperator, "bulk restore to candidate"); err != nil {
			return restored, err
		}
		restored++
	}

	if restored == 0 {
		return 0, nil
	}

	sm.writeStateLocked()
	go sm.emitPoolHealth()
	return restored, nil
}

func (sm *SubscriptionManager) BulkPurgeRemoved(ids []string) (int, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		idSet[id] = struct{}{}
	}

	if len(idSet) == 0 {
		return 0, nil
	}

	kept := sm.state.Nodes[:0]
	purged := 0
	for _, node := range sm.state.Nodes {
		if _, ok := idSet[node.ID]; !ok || node.Status != NodeStatusRemoved {
			kept = append(kept, node)
			continue
		}
		purged++
		errors.LogInfo(context.Background(), "node pool: purged removed node record ", node.Remark, " (", node.ID, ")")
	}
	sm.state.Nodes = kept

	if purged == 0 {
		return 0, nil
	}

	sm.writeStateLocked()
	go sm.emitPoolHealth()
	return purged, nil
}

func (sm *SubscriptionManager) BulkRemove(filter BulkRemoveFilter) (int, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	idSet := make(map[string]struct{}, len(filter.IDs))
	for _, id := range filter.IDs {
		idSet[id] = struct{}{}
	}
	statusSet := make(map[NodeStatus]struct{}, len(filter.Statuses))
	for _, status := range filter.Statuses {
		statusSet[status] = struct{}{}
	}
	cleanSet := make(map[CleanlinessStatus]struct{}, len(filter.Cleanliness))
	for _, value := range filter.Cleanliness {
		cleanSet[value] = struct{}{}
	}

	removed := 0
	for i := range sm.state.Nodes {
		node := sm.state.Nodes[i]
		if node.Status == NodeStatusRemoved {
			continue
		}
		if len(idSet) > 0 {
			if _, ok := idSet[node.ID]; !ok {
				continue
			}
		}
		if len(statusSet) > 0 {
			if _, ok := statusSet[node.Status]; !ok {
				continue
			}
		}
		if len(cleanSet) > 0 {
			if _, ok := cleanSet[node.Cleanliness]; !ok {
				continue
			}
		}
		if filter.OnlyUnstable && node.Status != NodeStatusQuarantine {
			continue
		}
		if err := sm.applyTransitionLocked(i, NodeStatusRemoved, TransitionReasonManualRemove, EventActorOperator, "bulk remove"); err != nil {
			return removed, err
		}
		removed++
	}

	if removed == 0 {
		return 0, nil
	}

	sm.writeStateLocked()
	go sm.emitPoolHealth()
	return removed, nil
}

func (sm *SubscriptionManager) refreshSubscriptionLocked(id string) error {
	sm.mu.RLock()
	var sub *SubscriptionRecord
	for i := range sm.state.Subscriptions {
		if sm.state.Subscriptions[i].ID == id {
			copySub := sm.state.Subscriptions[i]
			sub = &copySub
			break
		}
	}
	sm.mu.RUnlock()

	if sub == nil {
		return fmt.Errorf("subscription not found")
	}

	content, err := sm.loadSubscriptionContent(sub)
	if err != nil {
		return err
	}

	links, err := ParseSubscriptionContent(content)
	if err != nil {
		return fmt.Errorf("failed to parse subscription content: %w", err)
	}

	now := time.Now()

	sm.mu.Lock()
	defer sm.mu.Unlock()

	existingByID := make(map[string]int)
	for i, n := range sm.state.Nodes {
		if n.SubscriptionID == id {
			existingByID[n.ID] = i
		}
	}

	newNodeIDs := make(map[string]bool, len(links))
	for _, link := range links {
		uri, _ := GenerateShareLink(*link)
		if uri == "" {
			continue
		}
		nodeID := hashID(uri)
		newNodeIDs[nodeID] = true

		if idx, exists := existingByID[nodeID]; exists {
			node := &sm.state.Nodes[idx]
			node.URI = uri
			node.Remark = link.Remark
			node.Protocol = link.Protocol
			node.Address = link.Address
			node.Port = link.Port

			if node.Status == NodeStatusCandidate && node.StatusReason == TransitionReasonOutboundRegistrationFailed {
				if err := sm.applyTransitionLocked(idx, NodeStatusStaging, TransitionReasonSubscriptionReintroduced, EventActorSystem, "retry outbound registration"); err != nil {
					sm.applyTransitionLocked(idx, NodeStatusCandidate, TransitionReasonOutboundRegistrationFailed, EventActorSystem, err.Error())
				}
			}
			if node.Status == NodeStatusCandidate && node.StatusReason == TransitionReasonSubscriptionMissing {
				if err := sm.applyTransitionLocked(idx, NodeStatusStaging, TransitionReasonSubscriptionReintroduced, EventActorSystem, "subscription reintroduced"); err != nil {
					sm.applyTransitionLocked(idx, NodeStatusCandidate, TransitionReasonSubscriptionMissing, EventActorSystem, err.Error())
				}
			}
			if node.Status == NodeStatusRemoved && node.StatusReason == TransitionReasonSubscriptionMissing {
				if err := sm.applyTransitionLocked(idx, NodeStatusStaging, TransitionReasonSubscriptionReintroduced, EventActorSystem, "subscription reintroduced"); err != nil {
					sm.applyTransitionLocked(idx, NodeStatusRemoved, TransitionReasonSubscriptionMissing, EventActorSystem, err.Error())
				}
			}
			continue
		}

		node := NodeRecord{
			ID:             nodeID,
			URI:            uri,
			Remark:         link.Remark,
			Protocol:       link.Protocol,
			Address:        link.Address,
			Port:           link.Port,
			Status:         NodeStatusCandidate,
			StatusReason:   TransitionReasonSubscriptionNodeDiscovered,
			SubscriptionID: id,
			AddedAt:        now,
			Cleanliness:    CleanlinessUnknown,
			BandwidthTier:  BandwidthTierUnknown,
		}
		sm.state.Nodes = append(sm.state.Nodes, node)
		idx := len(sm.state.Nodes) - 1
		if err := sm.applyTransitionLocked(idx, NodeStatusStaging, TransitionReasonSubscriptionNodeDiscovered, EventActorSystem, "subscription refresh discovered a new node"); err != nil {
			sm.applyTransitionLocked(idx, NodeStatusCandidate, TransitionReasonOutboundRegistrationFailed, EventActorSystem, err.Error())
		}
	}

	for i := range sm.state.Nodes {
		node := sm.state.Nodes[i]
		if node.SubscriptionID != id {
			continue
		}
		if newNodeIDs[node.ID] {
			continue
		}
		if node.Status == NodeStatusRemoved && node.StatusReason != TransitionReasonSubscriptionMissing {
			continue
		}
		if node.Status == NodeStatusCandidate && node.StatusReason == TransitionReasonSubscriptionMissing {
			continue
		}
		if err := sm.applyTransitionLocked(i, NodeStatusCandidate, TransitionReasonSubscriptionMissing, EventActorSystem, "node disappeared from upstream subscription; parked in candidate"); err != nil {
			return err
		}
	}

	for i := range sm.state.Subscriptions {
		if sm.state.Subscriptions[i].ID != id {
			continue
		}
		sm.state.Subscriptions[i].LastRefresh = &now
		count := 0
		for _, n := range sm.state.Nodes {
			if n.SubscriptionID == id && isCountedSubscriptionNode(n) {
				count++
			}
		}
		sm.state.Subscriptions[i].NodeCount = count
		break
	}

	sm.writeStateLocked()
	go sm.emitPoolHealth()
	return nil
}

func (sm *SubscriptionManager) loadSubscriptionContent(sub *SubscriptionRecord) (string, error) {
	sourceType := normalizeSubscriptionSourceType(sub.SourceType)
	if sourceType != SubscriptionSourceURL {
		content := strings.TrimSpace(trimUTF8BOM(sub.Content))
		if content == "" {
			return "", fmt.Errorf("subscription content is empty")
		}
		return content, nil
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(sub.URL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch subscription: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return "", fmt.Errorf("failed to read subscription body: %w", err)
	}

	return string(body), nil
}

func (sm *SubscriptionManager) handleProbeResults(results []ProbeResult) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	cfg := sm.state.ValidationConfig
	changed := false

	for _, result := range results {
		idx := sm.findNodeByTag(result.Tag)
		if idx < 0 {
			continue
		}
		node := &sm.state.Nodes[idx]
		if !isProbeableStatus(node.Status) {
			continue
		}

		node.TotalPings++
		node.LastCheckedAt = &now
		node.LastEventAt = &now
		changed = true

		if result.Success {
			node.ConsecutiveFails = 0
			successCount := node.TotalPings - node.FailedPings
			if successCount <= 1 || node.AvgDelayMs == 0 {
				node.AvgDelayMs = result.DelayMs
			} else {
				node.AvgDelayMs = (node.AvgDelayMs*int64(successCount-1) + result.DelayMs) / int64(successCount)
			}
		} else {
			node.FailedPings++
			node.ConsecutiveFails++
		}

		failRate := float64(0)
		if node.TotalPings > 0 {
			failRate = float64(node.FailedPings) / float64(node.TotalPings)
		}

		switch node.Status {
		case NodeStatusStaging:
			if result.Success && node.TotalPings >= cfg.MinSamples && failRate <= cfg.MaxFailRate && node.AvgDelayMs > 0 && node.AvgDelayMs <= cfg.MaxAvgDelayMs {
				sm.applyTransitionLocked(idx, NodeStatusActive, TransitionReasonProbeQualified, EventActorSystem, "validation thresholds satisfied")
				continue
			}
			if node.TotalPings >= cfg.MinSamples && node.ConsecutiveFails >= cfg.DemoteAfterFails {
				sm.applyTransitionLocked(idx, NodeStatusQuarantine, TransitionReasonProbeFailuresExceeded, EventActorSystem, "validation probe failures exceeded threshold")
			}
		case NodeStatusActive:
			if node.ConsecutiveFails >= cfg.DemoteAfterFails {
				target := NodeStatusQuarantine
				details := "consecutive probe failures exceeded threshold"
				if cfg.AutoRemoveDemoted {
					target = NodeStatusRemoved
					details = "auto-remove enabled after repeated probe failures"
				}
				sm.applyTransitionLocked(idx, target, TransitionReasonProbeFailuresExceeded, EventActorSystem, details)
			}
		case NodeStatusQuarantine:
			if result.Success && node.TotalPings >= cfg.MinSamples && failRate <= cfg.MaxFailRate && node.AvgDelayMs > 0 && node.AvgDelayMs <= cfg.MaxAvgDelayMs {
				sm.applyTransitionLocked(idx, NodeStatusActive, TransitionReasonProbeRequalified, EventActorSystem, "quarantined node re-qualified")
			}
		}
	}

	if changed {
		sm.requestScheduledSave()
		go sm.emitPoolHealth()
	}
}

func (sm *SubscriptionManager) applyTransitionLocked(idx int, nextStatus NodeStatus, reason TransitionReason, actor EventActor, details string) error {
	if idx < 0 || idx >= len(sm.state.Nodes) {
		return fmt.Errorf("invalid node index")
	}

	node := &sm.state.Nodes[idx]
	currentStatus := node.Status
	if !isAllowedNodeTransition(currentStatus, nextStatus) {
		return fmt.Errorf("illegal node transition: %s -> %s", currentStatus, nextStatus)
	}

	switch {
	case !isProbeableStatus(currentStatus) && isProbeableStatus(nextStatus):
		if err := sm.ensureProbeableLocked(node); err != nil {
			return err
		}
	case isProbeableStatus(currentStatus) && !isProbeableStatus(nextStatus):
		sm.removeProbeableLocked(node)
	case isProbeableStatus(currentStatus) && isProbeableStatus(nextStatus):
		if node.OutboundTag == "" {
			if err := sm.ensureProbeableLocked(node); err != nil {
				return err
			}
		}
	}

	now := time.Now()
	node.Status = nextStatus
	node.StatusReason = reason
	node.StatusUpdatedAt = &now
	node.LastEventAt = &now
	if nextStatus == NodeStatusActive {
		node.PromotedAt = &now
		node.ConsecutiveFails = 0
	}
	if nextStatus == NodeStatusCandidate || nextStatus == NodeStatusRemoved {
		node.OutboundTag = ""
	}

	sm.appendNodeEventLocked(NodeEvent{
		NodeID:   node.ID,
		Remark:   node.Remark,
		Status:   node.Status,
		Reason:   reason,
		Actor:    actor,
		At:       now,
		Details:  details,
		NodeAddr: fmt.Sprintf("%s:%d", node.Address, node.Port),
	})

	switch reason {
	case TransitionReasonProbeFailuresExceeded:
		errors.LogWarning(context.Background(), "node pool: quarantined node ", node.Remark, " (", node.ID, ")")
	case TransitionReasonProbeQualified, TransitionReasonProbeRequalified, TransitionReasonManualPromote:
		errors.LogInfo(context.Background(), "node pool: activated node ", node.Remark, " (", node.ID, ")")
	case TransitionReasonSubscriptionMissing:
		errors.LogInfo(context.Background(), "node pool: parked missing subscription node in candidate ", node.Remark, " (", node.ID, ")")
	case TransitionReasonManualRestore:
		errors.LogInfo(context.Background(), "node pool: moved removed node back to candidate ", node.Remark, " (", node.ID, ")")
	case TransitionReasonManualRemove, TransitionReasonSubscriptionDeleted:
		errors.LogInfo(context.Background(), "node pool: removed node ", node.Remark, " (", node.ID, ")")
	}

	return nil
}

func (sm *SubscriptionManager) normalizeProbeStateLocked() bool {
	cfg := sm.state.ValidationConfig
	if cfg.DemoteAfterFails <= 0 {
		return false
	}

	changed := false
	for idx := range sm.state.Nodes {
		node := &sm.state.Nodes[idx]
		if node.Status != NodeStatusActive {
			continue
		}
		if node.ConsecutiveFails < cfg.DemoteAfterFails {
			continue
		}
		if err := sm.applyTransitionLocked(idx, NodeStatusQuarantine, TransitionReasonProbeFailuresExceeded, EventActorSystem, "startup normalization: active node exceeded consecutive probe failure threshold"); err == nil {
			changed = true
		}
	}

	return changed
}

func (sm *SubscriptionManager) ensureProbeableLocked(node *NodeRecord) error {
	if node.OutboundTag == "" {
		node.OutboundTag = probeOutboundTag(node.ID)
	}

	link, err := ParseShareLinkURI(node.URI)
	if err != nil {
		return fmt.Errorf("failed to parse node URI: %w", err)
	}

	if err := sm.addOutboundFromLink(link, node.OutboundTag); err != nil {
		return fmt.Errorf("failed to add outbound %q: %w", node.OutboundTag, err)
	}
	if sm.prober != nil {
		sm.prober.AddTag(node.OutboundTag)
	}
	return nil
}

func (sm *SubscriptionManager) removeProbeableLocked(node *NodeRecord) {
	if node.OutboundTag == "" {
		return
	}
	sm.removeOutboundAsync(node.OutboundTag)
	if sm.prober != nil {
		sm.prober.RemoveTag(node.OutboundTag)
	}
}

func (sm *SubscriptionManager) addOutboundFromLink(link *ShareLinkRequest, tag string) error {
	if sm.grpcClient == nil {
		return fmt.Errorf("grpc client is not initialized")
	}

	outboundConfig, err := buildOutboundHandlerConfigFromLink(link, tag)
	if err != nil {
		return err
	}

	_, err = sm.grpcClient.Handler().AddOutbound(sm.grpcClient.Context(), &handlerservice.AddOutboundRequest{
		Outbound: outboundConfig,
	})
	return err
}

func (sm *SubscriptionManager) removeOutboundAsync(tag string) {
	if tag == "" || sm.grpcClient == nil {
		return
	}
	go func() {
		_, err := sm.grpcClient.Handler().RemoveOutbound(sm.grpcClient.Context(), &handlerservice.RemoveOutboundRequest{
			Tag: tag,
		})
		if err != nil {
			errors.LogDebug(context.Background(), "node pool: failed to remove outbound ", tag, ": ", err.Error())
		}
	}()
}

func (sm *SubscriptionManager) removeSubscriptionLocked(id string) {
	remaining := sm.state.Subscriptions[:0]
	for _, s := range sm.state.Subscriptions {
		if s.ID != id {
			remaining = append(remaining, s)
		}
	}
	sm.state.Subscriptions = remaining
}

func (sm *SubscriptionManager) findSubscriptionIndexLocked(id string) int {
	for i := range sm.state.Subscriptions {
		if sm.state.Subscriptions[i].ID == id {
			return i
		}
	}
	return -1
}

func (sm *SubscriptionManager) subscriptionSourceExistsLocked(candidate SubscriptionRecord, excludeID string) bool {
	for _, existing := range sm.state.Subscriptions {
		if existing.ID == excludeID {
			continue
		}
		if sameSubscriptionSource(existing, candidate) {
			return true
		}
	}
	return false
}

func (sm *SubscriptionManager) findNodeByTag(tag string) int {
	for i, n := range sm.state.Nodes {
		if n.OutboundTag == tag {
			return i
		}
	}
	return -1
}

func (sm *SubscriptionManager) autoRefreshLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-sm.stopCh:
			return
		case <-ticker.C:
			sm.mu.RLock()
			subs := make([]SubscriptionRecord, len(sm.state.Subscriptions))
			copy(subs, sm.state.Subscriptions)
			sm.mu.RUnlock()

			for _, sub := range subs {
				if !sub.AutoRefresh {
					continue
				}
				if sub.LastRefresh != nil && time.Since(*sub.LastRefresh) < time.Duration(sub.RefreshInterval)*time.Minute {
					continue
				}
				if err := sm.RefreshSubscription(sub.ID); err != nil {
					errors.LogWarning(context.Background(), "node pool: auto-refresh failed for ", sub.Remark, ": ", err.Error())
				}
			}
		}
	}
}

func (sm *SubscriptionManager) reregisterOutbounds() {
	select {
	case <-sm.stopCh:
		return
	case <-time.After(2 * time.Second):
	}

	sm.mu.RLock()
	nodes := make([]NodeRecord, len(sm.state.Nodes))
	copy(nodes, sm.state.Nodes)
	sm.mu.RUnlock()

	for _, node := range nodes {
		if !isProbeableStatus(node.Status) || node.OutboundTag == "" {
			continue
		}
		link, err := ParseShareLinkURI(node.URI)
		if err != nil {
			continue
		}
		if err := sm.addOutboundFromLink(link, node.OutboundTag); err != nil {
			errors.LogDebug(context.Background(), "node pool: failed to re-register outbound ", node.OutboundTag, ": ", err.Error())
		}
	}
}

func (sm *SubscriptionManager) persistLoop() {
	var timer *time.Timer
	var timerCh <-chan time.Time

	for {
		select {
		case <-sm.stopCh:
			if timer != nil {
				stopTimer(timer)
			}
			sm.mu.Lock()
			sm.writeStateLocked()
			sm.mu.Unlock()
			return
		case <-sm.saveCh:
			if timer == nil {
				timer = time.NewTimer(sm.saveDelay)
			} else {
				stopTimer(timer)
				timer.Reset(sm.saveDelay)
			}
			timerCh = timer.C
		case <-timerCh:
			sm.mu.Lock()
			sm.writeStateLocked()
			sm.mu.Unlock()
			timerCh = nil
		}
	}
}

func (sm *SubscriptionManager) requestScheduledSave() {
	select {
	case sm.saveCh <- struct{}{}:
	default:
	}
}

func (sm *SubscriptionManager) emitPoolHealth() {
	sm.mu.RLock()
	callback := sm.onPoolHealthChange
	summary := sm.poolHealthLocked()
	sm.mu.RUnlock()
	if callback != nil {
		go callback(summary)
	}
}

func (sm *SubscriptionManager) poolSummaryLocked() NodePoolSummary {
	summary := NodePoolSummary{
		MinActiveNodes:  sm.state.ValidationConfig.MinActiveNodes,
		LastEvaluatedAt: time.Now(),
	}

	for _, node := range sm.state.Nodes {
		switch node.Status {
		case NodeStatusCandidate:
			summary.CandidateCount++
		case NodeStatusStaging:
			summary.StagingCount++
		case NodeStatusActive:
			summary.ActiveCount++
		case NodeStatusQuarantine:
			summary.QuarantineCount++
		case NodeStatusRemoved:
			summary.RemovedCount++
		}

		switch node.Cleanliness {
		case CleanlinessTrusted:
			summary.TrustedCount++
		case CleanlinessSuspicious:
			summary.SuspiciousCount++
		default:
			summary.UnknownCleanCount++
		}
	}

	summary.ActiveNodes = summary.ActiveCount
	summary.Healthy = summary.ActiveNodes >= summary.MinActiveNodes

	if events := sm.state.RecentNodeEvents; len(events) > 0 {
		latest := events[len(events)-1]
		summary.LatestEventAt = &latest.At
		summary.LatestEventReason = latest.Reason
		summary.LatestEventStatus = latest.Status
		summary.LatestEventActor = latest.Actor
		summary.LatestEventNodeID = latest.NodeID
		summary.LatestEventNodeAddr = latest.NodeAddr
	}

	return summary
}

func (sm *SubscriptionManager) poolHealthLocked() PoolHealthSummary {
	summary := sm.poolSummaryLocked()
	return PoolHealthSummary{
		ActiveNodes:     summary.ActiveNodes,
		MinActiveNodes:  summary.MinActiveNodes,
		Healthy:         summary.Healthy,
		LastEvaluatedAt: summary.LastEvaluatedAt,
	}
}

func (sm *SubscriptionManager) appendNodeEventLocked(event NodeEvent) {
	sm.state.RecentNodeEvents = append(sm.state.RecentNodeEvents, event)
	if len(sm.state.RecentNodeEvents) > nodeEventLimit {
		sm.state.RecentNodeEvents = append([]NodeEvent(nil), sm.state.RecentNodeEvents[len(sm.state.RecentNodeEvents)-nodeEventLimit:]...)
	}
}

// --- Persistence ---

func (sm *SubscriptionManager) loadState() (*NodePoolState, bool) {
	state := &NodePoolState{
		ValidationConfig: defaultValidationConfig(),
	}

	data, err := os.ReadFile(sm.statePath)
	if err != nil {
		return state, false
	}

	if err := json.Unmarshal(data, state); err != nil {
		errors.LogWarning(context.Background(), "node pool: failed to parse state file: ", err.Error())
		return &NodePoolState{ValidationConfig: defaultValidationConfig()}, false
	}

	applyValidationDefaults(&state.ValidationConfig)
	changed := false
	for i := range state.Subscriptions {
		if normalizeSubscriptionRecord(&state.Subscriptions[i]) {
			changed = true
		}
	}
	now := time.Now()
	for i := range state.Nodes {
		if normalizeNodeRecord(&state.Nodes[i], now) {
			changed = true
		}
	}
	if len(state.RecentNodeEvents) > nodeEventLimit {
		state.RecentNodeEvents = append([]NodeEvent(nil), state.RecentNodeEvents[len(state.RecentNodeEvents)-nodeEventLimit:]...)
		changed = true
	}

	return state, changed
}

func (sm *SubscriptionManager) writeStateLocked() {
	data, err := json.MarshalIndent(sm.state, "", "  ")
	if err != nil {
		errors.LogWarning(context.Background(), "node pool: failed to marshal state: ", err.Error())
		return
	}

	dir := filepath.Dir(sm.statePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		errors.LogWarning(context.Background(), "node pool: failed to create state dir: ", err.Error())
		return
	}

	if err := os.WriteFile(sm.statePath, data, 0o644); err != nil {
		errors.LogWarning(context.Background(), "node pool: failed to save state: ", err.Error())
	}
}

func defaultValidationConfig() ValidationConfig {
	return ValidationConfig{
		MinSamples:        10,
		MaxFailRate:       0.30,
		MaxAvgDelayMs:     1000,
		ProbeIntervalSec:  60,
		ProbeURL:          "https://www.gstatic.com/generate_204",
		DemoteAfterFails:  5,
		AutoRemoveDemoted: false,
		MinActiveNodes:    3,
		MinBandwidthKbps:  0,
	}
}

func applyValidationDefaults(cfg *ValidationConfig) {
	if cfg.MinSamples <= 0 {
		cfg.MinSamples = 10
	}
	if cfg.MaxFailRate <= 0 {
		cfg.MaxFailRate = 0.30
	}
	if cfg.MaxAvgDelayMs <= 0 {
		cfg.MaxAvgDelayMs = 1000
	}
	if cfg.ProbeIntervalSec <= 0 {
		cfg.ProbeIntervalSec = 60
	}
	if cfg.ProbeURL == "" {
		cfg.ProbeURL = "https://www.gstatic.com/generate_204"
	}
	if cfg.DemoteAfterFails <= 0 {
		cfg.DemoteAfterFails = 5
	}
	if cfg.MinActiveNodes <= 0 {
		cfg.MinActiveNodes = 3
	}
	if cfg.MinBandwidthKbps < 0 {
		cfg.MinBandwidthKbps = 0
	}
}

func normalizeSubscriptionSourceType(value SubscriptionSourceType) SubscriptionSourceType {
	switch value {
	case SubscriptionSourceManual, SubscriptionSourceFile:
		return value
	default:
		return SubscriptionSourceURL
	}
}

func buildSubscriptionRecord(input SubscriptionInput) (SubscriptionRecord, error) {
	sourceType := normalizeSubscriptionSourceType(input.SourceType)
	remark := strings.TrimSpace(input.Remark)
	sourceName := strings.TrimSpace(input.SourceName)

	rec := SubscriptionRecord{
		SourceType: sourceType,
		Remark:     remark,
	}

	switch sourceType {
	case SubscriptionSourceURL:
		urlStr := strings.TrimSpace(input.URL)
		if urlStr == "" {
			return SubscriptionRecord{}, fmt.Errorf("subscription URL is required")
		}
		rec.ID = hashID(urlStr)
		rec.URL = urlStr
		rec.AutoRefresh = input.AutoRefresh
		rec.RefreshInterval = input.RefreshInterval
		if rec.RefreshInterval <= 0 {
			rec.RefreshInterval = 60
		}
	case SubscriptionSourceManual, SubscriptionSourceFile:
		content := strings.TrimSpace(trimUTF8BOM(input.Content))
		if content == "" {
			return SubscriptionRecord{}, fmt.Errorf("subscription content is required")
		}
		rec.Content = content
		rec.SourceName = sourceName
		rec.ID = hashID(strings.Join([]string{string(sourceType), sourceName, content}, "\n"))
	default:
		return SubscriptionRecord{}, fmt.Errorf("unsupported subscription source type: %s", input.SourceType)
	}

	return rec, nil
}

func copySubscriptionRecordForAPI(sub SubscriptionRecord) SubscriptionRecord {
	sub.SourceType = normalizeSubscriptionSourceType(sub.SourceType)
	sub.Content = ""
	return sub
}

func sameSubscriptionSource(a, b SubscriptionRecord) bool {
	sourceType := normalizeSubscriptionSourceType(a.SourceType)
	if sourceType != normalizeSubscriptionSourceType(b.SourceType) {
		return false
	}

	switch sourceType {
	case SubscriptionSourceURL:
		return strings.TrimSpace(a.URL) == strings.TrimSpace(b.URL)
	case SubscriptionSourceManual, SubscriptionSourceFile:
		return strings.TrimSpace(a.SourceName) == strings.TrimSpace(b.SourceName) &&
			strings.TrimSpace(trimUTF8BOM(a.Content)) == strings.TrimSpace(trimUTF8BOM(b.Content))
	default:
		return false
	}
}

func normalizeSubscriptionRecord(sub *SubscriptionRecord) bool {
	changed := false

	normalizedType := normalizeSubscriptionSourceType(sub.SourceType)
	if sub.SourceType != normalizedType {
		sub.SourceType = normalizedType
		changed = true
	}

	trimmedURL := strings.TrimSpace(sub.URL)
	if sub.URL != trimmedURL {
		sub.URL = trimmedURL
		changed = true
	}
	trimmedContent := strings.TrimSpace(trimUTF8BOM(sub.Content))
	if sub.Content != trimmedContent {
		sub.Content = trimmedContent
		changed = true
	}
	trimmedRemark := strings.TrimSpace(sub.Remark)
	if sub.Remark != trimmedRemark {
		sub.Remark = trimmedRemark
		changed = true
	}
	trimmedSourceName := strings.TrimSpace(sub.SourceName)
	if sub.SourceName != trimmedSourceName {
		sub.SourceName = trimmedSourceName
		changed = true
	}

	switch sub.SourceType {
	case SubscriptionSourceURL:
		if sub.RefreshInterval <= 0 {
			sub.RefreshInterval = 60
			changed = true
		}
	case SubscriptionSourceManual, SubscriptionSourceFile:
		if sub.AutoRefresh {
			sub.AutoRefresh = false
			changed = true
		}
		if sub.RefreshInterval != 0 {
			sub.RefreshInterval = 0
			changed = true
		}
	}

	return changed
}

func normalizeNodeRecord(node *NodeRecord, now time.Time) bool {
	changed := false

	switch node.Status {
	case NodeStatus(""), NodeStatus(legacyDemotedStatus):
		if string(node.Status) == legacyDemotedStatus {
			node.Status = NodeStatusQuarantine
			node.StatusReason = TransitionReasonMigrationLegacyDemoted
		} else {
			node.Status = NodeStatusStaging
		}
		changed = true
	}

	if node.Cleanliness == "" {
		node.Cleanliness = CleanlinessUnknown
		changed = true
	}
	if node.BandwidthTier == "" {
		node.BandwidthTier = BandwidthTierUnknown
		changed = true
	}
	if isProbeableStatus(node.Status) {
		expectedTag := probeOutboundTag(node.ID)
		if node.OutboundTag == "" || hasLegacyStatusTag(node.OutboundTag) || node.OutboundTag != expectedTag {
			node.OutboundTag = expectedTag
			changed = true
		}
	} else if node.OutboundTag != "" {
		node.OutboundTag = ""
		changed = true
	}

	if node.StatusUpdatedAt == nil {
		ts := node.AddedAt
		if ts.IsZero() {
			ts = now
		}
		node.StatusUpdatedAt = &ts
		changed = true
	}
	if node.LastEventAt == nil && node.StatusUpdatedAt != nil {
		ts := *node.StatusUpdatedAt
		node.LastEventAt = &ts
		changed = true
	}
	if node.StatusReason == "" {
		switch node.Status {
		case NodeStatusActive:
			node.StatusReason = TransitionReasonManualPromote
		case NodeStatusQuarantine:
			node.StatusReason = TransitionReasonMigrationLegacyDemoted
		case NodeStatusRemoved:
			node.StatusReason = TransitionReasonManualRemove
		case NodeStatusCandidate:
			node.StatusReason = TransitionReasonOutboundRegistrationFailed
		default:
			node.StatusReason = TransitionReasonSubscriptionNodeDiscovered
		}
		changed = true
	}
	return changed
}

func copyRecentNodeEvents(events []NodeEvent, limit int) []NodeEvent {
	if limit <= 0 || limit > len(events) {
		limit = len(events)
	}
	result := make([]NodeEvent, 0, limit)
	for i := len(events) - 1; i >= 0 && len(result) < limit; i-- {
		result = append(result, events[i])
	}
	return result
}

func hasLegacyStatusTag(tag string) bool {
	return len(tag) > 0 && (len(tag) >= len(legacyStagingTagPrefix) && tag[:len(legacyStagingTagPrefix)] == legacyStagingTagPrefix ||
		len(tag) >= len(legacyActiveTagPrefix) && tag[:len(legacyActiveTagPrefix)] == legacyActiveTagPrefix)
}

func isProbeableStatus(status NodeStatus) bool {
	switch status {
	case NodeStatusStaging, NodeStatusActive, NodeStatusQuarantine:
		return true
	default:
		return false
	}
}

func isAllowedNodeTransition(current, next NodeStatus) bool {
	if current == next {
		return true
	}

	switch current {
	case NodeStatusCandidate:
		return next == NodeStatusStaging || next == NodeStatusRemoved
	case NodeStatusStaging:
		return next == NodeStatusActive || next == NodeStatusQuarantine || next == NodeStatusRemoved || next == NodeStatusCandidate
	case NodeStatusActive:
		return next == NodeStatusQuarantine || next == NodeStatusRemoved || next == NodeStatusCandidate
	case NodeStatusQuarantine:
		return next == NodeStatusActive || next == NodeStatusRemoved || next == NodeStatusCandidate
	case NodeStatusRemoved:
		return next == NodeStatusStaging || next == NodeStatusCandidate
	default:
		return false
	}
}

func isCountedSubscriptionNode(node NodeRecord) bool {
	return node.Status != NodeStatusRemoved && node.StatusReason != TransitionReasonSubscriptionMissing
}

func probeOutboundTag(nodeID string) string {
	return nodeProbeTagPrefix + nodeID
}

func nodeTime(node NodeRecord) time.Time {
	if node.StatusUpdatedAt != nil {
		return *node.StatusUpdatedAt
	}
	if node.LastEventAt != nil {
		return *node.LastEventAt
	}
	if node.LastCheckedAt != nil {
		return *node.LastCheckedAt
	}
	return node.AddedAt
}

func stopTimer(timer *time.Timer) {
	if timer == nil {
		return
	}
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}

func hashID(input string) string {
	h := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", h[:6])
}

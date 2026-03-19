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
	"sync"
	"time"

	handlerservice "github.com/xtls/xray-core/app/proxyman/command"
	"github.com/xtls/xray-core/common/errors"
	core "github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/features/routing"
	"google.golang.org/protobuf/encoding/protojson"
)

// NodePoolState is the top-level persisted state.
type NodePoolState struct {
	Subscriptions    []SubscriptionRecord `json:"subscriptions"`
	Nodes            []NodeRecord         `json:"nodes"`
	ValidationConfig ValidationConfig     `json:"validationConfig"`
}

// SubscriptionRecord represents a subscription source.
type SubscriptionRecord struct {
	ID              string     `json:"id"`
	URL             string     `json:"url"`
	Remark          string     `json:"remark"`
	AutoRefresh     bool       `json:"autoRefresh"`
	RefreshInterval int        `json:"refreshIntervalMin"`
	LastRefresh     *time.Time `json:"lastRefresh,omitempty"`
	NodeCount       int        `json:"nodeCount"`
}

// NodeRecord represents a node in the pool.
type NodeRecord struct {
	ID               string     `json:"id"`
	URI              string     `json:"uri"`
	Remark           string     `json:"remark"`
	Protocol         string     `json:"protocol"`
	Address          string     `json:"address"`
	Port             int        `json:"port"`
	OutboundTag      string     `json:"outboundTag"`
	Status           string     `json:"status"` // staging / active / demoted
	SubscriptionID   string     `json:"subscriptionId"`
	AddedAt          time.Time  `json:"addedAt"`
	PromotedAt       *time.Time `json:"promotedAt,omitempty"`
	TotalPings       int        `json:"totalPings"`
	FailedPings      int        `json:"failedPings"`
	AvgDelayMs       int64      `json:"avgDelayMs"`
	ConsecutiveFails int        `json:"consecutiveFails"`
	LastCheckedAt    *time.Time `json:"lastCheckedAt,omitempty"`
}

// ValidationConfig holds the criteria for promoting/demoting nodes.
type ValidationConfig struct {
	MinSamples       int     `json:"minSamples"`
	MaxFailRate      float64 `json:"maxFailRate"`
	MaxAvgDelayMs    int64   `json:"maxAvgDelayMs"`
	ProbeIntervalSec int     `json:"probeIntervalSec"`
	ProbeURL         string  `json:"probeUrl"`
	DemoteAfterFails int     `json:"demoteAfterFails"`
	AutoRemoveDemoted bool   `json:"autoRemoveDemoted"`
}

// SubscriptionManager manages subscriptions and the node pool lifecycle.
type SubscriptionManager struct {
	mu         sync.RWMutex
	state      *NodePoolState
	statePath  string
	grpcClient *GRPCClient
	prober     *NodeProber
	instance   *core.Instance
	stopCh     chan struct{}
	refreshMu  sync.Mutex // prevents concurrent refreshes
}

// NewSubscriptionManager creates a new SubscriptionManager.
func NewSubscriptionManager(configPath string, grpcClient *GRPCClient, instance *core.Instance) *SubscriptionManager {
	statePath := filepath.Join(filepath.Dir(configPath), "node_pool_state.json")

	sm := &SubscriptionManager{
		statePath:  statePath,
		grpcClient: grpcClient,
		instance:   instance,
		stopCh:     make(chan struct{}),
	}

	sm.state = sm.loadState()
	return sm
}

// Start initializes the prober and begins auto-refresh loops.
func (sm *SubscriptionManager) Start() error {
	// Get routing dispatcher for probing
	var dispatcher routing.Dispatcher
	if sm.instance != nil {
		if f := sm.instance.GetFeature(routing.DispatcherType()); f != nil {
			dispatcher = f.(routing.Dispatcher)
		}
	}

	if dispatcher == nil {
		errors.LogWarning(context.Background(), "node pool: routing dispatcher not available, probing disabled")
	} else {
		sm.mu.RLock()
		cfg := sm.state.ValidationConfig
		sm.mu.RUnlock()

		sm.prober = NewNodeProber(dispatcher, cfg.ProbeURL, cfg.ProbeIntervalSec, sm.handleProbeResults)
		// Register all existing staging/active nodes for probing
		sm.mu.RLock()
		for _, n := range sm.state.Nodes {
			if n.Status == "staging" || n.Status == "active" {
				sm.prober.AddTag(n.OutboundTag)
			}
		}
		sm.mu.RUnlock()
		sm.prober.Start()
	}

	// Start auto-refresh goroutine
	go sm.autoRefreshLoop()

	// Re-register existing staging/active outbounds
	go sm.reregisterOutbounds()

	return nil
}

// Stop shuts down the manager.
func (sm *SubscriptionManager) Stop() {
	close(sm.stopCh)
	if sm.prober != nil {
		sm.prober.Stop()
	}
}

// --- Subscription CRUD ---

// ListSubscriptions returns all subscriptions.
func (sm *SubscriptionManager) ListSubscriptions() []SubscriptionRecord {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	result := make([]SubscriptionRecord, len(sm.state.Subscriptions))
	copy(result, sm.state.Subscriptions)

	// Update node counts
	for i := range result {
		count := 0
		for _, n := range sm.state.Nodes {
			if n.SubscriptionID == result[i].ID {
				count++
			}
		}
		result[i].NodeCount = count
	}
	return result
}

// AddSubscription adds a new subscription and immediately fetches it.
func (sm *SubscriptionManager) AddSubscription(urlStr, remark string, autoRefresh bool, refreshIntervalMin int) (*SubscriptionRecord, error) {
	id := hashID(urlStr)

	sm.mu.Lock()
	// Check for duplicates
	for _, s := range sm.state.Subscriptions {
		if s.ID == id {
			sm.mu.Unlock()
			return nil, fmt.Errorf("subscription already exists")
		}
	}

	if refreshIntervalMin <= 0 {
		refreshIntervalMin = 60
	}

	rec := SubscriptionRecord{
		ID:              id,
		URL:             urlStr,
		Remark:          remark,
		AutoRefresh:     autoRefresh,
		RefreshInterval: refreshIntervalMin,
	}
	sm.state.Subscriptions = append(sm.state.Subscriptions, rec)
	sm.mu.Unlock()

	// Fetch and parse
	if err := sm.refreshSubscription(id); err != nil {
		// Roll back
		sm.mu.Lock()
		sm.removeSubscriptionLocked(id)
		sm.mu.Unlock()
		return nil, fmt.Errorf("failed to fetch subscription: %w", err)
	}

	sm.mu.RLock()
	for _, s := range sm.state.Subscriptions {
		if s.ID == id {
			rec = s
			break
		}
	}
	sm.mu.RUnlock()

	return &rec, nil
}

// DeleteSubscription removes a subscription and all its nodes.
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

	// Remove all nodes belonging to this subscription
	var remaining []NodeRecord
	for _, n := range sm.state.Nodes {
		if n.SubscriptionID == id {
			sm.removeOutboundAsync(n.OutboundTag)
			if sm.prober != nil {
				sm.prober.RemoveTag(n.OutboundTag)
			}
		} else {
			remaining = append(remaining, n)
		}
	}
	sm.state.Nodes = remaining
	sm.removeSubscriptionLocked(id)
	sm.saveStateLocked()
	return nil
}

// RefreshSubscription triggers a refresh of a specific subscription.
func (sm *SubscriptionManager) RefreshSubscription(id string) error {
	return sm.refreshSubscription(id)
}

// --- Node Pool Operations ---

// ListNodes returns nodes filtered by status (empty string = all).
func (sm *SubscriptionManager) ListNodes(status string) []NodeRecord {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var result []NodeRecord
	for _, n := range sm.state.Nodes {
		if status == "" || n.Status == status {
			result = append(result, n)
		}
	}
	return result
}

// PromoteNode manually promotes a staging node to active.
func (sm *SubscriptionManager) PromoteNode(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i, n := range sm.state.Nodes {
		if n.ID == id {
			if n.Status != "staging" {
				return fmt.Errorf("node is not in staging status")
			}
			return sm.promoteNodeLocked(i)
		}
	}
	return fmt.Errorf("node not found")
}

// DemoteNode manually demotes an active node.
func (sm *SubscriptionManager) DemoteNode(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i, n := range sm.state.Nodes {
		if n.ID == id {
			if n.Status != "active" {
				return fmt.Errorf("node is not in active status")
			}
			return sm.demoteNodeLocked(i)
		}
	}
	return fmt.Errorf("node not found")
}

// DeleteNode removes a single node.
func (sm *SubscriptionManager) DeleteNode(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i, n := range sm.state.Nodes {
		if n.ID == id {
			sm.removeOutboundAsync(n.OutboundTag)
			if sm.prober != nil {
				sm.prober.RemoveTag(n.OutboundTag)
			}
			sm.state.Nodes = append(sm.state.Nodes[:i], sm.state.Nodes[i+1:]...)
			sm.saveStateLocked()
			return nil
		}
	}
	return fmt.Errorf("node not found")
}

// GetValidationConfig returns the current validation config.
func (sm *SubscriptionManager) GetValidationConfig() ValidationConfig {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state.ValidationConfig
}

// UpdateValidationConfig updates the validation config.
func (sm *SubscriptionManager) UpdateValidationConfig(cfg ValidationConfig) {
	sm.mu.Lock()
	sm.state.ValidationConfig = cfg
	sm.saveStateLocked()
	sm.mu.Unlock()

	if sm.prober != nil {
		sm.prober.UpdateConfig(cfg.ProbeURL, cfg.ProbeIntervalSec)
	}
}

// --- Internal ---

func (sm *SubscriptionManager) refreshSubscription(id string) error {
	sm.refreshMu.Lock()
	defer sm.refreshMu.Unlock()

	sm.mu.RLock()
	var sub *SubscriptionRecord
	for i := range sm.state.Subscriptions {
		if sm.state.Subscriptions[i].ID == id {
			sub = &sm.state.Subscriptions[i]
			break
		}
	}
	sm.mu.RUnlock()

	if sub == nil {
		return fmt.Errorf("subscription not found")
	}

	// Fetch subscription URL
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(sub.URL)
	if err != nil {
		return fmt.Errorf("failed to fetch subscription: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10MB limit
	if err != nil {
		return fmt.Errorf("failed to read subscription body: %w", err)
	}

	links, err := ParseSubscriptionContent(string(body))
	if err != nil {
		return fmt.Errorf("failed to parse subscription content: %w", err)
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Build map of existing nodes for this subscription
	existingByID := make(map[string]int)
	for i, n := range sm.state.Nodes {
		if n.SubscriptionID == id {
			existingByID[n.ID] = i
		}
	}

	// Process new links
	newNodeIDs := make(map[string]bool)
	for _, link := range links {
		uri, _ := GenerateShareLink(*link)
		if uri == "" {
			continue
		}
		nodeID := hashID(uri)
		newNodeIDs[nodeID] = true

		if _, exists := existingByID[nodeID]; exists {
			continue // already tracked
		}

		// New node → register as staging
		tag := "staging_" + nodeID
		node := NodeRecord{
			ID:             nodeID,
			URI:            uri,
			Remark:         link.Remark,
			Protocol:       link.Protocol,
			Address:        link.Address,
			Port:           link.Port,
			OutboundTag:    tag,
			Status:         "staging",
			SubscriptionID: id,
			AddedAt:        time.Now(),
		}

		if err := sm.addOutboundFromLink(link, tag); err != nil {
			errors.LogWarning(context.Background(), "node pool: failed to add outbound for ", tag, ": ", err.Error())
			continue
		}

		sm.state.Nodes = append(sm.state.Nodes, node)
		if sm.prober != nil {
			sm.prober.AddTag(tag)
		}
	}

	// Remove nodes that are no longer in the subscription
	var remaining []NodeRecord
	for _, n := range sm.state.Nodes {
		if n.SubscriptionID == id && !newNodeIDs[n.ID] {
			sm.removeOutboundAsync(n.OutboundTag)
			if sm.prober != nil {
				sm.prober.RemoveTag(n.OutboundTag)
			}
			continue
		}
		remaining = append(remaining, n)
	}
	sm.state.Nodes = remaining

	// Update subscription record
	now := time.Now()
	for i := range sm.state.Subscriptions {
		if sm.state.Subscriptions[i].ID == id {
			sm.state.Subscriptions[i].LastRefresh = &now
			count := 0
			for _, n := range sm.state.Nodes {
				if n.SubscriptionID == id {
					count++
				}
			}
			sm.state.Subscriptions[i].NodeCount = count
			break
		}
	}

	sm.saveStateLocked()
	return nil
}

func (sm *SubscriptionManager) handleProbeResults(results []ProbeResult) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	cfg := sm.state.ValidationConfig
	needSave := false

	for _, r := range results {
		idx := sm.findNodeByTag(r.Tag)
		if idx < 0 {
			continue
		}
		node := &sm.state.Nodes[idx]
		node.TotalPings++
		node.LastCheckedAt = &now
		needSave = true

		if r.Success {
			node.ConsecutiveFails = 0
			// Update running average delay
			if node.AvgDelayMs == 0 {
				node.AvgDelayMs = r.DelayMs
			} else {
				node.AvgDelayMs = (node.AvgDelayMs*int64(node.TotalPings-node.FailedPings-1) + r.DelayMs) / int64(node.TotalPings-node.FailedPings)
			}
		} else {
			node.FailedPings++
			node.ConsecutiveFails++
		}

		// Auto-promote: staging → active
		if node.Status == "staging" && node.TotalPings >= cfg.MinSamples {
			failRate := float64(node.FailedPings) / float64(node.TotalPings)
			if failRate <= cfg.MaxFailRate && node.AvgDelayMs <= cfg.MaxAvgDelayMs {
				sm.promoteNodeLocked(idx)
			}
		}

		// Auto-demote: active → demoted
		if node.Status == "active" && node.ConsecutiveFails >= cfg.DemoteAfterFails {
			sm.demoteNodeLocked(idx)
		}
	}

	if needSave {
		sm.saveStateLocked()
	}
}

func (sm *SubscriptionManager) promoteNodeLocked(idx int) error {
	node := &sm.state.Nodes[idx]
	oldTag := node.OutboundTag
	newTag := "active_" + node.ID

	// Parse the original link to rebuild outbound
	link, err := ParseShareLinkURI(node.URI)
	if err != nil {
		return fmt.Errorf("failed to parse node URI: %w", err)
	}

	// Remove old staging outbound
	sm.removeOutboundAsync(oldTag)
	if sm.prober != nil {
		sm.prober.RemoveTag(oldTag)
	}

	// Add new active outbound
	if err := sm.addOutboundFromLink(link, newTag); err != nil {
		return fmt.Errorf("failed to add active outbound: %w", err)
	}

	now := time.Now()
	node.OutboundTag = newTag
	node.Status = "active"
	node.PromotedAt = &now

	if sm.prober != nil {
		sm.prober.AddTag(newTag)
	}

	errors.LogInfo(context.Background(), "node pool: promoted node ", node.Remark, " (", node.ID, ")")
	return nil
}

func (sm *SubscriptionManager) demoteNodeLocked(idx int) error {
	node := &sm.state.Nodes[idx]

	sm.removeOutboundAsync(node.OutboundTag)
	if sm.prober != nil {
		sm.prober.RemoveTag(node.OutboundTag)
	}

	node.Status = "demoted"
	node.OutboundTag = ""
	errors.LogInfo(context.Background(), "node pool: demoted node ", node.Remark, " (", node.ID, ")")

	if sm.state.ValidationConfig.AutoRemoveDemoted {
		sm.state.Nodes = append(sm.state.Nodes[:idx], sm.state.Nodes[idx+1:]...)
	}

	return nil
}

func (sm *SubscriptionManager) addOutboundFromLink(link *ShareLinkRequest, tag string) error {
	outJSON, err := BuildOutboundJSON(link, tag)
	if err != nil {
		return err
	}

	var outboundConfig core.OutboundHandlerConfig
	if err := protojson.Unmarshal(outJSON, &outboundConfig); err != nil {
		return fmt.Errorf("failed to unmarshal outbound config: %w", err)
	}

	_, err = sm.grpcClient.Handler().AddOutbound(sm.grpcClient.Context(), &handlerservice.AddOutboundRequest{
		Outbound: &outboundConfig,
	})
	return err
}

func (sm *SubscriptionManager) removeOutboundAsync(tag string) {
	if tag == "" {
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
	var remaining []SubscriptionRecord
	for _, s := range sm.state.Subscriptions {
		if s.ID != id {
			remaining = append(remaining, s)
		}
	}
	sm.state.Subscriptions = remaining
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
				if sub.LastRefresh != nil {
					elapsed := time.Since(*sub.LastRefresh)
					if elapsed < time.Duration(sub.RefreshInterval)*time.Minute {
						continue
					}
				}
				if err := sm.refreshSubscription(sub.ID); err != nil {
					errors.LogWarning(context.Background(), "node pool: auto-refresh failed for ", sub.Remark, ": ", err.Error())
				}
			}
		}
	}
}

func (sm *SubscriptionManager) reregisterOutbounds() {
	// Wait for gRPC to be ready
	time.Sleep(2 * time.Second)

	sm.mu.RLock()
	nodes := make([]NodeRecord, len(sm.state.Nodes))
	copy(nodes, sm.state.Nodes)
	sm.mu.RUnlock()

	for _, n := range nodes {
		if n.Status == "staging" || n.Status == "active" {
			link, err := ParseShareLinkURI(n.URI)
			if err != nil {
				continue
			}
			if err := sm.addOutboundFromLink(link, n.OutboundTag); err != nil {
				errors.LogDebug(context.Background(), "node pool: failed to re-register outbound ", n.OutboundTag, ": ", err.Error())
			}
		}
	}
}

// --- Persistence ---

func (sm *SubscriptionManager) loadState() *NodePoolState {
	state := &NodePoolState{
		ValidationConfig: defaultValidationConfig(),
	}

	data, err := os.ReadFile(sm.statePath)
	if err != nil {
		return state
	}

	if err := json.Unmarshal(data, state); err != nil {
		errors.LogWarning(context.Background(), "node pool: failed to parse state file: ", err.Error())
		return &NodePoolState{ValidationConfig: defaultValidationConfig()}
	}

	// Ensure defaults
	applyValidationDefaults(&state.ValidationConfig)
	return state
}

func (sm *SubscriptionManager) saveStateLocked() {
	data, err := json.MarshalIndent(sm.state, "", "  ")
	if err != nil {
		errors.LogWarning(context.Background(), "node pool: failed to marshal state: ", err.Error())
		return
	}

	dir := filepath.Dir(sm.statePath)
	os.MkdirAll(dir, 0o755)

	if err := os.WriteFile(sm.statePath, data, 0o644); err != nil {
		errors.LogWarning(context.Background(), "node pool: failed to save state: ", err.Error())
	}
}

func defaultValidationConfig() ValidationConfig {
	return ValidationConfig{
		MinSamples:        10,
		MaxFailRate:       0.3,
		MaxAvgDelayMs:     1000,
		ProbeIntervalSec:  60,
		ProbeURL:          "https://www.gstatic.com/generate_204",
		DemoteAfterFails:  5,
		AutoRemoveDemoted: false,
	}
}

func applyValidationDefaults(cfg *ValidationConfig) {
	if cfg.MinSamples <= 0 {
		cfg.MinSamples = 10
	}
	if cfg.MaxFailRate <= 0 {
		cfg.MaxFailRate = 0.3
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
}

func hashID(input string) string {
	h := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", h[:6]) // 12 hex chars
}

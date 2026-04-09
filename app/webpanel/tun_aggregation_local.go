package webpanel

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const tunAggregationPrototypeSessionTTL = 45 * time.Second

type TunAggregationPrototypeStatus struct {
	Ready              bool                             `json:"ready"`
	MetricSource       string                           `json:"metricSource"`
	SessionTTLSeconds  int                              `json:"sessionTtlSeconds"`
	CandidatePathCount int                              `json:"candidatePathCount"`
	SelectedPathCount  int                              `json:"selectedPathCount"`
	SessionCount       int                              `json:"sessionCount"`
	Paths              []TunAggregationPrototypePath    `json:"paths,omitempty"`
	Sessions           []TunAggregationPrototypeSession `json:"sessions,omitempty"`
	Note               string                           `json:"note,omitempty"`
}

type TunAggregationPrototypePath struct {
	NodeID           string     `json:"nodeId"`
	Remark           string     `json:"remark,omitempty"`
	OutboundTag      string     `json:"outboundTag"`
	State            string     `json:"state"`
	Eligible         bool       `json:"eligible"`
	Selected         bool       `json:"selected"`
	Score            float64    `json:"score"`
	LatencyMs        int64      `json:"latencyMs"`
	LossPct          float64    `json:"lossPct"`
	ConsecutiveFails int        `json:"consecutiveFails"`
	LastCheckedAt    *time.Time `json:"lastCheckedAt,omitempty"`
	Reason           string     `json:"reason"`
}

type TunAggregationPrototypeSession struct {
	SessionID        string    `json:"sessionId"`
	State            string    `json:"state"`
	Flow             string    `json:"flow"`
	SchedulerPolicy  string    `json:"schedulerPolicy"`
	CandidatePathIDs []string  `json:"candidatePathIds,omitempty"`
	SelectedPathIDs  []string  `json:"selectedPathIds,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
	LastSeenAt       time.Time `json:"lastSeenAt"`
	ExpiresAt        time.Time `json:"expiresAt"`
	Reason           string    `json:"reason"`
}

type tunAggregationPreviewFlowKey struct {
	Protocol string
	Target   string
	Port     int
}

func (k tunAggregationPreviewFlowKey) sessionID() string {
	return fmt.Sprintf("%s|%s|%d", strings.TrimSpace(k.Protocol), strings.TrimSpace(k.Target), k.Port)
}

type tunAggregationSessionStore struct {
	ttl      time.Duration
	sessions map[string]TunAggregationPrototypeSession
}

func newTunAggregationSessionStore(ttl time.Duration) *tunAggregationSessionStore {
	return &tunAggregationSessionStore{
		ttl:      ttl,
		sessions: map[string]TunAggregationPrototypeSession{},
	}
}

func (s *tunAggregationSessionStore) UpsertPreviewSession(key tunAggregationPreviewFlowKey, flow, policy string, candidatePathIDs, selectedPathIDs []string, now time.Time, reason string) {
	if s == nil {
		return
	}

	sessionID := key.sessionID()
	session, ok := s.sessions[sessionID]
	if !ok {
		session = TunAggregationPrototypeSession{
			SessionID: sessionID,
			State:     "planned",
			Flow:      flow,
			CreatedAt: now,
		}
	}

	session.SchedulerPolicy = policy
	session.CandidatePathIDs = append([]string(nil), candidatePathIDs...)
	session.SelectedPathIDs = append([]string(nil), selectedPathIDs...)
	session.LastSeenAt = now
	session.ExpiresAt = now.Add(s.ttl)
	session.Reason = reason
	s.sessions[sessionID] = session
}

func (s *tunAggregationSessionStore) Snapshot(now time.Time) []TunAggregationPrototypeSession {
	if s == nil {
		return nil
	}

	for id, session := range s.sessions {
		if !session.ExpiresAt.IsZero() && now.After(session.ExpiresAt) {
			delete(s.sessions, id)
		}
	}

	snapshots := make([]TunAggregationPrototypeSession, 0, len(s.sessions))
	for _, session := range s.sessions {
		snapshots = append(snapshots, session)
	}
	sort.Slice(snapshots, func(i, j int) bool {
		if snapshots[i].CreatedAt.Equal(snapshots[j].CreatedAt) {
			return snapshots[i].SessionID < snapshots[j].SessionID
		}
		return snapshots[i].CreatedAt.Before(snapshots[j].CreatedAt)
	})
	return snapshots
}

func attachTunAggregationPrototype(status *TunAggregationStatus, settings *TunFeatureSettings, activeNodes []NodeRecord, now time.Time) {
	if status == nil || settings == nil {
		return
	}

	status.Prototype = buildTunAggregationPrototype(settings.Aggregation, activeNodes, now)
}

func buildTunAggregationPrototype(settings TunAggregationSettings, activeNodes []NodeRecord, now time.Time) *TunAggregationPrototypeStatus {
	aggregation := normalizeTunAggregationSettings(settings)
	if !aggregation.Enabled {
		return nil
	}

	paths := buildTunAggregationPrototypePaths(activeNodes, now)
	candidatePathIDs := make([]string, 0, len(paths))
	for _, path := range paths {
		if path.Eligible {
			candidatePathIDs = append(candidatePathIDs, path.NodeID)
		}
	}

	selectedPathIDs, decisionReason := selectTunAggregationPrototypePaths(paths, aggregation)

	store := newTunAggregationSessionStore(tunAggregationPrototypeSessionTTL)
	if len(selectedPathIDs) > 0 {
		store.UpsertPreviewSession(
			tunAggregationPreviewFlowKey{
				Protocol: "quic",
				Target:   tunAggregationPreviewTarget(aggregation),
				Port:     443,
			},
			"QUIC preview session",
			aggregation.SchedulerPolicy,
			candidatePathIDs,
			selectedPathIDs,
			now,
			decisionReason,
		)
	}

	sessions := store.Snapshot(now)
	return &TunAggregationPrototypeStatus{
		Ready:              len(selectedPathIDs) > 0,
		MetricSource:       "node_pool_probe_history",
		SessionTTLSeconds:  int(tunAggregationPrototypeSessionTTL.Seconds()),
		CandidatePathCount: len(candidatePathIDs),
		SelectedPathCount:  len(selectedPathIDs),
		SessionCount:       len(sessions),
		Paths:              paths,
		Sessions:           sessions,
		Note:               "Local prototype only. Path quality comes from current node-pool probe history, while live packet steering and relay integration remain disabled.",
	}
}

func buildTunAggregationPrototypePaths(activeNodes []NodeRecord, now time.Time) []TunAggregationPrototypePath {
	paths := make([]TunAggregationPrototypePath, 0, len(activeNodes))
	for _, node := range activeNodes {
		eligible, reason := tunAggregationPrototypeEligibility(node, now)
		latencyMs := node.AvgDelayMs
		if latencyMs < 0 {
			latencyMs = 0
		}

		path := TunAggregationPrototypePath{
			NodeID:           node.ID,
			Remark:           strings.TrimSpace(node.Remark),
			OutboundTag:      buildTunOutboundTag(node.ID),
			State:            "excluded",
			Eligible:         eligible,
			Selected:         false,
			Score:            tunAggregationPrototypeScore(latencyMs, tunAggregationLossPct(node), node.ConsecutiveFails, eligible),
			LatencyMs:        latencyMs,
			LossPct:          tunAggregationLossPct(node),
			ConsecutiveFails: node.ConsecutiveFails,
			LastCheckedAt:    node.LastCheckedAt,
			Reason:           reason,
		}
		if eligible {
			path.State = "standby"
		}
		paths = append(paths, path)
	}

	sort.Slice(paths, func(i, j int) bool {
		if paths[i].Eligible != paths[j].Eligible {
			return paths[i].Eligible
		}
		if paths[i].Score != paths[j].Score {
			return paths[i].Score < paths[j].Score
		}
		if paths[i].LatencyMs != paths[j].LatencyMs {
			return paths[i].LatencyMs < paths[j].LatencyMs
		}
		return paths[i].NodeID < paths[j].NodeID
	})
	return paths
}

func selectTunAggregationPrototypePaths(paths []TunAggregationPrototypePath, aggregation TunAggregationSettings) ([]string, string) {
	candidateIndexes := make([]int, 0, len(paths))
	for i := range paths {
		if paths[i].Eligible {
			candidateIndexes = append(candidateIndexes, i)
		}
	}

	if len(candidateIndexes) == 0 {
		return nil, "No active nodes met the local prototype eligibility rules."
	}

	limit := aggregation.MaxPathsPerSession
	if limit < 1 {
		limit = 1
	}

	policy := normalizeTunAggregationSchedulerPolicy(aggregation.SchedulerPolicy)
	selectedCount := 1
	switch policy {
	case TunAggregationSchedulerPolicyRedundant2:
		selectedCount = 2
	case TunAggregationSchedulerPolicyWeightedSplit:
		selectedCount = limit
	default:
		selectedCount = 1
	}
	if selectedCount > limit {
		selectedCount = limit
	}
	if selectedCount > len(candidateIndexes) {
		selectedCount = len(candidateIndexes)
	}

	selectedPathIDs := make([]string, 0, selectedCount)
	for rank, index := range candidateIndexes {
		if rank < selectedCount {
			paths[index].Selected = true
			paths[index].State = "selected"
			paths[index].Reason = fmt.Sprintf("Selected by %s from %d eligible path(s).", policy, len(candidateIndexes))
			selectedPathIDs = append(selectedPathIDs, paths[index].NodeID)
			continue
		}

		paths[index].Selected = false
		paths[index].State = "standby"
		paths[index].Reason = fmt.Sprintf("Healthy standby path; %s keeps %d path(s) active.", policy, selectedCount)
	}

	return selectedPathIDs, fmt.Sprintf("%s chose %d of %d eligible path(s).", policy, selectedCount, len(candidateIndexes))
}

func tunAggregationPreviewTarget(aggregation TunAggregationSettings) string {
	if relay := strings.TrimSpace(aggregation.RelayEndpoint); relay != "" {
		return relay
	}
	return "preview.local"
}

func tunAggregationPrototypeEligibility(node NodeRecord, now time.Time) (bool, string) {
	switch {
	case strings.TrimSpace(node.ID) == "":
		return false, "Excluded because the active node has no stable id."
	case strings.EqualFold(strings.TrimSpace(node.Protocol), "hysteria2"):
		return false, "Excluded because transparent mode currently skips hysteria2 nodes."
	case node.ConsecutiveFails > 0:
		return false, fmt.Sprintf("Excluded because consecutiveFails=%d.", node.ConsecutiveFails)
	case node.AvgDelayMs <= 0:
		return false, "Excluded because there is no usable latency sample yet."
	case node.LastCheckedAt == nil:
		return false, "Excluded because the node has not been checked recently."
	case now.Sub(*node.LastCheckedAt) > tunEligibleProbeFreshness:
		return false, "Excluded because the last node-pool probe is stale."
	default:
		return true, "Eligible candidate path from current active node-pool probe history."
	}
}

func tunAggregationLossPct(node NodeRecord) float64 {
	if node.TotalPings <= 0 {
		return 0
	}
	return (float64(node.FailedPings) / float64(node.TotalPings)) * 100
}

func tunAggregationPrototypeScore(latencyMs int64, lossPct float64, consecutiveFails int, eligible bool) float64 {
	score := float64(latencyMs) + lossPct*1.5
	score += float64(consecutiveFails * 250)
	if !eligible {
		score += 5000
	}
	return score
}

func formatTunAggregationPrototypeDiagnostic(prototype *TunAggregationPrototypeStatus) string {
	if prototype == nil {
		return ""
	}

	return fmt.Sprintf(
		"Aggregation prototype: candidates=%d selected=%d sessions=%d ttl=%ds source=%s",
		prototype.CandidatePathCount,
		prototype.SelectedPathCount,
		prototype.SessionCount,
		prototype.SessionTTLSeconds,
		prototype.MetricSource,
	)
}

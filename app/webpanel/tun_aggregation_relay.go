package webpanel

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	tunAggregationRelayContractVersion    = "relay_preview_v1"
	tunAggregationBenchmarkPacketCount    = 48
	tunAggregationBenchmarkPayloadBytes   = 1200
	tunAggregationBenchmarkSendInterval   = 20 * time.Millisecond
	tunAggregationBenchmarkStallThreshold = 120 * time.Millisecond
)

type TunAggregationRelayStatus struct {
	Ready                 bool                               `json:"ready"`
	ContractVersion       string                             `json:"contractVersion"`
	Endpoint              string                             `json:"endpoint,omitempty"`
	SessionCount          int                                `json:"sessionCount"`
	PacketCount           int                                `json:"packetCount"`
	DeliveredPacketCount  int                                `json:"deliveredPacketCount"`
	DuplicateDrops        int                                `json:"duplicateDrops"`
	ReorderedPackets      int                                `json:"reorderedPackets"`
	MaxReorderBufferDepth int                                `json:"maxReorderBufferDepth"`
	Sessions              []TunAggregationRelaySessionStatus `json:"sessions,omitempty"`
	Note                  string                             `json:"note,omitempty"`
}

type TunAggregationRelaySessionStatus struct {
	SessionID             string    `json:"sessionId"`
	Flow                  string    `json:"flow"`
	SchedulerPolicy       string    `json:"schedulerPolicy"`
	PathIDs               []string  `json:"pathIds,omitempty"`
	PacketCount           int       `json:"packetCount"`
	DeliveredPacketCount  int       `json:"deliveredPacketCount"`
	DuplicateDrops        int       `json:"duplicateDrops"`
	ReorderedPackets      int       `json:"reorderedPackets"`
	MaxReorderBufferDepth int       `json:"maxReorderBufferDepth"`
	DeliveredBytes        int       `json:"deliveredBytes"`
	StartupLatencyMs      int64     `json:"startupLatencyMs"`
	StallCount            int       `json:"stallCount"`
	GoodputKbps           float64   `json:"goodputKbps"`
	Reason                string    `json:"reason"`
	CreatedAt             time.Time `json:"createdAt"`
}

type TunAggregationBenchmarkScenarioName string

const (
	TunAggregationBenchmarkScenarioCleanPaths      TunAggregationBenchmarkScenarioName = "clean_paths"
	TunAggregationBenchmarkScenarioDegradedPrimary TunAggregationBenchmarkScenarioName = "degraded_primary"
)

type TunAggregationBenchmarkStatus struct {
	Ready        bool                                    `json:"ready"`
	PacketCount  int                                     `json:"packetCount"`
	PayloadBytes int                                     `json:"payloadBytes"`
	Scenarios    []TunAggregationBenchmarkScenarioStatus `json:"scenarios,omitempty"`
	Note         string                                  `json:"note,omitempty"`
}

type TunAggregationBenchmarkScenarioStatus struct {
	Name                 string                        `json:"name"`
	Baseline             TunAggregationBenchmarkResult `json:"baseline"`
	Aggregated           TunAggregationBenchmarkResult `json:"aggregated"`
	StartupLatencyGainMs int64                         `json:"startupLatencyGainMs"`
	StallReduction       int                           `json:"stallReduction"`
	GoodputGainKbps      float64                       `json:"goodputGainKbps"`
	LossReductionPct     float64                       `json:"lossReductionPct"`
	StabilityGainPct     float64                       `json:"stabilityGainPct"`
}

type TunAggregationBenchmarkResult struct {
	StartupLatencyMs int64   `json:"startupLatencyMs"`
	StallCount       int     `json:"stallCount"`
	GoodputKbps      float64 `json:"goodputKbps"`
	LossPct          float64 `json:"lossPct"`
	StabilityPct     float64 `json:"stabilityPct"`
}

type tunAggregationPathProfile struct {
	ID        string
	LatencyMs int64
	JitterMs  int64
	LossPct   float64
}

type tunAggregationRelayEnvelope struct {
	SessionID    string
	Sequence     int
	PathID       string
	SentAt       time.Time
	ArrivedAt    time.Time
	PayloadBytes int
}

type tunAggregationSimulationResult struct {
	PacketCount           int
	DeliveredPacketCount  int
	DuplicateDrops        int
	ReorderedPackets      int
	MaxReorderBufferDepth int
	DeliveredBytes        int
	StartupLatencyMs      int64
	StallCount            int
	GoodputKbps           float64
	LossPct               float64
	StabilityPct          float64
}

type tunAggregationRelayAssembler struct {
	nextSequence     int
	buffered         map[int]tunAggregationRelayEnvelope
	deliveries       []tunAggregationRelayEnvelope
	duplicateDrops   int
	reorderedPackets int
	maxBufferDepth   int
}

func attachTunAggregationRelayDiagnostics(status *TunAggregationStatus, settings *TunFeatureSettings, now time.Time) {
	if status == nil || settings == nil || status.Prototype == nil {
		return
	}

	status.Relay = buildTunAggregationRelayStatus(settings.Aggregation, status.Prototype, now)
	status.Benchmark = buildTunAggregationBenchmarkStatus(settings.Aggregation, status.Prototype, now)
}

func buildTunAggregationRelayStatus(settings TunAggregationSettings, prototype *TunAggregationPrototypeStatus, now time.Time) *TunAggregationRelayStatus {
	aggregation := normalizeTunAggregationSettings(settings)
	if !aggregation.Enabled || strings.TrimSpace(aggregation.RelayEndpoint) == "" || prototype == nil {
		return nil
	}

	status := &TunAggregationRelayStatus{
		Ready:           false,
		ContractVersion: tunAggregationRelayContractVersion,
		Endpoint:        aggregation.RelayEndpoint,
		Note:            "Synthetic relay-side ingress/assembler preview driven by the local #41 session contract. The live transparent data path still stays on stable_single_path.",
	}

	for _, session := range prototype.Sessions {
		profiles := tunAggregationProfilesForSession(session, prototype, aggregation.Health, false)
		if len(profiles) == 0 {
			continue
		}

		policy := normalizeTunAggregationSchedulerPolicy(session.SchedulerPolicy)
		result := simulateTunAggregationSession(policy, profiles, now, tunAggregationBenchmarkPacketCount, tunAggregationBenchmarkPayloadBytes)

		pathIDs := make([]string, 0, len(profiles))
		for _, profile := range profiles {
			pathIDs = append(pathIDs, profile.ID)
		}

		status.SessionCount++
		status.PacketCount += result.PacketCount
		status.DeliveredPacketCount += result.DeliveredPacketCount
		status.DuplicateDrops += result.DuplicateDrops
		status.ReorderedPackets += result.ReorderedPackets
		if result.MaxReorderBufferDepth > status.MaxReorderBufferDepth {
			status.MaxReorderBufferDepth = result.MaxReorderBufferDepth
		}
		status.Sessions = append(status.Sessions, TunAggregationRelaySessionStatus{
			SessionID:             session.SessionID,
			Flow:                  session.Flow,
			SchedulerPolicy:       string(policy),
			PathIDs:               pathIDs,
			PacketCount:           result.PacketCount,
			DeliveredPacketCount:  result.DeliveredPacketCount,
			DuplicateDrops:        result.DuplicateDrops,
			ReorderedPackets:      result.ReorderedPackets,
			MaxReorderBufferDepth: result.MaxReorderBufferDepth,
			DeliveredBytes:        result.DeliveredBytes,
			StartupLatencyMs:      result.StartupLatencyMs,
			StallCount:            result.StallCount,
			GoodputKbps:           result.GoodputKbps,
			Reason:                fmt.Sprintf("%s relay preview over %s.", policy, strings.Join(pathIDs, ", ")),
			CreatedAt:             session.CreatedAt,
		})
	}

	sort.Slice(status.Sessions, func(i, j int) bool {
		if status.Sessions[i].CreatedAt.Equal(status.Sessions[j].CreatedAt) {
			return status.Sessions[i].SessionID < status.Sessions[j].SessionID
		}
		return status.Sessions[i].CreatedAt.Before(status.Sessions[j].CreatedAt)
	})

	status.Ready = status.SessionCount > 0
	if !status.Ready {
		status.Note = "Relay endpoint is configured, but the local prototype did not expose any selected session/path contract to replay."
	}
	return status
}

func buildTunAggregationBenchmarkStatus(settings TunAggregationSettings, prototype *TunAggregationPrototypeStatus, now time.Time) *TunAggregationBenchmarkStatus {
	aggregation := normalizeTunAggregationSettings(settings)
	if !aggregation.Enabled || strings.TrimSpace(aggregation.RelayEndpoint) == "" || prototype == nil || len(prototype.Sessions) == 0 {
		return nil
	}

	session := prototype.Sessions[0]
	cleanProfiles := tunAggregationProfilesForSession(session, prototype, aggregation.Health, false)
	if len(cleanProfiles) == 0 {
		return nil
	}
	degradedProfiles := tunAggregationProfilesForSession(session, prototype, aggregation.Health, true)
	policy := normalizeTunAggregationSchedulerPolicy(session.SchedulerPolicy)

	benchmark := &TunAggregationBenchmarkStatus{
		Ready:        true,
		PacketCount:  tunAggregationBenchmarkPacketCount,
		PayloadBytes: tunAggregationBenchmarkPayloadBytes,
		Note:         "Synthetic benchmark compares stable single-path against the experimental relay-side scheduler under clean and degraded path assumptions. It is evidence for continuation, not a claim of live public-Internet acceleration.",
		Scenarios: []TunAggregationBenchmarkScenarioStatus{
			buildTunAggregationBenchmarkScenario(
				TunAggregationBenchmarkScenarioCleanPaths,
				simulateTunAggregationSession(TunAggregationSchedulerPolicySingleBest, cleanProfiles[:1], now, tunAggregationBenchmarkPacketCount, tunAggregationBenchmarkPayloadBytes),
				simulateTunAggregationSession(policy, cleanProfiles, now, tunAggregationBenchmarkPacketCount, tunAggregationBenchmarkPayloadBytes),
			),
			buildTunAggregationBenchmarkScenario(
				TunAggregationBenchmarkScenarioDegradedPrimary,
				simulateTunAggregationSession(TunAggregationSchedulerPolicySingleBest, degradedProfiles[:1], now, tunAggregationBenchmarkPacketCount, tunAggregationBenchmarkPayloadBytes),
				simulateTunAggregationSession(policy, degradedProfiles, now, tunAggregationBenchmarkPacketCount, tunAggregationBenchmarkPayloadBytes),
			),
		},
	}

	return benchmark
}

func buildTunAggregationBenchmarkScenario(name TunAggregationBenchmarkScenarioName, baseline, aggregated tunAggregationSimulationResult) TunAggregationBenchmarkScenarioStatus {
	return TunAggregationBenchmarkScenarioStatus{
		Name:                 string(name),
		Baseline:             tunAggregationBenchmarkResultFromSimulation(baseline),
		Aggregated:           tunAggregationBenchmarkResultFromSimulation(aggregated),
		StartupLatencyGainMs: baseline.StartupLatencyMs - aggregated.StartupLatencyMs,
		StallReduction:       baseline.StallCount - aggregated.StallCount,
		GoodputGainKbps:      aggregated.GoodputKbps - baseline.GoodputKbps,
		LossReductionPct:     baseline.LossPct - aggregated.LossPct,
		StabilityGainPct:     aggregated.StabilityPct - baseline.StabilityPct,
	}
}

func tunAggregationBenchmarkResultFromSimulation(result tunAggregationSimulationResult) TunAggregationBenchmarkResult {
	return TunAggregationBenchmarkResult{
		StartupLatencyMs: result.StartupLatencyMs,
		StallCount:       result.StallCount,
		GoodputKbps:      result.GoodputKbps,
		LossPct:          result.LossPct,
		StabilityPct:     result.StabilityPct,
	}
}

func tunAggregationProfilesForSession(session TunAggregationPrototypeSession, prototype *TunAggregationPrototypeStatus, health TunAggregationHealthSettings, degradedPrimary bool) []tunAggregationPathProfile {
	if prototype == nil {
		return nil
	}

	pathByID := make(map[string]TunAggregationPrototypePath, len(prototype.Paths))
	selectedFallback := make([]string, 0, len(prototype.Paths))
	for _, path := range prototype.Paths {
		pathByID[path.NodeID] = path
		if path.Selected {
			selectedFallback = append(selectedFallback, path.NodeID)
		}
	}

	selectedIDs := append([]string(nil), session.SelectedPathIDs...)
	if len(selectedIDs) == 0 {
		selectedIDs = append(selectedIDs, selectedFallback...)
	}

	maxJitterMs := int64(health.MaxPathJitterMs)
	if maxJitterMs < 8 {
		maxJitterMs = 8
	}

	profiles := make([]tunAggregationPathProfile, 0, len(selectedIDs))
	for idx, id := range selectedIDs {
		path, ok := pathByID[id]
		if !ok {
			continue
		}

		latencyMs := path.LatencyMs
		if latencyMs <= 0 {
			latencyMs = 80 + int64(idx*25)
		}

		jitterMs := latencyMs / 3
		if jitterMs < 8 {
			jitterMs = 8
		}
		if jitterMs > maxJitterMs {
			jitterMs = maxJitterMs
		}

		lossPct := path.LossPct
		if lossPct < 0 {
			lossPct = 0
		}
		if lossPct > 50 {
			lossPct = 50
		}

		if degradedPrimary && idx == 0 {
			latencyMs += tunAggregationMaxInt64(90, maxJitterMs/2)
			jitterMs += 40
			if jitterMs > maxJitterMs {
				jitterMs = maxJitterMs
			}
			if lossPct < 18 {
				lossPct = 18
			}
		}

		profiles = append(profiles, tunAggregationPathProfile{
			ID:        id,
			LatencyMs: latencyMs,
			JitterMs:  jitterMs,
			LossPct:   lossPct,
		})
	}

	return profiles
}

func simulateTunAggregationSession(policy TunAggregationSchedulerPolicy, profiles []tunAggregationPathProfile, now time.Time, packetCount, payloadBytes int) tunAggregationSimulationResult {
	if len(profiles) == 0 || packetCount <= 0 || payloadBytes <= 0 {
		return tunAggregationSimulationResult{}
	}

	weightedOrder := tunAggregationWeightedOrder(profiles)
	envelopes := make([]tunAggregationRelayEnvelope, 0, packetCount*2)

	for sequence := 0; sequence < packetCount; sequence++ {
		sentAt := now.Add(time.Duration(sequence) * tunAggregationBenchmarkSendInterval)
		pathIndexes := tunAggregationTransmissionPaths(policy, weightedOrder, len(profiles), sequence)
		for _, pathIndex := range pathIndexes {
			profile := profiles[pathIndex]
			if tunAggregationTransmissionLost(sequence, pathIndex, profile.LossPct) {
				continue
			}
			envelopes = append(envelopes, tunAggregationRelayEnvelope{
				SessionID:    "relay-preview",
				Sequence:     sequence,
				PathID:       profile.ID,
				SentAt:       sentAt,
				ArrivedAt:    sentAt.Add(tunAggregationTransmissionLatency(sequence, pathIndex, profile)),
				PayloadBytes: payloadBytes,
			})
		}
	}

	sort.Slice(envelopes, func(i, j int) bool {
		if envelopes[i].ArrivedAt.Equal(envelopes[j].ArrivedAt) {
			if envelopes[i].Sequence == envelopes[j].Sequence {
				return envelopes[i].PathID < envelopes[j].PathID
			}
			return envelopes[i].Sequence < envelopes[j].Sequence
		}
		return envelopes[i].ArrivedAt.Before(envelopes[j].ArrivedAt)
	})

	assembler := newTunAggregationRelayAssembler()
	for _, envelope := range envelopes {
		assembler.ingest(envelope)
	}

	return tunAggregationSummarizeSimulation(assembler, now, packetCount)
}

func newTunAggregationRelayAssembler() *tunAggregationRelayAssembler {
	return &tunAggregationRelayAssembler{
		buffered: make(map[int]tunAggregationRelayEnvelope),
	}
}

func (a *tunAggregationRelayAssembler) ingest(envelope tunAggregationRelayEnvelope) {
	if envelope.Sequence < a.nextSequence {
		a.duplicateDrops++
		return
	}
	if _, exists := a.buffered[envelope.Sequence]; exists {
		a.duplicateDrops++
		return
	}

	if envelope.Sequence == a.nextSequence {
		a.deliver(envelope)
		a.flush()
		return
	}

	a.buffered[envelope.Sequence] = envelope
	a.reorderedPackets++
	if len(a.buffered) > a.maxBufferDepth {
		a.maxBufferDepth = len(a.buffered)
	}
}

func (a *tunAggregationRelayAssembler) deliver(envelope tunAggregationRelayEnvelope) {
	a.deliveries = append(a.deliveries, envelope)
	a.nextSequence = envelope.Sequence + 1
}

func (a *tunAggregationRelayAssembler) flush() {
	for {
		envelope, ok := a.buffered[a.nextSequence]
		if !ok {
			return
		}
		delete(a.buffered, a.nextSequence)
		a.deliver(envelope)
	}
}

func tunAggregationSummarizeSimulation(assembler *tunAggregationRelayAssembler, startedAt time.Time, packetCount int) tunAggregationSimulationResult {
	if assembler == nil {
		return tunAggregationSimulationResult{}
	}

	result := tunAggregationSimulationResult{
		PacketCount:           packetCount,
		DeliveredPacketCount:  len(assembler.deliveries),
		DuplicateDrops:        assembler.duplicateDrops,
		ReorderedPackets:      assembler.reorderedPackets,
		MaxReorderBufferDepth: assembler.maxBufferDepth,
		LossPct:               100,
	}

	if packetCount > 0 {
		result.LossPct = float64(packetCount-len(assembler.deliveries)) * 100 / float64(packetCount)
	}

	if len(assembler.deliveries) == 0 {
		return result
	}

	firstDelivery := assembler.deliveries[0]
	lastDelivery := assembler.deliveries[len(assembler.deliveries)-1]

	result.StartupLatencyMs = firstDelivery.ArrivedAt.Sub(startedAt).Milliseconds()
	for i, delivery := range assembler.deliveries {
		result.DeliveredBytes += delivery.PayloadBytes
		if i > 0 && delivery.ArrivedAt.Sub(assembler.deliveries[i-1].ArrivedAt) > tunAggregationBenchmarkStallThreshold {
			result.StallCount++
		}
	}

	duration := lastDelivery.ArrivedAt.Sub(startedAt)
	if duration <= 0 {
		duration = tunAggregationBenchmarkSendInterval
	}
	result.GoodputKbps = (float64(result.DeliveredBytes) * 8 / duration.Seconds()) / 1000
	result.StabilityPct = tunAggregationDeliveryStabilityPct(assembler.deliveries, startedAt)
	return result
}

func tunAggregationDeliveryStabilityPct(deliveries []tunAggregationRelayEnvelope, startedAt time.Time) float64 {
	if len(deliveries) == 0 {
		return 0
	}

	window := 5 * tunAggregationBenchmarkSendInterval
	lastAt := deliveries[len(deliveries)-1].ArrivedAt
	if lastAt.Before(startedAt) {
		return 0
	}

	windowCount := int(lastAt.Sub(startedAt)/window) + 1
	if windowCount < 1 {
		windowCount = 1
	}

	buckets := make([]float64, windowCount)
	for _, delivery := range deliveries {
		index := int(delivery.ArrivedAt.Sub(startedAt) / window)
		if index < 0 {
			index = 0
		}
		if index >= len(buckets) {
			index = len(buckets) - 1
		}
		buckets[index] += float64(delivery.PayloadBytes)
	}

	var total float64
	for _, bucket := range buckets {
		total += bucket
	}
	if total <= 0 {
		return 0
	}

	mean := total / float64(len(buckets))
	if mean <= 0 {
		return 0
	}

	var variance float64
	for _, bucket := range buckets {
		diff := bucket - mean
		variance += diff * diff
	}
	variance /= float64(len(buckets))
	cv := math.Sqrt(variance) / mean
	stability := 100 / (1 + cv)
	return tunAggregationRoundFloat(stability, 1)
}

func tunAggregationWeightedOrder(profiles []tunAggregationPathProfile) []int {
	if len(profiles) == 0 {
		return nil
	}

	maxLatency := int64(1)
	for _, profile := range profiles {
		if profile.LatencyMs > maxLatency {
			maxLatency = profile.LatencyMs
		}
	}

	order := make([]int, 0, len(profiles)*2)
	for index, profile := range profiles {
		latency := profile.LatencyMs
		if latency < 1 {
			latency = 1
		}
		weight := int(math.Round(float64(maxLatency) / float64(latency)))
		if profile.LossPct < 1 {
			weight++
		}
		if weight < 1 {
			weight = 1
		}
		if weight > 4 {
			weight = 4
		}
		for repeat := 0; repeat < weight; repeat++ {
			order = append(order, index)
		}
	}

	if len(order) == 0 {
		return []int{0}
	}
	return order
}

func tunAggregationTransmissionPaths(policy TunAggregationSchedulerPolicy, weightedOrder []int, pathCount, sequence int) []int {
	if pathCount <= 0 {
		return nil
	}

	switch policy {
	case TunAggregationSchedulerPolicyRedundant2:
		if pathCount == 1 {
			return []int{0}
		}
		return []int{0, 1}
	case TunAggregationSchedulerPolicyWeightedSplit:
		if len(weightedOrder) == 0 {
			return []int{0}
		}
		return []int{weightedOrder[sequence%len(weightedOrder)]}
	default:
		return []int{0}
	}
}

func tunAggregationTransmissionLost(sequence, pathIndex int, lossPct float64) bool {
	if lossPct <= 0 || sequence < 3 {
		return false
	}

	threshold := int(math.Round(lossPct))
	if threshold <= 0 {
		return false
	}
	if threshold > 95 {
		threshold = 95
	}

	signal := ((sequence + 1) * 17) + ((pathIndex + 1) * 23)
	return signal%100 < threshold
}

func tunAggregationTransmissionLatency(sequence, pathIndex int, profile tunAggregationPathProfile) time.Duration {
	jitterSpread := profile.JitterMs
	if jitterSpread < 1 {
		jitterSpread = 1
	}

	jitterSignal := ((sequence + 1) * (pathIndex + 3) * 11) % int((jitterSpread*2)+1)
	jitterOffset := int64(jitterSignal) - jitterSpread
	latencyMs := profile.LatencyMs + jitterOffset
	if latencyMs < 1 {
		latencyMs = 1
	}

	return time.Duration(latencyMs) * time.Millisecond
}

func formatTunAggregationRelayDiagnostic(status *TunAggregationRelayStatus) string {
	if status == nil {
		return ""
	}

	return fmt.Sprintf(
		"Aggregation relay prototype: sessions=%d delivered=%d/%d duplicateDrops=%d reordered=%d maxBuffer=%d endpoint=%s",
		status.SessionCount,
		status.DeliveredPacketCount,
		status.PacketCount,
		status.DuplicateDrops,
		status.ReorderedPackets,
		status.MaxReorderBufferDepth,
		status.Endpoint,
	)
}

func formatTunAggregationBenchmarkDiagnostic(status *TunAggregationBenchmarkStatus) string {
	if status == nil {
		return ""
	}

	for _, scenario := range status.Scenarios {
		if scenario.Name == string(TunAggregationBenchmarkScenarioDegradedPrimary) {
			return fmt.Sprintf(
				"Aggregation benchmark [%s]: startupGain=%dms stallReduction=%d goodputGain=%.1fkbps lossReduction=%.1fpp stabilityGain=%.1fpp",
				scenario.Name,
				scenario.StartupLatencyGainMs,
				scenario.StallReduction,
				scenario.GoodputGainKbps,
				scenario.LossReductionPct,
				scenario.StabilityGainPct,
			)
		}
	}

	return fmt.Sprintf("Aggregation benchmark: scenarios=%d packets=%d payloadBytes=%d", len(status.Scenarios), status.PacketCount, status.PayloadBytes)
}

func tunAggregationRoundFloat(value float64, digits int) float64 {
	factor := math.Pow(10, float64(digits))
	return math.Round(value*factor) / factor
}

func tunAggregationMaxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

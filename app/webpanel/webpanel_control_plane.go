package webpanel

import (
	"fmt"
	"strings"
	"time"
)

const (
	tunEligibleProbeFreshness = 10 * time.Minute
	tunStableModeDiagnostic   = "Strict transparent mode captures all non-bypassed IPv4 traffic into the TUN route table, disables IPv6 while enabled, and normalizes remote DNS to encrypted DoH resolvers. Only configured direct routes, local routes, helper/root traffic, and upstream proxy reachability bypass the tunnel."
)

type tunEligiblePoolSummary struct {
	ActiveNodes              int
	EligibleNodes            int
	MinActiveNodes           int
	ExcludedProtocol         int
	ExcludedConsecutiveFails int
	ExcludedMissingDelay     int
	ExcludedUncheckedOrStale int
}

func (wp *WebPanel) tunStatusSnapshot() *TunStatus {
	if wp.tunManager == nil {
		return &TunStatus{
			Status:    "unavailable",
			Available: false,
			Message:   "TUN manager is not configured",
		}
	}
	status := wp.decorateTunStatus(wp.tunManager.Status())
	return wp.appendTunStableDiagnostics(status, false)
}

func (wp *WebPanel) tunStatusSnapshotWithoutEgressProbe() *TunStatus {
	if wp.tunManager == nil {
		return &TunStatus{
			Status:    "unavailable",
			Available: false,
			Message:   "TUN manager is not configured",
		}
	}
	status := wp.decorateTunStatus(wp.tunManager.StatusWithoutEgressProbe())
	return wp.appendTunStableDiagnostics(status, false)
}

func (wp *WebPanel) startTransparentMode() *TunStatus {
	if wp.tunManager == nil {
		return &TunStatus{
			Status:    "unavailable",
			Available: false,
			Message:   "TUN manager is not configured",
		}
	}

	eligibleNodes, eligibleSummary := wp.eligibleTransparentNodes()
	if eligibleSummary.EligibleNodes < eligibleSummary.MinActiveNodes {
		if wp.controlPlane != nil {
			wp.controlPlane.Transition(
				MachineStateClean,
				MachineReasonEnableBlockedMinActiveNotMet,
				EventActorOperator,
				fmt.Sprintf("eligible active nodes %d below minimum %d", eligibleSummary.EligibleNodes, eligibleSummary.MinActiveNodes),
			)
		}
		status := wp.decorateTunStatus(wp.tunManager.Status())
		status.Status = "blocked"
		status.Message = "Transparent mode is blocked until the stable eligible pool reaches the minimum size"
		return wp.appendTunStableDiagnostics(status, true)
	}

	status := wp.tunManager.Start(eligibleNodes)
	if status.Status == "error" || status.Status == "unavailable" {
		if wp.controlPlane != nil {
			wp.controlPlane.Transition(MachineStateClean, MachineReasonTunStartFailed, EventActorOperator, status.Message)
		}
		return wp.appendTunStableDiagnostics(wp.decorateTunStatus(status), false)
	}
	if status.Status == "blocked" {
		return wp.appendTunStableDiagnostics(wp.decorateTunStatus(status), false)
	}

	if wp.controlPlane != nil {
		wp.controlPlane.Transition(MachineStateProxied, MachineReasonOperatorEnabled, EventActorOperator, "transparent mode enabled from the node pool workspace")
	}
	return wp.appendTunStableDiagnostics(wp.decorateTunStatus(status), false)
}

func (wp *WebPanel) restoreClean(requestReason, finalReason MachineStateReason, actor EventActor, details string) *TunStatus {
	if wp.tunManager == nil {
		return &TunStatus{
			Status:    "unavailable",
			Available: false,
			Message:   "TUN manager is not configured",
		}
	}

	if wp.controlPlane != nil {
		wp.controlPlane.Transition(MachineStateRecovering, requestReason, actor, details)
	}

	status := wp.tunManager.RestoreClean()
	if status.Status == "error" || status.Status == "unavailable" {
		if wp.controlPlane != nil {
			wp.controlPlane.Transition(MachineStateDegraded, MachineReasonFallbackFailed, actor, status.Message)
		}
		return wp.decorateTunStatus(status)
	}

	if wp.controlPlane != nil {
		wp.controlPlane.Transition(MachineStateClean, finalReason, actor, details)
	}
	return wp.decorateTunStatus(status)
}

func (wp *WebPanel) ensureCleanStartupState() {
	if wp.tunManager == nil || wp.controlPlane == nil {
		return
	}

	status := wp.tunManager.Status()
	switch status.Status {
	case "error", "unavailable":
		wp.controlPlane.Transition(MachineStateDegraded, MachineReasonStartupStatusUnavailable, EventActorSystem, status.Message)
		return
	}

	if status.Running {
		wp.controlPlane.Transition(MachineStateRecovering, MachineReasonStartupDefaultClean, EventActorSystem, "startup must restore a clean machine state")
		cleanStatus := wp.tunManager.RestoreClean()
		if cleanStatus.Status == "error" || cleanStatus.Status == "unavailable" {
			wp.controlPlane.Transition(MachineStateDegraded, MachineReasonStartupCleanupFailed, EventActorSystem, cleanStatus.Message)
			return
		}
	}

	wp.controlPlane.Transition(MachineStateClean, MachineReasonStartupDefaultClean, EventActorSystem, "startup defaults to a clean network state")
}

func (wp *WebPanel) handlePoolHealthChange(summary PoolHealthSummary) {
	if wp.tunManager == nil || wp.controlPlane == nil {
		return
	}

	current := wp.controlPlane.Snapshot()
	if current.MachineState != MachineStateProxied {
		return
	}

	_, eligibleSummary := wp.eligibleTransparentNodes()
	if eligibleSummary.EligibleNodes >= eligibleSummary.MinActiveNodes {
		if summary.Healthy {
			return
		}
		return
	}

	wp.restoreClean(
		MachineReasonPoolBelowMinActiveNodes,
		MachineReasonAutomaticFallbackMinActive,
		EventActorSystem,
		fmt.Sprintf("eligible active nodes %d fell below minimum %d", eligibleSummary.EligibleNodes, eligibleSummary.MinActiveNodes),
	)
}

func (wp *WebPanel) activePoolNodes() []NodeRecord {
	if wp.subManager == nil {
		return nil
	}
	return wp.subManager.ListNodesByStatuses(NodeStatusActive)
}

func (wp *WebPanel) currentPoolSummary() NodePoolSummary {
	if wp.subManager == nil {
		return NodePoolSummary{}
	}
	return wp.subManager.GetPoolSummary()
}

func (wp *WebPanel) decorateTunStatus(status *TunStatus) *TunStatus {
	if status == nil {
		status = &TunStatus{
			Status:    "unavailable",
			Available: false,
			Message:   "missing TUN status",
		}
	}
	if wp.controlPlane == nil {
		return status
	}

	snapshot := wp.controlPlane.Snapshot()
	status.MachineState = snapshot.MachineState
	status.LastStateReason = snapshot.LastStateReason
	status.LastStateChangedAt = &snapshot.LastStateChangedAt
	status.RecentMachineEvents = append([]MachineEvent(nil), snapshot.RecentMachineEvents...)
	return status
}

func (wp *WebPanel) eligibleTransparentNodes() ([]NodeRecord, tunEligiblePoolSummary) {
	summary := tunEligiblePoolSummary{}
	if wp.subManager == nil {
		return nil, summary
	}

	pool := wp.currentPoolSummary()
	summary.MinActiveNodes = pool.MinActiveNodes

	activeNodes := wp.activePoolNodes()
	summary.ActiveNodes = len(activeNodes)

	now := time.Now()
	eligible := make([]NodeRecord, 0, len(activeNodes))
	for _, node := range activeNodes {
		switch {
		case strings.EqualFold(strings.TrimSpace(node.Protocol), "hysteria2"):
			summary.ExcludedProtocol++
		case node.ConsecutiveFails > 0:
			summary.ExcludedConsecutiveFails++
		case node.AvgDelayMs <= 0:
			summary.ExcludedMissingDelay++
		case node.LastCheckedAt == nil || now.Sub(*node.LastCheckedAt) > tunEligibleProbeFreshness:
			summary.ExcludedUncheckedOrStale++
		default:
			eligible = append(eligible, node)
		}
	}
	summary.EligibleNodes = len(eligible)
	return eligible, summary
}

func (wp *WebPanel) appendTunStableDiagnostics(status *TunStatus, blocked bool) *TunStatus {
	if status == nil {
		return nil
	}

	wp.appendTunRoutingDiagnostics(status)
	wp.appendTunAggregationPrototype(status)

	_, summary := wp.eligibleTransparentNodes()
	appendUniqueTunDiagnostic(status, tunStableModeDiagnostic)
	appendUniqueTunDiagnostic(
		status,
		fmt.Sprintf(
			"Transparent-mode eligible nodes: %d / active %d / minimum required %d.",
			summary.EligibleNodes,
			summary.ActiveNodes,
			summary.MinActiveNodes,
		),
	)

	if summary.ExcludedProtocol > 0 || summary.ExcludedConsecutiveFails > 0 || summary.ExcludedMissingDelay > 0 || summary.ExcludedUncheckedOrStale > 0 {
		appendUniqueTunDiagnostic(
			status,
			fmt.Sprintf(
				"Excluded active nodes: hysteria2=%d, consecutive-fails=%d, missing-delay=%d, stale-or-unchecked=%d.",
				summary.ExcludedProtocol,
				summary.ExcludedConsecutiveFails,
				summary.ExcludedMissingDelay,
				summary.ExcludedUncheckedOrStale,
			),
		)
	}

	if blocked {
		appendUniqueTunDiagnostic(status, "Transparent mode only starts when the stable eligible pool meets the minimum size.")
	}

	return status
}

func (wp *WebPanel) appendTunAggregationPrototype(status *TunStatus) {
	if status == nil || wp.tunManager == nil || status.Aggregation == nil {
		return
	}

	settings, err := wp.tunManager.SettingsSnapshot()
	if err != nil {
		appendUniqueTunDiagnostic(status, "Unable to build aggregation prototype diagnostics: "+err.Error())
		return
	}

	attachTunAggregationPrototype(status.Aggregation, settings, wp.activePoolNodes(), time.Now())
	attachTunAggregationRelayDiagnostics(status.Aggregation, settings, time.Now())
	appendUniqueTunDiagnostic(status, formatTunAggregationPrototypeDiagnostic(status.Aggregation.Prototype))
	appendUniqueTunDiagnostic(status, formatTunAggregationRelayDiagnostic(status.Aggregation.Relay))
	appendUniqueTunDiagnostic(status, formatTunAggregationBenchmarkDiagnostic(status.Aggregation.Benchmark))
}

func (wp *WebPanel) appendTunRoutingDiagnostics(status *TunStatus) {
	if status == nil || wp.tunManager == nil {
		return
	}

	settings, err := wp.tunManager.SettingsSnapshot()
	if err != nil {
		appendUniqueTunDiagnostic(status, "Unable to build DNS/routing diagnostics: "+err.Error())
		return
	}

	diagnostics := buildTunRoutingDiagnostics(settings)
	status.RoutingDiagnostics = diagnostics
	for _, diagnostic := range diagnostics {
		appendUniqueTunDiagnostic(status, formatTunRoutingDiagnostic(diagnostic))
	}
}

func buildTunRoutingDiagnostics(settings *TunFeatureSettings) []TunRoutingDiagnostic {
	diagnostics := make([]TunRoutingDiagnostic, 0, 2)

	if len(settings.ProtectDomains) > 0 {
		diagnostics = append(diagnostics, TunRoutingDiagnostic{
			Category: "protected_direct_domains",
			DNSPath:  "dns-direct-local",
			Resolver: "localhost",
			Route:    "direct",
			Reason:   "Domains matched by webpanel.tun.protectDomains resolve through local system DNS and are routed direct.",
			Domains:  append([]string(nil), settings.ProtectDomains...),
		})
	}

	diagnostics = append(diagnostics, TunRoutingDiagnostic{
		Category: "default_proxy_domains",
		DNSPath:  "dns-remote",
		Resolver: strings.Join(settings.RemoteDNS, ", "),
		Route:    "proxy(node-pool-active)",
		Reason:   "Domains not matched by protected direct rules are resolved by remote DNS and routed through the active node pool.",
	})

	return diagnostics
}

func formatTunRoutingDiagnostic(diagnostic TunRoutingDiagnostic) string {
	domainHint := ""
	if len(diagnostic.Domains) > 0 {
		domainHint = " domains=" + strings.Join(diagnostic.Domains, ", ")
	}
	return fmt.Sprintf(
		"DNS/routing decision [%s]: dns=%s resolver=%s route=%s reason=%s%s",
		diagnostic.Category,
		diagnostic.DNSPath,
		diagnostic.Resolver,
		diagnostic.Route,
		diagnostic.Reason,
		domainHint,
	)
}

func appendUniqueTunDiagnostic(status *TunStatus, diagnostic string) {
	if status == nil {
		return
	}
	trimmed := strings.TrimSpace(diagnostic)
	if trimmed == "" {
		return
	}
	for _, existing := range status.Diagnostics {
		if existing == trimmed {
			return
		}
	}
	status.Diagnostics = append(status.Diagnostics, trimmed)
}

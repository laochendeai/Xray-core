package webpanel

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"
)

type ReadinessSeverity string

const (
	ReadinessSeverityOK       ReadinessSeverity = "ok"
	ReadinessSeverityWarning  ReadinessSeverity = "warning"
	ReadinessSeverityBlocking ReadinessSeverity = "blocking"
)

type ReadinessArea string

const (
	ReadinessAreaConfig        ReadinessArea = "config"
	ReadinessAreaSubscriptions ReadinessArea = "subscriptions"
	ReadinessAreaNodePool      ReadinessArea = "node_pool"
	ReadinessAreaTun           ReadinessArea = "tun"
	ReadinessAreaRuntime       ReadinessArea = "runtime"
	ReadinessAreaUpdates       ReadinessArea = "updates"
)

type ReadinessCheck struct {
	Key         string                 `json:"key"`
	Area        ReadinessArea          `json:"area"`
	Severity    ReadinessSeverity      `json:"severity"`
	ActionRoute string                 `json:"actionRoute,omitempty"`
	Facts       map[string]interface{} `json:"facts,omitempty"`
}

type ReadinessResponse struct {
	Healthy       bool             `json:"healthy"`
	BlockingCount int              `json:"blockingCount"`
	WarningCount  int              `json:"warningCount"`
	UpdatedAt     string           `json:"updatedAt"`
	Checks        []ReadinessCheck `json:"checks"`
}

func (wp *WebPanel) readinessSnapshot(ctx context.Context) ReadinessResponse {
	checks := make([]ReadinessCheck, 0, 7)
	configMap, configPathStatus, configPathFacts := wp.readinessLoadConfig()
	checks = append(checks, ReadinessCheck{
		Key:         "config_path",
		Area:        ReadinessAreaConfig,
		Severity:    configPathStatus,
		ActionRoute: "/config",
		Facts:       configPathFacts,
	})
	checks = append(checks, wp.readinessConfigSectionsCheck(configMap))
	checks = append(checks, wp.readinessSubscriptionsCheck())
	checks = append(checks, wp.readinessProbingCheck())
	checks = append(checks, wp.readinessNodePoolCheck())
	checks = append(checks, wp.readinessTunCheck())
	checks = append(checks, wp.readinessUpdatesCheck(ctx))

	response := ReadinessResponse{
		Healthy:   true,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Checks:    checks,
	}
	for _, check := range checks {
		switch check.Severity {
		case ReadinessSeverityBlocking:
			response.BlockingCount++
			response.Healthy = false
		case ReadinessSeverityWarning:
			response.WarningCount++
		}
	}
	return response
}

func (wp *WebPanel) readinessLoadConfig() (map[string]interface{}, ReadinessSeverity, map[string]interface{}) {
	configPath := strings.TrimSpace(wp.config.ConfigPath)
	facts := map[string]interface{}{
		"path": configPath,
	}
	if configPath == "" {
		facts["status"] = "missing"
		return nil, ReadinessSeverityBlocking, facts
	}

	info, err := os.Stat(configPath)
	if err != nil {
		facts["status"] = "not_found"
		facts["error"] = err.Error()
		return nil, ReadinessSeverityBlocking, facts
	}
	facts["sizeBytes"] = info.Size()

	cfm := NewConfigFileManager(configPath)
	raw, err := cfm.ReadConfig()
	if err != nil {
		facts["status"] = "invalid"
		facts["error"] = err.Error()
		return nil, ReadinessSeverityBlocking, facts
	}

	var configMap map[string]interface{}
	if err := json.Unmarshal(raw, &configMap); err != nil {
		facts["status"] = "invalid"
		facts["error"] = err.Error()
		return nil, ReadinessSeverityBlocking, facts
	}

	facts["status"] = "ok"
	return configMap, ReadinessSeverityOK, facts
}

func (wp *WebPanel) readinessConfigSectionsCheck(config map[string]interface{}) ReadinessCheck {
	facts := map[string]interface{}{}
	if config == nil {
		facts["status"] = "unavailable"
		return ReadinessCheck{
			Key:         "config_sections",
			Area:        ReadinessAreaConfig,
			Severity:    ReadinessSeverityBlocking,
			ActionRoute: "/config",
			Facts:       facts,
		}
	}

	missingSections := make([]string, 0, 3)
	missingStatsFlags := make([]string, 0, 4)

	if _, ok := config["api"].(map[string]interface{}); !ok {
		missingSections = append(missingSections, "api")
	}
	if _, ok := config["stats"].(map[string]interface{}); !ok {
		missingSections = append(missingSections, "stats")
	}

	policy, _ := config["policy"].(map[string]interface{})
	systemPolicy, _ := policy["system"].(map[string]interface{})
	for _, key := range []string{
		"statsInboundUplink",
		"statsInboundDownlink",
		"statsOutboundUplink",
		"statsOutboundDownlink",
	} {
		value, ok := systemPolicy[key].(bool)
		if !ok || !value {
			missingStatsFlags = append(missingStatsFlags, key)
		}
	}

	services := make([]string, 0)
	apiSection, _ := config["api"].(map[string]interface{})
	if rawServices, ok := apiSection["services"].([]interface{}); ok {
		for _, service := range rawServices {
			if value, ok := service.(string); ok && strings.TrimSpace(value) != "" {
				services = append(services, strings.TrimSpace(value))
			}
		}
	}
	missingServices := missingRequiredServices(services)

	facts["status"] = "ok"
	if len(missingSections) > 0 {
		facts["status"] = "missing_sections"
		facts["missingSections"] = missingSections
	}
	if len(missingStatsFlags) > 0 {
		facts["missingStatsFlags"] = missingStatsFlags
		if facts["status"] == "ok" {
			facts["status"] = "missing_stats_flags"
		}
	}
	if len(missingServices) > 0 {
		facts["missingServices"] = missingServices
		if facts["status"] == "ok" {
			facts["status"] = "missing_services"
		}
	}

	severity := ReadinessSeverityOK
	if len(missingSections) > 0 {
		severity = ReadinessSeverityBlocking
	} else if len(missingStatsFlags) > 0 || len(missingServices) > 0 {
		severity = ReadinessSeverityWarning
	}

	return ReadinessCheck{
		Key:         "config_sections",
		Area:        ReadinessAreaConfig,
		Severity:    severity,
		ActionRoute: "/config",
		Facts:       facts,
	}
}

func missingRequiredServices(services []string) []string {
	required := []string{
		"HandlerService",
		"StatsService",
		"LoggerService",
		"RoutingService",
		"ObservatoryService",
	}
	if len(services) == 0 {
		return append([]string{}, required...)
	}

	existing := make(map[string]struct{}, len(services))
	for _, service := range services {
		existing[strings.TrimSpace(service)] = struct{}{}
	}

	missing := make([]string, 0, len(required))
	for _, service := range required {
		if _, ok := existing[service]; !ok {
			missing = append(missing, service)
		}
	}
	return missing
}

func (wp *WebPanel) readinessSubscriptionsCheck() ReadinessCheck {
	if wp.subManager == nil {
		return ReadinessCheck{
			Key:         "subscriptions",
			Area:        ReadinessAreaSubscriptions,
			Severity:    ReadinessSeverityBlocking,
			ActionRoute: "/subscriptions",
			Facts: map[string]interface{}{
				"status": "unavailable",
			},
		}
	}

	snapshot := wp.subManager.ReadinessSnapshot()
	facts := map[string]interface{}{
		"subscriptionCount": snapshot.SubscriptionCount,
		"nodeCount":         snapshot.NodeCount,
		"activeCount":       snapshot.PoolSummary.ActiveCount,
		"candidateCount":    snapshot.PoolSummary.CandidateCount,
		"stagingCount":      snapshot.PoolSummary.StagingCount,
		"quarantineCount":   snapshot.PoolSummary.QuarantineCount,
		"removedCount":      snapshot.PoolSummary.RemovedCount,
	}

	severity := ReadinessSeverityOK
	status := "ok"
	switch {
	case snapshot.SubscriptionCount == 0:
		severity = ReadinessSeverityWarning
		status = "empty"
	case snapshot.NodeCount == 0:
		severity = ReadinessSeverityWarning
		status = "no_nodes"
	}
	facts["status"] = status

	return ReadinessCheck{
		Key:         "subscriptions",
		Area:        ReadinessAreaSubscriptions,
		Severity:    severity,
		ActionRoute: "/subscriptions",
		Facts:       facts,
	}
}

func (wp *WebPanel) readinessProbingCheck() ReadinessCheck {
	if wp.subManager == nil {
		return ReadinessCheck{
			Key:         "probing",
			Area:        ReadinessAreaRuntime,
			Severity:    ReadinessSeverityBlocking,
			ActionRoute: "/node-pool",
			Facts: map[string]interface{}{
				"status": "unavailable",
			},
		}
	}

	snapshot := wp.subManager.ReadinessSnapshot()
	facts := map[string]interface{}{
		"started":             snapshot.Started,
		"dispatcherAvailable": snapshot.DispatcherAvailable,
		"probeUrl":            snapshot.ProbeURL,
		"probeIntervalSec":    snapshot.ProbeIntervalSec,
		"tagCount":            snapshot.Prober.TagCount,
		"running":             snapshot.Prober.Running,
	}

	severity := ReadinessSeverityOK
	status := "ok"
	switch {
	case !snapshot.Started:
		severity = ReadinessSeverityWarning
		status = "not_started"
	case !snapshot.DispatcherAvailable:
		severity = ReadinessSeverityWarning
		status = "dispatcher_unavailable"
	case snapshot.Prober.TagCount == 0:
		severity = ReadinessSeverityWarning
		status = "idle"
	case !snapshot.Prober.Running:
		severity = ReadinessSeverityWarning
		status = "stopped"
	default:
		status = "running"
	}
	facts["status"] = status

	return ReadinessCheck{
		Key:         "probing",
		Area:        ReadinessAreaRuntime,
		Severity:    severity,
		ActionRoute: "/node-pool",
		Facts:       facts,
	}
}

func (wp *WebPanel) readinessNodePoolCheck() ReadinessCheck {
	if wp.subManager == nil {
		return ReadinessCheck{
			Key:         "node_pool",
			Area:        ReadinessAreaNodePool,
			Severity:    ReadinessSeverityBlocking,
			ActionRoute: "/node-pool",
			Facts: map[string]interface{}{
				"status": "unavailable",
			},
		}
	}

	snapshot := wp.subManager.ReadinessSnapshot()
	summary := snapshot.PoolSummary
	facts := map[string]interface{}{
		"status":          "ok",
		"healthy":         summary.Healthy,
		"activeCount":     summary.ActiveCount,
		"candidateCount":  summary.CandidateCount,
		"stagingCount":    summary.StagingCount,
		"quarantineCount": summary.QuarantineCount,
		"removedCount":    summary.RemovedCount,
		"minActiveNodes":  summary.MinActiveNodes,
	}

	severity := ReadinessSeverityOK
	switch {
	case snapshot.NodeCount == 0:
		severity = ReadinessSeverityWarning
		facts["status"] = "empty"
	case !summary.Healthy:
		severity = ReadinessSeverityWarning
		facts["status"] = "below_minimum"
	}

	return ReadinessCheck{
		Key:         "node_pool",
		Area:        ReadinessAreaNodePool,
		Severity:    severity,
		ActionRoute: "/node-pool",
		Facts:       facts,
	}
}

func (wp *WebPanel) readinessTunCheck() ReadinessCheck {
	if wp.tunManager == nil {
		return ReadinessCheck{
			Key:         "tun",
			Area:        ReadinessAreaTun,
			Severity:    ReadinessSeverityWarning,
			ActionRoute: "/settings",
			Facts: map[string]interface{}{
				"status": "unavailable",
			},
		}
	}

	status := wp.tunStatusSnapshotWithoutEgressProbe()
	facts := map[string]interface{}{
		"status":                      status.Status,
		"running":                     status.Running,
		"available":                   status.Available,
		"message":                     status.Message,
		"helperExists":                status.HelperExists,
		"elevationReady":              status.ElevationReady,
		"privilegeInstallRecommended": status.PrivilegeInstallRecommended,
		"machineState":                status.MachineState,
	}

	severity := ReadinessSeverityOK
	switch {
	case status.MachineState == MachineStateDegraded || status.Status == "error":
		severity = ReadinessSeverityBlocking
	case status.Status == "blocked" || status.Status == "unavailable" || status.PrivilegeInstallRecommended || !status.HelperExists:
		severity = ReadinessSeverityWarning
	}

	return ReadinessCheck{
		Key:         "tun",
		Area:        ReadinessAreaTun,
		Severity:    severity,
		ActionRoute: "/settings",
		Facts:       facts,
	}
}

func (wp *WebPanel) readinessUpdatesCheck(ctx context.Context) ReadinessCheck {
	if wp.releaseChecker == nil {
		return ReadinessCheck{
			Key:         "updates",
			Area:        ReadinessAreaUpdates,
			Severity:    ReadinessSeverityWarning,
			ActionRoute: "/dashboard",
			Facts: map[string]interface{}{
				"status": "unavailable",
			},
		}
	}

	checkCtx := ctx
	if checkCtx == nil {
		checkCtx = context.Background()
	}
	checkCtx, cancel := context.WithTimeout(checkCtx, 2*time.Second)
	defer cancel()

	status := wp.releaseChecker.Check(checkCtx, false)
	facts := map[string]interface{}{
		"status":            status.Status,
		"source":            status.Source,
		"currentVersion":    status.CurrentVersion,
		"latestVersion":     status.LatestVersion,
		"updateAvailable":   status.UpdateAvailable,
		"message":           status.Message,
		"latestPublishedAt": status.LatestPublishedAt,
	}

	severity := ReadinessSeverityOK
	if status.Status == "error" || status.Status == "stale" || status.UpdateAvailable {
		severity = ReadinessSeverityWarning
	}

	return ReadinessCheck{
		Key:         "updates",
		Area:        ReadinessAreaUpdates,
		Severity:    severity,
		ActionRoute: "/dashboard",
		Facts:       facts,
	}
}

func readinessCheckByKey(checks []ReadinessCheck, key string) (ReadinessCheck, bool) {
	for _, check := range checks {
		if check.Key == key {
			return check, true
		}
	}
	return ReadinessCheck{}, false
}

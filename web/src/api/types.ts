export interface SysStats {
  numGoroutine: number;
  numGC: number;
  alloc: number;
  totalAlloc: number;
  sys: number;
  mallocs: number;
  frees: number;
  liveObjects: number;
  pauseTotalNs: number;
  uptime: number;
}

export interface UpdateStatusResponse {
  currentVersion: string;
  latestVersion?: string;
  releaseTitle?: string;
  latestReleaseUrl?: string;
  latestPublishedAt?: string;
  checkedAt?: string;
  source: string;
  status: "ok" | "stale" | "error";
  message?: string;
  updateAvailable: boolean;
  stale: boolean;
}

export type ReadinessSeverity = "ok" | "warning" | "blocking";
export type ReadinessArea =
  | "config"
  | "subscriptions"
  | "node_pool"
  | "tun"
  | "runtime"
  | "updates";
export type ReadinessFacts = Record<string, unknown>;

export interface ReadinessCheck {
  key: string;
  area: ReadinessArea;
  severity: ReadinessSeverity;
  actionRoute?: string;
  facts?: ReadinessFacts;
}

export interface ReadinessResponse {
  healthy: boolean;
  blockingCount: number;
  warningCount: number;
  updatedAt: string;
  checks: ReadinessCheck[];
}

export interface StatItem {
  name: string;
  value: number;
}

export interface InboundConfig {
  tag: string;
  receiverSettings?: any;
  proxySettings?: any;
}

export interface OutboundConfig {
  tag: string;
  senderSettings?: any;
  proxySettings?: any;
}

export interface UserInfo {
  email: string;
  level: number;
  inboundTag: string;
  online: boolean;
  uplink?: number;
  downlink?: number;
  accountType?: string;
}

export interface RoutingRule {
  tag: string;
  ruleTag: string;
}

export interface ObservatoryStatus {
  outboundTag: string;
  alive: boolean;
  delay: number;
  lastSeenTime: number;
  lastTryTime: number;
  lastErrorReason: string;
}

export interface ShareLinkRequest {
  protocol: string;
  address: string;
  port: number;
  uuid?: string;
  password?: string;
  email?: string;
  security?: string;
  flow?: string;
  type?: string;
  host?: string;
  path?: string;
  tls?: string;
  sni?: string;
  alpn?: string;
  fingerprint?: string;
  publicKey?: string;
  shortId?: string;
  spiderX?: string;
  remark?: string;
}

export interface BackupInfo {
  name: string;
  size: number;
  modified: string;
}

export type SubscriptionSourceType = "url" | "manual" | "file";

export interface SubscriptionRecord {
  id: string;
  sourceType: SubscriptionSourceType;
  url?: string;
  sourceName?: string;
  remark: string;
  autoRefresh: boolean;
  refreshIntervalMin: number;
  lastRefresh?: string;
  nodeCount: number;
}

export interface SubscriptionUpdateRequest {
  sourceType?: SubscriptionSourceType;
  url?: string;
  remark?: string;
  autoRefresh?: boolean;
  refreshIntervalMin?: number;
}

export interface SubscriptionCreateRequest {
  url?: string;
  content?: string;
  sourceName?: string;
  sourceType?: SubscriptionSourceType;
  remark: string;
  autoRefresh: boolean;
  refreshIntervalMin: number;
}

export type NodeStatus =
  | "candidate"
  | "staging"
  | "active"
  | "quarantine"
  | "removed";
export type TransitionReason =
  | "subscription_node_discovered"
  | "outbound_registration_failed"
  | "probe_qualified"
  | "probe_requalified"
  | "probe_failures_exceeded"
  | "manual_validate"
  | "manual_promote"
  | "manual_restore"
  | "manual_quarantine"
  | "manual_remove"
  | "subscription_missing"
  | "subscription_deleted"
  | "subscription_reintroduced"
  | "migration_legacy_demoted";

export type CleanlinessStatus = "unknown" | "trusted" | "suspicious";
export type NodeExitIPStatus = "unknown" | "available" | "error";
export type NodeNetworkType =
  | "unknown"
  | "residential_likely"
  | "isp_likely"
  | "datacenter_likely";
export type NodeIntelligenceConfidence = "unknown" | "low" | "medium" | "high";
export type MachineState = "clean" | "proxied" | "degraded" | "recovering";
export type MachineStateReason =
  | "startup_default_clean"
  | "startup_status_unavailable"
  | "startup_cleanup_failed"
  | "operator_enabled"
  | "tun_start_failed"
  | "operator_restore_clean"
  | "enable_blocked_min_active_not_met"
  | "pool_below_min_active_nodes"
  | "automatic_fallback_min_active_not_met"
  | "fallback_failed"
  | "state_load_defaulted";

export interface ValidationConfig {
  minSamples: number;
  maxFailRate: number;
  maxAvgDelayMs: number;
  probeIntervalSec: number;
  probeUrl: string;
  demoteAfterFails: number;
  autoRemoveDemoted: boolean;
  minActiveNodes: number;
  minBandwidthKbps: number;
}

export interface NodeRecord {
  id: string;
  uri: string;
  remark: string;
  protocol: string;
  address: string;
  port: number;
  outboundTag: string;
  status: NodeStatus;
  statusReason: TransitionReason;
  subscriptionMissing?: boolean;
  subscriptionId: string;
  addedAt: string;
  promotedAt?: string;
  statusUpdatedAt?: string;
  lastEventAt?: string;
  totalPings: number;
  failedPings: number;
  avgDelayMs: number;
  consecutiveFails: number;
  lastCheckedAt?: string;
  cleanliness: CleanlinessStatus;
  cleanlinessReason?: string;
  cleanlinessDetail?: string;
  cleanlinessConfidence: NodeIntelligenceConfidence;
  bandwidthTier: string;
  exitIpStatus: NodeExitIPStatus;
  exitIp?: string;
  exitIpSource?: string;
  exitIpError?: string;
  exitIpCheckedAt?: string;
  networkType: NodeNetworkType;
  networkTypeReason?: string;
  networkTypeDetail?: string;
  networkTypeConfidence: NodeIntelligenceConfidence;
  intelligenceExitIp?: string;
  intelligenceCheckedAt?: string;
  intelligenceError?: string;
}

export interface NodeEvent {
  nodeId: string;
  remark?: string;
  status: NodeStatus;
  reason: TransitionReason;
  actor: "system" | "operator" | "migration";
  at: string;
  details?: string;
  nodeAddress?: string;
}

export interface NodePoolSummary {
  candidateCount: number;
  stagingCount: number;
  activeCount: number;
  quarantineCount: number;
  removedCount: number;
  trustedCount: number;
  suspiciousCount: number;
  unknownCleanCount: number;
  activeNodes: number;
  minActiveNodes: number;
  healthy: boolean;
  lastEvaluatedAt: string;
  latestEventAt?: string;
  latestEventReason?: TransitionReason;
  latestEventStatus?: NodeStatus;
  latestEventActor?: "system" | "operator" | "migration";
  latestEventNodeId?: string;
  latestEventNodeAddress?: string;
}

export interface NodePoolDashboardResponse {
  nodes: NodeRecord[];
  summary: NodePoolSummary;
  recentEvents: NodeEvent[];
}

export interface MachineEvent {
  state: MachineState;
  reason: MachineStateReason;
  actor: "system" | "operator" | "migration";
  at: string;
  details?: string;
}

export interface TunEgressObservation {
  status: string;
  route: string;
  ip?: string;
  checkedAt?: string;
  source?: string;
  stale?: boolean;
  note?: string;
  error?: string;
}

export interface TunRoutingDiagnostic {
  category: string;
  dnsPath: string;
  resolver: string;
  route: string;
  reason: string;
  domains?: string[];
}

export type TunAggregationMode =
  | "single_best"
  | "redundant_2"
  | "weighted_split";
export type TunAggregationSchedulerPolicy =
  | "single_best"
  | "redundant_2"
  | "weighted_split";
export type TunAggregationStatusCode =
  | "disabled"
  | "requested"
  | "fallback_stable";
export type TunAggregationRuntimePath =
  | "stable_single_path"
  | "experimental_udp_quic_aggregation";

export interface TunAggregationHealthSettings {
  maxSessionLossPct: number;
  maxPathJitterMs: number;
  rollbackOnConsecutiveFailures: number;
}

export interface TunAggregationSettings {
  enabled: boolean;
  mode: TunAggregationMode;
  maxPathsPerSession: number;
  schedulerPolicy: TunAggregationSchedulerPolicy;
  relayEndpoint: string;
  health: TunAggregationHealthSettings;
}

export type TunAggregationPrototypePathState =
  | "selected"
  | "standby"
  | "excluded";

export interface TunAggregationPrototypePath {
  nodeId: string;
  remark?: string;
  outboundTag: string;
  state: TunAggregationPrototypePathState;
  eligible: boolean;
  selected: boolean;
  score: number;
  latencyMs: number;
  lossPct: number;
  consecutiveFails: number;
  lastCheckedAt?: string;
  reason: string;
}

export interface TunAggregationPrototypeSession {
  sessionId: string;
  state: string;
  flow: string;
  schedulerPolicy: TunAggregationSchedulerPolicy;
  candidatePathIds?: string[];
  selectedPathIds?: string[];
  createdAt: string;
  lastSeenAt: string;
  expiresAt: string;
  reason: string;
}

export interface TunAggregationPrototypeStatus {
  ready: boolean;
  metricSource: string;
  sessionTtlSeconds: number;
  candidatePathCount: number;
  selectedPathCount: number;
  sessionCount: number;
  paths: TunAggregationPrototypePath[];
  sessions: TunAggregationPrototypeSession[];
  note?: string;
}

export interface TunAggregationRelaySession {
  sessionId: string;
  flow: string;
  schedulerPolicy: TunAggregationSchedulerPolicy;
  pathIds?: string[];
  packetCount: number;
  deliveredPacketCount: number;
  duplicateDrops: number;
  reorderedPackets: number;
  maxReorderBufferDepth: number;
  deliveredBytes: number;
  startupLatencyMs: number;
  stallCount: number;
  goodputKbps: number;
  reason: string;
  createdAt: string;
}

export interface TunAggregationRelayStatus {
  ready: boolean;
  contractVersion: string;
  endpoint?: string;
  sessionCount: number;
  packetCount: number;
  deliveredPacketCount: number;
  duplicateDrops: number;
  reorderedPackets: number;
  maxReorderBufferDepth: number;
  sessions: TunAggregationRelaySession[];
  note?: string;
}

export type TunAggregationBenchmarkScenarioName =
  | "clean_paths"
  | "degraded_primary";

export interface TunAggregationBenchmarkResult {
  startupLatencyMs: number;
  stallCount: number;
  goodputKbps: number;
  lossPct: number;
  stabilityPct: number;
}

export interface TunAggregationBenchmarkScenario {
  name: TunAggregationBenchmarkScenarioName;
  baseline: TunAggregationBenchmarkResult;
  aggregated: TunAggregationBenchmarkResult;
  startupLatencyGainMs: number;
  stallReduction: number;
  goodputGainKbps: number;
  lossReductionPct: number;
  stabilityGainPct: number;
}

export interface TunAggregationBenchmarkStatus {
  ready: boolean;
  packetCount: number;
  payloadBytes: number;
  scenarios: TunAggregationBenchmarkScenario[];
  note?: string;
}

export interface TunAggregationStatus {
  enabled: boolean;
  status: TunAggregationStatusCode;
  requestedPath: TunAggregationRuntimePath;
  effectivePath: TunAggregationRuntimePath;
  ready: boolean;
  relayConfigured: boolean;
  mode: TunAggregationMode;
  maxPathsPerSession: number;
  schedulerPolicy: TunAggregationSchedulerPolicy;
  relayEndpoint?: string;
  reason: string;
  prototype?: TunAggregationPrototypeStatus;
  relay?: TunAggregationRelayStatus;
  benchmark?: TunAggregationBenchmarkStatus;
}

export interface TunStatusResponse {
  status: string;
  running: boolean;
  available: boolean;
  allowRemote: boolean;
  useSudo: boolean;
  helperExists: boolean;
  elevationReady: boolean;
  helperCurrent: boolean;
  binaryCurrent: boolean;
  privilegeInstallRecommended: boolean;
  binaryPath: string;
  helperPath: string;
  stateDir: string;
  runtimeConfigPath: string;
  interfaceName: string;
  mtu: number;
  remoteDns: string[];
  configPath: string;
  xrayBinary: string;
  message: string;
  lastOutput?: string;
  diagnostics?: string[];
  directEgress?: TunEgressObservation;
  proxyEgress?: TunEgressObservation;
  routingDiagnostics?: TunRoutingDiagnostic[];
  aggregation?: TunAggregationStatus;
  machineState?: MachineState;
  lastStateReason?: MachineStateReason;
  lastStateChangedAt?: string;
  recentMachineEvents?: MachineEvent[];
}

export type TunSelectionPolicy =
  | "fastest"
  | "lowest_latency"
  | "lowest_fail_rate";
export type TunRouteMode = "strict_proxy" | "auto_tested";
export type TunDestinationBindingPreset =
  | "openai"
  | "chatgpt"
  | "claude"
  | "gemini"
  | "github_copilot"
  | "openrouter"
  | "cursor"
  | "qwen"
  | "perplexity"
  | "deepseek"
  | "custom";

export type TunDestinationBindingSelectionMode =
  | "primary_only"
  | "failover_ordered"
  | "failover_fastest";

export interface TunDestinationBinding {
  preset: TunDestinationBindingPreset;
  domains: string[];
  nodeId: string;
  fallbackNodeIds?: string[];
  selectionMode?: TunDestinationBindingSelectionMode;
}

export interface TunEditableSettings {
  selectionPolicy: TunSelectionPolicy;
  routeMode: TunRouteMode;
  remoteDns: string[];
  protectDomains: string[];
  protectCidrs: string[];
  destinationBindings: TunDestinationBinding[];
  aggregation: TunAggregationSettings;
}

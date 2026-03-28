export interface SysStats {
  numGoroutine: number
  numGC: number
  alloc: number
  totalAlloc: number
  sys: number
  mallocs: number
  frees: number
  liveObjects: number
  pauseTotalNs: number
  uptime: number
}

export interface StatItem {
  name: string
  value: number
}

export interface InboundConfig {
  tag: string
  receiverSettings?: any
  proxySettings?: any
}

export interface OutboundConfig {
  tag: string
  senderSettings?: any
  proxySettings?: any
}

export interface UserInfo {
  email: string
  level: number
  inboundTag: string
  online: boolean
  uplink?: number
  downlink?: number
  accountType?: string
}

export interface RoutingRule {
  tag: string
  ruleTag: string
}

export interface ObservatoryStatus {
  outboundTag: string
  alive: boolean
  delay: number
  lastSeenTime: number
  lastTryTime: number
  lastErrorReason: string
}

export interface ShareLinkRequest {
  protocol: string
  address: string
  port: number
  uuid?: string
  password?: string
  email?: string
  security?: string
  flow?: string
  type?: string
  host?: string
  path?: string
  tls?: string
  sni?: string
  alpn?: string
  fingerprint?: string
  publicKey?: string
  shortId?: string
  spiderX?: string
  remark?: string
}

export interface BackupInfo {
  name: string
  size: number
  modified: string
}

export type NodeStatus = 'candidate' | 'staging' | 'active' | 'quarantine' | 'removed'
export type TransitionReason =
  | 'subscription_node_discovered'
  | 'outbound_registration_failed'
  | 'probe_qualified'
  | 'probe_requalified'
  | 'probe_failures_exceeded'
  | 'manual_promote'
  | 'manual_quarantine'
  | 'manual_remove'
  | 'subscription_missing'
  | 'subscription_deleted'
  | 'subscription_reintroduced'
  | 'migration_legacy_demoted'

export type CleanlinessStatus = 'unknown' | 'trusted' | 'suspicious'
export type MachineState = 'clean' | 'proxied' | 'degraded' | 'recovering'
export type MachineStateReason =
  | 'startup_default_clean'
  | 'startup_status_unavailable'
  | 'startup_cleanup_failed'
  | 'operator_enabled'
  | 'tun_start_failed'
  | 'operator_restore_clean'
  | 'enable_blocked_min_active_not_met'
  | 'pool_below_min_active_nodes'
  | 'automatic_fallback_min_active_not_met'
  | 'fallback_failed'
  | 'state_load_defaulted'

export interface ValidationConfig {
  minSamples: number
  maxFailRate: number
  maxAvgDelayMs: number
  probeIntervalSec: number
  probeUrl: string
  demoteAfterFails: number
  autoRemoveDemoted: boolean
  minActiveNodes: number
  minBandwidthKbps: number
}

export interface NodeRecord {
  id: string
  uri: string
  remark: string
  protocol: string
  address: string
  port: number
  outboundTag: string
  status: NodeStatus
  statusReason: TransitionReason
  subscriptionId: string
  addedAt: string
  promotedAt?: string
  statusUpdatedAt?: string
  lastEventAt?: string
  totalPings: number
  failedPings: number
  avgDelayMs: number
  consecutiveFails: number
  lastCheckedAt?: string
  cleanliness: CleanlinessStatus
  bandwidthTier: string
}

export interface NodeEvent {
  nodeId: string
  remark?: string
  status: NodeStatus
  reason: TransitionReason
  actor: 'system' | 'operator' | 'migration'
  at: string
  details?: string
  nodeAddress?: string
}

export interface NodePoolSummary {
  candidateCount: number
  stagingCount: number
  activeCount: number
  quarantineCount: number
  removedCount: number
  trustedCount: number
  suspiciousCount: number
  unknownCleanCount: number
  activeNodes: number
  minActiveNodes: number
  healthy: boolean
  lastEvaluatedAt: string
  latestEventAt?: string
  latestEventReason?: TransitionReason
  latestEventStatus?: NodeStatus
  latestEventActor?: 'system' | 'operator' | 'migration'
  latestEventNodeId?: string
  latestEventNodeAddress?: string
}

export interface NodePoolDashboardResponse {
  nodes: NodeRecord[]
  summary: NodePoolSummary
  recentEvents: NodeEvent[]
}

export interface MachineEvent {
  state: MachineState
  reason: MachineStateReason
  actor: 'system' | 'operator' | 'migration'
  at: string
  details?: string
}

export interface TunStatusResponse {
  status: string
  running: boolean
  available: boolean
  allowRemote: boolean
  useSudo: boolean
  helperExists: boolean
  elevationReady: boolean
  helperCurrent: boolean
  binaryCurrent: boolean
  privilegeInstallRecommended: boolean
  binaryPath: string
  helperPath: string
  stateDir: string
  runtimeConfigPath: string
  interfaceName: string
  mtu: number
  remoteDns: string[]
  configPath: string
  xrayBinary: string
  message: string
  lastOutput?: string
  diagnostics?: string[]
  machineState?: MachineState
  lastStateReason?: MachineStateReason
  lastStateChangedAt?: string
  recentMachineEvents?: MachineEvent[]
}

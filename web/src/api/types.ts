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

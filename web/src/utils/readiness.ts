import type { ReadinessArea, ReadinessCheck, ReadinessResponse, ReadinessSeverity } from '@/api/types'

type Translate = (key: string, params?: Record<string, unknown>) => string
type StatusType = 'success' | 'warning' | 'error' | 'info'

export interface ReadinessDescription {
  title: string
  summary: string
  details: string[]
}

export interface ReadinessOverview {
  type: StatusType
  badgeLabel: string
  title: string
  description: string
}

function stringFact(check: ReadinessCheck, key: string): string {
  const value = check.facts?.[key]
  return typeof value === 'string' ? value : ''
}

function numberFact(check: ReadinessCheck, key: string): number {
  const value = check.facts?.[key]
  return typeof value === 'number' ? value : 0
}

function boolFact(check: ReadinessCheck, key: string): boolean {
  return check.facts?.[key] === true
}

function stringArrayFact(check: ReadinessCheck, key: string): string[] {
  const value = check.facts?.[key]
  return Array.isArray(value) ? value.filter((item): item is string => typeof item === 'string') : []
}

function joinItems(items: string[]): string {
  return items.join(', ')
}

export function readinessSeverityType(severity: ReadinessSeverity): Exclude<StatusType, 'info'> {
  switch (severity) {
    case 'blocking':
      return 'error'
    case 'warning':
      return 'warning'
    default:
      return 'success'
  }
}

export function readinessSeverityLabel(t: Translate, severity: ReadinessSeverity): string {
  switch (severity) {
    case 'blocking':
      return t('readiness.severity.blocking')
    case 'warning':
      return t('readiness.severity.warning')
    default:
      return t('readiness.severity.ok')
  }
}

export function readinessAreaLabel(t: Translate, area: ReadinessArea): string {
  return t(`readiness.area.${area}`)
}

export function describeReadinessOverview(t: Translate, readiness: ReadinessResponse | null): ReadinessOverview {
  if (!readiness) {
    return {
      type: 'info',
      badgeLabel: t('readiness.overview.unavailableBadge'),
      title: t('readiness.overview.unavailableTitle'),
      description: t('readiness.overview.unavailableBody')
    }
  }

  if (readiness.blockingCount > 0) {
    return {
      type: 'error',
      badgeLabel: t('readiness.overview.blockingBadge'),
      title: t('readiness.overview.blockingTitle'),
      description: t('readiness.overview.blockingBody', { count: readiness.blockingCount })
    }
  }

  if (readiness.warningCount > 0) {
    return {
      type: 'warning',
      badgeLabel: t('readiness.overview.warningBadge'),
      title: t('readiness.overview.warningTitle'),
      description: t('readiness.overview.warningBody', { count: readiness.warningCount })
    }
  }

  return {
    type: 'success',
    badgeLabel: t('readiness.overview.healthyBadge'),
    title: t('readiness.overview.healthyTitle'),
    description: t('readiness.overview.healthyBody')
  }
}

export function describeReadinessCheck(t: Translate, check: ReadinessCheck): ReadinessDescription {
  switch (check.key) {
    case 'config_path':
      return describeConfigPath(t, check)
    case 'config_sections':
      return describeConfigSections(t, check)
    case 'subscriptions':
      return describeSubscriptions(t, check)
    case 'probing':
      return describeProbing(t, check)
    case 'node_pool':
      return describeNodePool(t, check)
    case 'tun':
      return describeTun(t, check)
    case 'updates':
      return describeUpdates(t, check)
    default:
      return {
        title: check.key,
        summary: t('readiness.checks.unknown.summary'),
        details: []
      }
  }
}

function describeConfigPath(t: Translate, check: ReadinessCheck): ReadinessDescription {
  const path = stringFact(check, 'path')
  const status = stringFact(check, 'status')
  const error = stringFact(check, 'error')
  const details = path ? [t('readiness.checks.configPath.pathDetail', { path })] : []
  if (error) {
    details.push(t('readiness.checks.configPath.errorDetail', { error }))
  }

  let summary = t('readiness.checks.configPath.ok', { path })
  if (status === 'missing') {
    summary = t('readiness.checks.configPath.missing')
  } else if (status === 'not_found') {
    summary = t('readiness.checks.configPath.notFound', { path })
  } else if (status === 'invalid') {
    summary = t('readiness.checks.configPath.invalid', { path })
  }

  return {
    title: t('readiness.checks.configPath.title'),
    summary,
    details
  }
}

function describeConfigSections(t: Translate, check: ReadinessCheck): ReadinessDescription {
  const missingSections = stringArrayFact(check, 'missingSections')
  const missingStatsFlags = stringArrayFact(check, 'missingStatsFlags')
  const missingServices = stringArrayFact(check, 'missingServices')
  const status = stringFact(check, 'status')
  const details: string[] = []

  if (missingSections.length > 0) {
    details.push(t('readiness.checks.configSections.missingSections', { items: joinItems(missingSections) }))
  }
  if (missingStatsFlags.length > 0) {
    details.push(t('readiness.checks.configSections.missingStatsFlags', { items: joinItems(missingStatsFlags) }))
  }
  if (missingServices.length > 0) {
    details.push(t('readiness.checks.configSections.missingServices', { items: joinItems(missingServices) }))
  }

  let summary = t('readiness.checks.configSections.ok')
  if (status === 'unavailable') {
    summary = t('readiness.checks.configSections.unavailable')
  } else if (missingSections.length > 0) {
    summary = t('readiness.checks.configSections.missingSectionsSummary')
  } else if (missingStatsFlags.length > 0 || missingServices.length > 0) {
    summary = t('readiness.checks.configSections.partialSummary')
  }

  return {
    title: t('readiness.checks.configSections.title'),
    summary,
    details
  }
}

function describeSubscriptions(t: Translate, check: ReadinessCheck): ReadinessDescription {
  const status = stringFact(check, 'status')
  const subscriptionCount = numberFact(check, 'subscriptionCount')
  const nodeCount = numberFact(check, 'nodeCount')
  const details = [
    t('readiness.checks.subscriptions.poolDetail', {
      subscriptions: subscriptionCount,
      nodes: nodeCount,
      active: numberFact(check, 'activeCount'),
      staging: numberFact(check, 'stagingCount'),
      candidate: numberFact(check, 'candidateCount'),
      quarantine: numberFact(check, 'quarantineCount'),
      removed: numberFact(check, 'removedCount')
    })
  ]

  let summary = t('readiness.checks.subscriptions.ok', { subscriptions: subscriptionCount, nodes: nodeCount })
  if (status === 'unavailable') {
    summary = t('readiness.checks.subscriptions.unavailable')
  } else if (status === 'empty') {
    summary = t('readiness.checks.subscriptions.empty')
  } else if (status === 'no_nodes') {
    summary = t('readiness.checks.subscriptions.noNodes', { subscriptions: subscriptionCount })
  }

  return {
    title: t('readiness.checks.subscriptions.title'),
    summary,
    details
  }
}

function describeProbing(t: Translate, check: ReadinessCheck): ReadinessDescription {
  const status = stringFact(check, 'status')
  const probeUrl = stringFact(check, 'probeUrl')
  const intervalSec = numberFact(check, 'probeIntervalSec')
  const tagCount = numberFact(check, 'tagCount')
  const details = [t('readiness.checks.probing.configDetail', { probeUrl, intervalSec, tagCount })]

  let summary = t('readiness.checks.probing.running', { tagCount, intervalSec })
  if (status === 'unavailable') {
    summary = t('readiness.checks.probing.unavailable')
  } else if (status === 'not_started') {
    summary = t('readiness.checks.probing.notStarted')
  } else if (status === 'dispatcher_unavailable') {
    summary = t('readiness.checks.probing.dispatcherUnavailable')
  } else if (status === 'idle') {
    summary = t('readiness.checks.probing.idle')
  } else if (status === 'stopped') {
    summary = t('readiness.checks.probing.stopped')
  }

  return {
    title: t('readiness.checks.probing.title'),
    summary,
    details
  }
}

function describeNodePool(t: Translate, check: ReadinessCheck): ReadinessDescription {
  const status = stringFact(check, 'status')
  const activeCount = numberFact(check, 'activeCount')
  const minActiveNodes = numberFact(check, 'minActiveNodes')
  const details = [
    t('readiness.checks.nodePool.distributionDetail', {
      active: activeCount,
      candidate: numberFact(check, 'candidateCount'),
      staging: numberFact(check, 'stagingCount'),
      quarantine: numberFact(check, 'quarantineCount'),
      removed: numberFact(check, 'removedCount')
    })
  ]

  let summary = t('readiness.checks.nodePool.ok', { active: activeCount, minimum: minActiveNodes })
  if (status === 'unavailable') {
    summary = t('readiness.checks.nodePool.unavailable')
  } else if (status === 'empty') {
    summary = t('readiness.checks.nodePool.empty')
  } else if (status === 'below_minimum') {
    summary = t('readiness.checks.nodePool.belowMinimum', { active: activeCount, minimum: minActiveNodes })
  }

  return {
    title: t('readiness.checks.nodePool.title'),
    summary,
    details
  }
}

function describeTun(t: Translate, check: ReadinessCheck): ReadinessDescription {
  const status = stringFact(check, 'status')
  const message = stringFact(check, 'message')
  const details: string[] = []

  if (message) {
    details.push(t('readiness.checks.tun.messageDetail', { message }))
  }
  if (!boolFact(check, 'helperExists')) {
    details.push(t('readiness.checks.tun.helperMissingDetail'))
  }
  if (boolFact(check, 'privilegeInstallRecommended')) {
    details.push(t('readiness.checks.tun.privilegeRecommendedDetail'))
  }
  const machineState = stringFact(check, 'machineState')
  if (machineState) {
    details.push(t('readiness.checks.tun.machineStateDetail', { state: machineState }))
  }

  let summary = boolFact(check, 'running') ? t('readiness.checks.tun.running') : t('readiness.checks.tun.stopped')
  if (status === 'unavailable') {
    summary = t('readiness.checks.tun.unavailable')
  } else if (check.severity === 'blocking') {
    summary = t('readiness.checks.tun.degraded')
  } else if (check.severity === 'warning') {
    summary = t('readiness.checks.tun.warning')
  }

  return {
    title: t('readiness.checks.tun.title'),
    summary,
    details
  }
}

function describeUpdates(t: Translate, check: ReadinessCheck): ReadinessDescription {
  const status = stringFact(check, 'status')
  const currentVersion = stringFact(check, 'currentVersion')
  const latestVersion = stringFact(check, 'latestVersion')
  const source = stringFact(check, 'source')
  const message = stringFact(check, 'message')
  const details: string[] = []

  if (source) {
    details.push(t('readiness.checks.updates.sourceDetail', { source }))
  }
  if (message) {
    details.push(t('readiness.checks.updates.messageDetail', { message }))
  }

  let summary = t('readiness.checks.updates.ok', { currentVersion })
  if (status === 'unavailable') {
    summary = t('readiness.checks.updates.unavailable')
  } else if (status === 'error') {
    summary = t('readiness.checks.updates.error')
  } else if (status === 'stale') {
    summary = t('readiness.checks.updates.stale')
  } else if (boolFact(check, 'updateAvailable')) {
    summary = t('readiness.checks.updates.updateAvailable', { currentVersion, latestVersion })
  }

  return {
    title: t('readiness.checks.updates.title'),
    summary,
    details
  }
}

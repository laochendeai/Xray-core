import type { NodeRecord } from '@/api/types'

export type PoolSortMode =
  | 'quality'
  | 'last_checked_desc'
  | 'last_checked_asc'
  | 'fail_rate_asc'
  | 'fail_rate_desc'
  | 'avg_delay_asc'
  | 'avg_delay_desc'

export type RemovedSortMode =
  | 'removed_desc'
  | 'removed_asc'
  | 'fail_rate_asc'
  | 'fail_rate_desc'
  | 'avg_delay_asc'
  | 'avg_delay_desc'

export function failRateValue(node: Pick<NodeRecord, 'totalPings' | 'failedPings'>): number {
  if (!node.totalPings) return Number.POSITIVE_INFINITY
  return node.failedPings / node.totalPings
}

function delayValue(node: Pick<NodeRecord, 'avgDelayMs'>): number {
  return node.avgDelayMs > 0 ? node.avgDelayMs : Number.POSITIVE_INFINITY
}

function checkedAtValue(node: Pick<NodeRecord, 'lastCheckedAt' | 'statusUpdatedAt' | 'addedAt'>): number {
  const value = node.lastCheckedAt || node.statusUpdatedAt || node.addedAt
  const timestamp = value ? new Date(value).getTime() : 0
  return Number.isFinite(timestamp) ? timestamp : 0
}

function removedAtValue(node: Pick<NodeRecord, 'statusUpdatedAt' | 'lastEventAt' | 'addedAt'>): number {
  const value = node.statusUpdatedAt || node.lastEventAt || node.addedAt
  const timestamp = value ? new Date(value).getTime() : 0
  return Number.isFinite(timestamp) ? timestamp : 0
}

function compareNodesByQuality(a: NodeRecord, b: NodeRecord): number {
  const failRateDiff = failRateValue(a) - failRateValue(b)
  if (failRateDiff !== 0) return failRateDiff

  const delayDiff = delayValue(a) - delayValue(b)
  if (delayDiff !== 0) return delayDiff

  if (a.consecutiveFails !== b.consecutiveFails) {
    return a.consecutiveFails - b.consecutiveFails
  }

  if (a.totalPings !== b.totalPings) {
    return b.totalPings - a.totalPings
  }

  return checkedAtValue(b) - checkedAtValue(a)
}

function compareNullableMetric(aValue: number | null, bValue: number | null, direction: 'asc' | 'desc'): number {
  if (aValue == null && bValue == null) return 0
  if (aValue == null) return 1
  if (bValue == null) return -1
  return direction === 'asc' ? aValue - bValue : bValue - aValue
}

export function sortPoolNodes(entries: NodeRecord[], mode: PoolSortMode): NodeRecord[] {
  return [...entries].sort((a, b) => {
    switch (mode) {
      case 'last_checked_desc':
        return checkedAtValue(b) - checkedAtValue(a)
      case 'last_checked_asc':
        return checkedAtValue(a) - checkedAtValue(b)
      case 'fail_rate_asc': {
        const diff = compareNullableMetric(a.totalPings ? failRateValue(a) : null, b.totalPings ? failRateValue(b) : null, 'asc')
        return diff !== 0 ? diff : compareNodesByQuality(a, b)
      }
      case 'fail_rate_desc': {
        const diff = compareNullableMetric(a.totalPings ? failRateValue(a) : null, b.totalPings ? failRateValue(b) : null, 'desc')
        return diff !== 0 ? diff : compareNodesByQuality(a, b)
      }
      case 'avg_delay_asc': {
        const diff = compareNullableMetric(a.avgDelayMs > 0 ? a.avgDelayMs : null, b.avgDelayMs > 0 ? b.avgDelayMs : null, 'asc')
        return diff !== 0 ? diff : compareNodesByQuality(a, b)
      }
      case 'avg_delay_desc': {
        const diff = compareNullableMetric(a.avgDelayMs > 0 ? a.avgDelayMs : null, b.avgDelayMs > 0 ? b.avgDelayMs : null, 'desc')
        return diff !== 0 ? diff : compareNodesByQuality(a, b)
      }
      default:
        return compareNodesByQuality(a, b)
    }
  })
}

export function sortRemovedNodes(entries: NodeRecord[], mode: RemovedSortMode): NodeRecord[] {
  return [...entries].sort((a, b) => {
    switch (mode) {
      case 'removed_asc':
        return removedAtValue(a) - removedAtValue(b)
      case 'fail_rate_asc': {
        const diff = compareNullableMetric(a.totalPings ? failRateValue(a) : null, b.totalPings ? failRateValue(b) : null, 'asc')
        return diff !== 0 ? diff : removedAtValue(b) - removedAtValue(a)
      }
      case 'fail_rate_desc': {
        const diff = compareNullableMetric(a.totalPings ? failRateValue(a) : null, b.totalPings ? failRateValue(b) : null, 'desc')
        return diff !== 0 ? diff : removedAtValue(b) - removedAtValue(a)
      }
      case 'avg_delay_asc': {
        const diff = compareNullableMetric(a.avgDelayMs > 0 ? a.avgDelayMs : null, b.avgDelayMs > 0 ? b.avgDelayMs : null, 'asc')
        return diff !== 0 ? diff : removedAtValue(b) - removedAtValue(a)
      }
      case 'avg_delay_desc': {
        const diff = compareNullableMetric(a.avgDelayMs > 0 ? a.avgDelayMs : null, b.avgDelayMs > 0 ? b.avgDelayMs : null, 'desc')
        return diff !== 0 ? diff : removedAtValue(b) - removedAtValue(a)
      }
      default:
        return removedAtValue(b) - removedAtValue(a)
    }
  })
}

export function normalizeListInput(value: string): string[] {
  return Array.from(
    new Set(
      value
        .split(/[\n,]/)
        .map((item) => item.trim())
        .filter(Boolean)
    )
  )
}

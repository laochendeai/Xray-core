import { describe, expect, it } from 'vitest'

import type { NodeRecord } from '@/api/types'
import {
  bindingPreviewDomains,
  bindingPrimaryTestDomain,
  failRateValue,
  firstNodeIntelligenceDetail,
  normalizeListInput,
  sortPoolNodes,
  sortRemovedNodes,
  summarizeNodeIntelligence
} from '@/utils/nodePool'

function makeNode(overrides: Partial<NodeRecord>): NodeRecord {
  return {
    id: 'node',
    uri: 'vless://node',
    remark: 'node',
    protocol: 'vless',
    address: 'example.com',
    port: 443,
    outboundTag: '',
    status: 'candidate',
    statusReason: 'subscription_node_discovered',
    subscriptionId: 'sub-1',
    addedAt: '2026-01-01T00:00:00Z',
    promotedAt: undefined,
    statusUpdatedAt: '2026-01-01T00:00:00Z',
    lastEventAt: '2026-01-01T00:00:00Z',
    totalPings: 0,
    failedPings: 0,
    avgDelayMs: 0,
    consecutiveFails: 0,
    lastCheckedAt: '2026-01-01T00:00:00Z',
    cleanliness: 'unknown',
    cleanlinessConfidence: 'unknown',
    bandwidthTier: '',
    exitIpStatus: 'unknown',
    networkType: 'unknown',
    networkTypeConfidence: 'unknown',
    ...overrides
  }
}

describe('node pool utilities', () => {
  it('normalizes newline/comma separated lists and removes duplicates', () => {
    expect(normalizeListInput(' 1.1.1.1 \n8.8.8.8,1.1.1.1\n\nhttps://dns.google/dns-query '))
      .toEqual(['1.1.1.1', '8.8.8.8', 'https://dns.google/dns-query'])
  })

  it('sorts pool nodes by quality and explicit metric modes', () => {
    const highQuality = makeNode({
      id: 'high',
      totalPings: 20,
      failedPings: 0,
      avgDelayMs: 120,
      lastCheckedAt: '2026-01-03T00:00:00Z'
    })
    const mediumQuality = makeNode({
      id: 'medium',
      totalPings: 20,
      failedPings: 2,
      avgDelayMs: 200,
      lastCheckedAt: '2026-01-02T00:00:00Z'
    })
    const unknownQuality = makeNode({
      id: 'unknown',
      totalPings: 0,
      failedPings: 0,
      avgDelayMs: 0,
      lastCheckedAt: '2026-01-04T00:00:00Z'
    })

    expect(sortPoolNodes([mediumQuality, unknownQuality, highQuality], 'quality').map((node) => node.id))
      .toEqual(['high', 'medium', 'unknown'])
    expect(sortPoolNodes([mediumQuality, highQuality], 'last_checked_desc').map((node) => node.id))
      .toEqual(['high', 'medium'])
    expect(sortPoolNodes([mediumQuality, highQuality], 'avg_delay_desc').map((node) => node.id))
      .toEqual(['medium', 'high'])
  })

  it('sorts removed nodes by removal time and metrics', () => {
    const newest = makeNode({
      id: 'newest',
      status: 'removed',
      statusUpdatedAt: '2026-01-04T00:00:00Z',
      totalPings: 10,
      failedPings: 10,
      avgDelayMs: 900
    })
    const oldest = makeNode({
      id: 'oldest',
      status: 'removed',
      statusUpdatedAt: '2026-01-01T00:00:00Z',
      totalPings: 10,
      failedPings: 1,
      avgDelayMs: 100
    })

    expect(sortRemovedNodes([oldest, newest], 'removed_desc').map((node) => node.id)).toEqual(['newest', 'oldest'])
    expect(sortRemovedNodes([oldest, newest], 'fail_rate_asc').map((node) => node.id)).toEqual(['oldest', 'newest'])
  })

  it('returns infinity for nodes without probe samples', () => {
    expect(failRateValue(makeNode({ totalPings: 0, failedPings: 0 }))).toBe(Number.POSITIVE_INFINITY)
  })

  it('summarizes cleanliness and network-type verdicts', () => {
    const trustedResidential = makeNode({
      id: 'trusted',
      cleanliness: 'trusted',
      networkType: 'residential_likely'
    })
    const suspiciousDatacenter = makeNode({
      id: 'suspicious',
      cleanliness: 'suspicious',
      networkType: 'datacenter_likely'
    })
    const unknown = makeNode({
      id: 'unknown',
      cleanliness: 'unknown',
      networkType: 'unknown'
    })

    expect(summarizeNodeIntelligence([trustedResidential, suspiciousDatacenter, unknown])).toEqual({
      trustedCount: 1,
      suspiciousCount: 1,
      unknownCleanCount: 1,
      residentialCount: 1,
      datacenterCount: 1,
      unknownNetworkCount: 1
    })
  })

  it('picks the first usable intelligence detail', () => {
    expect(
      firstNodeIntelligenceDetail(
        makeNode({
          cleanlinessDetail: '',
          networkTypeDetail: 'network detail',
          intelligenceError: 'lookup error',
          exitIpError: 'exit-ip error'
        })
      )
    ).toBe('network detail')

    expect(
      firstNodeIntelligenceDetail(
        makeNode({
          cleanlinessDetail: '',
          networkTypeDetail: '',
          intelligenceError: '',
          exitIpError: ''
        })
      )
    ).toBe('')
  })

  it('expands preset destination bindings and picks a test domain', () => {
    expect(bindingPreviewDomains({ preset: 'openai', domains: [] })).toContain('domain:api.openai.com')
    expect(bindingPrimaryTestDomain({ preset: 'chatgpt', domains: [] })).toBe('chatgpt.com')
    expect(bindingPreviewDomains({ preset: 'custom', domains: ['*.ignored.example', 'domain:custom.example'] })).toEqual([
      'domain:ignored.example',
      'domain:custom.example'
    ])
  })
})

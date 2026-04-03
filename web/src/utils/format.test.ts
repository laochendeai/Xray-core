import { describe, expect, it, vi } from 'vitest'

import { copyToClipboard, formatBytes, formatSpeed, formatTimestamp, formatUptime, generateUUID } from '@/utils/format'

describe('format utilities', () => {
  it('formats bytes and speeds with readable units', () => {
    expect(formatBytes(0)).toBe('0 B')
    expect(formatBytes(1536)).toBe('1.5 KB')
    expect(formatSpeed(1048576)).toBe('1 MB/s')
  })

  it('formats uptime into compact parts', () => {
    expect(formatUptime(59)).toBe('59s')
    expect(formatUptime(3661)).toBe('1h 1m')
    expect(formatUptime(90061)).toBe('1d 1h 1m')
  })

  it('formats timestamps and generates UUID-like identifiers', () => {
    expect(formatTimestamp(0)).toBe('-')
    expect(formatTimestamp(1710000000)).not.toBe('-')
    expect(generateUUID()).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/)
  })

  it('delegates clipboard copies to the browser API', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.assign(navigator, {
      clipboard: { writeText }
    })

    await copyToClipboard('hello')

    expect(writeText).toHaveBeenCalledWith('hello')
  })
})

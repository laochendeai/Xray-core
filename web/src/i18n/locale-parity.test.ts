import { describe, expect, it } from 'vitest'

import en from '@/i18n/locales/en.json'
import zhCn from '@/i18n/locales/zh-CN.json'

function flattenKeys(input: unknown, prefix = ''): string[] {
  if (input == null || typeof input !== 'object' || Array.isArray(input)) {
    return prefix ? [prefix] : []
  }

  return Object.entries(input as Record<string, unknown>)
    .flatMap(([key, value]) => flattenKeys(value, prefix ? `${prefix}.${key}` : key))
}

describe('locale parity', () => {
  it('keeps zh-CN and en locale key paths aligned', () => {
    const zhKeys = flattenKeys(zhCn).sort()
    const enKeys = flattenKeys(en).sort()

    expect(zhKeys).toEqual(enKeys)
  })
})

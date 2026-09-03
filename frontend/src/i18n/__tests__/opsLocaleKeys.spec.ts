import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import en from '@/i18n/locales/en'
import zh from '@/i18n/locales/zh'

function flattenKeys(obj: Record<string, any>, prefix = ''): string[] {
  const keys: string[] = []
  for (const [k, v] of Object.entries(obj)) {
    const fullKey = prefix ? `${prefix}.${k}` : k
    if (typeof v === 'object' && v !== null && !Array.isArray(v)) {
      keys.push(...flattenKeys(v, fullKey))
    } else {
      keys.push(fullKey)
    }
  }
  return keys
}

function valueAtPath(source: Record<string, any>, path: string): unknown {
  return path.split('.').reduce<unknown>((value, segment) => {
    if (!value || typeof value !== 'object') return undefined
    return (value as Record<string, unknown>)[segment]
  }, source)
}

const opsSchemeBFiles = [
  '../../views/admin/ops/OpsDashboard.vue',
  '../../views/admin/ops/components/OpsAlertEventsCard.vue',
  '../../views/admin/ops/components/OpsDashboardHeader.vue',
  '../../views/admin/ops/components/OpsRequestDetailsModal.vue',
  '../../views/admin/ops/components/OpsSystemLogTable.vue',
  '../../views/admin/ops/components/OpsTrafficInvestigationRail.vue',
  '../../views/admin/ops/components/OpsWorkbenchShell.vue',
  '../../views/admin/ops/components/OpsWorkspaceNav.vue',
] as const

function collectStaticLocaleKeys(): string[] {
  const keys = new Set<string>()
  for (const relativePath of opsSchemeBFiles) {
    const source = readFileSync(fileURLToPath(new URL(relativePath, import.meta.url)), 'utf8')
    for (const match of source.matchAll(/\bt\(\s*['"]((?:admin\.ops|common)\.[^'"]+)['"]/g)) {
      keys.add(match[1])
    }
  }
  return [...keys].sort()
}

describe('ops locale key completeness', () => {
  const requiredKeys = [
    'admin.ops.result',
    'admin.ops.timeRange.custom',
    'admin.ops.customTimeRange.startTime',
    'admin.ops.customTimeRange.endTime',
  ]

  for (const key of requiredKeys) {
    it(`en locale has ${key}`, () => {
      const enKeys = flattenKeys(en)
      expect(enKeys).toContain(key)
    })
  }

  it('keeps every resource-health workspace label symmetric between English and Chinese', () => {
    const enResourceHealth = (en as Record<string, any>).admin.ops.resourceHealth as Record<string, any>
    const zhResourceHealth = (zh as Record<string, any>).admin.ops.resourceHealth as Record<string, any>

    expect(flattenKeys(enResourceHealth).sort()).toEqual(flattenKeys(zhResourceHealth).sort())
  })

  it('keeps the complete ops namespace symmetric between English and Chinese', () => {
    const enOps = (en as Record<string, any>).admin.ops as Record<string, any>
    const zhOps = (zh as Record<string, any>).admin.ops as Record<string, any>

    expect(flattenKeys(enOps).sort()).toEqual(flattenKeys(zhOps).sort())
  })

  it('resolves every static Scheme B locale key in both locales', () => {
    const keys = collectStaticLocaleKeys()
    const missingEn = keys.filter((key) => typeof valueAtPath(en as Record<string, any>, key) !== 'string')
    const missingZh = keys.filter((key) => typeof valueAtPath(zh as Record<string, any>, key) !== 'string')

    expect({ missingEn, missingZh }).toEqual({ missingEn: [], missingZh: [] })
  })
})

describe('groups locale key completeness', () => {
  it('en locale has admin.groups.failedToSave', () => {
    const enKeys = flattenKeys(en)
    expect(enKeys).toContain('admin.groups.failedToSave')
  })

  const webSearchPricingKeys = [
    'admin.groups.webSearchPricing.title',
    'admin.groups.webSearchPricing.pricePerCall',
    'admin.groups.webSearchPricing.pricePerCallHint',
    'admin.groups.webSearchPricing.finalPricePreview',
  ]

  for (const key of webSearchPricingKeys) {
    it(`en and zh locales both have ${key}`, () => {
      expect(flattenKeys(en)).toContain(key)
      expect(flattenKeys(zh)).toContain(key)
    })
  }
})

import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'

import UsageStatsCards from '@/components/shared-domain/usage/UsageStatsCards.vue'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../UsageStatsCards.vue')
const componentSource = readFileSync(componentPath, 'utf8')
const wrappers: VueWrapper[] = []

const messages: Record<string, string> = {
  'usage.totalRequests': 'Total Requests',
  'usage.inSelectedRange': 'in selected range',
  'usage.totalTokens': 'Total Tokens',
  'usage.in': 'In',
  'usage.out': 'Out',
  'usage.cacheTotal': 'Cache',
  'usage.cacheBreakdown': 'Cache Token Breakdown',
  'usage.cacheCreationTokensLabel': 'Cache Creation',
  'usage.cacheReadTokensLabel': 'Cache Read',
  'usage.tokenDetails': 'Token details',
  'usage.totalCost': 'Total Cost',
  'usage.accountCost': 'Cost',
  'usage.standardCost': 'Standard',
  'usage.costDetails': 'Cost details',
  'usage.avgDuration': 'Avg Duration',
  'usage.averageDurationHint': 'Average per request',
  'admin.usage.workspace.summaryLabel': 'Usage summary',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

const stats = {
  total_requests: 987,
  total_input_tokens: 987_654,
  total_output_tokens: 222_222,
  total_cache_tokens: 35_000,
  total_cache_creation_tokens: 12_345,
  total_cache_read_tokens: 22_655,
  total_tokens: 1_245_678,
  total_cost: 4.56789,
  total_actual_cost: 3.45678,
  total_account_cost: 2.34567,
  average_duration_ms: 1_250,
}

function mountStats(props: Record<string, unknown> = {}) {
  const wrapper = mount(UsageStatsCards, {
    props: {
      stats,
      ...props,
    },
    global: {
      stubs: {
        Icon: true,
      },
    },
  })
  wrappers.push(wrapper)
  return wrapper
}

afterEach(() => {
  wrappers.splice(0).reverse().forEach(wrapper => wrapper.unmount())
  document.body.innerHTML = ''
})

describe('UsageStatsCards', () => {
  it('renders four desktop metrics with exact values and supporting details', () => {
    const wrapper = mountStats()
    const rail = wrapper.get('.usage-stat-rail')
    const metrics = rail.findAll('.usage-stat')

    expect(rail.element.tagName).toBe('SECTION')
    expect(rail.attributes('aria-label')).toBe('Usage summary')
    expect(metrics).toHaveLength(4)
    expect(metrics.map(metric => metric.get('.usage-stat__label').text())).toEqual([
      'Total Requests',
      'Total Tokens',
      'Total Cost',
      'Avg Duration',
    ])
    expect(metrics.map(metric => metric.get('.usage-stat__value').text())).toEqual([
      '987',
      '1.25M',
      '$3.46',
      '1.25s',
    ])

    expect(metrics[0].get('.usage-stat__meta').text()).toBe('in selected range')
    expect(metrics[1].get('.usage-stat__meta').text()).toContain('In 987.65K')
    expect(metrics[1].get('.usage-stat__meta').text()).toContain('Out 222.22K')
    expect(metrics[2].get('.usage-stat__meta').text()).toContain('Cost $2.35')
    expect(metrics[2].get('.usage-stat__meta').text()).toContain('Standard $4.57')
    expect(metrics[3].get('.usage-stat__meta').text()).toBe('Average per request')
  })

  it('exposes exact token and cost details through focusable, described metric tooltips', () => {
    const wrapper = mountStats()
    const describedMetrics = wrapper.findAll('.usage-stat[aria-describedby]')

    expect(describedMetrics).toHaveLength(2)
    for (const metric of describedMetrics) {
      const tooltipId = metric.attributes('aria-describedby')
      const tooltip = document.getElementById(tooltipId ?? '')

      expect(metric.attributes('tabindex')).toBe('0')
      expect(metric.attributes('aria-label')).toBeTruthy()
      expect(tooltipId).toBeTruthy()
      expect(tooltip?.getAttribute('role')).toBe('tooltip')
      expect(tooltip?.getAttribute('data-ui-portal')).toBe('usage-stat-tooltip')
    }

    const [tokenMetric, costMetric] = describedMetrics
    const tokenTooltip = document.getElementById(
      tokenMetric?.attributes('aria-describedby') ?? '',
    )
    expect(tokenTooltip?.textContent).toContain('Token details')
    expect(tokenTooltip?.textContent).toContain('Total Tokens: 1,245,678')
    expect(tokenTooltip?.textContent).toContain('In: 987,654')
    expect(tokenTooltip?.textContent).toContain('Out: 222,222')
    expect(tokenTooltip?.textContent).toContain('Cache: 35,000')
    expect(tokenTooltip?.textContent).toContain('Cache Creation: 12,345')
    expect(tokenTooltip?.textContent).toContain('Cache Read: 22,655')

    const costTooltip = document.getElementById(
      costMetric?.attributes('aria-describedby') ?? '',
    )
    expect(costTooltip?.textContent).toContain('Cost details')
    expect(costTooltip?.textContent).toContain('Total Cost: 3.4568')
    expect(costTooltip?.textContent).toContain('Cost: $2.3457')
    expect(costTooltip?.textContent).toContain('Standard: $4.5679')
  })

  it('keeps the compact metrics in one horizontal rail on narrow screens', () => {
    expect(componentSource).not.toContain('grid grid-cols-2 gap-4 lg:grid-cols-4')
    expect(componentSource).not.toContain(
      'grid-template-columns: repeat(2, minmax(0, 1fr));',
    )
    expect(componentSource).not.toContain('grid-template-columns: minmax(0, 1fr);')
    expect(componentSource).toContain('overflow-x: auto;')
    expect(componentSource).toContain('<Teleport to="body">')
    expect(componentSource).toContain('position: fixed;')
  })

  it('shows actual cost as points while preserving account and standard USD', () => {
    const wrapper = mountStats({ creditMode: true })

    expect(wrapper.get('[data-testid="credit-amount-value"]').text()).toBe('3.46')
    expect(wrapper.text()).toContain('Cost $2.35')
    expect(wrapper.text()).toContain('Standard $4.57')
  })
})

import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import DashboardTopUsers from '../DashboardTopUsers.vue'
import type { UserSpendingRankingItem, UserUsageTrendPoint } from '@/types'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
      locale: { value: 'en-US' },
    }),
  }
})

const items: UserSpendingRankingItem[] = [
  {
    user_id: 7,
    email: 'operator@example.com',
    username: 'Operator',
    main_model: 'claude-sonnet-4',
    actual_cost: 12.3456,
    requests: 42,
    tokens: 2_450_000,
  },
]

const trend: UserUsageTrendPoint[] = Array.from({ length: 8 }, (_, index) => ({
  date: `2026-07-20 ${String(index).padStart(2, '0')}:00`,
  user_id: 7,
  email: 'operator@example.com',
  username: 'Operator',
  requests: index + 1,
  tokens: (index + 1) * 100,
  cost: index + 1,
  actual_cost: (index + 1) / 2,
}))

const mountComponent = (props: Record<string, unknown> = {}) => mount(DashboardTopUsers, {
  props: {
    items,
    trend,
    ...props,
  },
  global: {
    stubs: {
      Icon: true,
      CreditAmount: {
        props: ['value', 'iconSize'],
        template: '<span data-testid="credit-amount" :data-icon-size="iconSize">{{ value }}</span>',
      },
    },
  },
})

describe('DashboardTopUsers', () => {
  it('renders real ranking fields and the latest seven trend buckets', async () => {
    const wrapper = mountComponent({ granularity: 'hour' })

    expect(wrapper.get('tbody').findAll('tr')).toHaveLength(1)
    expect(wrapper.text()).toContain('Operator')
    expect(wrapper.text()).toContain('operator@example.com')
    expect(wrapper.text()).toContain('claude-sonnet-4')
    expect(wrapper.text()).toContain('2.5M')
    const actualCost = wrapper.get('[data-testid="credit-amount"]')
    expect(actualCost.text()).toBe('12.3456')
    expect(actualCost.attributes('data-icon-size')).toBe('xs')
    expect(wrapper.text()).not.toContain('$12.3456')
    expect(wrapper.get('.top-users-bars').findAll('span')).toHaveLength(7)

    await wrapper.get('.top-users-user').trigger('click')
    expect(wrapper.emitted('select')).toEqual([[items[0]]])
  })

  it('fills missing periods with a zero-height baseline instead of compressing time', () => {
    const sparseTrend = [trend[0], trend[2], trend[7]]
    const wrapper = mountComponent({ trend: sparseTrend, granularity: 'hour' })
    const bars = wrapper.get('.top-users-bars').findAll('span')

    expect(bars).toHaveLength(7)
    expect(bars[0].attributes('style')).toContain('height: 0%')
    expect(bars[6].attributes('style')).toContain('height: 100%')
  })

  it('exposes full-detail and retry business actions', async () => {
    const wrapper = mountComponent({ error: true })

    await wrapper.get('.top-users-view-all').trigger('click')
    await wrapper.get('.top-users-state button').trigger('click')

    expect(wrapper.emitted('view-all')).toHaveLength(1)
    expect(wrapper.emitted('retry')).toHaveLength(1)
  })

  it('has explicit loading and empty states', () => {
    const loading = mountComponent({ loading: true })
    expect(loading.get('[aria-busy="true"]')).toBeTruthy()
    expect(loading.findAll('.top-users-skeleton-row')).toHaveLength(6)

    const empty = mountComponent({ items: [], trend: [] })
    expect(empty.get('.top-users-empty').text()).toContain('admin.dashboard.noUsageRecords')
  })
})

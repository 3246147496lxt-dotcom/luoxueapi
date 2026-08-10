import { describe, expect, it, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'

import UserTokenRanking from '../UserTokenRanking.vue'

const getUserBreakdown = vi.fn()

vi.mock('@/api/admin/dashboard', () => ({
  getUserBreakdown: (...args: unknown[]) => getUserBreakdown(...args),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const item = (id: number, tokens: number, overrides: Record<string, unknown> = {}) => ({
  user_id: id,
  email: `u${id}@test.com`,
  username: '',
  requests: 1,
  input_tokens: tokens,
  output_tokens: 0,
  cache_tokens: 0,
  total_tokens: tokens,
  cost: 0.6,
  actual_cost: 0.5,
  account_cost: 0.4,
  ...overrides,
})

const mountRanking = (props: Record<string, unknown> = {}) =>
  mount(UserTokenRanking, {
    props: {
      startDate: '2026-07-01',
      endDate: '2026-07-08',
      filters: {},
      ...props,
    },
    global: { stubs: { Select: true, LoadingSpinner: true } },
  })

describe('UserTokenRanking', () => {
  beforeEach(() => {
    getUserBreakdown.mockReset()
    getUserBreakdown.mockResolvedValue({ users: [item(1, 100), item(2, 50)] })
  })

  it('loads on mount with shared filters and emits select-user with id + email on row click', async () => {
    const wrapper = mountRanking({ filters: { group_id: 3 }, model: 'claude-fable-5' })
    await flushPromises()

    expect(getUserBreakdown).toHaveBeenCalledWith(expect.objectContaining({
      group_id: 3,
      model: 'claude-fable-5',
      start_date: '2026-07-01',
      end_date: '2026-07-08',
      sort_by: 'total_tokens',
      limit: 50,
    }))

    const rows = wrapper.findAll('tbody tr')
    expect(rows).toHaveLength(2)
    expect(wrapper.findAll('[data-testid="credit-amount-value"]').map((value) => value.text()))
      .toEqual(['0.5000', '0.5000'])
    expect(wrapper.text()).not.toContain('$0.5000')

    await rows[0].trigger('click')
    expect(wrapper.emitted('select-user')![0]).toEqual([1, 'u1@test.com'])
  })

  it('reloads when shared filters change', async () => {
    const wrapper = mountRanking()
    await flushPromises()
    expect(getUserBreakdown).toHaveBeenCalledTimes(1)

    await wrapper.setProps({ filters: { user_id: 9 } })
    await flushPromises()

    expect(getUserBreakdown).toHaveBeenCalledTimes(2)
    expect(getUserBreakdown).toHaveBeenLastCalledWith(expect.objectContaining({ user_id: 9 }))
  })

  it('uses the requested initial sort for the first ranking request', async () => {
    mountRanking({ initialSortBy: 'actual_cost' })
    await flushPromises()

    expect(getUserBreakdown).toHaveBeenCalledTimes(1)
    expect(getUserBreakdown).toHaveBeenCalledWith(expect.objectContaining({
      sort_by: 'actual_cost',
    }))
  })

  it('uses semantic buttons for sorting and user drill-down with aria-sort state', async () => {
    const wrapper = mountRanking()
    await flushPromises()

    const sortableHeaders = wrapper.findAll('thead th[aria-sort]')
    expect(sortableHeaders).toHaveLength(6)
    expect(sortableHeaders[4].attributes('aria-sort')).toBe('descending')
    expect(sortableHeaders[0].get('button').attributes('type')).toBe('button')

    await sortableHeaders[0].get('button').trigger('click')
    await flushPromises()

    expect(getUserBreakdown).toHaveBeenLastCalledWith(expect.objectContaining({
      sort_by: 'requests',
    }))
    expect(wrapper.findAll('thead th[aria-sort]')[0].attributes('aria-sort')).toBe('descending')

    const userButton = wrapper.get('tbody tr td:nth-child(2) button')
    expect(userButton.attributes('type')).toBe('button')
    expect(userButton.attributes('aria-label')).toContain('u1@test.com')
    await userButton.trigger('click')
    expect(wrapper.emitted('select-user')?.at(-1)).toEqual([1, 'u1@test.com'])
  })

  it('shows a distinct error state and retries without presenting the failure as empty data', async () => {
    getUserBreakdown.mockRejectedValueOnce(new Error('network unavailable'))
    const wrapper = mountRanking()
    await flushPromises()

    const alert = wrapper.get('[role="alert"]')
    expect(alert.text()).toContain('admin.usage.tokenRanking.loadFailed')
    expect(wrapper.text()).not.toContain('admin.dashboard.noDataAvailable')

    getUserBreakdown.mockResolvedValueOnce({ users: [item(3, 300)] })
    await alert.get('button').trigger('click')
    await flushPromises()

    expect(getUserBreakdown).toHaveBeenCalledTimes(2)
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    expect(wrapper.findAll('tbody tr')).toHaveLength(1)
    expect(wrapper.text()).toContain('u3@test.com')
  })

  it('keeps the empty state for a successful response with no users', async () => {
    getUserBreakdown.mockResolvedValueOnce({ users: [] })
    const wrapper = mountRanking()
    await flushPromises()

    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('admin.dashboard.noDataAvailable')
  })

  it('uses username as the primary identity and naturally falls back to email then user ID', async () => {
    getUserBreakdown.mockResolvedValueOnce({
      users: [
        item(1, 100, { username: 'Alice', email: 'alice@example.com' }),
        item(2, 50, { email: 'bob@example.com' }),
        item(3, 25, { username: '   ', email: '' }),
      ],
    })
    const wrapper = mountRanking()
    await flushPromises()

    const identityCells = wrapper.findAll('tbody tr td:nth-child(2)')
    expect(identityCells[0].get('button').findAll('span').map((node) => node.text())).toEqual([
      'Alice',
      'alice@example.com',
    ])
    expect(identityCells[1].get('button').findAll('span').map((node) => node.text())).toEqual([
      'bob@example.com',
    ])
    expect(identityCells[2].get('button').findAll('span').map((node) => node.text())).toEqual([
      'User #3',
    ])
  })
})

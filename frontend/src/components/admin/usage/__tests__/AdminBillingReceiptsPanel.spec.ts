import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import AdminBillingReceiptsPanel from '../AdminBillingReceiptsPanel.vue'

const { listBillingReceipts, searchUsers, showError, routeQuery } = vi.hoisted(() => ({
  listBillingReceipts: vi.fn(),
  searchUsers: vi.fn(),
  showError: vi.fn(),
  routeQuery: {} as Record<string, string | undefined>,
}))

vi.mock('@/api/admin/usage', () => ({
  adminUsageAPI: {
    listBillingReceipts,
    searchUsers,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
  }),
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({
    query: routeQuery,
  }),
}))

vi.mock('@/utils/format', () => ({
  formatDateTime: (value: string) => value,
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

const DataTableStub = {
  props: ['data', 'loading', 'error'],
  template: `
    <div>
      <div v-for="row in data" :key="row.receipt_id" data-test="receipt-row">
        <slot name="cell-user" :row="row" />
        <slot name="cell-receipt_id" :row="row" />
        <slot name="cell-model" :row="row" />
        <slot name="cell-tokens" :row="row" />
        <slot name="cell-cost" :row="row" />
        <slot name="cell-balance" :row="row" />
        <slot name="cell-status" :row="row" />
        <slot name="cell-created_at" :row="row" :value="row.created_at" />
      </div>
      <div v-if="error" data-test="load-error">{{ error }}</div>
    </div>
  `,
}

const CreditAmountStub = {
  props: ['value'],
  template: '<span data-test="credit-amount">{{ value }}</span>',
}

const receipt = {
  id: 21,
  receipt_id: 'rcpt-chat-21',
  request_id: 'req-chat-21',
  user_id: 8,
  user_email: 'member@example.com',
  source: 'web_chat',
  requested_model: 'gpt-5.5',
  actual_model: 'gpt-5.5-2026-07-01',
  model: 'gpt-5.5',
  tokens: {
    input_tokens: 1200,
    output_tokens: 340,
    cache_tokens: 128,
  },
  gross_cost: 0.024,
  charged_amount: 0.02,
  balance_before: 4.5,
  balance_after: 4.48,
  status: 'charged',
  failure_reason: null,
  created_at: '2026-07-25T01:02:03Z',
}

function mountPanel() {
  return mount(AdminBillingReceiptsPanel, {
    props: {
      startDate: '2026-07-24',
      endDate: '2026-07-25',
    },
    global: {
      stubs: {
        DataTable: DataTableStub,
        Pagination: true,
        Select: true,
        CreditAmount: CreditAmountStub,
        EmptyState: true,
        Icon: true,
      },
    },
  })
}

describe('AdminBillingReceiptsPanel', () => {
  beforeEach(() => {
    listBillingReceipts.mockReset()
    searchUsers.mockReset()
    showError.mockReset()
    Object.keys(routeQuery).forEach((key) => delete routeQuery[key])
    listBillingReceipts.mockResolvedValue({
      items: [receipt],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })
  })

  it('loads read-only Web Chat receipts and renders charge facts', async () => {
    const wrapper = mountPanel()
    await flushPromises()

    expect(listBillingReceipts).toHaveBeenCalledWith(expect.objectContaining({
      source: 'web_chat',
      start_date: '2026-07-24',
      end_date: '2026-07-25',
    }), expect.objectContaining({
      signal: expect.any(AbortSignal),
    }))

    const row = wrapper.get('[data-test="receipt-row"]')
    expect(row.text()).toContain('member@example.com')
    expect(row.text()).toContain('rcpt-chat-21')
    expect(row.text()).toContain('gpt-5.5')
    expect(row.text()).toContain('gpt-5.5-2026-07-01')
    expect(row.text()).toContain('1,200')
    expect(row.text()).toContain('340')
    expect(row.text()).toContain('128')
    expect(row.text()).toContain('0.020000')
    expect(row.text()).toContain('$0.024000')
    expect(row.text()).toContain('4.500000')
    expect(row.text()).toContain('4.480000')
    expect(row.findAll('[data-test="credit-amount"]')).toHaveLength(3)
    expect(row.text()).toContain('admin.usage.billingReceipts.statuses.charged')
  })

  it('applies receipt deep-link filters to the first request', async () => {
    Object.assign(routeQuery, {
      receipt_id: 'rcpt-deep-link',
      user_id: '12',
      model: 'gpt-5.5',
      status: 'charged',
    })

    mountPanel()
    await flushPromises()

    expect(listBillingReceipts).toHaveBeenCalledWith(expect.objectContaining({
      receipt_id: 'rcpt-deep-link',
      user_id: 12,
      model: 'gpt-5.5',
      status: 'charged',
    }), expect.anything())
  })

  it('shows a stable load error without mutating billing state', async () => {
    listBillingReceipts.mockRejectedValueOnce(new Error('network'))

    const wrapper = mountPanel()
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('admin.usage.billingReceipts.loadFailed')
    expect(wrapper.get('[data-test="load-error"]').text()).toBe(
      'admin.usage.billingReceipts.loadFailed'
    )
  })
})

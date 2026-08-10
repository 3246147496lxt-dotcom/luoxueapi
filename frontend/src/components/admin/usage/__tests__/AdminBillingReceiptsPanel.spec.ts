import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import AdminBillingReceiptsPanel from '../AdminBillingReceiptsPanel.vue'

const { listBillingReceipts, searchUsers, showError, copyToClipboard, routeQuery } = vi.hoisted(() => ({
  listBillingReceipts: vi.fn(),
  searchUsers: vi.fn(),
  showError: vi.fn(),
  copyToClipboard: vi.fn(),
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

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard,
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
  props: ['data', 'loading', 'error', 'clickableRows', 'selectedRowKey', 'rowKey'],
  emits: ['rowClick'],
  template: `
    <div>
      <div
        v-for="row in data"
        :key="row.row_key"
        data-test="receipt-row"
        :data-row-key="row.row_key"
        :data-selected="String(selectedRowKey) === String(row.row_key) ? 'true' : 'false'"
        @click="$emit('rowClick', row)"
      >
        <slot name="cell-user" :row="row" />
        <slot name="cell-receipt_id" :row="row" />
        <slot name="cell-model" :row="row" />
        <slot name="cell-tokens" :row="row" />
        <slot name="cell-cost" :row="row" />
        <slot name="cell-balance" :row="row" />
        <slot name="cell-status" :row="row" />
        <slot name="cell-created_at" :row="row" :value="row.created_at" />
        <slot name="cell-actions" :row="row" />
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
  row_key: 'receipt:21',
  receipt_id: '21',
  request_id: 'client:22222222-2222-4222-8222-222222222222',
  user_id: 8,
  user: null,
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
        Teleport: true,
        Transition: false,
      },
    },
  })
}

describe('AdminBillingReceiptsPanel', () => {
  beforeEach(() => {
    listBillingReceipts.mockReset()
    searchUsers.mockReset()
    showError.mockReset()
    copyToClipboard.mockReset()
    copyToClipboard.mockResolvedValue(true)
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
    expect(wrapper.text()).toContain('admin.usage.billingReceipts.readOnly')
    expect(row.text()).toContain('member@example.com')
    expect(row.text()).toContain('21')
    expect(row.text()).toContain('client:22222222-2222-4222-8222-222222222222')
    expect(row.text()).not.toContain('member-name')
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

  it('opens a read-only drawer with the complete settlement snapshot and copies identifiers', async () => {
    listBillingReceipts.mockResolvedValueOnce({
      items: [{
        ...receipt,
        status: 'failed',
        overdraft: true,
        failure_code: 'billing_balance_rejected',
        failure_reason: 'Balance validation rejected the charge',
        tokens: {
          ...receipt.tokens,
          cache_read_tokens: 96,
          cache_creation_tokens: 32,
        },
      }],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })

    const wrapper = mountPanel()
    await flushPromises()
    await wrapper.get('[data-test="receipt-row"]').trigger('click')
    await flushPromises()

    const drawer = wrapper.get('[data-test="billing-receipt-drawer"]')
    expect(drawer.text()).toContain('admin.usage.billingReceipts.drawer.title')
    expect(drawer.text()).toContain('member@example.com')
    expect(drawer.text()).toContain('21')
    expect(drawer.text()).toContain('client:22222222-2222-4222-8222-222222222222')
    expect(drawer.text()).toContain('gpt-5.5')
    expect(drawer.text()).toContain('gpt-5.5-2026-07-01')
    expect(drawer.text()).toContain('1,200')
    expect(drawer.text()).toContain('340')
    expect(drawer.text()).toContain('96')
    expect(drawer.text()).toContain('32')
    expect(drawer.text()).toContain('0.024000')
    expect(drawer.text()).toContain('0.020000')
    expect(drawer.text()).toContain('4.500000')
    expect(drawer.text()).toContain('4.480000')
    expect(drawer.text()).toContain('billing_balance_rejected')
    expect(drawer.text()).toContain('Balance validation rejected the charge')
    expect(drawer.text()).toContain('admin.usage.billingReceipts.drawer.readOnlyNote')

    await wrapper.get('[data-test="drawer-copy-receipt-id"]').trigger('click')
    expect(copyToClipboard).toHaveBeenCalledWith(
      '21',
      'admin.usage.billingReceipts.copySuccess'
    )

    await wrapper.get('[data-test="drawer-copy-request-id"]').trigger('click')
    expect(copyToClipboard).toHaveBeenCalledWith(
      'client:22222222-2222-4222-8222-222222222222',
      'admin.usage.billingReceipts.copySuccess'
    )

    await wrapper.get('[data-test="billing-receipt-drawer-close"]').trigger('click')
    expect(wrapper.find('[data-test="billing-receipt-drawer"]').exists()).toBe(false)
  })

  it('renders receipt-less attempts as distinct request audit rows', async () => {
    listBillingReceipts.mockResolvedValueOnce({
      items: [
        {
          ...receipt,
          id: 0,
          row_key: 'request:client:attempt-one',
          receipt_id: '',
          request_id: 'client:attempt-one',
          status: 'failed',
        },
        {
          ...receipt,
          id: 0,
          row_key: 'request:client:attempt-two',
          receipt_id: '',
          request_id: 'client:attempt-two',
          status: 'pending',
        },
      ],
      total: 2,
      page: 1,
      page_size: 20,
      pages: 1,
    })

    const wrapper = mountPanel()
    await flushPromises()

    const rows = wrapper.findAll('[data-test="receipt-row"]')
    expect(rows).toHaveLength(2)
    expect(rows.map((row) => row.attributes('data-row-key'))).toEqual([
      'request:client:attempt-one',
      'request:client:attempt-two',
    ])
    expect(wrapper.text()).toContain('admin.usage.billingReceipts.identifiers.receiptNotGenerated')
    expect(wrapper.findAll('[title="admin.usage.billingReceipts.copyReceiptId"]')).toHaveLength(0)
    expect(wrapper.findAll('[title="admin.usage.billingReceipts.copyRequestId"]')).toHaveLength(2)

    await rows[1].trigger('click')
    await flushPromises()

    expect(wrapper.findAll('[data-test="receipt-row"]')[0].attributes('data-selected')).toBe('false')
    expect(wrapper.findAll('[data-test="receipt-row"]')[1].attributes('data-selected')).toBe('true')
    const drawer = wrapper.get('[data-test="billing-receipt-drawer"]')
    expect(drawer.text()).toContain('client:attempt-two')
    expect(drawer.text()).toContain('admin.usage.billingReceipts.identifiers.receiptNotGenerated')
    expect(drawer.find('[data-test="drawer-copy-receipt-id"]').exists()).toBe(false)
  })

  it('applies request-id deep-link filters using the compatible API query key', async () => {
    Object.assign(routeQuery, {
      request_id: 'req-deep-link',
      user_id: '12',
      model: 'gpt-5.5',
      status: 'charged',
    })

    mountPanel()
    await flushPromises()

    expect(listBillingReceipts).toHaveBeenCalledWith(expect.objectContaining({
      receipt_id: 'req-deep-link',
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

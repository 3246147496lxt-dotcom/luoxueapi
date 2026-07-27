import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import AdminAffiliateRecordsTable from '../AdminAffiliateRecordsTable.vue'

const api = vi.hoisted(() => ({
  listInviteRecords: vi.fn(),
  listRebateRecords: vi.fn(),
  listTransferRecords: vi.fn(),
  getUserOverview: vi.fn(),
}))

vi.mock('@/api/admin/affiliates', () => ({
  affiliatesAPI: api,
  default: api,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: vi.fn() }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

vi.mock('@/i18n', () => ({
  getLocale: () => 'en-US',
  i18n: { global: { te: () => true, t: (key: string) => key } },
}))

const DataTableStub = {
  props: ['columns', 'data'],
  template: `
    <div v-if="data[0]" data-testid="table-row">
      <div v-for="column in columns" :key="column.key">
        <slot :name="'cell-' + column.key" :row="data[0]" />
      </div>
    </div>
  `,
}

async function mountTable(type: 'invites' | 'rebates' | 'transfers') {
  const wrapper = mount(AdminAffiliateRecordsTable, {
    props: { type },
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: {
          template: '<div><slot name="header" /><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>',
        },
        AdminPageHeader: true,
        BaseDialog: { props: ['show'], template: '<div v-if="show"><slot /></div>' },
        DataTable: DataTableStub,
        Icon: true,
        OrderStatusBadge: true,
        Pagination: true,
      },
    },
  })
  await flushPromises()
  return wrapper
}

describe('AdminAffiliateRecordsTable billing units', () => {
  beforeEach(() => {
    Object.values(api).forEach((mock) => mock.mockReset())
    api.listInviteRecords.mockResolvedValue({ items: [], total: 0 })
    api.listRebateRecords.mockResolvedValue({ items: [], total: 0 })
    api.listTransferRecords.mockResolvedValue({ items: [], total: 0 })
  })

  it('renders accumulated rebates as snowflake credits', async () => {
    api.listInviteRecords.mockResolvedValue({
      items: [{
        inviter_id: 1,
        inviter_email: 'inviter@example.com',
        inviter_username: 'inviter',
        invitee_id: 2,
        invitee_email: 'invitee@example.com',
        invitee_username: 'invitee',
        aff_code: 'SNOW',
        total_rebate: 12.5,
        created_at: '2026-07-27T00:00:00Z',
      }],
      total: 1,
    })

    const wrapper = await mountTable('invites')

    expect(wrapper.get('[data-testid="credit-amount-value"]').text()).toBe('12.50')
    expect(wrapper.text()).not.toContain('$12.50')
  })

  it('separates credited amounts from the payment channel currency', async () => {
    api.listRebateRecords.mockResolvedValue({
      items: [{
        order_id: 3,
        out_trade_no: 'order-3',
        inviter_id: 1,
        inviter_email: 'inviter@example.com',
        inviter_username: 'inviter',
        invitee_id: 2,
        invitee_email: 'invitee@example.com',
        invitee_username: 'invitee',
        order_type: 'balance',
        order_amount: 100,
        pay_amount: 80,
        currency: 'HKD',
        rebate_amount: 10,
        payment_type: 'stripe',
        order_status: 'COMPLETED',
        created_at: '2026-07-27T00:00:00Z',
      }],
      total: 1,
    })

    const wrapper = await mountTable('rebates')

    expect(wrapper.findAll('[data-testid="credit-amount-value"]').map((node) => node.text()))
      .toEqual(['100.00', '10.00'])
    expect(wrapper.text()).toMatch(/HK\$80\.00|HKD\s*80\.00/)
    expect(wrapper.text()).not.toContain('¥80.00')
  })

  it('renders transfer and balance snapshots as snowflake credits', async () => {
    api.listTransferRecords.mockResolvedValue({
      items: [{
        ledger_id: 4,
        user_id: 1,
        user_email: 'inviter@example.com',
        username: 'inviter',
        amount: 5,
        balance_after: 25,
        available_quota_after: 6,
        frozen_quota_after: 2,
        history_quota_after: 30,
        snapshot_available: true,
        created_at: '2026-07-27T00:00:00Z',
      }],
      total: 1,
    })

    const wrapper = await mountTable('transfers')

    expect(wrapper.findAll('[data-testid="credit-amount-value"]').map((node) => node.text()))
      .toEqual(['5.00', '25.00', '6.00', '2.00', '30.00'])
    expect(wrapper.text()).not.toContain('$5.00')
  })
})

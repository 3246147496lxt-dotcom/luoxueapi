import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import UserBalanceHistoryModal from '../UserBalanceHistoryModal.vue'
import type { AdminUser } from '@/types'

const { getUserBalanceHistory } = vi.hoisted(() => ({
  getUserBalanceHistory: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    users: { getUserBalanceHistory }
  }
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

vi.mock('@/i18n', () => ({
  getLocale: () => 'en-US',
  i18n: {
    global: {
      te: () => true,
      t: (key: string) => key
    }
  }
}))

const user = {
  id: 17,
  email: 'snow@example.com',
  username: 'snow',
  balance: 42.5,
  notes: '',
  created_at: '2026-07-20T00:00:00Z'
} as AdminUser

describe('UserBalanceHistoryModal', () => {
  beforeEach(() => {
    getUserBalanceHistory.mockReset()
    getUserBalanceHistory.mockResolvedValue({
      items: [
        {
          id: 1,
          code: 'balance-code',
          type: 'balance',
          value: 12.5,
          status: 'used',
          used_by: 17,
          used_at: '2026-07-26T00:00:00Z',
          created_at: '2026-07-26T00:00:00Z',
          group_id: null,
          validity_days: 0,
          notes: ''
        },
        {
          id: 2,
          code: 'admin-balance-code',
          type: 'admin_balance',
          value: -2,
          status: 'used',
          used_by: 17,
          used_at: '2026-07-25T00:00:00Z',
          created_at: '2026-07-25T00:00:00Z',
          group_id: null,
          validity_days: 0,
          notes: 'adjustment'
        },
        {
          id: 3,
          code: 'concurrency-code',
          type: 'concurrency',
          value: 3,
          status: 'used',
          used_by: 17,
          used_at: '2026-07-24T00:00:00Z',
          created_at: '2026-07-24T00:00:00Z',
          group_id: null,
          validity_days: 0,
          notes: ''
        }
      ],
      total: 3,
      page: 1,
      page_size: 15,
      total_recharged: 100
    })
  })

  it('renders balances as snowflake credits while leaving concurrency as a count', async () => {
    const wrapper = mount(UserBalanceHistoryModal, {
      props: { show: false, user, hideActions: true },
      global: {
        stubs: {
          BaseDialog: {
            props: ['show'],
            template: '<div v-if="show"><slot /></div>'
          },
          Select: {
            props: ['modelValue', 'options'],
            template: '<div data-testid="select-stub" />'
          }
        }
      }
    })

    await wrapper.setProps({ show: true })
    await flushPromises()

    expect(getUserBalanceHistory).toHaveBeenCalledWith(17, 1, 15, undefined)
    expect(wrapper.findAll('[data-testid="credit-amount-value"]').map(node => node.text())).toEqual([
      '42.50',
      '100.00',
      '+12.50',
      '-2.00'
    ])
    expect(wrapper.findAll('[data-testid="snowflake-credit-icon"]')).toHaveLength(6)
    expect(wrapper.text()).toContain('+3')
    expect(wrapper.text()).not.toContain('$42.50')
    expect(wrapper.text()).not.toContain('$100.00')
    expect(wrapper.text()).not.toContain('$12.50')
    expect(wrapper.text()).not.toContain('$2.00')
  })
})

import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import AffiliateView from '../AffiliateView.vue'

const getAffiliateDetail = vi.hoisted(() => vi.fn())
const transferAffiliateQuota = vi.hoisted(() => vi.fn())

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

vi.mock('@/api/user', () => ({
  default: {
    getAffiliateDetail,
    transferAffiliateQuota,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: vi.fn(), showSuccess: vi.fn() }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ refreshUser: vi.fn() }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copyToClipboard: vi.fn() }),
}))

describe('AffiliateView credit display', () => {
  it('uses snowflake credits for rebate balances and invitee rebates', async () => {
    getAffiliateDetail.mockResolvedValue({
      aff_code: 'SNOW-01',
      aff_count: 1,
      aff_quota: 12.5,
      aff_history_quota: 30,
      aff_frozen_quota: 2,
      effective_rebate_rate_percent: 10,
      invitees: [{
        user_id: 9,
        email: 'snow@example.com',
        username: 'snow',
        total_rebate: 6.25,
        created_at: '2026-07-19T00:00:00Z',
      }],
    })

    const wrapper = mount(AffiliateView, {
      global: {
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
          Icon: true,
        },
      },
    })
    await flushPromises()

    const credits = wrapper.findAll('[data-testid="credit-amount"]')
    expect(credits).toHaveLength(4)
    expect(credits.map((item) => item.get('[data-testid="credit-amount-value"]').text())).toEqual([
      '12.50',
      '30.00',
      '2.00',
      '6.25',
    ])
    expect(wrapper.text()).not.toContain('$')
  })
})

import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import PricingPlanCard from '../PricingPlanCard.vue'
import type { SubscriptionPlan } from '@/types/payment'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    locale: { value: 'en-US' },
    t: (key: string, values?: Record<string, unknown>) => (
      values ? `${key}:${Object.values(values).join('|')}` : key
    ),
  }),
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    cachedPublicSettings: {
      server_utc_offset: '+08:00',
    },
  }),
}))

function planFixture(overrides: Partial<SubscriptionPlan> = {}): SubscriptionPlan {
  return {
    id: 7,
    group_id: 3,
    group_name: 'OpenAI',
    group_platform: 'antigravity',
    name: 'Starter',
    description: 'For individual API usage',
    price: 9.99,
    validity_days: 30,
    validity_unit: 'day',
    features: ['Priority access'],
    for_sale: true,
    sort_order: 1,
    rate_multiplier: 1,
    ...overrides,
  }
}

describe('PricingPlanCard', () => {
  it('uses neutral renewal wording and exposes real peak-rate and model-scope facts', () => {
    const wrapper = mount(PricingPlanCard, {
      props: {
        plan: planFixture({
          peak_rate_enabled: true,
          peak_start: '14:00',
          peak_end: '18:00',
          peak_rate_multiplier: 2,
          supported_model_scopes: [
            'claude',
            'gemini_text',
            'gemini_image',
            'custom_scope',
            'claude',
          ],
        }),
        renewal: true,
      },
      global: {
        stubs: {
          Icon: true,
        },
      },
    })

    expect(wrapper.get('.pricing-plan-card__status').text()).toBe('pricing.renewalOption')
    expect(wrapper.text()).not.toContain('pricing.currentSubscription')
    expect(wrapper.text()).toContain(
      'pricing.peakRateWindow:14:00-18:00 ×2 (UTC+08:00)',
    )
    expect(wrapper.text()).toContain(
      'pricing.modelScopes:Claude, Gemini, Imagen, custom_scope',
    )
    expect(wrapper.get('.pricing-plan-card__action').attributes('aria-label')).toContain('Starter')
    expect(wrapper.get('.pricing-plan-card__action').text()).toBe('pricing.renewPlan')
  })

  it('names the choose action with the plan and omits unproven facts', () => {
    const wrapper = mount(PricingPlanCard, {
      props: {
        plan: planFixture(),
        renewal: false,
      },
      global: {
        stubs: {
          Icon: true,
        },
      },
    })

    expect(wrapper.find('.pricing-plan-card__status').exists()).toBe(false)
    expect(wrapper.get('.pricing-plan-card__action').attributes('aria-label')).toContain('Starter')
    expect(wrapper.get('.pricing-plan-card__action').text()).toBe('pricing.choosePlan')
    expect(wrapper.text()).not.toContain('pricing.peakRateWindow')
    expect(wrapper.text()).not.toContain('pricing.modelScopes')
  })
})

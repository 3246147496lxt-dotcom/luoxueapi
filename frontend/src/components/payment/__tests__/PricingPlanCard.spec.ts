import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import PricingPlanCard from '../PricingPlanCard.vue'
import enMisc from '@/i18n/locales/en/misc'
import zhMisc from '@/i18n/locales/zh/misc'
import type { SubscriptionPlan } from '@/types/payment'

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()

  return {
    ...actual,
    useI18n: () => ({
      locale: { value: 'en-US' },
      t: (key: string, values?: Record<string, unknown>) => (
        values ? `${key}:${Object.values(values).join('|')}` : key
      ),
    }),
  }
})

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
    expect(wrapper.find('.pricing-plan-card__group').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('OpenAI')
    expect(wrapper.get('.pricing-plan-card__action').attributes('aria-label')).toContain('Starter')
    expect(wrapper.get('.pricing-plan-card__action').text()).toBe('pricing.choosePlan')
    expect(wrapper.text()).not.toContain('pricing.peakRateWindow')
    expect(wrapper.text()).not.toContain('pricing.modelScopes')
  })

  it('renders plan facts before four quota terms in one vertical feature list', () => {
    const wrapper = mount(PricingPlanCard, {
      props: {
        plan: planFixture({
          daily_limit_usd: 0,
          weekly_limit_usd: 200,
          monthly_limit_usd: 0,
          peak_rate_enabled: true,
          peak_start: '14:00',
          peak_end: '18:00',
          peak_rate_multiplier: 2,
          supported_model_scopes: ['claude'],
        }),
        renewal: false,
      },
      global: {
        stubs: {
          Icon: true,
        },
      },
    })

    const factsList = wrapper.get('.pricing-plan-card__facts')
    const rows = factsList.findAll('li')
    const metrics = factsList.findAll('[data-metric]')
    expect(factsList.element.tagName).toBe('UL')
    expect(rows.slice(0, 3).map((row) => row.text())).toEqual([
      'Priority access',
      'pricing.peakRateWindow:14:00-18:00 ×2 (UTC+08:00)',
      'pricing.modelScopes:Claude',
    ])
    expect(metrics).toHaveLength(4)
    expect(metrics.every((metric) => metric.element.tagName === 'LI')).toBe(true)
    expect(metrics.every((metric) => metric.element.parentElement === factsList.element)).toBe(true)
    expect(metrics.map((metric) => metric.attributes('data-metric'))).toEqual([
      'rate',
      'daily',
      'weekly',
      'monthly',
    ])
    expect(metrics.map((metric) => metric.text())).toEqual([
      'pricing.rateMultiplier:1',
      'pricing.dailyQuota:0.00',
      'pricing.weeklyQuota:200.00',
      'pricing.monthlyQuota:0.00',
    ])
    expect(wrapper.find('dl').exists()).toBe(false)
    expect(wrapper.find('.pricing-plan-card__metrics').exists()).toBe(false)
    expect(wrapper.findAllComponents({ name: 'CreditAmount' })).toHaveLength(0)

    const facts = factsList.text()
    expect(facts).toContain('Priority access')
    expect(facts).toContain('pricing.peakRateWindow')
    expect(facts).toContain('pricing.modelScopes')
    expect(facts).not.toContain('pricing.metricLabels.rate')
    expect(facts).not.toContain('pricing.metricLabels.daily')
  })

  it('keeps all four metrics when periodic quotas are unlimited', () => {
    const wrapper = mount(PricingPlanCard, {
      props: {
        plan: planFixture({
          daily_limit_usd: null,
          weekly_limit_usd: undefined,
          monthly_limit_usd: null,
          features: [],
        }),
        renewal: false,
      },
      global: {
        stubs: {
          Icon: true,
        },
      },
    })

    const facts = wrapper.get('.pricing-plan-card__facts')
    const metrics = facts.findAll('[data-metric]')
    expect(metrics).toHaveLength(4)
    expect(metrics.slice(1).every((metric) => (
      metric.text().includes('payment.planCard.unlimited')
    ))).toBe(true)
    expect(facts.findAll('li')).toHaveLength(4)
    expect(wrapper.text()).not.toContain('pricing.unlimitedQuota')
  })

  it('keeps the pricing metric labels explicit in both locales', () => {
    expect(zhMisc.pricing.metricLabels).toEqual({
      rate: '计费倍率',
      daily: '每日额度',
      weekly: '每周额度',
      monthly: '每月额度',
    })
    expect(enMisc.pricing.metricLabels).toEqual({
      rate: 'Billing rate',
      daily: 'Daily quota',
      weekly: 'Weekly quota',
      monthly: 'Monthly quota',
    })
  })
})

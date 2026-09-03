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

function planFixture(overrides: Partial<SubscriptionPlan> = {}): SubscriptionPlan {
  return {
    id: 7,
    group_id: 3,
    group_name: 'OpenAI',
    group_platform: 'openai',
    name: 'Standard',
    description: '日常 AI 工作',
    price: 99,
    currency: 'CNY',
    validity_days: 30,
    validity_unit: 'day',
    features: [
      '高频 AI 对话体验',
      '文档处理与内容创作',
      '更高使用额度',
      '支持更多 AI 模型',
    ],
    for_sale: true,
    sort_order: 1,
    rate_multiplier: 1,
    ...overrides,
  }
}

const global = {
  stubs: {
    Icon: true,
  },
}

describe('PricingPlanCard', () => {
  it('renders the approved featured anatomy and keeps its CTA copy unchanged', () => {
    const wrapper = mount(PricingPlanCard, {
      props: {
        plan: planFixture(),
        featured: true,
      },
      global,
    })

    expect(wrapper.classes()).toContain('pricing-plan-card--featured')
    expect(wrapper.get('.pricing-plan-card__badge').text()).toBe('pricing.recommended')
    expect(wrapper.get('.pricing-plan-card__description').text()).toBe('日常 AI 工作')
    expect(wrapper.get('.pricing-plan-card__price').text()).toBe('¥99.00/ pricing.validityDays:30')
    expect(wrapper.get('.pricing-plan-card__action').text()).toBe('pricing.choosePlan')
    expect(wrapper.get('.pricing-plan-card__action').attributes('aria-label')).toContain('Standard')
    expect(wrapper.get('.pricing-plan-card__details').text()).toContain('pricing.viewQuotaDetails')
  })

  it('keeps normal cards white-state only and does not add a recommendation badge', () => {
    const wrapper = mount(PricingPlanCard, {
      props: {
        plan: planFixture({ name: 'Try', price: 18, validity_days: 7 }),
      },
      global,
    })

    expect(wrapper.classes()).not.toContain('pricing-plan-card--featured')
    expect(wrapper.attributes('data-featured')).toBe('false')
    expect(wrapper.find('.pricing-plan-card__badge').exists()).toBe(false)
    expect(wrapper.get('.pricing-plan-card__action').text()).toBe('pricing.choosePlan')
  })

  it('shows exactly the first four unique user-facing benefits and no technical quota rows', () => {
    const wrapper = mount(PricingPlanCard, {
      props: {
        plan: planFixture({
          features: ['一', '二', '二', '三', '四', '五'],
          daily_limit_usd: 100,
          weekly_limit_usd: 200,
          monthly_limit_usd: 300,
          peak_rate_enabled: true,
          peak_start: '14:00',
          peak_end: '18:00',
          peak_rate_multiplier: 2,
          supported_model_scopes: ['claude'],
        }),
      },
      global,
    })

    expect(wrapper.findAll('.pricing-plan-card__facts li').map((row) => row.text())).toEqual([
      '一',
      '二',
      '三',
      '四',
    ])
    expect(wrapper.text()).not.toContain('pricing.rateMultiplier')
    expect(wrapper.text()).not.toContain('pricing.dailyQuota')
    expect(wrapper.text()).not.toContain('pricing.peakRateWindow')
  })

  it('emits the real plan from both approved card actions', async () => {
    const plan = planFixture()
    const wrapper = mount(PricingPlanCard, {
      props: { plan, featured: true },
      global,
    })

    await wrapper.get('.pricing-plan-card__action').trigger('click')
    await wrapper.get('.pricing-plan-card__details').trigger('click')

    expect(wrapper.emitted('select')).toEqual([[plan]])
    expect(wrapper.emitted('details')).toEqual([[plan]])
  })

  it('keeps the approved pricing labels in both locales', () => {
    expect(zhMisc.pricing.recommended).toBe('推荐选择')
    expect(zhMisc.pricing.viewQuotaDetails).toBe('查看额度详情')
    expect(zhMisc.pricing.tiers).toEqual({
      low: {
        name: '轻量级',
        description: '适合体验 AI、学习和日常轻任务',
      },
      mid: {
        name: '中量级',
        description: '适合高频办公、内容创作和专业个人用户',
      },
      high: {
        name: '高量级',
        description: '适合重度 AI 工作流、专业创作者和团队用户',
      },
    })
    expect(enMisc.pricing.recommended).toBe('Recommended')
    expect(enMisc.pricing.viewQuotaDetails).toBe('View quota details')
  })
})

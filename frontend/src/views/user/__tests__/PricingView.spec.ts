import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import PricingView from '../PricingView.vue'
import { paymentAPI } from '@/api/payment'
import type { CheckoutInfoResponse, SubscriptionPlan } from '@/types/payment'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      locale: { value: 'zh-CN' },
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/api/payment', () => ({
  paymentAPI: {
    getCheckoutInfo: vi.fn(),
    getConfig: vi.fn(),
    getPlans: vi.fn(),
    createOrder: vi.fn(),
    getOrder: vi.fn(),
  },
}))

function planFixture(overrides: Partial<SubscriptionPlan> = {}): SubscriptionPlan {
  return {
    id: 1,
    group_id: 1,
    group_name: 'OpenAI',
    group_platform: 'openai',
    name: 'Try',
    description: '体验 AI 能力',
    price: 18,
    currency: 'CNY',
    validity_days: 7,
    validity_unit: 'day',
    features: ['一', '二', '三', '四'],
    for_sale: true,
    sort_order: 1,
    ...overrides,
  }
}

function approvedPlans(): SubscriptionPlan[] {
  return [
    planFixture({ id: 6, group_id: 16, name: 'Max', price: 648 }),
    planFixture({ id: 3, group_id: 13, name: 'Basic', price: 49 }),
    planFixture({ id: 1, group_id: 11, name: 'Try', price: 18, validity_days: 7 }),
    planFixture({ id: 7, group_id: 17, name: 'Ultra', price: 1088 }),
    planFixture({ id: 4, group_id: 14, name: 'Plus', price: 188 }),
    planFixture({ id: 2, group_id: 12, name: 'Standard', price: 99 }),
    planFixture({ id: 5, group_id: 15, name: 'Pro', price: 328 }),
  ]
}

function checkoutInfoFixture(plans: SubscriptionPlan[] = []): CheckoutInfoResponse {
  return {
    methods: {},
    global_min: 0,
    global_max: 0,
    plans,
    balance_disabled: false,
    balance_recharge_multiplier: 1,
    subscription_usd_to_cny_rate: 0,
    recharge_fee_rate: 0,
    help_text: '',
    help_image_url: '',
    stripe_publishable_key: '',
  }
}

async function mountPricing(path = '/pricing', previousPath?: string) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/dashboard', component: { template: '<div />' } },
      { path: '/pricing', component: { template: '<div />' } },
      { path: '/purchase', component: { template: '<div />' } },
    ],
  })
  if (previousPath) await router.push(previousPath)
  await router.push(path)
  await router.isReady()

  const wrapper = shallowMount(PricingView, {
    global: {
      plugins: [router],
      stubs: {
        PricingFluidBackground: {
          props: ['tier'],
          template: '<canvas class="fluid-stub" :data-tier="tier" />',
        },
        PricingPlanCard: {
          props: ['plan', 'featured', 'current', 'actionLabel'],
          emits: ['select', 'details'],
          template: `
            <div
              class="plan-stub"
              :data-plan-id="String(plan.id)"
              :data-featured="String(featured)"
              :data-current="String(current)"
              :data-action-label="actionLabel"
            >
              <span>{{ plan.name }}</span>
              <button class="select-stub" @click="$emit('select', plan)">select</button>
              <button class="details-stub" @click="$emit('details', plan)">details</button>
            </div>
          `,
        },
      },
    },
  })
  await flushPromises()
  return { router, wrapper }
}

describe('PricingView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.mocked(paymentAPI.getCheckoutInfo).mockReset()
  })

  it('renders the approved persistent 3/3/1 panels in fixed visual order', async () => {
    vi.mocked(paymentAPI.getCheckoutInfo).mockResolvedValue({
      data: checkoutInfoFixture(approvedPlans()),
    } as never)

    const { wrapper } = await mountPricing()
    const low = wrapper.get('[data-testid="pricing-tier-panel-low"]')
    const mid = wrapper.get('[data-testid="pricing-tier-panel-mid"]')
    const high = wrapper.get('[data-testid="pricing-tier-panel-high"]')

    expect(wrapper.get('[data-testid="pricing-page"]').classes()).toContain('pricing-theme-light')
    expect(wrapper.get('[data-testid="pricing-page"]').attributes('data-tier')).toBe('low')
    expect(low.findAll('.plan-stub').map((card) => card.text())).toEqual([
      'Tryselectdetails',
      'Basicselectdetails',
      'Standardselectdetails',
    ])
    expect(low.findAll('.plan-stub').map((card) => card.attributes('data-plan-id'))).toEqual([
      '1',
      '3',
      '2',
    ])
    expect(mid.findAll('.plan-stub').map((card) => card.text())).toEqual([
      'Plusselectdetails',
      'Proselectdetails',
      'Maxselectdetails',
    ])
    expect(high.findAll('.plan-stub').map((card) => card.text())).toEqual([
      'Ultraselectdetails',
    ])
    expect(low.findAll('.plan-stub').map((card) => card.attributes('data-featured'))).toEqual([
      'false',
      'false',
      'true',
    ])
    expect(mid.findAll('.plan-stub').map((card) => card.attributes('data-featured'))).toEqual([
      'false',
      'true',
      'false',
    ])
    expect(high.get('.plan-stub').attributes('data-featured')).toBe('true')
    expect(mid.attributes('aria-hidden')).toBe('true')
    expect(low.attributes('aria-hidden')).toBe('false')
    expect(high.attributes('aria-hidden')).toBe('true')
  })

  it('keeps the tier heading hidden and exposes the lightweight back entry', async () => {
    vi.mocked(paymentAPI.getCheckoutInfo).mockResolvedValue({
      data: checkoutInfoFixture(approvedPlans()),
    } as never)

    const { router, wrapper } = await mountPricing()
    expect(wrapper.get('#pricing-tier-summary h2').attributes()).toHaveProperty('hidden')
    expect(wrapper.get('[data-testid="pricing-back"]').text()).toBe('common.back')

    await wrapper.get('[data-testid="pricing-back"]').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe('/dashboard')
  })

  it('returns to an existing same-origin route instead of using the dashboard fallback', async () => {
    vi.mocked(paymentAPI.getCheckoutInfo).mockResolvedValue({
      data: checkoutInfoFixture(approvedPlans()),
    } as never)

    const { router, wrapper } = await mountPricing(
      '/pricing',
      '/purchase?tab=subscription&plan=1',
    )
    vi.spyOn(router.options.history, 'state', 'get').mockReturnValue({
      back: '/purchase?tab=subscription&plan=1',
    } as never)
    await wrapper.get('[data-testid="pricing-back"]').trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.fullPath).toBe('/purchase?tab=subscription&plan=1')
  })

  it('switches the tier, background state, featured theme, and panels without navigation', async () => {
    vi.mocked(paymentAPI.getCheckoutInfo).mockResolvedValue({
      data: checkoutInfoFixture(approvedPlans()),
    } as never)

    const { router, wrapper } = await mountPricing()
    const tabs = wrapper.findAll('[role="tab"]')
    await tabs[0].trigger('click')

    expect(router.currentRoute.value.fullPath).toBe('/pricing')
    expect(wrapper.get('[data-testid="pricing-page"]').classes()).toContain('pricing-theme-light')
    expect(wrapper.get('.fluid-stub').attributes('data-tier')).toBe('low')
    expect(tabs[0].attributes('aria-selected')).toBe('true')
    expect(wrapper.get('[data-testid="pricing-tier-panel-low"]').attributes('aria-hidden')).toBe('false')

    await tabs[2].trigger('click')
    expect(wrapper.get('[data-testid="pricing-page"]').classes()).toContain('pricing-theme-high')
    expect(wrapper.get('.fluid-stub').attributes('data-tier')).toBe('high')
    expect(wrapper.get('[data-testid="pricing-tier-panel-high"]').attributes('aria-hidden')).toBe('false')
  })

  it('uses the legacy group query only to choose the initial tier and keeps the full catalogue', async () => {
    vi.mocked(paymentAPI.getCheckoutInfo).mockResolvedValue({
      data: checkoutInfoFixture(approvedPlans()),
    } as never)

    const { wrapper } = await mountPricing('/pricing?group=17')

    expect(wrapper.get('[data-testid="pricing-page"]').attributes('data-tier')).toBe('high')
    expect(wrapper.findAll('.plan-stub')).toHaveLength(7)
    expect(wrapper.find('[data-testid="pricing-group-filter"]').exists()).toBe(false)
  })

  it('uses an explicit tier or plan deep link before falling back to group', async () => {
    vi.mocked(paymentAPI.getCheckoutInfo).mockResolvedValue({
      data: checkoutInfoFixture(approvedPlans()),
    } as never)

    const explicitTier = await mountPricing('/pricing?tier=high&group=11')
    expect(explicitTier.wrapper.get('[data-testid="pricing-page"]').attributes('data-tier'))
      .toBe('high')

    const planName = await mountPricing('/pricing?plan=Pro')
    expect(planName.wrapper.get('[data-testid="pricing-page"]').attributes('data-tier'))
      .toBe('mid')

    const planId = await mountPricing('/pricing?plan=7')
    expect(planId.wrapper.get('[data-testid="pricing-page"]').attributes('data-tier'))
      .toBe('high')
  })

  it('does not let repeated group values choose a tier', async () => {
    vi.mocked(paymentAPI.getCheckoutInfo).mockResolvedValue({
      data: checkoutInfoFixture(approvedPlans()),
    } as never)

    const { wrapper } = await mountPricing('/pricing?group=17&group=11')
    expect(wrapper.get('[data-testid="pricing-page"]').attributes('data-tier')).toBe('low')
  })

  it.each(['subscribe', 'renew', 'upgrade'] as const)(
    'keeps the valid %s mode visible and forwards it to checkout',
    async (mode) => {
      vi.mocked(paymentAPI.getCheckoutInfo).mockResolvedValue({
        data: checkoutInfoFixture(approvedPlans()),
      } as never)

      const { router, wrapper } = await mountPricing(`/pricing?mode=${mode}`)

      expect(wrapper.get('[data-testid="pricing-page"]').attributes('data-mode')).toBe(mode)

      await wrapper.get('[data-plan-id="4"] .select-stub').trigger('click')
      await flushPromises()
      expect(router.currentRoute.value.fullPath)
        .toBe(`/purchase?tab=subscription&plan=4&mode=${mode}`)
    },
  )

  it('marks the renewal plan and keeps the upgrade target actionable', async () => {
    vi.mocked(paymentAPI.getCheckoutInfo).mockResolvedValue({
      data: checkoutInfoFixture(approvedPlans()),
    } as never)

    const renewal = await mountPricing('/pricing?mode=renew&tier=high&plan=Ultra&group=17')
    const renewalCard = renewal.wrapper.get('[data-plan-id="7"]')
    expect(renewalCard.attributes('data-current')).toBe('true')
    expect(renewalCard.attributes('data-action-label')).toBe('balanceMembership.renew')

    const upgrade = await mountPricing('/pricing?mode=upgrade&tier=high&group=12')
    const currentCard = upgrade.wrapper.get('[data-plan-id="2"]')
    const targetCard = upgrade.wrapper.get('[data-plan-id="7"]')
    expect(currentCard.attributes('data-current')).toBe('true')
    expect(currentCard.attributes('data-action-label')).toBe('pricing.currentPlan')
    expect(targetCard.attributes('data-action-label')).toBe('balanceMembership.upgrade')
  })

  it.each([
    'unknown',
    'subscribe&mode=upgrade',
  ])('ignores invalid or repeated membership mode %s', async (modeQuery) => {
    vi.mocked(paymentAPI.getCheckoutInfo).mockResolvedValue({
      data: checkoutInfoFixture(approvedPlans()),
    } as never)

    const { router, wrapper } = await mountPricing(`/pricing?mode=${modeQuery}`)

    expect(wrapper.get('[data-testid="pricing-page"]').attributes('data-mode')).toBeUndefined()
    await wrapper.get('[data-plan-id="4"] .select-stub').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe('/purchase?tab=subscription&plan=4')
  })

  it('opens the existing purchase flow from both the CTA and quota-details entry', async () => {
    vi.mocked(paymentAPI.getCheckoutInfo).mockResolvedValue({
      data: checkoutInfoFixture(approvedPlans()),
    } as never)

    const firstMount = await mountPricing()
    await firstMount.wrapper.get('[data-plan-id="4"] .select-stub').trigger('click')
    await flushPromises()
    expect(firstMount.router.currentRoute.value.fullPath).toBe('/purchase?tab=subscription&plan=4')

    const secondMount = await mountPricing()
    await secondMount.wrapper.get('[data-plan-id="5"] .details-stub').trigger('click')
    await flushPromises()
    expect(secondMount.router.currentRoute.value.fullPath).toBe('/purchase?tab=subscription&plan=5')
  })

  it('shows a retryable error without inventing fallback plans', async () => {
    vi.mocked(paymentAPI.getCheckoutInfo)
      .mockRejectedValueOnce(new Error('network unavailable'))
      .mockResolvedValueOnce({ data: checkoutInfoFixture(approvedPlans()) } as never)

    const { wrapper } = await mountPricing()
    expect(wrapper.find('[data-testid="pricing-error"]').exists()).toBe(true)
    expect(wrapper.findAll('.plan-stub')).toHaveLength(0)

    await wrapper.get('[data-testid="pricing-error"] button').trigger('click')
    await flushPromises()

    expect(paymentAPI.getCheckoutInfo).toHaveBeenCalledTimes(2)
    expect(wrapper.find('[data-testid="pricing-error"]').exists()).toBe(false)
    expect(wrapper.findAll('.plan-stub')).toHaveLength(7)
  })
})

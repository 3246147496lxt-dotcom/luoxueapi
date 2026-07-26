import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import PricingView from '../PricingView.vue'
import { paymentAPI } from '@/api/payment'
import subscriptionsAPI from '@/api/subscriptions'
import { useSubscriptionStore } from '@/stores/subscriptions'
import type { CheckoutInfoResponse, SubscriptionPlan } from '@/types/payment'

const authState = vi.hoisted(() => ({
  isAdmin: false,
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      locale: { value: 'en-US' },
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authState,
}))

vi.mock('@/api/payment', () => ({
  paymentAPI: {
    getCheckoutInfo: vi.fn(),
    getConfig: vi.fn(),
    getPlans: vi.fn(),
    createOrder: vi.fn(),
    getOrder: vi.fn(),
  },
}))

vi.mock('@/api/subscriptions', () => ({
  default: {
    getActiveSubscriptions: vi.fn(),
  },
}))

function planFixture(overrides: Partial<SubscriptionPlan> = {}): SubscriptionPlan {
  return {
    id: 7,
    group_id: 3,
    group_name: 'OpenAI',
    group_platform: 'openai',
    name: 'Starter',
    description: 'For individual API usage',
    price: 9.99,
    validity_days: 30,
    validity_unit: 'day',
    features: ['Priority access'],
    for_sale: true,
    sort_order: 1,
    ...overrides,
  }
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

async function mountPricing(path = '/pricing') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/pricing', component: { template: '<div />' } },
      { path: '/purchase', component: { template: '<div />' } },
      { path: '/subscriptions', component: { template: '<div />' } },
      { path: '/dashboard', component: { template: '<div />' } },
      { path: '/admin/dashboard', component: { template: '<div />' } },
    ],
  })
  await router.push(path)
  await router.isReady()

  const wrapper = shallowMount(PricingView, {
    global: {
      plugins: [router],
      stubs: {
        Icon: true,
        PricingPlanCard: {
          props: ['plan', 'renewal', 'focused'],
          emits: ['select'],
          template: '<button class="plan-stub" :data-plan-id="String(plan.id)" :data-renewal="String(renewal)" @click="$emit(\'select\', plan)">{{ plan.name }}</button>',
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
    authState.isAdmin = false
    vi.mocked(paymentAPI.getCheckoutInfo).mockReset()
    vi.mocked(subscriptionsAPI.getActiveSubscriptions).mockReset()
  })

  it('renders an independent catalogue, filters the selected group, and stays passive toward subscriptions', async () => {
    vi.mocked(paymentAPI.getCheckoutInfo).mockResolvedValue({
      data: checkoutInfoFixture([
        planFixture(),
        planFixture({ id: 8, group_id: 4, name: 'Pro' }),
      ]),
    } as never)

    const { wrapper } = await mountPricing('/pricing?group=3')

    expect(wrapper.find('[data-testid="pricing-page"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="pricing-close"]').attributes('aria-label')).toBe('pricing.close')
    expect(wrapper.findAll('.plan-stub')).toHaveLength(1)
    expect(wrapper.get('.plan-stub').text()).toBe('Starter')
    expect(wrapper.find('[data-testid="pricing-group-filter"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="pricing-group-filter"]').text()).toContain('pricing.groupFilter')
    expect(subscriptionsAPI.getActiveSubscriptions).not.toHaveBeenCalled()
  })

  it('marks every plan in an active group only as a renewal option, never as the current plan', async () => {
    const subscriptionStore = useSubscriptionStore()
    subscriptionStore.activeSubscriptions = [{
      id: 11,
      user_id: 9,
      group_id: 3,
      status: 'active',
    } as never]
    vi.mocked(paymentAPI.getCheckoutInfo).mockResolvedValue({
      data: checkoutInfoFixture([
        planFixture({ id: 7, group_id: 3, name: 'Monthly' }),
        planFixture({ id: 8, group_id: 3, name: 'Annual' }),
      ]),
    } as never)

    const { wrapper } = await mountPricing()
    const cards = wrapper.findAll('.plan-stub')

    expect(cards).toHaveLength(2)
    expect(cards.every((card) => card.attributes('data-renewal') === 'true')).toBe(true)
    expect(wrapper.html()).not.toContain('pricing.currentSubscription')
  })

  it('opens the mature /purchase checkout with the selected plan id', async () => {
    vi.mocked(paymentAPI.getCheckoutInfo).mockResolvedValue({
      data: checkoutInfoFixture([planFixture()]),
    } as never)

    const { router, wrapper } = await mountPricing()
    await wrapper.get('.plan-stub').trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.fullPath).toBe('/purchase?tab=subscription&plan=7')
  })

  it('shows a retryable error without inventing fallback plans', async () => {
    vi.mocked(paymentAPI.getCheckoutInfo)
      .mockRejectedValueOnce(new Error('network unavailable'))
      .mockResolvedValueOnce({ data: checkoutInfoFixture([planFixture()]) } as never)

    const { wrapper } = await mountPricing()
    expect(wrapper.find('[data-testid="pricing-error"]').exists()).toBe(true)
    expect(wrapper.findAll('.plan-stub')).toHaveLength(0)

    await wrapper.get('[data-testid="pricing-error"] button').trigger('click')
    await flushPromises()

    expect(paymentAPI.getCheckoutInfo).toHaveBeenCalledTimes(2)
    expect(wrapper.find('[data-testid="pricing-error"]').exists()).toBe(false)
    expect(wrapper.findAll('.plan-stub')).toHaveLength(1)
  })

  it('falls back to the role dashboard when no internal history is available', async () => {
    vi.mocked(paymentAPI.getCheckoutInfo).mockResolvedValue({
      data: checkoutInfoFixture(),
    } as never)
    authState.isAdmin = true
    const historyState = vi.spyOn(window.history, 'state', 'get').mockReturnValue(null)

    const { router, wrapper } = await mountPricing()
    await wrapper.get('[data-testid="pricing-close"]').trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.path).toBe('/admin/dashboard')
    historyState.mockRestore()
  })
})

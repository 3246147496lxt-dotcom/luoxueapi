import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'

import SubscriptionsView from '../SubscriptionsView.vue'
import subscriptionsAPI from '@/api/subscriptions'
import { useAppStore } from '@/stores/app'
import { usePaymentStore } from '@/stores/payment'
import type { PublicSettings, UserSubscription } from '@/types'
import type { CheckoutInfoResponse, SubscriptionPlan } from '@/types/payment'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/api/subscriptions', () => ({
  default: {
    getMySubscriptions: vi.fn(),
  },
}))

function checkoutInfoFixture(plans: SubscriptionPlan[] = []): CheckoutInfoResponse {
  return { plans } as CheckoutInfoResponse
}

function subscriptionFixture(): UserSubscription {
  return {
    id: 1,
    user_id: 7,
    group_id: 3,
    status: 'expired',
    starts_at: '2026-01-01T00:00:00Z',
    daily_usage_usd: 0,
    weekly_usage_usd: 0,
    monthly_usage_usd: 0,
    daily_window_start: null,
    weekly_window_start: null,
    monthly_window_start: null,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    expires_at: '2026-02-01T00:00:00Z',
  }
}

type MountOptions = {
  paymentEnabled?: boolean
  plans?: SubscriptionPlan[]
  checkoutError?: Error
  contactInfo?: string
  documentationUrl?: string
}

async function mountSubscriptions(options: MountOptions = {}) {
  const {
    paymentEnabled = true,
    plans = [],
    checkoutError,
    contactInfo = '',
    documentationUrl = '/tutorial-docs/',
  } = options

  const appStore = useAppStore()
  appStore.publicSettingsLoaded = true
  appStore.cachedPublicSettings = {
    payment_enabled: paymentEnabled,
    contact_info: contactInfo,
    doc_url: documentationUrl,
  } as PublicSettings

  const paymentStore = usePaymentStore()
  const ensureCheckoutInfo = vi.spyOn(paymentStore, 'ensureCheckoutInfo')
  if (checkoutError) {
    ensureCheckoutInfo.mockRejectedValue(checkoutError)
  } else {
    ensureCheckoutInfo.mockResolvedValue(checkoutInfoFixture(plans))
  }

  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/subscriptions', component: { template: '<div />' } },
      { path: '/pricing', component: { template: '<div />' } },
      { path: '/purchase', component: { template: '<div />' } },
      { path: '/quota-viewer', component: { template: '<div />' } },
    ],
  })
  await router.push('/subscriptions')
  await router.isReady()

  const wrapper = shallowMount(SubscriptionsView, {
    global: {
      plugins: [router],
      stubs: {
        AppLayout: { template: '<main><slot /></main>' },
        AdminPageHeader: { template: '<header><slot name="secondary-actions" /></header>' },
        CreditAmount: true,
        Icon: { template: '<svg />' },
        RouterLink: {
          props: ['to'],
          template: '<a :href="typeof to === \'string\' ? to : to.path"><slot /></a>',
        },
      },
    },
  })
  await flushPromises()

  return { ensureCheckoutInfo, router, wrapper }
}

describe('SubscriptionsView empty states', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.mocked(subscriptionsAPI.getMySubscriptions).mockReset()
    vi.mocked(subscriptionsAPI.getMySubscriptions).mockResolvedValue([])
  })

  it('offers the pricing catalogue when purchasable plans exist', async () => {
    const { wrapper } = await mountSubscriptions({
      plans: [{ id: 9 } as SubscriptionPlan],
    })

    expect(wrapper.get('[data-testid="subscriptions-empty-title"]').text())
      .toBe('userSubscriptions.emptyWithPlansTitle')
    expect(wrapper.get('[data-testid="subscriptions-view-plans"]').attributes('href'))
      .toBe('/pricing')
    expect(wrapper.get('[data-testid="subscriptions-recharge"]').attributes('href'))
      .toBe('/purchase')
    expect(wrapper.find('[data-testid="subscriptions-support"]').exists()).toBe(false)
  })

  it('explains that no plans are listed and keeps recharge and contact actions available', async () => {
    const { wrapper } = await mountSubscriptions()

    expect(wrapper.get('[data-testid="subscriptions-empty-title"]').text())
      .toBe('userSubscriptions.noPlansTitle')
    expect(wrapper.find('[data-testid="subscriptions-view-plans"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="subscriptions-recharge"]').attributes('href'))
      .toBe('/purchase')
    expect(wrapper.get('[data-testid="subscriptions-support"]').attributes('href'))
      .toBe('http://127.0.0.1:4179/tutorial-docs/#recharge')
    expect(wrapper.get('[data-testid="subscriptions-support"]').text())
      .toContain('userSubscriptions.viewSubscriptionHelp')
  })

  it('labels a configured support URL as direct administrator contact', async () => {
    const { wrapper } = await mountSubscriptions({
      contactInfo: 'https://support.example.com/contact',
    })

    expect(wrapper.get('[data-testid="subscriptions-support"]').attributes('href'))
      .toBe('https://support.example.com/contact')
    expect(wrapper.get('[data-testid="subscriptions-support"]').text())
      .toContain('userSubscriptions.contactAdmin')
  })

  it('skips checkout when self-service payment is disabled', async () => {
    const { ensureCheckoutInfo, wrapper } = await mountSubscriptions({
      paymentEnabled: false,
      plans: [{ id: 9 } as SubscriptionPlan],
    })

    expect(ensureCheckoutInfo).not.toHaveBeenCalled()
    expect(wrapper.get('[data-testid="subscriptions-empty-title"]').text())
      .toBe('userSubscriptions.selfServiceDisabledTitle')
    expect(wrapper.find('[data-testid="subscriptions-view-plans"]').exists()).toBe(false)
  })

  it('uses a neutral state when checkout capability cannot be confirmed', async () => {
    const consoleWarn = vi.spyOn(console, 'warn').mockImplementation(() => undefined)
    const { wrapper } = await mountSubscriptions({ checkoutError: new Error('offline') })

    expect(wrapper.get('[data-testid="subscriptions-empty-title"]').text())
      .toBe('userSubscriptions.purchaseOptionsUnknownTitle')
    expect(wrapper.find('[data-testid="subscriptions-view-plans"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="subscriptions-recharge"]').attributes('href'))
      .toBe('/purchase')
    consoleWarn.mockRestore()
  })

  it('shows a retryable error instead of misreporting an empty subscription list', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined)
    vi.mocked(subscriptionsAPI.getMySubscriptions)
      .mockRejectedValueOnce(new Error('network unavailable'))
      .mockResolvedValueOnce([])

    const { wrapper } = await mountSubscriptions()

    expect(wrapper.find('[data-testid="subscriptions-load-error"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="subscriptions-empty-state"]').exists()).toBe(false)

    await wrapper.get('[data-testid="subscriptions-load-error"] button').trigger('click')
    await flushPromises()

    expect(subscriptionsAPI.getMySubscriptions).toHaveBeenCalledTimes(2)
    expect(wrapper.find('[data-testid="subscriptions-load-error"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="subscriptions-empty-state"]').exists()).toBe(true)
    consoleError.mockRestore()
  })

  it('keeps existing subscription records out of the empty state', async () => {
    vi.mocked(subscriptionsAPI.getMySubscriptions).mockResolvedValue([subscriptionFixture()])

    const { wrapper } = await mountSubscriptions()

    expect(wrapper.find('[data-testid="subscriptions-empty-state"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('Group #3')
  })
})

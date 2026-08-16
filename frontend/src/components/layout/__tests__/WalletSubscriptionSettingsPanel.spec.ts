import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia, type Pinia } from 'pinia'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Group, UserSubscription } from '@/types'
import type { PaymentOrder } from '@/types/payment'
import type { AccountPanelSummary } from '../accountPanelTypes'

const { getMyOrders } = vi.hoisted(() => ({
  getMyOrders: vi.fn(),
}))

vi.mock('@/api/payment', () => ({
  paymentAPI: {
    getMyOrders,
  },
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const { ref } = await vi.importActual<typeof import('vue')>('vue')
  return {
    ...actual,
    useI18n: () => ({
      locale: ref('en'),
      t: (key: string, values?: Record<string, unknown>) => (
        values ? `${key}:${JSON.stringify(values)}` : key
      ),
    }),
  }
})

import WalletSubscriptionSettingsPanel from '../WalletSubscriptionSettingsPanel.vue'
import { useSubscriptionStore } from '@/stores/subscriptions'

const summary: AccountPanelSummary = {
  displayName: 'Riley Quinn',
  email: 'riley@example.com',
  initials: 'RQ',
  avatarUrl: '',
  frozenBalance: 3.5,
  formattedAvailableBalance: '24.50',
  formattedFrozenBalance: '3.50',
  activeSubscriptionCount: 1,
  subscriptionsLoaded: true,
}

const wrappers = new Set<VueWrapper>()

function makeSubscription(
  id: number,
  groupId: number,
  groupName: string,
  expiresAt = '2026-10-01T00:00:00Z',
): UserSubscription {
  return {
    id,
    user_id: 7,
    group_id: groupId,
    status: 'active',
    starts_at: '2026-07-01T00:00:00Z',
    daily_usage_usd: 0,
    weekly_usage_usd: 0,
    monthly_usage_usd: 0,
    daily_window_start: null,
    weekly_window_start: null,
    monthly_window_start: null,
    created_at: '2026-07-01T00:00:00Z',
    updated_at: '2026-07-01T00:00:00Z',
    expires_at: expiresAt,
    group: { name: groupName } as Group,
  }
}

function makeOrder(
  id: number,
  overrides: Partial<PaymentOrder> = {},
): PaymentOrder {
  return {
    id,
    user_id: 7,
    amount: 20,
    pay_amount: 20,
    currency: 'USD',
    fee_rate: 0,
    payment_type: 'stripe',
    out_trade_no: `ORDER-${id}`,
    status: 'COMPLETED',
    order_type: 'subscription',
    created_at: '2026-07-20T08:30:00Z',
    expires_at: '2026-07-20T09:00:00Z',
    refund_amount: 0,
    ...overrides,
  }
}

function orderResponse(items: PaymentOrder[]) {
  return {
    data: {
      items,
      total: items.length,
      page: 1,
      page_size: 3,
      pages: items.length > 0 ? 1 : 0,
    },
  }
}

interface MountOptions {
  paymentEnabled?: boolean
  simpleMode?: boolean
  panelSummary?: AccountPanelSummary
  subscriptions?: UserSubscription[]
  beforeMount?: (store: ReturnType<typeof useSubscriptionStore>) => void
}

async function mountPanel(options: MountOptions = {}) {
  const pinia = createPinia()
  setActivePinia(pinia)
  const subscriptionStore = useSubscriptionStore()
  subscriptionStore.activeSubscriptions = options.subscriptions ?? []
  options.beforeMount?.(subscriptionStore)

  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: { template: '<div />' } },
      { path: '/purchase', component: { template: '<div />' } },
      { path: '/pricing', component: { template: '<div />' } },
      { path: '/subscriptions', component: { template: '<div />' } },
      { path: '/orders', component: { template: '<div />' } },
    ],
  })
  await router.push('/')
  await router.isReady()

  const wrapper = mount(WalletSubscriptionSettingsPanel, {
    props: {
      summary: options.panelSummary ?? summary,
      paymentEnabled: options.paymentEnabled ?? true,
      simpleMode: options.simpleMode ?? false,
    },
    global: {
      plugins: [pinia, router],
      stubs: {
        CreditAmount: {
          props: ['value'],
          template: '<span data-testid="credit-amount">{{ value }}</span>',
        },
        Icon: {
          props: ['name'],
          template: '<span :data-icon="name" />',
        },
        OrderStatusBadge: {
          props: ['status'],
          template: '<span data-testid="order-status">{{ status }}</span>',
        },
      },
    },
  })
  wrappers.add(wrapper)

  return {
    wrapper,
    pinia,
    router,
    subscriptionStore,
  } satisfies {
    wrapper: VueWrapper
    pinia: Pinia
    router: Router
    subscriptionStore: ReturnType<typeof useSubscriptionStore>
  }
}

beforeEach(() => {
  getMyOrders.mockReset()
  getMyOrders.mockResolvedValue(orderResponse([]))
})

afterEach(() => {
  for (const wrapper of wrappers) wrapper.unmount()
  wrappers.clear()
  vi.restoreAllMocks()
})

describe('WalletSubscriptionSettingsPanel', () => {
  it('starts with wallet content without repeating the active category introduction', async () => {
    const { wrapper } = await mountPanel()
    await flushPromises()

    expect(wrapper.element.tagName).toBe('DIV')
    expect(wrapper.attributes('aria-labelledby')).toBeUndefined()
    expect(wrapper.find('.wallet-settings__header').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('personalSettings.wallet.description')
    expect(wrapper.get('#wallet-settings-balance-title').text())
      .toBe('personalSettings.wallet.balance')
  })

  it('loads recent orders only after mount and renders at most three store subscriptions and orders', async () => {
    const subscriptions = [
      makeSubscription(1, 101, 'Pro'),
      makeSubscription(2, 102, 'Team'),
      makeSubscription(3, 103, 'Scale'),
      makeSubscription(4, 104, 'Hidden fourth'),
    ]
    getMyOrders.mockResolvedValue(orderResponse([
      makeOrder(1),
      makeOrder(2, { order_type: 'balance' }),
      makeOrder(3),
      makeOrder(4),
    ]))
    expect(getMyOrders).not.toHaveBeenCalled()

    const fetchSubscriptions = vi.fn(async (_force?: boolean) => [] as UserSubscription[])
    const startPolling = vi.fn()
    const { wrapper } = await mountPanel({
      subscriptions,
      beforeMount(store) {
        vi.spyOn(store, 'fetchActiveSubscriptions').mockImplementation(fetchSubscriptions)
        vi.spyOn(store, 'startPolling').mockImplementation(startPolling)
      },
    })
    await flushPromises()

    expect(getMyOrders).toHaveBeenCalledOnce()
    expect(getMyOrders).toHaveBeenCalledWith({ page: 1, page_size: 3 })
    expect(fetchSubscriptions).not.toHaveBeenCalled()
    expect(startPolling).not.toHaveBeenCalled()
    expect(wrapper.findAll('[data-testid="wallet-subscription-row"]')).toHaveLength(3)
    expect(wrapper.findAll('[data-testid="wallet-order-row"]')).toHaveLength(3)

    expect(wrapper.get('[data-testid="wallet-purchase-link"]').attributes('href')).toBe('/purchase')
    expect(wrapper.get('[data-testid="wallet-redeem-link"]').attributes('href')).toBe('/purchase#redeem')
    expect(wrapper.get('[data-testid="wallet-subscriptions-link"]').attributes('href')).toBe('/subscriptions')
    expect(wrapper.get('[data-testid="wallet-orders-link"]').attributes('href')).toBe('/orders')
    expect(
      wrapper.get('[data-renew-group="101"]').attributes('href'),
    ).toBe('/pricing?mode=renew&group=101')

    await wrapper.setProps({
      summary: {
        ...summary,
        formattedAvailableBalance: '30.00',
      },
    })
    expect(getMyOrders).toHaveBeenCalledOnce()
  })

  it('shows a loading state and then a useful empty state', async () => {
    let resolveOrders!: (value: ReturnType<typeof orderResponse>) => void
    getMyOrders.mockImplementationOnce(
      () => new Promise((resolve) => {
        resolveOrders = resolve
      }),
    )

    const { wrapper } = await mountPanel()

    expect(wrapper.find('[data-testid="wallet-orders-loading"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="wallet-orders-empty"]').exists()).toBe(false)

    resolveOrders(orderResponse([]))
    await flushPromises()

    expect(wrapper.find('[data-testid="wallet-orders-loading"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="wallet-orders-empty"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="wallet-orders-empty"]').text()).toContain(
      'personalSettings.orders.emptyHint',
    )
  })

  it('keeps the frozen balance visible when it is zero', async () => {
    const { wrapper } = await mountPanel({
      panelSummary: {
        ...summary,
        frozenBalance: 0,
        formattedFrozenBalance: '0.00',
      },
    })
    await flushPromises()

    expect(wrapper.get('[data-testid="wallet-frozen-balance"]').text()).toContain('0.00')
    expect(wrapper.get('[data-testid="wallet-frozen-balance"]').classes())
      .not.toContain('wallet-settings__balance-row--frozen')
  })

  it('renders an inline error and retries with the same bounded query', async () => {
    getMyOrders.mockRejectedValueOnce(new Error('network unavailable'))
    const { wrapper } = await mountPanel()
    await flushPromises()

    const error = wrapper.get('[data-testid="wallet-orders-error"]')
    expect(error.attributes('role')).toBe('alert')
    expect(error.text()).toContain('personalSettings.orders.loadError')

    getMyOrders.mockResolvedValueOnce(orderResponse([makeOrder(9)]))
    await wrapper.get('[data-testid="wallet-orders-retry"]').trigger('click')
    await flushPromises()

    expect(getMyOrders).toHaveBeenCalledTimes(2)
    expect(getMyOrders).toHaveBeenNthCalledWith(2, { page: 1, page_size: 3 })
    expect(wrapper.find('[data-testid="wallet-orders-error"]').exists()).toBe(false)
    expect(wrapper.findAll('[data-testid="wallet-order-row"]')).toHaveLength(1)
  })

  it('reads subscription state passively and only retries after explicit user action', async () => {
    const fetchSubscriptions = vi.fn().mockRejectedValue(new Error('subscription unavailable'))
    const { wrapper } = await mountPanel({
      panelSummary: {
        ...summary,
        activeSubscriptionCount: 0,
        subscriptionsLoaded: false,
      },
      beforeMount(store) {
        vi.spyOn(store, 'fetchActiveSubscriptions').mockImplementation(fetchSubscriptions)
      },
    })
    await flushPromises()

    expect(fetchSubscriptions).not.toHaveBeenCalled()
    expect(wrapper.get('[data-testid="wallet-subscriptions-error"]').attributes('role'))
      .toBe('alert')

    await wrapper.get('[data-testid="wallet-subscriptions-retry"]').trigger('click')
    await flushPromises()

    expect(fetchSubscriptions).toHaveBeenCalledOnce()
    expect(fetchSubscriptions).toHaveBeenCalledWith(true)
  })

  it('gates purchase, renewal, order, and subscription-management entry points', async () => {
    const subscriptions = [makeSubscription(1, 101, 'Pro')]
    const disabled = await mountPanel({
      paymentEnabled: false,
      subscriptions,
    })
    await flushPromises()

    expect(disabled.wrapper.find('[data-testid="wallet-purchase-link"]').exists()).toBe(false)
    expect(disabled.wrapper.find('[data-testid="wallet-orders-section"]').exists()).toBe(false)
    expect(disabled.wrapper.find('[data-renew-group]').exists()).toBe(false)
    expect(disabled.wrapper.find('[data-testid="wallet-subscriptions-link"]').exists()).toBe(true)
    expect(disabled.wrapper.get('[data-testid="wallet-redeem-link"]').attributes('href')).toBe('/purchase#redeem')

    disabled.wrapper.unmount()
    wrappers.delete(disabled.wrapper)

    const simple = await mountPanel({
      paymentEnabled: true,
      simpleMode: true,
      subscriptions,
    })
    await flushPromises()

    expect(simple.wrapper.find('[data-testid="wallet-purchase-link"]').exists()).toBe(true)
    expect(simple.wrapper.find('[data-testid="wallet-orders-section"]').exists()).toBe(false)
    expect(simple.wrapper.find('[data-testid="wallet-subscriptions-link"]').exists()).toBe(false)
    expect(simple.wrapper.find('[data-testid="wallet-subscription-list"]').exists()).toBe(false)
    expect(simple.wrapper.find('[data-renew-group]').exists()).toBe(false)
    expect(getMyOrders).not.toHaveBeenCalled()
  })
})

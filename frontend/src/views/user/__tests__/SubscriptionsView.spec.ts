import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'

import SubscriptionsView from '../SubscriptionsView.vue'
import subscriptionsAPI from '@/api/subscriptions'
import { useAuthStore } from '@/stores/auth'
import type { User, UserSubscription } from '@/types'

const subscriptionsViewSource = readFileSync(
  resolve(process.cwd(), 'src/views/user/SubscriptionsView.vue'),
  'utf8',
)

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

function userFixture(overrides: Partial<User> = {}): User {
  return {
    id: 7,
    username: 'overview-user',
    email: 'overview@example.com',
    role: 'user',
    balance: 1280,
    concurrency: 2,
    status: 'active',
    allowed_groups: null,
    balance_notify_enabled: true,
    balance_notify_threshold: null,
    balance_notify_extra_emails: [],
    created_at: '2026-07-18T00:00:00Z',
    updated_at: '2026-07-18T00:00:00Z',
    ...overrides,
  }
}

function subscriptionFixture(
  overrides: Partial<UserSubscription> = {},
): UserSubscription {
  return {
    id: 1,
    user_id: 7,
    group_id: 3,
    status: 'active',
    starts_at: '2026-08-01T00:00:00Z',
    daily_usage_usd: 0,
    weekly_usage_usd: 0,
    monthly_usage_usd: 2720,
    daily_window_start: null,
    weekly_window_start: null,
    monthly_window_start: '2026-08-01T00:00:00Z',
    created_at: '2026-08-01T00:00:00Z',
    updated_at: '2026-08-01T00:00:00Z',
    expires_at: '2026-09-30T00:00:00Z',
    group: {
      id: 3,
      name: 'Ultra',
      description: null,
      platform: 'openai',
      rate_multiplier: 1,
      is_exclusive: false,
      status: 'active',
      subscription_type: 'subscription',
      daily_limit_usd: null,
      weekly_limit_usd: null,
      monthly_limit_usd: 10000,
      allow_image_generation: false,
      allow_batch_image_generation: false,
      image_rate_independent: false,
      image_rate_multiplier: 1,
      batch_image_discount_multiplier: 1,
      batch_image_hold_multiplier: 1,
      image_price_1k: null,
      image_price_2k: null,
      image_price_4k: null,
      video_rate_independent: false,
      video_rate_multiplier: 1,
      video_price_480p: null,
      video_price_720p: null,
      video_price_1080p: null,
      web_search_price_per_call: null,
      peak_rate_enabled: false,
      peak_start: '',
      peak_end: '',
      peak_rate_multiplier: 1,
      claude_code_only: false,
      fallback_group_id: null,
      fallback_group_id_on_invalid_request: null,
      require_oauth_only: false,
      require_privacy_set: false,
      created_at: '2026-08-01T00:00:00Z',
      updated_at: '2026-08-01T00:00:00Z',
    },
    ...overrides,
  }
}

async function mountOverview(
  subscriptions: UserSubscription[] = [],
): Promise<{
  router: ReturnType<typeof createRouter>
  wrapper: ReturnType<typeof shallowMount>
}> {
  const authStore = useAuthStore()
  authStore.user = userFixture()
  vi.mocked(subscriptionsAPI.getMySubscriptions).mockResolvedValue(subscriptions)

  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/subscriptions', component: { template: '<div />' } },
      { path: '/pricing', component: { template: '<div />' } },
      { path: '/purchase', component: { template: '<div />' } },
    ],
  })
  await router.push('/subscriptions')
  await router.isReady()

  const wrapper = shallowMount(SubscriptionsView, {
    global: {
      plugins: [router],
      stubs: {
        AppLayout: { template: '<main><slot /></main>' },
        AdminPageHeader: {
          props: ['title', 'description'],
          template: '<header><h1>{{ title }}</h1><p>{{ description }}</p></header>',
        },
        CreditAmount: {
          props: ['value'],
          template: '<span data-testid="credit-amount-stub">{{ value }}</span>',
        },
        Icon: { template: '<svg />' },
      },
    },
  })
  await flushPromises()
  return { router, wrapper }
}

describe('SubscriptionsView balance and membership overview', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.mocked(subscriptionsAPI.getMySubscriptions).mockReset()
  })

  it('uses one tokenized card surface without membership-tier color auras', () => {
    expect(subscriptionsViewSource).toContain('background: var(--workspace-card-surface);')
    expect(subscriptionsViewSource).toContain('border: 1px solid var(--workspace-border);')
    expect(subscriptionsViewSource).toContain('box-shadow: var(--workspace-work-shadow-card);')
    expect(subscriptionsViewSource).not.toContain('membership-overview-card--')
    expect(subscriptionsViewSource).not.toContain('radial-gradient')
    expect(subscriptionsViewSource).not.toContain(':global(.dark)')
  })

  it('renders the concise overview without an embedded plan catalogue', async () => {
    const { wrapper } = await mountOverview()

    expect(wrapper.get('h1').text()).toBe('userSubscriptions.title')
    expect(wrapper.get('header p').text()).toBe('userSubscriptions.description')
    expect(wrapper.findAll('[data-testid="balance-membership-overview"]')).toHaveLength(1)
    const balanceCard = wrapper.get('[data-testid="balance-card"]')
    expect(balanceCard.classes()).toContain('balance-membership-card')
    expect(balanceCard.text()).toContain('balanceMembership.balanceTitle')
    expect(balanceCard.text()).toContain('balanceMembership.recharge')
    expect(wrapper.get('[data-testid="free-member-card"]').classes()).toContain('balance-membership-card')
    expect(wrapper.findAll('[data-testid="pricing-section"]')).toHaveLength(0)
    expect(wrapper.text()).not.toContain('Try')
    expect(wrapper.text()).not.toContain('Standard')
    expect(wrapper.text()).not.toContain('Ultra')
  })

  it('keeps recharge and redeem actions on the existing purchase flow', async () => {
    const { router, wrapper } = await mountOverview()

    await wrapper.get('[data-testid="balance-recharge"]').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe('/purchase')

    await router.push('/subscriptions')
    await wrapper.get('[data-testid="balance-redeem"]').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe('/purchase#redeem')
  })

  it('shows a neutral free state with a clear catalogue entry and no fake quota', async () => {
    const { router, wrapper } = await mountOverview()

    const card = wrapper.get('[data-testid="free-member-card"]')
    expect(card.text()).toContain('balanceMembership.freeTitle')
    expect(card.text()).toContain('balanceMembership.freeDescription')
    expect(card.findAll('[role="progressbar"]')).toHaveLength(0)
    expect(card.text()).not.toContain('0 / 0')

    await card.get('[data-testid="member-subscribe"]').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.query).toEqual({ mode: 'subscribe', tier: 'low' })
  })

  it('shows only the monthly quota for an active member and routes renewal to pricing', async () => {
    const { router, wrapper } = await mountOverview([subscriptionFixture()])

    const card = wrapper.get('[data-testid="current-member-card"]')
    expect(card.classes()).toContain('balance-membership-card')
    expect(card.text()).toContain('Ultra')
    expect(card.text()).toContain('72.8')
    expect(card.get('[role="progressbar"]').attributes('aria-valuenow')).toBe('72.8')
    expect(card.text()).not.toMatch(/daily|weekly|每日|每周/i)
    expect(card.text()).toContain('balanceMembership.viewPlans')

    await card.get('[data-testid="member-renew"]').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.query).toEqual({
      mode: 'renew',
      tier: 'high',
      plan: 'Ultra',
      group: '3',
    })
  })

  it('recognizes branded high-tier names without adding a visual tier class', async () => {
    const base = subscriptionFixture()
    const { router, wrapper } = await mountOverview([
      subscriptionFixture({ group: { ...base.group!, name: 'Ultra Preview' } }),
    ])
    const card = wrapper.get('[data-testid="current-member-card"]')

    expect(card.classes()).toContain('balance-membership-card')
    expect(card.classes().some(className => className.startsWith('membership-overview-card--')))
      .toBe(false)

    await card.get('[data-testid="member-upgrade"]').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.query).toEqual({
      tier: 'high',
      plan: 'Ultra Preview',
      group: '3',
    })
  })

  it('uses a plain catalogue link for a highest-tier member', async () => {
    const { router, wrapper } = await mountOverview([subscriptionFixture()])
    const card = wrapper.get('[data-testid="current-member-card"]')

    await card.get('[data-testid="member-upgrade"]').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.query).toEqual({
      tier: 'high',
      plan: 'Ultra',
      group: '3',
    })
  })

  it('uses upgrade intent for members below the highest tier', async () => {
    const base = subscriptionFixture()
    const subscription = subscriptionFixture({
      group: { ...base.group!, name: 'Pro' },
    })
    const { router, wrapper } = await mountOverview([subscription])
    const card = wrapper.get('[data-testid="current-member-card"]')

    expect(card.text()).toContain('balanceMembership.upgrade')
    await card.get('[data-testid="member-upgrade"]').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.query).toEqual({
      mode: 'upgrade',
      tier: 'high',
      group: '3',
    })
  })

  it('falls back to viewing plans when the live tier cannot be classified', async () => {
    const base = subscriptionFixture()
    const { router, wrapper } = await mountOverview([
      subscriptionFixture({ group: { ...base.group!, name: 'Custom member' } }),
    ])
    const card = wrapper.get('[data-testid="current-member-card"]')

    expect(card.text()).toContain('balanceMembership.viewPlans')
    await card.get('[data-testid="member-upgrade"]').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.query).toEqual({
      plan: 'Custom member',
      group: '3',
    })
  })

  it('keeps a retryable error state and does not invent membership data', async () => {
    vi.mocked(subscriptionsAPI.getMySubscriptions)
      .mockRejectedValueOnce(new Error('network unavailable'))
      .mockResolvedValueOnce([])
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined)

    const { wrapper } = await mountOverview()
    expect(wrapper.findAll('[data-testid="subscriptions-load-error"]')).toHaveLength(1)
    expect(wrapper.get('[data-testid="subscriptions-load-error"]').classes())
      .toContain('balance-membership-card')
    expect(wrapper.findAll('[data-testid="current-member-card"]')).toHaveLength(0)

    await wrapper.get('[data-testid="subscriptions-load-error"] button').trigger('click')
    await flushPromises()
    expect(wrapper.findAll('[data-testid="subscriptions-load-error"]')).toHaveLength(0)
    expect(wrapper.findAll('[data-testid="free-member-card"]')).toHaveLength(1)
    consoleError.mockRestore()
  })

  it('does not turn missing balance or monthly usage into zero-valued data', async () => {
    const base = subscriptionFixture()
    const { wrapper } = await mountOverview([
      subscriptionFixture({
        monthly_usage_usd: null as unknown as number,
        group: { ...base.group!, monthly_limit_usd: 10000 },
      }),
    ])

    expect(wrapper.get('[data-testid="balance-card"] [data-testid="credit-amount-stub"]').text())
      .toBe('1,280')
    expect(wrapper.get('[data-testid="current-member-card"]').text())
      .toContain('balanceMembership.monthlyQuotaUnavailable')
    expect(wrapper.get('[data-testid="current-member-card"]').findAll('[role="progressbar"]'))
      .toHaveLength(0)
  })
})

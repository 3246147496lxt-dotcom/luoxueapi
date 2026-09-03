import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAuthStore } from '@/stores/auth'
import { useSubscriptionStore } from '@/stores/subscriptions'
import { useUserProfileStore } from '@/stores/userProfile'
import type { Group, User, UserSubscription } from '@/types'

const mockGetActiveSubscriptions = vi.hoisted(() => vi.fn())

vi.mock('@/api/subscriptions', () => ({
  default: {
    getActiveSubscriptions: (...args: unknown[]) => (
      mockGetActiveSubscriptions(...args)
    ),
  },
}))

function makeUser(id: number, overrides: Partial<User> = {}): User {
  return {
    id,
    username: `user-${id}`,
    email: `user-${id}@example.com`,
    role: 'user',
    balance: 10,
    frozen_balance: 2,
    concurrency: 1,
    status: 'active',
    allowed_groups: null,
    balance_notify_enabled: false,
    balance_notify_threshold: null,
    balance_notify_extra_emails: [],
    created_at: '2026-08-01T00:00:00Z',
    updated_at: '2026-08-01T00:00:00Z',
    ...overrides,
  }
}

function makeSubscription(
  id: number,
  userId: number,
  name: string,
  expiresAt = '2026-10-01T00:00:00Z',
): UserSubscription {
  return {
    id,
    user_id: userId,
    group_id: id,
    status: 'active',
    starts_at: '2026-08-01T00:00:00Z',
    daily_usage_usd: 0,
    weekly_usage_usd: 0,
    monthly_usage_usd: 0,
    daily_window_start: null,
    weekly_window_start: null,
    monthly_window_start: null,
    created_at: '2026-08-01T00:00:00Z',
    updated_at: '2026-08-01T00:00:00Z',
    expires_at: expiresAt,
    group: { name } as Group,
  }
}

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

describe('useUserProfileStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockGetActiveSubscriptions.mockReset()
  })

  it('derives one reactive profile from auth and subscription stores', async () => {
    const authStore = useAuthStore()
    const userProfileStore = useUserProfileStore()
    const refreshUser = vi.spyOn(authStore, 'refreshUser')
    authStore.user = makeUser(7, {
      username: '  Riley Quinn  ',
      email: '  riley@example.com  ',
      avatar_url: '  https://example.com/avatar.png  ',
    })
    mockGetActiveSubscriptions.mockResolvedValue([
      makeSubscription(1, 7, 'Team', '2026-11-01T00:00:00Z'),
      makeSubscription(2, 7, 'Pro', '2026-09-01T00:00:00Z'),
    ])

    await userProfileStore.initialize()
    await userProfileStore.initialize()

    expect(mockGetActiveSubscriptions).toHaveBeenCalledTimes(1)
    expect(refreshUser).not.toHaveBeenCalled()
    expect(userProfileStore.subscriptionsLoaded).toBe(true)
    expect(userProfileStore.activeSubscriptionCount).toBe(2)
    expect(userProfileStore.primarySubscription).toEqual({
      id: 2,
      name: 'Pro',
      expiresAt: '2026-09-01T00:00:00Z',
    })
    expect(userProfileStore.profile).toEqual({
      id: 7,
      username: 'Riley Quinn',
      displayName: 'Riley Quinn',
      email: 'riley@example.com',
      avatarUrl: 'https://example.com/avatar.png',
      initials: 'R',
      role: 'user',
      availableBalance: 10,
      frozenBalance: 2,
      currentPlan: {
        state: 'active',
        name: 'Pro',
        activeCount: 2,
        expiresAt: '2026-09-01T00:00:00Z',
      },
      currentPlanLevel: 'Pro',
      planState: 'active',
    })

    authStore.user = makeUser(7, {
      username: '',
      email: 'fallback@example.com',
      balance: 25,
    })

    expect(userProfileStore.profile?.displayName).toBe('fallback')
    expect(userProfileStore.profile?.availableBalance).toBe(25)
  })

  it('deduplicates explicit profile refreshes and forces both sources', async () => {
    const authStore = useAuthStore()
    const subscriptionStore = useSubscriptionStore()
    const userProfileStore = useUserProfileStore()
    authStore.user = makeUser(7)

    const userRequest = deferred<User>()
    const subscriptionRequest = deferred<UserSubscription[]>()
    const refreshUser = vi.spyOn(authStore, 'refreshUser')
      .mockReturnValue(userRequest.promise)
    const fetchSubscriptions = vi.spyOn(
      subscriptionStore,
      'fetchActiveSubscriptions',
    )
    mockGetActiveSubscriptions.mockReturnValue(subscriptionRequest.promise)

    const first = userProfileStore.refreshProfile()
    const second = userProfileStore.refreshProfile()

    expect(refreshUser).toHaveBeenCalledTimes(1)
    expect(fetchSubscriptions).toHaveBeenCalledTimes(1)
    expect(fetchSubscriptions).toHaveBeenCalledWith(true)

    userRequest.resolve(makeUser(7))
    subscriptionRequest.resolve([makeSubscription(1, 7, 'Pro')])
    await Promise.all([first, second])

    mockGetActiveSubscriptions.mockResolvedValue([makeSubscription(2, 7, 'Team')])
    refreshUser.mockResolvedValue(makeUser(7))
    await userProfileStore.syncAfterUpgrade()

    expect(refreshUser).toHaveBeenCalledTimes(2)
    expect(fetchSubscriptions).toHaveBeenCalledTimes(2)
    expect(fetchSubscriptions).toHaveBeenLastCalledWith(true)
  })

  it('invalidates a previous principal request before it can refill membership', async () => {
    const authStore = useAuthStore()
    const subscriptionStore = useSubscriptionStore()
    const userProfileStore = useUserProfileStore()
    const firstRequest = deferred<UserSubscription[]>()
    const secondRequest = deferred<UserSubscription[]>()
    mockGetActiveSubscriptions
      .mockReturnValueOnce(firstRequest.promise)
      .mockReturnValueOnce(secondRequest.promise)

    authStore.user = makeUser(7)
    const firstInitialization = userProfileStore.initialize()

    authStore.user = makeUser(9)
    expect(userProfileStore.profile?.currentPlan).toEqual({
      state: 'pending',
      name: null,
      activeCount: 0,
      expiresAt: null,
    })
    const secondInitialization = userProfileStore.initialize()

    firstRequest.resolve([makeSubscription(1, 7, 'Old plan')])
    await firstInitialization
    expect(subscriptionStore.activeSubscriptions).toEqual([])

    secondRequest.resolve([makeSubscription(2, 9, 'New plan')])
    await secondInitialization

    expect(userProfileStore.profile?.id).toBe(9)
    expect(userProfileStore.profile?.currentPlanLevel).toBe('New plan')
    expect(subscriptionStore.activeSubscriptions).toEqual([
      expect.objectContaining({ user_id: 9 }),
    ])
  })

  it('clear invalidates pending membership work', async () => {
    const authStore = useAuthStore()
    const subscriptionStore = useSubscriptionStore()
    const userProfileStore = useUserProfileStore()
    const request = deferred<UserSubscription[]>()
    authStore.user = makeUser(7)
    mockGetActiveSubscriptions.mockReturnValue(request.promise)

    const initialization = userProfileStore.initialize()
    userProfileStore.clear()
    request.resolve([makeSubscription(1, 7, 'Stale plan')])
    await initialization

    expect(subscriptionStore.activeSubscriptions).toEqual([])
    expect(userProfileStore.subscriptionsLoaded).toBe(false)
    expect(userProfileStore.profile?.currentPlanLevel).toBeNull()
  })
})

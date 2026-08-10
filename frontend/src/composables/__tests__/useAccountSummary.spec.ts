import { createPinia, setActivePinia, type Pinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { toRaw, type Ref } from 'vue'
import { useAnnouncementStore } from '@/stores/announcements'
import { useAuthStore } from '@/stores/auth'
import { useSubscriptionStore } from '@/stores/subscriptions'
import type { Group, User, UserAnnouncement, UserSubscription } from '@/types'
import { useAccountSummary } from '../useAccountSummary'

function makeUser(overrides: Partial<User> = {}): User {
  return {
    id: 7,
    username: 'Riley Quinn',
    email: 'riley@example.com',
    role: 'user',
    balance: 12.5,
    frozen_balance: 2.5,
    concurrency: 1,
    status: 'active',
    allowed_groups: null,
    balance_notify_enabled: false,
    balance_notify_threshold: null,
    balance_notify_extra_emails: [],
    created_at: '2026-07-01T00:00:00Z',
    updated_at: '2026-07-01T00:00:00Z',
    ...overrides,
  }
}

function makeSubscription(
  id: number,
  expiresAt: string | null,
  groupName?: string,
): UserSubscription {
  return {
    id,
    user_id: 7,
    group_id: id,
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
    group: groupName ? ({ name: groupName } as Group) : undefined,
  }
}

function makeAnnouncement(id: number, readAt?: string): UserAnnouncement {
  return {
    id,
    title: `Announcement ${id}`,
    content: 'Content',
    notify_mode: 'silent',
    read_at: readAt,
    created_at: '2026-07-01T00:00:00Z',
    updated_at: '2026-07-01T00:00:00Z',
  }
}

describe('useAccountSummary', () => {
  let pinia: Pinia

  beforeEach(() => {
    pinia = createPinia()
    setActivePinia(pinia)
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('derives identity, balances, subscription, and announcement summaries reactively', () => {
    const authStore = useAuthStore()
    const subscriptionStore = useSubscriptionStore()
    const announcementStore = useAnnouncementStore()
    const summary = useAccountSummary({ subscriptionStore })

    authStore.user = makeUser({
      username: '  Riley Quinn  ',
      email: '  riley@example.com  ',
      avatar_url: '  https://example.com/avatar.png  ',
      role: 'admin',
    })
    subscriptionStore.activeSubscriptions = [
      makeSubscription(1, '2026-10-01T00:00:00Z', 'Team'),
      makeSubscription(2, '2026-08-01T00:00:00Z', 'Pro'),
      makeSubscription(3, null, 'Lifetime'),
    ]
    subscriptionStore.loading = true
    announcementStore.announcements = [
      makeAnnouncement(1),
      makeAnnouncement(2, '2026-07-10T00:00:00Z'),
      makeAnnouncement(3),
    ]
    announcementStore.loading = true

    const rawAuthState = toRaw(pinia.state.value.auth) as unknown as {
      runMode: Ref<'standard' | 'simple'>
    }
    toRaw(rawAuthState.runMode).value = 'simple'

    expect(summary.hasUser.value).toBe(true)
    expect(summary.userId.value).toBe(7)
    expect(summary.role.value).toBe('admin')
    expect(summary.isAdmin.value).toBe(true)
    expect(summary.isSimpleMode.value).toBe(true)
    expect(summary.email.value).toBe('riley@example.com')
    expect(summary.avatarUrl.value).toBe('https://example.com/avatar.png')
    expect(summary.displayName.value).toBe('Riley Quinn')
    expect(summary.initials.value).toBe('R')
    expect(summary.availableBalance.value).toBe(12.5)
    expect(summary.frozenBalance.value).toBe(2.5)
    expect(summary.totalBalance.value).toBe(15)
    expect(summary.hasFrozenBalance.value).toBe(true)
    expect(summary.activeSubscriptionCount.value).toBe(3)
    expect(summary.hasActiveSubscriptions.value).toBe(true)
    expect(summary.subscriptionsLoading.value).toBe(true)
    expect(summary.subscriptionsLoaded.value).toBe(false)
    expect(summary.primarySubscription.value).toEqual({
      id: 2,
      name: 'Pro',
      expiresAt: '2026-08-01T00:00:00Z',
    })
    expect(summary.nearestSubscriptionExpiry.value).toBe('2026-08-01T00:00:00Z')
    expect(summary.unreadAnnouncementCount.value).toBe(2)
    expect(summary.announcementsLoading.value).toBe(true)

    authStore.user = makeUser({
      username: '',
      email: 'fallback@example.com',
      balance: Number.NaN,
      frozen_balance: Number.POSITIVE_INFINITY,
    })
    subscriptionStore.activeSubscriptions = []
    announcementStore.announcements = [makeAnnouncement(1, '2026-07-10T00:00:00Z')]

    expect(summary.displayName.value).toBe('fallback')
    expect(summary.initials.value).toBe('F')
    expect(summary.availableBalance.value).toBe(0)
    expect(summary.frozenBalance.value).toBe(0)
    expect(summary.totalBalance.value).toBe(0)
    expect(summary.hasFrozenBalance.value).toBe(false)
    expect(summary.activeSubscriptionCount.value).toBe(0)
    expect(summary.primarySubscription.value).toBeNull()
    expect(summary.nearestSubscriptionExpiry.value).toBeNull()
    expect(summary.unreadAnnouncementCount.value).toBe(0)
  })

  it('does not fetch data or schedule polling when the summary is created or read', () => {
    const subscriptionStore = useSubscriptionStore()
    const announcementStore = useAnnouncementStore()
    const fetchSubscriptions = vi.spyOn(
      subscriptionStore,
      'fetchActiveSubscriptions',
    )
    const fetchAnnouncements = vi.spyOn(announcementStore, 'fetchAnnouncements')
    const setIntervalSpy = vi.spyOn(globalThis, 'setInterval')

    const summary = useAccountSummary({ subscriptionStore })

    expect(summary.displayName.value).toBe('')
    expect(summary.activeSubscriptionCount.value).toBe(0)
    expect(summary.unreadAnnouncementCount.value).toBe(0)
    expect(fetchSubscriptions).not.toHaveBeenCalled()
    expect(fetchAnnouncements).not.toHaveBeenCalled()
    expect(setIntervalSpy).not.toHaveBeenCalled()
  })
})

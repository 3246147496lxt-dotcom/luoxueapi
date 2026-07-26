import { computed } from 'vue'
import { useAnnouncementStore } from '@/stores/announcements'
import { useAuthStore } from '@/stores/auth'
import { useSubscriptionStore } from '@/stores/subscriptions'
import type { UserSubscription } from '@/types'

export interface AccountSubscriptionSummary {
  readonly id: number
  readonly name: string | null
  readonly expiresAt: string | null
}

function normalizedText(value: string | null | undefined): string {
  return value?.trim() ?? ''
}

function finiteNumber(value: unknown): number {
  const number = Number(value ?? 0)
  return Number.isFinite(number) ? number : 0
}

function validExpiryTimestamp(value: string | null): number | null {
  if (!value) return null
  const timestamp = Date.parse(value)
  return Number.isFinite(timestamp) ? timestamp : null
}

function selectPrimarySubscription(
  subscriptions: readonly UserSubscription[],
): UserSubscription | null {
  const first = subscriptions[0] ?? null
  let nearest = first
  let nearestTimestamp = Number.POSITIVE_INFINITY

  for (const subscription of subscriptions) {
    const timestamp = validExpiryTimestamp(subscription.expires_at)
    if (timestamp !== null && timestamp < nearestTimestamp) {
      nearest = subscription
      nearestTimestamp = timestamp
    }
  }

  return nearest
}

/**
 * Passive, read-only account data for shell UI.
 *
 * Global app orchestration owns fetching and polling. This composable only
 * derives immutable primitives/snapshots from the three existing stores.
 */
export function useAccountSummary() {
  const authStore = useAuthStore()
  const subscriptionStore = useSubscriptionStore()
  const announcementStore = useAnnouncementStore()

  const hasUser = computed(() => authStore.user !== null)
  const userId = computed(() => authStore.user?.id ?? null)
  const role = computed(() => authStore.user?.role ?? null)
  const isAuthenticated = computed(() => authStore.isAuthenticated)
  const isAdmin = computed(() => authStore.isAdmin)
  const isSimpleMode = computed(() => authStore.isSimpleMode)

  const email = computed(() => normalizedText(authStore.user?.email))
  const avatarUrl = computed(() => normalizedText(authStore.user?.avatar_url))
  const displayName = computed(() => {
    const username = normalizedText(authStore.user?.username)
    if (username) return username

    const localPart = email.value.split('@')[0]?.trim()
    return localPart || ''
  })
  const initials = computed(() => {
    const firstCharacter = Array.from(displayName.value)[0]
    return firstCharacter?.toLocaleUpperCase() ?? ''
  })

  const availableBalance = computed(() => finiteNumber(authStore.user?.balance))
  const frozenBalance = computed(() => finiteNumber(authStore.user?.frozen_balance))
  const totalBalance = computed(() => availableBalance.value + frozenBalance.value)
  const hasFrozenBalance = computed(() => frozenBalance.value > 0)

  const activeSubscriptionCount = computed(
    () => subscriptionStore.activeSubscriptions.length,
  )
  const hasActiveSubscriptions = computed(
    () => subscriptionStore.hasActiveSubscriptions,
  )
  const subscriptionsLoading = computed(() => subscriptionStore.loading)
  const subscriptionsLoaded = computed(() => subscriptionStore.loaded)
  const primarySubscription = computed<Readonly<AccountSubscriptionSummary> | null>(() => {
    const subscription = selectPrimarySubscription(
      subscriptionStore.activeSubscriptions,
    )
    if (!subscription) return null

    const expiresAt = validExpiryTimestamp(subscription.expires_at) === null
      ? null
      : subscription.expires_at

    return {
      id: subscription.id,
      name: normalizedText(subscription.group?.name) || null,
      expiresAt,
    }
  })
  const nearestSubscriptionExpiry = computed(
    () => primarySubscription.value?.expiresAt ?? null,
  )

  const unreadAnnouncementCount = computed(
    () => announcementStore.unreadCount,
  )
  const announcementsLoading = computed(() => announcementStore.loading)

  return {
    hasUser,
    userId,
    role,
    isAuthenticated,
    isAdmin,
    isSimpleMode,
    email,
    avatarUrl,
    displayName,
    initials,
    availableBalance,
    frozenBalance,
    totalBalance,
    hasFrozenBalance,
    activeSubscriptionCount,
    hasActiveSubscriptions,
    subscriptionsLoading,
    subscriptionsLoaded,
    primarySubscription,
    nearestSubscriptionExpiry,
    unreadAnnouncementCount,
    announcementsLoading,
  } as const
}

export type AccountSummary = ReturnType<typeof useAccountSummary>

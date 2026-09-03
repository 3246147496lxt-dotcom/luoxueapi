/**
 * Workspace user profile facade.
 *
 * Identity remains owned by the auth store and membership remains owned by the
 * subscription store. This store exposes one computed profile for shell UI and
 * owns the lifecycle of profile synchronization without copying either source.
 */

import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { useAuthStore } from '@/stores/auth'
import { useSubscriptionStore } from '@/stores/subscriptions'
import type { UserSubscription } from '@/types'

export type WorkspaceUserPlanState = 'pending' | 'free' | 'active'

export interface WorkspacePrimarySubscription {
  readonly id: number
  readonly name: string | null
  readonly expiresAt: string | null
}

export interface WorkspaceUserProfile {
  readonly id: number
  readonly username: string
  readonly displayName: string
  readonly email: string
  readonly avatarUrl: string
  readonly initials: string
  readonly role: 'admin' | 'user'
  readonly availableBalance: number
  readonly frozenBalance: number
  readonly currentPlan: Readonly<WorkspaceCurrentPlan>
  readonly currentPlanLevel: string | null
  readonly planState: WorkspaceUserPlanState
}

export interface WorkspaceCurrentPlan {
  readonly state: WorkspaceUserPlanState
  readonly name: string | null
  readonly activeCount: number
  readonly expiresAt: string | null
}

interface PendingOperation {
  userId: number
  promise: Promise<void>
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

function firstRejectedResult(
  results: readonly PromiseSettledResult<unknown>[],
): PromiseRejectedResult | undefined {
  return results.find(
    (result): result is PromiseRejectedResult => result.status === 'rejected',
  )
}

export const useUserProfileStore = defineStore('userProfile', () => {
  const authStore = useAuthStore()
  const subscriptionStore = useSubscriptionStore()

  const subscriptionOwnerId = ref<number | null>(null)
  let initializedUserId: number | null = null
  let operationGeneration = 0
  let initialization: PendingOperation | null = null
  let refresh: PendingOperation | null = null

  const ownsCurrentSubscriptions = computed(() => (
    subscriptionOwnerId.value !== null
    && subscriptionOwnerId.value === (authStore.user?.id ?? null)
  ))
  const subscriptionsLoaded = computed(() => (
    ownsCurrentSubscriptions.value && subscriptionStore.loaded
  ))
  const subscriptionsLoading = computed(() => (
    ownsCurrentSubscriptions.value && subscriptionStore.loading
  ))
  const activeSubscriptionCount = computed(
    () => ownsCurrentSubscriptions.value
      ? subscriptionStore.activeSubscriptions.length
      : 0,
  )
  const primarySubscription = computed<Readonly<WorkspacePrimarySubscription> | null>(() => {
    if (!ownsCurrentSubscriptions.value) return null
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
  const planState = computed<WorkspaceUserPlanState>(() => {
    if (!subscriptionsLoaded.value) return 'pending'
    return activeSubscriptionCount.value > 0 ? 'active' : 'free'
  })
  const currentPlanLevel = computed(
    () => planState.value === 'active'
      ? primarySubscription.value?.name ?? null
      : null,
  )
  const currentPlan = computed<Readonly<WorkspaceCurrentPlan>>(() => ({
    state: planState.value,
    name: currentPlanLevel.value,
    activeCount: activeSubscriptionCount.value,
    expiresAt: planState.value === 'active'
      ? primarySubscription.value?.expiresAt ?? null
      : null,
  }))

  const profile = computed<Readonly<WorkspaceUserProfile> | null>(() => {
    const user = authStore.user
    if (!user) return null

    const username = normalizedText(user.username)
    const email = normalizedText(user.email)
    const displayName = username || email.split('@')[0]?.trim() || ''
    const firstCharacter = Array.from(displayName)[0]

    return {
      id: user.id,
      username,
      displayName,
      email,
      avatarUrl: normalizedText(user.avatar_url),
      initials: firstCharacter?.toLocaleUpperCase() ?? '',
      role: user.role,
      availableBalance: finiteNumber(user.balance),
      frozenBalance: finiteNumber(user.frozen_balance),
      currentPlan: currentPlan.value,
      currentPlanLevel: currentPlanLevel.value,
      planState: planState.value,
    }
  })

  function currentUserId(): number | null {
    return authStore.user?.id ?? null
  }

  function invalidateOperations(): void {
    operationGeneration += 1
    initializedUserId = null
    initialization = null
    refresh = null
  }

  function preparePrincipal(userId: number): void {
    if (subscriptionOwnerId.value === null) {
      subscriptionOwnerId.value = userId
      return
    }
    if (subscriptionOwnerId.value === userId) return

    invalidateOperations()
    subscriptionStore.clear()
    subscriptionOwnerId.value = userId
  }

  function isCurrentOperation(userId: number, generation: number): boolean {
    return subscriptionOwnerId.value === userId
      && currentUserId() === userId
      && operationGeneration === generation
  }

  /** Preload membership once for the current authenticated principal. */
  function initialize(): Promise<void> {
    const userId = currentUserId()
    if (userId === null) {
      clear()
      return Promise.resolve()
    }

    preparePrincipal(userId)

    if (initializedUserId === userId && subscriptionStore.loaded) {
      return Promise.resolve()
    }
    if (refresh?.userId === userId) return refresh.promise
    if (initialization?.userId === userId) return initialization.promise

    if (subscriptionStore.loaded) {
      initializedUserId = userId
      return Promise.resolve()
    }

    const generation = operationGeneration
    const record: PendingOperation = {
      userId,
      promise: Promise.resolve(),
    }
    const operation = subscriptionStore.fetchActiveSubscriptions()
      .then(() => {
        if (isCurrentOperation(userId, generation)) {
          initializedUserId = userId
        }
      })

    record.promise = operation.finally(() => {
      if (initialization === record) initialization = null
    })
    initialization = record
    return record.promise
  }

  /**
   * Explicitly synchronize identity and membership.
   *
   * Calls for one principal share a single in-flight operation. A forced
   * refresh supersedes an initialization request, so an upgrade cannot reuse a
   * subscription response dispatched before the upgrade completed.
   */
  function refreshProfile(): Promise<void> {
    const userId = currentUserId()
    if (userId === null) {
      clear()
      return Promise.reject(new Error('Cannot refresh a profile without a user'))
    }

    preparePrincipal(userId)
    if (refresh?.userId === userId) return refresh.promise

    operationGeneration += 1
    const generation = operationGeneration
    initialization = null

    const record: PendingOperation = {
      userId,
      promise: Promise.resolve(),
    }
    const operation = Promise.allSettled([
      authStore.refreshUser(),
      subscriptionStore.fetchActiveSubscriptions(true),
    ]).then((results) => {
      const subscriptionResult = results[1]
      if (
        subscriptionResult.status === 'fulfilled'
        && isCurrentOperation(userId, generation)
      ) {
        initializedUserId = userId
      }

      const rejection = firstRejectedResult(results)
      if (rejection) throw rejection.reason
    })

    record.promise = operation.finally(() => {
      if (refresh === record) refresh = null
    })
    refresh = record
    return record.promise
  }

  function syncAfterUpgrade(): Promise<void> {
    return refreshProfile()
  }

  /** Clear membership ownership and invalidate every pending profile action. */
  function clear(): void {
    invalidateOperations()
    subscriptionOwnerId.value = null
    subscriptionStore.clear()
  }

  return {
    profile,
    subscriptionsLoaded,
    subscriptionsLoading,
    activeSubscriptionCount,
    primarySubscription,
    planState,
    currentPlan,
    currentPlanLevel,
    initialize,
    refreshProfile,
    syncAfterUpgrade,
    clear,
    reset: clear,
  }
})

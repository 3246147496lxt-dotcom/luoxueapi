import { ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useUserMembership } from '../useUserMembership'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, values?: Record<string, unknown>) => (
      values ? `${key}:${JSON.stringify(values)}` : key
    ),
  }),
}))

describe('useUserMembership', () => {
  const subscriptionsLoaded = ref(false)
  const activeSubscriptionCount = ref(0)
  const primarySubscription = ref<{
    id: number
    name: string | null
    expiresAt: string | null
  } | null>(null)

  beforeEach(() => {
    subscriptionsLoaded.value = false
    activeSubscriptionCount.value = 0
    primarySubscription.value = null
  })

  function createPresentation() {
    return useUserMembership({
      subscriptionsLoaded,
      activeSubscriptionCount,
      primarySubscription,
    })
  }

  it('keeps an unresolved subscription state out of both Free and the product bar', () => {
    const membership = createPresentation()

    expect(membership.state.value).toBe('pending')
    expect(membership.accountPlanLabel.value).toBe('accountDock.subscriptionLoading')
    expect(membership.productPlanLabel.value).toBeNull()
    expect(membership.hasActiveMembership.value).toBe(false)
  })

  it('maps a confirmed empty subscription list to Free', () => {
    const membership = createPresentation()
    subscriptionsLoaded.value = true

    expect(membership.state.value).toBe('free')
    expect(membership.accountPlanLabel.value).toBe('accountDock.free')
    expect(membership.productPlanLabel.value).toBeNull()
  })

  it('uses the real active plan name without consulting wallet balance', () => {
    const unrelatedWalletBalance = ref(0)
    const membership = createPresentation()
    subscriptionsLoaded.value = true
    activeSubscriptionCount.value = 1
    primarySubscription.value = { id: 7, name: 'Pro', expiresAt: '2026-09-01T00:00:00Z' }

    expect(membership.state.value).toBe('active')
    expect(membership.accountPlanLabel.value).toBe('Pro')
    expect(membership.productPlanLabel.value).toBe('Pro')

    unrelatedWalletBalance.value = 10_000
    expect(membership.accountPlanLabel.value).toBe('Pro')
  })

  it('uses a subscription-count fallback only when an active plan has no name', () => {
    const membership = createPresentation()
    subscriptionsLoaded.value = true
    activeSubscriptionCount.value = 2
    primarySubscription.value = { id: 7, name: null, expiresAt: '2026-09-01T00:00:00Z' }

    const fallback = 'accountDock.activeSubscriptions:{"count":2}'
    expect(membership.accountPlanLabel.value).toBe(fallback)
    expect(membership.productPlanLabel.value).toBe(fallback)
  })
})

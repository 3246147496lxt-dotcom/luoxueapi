import { computed, type ComputedRef, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AccountSubscriptionSummary } from './useAccountSummary'

export type UserMembershipState = 'pending' | 'free' | 'active'

export interface UserMembershipSource {
  subscriptionsLoaded: Readonly<Ref<boolean>>
  activeSubscriptionCount: Readonly<Ref<number>>
  primarySubscription: Readonly<Ref<Readonly<AccountSubscriptionSummary> | null>>
}

export interface UserMembershipPresentation {
  state: ComputedRef<UserMembershipState>
  hasActiveMembership: ComputedRef<boolean>
  accountPlanLabel: ComputedRef<string>
  productPlanLabel: ComputedRef<string | null>
}

/**
 * Membership presentation for end-user surfaces.
 *
 * This deliberately consumes subscription state only. Wallet balance, credits,
 * and metered billing are separate concepts and must not determine membership.
 */
export function useUserMembership(
  source: UserMembershipSource,
): UserMembershipPresentation {
  const { t } = useI18n()

  const state = computed<UserMembershipState>(() => {
    if (!source.subscriptionsLoaded.value) return 'pending'
    return source.activeSubscriptionCount.value > 0 ? 'active' : 'free'
  })

  const productPlanLabel = computed<string | null>(() => {
    if (state.value !== 'active') return null

    return source.primarySubscription.value?.name
      || t('accountDock.activeSubscriptions', {
        count: source.activeSubscriptionCount.value,
      })
  })

  const accountPlanLabel = computed(() => {
    if (state.value === 'pending') {
      return t('accountDock.subscriptionLoading')
    }
    if (state.value === 'free') {
      return t('accountDock.free')
    }
    return productPlanLabel.value || t('accountDock.free')
  })

  const hasActiveMembership = computed(() => state.value === 'active')

  return {
    state,
    hasActiveMembership,
    accountPlanLabel,
    productPlanLabel,
  }
}

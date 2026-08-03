export interface AccountPanelSummary {
  displayName: string
  email: string
  initials: string
  avatarUrl: string
  frozenBalance: number
  formattedAvailableBalance: string
  formattedFrozenBalance: string
  activeSubscriptionCount: number
  subscriptionsLoaded: boolean
}

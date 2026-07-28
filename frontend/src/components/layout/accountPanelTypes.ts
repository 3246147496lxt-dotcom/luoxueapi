export type AccountPanelIcon =
  | 'wallet'
  | 'creditCard'
  | 'destinationHome'
  | 'destinationModels'
  | 'destinationContact'
  | 'destinationDocument'

export interface AccountPanelLink {
  id: string
  label: string
  to: string
  icon: AccountPanelIcon
}

export interface AccountResourceLink {
  id: string
  label: string
  href: string
  icon: AccountPanelIcon
}

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

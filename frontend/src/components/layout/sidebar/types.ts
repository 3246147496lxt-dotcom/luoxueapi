import type { CustomMenuItem } from '@/types'

export interface NavItem {
  path: string
  /** Additional route paths that should share this item's selected state. */
  activePaths?: readonly string[]
  label: string
  icon: unknown
  iconSvg?: string
  href?: string
  trailingIcon?: 'destinationArrowUpRight'
  hideInSimpleMode?: boolean
  children?: NavItem[]
  /** When true, the parent only toggles its children and never navigates. */
  expandOnly?: boolean
  /** `false` hides the item; `undefined` keeps it visible while settings load. */
  featureFlag?: () => boolean | undefined
}

export type UserNavSectionId = 'workbench' | 'api' | 'account'
export type AdminNavSectionId = 'overview' | 'business' | 'operations' | 'system'
export type NavSectionId = UserNavSectionId | AdminNavSectionId

export interface NavSection<TId extends NavSectionId = NavSectionId> {
  id: TId
  label?: string
  items: NavItem[]
}

export type AdminNavSection = NavSection<AdminNavSectionId>

export type SidebarSupportIcon =
  | 'home'
  | 'document'
  | 'destinationHome'
  | 'destinationModels'
  | 'destinationContact'
  | 'destinationDocument'

export interface SidebarSupportLink {
  id: string
  label: string
  href: string
  icon: SidebarSupportIcon
}

export interface UserNavigationIcons {
  dashboard: unknown
  model: unknown
  batchImage: unknown
  chart: unknown
  channel: unknown
  signal: unknown
  skillMarket: unknown
  quotaViewer: unknown
  rechargeSubscription: unknown
  creditCard: unknown
  orderList: unknown
  users: unknown
  user: unknown
}

export interface AdminNavigationIcons {
  dashboard: unknown
  chart: unknown
  opsChart: unknown
  usageChart: unknown
  users: unknown
  folder: unknown
  server: unknown
  channel: unknown
  priceTag: unknown
  signal: unknown
  skillMarket: unknown
  creditCard: unknown
  order: unknown
  ticket: unknown
  gift: unknown
  shield: unknown
  diagnostics: unknown
  book: unknown
  cog: unknown
}

export interface UserNavigationContext {
  t: (key: string) => string
  simpleMode: boolean
  canUseBatchImage: () => boolean | undefined
  customMenuItems: readonly CustomMenuItem[]
  icons: UserNavigationIcons
}

export interface AdminNavigationContext {
  t: (key: string) => string
  simpleMode: boolean
  isOpsShell: boolean
  opsMonitoringEnabled: () => boolean | undefined
  adminPaymentEnabled: () => boolean | undefined
  customMenuItems: readonly CustomMenuItem[]
  icons: AdminNavigationIcons
}

export interface AdminNavigationDefinition {
  buildItems(context: AdminNavigationContext): NavItem[]
  buildSections(
    items: readonly NavItem[],
    t: (key: string) => string,
    isOpsShell: boolean,
  ): AdminNavSection[]
  activeSectionId(path: string): AdminNavSectionId | null
  defaultExpandedSections(): Set<AdminNavSectionId>
  loadExpandedSections(): Set<AdminNavSectionId>
  persistExpandedSections(sections: ReadonlySet<AdminNavSectionId>): void
  sectionPanelId(sectionId: AdminNavSectionId): string
}

/** Recursively remove entries whose feature flag explicitly resolves false. */
export function applyFeatureFlags(items: NavItem[]): NavItem[] {
  const visible: NavItem[] = []
  for (const item of items) {
    if (item.featureFlag?.() === false) continue
    visible.push(item.children
      ? { ...item, children: applyFeatureFlags(item.children) }
      : item)
  }
  return visible
}

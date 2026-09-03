import accountPoolIconSvg from '@/assets/icons/account-pool.svg?raw'
import auditLogIconSvg from '@/assets/icons/audit-log.svg?raw'
import keyOutlineIconSvg from '@/assets/icons/key-outline.svg?raw'
import modelMarketplaceIconSvg from '@/assets/icons/model-marketplace.svg?raw'
import NotificationIcon from '@/components/icons/NotificationIcon.vue'
import { FeatureFlags, makeSidebarFlag } from '@/utils/featureFlags'
import {
  applyFeatureFlags,
  type AdminNavigationContext,
  type AdminNavigationDefinition,
  type AdminNavSection,
  type AdminNavSectionId,
  type NavItem,
} from './types'

const ADMIN_NAV_SECTION_STORAGE_KEY = 'app-sidebar-admin-expanded-sections-v2'

const ADMIN_NAV_SECTION_BY_PATH: Record<string, AdminNavSectionId> = {
  '/admin/dashboard': 'overview',
  '/admin/ops': 'overview',
  '/admin/usage': 'overview',
  '/admin/users': 'business',
  '/admin/groups': 'business',
  '/admin/accounts': 'business',
  '/admin/proxies': 'business',
  '/admin/channels': 'business',
  '/admin/model-catalog': 'business',
  '/admin/skills': 'business',
  '/keys': 'business',
  '/admin/subscriptions': 'operations',
  '/admin/orders': 'operations',
  '/admin/redeem': 'operations',
  '/admin/promo-codes': 'operations',
  '/admin/affiliates': 'operations',
  '/admin/announcements': 'operations',
  '/admin/risk-control': 'system',
  '/admin/audit-logs': 'system',
  '/admin/desktop-diagnostics': 'system',
  '/admin/documentation': 'system',
  '/admin/settings': 'system',
}

const ADMIN_NAV_SECTION_CONFIG: Array<{ id: AdminNavSectionId; labelKey: string }> = [
  { id: 'overview', labelKey: 'nav.adminSections.overview' },
  { id: 'business', labelKey: 'nav.adminSections.business' },
  { id: 'operations', labelKey: 'nav.adminSections.operations' },
  { id: 'system', labelKey: 'nav.adminSections.system' },
]

const ADMIN_NAV_SECTION_IDS = new Set<AdminNavSectionId>(
  ADMIN_NAV_SECTION_CONFIG.map(({ id }) => id),
)

const flagChannelMonitor = makeSidebarFlag(FeatureFlags.channelMonitor)
const flagAffiliate = makeSidebarFlag(FeatureFlags.affiliate)
const flagRiskControl = makeSidebarFlag(FeatureFlags.riskControl)

function buildItems(context: AdminNavigationContext): NavItem[] {
  const { t, icons } = context
  const customItems = [...context.customMenuItems]
    .filter((item) => item.visibility === 'admin')
    .sort((a, b) => a.sort_order - b.sort_order)

  const baseItems: NavItem[] = [
    {
      path: '/admin/dashboard',
      label: t('nav.adminDashboard'),
      // Keep the /admin/ops outline glyph on every administrator route. The
      // selected state is route-owned, but the icon shape is not.
      icon: icons.dashboard,
    },
    {
      path: '/admin/ops',
      label: t('nav.ops'),
      icon: icons.opsChart,
      featureFlag: context.opsMonitoringEnabled,
    },
    {
      path: '/admin/usage',
      label: t('nav.adminUsage'),
      icon: icons.usageChart,
    },
    { path: '/admin/users', label: t('nav.users'), icon: icons.users, hideInSimpleMode: true },
    { path: '/admin/groups', label: t('nav.groups'), icon: icons.folder, hideInSimpleMode: true },
    { path: '/admin/accounts', label: t('nav.accounts'), icon: null, iconSvg: accountPoolIconSvg },
    { path: '/admin/proxies', label: t('nav.proxies'), icon: icons.server },
    {
      path: '/admin/channels',
      label: t('nav.channelManagement'),
      icon: icons.channel,
      hideInSimpleMode: true,
      expandOnly: true,
      children: [
        { path: '/admin/channels/pricing', label: t('nav.channelPricing'), icon: icons.priceTag },
        {
          path: '/admin/channels/monitor',
          label: t('nav.channelMonitor'),
          icon: icons.signal,
          featureFlag: flagChannelMonitor,
        },
      ],
    },
    {
      path: '/admin/model-catalog',
      label: t('nav.modelCatalog'),
      icon: null,
      iconSvg: modelMarketplaceIconSvg,
      hideInSimpleMode: true,
    },
    { path: '/admin/skills', label: t('admin.skills.title'), icon: icons.skillMarket, hideInSimpleMode: true },
    { path: '/admin/subscriptions', label: t('nav.subscriptions'), icon: icons.creditCard, hideInSimpleMode: true },
    {
      path: '/admin/orders',
      label: t('nav.orderManagement'),
      icon: icons.order,
      hideInSimpleMode: true,
      expandOnly: true,
      featureFlag: context.adminPaymentEnabled,
      children: [
        { path: '/admin/orders/dashboard', label: t('nav.paymentDashboard'), icon: icons.chart },
        { path: '/admin/orders', label: t('nav.orderManagement'), icon: icons.order },
        { path: '/admin/orders/plans', label: t('nav.paymentPlans'), icon: icons.creditCard },
      ],
    },
    { path: '/admin/redeem', label: t('nav.redeemCodes'), icon: icons.ticket, hideInSimpleMode: true },
    { path: '/admin/promo-codes', label: t('nav.promoCodes'), icon: icons.gift, hideInSimpleMode: true },
    {
      path: '/admin/affiliates',
      label: t('nav.affiliateManagement'),
      icon: icons.users,
      hideInSimpleMode: true,
      expandOnly: true,
      featureFlag: flagAffiliate,
      children: [
        { path: '/admin/affiliates/invites', label: t('nav.affiliateInviteRecords'), icon: icons.users },
        { path: '/admin/affiliates/rebates', label: t('nav.affiliateRebateRecords'), icon: icons.order },
        { path: '/admin/affiliates/transfers', label: t('nav.affiliateTransferRecords'), icon: icons.creditCard },
      ],
    },
    { path: '/admin/announcements', label: t('nav.announcementManagement'), icon: NotificationIcon },
    {
      path: '/admin/risk-control',
      label: t('nav.riskControl'),
      icon: icons.shield,
      hideInSimpleMode: true,
      featureFlag: flagRiskControl,
    },
    {
      path: '/admin/audit-logs',
      label: t('nav.auditLogs'),
      icon: null,
      iconSvg: auditLogIconSvg,
      hideInSimpleMode: true,
    },
    {
      path: '/admin/desktop-diagnostics',
      label: t('nav.desktopDiagnostics'),
      icon: icons.diagnostics,
      hideInSimpleMode: true,
    },
    {
      path: '/admin/documentation',
      label: t('nav.documentationManagement'),
      icon: icons.book,
      hideInSimpleMode: true,
    },
  ]

  const visible = applyFeatureFlags(baseItems)
  const items = context.simpleMode
    ? visible.filter((item) => !item.hideInSimpleMode)
    : visible

  if (context.simpleMode) {
    items.push({ path: '/keys', label: t('nav.apiKeys'), icon: null, iconSvg: keyOutlineIconSvg })
  }
  items.push(...customItems.map((item): NavItem => ({
    path: `/custom/${item.id}`,
    label: item.label,
    icon: null,
    iconSvg: item.icon_svg,
  })))
  items.push({ path: '/admin/settings', label: t('nav.systemSettings'), icon: icons.cog })
  return items
}

function defaultExpandedSections(): Set<AdminNavSectionId> {
  return new Set(ADMIN_NAV_SECTION_CONFIG.map(({ id }) => id))
}

function loadExpandedSections(): Set<AdminNavSectionId> {
  if (typeof window === 'undefined') return defaultExpandedSections()

  try {
    const raw = window.localStorage.getItem(ADMIN_NAV_SECTION_STORAGE_KEY)
    if (raw === null) return defaultExpandedSections()
    const stored: unknown = JSON.parse(raw)
    if (!Array.isArray(stored)) return defaultExpandedSections()
    return new Set(stored.filter((id): id is AdminNavSectionId => (
      typeof id === 'string' && ADMIN_NAV_SECTION_IDS.has(id as AdminNavSectionId)
    )))
  } catch {
    return defaultExpandedSections()
  }
}

function persistExpandedSections(sections: ReadonlySet<AdminNavSectionId>): void {
  if (typeof window === 'undefined') return
  try {
    const orderedIds = ADMIN_NAV_SECTION_CONFIG
      .map(({ id }) => id)
      .filter((id) => sections.has(id))
    window.localStorage.setItem(ADMIN_NAV_SECTION_STORAGE_KEY, JSON.stringify(orderedIds))
  } catch {
    // Navigation remains usable when storage is unavailable.
  }
}

function activeSectionId(pathname: string): AdminNavSectionId | null {
  const matchingPath = Object.keys(ADMIN_NAV_SECTION_BY_PATH)
    .sort((a, b) => b.length - a.length)
    .find((path) => pathname === path || pathname.startsWith(`${path}/`))
  return matchingPath ? ADMIN_NAV_SECTION_BY_PATH[matchingPath] : null
}

function buildSections(
  items: readonly NavItem[],
  t: (key: string) => string,
): AdminNavSection[] {
  const grouped = new Map<AdminNavSectionId, NavItem[]>(
    ADMIN_NAV_SECTION_CONFIG.map(({ id }) => [id, []]),
  )
  for (const item of items) {
    const sectionId = item.path.startsWith('/custom/')
      ? 'system'
      : (ADMIN_NAV_SECTION_BY_PATH[item.path] ?? 'system')
    grouped.get(sectionId)?.push(item)
  }
  return ADMIN_NAV_SECTION_CONFIG
    .map(({ id, labelKey }) => ({
      id,
      label: t(labelKey),
      items: grouped.get(id) ?? [],
    }))
    .filter((section) => section.items.length > 0)
}

export const adminNavigationDefinition: AdminNavigationDefinition = {
  buildItems,
  buildSections,
  activeSectionId,
  defaultExpandedSections,
  loadExpandedSections,
  persistExpandedSections,
  sectionPanelId: (sectionId) => `sidebar-admin-${sectionId}-items`,
}

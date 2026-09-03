import keyOutlineIconSvg from '@/assets/icons/key-outline.svg?raw'
import skillMarketIconSvg from '@/assets/icons/skill.svg?raw'
import { FeatureFlags, makeSidebarFlag } from '@/utils/featureFlags'
import { ACCOUNT_DESTINATION_PATHS } from '@/navigation/shellDestinations'
import {
  applyFeatureFlags,
  type NavItem,
  type NavSection,
  type UserNavSectionId,
  type UserNavigationContext,
} from './types'

const flagChannelMonitor = makeSidebarFlag(FeatureFlags.channelMonitor)
const flagPayment = makeSidebarFlag(FeatureFlags.payment)
const flagAvailableChannels = makeSidebarFlag(FeatureFlags.availableChannels)
const flagPublicModelCatalog = makeSidebarFlag(FeatureFlags.publicModelCatalog)
const flagSkillMarketplace = makeSidebarFlag(FeatureFlags.skillMarketplace)

const USER_WORKBENCH_PATHS = new Set<string>([
  '/dashboard',
  '/models',
  '/skills',
])
const USER_API_PATHS = new Set<string>([
  '/keys',
  '/usage',
])
const USER_ACCOUNT_PATHS = new Set<string>([
  ACCOUNT_DESTINATION_PATHS.subscriptions,
  ACCOUNT_DESTINATION_PATHS.pricing,
  ACCOUNT_DESTINATION_PATHS.orders,
  '/affiliate',
])

function buildUserItems(context: UserNavigationContext): NavItem[] {
  const { t, icons } = context
  const customItems = [...context.customMenuItems]
    .filter((item) => item.visibility === 'user')
    .sort((a, b) => a.sort_order - b.sort_order)

  const items: NavItem[] = [
    { path: '/dashboard', label: t('nav.dashboard'), icon: icons.dashboard },
    {
      path: '/models',
      label: t('nav.modelCenter'),
      icon: icons.model,
      featureFlag: flagPublicModelCatalog,
    },
    {
      path: '/skills',
      label: t('skills.navLabel'),
      icon: null,
      iconSvg: skillMarketIconSvg,
      featureFlag: flagSkillMarketplace,
    },
    { path: '/keys', label: t('nav.apiKeys'), icon: null, iconSvg: keyOutlineIconSvg },
    {
      path: '/batch-image',
      label: t('nav.batchImage'),
      icon: icons.batchImage,
      hideInSimpleMode: true,
      featureFlag: context.canUseBatchImage,
    },
    { path: '/usage', label: t('nav.usage'), icon: icons.chart, hideInSimpleMode: true },
    {
      path: '/available-channels',
      label: t('nav.availableChannels'),
      icon: icons.channel,
      hideInSimpleMode: true,
      featureFlag: flagAvailableChannels,
    },
    { path: '/monitor', label: t('nav.serviceStatusNav'), icon: icons.signal, featureFlag: flagChannelMonitor },
    {
      path: ACCOUNT_DESTINATION_PATHS.quotaViewer,
      label: t('quotaViewerLanding.meta.title'),
      icon: icons.quotaViewer,
      trailingIcon: 'destinationArrowUpRight',
    },
    {
      path: ACCOUNT_DESTINATION_PATHS.subscriptions,
      activePaths: [ACCOUNT_DESTINATION_PATHS.wallet],
      label: t('nav.balanceAndMembership'),
      icon: icons.rechargeSubscription,
    },
    {
      path: ACCOUNT_DESTINATION_PATHS.pricing,
      label: t('nav.memberSubscription'),
      icon: icons.creditCard,
      hideInSimpleMode: true,
    },
    {
      path: ACCOUNT_DESTINATION_PATHS.orders,
      label: t('nav.orders'),
      icon: icons.orderList,
      hideInSimpleMode: true,
      featureFlag: flagPayment,
    },
    {
      // Keep the entry discoverable in the standard Work rail; the affiliate
      // view/API continues to enforce the backend capability for actions.
      path: '/affiliate',
      label: t('nav.affiliate'),
      icon: icons.users,
      hideInSimpleMode: true,
    },
    { path: ACCOUNT_DESTINATION_PATHS.profile, label: t('nav.profile'), icon: icons.user },
    ...customItems.map((item): NavItem => ({
      path: `/custom/${item.id}`,
      label: item.label,
      icon: null,
      iconSvg: item.icon_svg,
    })),
  ]

  const visible = applyFeatureFlags(items)
  return context.simpleMode
    ? visible.filter((item) => !item.hideInSimpleMode)
    : visible
}

export function buildUserNavigation(context: UserNavigationContext): NavSection<UserNavSectionId>[] {
  const items = buildUserItems(context)
  const baseSections: NavSection<UserNavSectionId>[] = [
    {
      id: 'workbench',
      label: context.t('nav.userSections.workbench'),
      items: items.filter((item) => USER_WORKBENCH_PATHS.has(item.path)),
    },
    {
      id: 'api',
      label: context.t('nav.userSections.api'),
      items: items.filter((item) => USER_API_PATHS.has(item.path)),
    },
    {
      id: 'account',
      label: context.t('nav.userSections.account'),
      items: items.filter((item) => USER_ACCOUNT_PATHS.has(item.path)),
    },
  ]
  return baseSections.filter((section) => section.items.length > 0)
}

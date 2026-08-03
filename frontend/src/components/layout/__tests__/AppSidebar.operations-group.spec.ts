import { nextTick, toRaw, type Ref } from 'vue'
import { mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia, type Pinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { PublicSettings, User } from '@/types'
import enCommon from '@/i18n/locales/en/common'
import zhCommon from '@/i18n/locales/zh/common'

const routeState = vi.hoisted(() => ({ path: '/admin/dashboard' }))
const routerPush = vi.hoisted(() => vi.fn())

vi.mock('vue-router', () => ({
  useRoute: () => routeState,
  useRouter: () => ({ push: routerPush }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const messages: Record<string, string> = {
    'nav.adminWorkspace': '管理后台',
    'nav.personalWorkspace': '我的工作台',
    'nav.switchToAdminWorkspace': '切换到管理后台',
    'nav.switchToPersonalWorkspace': '切换到我的工作台',
    'nav.adminSections.overview': '概览',
    'nav.adminSections.business': '用户与资源',
    'nav.adminSections.operations': '计费与运营',
    'nav.adminSections.system': '系统与审计',
    'nav.userSections.work': '工作台',
    'nav.userSections.usageTools': '使用与工具',
    'nav.userSections.billing': '账户与费用',
    'nav.adminUsage': '全站用量',
    'nav.announcementManagement': '公告管理',
    'nav.systemSettings': '系统设置',
    'nav.support': '支持',
    'nav.home': '首页',
    'nav.modelCatalog': '模型广场',
    'nav.contactUs': '联系我们',
    'nav.docsTutorial': '文档教程',
    'nav.rechargeAndRedeem': '充值/兑换',
    'announcements.title': '公告',
    'nav.helpAndResources': '帮助与资源',
    'quotaViewerLanding.meta.title': '桌面额度查看器',
  }
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

vi.mock('@/composables/useBatchImageAccess', async () => {
  const { ref } = await vi.importActual<typeof import('vue')>('vue')
  return {
    useBatchImageAccess: () => ({
      canUseBatchImage: ref(false),
      refreshBatchImageAccess: vi.fn().mockResolvedValue(false),
    }),
  }
})

import AppSidebar from '../AppSidebar.vue'
import {
  useAdminSettingsStore,
  useAppStore,
  useAuthStore,
} from '@/stores'

let pinia: Pinia

function createUser(role: User['role']): User {
  return {
    id: 7,
    username: 'sidebar-user',
    email: 'sidebar@example.com',
    role,
    balance: 10,
    concurrency: 2,
    status: 'active',
    allowed_groups: null,
    balance_notify_enabled: true,
    balance_notify_threshold: null,
    balance_notify_extra_emails: [],
    created_at: '2026-07-18T00:00:00Z',
    updated_at: '2026-07-18T00:00:00Z',
  }
}

function mountSidebar(role: User['role'] = 'admin'): VueWrapper {
  const authStore = useAuthStore()
  const appStore = useAppStore()
  const adminSettingsStore = useAdminSettingsStore()

  authStore.user = createUser(role)
  appStore.docUrl = '/docs/'
  appStore.publicSettingsLoaded = true
  vi.spyOn(adminSettingsStore, 'fetch').mockResolvedValue(undefined)

  return mount(AppSidebar, {
    global: {
      plugins: [pinia],
      stubs: {
        RouterLink: {
          props: ['to'],
          template: '<a :href="String(to)"><slot /></a>',
        },
        SidebarCollapseIcon: {
          props: ['collapsed'],
          template:
            '<svg data-component="sidebar-collapse-icon" :data-collapsed="String(collapsed)" />',
        },
        SidebarAccountDock: {
          template: '<div data-testid="sidebar-account-dock-stub" />',
        },
        AnnouncementBell: {
          props: ['variant'],
          template: `
            <button type="button" :data-variant="variant">
              <span>公告</span>
              <span>2</span>
            </button>
          `,
        },
        VersionBadge: true,
      },
    },
  })
}

function enableAllAdminNavigation(): void {
  const appStore = useAppStore()
  const adminSettingsStore = useAdminSettingsStore()

  appStore.cachedPublicSettings = {
    affiliate_enabled: true,
    available_channels_enabled: true,
    channel_monitor_enabled: true,
    payment_enabled: true,
    risk_control_enabled: true,
  } as PublicSettings
  adminSettingsStore.setOpsMonitoringEnabledLocal(true)
  adminSettingsStore.setPaymentEnabledLocal(true)
}

describe('AppSidebar grouped admin navigation', () => {
  beforeEach(() => {
    pinia = createPinia()
    setActivePinia(pinia)
    routeState.path = '/admin/dashboard'
    routerPush.mockReset()
    localStorage.clear()
  })

  it('renders admin navigation directly without the operations wrapper', () => {
    const wrapper = mountSidebar()
    const navigation = wrapper.get('nav.sidebar-nav')
    const dashboardIcon = navigation.get('a[href="/admin/dashboard"] .sidebar-svg-icon svg')
    const accountPoolIcon = navigation.get('a[href="/admin/accounts"] .sidebar-svg-icon svg')
    const auditLogIcon = navigation.get('a[href="/admin/audit-logs"] .sidebar-svg-icon svg')
    const modelMarketplaceIcon = navigation.get('a[href="/admin/model-catalog"] .sidebar-svg-icon svg')

    expect(wrapper.find('#sidebar-admin-operations-toggle').exists()).toBe(false)
    expect(wrapper.find('#sidebar-admin-operations').exists()).toBe(false)
    expect(navigation.get('a[href="/admin/dashboard"]').text()).toContain('nav.adminDashboard')
    expect(dashboardIcon.attributes('viewBox')).toBe('0 0 1024 1024')
    expect(dashboardIcon.get('path').attributes('fill')).toBe('currentColor')
    expect(accountPoolIcon.attributes('viewBox')).toBe('-112 -112 1248 1248')
    expect(accountPoolIcon.get('path').attributes('fill')).toBe('currentColor')
    expect(auditLogIcon.attributes('viewBox')).toBe('-32 -32 1088 1088')
    expect(auditLogIcon.findAll('path')).toHaveLength(5)
    expect(auditLogIcon.get('path').attributes('fill')).toBe('currentColor')
    expect(modelMarketplaceIcon.attributes('viewBox')).toBe('0 0 1024 1024')
    expect(modelMarketplaceIcon.attributes('fill')).toBe('currentColor')
    expect(modelMarketplaceIcon.findAll('path')).toHaveLength(3)
    expect(wrapper.text()).not.toContain('nav.operationsManagement')
  })

  it('keeps the full-height sidebar widths aligned with the application shell', () => {
    const wrapper = mountSidebar()
    const sidebar = wrapper.get('#app-sidebar')

    expect(sidebar.classes()).toContain('sidebar')
    expect(sidebar.classes()).toEqual(
      expect.arrayContaining([
        'w-[min(84vw,288px)]',
        'lg:w-[260px]',
      ]),
    )
    expect(wrapper.find('[data-testid="sidebar-brand-row"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="sidebar-brand"]').attributes('href')).toBe('/admin/dashboard')
    expect(sidebar.classes()).not.toContain('sidebar--snow-clay')
    expect(sidebar.classes()).not.toContain('sidebar--home-clay')
  })

  it('keeps the desktop collapse control beside the brand and synchronizes its state', async () => {
    const appStore = useAppStore()
    const wrapper = mountSidebar()
    const sidebar = wrapper.get('#app-sidebar')
    const brandRow = wrapper.get('[data-testid="sidebar-brand-row"]')
    const brand = brandRow.get('[data-testid="sidebar-brand"]')
    const toggle = brandRow.get('[data-testid="sidebar-collapse-toggle"]')
    const icon = toggle.get('[data-component="sidebar-collapse-icon"]')

    expect(toggle.element.tagName).toBe('BUTTON')
    expect(toggle.attributes('type')).toBe('button')
    expect(toggle.attributes('aria-controls')).toBe('app-sidebar')
    expect(toggle.attributes('aria-expanded')).toBe('true')
    expect(toggle.attributes('aria-label')).toBe('nav.collapse')
    expect(toggle.attributes('title')).toBe('nav.collapse')
    expect(icon.attributes('data-testid')).toBe('sidebar-collapse-toggle-icon')
    expect(icon.attributes('data-collapsed')).toBe('false')
    expect(sidebar.classes()).toEqual(
      expect.arrayContaining([
        'w-[min(84vw,288px)]',
        'lg:w-[260px]',
      ]),
    )
    expect(brandRow.classes()).not.toContain('sidebar-header-collapsed')
    expect(brand.classes()).not.toContain('app-brand--collapsed')

    await toggle.trigger('click')
    await nextTick()

    expect(appStore.sidebarCollapsed).toBe(true)
    expect(toggle.attributes('aria-expanded')).toBe('false')
    expect(toggle.attributes('aria-label')).toBe('nav.expand')
    expect(toggle.attributes('title')).toBe('nav.expand')
    expect(icon.attributes('data-collapsed')).toBe('true')
    expect(sidebar.classes()).toEqual(expect.arrayContaining(['w-[60px]', 'lg:w-[68px]']))
    expect(sidebar.classes()).not.toContain('w-[min(84vw,288px)]')
    expect(brandRow.classes()).toContain('sidebar-header-collapsed')
    expect(brand.classes()).toContain('app-brand--collapsed')

    await toggle.trigger('click')
    await nextTick()

    expect(appStore.sidebarCollapsed).toBe(false)
    expect(toggle.attributes('aria-expanded')).toBe('true')
    expect(toggle.attributes('aria-label')).toBe('nav.collapse')
    expect(toggle.attributes('title')).toBe('nav.collapse')
    expect(icon.attributes('data-collapsed')).toBe('false')
    expect(sidebar.classes()).toContain('lg:w-[260px]')
    expect(brandRow.classes()).not.toContain('sidebar-header-collapsed')
    expect(brand.classes()).not.toContain('app-brand--collapsed')
  })

  it('switches administrators between route-driven admin and personal workspaces', async () => {
    const adminWrapper = mountSidebar('admin')
    const adminOption = adminWrapper.get('[data-testid="sidebar-workspace-admin-option"]')
    const personalOption = adminWrapper.get('[data-testid="sidebar-workspace-personal-option"]')

    expect(adminOption.attributes('aria-pressed')).toBe('true')
    expect(personalOption.attributes('aria-pressed')).toBe('false')
    expect(adminWrapper.find('[data-testid="sidebar-admin-overview-section"]').exists()).toBe(true)
    expect(adminWrapper.find('[data-testid="sidebar-user-work-section"]').exists()).toBe(false)

    await personalOption.trigger('click')
    expect(routerPush).toHaveBeenCalledWith('/dashboard')

    adminWrapper.unmount()
    routerPush.mockReset()
    routeState.path = '/dashboard'

    const personalWrapper = mountSidebar('admin')
    expect(
      personalWrapper.get('[data-testid="sidebar-workspace-admin-option"]').attributes('aria-pressed'),
    ).toBe('false')
    expect(
      personalWrapper.get('[data-testid="sidebar-workspace-personal-option"]').attributes('aria-pressed'),
    ).toBe('true')
    expect(personalWrapper.find('[data-testid="sidebar-admin-overview-section"]').exists()).toBe(false)
    expect(personalWrapper.find('[data-testid="sidebar-user-work-section"]').exists()).toBe(true)

    await personalWrapper.get('[data-testid="sidebar-workspace-admin-option"]').trigger('click')
    expect(routerPush).toHaveBeenCalledWith('/admin/dashboard')
  })

  it('keeps the workspace switch keyboard-accessible in the collapsed rail', async () => {
    const appStore = useAppStore()
    appStore.setSidebarCollapsed(true)
    const wrapper = mountSidebar('admin')
    const compactSwitch = wrapper.get('[data-testid="sidebar-workspace-compact-switch"]')

    expect(compactSwitch.element.tagName).toBe('BUTTON')
    expect(compactSwitch.attributes('type')).toBe('button')
    expect(compactSwitch.attributes('aria-label')).toBe('切换到我的工作台')
    expect(wrapper.find('[data-testid="sidebar-workspace-admin-option"]').exists()).toBe(false)

    await compactSwitch.trigger('click')
    expect(routerPush).toHaveBeenCalledWith('/dashboard')
  })

  it('groups the complete admin navigation by task while retaining the production rail', async () => {
    enableAllAdminNavigation()
    const wrapper = mountSidebar('admin')
    const sections = wrapper.findAll('[data-testid^="sidebar-admin-"]')

    expect(sections.map(section => section.attributes('data-testid'))).toEqual([
      'sidebar-admin-overview-section',
      'sidebar-admin-business-section',
      'sidebar-admin-operations-section',
      'sidebar-admin-system-section',
    ])
    expect(sections.map(section => section.attributes('aria-label'))).toEqual([
      '概览',
      '用户与资源',
      '计费与运营',
      '系统与审计',
    ])
    expect(sections.every(section => section.attributes('role') === 'group')).toBe(true)
    expect(sections.map(section => section.get('.sidebar-section-title').text())).toEqual([
      '概览',
      '用户与资源',
      '计费与运营',
      '系统与审计',
    ])

    for (const panelId of [
      'sidebar-group-admin-channels',
      'sidebar-group-admin-affiliates',
      'sidebar-group-admin-orders',
    ]) {
      await wrapper.get(`[aria-controls="${panelId}"]`).trigger('click')
    }

    expect(sections[0].findAll('a[href]').map(link => link.attributes('href'))).toEqual([
      '/admin/dashboard',
      '/admin/ops',
      '/admin/usage',
    ])
    expect(wrapper.findAll('a[href="/chat"]')).toHaveLength(0)
    expect(sections[0].text()).toContain('全站用量')
    expect(sections[1].findAll('a[href]').map(link => link.attributes('href'))).toEqual([
      '/admin/users',
      '/admin/groups',
      '/admin/accounts',
      '/admin/proxies',
      '/admin/channels/pricing',
      '/admin/channels/monitor',
      '/admin/model-catalog',
      '/admin/skills',
    ])
    expect(sections[2].findAll('a[href]').map(link => link.attributes('href'))).toEqual([
      '/admin/subscriptions',
      '/admin/orders/dashboard',
      '/admin/orders',
      '/admin/orders/plans',
      '/admin/redeem',
      '/admin/promo-codes',
      '/admin/affiliates/invites',
      '/admin/affiliates/rebates',
      '/admin/affiliates/transfers',
      '/admin/announcements',
    ])
    expect(sections[3].findAll('a[href]').map(link => link.attributes('href'))).toEqual([
      '/admin/risk-control',
      '/admin/audit-logs',
      '/admin/desktop-diagnostics',
      '/admin/documentation',
      '/admin/settings',
    ])
  })

  it('opens overview and the active admin section while keeping other sections collapsible', async () => {
    routeState.path = '/admin/users'
    const wrapper = mountSidebar('admin')
    const overviewToggle = wrapper.get(
      '[data-testid="sidebar-admin-overview-section"] .sidebar-section-toggle',
    )
    const businessToggle = wrapper.get(
      '[data-testid="sidebar-admin-business-section"] .sidebar-section-toggle',
    )
    const operationsToggle = wrapper.get(
      '[data-testid="sidebar-admin-operations-section"] .sidebar-section-toggle',
    )

    expect(overviewToggle.attributes('aria-expanded')).toBe('true')
    expect(businessToggle.attributes('aria-expanded')).toBe('true')
    expect(businessToggle.attributes('aria-disabled')).toBe('true')
    expect(operationsToggle.attributes('aria-expanded')).toBe('false')
    expect(wrapper.get('[data-testid="sidebar-section-panel-operations"]').attributes('style')).toContain(
      'display: none',
    )

    await businessToggle.trigger('click')
    expect(businessToggle.attributes('aria-expanded')).toBe('true')

    await operationsToggle.trigger('click')
    expect(operationsToggle.attributes('aria-expanded')).toBe('true')
    expect(localStorage.getItem('app-sidebar-admin-expanded-sections')).toBe(
      JSON.stringify(['overview', 'operations']),
    )
  })

  it('restores manually expanded admin sections from local storage', () => {
    localStorage.setItem(
      'app-sidebar-admin-expanded-sections',
      JSON.stringify(['overview', 'system']),
    )
    const wrapper = mountSidebar('admin')

    expect(
      wrapper
        .get('[data-testid="sidebar-admin-system-section"] .sidebar-section-toggle')
        .attributes('aria-expanded'),
    ).toBe('true')
    expect(
      wrapper.get('[data-testid="sidebar-section-panel-system"]').attributes('style'),
    ).toBeUndefined()
  })

  it('keeps every admin section reachable when the global rail is collapsed', () => {
    localStorage.setItem('app-sidebar-admin-expanded-sections', JSON.stringify([]))
    const appStore = useAppStore()
    appStore.setSidebarCollapsed(true)
    const wrapper = mountSidebar('admin')

    expect(wrapper.find('.sidebar-section-toggle').exists()).toBe(false)
    expect(wrapper.get('a[href="/admin/users"]').classes()).toContain('sidebar-link-collapsed')
    for (const id of ['overview', 'business', 'operations', 'system']) {
      expect(
        wrapper.get(`[data-testid="sidebar-section-panel-${id}"]`).attributes('style'),
      ).toBeUndefined()
    }
  })

  it('keeps custom admin pages in the system group before the final settings destination', () => {
    const adminSettingsStore = useAdminSettingsStore()
    adminSettingsStore.customMenuItems = [
      {
        id: 'later-tool',
        label: '后置工具',
        icon_svg: '<svg viewBox="0 0 24 24"><path d="M3 3h18v18H3z" /></svg>',
        url: '',
        visibility: 'admin',
        sort_order: 2,
      },
      {
        id: 'internal-tool',
        label: '内部工具',
        icon_svg: '<svg viewBox="0 0 24 24"><path d="M4 4h16v16H4z" /></svg>',
        url: '',
        visibility: 'admin',
        sort_order: 1,
      },
    ]
    const wrapper = mountSidebar('admin')
    const systemSection = wrapper.get('[data-testid="sidebar-admin-system-section"]')
    const links = systemSection.findAll('a[href]').map(link => link.attributes('href'))

    expect(links.slice(-3)).toEqual([
      '/custom/internal-tool',
      '/custom/later-tool',
      '/admin/settings',
    ])
  })

  it('keeps simple mode grouped without empty headings or the personal-tools section', () => {
    useAuthStore()
    const adminSettingsStore = useAdminSettingsStore()
    const rawAuthState = toRaw(pinia.state.value.auth) as unknown as {
      runMode: Ref<'standard' | 'simple'>
    }
    toRaw(rawAuthState.runMode).value = 'simple'
    adminSettingsStore.customMenuItems = [
      {
        id: 'simple-tool',
        label: '简洁工具',
        icon_svg: '<svg viewBox="0 0 24 24"><path d="M4 4h16v16H4z" /></svg>',
        url: '',
        visibility: 'admin',
        sort_order: 1,
      },
    ]

    const wrapper = mountSidebar('admin')
    const sections = wrapper.findAll('[data-testid^="sidebar-admin-"]')

    expect(sections.map(section => section.attributes('data-testid'))).toEqual([
      'sidebar-admin-overview-section',
      'sidebar-admin-business-section',
      'sidebar-admin-operations-section',
      'sidebar-admin-system-section',
    ])
    expect(sections.every(section => section.findAll('.sidebar-link').length > 0)).toBe(true)
    expect(wrapper.findAll('a[href="/chat"]')).toHaveLength(0)
    expect(wrapper.get('[data-testid="sidebar-docs-tutorial"]').attributes('href')).toBe('/docs/')
    expect(wrapper.find('a[href="/admin/users"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="sidebar-admin-business-section"] a[href="/keys"]').exists()).toBe(true)
    const simpleSystemLinks = wrapper
      .get('[data-testid="sidebar-admin-system-section"]')
      .findAll('a[href]')
      .map(link => link.attributes('href'))
    expect(simpleSystemLinks.at(-1)).toBe('/admin/settings')
    expect(wrapper.text()).not.toContain('个人工具')
  })

  it('ships concise Chinese and English labels for the grouped administrator rail', () => {
    expect(zhCommon.nav.adminSections).toEqual({
      overview: '概览',
      business: '用户与资源',
      operations: '计费与运营',
      system: '系统与审计',
    })
    expect(zhCommon.nav.adminUsage).toBe('全站用量')
    expect(zhCommon.nav.support).toBe('支持')
    expect(zhCommon.nav.rechargeAndRedeem).toBe('充值/兑换')
    expect(enCommon.nav.adminSections).toEqual({
      overview: 'Overview',
      business: 'Users & Resources',
      operations: 'Billing & Operations',
      system: 'System & Audit',
    })
    expect(enCommon.nav.adminUsage).toBe('Platform Usage')
    expect(enCommon.nav.support).toBe('Support')
    expect(enCommon.nav.rechargeAndRedeem).toBe('Top Up / Redeem')
    expect(enCommon.nav.docsTutorial).toBe('Documentation Guide')
    expect(enCommon.nav.documentationManagement).toBe('Documentation Management')
  })

  it('does not inject a status card into the original navigation rail', () => {
    const wrapper = mountSidebar('admin')

    expect(wrapper.find('.sidebar-session-status').exists()).toBe(false)
  })

  it('keeps the active navigation item paired with its icon container', () => {
    const wrapper = mountSidebar('admin')
    const activeItems = wrapper.findAll('.sidebar-link-active')

    expect(activeItems).toHaveLength(1)
    expect(activeItems[0].attributes('href')).toBe('/admin/dashboard')
    expect(activeItems[0].find('.sidebar-nav-icon').exists()).toBe(true)
  })

  it('highlights only the specific payment dashboard child route', () => {
    routeState.path = '/admin/orders/dashboard'
    enableAllAdminNavigation()
    const wrapper = mountSidebar('admin')
    const orderGroup = wrapper.get('#sidebar-group-admin-orders')
    const paymentDashboard = orderGroup.get('a[href="/admin/orders/dashboard"]')
    const orderIndex = orderGroup.get('a[href="/admin/orders"]')
    const activeChildren = orderGroup.findAll('a.sidebar-link-active')

    expect(paymentDashboard.classes()).toContain('sidebar-link-active')
    expect(orderIndex.classes()).not.toContain('sidebar-link-active')
    expect(activeChildren).toHaveLength(1)
    expect(activeChildren[0].attributes('href')).toBe('/admin/orders/dashboard')
  })

  it('renders the mobile overlay as an accessible close button', async () => {
    const appStore = useAppStore()
    appStore.setMobileOpen(true)
    const wrapper = mountSidebar('admin')
    const overlay = wrapper.get('button[aria-label="nav.closeNavigation"]')

    expect(overlay.element.tagName).toBe('BUTTON')
    expect(overlay.attributes('type')).toBe('button')

    await overlay.trigger('click')
    expect(appStore.mobileOpen).toBe(false)
  })

  it('keeps the mobile drawer available behind the announcement modal', async () => {
    const appStore = useAppStore()
    appStore.setMobileOpen(true)
    const wrapper = mountSidebar('user')

    await wrapper.get('[data-testid="sidebar-announcements"]').trigger('click')
    await nextTick()

    expect(appStore.mobileOpen).toBe(true)
  })

  it('keeps real nested admin groups accessible', () => {
    routeState.path = '/admin/orders/plans'
    const adminSettingsStore = useAdminSettingsStore()
    adminSettingsStore.setPaymentEnabledLocal(true)
    const wrapper = mountSidebar()
    const orderToggle = wrapper.get('[aria-controls="sidebar-group-admin-orders"]')

    expect(orderToggle.attributes('aria-expanded')).toBe('true')
    expect(wrapper.get('#sidebar-group-admin-orders').attributes('role')).toBe('group')
  })

  it('expands the global sidebar when a collapsed nested group is activated', async () => {
    const appStore = useAppStore()
    appStore.setSidebarCollapsed(true)
    const wrapper = mountSidebar()
    const channelToggle = wrapper.get('[aria-controls="sidebar-group-admin-channels"]')

    expect(channelToggle.attributes('aria-expanded')).toBe('false')
    expect(wrapper.find('#sidebar-group-admin-channels').exists()).toBe(false)

    await channelToggle.trigger('click')
    await nextTick()

    expect(appStore.sidebarCollapsed).toBe(false)
    expect(channelToggle.attributes('aria-expanded')).toBe('true')
    expect(wrapper.find('#sidebar-group-admin-channels').exists()).toBe(true)
  })

  it('keeps normal admin links rendered while collapsed', () => {
    const appStore = useAppStore()
    appStore.setSidebarCollapsed(true)
    const wrapper = mountSidebar()
    const navigation = wrapper.get('nav.sidebar-nav')

    expect(navigation.get('a[href="/admin/dashboard"]').classes()).toContain('sidebar-link-collapsed')
    expect(wrapper.find('a[href="/admin/announcements"]').exists()).toBe(true)
  })

  it('uses scope-specific admin labels and the supplied notification glyph', () => {
    const wrapper = mountSidebar()
    const announcementPath = wrapper.get('a[href="/admin/announcements"] svg path')

    expect(wrapper.get('a[href="/admin/announcements"]').text()).toContain('公告管理')
    expect(wrapper.get('a[href="/admin/settings"]').text()).toContain('系统设置')
    expect(wrapper.text()).not.toContain('个人工具')
    expect(announcementPath.attributes('d')).toContain('M512 235.52')
    expect(announcementPath.attributes('fill')).toBe('currentColor')
  })

  it('does not render admin links for regular users', () => {
    const wrapper = mountSidebar('user')

    expect(wrapper.find('#sidebar-admin-operations-toggle').exists()).toBe(false)
    expect(wrapper.find('a[href="/admin/dashboard"]').exists()).toBe(false)
  })

  it('keeps only simple-mode account actions in the regular-user rail', () => {
    useAuthStore()
    const rawAuthState = toRaw(pinia.state.value.auth) as unknown as {
      runMode: Ref<'standard' | 'simple'>
    }
    toRaw(rawAuthState.runMode).value = 'simple'

    const wrapper = mountSidebar('user')

    expect(wrapper.find('a[href="/chat"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="sidebar-docs-tutorial"]').attributes('href')).toBe('/docs/')
    expect(wrapper.find('a[href="/quota-viewer"]').exists()).toBe(true)
    expect(wrapper.find('a[href="/purchase"]').exists()).toBe(true)
    expect(wrapper.find('a[href="/subscriptions"]').exists()).toBe(false)
    expect(wrapper.find('a[href="/orders"]').exists()).toBe(false)
    expect(wrapper.find('a[href="/profile"]').exists()).toBe(false)
    expect(wrapper.find('a[href="/models.html"]').exists()).toBe(false)
  })

  it('groups regular-user work, tools, and billing destinations in the scrolling rail', () => {
    const wrapper = mountSidebar('user')
    const workSection = wrapper.get('[data-testid="sidebar-user-work-section"]')
    const usageToolsSection = wrapper.get('[data-testid="sidebar-user-usageTools-section"]')
    const billingSection = wrapper.get('[data-testid="sidebar-user-billing-section"]')

    expect(workSection.attributes('aria-label')).toBe('工作台')
    expect(workSection.findAll('a').map((link) => link.attributes('href'))).toEqual([
      '/dashboard',
      '/chat',
      '/keys',
    ])
    expect(usageToolsSection.attributes('aria-label')).toBe('使用与工具')
    expect(usageToolsSection.findAll('a').map((link) => link.attributes('href'))).toEqual([
      '/usage',
      '/monitor',
      '/quota-viewer',
    ])
    const quotaViewerLink = usageToolsSection.get('a[href="/quota-viewer"]')
    const quotaViewerJumpIcon = quotaViewerLink.get(
      '[data-testid="sidebar-nav-trailing-icon"]',
    )
    expect(quotaViewerJumpIcon.attributes('aria-hidden')).toBe('true')
    expect(quotaViewerJumpIcon.get('path').attributes('d')).toContain('M9 6.65')
    expect(quotaViewerLink.attributes('target')).toBeUndefined()
    expect(quotaViewerLink.attributes('rel')).toBeUndefined()
    expect(quotaViewerLink.findAll('[data-testid="sidebar-nav-trailing-icon"]')).toHaveLength(1)
    expect(billingSection.attributes('aria-label')).toBe('账户与费用')
    expect(billingSection.findAll('a').map((link) => link.attributes('href'))).toEqual([
      '/purchase',
      '/subscriptions',
      '/orders',
    ])
    expect(billingSection.get('a[href="/purchase"]').text()).toContain('充值/兑换')
    expect(wrapper.find('nav a[href="/profile"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="sidebar-account-dock-stub"]').exists()).toBe(true)
  })

  it.each<User['role']>(['admin', 'user'])(
    'shows support destinations directly in the scrolling %s navigation',
    (role) => {
      const wrapper = mountSidebar(role)
      const navigation = wrapper.get('nav.sidebar-nav')
      const supportSection = navigation.get('[data-testid="sidebar-support-section"]')
      const docsLink = navigation.get('[data-testid="sidebar-docs-tutorial"]')
      const contactLink = navigation.get('[data-testid="sidebar-contact-us"]')

      expect(supportSection.get('[data-testid="sidebar-announcements"]').text()).toContain('公告')
      expect(supportSection.get('[data-testid="sidebar-announcements"]').attributes('data-variant'))
        .toBe('row')
      expect(docsLink.text()).toContain('文档教程')
      expect(docsLink.attributes('href')).toBe('/docs/')
      expect(docsLink.attributes('target')).toBe('_blank')
      expect(docsLink.attributes('rel')).toBe('noopener noreferrer')
      expect(contactLink.text()).toContain('联系我们')
      expect(contactLink.attributes('target')).toBe('_blank')
      expect(contactLink.attributes('rel')).toBe('noopener noreferrer')
      for (const link of [docsLink, contactLink]) {
        const jumpIcon = link.get('[data-testid="sidebar-nav-trailing-icon"]')
        expect(jumpIcon.classes()).toContain('sidebar-nav-trailing-icon')
        expect(jumpIcon.attributes('aria-hidden')).toBe('true')
        expect(jumpIcon.get('path').attributes('d')).toContain('M9 6.65')
      }
      expect(supportSection.find('[data-testid="sidebar-help-resources"]').exists()).toBe(false)
      expect(supportSection.find('a[href="/home"]').exists()).toBe(false)
      expect(wrapper.findAll('[data-testid="sidebar-docs-tutorial"]')).toHaveLength(1)
      expect(wrapper.get('[data-testid="sidebar-account-dock-stub"]').element.contains(
        supportSection.element,
      )).toBe(false)
    },
  )

  it('shows configured support resources as flat first-level links', () => {
    const appStore = useAppStore()
    appStore.cachedPublicSettings = {
      public_model_catalog_enabled: true,
      contact_info: '/support',
      doc_url: '/docs/',
    } as PublicSettings

    const wrapper = mountSidebar('user')
    const supportSection = wrapper.get('[data-testid="sidebar-support-section"]')

    expect(supportSection.findAll('.sidebar-support-link').map((link) => (
      link.attributes('href')
    ))).toEqual([
      '/docs/',
      '/models.html',
      '/support',
    ])
    expect(supportSection.findAll('.sidebar-support-link').every((link) => (
      link.attributes('target') === '_blank'
      && link.attributes('rel') === 'noopener noreferrer'
    ))).toBe(true)
  })

  it('keeps each support destination directly reachable in the collapsed rail', () => {
    const appStore = useAppStore()
    appStore.setSidebarCollapsed(true)
    const wrapper = mountSidebar('user')
    const docsLink = wrapper.get('[data-testid="sidebar-docs-tutorial"]')
    const contactLink = wrapper.get('[data-testid="sidebar-contact-us"]')

    expect(appStore.sidebarCollapsed).toBe(true)
    expect(docsLink.classes()).toContain('sidebar-link-collapsed')
    expect(contactLink.classes()).toContain('sidebar-link-collapsed')
    expect(docsLink.attributes('title')).toBe('文档教程')
    expect(contactLink.attributes('title')).toBe('联系我们')
    expect(docsLink.find('[data-testid="sidebar-nav-trailing-icon"]').exists()).toBe(false)
    expect(contactLink.find('[data-testid="sidebar-nav-trailing-icon"]').exists()).toBe(false)
  })

  it('keeps only the primary quota-viewer glyph in the collapsed rail', () => {
    const appStore = useAppStore()
    appStore.setSidebarCollapsed(true)
    const wrapper = mountSidebar('user')
    const quotaViewerLink = wrapper.get('a[href="/quota-viewer"]')

    expect(quotaViewerLink.classes()).toContain('sidebar-link-collapsed')
    expect(quotaViewerLink.find('[data-testid="sidebar-nav-trailing-icon"]').exists()).toBe(false)
    expect(quotaViewerLink.find('.sidebar-nav-icon').exists()).toBe(true)
  })

  it('keeps the documentation tutorial in navigation when backend mode hides user routes', () => {
    const appStore = useAppStore()
    appStore.cachedPublicSettings = {
      backend_mode_enabled: true,
      doc_url: '/docs/',
    } as PublicSettings

    const wrapper = mountSidebar('user')

    expect(wrapper.find('[data-testid="sidebar-user-work-section"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="sidebar-docs-tutorial"]').attributes('href')).toBe('/docs/')
    expect(wrapper.find('a[href="/models.html"]').exists()).toBe(false)
  })

  it('keeps custom user links after the three task sections', () => {
    const appStore = useAppStore()
    appStore.cachedPublicSettings = {
      custom_menu_items: [
        {
          id: 'support-center',
          label: '支持中心',
          icon_svg: '<svg viewBox="0 0 24 24"><path d="M4 4h16v16H4z" /></svg>',
          url: '',
          visibility: 'user',
          sort_order: 1,
        },
      ],
    } as unknown as PublicSettings
    const wrapper = mountSidebar('user')
    const sections = wrapper.get('nav.sidebar-nav').findAll('[data-testid^="sidebar-user-"]')

    expect(sections.map((section) => section.attributes('data-testid'))).toEqual([
      'sidebar-user-work-section',
      'sidebar-user-usageTools-section',
      'sidebar-user-billing-section',
      'sidebar-user-custom-section',
    ])
    expect(wrapper.get('[data-testid="sidebar-user-custom-section"] a').attributes('href')).toBe(
      '/custom/support-center',
    )
  })

  it.each<User['role']>(['admin', 'user'])('mounts the account dock below the scrollable %s menu', (role) => {
    const wrapper = mountSidebar(role)
    const sidebar = wrapper.get('#app-sidebar')
    const navigation = wrapper.get('nav.sidebar-nav')

    const dock = wrapper.get('[data-testid="sidebar-account-dock-stub"]')
    expect(dock.element.parentElement).toBe(sidebar.element)
    expect(navigation.element.contains(dock.element)).toBe(false)
  })
})

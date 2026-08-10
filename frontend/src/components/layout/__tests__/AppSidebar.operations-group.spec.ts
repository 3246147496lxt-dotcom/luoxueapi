import { nextTick, toRaw, type Ref } from 'vue'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
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
    'nav.userSections.workbench': '工作台',
    'nav.userSections.account': '账户',
    'nav.userSections.resources': '资源',
    'nav.userSections.more': '更多工具',
    'nav.adminUsage': '全站用量',
    'nav.announcementManagement': '公告管理',
    'nav.systemSettings': '系统设置',
    'nav.support': '支持',
    'nav.home': '首页',
    'nav.modelCatalog': '模型广场',
    'nav.contactUs': '联系我们',
    'nav.docsTutorial': '文档教程',
    'nav.rechargeAndRedeem': '充值/兑换',
    'nav.modelCenter': '模型中心',
    'nav.balance': '余额',
    'nav.subscription': '套餐',
    'nav.orders': '订单',
    'nav.serviceStatusNav': '服务状态',
    'accountDock.settings': '设置',
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
import { adminNavigationDefinition } from '../sidebar/adminNavigation'
import {
  useAppStore,
  useAuthStore,
} from '@/stores'
import { useAdminSettingsStore } from '@/stores/adminSettings'
import {
  WORKSPACE_MOBILE_DRAWER_MEDIA_QUERY,
  WORKSPACE_NARROW_SIDEBAR_MEDIA_QUERY,
} from '../workspaceResponsive'

let pinia: Pinia
const matchMediaSpy = vi.spyOn(window, 'matchMedia')

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
    props: {
      adminNavigation: adminNavigationDefinition,
    },
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
        UserAccountCard: {
          props: ['context', 'collapsed'],
          template: `
            <div
              data-testid="sidebar-account-dock-stub"
              :data-context="context"
              :data-collapsed="String(collapsed)"
            />
          `,
        },
        WorkspaceSidebarBrand: {
          props: ['homePath'],
          template: `
            <a :href="homePath" data-testid="workspace-sidebar-brand-stub">
              <strong data-testid="workspace-sidebar-brand-name">落雪AI</strong>
              <span data-testid="workspace-sidebar-plan">GPT-订阅会员</span>
            </a>
          `,
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
    matchMediaSpy.mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }) as MediaQueryList)
  })

  it('renders admin navigation directly without the operations wrapper', () => {
    const wrapper = mountSidebar()
    const navigation = wrapper.get('nav.workspace-sidebar-navigation')
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
    const entryMeta = document.createElement('meta')
    entryMeta.name = 'app-entry'
    entryMeta.content = 'admin'
    document.head.append(entryMeta)
    const wrapper = mountSidebar()
    const sidebar = wrapper.get('#app-sidebar')

    expect(sidebar.classes()).toContain('app-sidebar')
    expect(sidebar.classes()).not.toContain('sidebar')
    expect(sidebar.classes()).toEqual(
      expect.arrayContaining([
        'workspace-sidebar-frame',
        'workspace-sidebar-frame--fixed',
        'workspace-sidebar-frame--work',
        'workspace-sidebar-frame--content-admin',
      ]),
    )
    expect(sidebar.attributes('data-sidebar-collapsed')).toBe('false')
    expect(wrapper.find('[data-testid="sidebar-brand-row"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="sidebar-brand"]').attributes('href')).toBe('/admin/dashboard')
    expect(sidebar.classes()).not.toContain('sidebar--snow-clay')
    expect(sidebar.classes()).not.toContain('sidebar--home-clay')
    entryMeta.remove()
  })

  it('keeps the desktop collapse control beside the brand and synchronizes its state', async () => {
    const appStore = useAppStore()
    const wrapper = mountSidebar()
    const sidebar = wrapper.get('#app-sidebar')
    const brandRow = wrapper.get('[data-testid="sidebar-brand-row"]')
    let toggle = brandRow.get('[data-testid="sidebar-collapse-toggle"]')
    let icon = toggle.get('[data-testid="sidebar-collapse-toggle-icon"]')

    expect(toggle.element.tagName).toBe('BUTTON')
    expect(toggle.attributes('type')).toBe('button')
    expect(toggle.attributes('aria-controls')).toBe('app-sidebar')
    expect(toggle.attributes('aria-expanded')).toBe('true')
    expect(toggle.attributes('aria-label')).toBe('nav.collapse')
    expect(toggle.attributes('title')).toBe('nav.collapse')
    expect(icon.classes()).toContain('workspace-desktop-sidebar-header-icon--collapse')
    expect(icon.attributes('viewBox')).toBe('0 0 20 20')
    expect(sidebar.classes()).not.toContain('workspace-sidebar-frame--collapsed')
    expect(sidebar.attributes('data-sidebar-collapsed')).toBe('false')
    expect(wrapper.get('[data-testid="sidebar-account-dock-stub"]')
      .attributes('data-collapsed')).toBe('false')
    expect(brandRow.classes()).not.toContain('workspace-sidebar-header--collapsed')

    await toggle.trigger('click')
    await nextTick()

    expect(appStore.sidebarCollapsed).toBe(true)
    toggle = brandRow.get('[data-testid="sidebar-collapse-toggle"]')
    icon = toggle.get('[data-testid="sidebar-collapse-toggle-icon"]')
    expect(toggle.attributes('aria-expanded')).toBe('false')
    expect(toggle.attributes('aria-label')).toBe('nav.expand')
    expect(toggle.attributes('title')).toBe('nav.expand')
    expect(icon.attributes('data-collapsed')).toBe('true')
    expect(sidebar.classes()).toContain('workspace-sidebar-frame--collapsed')
    expect(sidebar.attributes('data-sidebar-collapsed')).toBe('true')
    expect(wrapper.get('[data-testid="sidebar-account-dock-stub"]')
      .attributes('data-collapsed')).toBe('true')
    expect(brandRow.classes()).toContain('workspace-sidebar-header--collapsed')
    expect(brandRow.find('[data-testid="sidebar-brand"]').exists()).toBe(false)

    await toggle.trigger('click')
    await nextTick()

    expect(appStore.sidebarCollapsed).toBe(false)
    toggle = brandRow.get('[data-testid="sidebar-collapse-toggle"]')
    icon = toggle.get('[data-testid="sidebar-collapse-toggle-icon"]')
    expect(toggle.attributes('aria-expanded')).toBe('true')
    expect(toggle.attributes('aria-label')).toBe('nav.collapse')
    expect(toggle.attributes('title')).toBe('nav.collapse')
    expect(icon.classes()).toContain('workspace-desktop-sidebar-header-icon--collapse')
    expect(sidebar.classes()).not.toContain('workspace-sidebar-frame--collapsed')
    expect(sidebar.attributes('data-sidebar-collapsed')).toBe('false')
    expect(wrapper.get('[data-testid="sidebar-account-dock-stub"]')
      .attributes('data-collapsed')).toBe('false')
    expect(brandRow.classes()).not.toContain('workspace-sidebar-header--collapsed')
    expect(brandRow.find('[data-testid="sidebar-brand"]').exists()).toBe(true)
  })

  it('keeps admin and user navigation visually isolated for administrators', () => {
    const adminWrapper = mountSidebar('admin')

    expect(adminWrapper.find('[data-testid="sidebar-workspace-switch"]').exists()).toBe(false)
    expect(adminWrapper.find('[data-testid="sidebar-brand-logo"]').exists()).toBe(true)
    expect(adminWrapper.find('[data-testid="sidebar-brand-wordmark"]').exists()).toBe(false)
    expect(adminWrapper.find('[data-testid="sidebar-admin-overview-section"]').exists()).toBe(true)
    expect(adminWrapper.find('[data-testid="sidebar-user-workbench-section"]').exists()).toBe(false)

    adminWrapper.unmount()
    routeState.path = '/dashboard'

    const personalWrapper = mountSidebar('admin')
    expect(personalWrapper.find('[data-testid="sidebar-workspace-switch"]').exists()).toBe(false)
    expect(personalWrapper.get('[data-testid="workspace-sidebar-brand-name"]').text()).toBe('落雪AI')
    expect(personalWrapper.get('[data-testid="workspace-sidebar-plan"]').text()).toBe('GPT-订阅会员')
    expect(personalWrapper.find('[data-testid="sidebar-brand-logo"]').exists()).toBe(false)
    expect(personalWrapper.find('a[href="/admin/dashboard"]').exists()).toBe(false)
    expect(personalWrapper.find('[data-testid="sidebar-admin-overview-section"]').exists()).toBe(false)
    expect(personalWrapper.find('[data-testid="sidebar-user-workbench-section"]').exists()).toBe(true)
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

  it('opens every admin task section by default while keeping manual collapse persistent', async () => {
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
    const systemToggle = wrapper.get(
      '[data-testid="sidebar-admin-system-section"] .sidebar-section-toggle',
    )

    expect(overviewToggle.attributes('aria-expanded')).toBe('true')
    expect(businessToggle.attributes('aria-expanded')).toBe('true')
    expect(businessToggle.attributes('aria-disabled')).toBe('true')
    expect(operationsToggle.attributes('aria-expanded')).toBe('true')
    expect(systemToggle.attributes('aria-expanded')).toBe('true')

    await businessToggle.trigger('click')
    expect(businessToggle.attributes('aria-expanded')).toBe('true')

    await operationsToggle.trigger('click')
    expect(operationsToggle.attributes('aria-expanded')).toBe('false')
    expect(wrapper.get('[data-testid="sidebar-section-panel-operations"]').attributes('style')).toContain(
      'display: none',
    )
    expect(localStorage.getItem('app-sidebar-admin-expanded-sections-v2')).toBe(
      JSON.stringify(['overview', 'business', 'system']),
    )

    wrapper.unmount()
    const restoredWrapper = mountSidebar('admin')
    expect(
      restoredWrapper
        .get('[data-testid="sidebar-admin-operations-section"] .sidebar-section-toggle')
        .attributes('aria-expanded'),
    ).toBe('false')
  })

  it('restores manually expanded admin sections from local storage', () => {
    localStorage.setItem(
      'app-sidebar-admin-expanded-sections-v2',
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
    localStorage.setItem('app-sidebar-admin-expanded-sections-v2', JSON.stringify([]))
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

  it('keeps custom admin pages in the system group before the final settings destination', async () => {
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
    await flushPromises()
    await nextTick()
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
    expect(wrapper.find('[data-testid="sidebar-support-section"]').exists()).toBe(false)
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
    matchMediaSpy.mockImplementation((query: string) => ({
      matches: query === WORKSPACE_MOBILE_DRAWER_MEDIA_QUERY,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }) as MediaQueryList)
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
    matchMediaSpy.mockImplementation((query: string) => ({
      matches: query === WORKSPACE_MOBILE_DRAWER_MEDIA_QUERY,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }) as MediaQueryList)
    const appStore = useAppStore()
    appStore.setMobileOpen(true)
    const wrapper = mountSidebar('user')

    await wrapper.get('[data-testid="sidebar-announcements"]').trigger('click')
    await nextTick()

    expect(appStore.mobileOpen).toBe(true)
  })

  it('renders mobile expanded without clearing the shared desktop collapse request', async () => {
    matchMediaSpy.mockImplementation((query: string) => ({
      matches: query === WORKSPACE_MOBILE_DRAWER_MEDIA_QUERY,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }) as MediaQueryList)
    const appStore = useAppStore()
    appStore.setSidebarCollapsed(true)
    const wrapper = mountSidebar('admin')
    await nextTick()

    expect(appStore.sidebarCollapsed).toBe(true)
    expect(wrapper.get('#app-sidebar').attributes('data-sidebar-collapsed')).toBe('false')
    expect(wrapper.get('[data-testid="sidebar-account-dock-stub"]')
      .attributes('data-collapsed')).toBe('false')
    expect(wrapper.find('[data-testid="sidebar-collapse-toggle"]').exists()).toBe(false)

    await wrapper.get('[aria-controls="sidebar-group-admin-channels"]').trigger('click')
    expect(appStore.sidebarCollapsed).toBe(true)
  })

  it.each([
    ['large desktop', 1440, 'hover', 'fine', false, false],
    ['small desktop', 1024, 'hover', 'fine', false, false],
    ['fine-pointer boundary', 768, 'hover', 'fine', false, false],
    ['touch-first tablet boundary', 768, 'none', 'coarse', false, false],
    ['narrow desktop', 614, 'hover', 'fine', false, true],
    ['touch-first narrow tablet', 614, 'none', 'coarse', true, false],
    ['touch-first phone', 390, 'none', 'coarse', true, false],
  ] as const)(
    'renders %s at %ipx with hover=%s pointer=%s, drawer=%s, narrow overlay=%s',
    async (_environment, viewportWidth, hover, pointer, mobileDrawer, narrowSidebar) => {
      matchMediaSpy.mockImplementation((query: string) => ({
        matches: (
          query === WORKSPACE_MOBILE_DRAWER_MEDIA_QUERY
          && viewportWidth <= 767
          && hover === 'none'
          && pointer === 'coarse'
        ) || (
          query === WORKSPACE_NARROW_SIDEBAR_MEDIA_QUERY
          && viewportWidth <= 767
          && hover === 'hover'
          && pointer === 'fine'
        ),
        media: query,
        onchange: null,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      }) as MediaQueryList)
      const appStore = useAppStore()
      const wrapper = mountSidebar('user')
      await nextTick()
      const sidebar = narrowSidebar
        ? document.body.querySelector<HTMLElement>('#app-sidebar')
        : wrapper.get('#app-sidebar').element as HTMLElement
      const accountDock = narrowSidebar
        ? document.body.querySelector<HTMLElement>('[data-testid="sidebar-account-dock-stub"]')
        : wrapper.get('[data-testid="sidebar-account-dock-stub"]').element as HTMLElement

      expect(appStore.sidebarCollapsed).toBe(false)
      expect(appStore.workspaceNarrowSidebar).toBe(narrowSidebar)
      expect(sidebar?.dataset.sidebarCollapsed).toBe('false')
      expect(sidebar?.getAttribute('aria-hidden') ?? undefined)
        .toBe(mobileDrawer ? 'true' : undefined)
      expect(wrapper.find('[data-testid="sidebar-collapse-toggle"]').exists())
        .toBe(!mobileDrawer && !narrowSidebar)
      expect(accountDock?.dataset.collapsed).toBe('false')
      expect(matchMediaSpy).toHaveBeenCalledWith(WORKSPACE_MOBILE_DRAWER_MEDIA_QUERY)
      expect(matchMediaSpy).toHaveBeenCalledWith(WORKSPACE_NARROW_SIDEBAR_MEDIA_QUERY)

      wrapper.unmount()
      document.body.innerHTML = ''
    },
  )

  it('preserves the manual 68px rail at and above the 768px desktop boundary', async () => {
    matchMediaSpy.mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }) as MediaQueryList)
    const appStore = useAppStore()
    appStore.setSidebarCollapsed(true)
    const wrapper = mountSidebar('user')
    await nextTick()

    const sidebar = wrapper.get('#app-sidebar')
    expect(appStore.sidebarCollapsed).toBe(true)
    expect(sidebar.attributes('data-sidebar-collapsed')).toBe('true')
    expect(sidebar.classes()).toContain('workspace-sidebar-frame--collapsed')
    expect(wrapper.get('[data-testid="sidebar-account-dock-stub"]')
      .attributes('data-collapsed')).toBe('true')
    expect(wrapper.get('[data-testid="sidebar-collapse-toggle"]')
      .attributes('aria-label')).toBe('nav.expand')
  })

  it('uses a full overlay below 768px and restores the manual desktop rail when widened', async () => {
    let narrowMatches = false
    const breakpointListeners = new Map<string, (event: MediaQueryListEvent) => void>()
    matchMediaSpy.mockImplementation((query: string) => ({
      get matches() {
        return query === WORKSPACE_NARROW_SIDEBAR_MEDIA_QUERY && narrowMatches
      },
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn((_type: string, listener: (event: MediaQueryListEvent) => void) => {
        breakpointListeners.set(query, listener)
      }),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }) as MediaQueryList)
    const appStore = useAppStore()
    appStore.setSidebarCollapsed(true)
    const wrapper = mountSidebar('user')
    await nextTick()

    expect(appStore.sidebarCollapsed).toBe(true)
    expect(wrapper.get('#app-sidebar').attributes('data-sidebar-collapsed')).toBe('true')

    narrowMatches = true
    breakpointListeners.get(WORKSPACE_NARROW_SIDEBAR_MEDIA_QUERY)?.(
      { matches: true } as MediaQueryListEvent,
    )
    await nextTick()

    expect(appStore.sidebarCollapsed).toBe(true)
    expect(appStore.workspaceNarrowSidebar).toBe(true)
    expect(appStore.workspaceNarrowSidebarOpen).toBe(false)
    const narrowSidebar = document.body.querySelector<HTMLElement>('#app-sidebar')
    const narrowAccountDock = document.body.querySelector<HTMLElement>(
      '[data-testid="sidebar-account-dock-stub"]',
    )
    expect(narrowSidebar?.dataset.sidebarCollapsed).toBe('false')
    expect(narrowSidebar?.classList.contains('workspace-sidebar-frame--overlay')).toBe(true)
    expect(narrowAccountDock?.dataset.collapsed).toBe('false')
    expect(wrapper.find('[data-testid="sidebar-collapse-toggle"]').exists()).toBe(false)

    appStore.setWorkspaceNarrowSidebarOpen(true)
    await nextTick()

    const overlay = document.body.querySelector<HTMLElement>(
      '[data-testid="workspace-sidebar-overlay-layer"]',
    )
    const closeButton = document.body.querySelector<HTMLButtonElement>(
      '.workspace-sidebar-header__close',
    )
    expect(overlay?.classList.contains('workspace-sidebar-overlay-layer--open')).toBe(true)
    expect(overlay?.getAttribute('aria-hidden')).toBeNull()
    expect(closeButton?.getAttribute('aria-label')).toBe('nav.closeNavigation')
    expect(closeButton?.querySelector('.workspace-responsive-sidebar-icon--close')).not.toBeNull()
    expect(closeButton?.querySelector('[data-component="sidebar-collapse-icon"]')).toBeNull()

    closeButton?.click()
    await nextTick()
    expect(appStore.workspaceNarrowSidebarOpen).toBe(false)
    expect(appStore.sidebarCollapsed).toBe(true)

    appStore.setWorkspaceNarrowSidebarOpen(true)
    narrowMatches = false
    breakpointListeners.get(WORKSPACE_NARROW_SIDEBAR_MEDIA_QUERY)?.(
      { matches: false } as MediaQueryListEvent,
    )
    await nextTick()

    expect(appStore.workspaceNarrowSidebar).toBe(false)
    expect(appStore.workspaceNarrowSidebarOpen).toBe(false)
    expect(appStore.sidebarCollapsed).toBe(true)
    expect(wrapper.get('#app-sidebar').attributes('data-sidebar-collapsed')).toBe('true')

    wrapper.unmount()
    document.body.innerHTML = ''
  })

  it('closes a mobile drawer when crossing into the desktop range', async () => {
    let mobileMatches = true
    const breakpointListeners = new Map<string, (event: MediaQueryListEvent) => void>()
    matchMediaSpy.mockImplementation((query: string) => ({
      get matches() {
        return query === WORKSPACE_MOBILE_DRAWER_MEDIA_QUERY && mobileMatches
      },
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn((_type: string, listener: (event: MediaQueryListEvent) => void) => {
        breakpointListeners.set(query, listener)
      }),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }) as MediaQueryList)
    const appStore = useAppStore()
    appStore.setSidebarCollapsed(true)
    appStore.setMobileOpen(true)
    const wrapper = mountSidebar('user')
    await nextTick()

    expect(appStore.sidebarCollapsed).toBe(true)
    expect(wrapper.get('#app-sidebar').attributes('data-sidebar-collapsed')).toBe('false')
    expect(wrapper.get('#app-sidebar').attributes('aria-hidden')).toBeUndefined()
    mobileMatches = false
    breakpointListeners.get(WORKSPACE_MOBILE_DRAWER_MEDIA_QUERY)?.(
      { matches: false } as MediaQueryListEvent,
    )
    await nextTick()

    expect(appStore.mobileOpen).toBe(false)
    expect(appStore.sidebarCollapsed).toBe(true)
    expect(wrapper.get('#app-sidebar').attributes('data-sidebar-collapsed')).toBe('true')
    expect(wrapper.get('#app-sidebar').attributes('aria-hidden')).toBeUndefined()

    mobileMatches = true
    breakpointListeners.get(WORKSPACE_MOBILE_DRAWER_MEDIA_QUERY)?.(
      { matches: true } as MediaQueryListEvent,
    )
    await nextTick()
    expect(appStore.sidebarCollapsed).toBe(true)
    expect(wrapper.get('#app-sidebar').attributes('data-sidebar-collapsed')).toBe('false')
    expect(wrapper.get('#app-sidebar').attributes('aria-hidden')).toBe('true')
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
    const navigation = wrapper.get('nav.workspace-sidebar-navigation')

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

  it('keeps only simple-mode account actions in the regular-user rail', async () => {
    useAuthStore()
    const rawAuthState = toRaw(pinia.state.value.auth) as unknown as {
      runMode: Ref<'standard' | 'simple'>
    }
    toRaw(rawAuthState.runMode).value = 'simple'

    const wrapper = mountSidebar('user')

    expect(wrapper.find('nav a[href="/chat"]').exists()).toBe(false)
    const modeSwitch = wrapper.get('[data-testid="app-mode-switch"]')
    expect(modeSwitch.attributes('data-active-mode')).toBe('work')
    expect(modeSwitch.findAll('button')).toHaveLength(2)
    await modeSwitch.findAll('button')[0]!.trigger('click')
    expect(routerPush).toHaveBeenCalledWith('/chat')
    expect(wrapper.get('[data-testid="sidebar-docs-tutorial"]').attributes('href')).toBe('/docs/')
    expect(wrapper.find('a[href="/quota-viewer"]').exists()).toBe(false)
    expect(wrapper.find('a[href="/purchase"]').exists()).toBe(true)
    expect(wrapper.find('a[href="/subscriptions"]').exists()).toBe(false)
    expect(wrapper.find('a[href="/orders"]').exists()).toBe(false)
    expect(wrapper.find('a[href="/profile"]').exists()).toBe(false)
    expect(wrapper.find('a[href="/models"]').exists()).toBe(false)
  })

  it('groups the Work workspace into the approved workbench, account, and resources IA', () => {
    const wrapper = mountSidebar('user')
    const workbenchSection = wrapper.get('[data-testid="sidebar-user-workbench-section"]')
    const accountSection = wrapper.get('[data-testid="sidebar-user-account-section"]')
    const resourcesSection = wrapper.get('[data-testid="sidebar-support-section"]')

    expect(workbenchSection.attributes('aria-label')).toBe('工作台')
    expect(workbenchSection.findAll('a').map((link) => link.attributes('href'))).toEqual([
      '/dashboard',
      '/keys',
      '/usage',
    ])
    expect(accountSection.attributes('aria-label')).toBe('账户')
    expect(accountSection.findAll('a').map((link) => link.attributes('href'))).toEqual([
      '/purchase',
      '/subscriptions',
      '/orders',
    ])
    expect(accountSection.get('a[href="/purchase"]').text()).toContain('余额')
    expect(accountSection.get('a[href="/subscriptions"]').text()).toContain('套餐')
    expect(wrapper.find('[data-testid="sidebar-user-more-section"]').exists()).toBe(false)
    expect(wrapper.find('a[href="/monitor"]').exists()).toBe(false)
    expect(wrapper.find('a[href="/quota-viewer"]').exists()).toBe(false)
    expect(resourcesSection.attributes('aria-label')).toBe('资源')
    expect(resourcesSection.find('[data-testid="sidebar-docs-tutorial"]').exists()).toBe(true)
    expect(resourcesSection.find('[data-testid="sidebar-announcements"]').exists()).toBe(true)
    expect(resourcesSection.find('[data-testid="sidebar-settings"]').exists()).toBe(true)
    expect(wrapper.find('nav a[href="/chat"]').exists()).toBe(false)
    expect(wrapper.find('nav a[href="/profile"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="sidebar-account-dock-stub"]').exists()).toBe(true)
  })

  it('shows documentation, announcements, and settings as Work resources', () => {
    const wrapper = mountSidebar('user')
    const navigation = wrapper.get('nav.workspace-sidebar-navigation')
    const supportSection = navigation.get('[data-testid="sidebar-support-section"]')
    const docsLink = navigation.get('[data-testid="sidebar-docs-tutorial"]')
    const settings = navigation.get('[data-testid="sidebar-settings"]')

    expect(supportSection.get('[data-testid="sidebar-announcements"]').text()).toContain('公告')
    expect(supportSection.get('[data-testid="sidebar-announcements"]').attributes('data-variant'))
      .toBe('row')
    expect(docsLink.text()).toContain('文档教程')
    expect(docsLink.attributes('href')).toBe('/docs/')
    expect(docsLink.attributes('target')).toBe('_blank')
    expect(docsLink.attributes('rel')).toBe('noopener noreferrer')
    expect(settings.text()).toContain('设置')
    expect(navigation.find('[data-testid="sidebar-contact-us"]').exists()).toBe(false)
    const jumpIcon = docsLink.get('[data-testid="sidebar-nav-trailing-icon"]')
    expect(jumpIcon.classes()).toContain('sidebar-nav-trailing-icon')
    expect(jumpIcon.attributes('aria-hidden')).toBe('true')
    expect(jumpIcon.get('path').attributes('d')).toContain('M9 6.65')
    expect(supportSection.find('[data-testid="sidebar-help-resources"]').exists()).toBe(false)
    expect(supportSection.find('a[href="/home"]').exists()).toBe(false)
    expect(wrapper.findAll('[data-testid="sidebar-docs-tutorial"]')).toHaveLength(1)
    expect(wrapper.get('[data-testid="sidebar-account-dock-stub"]').element.contains(
      supportSection.element,
    )).toBe(false)
  })

  it('omits end-user support actions from the admin workspace', () => {
    const wrapper = mountSidebar('admin')

    expect(wrapper.find('[data-testid="sidebar-support-section"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="sidebar-announcements"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="sidebar-docs-tutorial"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="sidebar-contact-us"]').exists()).toBe(false)
    expect(wrapper.find('a[href="/admin/announcements"]').exists()).toBe(true)
    expect(wrapper.find('a[href="/admin/documentation"]').exists()).toBe(true)
  })

  it('keeps support actions in an administrator personal workspace', () => {
    routeState.path = '/dashboard'
    const wrapper = mountSidebar('admin')

    expect(wrapper.find('[data-testid="sidebar-support-section"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="sidebar-announcements"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="sidebar-docs-tutorial"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="sidebar-contact-us"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="sidebar-settings"]').exists()).toBe(true)
  })

  it('moves the configured model catalog into Workbench and keeps Resources focused', () => {
    const appStore = useAppStore()
    appStore.cachedPublicSettings = {
      public_model_catalog_enabled: true,
      contact_info: '/support',
      doc_url: '/docs/',
    } as PublicSettings

    const wrapper = mountSidebar('user')
    const supportSection = wrapper.get('[data-testid="sidebar-support-section"]')
    const workbenchSection = wrapper.get('[data-testid="sidebar-user-workbench-section"]')

    expect(supportSection.findAll('.sidebar-support-link').map((link) => (
      link.attributes('href')
    ))).toEqual(['/docs/'])
    expect(supportSection.findAll('.sidebar-support-link').every((link) => (
      link.attributes('target') === '_blank'
      && link.attributes('rel') === 'noopener noreferrer'
    ))).toBe(true)
    const modelCenterLink = workbenchSection.get('a[href="/models"]')
    expect(modelCenterLink.text()).toContain('模型中心')
    expect(modelCenterLink.attributes('target')).toBeUndefined()
    expect(modelCenterLink.attributes('rel')).toBeUndefined()
    expect(wrapper.find('a[href="/support"]').exists()).toBe(false)
  })

  it('keeps personal Work on the shared collapsed width token when state is collapsed', () => {
    const appStore = useAppStore()
    appStore.setSidebarCollapsed(true)
    const wrapper = mountSidebar('user')
    const docsLink = wrapper.get('[data-testid="sidebar-docs-tutorial"]')
    const settings = wrapper.get('[data-testid="sidebar-settings"]')
    const announcements = wrapper.get('[data-testid="sidebar-announcements"]')

    expect(appStore.sidebarCollapsed).toBe(true)
    expect(wrapper.get('#app-sidebar').classes()).toContain('workspace-sidebar-frame--collapsed')
    expect(wrapper.get('#app-sidebar').attributes('data-sidebar-collapsed')).toBe('true')
    expect(wrapper.get('[data-testid="sidebar-account-dock-stub"]')
      .attributes('data-collapsed')).toBe('true')
    expect(wrapper.get('[data-testid="sidebar-collapse-toggle"]').attributes('aria-expanded'))
      .toBe('false')
    expect(docsLink.classes()).toContain('sidebar-link-collapsed')
    expect(settings.classes()).toContain('sidebar-link-collapsed')
    expect(announcements.classes()).toContain('sidebar-announcement-entry--collapsed')
    expect(docsLink.attributes('title')).toBe('文档教程')
    expect(settings.attributes('title')).toBe('设置')
    expect(docsLink.find('[data-testid="sidebar-nav-trailing-icon"]').exists()).toBe(false)
  })

  it('keeps utility destinations routable but outside the primary Work rail', () => {
    const wrapper = mountSidebar('user')

    for (const path of ['/monitor', '/quota-viewer', '/batch-image', '/available-channels', '/skills']) {
      expect(wrapper.find(`a[href="${path}"]`).exists()).toBe(false)
    }
  })

  it('keeps the documentation tutorial in navigation when backend mode hides user routes', () => {
    const appStore = useAppStore()
    appStore.cachedPublicSettings = {
      backend_mode_enabled: true,
      doc_url: '/docs/',
    } as PublicSettings

    const wrapper = mountSidebar('user')

    expect(wrapper.find('[data-testid="sidebar-user-workbench-section"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="sidebar-docs-tutorial"]').attributes('href')).toBe('/docs/')
    expect(wrapper.find('a[href="/models"]').exists()).toBe(false)
  })

  it('keeps custom user routes outside the strict primary Work rail', () => {
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
    const sections = wrapper.get('nav.workspace-sidebar-navigation')
      .findAll('[data-testid^="sidebar-user-"]')

    expect(sections.map((section) => section.attributes('data-testid'))).toEqual([
      'sidebar-user-workbench-section',
      'sidebar-user-account-section',
    ])
    expect(wrapper.find('a[href="/custom/support-center"]').exists()).toBe(false)
  })

  it.each<User['role']>(['admin', 'user'])('mounts the account dock below the scrollable %s menu', (role) => {
    const wrapper = mountSidebar(role)
    const sidebar = wrapper.get('#app-sidebar')
    const navigation = wrapper.get('nav.workspace-sidebar-navigation')

    const dock = wrapper.get('[data-testid="sidebar-account-dock-stub"]')
    const footer = wrapper.get('.workspace-sidebar-frame__footer')
    expect(dock.element.parentElement).toBe(footer.element)
    expect(footer.element.parentElement).toBe(sidebar.element)
    expect(navigation.element.contains(dock.element)).toBe(false)
  })
})

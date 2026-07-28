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
    'nav.personalTools': '个人工具',
    'nav.adminSections.overview': '概览',
    'nav.adminSections.business': '用户与资源',
    'nav.adminSections.operations': '计费与运营',
    'nav.adminSections.system': '系统与审计',
    'nav.adminUsage': '全站用量',
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
      '/chat',
      '/admin/ops',
      '/admin/usage',
    ])
    expect(wrapper.findAll('a[href="/chat"]')).toHaveLength(1)
    expect(sections[0].text()).toContain('全站用量')
    expect(sections[1].findAll('a[href]').map(link => link.attributes('href'))).toEqual([
      '/admin/users',
      '/admin/groups',
      '/admin/accounts',
      '/admin/proxies',
      '/admin/channels/pricing',
      '/admin/channels/monitor',
      '/admin/model-catalog',
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
    expect(
      wrapper
        .get('[data-testid="sidebar-admin-overview-section"]')
        .find('a[href="/chat"]')
        .exists(),
    ).toBe(true)
    expect(wrapper.findAll('a[href="/chat"]')).toHaveLength(1)
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
    expect(enCommon.nav.adminSections).toEqual({
      overview: 'Overview',
      business: 'Users & Resources',
      operations: 'Billing & Operations',
      system: 'System & Audit',
    })
    expect(enCommon.nav.adminUsage).toBe('Platform Usage')
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

  it('uses the personal-tools heading and the supplied notification glyph', () => {
    const wrapper = mountSidebar()
    const announcementPath = wrapper.get('a[href="/admin/announcements"] svg path')

    expect(wrapper.text()).toContain('个人工具')
    expect(announcementPath.attributes('d')).toContain('M512 235.52')
    expect(announcementPath.attributes('fill')).toBe('currentColor')
  })

  it('does not render admin links for regular users', () => {
    const wrapper = mountSidebar('user')

    expect(wrapper.find('#sidebar-admin-operations-toggle').exists()).toBe(false)
    expect(wrapper.find('a[href="/admin/dashboard"]').exists()).toBe(false)
  })

  it('keeps account destinations out of regular-user navigation in simple mode', () => {
    useAuthStore()
    const rawAuthState = toRaw(pinia.state.value.auth) as unknown as {
      runMode: Ref<'standard' | 'simple'>
    }
    toRaw(rawAuthState.runMode).value = 'simple'

    const wrapper = mountSidebar('user')

    expect(wrapper.find('a[href="/chat"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="sidebar-docs-tutorial"]').attributes('href')).toBe('/docs/')
    expect(wrapper.find('a[href="/purchase"]').exists()).toBe(false)
    expect(wrapper.find('a[href="/subscriptions"]').exists()).toBe(false)
    expect(wrapper.find('a[href="/orders"]').exists()).toBe(false)
    expect(wrapper.find('a[href="/profile"]').exists()).toBe(false)
  })

  it('keeps regular-user account links out of the scrolling rail', () => {
    const wrapper = mountSidebar('user')
    const mainSection = wrapper.get('[data-testid="sidebar-user-main-section"]')

    expect(mainSection.findAll('a').map((link) => link.attributes('href'))).toEqual([
      '/dashboard',
      '/chat',
      '/keys',
      '/usage',
      '/monitor',
    ])
    expect(wrapper.find('[data-testid="sidebar-user-personal-section"]').exists()).toBe(false)
    for (const path of ['/subscriptions', '/purchase', '/orders', '/profile']) {
      expect(wrapper.find(`nav a[href="${path}"]`).exists()).toBe(false)
    }
    expect(wrapper.find('[data-testid="sidebar-account-dock-stub"]').exists()).toBe(true)
  })

  it.each<User['role']>(['admin', 'user'])(
    'moves the configured documentation tutorial into the scrolling %s navigation',
    (role) => {
      const wrapper = mountSidebar(role)
      const navigation = wrapper.get('nav.sidebar-nav')
      const docsLink = navigation.get('[data-testid="sidebar-docs-tutorial"]')

      expect(docsLink.text()).toContain('nav.docsTutorial')
      expect(docsLink.attributes('href')).toBe('/docs/')
      expect(docsLink.attributes('target')).toBe('_blank')
      expect(docsLink.attributes('rel')).toBe('noopener noreferrer')
      expect(wrapper.findAll('[data-testid="sidebar-docs-tutorial"]')).toHaveLength(1)
    },
  )

  it('keeps the documentation tutorial in navigation when backend mode hides user routes', () => {
    const appStore = useAppStore()
    appStore.cachedPublicSettings = {
      backend_mode_enabled: true,
      doc_url: '/docs/',
    } as PublicSettings

    const wrapper = mountSidebar('user')

    expect(wrapper.find('[data-testid="sidebar-user-main-section"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="sidebar-docs-tutorial"]').attributes('href')).toBe('/docs/')
  })

  it('keeps custom user links after the main section', () => {
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
      'sidebar-user-main-section',
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

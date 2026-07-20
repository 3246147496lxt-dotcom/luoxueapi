import { nextTick } from 'vue'
import { mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia, type Pinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { PublicSettings, User } from '@/types'

const routeState = vi.hoisted(() => ({ path: '/admin/dashboard' }))
const routerPush = vi.hoisted(() => vi.fn())

vi.mock('vue-router', () => ({
  useRoute: () => routeState,
  useRouter: () => ({ push: routerPush }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key === 'nav.myAccount' ? '个人中心' : key,
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
        VersionBadge: true,
      },
    },
  })
}

function setPublicModelCatalog(enabled: boolean, backendMode = false): void {
  const appStore = useAppStore()
  appStore.cachedPublicSettings = {
    public_model_catalog_enabled: enabled,
    backend_mode_enabled: backendMode,
  } as PublicSettings
}

describe('AppSidebar flattened admin navigation', () => {
  beforeEach(() => {
    pinia = createPinia()
    setActivePinia(pinia)
    routeState.path = '/admin/dashboard'
    routerPush.mockReset()
    localStorage.clear()
  })

  it('renders admin navigation directly without the operations wrapper', () => {
    const wrapper = mountSidebar()
    const dashboardIcon = wrapper.get('a[href="/admin/dashboard"] .sidebar-svg-icon svg')
    const accountPoolIcon = wrapper.get('a[href="/admin/accounts"] .sidebar-svg-icon svg')
    const auditLogIcon = wrapper.get('a[href="/admin/audit-logs"] .sidebar-svg-icon svg')
    const modelMarketplaceIcon = wrapper.get('a[href="/admin/model-catalog"] .sidebar-svg-icon svg')

    expect(wrapper.find('#sidebar-admin-operations-toggle').exists()).toBe(false)
    expect(wrapper.find('#sidebar-admin-operations').exists()).toBe(false)
    expect(wrapper.get('a[href="/admin/dashboard"]').text()).toContain('nav.adminDashboard')
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

    expect(wrapper.get('a[href="/admin/dashboard"]').classes()).toContain('sidebar-link-collapsed')
    expect(wrapper.get('a[href="/admin/announcements"]').exists()).toBe(true)
  })

  it('uses the personal-center heading and the supplied notification glyph', () => {
    const wrapper = mountSidebar()
    const announcementPath = wrapper.get('a[href="/admin/announcements"] svg path')

    expect(wrapper.text()).toContain('个人中心')
    expect(announcementPath.attributes('d')).toContain('M512 235.52')
    expect(announcementPath.attributes('fill')).toBe('currentColor')
  })

  it('does not render admin links for regular users', () => {
    const wrapper = mountSidebar('user')

    expect(wrapper.find('#sidebar-admin-operations-toggle').exists()).toBe(false)
    expect(wrapper.find('a[href="/admin/dashboard"]').exists()).toBe(false)
  })

  it('groups regular-user account links under the personal-center heading', () => {
    const wrapper = mountSidebar('user')
    const mainSection = wrapper.get('[data-testid="sidebar-user-main-section"]')
    const personalSection = wrapper.get('[data-testid="sidebar-user-personal-section"]')

    expect(personalSection.attributes('role')).toBe('group')
    expect(personalSection.attributes('aria-label')).toBe('个人中心')
    expect(personalSection.get('.sidebar-section-title').text()).toBe('个人中心')
    expect(personalSection.get('.sidebar-section-title').attributes('aria-hidden')).toBe('true')
    expect(mainSection.findAll('a').map((link) => link.attributes('href'))).toEqual([
      '/dashboard',
      '/keys',
      '/usage',
      '/monitor',
    ])
    expect(personalSection.findAll('a').map((link) => link.attributes('href'))).toEqual([
      '/subscriptions',
      '/purchase',
      '/orders',
      '/profile',
    ])
  })

  it('keeps custom user links after the personal-center section', () => {
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
    } as PublicSettings
    const wrapper = mountSidebar('user')
    const sections = wrapper.get('nav.sidebar-nav').findAll('[data-testid^="sidebar-user-"]')

    expect(sections.map((section) => section.attributes('data-testid'))).toEqual([
      'sidebar-user-main-section',
      'sidebar-user-personal-section',
      'sidebar-user-custom-section',
    ])
    expect(wrapper.get('[data-testid="sidebar-user-custom-section"] a').attributes('href')).toBe(
      '/custom/support-center',
    )
  })

  it('keeps the remaining account links grouped when payments are disabled', () => {
    const appStore = useAppStore()
    appStore.cachedPublicSettings = {
      custom_menu_items: [],
      payment_enabled: false,
    } as PublicSettings
    const wrapper = mountSidebar('user')
    const personalSection = wrapper.get('[data-testid="sidebar-user-personal-section"]')

    expect(personalSection.findAll('a').map((link) => link.attributes('href'))).toEqual([
      '/subscriptions',
      '/purchase',
      '/profile',
    ])
  })

  it('links regular users to the configured documentation center', () => {
    const appStore = useAppStore()
    appStore.docUrl = 'https://docs.example.com/tutorial-docs/'
    const wrapper = mountSidebar('user')
    const docsLink = wrapper.get('[data-destination="docs"]')

    expect(docsLink.text()).toContain('nav.docsTutorial')
    expect(docsLink.attributes('href')).toBe('https://docs.example.com/tutorial-docs/')
    expect(docsLink.attributes('target')).toBe('_blank')
    expect(docsLink.attributes('rel')).toBe('noopener noreferrer')
    expect(wrapper.findAll('[data-destination="docs"]')).toHaveLength(1)
  })

  it('shows administrators the documentation link and falls back to the canonical center', () => {
    const wrapper = mountSidebar()
    const docsLink = wrapper.get('[data-destination="docs"]')

    expect(docsLink.attributes('href')).toBe('https://luoxueapi.cc/tutorial-docs/')
  })

  it('uses a configured support URL or falls back to the contact section in documentation', () => {
    const appStore = useAppStore()
    appStore.docUrl = 'https://docs.example.com/tutorial-docs/'
    appStore.contactInfo = 'QQ: 2456772148'
    const fallbackWrapper = mountSidebar('user')

    expect(fallbackWrapper.get('[data-destination="contact"]').attributes('href')).toBe(
      'https://docs.example.com/tutorial-docs/#recharge',
    )

    fallbackWrapper.unmount()
    appStore.contactInfo = 'https://support.example.com/contact'
    const configuredWrapper = mountSidebar('user')

    expect(configuredWrapper.get('[data-destination="contact"]').attributes('href')).toBe(
      'https://support.example.com/contact',
    )
  })

  it.each<User['role']>(['admin', 'user'])('pins the enabled public destinations below the scrollable %s menu', (role) => {
    setPublicModelCatalog(true)
    const wrapper = mountSidebar(role)
    const sidebar = wrapper.get('#app-sidebar')
    const navigation = wrapper.get('nav.sidebar-nav')
    const destinations = wrapper.get('[data-testid="sidebar-destination-links"]')
    const links = destinations.findAll('[data-destination]')

    expect(destinations.element.parentElement).toBe(sidebar.element)
    expect(navigation.element.contains(destinations.element)).toBe(false)
    expect(links.map(link => link.attributes('data-destination'))).toEqual([
      'home',
      'models',
      'contact',
      'docs',
    ])
    expect(destinations.get('[data-destination="home"]').attributes('href')).toBe('/home')
    expect(destinations.get('[data-destination="models"]').attributes('href')).toBe('/models.html')
    expect(destinations.get('[data-destination="contact"]').attributes('href')).toBe('https://luoxueapi.cc/tutorial-docs/#recharge')
    expect(destinations.findAll('[data-role="destination-icon"]')).toHaveLength(4)
    expect(destinations.findAll('[data-role="destination-label"]')).toHaveLength(4)
    expect(destinations.findAll('[data-role="destination-jump"]')).toHaveLength(4)
    expect(links.every(link => link.attributes('target') === '_blank')).toBe(true)

    const expectedIcons = [
      'destinationHome',
      'destinationModels',
      'destinationContact',
      'destinationDocument',
    ]

    links.forEach((link, index) => {
      const leading = link.get('.sidebar-destination-leading')
      const icon = link.get('[data-role="destination-icon"]')
      const jump = link.get('[data-role="destination-jump"]')

      expect(leading.element.contains(icon.element)).toBe(true)
      expect(icon.attributes('data-icon-name')).toBe(expectedIcons[index])
      expect(jump.attributes('data-icon-name')).toBe('destinationArrowUpRight')
      expect(link.element.lastElementChild).toBe(jump.element)
    })
  })

  it('hides the model catalog destination when its public flag is disabled', () => {
    setPublicModelCatalog(false)
    const wrapper = mountSidebar('user')

    expect(wrapper.find('[data-destination="models"]').exists()).toBe(false)
    expect(wrapper.findAll('[data-destination]')).toHaveLength(3)
  })

  it('shows the model catalog destination when its public flag is enabled', () => {
    setPublicModelCatalog(true)
    const wrapper = mountSidebar('user')

    expect(wrapper.get('[data-destination="models"]').attributes('href')).toBe('/models.html')
  })

  it('hides the model catalog destination in backend mode even when enabled', () => {
    setPublicModelCatalog(true, true)
    const wrapper = mountSidebar('user')

    expect(wrapper.find('[data-destination="models"]').exists()).toBe(false)
    expect(wrapper.findAll('[data-destination]')).toHaveLength(3)
  })

  it('keeps destination icons accessible when the sidebar is collapsed', () => {
    const appStore = useAppStore()
    setPublicModelCatalog(true)
    appStore.setSidebarCollapsed(true)
    const wrapper = mountSidebar()
    const destinations = wrapper.get('[data-testid="sidebar-destination-links"]')
    const links = destinations.findAll('[data-destination]')

    expect(links).toHaveLength(4)
    for (const link of links) {
      expect(link.classes()).toContain('sidebar-destination-link-collapsed')
      expect(link.attributes('aria-label')).toBeTruthy()
      expect(link.get('[data-role="destination-icon"]').exists()).toBe(true)
      expect(link.get('[data-role="destination-label"]').attributes('aria-hidden')).toBe('true')
      expect(link.get('[data-role="destination-jump"]').attributes('aria-hidden')).toBe('true')
    }
  })

  it('closes the mobile sidebar after a destination is activated', async () => {
    vi.useFakeTimers()
    const appStore = useAppStore()
    appStore.setMobileOpen(true)
    const wrapper = mountSidebar('user')

    await wrapper.get('[data-destination="home"]').trigger('click')
    await vi.advanceTimersByTimeAsync(150)

    expect(appStore.mobileOpen).toBe(false)
    vi.useRealTimers()
  })
})

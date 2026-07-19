import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import type { User } from '@/types'

const routeState = vi.hoisted(() => ({
  name: 'AdminGroups',
  params: {},
  meta: { title: 'Groups', description: 'Groups description' },
}))

vi.mock('vue-router', () => ({
  useRoute: () => routeState,
  useRouter: () => ({ push: vi.fn() }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

import AppHeader from '../AppHeader.vue'
import { useAppStore, useAuthStore } from '@/stores'

function createUser(): User {
  return {
    id: 7,
    username: 'header-user',
    email: 'header@example.com',
    role: 'user',
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

function mountHeader(userOverrides: Partial<User> = {}) {
  const pinia = createPinia()
  setActivePinia(pinia)
  useAuthStore().user = { ...createUser(), ...userOverrides }

  return mount(AppHeader, {
    global: {
      plugins: [pinia],
      mocks: {
        $t: (key: string) => key,
      },
      stubs: {
        RouterLink: {
          props: ['to'],
          template: '<a :href="String(to)"><slot /></a>',
        },
        AnnouncementBell: {
          template: '<div data-testid="announcement-bell" />',
        },
        LocaleSwitcher: {
          template: '<div data-testid="locale-switcher" />',
        },
        SidebarCollapseIcon: {
          template: '<svg data-component="sidebar-collapse-icon" />',
        },
        SubscriptionProgressMini: true,
      },
    },
  })
}

describe('AppHeader theme toggle', () => {
  beforeEach(() => {
    document.documentElement.classList.remove('dark')
    localStorage.removeItem('theme')
  })

  afterEach(() => {
    document.documentElement.classList.remove('dark')
    localStorage.removeItem('theme')
  })

  it('renders an icon-only action between announcements and language', () => {
    const wrapper = mountHeader()
    const toggle = wrapper.get('[data-testid="header-theme-toggle"]')

    expect(toggle.text()).toBe('')
    expect(toggle.attributes('aria-label')).toBe('nav.darkMode')
    expect(toggle.element.previousElementSibling).toBe(
      wrapper.get('[data-testid="announcement-bell"]').element,
    )
    expect(toggle.element.nextElementSibling).toBe(
      wrapper.get('[data-testid="locale-switcher"]').element,
    )
  })

  it('updates the root theme class, persistent preference, and accessible label', async () => {
    const wrapper = mountHeader()
    const toggle = wrapper.get('[data-testid="header-theme-toggle"]')

    await toggle.trigger('click')

    expect(document.documentElement.classList.contains('dark')).toBe(true)
    expect(localStorage.getItem('theme')).toBe('dark')
    expect(toggle.attributes('aria-label')).toBe('nav.lightMode')

    await toggle.trigger('click')

    expect(document.documentElement.classList.contains('dark')).toBe(false)
    expect(localStorage.getItem('theme')).toBe('light')
    expect(toggle.attributes('aria-label')).toBe('nav.darkMode')
  })
})

describe('AppHeader global navigation shell', () => {
  it('renders the reference header proportions and compact mobile control', () => {
    const wrapper = mountHeader()

    expect(wrapper.get('[data-testid="app-header"]').classes()).toEqual(
      expect.arrayContaining(['fixed', 'inset-x-0', 'top-0', 'h-[81px]', 'bg-transparent', 'p-2', 'z-50']),
    )
    expect(wrapper.get('[data-testid="app-header"]').classes()).not.toContain('sticky')
    expect(wrapper.get('[data-testid="header-surface"]').classes()).toEqual(
      expect.arrayContaining(['h-[65px]', 'rounded-2xl']),
    )
    const mobileMenu = wrapper.get('[data-testid="header-mobile-menu"]')
    expect(mobileMenu.classes()).toEqual(
      expect.arrayContaining([
        'relative',
        'h-8',
        'w-8',
        'after:absolute',
        'after:-inset-1.5',
        "after:content-['']",
      ]),
    )
    expect(wrapper.get('[data-testid="header-brand-slot"]').classes()).toEqual(
      expect.arrayContaining([
        'md:w-[196px]',
        'lg:w-[184px]',
        'min-[1025px]:w-[196px]',
        'min-[1281px]:w-[208px]',
        'lg:justify-start',
      ]),
    )
    expect(wrapper.get('[data-testid="header-brand"]').classes()).toEqual(
      expect.arrayContaining([
        'lg:w-44',
        'min-[1025px]:w-[188px]',
        'min-[1281px]:w-[200px]',
      ]),
    )
    expect(wrapper.get('[data-testid="header-brand"]').attributes('href')).toBe('/dashboard')
    expect(wrapper.get('[data-testid="header-brand"]').attributes('aria-label')).toBe('落雪API')
    const brandImage = wrapper.get('[data-testid="header-brand"] img')
    expect(brandImage.attributes('src')).toBe('/logo.png')
    expect(Array.from(brandImage.element.parentElement?.classList ?? [])).toEqual(
      expect.arrayContaining(['h-10', 'w-10', 'md:h-12', 'md:w-12']),
    )
    expect(wrapper.get('[data-testid="header-brand"]').text()).toContain('落雪API')
    expect(wrapper.find('[data-testid="header-surface"] p').exists()).toBe(false)
  })

  it('uses configured brand data and links administrators to their dashboard', async () => {
    const wrapper = mountHeader({ role: 'admin' })
    const appStore = useAppStore()

    appStore.siteName = 'Snow Console'
    appStore.siteLogo = '/brand.svg'
    await wrapper.vm.$nextTick()

    const brand = wrapper.get('[data-testid="header-brand"]')
    expect(brand.attributes('href')).toBe('/admin/dashboard')
    expect(brand.attributes('aria-label')).toBe('Snow Console')
    expect(brand.get('img').attributes('src')).toBe('/brand.svg')
    expect(brand.text()).toContain('Snow Console')
  })

  it('moves desktop sidebar collapse control into the header with synchronized aria state', async () => {
    const wrapper = mountHeader()
    const appStore = useAppStore()
    const toggle = wrapper.get('[data-testid="header-sidebar-toggle"]')

    expect(toggle.classes()).toEqual(expect.arrayContaining(['hidden', 'h-8', 'w-8', 'lg:-ml-px', 'lg:flex']))
    expect(toggle.attributes('aria-controls')).toBe('app-sidebar')
    expect(toggle.attributes('aria-expanded')).toBe('true')
    expect(toggle.attributes('aria-label')).toBe('nav.collapse')
    const collapseIcon = toggle.get('[data-component="sidebar-collapse-icon"]')
    expect(collapseIcon.attributes('data-testid')).toBe('header-sidebar-toggle-icon')
    expect(collapseIcon.classes()).toEqual(expect.arrayContaining(['h-5', 'w-5']))
    expect(collapseIcon.classes()).not.toContain('rotate-180')

    const pageContext = wrapper.get('h1').element.parentElement
    expect(toggle.element.nextElementSibling).toBe(pageContext)
    expect(toggle.element.parentElement?.querySelector('.w-px')).toBeNull()

    await toggle.trigger('click')

    expect(appStore.sidebarCollapsed).toBe(true)
    expect(toggle.attributes('aria-expanded')).toBe('false')
    expect(toggle.attributes('aria-label')).toBe('nav.expand')
    expect(collapseIcon.classes()).toContain('rotate-180')
    expect(wrapper.get('[data-testid="header-brand-slot"]').classes()).toEqual(
      expect.arrayContaining(['md:w-[68px]']),
    )
    expect(wrapper.get('[data-testid="header-brand-slot"]').classes()).not.toContain('lg:w-[76px]')
    expect(wrapper.get('[data-testid="header-brand"]').classes()).toContain('lg:w-[60px]')

    await toggle.trigger('click')

    expect(appStore.sidebarCollapsed).toBe(false)
    expect(toggle.attributes('aria-expanded')).toBe('true')
    expect(toggle.attributes('aria-label')).toBe('nav.collapse')
    expect(collapseIcon.classes()).not.toContain('rotate-180')
  })

  it('keeps the mobile menu state and sidebar aria relationship in sync', async () => {
    const wrapper = mountHeader()
    const appStore = useAppStore()
    const toggle = wrapper.get('[data-testid="header-mobile-menu"]')

    expect(toggle.classes()).toContain('lg:hidden')
    expect(toggle.attributes('aria-controls')).toBe('app-sidebar')
    expect(toggle.attributes('aria-expanded')).toBe('false')
    expect(toggle.attributes('aria-label')).toBe('nav.expand')

    appStore.setSidebarCollapsed(true)
    await toggle.trigger('click')

    expect(appStore.sidebarCollapsed).toBe(false)
    expect(appStore.mobileOpen).toBe(true)
    expect(toggle.attributes('aria-expanded')).toBe('true')
    expect(toggle.attributes('aria-label')).toBe('nav.collapse')

    await toggle.trigger('click')

    expect(appStore.mobileOpen).toBe(false)
    expect(toggle.attributes('aria-expanded')).toBe('false')
    expect(toggle.attributes('aria-label')).toBe('nav.expand')
  })

  it('groups compact utility icons at the medium breakpoint', () => {
    const wrapper = mountHeader()
    const actions = wrapper.get('[data-testid="header-utility-actions"]')

    expect(actions.classes()).toEqual(
      expect.arrayContaining(['hidden', 'items-center', 'gap-3', 'md:flex']),
    )
    expect(actions.find('[data-testid="announcement-bell"]').exists()).toBe(true)
    expect(actions.get('[data-testid="header-theme-toggle"]').classes()).toEqual(
      expect.arrayContaining(['h-8', 'w-8']),
    )
    expect(actions.find('[data-testid="locale-switcher"]').exists()).toBe(true)
  })

  it('uses a rounded account pill with responsive identity and accessible menu state', async () => {
    const wrapper = mountHeader()
    const trigger = wrapper.get('[data-testid="header-account-trigger"]')

    expect(trigger.classes()).toEqual(
      expect.arrayContaining(['h-8', 'rounded-full']),
    )
    expect(trigger.text()).toContain('header-user')
    const avatar = trigger.get('[data-testid="header-account-avatar"]')
    expect(avatar.text()).toBe('H')
    expect(avatar.classes()).toEqual(expect.arrayContaining(['h-6', 'w-6']))
    expect(trigger.get('.text-left').classes()).toContain('md:block')
    expect(trigger.attributes('aria-expanded')).toBe('false')
    expect(trigger.attributes('aria-controls')).toBe('header-account-menu')

    await trigger.trigger('click')

    expect(trigger.attributes('aria-expanded')).toBe('true')
    const menu = wrapper.get('#header-account-menu')
    expect(menu.attributes('role')).toBe('menu')
    expect(menu.classes()).toContain('w-56')
    expect(menu.classes()).not.toContain('w-64')
    expect(menu.get('.border-b').classes()).toEqual(expect.arrayContaining(['px-3', 'py-2']))
    expect(trigger.find('svg.rotate-180').exists()).toBe(true)

    await trigger.trigger('keydown', { key: 'Escape' })
    expect(trigger.attributes('aria-expanded')).toBe('false')
  })

  it('does not expose the upstream GitHub entry in the account menu', async () => {
    const wrapper = mountHeader({ role: 'admin' })

    await wrapper.get('[data-testid="header-account-trigger"]').trigger('click')

    const menu = wrapper.get('#header-account-menu')
    expect(menu.find('a[href="https://github.com/Wei-Shaw/sub2api"]').exists()).toBe(false)
    expect(menu.text()).not.toContain('nav.github')
  })
})

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
    const icon = toggle.get('svg')

    expect(toggle.text()).toBe('')
    expect(toggle.attributes('aria-label')).toBe('nav.darkMode')
    expect(icon.attributes('stroke-width')).toBe('2')
    expect(icon.classes()).toEqual(expect.arrayContaining(['h-5', 'w-5']))
    expect(icon.get('path').attributes('d'))
      .toBe('M20.985 12.486a9 9 0 11-9.473-9.472c.405-.022.617.46.402.803a6 6 0 008.268 8.268c.344-.215.825-.004.803.401')
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
    expect(toggle.get('path').attributes('d'))
      .toBe('M16 12a4 4 0 11-8 0 4 4 0 018 0M12 2v2m0 16v2M4.93 4.93l1.41 1.41m11.32 11.32l1.41 1.41M2 12h2m16 0h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41')

    await toggle.trigger('click')

    expect(document.documentElement.classList.contains('dark')).toBe(false)
    expect(localStorage.getItem('theme')).toBe('light')
    expect(toggle.attributes('aria-label')).toBe('nav.darkMode')
  })
})

describe('AppHeader global navigation shell', () => {
  it('keeps documentation access out of the top header', async () => {
    const wrapper = mountHeader()
    const appStore = useAppStore()

    expect(wrapper.find('[data-testid="header-docs-link"]').exists()).toBe(false)

    appStore.docUrl = '/tutorial-docs/?source=header#quick-start'
    await wrapper.vm.$nextTick()
    expect(wrapper.find('[data-testid="header-docs-link"]').exists()).toBe(false)
  })

  it('uses the snowflake credit mark for account balances', () => {
    const wrapper = mountHeader()

    expect(wrapper.findAll('[data-testid="snowflake-credit-icon"]').length).toBeGreaterThan(0)
    expect(wrapper.get('[data-testid="credit-amount"]').classes()).toEqual(
      expect.arrayContaining(['text-primary-700', 'dark:text-primary-300']),
    )
    expect(wrapper.text()).toContain('10.00')
    expect(wrapper.text()).not.toContain('$10.00')
  })

  it('renders the reference header proportions and compact mobile control', () => {
    const wrapper = mountHeader()

    expect(wrapper.get('[data-testid="app-header"]').classes()).toEqual(
      expect.arrayContaining(['fixed', 'inset-x-0', 'top-0', 'h-[81px]', 'bg-transparent', 'p-2', 'z-50']),
    )
    expect(wrapper.get('[data-testid="app-header"]').classes()).not.toContain('sticky')
    expect(wrapper.get('[data-testid="header-surface"]').classes()).toEqual(
      expect.arrayContaining([
        'topup-header-surface',
        'relative',
        'isolate',
        'h-[65px]',
        'overflow-visible',
        'rounded-2xl',
      ]),
    )
    expect(wrapper.get('[data-testid="header-surface"]').classes()).not.toContain('overflow-hidden')
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
    expect(wrapper.get('[data-testid="header-brand"]').classes()).not.toContain('hover:bg-gray-100/80')
    expect(wrapper.get('[data-testid="header-brand"]').attributes('href')).toBe('/dashboard')
    expect(wrapper.get('[data-testid="header-brand"]').attributes('aria-label')).toBe('落雪API')
    const brandImage = wrapper.get('[data-testid="header-brand"] img')
    expect(brandImage.attributes('src')).toBe('/logo.png')
    const brandLogo = wrapper.get('[data-testid="header-brand-logo"]')
    expect(brandLogo.element).toBe(brandImage.element.parentElement)
    expect(brandLogo.classes()).toEqual(
      expect.arrayContaining([
        'brand-logo-frame',
        'h-10',
        'w-10',
        'md:h-12',
        'md:w-12',
      ]),
    )
    expect(brandLogo.classes()).not.toContain('overflow-hidden')
    expect(brandLogo.classes()).not.toContain('rounded-[10px]')
    expect(brandLogo.classes()).not.toContain('bg-white')
    expect(brandLogo.classes()).not.toContain('ring-1')
    expect(brandLogo.classes()).not.toContain('transition-transform')
    expect(brandLogo.classes()).not.toContain('group-hover:scale-105')
    expect(brandImage.classes()).toEqual(
      expect.arrayContaining(['brand-logo-image', 'max-w-none']),
    )
    expect(brandImage.classes()).not.toContain('brand-logo-image-luoxue')
    const wordmark = wrapper.get('[data-testid="header-brand-wordmark"]')
    expect(wordmark.classes()).toEqual(
      expect.arrayContaining([
        'header-brand-wordmark-luoxue',
        'text-[22px]',
        'font-bold',
        'leading-7',
        'text-[#1c1f23]',
        'dark:text-[#f9f9f9]',
      ]),
    )
    expect(wordmark.classes()).not.toContain('tracking-tight')
    expect(wordmark.get('.inline-flex').classes()).toEqual(
      expect.arrayContaining(['items-baseline', 'gap-1.5', 'whitespace-nowrap']),
    )
    expect(wrapper.get('[data-testid="header-brand-name"]').text()).toBe('落雪')
    expect(wrapper.get('[data-testid="header-brand-name"]').classes()).toContain('header-brand-name-luoxue')
    expect(wrapper.get('[data-testid="header-brand-suffix"]').text()).toBe('API')
    expect(wrapper.get('[data-testid="header-brand-suffix"]').classes()).toContain('header-brand-suffix-luoxue')
    expect(wrapper.get('[data-testid="header-brand"]').text()).toContain('落雪API')
    expect(wrapper.find('[data-testid="header-surface"] p').exists()).toBe(false)
  })

  it('keeps the new snowpuff mark free of the retired snowflake effects', async () => {
    const wrapper = mountHeader()
    const appStore = useAppStore()
    const brandImage = wrapper.get('[data-testid="header-brand"] img')

    appStore.siteLogo = '/brand/luoxue-snowpuff-mark.svg'
    await wrapper.vm.$nextTick()

    expect(brandImage.attributes('src')).toBe('/brand/luoxue-snowpuff-mark.svg')
    expect(brandImage.classes()).not.toContain('brand-logo-image-luoxue')
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
    expect(brand.get('img').classes()).not.toContain('brand-logo-image-luoxue')
    expect(brand.text()).toContain('Snow Console')
    expect(brand.get('[data-testid="header-brand-wordmark"]').classes()).not.toContain('header-brand-wordmark-luoxue')
    expect(brand.get('[data-testid="header-brand-name"]').text()).toBe('Snow Console')
    expect(brand.find('[data-testid="header-brand-suffix"]').exists()).toBe(false)
  })

  it('applies tight logo treatment only to the approved Luoxue snowflake asset', async () => {
    const wrapper = mountHeader()
    const appStore = useAppStore()
    const brandImage = wrapper.get('[data-testid="header-brand"] img')

    appStore.siteLogo = '/brand/luoxue-snowflake-cloud-palette.svg'
    await wrapper.vm.$nextTick()

    expect(brandImage.classes()).toContain('brand-logo-image-luoxue')

    appStore.siteLogo = '/brand/another-logo.svg'
    await wrapper.vm.$nextTick()

    expect(brandImage.classes()).not.toContain('brand-logo-image-luoxue')
  })

  it('moves desktop sidebar collapse control into the header with synchronized aria state', async () => {
    const wrapper = mountHeader()
    const appStore = useAppStore()
    const toggle = wrapper.get('[data-testid="header-sidebar-toggle"]')

    expect(toggle.classes()).toEqual(expect.arrayContaining(['hidden', 'h-8', 'w-8', 'lg:flex']))
    expect(toggle.classes()).not.toContain('lg:-ml-px')
    expect(toggle.attributes('aria-controls')).toBe('app-sidebar')
    expect(toggle.attributes('aria-expanded')).toBe('true')
    expect(toggle.attributes('aria-label')).toBe('nav.collapse')
    const collapseIcon = toggle.get('[data-component="sidebar-collapse-icon"]')
    expect(collapseIcon.attributes('data-testid')).toBe('header-sidebar-toggle-icon')
    expect(collapseIcon.classes()).toEqual(expect.arrayContaining(['h-5', 'w-5']))
    expect(collapseIcon.classes()).not.toContain('rotate-180')

    const pageLabel = wrapper.get('[data-testid="header-surface"] h1')
    const pageContext = pageLabel.element.parentElement
    expect(pageLabel.text()).toBe('Groups')
    expect(wrapper.findAll('[data-testid="header-surface"] h1')).toHaveLength(1)
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
      expect.arrayContaining(['brand-utility-icon', 'h-8', 'w-8', 'hover:bg-[rgba(46,50,56,0.05)]']),
    )
    const localeSwitcher = actions.get('[data-testid="locale-switcher"]')
    expect(localeSwitcher.attributes('icon-variant') ?? localeSwitcher.attributes('iconvariant'))
      .toBe('lucide')
  })

  it('uses a rounded account pill with responsive identity and accessible menu state', async () => {
    const wrapper = mountHeader()
    const trigger = wrapper.get('[data-testid="header-account-trigger"]')

    expect(trigger.classes()).toEqual(
      expect.arrayContaining(['h-8', 'gap-0', 'rounded-full', 'bg-slate-900/[0.04]']),
    )
    expect(trigger.text()).toContain('header-user')
    const avatar = trigger.get('[data-testid="header-account-avatar"]')
    expect(avatar.text()).toBe('H')
    expect(avatar.classes()).toEqual(expect.arrayContaining([
      'header-account-avatar',
      'mr-1',
      'h-6',
      'w-6',
      'text-[10px]',
    ]))
    expect(avatar.classes()).not.toEqual(expect.arrayContaining([
      'bg-primary-600',
      'dark:bg-primary-500',
      'ring-1',
    ]))
    expect(trigger.get('.text-left').classes()).toContain('md:block')
    expect(trigger.attributes('aria-expanded')).toBe('false')
    expect(trigger.attributes('aria-controls')).toBe('header-account-menu')

    await trigger.trigger('click')

    expect(trigger.attributes('aria-expanded')).toBe('true')
    const menu = wrapper.get('#header-account-menu')
    expect(menu.attributes('role')).toBe('menu')
    expect(menu.classes()).toContain('w-56')
    expect(wrapper.get('[data-testid="header-surface"]').classes()).toContain('overflow-visible')
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

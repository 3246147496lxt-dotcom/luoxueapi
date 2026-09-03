import { nextTick } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/composables/useOnboardingTour', () => ({
  useOnboardingTour: () => ({ replayTour: vi.fn() }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

import AppLayout from '../AppLayout.vue'
import { useAppStore } from '@/stores'

async function createLayoutRouter(initialPath = '/dashboard') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      {
        path: '/dashboard',
        component: { template: '<div />' },
        meta: { requiresAuth: true },
      },
      {
        path: '/chat',
        component: { template: '<div />' },
        meta: { requiresAuth: true },
      },
      ...['/keys', '/usage', '/purchase', '/subscriptions', '/orders'].map(path => ({
        path,
        component: { template: '<div />' },
        meta: { requiresAuth: true },
      })),
      {
        path: '/admin/promo-codes',
        component: { template: '<div />' },
        meta: { requiresAuth: true, requiresAdmin: true },
      },
      {
        path: '/home',
        component: { template: '<div />' },
        meta: { requiresAuth: false },
      },
    ],
  })

  await router.push(initialPath)
  await router.isReady()
  return router
}

describe('AppLayout navigation structure', () => {
  afterEach(() => {
    document.body.classList.remove('admin-home-clay-portals')
    document.body.classList.remove('app-flat-workspace-active')
  })

  it('places the mobile header before the sidebar and content shell without a desktop header offset class', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = await createLayoutRouter()
    const wrapper = mount(AppLayout, {
      global: {
        plugins: [pinia, router],
        stubs: {
          AppMobileHeader: { template: '<header data-testid="mobile-header-stub" />' },
          AppSidebar: { template: '<aside data-testid="sidebar-stub" />' },
          PersonalSettingsDialog: { template: '<div v-if="false" />' },
        },
      },
      slots: { default: '<div data-testid="page-content" />' },
    })

    const rootChildren = Array.from(wrapper.element.children)
    expect(rootChildren).toHaveLength(3)
    expect(rootChildren[0]).toBe(wrapper.get('[data-testid="mobile-header-stub"]').element)
    expect(rootChildren[1]).toBe(wrapper.get('[data-testid="sidebar-stub"]').element)
    expect(rootChildren[2]).toBe(wrapper.get('[data-testid="app-main-shell"]').element)
    expect(wrapper.classes()).not.toContain('pt-[81px]')
    expect(wrapper.findAll('[data-testid="mobile-header-stub"]')).toHaveLength(1)
    expect(wrapper.findAll('[data-testid="sidebar-stub"]')).toHaveLength(1)
    expect(wrapper.find('[data-testid="page-content"]').exists()).toBe(true)
  })

  it('requests the async settings host only after the query first appears', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = await createLayoutRouter()
    const wrapper = mount(AppLayout, {
      global: {
        plugins: [pinia, router],
        stubs: {
          AppMobileHeader: true,
          AppSidebar: true,
          PersonalSettingsDialog: {
            template: '<div data-testid="personal-settings-dialog-stub" />',
          },
        },
      },
    })

    expect(wrapper.find('[data-testid="personal-settings-dialog-stub"]').exists()).toBe(false)

    await router.replace('/dashboard?account_settings=general')
    await flushPromises()

    expect(wrapper.find('[data-testid="personal-settings-dialog-stub"]').exists()).toBe(true)

    await router.replace('/dashboard')
    await flushPromises()

    // Once requested, the lightweight host stays mounted so its dialog can
    // release modal resources and restore focus after the query is removed.
    expect(wrapper.find('[data-testid="personal-settings-dialog-stub"]').exists()).toBe(true)
  })

  it('keeps the narrow Work shell at zero offset while preserving the desktop rail preference', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = await createLayoutRouter()
    const appStore = useAppStore()
    const wrapper = mount(AppLayout, {
      global: {
        plugins: [pinia, router],
        stubs: {
          AppMobileHeader: true,
          AppSidebar: true,
          PersonalSettingsDialog: { template: '<div v-if="false" />' },
        },
      },
    })
    const mainShell = wrapper.get('[data-testid="app-main-shell"]')

    expect(mainShell.classes()).not.toContain('min-h-[calc(100vh-81px)]')
    expect(mainShell.classes()).toEqual(
      expect.arrayContaining([
        'ml-[var(--workspace-sidebar-width)]',
      ]),
    )
    expect(mainShell.classes()).not.toContain('ml-[var(--workspace-sidebar-width-collapsed)]')

    appStore.setWorkspaceMobileDrawer(false)
    appStore.setWorkspaceNarrowSidebar(true)
    await nextTick()

    expect(appStore.sidebarCollapsed).toBe(false)
    expect(wrapper.attributes('data-sidebar-collapsed')).toBe('false')
    expect(mainShell.classes()).toContain('ml-0')
    expect(mainShell.classes()).not.toContain('ml-[var(--workspace-sidebar-width)]')
    expect(mainShell.classes()).not.toContain('ml-[var(--workspace-sidebar-width-collapsed)]')
    expect(wrapper.get('.workspace-sidebar-overlay-trigger').attributes('aria-expanded')).toBe('false')

    appStore.setWorkspaceNarrowSidebarOpen(true)
    await nextTick()

    expect(appStore.sidebarCollapsed).toBe(false)
    expect(wrapper.attributes('data-sidebar-collapsed')).toBe('false')
    expect(mainShell.classes()).toContain('ml-0')
    expect(mainShell.attributes('aria-hidden')).toBe('true')
    expect(mainShell.attributes('inert')).toBeDefined()
    expect(wrapper.get('.workspace-sidebar-overlay-trigger').attributes('aria-expanded')).toBe('true')

    appStore.setWorkspaceNarrowSidebar(false)
    await nextTick()

    expect(appStore.workspaceNarrowSidebarOpen).toBe(false)
    expect(wrapper.find('.workspace-sidebar-overlay-trigger').exists()).toBe(false)
    expect(mainShell.classes()).toContain('ml-[var(--workspace-sidebar-width)]')

    appStore.setSidebarCollapsed(true)
    await nextTick()

    expect(wrapper.attributes('data-sidebar-collapsed')).toBe('true')
    expect(mainShell.classes()).toContain('ml-[var(--workspace-sidebar-width-collapsed)]')
    expect(mainShell.classes()).not.toContain('ml-[var(--workspace-sidebar-width)]')

    await router.push('/admin/promo-codes')
    await nextTick()

    expect(wrapper.attributes('data-sidebar-collapsed')).toBe('true')
    expect(mainShell.classes()).toContain('ml-[var(--workspace-sidebar-width-collapsed)]')
    expect(mainShell.classes()).not.toContain('ml-[var(--workspace-sidebar-width)]')

    appStore.setSidebarCollapsed(false)
    await nextTick()

    expect(mainShell.classes()).toEqual(
      expect.arrayContaining([
        'ml-[var(--workspace-sidebar-width)]',
      ]),
    )
  })

  it.each([
    ['/dashboard', 'default'],
    ['/keys', 'chat'],
    ['/usage', 'default'],
    ['/purchase', 'purchase'],
    ['/subscriptions', 'default'],
    ['/orders', 'default'],
  ] as const)('keeps %s on the shared Work sidebar width token', async (path, variant) => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = await createLayoutRouter(path)
    const wrapper = mount(AppLayout, {
      props: { variant },
      global: {
        plugins: [pinia, router],
        stubs: {
          AppMobileHeader: { template: '<header data-testid="mobile-header-stub" />' },
          AppSidebar: { template: '<aside data-testid="sidebar-stub" />' },
          PersonalSettingsDialog: { template: '<div v-if="false" />' },
        },
      },
    })

    const mainShell = wrapper.get('[data-testid="app-main-shell"]')
    expect(wrapper.attributes('data-shell-mode')).toBe('work')
    expect(wrapper.attributes('data-sidebar-collapsed')).toBe('false')
    expect(wrapper.findAll('[data-testid="sidebar-stub"]')).toHaveLength(1)
    expect(mainShell.classes()).toContain('ml-[var(--workspace-sidebar-width)]')
    expect(mainShell.classes()).not.toContain('ml-[var(--workspace-sidebar-width-collapsed)]')
  })

  it('keeps the flat authenticated shell while the admin content adapter remains opt-in', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = await createLayoutRouter()
    const wrapper = mount(AppLayout, {
      global: {
        plugins: [pinia, router],
        stubs: {
          AppMobileHeader: {
            template: '<header data-testid="mobile-header-original" />',
          },
          AppSidebar: { template: '<aside data-testid="sidebar-original" />' },
          PersonalSettingsDialog: { template: '<div v-if="false" />' },
        },
      },
    })

    expect(wrapper.classes()).toContain('app-layout--snow-shell')
    expect(wrapper.classes()).toContain('app-layout--flat-workspace-shell')
    expect(wrapper.classes()).not.toContain('app-layout--admin-shell')
    expect(wrapper.get('[data-testid="app-main-shell"]').classes()).not.toContain('app-layout--home-clay')
    expect(wrapper.get('[data-testid="app-main-shell"]').classes()).not.toContain('app-layout--chat')
    expect(wrapper.find('[data-testid="mobile-header-original"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="sidebar-original"]').attributes('variant')).toBeUndefined()
    expect(document.body.classList.contains('admin-home-clay-portals')).toBe(false)
    expect(document.body.classList.contains('app-flat-workspace-active')).toBe(true)

    await wrapper.setProps({ variant: 'home-clay' })

    expect(wrapper.classes()).toContain('app-layout--snow-shell')
    expect(wrapper.classes()).toContain('app-layout--flat-workspace-shell')
    expect(wrapper.classes()).toContain('app-layout--admin-shell')
    expect(wrapper.get('[data-testid="app-main-shell"]').classes()).toContain('app-layout--home-clay')
    expect(wrapper.get('[data-testid="mobile-header-original"]').attributes('data-variant')).toBeUndefined()
    expect(wrapper.get('[data-testid="sidebar-original"]').attributes('variant')).toBeUndefined()
    expect(document.documentElement.classList.contains('app-layout--home-clay')).toBe(false)
    expect(document.body.classList.contains('app-layout--home-clay')).toBe(false)
    expect(document.body.classList.contains('admin-home-clay-portals')).toBe(true)
    expect(document.body.classList.contains('app-flat-workspace-active')).toBe(true)

    await wrapper.setProps({ variant: 'default' })

    expect(wrapper.classes()).toContain('app-layout--snow-shell')
    expect(wrapper.classes()).toContain('app-layout--flat-workspace-shell')
    expect(wrapper.classes()).not.toContain('app-layout--admin-shell')
    expect(wrapper.get('[data-testid="app-main-shell"]').classes()).not.toContain('app-layout--home-clay')
    expect(wrapper.get('[data-testid="mobile-header-original"]').attributes('data-variant')).toBeUndefined()
    expect(wrapper.get('[data-testid="sidebar-original"]').attributes('variant')).toBeUndefined()
    expect(document.body.classList.contains('admin-home-clay-portals')).toBe(false)
    expect(document.body.classList.contains('app-flat-workspace-active')).toBe(true)
  })

  it('derives authenticated, admin, and public shell layers from route metadata', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = await createLayoutRouter('/admin/promo-codes')
    const wrapper = mount(AppLayout, {
      global: {
        plugins: [pinia, router],
        stubs: {
          AppMobileHeader: true,
          AppSidebar: true,
          PersonalSettingsDialog: { template: '<div v-if="false" />' },
        },
      },
    })

    expect(wrapper.classes()).toContain('app-layout--flat-workspace-shell')
    expect(wrapper.classes()).toContain('app-layout--admin-shell')
    expect(wrapper.get('[data-testid="app-main-shell"]').classes()).toContain('app-layout--home-clay')
    expect(document.body.classList.contains('admin-home-clay-portals')).toBe(true)
    expect(document.body.classList.contains('app-flat-workspace-active')).toBe(true)

    await router.push('/dashboard')
    await nextTick()

    expect(wrapper.classes()).toContain('app-layout--flat-workspace-shell')
    expect(wrapper.classes()).not.toContain('app-layout--admin-shell')
    expect(wrapper.get('[data-testid="app-main-shell"]').classes()).not.toContain('app-layout--home-clay')
    expect(document.body.classList.contains('admin-home-clay-portals')).toBe(false)
    expect(document.body.classList.contains('app-flat-workspace-active')).toBe(true)

    await router.push('/home')
    await nextTick()

    expect(wrapper.classes()).not.toContain('app-layout--flat-workspace-shell')
    expect(wrapper.classes()).not.toContain('app-layout--admin-shell')
    expect(wrapper.get('[data-testid="app-main-shell"]').classes()).not.toContain('app-layout--home-clay')
    expect(document.body.classList.contains('admin-home-clay-portals')).toBe(false)
    expect(document.body.classList.contains('app-flat-workspace-active')).toBe(false)
  })

  it('layers the chat workspace behavior on the authenticated flat shell', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = await createLayoutRouter('/chat')
    const wrapper = mount(AppLayout, {
      props: { variant: 'chat', shellMode: 'chat' },
      global: {
        plugins: [pinia, router],
        stubs: {
          AppMobileHeader: true,
          AppSidebar: true,
          PersonalSettingsDialog: { template: '<div v-if="false" />' },
        },
      },
    })

    expect(wrapper.classes()).toContain('app-layout--flat-workspace-shell')
    expect(wrapper.classes()).not.toContain('app-layout--admin-shell')
    expect(wrapper.get('[data-testid="app-main-shell"]').classes()).toContain('app-layout--chat')
    expect(wrapper.get('[data-testid="app-main-shell"]').classes()).not.toContain('app-layout--home-clay')
    expect(wrapper.attributes('data-sidebar-collapsed')).toBeUndefined()
    expect(document.body.classList.contains('admin-home-clay-portals')).toBe(false)
    expect(document.body.classList.contains('app-flat-workspace-active')).toBe(true)

    wrapper.unmount()

    expect(document.body.classList.contains('app-flat-workspace-active')).toBe(false)
  })
})

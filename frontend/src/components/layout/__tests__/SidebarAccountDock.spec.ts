import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { PublicSettings } from '@/types'

const summaryState = vi.hoisted(() => ({
  hasUser: true,
  displayName: 'Riley Quinn',
  email: 'riley@example.com',
  initials: 'R',
  avatarUrl: '',
  unreadAnnouncementCount: 2,
  isAdmin: false,
  isSimpleMode: false,
  availableBalance: 24.5,
  frozenBalance: 3,
  activeSubscriptionCount: 1,
  subscriptionsLoaded: true,
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, values?: Record<string, unknown>) => (
        values ? `${key}:${JSON.stringify(values)}` : key
      ),
    }),
  }
})

vi.mock('@/composables/useAccountSummary', async () => {
  const { computed } = await vi.importActual<typeof import('vue')>('vue')
  return {
    useAccountSummary: () => ({
      hasUser: computed(() => summaryState.hasUser),
      displayName: computed(() => summaryState.displayName),
      email: computed(() => summaryState.email),
      initials: computed(() => summaryState.initials),
      avatarUrl: computed(() => summaryState.avatarUrl),
      unreadAnnouncementCount: computed(() => summaryState.unreadAnnouncementCount),
      isAdmin: computed(() => summaryState.isAdmin),
      isSimpleMode: computed(() => summaryState.isSimpleMode),
      availableBalance: computed(() => summaryState.availableBalance),
      frozenBalance: computed(() => summaryState.frozenBalance),
      activeSubscriptionCount: computed(() => summaryState.activeSubscriptionCount),
      subscriptionsLoaded: computed(() => summaryState.subscriptionsLoaded),
    }),
  }
})

import SidebarAccountDock from '../SidebarAccountDock.vue'
import { useAppStore, useAuthStore } from '@/stores'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../SidebarAccountDock.vue')
const componentSource = readFileSync(componentPath, 'utf8')
const mountedWrappers: VueWrapper[] = []

function installMatchMedia(mobile: boolean) {
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    value: vi.fn((query: string) => ({
      matches: query.includes('max-width: 1023px') ? mobile : false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  })
}

async function mountDock(settings: Partial<PublicSettings> = {}) {
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: { template: '<div />' } },
      { path: '/dashboard', component: { template: '<div />' } },
      { path: '/admin/dashboard', component: { template: '<div />' } },
      {
        path: '/monitor',
        component: { template: '<div />' },
        meta: { requiresAuth: true },
      },
      {
        path: '/admin/users',
        component: { template: '<div />' },
        meta: { requiresAuth: true, requiresAdmin: true },
      },
      { path: '/pricing', component: { template: '<div />' } },
      { path: '/subscriptions', component: { template: '<div />' } },
      { path: '/login', component: { template: '<div />' } },
    ],
  })
  await router.push('/')
  await router.isReady()

  const appStore = useAppStore()
  appStore.cachedPublicSettings = {
    payment_enabled: true,
    public_model_catalog_enabled: true,
    doc_url: '/docs/',
    contact_info: '/support',
    ...settings,
  } as PublicSettings

  const wrapper = mount(SidebarAccountDock, {
    global: {
      plugins: [pinia, router],
      stubs: {
        CreditAmount: {
          props: ['value', 'iconSize', 'label'],
          template: '<span data-testid="credit-amount" :data-value="value" :aria-label="label">{{ value }}</span>',
        },
        SidebarAccountOverlay: {
          props: ['open', 'anchorElement', 'summary', 'showOnboarding'],
          emits: ['close', 'logout', 'replay', 'open-settings'],
          template: `
            <section v-if="open" data-testid="overlay-stub">
              <button
                data-testid="stub-profile"
                @click="$emit('open-settings', 'account')"
              />
              <button
                data-testid="stub-preferences"
                @click="$emit('open-settings', 'general')"
              />
              <button
                v-if="showOnboarding"
                data-testid="stub-admin-guide"
                @click="$emit('replay')"
              />
              <button data-testid="stub-close" @click="$emit('close', true)" />
              <button data-testid="stub-logout" @click="$emit('logout')" />
            </section>
          `,
        },
      },
    },
  })
  mountedWrappers.push(wrapper)

  return { wrapper, router, appStore, authStore: useAuthStore() }
}

describe('SidebarAccountDock', () => {
  beforeEach(() => {
    Object.assign(summaryState, {
      hasUser: true,
      displayName: 'Riley Quinn',
      email: 'riley@example.com',
      initials: 'R',
      avatarUrl: '',
      unreadAnnouncementCount: 2,
      isAdmin: false,
      isSimpleMode: false,
      availableBalance: 24.5,
      frozenBalance: 3,
      activeSubscriptionCount: 1,
      subscriptionsLoaded: true,
    })
    installMatchMedia(false)
    document.documentElement.classList.remove('dark')
    localStorage.clear()
  })

  afterEach(() => {
    mountedWrappers.splice(0).forEach((wrapper) => wrapper.unmount())
    document.documentElement.classList.remove('dark')
    localStorage.clear()
  })

  it('uses the passive account summary without starting fetch or polling work', () => {
    expect(componentSource).toContain("import { useAccountSummary } from '@/composables/useAccountSummary'")
    expect(componentSource).toContain('const summary = useAccountSummary()')
    expect(componentSource).not.toMatch(/\bfetch(?:Subscriptions|Announcements|ActiveSubscriptions)\b/)
    expect(componentSource).not.toContain('setInterval(')
  })

  it('keeps announcements and help out of the account dock', async () => {
    const { wrapper } = await mountDock()

    expect(wrapper.find('[data-testid="sidebar-account-tools"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="account-notifications-tool"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="account-help-link"]').exists()).toBe(false)
    expect(componentSource).not.toContain('AnnouncementBell')
    expect(componentSource).not.toContain('resolveDocumentationUrl')
    expect(componentSource).not.toContain('.sidebar-account-tool')
  })

  it('renders a labeled balance and a complete accessible account summary', async () => {
    const { wrapper } = await mountDock()
    const trigger = wrapper.get('.sidebar-account-trigger')

    expect(wrapper.get('[data-testid="sidebar-account-dock"]').text()).toContain('Riley Quinn')
    expect(wrapper.get('.sidebar-account-trigger__balance-label').text()).toBe(
      'accountDock.balanceShort',
    )
    expect(wrapper.get('[data-testid="credit-amount"]').attributes('data-value')).toBe('24.50')
    expect(wrapper.get('.sidebar-account-trigger__avatar').text()).toBe('R')
    expect(wrapper.find('.sidebar-account-trigger__badge').exists()).toBe(false)
    expect(trigger.attributes('aria-label')).toContain('Riley Quinn')
    expect(trigger.attributes('aria-label')).toContain('24.50')
    expect(trigger.attributes('aria-label')).toContain('accountDock.activeSubscriptions')
    expect(trigger.attributes('aria-haspopup')).toBe('dialog')
    expect(trigger.attributes('aria-controls')).toBe('sidebar-account-panel')
    expect(trigger.attributes('aria-expanded')).toBe('false')

    await trigger.trigger('click')

    expect(trigger.attributes('aria-expanded')).toBe('true')
    expect(wrapper.find('[data-testid="overlay-stub"]').exists()).toBe(true)
  })

  it('restores focus to the account trigger without scrolling the page', async () => {
    const focus = vi.spyOn(HTMLElement.prototype, 'focus')
    const { wrapper } = await mountDock()
    const trigger = wrapper.get('.sidebar-account-trigger')

    await trigger.trigger('click')
    await wrapper.get('[data-testid="stub-close"]').trigger('click')
    await flushPromises()

    expect(focus).toHaveBeenCalledWith({ preventScroll: true })
    expect(focus.mock.instances).toContain(trigger.element)
  })

  it('always offers Upgrade to pricing for an active subscription', async () => {
    const { wrapper } = await mountDock()
    const upgrade = wrapper.get('[data-testid="account-upgrade-link"]')

    expect(upgrade.attributes('href')).toBe('/pricing')
    expect(upgrade.text()).toBe('accountDock.upgrade')
    expect(componentSource).not.toContain("findVisibleRouteDestination('subscriptions')")
    expect(componentSource).not.toContain('accountDock.manageSubscription')
  })

  it('uses destination visibility gates without waiting for subscriptions to load', async () => {
    const disabled = await mountDock({ payment_enabled: false })
    expect(disabled.wrapper.find('[data-testid="account-upgrade-link"]').exists()).toBe(false)

    summaryState.subscriptionsLoaded = false
    const loading = await mountDock()
    expect(loading.wrapper.get('[data-testid="account-upgrade-link"]').attributes('href'))
      .toBe('/pricing')
    expect(loading.wrapper.text()).toContain('accountDock.subscriptionLoading')

    summaryState.isSimpleMode = true
    const simple = await mountDock()
    expect(simple.wrapper.find('[data-testid="account-upgrade-link"]').exists()).toBe(false)
  })

  it('keeps Upgrade available while payment capability is unknown', async () => {
    const { wrapper } = await mountDock({ payment_enabled: undefined })

    expect(wrapper.get('[data-testid="account-upgrade-link"]').attributes('href'))
      .toBe('/pricing')
  })

  it('hides the upgrade CTA and preserves an accessible trigger when collapsed', async () => {
    const { wrapper, appStore } = await mountDock()
    appStore.setSidebarCollapsed(true)
    await flushPromises()

    expect(wrapper.find('[data-testid="account-upgrade-link"]').exists()).toBe(false)
    expect(wrapper.get('.sidebar-account-trigger').attributes('title')).toContain('Riley Quinn')
  })

  it.each([
    { control: 'stub-profile', section: 'account' },
    { control: 'stub-preferences', section: 'general' },
  ])('opens the $section personal settings section and preserves route state', async ({
    control,
    section,
  }) => {
    const { wrapper, router } = await mountDock()
    await router.replace({
      path: '/monitor',
      query: { source: 'account-menu' },
      hash: '#usage',
    })
    await wrapper.get('.sidebar-account-trigger').trigger('click')

    await wrapper.get(`[data-testid="${control}"]`).trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.query).toEqual({
      source: 'account-menu',
      account_settings: section,
    })
    expect(router.currentRoute.value.path).toBe('/monitor')
    expect(router.currentRoute.value.hash).toBe('#usage')
    expect(wrapper.find('[data-testid="overlay-stub"]').exists()).toBe(false)
  })

  it.each([
    { control: 'stub-profile', section: 'account' },
    { control: 'stub-preferences', section: 'general' },
  ])('keeps an admin\'s $section settings over the personal workspace', async ({
    control,
    section,
  }) => {
    summaryState.isAdmin = true
    const { wrapper, router } = await mountDock()
    await router.replace('/monitor')
    await wrapper.get('.sidebar-account-trigger').trigger('click')

    await wrapper.get(`[data-testid="${control}"]`).trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.path).toBe('/monitor')
    expect(router.currentRoute.value.query.account_settings).toBe(section)
  })

  it('keeps admin settings over the current admin workspace and exposes the guide', async () => {
    summaryState.isAdmin = true
    const { wrapper, router } = await mountDock()
    await router.replace('/admin/users')
    await wrapper.get('.sidebar-account-trigger').trigger('click')

    expect(wrapper.find('[data-testid="stub-admin-guide"]').exists()).toBe(true)
    await wrapper.get('[data-testid="stub-preferences"]').trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.path).toBe('/admin/users')
    expect(router.currentRoute.value.query.account_settings).toBe('general')
  })

  it('logs out and replaces the current route with login', async () => {
    const { wrapper, router, authStore } = await mountDock()
    const logout = vi.spyOn(authStore, 'logout').mockResolvedValue(undefined)
    await wrapper.get('.sidebar-account-trigger').trigger('click')

    await wrapper.get('[data-testid="stub-logout"]').trigger('click')
    await flushPromises()

    expect(logout).toHaveBeenCalledOnce()
    expect(router.currentRoute.value.path).toBe('/login')
  })

  it('closes the mobile sidebar before opening the account panel', async () => {
    installMatchMedia(true)
    const { wrapper, appStore } = await mountDock()
    appStore.setMobileOpen(true)

    await wrapper.get('.sidebar-account-trigger').trigger('click')

    expect(appStore.mobileOpen).toBe(false)
    expect(wrapper.get('.sidebar-account-trigger').attributes('aria-expanded')).toBe('true')
  })

  it('closes the mobile sidebar when Upgrade is activated', async () => {
    installMatchMedia(true)
    summaryState.subscriptionsLoaded = false
    const { wrapper, appStore, router } = await mountDock()
    appStore.setMobileOpen(true)

    await wrapper.get('[data-testid="account-upgrade-link"]').trigger('click')
    await flushPromises()

    expect(appStore.mobileOpen).toBe(false)
    expect(router.currentRoute.value.path).toBe('/pricing')
    expect(wrapper.get('.sidebar-account-trigger').attributes('aria-expanded')).toBe('false')
  })
})

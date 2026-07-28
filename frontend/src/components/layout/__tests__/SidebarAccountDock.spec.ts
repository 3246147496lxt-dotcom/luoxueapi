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
        Icon: {
          props: ['name'],
          template: '<span :data-icon="name" />',
        },
        SidebarAccountOverlay: {
          props: [
            'open',
            'anchorElement',
            'summary',
            'purchaseLink',
            'subscriptionLink',
            'resourceLinks',
            'showOnboarding',
          ],
          emits: ['close', 'logout', 'replay', 'open-settings'],
          template: `
            <section v-if="open" data-testid="overlay-stub">
              <a
                v-if="purchaseLink"
                :data-account-link="purchaseLink.id"
                :href="purchaseLink.to"
              >{{ purchaseLink.label }}</a>
              <RouterLink
                v-if="subscriptionLink"
                data-testid="stub-subscriptions"
                :data-account-link="subscriptionLink.id"
                :data-account-label="subscriptionLink.label"
                :data-account-icon="subscriptionLink.icon"
                :to="subscriptionLink.to"
                @click="$emit('close', false)"
              >{{ subscriptionLink.label }}</RouterLink>
              <a
                v-for="link in resourceLinks"
                :key="link.id"
                :data-resource-link="link.id"
                :href="link.href"
              >{{ link.label }}</a>
              <button data-testid="stub-settings" @click="$emit('open-settings')" />
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

function renderedIds(wrapper: VueWrapper, attribute: string) {
  return wrapper.findAll(`[${attribute}]`).map((element) => element.attributes(attribute))
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

  it('renders the passive identity summary and exposes dialog trigger semantics', async () => {
    const { wrapper } = await mountDock()
    const trigger = wrapper.get('.sidebar-account-trigger')
    const upgrade = wrapper.get('[data-testid="account-upgrade-link"]')

    expect(wrapper.get('[data-testid="sidebar-account-dock"]').text()).toContain('Riley Quinn')
    expect(wrapper.get('[data-testid="credit-amount"]').attributes('data-value')).toBe('24.50')
    expect(wrapper.get('.sidebar-account-trigger__avatar').text()).toBe('R')
    expect(wrapper.find('.sidebar-account-trigger__badge').exists()).toBe(true)
    expect(trigger.element.tagName).toBe('BUTTON')
    expect(upgrade.element.tagName).toBe('A')
    expect(trigger.element.contains(upgrade.element)).toBe(false)
    expect(upgrade.attributes('href')).toBe('/pricing')
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

  it('applies feature gates to account and resource destinations', async () => {
    const disabled = await mountDock({
      payment_enabled: false,
      public_model_catalog_enabled: false,
    })
    await disabled.wrapper.get('.sidebar-account-trigger').trigger('click')

    expect(disabled.wrapper.find('[data-testid="account-upgrade-link"]').exists()).toBe(false)
    expect(renderedIds(disabled.wrapper, 'data-account-link')).toEqual([
      'wallet',
      'subscriptions',
    ])
    expect(renderedIds(disabled.wrapper, 'data-resource-link')).toEqual([
      'home',
      'contact',
    ])

    const enabled = await mountDock()
    await enabled.wrapper.get('.sidebar-account-trigger').trigger('click')

    expect(enabled.wrapper.get('[data-testid="account-upgrade-link"]').attributes('href')).toBe(
      '/pricing',
    )
    expect(renderedIds(enabled.wrapper, 'data-account-link')).toEqual([
      'wallet',
      'subscriptions',
    ])
    const subscriptionLink = enabled.wrapper.get('[data-testid="stub-subscriptions"]')
    expect(subscriptionLink.attributes('href')).toBe('/subscriptions')
    expect(subscriptionLink.attributes('data-account-label')).toBe('nav.mySubscriptions')
    expect(subscriptionLink.attributes('data-account-icon')).toBe('creditCard')
    expect(renderedIds(enabled.wrapper, 'data-resource-link')).toEqual([
      'home',
      'models',
      'contact',
    ])
  })

  it('keeps the recharge entry visible without exposing deep pages in simple mode', async () => {
    summaryState.isSimpleMode = true
    const { wrapper } = await mountDock()

    await wrapper.get('.sidebar-account-trigger').trigger('click')

    expect(wrapper.find('[data-testid="account-upgrade-link"]').exists()).toBe(false)
    expect(renderedIds(wrapper, 'data-account-link')).toEqual(['wallet'])
    expect(wrapper.find('[data-testid="stub-subscriptions"]').exists()).toBe(false)
  })

  it('exposes the subscriptions destination for administrator accounts', async () => {
    summaryState.isAdmin = true
    const { wrapper } = await mountDock()

    await wrapper.get('.sidebar-account-trigger').trigger('click')

    expect(wrapper.get('[data-testid="stub-subscriptions"]').attributes('href'))
      .toBe('/subscriptions')
  })

  it('keeps the upgrade destination visible while payment capability is unknown', async () => {
    const { wrapper } = await mountDock({ payment_enabled: undefined })

    expect(wrapper.get('[data-testid="account-upgrade-link"]').attributes('href')).toBe('/pricing')
  })

  it('navigates with the independent upgrade link without reopening the account panel', async () => {
    const { wrapper, router } = await mountDock()
    const trigger = wrapper.get('.sidebar-account-trigger')

    await trigger.trigger('click')
    expect(trigger.attributes('aria-expanded')).toBe('true')

    await wrapper.get('[data-testid="account-upgrade-link"]').trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.path).toBe('/pricing')
    expect(trigger.attributes('aria-expanded')).toBe('false')
    expect(wrapper.find('[data-testid="overlay-stub"]').exists()).toBe(false)
  })

  it.each([
    { label: 'desktop', mobile: false },
    { label: 'mobile', mobile: true },
  ])('navigates to subscriptions and closes the $label account panel', async ({ mobile }) => {
    installMatchMedia(mobile)
    const { wrapper, router } = await mountDock()
    const trigger = wrapper.get('.sidebar-account-trigger')

    await trigger.trigger('click')
    expect(trigger.attributes('aria-expanded')).toBe('true')

    await wrapper.get('[data-testid="stub-subscriptions"]').trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.path).toBe('/subscriptions')
    expect(trigger.attributes('aria-expanded')).toBe('false')
    expect(wrapper.find('[data-testid="overlay-stub"]').exists()).toBe(false)
  })

  it('hides the upgrade action when the desktop sidebar is collapsed', async () => {
    const { wrapper, appStore } = await mountDock()

    expect(wrapper.find('[data-testid="account-upgrade-link"]').exists()).toBe(true)
    appStore.setSidebarCollapsed(true)
    await flushPromises()

    expect(wrapper.find('[data-testid="account-upgrade-link"]').exists()).toBe(false)
  })

  it('opens general settings in place and preserves unrelated route state', async () => {
    const { wrapper, router } = await mountDock()
    await router.replace({
      path: '/',
      query: { source: 'account-menu' },
      hash: '#usage',
    })
    await wrapper.get('.sidebar-account-trigger').trigger('click')

    await wrapper.get('[data-testid="stub-settings"]').trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.query).toEqual({
      source: 'account-menu',
      account_settings: 'general',
    })
    expect(router.currentRoute.value.path).toBe('/dashboard')
    expect(router.currentRoute.value.hash).toBe('#usage')
    expect(wrapper.find('[data-testid="overlay-stub"]').exists()).toBe(false)
  })

  it('uses the admin settings host for administrator accounts', async () => {
    summaryState.isAdmin = true
    const { wrapper, router } = await mountDock()
    await wrapper.get('.sidebar-account-trigger').trigger('click')

    await wrapper.get('[data-testid="stub-settings"]').trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.path).toBe('/admin/dashboard')
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

  it('closes the mobile sidebar when the independent upgrade link is activated', async () => {
    installMatchMedia(true)
    const { wrapper, appStore, router } = await mountDock()
    appStore.setMobileOpen(true)

    await wrapper.get('[data-testid="account-upgrade-link"]').trigger('click')
    await flushPromises()

    expect(appStore.mobileOpen).toBe(false)
    expect(router.currentRoute.value.path).toBe('/pricing')
    expect(wrapper.get('.sidebar-account-trigger').attributes('aria-expanded')).toBe('false')
  })
})

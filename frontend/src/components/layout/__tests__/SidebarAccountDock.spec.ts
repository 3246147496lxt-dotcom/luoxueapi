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
  primarySubscription: {
    id: 1,
    name: 'Pro',
    expiresAt: null,
  } as { id: number; name: string | null; expiresAt: string | null } | null,
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

vi.mock('@/stores/userProfile', async () => {
  const { computed, reactive } = await vi.importActual<typeof import('vue')>('vue')
  return {
    useUserProfileStore: () => reactive({
      profile: computed(() => summaryState.hasUser ? {
        id: 7,
        username: summaryState.displayName,
        displayName: summaryState.displayName,
        email: summaryState.email,
        initials: summaryState.initials,
        avatarUrl: summaryState.avatarUrl,
        role: summaryState.isAdmin ? 'admin' : 'user',
        availableBalance: summaryState.availableBalance,
        frozenBalance: summaryState.frozenBalance,
        currentPlan: {
          state: summaryState.subscriptionsLoaded
            ? (summaryState.activeSubscriptionCount > 0 ? 'active' : 'free')
            : 'pending',
          name: summaryState.primarySubscription?.name ?? null,
          activeCount: summaryState.activeSubscriptionCount,
          expiresAt: summaryState.primarySubscription?.expiresAt ?? null,
        },
      } : null),
      activeSubscriptionCount: computed(() => summaryState.activeSubscriptionCount),
      subscriptionsLoaded: computed(() => summaryState.subscriptionsLoaded),
      primarySubscription: computed(() => summaryState.primarySubscription),
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

async function mountDock(
  settings: Partial<PublicSettings> = {},
  context: 'work' | 'chat' = 'work',
  collapsed = false,
) {
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
      {
        path: '/admin/ops',
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

  const authStore = useAuthStore()
  authStore.user = {
    id: 7,
    username: summaryState.displayName,
    email: summaryState.email,
    role: summaryState.isAdmin ? 'admin' : 'user',
    balance: summaryState.availableBalance,
    frozen_balance: summaryState.frozenBalance,
    concurrency: 1,
    status: 'active',
    allowed_groups: null,
    balance_notify_enabled: false,
    balance_notify_threshold: null,
    balance_notify_extra_emails: [],
    created_at: '2026-08-01T00:00:00Z',
    updated_at: '2026-08-01T00:00:00Z',
  }

  const wrapper = mount(SidebarAccountDock, {
    props: { context, collapsed },
    global: {
      plugins: [pinia, router],
      stubs: {
        CreditAmount: {
          props: ['value', 'iconSize', 'label'],
          template: '<span data-testid="credit-amount" :data-value="value" :aria-label="label">{{ value }}</span>',
        },
        SidebarAccountOverlay: {
          props: [
            'open',
            'anchorElement',
            'summary',
            'showOnboarding',
            'context',
            'variant',
            'appearance',
            'planLabel',
            'helpHref',
            'workspaceTarget',
          ],
          emits: ['close', 'logout', 'replay', 'open-settings'],
          template: `
            <section
              v-if="open"
              data-testid="overlay-stub"
              :data-workspace-target="workspaceTarget?.href ?? ''"
              :data-variant="variant"
              :data-appearance="appearance"
              :data-plan-label="planLabel"
              :data-help-href="helpHref"
            >
              <button
                data-testid="stub-profile"
                @click="$emit('open-settings', 'account')"
              />
              <button
                data-testid="stub-preferences"
                @click="$emit('open-settings', 'general')"
              />
              <button
                data-testid="stub-settings"
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

  return { wrapper, router, appStore, authStore }
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
      primarySubscription: {
        id: 1,
        name: 'Pro',
        expiresAt: null,
      },
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

  it('reads the shared user profile without starting fetch or polling work', () => {
    expect(componentSource).toContain("import { useUserProfileStore } from '@/stores/userProfile'")
    expect(componentSource).toContain('const userProfileStore = useUserProfileStore()')
    expect(componentSource).not.toContain('useAccountSummary')
    expect(componentSource).not.toMatch(/\bfetch(?:Subscriptions|Announcements|ActiveSubscriptions)\b/)
    expect(componentSource).not.toContain('setInterval(')
  })

  it('keeps help out of the dock row and passes the existing docs destination to the overlay', async () => {
    const { wrapper } = await mountDock()

    expect(wrapper.find('[data-testid="sidebar-account-tools"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="account-notifications-tool"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="account-help-link"]').exists()).toBe(false)
    expect(componentSource).not.toContain('AnnouncementBell')
    expect(componentSource).toContain('resolveDocumentationUrl')
    expect(componentSource).not.toContain('.sidebar-account-tool')

    await wrapper.get('.sidebar-account-trigger').trigger('click')
    expect(wrapper.get('[data-testid="overlay-stub"]').attributes('data-help-href')).toBe('/docs/')
  })

  it('renders the plan state and a complete accessible account summary', async () => {
    const { wrapper } = await mountDock()
    const trigger = wrapper.get('.sidebar-account-trigger')

    expect(wrapper.get('[data-testid="sidebar-account-dock"]').text()).toContain('Riley Quinn')
    expect(wrapper.get('.sidebar-account-trigger__meta').text()).toContain('Pro')
    expect(wrapper.find('[data-testid="credit-amount"]').exists()).toBe(false)
    expect(wrapper.get('.sidebar-account-trigger__avatar').text()).toBe('R')
    expect(wrapper.find('.sidebar-account-trigger__chevrons').exists()).toBe(true)
    expect(wrapper.find('.sidebar-account-trigger__badge').exists()).toBe(false)
    expect(trigger.attributes('aria-label')).toContain('Riley Quinn')
    expect(trigger.attributes('aria-label')).toContain('Pro')
    expect(trigger.attributes('aria-label')).not.toContain('24.50')
    expect(trigger.attributes('aria-haspopup')).toBe('dialog')
    expect(trigger.attributes('aria-controls')).toBe('sidebar-account-panel')
    expect(trigger.attributes('aria-expanded')).toBe('false')

    await trigger.trigger('click')

    expect(trigger.attributes('aria-expanded')).toBe('true')
    expect(wrapper.find('[data-testid="overlay-stub"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="overlay-stub"]').attributes('data-plan-label')).toBe('Pro')
    expect(wrapper.get('[data-testid="overlay-stub"]').attributes('data-variant')).toBe('personal')
  })

  it.each(['work', 'chat'] as const)(
    'shows Free for a confirmed user without a subscription in %s mode',
    async (context) => {
      summaryState.availableBalance = 10_000
      summaryState.activeSubscriptionCount = 0
      summaryState.subscriptionsLoaded = true
      summaryState.primarySubscription = null

      const { wrapper } = await mountDock({}, context)
      const trigger = wrapper.get('.sidebar-account-trigger')

      expect(wrapper.get('.sidebar-account-trigger__meta').text()).toBe('accountDock.free')
      expect(trigger.attributes('aria-label')).toContain('accountDock.free')
      expect(trigger.attributes('aria-label')).not.toContain('accountDock.payAsYouGo')

      await trigger.trigger('click')
      expect(wrapper.get('[data-testid="overlay-stub"]').attributes('data-plan-label'))
        .toBe('accountDock.free')
    },
  )

  it('keeps a real subscription name independent from a zero wallet balance', async () => {
    summaryState.availableBalance = 0
    summaryState.activeSubscriptionCount = 1
    summaryState.primarySubscription = { id: 1, name: 'Pro', expiresAt: null }

    const { wrapper } = await mountDock()

    expect(wrapper.get('.sidebar-account-trigger__meta').text()).toBe('Pro')
    expect(wrapper.get('.sidebar-account-trigger').attributes('aria-label')).toContain('Pro')
  })

  it('keeps the Chat rail account entry quiet and free of billing UI', async () => {
    const { wrapper } = await mountDock({}, 'chat')
    const trigger = wrapper.get('.sidebar-account-trigger')

    expect(wrapper.text()).toContain('Pro')
    expect(wrapper.text()).not.toContain('riley@example.com')
    expect(wrapper.find('[data-testid="credit-amount"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="account-upgrade-link"]').exists()).toBe(false)
    expect(wrapper.find('.sidebar-account-trigger__chevrons').exists()).toBe(true)
    expect(trigger.attributes('aria-label')).toContain('Riley Quinn')
    expect(trigger.attributes('aria-label')).toContain('Pro')
    expect(trigger.attributes('aria-label')).not.toContain('riley@example.com')
    expect(trigger.attributes('aria-label')).not.toContain('24.50')
    expect(componentSource).toMatch(
      /\.sidebar-account-dock--personal \.sidebar-account-trigger__avatar\s*\{[^}]*width: 24px;[^}]*height: 24px;[^}]*border-radius: var\(--workspace-radius-pill\);/,
    )
    expect(componentSource).toMatch(
      /\.sidebar-account-dock--personal \.sidebar-account-trigger__name\s*\{[^}]*font-size: 14px;[^}]*font-weight: 400;[^}]*line-height: 20px;/,
    )
    expect(componentSource).toMatch(
      /\.sidebar-account-dock--personal \.sidebar-account-trigger__meta\s*\{[^}]*color: var\(--workspace-identity-text-tertiary\);[^}]*font-size: 12px;[^}]*font-weight: 400;[^}]*line-height: 16px;/,
    )
    expect(componentSource).toMatch(
      /\.sidebar-account-trigger__chevrons\s*\{[^}]*width: 36px;[^}]*height: 36px;[^}]*border-radius: var\(--workspace-radius-compact\);/,
    )
    expect(componentSource).toContain(
      ':global(html.dark .sidebar-account-dock--personal .sidebar-account-trigger__avatar)',
    )
  })

  it('uses one personal identity visual contract in Chat and Work', () => {
    expect(componentSource).toContain("'sidebar-account-dock--chat': context === 'chat'")
    expect(componentSource).toContain(
      "'sidebar-account-dock--work': context === 'work' && !isAdminWorkspace",
    )
    expect(componentSource).not.toContain(
      'html:not(.dark) .sidebar-account-dock--work .sidebar-account-trigger__avatar',
    )
    expect(componentSource).not.toContain(
      'html:not(.dark) .sidebar-account-dock--work .sidebar-account-trigger__meta',
    )
    expect(componentSource).toContain('color: var(--workspace-identity-text);')
    expect(componentSource).toContain('color: var(--workspace-identity-text-tertiary);')
    expect(componentSource).not.toMatch(/#(?:0d0d0d|fff|ffffff|8f8f8f|afafaf)\b/i)
  })

  it.each(['work', 'chat'] as const)(
    'collapses the %s account entry only when the host requests it',
    async (context) => {
      const { wrapper } = await mountDock({}, context, true)
      const dock = wrapper.get('[data-testid="sidebar-account-dock"]')
      const trigger = wrapper.get('.sidebar-account-trigger')

      expect(dock.classes()).toContain('sidebar-account-dock--collapsed')
      expect(wrapper.get('.sidebar-account-trigger__copy').attributes('aria-hidden')).toBe('true')
      expect(wrapper.find('.sidebar-account-trigger__chevrons').exists()).toBe(false)
      expect(trigger.attributes('title')).toContain('Riley Quinn')
      expect(trigger.attributes('aria-label')).toContain('Pro')
      expect(componentSource).toMatch(
        /\.sidebar-account-dock--collapsed \.sidebar-account-row\s*\{[^}]*width: var\(--workspace-sidebar-touch-target\);/s,
      )
      expect(componentSource).toMatch(
        /\.sidebar-account-dock--collapsed \.sidebar-account-trigger\s*\{[^}]*justify-content: start;/s,
      )
      expect(componentSource).not.toMatch(
        /\.sidebar-account-dock--collapsed \.sidebar-account-trigger\s*\{[^}]*justify-content: center;/s,
      )
      expect(componentSource).toMatch(
        /\.sidebar-account-dock--collapsed \.sidebar-account-trigger__copy\s*\{[^}]*max-width: 0;[^}]*opacity: 0;/s,
      )
      expect(componentSource).not.toMatch(
        /\.sidebar-account-dock--collapsed \.sidebar-account-trigger__copy\s*\{[^}]*display: none;/s,
      )
    },
  )

  it('keeps administrator destinations out of the Chat account overlay', async () => {
    summaryState.isAdmin = true
    const { wrapper } = await mountDock({}, 'chat')

    await wrapper.get('.sidebar-account-trigger').trigger('click')

    expect(wrapper.get('[data-testid="overlay-stub"]').attributes('data-workspace-target')).toBe('')
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

  it('restores mobile Chat focus to its own account trigger', async () => {
    installMatchMedia(true)
    const focus = vi.spyOn(HTMLElement.prototype, 'focus')
    const { wrapper } = await mountDock({}, 'chat')
    const trigger = wrapper.get('.sidebar-account-trigger')

    await trigger.trigger('click')
    await wrapper.get('[data-testid="stub-close"]').trigger('click')
    await flushPromises()

    expect(focus.mock.instances).toContain(trigger.element)
  })

  it('does not render an Upgrade control in the shared account area', async () => {
    const { wrapper } = await mountDock()

    expect(wrapper.find('[data-testid="account-upgrade-link"]').exists()).toBe(false)
    expect(componentSource).not.toContain('account-upgrade-link')
    expect(componentSource).not.toContain('pricingTarget')
    expect(componentSource).not.toContain("findVisibleRouteDestination('subscriptions')")
    expect(componentSource).not.toContain('accountDock.manageSubscription')
  })

  it('keeps the administrator footer on the same identity-only pattern', async () => {
    summaryState.isAdmin = true
    const { wrapper, router } = await mountDock({ payment_enabled: true })
    await router.replace('/admin/users')
    await flushPromises()

    expect(wrapper.find('[data-testid="account-upgrade-link"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="sidebar-account-dock"]').classes())
      .toContain('sidebar-account-dock--personal')
    expect(wrapper.find('.sidebar-account-trigger__chevrons').exists()).toBe(true)
  })

  it('keeps the real account identity and user-side appearance on every admin route', async () => {
    summaryState.isAdmin = true
    summaryState.avatarUrl = '/avatars/riley.png'
    summaryState.primarySubscription = { id: 1, name: 'Ultra', expiresAt: null }
    const { wrapper, router } = await mountDock({ payment_enabled: true })

    const readAccountState = async (path: '/admin/users' | '/admin/ops') => {
      await router.replace(path)
      await flushPromises()

      const trigger = wrapper.get('.sidebar-account-trigger')
      const avatar = wrapper.get<HTMLImageElement>('.sidebar-account-trigger__avatar img')
      const state = {
        name: wrapper.get('.sidebar-account-trigger__name').text(),
        meta: wrapper.get('.sidebar-account-trigger__meta').text(),
        avatar: avatar.attributes('src'),
        avatarAlt: avatar.attributes('alt'),
        ariaLabel: trigger.attributes('aria-label'),
        hasChevrons: wrapper.find('.sidebar-account-trigger__chevrons').exists(),
        hasUpgrade: wrapper.find('[data-testid="account-upgrade-link"]').exists(),
      }

      await trigger.trigger('click')
      const overlay = wrapper.get('[data-testid="overlay-stub"]')
      expect(overlay.attributes('data-variant')).toBe('admin')
      expect(overlay.attributes('data-appearance')).toBe('personal')
      expect(overlay.attributes('data-workspace-target')).toBe('/dashboard')
      expect(wrapper.find('[data-testid="stub-admin-guide"]').exists()).toBe(true)
      await wrapper.get('[data-testid="stub-close"]').trigger('click')
      return state
    }

    const usersState = await readAccountState('/admin/users')
    const operationsState = await readAccountState('/admin/ops')

    expect(operationsState).toEqual(usersState)
    expect(operationsState).toEqual({
      name: 'Riley Quinn',
      meta: 'Ultra',
      avatar: '/avatars/riley.png',
      avatarAlt: 'Riley Quinn',
      ariaLabel: 'accountDock.open · Riley Quinn · Ultra',
      hasChevrons: true,
      hasUpgrade: false,
    })
    expect(componentSource).not.toContain('isOpsOptionB')
    expect(componentSource).not.toContain('admin.ops.sidebar')
  })

  it('does not infer personal Work collapse from the application store', async () => {
    const { wrapper, appStore } = await mountDock()
    appStore.setSidebarCollapsed(true)
    await flushPromises()

    expect(wrapper.get('[data-testid="sidebar-account-dock"]').classes())
      .not.toContain('sidebar-account-dock--collapsed')
    expect(wrapper.find('[data-testid="account-upgrade-link"]').exists()).toBe(false)
    expect(wrapper.get('.sidebar-account-trigger').attributes('title')).toBeUndefined()
  })

  it('uses the host prop instead of inferring admin collapse from the store', async () => {
    summaryState.isAdmin = true
    const { wrapper, router, appStore } = await mountDock({}, 'work', true)
    await router.replace('/admin/users')
    appStore.setSidebarCollapsed(false)
    await flushPromises()

    expect(wrapper.get('[data-testid="sidebar-account-dock"]').classes())
      .toContain('sidebar-account-dock--collapsed')
    expect(wrapper.get('.sidebar-account-trigger').attributes('title')).toContain('Riley Quinn')
    expect(componentSource).not.toContain('appStore.sidebarCollapsed')
  })

  it.each([
    { control: 'stub-profile', section: 'account' },
    { control: 'stub-preferences', section: 'general' },
    { control: 'stub-settings', section: 'general' },
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
    { control: 'stub-settings', section: 'general' },
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

  it('keeps the mobile sidebar open while the personal fixed card is visible', async () => {
    installMatchMedia(true)
    const { wrapper, appStore } = await mountDock()
    appStore.setMobileOpen(true)

    await wrapper.get('.sidebar-account-trigger').trigger('click')

    expect(appStore.mobileOpen).toBe(true)
    expect(wrapper.get('.sidebar-account-trigger').attributes('aria-expanded')).toBe('true')

    appStore.setMobileOpen(false)
    await flushPromises()

    expect(wrapper.find('[data-testid="overlay-stub"]').exists()).toBe(false)
    expect(wrapper.get('.sidebar-account-trigger').attributes('aria-expanded')).toBe('false')
  })

  it('opens the administrator account sheet after closing the mobile sidebar', async () => {
    installMatchMedia(true)
    summaryState.isAdmin = true
    const { wrapper, appStore, router } = await mountDock()
    await router.replace('/admin/users')
    await flushPromises()
    appStore.setWorkspaceMobileDrawer(true)
    appStore.setMobileOpen(true)

    await wrapper.get('.sidebar-account-trigger').trigger('click')
    await flushPromises()

    expect(appStore.mobileOpen).toBe(false)
    expect(router.currentRoute.value.path).toBe('/admin/users')
    expect(wrapper.get('.sidebar-account-trigger').attributes('aria-expanded')).toBe('true')
    expect(wrapper.find('[data-testid="overlay-stub"]').exists()).toBe(true)
  })
})

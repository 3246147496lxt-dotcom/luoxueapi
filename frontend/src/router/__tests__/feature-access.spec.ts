import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'

type NavigationGuard = (
  to: Record<string, any>,
  from: Record<string, any>,
  next: ReturnType<typeof vi.fn>
) => Promise<void>

const routerHarness = vi.hoisted(() => ({
  guards: {
    user: null as NavigationGuard | null,
    admin: null as NavigationGuard | null,
  },
}))

const authStore = vi.hoisted(() => ({
  checkAuth: vi.fn(),
  isAuthenticated: true,
  isAdmin: false,
  isSimpleMode: false,
  hasPendingAuthSession: false,
}))

const appStore = vi.hoisted(() => ({
  siteName: 'Sub2API',
  backendModeEnabled: false,
  publicSettingsLoaded: false,
  cachedPublicSettings: null as null | {
    payment_enabled?: boolean
    risk_control_enabled?: boolean
    skill_marketplace_enabled?: boolean
    custom_menu_items?: []
  },
  fetchPublicSettings: vi.fn(),
}))

const adminComplianceStore = vi.hoisted(() => ({
  initialized: true,
  shouldShow: false,
  fetchStatus: vi.fn(),
  requireAcknowledgement: vi.fn(),
}))

vi.mock('vue-router', () => ({
  createWebHistory: vi.fn(() => ({})),
  createRouter: vi.fn((options: { routes: Array<{ name?: string }> }) => {
    const entry = options.routes.some((route) => route.name === 'AdminDashboard')
      ? 'admin'
      : 'user'
    return {
      beforeEach: vi.fn((guard: NavigationGuard) => {
        routerHarness.guards[entry] = guard
      }),
      afterEach: vi.fn(),
      onError: vi.fn(),
    }
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStore,
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => ({ customMenuItems: [] }),
}))

vi.mock('@/stores/adminCompliance', () => ({
  useAdminComplianceStore: () => adminComplianceStore,
}))

vi.mock('@/composables/useNavigationLoading', () => ({
  useNavigationLoadingState: () => ({
    startNavigation: vi.fn(),
    endNavigation: vi.fn(),
    isLoading: { value: false },
  }),
}))

vi.mock('@/composables/useRoutePrefetch', () => ({
  useRoutePrefetch: () => ({
    triggerPrefetch: vi.fn(),
    cancelPendingPrefetch: vi.fn(),
    resetPrefetchState: vi.fn(),
  }),
}))

function createDeferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  const promise = new Promise<T>((resolvePromise) => {
    resolve = resolvePromise
  })
  return { promise, resolve }
}

function runGuard(
  meta: Record<string, unknown>,
  path: string,
  query: Record<string, string | string[]> = {},
  hash = '',
) {
  const entry = path === '/admin' || path.startsWith('/admin/') ? 'admin' : 'user'
  const guard = routerHarness.guards[entry]
  if (!guard) {
    throw new Error('router guard was not registered')
  }

  const next = vi.fn()
  const searchParams = new URLSearchParams()
  Object.entries(query).forEach(([key, value]) => {
    const values = Array.isArray(value) ? value : [value]
    values.forEach((entry) => searchParams.append(key, entry))
  })
  const search = searchParams.toString()
  const fullPath = `${path}${search ? `?${search}` : ''}${hash}`
  const navigation = guard(
    {
      path,
      fullPath,
      name: 'FeatureRoute',
      params: {},
      query,
      hash,
      meta: { requiresAuth: true, ...meta },
    },
    {},
    next
  )
  return { navigation, next }
}

describe('feature route guard', () => {
  beforeAll(async () => {
    await import('@/router')
    await import('@/router/admin')
  })

  beforeEach(() => {
    authStore.isAuthenticated = true
    authStore.isAdmin = false
    authStore.isSimpleMode = false
    appStore.publicSettingsLoaded = false
    appStore.cachedPublicSettings = null
    appStore.fetchPublicSettings.mockReset()
    adminComplianceStore.initialized = true
    adminComplianceStore.shouldShow = false
    adminComplianceStore.fetchStatus.mockReset()
    adminComplianceStore.requireAcknowledgement.mockReset()
    adminComplianceStore.requireAcknowledgement.mockImplementation(() => {
      adminComplianceStore.initialized = true
      adminComplianceStore.shouldShow = true
    })
  })

  it('waits for the first public-settings request before deciding payment access', async () => {
    const deferred = createDeferred<{ payment_enabled: boolean }>()
    appStore.fetchPublicSettings.mockImplementation(async () => {
      const settings = await deferred.promise
      appStore.cachedPublicSettings = settings
      appStore.publicSettingsLoaded = true
      return settings
    })

    const { navigation, next } = runGuard({ requiresPayment: true }, '/purchase')

    await vi.waitFor(() => expect(appStore.fetchPublicSettings).toHaveBeenCalledTimes(1))
    expect(next).not.toHaveBeenCalled()

    deferred.resolve({ payment_enabled: true })
    await navigation
    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith()
  })

  it('always refreshes the Skill marketplace flag before opening a public Skill route', async () => {
    appStore.publicSettingsLoaded = true
    appStore.cachedPublicSettings = { skill_marketplace_enabled: true }
    appStore.fetchPublicSettings.mockImplementation(async () => {
      const settings = { skill_marketplace_enabled: false }
      appStore.cachedPublicSettings = settings
      return settings
    })

    const { navigation, next } = runGuard(
      { requiresAuth: false, requiresSkillMarketplace: true },
      '/skills/frontend-design',
    )
    await navigation

    expect(appStore.fetchPublicSettings).toHaveBeenCalledOnce()
    expect(appStore.fetchPublicSettings).toHaveBeenCalledWith(true)
    expect(next).toHaveBeenCalledWith('/home')
  })

  it('opens a public Skill route after the refreshed flag is enabled', async () => {
    appStore.publicSettingsLoaded = true
    appStore.cachedPublicSettings = { skill_marketplace_enabled: false }
    appStore.fetchPublicSettings.mockImplementation(async () => {
      const settings = { skill_marketplace_enabled: true }
      appStore.cachedPublicSettings = settings
      return settings
    })

    const { navigation, next } = runGuard(
      { requiresAuth: false, requiresSkillMarketplace: true },
      '/skills',
    )
    await navigation

    expect(appStore.fetchPublicSettings).toHaveBeenCalledOnce()
    expect(appStore.fetchPublicSettings).toHaveBeenCalledWith(true)
    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith()
  })

  it('fails closed when the Skill marketplace flag cannot be refreshed', async () => {
    appStore.publicSettingsLoaded = true
    appStore.cachedPublicSettings = { skill_marketplace_enabled: true }
    appStore.fetchPublicSettings.mockResolvedValue(null)

    const { navigation, next } = runGuard(
      { requiresAuth: false, requiresSkillMarketplace: true },
      '/skills',
    )
    await navigation

    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith('/home')
  })

  it('revalidates the Skill marketplace flag inside the authenticated Work shell', async () => {
    appStore.publicSettingsLoaded = true
    appStore.cachedPublicSettings = { skill_marketplace_enabled: true }
    appStore.fetchPublicSettings.mockResolvedValue({ skill_marketplace_enabled: false })

    const { navigation, next } = runGuard(
      { requiresAuth: true, requiresSkillMarketplace: true },
      '/skills',
    )
    await navigation

    expect(appStore.fetchPublicSettings).toHaveBeenCalledOnce()
    expect(appStore.fetchPublicSettings).toHaveBeenCalledWith(true)
    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith('/dashboard')
  })

  it('waits for public settings before opening a fresh subscription plan bridge', async () => {
    const deferred = createDeferred<{ payment_enabled: boolean }>()
    appStore.fetchPublicSettings.mockImplementation(async () => {
      const settings = await deferred.promise
      appStore.cachedPublicSettings = settings
      appStore.publicSettingsLoaded = true
      return settings
    })

    const { navigation, next } = runGuard(
      {},
      '/purchase',
      { tab: 'subscription', plan: '7' },
    )

    await vi.waitFor(() => expect(appStore.fetchPublicSettings).toHaveBeenCalledTimes(1))
    expect(next).not.toHaveBeenCalled()

    deferred.resolve({ payment_enabled: true })
    await navigation
    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith()
  })

  it('redirects a fresh subscription plan bridge when payment is disabled', async () => {
    appStore.cachedPublicSettings = { payment_enabled: false }
    appStore.publicSettingsLoaded = true

    const { navigation, next } = runGuard(
      {},
      '/purchase',
      { tab: 'subscription', plan: '7' },
    )
    await navigation

    expect(appStore.fetchPublicSettings).not.toHaveBeenCalled()
    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith('/dashboard')
  })

  it('keeps the integrated purchase page available in simple mode for web-chat recharge', async () => {
    authStore.isSimpleMode = true

    const { navigation, next } = runGuard({}, '/purchase')
    await navigation

    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith()
  })

  it('blocks a fresh subscription plan bridge in simple mode', async () => {
    authStore.isSimpleMode = true
    appStore.cachedPublicSettings = { payment_enabled: true }
    appStore.publicSettingsLoaded = true

    const { navigation, next } = runGuard(
      {},
      '/purchase',
      { tab: 'subscription', plan: '7' },
    )
    await navigation

    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith('/dashboard')
  })

  it('does not let auxiliary callback state bypass the simple-mode plan gate', async () => {
    authStore.isSimpleMode = true
    appStore.cachedPublicSettings = { payment_enabled: true }
    appStore.publicSettingsLoaded = true

    const { navigation, next } = runGuard(
      {},
      '/purchase',
      {
        tab: 'subscription',
        plan: '7',
        state: 'x',
      },
    )
    await navigation

    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith('/dashboard')
  })

  it('canonicalizes a repeated tab before simple-mode capability checks', async () => {
    authStore.isSimpleMode = true
    appStore.cachedPublicSettings = { payment_enabled: false }
    appStore.publicSettingsLoaded = true

    const { navigation, next } = runGuard(
      {},
      '/purchase',
      {
        tab: ['subscription', 'recharge'],
        plan: '7',
        group: '3',
        source: 'account-menu',
      },
    )
    await navigation

    expect(appStore.fetchPublicSettings).not.toHaveBeenCalled()
    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith({
      path: '/purchase',
      query: { source: 'account-menu' },
      hash: '',
      replace: true,
    })
  })

  it('canonicalizes redemption before simple-mode capability checks', async () => {
    authStore.isSimpleMode = true
    appStore.cachedPublicSettings = { payment_enabled: false }
    appStore.publicSettingsLoaded = true

    const { navigation, next } = runGuard(
      {},
      '/purchase',
      {
        tab: 'subscription',
        plan: '7',
        group: '3',
        source: 'wallet',
      },
      '#redeem',
    )
    await navigation

    expect(appStore.fetchPublicSettings).not.toHaveBeenCalled()
    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith({
      path: '/purchase',
      query: { source: 'wallet' },
      hash: '#redeem',
      replace: true,
    })
  })

  it('keeps a complete WeChat recovery available in simple mode', async () => {
    authStore.isSimpleMode = true
    appStore.cachedPublicSettings = { payment_enabled: false }
    appStore.publicSettingsLoaded = true

    const { navigation, next } = runGuard(
      {},
      '/purchase',
      {
        tab: 'subscription',
        plan: '7',
        wechat_resume: '1',
        wechat_resume_token: 'resume-7',
      },
    )
    await navigation

    expect(appStore.fetchPublicSettings).not.toHaveBeenCalled()
    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith()
  })

  it('keeps ordinary recharge available when payment and simple mode are disabled', async () => {
    authStore.isSimpleMode = true
    appStore.cachedPublicSettings = { payment_enabled: false }
    appStore.publicSettingsLoaded = true

    const { navigation, next } = runGuard(
      {},
      '/purchase',
      { tab: 'recharge' },
    )
    await navigation

    expect(appStore.fetchPublicSettings).not.toHaveBeenCalled()
    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith()
  })

  it('canonicalizes the retired subscription catalogue before capability checks', async () => {
    const { navigation, next } = runGuard(
      {},
      '/purchase',
      { tab: 'subscription', source: 'account-menu' },
      '#plans',
    )
    await navigation

    expect(appStore.fetchPublicSettings).not.toHaveBeenCalled()
    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith({
      path: '/pricing',
      query: { source: 'account-menu' },
      hash: '#plans',
      replace: true,
    })
  })

  it.each([
    ['payment', { requiresPayment: true }, '/purchase'],
    ['risk control', { requiresRiskControl: true }, '/admin/risk-control'],
  ])('does not treat a failed %s settings load as explicitly disabled', async (_name, meta, path) => {
    authStore.isAdmin = 'requiresRiskControl' in meta && meta.requiresRiskControl === true
    appStore.fetchPublicSettings.mockResolvedValue(null)

    const { navigation, next } = runGuard(meta, path)
    await navigation

    expect(appStore.publicSettingsLoaded).toBe(false)
    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith()
  })

  it.each([
    ['payment', { requiresPayment: true }, { payment_enabled: false }, '/dashboard'],
    [
      'risk control',
      { requiresRiskControl: true },
      { risk_control_enabled: false },
      '/admin/settings',
    ],
  ])('redirects when loaded settings explicitly disable %s', async (_name, meta, settings, target) => {
    authStore.isAdmin = 'requiresRiskControl' in meta && meta.requiresRiskControl === true
    appStore.cachedPublicSettings = settings
    appStore.publicSettingsLoaded = true

    const { navigation, next } = runGuard(meta, '/feature')
    await navigation

    expect(appStore.fetchPublicSettings).not.toHaveBeenCalled()
    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith(target)
  })

  it('removes settings state when mandatory admin compliance is incomplete', async () => {
    authStore.isAdmin = true
    adminComplianceStore.shouldShow = true

    const { navigation, next } = runGuard(
      { requiresAdmin: true },
      '/admin/dashboard',
      {
        source: 'operations',
        account_settings: 'security',
        account_settings_detail: 'totp',
      },
      '#review',
    )
    await navigation

    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith({
      path: '/admin/dashboard',
      query: {
        source: 'operations',
      },
      hash: '#review',
      state: {
        __sub2api_personal_settings_owner: undefined,
      },
      replace: true,
    })
  })

  it('checks compliance before an admin activates settings on the user dashboard', async () => {
    authStore.isAdmin = true
    adminComplianceStore.initialized = false
    adminComplianceStore.fetchStatus.mockImplementation(async () => {
      adminComplianceStore.initialized = true
      adminComplianceStore.shouldShow = true
    })

    const { navigation, next } = runGuard(
      {},
      '/dashboard',
      {
        return_to: 'usage',
        account_settings: 'general',
      },
      '#limits',
    )
    await navigation

    expect(adminComplianceStore.fetchStatus).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith({
      path: '/dashboard',
      query: {
        return_to: 'usage',
      },
      hash: '#limits',
      state: {
        __sub2api_personal_settings_owner: undefined,
      },
      replace: true,
    })
  })
})

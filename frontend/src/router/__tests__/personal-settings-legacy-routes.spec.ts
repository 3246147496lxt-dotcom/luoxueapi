import { describe, expect, it, vi } from 'vitest'
import type {
  NavigationGuard,
  RouteLocationNormalized,
  RouteLocationRaw,
  RouteLocationResolved,
} from 'vue-router'

const authStore = vi.hoisted(() => ({
  checkAuth: vi.fn(),
  isAuthenticated: true,
  isAdmin: false,
  isSimpleMode: false,
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    siteName: 'Sub2API',
    backendModeEnabled: false,
    cachedPublicSettings: null,
  }),
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => ({ customMenuItems: [] }),
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

function callBeforeEnter(
  guard: NavigationGuard | NavigationGuard[] | undefined,
  to: RouteLocationNormalized | RouteLocationResolved,
): RouteLocationRaw | undefined {
  if (typeof guard !== 'function') return undefined
  return guard(
    to as RouteLocationNormalized,
    to as RouteLocationNormalized,
    vi.fn(),
  ) as RouteLocationRaw | undefined
}

describe('legacy personal settings routes', () => {
  it.each([
    ['/profile', 'account', 'profile'],
    ['/profile/security', 'security', undefined],
    ['/profile/connections', 'account', 'connections'],
    ['/settings/profile', 'account', 'connections'],
  ])('bridges %s into the user dashboard modal', async (path, section, detail) => {
    authStore.isAdmin = false
    const { default: router } = await import('@/router')
    const record = router.getRoutes().find((candidate) => candidate.path === path)
    const resolved = router.resolve(`${path}?oauth=complete#binding`)

    expect(record?.meta.requiresAuth).toBe(true)
    expect(record?.meta.requiresAdmin).toBe(false)
    expect(record?.components?.default).toBeTruthy()
    expect(record?.redirect).toBeUndefined()

    expect(callBeforeEnter(record?.beforeEnter, resolved)).toEqual({
      path: '/dashboard',
      query: {
        oauth: 'complete',
        account_settings: section,
        ...(detail ? { account_settings_detail: detail } : {}),
      },
      hash: '#binding',
    })
  })

  it('bridges authenticated administrators to the admin dashboard', async () => {
    authStore.isAdmin = true
    const { default: router } = await import('@/router')
    const record = router.getRoutes().find((candidate) => candidate.path === '/profile')
    const resolved = router.resolve('/profile?source=oauth#profile')

    expect(callBeforeEnter(record?.beforeEnter, resolved)).toEqual({
      path: '/admin/dashboard',
      query: {
        source: 'oauth',
        account_settings: 'account',
        account_settings_detail: 'profile',
      },
      hash: '#profile',
    })
  })

  it('leaves full transaction routes renderable and unchanged', async () => {
    const { default: router } = await import('@/router')
    const routes = router.getRoutes()

    for (const path of [
      '/subscriptions',
      '/orders',
      '/purchase',
      '/payment/qrcode',
      '/payment/result',
      '/payment/stripe',
      '/payment/airwallex',
      '/payment/stripe-popup',
    ]) {
      const record = routes.find((candidate) => candidate.path === path)
      expect(record?.components?.default, path).toBeTruthy()
      expect(record?.redirect, path).toBeUndefined()
    }
  })
})

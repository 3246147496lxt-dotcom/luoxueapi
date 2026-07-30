import { describe, expect, it, vi } from 'vitest'

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    checkAuth: vi.fn(),
    isAuthenticated: true,
    isAdmin: false,
    isSimpleMode: false,
    hasPendingAuthSession: false,
  }),
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    siteName: 'Sub2API',
    backendModeEnabled: false,
    publicSettingsLoaded: true,
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

describe('quota viewer routes', () => {
  it('registers the authorization page as an authenticated user route', async () => {
    const { default: router } = await import('@/router')
    const route = router.getRoutes().find((record) => record.name === 'QuotaViewerAuthorize')

    expect(route?.path).toBe('/quota-viewer/authorize')
    expect(route?.components?.default).toBeTypeOf('function')
    expect(route?.meta).toMatchObject({
      requiresAuth: true,
      requiresAdmin: false,
      titleKey: 'quotaViewerAuthorization.pageTitle',
    })
  })

  it('registers device management as an authenticated user route', async () => {
    const { default: router } = await import('@/router')
    const route = router.getRoutes().find((record) => record.name === 'QuotaViewerDevices')

    expect(route?.path).toBe('/quota-viewer/devices')
    expect(route?.components?.default).toBeTypeOf('function')
    expect(route?.meta).toMatchObject({
      requiresAuth: true,
      requiresAdmin: false,
      titleKey: 'quotaViewerDevices.title',
    })
  })
})

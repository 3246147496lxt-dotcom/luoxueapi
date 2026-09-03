import { describe, expect, it, vi } from 'vitest'

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    checkAuth: vi.fn(),
    isAuthenticated: true,
    isAdmin: true,
    isSimpleMode: false
  })
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    siteName: 'Sub2API',
    backendModeEnabled: false,
    cachedPublicSettings: null
  })
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => ({ customMenuItems: [] })
}))

vi.mock('@/composables/useNavigationLoading', () => ({
  useNavigationLoadingState: () => ({
    startNavigation: vi.fn(),
    endNavigation: vi.fn(),
    isLoading: { value: false }
  })
}))

vi.mock('@/composables/useRoutePrefetch', () => ({
  useRoutePrefetch: () => ({
    triggerPrefetch: vi.fn(),
    cancelPendingPrefetch: vi.fn(),
    resetPrefetchState: vi.fn()
  })
}))

describe('admin shell route contract', () => {
  it('marks every renderable admin route for the shared admin shell', async () => {
    const { default: router } = await import('@/router/admin')
    const adminRoutes = router.getRoutes().filter((route) => route.path.startsWith('/admin'))
    const renderableRoutes = adminRoutes.filter((route) => route.components?.default)

    expect(renderableRoutes.length).toBeGreaterThan(0)
    renderableRoutes.forEach((route) => {
      expect(route.meta.requiresAuth, route.path).toBe(true)
      expect(route.meta.requiresAdmin, route.path).toBe(true)
    })
  })

  it('keeps non-rendering admin entries as explicit redirects', async () => {
    const { default: router } = await import('@/router/admin')
    const redirectRoutes = router
      .getRoutes()
      .filter((route) => route.path.startsWith('/admin') && !route.components?.default)

    expect(redirectRoutes.map((route) => [route.path, route.redirect])).toEqual(
      expect.arrayContaining([
        ['/admin', '/admin/dashboard'],
        ['/admin/channels', '/admin/channels/pricing'],
        ['/admin/affiliates', '/admin/affiliates/invites']
      ])
    )
    redirectRoutes.forEach((route) => {
      expect(route.redirect, route.path).toBeTruthy()
    })
  })
})

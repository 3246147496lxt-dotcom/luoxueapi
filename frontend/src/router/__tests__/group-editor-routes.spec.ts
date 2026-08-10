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

describe('group editor routes', () => {
  it('registers admin-only create and edit workspaces', async () => {
    const { default: router } = await import('@/router/admin')
    const createRoute = router.getRoutes().find(record => record.name === 'AdminGroupCreate')
    const editRoute = router.getRoutes().find(record => record.name === 'AdminGroupEdit')

    expect(createRoute?.path).toBe('/admin/groups/new')
    expect(createRoute?.meta).toMatchObject({ requiresAuth: true, requiresAdmin: true })
    expect(editRoute?.path).toBe('/admin/groups/:id([1-9]\\d*)/edit')
    expect(editRoute?.meta).toMatchObject({ requiresAuth: true, requiresAdmin: true })
  })

  it('resolves numeric edit IDs without shadowing the create route', async () => {
    const { default: router } = await import('@/router/admin')

    expect(router.resolve('/admin/groups/new').name).toBe('AdminGroupCreate')
    expect(router.resolve('/admin/groups/42/edit').name).toBe('AdminGroupEdit')
    expect(router.resolve('/admin/groups/not-a-number/edit').name).not.toBe('AdminGroupEdit')
  })
})

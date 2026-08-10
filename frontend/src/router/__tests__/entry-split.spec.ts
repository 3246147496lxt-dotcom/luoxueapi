import { createMemoryHistory } from 'vue-router'
import { describe, expect, it, vi } from 'vitest'
import { createAppRouter, getAppRoutes } from '@/router/createAppRouter'
import {
  isAdminPath,
  shouldBridgeToOtherEntry,
} from '@/router/guards'
import { adminRoutes } from '@/router/routes/admin'
import { publicRoutes } from '@/router/routes/public'
import { userRoutes } from '@/router/routes/user'

describe('router entry split', () => {
  it('keeps public, user, and admin route ownership disjoint', () => {
    expect(publicRoutes.length).toBeGreaterThan(0)
    expect(userRoutes.length).toBeGreaterThan(0)
    expect(adminRoutes.length).toBeGreaterThan(0)

    expect(publicRoutes.some((route) => isAdminPath(route.path))).toBe(false)
    expect(userRoutes.some((route) => isAdminPath(route.path))).toBe(false)
    expect(adminRoutes.every((route) => isAdminPath(route.path))).toBe(true)
  })

  it('only gives each entry its owned renderable routes', () => {
    const userEntryRoutes = getAppRoutes('user', [...publicRoutes, ...userRoutes])
    const adminEntryRoutes = getAppRoutes('admin', adminRoutes)

    expect(userEntryRoutes.some((route) => route.name === 'Dashboard')).toBe(true)
    expect(userEntryRoutes.some((route) => route.name === 'AdminDashboard')).toBe(false)
    expect(adminEntryRoutes.some((route) => route.name === 'AdminDashboard')).toBe(true)
    expect(adminEntryRoutes.some((route) => route.name === 'Dashboard')).toBe(false)
    expect(adminEntryRoutes[adminEntryRoutes.length - 2]?.name).toBe('AdminNotFound')
    expect(adminEntryRoutes[adminEntryRoutes.length - 1]?.name).toBe('AdminToUserFullPageBridge')
  })

  it('classifies cross-entry paths without treating lookalikes as admin routes', () => {
    expect(shouldBridgeToOtherEntry('user', '/admin')).toBe(true)
    expect(shouldBridgeToOtherEntry('user', '/admin/users')).toBe(true)
    expect(shouldBridgeToOtherEntry('user', '/administrator')).toBe(false)
    expect(shouldBridgeToOtherEntry('admin', '/admin/settings')).toBe(false)
    expect(shouldBridgeToOtherEntry('admin', '/dashboard')).toBe(true)
  })

  it('uses a full-page bridge from the user entry while preserving query and hash', async () => {
    const fullPageNavigator = vi.fn()
    const router = createAppRouter({
      entry: 'user',
      routes: [...publicRoutes, ...userRoutes],
      history: createMemoryHistory(),
      fullPageNavigator,
    })

    await router.push('/admin/users?status=active#results')

    expect(fullPageNavigator).toHaveBeenCalledOnce()
    expect(fullPageNavigator).toHaveBeenCalledWith('/admin/users?status=active#results')
    expect(router.currentRoute.value.path).not.toBe('/admin/users')
  })

  it('uses a full-page bridge from the admin entry for user-owned paths', async () => {
    const fullPageNavigator = vi.fn()
    const router = createAppRouter({
      entry: 'admin',
      routes: adminRoutes,
      history: createMemoryHistory(),
      fullPageNavigator,
    })

    await router.push('/dashboard?source=workspace-switch')

    expect(fullPageNavigator).toHaveBeenCalledOnce()
    expect(fullPageNavigator).toHaveBeenCalledWith('/dashboard?source=workspace-switch')
    expect(router.currentRoute.value.path).not.toBe('/dashboard')
  })
})

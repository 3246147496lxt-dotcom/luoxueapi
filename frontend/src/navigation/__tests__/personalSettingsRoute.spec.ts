import { afterEach, describe, expect, it, vi } from 'vitest'
import type { LocationQuery, RouteLocationRaw, Router } from 'vue-router'
import {
  canonicalizePersonalSettingsRoute,
  closePersonalSettings,
  getPersonalSettingsDetailSection,
  getPersonalSettingsHostPath,
  openPersonalSettings,
  parsePersonalSettingsRoute,
  replacePersonalSettingsDetail,
  replacePersonalSettingsSection,
  resolveLegacyPersonalSettingsRedirect,
  resolvePersonalSettingsCanonicalization,
  type PersonalSettingsRouteLocation,
} from '../personalSettingsRoute'

function createRoute(
  query: LocationQuery = {},
  path = '/dashboard',
  hash = '#usage',
): PersonalSettingsRouteLocation {
  return { path, query, hash }
}

function createRouter() {
  return {
    push: vi.fn().mockResolvedValue(undefined),
    replace: vi.fn().mockResolvedValue(undefined),
    back: vi.fn(),
    forward: vi.fn(),
    go: vi.fn(),
  } as unknown as Router
}

function getFirstNavigationTarget(
  navigation: ReturnType<typeof vi.fn>,
): Exclude<RouteLocationRaw, string> {
  return navigation.mock.calls[0]?.[0] as Exclude<RouteLocationRaw, string>
}

interface SettingsHistoryOwner {
  version: number
  ownerToken: string
  documentId: string
  chainId: string
  depth: 0 | 1
  originPosition: number
}

function getHistoryOwner(
  target: Exclude<RouteLocationRaw, string>,
): SettingsHistoryOwner {
  const state = target.state as Record<string, unknown> | undefined
  return state?.__sub2api_personal_settings_owner as SettingsHistoryOwner
}

function restoreHistoryTarget(
  target: Exclude<RouteLocationRaw, string>,
  position: number,
) {
  window.history.replaceState({
    ...(target.state ?? {}),
    position,
  }, '')
}

afterEach(() => {
  window.history.replaceState({ position: 0 }, '')
})

describe('personalSettingsRoute', () => {
  it('opens only on the two controlled dashboard hosts', () => {
    for (const path of ['/dashboard', '/admin/dashboard']) {
      expect(parsePersonalSettingsRoute(createRoute({
        account_settings: 'general',
      }, path))).toMatchObject({
        isOpen: true,
        section: 'general',
        detail: null,
        needsCanonicalization: false,
      })
    }

    for (const path of [
      '/purchase',
      '/subscriptions',
      '/orders',
      '/payment/qrcode',
      '/payment/result',
      '/payment/stripe',
      '/payment/airwallex',
      '/payment/stripe-popup',
      '/keys',
    ]) {
      expect(parsePersonalSettingsRoute(createRoute({
        account_settings: 'general',
      }, path))).toMatchObject({
        isOpen: false,
        needsCanonicalization: true,
      })
    }
  })

  it('normalizes invalid, repeated, and mismatched query values', () => {
    for (const query of ([
      { account_settings: 'unknown' },
      { account_settings: ['account', 'security'] },
      { account_settings_detail: 'connections' },
      {
        account_settings: 'account',
        account_settings_detail: ['connections', 'profile'],
      },
      {
        account_settings: 'account',
        account_settings_detail: 'password',
      },
    ] as LocationQuery[])) {
      const parsed = parsePersonalSettingsRoute(createRoute(query))
      expect(parsed.needsCanonicalization).toBe(true)
    }

    expect(resolvePersonalSettingsCanonicalization(createRoute({
      source: 'oauth',
      account_settings: 'account',
      account_settings_detail: 'password',
    }))).toEqual({
      path: '/dashboard',
      query: {
        source: 'oauth',
        account_settings: 'account',
      },
      hash: '#usage',
      replace: true,
    })
  })

  it('removes settings query from payment and transaction routes without changing their location', () => {
    for (const path of [
      '/purchase',
      '/subscriptions',
      '/orders',
      '/payment/qrcode',
      '/payment/result',
      '/payment/stripe',
      '/payment/airwallex',
      '/payment/stripe-popup',
    ]) {
      expect(resolvePersonalSettingsCanonicalization(createRoute({
        order_id: 'order-1',
        account_settings: 'billing',
        account_settings_detail: 'totp',
      }, path, '#checkout'))).toEqual({
        path,
        query: {
          order_id: 'order-1',
        },
        hash: '#checkout',
        state: {
          __sub2api_personal_settings_owner: undefined,
        },
        replace: true,
      })
    }
  })

  it('executes replace canonicalization only when cleanup is required', async () => {
    const router = createRouter()
    const invalidRoute = createRoute({
      account_settings: 'security',
      account_settings_detail: 'connections',
      tab: 'limits',
    })

    await canonicalizePersonalSettingsRoute(router, invalidRoute)
    expect(router.replace).toHaveBeenCalledWith({
      path: '/dashboard',
      query: {
        account_settings: 'security',
        tab: 'limits',
      },
      hash: '#usage',
      replace: true,
    })

    vi.mocked(router.replace).mockClear()
    expect(canonicalizePersonalSettingsRoute(router, createRoute({
      account_settings: 'security',
      tab: 'limits',
    }))).toBeNull()
    expect(router.replace).not.toHaveBeenCalled()
  })

  it('pushes menu opens to the role dashboard and preserves unrelated query and hash', async () => {
    const router = createRouter()
    const sourceRoute = createRoute({
      order_id: 'order-1',
      account_settings: 'billing',
    }, '/purchase', '#redeem')
    window.history.replaceState({ position: 7 }, '')

    await openPersonalSettings(router, sourceRoute, 'admin', 'account')

    expect(router.push).toHaveBeenCalledOnce()
    const target = getFirstNavigationTarget(vi.mocked(router.push))
    expect(target).toMatchObject({
      path: '/admin/dashboard',
      query: {
        order_id: 'order-1',
        account_settings: 'account',
      },
      hash: '#redeem',
      state: {
        __sub2api_personal_settings_owner: {
          version: 1,
          ownerToken: expect.stringMatching(/^personal-settings-session-/),
          documentId: expect.stringMatching(/^personal-settings-document-/),
          chainId: expect.stringMatching(/^personal-settings-chain-/),
          depth: 0,
          originPosition: 7,
        },
      },
    })
    expect(sessionStorage.getItem('__sub2api_personal_settings_owner_token'))
      .toBe(getHistoryOwner(target).ownerToken)
    expect(getPersonalSettingsHostPath('user')).toBe('/dashboard')
    expect(getPersonalSettingsHostPath('admin')).toBe('/admin/dashboard')
  })

  it('replaces depth-zero sections, pushes the first detail, and replaces detail-to-detail', async () => {
    const router = createRouter()
    window.history.replaceState({ position: 11 }, '')
    await openPersonalSettings(
      router,
      createRoute({ filter: 'active' }),
      'user',
      'account',
    )
    const depthZeroTarget = getFirstNavigationTarget(vi.mocked(router.push))
    const depthZeroOwner = getHistoryOwner(depthZeroTarget)
    restoreHistoryTarget(depthZeroTarget, 12)

    const accountRoute = createRoute({
      filter: 'active',
      account_settings: 'account',
    })

    await replacePersonalSettingsSection(router, accountRoute, 'security')
    expect(router.replace).toHaveBeenCalledWith({
      path: '/dashboard',
      query: {
        filter: 'active',
        account_settings: 'security',
      },
      hash: '#usage',
      state: {
        __sub2api_personal_settings_owner: depthZeroOwner,
      },
    })

    vi.mocked(router.push).mockClear()
    await replacePersonalSettingsDetail(router, accountRoute, 'profile')
    expect(router.push).toHaveBeenCalledOnce()
    const depthOneTarget = getFirstNavigationTarget(vi.mocked(router.push))
    expect(depthOneTarget).toMatchObject({
      path: '/dashboard',
      query: {
        filter: 'active',
        account_settings: 'account',
        account_settings_detail: 'profile',
      },
      hash: '#usage',
    })
    const depthOneOwner = getHistoryOwner(depthOneTarget)
    expect(depthOneOwner).toEqual({
      ...depthZeroOwner,
      depth: 1,
    })

    restoreHistoryTarget(depthOneTarget, 13)
    await replacePersonalSettingsDetail(
      router,
      createRoute({
        filter: 'active',
        account_settings: 'account',
        account_settings_detail: 'profile',
      }),
      'connections',
    )
    expect(router.replace).toHaveBeenLastCalledWith({
      path: '/dashboard',
      query: {
        filter: 'active',
        account_settings: 'account',
        account_settings_detail: 'connections',
      },
      hash: '#usage',
      state: {
        __sub2api_personal_settings_owner: depthOneOwner,
      },
    })
    expect(getPersonalSettingsDetailSection('notification-emails')).toBe('notifications')
  })

  it('lets browser Back restore depth zero and closes restored Forward entries safely', async () => {
    const router = createRouter()
    window.history.replaceState({ position: 20 }, '')

    await openPersonalSettings(router, createRoute({ campaign: 'summer' }), 'user')
    const depthZeroTarget = getFirstNavigationTarget(vi.mocked(router.push))
    restoreHistoryTarget(depthZeroTarget, 21)

    vi.mocked(router.push).mockClear()
    await replacePersonalSettingsDetail(
      router,
      createRoute({
        account_settings: 'general',
        campaign: 'summer',
      }),
      'profile',
    )
    const depthOneTarget = getFirstNavigationTarget(vi.mocked(router.push))
    expect(getHistoryOwner(depthOneTarget).depth).toBe(1)

    // Browser Back naturally restores the existing primary entry.
    restoreHistoryTarget(depthZeroTarget, 21)
    closePersonalSettings(router, createRoute({
      account_settings: 'general',
      campaign: 'summer',
    }))
    expect(router.back).toHaveBeenCalledOnce()
    expect(router.go).not.toHaveBeenCalled()
    expect(router.replace).not.toHaveBeenCalled()

    // Forward restores the same validated depth-one chain; closing skips both
    // settings entries and returns directly to the origin.
    vi.mocked(router.back).mockClear()
    restoreHistoryTarget(depthOneTarget, 22)
    closePersonalSettings(router, createRoute({
      account_settings: 'account',
      account_settings_detail: 'profile',
      campaign: 'summer',
    }))
    expect(router.go).toHaveBeenCalledOnce()
    expect(router.go).toHaveBeenCalledWith(-2)
    expect(router.back).not.toHaveBeenCalled()
  })

  it('falls back to query cleanup when a depth-one chain is incomplete or tampered', async () => {
    const router = createRouter()
    window.history.replaceState({ position: 30 }, '')
    await openPersonalSettings(router, createRoute(), 'user', 'account')
    const depthZeroTarget = getFirstNavigationTarget(vi.mocked(router.push))
    restoreHistoryTarget(depthZeroTarget, 31)

    vi.mocked(router.push).mockClear()
    await replacePersonalSettingsDetail(
      router,
      createRoute({ account_settings: 'account' }),
      'profile',
    )
    const depthOneTarget = getFirstNavigationTarget(vi.mocked(router.push))
    const owner = getHistoryOwner(depthOneTarget)
    window.history.replaceState({
      ...(depthOneTarget.state ?? {}),
      position: 32,
      __sub2api_personal_settings_owner: {
        ...owner,
        originPosition: 29,
      },
    }, '')

    closePersonalSettings(router, createRoute({
      source: 'menu',
      account_settings: 'account',
      account_settings_detail: 'profile',
    }))

    expect(router.go).not.toHaveBeenCalled()
    expect(router.replace).toHaveBeenCalledWith({
      path: '/dashboard',
      query: {
        source: 'menu',
      },
      hash: '#usage',
      state: {
        __sub2api_personal_settings_owner: undefined,
      },
    })
  })

  it('closes direct links, legacy redirects, and OAuth returns with replace', () => {
    const router = createRouter()
    const deepLink = createRoute({
      account_settings: 'account',
      account_settings_detail: 'connections',
      oauth: 'complete',
    })

    window.history.replaceState({
      __sub2api_personal_settings_owner: 'stale-token-from-before-refresh',
    }, '')
    closePersonalSettings(router, deepLink)

    expect(router.back).not.toHaveBeenCalled()
    expect(router.replace).toHaveBeenCalledWith({
      path: '/dashboard',
      query: {
        oauth: 'complete',
      },
      hash: '#usage',
      state: {
        __sub2api_personal_settings_owner: undefined,
      },
    })
  })

  it('recognizes the session token after refresh but refuses to reuse the stale live chain', async () => {
    const router = createRouter()
    window.history.replaceState({ position: 40 }, '')
    await openPersonalSettings(router, createRoute(), 'user', 'account')
    const openedTarget = getFirstNavigationTarget(vi.mocked(router.push))
    const openedOwner = getHistoryOwner(openedTarget)
    restoreHistoryTarget(openedTarget, 41)

    vi.resetModules()
    const refreshedModule = await import('../personalSettingsRoute')
    const refreshedRouter = createRouter()
    refreshedModule.closePersonalSettings(
      refreshedRouter,
      createRoute({ account_settings: 'account' }),
    )

    expect(refreshedRouter.back).not.toHaveBeenCalled()
    expect(refreshedRouter.go).not.toHaveBeenCalled()
    expect(refreshedRouter.replace).toHaveBeenCalledWith({
      path: '/dashboard',
      query: {},
      hash: '#usage',
      state: {
        __sub2api_personal_settings_owner: undefined,
      },
    })

    window.history.replaceState({ position: 50 }, '')
    await refreshedModule.openPersonalSettings(
      refreshedRouter,
      createRoute(),
      'user',
    )
    const refreshedTarget = getFirstNavigationTarget(vi.mocked(refreshedRouter.push))
    expect(getHistoryOwner(refreshedTarget).ownerToken).toBe(openedOwner.ownerToken)
    expect(getHistoryOwner(refreshedTarget).documentId).not.toBe(openedOwner.documentId)
  })

  it('clears valid settings query when a higher-priority activation gate blocks it', () => {
    expect(resolvePersonalSettingsCanonicalization(
      createRoute({
        campaign: 'summer',
        account_settings: 'security',
        account_settings_detail: 'totp',
      }, '/admin/dashboard', '#compliance'),
      { activationBlocked: true },
    )).toEqual({
      path: '/admin/dashboard',
      query: {
        campaign: 'summer',
      },
      hash: '#compliance',
      state: {
        __sub2api_personal_settings_owner: undefined,
      },
      replace: true,
    })
  })

  it('builds role-aware legacy targets and keeps OAuth state', () => {
    expect(resolveLegacyPersonalSettingsRedirect(
      createRoute({
        provider: 'wechat',
        account_settings: 'billing',
      }),
      'admin',
      'account',
      'connections',
    )).toEqual({
      path: '/admin/dashboard',
      query: {
        provider: 'wechat',
        account_settings: 'account',
        account_settings_detail: 'connections',
      },
      hash: '#usage',
    })
  })
})

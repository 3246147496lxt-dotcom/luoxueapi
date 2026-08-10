import type { Router } from 'vue-router'
import { getSetupStatus } from '@/api/setup'
import { useNavigationLoadingState } from '@/composables/useNavigationLoading'
import { useRoutePrefetch } from '@/composables/useRoutePrefetch'
import {
  parsePersonalSettingsRoute,
  resolvePersonalSettingsCanonicalization,
} from '@/navigation/personalSettingsRoute'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import type { CustomMenuItem } from '@/types'
import {
  isFreshSubscriptionPlanBridge,
  resolveLegacySubscriptionPricingRedirect,
} from './purchasePricingRedirect'
import { resolveCompletedSetupRedirectPath } from './setupRedirect'
import { resolveRouteDocumentTitle } from './title'

export type AppRouterEntry = 'user' | 'admin'
export type FullPageNavigator = (fullPath: string) => void

export interface AppRouterGuardOptions {
  entry: AppRouterEntry
  fullPageNavigator?: FullPageNavigator
}

const BACKEND_MODE_ALLOWED_PATHS = [
  '/login',
  '/key-usage',
  '/setup',
  '/payment/result',
  '/payment/airwallex',
  '/legal',
]
const BACKEND_MODE_CALLBACK_PATHS = [
  '/auth/callback',
  '/auth/linuxdo/callback',
  '/auth/dingtalk/callback',
  '/auth/dingtalk/email-completion',
  '/auth/oidc/callback',
  '/auth/wechat/callback',
  '/auth/wechat/payment/callback',
]
const BACKEND_MODE_PENDING_AUTH_PATHS = ['/register', '/email-verify']

export function isAdminPath(path: string): boolean {
  return path === '/admin' || path.startsWith('/admin/')
}

export function shouldBridgeToOtherEntry(entry: AppRouterEntry, path: string): boolean {
  return entry === 'user' ? isAdminPath(path) : !isAdminPath(path)
}

export function navigateToFullPage(fullPath: string): void {
  if (typeof window === 'undefined') return
  const destination = new URL(fullPath, window.location.href)
  window.location.assign(destination.href)
}

export function isBackendModePublicRouteAllowed(
  path: string,
  hasPendingAuthSession: boolean,
): boolean {
  if (
    BACKEND_MODE_ALLOWED_PATHS.some(
      (allowedPath) => path === allowedPath || path.startsWith(allowedPath),
    )
  ) {
    return true
  }

  if (BACKEND_MODE_CALLBACK_PATHS.some((callbackPath) => path === callbackPath)) {
    return true
  }

  return hasPendingAuthSession
    && BACKEND_MODE_PENDING_AUTH_PATHS.some((allowedPath) => path === allowedPath)
}

/** Install the shared authorization, feature-access, loading, and prefetch guards. */
export function installAppRouterGuards(
  router: Router,
  options: AppRouterGuardOptions,
): void {
  let authInitialized = false
  let routePrefetch: ReturnType<typeof useRoutePrefetch> | null = null
  const navigationLoading = useNavigationLoadingState()
  const fullPageNavigator = options.fullPageNavigator ?? navigateToFullPage

  router.beforeEach(async (to, _from, next) => {
    navigationLoading.startNavigation()

    // A route owned by the other application must be resolved by the server so
    // the correct entry bundle boots. Run this before local canonicalization or
    // auth redirects; the destination app owns those decisions.
    if (shouldBridgeToOtherEntry(options.entry, to.path)) {
      navigationLoading.endNavigation()
      try {
        fullPageNavigator(to.fullPath)
      } catch (error) {
        console.error('Failed to navigate to the other application entry', error)
      }
      next(false)
      return
    }

    const personalSettingsCanonicalization = resolvePersonalSettingsCanonicalization(to)
    if (personalSettingsCanonicalization) {
      next(personalSettingsCanonicalization)
      return
    }

    const purchasePricingCanonicalization = resolveLegacySubscriptionPricingRedirect(to)
    if (purchasePricingCanonicalization) {
      next(purchasePricingCanonicalization)
      return
    }

    const authStore = useAuthStore()
    if (!authInitialized) {
      authStore.checkAuth()
      authInitialized = true
    }

    const appStore = useAppStore()
    let adminCustomMenuItems: CustomMenuItem[] = []
    if (options.entry === 'admin' && authStore.isAdmin) {
      const { useAdminSettingsStore } = await import('@/stores/adminSettings')
      adminCustomMenuItems = useAdminSettingsStore().customMenuItems
    }
    const customMenuItems = [
      ...(appStore.cachedPublicSettings?.custom_menu_items ?? []),
      ...adminCustomMenuItems,
    ]
    document.title = resolveRouteDocumentTitle(to, appStore.siteName, customMenuItems)

    const requiresAuth = to.meta.requiresAuth !== false
    const requiresAdmin = to.meta.requiresAdmin === true

    if (to.path === '/setup') {
      try {
        const status = await getSetupStatus()
        if (!status.needs_setup) {
          next(resolveCompletedSetupRedirectPath(authStore.isAuthenticated, authStore.isAdmin))
          return
        }
      } catch {
        // If setup status cannot be determined, keep the setup page reachable.
      }
    }

    if (!requiresAuth) {
      if (authStore.isAuthenticated && (to.path === '/login' || to.path === '/register')) {
        if (appStore.backendModeEnabled && !authStore.isAdmin) {
          next()
          return
        }
        next(authStore.isAdmin ? '/admin/dashboard' : '/dashboard')
        return
      }

      if (to.meta.requiresPublicModelCatalog) {
        if (!appStore.publicSettingsLoaded) {
          try {
            await appStore.fetchPublicSettings()
          } catch (error) {
            console.warn('Failed to load public model catalog setting', error)
          }
        }
        if (
          appStore.cachedPublicSettings?.public_model_catalog_enabled !== true
          || appStore.backendModeEnabled
        ) {
          next('/home')
          return
        }
      }

      if (to.meta.requiresSkillMarketplace) {
        const refreshedSettings = await appStore.fetchPublicSettings(true)
        if (
          refreshedSettings?.skill_marketplace_enabled !== true
          || appStore.backendModeEnabled
        ) {
          next('/home')
          return
        }
      }

      if (appStore.backendModeEnabled && !authStore.isAuthenticated) {
        const isAllowed = isBackendModePublicRouteAllowed(
          to.path,
          authStore.hasPendingAuthSession,
        )
        if (!isAllowed) {
          next('/login')
          return
        }
      }
      next()
      return
    }

    if (!authStore.isAuthenticated) {
      next({
        path: '/login',
        query: { redirect: to.fullPath },
      })
      return
    }

    if (requiresAdmin && !authStore.isAdmin) {
      next('/dashboard')
      return
    }

    const personalSettingsState = parsePersonalSettingsRoute(to)
    if (authStore.isAdmin && (requiresAdmin || personalSettingsState.isOpen)) {
      const { useAdminComplianceStore } = await import('@/stores/adminCompliance')
      const adminComplianceStore = useAdminComplianceStore()
      if (!adminComplianceStore.initialized) {
        try {
          await adminComplianceStore.fetchStatus()
        } catch (error) {
          const err = error as {
            status?: number
            code?: string
            metadata?: Record<string, string>
          }
          if (err.status === 423 && err.code === 'ADMIN_COMPLIANCE_ACK_REQUIRED') {
            adminComplianceStore.requireAcknowledgement(err.metadata)
          }
        }
      }

      if (adminComplianceStore.shouldShow) {
        const complianceCanonicalization = resolvePersonalSettingsCanonicalization(to, {
          activationBlocked: true,
        })
        if (complianceCanonicalization) {
          next(complianceCanonicalization)
          return
        }
      }
    }

    const freshSubscriptionPlanBridge = isFreshSubscriptionPlanBridge(to)

    if (
      (
        to.meta.requiresPayment
        || freshSubscriptionPlanBridge
        || to.meta.requiresRiskControl
      )
      && !appStore.publicSettingsLoaded
    ) {
      try {
        await appStore.fetchPublicSettings()
      } catch (error) {
        console.warn('Failed to load public settings in route guard', error)
      }
    }

    if (
      (to.meta.requiresPayment || freshSubscriptionPlanBridge)
      && appStore.publicSettingsLoaded
      && appStore.cachedPublicSettings?.payment_enabled === false
    ) {
      next(authStore.isAdmin ? '/admin/dashboard' : '/dashboard')
      return
    }

    if (
      to.meta.requiresRiskControl
      && appStore.publicSettingsLoaded
      && appStore.cachedPublicSettings?.risk_control_enabled === false
    ) {
      next(authStore.isAdmin ? '/admin/settings' : '/dashboard')
      return
    }

    if (authStore.isSimpleMode) {
      const restrictedPaths = [
        '/admin/groups',
        '/admin/subscriptions',
        '/admin/redeem',
        '/subscriptions',
        '/pricing',
        '/redeem',
      ]

      if (
        freshSubscriptionPlanBridge
        || restrictedPaths.some((path) => to.path.startsWith(path))
      ) {
        next(authStore.isAdmin ? '/admin/dashboard' : '/dashboard')
        return
      }
    }

    if (appStore.backendModeEnabled) {
      if (authStore.isAuthenticated && authStore.isAdmin) {
        next()
        return
      }
      const isAllowed = isBackendModePublicRouteAllowed(
        to.path,
        authStore.hasPendingAuthSession,
      )
      if (!isAllowed) {
        next('/login')
        return
      }
    }

    next()
  })

  router.afterEach((to, _from, failure) => {
    navigationLoading.endNavigation()
    if (failure) return
    if (!routePrefetch) {
      routePrefetch = useRoutePrefetch(router)
    }
    routePrefetch.triggerPrefetch(to)
  })
}

/** Recover once from stale lazy chunks after a deployment. */
export function installRouterErrorHandler(router: Router): void {
  router.onError((error) => {
    console.error('Router error:', error)

    const isChunkLoadError =
      error.message?.includes('Failed to fetch dynamically imported module')
      || error.message?.includes('Loading chunk')
      || error.message?.includes('Loading CSS chunk')
      || error.name === 'ChunkLoadError'

    if (!isChunkLoadError) return

    const reloadKey = 'chunk_reload_attempted'
    const lastReload = sessionStorage.getItem(reloadKey)
    const now = Date.now()

    if (!lastReload || now - parseInt(lastReload) > 10000) {
      sessionStorage.setItem(reloadKey, now.toString())
      console.warn('Chunk load error detected, reloading page to fetch latest version...')
      window.location.reload()
      return
    }

    console.error('Chunk load error persists after reload. Please clear browser cache.')
  })
}

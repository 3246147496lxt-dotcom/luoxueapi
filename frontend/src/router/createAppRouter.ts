import {
  createRouter,
  createWebHistory,
  type RouteRecordRaw,
  type Router,
  type RouterHistory,
} from 'vue-router'
import {
  installAppRouterGuards,
  installRouterErrorHandler,
  type AppRouterEntry,
  type FullPageNavigator,
} from './guards'

export interface CreateAppRouterOptions {
  entry: AppRouterEntry
  routes: readonly RouteRecordRaw[]
  history?: RouterHistory
  fullPageNavigator?: FullPageNavigator
}

const FullPageBridgeView = { render: () => null }

export const userToAdminBridgeRoute: RouteRecordRaw = {
  path: '/admin/:pathMatch(.*)*',
  name: 'UserToAdminFullPageBridge',
  component: FullPageBridgeView,
  meta: {
    requiresAuth: true,
    requiresAdmin: true,
    title: 'Admin',
  },
}

export const adminToUserBridgeRoute: RouteRecordRaw = {
  path: '/:pathMatch(.*)*',
  name: 'AdminToUserFullPageBridge',
  component: FullPageBridgeView,
  meta: {
    requiresAuth: false,
    title: 'Redirecting',
  },
}

export const adminNotFoundRoute: RouteRecordRaw = {
  path: '/admin/:pathMatch(.*)*',
  name: 'AdminNotFound',
  component: () => import('@/views/NotFoundView.vue'),
  meta: {
    requiresAuth: true,
    requiresAdmin: true,
    title: '404 Not Found',
  },
}

export const notFoundRoute: RouteRecordRaw = {
  path: '/:pathMatch(.*)*',
  name: 'NotFound',
  component: () => import('@/views/NotFoundView.vue'),
  meta: {
    title: '404 Not Found',
  },
}

export function getAppRoutes(
  entry: AppRouterEntry,
  ownedRoutes: readonly RouteRecordRaw[],
): RouteRecordRaw[] {
  if (entry === 'admin') {
    return [...ownedRoutes, adminNotFoundRoute, adminToUserBridgeRoute]
  }

  return [
    ...ownedRoutes,
    userToAdminBridgeRoute,
    notFoundRoute,
  ]
}

export function createAppRouter(options: CreateAppRouterOptions): Router {
  const router = createRouter({
    history: options.history ?? createWebHistory(import.meta.env.BASE_URL),
    routes: getAppRoutes(options.entry, options.routes),
    scrollBehavior(_to, _from, savedPosition) {
      if (savedPosition) return savedPosition
      return { top: 0 }
    },
  })

  installAppRouterGuards(router, {
    entry: options.entry,
    fullPageNavigator: options.fullPageNavigator,
  })
  installRouterErrorHandler(router)
  return router
}

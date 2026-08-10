import { createAppRouter } from './createAppRouter'
import { publicRoutes } from './routes/public'
import { userRoutes } from './routes/user'

const router = createAppRouter({
  entry: 'user',
  routes: [...publicRoutes, ...userRoutes],
})

export { createAppRouter, getAppRoutes } from './createAppRouter'
export { publicRoutes } from './routes/public'
export { userRoutes } from './routes/user'
export type { AppRouterEntry, FullPageNavigator } from './guards'

export default router

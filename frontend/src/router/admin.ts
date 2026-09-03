import { createAppRouter } from './createAppRouter'
import { adminRoutes } from './routes/admin'

const router = createAppRouter({
  entry: 'admin',
  routes: adminRoutes,
})

export { adminRoutes } from './routes/admin'
export default router

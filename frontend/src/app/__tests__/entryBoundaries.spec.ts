import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const readSource = (relativePath: string) => (
  readFileSync(resolve(process.cwd(), relativePath), 'utf8')
)

describe('user and admin application entry boundaries', () => {
  it('keeps the user root free of static admin runtime dependencies', () => {
    const userApp = readSource('src/App.vue')
    const appRoot = readSource('src/app/AppRoot.vue')
    const userSubscriptionRuntime = readSource('src/app/UserSubscriptionRuntime.vue')
    const userMain = readSource('src/main.ts')

    expect(userApp).not.toContain('@/components/admin')
    expect(userApp).not.toContain('@/stores/admin')
    expect(appRoot).not.toContain('@/components/admin')
    expect(appRoot).not.toContain('@/stores/admin')
    expect(appRoot).not.toContain('@/stores/subscriptions')
    expect(userApp).toContain('UserSubscriptionRuntime')
    expect(userSubscriptionRuntime).toContain("from '@/stores/userProfile'")
    expect(userSubscriptionRuntime).not.toContain('startPolling')
    expect(userMain).toContain("import router from './router'")
    expect(userMain).toContain("bootstrapApp(App, router, 'user')")
  })

  it('boots the admin root through the admin-only router entry', () => {
    const adminApp = readSource('src/AdminApp.vue')
    const adminMain = readSource('src/main-admin.ts')

    expect(adminApp).toContain('AdminComplianceRuntime')
    expect(adminApp).toContain('UserSubscriptionRuntime')
    expect(adminApp).toContain('useAdminSettingsStore')
    expect(adminMain).toContain("import router from './router/admin'")
    expect(adminMain).toContain("bootstrapApp(AdminApp, router, 'admin')")
  })

  it('keeps route manifests out of the opposite static entry graph', () => {
    const userRouter = readSource('src/router/index.ts')
    const adminRouter = readSource('src/router/admin.ts')
    const routerFactory = readSource('src/router/createAppRouter.ts')

    expect(userRouter).toContain("from './routes/public'")
    expect(userRouter).toContain("from './routes/user'")
    expect(userRouter).not.toContain("from './routes/admin'")
    expect(adminRouter).toContain("from './routes/admin'")
    expect(adminRouter).not.toContain("from './routes/public'")
    expect(adminRouter).not.toContain("from './routes/user'")
    expect(routerFactory).not.toContain("from './routes/")
  })

  it('builds distinct same-origin HTML documents', () => {
    const userHTML = readSource('index.html')
    const adminHTML = readSource('admin/index.html')
    const viteConfig = readSource('vite.config.ts')

    expect(userHTML).toContain('<meta name="app-entry" content="user" />')
    expect(userHTML).toContain('src="/src/main.ts"')
    expect(adminHTML).toContain('<meta name="app-entry" content="admin" />')
    expect(adminHTML).toContain('<meta name="robots" content="noindex, nofollow" />')
    expect(adminHTML).toContain('src="/src/main-admin.ts"')
    expect(viteConfig).toContain("user: resolve(__dirname, 'index.html')")
    expect(viteConfig).toContain("admin: resolve(__dirname, 'admin/index.html')")
    expect(viteConfig).toContain('configurePreviewServer')
    expect(viteConfig).toContain("request.headers.accept?.includes('text/html')")
  })
})

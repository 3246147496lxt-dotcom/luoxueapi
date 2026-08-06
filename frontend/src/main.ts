import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import i18n, { initI18n } from './i18n'
import { initializeThemePreference } from '@/composables/useThemePreference'
import { useAppStore } from '@/stores/app'
import './style.css'
import './styles/luoxue-clay-tokens.css'
import './styles/luoxue-clay-components.css'

async function bootstrap() {
  // Apply theme class globally before app mount to keep all routes consistent.
  initializeThemePreference()

  const app = createApp(App)
  const pinia = createPinia()
  app.use(pinia)

  // Initialize settings from injected config BEFORE mounting (prevents flash)
  // This must happen after pinia is installed but before router and i18n
  const appStore = useAppStore()
  appStore.initFromInjectedConfig()

  // The injected snapshot prevents first-paint flicker, but it may have been
  // rendered by another instance before a feature flag changed. Revalidate it
  // in the background; route guards reuse this in-flight request.
  const publicSettingsRefresh = appStore.fetchPublicSettings(true)

  // Set document title immediately after config is loaded
  if (appStore.siteName && appStore.siteName !== '落雪API') {
    document.title = `${appStore.siteName} - AI API Gateway`
  }

  await initI18n()

  app.use(router)
  app.use(i18n)

  // 等待路由器完成初始导航后再挂载，避免竞态条件导致的空白渲染
  await router.isReady()
  app.mount('#app')

  void publicSettingsRefresh
}

bootstrap()

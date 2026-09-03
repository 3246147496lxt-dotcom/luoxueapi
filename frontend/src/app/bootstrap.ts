import { createApp, type Component } from 'vue'
import { createPinia } from 'pinia'
import type { Router } from 'vue-router'
import i18n, { initI18n, type I18nSurface } from '@/i18n'
import { initializeThemePreference } from '@/composables/useThemePreference'
import { useAppStore } from '@/stores/app'
import '@/style.css'
import '@/styles/luoxue-clay-tokens.css'
import '@/styles/luoxue-clay-components.css'

export async function bootstrapApp(
  rootComponent: Component,
  router: Router,
  surface: I18nSurface,
) {
  initializeThemePreference()

  const app = createApp(rootComponent)
  const pinia = createPinia()
  app.use(pinia)

  const appStore = useAppStore()
  appStore.initFromInjectedConfig()
  const publicSettingsRefresh = appStore.fetchPublicSettings(true)

  if (appStore.siteName && appStore.siteName !== '落雪API') {
    document.title = `${appStore.siteName} - AI API Gateway`
  }

  await initI18n({ surface, router })

  app.use(router)
  app.use(i18n)

  await router.isReady()
  app.mount('#app')

  void publicSettingsRefresh
}

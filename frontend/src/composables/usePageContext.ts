import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { useAdminSettingsStore } from '@/stores/adminSettings'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

export function usePageContext() {
  const route = useRoute()
  const { t } = useI18n()
  const appStore = useAppStore()
  const authStore = useAuthStore()
  const adminSettingsStore = useAdminSettingsStore()

  const pageTitle = computed(() => {
    if (route.name === 'CustomPage') {
      const id = typeof route.params.id === 'string' ? route.params.id : ''
      const publicItems = appStore.cachedPublicSettings?.custom_menu_items ?? []
      const adminItems = authStore.isAdmin ? adminSettingsStore.customMenuItems : []
      const menuItem = [...publicItems, ...adminItems].find((item) => item.id === id)

      if (menuItem?.label?.trim()) {
        return menuItem.label.trim()
      }
    }

    const titleKey = typeof route.meta.titleKey === 'string' ? route.meta.titleKey : ''
    if (titleKey) {
      const translated = t(titleKey)
      if (translated !== titleKey) {
        return translated
      }
    }

    if (typeof route.meta.title === 'string' && route.meta.title.trim()) {
      return route.meta.title.trim()
    }

    return appStore.siteName
  })

  return {
    pageTitle,
  }
}

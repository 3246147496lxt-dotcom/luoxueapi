<script setup lang="ts">
import { RouterView, useRoute, useRouter } from 'vue-router'
import { onBeforeUnmount, onMounted, watch } from 'vue'
import Toast from '@/components/common/Toast.vue'
import NavigationProgress from '@/components/common/NavigationProgress.vue'
import AnnouncementPopup from '@/components/common/AnnouncementPopup.vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useAnnouncementStore } from '@/stores/announcements'
import { getSetupStatus } from '@/api/setup'
import { resolveRouteDocumentTitle } from '@/router/title'
import { syncFavicon } from '@/utils/favicon'
import type { CustomMenuItem } from '@/types'

const props = withDefaults(defineProps<{
  customMenuItems?: CustomMenuItem[]
}>(), {
  customMenuItems: () => [],
})

const router = useRouter()
const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()
const announcementStore = useAnnouncementStore()

function updateDocumentTitle() {
  const customMenuItems = [
    ...(appStore.cachedPublicSettings?.custom_menu_items ?? []),
    ...props.customMenuItems,
  ]
  document.title = resolveRouteDocumentTitle(route, appStore.siteName, customMenuItems)
}

watch(
  () => appStore.siteLogo,
  (newLogo) => {
    syncFavicon(newLogo)
  },
  { immediate: true },
)

watch(
  [
    () => route.fullPath,
    () => route.meta.title,
    () => route.meta.titleKey,
    () => appStore.siteName,
    () => appStore.cachedPublicSettings?.custom_menu_items,
    () => props.customMenuItems,
  ],
  updateDocumentTitle,
  { deep: true },
)

function onVisibilityChange() {
  if (document.visibilityState === 'visible' && authStore.isAuthenticated) {
    announcementStore.fetchAnnouncements()
  }
}

watch(
  () => authStore.isAuthenticated,
  (isAuthenticated, oldValue) => {
    if (isAuthenticated) {
      if (oldValue === false) {
        setTimeout(() => announcementStore.fetchAnnouncements(true), 3000)
      } else {
        announcementStore.fetchAnnouncements()
      }

      document.addEventListener('visibilitychange', onVisibilityChange)
    } else {
      announcementStore.reset()
      document.removeEventListener('visibilitychange', onVisibilityChange)
    }
  },
  { immediate: true },
)

const removeAfterEach = router.afterEach(() => {
  if (authStore.isAuthenticated) {
    announcementStore.fetchAnnouncements()
  }
})

onBeforeUnmount(() => {
  removeAfterEach()
  document.removeEventListener('visibilitychange', onVisibilityChange)
})

onMounted(async () => {
  try {
    const status = await getSetupStatus()
    if (status.needs_setup && route.path !== '/setup') {
      await router.replace('/setup')
      return
    }
  } catch {
    // If setup status cannot be determined, keep the current route available.
  }

  await appStore.fetchPublicSettings()
  updateDocumentTitle()
})
</script>

<template>
  <NavigationProgress />
  <RouterView />
  <Toast />
  <AnnouncementPopup />
</template>

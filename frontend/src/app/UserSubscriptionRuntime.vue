<script setup lang="ts">
import { watch } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useUserProfileStore } from '@/stores/userProfile'

const authStore = useAuthStore()
const userProfileStore = useUserProfileStore()

watch(
  [
    () => authStore.user?.id ?? null,
    () => authStore.isAuthenticated,
  ],
  ([userId, isAuthenticated]) => {
    if (isAuthenticated && userId !== null) {
      userProfileStore.initialize().catch((error) => {
        console.error('Failed to initialize the workspace user profile:', error)
      })
      return
    }

    userProfileStore.clear()
  },
  { immediate: true, flush: 'sync' },
)
</script>

<template>
  <span class="hidden" aria-hidden="true" />
</template>

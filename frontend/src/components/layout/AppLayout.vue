<template>
  <div class="min-h-screen bg-[#eef1ef] pt-[81px] dark:bg-[#0e1211]">
    <!-- Global Header -->
    <AppHeader />

    <!-- Sidebar -->
    <AppSidebar />

    <!-- Main Content Area -->
    <div
      data-testid="app-main-shell"
      class="relative min-h-[calc(100vh-81px)] bg-[#eef1ef] transition-[margin] duration-300 ease-out motion-reduce:transition-none dark:bg-[#0e1211]"
      :class="[
        sidebarCollapsed
          ? 'lg:ml-[68px]'
          : 'lg:ml-[184px] min-[1025px]:ml-[196px] min-[1281px]:ml-[208px]'
      ]"
    >
      <!-- Main Content -->
      <main class="px-[13px] pb-[13px] pt-0 sm:px-5 sm:pb-5 md:px-6 md:pb-6 lg:px-8 lg:pb-8">
        <div class="mx-auto w-full max-w-[1600px]">
          <slot />
        </div>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@/styles/onboarding.css'
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingTour } from '@/composables/useOnboardingTour'
import { useOnboardingStore } from '@/stores/onboarding'
import AppSidebar from './AppSidebar.vue'
import AppHeader from './AppHeader.vue'

const appStore = useAppStore()
const authStore = useAuthStore()
const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const isAdmin = computed(() => authStore.user?.role === 'admin')

const { replayTour } = useOnboardingTour({
  storageKey: isAdmin.value ? 'admin_guide' : 'user_guide',
  autoStart: true
})

const onboardingStore = useOnboardingStore()

onMounted(() => {
  onboardingStore.setReplayCallback(replayTour)
})

defineExpose({ replayTour })
</script>

<template>
  <div
    class="app-layout app-layout--snow-shell min-h-screen pt-[81px]"
    :data-sidebar-collapsed="sidebarCollapsed"
  >
    <!-- Global Header -->
    <AppHeader />

    <!-- Sidebar -->
    <AppSidebar />

    <!-- Main Content Area -->
    <div
      data-testid="app-main-shell"
      class="app-main-shell relative min-h-[calc(100vh-81px)] transition-[margin] duration-300 ease-out motion-reduce:transition-none"
      :class="[
        sidebarCollapsed
          ? 'lg:ml-[68px]'
          : 'lg:ml-[184px] min-[1025px]:ml-[196px] min-[1281px]:ml-[208px]',
        variant === 'home-clay' && 'app-layout--home-clay'
      ]"
    >
      <!-- Main Content -->
      <main class="app-main-content px-[13px] pb-[13px] pt-0 sm:px-5 sm:pb-5 md:px-6 md:pb-6 lg:px-8 lg:pb-8">
        <div class="mx-auto w-full max-w-[1600px]">
          <slot />
        </div>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@/styles/onboarding.css'
import './AdminClayTheme.css'
import { computed, onBeforeUnmount, onMounted, watch } from 'vue'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingTour } from '@/composables/useOnboardingTour'
import { useOnboardingStore } from '@/stores/onboarding'
import AppSidebar from './AppSidebar.vue'
import AppHeader from './AppHeader.vue'

export type AppLayoutVariant = 'default' | 'home-clay'

const props = withDefaults(defineProps<{
  variant?: AppLayoutVariant
}>(), {
  variant: 'default'
})

const HOME_CLAY_PORTAL_CLASS = 'admin-home-clay-portals'
let ownsHomeClayPortalClass = false

const syncHomeClayPortalClass = (enabled: boolean) => {
  if (typeof document === 'undefined' || enabled === ownsHomeClayPortalClass) return

  document.body.classList.toggle(HOME_CLAY_PORTAL_CLASS, enabled)
  ownsHomeClayPortalClass = enabled
}

const appStore = useAppStore()
const authStore = useAuthStore()
const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const isAdmin = computed(() => authStore.user?.role === 'admin')

const { replayTour } = useOnboardingTour({
  storageKey: isAdmin.value ? 'admin_guide' : 'user_guide',
  autoStart: true
})

const onboardingStore = useOnboardingStore()

watch(
  () => props.variant === 'home-clay',
  syncHomeClayPortalClass,
  { immediate: true }
)

onMounted(() => {
  onboardingStore.setReplayCallback(replayTour)
})

onBeforeUnmount(() => {
  syncHomeClayPortalClass(false)
})

defineExpose({ replayTour })
</script>

<style scoped>
.app-layout--snow-shell {
  min-height: 100vh;
  padding-top: 81px !important;
  color: var(--lx-clay-text);
  background: var(--lx-clay-canvas) !important;
  font-family: var(--lx-clay-font-ui);
}

.app-layout--snow-shell .app-main-shell {
  min-height: calc(100vh - 81px);
  background: var(--lx-clay-canvas) !important;
}

.app-layout--snow-shell .app-main-content {
  padding: 32px 28px 48px;
}

@media (max-width: 767px) {
  .app-layout--snow-shell .app-main-shell {
    min-height: calc(100vh - 81px);
  }

  .app-layout--snow-shell .app-main-content {
    padding: 22px 16px 40px;
  }
}
</style>

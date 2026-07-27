<template>
  <div
    class="app-layout app-layout--snow-shell min-h-screen"
    :class="{
      'app-layout--flat-workspace-shell': isFlatWorkspaceShell,
      'app-layout--admin-shell': isAdminShell,
      'app-layout--purchase': variant === 'purchase'
    }"
    :data-sidebar-collapsed="sidebarCollapsed"
  >
    <!-- Mobile navigation context -->
    <AppMobileHeader />

    <!-- Sidebar -->
    <AppSidebar />

    <!-- Main Content Area -->
    <div
      data-testid="app-main-shell"
      class="app-main-shell relative transition-[margin] duration-300 ease-out motion-reduce:transition-none"
      :class="[
        sidebarCollapsed
          ? 'lg:ml-[68px]'
          : 'lg:ml-[260px]',
        isAdminShell && 'app-layout--home-clay',
        variant === 'chat' && 'app-layout--chat'
      ]"
    >
      <!-- Main Content -->
      <main class="app-main-content px-[13px] pb-[13px] pt-0 sm:px-5 sm:pb-5 md:px-6 md:pb-6 lg:px-8 lg:pb-8">
        <div class="mx-auto w-full max-w-[1600px]">
          <slot />
        </div>
      </main>
    </div>

    <PersonalSettingsDialog v-if="settingsDialogRequested" />
  </div>
</template>

<script setup lang="ts">
import '@/styles/onboarding.css'
import './AdminClayTheme.css'
import {
  computed,
  defineAsyncComponent,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
} from 'vue'
import { useRoute } from 'vue-router'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingTour } from '@/composables/useOnboardingTour'
import { useOnboardingStore } from '@/stores/onboarding'
import AppSidebar from './AppSidebar.vue'
import AppMobileHeader from './AppMobileHeader.vue'

const PersonalSettingsDialog = defineAsyncComponent(
  () => import('@/components/settings/PersonalSettingsDialog.vue'),
)

export type AppLayoutVariant = 'default' | 'home-clay' | 'chat' | 'purchase'

const props = withDefaults(defineProps<{
  variant?: AppLayoutVariant
}>(), {
  variant: 'default'
})

const route = useRoute()
const settingsDialogRequested = ref(
  Object.prototype.hasOwnProperty.call(route.query, 'account_settings'),
)
const isAdminShell = computed(
  () => props.variant === 'home-clay' || route.meta.requiresAdmin === true
)
const isFlatWorkspaceShell = computed(
  () => isAdminShell.value || route.meta.requiresAuth === true
)

const HOME_CLAY_PORTAL_CLASS = 'admin-home-clay-portals'
const FLAT_WORKSPACE_BODY_CLASS = 'app-flat-workspace-active'
let ownsHomeClayPortalClass = false
let ownsFlatWorkspaceBodyClass = false

const syncHomeClayPortalClass = (enabled: boolean) => {
  if (typeof document === 'undefined' || enabled === ownsHomeClayPortalClass) return

  document.body.classList.toggle(HOME_CLAY_PORTAL_CLASS, enabled)
  ownsHomeClayPortalClass = enabled
}

const syncFlatWorkspaceBodyClass = (enabled: boolean) => {
  if (typeof document === 'undefined' || enabled === ownsFlatWorkspaceBodyClass) return

  document.body.classList.toggle(FLAT_WORKSPACE_BODY_CLASS, enabled)
  ownsFlatWorkspaceBodyClass = enabled
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
  isAdminShell,
  syncHomeClayPortalClass,
  { immediate: true }
)

watch(
  isFlatWorkspaceShell,
  syncFlatWorkspaceBodyClass,
  { immediate: true }
)

watch(
  () => route.query.account_settings,
  (queryValue) => {
    if (queryValue !== undefined) {
      // Keep the host mounted after its first request so the dialog can finish
      // its close transition, restore focus, and release modal resources.
      settingsDialogRequested.value = true
    }
  },
)

onMounted(() => {
  onboardingStore.setReplayCallback(replayTour)
})

onBeforeUnmount(() => {
  syncHomeClayPortalClass(false)
  syncFlatWorkspaceBodyClass(false)
})

defineExpose({ replayTour })
</script>

<style scoped>
.app-layout--snow-shell {
  min-height: 100vh;
  padding-top: var(--app-shell-top-offset) !important;
  color: var(--lx-clay-text);
  background: var(--app-shell-canvas, var(--lx-clay-canvas)) !important;
  font-family: var(--lx-clay-font-ui);
}

.app-layout--snow-shell .app-main-shell {
  min-height: calc(100vh - var(--app-shell-top-offset));
  background: var(--app-shell-canvas, var(--lx-clay-canvas)) !important;
}

.app-layout--snow-shell .app-main-content {
  padding: 32px 28px 48px;
}

:global(html:not(.dark) .app-layout--snow-shell.app-layout--flat-workspace-shell) {
  --app-shell-canvas: #ffffff;
  --app-shell-sidebar-bg: #fcfcfc;
  --app-shell-sidebar-border: #e5e7eb;
  --app-shell-sidebar-shadow: none;
  --app-shell-sidebar-backdrop: none;
  --app-shell-sidebar-decoration: none;
  --app-shell-sidebar-highlight: none;
  --app-shell-sidebar-hover-color: #0d0d0d;
  --app-shell-sidebar-hover-bg: rgb(0 0 0 / 0.05);
  --app-shell-sidebar-hover-shadow: none;
  --app-shell-sidebar-hover-transform: none;
  --app-shell-sidebar-focus: color-mix(in srgb, var(--lx-clay-accent) 44%, transparent);
  --app-shell-sidebar-section-color: #5f6b7a;
  --app-shell-sidebar-active-color: #0d0d0d;
  --app-shell-sidebar-active-icon: #0d0d0d;
  --app-shell-sidebar-active-bg: rgb(0 0 0 / 0.05);
  --app-shell-sidebar-active-marker: none;
}

:global(html:not(.dark) .app-layout--snow-shell.app-layout--purchase) {
  --app-shell-canvas: #f4f1fa;
}

.app-layout--snow-shell.app-layout--purchase .app-main-content {
  padding: 40px;
}

:global(html:not(.dark) body.app-flat-workspace-active) {
  background: #ffffff;
}

@media (max-width: 767px) {
  .app-layout--snow-shell .app-main-shell {
    min-height: calc(100vh - var(--app-shell-top-offset));
  }

  .app-layout--snow-shell .app-main-content {
    padding: 22px 16px 40px;
  }
}

@media (max-width: 1023px) {
  .app-layout--snow-shell.app-layout--purchase .app-main-content {
    padding: 24px;
  }
}

.app-layout--snow-shell .app-main-shell.app-layout--chat {
  height: calc(100dvh - var(--app-shell-top-offset));
  min-height: 0;
  overflow: hidden;
}

.app-layout--snow-shell .app-main-shell.app-layout--chat .app-main-content {
  height: 100%;
  padding: 0 8px 8px;
}

.app-layout--snow-shell .app-main-shell.app-layout--chat .app-main-content > div {
  height: 100%;
}

@media (max-width: 767px) {
  .app-layout--snow-shell .app-main-shell.app-layout--chat .app-main-content {
    padding: 0 5px 5px;
  }
}
</style>

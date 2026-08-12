<template>
  <div
    class="app-layout app-layout--snow-shell min-h-screen"
    :class="{
      'app-layout--flat-workspace-shell': isFlatWorkspaceShell,
      'app-layout--admin-shell': isAdminShell,
      'app-layout--personal-work-shell': isPersonalWorkShell,
      'app-layout--chat-shell': isChatShell,
      'app-layout--purchase': variant === 'purchase',
      'app-layout--workbench-content': contentMode === 'workbench',
      'app-layout--ops-option-b': isOpsOptionB
    }"
    :data-sidebar-collapsed="isChatShell ? undefined : renderedSidebarCollapsed"
    :data-content-mode="contentMode"
    :data-shell-mode="resolvedShellMode"
  >
    <!-- Mobile navigation context -->
    <AppMobileHeader v-if="!isChatShell" />

    <WorkspaceSidebarOverlayTrigger
      v-if="personalWorkspaceNarrow"
      id="workspace-sidebar-overlay-trigger"
      :controls="narrowSidebarPanelId"
      :label="t('nav.openNavigation')"
      :open="narrowSidebarOpen"
      @open="appStore.setWorkspaceNarrowSidebarOpen(true)"
    />

    <!-- Sidebar -->
    <AppSidebar v-if="!isChatShell" :admin-navigation="adminNavigation" />

    <!-- Main Content Area -->
    <div
      data-testid="app-main-shell"
      class="app-main-shell relative transition-[margin] duration-[var(--workspace-sidebar-transition-duration)] ease-[var(--workspace-sidebar-transition-easing)] motion-reduce:transition-none"
      :aria-hidden="workNarrowModalActive ? 'true' : undefined"
      :inert="workNarrowModalActive ? true : undefined"
      :class="[
        !isChatShell && (
          personalWorkspaceNarrow
            ? 'ml-0'
            : renderedSidebarCollapsed
            ? 'ml-[var(--workspace-sidebar-width-collapsed)]'
            : 'ml-[var(--workspace-sidebar-width)]'
        ),
        isAdminShell && 'app-layout--home-clay',
        variant === 'chat' && 'app-layout--chat'
      ]"
    >
      <!-- Main Content -->
      <main
        class="app-main-content px-[13px] pb-[13px] pt-0 sm:px-5 sm:pb-5 md:px-6 md:pb-6 lg:px-8 lg:pb-8"
        :class="{ 'app-main-content--workbench': contentMode === 'workbench' }"
      >
        <div
          class="app-main-inner mx-auto w-full max-w-[1600px]"
          :class="{ 'app-main-inner--workbench': contentMode === 'workbench' }"
        >
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
  shallowRef,
  watch,
} from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useOnboardingTour } from '@/composables/useOnboardingTour'
import { useOnboardingStore } from '@/stores/onboarding'
import AppSidebar from './AppSidebar.vue'
import AppMobileHeader from './AppMobileHeader.vue'
import WorkspaceSidebarOverlayTrigger from './WorkspaceSidebarOverlayTrigger.vue'
import { useWorkspaceResponsiveState } from './workspaceResponsive'
import { loadAdminNavigation } from './sidebar/adminNavigationLoader'
import type { AdminNavigationDefinition } from './sidebar/types'

const PersonalSettingsDialog = defineAsyncComponent(
  () => import('@/components/settings/PersonalSettingsDialog.vue'),
)

export type AppLayoutVariant = 'default' | 'home-clay' | 'chat' | 'purchase'
export type AppLayoutContentMode = 'contained' | 'workbench'
export type AppShellMode = 'chat' | 'work'

const props = withDefaults(defineProps<{
  variant?: AppLayoutVariant
  contentMode?: AppLayoutContentMode
  shellMode?: AppShellMode
}>(), {
  variant: 'default',
  contentMode: 'contained'
})

const route = useRoute()
const { t } = useI18n()
const resolvedShellMode = computed<AppShellMode>(() => (
  props.shellMode
  ?? route.meta.shellMode
  ?? 'work'
))
const settingsDialogRequested = ref(
  Object.prototype.hasOwnProperty.call(route.query, 'account_settings'),
)
const isAdminShell = computed(
  () => props.variant === 'home-clay' || route.meta.requiresAdmin === true
)
const isFlatWorkspaceShell = computed(
  () => isAdminShell.value || route.meta.requiresAuth === true
)
const isChatShell = computed(() => resolvedShellMode.value === 'chat')
const isPersonalWorkShell = computed(() => (
  resolvedShellMode.value === 'work'
  && route.meta.requiresAuth === true
  && !isAdminShell.value
))
const isOpsOptionB = computed(
  () => props.contentMode === 'workbench' && route.path.startsWith('/admin/ops')
)
const adminNavigation = shallowRef<AdminNavigationDefinition | null>(null)
const adminNavigationRequested = computed(
  () => route.meta.requiresAdmin === true || route.path.startsWith('/admin'),
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
const { narrowSidebar } = useWorkspaceResponsiveState()
const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const personalWorkspaceNarrow = computed(() => (
  narrowSidebar.value
  && !isAdminShell.value
  && (isPersonalWorkShell.value || isChatShell.value)
))
const renderedSidebarCollapsed = computed(() => (
  sidebarCollapsed.value && !personalWorkspaceNarrow.value
))
const narrowSidebarOpen = computed(() => appStore.workspaceNarrowSidebarOpen)
const workNarrowModalActive = computed(() => (
  isPersonalWorkShell.value
  && personalWorkspaceNarrow.value
  && narrowSidebarOpen.value
))
const narrowSidebarPanelId = computed(() => (
  isChatShell.value ? 'workspace-chat-sidebar-overlay' : 'app-sidebar'
))

const { replayTour } = useOnboardingTour({
  storageKey: isAdminShell.value ? 'admin_guide' : 'user_guide',
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
  adminNavigationRequested,
  (requested) => {
    if (requested && !adminNavigation.value) {
      void loadAdminNavigation().then((definition) => {
        adminNavigation.value = definition
      })
    }
  },
  { immediate: true },
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

watch(
  () => route.fullPath,
  (fullPath, previousPath) => {
    if (
      previousPath !== undefined
      && fullPath !== previousPath
      && appStore.workspaceNarrowSidebarOpen
    ) {
      appStore.setWorkspaceNarrowSidebarOpen(false)
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
}

.app-layout--snow-shell .app-main-shell {
  min-height: calc(100vh - var(--app-shell-top-offset));
  background: var(--app-shell-canvas, var(--lx-clay-canvas)) !important;
}

.app-layout--snow-shell .app-main-content {
  padding: var(--workspace-space-8) var(--workspace-space-7) var(--workspace-space-12);
}

:global(html:not(.dark) .app-layout--snow-shell.app-layout--flat-workspace-shell) {
  --app-shell-canvas: var(--workspace-canvas);
  --app-shell-sidebar-bg: var(--workspace-sidebar-surface);
  --app-shell-sidebar-border: var(--workspace-divider);
  --app-shell-sidebar-shadow: none;
  --app-shell-sidebar-backdrop: none;
  --app-shell-sidebar-decoration: none;
  --app-shell-sidebar-highlight: none;
  --app-shell-sidebar-hover-color: var(--workspace-text);
  --app-shell-sidebar-hover-bg: var(--workspace-hover);
  --app-shell-sidebar-hover-shadow: none;
  --app-shell-sidebar-hover-transform: none;
  --app-shell-sidebar-focus: color-mix(in srgb, var(--lx-clay-accent) 44%, transparent);
  --app-shell-sidebar-section-color: var(--workspace-text-secondary);
  --app-shell-sidebar-active-color: var(--workspace-text);
  --app-shell-sidebar-active-icon: var(--workspace-text);
  --app-shell-sidebar-active-bg: var(--workspace-selected);
  --app-shell-sidebar-active-marker: none;
}

:global(html:not(.dark) .app-layout--snow-shell.app-layout--purchase) {
  --app-shell-canvas: var(--workspace-canvas);
}

:global(.app-layout--snow-shell.app-layout--personal-work-shell) {
  --lx-clay-canvas: var(--workspace-canvas);
  --lx-clay-surface: var(--workspace-card-surface);
  --lx-clay-surface-soft: var(--workspace-surface-subtle);
  --lx-clay-surface-subtle: var(--workspace-surface-subtle);
  --lx-clay-surface-elevated: var(--workspace-popup-surface);
  --lx-clay-recessed: var(--workspace-surface-subtle);
  --lx-clay-recessed-strong: var(--workspace-selected);
  --lx-clay-text: var(--workspace-text);
  --lx-clay-text-secondary: var(--workspace-text-secondary);
  --lx-clay-text-muted: var(--workspace-text-muted);
  --lx-clay-text-subtle: var(--workspace-text-muted);
  --lx-clay-border: var(--workspace-border);
  --lx-clay-border-strong: var(--workspace-border-strong);
  --lx-clay-hover: var(--workspace-hover);
  --lx-clay-selected: var(--workspace-selected);
  --lx-clay-accent: var(--workspace-work-accent);
  --lx-clay-accent-deep: var(--workspace-work-accent-hover);
  --lx-clay-accent-soft: var(--workspace-work-accent-soft);
  --lx-clay-radius-overview: var(--workspace-radius-card);
  --lx-clay-radius-surface: var(--workspace-radius-card);
  --lx-clay-radius-form: var(--workspace-radius-card);
  --lx-clay-radius-control: var(--workspace-radius-input);
  --lx-clay-radius-button: var(--workspace-radius-button);
  --lx-clay-shadow-surface: var(--workspace-shadow-surface);
  --lx-clay-shadow-flat: none;
  --lx-clay-shadow-form: none;
}

:global(html:not(.dark) .app-layout--snow-shell.app-layout--personal-work-shell) {
  --app-shell-canvas: var(--workspace-canvas);
  --app-shell-sidebar-bg: var(--workspace-sidebar-surface);
  --app-shell-sidebar-border: var(--workspace-divider);
  --app-shell-sidebar-hover-color: var(--workspace-work-text);
  --app-shell-sidebar-hover-bg: var(--workspace-hover);
  --app-shell-sidebar-hover-shadow: none;
  --app-shell-sidebar-focus: var(--workspace-work-text-muted);
  --app-shell-sidebar-section-color: var(--workspace-work-text-muted);
  --app-shell-sidebar-active-color: var(--workspace-work-text);
  --app-shell-sidebar-active-icon: var(--workspace-work-text);
  --app-shell-sidebar-active-bg: var(--workspace-selected);
  --app-shell-sidebar-active-marker: none;
}

:global(html.dark .app-layout--snow-shell.app-layout--personal-work-shell) {
  --app-shell-canvas: var(--workspace-canvas);
  --app-shell-sidebar-bg: var(--workspace-sidebar-surface);
  --app-shell-sidebar-border: var(--workspace-divider);
  --app-shell-sidebar-hover-color: var(--workspace-text);
  --app-shell-sidebar-hover-bg: var(--workspace-hover);
  --app-shell-sidebar-hover-shadow: none;
  --app-shell-sidebar-focus: var(--workspace-text-muted);
  --app-shell-sidebar-section-color: var(--workspace-text-muted);
  --app-shell-sidebar-active-color: var(--workspace-text);
  --app-shell-sidebar-active-icon: var(--workspace-text);
  --app-shell-sidebar-active-bg: var(--workspace-selected);
  --app-shell-sidebar-active-marker: none;
  --app-shell-sidebar-shadow: none;
  --app-shell-sidebar-decoration: none;
  --app-shell-sidebar-highlight: none;
  --app-shell-sidebar-backdrop: none;
}

.app-layout--snow-shell.app-layout--purchase .app-main-content {
  padding: var(--workspace-space-10);
}

:global(body.app-flat-workspace-active:not(.admin-home-clay-portals)) {
  --lx-clay-canvas: var(--workspace-canvas);
  --lx-clay-surface: var(--workspace-card-surface);
  --lx-clay-surface-soft: var(--workspace-surface-subtle);
  --lx-clay-surface-subtle: var(--workspace-surface-subtle);
  --lx-clay-surface-elevated: var(--workspace-popup-surface);
  --lx-clay-overlay-surface-soft: var(--workspace-popup-surface);
  --lx-clay-sidebar: var(--workspace-sidebar-surface);
  --lx-clay-recessed: var(--workspace-surface-subtle);
  --lx-clay-recessed-strong: var(--workspace-selected);
  --lx-clay-text: var(--workspace-text);
  --lx-clay-text-secondary: var(--workspace-text-secondary);
  --lx-clay-text-muted: var(--workspace-text-muted);
  --lx-clay-text-subtle: var(--workspace-text-muted);
  --lx-clay-border: var(--workspace-border);
  --lx-clay-border-strong: var(--workspace-border-strong);
  --lx-clay-hover: var(--workspace-hover);
  --lx-clay-selected: var(--workspace-selected);
  background: var(--workspace-canvas);
  color: var(--workspace-text);
}

@media (max-width: 767px) {
  .app-layout--snow-shell .app-main-shell {
    min-height: calc(100vh - var(--app-shell-top-offset));
  }

  .app-layout--snow-shell .app-main-content {
    padding: var(--workspace-space-5-5) var(--workspace-space-4) var(--workspace-space-10);
  }
}

@media (max-width: 767px) and (hover: none) and (pointer: coarse) {
  .app-layout--snow-shell:not(.app-layout--chat-shell) .app-main-shell {
    margin-left: 0 !important;
  }
}

@media (max-width: 1023px) {
  .app-layout--snow-shell.app-layout--purchase .app-main-content {
    padding: var(--workspace-space-6);
  }
}

.app-layout--snow-shell .app-main-shell.app-layout--chat {
  height: calc(100dvh - var(--app-shell-top-offset));
  min-height: 0;
  overflow: hidden;
}

.app-layout--snow-shell.app-layout--chat-shell {
  --app-shell-top-offset: 0px;
  --app-shell-canvas: var(--workspace-canvas);
  --lx-clay-canvas: var(--workspace-canvas);
  --lx-clay-surface: var(--workspace-card-surface);
  --lx-clay-surface-soft: var(--workspace-surface-subtle);
  --lx-clay-surface-subtle: var(--workspace-surface-subtle);
  --lx-clay-surface-elevated: var(--workspace-popup-surface);
  --lx-clay-recessed: var(--workspace-surface-subtle);
  --lx-clay-recessed-strong: var(--workspace-border);
  --lx-clay-text: var(--workspace-text);
  --lx-clay-text-secondary: var(--workspace-text-secondary);
  --lx-clay-text-muted: var(--workspace-text-muted);
  --lx-clay-text-subtle: var(--workspace-text-muted);
  --lx-clay-border: var(--workspace-border);
  --lx-clay-border-strong: var(--workspace-border-strong);
  --lx-clay-hover: var(--workspace-hover);
  --lx-clay-selected: var(--workspace-selected);
  --lx-clay-accent: var(--workspace-action);
  --lx-clay-accent-deep: var(--workspace-action-hover);
  --lx-clay-accent-soft: var(--workspace-action-soft);
  --lx-clay-radius-control: var(--workspace-radius-input);
  --lx-clay-radius-button: var(--workspace-radius-button);
  --lx-clay-shadow-surface: var(--workspace-shadow-float);
  --lx-clay-shadow-flat: none;
  --lx-clay-shadow-form: none;
  height: 100dvh;
  min-height: 0;
  padding-top: 0 !important;
  overflow: hidden;
  background: var(--workspace-canvas) !important;
}

.app-layout--chat-shell .app-main-shell,
.app-layout--chat-shell .app-main-shell.app-layout--chat,
.app-layout--chat-shell .app-main-content,
.app-layout--chat-shell .app-main-inner {
  width: 100%;
  height: 100dvh;
  min-height: 0;
  margin-left: 0 !important;
}

.app-layout--chat-shell .app-main-inner {
  max-width: none;
  margin-right: 0 !important;
}

.app-layout--snow-shell.app-layout--chat-shell .app-main-shell.app-layout--chat .app-main-content {
  padding: 0;
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

.app-layout--snow-shell .app-main-content.app-main-content--workbench {
  min-height: calc(100dvh - var(--app-shell-top-offset));
  padding: 0;
}

.app-layout--snow-shell .app-main-inner.app-main-inner--workbench {
  min-height: inherit;
  max-width: none;
}

/* Superdesign Option B: the operator workbench owns the full desktop frame. */
.app-layout--snow-shell.app-layout--ops-option-b {
  --app-shell-top-offset: 0px;
  --app-shell-canvas: var(--workspace-canvas);
  min-height: 100dvh;
  padding-top: 0 !important;
  background: var(--workspace-canvas) !important;
}

.app-layout--ops-option-b .app-main-shell,
.app-layout--ops-option-b .app-main-content,
.app-layout--ops-option-b .app-main-inner {
  min-height: 100dvh;
}

.app-layout--ops-option-b .app-main-shell {
  background: var(--workspace-canvas) !important;
}

</style>

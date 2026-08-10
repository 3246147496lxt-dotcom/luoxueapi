<template>
  <div
    v-if="profile"
    class="sidebar-account-dock"
    :class="{
      'sidebar-account-dock--collapsed': sidebarCollapsed,
      'sidebar-account-dock--chat': context === 'chat',
      'sidebar-account-dock--personal': !isAdminWorkspace,
      'sidebar-account-dock--work': context === 'work' && !isAdminWorkspace,
    }"
    data-testid="sidebar-account-dock"
  >
    <div
      ref="dockRowRef"
      class="sidebar-account-row"
      :class="{ 'sidebar-account-row--open': panelOpen }"
    >
      <button
        ref="triggerRef"
        type="button"
        class="sidebar-account-trigger"
        :title="sidebarCollapsed ? triggerAriaLabel : undefined"
        :aria-label="triggerAriaLabel"
        aria-haspopup="dialog"
        aria-controls="sidebar-account-panel"
        :aria-expanded="panelOpen"
        @click="togglePanel"
      >
        <span class="sidebar-account-trigger__avatar">
          <img
            v-if="profile.avatarUrl && !isOpsOptionB"
            :src="profile.avatarUrl"
            :alt="profile.displayName"
          >
          <span v-else>{{ isOpsOptionB ? t('admin.ops.sidebar.avatar') : profile.initials }}</span>
        </span>

        <span
          class="sidebar-account-trigger__copy"
          :aria-hidden="sidebarCollapsed ? 'true' : undefined"
        >
          <span class="sidebar-account-trigger__name">
            {{ isOpsOptionB ? t('admin.ops.sidebar.admin') : profile.displayName }}
          </span>
          <span v-if="isOpsOptionB" class="sidebar-account-trigger__meta">
            {{ t('admin.ops.sidebar.balanceSubscription') }}
          </span>
          <span v-else class="sidebar-account-trigger__meta">
            {{ accountPlanLabel }}
          </span>
        </span>

        <span
          v-if="(isOpsOptionB || !isAdminWorkspace) && !sidebarCollapsed"
          class="sidebar-account-trigger__chevrons"
          aria-hidden="true"
        >
          <Icon name="chevronsUpDown" size="sm" />
        </span>
      </button>

      <RouterLink
        v-if="isAdminWorkspace && pricingTarget && !sidebarCollapsed && !isOpsOptionB"
        data-testid="account-upgrade-link"
        class="sidebar-account-cta"
        :to="pricingTarget.path"
        @click="handleCtaClick"
      >
        {{ t('accountDock.upgrade') }}
      </RouterLink>
    </div>

    <SidebarAccountOverlay
      :open="panelOpen"
      :anchor-element="dockRowRef"
      :summary="panelSummary"
      :show-onboarding="showOnboarding"
      :context="context"
      :variant="isAdminWorkspace ? 'admin' : 'personal'"
      :plan-label="accountPlanLabel"
      :help-href="helpHref"
      :workspace-target="workspaceTarget"
      @close="closePanel"
      @logout="handleLogout"
      @replay="handleReplay"
      @open-settings="handleOpenSettings"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { useUserMembership } from '@/composables/useUserMembership'
import { openPersonalSettings } from '@/navigation/personalSettingsRoute'
import { resolveDocumentationUrl } from '@/utils/documentationUrl'
import {
  getShellDestinationSpecs,
  selectVisibleShellDestinations,
  toShellCapabilityState,
  type ShellDestinationSpec,
} from '@/navigation/shellDestinations'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingStore } from '@/stores/onboarding'
import { useUserProfileStore } from '@/stores/userProfile'
import Icon from '@/components/icons/Icon.vue'
import SidebarAccountOverlay from './SidebarAccountOverlay.vue'
import type { PersonalSettingsSection } from '@/navigation/personalSettingsRoute'
import type { AccountPanelSummary } from './accountPanelTypes'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()
const onboardingStore = useOnboardingStore()
const userProfileStore = useUserProfileStore()
const props = withDefaults(defineProps<{
  context?: 'work' | 'chat'
  collapsed?: boolean
}>(), {
  context: 'work',
  collapsed: false,
})
const context = computed(() => props.context)
const {
  profile,
  subscriptionsLoaded,
  activeSubscriptionCount,
  primarySubscription,
} = storeToRefs(userProfileStore)
const membership = useUserMembership({
  subscriptionsLoaded,
  activeSubscriptionCount,
  primarySubscription,
})

const panelOpen = ref(false)
const dockRowRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLButtonElement | null>(null)
const isOpsOptionB = computed(() => route.path.startsWith('/admin/ops'))
const isAdminWorkspace = computed(() => (
  authStore.isAdmin && route.path.startsWith('/admin')
))
const sidebarCollapsed = computed(() => props.collapsed)
const audience = computed(() => (isAdminWorkspace.value ? 'admin' as const : 'user' as const))
const settingsAudience = computed(() => (
  isAdminWorkspace.value ? 'admin' as const : 'user' as const
))
const showOnboarding = computed(
  () => !authStore.isSimpleMode && isAdminWorkspace.value,
)

const destinationContext = computed(() => ({
  audience: audience.value,
  simpleMode: authStore.isSimpleMode,
  capabilities: {
    payment: toShellCapabilityState(
      appStore.cachedPublicSettings?.payment_enabled,
    ),
    'public-model-catalog': toShellCapabilityState(
      appStore.backendModeEnabled
        ? false
        : appStore.cachedPublicSettings?.public_model_catalog_enabled,
    ),
  },
}))

function findVisibleRouteDestination(id: ShellDestinationSpec['id']) {
  const spec = selectVisibleShellDestinations(
    getShellDestinationSpecs(audience.value),
    destinationContext.value,
  ).find((destination) => (
    destination.id === id
    && destination.target.kind === 'route'
  ))

  return spec?.target.kind === 'route' ? spec.target : null
}

function formatCredit(value: number) {
  return Number.isFinite(value) ? value.toFixed(2) : '0.00'
}

const adminSubscriptionStatusText = computed(() => t('accountDock.payAsYouGo'))

const accountPlanLabel = computed(() => (
  isAdminWorkspace.value
    ? adminSubscriptionStatusText.value
    : membership.accountPlanLabel.value
))

const triggerAriaLabel = computed(() => (
  isAdminWorkspace.value
    ? [
        t('accountDock.open'),
        profile.value?.displayName ?? '',
        t('accountDock.summary', {
          balance: formatCredit(profile.value?.availableBalance ?? 0),
          subscription: accountPlanLabel.value,
        }),
      ].join(' · ')
    : [
        t('accountDock.open'),
        profile.value?.displayName ?? '',
        accountPlanLabel.value,
      ].filter(Boolean).join(' · ')
))

const pricingTarget = computed(() => (
  props.context === 'chat' ? null : findVisibleRouteDestination('pricing')
))

const helpHref = computed(() => resolveDocumentationUrl(
  appStore.cachedPublicSettings?.doc_url || appStore.docUrl,
))

const workspaceTarget = computed(() => {
  if (props.context === 'chat' || !authStore.isAdmin) return null
  return isAdminWorkspace.value
    ? { href: '/dashboard', label: t('nav.switchToPersonalWorkspace') }
    : { href: '/admin/dashboard', label: t('nav.switchToAdminWorkspace') }
})

const panelSummary = computed<AccountPanelSummary>(() => ({
  displayName: profile.value?.displayName ?? '',
  email: profile.value?.email ?? '',
  initials: profile.value?.initials ?? '',
  avatarUrl: profile.value?.avatarUrl ?? '',
  frozenBalance: profile.value?.frozenBalance ?? 0,
  formattedAvailableBalance: formatCredit(profile.value?.availableBalance ?? 0),
  formattedFrozenBalance: formatCredit(profile.value?.frozenBalance ?? 0),
  activeSubscriptionCount: isAdminWorkspace.value ? 0 : activeSubscriptionCount.value,
  subscriptionsLoaded: isAdminWorkspace.value ? true : subscriptionsLoaded.value,
}))

function isMobileViewport() {
  return appStore.workspaceMobileDrawer
}

function togglePanel() {
  if (panelOpen.value) {
    closePanel()
    return
  }

  panelOpen.value = true
  if (isMobileViewport() && isAdminWorkspace.value) {
    appStore.setMobileOpen(false)
  }
}

function focusReturnTarget() {
  const target = isMobileViewport() && props.context !== 'chat'
    ? document.querySelector<HTMLButtonElement>('[data-testid="mobile-header-menu"]')
    : triggerRef.value
  target?.focus({ preventScroll: true })
}

function closePanel(restoreFocus = true) {
  if (!panelOpen.value) return
  panelOpen.value = false
  if (restoreFocus) {
    void nextTick(focusReturnTarget)
  }
}

async function handleLogout() {
  closePanel(false)
  try {
    await authStore.logout()
  } catch (error) {
    console.error('Logout error:', error)
  }
  await router.replace('/login')
}

function handleReplay() {
  closePanel(false)
  onboardingStore.replay()
}

function handleCtaClick() {
  closePanel(false)
  if (isMobileViewport() || appStore.mobileOpen) {
    appStore.setMobileOpen(false)
  }
  if (appStore.workspaceNarrowSidebar) {
    appStore.setWorkspaceNarrowSidebarOpen(false)
  }
}

async function handleOpenSettings(section: PersonalSettingsSection) {
  closePanel(false)
  await nextTick()
  await openPersonalSettings(
    router,
    route,
    settingsAudience.value,
    section,
  )
}

watch(
  () => route.fullPath,
  () => closePanel(false),
)

watch(sidebarCollapsed, () => closePanel(false))

watch(
  () => appStore.mobileOpen,
  (mobileOpen) => {
    if (!mobileOpen && props.context === 'work' && !isAdminWorkspace.value) {
      closePanel(false)
    }
  },
)

watch(
  () => appStore.workspaceNarrowSidebarOpen,
  (open) => {
    if (!open && appStore.workspaceNarrowSidebar) closePanel(false)
  },
)
</script>

<style scoped>
.sidebar-account-dock {
  position: relative;
  z-index: 2;
  width: 100%;
  flex: 0 0 auto;
  padding:
    0
    var(--workspace-space-1-5)
    calc(var(--workspace-space-1-5) + env(safe-area-inset-bottom))
    var(--workspace-space-2);
}

.sidebar-account-row {
  display: grid;
  width: 100%;
  min-height: var(--workspace-sidebar-footer-row-height);
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: stretch;
  padding-right: var(--workspace-space-1-5);
  overflow: hidden;
  border-radius: var(--workspace-radius-button);
  color: var(--workspace-dock-text);
  transition:
    width var(--workspace-sidebar-transition-duration) var(--workspace-sidebar-transition-easing),
    color 150ms ease,
    background-color 150ms ease;
}

.sidebar-account-row:hover,
.sidebar-account-row--open {
  color: var(--workspace-dock-text-strong);
  background: var(--workspace-dock-hover);
}

.sidebar-account-dock--personal .sidebar-account-row,
.sidebar-account-dock--personal .sidebar-account-row:hover,
.sidebar-account-dock--personal .sidebar-account-row--open {
  color: var(--workspace-identity-text);
}

.sidebar-account-dock--personal .sidebar-account-row {
  padding-right: 0;
}

.sidebar-account-trigger {
  display: grid;
  min-width: 0;
  min-height: var(--workspace-sidebar-footer-row-height);
  grid-template-columns: var(--workspace-sidebar-touch-target) minmax(0, 1fr) auto;
  align-items: center;
  gap: var(--workspace-space-2-5);
  padding:
    var(--workspace-space-1-5)
    var(--workspace-space-1)
    var(--workspace-space-1-5)
    0;
  border-radius: inherit;
  color: inherit;
  text-align: left;
}

.sidebar-account-trigger:focus-visible,
.sidebar-account-cta:focus-visible {
  outline: 2px solid var(--workspace-dock-focus);
  outline-offset: -2px;
}

.sidebar-account-dock--personal .sidebar-account-trigger:focus-visible {
  outline-color: var(--workspace-dock-focus-personal);
}

.sidebar-account-dock--personal .sidebar-account-trigger {
  grid-template-columns: 24px minmax(0, 1fr) 36px;
  gap: var(--workspace-space-2);
  padding: var(--workspace-space-2);
}

.sidebar-account-trigger__avatar {
  position: relative;
  display: flex;
  width: var(--workspace-avatar-size-md);
  height: var(--workspace-avatar-size-md);
  align-items: center;
  justify-content: center;
  justify-self: center;
  overflow: visible;
  border-radius: var(--workspace-radius-pill);
  color: var(--workspace-identity-avatar-text);
  background: var(--workspace-identity-avatar-surface);
  font-size: 0.75rem;
  font-weight: 700;
}

.sidebar-account-trigger__avatar img {
  width: 100%;
  height: 100%;
  overflow: hidden;
  border-radius: inherit;
  object-fit: cover;
}

.sidebar-account-dock--personal .sidebar-account-trigger__avatar {
  width: 24px;
  height: 24px;
  border-radius: var(--workspace-radius-pill);
  color: var(--workspace-identity-text);
  background: var(--workspace-hover);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
}

:global(html.dark .sidebar-account-dock--personal .sidebar-account-trigger__avatar) {
  color: var(--workspace-identity-text);
  background: var(--workspace-surface-subtle);
  box-shadow: inset 0 0 0 1px var(--workspace-border-strong);
}

.sidebar-account-trigger__copy {
  display: flex;
  min-width: 0;
  max-width: 12rem;
  flex-direction: column;
  gap: var(--workspace-space-0-25);
  overflow: hidden;
  opacity: 1;
  transition:
    max-width 0.2s ease,
    opacity 0.12s ease;
}

.sidebar-account-trigger__name,
.sidebar-account-trigger__meta {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sidebar-account-trigger__chevrons {
  display: grid;
  width: 36px;
  height: 36px;
  place-items: center;
  flex: 0 0 36px;
  border-radius: var(--workspace-radius-compact);
  color: var(--workspace-dock-chevron);
}

.sidebar-account-dock--personal .sidebar-account-trigger__chevrons {
  color: var(--workspace-identity-text-tertiary);
}

.sidebar-account-trigger__name {
  color: inherit;
  font-size: 0.875rem;
  font-weight: 600;
  line-height: 1.15rem;
}

.sidebar-account-trigger__meta {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: var(--workspace-space-0-75);
  color: var(--workspace-dock-text-muted);
  font-size: 0.6875rem;
  font-weight: 400;
  line-height: 1rem;
}

.sidebar-account-dock--personal .sidebar-account-trigger__name {
  font-size: 14px;
  font-weight: 400;
  line-height: 20px;
}

.sidebar-account-dock--personal .sidebar-account-trigger__meta {
  color: var(--workspace-identity-text-tertiary);
  font-size: 12px;
  font-weight: 400;
  line-height: 16px;
}

.sidebar-account-cta {
  display: inline-flex;
  min-width: 0;
  min-height: var(--workspace-avatar-size-md);
  align-self: center;
  align-items: center;
  justify-content: center;
  padding: 0 var(--workspace-space-2-75);
  border: 1px solid var(--workspace-dock-cta-border);
  border-radius: var(--workspace-radius-control-sm);
  color: var(--workspace-dock-cta-text);
  background: var(--workspace-dock-cta-surface);
  font-size: 0.8125rem;
  font-weight: 500;
  line-height: 1;
  text-decoration: none;
  white-space: nowrap;
  transition:
    border-color 150ms ease,
    background-color 150ms ease;
}

.sidebar-account-dock--personal .sidebar-account-cta {
  display: none;
}

.sidebar-account-cta:hover {
  border-color: var(--workspace-dock-cta-border-hover);
  background: var(--workspace-dock-cta-surface-hover);
}

.sidebar-account-dock--collapsed .sidebar-account-row {
  width: var(--workspace-sidebar-touch-target);
  padding-right: 0;
}

.sidebar-account-dock--collapsed .sidebar-account-trigger {
  width: var(--workspace-sidebar-touch-target);
  min-height: var(--workspace-sidebar-footer-row-height);
  grid-template-columns: var(--workspace-sidebar-touch-target) minmax(0, 0) 0;
  justify-content: start;
  gap: 0;
  padding: 0;
}

.sidebar-account-dock--personal.sidebar-account-dock--collapsed .sidebar-account-trigger {
  grid-template-columns: var(--workspace-sidebar-touch-target) minmax(0, 0) 0;
  gap: 0;
  padding: 0;
}

.sidebar-account-dock--collapsed .sidebar-account-trigger__copy {
  max-width: 0;
  opacity: 0;
  pointer-events: none;
}

@media (prefers-reduced-motion: reduce) {
  .sidebar-account-row,
  .sidebar-account-cta,
  .sidebar-account-trigger__copy {
    transition-duration: 0.01ms;
  }
}
</style>

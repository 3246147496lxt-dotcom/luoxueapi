<template>
  <div
    v-if="summary.hasUser.value"
    class="sidebar-account-dock"
    :class="{ 'sidebar-account-dock--collapsed': sidebarCollapsed }"
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
            v-if="summary.avatarUrl.value && !isOpsOptionB"
            :src="summary.avatarUrl.value"
            :alt="summary.displayName.value"
          >
          <span v-else>{{ isOpsOptionB ? t('admin.ops.sidebar.avatar') : summary.initials.value }}</span>
        </span>

        <span
          class="sidebar-account-trigger__copy"
          :aria-hidden="sidebarCollapsed ? 'true' : undefined"
        >
          <span class="sidebar-account-trigger__name">
            {{ isOpsOptionB ? t('admin.ops.sidebar.admin') : summary.displayName.value }}
          </span>
          <span v-if="isOpsOptionB" class="sidebar-account-trigger__meta">
            {{ t('admin.ops.sidebar.balanceSubscription') }}
          </span>
          <span v-else class="sidebar-account-trigger__meta">
            <span class="sidebar-account-trigger__balance-label">
              {{ t('accountDock.balanceShort') }}
            </span>
            <CreditAmount
              :value="formatCredit(summary.availableBalance.value)"
              icon-size="xs"
              :label="`${t('accountDock.availableBalance')} ${formatCredit(summary.availableBalance.value)}`"
            />
            <span aria-hidden="true">·</span>
            <span class="truncate">{{ subscriptionStatusText }}</span>
          </span>
        </span>

        <Icon
          v-if="isOpsOptionB && !sidebarCollapsed"
          name="chevronsUpDown"
          size="sm"
          class="sidebar-account-trigger__chevrons"
          aria-hidden="true"
        />
      </button>

      <RouterLink
        v-if="pricingTarget && !sidebarCollapsed && !isOpsOptionB"
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
      @close="closePanel"
      @logout="handleLogout"
      @replay="handleReplay"
      @open-settings="handleOpenSettings"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { useAccountSummary } from '@/composables/useAccountSummary'
import { openPersonalSettings } from '@/navigation/personalSettingsRoute'
import {
  getShellDestinationSpecs,
  selectVisibleShellDestinations,
  toShellCapabilityState,
  type ShellDestinationSpec,
} from '@/navigation/shellDestinations'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingStore } from '@/stores/onboarding'
import CreditAmount from '@/components/common/CreditAmount.vue'
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
const summary = useAccountSummary()

const panelOpen = ref(false)
const dockRowRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLButtonElement | null>(null)
const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const isOpsOptionB = computed(() => route.path.startsWith('/admin/ops'))
const audience = computed(() => summary.isAdmin.value ? 'admin' as const : 'user' as const)
const settingsAudience = computed(() => (
  summary.isAdmin.value && route.path.startsWith('/admin')
    ? 'admin' as const
    : 'user' as const
))
const showOnboarding = computed(
  () => !summary.isSimpleMode.value && summary.isAdmin.value,
)

const destinationContext = computed(() => ({
  audience: audience.value,
  simpleMode: summary.isSimpleMode.value,
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

const subscriptionStatusText = computed(() => {
  if (
    !summary.subscriptionsLoaded.value
  ) {
    return t('accountDock.subscriptionLoading')
  }
  if (summary.activeSubscriptionCount.value > 0) {
    return t('accountDock.activeSubscriptions', {
      count: summary.activeSubscriptionCount.value,
    })
  }
  return t('accountDock.payAsYouGo')
})

const triggerAriaLabel = computed(() => [
  t('accountDock.open'),
  summary.displayName.value,
  t('accountDock.summary', {
    balance: formatCredit(summary.availableBalance.value),
    subscription: subscriptionStatusText.value,
  }),
].join(' · '))

const pricingTarget = computed(() => findVisibleRouteDestination('pricing'))

const panelSummary = computed<AccountPanelSummary>(() => ({
  displayName: summary.displayName.value,
  email: summary.email.value,
  initials: summary.initials.value,
  avatarUrl: summary.avatarUrl.value,
  frozenBalance: summary.frozenBalance.value,
  formattedAvailableBalance: formatCredit(summary.availableBalance.value),
  formattedFrozenBalance: formatCredit(summary.frozenBalance.value),
  activeSubscriptionCount: summary.activeSubscriptionCount.value,
  subscriptionsLoaded: summary.subscriptionsLoaded.value,
}))

function isMobileViewport() {
  return typeof window !== 'undefined' && window.matchMedia('(max-width: 1023px)').matches
}

function togglePanel() {
  if (panelOpen.value) {
    closePanel()
    return
  }

  panelOpen.value = true
  if (isMobileViewport()) {
    appStore.setMobileOpen(false)
  }
}

function focusReturnTarget() {
  const target = isMobileViewport()
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
  if (isMobileViewport()) {
    appStore.setMobileOpen(false)
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
</script>

<style scoped>
.sidebar-account-dock {
  position: relative;
  z-index: 2;
  width: 100%;
  flex: 0 0 auto;
  padding: 0 6px calc(6px + env(safe-area-inset-bottom)) 8px;
}

.sidebar-account-row {
  display: grid;
  width: 100%;
  min-height: 52px;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: stretch;
  padding-right: 6px;
  overflow: hidden;
  border-radius: 10px;
  color: rgb(30 41 59);
  transition:
    color 150ms ease,
    background-color 150ms ease;
}

.sidebar-account-row:hover,
.sidebar-account-row--open {
  color: rgb(15 23 42);
  background: rgb(15 23 42 / 0.055);
}

.sidebar-account-trigger {
  display: grid;
  min-width: 0;
  min-height: 52px;
  grid-template-columns: 32px minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
  padding: 6px 4px 6px 8px;
  border-radius: inherit;
  color: inherit;
  text-align: left;
}

.sidebar-account-trigger:focus-visible,
.sidebar-account-cta:focus-visible {
  outline: 2px solid var(--app-shell-sidebar-focus, rgb(0 132 255 / 0.5));
  outline-offset: -2px;
}

.sidebar-account-trigger__avatar {
  position: relative;
  display: flex;
  width: 32px;
  height: 32px;
  align-items: center;
  justify-content: center;
  overflow: visible;
  border-radius: 999px;
  color: #1c1f23;
  background: #fce865;
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

.sidebar-account-trigger__copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 1px;
}

.sidebar-account-trigger__name,
.sidebar-account-trigger__meta {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sidebar-account-trigger__chevrons {
  flex: 0 0 auto;
  color: #5f6b7a;
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
  gap: 3px;
  color: rgb(100 116 139);
  font-size: 0.6875rem;
  font-weight: 400;
  line-height: 1rem;
}

.sidebar-account-cta {
  display: inline-flex;
  min-width: 0;
  min-height: 32px;
  align-self: center;
  align-items: center;
  justify-content: center;
  padding: 0 11px;
  border: 1px solid rgb(15 23 42 / 0.15);
  border-radius: 999px;
  color: rgb(15 23 42);
  background: rgb(255 255 255 / 0.9);
  font-size: 0.8125rem;
  font-weight: 500;
  line-height: 1;
  text-decoration: none;
  white-space: nowrap;
  transition:
    border-color 150ms ease,
    background-color 150ms ease;
}

.sidebar-account-cta:hover {
  border-color: rgb(15 23 42 / 0.24);
  background: #fff;
}

.sidebar-account-dock--collapsed {
  padding-right: 7px;
  padding-left: 7px;
}

.sidebar-account-dock--collapsed .sidebar-account-row {
  display: block;
  padding-right: 0;
}

.sidebar-account-dock--collapsed .sidebar-account-trigger {
  width: 100%;
  min-height: 52px;
  grid-template-columns: 32px;
  justify-content: center;
  gap: 0;
  padding: 0;
}

.sidebar-account-dock--collapsed .sidebar-account-trigger__copy {
  display: none;
}

:global(html.dark .sidebar-account-row) {
  color: rgb(226 232 240);
}

:global(html.dark .sidebar-account-row:hover),
:global(html.dark .sidebar-account-row--open) {
  color: #fff;
  background: rgb(255 255 255 / 0.08);
}

:global(html.dark .sidebar-account-trigger__meta) {
  color: rgb(148 163 184);
}

:global(html.dark .sidebar-account-cta) {
  border-color: rgb(255 255 255 / 0.16);
  color: rgb(248 250 252);
  background: rgb(255 255 255 / 0.08);
}

:global(html.dark .sidebar-account-cta:hover) {
  border-color: rgb(255 255 255 / 0.28);
  background: rgb(255 255 255 / 0.12);
}

@media (prefers-reduced-motion: reduce) {
  .sidebar-account-row,
  .sidebar-account-cta {
    transition-duration: 0.01ms;
  }
}
</style>

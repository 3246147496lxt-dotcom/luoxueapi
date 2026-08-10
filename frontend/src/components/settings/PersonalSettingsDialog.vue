<template>
  <Teleport to="body">
    <Transition name="personal-settings">
      <div
        v-if="isOpen"
        class="personal-settings-overlay"
        data-testid="personal-settings-overlay"
        @mousedown.self="handleBackdrop"
      >
        <div
          ref="dialogRef"
          class="personal-settings-dialog"
          data-testid="personal-settings-dialog"
          role="dialog"
          aria-modal="true"
          :aria-labelledby="dialogTitleId"
          tabindex="-1"
        >
          <aside class="personal-settings-sidebar">
            <button
              type="button"
              class="personal-settings-close"
              :aria-label="t('personalSettings.dialog.close')"
              data-testid="personal-settings-close-desktop"
              @click="closeDialog"
            >
              <Icon name="x" size="sm" aria-hidden="true" />
            </button>

            <div
              class="personal-settings-tabs"
              role="tablist"
              :aria-label="t('personalSettings.dialog.navigation')"
            >
              <button
                v-for="(tab, index) in tabs"
                :key="tab.id"
                type="button"
                class="personal-settings-tab"
                role="tab"
                :id="`personal-settings-tab-${tab.id}`"
                :aria-selected="section === tab.id"
                :aria-controls="`personal-settings-panel-${tab.id}`"
                :tabindex="section === tab.id ? 0 : -1"
                :data-settings-section="tab.id"
                @click="selectSection(tab.id)"
                @keydown="handleTabKeydown($event, index)"
              >
                <Icon :name="tab.icon" size="sm" aria-hidden="true" />
                <span>{{ t(tab.labelKey) }}</span>
              </button>
            </div>
          </aside>

          <div class="personal-settings-main">
            <header class="personal-settings-header">
              <button
                v-if="detail"
                type="button"
                class="personal-settings-back"
                :aria-label="t('personalSettings.dialog.back')"
                data-testid="personal-settings-back"
                @click="goBackToSection"
              >
                <Icon name="arrowLeft" size="sm" aria-hidden="true" />
              </button>
              <h2 :id="dialogTitleId">
                {{ dialogTitle }}
              </h2>
              <button
                type="button"
                class="personal-settings-close personal-settings-close--mobile"
                :aria-label="t('personalSettings.dialog.close')"
                data-testid="personal-settings-close-mobile"
                @click="closeDialog"
              >
                <Icon name="x" size="sm" aria-hidden="true" />
              </button>
            </header>

            <div class="personal-settings-body">
              <div
                v-if="hasVisited('general')"
                id="personal-settings-panel-general"
                role="tabpanel"
                aria-labelledby="personal-settings-tab-general"
                :hidden="section !== 'general'"
              >
                <PersonalSettingsGeneralPanel />
              </div>

              <div
                v-if="hasVisited('account')"
                id="personal-settings-panel-account"
                role="tabpanel"
                aria-labelledby="personal-settings-tab-account"
                :hidden="section !== 'account'"
              >
                <PersonalSettingsAccountPanel
                  :user="authStore.user"
                  :public-settings="appStore.cachedPublicSettings"
                  :detail="accountDetail"
                  :loading="accountLoading || publicSettingsLoading"
                  :error="accountError || publicSettingsError"
                  @retry="retryAccountData"
                  @open-detail="openDetail"
                  @open-devices="openDesktopDevices"
                />
              </div>

              <div
                v-if="hasVisited('security')"
                id="personal-settings-panel-security"
                role="tabpanel"
                aria-labelledby="personal-settings-tab-security"
                :hidden="section !== 'security'"
              >
                <PersonalSettingsSecurityPanel
                  :detail="securityDetail"
                  @open-detail="openDetail"
                />
              </div>

              <div
                v-if="hasVisited('notifications')"
                id="personal-settings-panel-notifications"
                role="tabpanel"
                aria-labelledby="personal-settings-tab-notifications"
                :hidden="section !== 'notifications'"
              >
                <PersonalSettingsNotificationsPanel
                  :user="authStore.user"
                  :public-settings="appStore.cachedPublicSettings"
                  :detail="notificationsDetail"
                  :loading="publicSettingsLoading"
                  :error="publicSettingsError"
                  @retry="retryPublicSettings"
                  @open-detail="openDetail"
                />
              </div>

              <div
                v-if="hasVisited('billing')"
                id="personal-settings-panel-billing"
                class="personal-settings-panel"
                role="tabpanel"
                aria-labelledby="personal-settings-tab-billing"
                :hidden="section !== 'billing'"
              >
                <div
                  v-if="publicSettingsError && !appStore.cachedPublicSettings"
                  class="personal-settings-error mb-3"
                  role="alert"
                  data-testid="personal-settings-billing-error"
                >
                  <span>{{ t('personalSettings.wallet.capabilityLoadError') }}</span>
                  <button type="button" @click="retryPublicSettings">
                    {{ t('personalSettings.common.retry') }}
                  </button>
                </div>
                <div
                  v-if="publicSettingsLoading && !appStore.cachedPublicSettings"
                  class="space-y-5 py-3"
                  aria-busy="true"
                  data-testid="personal-settings-billing-loading"
                >
                  <div class="personal-settings-skeleton w-2/3"></div>
                  <div class="personal-settings-skeleton w-full"></div>
                  <div class="personal-settings-skeleton w-5/6"></div>
                </div>
                <WalletSubscriptionSettingsPanel
                  v-else-if="!publicSettingsError || !!appStore.cachedPublicSettings"
                  :summary="billingSummary"
                  :payment-enabled="paymentEnabled"
                  :simple-mode="authStore.isSimpleMode"
                  @navigate="handleDeepNavigation"
                />
              </div>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import './personal-settings.css'
import {
  computed,
  defineAsyncComponent,
  nextTick,
  onBeforeUnmount,
  ref,
  watch,
} from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'
import type { AccountPanelSummary } from '@/components/layout/accountPanelTypes'
import {
  canonicalizePersonalSettingsRoute,
  closePersonalSettings,
  parsePersonalSettingsRoute,
  replacePersonalSettingsDetail,
  replacePersonalSettingsSection,
  type PersonalSettingsDetail,
  type PersonalSettingsSection,
} from '@/navigation/personalSettingsRoute'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useUserProfileStore } from '@/stores/userProfile'
import { acquireBodyScrollLock, releaseBodyScrollLock } from '@/utils/bodyScrollLock'
import {
  focusFirstVisibleTarget,
  getVisibleFocusableElements,
  isTopModalLayer,
  isVisibleFocusableElement,
  registerModalLayer,
  unregisterModalLayer,
} from '@/utils/modalStack'

const PersonalSettingsGeneralPanel = defineAsyncComponent(
  () => import('./PersonalSettingsGeneralPanel.vue'),
)
const PersonalSettingsAccountPanel = defineAsyncComponent(
  () => import('./PersonalSettingsAccountPanel.vue'),
)
const PersonalSettingsSecurityPanel = defineAsyncComponent(
  () => import('./PersonalSettingsSecurityPanel.vue'),
)
const PersonalSettingsNotificationsPanel = defineAsyncComponent(
  () => import('./PersonalSettingsNotificationsPanel.vue'),
)
const WalletSubscriptionSettingsPanel = defineAsyncComponent(
  () => import('@/components/layout/WalletSubscriptionSettingsPanel.vue'),
)

const dialogTitleId = 'personal-settings-dialog-title'
const modalToken = Symbol('personal-settings-dialog')
const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const userProfileStore = useUserProfileStore()
const {
  profile,
  activeSubscriptionCount,
  subscriptionsLoaded,
} = storeToRefs(userProfileStore)
const dialogRef = ref<HTMLElement | null>(null)
const visitedSections = ref<ReadonlySet<PersonalSettingsSection>>(new Set())
const accountLoading = ref(false)
const accountError = ref(false)
const publicSettingsLoading = ref(false)
const publicSettingsError = ref(false)

let publicSettingsAttempted = false
let previousFocus: HTMLElement | null = null
let inertHost: HTMLElement | null = null
let inertHostWasInert = false
let inertHostAriaHidden: string | null = null
let modalActive = false

const tabs = [
  {
    id: 'general',
    icon: 'cog',
    labelKey: 'personalSettings.dialog.sections.general',
  },
  {
    id: 'account',
    icon: 'user',
    labelKey: 'personalSettings.dialog.sections.account',
  },
  {
    id: 'security',
    icon: 'shield',
    labelKey: 'personalSettings.dialog.sections.security',
  },
  {
    id: 'notifications',
    icon: 'bell',
    labelKey: 'personalSettings.dialog.sections.notifications',
  },
  {
    id: 'billing',
    icon: 'wallet',
    labelKey: 'personalSettings.dialog.sections.billing',
  },
] as const satisfies ReadonlyArray<{
  id: PersonalSettingsSection
  icon: 'cog' | 'user' | 'shield' | 'bell' | 'wallet'
  labelKey: string
}>

const routeState = computed(() => parsePersonalSettingsRoute(route))
const isOpen = computed(() => routeState.value.isOpen)
const section = computed(() => routeState.value.section)
const detail = computed(() => routeState.value.detail)
const accountDetail = computed(() => (
  section.value === 'account' ? detail.value : null
))
const securityDetail = computed(() => (
  section.value === 'security' ? detail.value : null
))
const notificationsDetail = computed(() => (
  section.value === 'notifications' ? detail.value : null
))
const paymentEnabled = computed(
  () => appStore.cachedPublicSettings?.payment_enabled ?? false,
)

function formatCredit(value: number) {
  return Number.isFinite(value) ? value.toFixed(2) : '0.00'
}

const billingSummary = computed<AccountPanelSummary>(() => ({
  displayName: profile.value?.displayName ?? '',
  email: profile.value?.email ?? '',
  initials: profile.value?.initials ?? '',
  avatarUrl: profile.value?.avatarUrl ?? '',
  frozenBalance: profile.value?.frozenBalance ?? 0,
  formattedAvailableBalance: formatCredit(profile.value?.availableBalance ?? 0),
  formattedFrozenBalance: formatCredit(profile.value?.frozenBalance ?? 0),
  activeSubscriptionCount: activeSubscriptionCount.value,
  subscriptionsLoaded: subscriptionsLoaded.value,
}))

const detailTitleKeys: Partial<Record<PersonalSettingsDetail, string>> = {
  profile: 'personalSettings.account.profileDetailTitle',
  connections: 'personalSettings.account.connectionsDetailTitle',
  password: 'personalSettings.security.passwordDetailTitle',
  totp: 'personalSettings.security.totpDetailTitle',
  'notification-emails': 'personalSettings.notifications.emailDetailTitle',
}

const dialogTitle = computed(() => {
  const detailKey = detail.value ? detailTitleKeys[detail.value] : undefined
  return detailKey
    ? t(detailKey)
    : t(`personalSettings.dialog.sections.${section.value}`)
})

function hasVisited(target: PersonalSettingsSection) {
  return visitedSections.value.has(target)
}

function markVisited(target: PersonalSettingsSection) {
  if (visitedSections.value.has(target)) return
  visitedSections.value = new Set([...visitedSections.value, target])
}

async function refreshAccountData() {
  if (accountLoading.value) return
  accountLoading.value = true
  accountError.value = false
  try {
    await userProfileStore.refreshProfile()
  } catch {
    accountError.value = true
  } finally {
    accountLoading.value = false
  }
}

async function ensurePublicSettings(force = false) {
  if (appStore.publicSettingsLoaded && !force) {
    publicSettingsError.value = false
    return
  }
  if ((publicSettingsAttempted && !force) || publicSettingsLoading.value) return

  publicSettingsAttempted = true
  publicSettingsLoading.value = true
  publicSettingsError.value = false
  try {
    const settings = await appStore.fetchPublicSettings(force)
    publicSettingsError.value = settings === null
  } catch {
    publicSettingsError.value = true
  } finally {
    publicSettingsLoading.value = false
  }
}

function ensureSectionData(target: PersonalSettingsSection) {
  if (target === 'account') {
    void ensurePublicSettings()
    return
  }
  if (target === 'notifications' || target === 'billing') {
    void ensurePublicSettings()
  }
}

function retryAccountData() {
  void refreshAccountData()
  void ensurePublicSettings(true)
}

function retryPublicSettings() {
  void ensurePublicSettings(true)
}

function selectSection(target: PersonalSettingsSection) {
  if (target === section.value && !detail.value) return
  void replacePersonalSettingsSection(router, route, target)
}

function openDetail(target: PersonalSettingsDetail) {
  void replacePersonalSettingsDetail(router, route, target)
}

function openDesktopDevices() {
  void router.push('/desktop/devices')
}

function goBackToSection() {
  void replacePersonalSettingsSection(router, route, section.value)
}

function closeDialog() {
  void closePersonalSettings(router, route)
}

function handleDeepNavigation() {
  // RouterLink owns the navigation. The destination does not carry settings
  // query state, so the dialog tears down without mounting any payment widget.
}

function handleTabKeydown(event: KeyboardEvent, currentIndex: number) {
  let nextIndex: number | null = null
  if (event.key === 'ArrowDown' || event.key === 'ArrowRight') {
    nextIndex = (currentIndex + 1) % tabs.length
  } else if (event.key === 'ArrowUp' || event.key === 'ArrowLeft') {
    nextIndex = (currentIndex - 1 + tabs.length) % tabs.length
  } else if (event.key === 'Home') {
    nextIndex = 0
  } else if (event.key === 'End') {
    nextIndex = tabs.length - 1
  }

  if (nextIndex === null) return
  event.preventDefault()
  const next = tabs[nextIndex]
  if (!next) return
  selectSection(next.id)
  void nextTick(() => {
    document
      .querySelector<HTMLButtonElement>(`[data-settings-section="${next.id}"]`)
      ?.focus()
  })
}

function revealActiveTab(target: PersonalSettingsSection) {
  void nextTick(() => {
    document
      .querySelector<HTMLElement>(`[data-settings-section="${target}"]`)
      ?.scrollIntoView?.({
        block: 'nearest',
        inline: 'nearest',
      })
  })
}

function getFocusableElements() {
  return dialogRef.value
    ? getVisibleFocusableElements(dialogRef.value)
    : []
}

function handleDocumentKeydown(event: KeyboardEvent) {
  if (!isTopModalLayer(modalToken)) return
  if (event.key === 'Escape') {
    event.preventDefault()
    closeDialog()
    return
  }
  if (event.key !== 'Tab') return

  const focusable = getFocusableElements()
  if (focusable.length === 0) {
    event.preventDefault()
    dialogRef.value?.focus()
    return
  }

  const first = focusable[0]
  const last = focusable.at(-1)
  const activeElement = document.activeElement
  const focusIsOutside = !dialogRef.value?.contains(activeElement)
  if (
    event.shiftKey
    && (
      activeElement === first
      || activeElement === dialogRef.value
      || focusIsOutside
    )
  ) {
    event.preventDefault()
    last?.focus()
  } else if (
    !event.shiftKey
    && (
      activeElement === last
      || activeElement === dialogRef.value
      || focusIsOutside
    )
  ) {
    event.preventDefault()
    first?.focus()
  }
}

function handleBackdrop() {
  if (isTopModalLayer(modalToken)) {
    closeDialog()
  }
}

function setHostInert() {
  inertHost = document.querySelector<HTMLElement>('.app-layout')
  if (!inertHost) return
  inertHostWasInert = inertHost.inert
  inertHostAriaHidden = inertHost.getAttribute('aria-hidden')
  inertHost.inert = true
  inertHost.setAttribute('aria-hidden', 'true')
}

function restoreHostInert() {
  if (!inertHost) return
  inertHost.inert = inertHostWasInert
  if (inertHostAriaHidden === null) {
    inertHost.removeAttribute('aria-hidden')
  } else {
    inertHost.setAttribute('aria-hidden', inertHostAriaHidden)
  }
  inertHost = null
}

async function activateModal() {
  if (modalActive || typeof document === 'undefined') return
  modalActive = true
  const activeElement = document.activeElement
  previousFocus = activeElement instanceof HTMLElement
    && isVisibleFocusableElement(activeElement)
    ? activeElement
    : null
  acquireBodyScrollLock(modalToken)
  registerModalLayer(modalToken)
  setHostInert()
  document.addEventListener('keydown', handleDocumentKeydown)
  await nextTick()
  dialogRef.value?.focus()
}

function focusReturnTarget() {
  focusFirstVisibleTarget([
    previousFocus,
    document.querySelector<HTMLElement>('.sidebar-account-trigger'),
    document.querySelector<HTMLElement>('[data-testid="mobile-header-menu"]'),
  ])
  previousFocus = null
}

function deactivateModal(restoreFocus = true) {
  if (!modalActive || typeof document === 'undefined') return
  modalActive = false
  document.removeEventListener('keydown', handleDocumentKeydown)
  unregisterModalLayer(modalToken)
  releaseBodyScrollLock(modalToken)
  restoreHostInert()
  if (restoreFocus) {
    void nextTick(focusReturnTarget)
  }
}

watch(
  () => route.fullPath,
  () => {
    void canonicalizePersonalSettingsRoute(router, route)
  },
  { immediate: true },
)

watch(
  [isOpen, section],
  ([open, currentSection], [wasOpen]) => {
    if (!open) {
      if (wasOpen) {
        deactivateModal()
        visitedSections.value = new Set()
        publicSettingsAttempted = false
        accountError.value = false
        publicSettingsError.value = false
      }
      return
    }

    markVisited(currentSection)
    revealActiveTab(currentSection)
    ensureSectionData(currentSection)
    if (!wasOpen) {
      void activateModal()
    }
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  deactivateModal(false)
})
</script>

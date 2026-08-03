<template>
  <Teleport to="body">
    <Transition name="account-panel">
      <div v-if="open" data-testid="sidebar-account-overlay">
        <button
          v-if="isMobile"
          type="button"
          class="account-panel-backdrop"
          :aria-label="t('accountDock.close')"
          tabindex="-1"
          @click="emit('close', true)"
        ></button>

        <section
          id="sidebar-account-panel"
          ref="panelRef"
          data-testid="sidebar-account-panel"
          class="account-panel"
          :class="{ 'account-panel--mobile': isMobile }"
          :style="panelStyle"
          role="dialog"
          :aria-modal="isMobile ? 'true' : undefined"
          aria-labelledby="sidebar-account-panel-title"
          tabindex="-1"
        >
          <div class="account-panel__handle lg:hidden" aria-hidden="true"></div>

          <header class="account-panel__identity">
            <div class="account-panel__avatar">
              <img
                v-if="summary.avatarUrl"
                :src="summary.avatarUrl"
                :alt="summary.displayName"
              >
              <span v-else>{{ summary.initials }}</span>
            </div>
            <div class="min-w-0 flex-1">
              <h2
                id="sidebar-account-panel-title"
                class="account-panel__name"
              >
                {{ summary.displayName }}
              </h2>
              <p class="account-panel__identity-meta">
                <span class="account-panel__balance-line">
                  <span>{{ t('accountDock.balanceShort') }}</span>
                  <CreditAmount
                    :value="summary.formattedAvailableBalance"
                    icon-size="xs"
                    :label="`${t('accountDock.availableBalance')} ${summary.formattedAvailableBalance}`"
                  />
                  <span aria-hidden="true">·</span>
                  <span class="truncate">{{ subscriptionStatusText }}</span>
                </span>
                <span
                  v-if="summary.frozenBalance > 0"
                  class="account-panel__frozen"
                >
                  <span aria-hidden="true">·</span>
                  <span>{{ t('accountDock.frozenBalance') }}</span>
                  <CreditAmount
                    :value="summary.formattedFrozenBalance"
                    icon-size="xs"
                    :label="`${t('accountDock.frozenBalance')} ${summary.formattedFrozenBalance}`"
                  />
                </span>
              </p>
            </div>
            <button
              type="button"
              class="account-panel__close"
              :aria-label="t('accountDock.close')"
              @click="emit('close', true)"
            >
              <Icon name="x" size="sm" aria-hidden="true" />
            </button>
          </header>

          <nav class="account-panel__section" :aria-label="t('accountDock.personalActions')">
            <button
              type="button"
              data-testid="account-open-profile"
              class="account-panel__row w-full"
              @click="emit('open-settings', 'account')"
            >
              <Icon name="user" size="sm" aria-hidden="true" />
              <span class="min-w-0 flex-1 truncate text-left">
                {{ t('accountDock.personalProfile') }}
              </span>
              <Icon name="chevronRight" size="xs" aria-hidden="true" />
            </button>

            <button
              type="button"
              data-testid="account-open-preferences"
              class="account-panel__row w-full"
              @click="emit('open-settings', 'general')"
            >
              <Icon name="slidersHorizontal" size="sm" aria-hidden="true" />
              <span class="min-w-0 flex-1 truncate text-left">
                {{ t('accountDock.personalPreferences') }}
              </span>
              <Icon name="chevronRight" size="xs" aria-hidden="true" />
            </button>

            <button
              v-if="showOnboarding"
              type="button"
              data-testid="account-admin-guide"
              class="account-panel__row w-full"
              @click="emit('replay')"
            >
              <Icon name="questionCircle" size="sm" aria-hidden="true" />
              <span class="min-w-0 flex-1 truncate text-left">
                {{ t('accountDock.adminGuide') }}
              </span>
            </button>
          </nav>

          <div class="account-panel__section account-panel__section--danger">
            <button
              type="button"
              data-testid="account-logout"
              class="account-panel__row account-panel__row--danger w-full"
              @click="emit('logout')"
            >
              <Icon name="logout" size="sm" aria-hidden="true" />
              <span class="min-w-0 flex-1 truncate text-left">{{ t('nav.logout') }}</span>
            </button>
          </div>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, toRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAnchoredOverlay } from '@/composables/useAnchoredOverlay'
import { acquireBodyScrollLock, releaseBodyScrollLock } from '@/utils/bodyScrollLock'
import {
  getVisibleFocusableElements,
  isTopModalLayer,
  registerModalLayer,
  unregisterModalLayer,
} from '@/utils/modalStack'
import CreditAmount from '@/components/common/CreditAmount.vue'
import Icon from '@/components/icons/Icon.vue'
import type { PersonalSettingsSection } from '@/navigation/personalSettingsRoute'
import type { AccountPanelSummary } from './accountPanelTypes'

const props = defineProps<{
  open: boolean
  anchorElement: HTMLElement | null
  summary: AccountPanelSummary
  showOnboarding: boolean
}>()

const emit = defineEmits<{
  close: [restoreFocus: boolean]
  logout: []
  replay: []
  'open-settings': [section: PersonalSettingsSection]
}>()

const { t } = useI18n()
const panelRef = ref<HTMLElement | null>(null)
const anchorRef = toRef(props, 'anchorElement')
const openRef = toRef(props, 'open')
const { desktopStyle, updatePosition } = useAnchoredOverlay(openRef, anchorRef, panelRef)
const isMobile = ref(false)
const scrollLockToken = Symbol('account-bottom-sheet')
const modalLayerToken = Symbol('account-panel-layer')
let mediaQuery: MediaQueryList | null = null
let shellElement: HTMLElement | null = null
let shellWasInert = false
let shellAriaHidden: string | null = null

const panelStyle = computed(() => isMobile.value ? undefined : desktopStyle.value)
const subscriptionStatusText = computed(() => {
  if (!props.summary.subscriptionsLoaded) {
    return t('accountDock.subscriptionLoading')
  }
  if (props.summary.activeSubscriptionCount > 0) {
    return t('accountDock.activeSubscriptions', {
      count: props.summary.activeSubscriptionCount,
    })
  }
  return t('accountDock.payAsYouGo')
})
function updateMediaQuery() {
  isMobile.value = mediaQuery?.matches ?? window.innerWidth < 1024
}

function getFocusableElements() {
  if (!panelRef.value) return []
  return getVisibleFocusableElements(panelRef.value)
}

function handleKeydown(event: KeyboardEvent) {
  if (!props.open) return
  if (!isTopModalLayer(modalLayerToken)) return
  if (document.querySelector('[data-announcement-modal="true"]')) return

  if (event.key === 'Escape') {
    event.preventDefault()
    emit('close', true)
    return
  }

  if (event.key !== 'Tab' || !isMobile.value) return

  const focusable = getFocusableElements()
  if (focusable.length === 0) {
    event.preventDefault()
    panelRef.value?.focus()
    return
  }

  const first = focusable[0]
  const last = focusable[focusable.length - 1]
  if (event.shiftKey && (document.activeElement === first || document.activeElement === panelRef.value)) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first.focus()
  }
}

function handlePointerDown(event: PointerEvent) {
  if (!props.open || isMobile.value) return
  if (!isTopModalLayer(modalLayerToken)) return
  const target = event.target as Node
  if (
    target instanceof Element
    && target.closest('[data-announcement-modal="true"]')
  ) {
    return
  }
  if (!panelRef.value?.contains(target) && !props.anchorElement?.contains(target)) {
    emit('close', false)
  }
}

function lockMobileBackground() {
  if (!isMobile.value || !props.open) return
  acquireBodyScrollLock(scrollLockToken)
  shellElement = document.querySelector<HTMLElement>('.app-layout')
  if (!shellElement) return

  shellWasInert = shellElement.inert
  shellAriaHidden = shellElement.getAttribute('aria-hidden')
  shellElement.inert = true
  shellElement.setAttribute('aria-hidden', 'true')
}

function unlockMobileBackground() {
  releaseBodyScrollLock(scrollLockToken)
  if (!shellElement) return

  shellElement.inert = shellWasInert
  if (shellAriaHidden === null) {
    shellElement.removeAttribute('aria-hidden')
  } else {
    shellElement.setAttribute('aria-hidden', shellAriaHidden)
  }
  shellElement = null
}

async function focusPanel() {
  await nextTick()
  updatePosition()
  await nextTick()
  panelRef.value?.focus({ preventScroll: true })
}

watch(
  [() => props.open, isMobile],
  ([open]) => {
    unlockMobileBackground()
    unregisterModalLayer(modalLayerToken)
    if (open) {
      registerModalLayer(modalLayerToken)
      lockMobileBackground()
      void focusPanel()
    }
  },
  { immediate: true },
)

onMounted(() => {
  mediaQuery = window.matchMedia('(max-width: 1023px)')
  updateMediaQuery()
  mediaQuery.addEventListener('change', updateMediaQuery)
  document.addEventListener('keydown', handleKeydown)
  document.addEventListener('pointerdown', handlePointerDown, true)
})

onBeforeUnmount(() => {
  mediaQuery?.removeEventListener('change', updateMediaQuery)
  document.removeEventListener('keydown', handleKeydown)
  document.removeEventListener('pointerdown', handlePointerDown, true)
  unregisterModalLayer(modalLayerToken)
  unlockMobileBackground()
})
</script>

<style scoped>
.account-panel-backdrop {
  position: fixed;
  inset: 0;
  z-index: 70;
  border: 0;
  background: rgb(15 23 42 / 0.32);
  backdrop-filter: blur(3px);
  -webkit-backdrop-filter: blur(3px);
}

.account-panel {
  position: fixed;
  z-index: 80;
  width: 248px;
  max-height: calc(100dvh - 16px);
  overflow: auto;
  padding: 6px 0;
  border: 0;
  border-radius: 16px;
  color: rgb(51 65 85);
  background: #fff;
  box-shadow:
    0 8px 12px rgb(0 0 0 / 0.08),
    0 0 1px rgb(0 0 0 / 0.62);
  outline: none;
  transform-origin: left bottom;
}

.account-panel--mobile {
  inset: auto 0 0 !important;
  width: 100%;
  max-height: min(86dvh, 760px);
  padding: 8px 10px calc(10px + env(safe-area-inset-bottom));
  border-top: 1px solid rgb(226 232 240 / 0.92);
  border-radius: 22px 22px 0 0;
  transform-origin: center bottom;
}

.account-panel__handle {
  width: 36px;
  height: 4px;
  margin: 2px auto 7px;
  border-radius: 999px;
  background: rgb(148 163 184 / 0.55);
}

.account-panel__identity {
  display: flex;
  min-height: 48px;
  align-items: center;
  gap: 10px;
  padding: 4px 10px 6px;
}

.account-panel__avatar {
  display: flex;
  width: 32px;
  height: 32px;
  flex: 0 0 32px;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border-radius: 999px;
  color: #1c1f23;
  background: #fce865;
  font-size: 0.75rem;
  font-weight: 700;
}

.account-panel__avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.account-panel__name {
  overflow: hidden;
  color: rgb(15 23 42);
  font-size: 0.875rem;
  font-weight: 600;
  line-height: 1.125rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.account-panel__identity-meta {
  display: flex;
  min-width: 0;
  overflow: hidden;
  align-items: center;
  gap: 3px;
  margin-top: 1px;
  color: rgb(100 116 139);
  font-size: 0.6875rem;
  font-weight: 400;
  line-height: 1rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.account-panel__balance-line,
.account-panel__frozen {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  gap: 3px;
}

.account-panel__frozen {
  flex: 0 0 auto;
  color: rgb(180 83 9);
}

.account-panel__close {
  display: inline-flex;
  width: 44px;
  height: 44px;
  flex: 0 0 44px;
  align-items: center;
  justify-content: center;
  border-radius: 10px;
  color: rgb(100 116 139);
}

.account-panel__close:hover {
  color: rgb(15 23 42);
  background: rgb(15 23 42 / 0.05);
}

.account-panel__section {
  margin-top: 4px;
  padding: 4px 6px 0;
  border-top: 1px solid rgb(226 232 240 / 0.8);
}

.account-panel__row {
  display: flex;
  min-height: 36px;
  align-items: center;
  gap: 10px;
  padding: 6px 8px;
  border-radius: 10px;
  color: rgb(51 65 85);
  font-size: 0.875rem;
  font-weight: 400;
  text-decoration: none;
  transition:
    color 150ms ease,
    background-color 150ms ease;
}

.account-panel__row:hover {
  color: rgb(15 23 42);
  background: rgb(15 23 42 / 0.045);
}

.account-panel__row:focus-visible,
.account-panel__close:focus-visible {
  outline: 2px solid var(--app-shell-sidebar-focus, rgb(0 132 255 / 0.5));
  outline-offset: 1px;
}

.account-panel__section--danger {
  margin-top: 4px;
}

.account-panel__row--danger {
  color: rgb(220 38 38);
}

.account-panel__row--danger:hover {
  color: rgb(185 28 28);
  background: rgb(254 226 226 / 0.72);
}

:global(html.dark .account-panel) {
  border-color: rgb(255 255 255 / 0.1);
  color: rgb(203 213 225);
  background: rgb(15 20 31 / 0.98);
  box-shadow:
    0 12px 32px rgb(0 0 0 / 0.42),
    0 2px 8px rgb(0 0 0 / 0.24);
}

:global(html.dark .account-panel__name) {
  color: #fff;
}

:global(html.dark .account-panel__identity-meta) {
  color: rgb(148 163 184);
}

:global(html.dark .account-panel__section) {
  border-color: rgb(255 255 255 / 0.09);
  background-color: transparent;
}

:global(html.dark .account-panel__row) {
  color: rgb(226 232 240);
}

:global(html.dark .account-panel__row:hover) {
  color: #fff;
  background: rgb(255 255 255 / 0.07);
}

:global(html.dark .account-panel__close) {
  color: rgb(148 163 184);
}

:global(html.dark .account-panel__close:hover) {
  color: #fff;
  background: rgb(255 255 255 / 0.08);
}

:global(html.dark .account-panel__row--danger) {
  color: rgb(248 113 113);
}

:global(html.dark .account-panel__row--danger:hover) {
  color: rgb(254 202 202);
  background: rgb(127 29 29 / 0.28);
}

.account-panel-enter-active,
.account-panel-leave-active {
  transition:
    opacity 160ms ease,
    transform 180ms cubic-bezier(0.2, 0.8, 0.2, 1);
}

.account-panel-enter-from,
.account-panel-leave-to {
  opacity: 0;
  transform: translateY(8px) scale(0.985);
}

@media (max-width: 1023px) {
  .account-panel__identity {
    min-height: 58px;
    gap: 11px;
    padding: 4px 8px 8px;
  }

  .account-panel__avatar {
    width: 42px;
    height: 42px;
    flex-basis: 42px;
    border-radius: 14px;
    font-size: 0.8125rem;
  }

  .account-panel__row {
    min-height: 44px;
    padding: 8px 11px;
    border-radius: 12px;
  }

  .account-panel-enter-from,
  .account-panel-leave-to {
    transform: translateY(24px);
  }
}

@media (min-width: 1024px) {
  .account-panel__close {
    display: none;
  }
}

@media (prefers-reduced-motion: reduce) {
  .account-panel-enter-active,
  .account-panel-leave-active,
  .account-panel__row {
    transition-duration: 0.01ms;
  }
}
</style>

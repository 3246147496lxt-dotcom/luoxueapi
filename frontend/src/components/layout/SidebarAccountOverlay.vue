<template>
  <Teleport to="body">
    <Transition
      name="account-panel"
      :duration="{ enter: 180, leave: 180 }"
    >
      <div
        v-if="open"
        class="sidebar-account-overlay"
        data-testid="sidebar-account-overlay"
      >
        <button
          v-if="usesMobileSheet"
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
          :class="{
            'account-panel--mobile': usesMobileSheet,
            'account-panel--personal': variant === 'personal',
          }"
          :style="panelStyle"
          role="dialog"
          :aria-modal="usesMobileSheet ? 'true' : undefined"
          aria-labelledby="sidebar-account-panel-title"
          tabindex="-1"
        >
          <div v-if="usesMobileSheet" class="account-panel__handle" aria-hidden="true"></div>

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
                <span v-if="variant === 'personal'" class="truncate">{{ planLabel }}</span>
                <template v-else>
                  <span v-if="context === 'chat'" class="truncate">
                    {{ summary.email }}
                  </span>
                  <span v-else class="account-panel__balance-line">
                    <span>{{ t('accountDock.balanceShort') }}</span>
                    <CreditAmount
                      :value="summary.formattedAvailableBalance"
                      icon-size="xs"
                      :label="`${t('accountDock.availableBalance')} ${summary.formattedAvailableBalance}`"
                    />
                    <span aria-hidden="true">·</span>
                    <span class="truncate">{{ planLabel }}</span>
                  </span>
                  <span
                    v-if="context !== 'chat' && summary.frozenBalance > 0"
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
                </template>
              </p>
            </div>
            <AccountMenuIcon
              v-if="variant === 'personal'"
              name="chevronRight"
              :size="16"
              class="account-panel__identity-chevron"
              aria-hidden="true"
            />
            <button
              type="button"
              class="account-panel__close"
              :aria-label="t('accountDock.close')"
              @click="emit('close', true)"
            >
              <Icon name="x" size="sm" aria-hidden="true" />
            </button>
          </header>

          <template v-if="variant === 'personal'">
            <div class="account-panel__divider" aria-hidden="true"></div>

            <nav
              class="account-panel__section account-panel__section--personal"
              :aria-label="t('accountDock.commonSettings')"
            >
              <button
                type="button"
                data-testid="account-open-preferences"
                class="account-panel__row account-panel__row--reserved w-full"
                @click="emit('open-settings', 'general')"
              >
                <AccountMenuIcon
                  name="face"
                  class="account-panel__row-icon"
                  aria-hidden="true"
                />
                <span class="min-w-0 flex-1 truncate text-left">
                  {{ t('accountDock.personalization') }}
                </span>
              </button>

              <button
                type="button"
                data-testid="account-open-profile"
                class="account-panel__row account-panel__row--reserved w-full"
                @click="emit('open-settings', 'account')"
              >
                <AccountMenuIcon
                  name="avatar"
                  class="account-panel__row-icon"
                  aria-hidden="true"
                />
                <span class="min-w-0 flex-1 truncate text-left">
                  {{ t('accountDock.personalProfile') }}
                </span>
              </button>

              <button
                type="button"
                data-testid="account-open-settings"
                class="account-panel__row account-panel__row--reserved w-full"
                @click="emit('open-settings', 'general')"
              >
                <AccountMenuIcon
                  name="settings"
                  class="account-panel__row-icon"
                  aria-hidden="true"
                />
                <span class="min-w-0 flex-1 truncate text-left">
                  {{ t('accountDock.settings') }}
                </span>
              </button>
            </nav>

            <div class="account-panel__divider" aria-hidden="true"></div>

            <nav
              class="account-panel__section account-panel__section--personal account-panel__section--support"
              :aria-label="t('accountDock.assistance')"
            >
              <a
                data-testid="account-help"
                class="account-panel__row w-full"
                :href="helpHref"
                target="_blank"
                rel="noopener noreferrer"
                @click="emit('close', false)"
              >
                <AccountMenuIcon
                  name="help"
                  class="account-panel__row-icon"
                  aria-hidden="true"
                />
                <span class="min-w-0 flex-1 truncate text-left">
                  {{ t('accountDock.help') }}
                </span>
                <AccountMenuIcon
                  name="chevronRight"
                  :size="16"
                  class="account-panel__row-chevron"
                  aria-hidden="true"
                />
              </a>

              <a
                v-if="workspaceTarget"
                data-testid="account-switch-workspace"
                class="account-panel__row account-panel__row--reserved w-full"
                :href="workspaceTarget.href"
              >
                <Icon
                  name="swap"
                  size="md"
                  class="account-panel__row-icon"
                  aria-hidden="true"
                />
                <span class="min-w-0 flex-1 truncate text-left">
                  {{ workspaceTarget.label }}
                </span>
              </a>

              <button
                type="button"
                data-testid="account-logout"
                class="account-panel__row account-panel__row--danger account-panel__row--reserved w-full"
                @click="emit('logout')"
              >
                <AccountMenuIcon
                  name="exit"
                  class="account-panel__row-icon"
                  aria-hidden="true"
                />
                <span class="min-w-0 flex-1 truncate text-left">{{ t('nav.logout') }}</span>
              </button>
            </nav>
          </template>

          <template v-else>
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

            <a
              v-if="workspaceTarget"
              data-testid="account-switch-workspace"
              class="account-panel__row w-full"
              :href="workspaceTarget.href"
            >
              <Icon name="swap" size="sm" aria-hidden="true" />
              <span class="min-w-0 flex-1 truncate text-left">
                {{ workspaceTarget.label }}
              </span>
              <Icon name="chevronRight" size="xs" aria-hidden="true" />
            </a>
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
          </template>
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
import AccountMenuIcon from './AccountMenuIcon.vue'
import type { AccountPanelSummary } from './accountPanelTypes'

const props = defineProps<{
  open: boolean
  anchorElement: HTMLElement | null
  summary: AccountPanelSummary
  showOnboarding: boolean
  context?: 'work' | 'chat'
  variant: 'personal' | 'admin'
  planLabel: string
  helpHref: string
  workspaceTarget?: {
    href: string
    label: string
  } | null
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
const usesMobileSheet = computed(() => isMobile.value && props.variant === 'admin')
const scrollLockToken = Symbol('account-bottom-sheet')
const modalLayerToken = Symbol('account-panel-layer')
let mediaQuery: MediaQueryList | null = null
let personalSidebarQuery: MediaQueryList | null = null
let shellElement: HTMLElement | null = null
let shellWasInert = false
let shellAriaHidden: string | null = null

const panelStyle = computed(() => usesMobileSheet.value ? undefined : desktopStyle.value)
function updateMediaQuery() {
  isMobile.value = mediaQuery?.matches ?? window.innerWidth < 1024
}

function closePersonalPanelAtSidebarBreakpoint(event: MediaQueryListEvent) {
  if (event.matches && props.open && props.variant === 'personal') {
    emit('close', false)
  }
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

  if (event.key !== 'Tab' || !usesMobileSheet.value) return

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
  if (!props.open || usesMobileSheet.value) return
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
  if (!usesMobileSheet.value || !props.open) return
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
  [() => props.open, usesMobileSheet],
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
  personalSidebarQuery = window.matchMedia('(max-width: 767px)')
  updateMediaQuery()
  mediaQuery.addEventListener('change', updateMediaQuery)
  personalSidebarQuery.addEventListener('change', closePersonalPanelAtSidebarBreakpoint)
  document.addEventListener('keydown', handleKeydown)
  document.addEventListener('pointerdown', handlePointerDown, true)
})

onBeforeUnmount(() => {
  mediaQuery?.removeEventListener('change', updateMediaQuery)
  personalSidebarQuery?.removeEventListener('change', closePersonalPanelAtSidebarBreakpoint)
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
  background: var(--workspace-overlay-backdrop);
  backdrop-filter: blur(3px);
  -webkit-backdrop-filter: blur(3px);
}

.sidebar-account-overlay {
  position: fixed;
  inset: 0;
  z-index: var(--workspace-layer-account-overlay);
  pointer-events: none;
}

.sidebar-account-overlay .account-panel,
.sidebar-account-overlay .account-panel-backdrop {
  pointer-events: auto;
}

.account-panel {
  position: fixed;
  z-index: 80;
  width: var(--workspace-popover-width);
  max-height: calc(100dvh - var(--workspace-space-4));
  overflow: auto;
  padding: var(--workspace-space-1-5) 0;
  border: 0;
  border-radius: var(--workspace-radius-card);
  color: var(--workspace-menu-text);
  background: var(--workspace-menu-surface);
  box-shadow: var(--workspace-menu-shadow);
  outline: none;
  transform-origin: left bottom;
}

.account-panel--mobile {
  inset: auto 0 0 !important;
  width: 100%;
  max-height: min(86dvh, 760px);
  padding: var(--workspace-space-2) var(--workspace-space-2-5) calc(var(--workspace-space-2-5) + env(safe-area-inset-bottom));
  border-top: 1px solid var(--workspace-menu-sheet-border);
  border-radius: var(--workspace-radius-mobile-sheet) var(--workspace-radius-mobile-sheet) 0 0;
  transform-origin: center bottom;
}

.account-panel__handle {
  width: var(--workspace-sheet-handle-width);
  height: var(--workspace-sheet-handle-height);
  margin:
    var(--workspace-space-0-5)
    auto
    var(--workspace-space-1-75);
  border-radius: var(--workspace-radius-pill);
  background: var(--workspace-menu-handle);
}

.account-panel__identity {
  display: flex;
  min-height: var(--workspace-menu-identity-height);
  align-items: center;
  gap: var(--workspace-space-2-5);
  padding:
    var(--workspace-space-1)
    var(--workspace-space-2-5)
    var(--workspace-space-1-5);
}

.account-panel__avatar {
  display: flex;
  width: var(--workspace-avatar-size-md);
  height: var(--workspace-avatar-size-md);
  flex: 0 0 var(--workspace-avatar-size-md);
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border-radius: var(--workspace-radius-pill);
  color: var(--workspace-identity-avatar-text);
  background: var(--workspace-identity-avatar-surface);
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
  color: var(--workspace-menu-text-strong);
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
  gap: var(--workspace-space-0-75);
  margin-top: var(--workspace-space-0-25);
  color: var(--workspace-menu-text-muted);
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
  gap: var(--workspace-space-0-75);
}

.account-panel__frozen {
  flex: 0 0 auto;
  color: var(--workspace-menu-frozen);
}

.account-panel__close {
  display: inline-flex;
  width: var(--workspace-menu-row-height-touch);
  height: var(--workspace-menu-row-height-touch);
  flex: 0 0 var(--workspace-menu-row-height-touch);
  align-items: center;
  justify-content: center;
  border-radius: var(--workspace-radius-button);
  color: var(--workspace-menu-text-muted);
}

.account-panel__close:hover {
  color: var(--workspace-menu-text-strong);
  background: var(--workspace-menu-close-hover);
}

.account-panel__section {
  margin-top: var(--workspace-space-1);
  padding: var(--workspace-space-1) var(--workspace-space-1-5) 0;
  border-top: 1px solid var(--workspace-menu-divider);
}

.account-panel__row {
  display: flex;
  min-height: var(--workspace-menu-row-height);
  align-items: center;
  gap: var(--workspace-space-2-5);
  padding: var(--workspace-space-1-5) var(--workspace-space-2);
  border-radius: var(--workspace-radius-button);
  color: var(--workspace-menu-row-text);
  font-size: 0.875rem;
  font-weight: 400;
  text-decoration: none;
  transition:
    color 150ms ease,
    background-color 150ms ease;
}

.account-panel__row:hover {
  color: var(--workspace-menu-text-strong);
  background: var(--workspace-menu-hover);
}

.account-panel__row:focus-visible,
.account-panel__close:focus-visible {
  outline: 2px solid var(--workspace-menu-focus);
  outline-offset: var(--workspace-space-0-25);
}

.account-panel__section--danger {
  margin-top: var(--workspace-space-1);
}

.account-panel__row--danger {
  color: var(--workspace-menu-danger);
}

.account-panel__row--danger:hover {
  color: var(--workspace-menu-danger-hover);
  background: var(--workspace-menu-danger-surface);
}

.account-panel--personal {
  padding: var(--workspace-space-2-5) 0;
  border: 1px solid var(--workspace-popover-border);
  border-radius: var(--workspace-radius-popover);
  color: var(--workspace-popover-text);
  background: var(--workspace-popover-surface);
  font-weight: var(--workspace-type-body-weight);
  letter-spacing: normal;
  box-shadow: var(--workspace-popover-shadow);
}

.account-panel--personal .account-panel__identity {
  min-height: var(--workspace-popover-identity-height);
  gap: var(--workspace-space-2);
  margin: 0 var(--workspace-space-2-5);
  padding: var(--workspace-space-1-5) var(--workspace-space-2-5);
  border-radius: var(--workspace-radius-input);
  background: var(--workspace-popover-identity-surface);
}

.account-panel--personal .account-panel__avatar {
  width: var(--workspace-avatar-size-sm);
  height: var(--workspace-avatar-size-sm);
  flex-basis: var(--workspace-avatar-size-sm);
  border-radius: var(--workspace-radius-pill);
  color: var(--workspace-popover-avatar-text);
  background: var(--workspace-popover-avatar-surface);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
}

.account-panel--personal .account-panel__name {
  color: var(--workspace-popover-text);
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
  line-height: 1.25rem;
  letter-spacing: normal;
}

.account-panel--personal .account-panel__identity-meta {
  margin-top: 0;
  color: var(--workspace-popover-text-secondary);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  line-height: 1rem;
}

.account-panel__divider {
  width: auto;
  height: var(--workspace-space-0-25);
  flex: 0 0 var(--workspace-space-0-25);
  margin: var(--workspace-space-2) var(--workspace-space-4);
  background: var(--workspace-popover-divider);
}

.account-panel--personal .account-panel__section {
  margin: 0;
  padding: 0;
  border: 0;
  background: transparent;
}

.account-panel--personal .account-panel__row {
  width: calc(100% - var(--workspace-space-5));
  min-height: var(--workspace-menu-row-height);
  gap: var(--workspace-space-1-5);
  margin: 0 var(--workspace-space-2-5);
  padding: var(--workspace-space-1-5) var(--workspace-space-2-5);
  border-radius: var(--workspace-radius-input);
  color: var(--workspace-popover-text);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
  line-height: 1.25rem;
  letter-spacing: normal;
}

.account-panel--personal .account-panel__row--reserved {
  padding-right: var(--workspace-space-8);
}

.account-panel__row-icon {
  flex: 0 0 auto;
  color: var(--workspace-popover-text);
}

.account-panel__identity-chevron,
.account-panel__row-chevron {
  flex: 0 0 auto;
  color: var(--workspace-popover-text);
}

.account-panel__identity-chevron {
  margin-right: calc(-1 * var(--workspace-space-0-25));
}

.account-panel--personal .account-panel__row:hover,
.account-panel--personal .account-panel__close:hover,
.account-panel--personal .account-panel__row--danger:hover {
  color: var(--workspace-popover-text);
  background: var(--workspace-popover-hover);
}

.account-panel--personal .account-panel__row:focus-visible,
.account-panel--personal .account-panel__close:focus-visible {
  outline-color: var(--workspace-popover-focus);
}

.account-panel--personal .account-panel__row--danger {
  color: var(--workspace-popover-text);
}

:global(html.dark .account-panel.account-panel--personal) {
  border-width: 0;
}

.account-panel-enter-active .account-panel,
.account-panel-leave-active .account-panel {
  transition:
    opacity 160ms ease,
    transform 180ms cubic-bezier(0.2, 0.8, 0.2, 1);
}

.account-panel-enter-active .account-panel-backdrop,
.account-panel-leave-active .account-panel-backdrop {
  transition: opacity 160ms ease;
}

.account-panel-enter-from .account-panel,
.account-panel-leave-to .account-panel {
  opacity: 0;
  transform: translateY(var(--workspace-space-2)) scale(0.985);
}

.account-panel-enter-from .account-panel-backdrop,
.account-panel-leave-to .account-panel-backdrop {
  opacity: 0;
}

@media (max-width: 1023px) {
  .account-panel--mobile .account-panel__identity {
    min-height: var(--workspace-sheet-identity-height);
    gap: var(--workspace-space-2-75);
    padding: var(--workspace-space-1) var(--workspace-space-2) var(--workspace-space-2);
  }

  .account-panel--mobile .account-panel__avatar {
    width: var(--workspace-avatar-size-sheet);
    height: var(--workspace-avatar-size-sheet);
    flex-basis: var(--workspace-avatar-size-sheet);
    border-radius: var(--workspace-radius-control-lg);
    font-size: 0.8125rem;
  }

  .account-panel--mobile .account-panel__row {
    min-height: var(--workspace-menu-row-height-touch);
    padding: var(--workspace-space-2) var(--workspace-space-2-75);
    border-radius: var(--workspace-radius-input);
  }

  .account-panel--mobile .account-panel__identity-chevron {
    display: none;
  }

  .account-panel-enter-from .account-panel,
  .account-panel-leave-to .account-panel {
    transform: translateY(var(--workspace-space-6));
  }
}

.account-panel--personal .account-panel__close {
  display: none;
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

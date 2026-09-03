<template>
  <header ref="rootRef" class="project-detail-header">
    <div class="project-detail-header__mode-switch">
      <AppModeSwitch
        active-mode="chat"
        :chat-label="t('projects.chatMode')"
        :work-label="t('projects.workMode')"
        @change="handleModeChange"
      />
    </div>

    <div class="project-detail-header__mobile-bar">
      <span class="project-detail-header__mobile-leading-placeholder" aria-hidden="true" />

      <strong class="project-detail-header__mobile-title" :title="project.name">
        {{ project.name }}
      </strong>

      <button
        ref="mobileMenuTriggerRef"
        type="button"
        class="project-detail-header__icon-button project-detail-header__mobile-menu-trigger"
        :aria-label="t('projects.projectMenu')"
        aria-haspopup="menu"
        :aria-controls="menuId"
        :aria-expanded="menuOpen"
        @click="toggleMenu"
        @keydown.down.prevent="openMenu($event, 'first')"
        @keydown.up.prevent="openMenu($event, 'last')"
      >
        <Icon name="more" size="md" aria-hidden="true" />
      </button>
    </div>

    <div class="project-detail-header__identity-row">
      <div class="project-detail-header__identity">
        <span class="project-detail-header__emoji" aria-hidden="true">
          <Icon v-if="projectUsesDefaultIcon" name="chatSidebarProjects" size="lg" />
          <template v-else>{{ projectEmoji }}</template>
        </span>
        <h1 class="project-detail-header__title" :title="project.name">{{ project.name }}</h1>
      </div>

      <div class="project-detail-header__actions">
        <button
          type="button"
          class="project-detail-header__share"
          @click="emit('share')"
        >
          <Icon name="chatHistoryShare" size="sm" aria-hidden="true" />
          <span>{{ t('projects.shareAction') }}</span>
        </button>

        <button
          ref="desktopMenuTriggerRef"
          type="button"
          class="project-detail-header__icon-button"
          :aria-label="t('projects.projectMenu')"
          aria-haspopup="menu"
          :aria-controls="menuId"
          :aria-expanded="menuOpen"
          @click="toggleMenu"
          @keydown.down.prevent="openMenu($event, 'first')"
          @keydown.up.prevent="openMenu($event, 'last')"
        >
          <Icon name="more" size="md" aria-hidden="true" />
        </button>
      </div>
    </div>

    <div
      v-if="menuOpen"
      :id="menuId"
      ref="menuSurfaceRef"
      class="project-detail-header__menu"
      role="menu"
      :aria-label="t('projects.projectMenu')"
      @keydown="handleMenuKeydown"
    >
      <button type="button" role="menuitem" @click="handleSettings">
        <Icon name="cog" size="sm" aria-hidden="true" />
        <span>{{ t('projects.projectSettings') }}</span>
      </button>
      <button
        type="button"
        role="menuitem"
        class="project-detail-header__menu-delete"
        @click="handleDelete"
      >
        <Icon name="trash" size="sm" aria-hidden="true" />
        <span>{{ t('projects.delete') }}</span>
      </button>
    </div>

    <form class="project-detail-header__composer" @submit.prevent="submitPrompt">
      <button
        type="button"
        class="project-detail-header__composer-control project-detail-header__add-chat"
        :aria-label="t('projects.addChat')"
        @click="emit('add-chat')"
      >
        <Icon name="chatPlus" size="md" aria-hidden="true" />
      </button>

      <textarea
        class="project-detail-header__prompt"
        rows="1"
        :value="prompt"
        :placeholder="composerPlaceholder"
        :aria-label="t('projects.startChatPlaceholder')"
        @input="updatePrompt"
        @keydown="handlePromptKeydown"
      />

      <div class="project-detail-header__composer-trailing">
        <span class="project-detail-header__model" aria-hidden="true">Pro</span>
        <span class="project-detail-header__microphone" aria-hidden="true">
          <Icon name="chatMicrophone" size="md" />
        </span>
        <button
          type="submit"
          class="project-detail-header__submit"
          :aria-label="t('projects.sendMessage')"
        >
          <Icon
            :name="prompt.trim() ? 'chatSend' : 'chatVoiceMode'"
            size="md"
            aria-hidden="true"
          />
        </button>
      </div>
    </form>

    <div class="project-detail-header__tabs" role="tablist" :aria-label="t('projects.tabsLabel')">
      <button
        id="project-chats-tab"
        ref="chatsTabRef"
        type="button"
        role="tab"
        aria-controls="project-chats-panel"
        :aria-selected="activeTab === 'chats'"
        :tabindex="activeTab === 'chats' ? 0 : -1"
        :class="{ 'is-active': activeTab === 'chats' }"
        @click="selectTab('chats')"
        @keydown="handleTabKeydown($event, 'chats')"
      >
        {{ t('projects.chatsTab') }}
      </button>
      <button
        id="project-sources-tab"
        ref="sourcesTabRef"
        type="button"
        role="tab"
        aria-controls="project-sources-panel"
        :aria-selected="activeTab === 'sources'"
        :tabindex="activeTab === 'sources' ? 0 : -1"
        :class="{ 'is-active': activeTab === 'sources' }"
        @click="selectTab('sources')"
        @keydown="handleTabKeydown($event, 'sources')"
      >
        {{ t('projects.sourcesTab') }}
      </button>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import AppModeSwitch from '@/components/layout/AppModeSwitch.vue'
import type { Project } from '@/types/projects'

type ProjectTab = 'chats' | 'sources'
type MenuFocusEdge = 'first' | 'last'

const props = defineProps<{
  project: Project
  activeTab: ProjectTab
  prompt: string
}>()

const emit = defineEmits<{
  (event: 'update:activeTab', value: ProjectTab): void
  (event: 'update:prompt', value: string): void
  (event: 'submit', prompt: string): void
  (event: 'add-chat'): void
  (event: 'settings'): void
  (event: 'share'): void
  (event: 'delete'): void
  (event: 'back'): void
  (event: 'work'): void
}>()

const { t } = useI18n()
const rootRef = ref<HTMLElement | null>(null)
const menuSurfaceRef = ref<HTMLElement | null>(null)
const desktopMenuTriggerRef = ref<HTMLButtonElement | null>(null)
const mobileMenuTriggerRef = ref<HTMLButtonElement | null>(null)
const chatsTabRef = ref<HTMLButtonElement | null>(null)
const sourcesTabRef = ref<HTMLButtonElement | null>(null)
const lastMenuTriggerRef = ref<HTMLButtonElement | null>(null)
const menuOpen = ref(false)
const menuId = `project-detail-actions-${useId()}`

const projectEmoji = computed(() => {
  const icon = props.project.icon?.trim()
  return icon && icon !== 'folder' ? icon : '📁'
})
const projectUsesDefaultIcon = computed(() => {
  const icon = props.project.icon?.trim()
  return !icon || icon === 'folder' || icon === '📁'
})

const composerPlaceholder = computed(
  () => t('projects.newChatInProject', { name: props.project.name })
)

function handleModeChange(mode: 'chat' | 'work'): void {
  if (mode === 'work') emit('work')
}

function getMenuItems(): HTMLButtonElement[] {
  if (!menuSurfaceRef.value) return []
  return Array.from(menuSurfaceRef.value.querySelectorAll<HTMLButtonElement>('[role="menuitem"]'))
}

async function openMenu(event: MouseEvent | KeyboardEvent, edge: MenuFocusEdge): Promise<void> {
  if (event.currentTarget instanceof HTMLButtonElement) {
    lastMenuTriggerRef.value = event.currentTarget
  }

  menuOpen.value = true
  await nextTick()

  const items = getMenuItems()
  const target = edge === 'last' ? items.at(-1) : items[0]
  target?.focus()
}

function toggleMenu(event: MouseEvent): void {
  if (menuOpen.value) {
    closeMenu(true)
    return
  }

  void openMenu(event, 'first')
}

function closeMenu(restoreFocus = false): void {
  const trigger = lastMenuTriggerRef.value
  menuOpen.value = false

  if (restoreFocus) {
    void nextTick(() => trigger?.focus())
  }
}

function handleMenuKeydown(event: KeyboardEvent): void {
  const items = getMenuItems()
  if (!items.length) return

  const currentIndex = items.indexOf(document.activeElement as HTMLButtonElement)

  if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
    event.preventDefault()
    const direction = event.key === 'ArrowDown' ? 1 : -1
    const nextIndex = (currentIndex + direction + items.length) % items.length
    items[nextIndex]?.focus()
    return
  }

  if (event.key === 'Home' || event.key === 'End') {
    event.preventDefault()
    items[event.key === 'Home' ? 0 : items.length - 1]?.focus()
    return
  }

  if (event.key === 'Tab') closeMenu()
}

function handleDocumentPointerDown(event: PointerEvent): void {
  if (!menuOpen.value || !(event.target instanceof Node)) return

  const clickedMenu = menuSurfaceRef.value?.contains(event.target)
  const clickedDesktopTrigger = desktopMenuTriggerRef.value?.contains(event.target)
  const clickedMobileTrigger = mobileMenuTriggerRef.value?.contains(event.target)

  if (!clickedMenu && !clickedDesktopTrigger && !clickedMobileTrigger) closeMenu()
}

function handleDocumentKeydown(event: KeyboardEvent): void {
  if (event.key !== 'Escape' || !menuOpen.value) return
  event.preventDefault()
  closeMenu(true)
}

function handleSettings(): void {
  closeMenu()
  emit('settings')
}

function handleDelete(): void {
  closeMenu()
  emit('delete')
}

function updatePrompt(event: Event): void {
  if (event.target instanceof HTMLTextAreaElement) {
    emit('update:prompt', event.target.value)
  }
}

function submitPrompt(): void {
  emit('submit', props.prompt)
}

function handlePromptKeydown(event: KeyboardEvent): void {
  if (event.key !== 'Enter' || event.shiftKey || event.isComposing) return
  event.preventDefault()
  submitPrompt()
}

function selectTab(tab: ProjectTab, focus = false): void {
  emit('update:activeTab', tab)
  if (focus) {
    void nextTick(() => (tab === 'chats' ? chatsTabRef.value : sourcesTabRef.value)?.focus())
  }
}

function handleTabKeydown(event: KeyboardEvent, tab: ProjectTab): void {
  if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return
  event.preventDefault()

  if (event.key === 'Home') {
    selectTab('chats', true)
    return
  }

  if (event.key === 'End') {
    selectTab('sources', true)
    return
  }

  selectTab(tab === 'chats' ? 'sources' : 'chats', true)
}

watch(() => props.project.id, () => closeMenu())

onMounted(() => {
  document.addEventListener('pointerdown', handleDocumentPointerDown)
  document.addEventListener('keydown', handleDocumentKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handleDocumentPointerDown)
  document.removeEventListener('keydown', handleDocumentKeydown)
})
</script>

<style scoped>
.project-detail-header {
  --project-page: #ffffff;
  --project-raised: #f4f4f4;
  --project-hover: #ececec;
  --project-text: #0d0d0d;
  --project-muted: #5d5d5d;
  --project-faint: #7d7d7d;
  --project-divider: rgb(0 0 0 / 10%);
  --project-ring: rgb(0 0 0 / 12%);
  --project-danger: #c93434;
  --project-submit: #3267d6;
  position: relative;
  box-sizing: border-box;
  width: 100%;
  max-width: 800px;
  margin-inline: auto;
  padding: 120px 16px 0;
  color: var(--project-text);
}

:global(html.dark .project-detail-header) {
  --project-page: #000000;
  --project-raised: #212121;
  --project-hover: #2f2f2f;
  --project-text: #f2f2f2;
  --project-muted: #b4b4b4;
  --project-faint: #8f8f8f;
  --project-divider: rgb(255 255 255 / 10%);
  --project-ring: rgb(255 255 255 / 15%);
  --project-danger: #ff6b6b;
  --project-submit: #3f75dd;
}

.project-detail-header button,
.project-detail-header textarea {
  font: inherit;
}

.project-detail-header button {
  color: inherit;
}

.project-detail-header__mobile-bar {
  display: none;
}

.project-detail-header__mode-switch {
  position: absolute;
  z-index: 5;
  top: 8px;
  left: 50%;
  width: 218px;
  height: 38px;
  transform: translateX(-50%);
}

.project-detail-header__mode-switch :deep(.app-mode-switch) {
  box-sizing: border-box;
  width: 100%;
  height: 100%;
  flex: 0 0 auto;
  padding: 0;
}

.project-detail-header__mode-switch :deep(.app-mode-switch)::after {
  display: none;
}

.project-detail-header__mode-switch :deep(.app-mode-switch__control) {
  box-sizing: border-box;
  height: 100%;
}

.project-detail-header__mode-switch :deep(.app-mode-switch__option) {
  min-height: 30px;
}

.project-detail-header__identity-row {
  display: flex;
  min-height: 36px;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
}

.project-detail-header__identity {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 10px;
}

.project-detail-header__emoji {
  display: grid;
  width: 30px;
  height: 34px;
  flex: 0 0 auto;
  place-items: center;
  font-size: 28px;
  line-height: 34px;
}

.project-detail-header__emoji svg {
  width: 27px;
  height: 27px;
}

.project-detail-header__title {
  min-width: 0;
  margin: 0;
  overflow: hidden;
  font-size: 28px;
  font-weight: 500;
  line-height: 34px;
  letter-spacing: -0.02em;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-detail-header__actions {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 8px;
}

.project-detail-header__share,
.project-detail-header__icon-button,
.project-detail-header__composer-control,
.project-detail-header__submit,
.project-detail-header__tabs button,
.project-detail-header__menu button {
  border: 0;
  cursor: pointer;
}

.project-detail-header__share,
.project-detail-header__icon-button,
.project-detail-header__composer-control {
  background: transparent;
  transition: background-color 140ms ease;
}

.project-detail-header__share {
  display: inline-flex;
  width: 78px;
  height: 36px;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 0 12px;
  border: 1px solid var(--project-divider);
  border-radius: 9999px;
  font-size: 14px;
  font-weight: 500;
  line-height: 20px;
  white-space: nowrap;
}

:global(html.dark .project-detail-header__share) {
  background: var(--project-raised);
}

.project-detail-header__icon-button,
.project-detail-header__composer-control {
  display: inline-flex;
  width: 36px;
  height: 36px;
  flex: 0 0 36px;
  align-items: center;
  justify-content: center;
  padding: 0;
  border-radius: 9999px;
}

.project-detail-header__share:hover,
.project-detail-header__icon-button:hover,
.project-detail-header__composer-control:hover {
  background: var(--project-hover);
}

.project-detail-header__menu {
  position: absolute;
  z-index: 50;
  top: 164px;
  right: 16px;
  display: grid;
  width: 184px;
  padding: 6px;
  border: 1px solid var(--project-divider);
  border-radius: 12px;
  background: var(--project-raised);
  box-shadow: 0 12px 28px rgb(0 0 0 / 18%);
}

.project-detail-header__menu button {
  display: flex;
  width: 100%;
  min-height: 36px;
  align-items: center;
  gap: 10px;
  padding: 7px 10px;
  border-radius: 8px;
  background: transparent;
  font-size: 14px;
  line-height: 20px;
  text-align: left;
}

.project-detail-header__menu button:hover,
.project-detail-header__menu button:focus-visible {
  background: var(--project-hover);
}

.project-detail-header__menu-delete {
  color: var(--project-danger) !important;
}

.project-detail-header__composer {
  display: flex;
  box-sizing: border-box;
  width: 100%;
  height: 52px;
  align-items: center;
  gap: 6px;
  margin-top: 20px;
  padding: 8px;
  border-radius: 28px;
  background: var(--project-raised);
}

.project-detail-header__add-chat {
  color: var(--project-muted);
}

.project-detail-header__prompt {
  min-width: 0;
  height: 32px;
  flex: 1 1 auto;
  resize: none;
  overflow: auto;
  padding: 6px 4px;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--project-text);
  font-size: 15px;
  line-height: 20px;
  scrollbar-width: none;
}

.project-detail-header__prompt::-webkit-scrollbar {
  display: none;
}

.project-detail-header__prompt::placeholder {
  color: var(--project-faint);
  opacity: 1;
}

.project-detail-header__composer-trailing {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 6px;
}

.project-detail-header__model {
  color: var(--project-muted);
  font-size: 12px;
  font-weight: 500;
  line-height: 18px;
}

.project-detail-header__microphone {
  display: inline-flex;
  width: 30px;
  height: 30px;
  align-items: center;
  justify-content: center;
  color: var(--project-muted);
}

.project-detail-header__submit {
  display: inline-flex;
  width: 36px;
  height: 36px;
  flex: 0 0 36px;
  align-items: center;
  justify-content: center;
  padding: 0;
  border-radius: 9999px;
  background: var(--project-submit);
  color: #ffffff !important;
  transition: filter 140ms ease, transform 140ms ease;
}

.project-detail-header__submit:hover {
  filter: brightness(1.08);
}

.project-detail-header__submit:active {
  transform: scale(0.96);
}

.project-detail-header__tabs {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 30px;
}

.project-detail-header__tabs button {
  min-width: 64px;
  height: 38px;
  padding: 0 14px;
  border-radius: 9999px;
  background: transparent;
  color: var(--project-muted);
  font-size: 14px;
  font-weight: 500;
  line-height: 20px;
  transition: background-color 140ms ease, color 140ms ease, box-shadow 140ms ease;
}

.project-detail-header__tabs button:hover {
  background: var(--project-hover);
  color: var(--project-text);
}

.project-detail-header__tabs button.is-active {
  background: var(--project-raised);
  box-shadow: inset 0 0 0 1px var(--project-ring);
  color: var(--project-text);
}

.project-detail-header button:focus-visible,
.project-detail-header textarea:focus-visible {
  outline: 2px solid currentColor;
  outline-offset: 2px;
}

@media (max-width: 767px) {
  .project-detail-header {
    padding: 0 16px;
  }

  .project-detail-header__mode-switch {
    top: 26px;
    width: 112px;
    height: 26px;
  }

  .project-detail-header__mode-switch :deep(.app-mode-switch__control) {
    gap: 1px;
    padding: 1px;
    border-radius: 7px;
  }

  .project-detail-header__mode-switch :deep(.app-mode-switch__option) {
    min-height: 22px;
    padding: 0 4px;
    border-radius: 5px;
    font-size: 10px;
    line-height: 1;
  }

  .project-detail-header__identity-row {
    display: none;
  }

  .project-detail-header__mobile-bar {
    position: relative;
    display: grid;
    height: 62px;
    grid-template-columns: 44px minmax(0, 1fr) 44px;
    align-items: center;
    margin-inline: -4px;
  }

  .project-detail-header__mobile-leading-placeholder,
  .project-detail-header__mobile-menu-trigger {
    width: 40px;
    height: 40px;
  }

  .project-detail-header__mobile-menu-trigger {
    justify-self: end;
  }

  .project-detail-header__mobile-title {
    min-width: 0;
    align-self: start;
    margin-top: 4px;
    overflow: hidden;
    font-size: 16px;
    font-weight: 500;
    line-height: 22px;
    text-align: center;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .project-detail-header__menu {
    position: fixed;
    top: 64px;
    right: max(12px, env(safe-area-inset-right));
  }

  .project-detail-header__tabs {
    margin-top: 12px;
  }

  .project-detail-header__composer {
    position: fixed;
    z-index: 40;
    right: max(16px, env(safe-area-inset-right));
    bottom: max(12px, env(safe-area-inset-bottom));
    left: max(16px, env(safe-area-inset-left));
    display: grid;
    width: auto;
    height: auto;
    min-height: 88px;
    grid-template-columns: 40px minmax(0, 1fr) auto;
    grid-template-rows: minmax(24px, auto) 40px;
    gap: 2px 8px;
    margin: 0;
    padding: 8px;
    border-radius: 24px;
  }

  .project-detail-header__prompt {
    width: 100%;
    height: auto;
    min-height: 24px;
    max-height: 88px;
    grid-column: 1 / -1;
    grid-row: 1;
    padding: 2px 6px;
  }

  .project-detail-header__add-chat {
    width: 40px;
    height: 40px;
    grid-column: 1;
    grid-row: 2;
  }

  .project-detail-header__composer-trailing {
    grid-column: 3;
    grid-row: 2;
    justify-self: end;
  }

  .project-detail-header__microphone {
    width: 36px;
    height: 36px;
  }

  .project-detail-header__submit {
    width: 40px;
    height: 40px;
    flex-basis: 40px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .project-detail-header *,
  .project-detail-header *::before,
  .project-detail-header *::after {
    scroll-behavior: auto !important;
    transition-duration: 0.01ms !important;
  }
}
</style>

<template>
  <section
    id="project-chats-panel"
    ref="rootRef"
    class="project-chats-panel"
    data-ui-component="project-chats-panel"
    role="tabpanel"
    aria-labelledby="project-chats-tab"
    :aria-label="t('projects.chats')"
    tabindex="-1"
  >
    <Transition name="project-picker">
      <div
        v-if="pickerOpen"
        :id="pickerId"
        ref="pickerRef"
        class="project-chats-panel__picker"
        role="menu"
        :aria-label="t('projects.addChat')"
        tabindex="-1"
        @keydown="handlePickerKeydown"
      >
        <p class="project-chats-panel__picker-title">{{ t('projects.addChat') }}</p>
        <div v-if="availableConversations.length" class="project-chats-panel__picker-list">
          <button
            v-for="conversation in availableConversations"
            :key="conversation.id"
            ref="pickerItemRefs"
            type="button"
            role="menuitem"
            @click="addConversation(conversation.id)"
          >
            <span>{{ conversation.title }}</span>
            <Icon name="plus" size="xs" aria-hidden="true" />
          </button>
        </div>
        <p v-else class="project-chats-panel__picker-empty" role="status">
          {{ t('projects.noAvailableChats') }}
        </p>
      </div>
    </Transition>

    <ul v-if="conversations.length" class="project-chats-panel__list" role="list">
      <li
        v-for="conversation in conversations"
        :key="conversation.id"
        class="project-chat-row"
        :class="{ 'project-chat-row--menu-open': openMenuId === conversation.id }"
      >
        <button
          type="button"
          class="project-chat-row__open"
          @click="emit('open', conversation.id)"
        >
          <span class="project-chat-row__copy">
            <strong>{{ conversation.title }}</strong>
            <time
              v-if="formatConversationDate(conversation)"
              :datetime="conversationDateTime(conversation)"
            >
              {{ formatConversationDate(conversation) }}
            </time>
          </span>
        </button>

        <div class="project-chat-row__menu-wrap">
          <button
            type="button"
            class="project-chat-row__more"
            :aria-label="t('projects.conversationActions', { title: conversation.title })"
            aria-haspopup="menu"
            :aria-expanded="openMenuId === conversation.id"
            :aria-controls="conversationMenuId(conversation.id)"
            @click="toggleConversationMenu(conversation.id, $event)"
            @keydown.down.prevent="openConversationMenu(conversation.id, $event)"
            @keydown.up.prevent="openConversationMenu(conversation.id, $event)"
          >
            <Icon name="more" size="sm" aria-hidden="true" />
          </button>

          <Transition name="project-row-menu">
            <div
              v-if="openMenuId === conversation.id"
              :id="conversationMenuId(conversation.id)"
              ref="rowMenuRef"
              class="project-row-menu"
              role="menu"
              @keydown.esc.prevent.stop="closeConversationMenu(true)"
            >
              <button
                type="button"
                role="menuitem"
                @click="removeConversation(conversation.id)"
              >
                <Icon name="x" size="sm" aria-hidden="true" />
                <span>{{ t('projects.removeChat') }}</span>
              </button>
            </div>
          </Transition>
        </div>
      </li>
    </ul>

    <div v-else class="project-chats-panel__empty" role="status">
      <Icon name="chat" size="md" aria-hidden="true" />
      <p>{{ t('projects.noChats') }}</p>
    </div>
  </section>
</template>

<script setup lang="ts">
import {
  computed,
  getCurrentInstance,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
} from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { ProjectConversation } from '@/types/projects'

const props = defineProps<{
  conversations: ProjectConversation[]
  availableConversations: ProjectConversation[]
  pickerOpen: boolean
}>()

const emit = defineEmits<{
  open: [id: string]
  remove: [id: string]
  add: [id: string]
  'update:pickerOpen': [value: boolean]
}>()

const { locale, t } = useI18n()
const instanceId = `project-chats-${getCurrentInstance()?.uid ?? 'panel'}`
const pickerId = `${instanceId}-picker`
const rootRef = ref<HTMLElement | null>(null)
const pickerRef = ref<HTMLElement | null>(null)
const pickerItemRefs = ref<HTMLButtonElement[]>([])
const rowMenuRef = ref<HTMLElement | null>(null)
const openMenuId = ref<string | null>(null)
let activeMenuTrigger: HTMLButtonElement | null = null
let pickerReturnFocus: HTMLElement | null = null

const dateFormatter = computed(() => new Intl.DateTimeFormat(locale.value, {
  month: 'short',
  day: 'numeric',
}))

function conversationDate(conversation: ProjectConversation): Date | null {
  const value = conversation.updatedAt ?? conversation.createdAt
  if (!value) return null
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? null : date
}

function conversationDateTime(conversation: ProjectConversation): string | undefined {
  return conversationDate(conversation)?.toISOString()
}

function formatConversationDate(conversation: ProjectConversation): string {
  const date = conversationDate(conversation)
  return date ? dateFormatter.value.format(date) : ''
}

function safeId(value: string): string {
  return value.replace(/[^a-zA-Z0-9_-]/g, '-')
}

function conversationMenuId(id: string): string {
  return `${instanceId}-menu-${safeId(id)}`
}

async function openConversationMenu(id: string, event: Event): Promise<void> {
  activeMenuTrigger = event.currentTarget as HTMLButtonElement
  openMenuId.value = id
  await nextTick()
  rowMenuRef.value?.querySelector<HTMLButtonElement>('[role="menuitem"]')?.focus()
}

function toggleConversationMenu(id: string, event: MouseEvent): void {
  if (openMenuId.value === id) {
    closeConversationMenu()
    return
  }
  void openConversationMenu(id, event)
}

function closeConversationMenu(restoreFocus = false): void {
  if (!openMenuId.value) return
  openMenuId.value = null
  if (restoreFocus) {
    void nextTick(() => activeMenuTrigger?.focus({ preventScroll: true }))
  }
}

function removeConversation(id: string): void {
  emit('remove', id)
  closeConversationMenu(true)
}

function closePicker(restoreFocus = false): void {
  if (!props.pickerOpen) return
  emit('update:pickerOpen', false)
  if (restoreFocus) {
    void nextTick(() => pickerReturnFocus?.focus({ preventScroll: true }))
  }
}

function addConversation(id: string): void {
  emit('add', id)
  closePicker(true)
}

function pickerControls(): HTMLButtonElement[] {
  return pickerItemRefs.value.filter((item) => !item.disabled)
}

function handlePickerKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape') {
    event.preventDefault()
    event.stopPropagation()
    closePicker(true)
    return
  }
  if (!['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) return

  const controls = pickerControls()
  if (controls.length === 0) return
  event.preventDefault()
  const currentIndex = controls.indexOf(document.activeElement as HTMLButtonElement)
  const nextIndex = event.key === 'Home'
    ? 0
    : event.key === 'End'
      ? controls.length - 1
      : (currentIndex + (event.key === 'ArrowDown' ? 1 : -1) + controls.length) % controls.length
  controls[nextIndex]?.focus()
}

function handleDocumentPointerDown(event: PointerEvent): void {
  const target = event.target as Node | null
  if (!target) return

  if (
    openMenuId.value
    && !rowMenuRef.value?.contains(target)
    && !activeMenuTrigger?.contains(target)
  ) {
    closeConversationMenu()
  }

  const targetElement = target instanceof Element ? target : target.parentElement
  const isPickerTrigger = targetElement?.closest(
    '[data-project-chat-picker-trigger], .project-detail-header__add-chat',
  )
  if (props.pickerOpen && !pickerRef.value?.contains(target) && !isPickerTrigger) {
    closePicker()
  }
}

function handleDocumentKeydown(event: KeyboardEvent): void {
  if (event.key !== 'Escape') return
  if (openMenuId.value) {
    event.preventDefault()
    closeConversationMenu(true)
  } else if (props.pickerOpen) {
    event.preventDefault()
    closePicker(true)
  }
}

watch(() => props.pickerOpen, async (visible) => {
  if (!visible) {
    pickerItemRefs.value = []
    return
  }
  pickerReturnFocus = document.activeElement instanceof HTMLElement
    ? document.activeElement
    : null
  await nextTick()
  const firstControl = pickerControls()[0]
  if (firstControl) firstControl.focus({ preventScroll: true })
  else pickerRef.value?.focus({ preventScroll: true })
})

onMounted(() => {
  document.addEventListener('pointerdown', handleDocumentPointerDown, true)
  document.addEventListener('keydown', handleDocumentKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handleDocumentPointerDown, true)
  document.removeEventListener('keydown', handleDocumentKeydown)
})
</script>

<style scoped>
.project-chats-panel {
  --project-panel-text: var(--workspace-text);
  --project-panel-text-secondary: var(--workspace-text-secondary);
  --project-panel-text-muted: var(--workspace-text-muted);
  --project-panel-hover: var(--workspace-hover);
  --project-panel-surface: var(--workspace-popover-surface);
  --project-panel-border: var(--workspace-popover-border);
  --project-panel-focus: var(--workspace-popover-focus);
  --project-panel-row: #f7f7f7;
  position: relative;
  width: 100%;
  color: var(--project-panel-text);
  outline: none;
}

:global(html.light .project-chats-panel) {
  --project-panel-hover: #f3f3f3;
  --project-panel-surface: #fff;
  --project-panel-border: rgb(0 0 0 / 0.12);
}

:global(html.dark .project-chats-panel) {
  --project-panel-hover: #212121;
  --project-panel-surface: #2f2f2f;
  --project-panel-border: rgb(255 255 255 / 0.12);
  --project-panel-row: #181818;
}

.project-chats-panel__list {
  width: 100%;
  margin: 0;
  padding: 0;
  list-style: none;
}

.project-chat-row {
  position: relative;
  display: flex;
  min-height: 64px;
  align-items: center;
  border-radius: 0;
  background: var(--project-panel-row);
}

.project-chat-row:hover,
.project-chat-row:focus-within,
.project-chat-row--menu-open {
  background: var(--project-panel-hover);
}

.project-chat-row__open {
  display: flex;
  min-width: 0;
  min-height: 64px;
  flex: 1;
  align-items: center;
  border: 0;
  padding: 8px 6px 8px 12px;
  color: inherit;
  background: transparent;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.project-chat-row__copy {
  display: flex;
  min-width: 0;
  flex: 1;
  align-items: center;
  gap: 12px;
}

.project-chat-row__copy strong {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  font-size: 14px;
  font-weight: 500;
  line-height: 21px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-chat-row__copy time {
  display: none;
  flex: 0 0 auto;
  color: var(--project-panel-text-secondary);
  font-size: 12px;
  line-height: 18px;
}

.project-chat-row__menu-wrap {
  position: relative;
  display: grid;
  flex: 0 0 48px;
  place-items: center;
}

.project-chat-row__more {
  display: grid;
  width: 36px;
  height: 36px;
  place-items: center;
  border: 0;
  border-radius: 9px;
  padding: 0;
  color: var(--project-panel-text-secondary);
  background: transparent;
  cursor: pointer;
}

.project-chat-row__more:hover,
.project-chat-row__more[aria-expanded='true'] {
  color: var(--project-panel-text);
  background: color-mix(in srgb, var(--project-panel-text) 8%, transparent);
}

.project-chat-row__open:focus-visible,
.project-chat-row__more:focus-visible,
.project-row-menu button:focus-visible,
.project-chats-panel__picker button:focus-visible {
  outline: 2px solid var(--project-panel-focus);
  outline-offset: 2px;
}

.project-row-menu {
  position: absolute;
  z-index: 35;
  top: 40px;
  right: 6px;
  width: max-content;
  min-width: 148px;
  padding: 5px;
  border: 1px solid var(--project-panel-border);
  border-radius: 12px;
  color: var(--project-panel-text);
  background: var(--project-panel-surface);
  box-shadow: 0 6px 18px rgb(0 0 0 / 0.12);
}

.project-row-menu button {
  display: flex;
  width: 100%;
  min-height: 36px;
  align-items: center;
  gap: 9px;
  border: 0;
  border-radius: 8px;
  padding: 0 10px;
  color: inherit;
  background: transparent;
  font: inherit;
  font-size: 13px;
  white-space: nowrap;
  cursor: pointer;
}

.project-row-menu button:hover {
  background: var(--project-panel-hover);
}

.project-chats-panel__empty {
  display: grid;
  min-height: 228px;
  place-items: center;
  align-content: center;
  gap: 10px;
  color: var(--project-panel-text-secondary);
  text-align: center;
}

.project-chats-panel__empty p {
  margin: 0;
  font-size: 14px;
  line-height: 21px;
}

.project-chats-panel__picker {
  position: absolute;
  z-index: 40;
  top: -74px;
  left: 0;
  width: min(360px, calc(100vw - 32px));
  max-height: min(300px, calc(100dvh - 180px));
  overflow: auto;
  padding: 6px;
  border: 1px solid var(--project-panel-border);
  border-radius: 14px;
  color: var(--project-panel-text);
  background: var(--project-panel-surface);
  box-shadow: 0 8px 24px rgb(0 0 0 / 0.14);
}

.project-chats-panel__picker-title {
  margin: 0;
  padding: 7px 9px 5px;
  color: var(--project-panel-text-secondary);
  font-size: 12px;
  font-weight: 500;
  line-height: 18px;
}

.project-chats-panel__picker-list {
  display: grid;
  gap: 1px;
}

.project-chats-panel__picker-list button {
  display: flex;
  width: 100%;
  min-height: 40px;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  border: 0;
  border-radius: 9px;
  padding: 0 10px;
  color: inherit;
  background: transparent;
  font: inherit;
  font-size: 13px;
  text-align: left;
  cursor: pointer;
}

.project-chats-panel__picker-list button:hover {
  background: var(--project-panel-hover);
}

.project-chats-panel__picker-list button span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-chats-panel__picker-empty {
  margin: 0;
  padding: 12px 10px;
  color: var(--project-panel-text-secondary);
  font-size: 13px;
  line-height: 20px;
}

.project-picker-enter-active,
.project-picker-leave-active,
.project-row-menu-enter-active,
.project-row-menu-leave-active {
  transition: opacity 100ms ease, transform 120ms ease;
  transform-origin: top left;
}

.project-picker-enter-from,
.project-picker-leave-to,
.project-row-menu-enter-from,
.project-row-menu-leave-to {
  opacity: 0;
  transform: translateY(-3px) scale(0.985);
}

@media (max-width: 700px) {
  .project-chat-row {
    min-height: 58px;
    background: transparent;
  }

  .project-chat-row__open {
    min-height: 58px;
    padding-left: 12px;
  }

  .project-chat-row__copy time {
    display: inline;
  }

  .project-chat-row__menu-wrap {
    flex-basis: 42px;
  }

  .project-chats-panel__picker {
    position: fixed;
    top: auto;
    right: 16px;
    bottom: calc(102px + env(safe-area-inset-bottom));
    left: 16px;
    width: auto;
    max-height: min(300px, calc(100dvh - 168px));
  }
}

@media (prefers-reduced-motion: reduce) {
  .project-picker-enter-active,
  .project-picker-leave-active,
  .project-row-menu-enter-active,
  .project-row-menu-leave-active {
    transition: none;
  }
}
</style>

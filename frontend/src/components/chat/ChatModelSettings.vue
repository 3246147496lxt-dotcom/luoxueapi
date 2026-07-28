<template>
  <div class="chat-model-settings">
    <button
      ref="triggerRef"
      type="button"
      class="chat-model-settings__trigger"
      :disabled="disabled"
      :aria-expanded="open"
      aria-haspopup="menu"
      :aria-controls="panelId"
      :aria-label="t('chat.settings.label', {
        model: selectedModelLabel,
        effort: selectedEffortLabel,
      })"
      data-test="chat-model-settings-trigger"
      @click="toggleMenu"
    >
      <Icon name="sparkles" size="xs" />
      <span class="chat-model-settings__model">{{ selectedModelLabel }}</span>
      <span class="chat-model-settings__separator" aria-hidden="true">·</span>
      <span class="chat-model-settings__effort">{{ selectedEffortLabel }}</span>
      <Icon
        name="chevronDown"
        size="xs"
        class="chat-model-settings__chevron"
        :class="{ 'chat-model-settings__chevron--open': open }"
      />
    </button>

    <Teleport to="body">
      <Transition name="chat-settings-popover">
        <div
          v-if="open"
          :id="panelId"
          ref="panelRef"
          class="chat-model-settings-popover"
          :style="panelStyle"
          role="menu"
          :aria-label="t('chat.settings.menuLabel')"
          data-ui-portal="chat-model-settings"
          @keydown="onPanelKeydown"
        >
          <template v-if="pane === 'root'">
            <button
              type="button"
              class="chat-model-settings-popover__row"
              role="menuitem"
              aria-haspopup="menu"
              data-test="chat-settings-model-menu"
              @click="openPane('models')"
            >
              <span class="chat-model-settings-popover__row-label">
                <Icon name="sparkles" size="sm" />
                {{ t('chat.settings.model') }}
              </span>
              <span class="chat-model-settings-popover__row-value">
                <span>{{ selectedModelLabel }}</span>
                <Icon name="chevronRight" size="xs" />
              </span>
            </button>

            <button
              type="button"
              class="chat-model-settings-popover__row"
              role="menuitem"
              aria-haspopup="menu"
              data-test="chat-settings-reasoning-menu"
              @click="openPane('reasoning')"
            >
              <span class="chat-model-settings-popover__row-label">
                <Icon name="brain" size="sm" />
                {{ t('chat.settings.reasoning') }}
              </span>
              <span class="chat-model-settings-popover__row-value">
                <span>{{ selectedEffortLabel }}</span>
                <Icon name="chevronRight" size="xs" />
              </span>
            </button>
          </template>

          <template v-else>
            <div class="chat-model-settings-popover__header">
              <button
                type="button"
                class="chat-model-settings-popover__back"
                :aria-label="t('chat.settings.back')"
                @click="openPane('root')"
              >
                <Icon name="chevronLeft" size="sm" />
              </button>
              <strong>
                {{ pane === 'models' ? t('chat.settings.model') : t('chat.settings.reasoning') }}
              </strong>
            </div>

            <div v-if="pane === 'models' && modelOptions.length > 6" class="chat-model-settings-popover__search">
              <Icon name="search" size="sm" />
              <input
                ref="searchInputRef"
                v-model="searchQuery"
                type="search"
                :placeholder="t('chat.settings.searchModels')"
                :aria-label="t('chat.settings.searchModels')"
              />
            </div>

            <div v-if="pane === 'models'" class="chat-model-settings-popover__options">
              <button
                v-for="option in filteredModels"
                :key="option.value"
                type="button"
                class="chat-model-settings-popover__option chat-model-settings-popover__option--model"
                :class="{ 'chat-model-settings-popover__option--selected': option.value === modelValue }"
                role="menuitemradio"
                :aria-checked="option.value === modelValue"
                :data-value="option.value"
                @click="selectModel(option.value)"
              >
                <span class="chat-model-settings-popover__model-mark">
                  <Icon name="sparkles" size="xs" />
                </span>
                <span class="chat-model-settings-popover__option-copy">
                  <strong>{{ option.label }}</strong>
                  <small v-if="option.description">{{ option.description }}</small>
                </span>
                <span v-if="option.recommended" class="chat-model-settings-popover__recommended">
                  {{ t('chat.models.recommended') }}
                </span>
                <Icon
                  v-if="option.value === modelValue"
                  name="check"
                  size="sm"
                  class="chat-model-settings-popover__check"
                />
              </button>

              <p v-if="filteredModels.length === 0" class="chat-model-settings-popover__empty">
                {{ t('chat.models.empty') }}
              </p>
            </div>

            <div v-else class="chat-model-settings-popover__options">
              <button
                v-for="option in reasoningOptions"
                :key="option.value || 'auto'"
                type="button"
                class="chat-model-settings-popover__option"
                :class="{ 'chat-model-settings-popover__option--selected': option.value === reasoningEffort }"
                role="menuitemradio"
                :aria-checked="option.value === reasoningEffort"
                :data-value="option.value || 'auto'"
                @click="selectReasoning(option.value)"
              >
                <span class="chat-model-settings-popover__option-copy">
                  <strong>{{ option.label }}</strong>
                  <small>{{ option.description }}</small>
                </span>
                <Icon
                  v-if="option.value === reasoningEffort"
                  name="check"
                  size="sm"
                  class="chat-model-settings-popover__check"
                />
              </button>
            </div>
          </template>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
  type CSSProperties,
} from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { ChatReasoningEffort } from '@/types/chat'

export interface ChatModelSettingsOption {
  value: string
  label: string
  description?: string
  recommended?: boolean
}

type SettingsPane = 'root' | 'models' | 'reasoning'

const props = withDefaults(defineProps<{
  modelValue: string
  reasoningEffort: ChatReasoningEffort
  modelOptions: ChatModelSettingsOption[]
  disabled?: boolean
  loading?: boolean
}>(), {
  disabled: false,
  loading: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
  'update:reasoningEffort': [value: ChatReasoningEffort]
}>()

const { t } = useI18n()
const panelId = `chat-model-settings-${Math.random().toString(36).slice(2, 9)}`
const open = ref(false)
const pane = ref<SettingsPane>('root')
const searchQuery = ref('')
const panelStyle = ref<CSSProperties>({})
const triggerRef = ref<HTMLButtonElement | null>(null)
const panelRef = ref<HTMLElement | null>(null)
const searchInputRef = ref<HTMLInputElement | null>(null)

const selectedModel = computed(() => (
  props.modelOptions.find((option) => option.value === props.modelValue) ?? null
))
const selectedModelLabel = computed(() => {
  if (props.loading) return t('chat.models.loading')
  return selectedModel.value?.label || t('chat.models.select')
})

const reasoningOptions = computed<Array<{
  value: ChatReasoningEffort
  label: string
  description: string
}>>(() => [
  {
    value: '',
    label: t('chat.settings.reasoningLevels.auto'),
    description: t('chat.settings.reasoningDescriptions.auto'),
  },
  {
    value: 'low',
    label: t('chat.settings.reasoningLevels.low'),
    description: t('chat.settings.reasoningDescriptions.low'),
  },
  {
    value: 'medium',
    label: t('chat.settings.reasoningLevels.medium'),
    description: t('chat.settings.reasoningDescriptions.medium'),
  },
  {
    value: 'high',
    label: t('chat.settings.reasoningLevels.high'),
    description: t('chat.settings.reasoningDescriptions.high'),
  },
  {
    value: 'xhigh',
    label: t('chat.settings.reasoningLevels.xhigh'),
    description: t('chat.settings.reasoningDescriptions.xhigh'),
  },
])
const selectedEffortLabel = computed(() => (
  reasoningOptions.value.find((option) => option.value === props.reasoningEffort)?.label
  || t('chat.settings.reasoningLevels.auto')
))
const filteredModels = computed(() => {
  const query = searchQuery.value.trim().toLocaleLowerCase()
  if (!query) return props.modelOptions
  return props.modelOptions.filter((option) => (
    option.label.toLocaleLowerCase().includes(query)
    || option.value.toLocaleLowerCase().includes(query)
    || option.description?.toLocaleLowerCase().includes(query)
  ))
})

watch(() => props.disabled, (disabled) => {
  if (disabled) closeMenu(false)
})

function toggleMenu() {
  if (props.disabled) return
  if (open.value) {
    closeMenu(false)
    return
  }
  open.value = true
  pane.value = 'root'
  searchQuery.value = ''
  void nextTick(() => {
    updatePanelPosition()
    focusFirstPanelControl()
  })
}

function closeMenu(restoreFocus = true) {
  if (!open.value) return
  open.value = false
  pane.value = 'root'
  searchQuery.value = ''
  if (restoreFocus) void nextTick(() => triggerRef.value?.focus())
}

function openPane(nextPane: SettingsPane) {
  pane.value = nextPane
  if (nextPane !== 'models') searchQuery.value = ''
  void nextTick(() => {
    updatePanelPosition()
    if (nextPane === 'models' && props.modelOptions.length > 6) {
      searchInputRef.value?.focus()
      return
    }
    focusFirstPanelControl()
  })
}

function selectModel(value: string) {
  emit('update:modelValue', value)
  closeMenu()
}

function selectReasoning(value: ChatReasoningEffort) {
  emit('update:reasoningEffort', value)
  closeMenu()
}

function focusFirstPanelControl() {
  panelRef.value?.querySelector<HTMLElement>('button:not([disabled]), input:not([disabled])')?.focus()
}

function updatePanelPosition() {
  const trigger = triggerRef.value
  const panel = panelRef.value
  if (!trigger || !panel) return

  const triggerRect = trigger.getBoundingClientRect()
  const viewportPadding = 12
  const panelWidth = Math.min(360, window.innerWidth - viewportPadding * 2)
  const panelHeight = panel.offsetHeight
  const left = Math.min(
    Math.max(viewportPadding, triggerRect.right - panelWidth),
    window.innerWidth - panelWidth - viewportPadding,
  )
  const hasRoomAbove = triggerRect.top >= panelHeight + viewportPadding * 2

  panelStyle.value = {
    position: 'fixed',
    left: `${Math.max(viewportPadding, left)}px`,
    width: `${panelWidth}px`,
    maxHeight: `calc(100vh - ${viewportPadding * 2}px)`,
    ...(hasRoomAbove
      ? { bottom: `${window.innerHeight - triggerRect.top + 8}px` }
      : { top: `${triggerRect.bottom + 8}px` }),
  }
}

function onPanelKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.preventDefault()
    closeMenu()
    return
  }
  if (event.key === 'ArrowLeft' && pane.value !== 'root') {
    event.preventDefault()
    openPane('root')
  }
}

function onDocumentPointerDown(event: PointerEvent) {
  const target = event.target as Node | null
  if (
    !open.value
    || (target && triggerRef.value?.contains(target))
    || (target && panelRef.value?.contains(target))
  ) {
    return
  }
  closeMenu(false)
}

onMounted(() => {
  document.addEventListener('pointerdown', onDocumentPointerDown)
  window.addEventListener('resize', updatePanelPosition)
  window.addEventListener('scroll', updatePanelPosition, true)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onDocumentPointerDown)
  window.removeEventListener('resize', updatePanelPosition)
  window.removeEventListener('scroll', updatePanelPosition, true)
})
</script>

<style scoped>
.chat-model-settings {
  min-width: 0;
  flex: 0 1 auto;
}

.chat-model-settings__trigger {
  display: flex;
  width: min(250px, 32vw);
  min-width: 154px;
  height: 44px;
  align-items: center;
  gap: 7px;
  border: 1px solid var(--lx-clay-border);
  border-radius: 8px;
  padding: 0 11px;
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-recessed);
  box-shadow: var(--lx-clay-shadow-inset);
  font-family: var(--lx-clay-font-ui);
  font-size: 12px;
  font-weight: 750;
  transition:
    border-color 160ms ease,
    background-color 160ms ease,
    color 160ms ease,
    box-shadow 160ms ease;
}

.chat-model-settings__trigger:hover:not(:disabled),
.chat-model-settings__trigger[aria-expanded="true"] {
  border-color: color-mix(in srgb, var(--lx-clay-accent) 38%, var(--lx-clay-border));
  color: var(--lx-clay-accent-deep);
  background: color-mix(in srgb, var(--lx-clay-recessed) 84%, var(--lx-clay-accent-soft));
}

html.dark .chat-model-settings__trigger:hover:not(:disabled),
html.dark .chat-model-settings__trigger[aria-expanded="true"] {
  color: var(--lx-clay-accent);
}

.chat-model-settings__trigger:disabled {
  cursor: not-allowed;
  opacity: 0.52;
}

.chat-model-settings__model,
.chat-model-settings__effort {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-model-settings__model {
  min-width: 0;
  flex: 1;
  text-align: left;
}

.chat-model-settings__effort {
  flex: 0 0 auto;
  color: var(--lx-clay-text-muted);
}

.chat-model-settings__separator {
  flex: 0 0 auto;
  color: var(--lx-clay-text-subtle);
}

.chat-model-settings__chevron {
  flex: 0 0 auto;
  color: var(--lx-clay-text-muted);
  transition: transform 160ms cubic-bezier(0.22, 1, 0.36, 1);
}

.chat-model-settings__chevron--open {
  transform: rotate(180deg);
}

@media (max-width: 640px) {
  .chat-model-settings__trigger {
    width: min(210px, 55vw);
    min-width: 0;
  }
}

@media (max-width: 430px) {
  .chat-model-settings__trigger {
    width: min(174px, calc(100vw - 94px));
  }
}

@media (prefers-reduced-motion: reduce) {
  .chat-model-settings__trigger,
  .chat-model-settings__chevron {
    transition: none;
  }
}
</style>

<style>
.chat-model-settings-popover {
  z-index: 60;
  overflow: hidden;
  border: 1px solid var(--lx-clay-border-strong);
  border-radius: 8px;
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-overlay);
}

.chat-model-settings-popover__row {
  display: flex;
  width: 100%;
  min-height: 54px;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  border: 0;
  border-bottom: 1px solid var(--lx-clay-border);
  padding: 0 16px;
  color: var(--lx-clay-text);
  background: transparent;
  font-family: var(--lx-clay-font-ui);
  font-size: 14px;
  transition: background-color 140ms ease, color 140ms ease;
}

.chat-model-settings-popover__row:last-child {
  border-bottom: 0;
}

.chat-model-settings-popover__row:hover,
.chat-model-settings-popover__row:focus-visible {
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
  outline: none;
}

html.dark .chat-model-settings-popover__row:hover,
html.dark .chat-model-settings-popover__row:focus-visible {
  color: var(--lx-clay-accent);
}

.chat-model-settings-popover__row-label,
.chat-model-settings-popover__row-value {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 8px;
}

.chat-model-settings-popover__row-label {
  font-weight: 800;
}

.chat-model-settings-popover__row-value {
  color: var(--lx-clay-text-muted);
  font-weight: 700;
}

.chat-model-settings-popover__row-value > span {
  overflow: hidden;
  max-width: 172px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-model-settings-popover__header {
  display: flex;
  height: 48px;
  align-items: center;
  gap: 8px;
  border-bottom: 1px solid var(--lx-clay-border);
  padding: 0 12px;
}

.chat-model-settings-popover__header strong {
  font-family: var(--lx-clay-font-ui);
  font-size: 14px;
}

.chat-model-settings-popover__back {
  display: grid;
  width: 32px;
  height: 32px;
  place-items: center;
  border: 0;
  border-radius: 7px;
  color: var(--lx-clay-text-secondary);
  background: transparent;
}

.chat-model-settings-popover__back:hover,
.chat-model-settings-popover__back:focus-visible {
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
  outline: none;
}

.chat-model-settings-popover__search {
  display: flex;
  min-height: 46px;
  align-items: center;
  gap: 8px;
  border-bottom: 1px solid var(--lx-clay-border);
  padding: 0 14px;
  color: var(--lx-clay-text-muted);
  background: var(--lx-clay-recessed);
}

.chat-model-settings-popover__search input {
  min-width: 0;
  flex: 1;
  border: 0;
  padding: 8px 0;
  color: var(--lx-clay-text);
  background: transparent;
  font-family: var(--lx-clay-font-ui);
  font-size: 13px;
  outline: none;
}

.chat-model-settings-popover__search input::placeholder {
  color: var(--lx-clay-text-muted);
}

.chat-model-settings-popover__options {
  max-height: min(380px, calc(100vh - 120px));
  overflow-y: auto;
  padding: 6px;
  overscroll-behavior: contain;
}

.chat-model-settings-popover__option {
  display: flex;
  width: 100%;
  min-height: 52px;
  align-items: center;
  gap: 10px;
  border: 0;
  border-radius: 7px;
  padding: 8px 10px;
  color: var(--lx-clay-text-secondary);
  background: transparent;
  text-align: left;
  transition: background-color 140ms ease, color 140ms ease;
}

.chat-model-settings-popover__option:hover,
.chat-model-settings-popover__option:focus-visible,
.chat-model-settings-popover__option--selected {
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
  outline: none;
}

html.dark .chat-model-settings-popover__option:hover,
html.dark .chat-model-settings-popover__option:focus-visible,
html.dark .chat-model-settings-popover__option--selected {
  color: var(--lx-clay-accent);
}

.chat-model-settings-popover__model-mark {
  display: grid;
  width: 30px;
  height: 30px;
  flex: 0 0 30px;
  place-items: center;
  border-radius: 7px;
  color: var(--lx-clay-accent);
  background: var(--lx-clay-accent-soft);
}

.chat-model-settings-popover__option-copy {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 2px;
}

.chat-model-settings-popover__option-copy strong,
.chat-model-settings-popover__option-copy small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-model-settings-popover__option-copy strong {
  color: inherit;
  font-family: var(--lx-clay-font-ui);
  font-size: 13px;
  font-weight: 800;
}

.chat-model-settings-popover__option-copy small {
  color: var(--lx-clay-text-muted);
  font-size: 11px;
}

.chat-model-settings-popover__recommended {
  flex: 0 0 auto;
  border-radius: 6px;
  padding: 2px 6px;
  color: var(--lx-clay-accent-deep);
  background: color-mix(in srgb, var(--lx-clay-accent-soft) 74%, var(--lx-clay-surface));
  font-size: 9px;
  font-weight: 850;
}

.chat-model-settings-popover__check {
  flex: 0 0 auto;
  color: var(--lx-clay-accent);
}

.chat-model-settings-popover__empty {
  margin: 0;
  padding: 28px 16px;
  color: var(--lx-clay-text-muted);
  font-size: 13px;
  text-align: center;
}

.chat-settings-popover-enter-active,
.chat-settings-popover-leave-active {
  transition:
    opacity 150ms ease,
    transform 150ms cubic-bezier(0.22, 1, 0.36, 1);
}

.chat-settings-popover-enter-from,
.chat-settings-popover-leave-to {
  opacity: 0;
  transform: translateY(6px);
}

@media (prefers-reduced-motion: reduce) {
  .chat-model-settings-popover__row,
  .chat-model-settings-popover__option,
  .chat-settings-popover-enter-active,
  .chat-settings-popover-leave-active {
    transition: none;
  }
}
</style>

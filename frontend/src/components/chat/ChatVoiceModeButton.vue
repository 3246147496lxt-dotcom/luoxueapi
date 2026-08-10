<template>
  <ChatControlTooltip
    class="chat-voice-mode"
    :label="label"
    :shortcut="available && !disabled ? ['⌃', '⇧', 'V'] : []"
    :enabled="!disabled"
    :accessible="false"
  >
    <button
      ref="triggerRef"
      type="button"
      class="chat-voice-mode__trigger"
      :disabled="disabled"
      :aria-disabled="disabled || !available"
      :aria-label="label"
      :aria-keyshortcuts="available && !disabled ? 'Control+Shift+V' : undefined"
      data-chat-control-anchor
      data-test="chat-voice-mode-trigger"
      @click="activate"
    >
      <Icon name="chatVoiceMode" size="lg" aria-hidden="true" />
    </button>
  </ChatControlTooltip>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ChatControlTooltip from './ChatControlTooltip.vue'
import { chatControlShortcutIsInScope } from './chatControlShortcut'
import Icon from '@/components/icons/Icon.vue'

const props = withDefaults(defineProps<{
  disabled?: boolean
  available?: boolean
}>(), {
  disabled: false,
  available: true,
})

const emit = defineEmits<{
  activate: []
}>()

const { t } = useI18n()
const triggerRef = ref<HTMLButtonElement | null>(null)
const label = computed(() => (
  props.available ? t('chat.voice.modeStart') : t('chat.voice.modeUnavailable')
))

function activate() {
  if (props.disabled || !props.available) return
  emit('activate')
}

function onVoiceModeShortcut(event: KeyboardEvent): void {
  if (
    event.repeat
    || event.code !== 'KeyV'
    || !event.ctrlKey
    || !event.shiftKey
    || event.altKey
    || event.metaKey
    || props.disabled
    || !props.available
    || !chatControlShortcutIsInScope(event, triggerRef.value)
  ) return
  event.preventDefault()
  activate()
}

onMounted(() => window.addEventListener('keydown', onVoiceModeShortcut))
onBeforeUnmount(() => window.removeEventListener('keydown', onVoiceModeShortcut))
</script>

<style scoped>
.chat-voice-mode {
  position: relative;
  display: inline-grid;
  width: 36px;
  height: 36px;
  place-items: center;
}

.chat-voice-mode__trigger {
  display: grid;
  width: 36px;
  height: 36px;
  place-items: center;
  border: 0;
  border-radius: 50%;
  padding: 0;
  color: var(--chat-composer-action-fg, #fff);
  background: var(--chat-composer-action-bg, #000);
  box-shadow: none;
  transition:
    background-color 140ms ease,
    transform 140ms cubic-bezier(0.22, 1, 0.36, 1);
}

.chat-voice-mode__trigger:hover:not(:disabled):not([aria-disabled="true"]) {
  color: var(--chat-composer-action-fg, #fff);
  background: var(--chat-composer-action-hover, #2f2f2f);
}

.chat-voice-mode__trigger:active:not(:disabled):not([aria-disabled="true"]) {
  transform: scale(0.96);
}

.chat-voice-mode__trigger:focus-visible {
  outline: 3px solid var(--lx-clay-accent-soft);
  outline-offset: 1px;
}

.chat-voice-mode__trigger:disabled {
  cursor: not-allowed;
  opacity: 0.38;
}

@media (prefers-reduced-motion: reduce) {
  .chat-voice-mode__trigger {
    transition: none;
  }
}
</style>

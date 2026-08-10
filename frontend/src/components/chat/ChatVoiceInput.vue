<template>
  <div
    class="chat-voice-input"
    :class="`chat-voice-input--${state}`"
    :data-state="state"
    @keydown.esc.prevent.stop="cancelOperation"
  >
    <ChatControlTooltip
      v-slot="{ tooltipId }"
      :label="state === 'idle' ? t('chat.voice.dictation') : buttonLabel"
      :shortcut="state === 'idle' ? ['⌃', '⇧', 'D'] : []"
      :enabled="tooltipEnabled"
      :accessible="state === 'idle'"
    >
      <button
        ref="triggerRef"
        type="button"
        class="chat-voice-input__trigger"
        :class="{ 'chat-voice-input__trigger--wide': busy }"
        :disabled="buttonDisabled"
        :aria-label="buttonLabel"
        :aria-pressed="state === 'recording'"
        :aria-describedby="state === 'idle' ? `${noticeId} ${tooltipId}` : noticeId"
        :aria-keyshortcuts="buttonDisabled ? undefined : 'Control+Shift+D'"
        data-chat-control-anchor
        data-test="chat-voice-trigger"
        @click="toggleRecording"
      >
        <span
          v-if="state === 'requesting' || state === 'transcribing'"
          class="chat-voice-input__spinner"
          aria-hidden="true"
        ></span>
        <span
          v-else-if="state === 'recording'"
          class="chat-voice-input__stop-mark"
          aria-hidden="true"
        ></span>
        <Icon v-else name="chatMicrophone" size="md" aria-hidden="true" />

        <span v-if="state === 'recording'" class="chat-voice-input__timer" aria-hidden="true">
          {{ formattedElapsed }}
        </span>
        <span
          v-else-if="state === 'requesting' || state === 'transcribing'"
          class="chat-voice-input__label"
          aria-hidden="true"
        >
          {{ state === 'requesting' ? t('chat.voice.requestingShort') : t('chat.voice.transcribingShort') }}
        </span>
      </button>
    </ChatControlTooltip>

    <div
      v-if="state === 'error' && errorMessage"
      class="chat-voice-input__error"
      role="alert"
      data-test="chat-voice-error"
    >
      <div class="chat-voice-input__error-header">
        <span>{{ errorMessage }}</span>
        <button
          type="button"
          class="chat-voice-input__dismiss"
          :aria-label="t('chat.voice.dismissError')"
          @click="dismissError"
        >
          <Icon name="x" size="xs" />
        </button>
      </div>
      <textarea
        v-if="pendingTranscript"
        class="chat-voice-input__pending-transcript"
        :value="pendingTranscript"
        :aria-label="t('chat.voice.pendingTranscriptLabel')"
        readonly
        data-test="chat-voice-pending-transcript"
        @focus="selectPendingTranscript"
      ></textarea>
      <button
        v-if="pendingTranscript"
        type="button"
        class="chat-voice-input__retry-insert"
        :disabled="props.disabled"
        data-test="chat-voice-retry-insert"
        @click="retryPendingTranscript"
      >
        {{ t('chat.voice.retryInsert') }}
      </button>
    </div>

    <span :id="noticeId" class="sr-only">{{ t('chat.voice.cloudNotice') }}</span>
    <span :id="statusId" class="sr-only" role="status" aria-live="polite" aria-atomic="true">
      {{ announcement }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ChatControlTooltip from './ChatControlTooltip.vue'
import { chatControlShortcutIsInScope } from './chatControlShortcut'
import Icon from '@/components/icons/Icon.vue'
import {
  ChatAPIError,
  createChatIdempotencyKey,
  isAbortError,
  transcribeChatAudio,
} from '@/api/chat'
import {
  AudioRecorderError,
  CHAT_AUDIO_MAX_BYTES,
  CHAT_AUDIO_MAX_DURATION_MS,
  useAudioRecorder,
} from '@/composables/useAudioRecorder'

type VoiceInputState = 'idle' | 'requesting' | 'recording' | 'transcribing' | 'error'

let voiceInputSequence = 0

const props = withDefaults(defineProps<{
  disabled?: boolean
  contextKey?: string
  maxDurationMs?: number
  maxBytes?: number
  acceptedMimeTypes?: string[]
}>(), {
  disabled: false,
  contextKey: '',
  maxDurationMs: CHAT_AUDIO_MAX_DURATION_MS,
  maxBytes: CHAT_AUDIO_MAX_BYTES,
  acceptedMimeTypes: () => [],
})

const emit = defineEmits<{
  transcribed: [text: string, acknowledge: (inserted: boolean) => void]
  'busy-change': [busy: boolean]
}>()

const { t } = useI18n()
const inputId = ++voiceInputSequence
const noticeId = `chat-voice-notice-${inputId}`
const statusId = `chat-voice-status-${inputId}`
const triggerRef = ref<HTMLButtonElement | null>(null)
const state = ref<VoiceInputState>('idle')
const errorMessage = ref('')
const pendingTranscript = ref('')
const uploadController = ref<AbortController | null>(null)
let operationVersion = 0
let recordingIdempotencyKey = ''

const recorder = useAudioRecorder({
  maxDurationMs: Math.min(CHAT_AUDIO_MAX_DURATION_MS, props.maxDurationMs),
  maxBytes: Math.min(CHAT_AUDIO_MAX_BYTES, props.maxBytes),
  acceptedMimeTypes: props.acceptedMimeTypes,
  onMaximumDuration: () => {
    if (state.value === 'recording') void finishRecording()
  },
  onError: (error) => {
    if (state.value === 'recording') showError(error)
  },
})

const busy = computed(() => (
  state.value === 'requesting'
  || state.value === 'recording'
  || state.value === 'transcribing'
))
const buttonDisabled = computed(() => props.disabled && !busy.value)
const tooltipEnabled = computed(() => state.value !== 'error' && !buttonDisabled.value)
const formattedElapsed = computed(() => {
  const maximumSeconds = Math.ceil(Math.min(CHAT_AUDIO_MAX_DURATION_MS, props.maxDurationMs) / 1_000)
  const totalSeconds = Math.min(maximumSeconds, Math.floor(recorder.elapsedMs.value / 1_000))
  const minutes = Math.floor(totalSeconds / 60).toString().padStart(2, '0')
  const seconds = (totalSeconds % 60).toString().padStart(2, '0')
  return `${minutes}:${seconds}`
})
const buttonLabel = computed(() => {
  if (state.value === 'requesting') return t('chat.voice.cancelRequest')
  if (state.value === 'recording') return t('chat.voice.stop')
  if (state.value === 'transcribing') return t('chat.voice.cancelTranscription')
  if (state.value === 'error') {
    return pendingTranscript.value ? t('chat.voice.retryInsert') : t('chat.voice.retry')
  }
  return t('chat.voice.start')
})
const announcement = computed(() => {
  if (state.value === 'requesting') return t('chat.voice.requesting')
  if (state.value === 'recording') return t('chat.voice.recording')
  if (state.value === 'transcribing') return t('chat.voice.transcribing')
  // The visible error card already announces itself via role="alert".
  if (state.value === 'error') return ''
  return ''
})

watch(busy, (value) => emit('busy-change', value), { immediate: true, flush: 'sync' })
watch(() => props.contextKey, (next, previous) => {
  if (next !== previous) cancelOperation()
})
watch(() => props.disabled, (disabled) => {
  if (disabled && busy.value) cancelOperation()
})

function isCancelled(error: unknown): boolean {
  return isAbortError(error)
    || (error instanceof AudioRecorderError && error.code === 'RECORDING_CANCELLED')
}

function errorText(error: unknown): string {
  if (error instanceof AudioRecorderError) {
    if (error.code === 'AUDIO_RECORDING_UNSUPPORTED') return t('chat.voice.errors.unsupported')
    if (error.code === 'MICROPHONE_PERMISSION_DENIED') return t('chat.voice.errors.permissionDenied')
    if (error.code === 'MICROPHONE_NOT_FOUND') return t('chat.voice.errors.notFound')
    if (error.code === 'MICROPHONE_BUSY') return t('chat.voice.errors.busy')
    if (error.code === 'EMPTY_AUDIO') return t('chat.voice.errors.empty')
    if (error.code === 'AUDIO_TOO_LARGE') {
      return t('chat.voice.errors.tooLarge', { size: formattedMaximumSize() })
    }
    return t('chat.voice.errors.recordingFailed')
  }

  if (error instanceof ChatAPIError) {
    const code = String(error.code || '')
    if (code === 'AUDIO_TOO_LARGE') {
      return t('chat.voice.errors.tooLarge', { size: formattedMaximumSize() })
    }
    if (code === 'AUDIO_TOO_LONG') {
      return t('chat.voice.errors.tooLong', { seconds: maximumDurationSeconds() })
    }
    if (
      code === 'SPEECH_NOT_DETECTED'
      || code === 'NO_SPEECH_DETECTED'
      || code === 'TRANSCRIPTION_EMPTY'
      || code === 'INVALID_TRANSCRIPTION_RESPONSE'
    ) {
      return t('chat.voice.errors.noSpeech')
    }
    if (code === 'TRANSCRIPTION_DAILY_QUOTA_EXCEEDED') {
      return t('chat.voice.errors.dailyQuotaExceeded')
    }
    if (code === 'TRANSCRIPTION_RATE_LIMITED' || error.status === 429) {
      return t('chat.voice.errors.rateLimited')
    }
    if (code === 'TRANSCRIPTION_UNAVAILABLE' || error.status === 503) {
      return t('chat.voice.errors.unavailable')
    }
  }
  return t('chat.voice.errors.transcriptionFailed')
}

function maximumDurationSeconds(): number {
  return Math.ceil(Math.min(CHAT_AUDIO_MAX_DURATION_MS, props.maxDurationMs) / 1_000)
}

function formattedMaximumSize(): string {
  const mebibytes = Math.min(CHAT_AUDIO_MAX_BYTES, props.maxBytes) / (1024 * 1024)
  return Number.isInteger(mebibytes) ? String(mebibytes) : mebibytes.toFixed(1)
}

function showError(error: unknown): void {
  pendingTranscript.value = ''
  errorMessage.value = errorText(error)
  state.value = 'error'
}

function insertTranscript(text: string): boolean {
  let inserted = false
  emit('transcribed', text, (acknowledged) => {
    inserted = acknowledged
  })
  return inserted
}

function preserveTranscript(text: string): void {
  pendingTranscript.value = text
  errorMessage.value = t('chat.voice.errors.draftFull')
  state.value = 'error'
}

function retryPendingTranscript(): void {
  if (props.disabled || !pendingTranscript.value) return
  const transcript = pendingTranscript.value
  if (!insertTranscript(transcript)) {
    preserveTranscript(transcript)
    return
  }
  pendingTranscript.value = ''
  errorMessage.value = ''
  state.value = 'idle'
}

function selectPendingTranscript(event: FocusEvent): void {
  if (event.currentTarget instanceof HTMLTextAreaElement) event.currentTarget.select()
}

async function startRecording(): Promise<void> {
  if (props.disabled) return
  cancelOperation()
  const version = ++operationVersion
  recordingIdempotencyKey = createChatIdempotencyKey()
  errorMessage.value = ''
  state.value = 'requesting'
  try {
    await recorder.start()
    if (version !== operationVersion) return
    state.value = 'recording'
  } catch (error) {
    if (version !== operationVersion || isCancelled(error)) return
    showError(error)
  }
}

async function finishRecording(): Promise<void> {
  if (state.value !== 'recording') return
  const version = operationVersion
  state.value = 'transcribing'
  try {
    const recording = await recorder.stop()
    if (version !== operationVersion) return
    const controller = new AbortController()
    uploadController.value = controller
    const result = await transcribeChatAudio(recording.blob, {
      signal: controller.signal,
      idempotencyKey: recordingIdempotencyKey,
    })
    if (version !== operationVersion) return
    if (!insertTranscript(result.text)) {
      preserveTranscript(result.text)
      return
    }
    pendingTranscript.value = ''
    errorMessage.value = ''
    state.value = 'idle'
  } catch (error) {
    if (version !== operationVersion || isCancelled(error)) return
    showError(error)
  } finally {
    if (version === operationVersion) uploadController.value = null
  }
}

function cancelOperation(): void {
  operationVersion += 1
  uploadController.value?.abort()
  uploadController.value = null
  recorder.cancel()
  recordingIdempotencyKey = ''
  pendingTranscript.value = ''
  errorMessage.value = ''
  state.value = 'idle'
}

function dismissError(): void {
  pendingTranscript.value = ''
  errorMessage.value = ''
  state.value = 'idle'
}

function toggleRecording(): void {
  if (state.value === 'recording') {
    void finishRecording()
    return
  }
  if (state.value === 'requesting' || state.value === 'transcribing') {
    cancelOperation()
    return
  }
  if (state.value === 'error' && pendingTranscript.value) {
    retryPendingTranscript()
    return
  }
  void startRecording()
}

function startFromExternal(): void {
  if (buttonDisabled.value || busy.value) return
  if (state.value === 'error' && pendingTranscript.value) {
    retryPendingTranscript()
    return
  }
  void startRecording()
}

function onDictationShortcut(event: KeyboardEvent): void {
  if (
    event.repeat
    || event.code !== 'KeyD'
    || !event.ctrlKey
    || !event.shiftKey
    || event.altKey
    || event.metaKey
    || buttonDisabled.value
    || !chatControlShortcutIsInScope(event, triggerRef.value)
  ) return
  event.preventDefault()
  toggleRecording()
}

onMounted(() => window.addEventListener('keydown', onDictationShortcut))
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onDictationShortcut)
  cancelOperation()
})

defineExpose({ cancel: cancelOperation, start: startFromExternal, activate: toggleRecording })
</script>

<style scoped>
.chat-voice-input {
  position: relative;
  display: flex;
  min-width: 36px;
  flex: 0 0 auto;
  align-items: center;
}

.chat-voice-input__trigger {
  display: flex;
  width: 36px;
  height: 36px;
  align-items: center;
  justify-content: center;
  gap: 7px;
  flex: 0 0 auto;
  border: 0;
  border-radius: 50%;
  padding: 0;
  color: var(--chat-composer-secondary-fg, #0d0d0d);
  background: transparent;
  box-shadow: none;
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
  transition:
    width 160ms ease,
    border-color 160ms ease,
    color 160ms ease,
    background-color 160ms ease,
    opacity 160ms ease;
}

.chat-voice-input__trigger--wide {
  width: auto;
  min-width: 84px;
  border-radius: 999px;
  padding: 0 11px;
}

.chat-voice-input__trigger:hover:not(:disabled) {
  color: var(--chat-composer-secondary-fg, #0d0d0d);
  background: var(--chat-composer-secondary-hover, rgb(0 0 0 / 5%));
}

.chat-voice-input__trigger:focus-visible {
  outline: 3px solid var(--lx-clay-accent-soft);
  outline-offset: 1px;
}

.chat-voice-input__trigger:disabled {
  cursor: not-allowed;
  opacity: 0.42;
}

.chat-voice-input--recording .chat-voice-input__trigger {
  border-color: color-mix(in srgb, var(--lx-clay-danger) 42%, var(--lx-clay-border));
  color: var(--lx-clay-danger);
  background: var(--lx-clay-danger-soft);
}

.chat-voice-input--transcribing .chat-voice-input__trigger,
.chat-voice-input--requesting .chat-voice-input__trigger {
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

.chat-voice-input__stop-mark {
  width: 10px;
  height: 10px;
  flex: 0 0 10px;
  border-radius: 2px;
  background: currentColor;
}

.chat-voice-input__spinner {
  width: 14px;
  height: 14px;
  flex: 0 0 14px;
  border: 2px solid color-mix(in srgb, currentColor 28%, transparent);
  border-top-color: currentColor;
  border-radius: 50%;
  animation: chat-voice-spin 800ms linear infinite;
}

.chat-voice-input__timer,
.chat-voice-input__label {
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.chat-voice-input__error {
  position: absolute;
  right: 0;
  bottom: calc(100% + 8px);
  z-index: 70;
  display: flex;
  width: min(290px, calc(100vw - 64px));
  max-width: calc(100vw - 64px);
  flex-direction: column;
  gap: 9px;
  border: 1px solid color-mix(in srgb, var(--lx-clay-danger) 28%, var(--lx-clay-border));
  border-radius: 8px;
  padding: 9px 9px 9px 11px;
  color: var(--lx-clay-danger);
  background: var(--lx-clay-surface-elevated);
  box-shadow: var(--lx-clay-shadow-overlay);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  line-height: 1.45;
}

.chat-voice-input__error-header {
  display: flex;
  width: 100%;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
}

.chat-voice-input__pending-transcript {
  width: 100%;
  max-width: 100%;
  min-height: 82px;
  max-height: 150px;
  resize: vertical;
  border: 1px solid var(--lx-clay-border);
  border-radius: 7px;
  padding: 8px 9px;
  color: var(--lx-clay-text);
  background: var(--lx-clay-recessed);
  font: inherit;
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
  line-height: 1.5;
}

.chat-voice-input__pending-transcript:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--lx-clay-accent) 58%, transparent);
  outline-offset: 1px;
}

.chat-voice-input__retry-insert {
  min-height: 32px;
  border: 1px solid color-mix(in srgb, var(--lx-clay-danger) 28%, var(--lx-clay-border));
  border-radius: 7px;
  padding: 5px 10px;
  color: var(--lx-clay-danger);
  background: var(--lx-clay-danger-soft);
  font: inherit;
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
}

.chat-voice-input__retry-insert:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}

.chat-voice-input__dismiss {
  display: grid;
  width: 24px;
  height: 24px;
  place-items: center;
  flex: 0 0 24px;
  border: 0;
  border-radius: 6px;
  color: currentColor;
  background: transparent;
}

.chat-voice-input__dismiss:hover {
  background: var(--lx-clay-danger-soft);
}

@keyframes chat-voice-spin {
  to { transform: rotate(360deg); }
}

@media (max-width: 380px) {
  .chat-voice-input__trigger--wide {
    min-width: 78px;
    padding-inline: 8px;
  }

  .chat-voice-input__label {
    max-width: 48px;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}

@media (max-width: 720px) {
  .chat-voice-input {
    min-width: 36px;
  }

  .chat-voice-input__trigger {
    width: 36px;
    height: 36px;
  }

  .chat-voice-input__trigger--wide {
    width: auto;
  }
}

@media (prefers-reduced-motion: reduce) {
  .chat-voice-input__trigger {
    transition: none;
  }

  .chat-voice-input__spinner {
    animation: none;
  }
}
</style>

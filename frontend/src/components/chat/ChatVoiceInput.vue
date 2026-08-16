<template>
  <div
    ref="rootRef"
    class="chat-voice-input"
    :class="`chat-voice-input--${state}`"
    :data-state="state"
  >
    <div
      v-if="waveformShellActive"
      class="chat-voice-input__recording-shell"
      data-test="chat-voice-recording-shell"
    >
      <span class="chat-voice-input__recording-plus" aria-hidden="true">
        <Icon name="chatPlus" size="md" />
      </span>
      <canvas
        ref="recordingCanvasRef"
        class="chat-voice-input__waveform"
        :class="{ 'chat-voice-input__waveform--hidden': !waveformVisible }"
        data-test="chat-voice-waveform"
        aria-hidden="true"
      ></canvas>
      <button
        ref="recordingCancelRef"
        type="button"
        class="chat-voice-input__recording-action"
        :class="{ 'chat-voice-input__recording-action--muted': state === 'transcribing' }"
        :aria-label="cancelButtonLabel"
        :aria-describedby="noticeId"
        data-test="chat-voice-cancel"
        @click="cancelWaveformOperation"
      >
        <Icon name="x" size="md" :stroke-width="2" aria-hidden="true" />
      </button>
      <button
        v-if="state !== 'transcribing'"
        ref="recordingConfirmRef"
        type="button"
        class="chat-voice-input__recording-action chat-voice-input__recording-action--confirm"
        :disabled="preparingState"
        :aria-label="confirmButtonLabel"
        :aria-describedby="noticeId"
        data-test="chat-voice-confirm"
        @click="confirmRecording"
      >
        <Icon name="check" size="lg" :stroke-width="2.1" aria-hidden="true" />
      </button>
      <span
        v-else
        class="chat-voice-input__recording-progress"
        aria-hidden="true"
        data-test="chat-voice-transcribing-spinner"
      >
        <span class="chat-voice-input__recording-spinner"></span>
      </span>
    </div>

    <ChatControlTooltip
      v-else
      v-slot="{ tooltipId }"
      :label="idleState ? t('chat.voice.dictation') : buttonLabel"
      :shortcut="idleState ? ['⌃', '⇧', 'D'] : []"
      :enabled="tooltipEnabled"
      :accessible="idleState"
    >
      <button
        ref="triggerRef"
        type="button"
        class="chat-voice-input__trigger"
        :disabled="buttonDisabled"
        :aria-label="buttonLabel"
        :aria-describedby="idleState ? `${noticeId} ${tooltipId}` : noticeId"
        :aria-keyshortcuts="buttonDisabled ? undefined : 'Control+Shift+D'"
        data-chat-control-anchor
        data-test="chat-voice-trigger"
        @click="toggleRecording"
      >
        <Icon name="chatMicrophone" size="md" aria-hidden="true" />
      </button>
    </ChatControlTooltip>

    <div
      v-if="errorState && errorMessage"
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
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ChatControlTooltip from './ChatControlTooltip.vue'
import { chatControlShortcutIsInScope } from './chatControlShortcut'
import {
  appendChatVoiceWaveformSample,
  CHAT_VOICE_WAVEFORM_SAMPLE_INTERVAL_MS,
  CHAT_VOICE_WAVEFORM_STEP_PX,
  type ChatVoiceWaveformSample,
  chatVoiceWaveformCapacity,
  chatVoiceWaveformHeight,
  chatVoiceWaveformPosition,
  resizeChatVoiceWaveformTrack,
} from './chatVoiceWaveform'
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

type VoiceCapabilityState = 'idle' | 'initializing' | 'ready' | 'unavailable'
type VoiceInputState =
  | 'idle'
  | 'initializing'
  | 'requesting'
  | 'recording'
  | 'transcribing'
  | 'denied'
  | 'unsupported'
  | 'unavailable'
  | 'error'

let voiceInputSequence = 0

const props = withDefaults(defineProps<{
  disabled?: boolean
  contextKey?: string
  capabilityState?: VoiceCapabilityState
  initializeCapability?: () => Promise<VoiceCapabilityState>
  maxDurationMs?: number
  maxBytes?: number
  acceptedMimeTypes?: string[]
}>(), {
  disabled: false,
  contextKey: '',
  capabilityState: 'ready',
  initializeCapability: undefined,
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
const rootRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLButtonElement | null>(null)
const recordingCancelRef = ref<HTMLButtonElement | null>(null)
const recordingConfirmRef = ref<HTMLButtonElement | null>(null)
const recordingCanvasRef = ref<HTMLCanvasElement | null>(null)
const state = ref<VoiceInputState>('idle')
const errorMessage = ref('')
const pendingTranscript = ref('')
const uploadController = ref<AbortController | null>(null)
let operationVersion = 0
let initializationPromise: Promise<void> | null = null
let recordingIdempotencyKey = ''
let waveformFrame: number | null = null
let waveformResizeObserver: ResizeObserver | null = null
let waveformLastSampleAt = 0
let waveformSampleProgress = 0
let waveformPendingLevel = 0
let waveformHistory: ChatVoiceWaveformSample[] = []
let focusRecordingConfirmWhenReady = false
let restoreTriggerFocusWhenReady = false
let motionPreference: MediaQueryList | null = null
let componentUnmounting = false
const isolatedComposerElements: Array<{
  element: HTMLElement
  inert: string | null
  ariaHidden: string | null
}> = []

const recorder = useAudioRecorder({
  maxDurationMs: () => Math.min(CHAT_AUDIO_MAX_DURATION_MS, props.maxDurationMs),
  maxBytes: () => Math.min(CHAT_AUDIO_MAX_BYTES, props.maxBytes),
  acceptedMimeTypes: () => props.acceptedMimeTypes,
  onMaximumDuration: () => {
    if (state.value === 'recording') void finishRecording()
  },
  onError: (error) => {
    if (state.value === 'recording') showError(error)
  },
})

const operationActive = computed(() => (
  state.value === 'initializing'
  || state.value === 'requesting'
  || state.value === 'recording'
  || state.value === 'transcribing'
))
const busy = computed(() => (
  operationActive.value
))
const idleState = computed(() => state.value === 'idle')
const preparingState = computed(() => (
  state.value === 'initializing' || state.value === 'requesting'
))
const waveformVisible = computed(() => (
  state.value === 'recording' || state.value === 'transcribing'
))
const waveformShellActive = computed(() => (
  operationActive.value
))
const errorState = computed(() => (
  state.value === 'denied'
  || state.value === 'unsupported'
  || state.value === 'unavailable'
  || state.value === 'error'
))
const buttonDisabled = computed(() => props.disabled && !operationActive.value)
const tooltipEnabled = computed(() => !errorState.value && !buttonDisabled.value)
const cancelButtonLabel = computed(() => {
  if (state.value === 'transcribing') return t('chat.voice.cancelTranscription')
  if (preparingState.value) return t('chat.voice.cancelRequest')
  return t('chat.voice.cancelRecording')
})
const confirmButtonLabel = computed(() => (
  preparingState.value ? t('chat.voice.preparing') : t('chat.voice.stop')
))
const buttonLabel = computed(() => {
  if (state.value === 'initializing') return t('chat.voice.preparing')
  if (state.value === 'requesting') return t('chat.voice.cancelRequest')
  if (state.value === 'recording') return t('chat.voice.stop')
  if (state.value === 'transcribing') return t('chat.voice.cancelTranscription')
  if (errorState.value) {
    if (state.value === 'unsupported') return t('chat.voice.errors.unsupported')
    if (state.value === 'denied') return t('chat.voice.errors.permissionDenied')
    if (state.value === 'unavailable') return t('chat.voice.errors.unavailable')
    return pendingTranscript.value ? t('chat.voice.retryInsert') : t('chat.voice.retry')
  }
  return t('chat.voice.start')
})
const announcement = computed(() => {
  if (state.value === 'initializing') return t('chat.voice.preparing')
  if (state.value === 'requesting') return t('chat.voice.requesting')
  if (state.value === 'recording') return t('chat.voice.recording')
  if (state.value === 'transcribing') return t('chat.voice.transcribing')
  // The visible error card already announces itself via role="alert".
  if (errorState.value) return ''
  return ''
})

watch(busy, (value) => emit('busy-change', value), { immediate: true, flush: 'sync' })
watch(() => props.contextKey, (next, previous) => {
  if (next !== previous) cancelOperation()
})
watch(() => props.disabled, (disabled) => {
  if (disabled && operationActive.value) cancelOperation()
})
watch(state, async (nextState, previousState) => {
  if (!waveformShellActive.value) {
    stopWaveformAnimation({ clearHistory: true })
    restoreComposerInteraction()
    const shouldRestoreFocus = (
      restoreTriggerFocusWhenReady || focusRecordingConfirmWhenReady
    ) && !componentUnmounting
    restoreTriggerFocusWhenReady = false
    focusRecordingConfirmWhenReady = false
    if (shouldRestoreFocus) {
      await nextTick()
      await nextTick()
      triggerRef.value?.focus({ preventScroll: true })
    }
    return
  }

  if (preparingState.value) {
    stopWaveformAnimation({ clearHistory: true })
    await nextTick()
    isolateComposerInteraction()
    if (focusRecordingConfirmWhenReady) {
      recordingCancelRef.value?.focus({ preventScroll: true })
    }
    return
  }

  if (nextState === 'transcribing') {
    stopWaveformAnimation({ clearHistory: false, keepResizeObserver: true })
    await nextTick()
    isolateComposerInteraction()
    drawWaveform()
    await nextTick()
    recordingCancelRef.value?.focus({ preventScroll: true })
    return
  }

  const shouldFocusConfirm = focusRecordingConfirmWhenReady
  focusRecordingConfirmWhenReady = false
  await nextTick()
  isolateComposerInteraction()
  startWaveformAnimation(previousState !== 'transcribing')
  if (shouldFocusConfirm) recordingConfirmRef.value?.focus({ preventScroll: true })
}, { flush: 'post' })

function restoreComposerInteraction(): void {
  for (const { element, inert, ariaHidden } of isolatedComposerElements.splice(0)) {
    if (inert === null) element.removeAttribute('inert')
    else element.setAttribute('inert', inert)
    if (ariaHidden === null) element.removeAttribute('aria-hidden')
    else element.setAttribute('aria-hidden', ariaHidden)
  }
}

function isolateComposerInteraction(): void {
  restoreComposerInteraction()
  const root = rootRef.value
  const composer = root?.closest<HTMLElement>('.chat-composer')
  if (!root || !composer) return

  const candidates = [
    ...Array.from(composer.children),
    ...Array.from(root.parentElement?.children ?? []),
  ]
  const seen = new Set<HTMLElement>()
  for (const candidate of candidates) {
    if (!(candidate instanceof HTMLElement) || seen.has(candidate)) continue
    seen.add(candidate)
    if (candidate === root || candidate.contains(root)) continue
    isolatedComposerElements.push({
      element: candidate,
      inert: candidate.getAttribute('inert'),
      ariaHidden: candidate.getAttribute('aria-hidden'),
    })
    candidate.setAttribute('inert', '')
    candidate.setAttribute('aria-hidden', 'true')
  }
}

function rememberRecordingFocus(): void {
  const activeElement = document.activeElement
  restoreTriggerFocusWhenReady ||= !!(
    waveformShellActive.value
    && activeElement instanceof HTMLElement
    && rootRef.value?.contains(activeElement)
  )
}

function cancelWaveformOperation(): void {
  restoreTriggerFocusWhenReady = true
  cancelOperation()
}

function confirmRecording(): void {
  restoreTriggerFocusWhenReady = true
  void finishRecording()
}

function roundedLine(
  context: CanvasRenderingContext2D,
  x: number,
  centerY: number,
  height: number,
): void {
  context.beginPath()
  context.moveTo(x, centerY - height / 2)
  context.lineTo(x, centerY + height / 2)
  context.stroke()
}

function drawWaveform(): void {
  const canvas = recordingCanvasRef.value
  if (!canvas) return
  const width = canvas.clientWidth
  const height = canvas.clientHeight
  if (width <= 0 || height <= 0) return

  let context: CanvasRenderingContext2D | null = null
  try {
    context = canvas.getContext('2d')
  } catch {
    return
  }
  if (!context) return

  const pixelRatio = Math.min(2, Math.max(1, window.devicePixelRatio || 1))
  const renderWidth = Math.round(width * pixelRatio)
  const renderHeight = Math.round(height * pixelRatio)
  if (canvas.width !== renderWidth || canvas.height !== renderHeight) {
    canvas.width = renderWidth
    canvas.height = renderHeight
  }
  context.setTransform(pixelRatio, 0, 0, pixelRatio, 0, 0)
  context.clearRect(0, 0, width, height)

  const centerY = height / 2
  const { trackStart, capacity } = waveformMetrics(width)
  const reducedMotion = motionPreference?.matches ?? false

  waveformHistory = resizeChatVoiceWaveformTrack(waveformHistory, capacity)
  const visibleHistory = waveformHistory
  const slidingProgress = waveformSampleProgress
  const styles = getComputedStyle(canvas)
  const primaryColor = styles.getPropertyValue('--chat-composer-primary-fg').trim() || '#0d0d0d'
  const tertiaryColor = styles.getPropertyValue('--chat-composer-tertiary-fg').trim() || '#8f8f8f'

  context.lineWidth = 2.6
  context.lineCap = 'round'
  visibleHistory.forEach((sample, index) => {
    const x = chatVoiceWaveformPosition(index, slidingProgress, trackStart)
    if (x < trackStart - CHAT_VOICE_WAVEFORM_STEP_PX) return
    if (sample === null) {
      context.fillStyle = tertiaryColor
      context.globalAlpha = 0.56
      context.beginPath()
      context.arc(x, centerY, 1.35, 0, Math.PI * 2)
      context.fill()
      return
    }

    const distanceFromRightEdge = visibleHistory.length - index - 1 + slidingProgress
    context.strokeStyle = primaryColor
    context.globalAlpha = distanceFromRightEdge < 0.55
      ? 0.22
      : distanceFromRightEdge < 1.55
        ? 0.48
        : distanceFromRightEdge < 2.55
          ? 0.7
          : 0.58
    roundedLine(
      context,
      x,
      centerY,
      sample,
    )
  })
  if (!reducedMotion) {
    const pendingX = chatVoiceWaveformPosition(
      visibleHistory.length,
      slidingProgress,
      trackStart,
    )
    context.strokeStyle = primaryColor
    context.globalAlpha = Math.min(0.22, Math.max(0, slidingProgress * 0.22))
    roundedLine(
      context,
      pendingX,
      centerY,
      chatVoiceWaveformHeight(waveformPendingLevel),
    )
  }
  context.globalAlpha = 1
}

function waveformMetrics(width: number): {
  trackStart: number
  capacity: number
} {
  const trackStart = Math.min(52, Math.max(18, width * 0.1))
  const trackEnd = Math.max(trackStart, width - 11)
  return {
    trackStart,
    capacity: chatVoiceWaveformCapacity(Math.max(0, trackEnd - trackStart)),
  }
}

function stopWaveformAnimation(options: {
  clearHistory: boolean
  keepResizeObserver?: boolean
} = { clearHistory: true }): void {
  if (waveformFrame !== null) cancelAnimationFrame(waveformFrame)
  waveformFrame = null
  if (!options.keepResizeObserver) {
    waveformResizeObserver?.disconnect()
    waveformResizeObserver = null
  }
  waveformLastSampleAt = 0
  if (options.clearHistory) {
    waveformSampleProgress = 0
    waveformPendingLevel = 0
    waveformHistory = []
  }
}

function startWaveformAnimation(clearHistory = true): void {
  stopWaveformAnimation({ clearHistory })
  const canvas = recordingCanvasRef.value
  if (!canvas) return
  const startedAt = performance.now()
  const reducedMotion = motionPreference?.matches ?? false
  const render = (timestamp: number) => {
    if (state.value !== 'recording') return
    let elapsedSinceSample = timestamp - waveformLastSampleAt
    while (elapsedSinceSample >= CHAT_VOICE_WAVEFORM_SAMPLE_INTERVAL_MS) {
      const { capacity } = waveformMetrics(canvas.clientWidth)
      waveformHistory = appendChatVoiceWaveformSample(
        waveformHistory,
        waveformPendingLevel,
        capacity,
      )
      waveformLastSampleAt += CHAT_VOICE_WAVEFORM_SAMPLE_INTERVAL_MS
      elapsedSinceSample = timestamp - waveformLastSampleAt
    }
    waveformSampleProgress = Math.min(
      1,
      Math.max(0, elapsedSinceSample / CHAT_VOICE_WAVEFORM_SAMPLE_INTERVAL_MS),
    )
    waveformPendingLevel = recorder.audioLevel.value
    drawWaveform()
    waveformFrame = requestAnimationFrame(render)
  }

  if (clearHistory) {
    const { capacity } = waveformMetrics(canvas.clientWidth)
    waveformHistory = resizeChatVoiceWaveformTrack([], capacity)
  }
  waveformSampleProgress = 0
  waveformPendingLevel = recorder.audioLevel.value
  waveformLastSampleAt = startedAt - CHAT_VOICE_WAVEFORM_SAMPLE_INTERVAL_MS
  if (reducedMotion) {
    const { capacity } = waveformMetrics(canvas.clientWidth)
    waveformHistory = appendChatVoiceWaveformSample(
      waveformHistory,
      recorder.audioLevel.value,
      capacity,
    )
  }
  drawWaveform()
  if (!reducedMotion && typeof requestAnimationFrame === 'function') {
    waveformFrame = requestAnimationFrame(render)
  }
  if (typeof ResizeObserver !== 'undefined') {
    waveformResizeObserver = new ResizeObserver(() => drawWaveform())
    waveformResizeObserver.observe(canvas)
  }
}

function onMotionPreferenceChange(): void {
  if (state.value !== 'recording') return
  startWaveformAnimation(false)
}

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

function errorStateFor(error: unknown): VoiceInputState {
  if (error instanceof AudioRecorderError) {
    if (error.code === 'AUDIO_RECORDING_UNSUPPORTED') return 'unsupported'
    if (error.code === 'MICROPHONE_PERMISSION_DENIED') return 'denied'
  }
  if (error instanceof ChatAPIError) {
    const code = String(error.code || '')
    if (code === 'TRANSCRIPTION_UNAVAILABLE' || error.status === 503) return 'unavailable'
  }
  return 'error'
}

function showError(error: unknown): void {
  rememberRecordingFocus()
  pendingTranscript.value = ''
  errorMessage.value = errorText(error)
  state.value = errorStateFor(error)
}

function showUnavailable(): void {
  rememberRecordingFocus()
  pendingTranscript.value = ''
  errorMessage.value = t('chat.voice.errors.unavailable')
  state.value = 'unavailable'
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

function startRecording(focusConfirm = false): Promise<void> {
  if (props.disabled) return Promise.resolve()
  if (initializationPromise) return initializationPromise
  if (operationActive.value) return Promise.resolve()
  cancelOperation()
  focusRecordingConfirmWhenReady = focusConfirm
  const version = ++operationVersion
  recordingIdempotencyKey = createChatIdempotencyKey()
  errorMessage.value = ''
  state.value = 'initializing'
  const pending = (async () => {
    try {
      let capabilityState: VoiceCapabilityState | undefined = props.capabilityState
      if (capabilityState !== 'ready') {
        capabilityState = await props.initializeCapability?.()
      }
      if (version !== operationVersion) return
      if (capabilityState !== 'ready') {
        showUnavailable()
        return
      }
      state.value = 'requesting'
      await recorder.start()
      if (version !== operationVersion) return
      state.value = 'recording'
    } catch (error) {
      if (version !== operationVersion || isCancelled(error)) return
      showError(error)
    } finally {
      if (version === operationVersion) initializationPromise = null
    }
  })()
  initializationPromise = pending
  return pending
}

async function finishRecording(): Promise<void> {
  if (state.value !== 'recording') return
  rememberRecordingFocus()
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
  rememberRecordingFocus()
  operationVersion += 1
  initializationPromise = null
  uploadController.value?.abort()
  uploadController.value = null
  recorder.cancel()
  recordingIdempotencyKey = ''
  pendingTranscript.value = ''
  errorMessage.value = ''
  focusRecordingConfirmWhenReady = false
  state.value = 'idle'
}

function dismissError(): void {
  pendingTranscript.value = ''
  errorMessage.value = ''
  state.value = 'idle'
}

function toggleRecording(event?: MouseEvent | KeyboardEvent): void {
  if (state.value === 'recording') {
    void finishRecording()
    return
  }
  if (preparingState.value || state.value === 'transcribing') {
    cancelWaveformOperation()
    return
  }
  if (errorState.value && pendingTranscript.value) {
    retryPendingTranscript()
    return
  }
  const focusConfirm = event instanceof KeyboardEvent
    || (event instanceof MouseEvent && event.detail === 0)
  void startRecording(focusConfirm)
}

function startFromExternal(): void {
  if (buttonDisabled.value || operationActive.value) return
  if (errorState.value && pendingTranscript.value) {
    retryPendingTranscript()
    return
  }
  void startRecording()
}

function onDictationShortcut(event: KeyboardEvent): void {
  const shortcutAnchor = triggerRef.value ?? recordingConfirmRef.value ?? recordingCancelRef.value
  if (event.key === 'Escape' && operationActive.value) {
    if (!chatControlShortcutIsInScope(event, shortcutAnchor)) return
    event.preventDefault()
    event.stopPropagation()
    cancelWaveformOperation()
    return
  }
  if (
    event.repeat
    || event.code !== 'KeyD'
    || !event.ctrlKey
    || !event.shiftKey
    || event.altKey
    || event.metaKey
    || buttonDisabled.value
    || !chatControlShortcutIsInScope(event, shortcutAnchor)
  ) return
  event.preventDefault()
  toggleRecording(event)
}

onMounted(() => {
  window.addEventListener('keydown', onDictationShortcut)
  if (typeof window.matchMedia === 'function') {
    motionPreference = window.matchMedia('(prefers-reduced-motion: reduce)')
    motionPreference.addEventListener?.('change', onMotionPreferenceChange)
  }
})
onBeforeUnmount(() => {
  componentUnmounting = true
  window.removeEventListener('keydown', onDictationShortcut)
  motionPreference?.removeEventListener?.('change', onMotionPreferenceChange)
  motionPreference = null
  stopWaveformAnimation()
  restoreComposerInteraction()
  cancelOperation()
})

defineExpose({ cancel: cancelOperation, start: startFromExternal, activate: toggleRecording })
</script>

<style scoped>
.chat-voice-input {
  position: relative;
  display: flex;
  width: 36px;
  height: 36px;
  min-width: 36px;
  flex: 0 0 36px;
  align-items: center;
}

.chat-voice-input--initializing,
.chat-voice-input--requesting,
.chat-voice-input--recording,
.chat-voice-input--transcribing {
  position: static;
}

.chat-voice-input__recording-shell {
  position: absolute;
  inset: 0;
  z-index: 20;
  display: grid;
  box-sizing: border-box;
  grid-template-columns: 36px minmax(0, 1fr) 36px 36px;
  grid-template-rows: 36px;
  align-items: center;
  column-gap: 6px;
  height: 52px;
  border-radius: 28px;
  padding: 8px;
  color: var(--chat-composer-primary-fg, #0d0d0d);
  background: transparent;
  box-shadow: none;
}

.chat-voice-input__recording-plus {
  display: grid;
  width: 36px;
  height: 36px;
  place-items: center;
  color: color-mix(
    in srgb,
    var(--chat-composer-primary-fg, #0d0d0d) 30%,
    transparent
  );
  pointer-events: none;
}

.chat-voice-input__waveform {
  display: block;
  width: 100%;
  min-width: 0;
  height: 44px;
}

.chat-voice-input__waveform--hidden {
  visibility: hidden;
}

.chat-voice-input__recording-action {
  display: grid;
  width: 36px;
  height: 36px;
  place-items: center;
  border: 0;
  border-radius: 50%;
  padding: 0;
  color: var(--chat-composer-primary-fg, #0d0d0d);
  background: transparent;
  transition: background-color 140ms ease, transform 140ms ease;
}

.chat-voice-input__recording-action:active {
  transform: scale(0.94);
}

.chat-voice-input__recording-action:disabled {
  cursor: default;
  opacity: 1;
}

.chat-voice-input__recording-action--muted {
  color: var(--chat-composer-tertiary-fg, #8f8f8f);
}

.chat-voice-input__recording-action:focus-visible {
  outline: 2px solid var(--chat-composer-primary-fg, #0d0d0d);
  outline-offset: -2px;
}

.chat-voice-input__recording-progress {
  display: grid;
  width: 36px;
  height: 36px;
  place-items: center;
  color: var(--chat-composer-primary-fg, #0d0d0d);
}

.chat-voice-input__recording-spinner {
  width: 18px;
  height: 18px;
  border: 3px dotted currentColor;
  border-radius: 50%;
  animation: chat-voice-spin 780ms linear infinite;
}

:global(.chat-composer:has(.chat-voice-input--initializing)),
:global(.chat-composer:has(.chat-voice-input--requesting)),
:global(.chat-composer:has(.chat-voice-input--recording)),
:global(.chat-composer:has(.chat-voice-input--transcribing)) {
  min-height: 52px;
  height: 52px;
  max-height: 52px;
  grid-template-rows: 36px;
  border-radius: 28px;
  padding: 8px;
  overflow: hidden;
}

:global(.chat-composer:has(.chat-voice-input--initializing) > .chat-composer__attachments),
:global(.chat-composer:has(.chat-voice-input--initializing) > .chat-composer__leading),
:global(.chat-composer:has(.chat-voice-input--initializing) > .chat-composer__input-shell),
:global(.chat-composer:has(.chat-voice-input--initializing) > .chat-composer__empty-action),
:global(.chat-composer:has(.chat-voice-input--initializing) > .chat-composer__action),
:global(.chat-composer:has(.chat-voice-input--initializing) > .chat-composer__trailing > :not(.chat-voice-input)),
:global(.chat-composer:has(.chat-voice-input--requesting) > .chat-composer__attachments),
:global(.chat-composer:has(.chat-voice-input--requesting) > .chat-composer__leading),
:global(.chat-composer:has(.chat-voice-input--requesting) > .chat-composer__input-shell),
:global(.chat-composer:has(.chat-voice-input--requesting) > .chat-composer__empty-action),
:global(.chat-composer:has(.chat-voice-input--requesting) > .chat-composer__action),
:global(.chat-composer:has(.chat-voice-input--requesting) > .chat-composer__trailing > :not(.chat-voice-input)),
:global(.chat-composer:has(.chat-voice-input--recording) > .chat-composer__attachments),
:global(.chat-composer:has(.chat-voice-input--recording) > .chat-composer__leading),
:global(.chat-composer:has(.chat-voice-input--recording) > .chat-composer__input-shell),
:global(.chat-composer:has(.chat-voice-input--recording) > .chat-composer__empty-action),
:global(.chat-composer:has(.chat-voice-input--recording) > .chat-composer__action),
:global(.chat-composer:has(.chat-voice-input--recording) > .chat-composer__trailing > :not(.chat-voice-input)),
:global(.chat-composer:has(.chat-voice-input--transcribing) > .chat-composer__attachments),
:global(.chat-composer:has(.chat-voice-input--transcribing) > .chat-composer__leading),
:global(.chat-composer:has(.chat-voice-input--transcribing) > .chat-composer__input-shell),
:global(.chat-composer:has(.chat-voice-input--transcribing) > .chat-composer__empty-action),
:global(.chat-composer:has(.chat-voice-input--transcribing) > .chat-composer__action),
:global(.chat-composer:has(.chat-voice-input--transcribing) > .chat-composer__trailing > :not(.chat-voice-input)) {
  visibility: hidden;
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
    border-color 160ms ease,
    color 160ms ease,
    background-color 160ms ease,
    opacity 160ms ease;
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

@media (max-width: 720px) {
  .chat-voice-input {
    width: 36px;
    height: 36px;
    min-width: 36px;
    flex-basis: 36px;
  }

  .chat-voice-input__trigger {
    width: 36px;
    height: 36px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .chat-voice-input__trigger,
  .chat-voice-input__recording-action {
    transition: none;
  }

  .chat-voice-input__recording-spinner {
    animation: none;
  }
}
</style>

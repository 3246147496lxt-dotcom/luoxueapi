import { computed, onBeforeUnmount, ref } from 'vue'

export const CHAT_AUDIO_MAX_DURATION_MS = 60_000
export const CHAT_AUDIO_MAX_BYTES = 10 * 1024 * 1024

const MIME_TYPE_CANDIDATES = [
  'audio/webm;codecs=opus',
  'audio/mp4',
  'audio/ogg;codecs=opus',
] as const

export type AudioRecorderState = 'idle' | 'requesting' | 'recording' | 'stopping' | 'error'

export type AudioRecorderErrorCode =
  | 'AUDIO_RECORDING_UNSUPPORTED'
  | 'MICROPHONE_PERMISSION_DENIED'
  | 'MICROPHONE_NOT_FOUND'
  | 'MICROPHONE_BUSY'
  | 'RECORDING_FAILED'
  | 'EMPTY_AUDIO'
  | 'AUDIO_TOO_LARGE'
  | 'RECORDING_CANCELLED'

export class AudioRecorderError extends Error {
  readonly code: AudioRecorderErrorCode

  constructor(code: AudioRecorderErrorCode, message: string) {
    super(message)
    this.name = 'AudioRecorderError'
    this.code = code
  }
}

export interface RecordedAudio {
  blob: Blob
  durationMs: number
  mimeType: string
}

interface UseAudioRecorderOptions {
  maxDurationMs?: number | (() => number)
  maxBytes?: number | (() => number)
  acceptedMimeTypes?: readonly string[] | (() => readonly string[])
  onMaximumDuration?: () => void
  onError?: (error: AudioRecorderError) => void
}

function stopTracks(stream: MediaStream | null): void {
  stream?.getTracks().forEach((track) => track.stop())
}

function cancelledError(): AudioRecorderError {
  return new AudioRecorderError('RECORDING_CANCELLED', 'Audio recording was cancelled.')
}

function optionValue<T>(value: T | (() => T) | undefined, fallback: T): T {
  return typeof value === 'function' ? (value as () => T)() : (value ?? fallback)
}

function normalizeCaptureError(error: unknown): AudioRecorderError {
  const name = error && typeof error === 'object'
    ? String((error as { name?: unknown }).name ?? '')
    : ''
  if (name === 'NotAllowedError' || name === 'SecurityError') {
    return new AudioRecorderError(
      'MICROPHONE_PERMISSION_DENIED',
      'Microphone permission was denied.',
    )
  }
  if (name === 'NotFoundError' || name === 'DevicesNotFoundError') {
    return new AudioRecorderError('MICROPHONE_NOT_FOUND', 'No microphone was found.')
  }
  if (name === 'NotReadableError' || name === 'TrackStartError') {
    return new AudioRecorderError('MICROPHONE_BUSY', 'The microphone is unavailable.')
  }
  return new AudioRecorderError('RECORDING_FAILED', 'Unable to start audio recording.')
}

export function audioRecordingSupported(): boolean {
  return typeof navigator !== 'undefined'
    && typeof navigator.mediaDevices?.getUserMedia === 'function'
    && typeof MediaRecorder !== 'undefined'
}

function baseMimeType(mimeType: string): string {
  return mimeType.split(';', 1)[0]?.trim().toLowerCase() ?? ''
}

export function preferredAudioMimeType(acceptedMimeTypes: readonly string[] = []): string {
  if (typeof MediaRecorder === 'undefined') return ''
  if (typeof MediaRecorder.isTypeSupported !== 'function') return ''
  const acceptedBases = new Set(acceptedMimeTypes.map(baseMimeType).filter(Boolean))
  return MIME_TYPE_CANDIDATES.find((mimeType) => (
    (acceptedBases.size === 0 || acceptedBases.has(baseMimeType(mimeType)))
    && MediaRecorder.isTypeSupported(mimeType)
  )) ?? ''
}

export function useAudioRecorder(options: UseAudioRecorderOptions = {}) {
  const state = ref<AudioRecorderState>('idle')
  const elapsedMs = ref(0)
  const audioLevel = ref(0)
  const error = ref<AudioRecorderError | null>(null)
  const supported = computed(audioRecordingSupported)

  let stream: MediaStream | null = null
  let mediaRecorder: MediaRecorder | null = null
  let audioContext: AudioContext | null = null
  let audioSource: MediaStreamAudioSourceNode | null = null
  let audioAnalyser: AnalyserNode | null = null
  let audioSamples: Uint8Array | null = null
  let audioAnalysisFrame: number | null = null
  let chunks: Blob[] = []
  let startedAt = 0
  let requestedDurationMs = 0
  let sessionVersion = 0
  let elapsedTimer: ReturnType<typeof setInterval> | null = null
  let maximumTimer: ReturnType<typeof setTimeout> | null = null
  let stopResolve: ((recording: RecordedAudio) => void) | null = null
  let stopReject: ((reason: AudioRecorderError) => void) | null = null
  let activeMaxDurationMs = CHAT_AUDIO_MAX_DURATION_MS
  let activeMaxBytes = CHAT_AUDIO_MAX_BYTES

  function clearTimers(): void {
    if (elapsedTimer !== null) clearInterval(elapsedTimer)
    if (maximumTimer !== null) clearTimeout(maximumTimer)
    elapsedTimer = null
    maximumTimer = null
  }

  function releaseAudioAnalysis(): void {
    if (audioAnalysisFrame !== null && typeof cancelAnimationFrame === 'function') {
      cancelAnimationFrame(audioAnalysisFrame)
    }
    audioAnalysisFrame = null
    try {
      audioSource?.disconnect()
      audioAnalyser?.disconnect()
    } catch {
      // Audio nodes may already be disconnected when the capture device disappears.
    }
    audioSource = null
    audioAnalyser = null
    audioSamples = null
    audioLevel.value = 0
    const context = audioContext
    audioContext = null
    if (context) {
      try {
        void context.close().catch(() => undefined)
      } catch {
        // Closing an already-closed context is harmless for recorder cleanup.
      }
    }
  }

  function startAudioAnalysis(capturedStream: MediaStream): void {
    if (typeof window === 'undefined' || typeof requestAnimationFrame !== 'function') return
    const AudioContextConstructor = window.AudioContext
      ?? (window as typeof window & { webkitAudioContext?: typeof AudioContext }).webkitAudioContext
    if (!AudioContextConstructor) return

    try {
      audioContext = new AudioContextConstructor()
      audioSource = audioContext.createMediaStreamSource(capturedStream)
      audioAnalyser = audioContext.createAnalyser()
      audioAnalyser.fftSize = 256
      audioAnalyser.smoothingTimeConstant = 0.72
      audioSamples = new Uint8Array(audioAnalyser.fftSize)
      audioSource.connect(audioAnalyser)
      if (audioContext.state === 'suspended') void audioContext.resume().catch(() => undefined)

      const sample = () => {
        if (!audioAnalyser || !audioSamples) return
        try {
          audioAnalyser.getByteTimeDomainData(audioSamples)
          let squaredTotal = 0
          for (const value of audioSamples) {
            const centered = (value - 128) / 128
            squaredTotal += centered * centered
          }
          const rms = Math.sqrt(squaredTotal / audioSamples.length)
          const normalized = Math.min(1, Math.max(0, (rms - 0.008) * 7.5))
          const smoothing = normalized > audioLevel.value ? 0.52 : 0.18
          audioLevel.value += (normalized - audioLevel.value) * smoothing
          audioAnalysisFrame = requestAnimationFrame(sample)
        } catch {
          // Analyser failure must not interrupt the MediaRecorder session.
          releaseAudioAnalysis()
        }
      }
      audioAnalysisFrame = requestAnimationFrame(sample)
    } catch {
      // Audio analysis is progressive enhancement; recording must still work without it.
      releaseAudioAnalysis()
    }
  }

  function releaseStream(): void {
    releaseAudioAnalysis()
    stopTracks(stream)
    stream = null
  }

  function resetRecorder(): void {
    if (mediaRecorder) {
      mediaRecorder.ondataavailable = null
      mediaRecorder.onstop = null
      mediaRecorder.onerror = null
    }
    mediaRecorder = null
    chunks = []
    releaseStream()
  }

  function rejectPendingStop(reason: AudioRecorderError): void {
    const reject = stopReject
    stopResolve = null
    stopReject = null
    reject?.(reason)
  }

  function reportError(nextError: AudioRecorderError): void {
    clearTimers()
    rejectPendingStop(nextError)
    resetRecorder()
    error.value = nextError
    state.value = 'error'
    options.onError?.(nextError)
  }

  function elapsedDuration(): number {
    if (!startedAt) return 0
    return Math.min(activeMaxDurationMs, Math.max(0, Date.now() - startedAt))
  }

  async function start(): Promise<void> {
    cancel()
    error.value = null
    if (!audioRecordingSupported()) {
      const unsupported = new AudioRecorderError(
        'AUDIO_RECORDING_UNSUPPORTED',
        'Audio recording is not supported in this browser.',
      )
      error.value = unsupported
      state.value = 'error'
      throw unsupported
    }

    const acceptedMimeTypes = optionValue(options.acceptedMimeTypes, [])
    activeMaxDurationMs = Math.min(
      CHAT_AUDIO_MAX_DURATION_MS,
      optionValue(options.maxDurationMs, CHAT_AUDIO_MAX_DURATION_MS),
    )
    activeMaxBytes = Math.min(
      CHAT_AUDIO_MAX_BYTES,
      optionValue(options.maxBytes, CHAT_AUDIO_MAX_BYTES),
    )
    const mimeType = preferredAudioMimeType(acceptedMimeTypes)
    if (acceptedMimeTypes.length > 0 && !mimeType) {
      const unsupported = new AudioRecorderError(
        'AUDIO_RECORDING_UNSUPPORTED',
        'This browser cannot record an audio format accepted by the server.',
      )
      error.value = unsupported
      state.value = 'error'
      throw unsupported
    }

    const version = ++sessionVersion
    state.value = 'requesting'
    let capturedStream: MediaStream
    try {
      capturedStream = await navigator.mediaDevices.getUserMedia({
        audio: {
          channelCount: 1,
          echoCancellation: true,
          noiseSuppression: true,
          autoGainControl: true,
        },
      })
    } catch (captureError) {
      if (version !== sessionVersion) throw cancelledError()
      const normalized = normalizeCaptureError(captureError)
      error.value = normalized
      state.value = 'error'
      throw normalized
    }

    if (version !== sessionVersion) {
      stopTracks(capturedStream)
      throw cancelledError()
    }

    stream = capturedStream
    startAudioAnalysis(capturedStream)
    chunks = []
    try {
      mediaRecorder = mimeType
        ? new MediaRecorder(capturedStream, { mimeType })
        : new MediaRecorder(capturedStream)
    } catch (recorderError) {
      releaseStream()
      const normalized = normalizeCaptureError(recorderError)
      error.value = normalized
      state.value = 'error'
      throw normalized
    }

    const activeRecorder = mediaRecorder
    activeRecorder.ondataavailable = (event: BlobEvent) => {
      if (version === sessionVersion && event.data.size > 0) chunks.push(event.data)
    }
    activeRecorder.onerror = () => {
      if (version !== sessionVersion) return
      reportError(new AudioRecorderError('RECORDING_FAILED', 'Audio recording failed.'))
    }
    activeRecorder.onstop = () => {
      if (version !== sessionVersion) return
      clearTimers()
      const durationMs = requestedDurationMs || elapsedDuration()
      const blob = new Blob(chunks, {
        type: activeRecorder.mimeType || mimeType || chunks[0]?.type || 'audio/webm',
      })
      const resolve = stopResolve
      const reject = stopReject
      stopResolve = null
      stopReject = null
      resetRecorder()
      elapsedMs.value = durationMs
      state.value = 'idle'
      if (!resolve) {
        const interrupted = new AudioRecorderError(
          'RECORDING_FAILED',
          'Audio recording stopped unexpectedly.',
        )
        error.value = interrupted
        state.value = 'error'
        options.onError?.(interrupted)
        return
      }
      if (blob.size <= 0) {
        const empty = new AudioRecorderError('EMPTY_AUDIO', 'The audio recording is empty.')
        error.value = empty
        state.value = 'error'
        reject?.(empty)
        return
      }
      if (blob.size > activeMaxBytes) {
        const tooLarge = new AudioRecorderError('AUDIO_TOO_LARGE', 'The audio recording is too large.')
        error.value = tooLarge
        state.value = 'error'
        reject?.(tooLarge)
        return
      }
      resolve({ blob, durationMs, mimeType: blob.type })
    }

    try {
      activeRecorder.start(1_000)
    } catch {
      const startError = new AudioRecorderError('RECORDING_FAILED', 'Unable to start audio recording.')
      reportError(startError)
      throw startError
    }
    startedAt = Date.now()
    requestedDurationMs = 0
    elapsedMs.value = 0
    state.value = 'recording'
    elapsedTimer = setInterval(() => {
      elapsedMs.value = elapsedDuration()
    }, 250)
    maximumTimer = setTimeout(() => {
      if (version !== sessionVersion || state.value !== 'recording') return
      elapsedMs.value = activeMaxDurationMs
      options.onMaximumDuration?.()
    }, activeMaxDurationMs)
  }

  function stop(): Promise<RecordedAudio> {
    if (!mediaRecorder || state.value !== 'recording') {
      return Promise.reject(new AudioRecorderError('RECORDING_FAILED', 'No recording is active.'))
    }
    clearTimers()
    requestedDurationMs = elapsedDuration()
    elapsedMs.value = requestedDurationMs
    state.value = 'stopping'

    return new Promise<RecordedAudio>((resolve, reject) => {
      stopResolve = resolve
      stopReject = reject
      try {
        mediaRecorder?.stop()
      } catch {
        reportError(new AudioRecorderError('RECORDING_FAILED', 'Unable to stop audio recording.'))
      }
    })
  }

  function cancel(): void {
    sessionVersion += 1
    clearTimers()
    rejectPendingStop(cancelledError())
    if (mediaRecorder && mediaRecorder.state !== 'inactive') {
      mediaRecorder.ondataavailable = null
      mediaRecorder.onstop = null
      mediaRecorder.onerror = null
      try {
        mediaRecorder.stop()
      } catch {
        // The stream is released below even when the recorder already stopped.
      }
    }
    resetRecorder()
    startedAt = 0
    requestedDurationMs = 0
    elapsedMs.value = 0
    error.value = null
    state.value = 'idle'
  }

  onBeforeUnmount(cancel)

  return {
    state,
    elapsedMs,
    audioLevel,
    error,
    supported,
    start,
    stop,
    cancel,
  }
}

import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'

const apiMocks = vi.hoisted(() => ({
  createChatIdempotencyKey: vi.fn(),
  isAbortError: vi.fn(),
  transcribeChatAudio: vi.fn(),
}))

vi.mock('@/api/chat', () => {
  class MockChatAPIError extends Error {
    status: number
    code: string | number | null

    constructor(message: string, options: { status?: number; code?: string | number | null } = {}) {
      super(message)
      this.name = 'ChatAPIError'
      this.status = options.status ?? 0
      this.code = options.code ?? null
    }
  }
  return {
    ChatAPIError: MockChatAPIError,
    createChatIdempotencyKey: apiMocks.createChatIdempotencyKey,
    isAbortError: apiMocks.isAbortError,
    transcribeChatAudio: apiMocks.transcribeChatAudio,
  }
})

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const messages: Record<string, string> = {
    'chat.voice.errors.dailyQuotaExceeded': '今日语音额度已用完，将在下一个北京时间 08:00 重置；文字聊天仍可继续使用。',
  }
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

import { ChatAPIError } from '@/api/chat'
import ChatVoiceInput from '../ChatVoiceInput.vue'

const COMPONENT_SOURCE = readFileSync(
  resolve(process.cwd(), 'src/components/chat/ChatVoiceInput.vue'),
  'utf8',
)
const COMPONENT_STYLE = COMPONENT_SOURCE.match(/<style scoped>([\s\S]*?)<\/style>/)?.[1] ?? ''

const IconStub = {
  props: ['name', 'size'],
  template: '<span :data-icon="name" :data-size="size" />',
}

let nextAudioBlob: Blob
let trackStop: ReturnType<typeof vi.fn>
let getUserMedia: ReturnType<typeof vi.fn>
let originalMediaDevices: MediaDevices | undefined
let wrapper: VueWrapper | undefined

class FakeMediaRecorder {
  static instances: FakeMediaRecorder[] = []
  static isTypeSupported = vi.fn((mimeType: string) => mimeType === 'audio/webm;codecs=opus')

  readonly stream: MediaStream
  readonly mimeType: string
  state: RecordingState = 'inactive'
  ondataavailable: ((event: BlobEvent) => void) | null = null
  onerror: ((event: Event) => void) | null = null
  onstop: ((event: Event) => void) | null = null

  constructor(stream: MediaStream, options?: MediaRecorderOptions) {
    this.stream = stream
    this.mimeType = options?.mimeType ?? ''
    FakeMediaRecorder.instances.push(this)
  }

  start(): void {
    this.state = 'recording'
  }

  stop(): void {
    if (this.state === 'inactive') throw new DOMException('Inactive recorder', 'InvalidStateError')
    this.state = 'inactive'
    this.ondataavailable?.({ data: nextAudioBlob } as BlobEvent)
    this.onstop?.(new Event('stop'))
  }
}

function mountVoiceInput(props: Record<string, unknown> = {}) {
  wrapper = mount(ChatVoiceInput, {
    props: {
      contextKey: 'user-7:new',
      onTranscribed: (_text: string, acknowledge: (inserted: boolean) => void) => acknowledge(true),
      ...props,
    },
    global: { stubs: { Icon: IconStub } },
  })
  return wrapper
}

describe('ChatVoiceInput', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-07T00:00:00Z'))
    vi.resetAllMocks()
    nextAudioBlob = new Blob(['recorded voice'], { type: 'audio/webm;codecs=opus' })
    trackStop = vi.fn()
    const stream = {
      getTracks: () => [{ stop: trackStop }],
    } as unknown as MediaStream
    getUserMedia = vi.fn().mockResolvedValue(stream)
    originalMediaDevices = navigator.mediaDevices
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: { getUserMedia },
    })
    FakeMediaRecorder.instances = []
    vi.stubGlobal('MediaRecorder', FakeMediaRecorder)
    apiMocks.createChatIdempotencyKey.mockReturnValue('11111111-2222-4333-8444-555555555555')
    apiMocks.isAbortError.mockImplementation((error: unknown) => (
      !!error && typeof error === 'object' && (error as { name?: unknown }).name === 'AbortError'
    ))
    apiMocks.transcribeChatAudio.mockResolvedValue({ text: '转写文字' })
  })

  afterEach(() => {
    wrapper?.unmount()
    wrapper = undefined
    document.body.innerHTML = ''
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: originalMediaDevices,
    })
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('uses the dictation tooltip and supports its advertised keyboard shortcut', async () => {
    const view = mountVoiceInput()
    const trigger = view.get('[data-test="chat-voice-trigger"]')
    const tooltip = document.body.querySelector<HTMLElement>(
      '[data-ui-portal="chat-control-tooltip"][role="tooltip"]',
    )

    expect(tooltip).not.toBeNull()

    expect(trigger.get('[data-icon="chatMicrophone"]').attributes('data-icon'))
      .toBe('chatMicrophone')
    expect(trigger.get('[data-icon="chatMicrophone"]').attributes('data-size'))
      .toBe('md')
    expect(COMPONENT_STYLE).toContain('color: var(--chat-composer-secondary-fg, #0d0d0d);')
    expect(COMPONENT_STYLE).toContain('background: var(--chat-composer-secondary-hover, rgb(0 0 0 / 5%));')
    expect(trigger.attributes('aria-label')).toBe('chat.voice.start')
    expect(trigger.attributes('aria-keyshortcuts')).toBe('Control+Shift+D')
    expect(trigger.attributes('aria-describedby')).toContain(
      view.get('.sr-only:not([role="status"])').attributes('id'),
    )
    expect(trigger.attributes('aria-describedby')).not.toContain(
      view.get('[role="status"]').attributes('id'),
    )
    expect(tooltip?.textContent).toContain('chat.voice.dictation')
    expect(tooltip?.textContent).toContain('D')

    const shortcut = new KeyboardEvent('keydown', {
      code: 'KeyD',
      ctrlKey: true,
      shiftKey: true,
      bubbles: true,
      cancelable: true,
    })
    window.dispatchEvent(shortcut)
    await flushPromises()

    expect(shortcut.defaultPrevented).toBe(true)
    expect(view.attributes('data-state')).toBe('recording')
  })

  it('can start the same truthful transcription flow from the voice action', async () => {
    const view = mountVoiceInput()
    const api = view.vm as unknown as { start: () => void }

    api.start()
    await flushPromises()

    expect(view.attributes('data-state')).toBe('recording')
  })

  it('records with the preferred Opus format, uploads once, and emits text without sending', async () => {
    const view = mountVoiceInput()
    const trigger = view.get('[data-test="chat-voice-trigger"]')

    await trigger.trigger('click')
    await flushPromises()
    expect(view.attributes('data-state')).toBe('recording')
    expect(trigger.attributes('aria-pressed')).toBe('true')
    expect(FakeMediaRecorder.instances[0]?.mimeType).toBe('audio/webm;codecs=opus')

    await vi.advanceTimersByTimeAsync(1_250)
    expect(trigger.text()).toContain('00:01')

    await trigger.trigger('click')
    await flushPromises()

    expect(trackStop).toHaveBeenCalledOnce()
    expect(apiMocks.transcribeChatAudio).toHaveBeenCalledTimes(1)
    const [blob, options] = apiMocks.transcribeChatAudio.mock.calls[0]
    expect(blob).toBeInstanceOf(Blob)
    expect(blob.type).toBe('audio/webm;codecs=opus')
    expect(options).toMatchObject({
      idempotencyKey: '11111111-2222-4333-8444-555555555555',
    })
    expect(options.signal).toBeInstanceOf(AbortSignal)
    expect(view.emitted('transcribed')?.[0]?.[0]).toBe('转写文字')
    expect(view.emitted('transcribed')?.[0]?.[1]).toEqual(expect.any(Function))
    expect(view.attributes('data-state')).toBe('idle')
  })

  it('explains daily voice quota exhaustion without implying text chat is blocked', async () => {
    apiMocks.transcribeChatAudio.mockRejectedValueOnce(new ChatAPIError(
      'Daily voice transcription limit reached',
      { status: 429, code: 'TRANSCRIPTION_DAILY_QUOTA_EXCEEDED' },
    ))
    const view = mountVoiceInput()

    await view.get('[data-test="chat-voice-trigger"]').trigger('click')
    await flushPromises()
    await view.get('[data-test="chat-voice-trigger"]').trigger('click')
    await flushPromises()

    expect(view.attributes('data-state')).toBe('error')
    expect(view.get('[data-test="chat-voice-error"]').text()).toContain(
      '今日语音额度已用完，将在下一个北京时间 08:00 重置；文字聊天仍可继续使用。',
    )
  })

  it('keeps an ordinary 429 on the generic rate-limit message', async () => {
    apiMocks.transcribeChatAudio.mockRejectedValueOnce(new ChatAPIError(
      'Voice transcription rate limit reached',
      { status: 429, code: 'TRANSCRIPTION_RATE_LIMITED' },
    ))
    const view = mountVoiceInput()

    await view.get('[data-test="chat-voice-trigger"]').trigger('click')
    await flushPromises()
    await view.get('[data-test="chat-voice-trigger"]').trigger('click')
    await flushPromises()

    const message = view.get('[data-test="chat-voice-error"]').text()
    expect(message).toContain('chat.voice.errors.rateLimited')
    expect(message).not.toContain('今日语音额度已用完')
  })

  it('shows a localized permission error and remains retryable', async () => {
    getUserMedia.mockRejectedValue(Object.assign(new Error('blocked'), { name: 'NotAllowedError' }))
    const view = mountVoiceInput()

    await view.get('[data-test="chat-voice-trigger"]').trigger('click')
    await flushPromises()

    expect(view.attributes('data-state')).toBe('error')
    expect(view.get('[data-test="chat-voice-error"]').text())
      .toContain('chat.voice.errors.permissionDenied')
    expect(view.get('[data-test="chat-voice-trigger"]').attributes('aria-label'))
      .toBe('chat.voice.retry')

    await view.setProps({ contextKey: 'user-7:conversation-2' })

    expect(view.attributes('data-state')).toBe('idle')
    expect(view.find('[data-test="chat-voice-error"]').exists()).toBe(false)
  })

  it('cancels a pending permission request when its chat context changes', async () => {
    let resolveCapture: ((stream: MediaStream) => void) | undefined
    const capturedStream = {
      getTracks: () => [{ stop: trackStop }],
    } as unknown as MediaStream
    getUserMedia.mockReturnValue(new Promise<MediaStream>((resolve) => {
      resolveCapture = resolve
    }))
    const view = mountVoiceInput()

    await view.get('[data-test="chat-voice-trigger"]').trigger('click')
    await nextTick()
    expect(view.attributes('data-state')).toBe('requesting')

    await view.setProps({ contextKey: 'user-7:conversation-2' })
    resolveCapture?.(capturedStream)
    await flushPromises()

    expect(trackStop).toHaveBeenCalledOnce()
    expect(FakeMediaRecorder.instances).toHaveLength(0)
    expect(apiMocks.transcribeChatAudio).not.toHaveBeenCalled()
    expect(view.attributes('data-state')).toBe('idle')
  })

  it('automatically stops at 60 seconds and begins transcription', async () => {
    const view = mountVoiceInput()
    await view.get('[data-test="chat-voice-trigger"]').trigger('click')
    await flushPromises()

    await vi.advanceTimersByTimeAsync(60_000)
    await flushPromises()

    expect(apiMocks.transcribeChatAudio).toHaveBeenCalledOnce()
    expect(view.emitted('transcribed')?.[0]?.[0]).toBe('转写文字')
    expect(trackStop).toHaveBeenCalledOnce()
  })

  it('preserves the full transcription and retries insertion without another upload', async () => {
    let draftHasSpace = false
    const onTranscribed = vi.fn((
      _text: string,
      acknowledge: (inserted: boolean) => void,
    ) => acknowledge(draftHasSpace))
    const view = mountVoiceInput({ onTranscribed })

    await view.get('[data-test="chat-voice-trigger"]').trigger('click')
    await flushPromises()
    await view.get('[data-test="chat-voice-trigger"]').trigger('click')
    await flushPromises()

    expect(view.attributes('data-state')).toBe('error')
    expect(view.get('[data-test="chat-voice-error"]').text())
      .toContain('chat.voice.errors.draftFull')
    expect((view.get('[data-test="chat-voice-pending-transcript"]')
      .element as HTMLTextAreaElement).value).toBe('转写文字')
    expect(apiMocks.transcribeChatAudio).toHaveBeenCalledOnce()

    draftHasSpace = true
    await view.get('[data-test="chat-voice-retry-insert"]').trigger('click')

    expect(onTranscribed).toHaveBeenCalledTimes(2)
    expect(onTranscribed).toHaveBeenLastCalledWith('转写文字', expect.any(Function))
    expect(apiMocks.transcribeChatAudio).toHaveBeenCalledOnce()
    expect(view.attributes('data-state')).toBe('idle')
    expect(view.find('[data-test="chat-voice-pending-transcript"]').exists()).toBe(false)
  })

  it('keeps the preserved transcript inside the viewport-clamped error card', () => {
    expect(COMPONENT_STYLE).toMatch(
      /\.chat-voice-input__error\s*\{[\s\S]*?z-index: 70;/,
    )
    expect(COMPONENT_STYLE).toMatch(
      /\.chat-voice-input__error\s*\{[\s\S]*?width: min\(290px, calc\(100vw - 64px\)\);[\s\S]*?max-width: calc\(100vw - 64px\);/,
    )
    expect(COMPONENT_STYLE).toMatch(
      /\.chat-voice-input__pending-transcript\s*\{[\s\S]*?width: 100%;[\s\S]*?max-width: 100%;/,
    )
  })

  it('reports an unexpected recorder stop instead of remaining stuck as recording', async () => {
    const view = mountVoiceInput()
    await view.get('[data-test="chat-voice-trigger"]').trigger('click')
    await flushPromises()

    FakeMediaRecorder.instances[0]?.stop()
    await flushPromises()

    expect(view.attributes('data-state')).toBe('error')
    expect(view.get('[data-test="chat-voice-error"]').text())
      .toContain('chat.voice.errors.recordingFailed')
    expect(trackStop).toHaveBeenCalledOnce()
    expect(apiMocks.transcribeChatAudio).not.toHaveBeenCalled()
  })

  it('aborts cloud transcription and clears busy state when unmounted', async () => {
    let uploadSignal: AbortSignal | undefined
    apiMocks.transcribeChatAudio.mockImplementation((_blob: Blob, options: { signal: AbortSignal }) => {
      uploadSignal = options.signal
      return new Promise((_resolve, reject) => {
        options.signal.addEventListener('abort', () => {
          reject(new DOMException('Aborted', 'AbortError'))
        }, { once: true })
      })
    })
    const view = mountVoiceInput()
    await view.get('[data-test="chat-voice-trigger"]').trigger('click')
    await flushPromises()
    await view.get('[data-test="chat-voice-trigger"]').trigger('click')
    await flushPromises()
    expect(view.attributes('data-state')).toBe('transcribing')

    view.unmount()
    wrapper = undefined
    await flushPromises()

    expect(uploadSignal?.aborted).toBe(true)
    expect(view.emitted('transcribed')).toBeUndefined()
    const busyEvents = view.emitted('busy-change') ?? []
    expect(busyEvents[busyEvents.length - 1]).toEqual([false])
  })
})

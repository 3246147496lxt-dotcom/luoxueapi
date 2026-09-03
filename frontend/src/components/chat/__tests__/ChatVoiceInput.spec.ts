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
    'chat.voice.errors.permissionDenied': '无法使用麦克风，请在浏览器设置中允许麦克风权限。',
    'chat.voice.errors.unsupported': '当前浏览器暂不支持语音输入。',
    'chat.voice.errors.unavailable': '语音转写服务暂时不可用，请稍后重试。',
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
    vi.stubGlobal('requestAnimationFrame', vi.fn(() => 1))
    vi.stubGlobal('cancelAnimationFrame', vi.fn())
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
    const focusSpy = vi.spyOn(HTMLElement.prototype, 'focus')
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
    expect(view.find('[data-test="chat-voice-recording-shell"]').exists()).toBe(true)
    expect(focusSpy).toHaveBeenCalledWith({ preventScroll: true })

    const confirmShortcut = new KeyboardEvent('keydown', {
      code: 'KeyD',
      ctrlKey: true,
      shiftKey: true,
      bubbles: true,
      cancelable: true,
    })
    window.dispatchEvent(confirmShortcut)
    await flushPromises()

    expect(confirmShortcut.defaultPrevented).toBe(true)
    expect(apiMocks.transcribeChatAudio).toHaveBeenCalledOnce()
    expect(view.attributes('data-state')).toBe('idle')
    focusSpy.mockRestore()
  })

  it('can start the same truthful transcription flow from the voice action', async () => {
    const view = mountVoiceInput()
    const api = view.vm as unknown as { start: () => void }

    api.start()
    await flushPromises()

    expect(view.attributes('data-state')).toBe('recording')
  })

  it('shows the blank recording shell while voice resources are initializing', async () => {
    let resolveCapability: ((state: 'ready') => void) | undefined
    const initializeCapability = vi.fn(() => new Promise<'ready'>((resolve) => {
      resolveCapability = resolve
    }))
    const view = mountVoiceInput({
      capabilityState: 'initializing',
      initializeCapability,
    })

    await view.get('[data-test="chat-voice-trigger"]').trigger('click')
    await nextTick()

    expect(view.attributes('data-state')).toBe('initializing')
    expect(view.find('[data-test="chat-voice-trigger"]').exists()).toBe(false)
    expect(view.find('[data-icon="chatMicrophone"]').exists()).toBe(false)
    expect(view.find('.chat-voice-input__spinner').exists()).toBe(false)
    expect(view.find('[data-test="chat-voice-recording-shell"]').exists()).toBe(true)
    expect(view.find('[data-test="chat-voice-cancel"] [data-icon="x"]').exists()).toBe(true)
    expect(view.find('[data-test="chat-voice-confirm"] [data-icon="check"]').exists()).toBe(true)
    expect(view.get('[data-test="chat-voice-cancel"]').attributes('aria-label'))
      .toBe('chat.voice.cancelRequest')
    const confirm = view.get('[data-test="chat-voice-confirm"]')
    expect((confirm.element as HTMLButtonElement).disabled).toBe(true)
    expect(confirm.attributes('aria-label')).toBe('chat.voice.preparing')
    const canvas = view.get('[data-test="chat-voice-waveform"]')
    expect(canvas.classes()).toContain('chat-voice-input__waveform--hidden')
    expect((canvas.element as HTMLCanvasElement).style.display).toBe('')
    expect(initializeCapability).toHaveBeenCalledOnce()
    expect(getUserMedia).not.toHaveBeenCalled()
    expect(requestAnimationFrame).not.toHaveBeenCalled()
    expect(view.emitted('busy-change')?.at(-1)).toEqual([true])

    resolveCapability?.('ready')
    await flushPromises()

    expect(view.attributes('data-state')).toBe('recording')
    expect(view.get('[data-test="chat-voice-waveform"]').classes())
      .not.toContain('chat-voice-input__waveform--hidden')
    expect((view.get('[data-test="chat-voice-confirm"]')
      .element as HTMLButtonElement).disabled).toBe(false)
    expect(getUserMedia).toHaveBeenCalledOnce()
  })

  it('cancels capability initialization from the X before microphone capture starts', async () => {
    let resolveCapability: ((state: 'ready') => void) | undefined
    const initializeCapability = vi.fn(() => new Promise<'ready'>((resolve) => {
      resolveCapability = resolve
    }))
    const view = mountVoiceInput({
      capabilityState: 'initializing',
      initializeCapability,
    })

    await view.get('[data-test="chat-voice-trigger"]').trigger('click')
    await nextTick()
    await view.get('[data-test="chat-voice-cancel"]').trigger('click')
    await flushPromises()

    expect(view.attributes('data-state')).toBe('idle')
    expect(view.find('[data-test="chat-voice-recording-shell"]').exists()).toBe(false)
    expect(view.find('[data-test="chat-voice-trigger"]').exists()).toBe(true)

    resolveCapability?.('ready')
    await flushPromises()

    expect(getUserMedia).not.toHaveBeenCalled()
    expect(FakeMediaRecorder.instances).toHaveLength(0)
    expect(requestAnimationFrame).not.toHaveBeenCalled()
  })

  it('uses capability limits that arrive after the component is already mounted', async () => {
    let resolveCapability: ((state: 'ready') => void) | undefined
    const initializeCapability = vi.fn(() => new Promise<'ready'>((resolve) => {
      resolveCapability = resolve
    }))
    const view = mountVoiceInput({
      capabilityState: 'initializing',
      initializeCapability,
    })

    await view.get('[data-test="chat-voice-trigger"]').trigger('click')
    await view.setProps({
      capabilityState: 'ready',
      acceptedMimeTypes: ['audio/mp4'],
    })
    resolveCapability?.('ready')
    await flushPromises()

    expect(view.attributes('data-state')).toBe('unsupported')
    expect(getUserMedia).not.toHaveBeenCalled()
  })

  it('keeps the microphone visible and shows a specific unavailable message', async () => {
    const initializeCapability = vi.fn().mockResolvedValue('unavailable')
    const view = mountVoiceInput({
      capabilityState: 'unavailable',
      initializeCapability,
    })

    await view.get('[data-test="chat-voice-trigger"]').trigger('click')
    await flushPromises()

    expect(view.attributes('data-state')).toBe('unavailable')
    expect(view.get('[data-test="chat-voice-error"]').text())
      .toContain('语音转写服务暂时不可用，请稍后重试。')
    expect(view.find('[data-test="chat-voice-trigger"]').exists()).toBe(true)
    expect(getUserMedia).not.toHaveBeenCalled()
  })

  it('shows the same specific message when the transcription backend returns 503', async () => {
    apiMocks.transcribeChatAudio.mockRejectedValueOnce(new ChatAPIError(
      'Transcription unavailable',
      { status: 503, code: 'TRANSCRIPTION_UNAVAILABLE' },
    ))
    const view = mountVoiceInput()

    await view.get('[data-test="chat-voice-trigger"]').trigger('click')
    await flushPromises()
    await view.get('[data-test="chat-voice-confirm"]').trigger('click')
    await flushPromises()

    expect(view.attributes('data-state')).toBe('unavailable')
    expect(view.get('[data-test="chat-voice-error"]').text())
      .toContain('语音转写服务暂时不可用，请稍后重试。')
    expect(view.find('[data-test="chat-voice-trigger"]').exists()).toBe(true)
  })

  it('keeps the microphone visible and explains unsupported browsers', async () => {
    vi.unstubAllGlobals()
    const view = mountVoiceInput()

    await view.get('[data-test="chat-voice-trigger"]').trigger('click')
    await flushPromises()

    expect(view.attributes('data-state')).toBe('unsupported')
    expect(view.get('[data-test="chat-voice-error"]').text())
      .toContain('当前浏览器暂不支持语音输入。')
    expect(view.find('[data-test="chat-voice-trigger"]').exists()).toBe(true)
  })

  it('records with the preferred Opus format, uploads once, and emits text without sending', async () => {
    const view = mountVoiceInput()
    const trigger = view.get('[data-test="chat-voice-trigger"]')

    await trigger.trigger('click')
    await flushPromises()
    expect(view.attributes('data-state')).toBe('recording')
    expect(view.find('[data-test="chat-voice-trigger"]').exists()).toBe(false)
    expect(view.get('[data-test="chat-voice-cancel"]').attributes('aria-label'))
      .toBe('chat.voice.cancelRecording')
    expect(view.get('[data-test="chat-voice-confirm"]').attributes('aria-label'))
      .toBe('chat.voice.stop')
    expect(view.find('[data-test="chat-voice-cancel"] [data-icon="x"]').exists()).toBe(true)
    expect(view.find('[data-test="chat-voice-confirm"] [data-icon="check"]').exists()).toBe(true)
    expect(view.find('[data-test="chat-voice-waveform"]').exists()).toBe(true)
    expect(FakeMediaRecorder.instances[0]?.mimeType).toBe('audio/webm;codecs=opus')

    await view.get('[data-test="chat-voice-confirm"]').trigger('click')
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

  it('keeps the final moving waveform visible while cloud transcription is pending', async () => {
    let resolveTranscription: ((value: { text: string }) => void) | undefined
    apiMocks.transcribeChatAudio.mockReturnValueOnce(new Promise((resolve) => {
      resolveTranscription = resolve
    }))
    const view = mountVoiceInput()

    await view.get('[data-test="chat-voice-trigger"]').trigger('click')
    await flushPromises()
    await view.get('[data-test="chat-voice-confirm"]').trigger('click')
    await flushPromises()

    expect(view.attributes('data-state')).toBe('transcribing')
    expect(view.find('[data-test="chat-voice-recording-shell"]').exists()).toBe(true)
    expect(view.find('[data-test="chat-voice-waveform"]').exists()).toBe(true)
    expect(view.find('[data-test="chat-voice-confirm"]').exists()).toBe(false)
    expect(view.find('[data-test="chat-voice-transcribing-spinner"]').exists()).toBe(true)
    expect(view.get('[data-test="chat-voice-cancel"]').attributes('aria-label'))
      .toBe('chat.voice.cancelTranscription')

    resolveTranscription?.({ text: '转写文字' })
    await flushPromises()
    expect(view.attributes('data-state')).toBe('idle')
  })

  it('cancels cloud transcription with Escape while the frozen waveform is visible', async () => {
    let uploadSignal: AbortSignal | undefined
    apiMocks.transcribeChatAudio.mockImplementationOnce((_blob, options) => {
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
    await view.get('[data-test="chat-voice-confirm"]').trigger('click')
    await flushPromises()
    const escape = new KeyboardEvent('keydown', {
      key: 'Escape',
      bubbles: true,
      cancelable: true,
    })
    window.dispatchEvent(escape)
    await flushPromises()

    expect(escape.defaultPrevented).toBe(true)
    expect(uploadSignal?.aborted).toBe(true)
    expect(view.attributes('data-state')).toBe('idle')
  })

  it('cancels the active recording from the X without uploading audio', async () => {
    const view = mountVoiceInput()
    const focusSpy = vi.spyOn(HTMLElement.prototype, 'focus')

    await view.get('[data-test="chat-voice-trigger"]').trigger('click')
    await flushPromises()
    focusSpy.mockClear()
    await view.get('[data-test="chat-voice-cancel"]').trigger('click')
    await flushPromises()

    expect(trackStop).toHaveBeenCalledOnce()
    expect(apiMocks.transcribeChatAudio).not.toHaveBeenCalled()
    expect(view.attributes('data-state')).toBe('idle')
    expect(view.find('[data-test="chat-voice-trigger"]').exists()).toBe(true)
    expect(view.find('[data-test="chat-voice-recording-shell"]').exists()).toBe(false)
    expect(focusSpy).toHaveBeenCalledWith({ preventScroll: true })
    focusSpy.mockRestore()
  })

  it('makes covered composer controls inert as soon as the voice shell starts preparing', async () => {
    let resolveCapability: ((state: 'ready') => void) | undefined
    const initializeCapability = vi.fn(() => new Promise<'ready'>((resolve) => {
      resolveCapability = resolve
    }))
    const composer = document.createElement('form')
    composer.className = 'chat-composer'
    const coveredButton = document.createElement('button')
    coveredButton.setAttribute('aria-label', 'covered control')
    composer.appendChild(coveredButton)
    document.body.appendChild(composer)
    wrapper = mount(ChatVoiceInput, {
      attachTo: composer,
      props: {
        contextKey: 'user-7:new',
        capabilityState: 'initializing',
        initializeCapability,
        onTranscribed: (_text: string, acknowledge: (inserted: boolean) => void) => acknowledge(true),
      },
      global: { stubs: { Icon: IconStub } },
    })

    await wrapper.get('[data-test="chat-voice-trigger"]').trigger('click')
    await nextTick()

    expect(coveredButton.hasAttribute('inert')).toBe(true)
    expect(coveredButton.getAttribute('aria-hidden')).toBe('true')

    await wrapper.get('[data-test="chat-voice-cancel"]').trigger('click')
    await flushPromises()

    expect(coveredButton.hasAttribute('inert')).toBe(false)
    expect(coveredButton.hasAttribute('aria-hidden')).toBe(false)

    resolveCapability?.('ready')
    await flushPromises()
    expect(getUserMedia).not.toHaveBeenCalled()
  })

  it('cancels the active recording with Escape after the original trigger is replaced', async () => {
    const view = mountVoiceInput()

    await view.get('[data-test="chat-voice-trigger"]').trigger('click')
    await flushPromises()
    const escape = new KeyboardEvent('keydown', {
      key: 'Escape',
      bubbles: true,
      cancelable: true,
    })
    window.dispatchEvent(escape)
    await flushPromises()

    expect(escape.defaultPrevented).toBe(true)
    expect(trackStop).toHaveBeenCalledOnce()
    expect(apiMocks.transcribeChatAudio).not.toHaveBeenCalled()
    expect(view.attributes('data-state')).toBe('idle')
  })

  it('does not consume Escape while an active modal owns the keyboard', async () => {
    const view = mountVoiceInput()

    await view.get('[data-test="chat-voice-trigger"]').trigger('click')
    await flushPromises()
    const modal = document.createElement('div')
    modal.setAttribute('role', 'dialog')
    modal.setAttribute('aria-modal', 'true')
    document.body.appendChild(modal)
    const escape = new KeyboardEvent('keydown', {
      key: 'Escape',
      bubbles: true,
      cancelable: true,
    })
    window.dispatchEvent(escape)
    await flushPromises()

    expect(escape.defaultPrevented).toBe(false)
    expect(view.attributes('data-state')).toBe('recording')
    expect(trackStop).not.toHaveBeenCalled()
  })

  it('explains daily voice quota exhaustion without implying text chat is blocked', async () => {
    apiMocks.transcribeChatAudio.mockRejectedValueOnce(new ChatAPIError(
      'Daily voice transcription limit reached',
      { status: 429, code: 'TRANSCRIPTION_DAILY_QUOTA_EXCEEDED' },
    ))
    const view = mountVoiceInput()

    await view.get('[data-test="chat-voice-trigger"]').trigger('click')
    await flushPromises()
    await view.get('[data-test="chat-voice-confirm"]').trigger('click')
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
    await view.get('[data-test="chat-voice-confirm"]').trigger('click')
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

    expect(view.attributes('data-state')).toBe('denied')
    expect(view.get('[data-test="chat-voice-error"]').text())
      .toContain('无法使用麦克风，请在浏览器设置中允许麦克风权限。')
    expect(view.get('[data-test="chat-voice-trigger"]').attributes('aria-label'))
      .toBe('无法使用麦克风，请在浏览器设置中允许麦克风权限。')

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
    expect(view.find('[data-test="chat-voice-recording-shell"]').exists()).toBe(true)
    expect(view.get('[data-test="chat-voice-waveform"]').classes())
      .toContain('chat-voice-input__waveform--hidden')
    expect((view.get('[data-test="chat-voice-confirm"]')
      .element as HTMLButtonElement).disabled).toBe(true)
    expect(view.find('.chat-voice-input__spinner').exists()).toBe(false)
    expect(requestAnimationFrame).not.toHaveBeenCalled()

    await view.setProps({ contextKey: 'user-7:conversation-2' })
    resolveCapture?.(capturedStream)
    await flushPromises()

    expect(trackStop).toHaveBeenCalledOnce()
    expect(FakeMediaRecorder.instances).toHaveLength(0)
    expect(apiMocks.transcribeChatAudio).not.toHaveBeenCalled()
    expect(view.attributes('data-state')).toBe('idle')
  })

  it('keeps permission loading blank, then starts the waveform only after capture resolves', async () => {
    let resolveCapture: ((stream: MediaStream) => void) | undefined
    const capturedStream = {
      getTracks: () => [{ stop: trackStop }],
    } as unknown as MediaStream
    getUserMedia.mockReturnValue(new Promise<MediaStream>((resolve) => {
      resolveCapture = resolve
    }))
    vi.stubGlobal('matchMedia', vi.fn(() => ({
      matches: false,
      media: '(prefers-reduced-motion: reduce)',
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(() => true),
    })))
    const view = mountVoiceInput()

    await view.get('[data-test="chat-voice-trigger"]').trigger('click')
    await nextTick()

    expect(view.attributes('data-state')).toBe('requesting')
    expect(view.get('[data-test="chat-voice-waveform"]').classes())
      .toContain('chat-voice-input__waveform--hidden')
    expect(requestAnimationFrame).not.toHaveBeenCalled()

    resolveCapture?.(capturedStream)
    await flushPromises()

    expect(view.attributes('data-state')).toBe('recording')
    expect(view.get('[data-test="chat-voice-waveform"]').classes())
      .not.toContain('chat-voice-input__waveform--hidden')
    expect((view.get('[data-test="chat-voice-confirm"]')
      .element as HTMLButtonElement).disabled).toBe(false)
    expect(requestAnimationFrame).toHaveBeenCalled()
  })

  it('cancels a pending microphone request from the X and stops a late stream', async () => {
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
    await view.get('[data-test="chat-voice-cancel"]').trigger('click')
    await flushPromises()

    expect(view.attributes('data-state')).toBe('idle')
    expect(view.find('[data-test="chat-voice-recording-shell"]').exists()).toBe(false)

    resolveCapture?.(capturedStream)
    await flushPromises()

    expect(trackStop).toHaveBeenCalledOnce()
    expect(FakeMediaRecorder.instances).toHaveLength(0)
    expect(apiMocks.transcribeChatAudio).not.toHaveBeenCalled()
    expect(requestAnimationFrame).not.toHaveBeenCalled()
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
    await view.get('[data-test="chat-voice-confirm"]').trigger('click')
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
      /\.chat-voice-input\s*\{[\s\S]*?width: 36px;[\s\S]*?height: 36px;[\s\S]*?flex: 0 0 36px;/,
    )
    expect(COMPONENT_STYLE).not.toContain('chat-voice-input__trigger--wide')
    expect(COMPONENT_STYLE).toMatch(
      /\.chat-voice-input__recording-shell\s*\{[\s\S]*?grid-template-rows: 36px;[\s\S]*?height: 52px;[\s\S]*?border-radius: 28px;[\s\S]*?color: var\(--chat-composer-primary-fg, #0d0d0d\);[\s\S]*?background: transparent;[\s\S]*?box-shadow: none;/,
    )
    expect(COMPONENT_STYLE).not.toContain('background: #212121;')
    expect(COMPONENT_STYLE).not.toContain('box-shadow: inset 0 0 0 1px #292929;')
    expect(COMPONENT_STYLE).toMatch(
      /\.chat-voice-input__waveform\s*\{[\s\S]*?width: 100%;[\s\S]*?height: 44px;/,
    )
    expect(COMPONENT_STYLE).toMatch(
      /\.chat-voice-input__waveform--hidden\s*\{[\s\S]*?visibility: hidden;/,
    )
    expect(COMPONENT_SOURCE).not.toContain('v-show="waveformVisible"')
    expect(COMPONENT_SOURCE).not.toContain('chat-voice-input__spinner')
    expect(COMPONENT_STYLE).toMatch(
      /\.chat-composer:has\(\.chat-voice-input--initializing\)[\s\S]*?height: 52px;/,
    )
    expect(COMPONENT_STYLE).toMatch(
      /\.chat-composer:has\(\.chat-voice-input--requesting\)[\s\S]*?height: 52px;/,
    )
    expect(COMPONENT_STYLE).toMatch(
      /\.chat-composer:has\(\.chat-voice-input--initializing\) > \.chat-composer__input-shell[\s\S]*?visibility: hidden;/,
    )
    expect(COMPONENT_STYLE).toMatch(
      /\.chat-composer:has\(\.chat-voice-input--requesting\) > \.chat-composer__trailing > :not\(\.chat-voice-input\)[\s\S]*?visibility: hidden;/,
    )
    expect(COMPONENT_STYLE).toMatch(
      /\.chat-composer:has\(\.chat-voice-input--recording\) > \.chat-composer__input-shell[\s\S]*?visibility: hidden;/,
    )
    expect(COMPONENT_STYLE).toMatch(
      /\.chat-composer:has\(\.chat-voice-input--transcribing\) > \.chat-composer__trailing > :not\(\.chat-voice-input\)[\s\S]*?visibility: hidden;/,
    )
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

  it('keeps pending waveform samples in the raw audio-level domain', () => {
    expect(COMPONENT_SOURCE).toContain(
      'waveformPendingLevel = recorder.audioLevel.value',
    )
    expect(COMPONENT_SOURCE).not.toContain(
      'waveformPendingLevel = chatVoiceWaveformHeight(recorder.audioLevel.value)',
    )
    expect(COMPONENT_SOURCE).toMatch(
      /appendChatVoiceWaveformSample\([\s\S]*?waveformPendingLevel,[\s\S]*?capacity,/,
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
    await view.get('[data-test="chat-voice-confirm"]').trigger('click')
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

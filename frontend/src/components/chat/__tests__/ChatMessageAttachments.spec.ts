import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { ChatAttachment } from '@/types/chat'

const apiMocks = vi.hoisted(() => ({
  getContent: vi.fn(),
}))

vi.mock('@/api/chat', () => ({
  getChatAttachmentContent: apiMocks.getContent,
  isAbortError: (error: unknown) => (
    !!error && typeof error === 'object' && (error as { name?: unknown }).name === 'AbortError'
  ),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => (
        params ? `${key} ${Object.values(params).join(' ')}` : key
      ),
    }),
  }
})

import ChatMessageAttachments from '../ChatMessageAttachments.vue'

const wrappers: VueWrapper[] = []
const readyImage: ChatAttachment = {
  id: 'image-1',
  name: 'private.png',
  kind: 'image',
  mimeType: 'image/png',
  size: 100,
  status: 'ready',
  expiresAt: '2099-01-01T00:00:00Z',
}

function mountAttachments(attachments: ChatAttachment[]) {
  const wrapper = mount(ChatMessageAttachments, {
    props: { attachments },
    global: {
      stubs: {
        Icon: { props: ['name'], template: '<span :data-icon="name" />' },
      },
    },
  })
  wrappers.push(wrapper)
  return wrapper
}

beforeEach(() => {
  vi.resetAllMocks()
  vi.stubGlobal('IntersectionObserver', undefined)
  vi.stubGlobal('URL', {
    ...URL,
    createObjectURL: vi.fn(() => 'blob:authenticated-image'),
    revokeObjectURL: vi.fn(),
  })
})

afterEach(() => {
  wrappers.splice(0).forEach((wrapper) => wrapper.unmount())
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('ChatMessageAttachments', () => {
  it('defers authenticated image loading until the card approaches the viewport', async () => {
    let callback!: IntersectionObserverCallback
    const observe = vi.fn()
    const unobserve = vi.fn()
    const disconnect = vi.fn()
    class IntersectionObserverStub {
      observe = observe
      unobserve = unobserve
      disconnect = disconnect

      constructor(nextCallback: IntersectionObserverCallback) {
        callback = nextCallback
      }
    }
    vi.stubGlobal('IntersectionObserver', IntersectionObserverStub)
    apiMocks.getContent.mockResolvedValue(new Blob(['image'], { type: 'image/png' }))
    const wrapper = mountAttachments([readyImage])
    await flushPromises()

    expect(observe).toHaveBeenCalledWith(wrapper.get('.chat-message-attachments').element)
    expect(apiMocks.getContent).not.toHaveBeenCalled()

    callback([{
      isIntersecting: true,
      intersectionRatio: 0.1,
      target: wrapper.get('.chat-message-attachments').element,
    } as IntersectionObserverEntry], {} as IntersectionObserver)
    await flushPromises()

    expect(apiMocks.getContent).toHaveBeenCalledWith('image-1', expect.any(AbortSignal))
    expect(wrapper.find('img').exists()).toBe(true)
    expect(unobserve).toHaveBeenCalled()
    expect(disconnect).toHaveBeenCalled()
  })

  it('loads image bytes through the authenticated API and revokes the object URL', async () => {
    apiMocks.getContent.mockResolvedValue(new Blob(['image'], { type: 'image/png' }))
    const wrapper = mountAttachments([readyImage])
    await flushPromises()

    expect(apiMocks.getContent).toHaveBeenCalledWith('image-1', expect.any(AbortSignal))
    expect(wrapper.get('img').attributes('src')).toBe('blob:authenticated-image')
    wrapper.unmount()
    wrappers.splice(wrappers.indexOf(wrapper), 1)
    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:authenticated-image')
  })

  it('does not fetch expired images and renders document metadata as a non-executable card', async () => {
    const wrapper = mountAttachments([
      { ...readyImage, id: 'expired-image', status: 'expired' },
      {
        id: 'document-1',
        name: 'brief.docx',
        kind: 'document',
        mimeType: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
        size: 4096,
        status: 'ready',
        expiresAt: '2099-01-01T00:00:00Z',
        pageCount: 4,
      },
    ])
    await flushPromises()

    expect(apiMocks.getContent).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('chat.attachments.expired')
    expect(wrapper.text()).toContain('brief.docx')
    expect(wrapper.text()).toContain('DOCX')
    expect(wrapper.find('a').exists()).toBe(false)
  })

  it('revokes a loaded image when its expiry passes while the page stays open', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-07T00:00:00Z'))
    apiMocks.getContent.mockResolvedValue(new Blob(['image'], { type: 'image/png' }))
    const wrapper = mountAttachments([{
      ...readyImage,
      expiresAt: '2026-08-07T00:00:01Z',
    }])
    await flushPromises()
    expect(wrapper.find('img').exists()).toBe(true)

    await vi.advanceTimersByTimeAsync(1_100)
    await flushPromises()

    expect(wrapper.find('img').exists()).toBe(false)
    expect(wrapper.text()).toContain('chat.attachments.expired')
    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:authenticated-image')
  })
})

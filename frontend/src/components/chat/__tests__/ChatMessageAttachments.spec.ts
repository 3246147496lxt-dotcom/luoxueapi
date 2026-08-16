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
    attachTo: document.body,
    props: { attachments },
    global: {
      stubs: {
        Icon: { props: ['name'], template: '<span :data-icon="name" />' },
        Transition: { props: ['name'], template: '<slot />' },
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
  document.body.innerHTML = ''
  document.body.style.overflow = ''
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

  it('routes image MIME attachments to a filename-free image preview even when kind is stale', async () => {
    apiMocks.getContent.mockResolvedValue(new Blob(['image'], { type: 'image/png' }))
    const wrapper = mountAttachments([{
      ...readyImage,
      name: 'CleanShot 2026-08-13 at 23.40.09@2x.png',
      kind: 'document',
    }])
    await flushPromises()

    const trigger = wrapper.get('[data-test="chat-message-image-trigger"]')
    expect(trigger.element.tagName).toBe('BUTTON')
    expect(trigger.attributes()).toMatchObject({
      type: 'button',
      'aria-haspopup': 'dialog',
      'aria-label': 'chat.attachments.viewImage CleanShot 2026-08-13 at 23.40.09@2x.png',
    })
    expect(trigger.get('img').attributes('alt')).toBe('')
    expect(wrapper.text()).not.toContain('CleanShot')
    expect(wrapper.find('.chat-message-attachments__document-copy').exists()).toBe(false)
  })

  it('keeps PDF and ZIP attachments in the existing metadata cards', async () => {
    const wrapper = mountAttachments([
      {
        id: 'pdf-1',
        name: 'brief.pdf',
        kind: 'document',
        mimeType: 'application/pdf',
        size: 4096,
        status: 'ready',
        expiresAt: '2099-01-01T00:00:00Z',
      },
      {
        id: 'zip-1',
        name: 'assets.zip',
        kind: 'document',
        mimeType: 'application/zip',
        size: 8192,
        status: 'ready',
        expiresAt: '2099-01-01T00:00:00Z',
      },
    ])
    await flushPromises()

    expect(wrapper.findAll('.chat-message-attachments__item')).toHaveLength(2)
    expect(wrapper.text()).toContain('brief.pdf')
    expect(wrapper.text()).toContain('PDF')
    expect(wrapper.text()).toContain('assets.zip')
    expect(wrapper.text()).toContain('FILE')
    expect(wrapper.find('[data-test="chat-message-image-trigger"]').exists()).toBe(false)
    expect(apiMocks.getContent).not.toHaveBeenCalled()
  })

  it('right-groups multiple images without rendering any filename strip', async () => {
    apiMocks.getContent.mockResolvedValue(new Blob(['image'], { type: 'image/png' }))
    vi.mocked(URL.createObjectURL)
      .mockReturnValueOnce('blob:first-image')
      .mockReturnValueOnce('blob:second-image')
    const wrapper = mountAttachments([
      { ...readyImage, name: 'first.png' },
      { ...readyImage, id: 'image-2', name: 'second.png' },
    ])
    await flushPromises()

    expect(wrapper.get('.chat-message-attachments__images').classes())
      .toContain('chat-message-attachments__images--multiple')
    expect(wrapper.findAll('[data-test="chat-message-image-trigger"]')).toHaveLength(2)
    expect(wrapper.findAll('img').map((image) => image.attributes('src')))
      .toEqual(['blob:first-image', 'blob:second-image'])
    expect(wrapper.text()).not.toContain('first.png')
    expect(wrapper.text()).not.toContain('second.png')
    expect(wrapper.find('.chat-message-attachments__image-name').exists()).toBe(false)
  })

  it('opens the selected sent image with the existing blob URL and restores focus', async () => {
    apiMocks.getContent.mockResolvedValue(new Blob(['image'], { type: 'image/png' }))
    vi.mocked(URL.createObjectURL)
      .mockReturnValueOnce('blob:first-image')
      .mockReturnValueOnce('blob:second-image')
    const wrapper = mountAttachments([
      { ...readyImage, name: 'first.png' },
      { ...readyImage, id: 'image-2', name: 'second.png' },
    ])
    await flushPromises()
    const trigger = wrapper.findAll('[data-test="chat-message-image-trigger"]')[1]!
    const requestsBeforePreview = apiMocks.getContent.mock.calls.length

    await trigger.trigger('click')
    await flushPromises()

    const dialog = document.body.querySelector<HTMLElement>(
      '[data-test="chat-image-preview-dialog"]',
    )
    const preview = dialog?.querySelector<HTMLImageElement>(
      '[data-test="chat-image-preview-image"]',
    )
    expect(dialog?.getAttribute('role')).toBe('dialog')
    expect(dialog?.getAttribute('aria-modal')).toBe('true')
    expect(dialog?.getAttribute('aria-label')).toBe('chat.attachments.imagePreview second.png')
    expect(preview?.getAttribute('src')).toBe('blob:second-image')
    expect(preview?.getAttribute('alt')).toBe('second.png')
    expect(apiMocks.getContent).toHaveBeenCalledTimes(requestsBeforePreview)
    expect(URL.createObjectURL).toHaveBeenCalledTimes(2)

    document.body.querySelector<HTMLButtonElement>(
      '[data-test="chat-image-preview-close"]',
    )?.click()
    await flushPromises()

    expect(document.body.querySelector('[data-test="chat-image-preview-dialog"]')).toBeNull()
    expect(document.activeElement).toBe(trigger.element)
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

  it('labels durable spreadsheet and presentation attachments by their real format', async () => {
    const wrapper = mountAttachments([
      {
        id: 'spreadsheet-1',
        name: 'budget.xlsx',
        kind: 'document',
        mimeType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        size: 4096,
        status: 'ready',
        expiresAt: '2099-01-01T00:00:00Z',
      },
      {
        id: 'presentation-1',
        name: 'roadmap.pptx',
        kind: 'document',
        mimeType: 'application/vnd.openxmlformats-officedocument.presentationml.presentation',
        size: 8192,
        status: 'ready',
        expiresAt: '2099-01-01T00:00:00Z',
      },
    ])
    await flushPromises()

    expect(wrapper.text()).toContain('budget.xlsx')
    expect(wrapper.text()).toContain('XLSX')
    expect(wrapper.text()).toContain('roadmap.pptx')
    expect(wrapper.text()).toContain('PPTX')
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

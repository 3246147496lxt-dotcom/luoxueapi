import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import enChat from '@/i18n/locales/en/chat'
import zhChat from '@/i18n/locales/zh/chat'
import type { ChatAttachment } from '@/types/chat'
import type { LibraryFile } from '@/types/library'
import {
  CHAT_ATTACHMENT_TOTAL_MAX_BYTES,
  type ChatAttachmentDraft,
} from '../chatAttachmentUi'

const apiMocks = vi.hoisted(() => ({
  upload: vi.fn(),
  remove: vi.fn(),
}))

vi.mock('@/api/chat', () => {
  class MockChatAPIError extends Error {
    code: string | number | null
    reason: unknown
    metadata: unknown

    constructor(message: string, options: {
      code?: string | number
      reason?: unknown
      metadata?: unknown
    } = {}) {
      super(message)
      this.code = options.code ?? null
      this.reason = options.reason
      this.metadata = options.metadata
    }
  }
  return {
    ChatAPIError: MockChatAPIError,
    uploadChatAttachment: apiMocks.upload,
    deleteChatAttachment: apiMocks.remove,
    isAbortError: (error: unknown) => (
      !!error && typeof error === 'object' && (error as { name?: unknown }).name === 'AbortError'
    ),
  }
})

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

import ChatAttachmentPicker from '../ChatAttachmentPicker.vue'

const attachment: ChatAttachment = {
  id: 'attachment-1',
  name: 'snow.png',
  kind: 'image',
  mimeType: 'image/png',
  size: 1024,
  status: 'ready',
  expiresAt: '2099-01-01T00:00:00Z',
  width: 32,
  height: 32,
}

const libraryFile: LibraryFile = {
  id: 'library-image-1',
  name: 'library-snow.png',
  mimeType: 'image/png',
  size: 2048,
  source: 'uploaded',
  type: 'image',
  category: 'image',
  status: 'ready',
  createdAt: '2026-08-14T01:00:00.000Z',
  updatedAt: '2026-08-14T02:00:00.000Z',
  width: 64,
  height: 48,
}

const wrappers: VueWrapper[] = []

function mountPicker(props: Record<string, unknown> = {}) {
  const wrapper = mount(ChatAttachmentPicker, {
    attachTo: document.body,
    props: { supportsVision: true, ...props },
  })
  wrappers.push(wrapper)
  return wrapper
}

function pickerApi(wrapper: VueWrapper) {
  return wrapper.vm as unknown as {
    addFiles: (files: File[]) => void
    addLibraryFiles: (files: LibraryFile[]) => void
    cancel: (key: string) => void
    retry: (key: string) => void
    remove: (key: string) => void
    getReadyAttachments: () => ChatAttachment[]
    getReadySelection: () => {
      attachments: ChatAttachment[]
      uploadAttachmentIds: string[]
      libraryAttachments: Array<{ source: 'library'; fileId: string }>
    }
    discardAll: () => Promise<void>
  }
}

function lastDrafts(wrapper: VueWrapper): ChatAttachmentDraft[] {
  const changes = wrapper.emitted('change') ?? []
  return (changes.at(-1)?.[0] ?? []) as ChatAttachmentDraft[]
}

function attachmentMenu(): HTMLElement | null {
  return document.querySelector<HTMLElement>(
    '[data-test="chat-attachment-menu"][role="menu"]',
  )
}

function getAttachmentMenu(): HTMLElement {
  const menu = attachmentMenu()
  if (!menu) throw new Error('Expected the attachment menu to be open')
  return menu
}

function getUploadMenuItem(): HTMLButtonElement {
  const uploadItem = getAttachmentMenu().querySelector<HTMLButtonElement>(
    '[data-test="chat-attachment-menu-upload"][role="menuitem"]',
  )
  if (!uploadItem) throw new Error('Expected an attachment upload menu item')
  return uploadItem
}

beforeEach(() => {
  vi.resetAllMocks()
  apiMocks.remove.mockResolvedValue(undefined)
  vi.stubGlobal('URL', {
    ...URL,
    createObjectURL: vi.fn(() => 'blob:local-preview'),
    revokeObjectURL: vi.fn(),
  })
})

afterEach(() => {
  wrappers.splice(0).forEach((wrapper) => wrapper.unmount())
  document.body.innerHTML = ''
  vi.unstubAllGlobals()
})

describe('ChatAttachmentPicker', () => {
  it('keeps the plus hover guidance aligned with the content menu', () => {
    expect(zhChat.chat.attachments.tooltip).toBe('添加文件等')
    expect(enChat.chat.attachments.tooltip).toBe('Add files and more')
  })

  it('uses the ChatGPT-style plus control as an accessible menu trigger', async () => {
    const wrapper = mountPicker()
    const trigger = wrapper.get('[data-test="chat-attachment-menu-trigger"]')
    const tooltipId = trigger.attributes('aria-describedby')
    const tooltip = tooltipId ? document.getElementById(tooltipId) : null

    expect(tooltip).not.toBeNull()

    const plusIcon = trigger.get('svg')
    expect(plusIcon.attributes('viewBox')).toBe('0 0 20 20')
    expect(plusIcon.classes()).toEqual(expect.arrayContaining(['h-5', 'w-5']))
    expect(plusIcon.get('g').attributes()).toMatchObject({
      fill: 'currentColor',
      stroke: 'none',
    })
    expect(plusIcon.get('path').attributes('d')).toBe(
      'M10 2.533a.8.8 0 0 1 .8.8V9.2h5.866a.8.8 0 1 1 0 1.6H10.8v5.866a.8.8 0 1 1-1.6 0V10.8H3.333a.8.8 0 0 1 0-1.6H9.2V3.333a.8.8 0 0 1 .8-.8',
    )
    expect(trigger.attributes('aria-label')).toBe('chat.attachments.addMenu')
    expect(trigger.attributes('aria-haspopup')).toBe('menu')
    expect(trigger.attributes('aria-expanded')).toBe('false')
    expect(trigger.attributes('aria-controls')).toBeTruthy()
    expect(trigger.attributes('aria-describedby')).toBe(tooltip?.id)
    expect(trigger.attributes('aria-keyshortcuts')).toBe('@')
    expect(tooltip?.querySelector('.chat-control-tooltip__label')?.textContent)
      .toBe('chat.attachments.tooltip')
    expect(tooltip?.textContent).toContain('@')
    expect(wrapper.get('input[type="file"]').attributes('tabindex')).toBe('-1')
    expect(wrapper.get('input[type="file"]').attributes('aria-hidden')).toBe('true')
  })

  it('opens the menu from the plus without immediately opening the native picker', async () => {
    const wrapper = mountPicker()
    const trigger = wrapper.get('[data-test="chat-attachment-menu-trigger"]')
    const input = wrapper.get('input[type="file"]')
    const click = vi.spyOn(input.element as HTMLInputElement, 'click')

    await trigger.trigger('click')

    const menu = getAttachmentMenu()
    expect(trigger.attributes('aria-expanded')).toBe('true')
    expect(menu.id).toBe(trigger.attributes('aria-controls'))
    expect(click).not.toHaveBeenCalled()
  })

  it('keeps the nine reference rows in order and exposes only implemented tools', async () => {
    const wrapper = mountPicker()

    await wrapper.get('[data-test="chat-attachment-menu-trigger"]').trigger('click')
    const items = Array.from(getAttachmentMenu().querySelectorAll<HTMLButtonElement>(
      '[role="menuitem"]',
    ))

    expect(items.map((item) => (
      item.querySelector('.chat-attachment-menu__label')?.textContent
    ))).toEqual([
      'chat.tools.upload.label',
      'chat.tools.fileLibrary.label',
      'chat.tools.createImage.label',
      'chat.tools.webSearch.label',
      'chat.tools.deepResearch.label',
      'chat.tools.integrations.github.label',
      'chat.tools.integrations.figma.label',
      'chat.tools.integrations.heygen.label',
      'chat.tools.integrations.gmail.label',
    ])
    expect(items.map((item) => (
      item.querySelector('svg')?.getAttribute('data-chat-tool-icon')
    ))).toEqual([
      'paperclip',
      'library',
      'create-image-plugin',
      'skill-globe-light',
      'skill-deep-research-light',
      'github',
      'figma',
      'heygen',
      'gmail',
    ])
    expect(items.slice(0, 2).every((item) => !item.hasAttribute('aria-disabled'))).toBe(true)
    expect(items.slice(2).every((item) => item.getAttribute('aria-disabled') === 'true')).toBe(true)
    expect(items.at(-1)?.querySelector('.chat-attachment-menu__status')?.classList)
      .toContain('chat-attachment-menu__status--persistent')
    expect(items.at(-1)?.querySelector('.chat-attachment-menu__status')?.textContent)
      .toBe('chat.tools.notSupported')
  })

  it('keeps the reference labels and descriptions exact in Chinese', () => {
    expect(zhChat.chat.tools).toMatchObject({
      upload: { label: '添加照片和文件', description: '从电脑上传' },
      fileLibrary: { label: '从文件库添加', description: '浏览和搜索你的文件' },
      createImage: { label: '创建图片', description: '可视化呈现任何内容' },
      webSearch: { label: '网页搜索', description: '查找实时新闻和信息' },
      deepResearch: { label: '深度研究', description: '获取详细报告' },
      integrations: {
        github: { label: 'GitHub', description: 'Triage PRs, issues, CI, and publish flows' },
        figma: {
          label: 'Figma',
          description: 'Design-to-code workflows powered by the Figma integration',
        },
        heygen: { label: 'HeyGen', description: 'Create AI videos' },
        gmail: { label: 'Gmail', description: 'Read and manage Gmail' },
      },
    })
  })

  it('opens the menu from the advertised focused @ shortcut without opening the native picker', async () => {
    const wrapper = mountPicker()
    const input = wrapper.get('input[type="file"]')
    const click = vi.spyOn(input.element as HTMLInputElement, 'click')
    const shortcut = new KeyboardEvent('keydown', {
      key: '@',
      shiftKey: true,
      bubbles: true,
      cancelable: true,
    })

    wrapper.get('[data-test="chat-attachment-menu-trigger"]').element.dispatchEvent(shortcut)
    await wrapper.vm.$nextTick()

    expect(shortcut.defaultPrevented).toBe(true)
    expect(wrapper.get('[data-test="chat-attachment-menu-trigger"]')
      .attributes('aria-expanded')).toBe('true')
    expect(attachmentMenu()).not.toBeNull()
    expect(click).not.toHaveBeenCalled()
  })

  it('opens the native picker only from the upload menu item and then closes the menu', async () => {
    const wrapper = mountPicker()
    const trigger = wrapper.get('[data-test="chat-attachment-menu-trigger"]')
    const input = wrapper.get('input[type="file"]')
    const click = vi.spyOn(input.element as HTMLInputElement, 'click')

    await trigger.trigger('click')
    getUploadMenuItem().click()
    await wrapper.vm.$nextTick()

    expect(click).toHaveBeenCalledOnce()
    expect(trigger.attributes('aria-expanded')).toBe('false')
    await vi.waitFor(() => expect(attachmentMenu()).toBeNull())
  })

  it('emits the library selection request without opening the native picker', async () => {
    const wrapper = mountPicker()
    const input = wrapper.get('input[type="file"]')
    const click = vi.spyOn(input.element as HTMLInputElement, 'click')

    await wrapper.get('[data-test="chat-attachment-menu-trigger"]').trigger('click')
    const libraryItem = getAttachmentMenu().querySelector<HTMLButtonElement>(
      '[aria-label^="chat.tools.fileLibrary.label"]',
    )
    expect(libraryItem?.hasAttribute('aria-disabled')).toBe(false)
    libraryItem?.click()
    await wrapper.vm.$nextTick()

    expect(click).not.toHaveBeenCalled()
    expect(wrapper.emitted('select-library')).toEqual([[]])
    await vi.waitFor(() => expect(attachmentMenu()).toBeNull())
  })

  it('filters menu rows and supports keyboard access to search and the last item', async () => {
    const wrapper = mountPicker()
    const trigger = wrapper.get('[data-test="chat-attachment-menu-trigger"]')

    trigger.element.dispatchEvent(new KeyboardEvent('keydown', {
      key: 'ArrowUp',
      bubbles: true,
      cancelable: true,
    }))
    await wrapper.vm.$nextTick()

    const initialItems = Array.from(getAttachmentMenu().querySelectorAll<HTMLButtonElement>(
      '[role="menuitem"]',
    ))
    expect(document.activeElement).toBe(initialItems.at(-1))

    getAttachmentMenu().dispatchEvent(new KeyboardEvent('keydown', {
      key: '/',
      bubbles: true,
      cancelable: true,
    }))
    const search = document.querySelector<HTMLInputElement>(
      '[data-test="chat-attachment-menu-search"]',
    )
    expect(document.activeElement).toBe(search)

    if (!search) throw new Error('Expected an attachment menu search input')
    search.value = 'GitHub'
    search.dispatchEvent(new Event('input', { bubbles: true }))
    await wrapper.vm.$nextTick()

    expect(Array.from(getAttachmentMenu().querySelectorAll(
      '.chat-attachment-menu__label',
    )).map((item) => item.textContent)).toEqual([
      'chat.tools.integrations.github.label',
    ])
  })

  it('closes on Escape and restores focus to the plus trigger', async () => {
    const wrapper = mountPicker()
    const trigger = wrapper.get('[data-test="chat-attachment-menu-trigger"]')

    await trigger.trigger('click')
    const escape = new KeyboardEvent('keydown', {
      key: 'Escape',
      bubbles: true,
      cancelable: true,
    })
    getAttachmentMenu().dispatchEvent(escape)
    await wrapper.vm.$nextTick()

    expect(escape.defaultPrevented).toBe(true)
    expect(trigger.attributes('aria-expanded')).toBe('false')
    await vi.waitFor(() => expect(attachmentMenu()).toBeNull())
    expect(document.activeElement).toBe(trigger.element)
  })

  it('closes when pointer interaction moves outside the picker', async () => {
    const wrapper = mountPicker()
    const trigger = wrapper.get('[data-test="chat-attachment-menu-trigger"]')

    await trigger.trigger('click')
    document.body.dispatchEvent(new MouseEvent('pointerdown', {
      bubbles: true,
      cancelable: true,
    }))
    await wrapper.vm.$nextTick()

    expect(trigger.attributes('aria-expanded')).toBe('false')
    await vi.waitFor(() => expect(attachmentMenu()).toBeNull())
  })

  it('closes when focus moves outside the picker', async () => {
    const outside = document.createElement('button')
    document.body.append(outside)
    const wrapper = mountPicker()
    const trigger = wrapper.get('[data-test="chat-attachment-menu-trigger"]')

    await trigger.trigger('click')
    outside.focus()
    await wrapper.vm.$nextTick()

    expect(document.activeElement).toBe(outside)
    expect(trigger.attributes('aria-expanded')).toBe('false')
    await vi.waitFor(() => expect(attachmentMenu()).toBeNull())
  })

  it('does not open either the menu or native picker while disabled', async () => {
    const wrapper = mountPicker({ disabled: true })
    const trigger = wrapper.get('[data-test="chat-attachment-menu-trigger"]')
    const input = wrapper.get('input[type="file"]')
    const click = vi.spyOn(input.element as HTMLInputElement, 'click')

    await trigger.trigger('click')
    trigger.element.dispatchEvent(new KeyboardEvent('keydown', {
      key: '@',
      shiftKey: true,
      bubbles: true,
      cancelable: true,
    }))
    await wrapper.vm.$nextTick()

    expect(trigger.attributes('disabled')).toBeDefined()
    expect(trigger.attributes('aria-expanded')).toBe('false')
    expect(attachmentMenu()).toBeNull()
    expect(click).not.toHaveBeenCalled()
  })

  it('ignores files passed through the exposed API while disabled', async () => {
    const wrapper = mountPicker({ disabled: true })

    pickerApi(wrapper).addFiles([
      new File(['png'], 'snow.png', { type: 'image/png' }),
    ])
    await flushPromises()

    expect(apiMocks.upload).not.toHaveBeenCalled()
    expect(lastDrafts(wrapper)).toEqual([])
    expect(wrapper.emitted('change')).toBeUndefined()
  })

  it('adds a ready library file without uploading and serializes only its library reference', async () => {
    const wrapper = mountPicker()

    pickerApi(wrapper).addLibraryFiles([libraryFile])
    await wrapper.vm.$nextTick()

    expect(apiMocks.upload).not.toHaveBeenCalled()
    expect(lastDrafts(wrapper)).toHaveLength(1)
    expect(lastDrafts(wrapper)[0]).toMatchObject({
      source: 'library',
      libraryFileId: 'library-image-1',
      state: 'ready',
      progress: 100,
      attachment: {
        id: 'library-image-1',
        name: 'library-snow.png',
        kind: 'image',
        expiresAt: '2126-01-01T00:00:00Z',
      },
    })
    expect(pickerApi(wrapper).getReadySelection()).toEqual({
      attachments: [expect.objectContaining({ id: 'library-image-1' })],
      uploadAttachmentIds: [],
      libraryAttachments: [{ source: 'library', fileId: 'library-image-1' }],
    })
  })

  it('rejects library files that would exceed the shared attachment byte budget', async () => {
    const wrapper = mountPicker()
    const first = {
      ...libraryFile,
      id: 'library-large-1',
      name: 'first-large.png',
      size: 12 * 1024 * 1024,
    }
    const second = {
      ...libraryFile,
      id: 'library-large-2',
      name: 'second-large.png',
      size: 9 * 1024 * 1024,
    }

    pickerApi(wrapper).addLibraryFiles([first, second])
    await wrapper.vm.$nextTick()

    expect(lastDrafts(wrapper)).toHaveLength(1)
    expect(lastDrafts(wrapper)[0]?.libraryFileId).toBe('library-large-1')
    expect(lastDrafts(wrapper).reduce((total, draft) => total + draft.file.size, 0))
      .toBeLessThanOrEqual(CHAT_ATTACHMENT_TOTAL_MAX_BYTES)
    expect(wrapper.text()).toContain('chat.attachments.errors.totalTooLarge 20')
  })

  it('removes a library selection without deleting a chat upload', async () => {
    const wrapper = mountPicker()
    pickerApi(wrapper).addLibraryFiles([libraryFile])
    await wrapper.vm.$nextTick()

    pickerApi(wrapper).remove(lastDrafts(wrapper)[0]!.key)
    await flushPromises()

    expect(apiMocks.upload).not.toHaveBeenCalled()
    expect(apiMocks.remove).not.toHaveBeenCalled()
    expect(lastDrafts(wrapper)).toEqual([])
  })

  it('discards a library selection without deleting a chat upload', async () => {
    const wrapper = mountPicker()
    pickerApi(wrapper).addLibraryFiles([libraryFile])
    await wrapper.vm.$nextTick()

    await pickerApi(wrapper).discardAll()

    expect(apiMocks.upload).not.toHaveBeenCalled()
    expect(apiMocks.remove).not.toHaveBeenCalled()
    expect(lastDrafts(wrapper)).toEqual([])
  })

  it('unmounts a library selection without deleting a chat upload', async () => {
    const wrapper = mountPicker()
    pickerApi(wrapper).addLibraryFiles([libraryFile])
    await wrapper.vm.$nextTick()

    wrapper.unmount()
    wrappers.splice(wrappers.indexOf(wrapper), 1)
    await flushPromises()

    expect(apiMocks.upload).not.toHaveBeenCalled()
    expect(apiMocks.remove).not.toHaveBeenCalled()
  })

  it('reports upload progress, keeps server metadata, and deletes an unbound upload on remove', async () => {
    apiMocks.upload.mockImplementation(async (
      _file: File,
      options: { onProgress?: (progress: number) => void },
    ) => {
      options.onProgress?.(100)
      return attachment
    })
    const wrapper = mountPicker()

    pickerApi(wrapper).addFiles([new File(['png'], 'snow.png', { type: '' })])
    await flushPromises()

    expect(apiMocks.upload).toHaveBeenCalledTimes(1)
    expect(pickerApi(wrapper).getReadyAttachments()).toEqual([attachment])
    expect(lastDrafts(wrapper)[0]).toMatchObject({ state: 'ready', progress: 100 })

    pickerApi(wrapper).remove(lastDrafts(wrapper)[0]!.key)
    await flushPromises()
    expect(apiMocks.remove).toHaveBeenCalledWith('attachment-1')
    expect(lastDrafts(wrapper)).toEqual([])
  })

  it('enforces one document per message before starting a second upload', async () => {
    apiMocks.upload.mockResolvedValue({
      ...attachment,
      id: 'document-1',
      name: 'brief.docx',
      kind: 'document',
      mimeType: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    })
    const wrapper = mountPicker()
    pickerApi(wrapper).addFiles([
      new File(['docx'], 'brief.docx', {
        type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
      }),
      new File(['docx'], 'appendix.docx', {
        type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
      }),
    ])
    await flushPromises()

    expect(apiMocks.upload).toHaveBeenCalledTimes(1)
    expect(lastDrafts(wrapper)).toHaveLength(1)
    expect(wrapper.text()).toContain('chat.attachments.errors.oneDocument')
  })

  it('rejects PDF files before upload while PDF parsing is security-disabled', async () => {
    const wrapper = mountPicker()
    pickerApi(wrapper).addFiles([
      new File(['pdf'], 'brief.pdf', { type: 'application/pdf' }),
    ])
    await flushPromises()

    expect(lastDrafts(wrapper)[0]).toMatchObject({
      state: 'error',
      errorKey: 'chat.attachments.errors.type',
    })
    expect(apiMocks.upload).not.toHaveBeenCalled()
  })

  it('ignores stale progress and deletes a stale upload that resolves after cancel and retry', async () => {
    let resolveFirst!: (value: ChatAttachment) => void
    let resolveSecond!: (value: ChatAttachment) => void
    let firstProgress: ((progress: number) => void) | undefined
    apiMocks.upload
      .mockImplementationOnce((
        _file: File,
        options: { onProgress?: (progress: number) => void },
      ) => {
        firstProgress = options.onProgress
        return new Promise<ChatAttachment>((resolve) => {
          resolveFirst = resolve
        })
      })
      .mockImplementationOnce(() => new Promise<ChatAttachment>((resolve) => {
        resolveSecond = resolve
      }))

    const wrapper = mountPicker()
    pickerApi(wrapper).addFiles([new File(['png'], 'snow.png', { type: 'image/png' })])
    const key = lastDrafts(wrapper)[0]!.key
    pickerApi(wrapper).cancel(key)
    pickerApi(wrapper).retry(key)

    firstProgress?.(100)
    expect(lastDrafts(wrapper)[0]).toMatchObject({ state: 'uploading', progress: 0 })

    const retriedAttachment = { ...attachment, id: 'attachment-retried' }
    resolveSecond(retriedAttachment)
    await flushPromises()
    expect(lastDrafts(wrapper)[0]).toMatchObject({
      state: 'ready',
      attachment: retriedAttachment,
    })

    resolveFirst({ ...attachment, id: 'attachment-stale' })
    await flushPromises()
    expect(lastDrafts(wrapper)[0]).toMatchObject({
      state: 'ready',
      attachment: retriedAttachment,
    })
    expect(apiMocks.remove).toHaveBeenCalledWith('attachment-stale')
    expect(apiMocks.remove).not.toHaveBeenCalledWith('attachment-retried')
  })
})

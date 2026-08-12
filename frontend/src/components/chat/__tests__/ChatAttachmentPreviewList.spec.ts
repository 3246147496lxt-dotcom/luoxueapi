import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import type { ChatAttachment } from '@/types/chat'
import type { ChatAttachmentDraft } from '../chatAttachmentUi'

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

import ChatAttachmentPreviewList from '../ChatAttachmentPreviewList.vue'

const COMPONENT_SOURCE = readFileSync(
  resolve(process.cwd(), 'src/components/chat/ChatAttachmentPreviewList.vue'),
  'utf8',
)
const SCOPED_STYLE = COMPONENT_SOURCE.match(/<style scoped>([\s\S]*?)<\/style>/)?.[1] ?? ''

const IconStub = {
  props: ['name'],
  template: '<span :data-icon="name" />',
}

const mountedWrappers: VueWrapper[] = []

function makeFile(
  name = 'photo.png',
  type = 'image/png',
  size = 1024,
): File {
  return new File([new Uint8Array(size)], name, { type })
}

function makeAttachment(
  overrides: Partial<ChatAttachment> = {},
): ChatAttachment {
  return {
    id: 'attachment-1',
    name: 'photo.png',
    kind: 'image',
    mimeType: 'image/png',
    size: 1024,
    status: 'ready',
    expiresAt: '2099-01-01T00:00:00Z',
    ...overrides,
  }
}

function makeImageDraft(
  overrides: Partial<ChatAttachmentDraft> = {},
): ChatAttachmentDraft {
  const file = overrides.file ?? makeFile()
  return {
    key: 'image-draft-1',
    file,
    kind: 'image',
    state: 'ready',
    progress: 100,
    previewUrl: 'blob:photo-preview',
    attachment: makeAttachment({
      name: file.name,
      mimeType: file.type,
      size: file.size,
    }),
    ...overrides,
  }
}

function makeDocumentDraft(
  overrides: Partial<ChatAttachmentDraft> = {},
): ChatAttachmentDraft {
  const file = overrides.file ?? makeFile(
    'brief.docx',
    'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    2048,
  )
  return {
    key: 'document-draft-1',
    file,
    kind: 'document',
    state: 'ready',
    progress: 100,
    attachment: makeAttachment({
      id: 'document-attachment-1',
      name: file.name,
      kind: 'document',
      mimeType: file.type,
      size: file.size,
    }),
    ...overrides,
  }
}

function mountPreviewList(
  items: ChatAttachmentDraft[],
  props: { supportsVision?: boolean } = {},
) {
  const wrapper = mount(ChatAttachmentPreviewList, {
    attachTo: document.body,
    props: {
      items,
      supportsVision: true,
      ...props,
    },
    global: {
      stubs: {
        Icon: IconStub,
      },
    },
  })
  mountedWrappers.push(wrapper)
  return wrapper
}

afterEach(() => {
  mountedWrappers.splice(0).forEach((wrapper) => wrapper.unmount())
  document.body.innerHTML = ''
  vi.restoreAllMocks()
})

describe('ChatAttachmentPreviewList', () => {
  it('uses a large single-image tile and compact tiles for a multi-image rail', async () => {
    const first = makeImageDraft()
    const second = makeImageDraft({
      key: 'image-draft-2',
      file: makeFile('second.png'),
      previewUrl: 'blob:second-preview',
    })
    const wrapper = mountPreviewList([first])

    expect(wrapper.get('.chat-attachment-preview').classes())
      .toContain('chat-attachment-preview--single-image')
    expect(wrapper.get('[data-test="chat-attachment-item"]').classes())
      .toEqual(expect.arrayContaining([
        'chat-attachment-preview__item--image',
        'chat-attachment-preview__item--ready',
      ]))
    expect(wrapper.get('img').attributes('src')).toBe('blob:photo-preview')

    await wrapper.setProps({ items: [first, second] })

    expect(wrapper.get('.chat-attachment-preview').classes())
      .toContain('chat-attachment-preview--multiple')
    expect(wrapper.get('.chat-attachment-preview').classes())
      .not.toContain('chat-attachment-preview--single-image')
    expect(wrapper.findAll('[data-test="chat-attachment-item"]')).toHaveLength(2)
    expect(wrapper.findAll('img').map((image) => image.attributes('src')))
      .toEqual(['blob:photo-preview', 'blob:second-preview'])

    expect(SCOPED_STYLE).toMatch(
      /\.chat-attachment-preview__item--image\s*\{[^}]*width: 56px;[^}]*height: 56px;/,
    )
    expect(SCOPED_STYLE).toMatch(
      /\.chat-attachment-preview--single-image \.chat-attachment-preview__item--image\s*\{[^}]*width: 144px;[^}]*height: 144px;/,
    )
  })

  it('keeps ready image metadata visually quiet while preserving an accessible status', () => {
    const wrapper = mountPreviewList([makeImageDraft()])
    const item = wrapper.get('[data-test="chat-attachment-item"]')

    expect(item.find('.chat-attachment-preview__document-copy').exists()).toBe(false)
    expect(item.find('strong').exists()).toBe(false)
    expect(item.get('.sr-only').text()).toBe('photo.png: chat.attachments.ready')
  })

  it('labels the remove control, exposes its tooltip, and emits the attachment key', async () => {
    const wrapper = mountPreviewList([makeImageDraft()])
    const remove = wrapper.get('[data-test="chat-attachment-remove"]')
    const tooltip = document.querySelector<HTMLElement>('[data-test="chat-control-tooltip"]')

    expect(remove.attributes('aria-label')).toBe('chat.attachments.remove photo.png')
    expect(tooltip?.getAttribute('aria-hidden')).toBe('true')
    expect(tooltip?.querySelector('.chat-control-tooltip__label')?.textContent)
      .toBe('chat.attachments.removeFile')

    await remove.trigger('click')

    expect(wrapper.emitted('remove')).toEqual([['image-draft-1']])
  })

  it('renders uploading and processing images as distinct, politely announced states', () => {
    const wrapper = mountPreviewList([
      makeImageDraft({
        key: 'uploading-image',
        state: 'uploading',
        progress: 42,
        attachment: undefined,
      }),
      makeImageDraft({
        key: 'processing-image',
        state: 'processing',
        progress: 100,
        attachment: undefined,
      }),
    ])
    const [uploading, processing] = wrapper.findAll('[data-test="chat-attachment-item"]')

    expect(uploading?.attributes('data-state')).toBe('uploading')
    expect(uploading?.get('.chat-attachment-preview__progress-copy').text()).toBe('42%')
    expect(uploading?.get('progress').attributes()).toMatchObject({
      max: '100',
      value: '42',
      'aria-label': 'chat.attachments.uploadProgress 42',
    })
    expect(uploading?.get('.sr-only').text())
      .toBe('photo.png: chat.attachments.uploading 42')

    expect(processing?.attributes('data-state')).toBe('processing')
    expect(processing?.find('.chat-attachment-preview__progress-copy').exists()).toBe(false)
    expect(processing?.find('progress').exists()).toBe(false)
    expect(processing?.get('.sr-only').text())
      .toBe('photo.png: chat.attachments.processing')
  })

  it('keeps a failed image in the rail and emits retry from its accessible action', async () => {
    const wrapper = mountPreviewList([makeImageDraft({
      key: 'failed-image',
      state: 'error',
      progress: 0,
      attachment: undefined,
      errorKey: 'chat.attachments.errors.uploadFailed',
    })])
    const item = wrapper.get('[data-test="chat-attachment-item"]')
    const retry = item.get('[data-test="chat-attachment-retry"]')

    expect(item.classes()).toContain('chat-attachment-preview__item--error')
    expect(item.get('.chat-attachment-preview__scrim').classes())
      .toContain('chat-attachment-preview__scrim--error')
    expect(item.get('[data-icon="exclamationCircle"]').attributes('data-icon'))
      .toBe('exclamationCircle')
    expect(item.get('.sr-only').text())
      .toBe('photo.png: chat.attachments.errors.uploadFailed')
    expect(retry.attributes()).toMatchObject({
      'aria-label': 'chat.attachments.retry photo.png',
      title: 'chat.attachments.retry photo.png',
    })

    await retry.trigger('click')

    expect(wrapper.emitted('retry')).toEqual([['failed-image']])
  })

  it('falls back to a compact document card for a ready DOCX attachment', () => {
    const wrapper = mountPreviewList([makeDocumentDraft()])
    const item = wrapper.get('[data-test="chat-attachment-item"]')
    const copy = item.get('.chat-attachment-preview__document-copy')

    expect(item.attributes()).toMatchObject({
      'data-kind': 'document',
      'data-state': 'ready',
    })
    expect(item.find('img').exists()).toBe(false)
    expect(item.get('[data-icon="document"]').attributes('data-icon')).toBe('document')
    expect(copy.get('strong').text()).toBe('brief.docx')
    expect(copy.get('strong').attributes('title')).toBe('brief.docx')
    expect(copy.get('span').text()).toBe('2 KB · chat.attachments.docxTextOnly')
    expect(item.get('.sr-only').text()).toBe('brief.docx: chat.attachments.ready')
  })
})

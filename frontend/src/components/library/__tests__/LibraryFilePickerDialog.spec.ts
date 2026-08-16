import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { CHAT_ATTACHMENT_TOTAL_MAX_BYTES } from '@/components/chat/chatAttachmentUi'
import type { LibraryFile } from '@/types/library'

const apiMocks = vi.hoisted(() => ({
  list: vi.fn(),
  thumbnail: vi.fn(),
}))

vi.mock('@/api/library', () => ({
  listLibraryFiles: apiMocks.list,
  getLibraryThumbnail: apiMocks.thumbnail,
  isLibraryAbortError: (error: unknown) => (
    !!error && typeof error === 'object' && (error as { name?: unknown }).name === 'AbortError'
  ),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const { ref } = await vi.importActual<typeof import('vue')>('vue')
  return {
    ...actual,
    useI18n: () => ({
      locale: ref('zh-CN'),
      t: (key: string, params?: Record<string, unknown>) => (
        params ? `${key} ${Object.values(params).join(' ')}` : key
      ),
    }),
  }
})

import LibraryFilePickerDialog from '../LibraryFilePickerDialog.vue'

const baseFile: LibraryFile = {
  id: 'library-file-1',
  name: 'first.png',
  mimeType: 'image/png',
  size: 12 * 1024 * 1024,
  source: 'uploaded',
  type: 'image',
  category: 'image',
  status: 'ready',
  createdAt: '2026-08-14T01:00:00.000Z',
  updatedAt: '2026-08-14T02:00:00.000Z',
}

let wrapper: VueWrapper | undefined

function mountPicker() {
  return mount(LibraryFilePickerDialog, {
    props: {
      show: false,
      maxSelection: 4,
      maxSelectionBytes: CHAT_ATTACHMENT_TOTAL_MAX_BYTES,
      totalLimitBytes: CHAT_ATTACHMENT_TOTAL_MAX_BYTES,
      supportsVision: true,
    },
    global: {
      stubs: {
        BaseDialog: {
          props: ['show', 'title'],
          emits: ['close'],
          template: '<div v-if="show"><slot /><footer><slot name="footer" /></footer></div>',
        },
        LibraryFileVisual: { template: '<span />' },
        Icon: { template: '<span />' },
      },
    },
  })
}

beforeEach(() => {
  vi.resetAllMocks()
  apiMocks.thumbnail.mockResolvedValue(new Blob(['thumbnail'], { type: 'image/jpeg' }))
  apiMocks.list.mockResolvedValue({
    files: [
      baseFile,
      {
        ...baseFile,
        id: 'library-file-2',
        name: 'second.png',
        size: 9 * 1024 * 1024,
      },
    ],
    total: 2,
    page: 1,
    pageSize: 60,
  })
})

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
})

describe('LibraryFilePickerDialog', () => {
  it('disables and explains a choice that would exceed the available chat bytes', async () => {
    wrapper = mountPicker()
    await wrapper.setProps({ show: true })
    await flushPromises()

    const options = wrapper.findAll<HTMLButtonElement>('[aria-pressed]')
    expect(options).toHaveLength(2)
    await options[0]!.trigger('click')

    expect(options[0]!.attributes('aria-pressed')).toBe('true')
    expect(options[1]!.attributes('aria-disabled')).toBe('true')
    expect(options[1]!.attributes('title')).toBe('chat.attachments.errors.totalTooLarge 20')

    await options[1]!.trigger('click')
    expect(wrapper.get('[role="status"]').text()).toBe('chat.attachments.errors.totalTooLarge 20')
    expect(wrapper.get('.library-picker__confirm').text()).toBe('library.picker.addOne')
  })

  it('uses native list and toggle-button semantics with localized type labels', async () => {
    wrapper = mountPicker()
    await wrapper.setProps({ show: true })
    await flushPromises()

    const list = wrapper.get('[role="list"]')
    expect(list.attributes('aria-label')).toBe('library.picker.resultsLabel')
    expect(list.findAll('li')).toHaveLength(2)
    expect(list.findAll('[role="option"]')).toHaveLength(0)
    expect(list.findAll('[aria-pressed="false"]')).toHaveLength(2)
    expect(list.findAll('.library-picker__meta')[0]!.text()).toContain(
      'library.filters.types.image',
    )
  })

  it('revalidates an existing choice when the remaining chat byte budget changes', async () => {
    wrapper = mountPicker()
    await wrapper.setProps({ show: true })
    await flushPromises()

    const first = wrapper.findAll<HTMLButtonElement>('[aria-pressed]')[0]!
    await first.trigger('click')
    expect(wrapper.get<HTMLButtonElement>('.library-picker__confirm').element.disabled).toBe(false)

    await wrapper.setProps({ maxSelectionBytes: 4 * 1024 * 1024 })
    expect(wrapper.get<HTMLButtonElement>('.library-picker__confirm').element.disabled).toBe(true)

    await first.trigger('click')
    expect(first.attributes('aria-pressed')).toBe('false')
  })

  it('loads subsequent result pages without losing the current selection', async () => {
    apiMocks.list
      .mockResolvedValueOnce({
        files: [baseFile],
        total: 2,
        page: 1,
        pageSize: 60,
      })
      .mockResolvedValueOnce({
        files: [{ ...baseFile, id: 'library-file-2', name: 'second.png' }],
        total: 2,
        page: 2,
        pageSize: 60,
      })

    wrapper = mountPicker()
    await wrapper.setProps({ show: true })
    await flushPromises()
    await wrapper.get('[aria-pressed]').trigger('click')

    await wrapper.get('.library-picker__load-more').trigger('click')
    await flushPromises()

    expect(apiMocks.list).toHaveBeenLastCalledWith(
      expect.objectContaining({ page: 2, pageSize: 60 }),
      expect.any(AbortSignal),
    )
    const options = wrapper.findAll('[aria-pressed]')
    expect(options).toHaveLength(2)
    expect(options[0]!.attributes('aria-pressed')).toBe('true')
    expect(wrapper.find('.library-picker__load-more').exists()).toBe(false)
  })
})

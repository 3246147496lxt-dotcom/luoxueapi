import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'
import type { LibraryFile } from '@/types/library'

const apiMocks = vi.hoisted(() => ({
  preview: vi.fn(),
}))

vi.mock('@/api/library', () => ({
  getLibraryPreview: apiMocks.preview,
  isLibraryAbortError: (error: unknown) => (
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

import LibraryFilePreviewDialog from '../LibraryFilePreviewDialog.vue'

interface Deferred<T> {
  promise: Promise<T>
  resolve: (value: T) => void
  reject: (reason?: unknown) => void
}

function deferred<T>(): Deferred<T> {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

function imageFile(overrides: Partial<LibraryFile> = {}): LibraryFile {
  return {
    id: 'library-image-1',
    name: 'snow.png',
    mimeType: 'image/png',
    size: 4096,
    source: 'uploaded',
    type: 'image',
    category: 'image',
    status: 'ready',
    previewable: true,
    createdAt: '2026-08-14T01:00:00.000Z',
    updatedAt: '2026-08-14T02:00:00.000Z',
    ...overrides,
  }
}

function pdfFile(overrides: Partial<LibraryFile> = {}): LibraryFile {
  return imageFile({
    id: 'library-pdf-1',
    name: 'brief.pdf',
    mimeType: 'application/pdf',
    type: 'pdf',
    category: 'file',
    ...overrides,
  })
}

let wrapper: VueWrapper | undefined
let host: HTMLDivElement | undefined
let objectUrlSequence = 0

function mountViewer(file: LibraryFile, show = true): VueWrapper {
  host = document.createElement('div')
  document.body.appendChild(host)
  wrapper = mount(LibraryFilePreviewDialog, {
    attachTo: host,
    props: { show, file },
    global: {
      stubs: {
        Icon: {
          props: ['name'],
          template: '<span :data-icon="name" />',
        },
      },
    },
  })
  return wrapper
}

beforeEach(() => {
  vi.resetAllMocks()
  objectUrlSequence = 0
  vi.stubGlobal('URL', {
    createObjectURL: vi.fn(() => `blob:library-preview-${++objectUrlSequence}`),
    revokeObjectURL: vi.fn(),
  })
  apiMocks.preview.mockResolvedValue(new Blob(['preview'], { type: 'image/png' }))
})

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  host?.remove()
  host = undefined
  vi.unstubAllGlobals()
})

describe('LibraryFilePreviewDialog', () => {
  it('loads the authenticated preview Blob and waits for image decoding before ending loading', async () => {
    const blob = new Blob(['safe image'], { type: 'image/png' })
    apiMocks.preview.mockResolvedValueOnce(blob)

    const view = mountViewer(imageFile())
    await flushPromises()

    expect(apiMocks.preview).toHaveBeenCalledWith(
      'library-image-1',
      expect.any(AbortSignal),
    )
    expect(URL.createObjectURL).toHaveBeenCalledWith(blob)
    expect(view.get('img').attributes('src')).toBe('blob:library-preview-1')
    expect(view.get('img').classes()).toContain('library-viewer__asset--loading')
    expect(view.get('[role="status"]').text()).toContain('library.preview.loading')

    await view.get('img').trigger('load')

    expect(view.find('[role="status"]').exists()).toBe(false)
    expect(view.get('img').classes()).not.toContain('library-viewer__asset--loading')
  })

  it('emits the current file for download and removal actions', async () => {
    const file = imageFile()
    const view = mountViewer(file)
    await flushPromises()

    await view.get('.library-viewer__pill--danger').trigger('click')
    await view.get('button[aria-label^="library.actions.downloadNamed"]').trigger('click')

    expect(view.emitted('remove')).toEqual([[file]])
    expect(view.emitted('download')).toEqual([[file]])
  })

  it('switches between fit-to-window and actual image dimensions', async () => {
    const view = mountViewer(imageFile())
    await flushPromises()

    const toggle = view.get('button[aria-label="library.preview.actualSize"]')
    expect(toggle.attributes('aria-pressed')).toBe('false')
    expect(view.get('.library-viewer__stage').classes()).not.toContain(
      'library-viewer__stage--actual',
    )

    await toggle.trigger('click')

    expect(view.get('button[aria-label="library.preview.fitToWindow"]').attributes('aria-pressed'))
      .toBe('true')
    expect(view.get('.library-viewer__stage').classes()).toContain(
      'library-viewer__stage--actual',
    )

    await view.get('button[aria-label="library.preview.fitToWindow"]').trigger('click')

    expect(view.get('button[aria-label="library.preview.actualSize"]').attributes('aria-pressed'))
      .toBe('false')
    expect(view.get('.library-viewer__stage').classes()).not.toContain(
      'library-viewer__stage--actual',
    )
  })

  it('closes on Escape and restores focus after the parent hides the viewer', async () => {
    const opener = document.createElement('button')
    opener.textContent = 'Open preview'
    document.body.appendChild(opener)
    opener.focus()

    const view = mountViewer(imageFile())
    await flushPromises()

    expect(document.activeElement).toBe(view.get('.library-viewer__close').element)

    const escape = new KeyboardEvent('keydown', { key: 'Escape', cancelable: true })
    document.dispatchEvent(escape)

    expect(escape.defaultPrevented).toBe(true)
    expect(view.emitted('close')).toHaveLength(1)

    await view.setProps({ show: false })
    await nextTick()
    await flushPromises()

    expect(document.activeElement).toBe(opener)
    opener.remove()
  })

  it('aborts obsolete work, ignores a stale Blob, and revokes the active URL on close', async () => {
    const first = deferred<Blob>()
    const second = deferred<Blob>()
    let firstSignal: AbortSignal | undefined
    let secondSignal: AbortSignal | undefined
    apiMocks.preview
      .mockImplementationOnce((_id: string, signal: AbortSignal) => {
        firstSignal = signal
        return first.promise
      })
      .mockImplementationOnce((_id: string, signal: AbortSignal) => {
        secondSignal = signal
        return second.promise
      })

    const firstFile = imageFile()
    const secondFile = imageFile({
      id: 'library-image-2',
      name: 'second.png',
      updatedAt: '2026-08-14T03:00:00.000Z',
    })
    const view = mountViewer(firstFile)
    await nextTick()

    expect(firstSignal?.aborted).toBe(false)

    await view.setProps({ file: secondFile })
    await nextTick()

    expect(firstSignal?.aborted).toBe(true)
    expect(secondSignal?.aborted).toBe(false)

    const staleBlob = new Blob(['stale'], { type: 'image/png' })
    first.resolve(staleBlob)
    await flushPromises()

    expect(URL.createObjectURL).not.toHaveBeenCalledWith(staleBlob)

    const activeBlob = new Blob(['active'], { type: 'image/png' })
    second.resolve(activeBlob)
    await flushPromises()

    expect(URL.createObjectURL).toHaveBeenCalledTimes(1)
    expect(URL.createObjectURL).toHaveBeenCalledWith(activeBlob)
    expect(view.get('img').attributes('src')).toBe('blob:library-preview-1')

    await view.setProps({ show: false })
    await flushPromises()

    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:library-preview-1')
    expect(view.find('img').exists()).toBe(false)
  })

  it('keeps PDF previews sandboxed and strips referrer information', async () => {
    const blob = new Blob(['%PDF-1.7'], { type: 'application/pdf' })
    apiMocks.preview.mockResolvedValueOnce(blob)

    const view = mountViewer(pdfFile())
    await flushPromises()

    const frame = view.get('iframe')
    expect(apiMocks.preview).toHaveBeenCalledWith('library-pdf-1', expect.any(AbortSignal))
    expect(frame.attributes('src')).toBe('blob:library-preview-1')
    expect(frame.attributes('sandbox')).toBe('')
    expect(frame.attributes('referrerpolicy')).toBe('no-referrer')
    expect(frame.attributes('title')).toContain('brief.pdf')
    expect(view.find('button[aria-label="library.preview.actualSize"]').exists()).toBe(false)

    await frame.trigger('load')
    expect(view.find('[role="status"]').exists()).toBe(false)
  })
})

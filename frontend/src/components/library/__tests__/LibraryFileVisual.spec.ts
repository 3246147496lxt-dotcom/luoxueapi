import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { LibraryFile } from '@/types/library'

const apiMocks = vi.hoisted(() => ({
  getThumbnail: vi.fn(),
}))

vi.mock('@/api/library', () => ({
  getLibraryThumbnail: apiMocks.getThumbnail,
  isLibraryAbortError: (error: unknown) => (
    !!error && typeof error === 'object' && (error as { name?: unknown }).name === 'AbortError'
  ),
}))

import LibraryFileVisual from '../LibraryFileVisual.vue'

const wrappers: VueWrapper[] = []
let fileSequence = 0

function imageFile(overrides: Partial<LibraryFile> = {}): LibraryFile {
  fileSequence += 1
  return {
    id: `image-${fileSequence}`,
    name: 'snow.png',
    mimeType: 'image/png',
    size: 1024,
    source: 'uploaded',
    type: 'image',
    category: 'image',
    status: 'ready',
    createdAt: '2026-08-14T01:00:00.000Z',
    updatedAt: '2026-08-14T02:00:00.000Z',
    ...overrides,
  }
}

function mountVisual(file: LibraryFile): VueWrapper {
  const wrapper = mount(LibraryFileVisual, {
    props: { file },
    global: {
      stubs: {
        Icon: { props: ['name'], template: '<span :data-icon="name" />' },
      },
    },
  })
  wrappers.push(wrapper)
  return wrapper
}

function unmount(wrapper: VueWrapper): void {
  wrapper.unmount()
  wrappers.splice(wrappers.indexOf(wrapper), 1)
}

beforeEach(() => {
  vi.resetAllMocks()
  vi.stubGlobal('IntersectionObserver', undefined)
  let objectUrlSequence = 0
  vi.stubGlobal('URL', {
    createObjectURL: vi.fn(() => `blob:library-thumbnail-${++objectUrlSequence}`),
    revokeObjectURL: vi.fn(),
  })
})

afterEach(() => {
  wrappers.splice(0).forEach((wrapper) => wrapper.unmount())
  vi.unstubAllGlobals()
})

describe('LibraryFileVisual', () => {
  it('does not request an offscreen thumbnail before IntersectionObserver admits it', async () => {
    let callback!: IntersectionObserverCallback
    const observe = vi.fn()
    const disconnect = vi.fn()
    class IntersectionObserverStub {
      observe = observe
      disconnect = disconnect
      unobserve = vi.fn()

      constructor(nextCallback: IntersectionObserverCallback) {
        callback = nextCallback
      }
    }
    vi.stubGlobal('IntersectionObserver', IntersectionObserverStub)
    apiMocks.getThumbnail.mockResolvedValue(new Blob(['image'], { type: 'image/png' }))

    const wrapper = mountVisual(imageFile())
    await flushPromises()

    expect(observe).toHaveBeenCalledWith(wrapper.get('.library-file-visual').element)
    expect(apiMocks.getThumbnail).not.toHaveBeenCalled()

    callback([{
      isIntersecting: true,
      intersectionRatio: 0.1,
      target: wrapper.get('.library-file-visual').element,
    } as IntersectionObserverEntry], {} as IntersectionObserver)
    await flushPromises()

    expect(apiMocks.getThumbnail).toHaveBeenCalledTimes(1)
    expect(wrapper.get('img').attributes('src')).toBe('blob:library-thumbnail-1')
    expect(disconnect).toHaveBeenCalled()
  })

  it('reuses the session Blob cache after the component is rebuilt', async () => {
    const file = imageFile({ id: 'cache-rebuild', updatedAt: '2026-08-14T03:00:00.000Z' })
    apiMocks.getThumbnail.mockResolvedValue(new Blob(['cached-image'], { type: 'image/png' }))

    const first = mountVisual(file)
    await flushPromises()
    expect(apiMocks.getThumbnail).toHaveBeenCalledTimes(1)
    expect(first.get('img').attributes('src')).toBe('blob:library-thumbnail-1')

    unmount(first)
    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:library-thumbnail-1')

    const rebuilt = mountVisual(file)
    await flushPromises()
    expect(apiMocks.getThumbnail).toHaveBeenCalledTimes(1)
    expect(rebuilt.get('img').attributes('src')).toBe('blob:library-thumbnail-2')
  })

  it('aborts an active subscriber and never installs a stale URL after unmount', async () => {
    let requestSignal: AbortSignal | undefined
    apiMocks.getThumbnail.mockImplementation((_id: string, signal: AbortSignal) => {
      requestSignal = signal
      return new Promise<Blob>((_resolve, reject) => {
        signal.addEventListener('abort', () => {
          reject(new DOMException('aborted', 'AbortError'))
        }, { once: true })
      })
    })

    const wrapper = mountVisual(imageFile())
    await flushPromises()
    expect(requestSignal?.aborted).toBe(false)

    unmount(wrapper)
    await flushPromises()

    expect(requestSignal?.aborted).toBe(true)
    expect(URL.createObjectURL).not.toHaveBeenCalled()
  })
})

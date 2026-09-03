import { describe, expect, it, vi } from 'vitest'
import { createLibraryThumbnailLoader } from '../libraryThumbnailLoader'

function flushMicrotasks(): Promise<void> {
  return new Promise((resolve) => queueMicrotask(resolve))
}

describe('libraryThumbnailLoader', () => {
  it('never starts more than four thumbnail requests concurrently', async () => {
    const releases: Array<() => void> = []
    const fetchThumbnail = vi.fn((id: string) => new Promise<Blob>((resolve) => {
      releases.push(() => resolve(new Blob([id], { type: 'image/png' })))
    }))
    const loader = createLibraryThumbnailLoader(fetchThumbnail, 4)

    const requests = Array.from({ length: 6 }, (_, index) => loader.load({
      id: `concurrent-${index}`,
      updatedAt: '2026-08-14T02:00:00.000Z',
    }))

    expect(fetchThumbnail).toHaveBeenCalledTimes(4)

    releases[0]()
    await flushMicrotasks()
    expect(fetchThumbnail).toHaveBeenCalledTimes(5)

    releases[1]()
    await flushMicrotasks()
    expect(fetchThumbnail).toHaveBeenCalledTimes(6)

    for (const release of releases.slice(2)) release()
    await Promise.all(requests)
  })

  it('deduplicates an in-flight revision, caches it, and refetches after updatedAt changes', async () => {
    let release!: (blob: Blob) => void
    const fetchThumbnail = vi.fn(() => new Promise<Blob>((resolve) => {
      release = resolve
    }))
    const loader = createLibraryThumbnailLoader(fetchThumbnail, 4)
    const identity = { id: 'shared-file', updatedAt: 'revision-1' }

    const first = loader.load(identity)
    const second = loader.load(identity)
    expect(fetchThumbnail).toHaveBeenCalledTimes(1)

    const blob = new Blob(['shared'], { type: 'image/png' })
    release(blob)
    await expect(Promise.all([first, second])).resolves.toEqual([blob, blob])

    await expect(loader.load(identity)).resolves.toBe(blob)
    expect(fetchThumbnail).toHaveBeenCalledTimes(1)

    const changed = loader.load({ ...identity, updatedAt: 'revision-2' })
    expect(fetchThumbnail).toHaveBeenCalledTimes(2)
    const changedBlob = new Blob(['changed'], { type: 'image/png' })
    release(changedBlob)
    await expect(changed).resolves.toBe(changedBlob)
  })

  it('keeps shared work alive until the final subscriber aborts', async () => {
    let upstreamSignal!: AbortSignal
    const fetchThumbnail = vi.fn((_id: string, signal: AbortSignal) => {
      upstreamSignal = signal
      return new Promise<Blob>((_resolve, reject) => {
        signal.addEventListener('abort', () => reject(new DOMException('aborted', 'AbortError')), {
          once: true,
        })
      })
    })
    const loader = createLibraryThumbnailLoader(fetchThumbnail, 4)
    const firstController = new AbortController()
    const secondController = new AbortController()
    const identity = { id: 'shared-abort', updatedAt: 'revision-1' }

    const first = loader.load(identity, firstController.signal)
    const second = loader.load(identity, secondController.signal)
    firstController.abort()

    await expect(first).rejects.toMatchObject({ name: 'AbortError' })
    expect(upstreamSignal.aborted).toBe(false)

    secondController.abort()
    await expect(second).rejects.toMatchObject({ name: 'AbortError' })
    expect(upstreamSignal.aborted).toBe(true)
  })
})

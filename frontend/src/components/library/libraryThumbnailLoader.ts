import { getLibraryThumbnail } from '@/api/library'

const DEFAULT_MAX_CONCURRENT = 4
const MAX_SESSION_CACHE_ENTRIES = 256

export interface LibraryThumbnailIdentity {
  id: string
  updatedAt: string
}

type ThumbnailFetcher = (id: string, signal: AbortSignal) => Promise<Blob>

interface Subscriber {
  signal?: AbortSignal
  abortListener?: () => void
  settled: boolean
  resolve: (blob: Blob) => void
  reject: (error: unknown) => void
}

interface ThumbnailJob {
  key: string
  fileId: string
  controller: AbortController
  state: 'queued' | 'active' | 'cancelled' | 'settled'
  subscribers: Set<Subscriber>
}

export interface LibraryThumbnailLoader {
  load: (identity: LibraryThumbnailIdentity, signal?: AbortSignal) => Promise<Blob>
}

function thumbnailKey(identity: LibraryThumbnailIdentity): string {
  return JSON.stringify([identity.id.trim(), identity.updatedAt.trim()])
}

function abortError(): DOMException {
  return new DOMException('The operation was aborted.', 'AbortError')
}

function concurrencyLimit(value: number): number {
  return Number.isFinite(value) && value > 0
    ? Math.max(1, Math.floor(value))
    : DEFAULT_MAX_CONCURRENT
}

/**
 * Builds a subscriber-aware thumbnail request pool. Calls for the same file
 * revision share one request; cancelling one subscriber does not interrupt the
 * other consumers, while cancelling the final subscriber aborts queued or
 * active work.
 */
export function createLibraryThumbnailLoader(
  fetchThumbnail?: ThumbnailFetcher,
  maxConcurrent = DEFAULT_MAX_CONCURRENT,
): LibraryThumbnailLoader {
  // Resolve the API lazily. Some parent surfaces mock only the API methods
  // they exercise; touching this named export during module evaluation would
  // make a document-only library view fail before any thumbnail is requested.
  const fetcher = fetchThumbnail ?? ((id: string, signal: AbortSignal) => (
    getLibraryThumbnail(id, signal)
  ))
  const limit = concurrencyLimit(maxConcurrent)
  const cache = new Map<string, Blob>()
  const pending = new Map<string, ThumbnailJob>()
  const queue: ThumbnailJob[] = []
  let activeCount = 0

  function cachedBlob(key: string): Blob | undefined {
    const blob = cache.get(key)
    if (!blob) return undefined
    // Touch the entry so the bounded session cache behaves as an LRU.
    cache.delete(key)
    cache.set(key, blob)
    return blob
  }

  function cacheBlob(key: string, blob: Blob): void {
    cache.delete(key)
    cache.set(key, blob)
    while (cache.size > MAX_SESSION_CACHE_ENTRIES) {
      const oldest = cache.keys().next().value as string | undefined
      if (oldest === undefined) break
      cache.delete(oldest)
    }
  }

  function settleSubscriber(
    job: ThumbnailJob,
    subscriber: Subscriber,
    outcome: { blob: Blob } | { error: unknown },
  ): void {
    if (subscriber.settled) return
    subscriber.settled = true
    job.subscribers.delete(subscriber)
    if (subscriber.signal && subscriber.abortListener) {
      subscriber.signal.removeEventListener('abort', subscriber.abortListener)
    }
    if ('blob' in outcome) subscriber.resolve(outcome.blob)
    else subscriber.reject(outcome.error)
  }

  function cancelJob(job: ThumbnailJob): void {
    if (job.state === 'cancelled' || job.state === 'settled') return
    const wasQueued = job.state === 'queued'
    job.state = 'cancelled'
    job.controller.abort()
    if (pending.get(job.key) === job) pending.delete(job.key)
    if (wasQueued) pump()
  }

  async function run(job: ThumbnailJob): Promise<void> {
    try {
      const blob = await fetcher(job.fileId, job.controller.signal)
      if (job.state === 'cancelled' || job.controller.signal.aborted || job.subscribers.size === 0) {
        return
      }
      cacheBlob(job.key, blob)
      for (const subscriber of [...job.subscribers]) {
        settleSubscriber(job, subscriber, { blob })
      }
    } catch (error) {
      for (const subscriber of [...job.subscribers]) {
        settleSubscriber(job, subscriber, { error })
      }
    } finally {
      job.state = 'settled'
      activeCount = Math.max(0, activeCount - 1)
      if (pending.get(job.key) === job) pending.delete(job.key)
      pump()
    }
  }

  function pump(): void {
    while (activeCount < limit) {
      const job = queue.shift()
      if (!job) return
      if (job.state !== 'queued') continue
      if (job.subscribers.size === 0) {
        cancelJob(job)
        continue
      }
      job.state = 'active'
      activeCount += 1
      void run(job)
    }
  }

  function load(identity: LibraryThumbnailIdentity, signal?: AbortSignal): Promise<Blob> {
    if (signal?.aborted) return Promise.reject(abortError())

    const key = thumbnailKey(identity)
    const cached = cachedBlob(key)
    if (cached) return Promise.resolve(cached)

    let job = pending.get(key)
    const isNewJob = !job
    if (!job) {
      job = {
        key,
        fileId: identity.id.trim(),
        controller: new AbortController(),
        state: 'queued',
        subscribers: new Set(),
      }
      pending.set(key, job)
    }
    const activeJob = job

    const result = new Promise<Blob>((resolve, reject) => {
      const subscriber: Subscriber = { signal, settled: false, resolve, reject }
      subscriber.abortListener = () => {
        settleSubscriber(activeJob, subscriber, { error: abortError() })
        if (activeJob.subscribers.size === 0) cancelJob(activeJob)
      }
      activeJob.subscribers.add(subscriber)
      signal?.addEventListener('abort', subscriber.abortListener, { once: true })
    })

    if (isNewJob) {
      queue.push(activeJob)
      pump()
    }
    return result
  }

  return { load }
}

/** One pool is shared by every LibraryFileVisual instance on the page. */
export const libraryThumbnailLoader = createLibraryThumbnailLoader()

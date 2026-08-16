import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const libraryMocks = vi.hoisted(() => ({
  list: vi.fn(),
  upload: vi.fn(),
  remove: vi.fn(),
  storage: vi.fn(),
}))

vi.mock('@/api/library', () => {
  class MockLibraryAPIError extends Error {
    readonly status: number
    readonly code: string
    readonly metadata: unknown

    constructor(
      message: string,
      options: { status?: number; code?: string; metadata?: unknown } = {},
    ) {
      super(message)
      this.name = 'LibraryAPIError'
      this.status = options.status ?? 0
      this.code = options.code ?? 'LIBRARY_REQUEST_FAILED'
      this.metadata = options.metadata
    }
  }

  return {
    LibraryAPIError: MockLibraryAPIError,
    listLibraryFiles: libraryMocks.list,
    uploadLibraryFile: libraryMocks.upload,
    deleteLibraryFile: libraryMocks.remove,
    getLibraryStorage: libraryMocks.storage,
    isLibraryAbortError: (error: unknown) => (
      !!error && typeof error === 'object' && (error as { name?: unknown }).name === 'AbortError'
    ),
  }
})

import { LibraryAPIError } from '@/api/library'
import { useLibraryStore } from '@/stores/library'
import type { LibraryFile, LibraryFilePage } from '@/types/library'

function libraryFile(id: string, overrides: Partial<LibraryFile> = {}): LibraryFile {
  return {
    id,
    name: `${id}.pdf`,
    mimeType: 'application/pdf',
    size: 2048,
    source: 'uploaded',
    type: 'pdf',
    category: 'file',
    status: 'ready',
    createdAt: '2026-08-13T10:00:00Z',
    updatedAt: '2026-08-14T10:00:00Z',
    ...overrides,
  }
}

function page(
  files: LibraryFile[],
  overrides: Partial<LibraryFilePage> = {},
): LibraryFilePage {
  return {
    files,
    total: files.length,
    page: 1,
    pageSize: 40,
    ...overrides,
  }
}

function deferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((promiseResolve, promiseReject) => {
    resolve = promiseResolve
    reject = promiseReject
  })
  return { promise, resolve, reject }
}

beforeEach(() => {
  setActivePinia(createPinia())
  localStorage.clear()
  vi.resetAllMocks()
  libraryMocks.list.mockResolvedValue(page([]))
  libraryMocks.storage.mockResolvedValue({
    usedBytes: 0,
    limitBytes: 1024 * 1024,
    remainingBytes: 1024 * 1024,
    overLimit: false,
    unlimited: false,
  })
  libraryMocks.remove.mockResolvedValue(undefined)
})

describe('useLibraryStore', () => {
  it('persists the view preference and emits one complete default query', async () => {
    localStorage.setItem('luoxue-library-view', 'grid')
    const store = useLibraryStore()

    expect(store.currentView).toBe('grid')
    store.setView('list')
    expect(localStorage.getItem('luoxue-library-view')).toBe('list')

    store.setSearchKeyword('brief')
    store.setCategory('file')
    store.setFilters({ source: 'generated', type: 'document' })
    store.setSort('name_asc')
    await store.load()

    expect(libraryMocks.list).toHaveBeenCalledWith({
      q: 'brief',
      category: 'file',
      source: 'generated',
      type: 'document',
      sort: 'name_asc',
      page: 1,
      pageSize: 40,
    }, expect.any(AbortSignal))
  })

  it('delegates raw modified-time descending order to the paginated API and preserves ties', async () => {
    const sharedTimestamp = '2026-08-14T13:38:00+08:00'
    libraryMocks.list.mockResolvedValueOnce(page([
      libraryFile('second', { updatedAt: sharedTimestamp }),
      libraryFile('first', { updatedAt: sharedTimestamp }),
    ]))
    const store = useLibraryStore()

    await store.load()

    expect(libraryMocks.list).toHaveBeenCalledWith(
      expect.objectContaining({ sort: 'updated_desc', page: 1, pageSize: 40 }),
      expect.any(AbortSignal),
    )
    expect(store.files.map(({ id }) => id)).toEqual(['second', 'first'])
  })

  it('aborts an obsolete list request and ignores its late response', async () => {
    const first = deferred<LibraryFilePage>()
    const second = deferred<LibraryFilePage>()
    libraryMocks.list
      .mockReturnValueOnce(first.promise)
      .mockReturnValueOnce(second.promise)
    const store = useLibraryStore()

    const firstLoad = store.load()
    const firstSignal = libraryMocks.list.mock.calls[0]?.[1] as AbortSignal
    store.setSearchKeyword('latest')
    const secondLoad = store.load()

    expect(firstSignal.aborted).toBe(true)
    second.resolve(page([libraryFile('latest')], { total: 1 }))
    await secondLoad
    first.resolve(page([libraryFile('stale')], { total: 1 }))
    await firstLoad

    expect(store.files.map(({ id }) => id)).toEqual(['latest'])
    expect(store.loading).toBe(false)
    expect(store.initialized).toBe(true)
  })

  it('appends pages without duplicates and keeps selections that remain visible', async () => {
    const firstPage = [libraryFile('a'), libraryFile('b')]
    const secondPage = [libraryFile('b'), libraryFile('c')]
    libraryMocks.list
      .mockResolvedValueOnce(page(firstPage, { total: 3, page: 1 }))
      .mockResolvedValueOnce(page(secondPage, { total: 3, page: 2 }))
    const store = useLibraryStore()

    await store.load()
    store.toggleSelection('a', true)
    await store.loadMore()

    expect(libraryMocks.list.mock.calls[1]?.[0]).toMatchObject({ page: 2 })
    expect(store.files.map(({ id }) => id)).toEqual(['a', 'b', 'c'])
    expect([...store.selectedFileIds]).toEqual(['a'])
    expect(store.page).toBe(2)
    expect(store.hasMore).toBe(false)
  })

  it('keeps the previous rows visible when a refresh fails', async () => {
    libraryMocks.list
      .mockResolvedValueOnce(page([libraryFile('existing')]))
      .mockRejectedValueOnce(new LibraryAPIError('网络暂时不可用', {
        code: 'NETWORK_ERROR',
      }))
    const store = useLibraryStore()

    await store.load()
    await store.load()

    expect(store.files.map(({ id }) => id)).toEqual(['existing'])
    expect(store.error).toBe('网络暂时不可用')
    expect(store.loading).toBe(false)
    expect(store.initialized).toBe(true)
  })

  it('tracks per-file upload progress and refreshes files and storage after success', async () => {
    const uploaded = libraryFile('uploaded', {
      name: 'uploaded.png',
      mimeType: 'image/png',
      type: 'image',
      category: 'image',
    })
    libraryMocks.upload.mockImplementationOnce(async (
      _file: File,
      options: { onProgress?: (progress: number) => void },
    ) => {
      options.onProgress?.(48)
      options.onProgress?.(100)
      return uploaded
    })
    libraryMocks.list.mockResolvedValueOnce(page([uploaded]))
    libraryMocks.storage.mockResolvedValueOnce({
      usedBytes: 2048,
      limitBytes: 8192,
      remainingBytes: 6144,
      overLimit: false,
      unlimited: false,
    })
    const store = useLibraryStore()

    await expect(store.uploadFiles([
      new File(['image'], 'uploaded.png', { type: 'image/png' }),
    ])).resolves.toEqual([uploaded])

    expect(store.uploads).toHaveLength(1)
    expect(store.uploads[0]).toMatchObject({ state: 'ready', progress: 100 })
    expect(store.uploadProgress[store.uploads[0]!.key]).toBe(100)
    expect(store.uploading).toBe(false)
    expect(store.files).toEqual([uploaded])
    expect(store.storageUsage).toEqual({
      usedBytes: 2048,
      limitBytes: 8192,
      remainingBytes: 6144,
      overLimit: false,
      unlimited: false,
    })
  })

  it('pre-registers every task and runs concurrent batches through one serial queue', async () => {
    const gates = new Map([
      ['first.pdf', deferred<LibraryFile>()],
      ['second.pdf', deferred<LibraryFile>()],
      ['third.pdf', deferred<LibraryFile>()],
    ])
    const progress = new Map<string, (value: number) => void>()
    const callOrder: string[] = []
    libraryMocks.upload.mockImplementation((
      file: File,
      options: { onProgress?: (value: number) => void },
    ) => {
      callOrder.push(file.name)
      if (options.onProgress) progress.set(file.name, options.onProgress)
      return gates.get(file.name)!.promise
    })
    const store = useLibraryStore()
    const firstFile = new File(['first'], 'first.pdf', { type: 'application/pdf' })
    const secondFile = new File(['second'], 'second.pdf', { type: 'application/pdf' })
    const thirdFile = new File(['third'], 'third.pdf', { type: 'application/pdf' })

    const firstBatch = store.uploadFiles([firstFile, secondFile])
    const secondBatch = store.uploadFiles([thirdFile])

    expect(store.uploads.map(({ state }) => state)).toEqual(['queued', 'queued', 'queued'])
    expect(store.uploading).toBe(true)
    expect(libraryMocks.upload).not.toHaveBeenCalled()

    await vi.waitFor(() => expect(libraryMocks.upload).toHaveBeenCalledTimes(1))
    expect(callOrder).toEqual(['first.pdf'])
    expect(store.uploads.map(({ state }) => state)).toEqual(['uploading', 'queued', 'queued'])

    progress.get('first.pdf')?.(100)
    expect(store.uploads[0]).toMatchObject({ state: 'processing', progress: 100 })
    expect(libraryMocks.upload).toHaveBeenCalledTimes(1)

    gates.get('first.pdf')!.resolve(libraryFile('first'))
    await vi.waitFor(() => expect(libraryMocks.upload).toHaveBeenCalledTimes(2))
    expect(callOrder).toEqual(['first.pdf', 'second.pdf'])
    expect(store.uploads.map(({ state }) => state)).toEqual(['ready', 'uploading', 'queued'])

    gates.get('second.pdf')!.resolve(libraryFile('second'))
    await vi.waitFor(() => expect(libraryMocks.upload).toHaveBeenCalledTimes(3))
    expect(callOrder).toEqual(['first.pdf', 'second.pdf', 'third.pdf'])
    expect(store.uploads.map(({ state }) => state)).toEqual(['ready', 'ready', 'uploading'])

    gates.get('third.pdf')!.resolve(libraryFile('third'))
    await expect(firstBatch).resolves.toEqual([libraryFile('first'), libraryFile('second')])
    await expect(secondBatch).resolves.toEqual([libraryFile('third')])
    expect(store.uploads.map(({ state }) => state)).toEqual(['ready', 'ready', 'ready'])
    expect(store.uploading).toBe(false)
  })

  it('dismisses only settled rows without cancelling active or queued work', async () => {
    const first = deferred<LibraryFile>()
    const second = deferred<LibraryFile>()
    libraryMocks.upload
      .mockReturnValueOnce(first.promise)
      .mockReturnValueOnce(second.promise)
    const store = useLibraryStore()
    const batch = store.uploadFiles([
      new File(['first'], 'first.pdf', { type: 'application/pdf' }),
      new File(['second'], 'second.pdf', { type: 'application/pdf' }),
    ])

    await vi.waitFor(() => expect(libraryMocks.upload).toHaveBeenCalledOnce())
    const activeSignal = libraryMocks.upload.mock.calls[0]?.[1].signal as AbortSignal
    store.uploads = [
      ...store.uploads,
      {
        key: 'already-ready',
        file: new File(['ready'], 'ready.pdf', { type: 'application/pdf' }),
        progress: 100,
        state: 'ready',
      },
      {
        key: 'already-failed',
        file: new File(['failed'], 'failed.pdf', { type: 'application/pdf' }),
        progress: 0,
        state: 'error',
        errorMessage: 'failed',
      },
    ]

    store.dismissSettledUploads()

    expect(activeSignal.aborted).toBe(false)
    expect(store.uploads.map(({ file }) => file.name)).toEqual(['first.pdf', 'second.pdf'])
    expect(store.uploads.map(({ state }) => state)).toEqual(['uploading', 'queued'])

    first.resolve(libraryFile('first'))
    await vi.waitFor(() => expect(libraryMocks.upload).toHaveBeenCalledTimes(2))
    second.resolve(libraryFile('second'))
    await expect(batch).resolves.toEqual([libraryFile('first'), libraryFile('second')])
  })

  it('aborts the active upload and releases queued callers when reset', async () => {
    libraryMocks.upload.mockImplementation((
      _file: File,
      options: { signal: AbortSignal },
    ) => new Promise<LibraryFile>((_resolve, reject) => {
      options.signal.addEventListener('abort', () => {
        reject(new DOMException('The operation was aborted.', 'AbortError'))
      }, { once: true })
    }))
    const store = useLibraryStore()
    const pending = store.uploadFiles([
      new File(['first'], 'first.pdf', { type: 'application/pdf' }),
      new File(['second'], 'second.pdf', { type: 'application/pdf' }),
    ])

    await vi.waitFor(() => expect(libraryMocks.upload).toHaveBeenCalledOnce())
    const activeSignal = libraryMocks.upload.mock.calls[0]?.[1].signal as AbortSignal
    store.reset()

    expect(activeSignal.aborted).toBe(true)
    expect(store.uploads).toEqual([])
    expect(store.uploading).toBe(false)
    await expect(pending).resolves.toEqual([])
    expect(libraryMocks.list).not.toHaveBeenCalled()
    expect(libraryMocks.storage).not.toHaveBeenCalled()
  })

  it('keeps a failed upload retryable and clears its error after retry', async () => {
    const uploaded = libraryFile('retried')
    libraryMocks.upload
      .mockRejectedValueOnce(new LibraryAPIError('文件过大', { code: 'FILE_TOO_LARGE' }))
      .mockResolvedValueOnce(uploaded)
    libraryMocks.list.mockResolvedValueOnce(page([uploaded]))
    const store = useLibraryStore()

    await expect(store.uploadFiles([
      new File(['document'], 'brief.pdf', { type: 'application/pdf' }),
    ])).resolves.toEqual([])
    expect(store.uploads[0]).toMatchObject({
      state: 'error',
      progress: 0,
      errorCode: 'FILE_TOO_LARGE',
      errorMessage: '文件过大',
    })

    const key = store.uploads[0]!.key
    await expect(store.retryUpload(key)).resolves.toEqual(uploaded)
    expect(store.uploads[0]).toMatchObject({ state: 'ready', progress: 100 })
    expect(store.uploads[0]?.errorCode).toBeUndefined()
    expect(store.uploads[0]?.errorMessage).toBeUndefined()
  })

  it('routes retries through the same serial queue as newly selected files', async () => {
    const active = deferred<LibraryFile>()
    const otherUploaded = libraryFile('other')
    const retried = libraryFile('retried')
    libraryMocks.upload
      .mockRejectedValueOnce(new LibraryAPIError('暂时失败', { code: 'NETWORK_ERROR' }))
      .mockReturnValueOnce(active.promise)
      .mockResolvedValueOnce(retried)
    const store = useLibraryStore()

    await store.uploadFiles([
      new File(['failed'], 'failed.pdf', { type: 'application/pdf' }),
    ])
    const failedKey = store.uploads[0]!.key
    const activeBatch = store.uploadFiles([
      new File(['other'], 'other.pdf', { type: 'application/pdf' }),
    ])
    await vi.waitFor(() => expect(libraryMocks.upload).toHaveBeenCalledTimes(2))

    const retry = store.retryUpload(failedKey)
    expect(store.uploads[0]).toMatchObject({ state: 'queued', progress: 0 })
    expect(libraryMocks.upload).toHaveBeenCalledTimes(2)

    active.resolve(otherUploaded)
    await vi.waitFor(() => expect(libraryMocks.upload).toHaveBeenCalledTimes(3))
    expect(libraryMocks.upload.mock.calls.map(([file]) => (file as File).name)).toEqual([
      'failed.pdf',
      'other.pdf',
      'failed.pdf',
    ])
    await expect(retry).resolves.toEqual(retried)
    await expect(activeBatch).resolves.toEqual([otherUploaded])
  })

  it('optimistically removes a batch and restores only failed deletions', async () => {
    const firstDelete = deferred<void>()
    const secondDelete = deferred<void>()
    libraryMocks.list.mockResolvedValueOnce(page([
      libraryFile('a'),
      libraryFile('b'),
    ]))
    libraryMocks.remove
      .mockReturnValueOnce(firstDelete.promise)
      .mockReturnValueOnce(secondDelete.promise)
    const store = useLibraryStore()
    await store.load()
    store.toggleSelection('a', true)
    store.toggleSelection('b', true)

    const removal = store.removeFiles(['a', 'b', 'a'])
    expect(store.files).toEqual([])
    expect(store.total).toBe(0)
    expect([...store.selectedFileIds]).toEqual([])

    firstDelete.resolve()
    await vi.waitFor(() => expect(libraryMocks.remove).toHaveBeenCalledTimes(2))
    secondDelete.reject(new Error('delete failed'))

    await expect(removal).resolves.toEqual({ deleted: ['a'], failed: ['b'] })
    expect(libraryMocks.remove.mock.calls.map(([id]) => id)).toEqual(['a', 'b'])
    expect(store.files.map(({ id }) => id)).toEqual(['b'])
    expect(store.total).toBe(1)
    await vi.waitFor(() => expect(libraryMocks.storage).toHaveBeenCalledOnce())
  })
})

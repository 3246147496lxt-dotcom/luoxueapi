import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import {
  LibraryAPIError,
  deleteLibraryFile,
  getLibraryStorage,
  isLibraryAbortError,
  listLibraryFiles,
  uploadLibraryFile,
} from '@/api/library'
import type {
  LibraryCategory,
  LibraryFile,
  LibraryFileType,
  LibrarySort,
  LibrarySource,
  LibraryStorageUsage,
  LibraryUploadItem,
  LibraryViewMode,
} from '@/types/library'

const DEFAULT_PAGE_SIZE = 40
const VIEW_STORAGE_KEY = 'luoxue-library-view'

function initialView(): LibraryViewMode {
  try {
    return localStorage.getItem(VIEW_STORAGE_KEY) === 'grid' ? 'grid' : 'list'
  } catch {
    return 'list'
  }
}

function localKey(prefix: string): string {
  try {
    if (typeof crypto?.randomUUID === 'function') return `${prefix}-${crypto.randomUUID()}`
  } catch {
    // The fallback only identifies an in-memory upload row.
  }
  return `${prefix}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`
}

interface UploadCompletion {
  promise: Promise<LibraryFile | null>
  resolve: (result: LibraryFile | null) => void
}

function createUploadCompletion(): UploadCompletion {
  let resolve!: (result: LibraryFile | null) => void
  const promise = new Promise<LibraryFile | null>((promiseResolve) => {
    resolve = promiseResolve
  })
  return { promise, resolve }
}

export function libraryErrorMessage(error: unknown): string {
  if (error instanceof LibraryAPIError) return error.message
  if (error instanceof Error && error.message.trim()) return error.message
  return '资料库请求失败，请稍后重试。'
}

export const useLibraryStore = defineStore('library', () => {
  const files = ref<LibraryFile[]>([])
  const total = ref(0)
  const loading = ref(false)
  const initialized = ref(false)
  const error = ref('')
  const searchKeyword = ref('')
  const category = ref<LibraryCategory>('all')
  const source = ref<LibrarySource>('all')
  const fileType = ref<LibraryFileType>('all')
  const sort = ref<LibrarySort>('updated_desc')
  const currentView = ref<LibraryViewMode>(initialView())
  const selectedFileIds = ref<Set<string>>(new Set())
  const page = ref(1)
  const pageSize = ref(DEFAULT_PAGE_SIZE)
  const uploads = ref<LibraryUploadItem[]>([])
  const storageUsage = ref<LibraryStorageUsage | null>(null)
  const storageLoading = ref(false)

  let listController: AbortController | null = null
  let listRequestSequence = 0
  let storageController: AbortController | null = null
  const uploadControllers = new Map<string, AbortController>()
  const uploadCompletions = new Map<string, UploadCompletion>()
  const uploadQueue: string[] = []
  let uploadQueueRunner: Promise<void> | null = null
  let uploadQueueScheduled = false

  const uploading = computed(() => uploads.value.some((item) => (
    item.state === 'queued' || item.state === 'uploading' || item.state === 'processing'
  )))
  const uploadProgress = computed<Record<string, number>>(() => Object.fromEntries(
    uploads.value.map(({ key, progress }) => [key, progress]),
  ))
  const selectedFiles = computed(() => files.value.filter(({ id }) => (
    selectedFileIds.value.has(id)
  )))
  const hasMore = computed(() => files.value.length < total.value)

  function query(nextPage = page.value) {
    return {
      q: searchKeyword.value,
      category: category.value,
      source: source.value,
      type: fileType.value,
      sort: sort.value,
      page: nextPage,
      pageSize: pageSize.value,
    }
  }

  async function load(options: { append?: boolean } = {}): Promise<void> {
    const append = options.append === true
    const nextPage = append ? page.value + 1 : 1
    listController?.abort()
    const controller = new AbortController()
    const sequence = ++listRequestSequence
    listController = controller
    loading.value = true
    error.value = ''
    try {
      const result = await listLibraryFiles(query(nextPage), controller.signal)
      if (sequence !== listRequestSequence || controller.signal.aborted) return
      files.value = append
        ? [
            ...files.value,
            ...result.files.filter((file) => !files.value.some(({ id }) => id === file.id)),
          ]
        : result.files
      total.value = result.total
      page.value = result.page
      const visibleIds = new Set(files.value.map(({ id }) => id))
      selectedFileIds.value = new Set(
        [...selectedFileIds.value].filter((id) => visibleIds.has(id)),
      )
      initialized.value = true
    } catch (requestError) {
      if (isLibraryAbortError(requestError) || sequence !== listRequestSequence) return
      error.value = libraryErrorMessage(requestError)
      initialized.value = true
    } finally {
      if (sequence === listRequestSequence) {
        loading.value = false
        if (listController === controller) listController = null
      }
    }
  }

  async function loadMore(): Promise<void> {
    if (loading.value || !hasMore.value) return
    await load({ append: true })
  }

  function setSearchKeyword(value: string): void {
    searchKeyword.value = value
    page.value = 1
  }

  function setCategory(value: LibraryCategory): void {
    category.value = value
    if (value === 'image' || (value === 'file' && fileType.value === 'image')) {
      fileType.value = 'all'
    }
    page.value = 1
    clearSelection()
  }

  function setFilters(filters: { source: LibrarySource; type: LibraryFileType }): void {
    source.value = filters.source
    fileType.value = filters.type
    if (filters.type === 'image') category.value = 'image'
    else if (category.value === 'image' && filters.type !== 'all') category.value = 'all'
    page.value = 1
    clearSelection()
  }

  function setSort(value: LibrarySort): void {
    sort.value = value
    page.value = 1
  }

  function setView(value: LibraryViewMode): void {
    currentView.value = value
    try {
      localStorage.setItem(VIEW_STORAGE_KEY, value)
    } catch {
      // View persistence is optional.
    }
  }

  function toggleSelection(id: string, selected?: boolean): void {
    const next = new Set(selectedFileIds.value)
    const shouldSelect = selected ?? !next.has(id)
    if (shouldSelect) next.add(id)
    else next.delete(id)
    selectedFileIds.value = next
  }

  function selectOnly(id: string): void {
    selectedFileIds.value = new Set([id])
  }

  function clearSelection(): void {
    selectedFileIds.value = new Set()
  }

  function replaceUpload(key: string, patch: Partial<LibraryUploadItem>): void {
    const index = uploads.value.findIndex((item) => item.key === key)
    if (index < 0) return
    uploads.value[index] = { ...uploads.value[index]!, ...patch }
    uploads.value = [...uploads.value]
  }

  function settleUpload(key: string, result: LibraryFile | null): void {
    const completion = uploadCompletions.get(key)
    if (!completion) return
    uploadCompletions.delete(key)
    completion.resolve(result)
  }

  function queueUpload(key: string): Promise<LibraryFile | null> {
    const completion = createUploadCompletion()
    uploadCompletions.set(key, completion)
    uploadQueue.push(key)
    return completion.promise
  }

  function updateUploadProgress(key: string, controller: AbortController, progress: number): void {
    if (controller.signal.aborted || uploadControllers.get(key) !== controller) return
    const item = uploads.value.find((candidate) => candidate.key === key)
    if (!item) return
    const normalizedProgress = Math.max(
      item.progress,
      Math.max(0, Math.min(100, Math.round(progress))),
    )
    replaceUpload(key, {
      progress: normalizedProgress,
      state: normalizedProgress >= 100 ? 'processing' : 'uploading',
    })
  }

  async function executeQueuedUpload(key: string): Promise<void> {
    const item = uploads.value.find((candidate) => candidate.key === key)
    if (!item || item.state !== 'queued' || !uploadCompletions.has(key)) {
      settleUpload(key, null)
      return
    }
    const controller = new AbortController()
    uploadControllers.set(key, controller)
    replaceUpload(key, {
      state: 'uploading',
      progress: 0,
      errorCode: undefined,
      errorMessage: undefined,
    })
    try {
      const uploaded = await uploadLibraryFile(item.file, {
        signal: controller.signal,
        onProgress(progress) {
          updateUploadProgress(key, controller, progress)
        },
      })
      if (
        controller.signal.aborted
        || uploadControllers.get(key) !== controller
        || !uploads.value.some((candidate) => candidate.key === key)
      ) {
        settleUpload(key, null)
        return
      }
      replaceUpload(key, { progress: 100, state: 'ready' })
      settleUpload(key, uploaded)
    } catch (uploadError) {
      if (isLibraryAbortError(uploadError) || controller.signal.aborted) {
        settleUpload(key, null)
        return
      }
      const normalized = uploadError instanceof LibraryAPIError ? uploadError : null
      replaceUpload(key, {
        progress: 0,
        state: 'error',
        errorCode: normalized?.code ?? 'LIBRARY_UPLOAD_FAILED',
        errorMessage: libraryErrorMessage(uploadError),
      })
      settleUpload(key, null)
    } finally {
      if (uploadControllers.get(key) === controller) uploadControllers.delete(key)
    }
  }

  function finishUploadQueue(runner: Promise<void>): void {
    if (uploadQueueRunner !== runner) return
    uploadQueueRunner = null
    scheduleUploadQueue()
  }

  function startUploadQueue(): void {
    if (uploadQueueRunner || uploadQueue.length === 0) return
    const runner = (async () => {
      while (uploadQueue.length > 0) {
        const key = uploadQueue.shift()
        if (!key || !uploadCompletions.has(key)) continue
        await executeQueuedUpload(key)
      }
    })()
    uploadQueueRunner = runner
    void runner.then(
      () => finishUploadQueue(runner),
      () => finishUploadQueue(runner),
    )
  }

  function scheduleUploadQueue(): void {
    if (uploadQueueRunner || uploadQueueScheduled || uploadQueue.length === 0) return
    uploadQueueScheduled = true
    void Promise.resolve().then(() => {
      uploadQueueScheduled = false
      startUploadQueue()
    })
  }

  async function uploadFiles(input: File[]): Promise<LibraryFile[]> {
    if (input.length === 0) return []
    const items = input.map<LibraryUploadItem>((file) => ({
      key: localKey('library-upload'),
      file,
      progress: 0,
      state: 'queued',
    }))
    uploads.value = [...uploads.value, ...items]
    const results = items.map(({ key }) => queueUpload(key))
    scheduleUploadQueue()
    const uploaded = (await Promise.all(results)).filter(
      (file): file is LibraryFile => file !== null,
    )
    if (uploaded.length > 0) {
      await Promise.all([load(), fetchStorage()])
    }
    return uploaded
  }

  async function retryUpload(key: string): Promise<LibraryFile | null> {
    const item = uploads.value.find((candidate) => candidate.key === key)
    if (!item || item.state !== 'error') return null
    replaceUpload(key, {
      state: 'queued',
      progress: 0,
      errorCode: undefined,
      errorMessage: undefined,
    })
    const completion = queueUpload(key)
    scheduleUploadQueue()
    const uploaded = await completion
    if (uploaded) {
      await Promise.all([load(), fetchStorage()])
    }
    return uploaded
  }

  function dismissUpload(key: string): void {
    uploadControllers.get(key)?.abort()
    uploadControllers.delete(key)
    for (let index = uploadQueue.length - 1; index >= 0; index -= 1) {
      if (uploadQueue[index] === key) uploadQueue.splice(index, 1)
    }
    settleUpload(key, null)
    uploads.value = uploads.value.filter((item) => item.key !== key)
  }

  function dismissSettledUploads(): void {
    uploads.value = uploads.value.filter((item) => (
      item.state !== 'ready' && item.state !== 'error'
    ))
  }

  async function removeFiles(ids: string[]): Promise<{ deleted: string[]; failed: string[] }> {
    const requested = [...new Set(ids.filter(Boolean))]
    const removed = files.value.filter(({ id }) => requested.includes(id))
    if (removed.length === 0) return { deleted: [], failed: [] }
    const removedIndex = new Map(removed.map((file) => [file.id, files.value.indexOf(file)]))
    files.value = files.value.filter(({ id }) => !requested.includes(id))
    total.value = Math.max(0, total.value - removed.length)
    selectedFileIds.value = new Set(
      [...selectedFileIds.value].filter((id) => !requested.includes(id)),
    )

    const deleted: string[] = []
    const failed: string[] = []
    for (const file of removed) {
      try {
        await deleteLibraryFile(file.id)
        deleted.push(file.id)
      } catch {
        failed.push(file.id)
      }
    }
    if (failed.length > 0) {
      const restored = removed.filter(({ id }) => failed.includes(id))
      const next = [...files.value]
      for (const file of restored) {
        const index = Math.min(removedIndex.get(file.id) ?? next.length, next.length)
        next.splice(index, 0, file)
      }
      files.value = next
      total.value += restored.length
    }
    if (deleted.length > 0) void fetchStorage()
    return { deleted, failed }
  }

  async function fetchStorage(): Promise<void> {
    storageController?.abort()
    const controller = new AbortController()
    storageController = controller
    storageLoading.value = true
    try {
      storageUsage.value = await getLibraryStorage(controller.signal)
    } catch (storageError) {
      if (!isLibraryAbortError(storageError)) storageUsage.value = null
    } finally {
      if (storageController === controller) {
        storageController = null
        storageLoading.value = false
      }
    }
  }

  function reset(): void {
    listController?.abort()
    storageController?.abort()
    uploadControllers.forEach((controller) => controller.abort())
    uploadControllers.clear()
    uploadQueue.splice(0, uploadQueue.length)
    const pendingUploadCompletions = [...uploadCompletions.values()]
    uploadCompletions.clear()
    pendingUploadCompletions.forEach(({ resolve }) => resolve(null))
    listRequestSequence += 1
    files.value = []
    total.value = 0
    loading.value = false
    initialized.value = false
    error.value = ''
    searchKeyword.value = ''
    category.value = 'all'
    source.value = 'all'
    fileType.value = 'all'
    sort.value = 'updated_desc'
    selectedFileIds.value = new Set()
    page.value = 1
    uploads.value = []
    storageUsage.value = null
  }

  return {
    files,
    total,
    loading,
    initialized,
    error,
    searchKeyword,
    category,
    source,
    fileType,
    sort,
    currentView,
    selectedFileIds,
    selectedFiles,
    page,
    pageSize,
    hasMore,
    uploads,
    uploading,
    uploadProgress,
    storageUsage,
    storageLoading,
    load,
    loadMore,
    setSearchKeyword,
    setCategory,
    setFilters,
    setSort,
    setView,
    toggleSelection,
    selectOnly,
    clearSelection,
    uploadFiles,
    retryUpload,
    dismissUpload,
    dismissSettledUploads,
    removeFiles,
    fetchStorage,
    reset,
  }
})

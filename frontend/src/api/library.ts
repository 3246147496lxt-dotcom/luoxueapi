import axios, { type AxiosProgressEvent } from 'axios'
import { apiClient } from './client'
import type {
  LibraryFile,
  LibraryFilePage,
  LibraryFileQuery,
  LibraryFileSource,
  LibraryFileType,
  LibraryDownload,
  LibraryDownloadPath,
  LibraryDownloadTicket,
  LibraryStorageUsage,
  LibraryUploadOptions,
} from '@/types/library'

export class LibraryAPIError extends Error {
  readonly status: number
  readonly code: string
  readonly reason: string
  readonly metadata: unknown

  constructor(
    message: string,
    options: { status?: number; code?: string; reason?: string; metadata?: unknown } = {},
  ) {
    super(message)
    this.name = 'LibraryAPIError'
    this.status = options.status ?? 0
    this.reason = options.reason?.trim() ?? ''
    this.code = options.code?.trim() || this.reason || 'LIBRARY_REQUEST_FAILED'
    this.metadata = options.metadata
  }
}

function record(value: unknown): Record<string, unknown> | null {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
    ? value as Record<string, unknown>
    : null
}

function text(value: unknown): string {
  return typeof value === 'string' ? value.trim() : ''
}

function finiteNonNegative(value: unknown, fallback = 0): number {
  const parsed = typeof value === 'number' ? value : Number(value)
  return Number.isFinite(parsed) && parsed >= 0 ? parsed : fallback
}

function positiveInteger(value: unknown): number | undefined {
  const parsed = finiteNonNegative(value)
  return Number.isInteger(parsed) && parsed > 0 ? parsed : undefined
}

function boolean(value: unknown, fallback = false): boolean {
  return typeof value === 'boolean' ? value : fallback
}

function unwrappedPayload(value: unknown): unknown {
  const candidate = record(value)
  if (!candidate || !('code' in candidate)) return value
  if (candidate.code === 0) return candidate.data
  const reason = text(candidate.reason)
  throw new LibraryAPIError(text(candidate.message) || 'The library request failed.', {
    status: finiteNonNegative(candidate.code),
    code: reason || text(candidate.code) || 'LIBRARY_REQUEST_FAILED',
    reason,
    metadata: candidate.metadata,
  })
}

function inferLibraryFileType(mimeType: string, name: string): Exclude<LibraryFileType, 'all'> {
  const mime = mimeType.toLowerCase()
  const extension = name.split('.').pop()?.toLowerCase() ?? ''
  if (mime.startsWith('image/')) return 'image'
  if (mime === 'application/pdf' || extension === 'pdf') return 'pdf'
  if (
    mime.includes('spreadsheet')
    || mime.includes('excel')
    || ['xls', 'xlsx', 'csv'].includes(extension)
  ) return 'spreadsheet'
  if (
    mime.includes('presentation')
    || mime.includes('powerpoint')
    || ['ppt', 'pptx'].includes(extension)
  ) return 'presentation'
  if (
    mime.startsWith('text/')
    || mime.includes('word')
    || ['doc', 'docx', 'txt', 'md', 'rtf'].includes(extension)
  ) return 'document'
  return 'other'
}

export function normalizeLibraryFile(value: unknown): LibraryFile | null {
  const candidate = record(value)
  if (!candidate) return null
  const id = text(candidate.id ?? candidate.file_id ?? candidate.fileId)
  const name = text(candidate.name ?? candidate.original_name ?? candidate.originalName)
  const mimeType = text(candidate.mime_type ?? candidate.mimeType) || 'application/octet-stream'
  const size = finiteNonNegative(candidate.size ?? candidate.byte_size ?? candidate.byteSize, -1)
  if (!id || !name || size < 0) return null

  const rawSource = text(candidate.source).toLowerCase()
  const source: LibraryFileSource = rawSource === 'generated' ? 'generated' : 'uploaded'
  const inferredType = inferLibraryFileType(mimeType, name)
  const rawType = text(candidate.type ?? candidate.file_type ?? candidate.fileType).toLowerCase()
  const allowedTypes = new Set(['image', 'pdf', 'document', 'spreadsheet', 'presentation', 'other'])
  const type = (allowedTypes.has(rawType) ? rawType : inferredType) as Exclude<LibraryFileType, 'all'>
  const createdAt = text(candidate.created_at ?? candidate.createdAt)
  const updatedAt = text(candidate.updated_at ?? candidate.updatedAt) || createdAt
  const extension = text(candidate.extension) || name.split('.').pop()?.toLowerCase() || ''
  const previewable = boolean(
    candidate.previewable,
    type === 'image' || type === 'pdf',
  )
  const file: LibraryFile = {
    id,
    name,
    mimeType,
    extension,
    size,
    source,
    type,
    category: type === 'image' ? 'image' : 'file',
    status: 'ready',
    previewable,
    createdAt,
    updatedAt,
  }
  const width = positiveInteger(candidate.width)
  const height = positiveInteger(candidate.height)
  const pageCount = positiveInteger(candidate.page_count ?? candidate.pageCount)
  if (width) file.width = width
  if (height) file.height = height
  if (pageCount) file.pageCount = pageCount
  return file
}

function normalizeError(error: unknown, fallback: string): LibraryAPIError {
  if (error instanceof LibraryAPIError) return error
  if (axios.isAxiosError(error)) {
    const payload = record(error.response?.data)
    const nested = record(payload?.error)
    const reason = text(nested?.reason ?? payload?.reason)
    const message = text(nested?.message ?? payload?.message) || error.message || fallback
    const code = text(nested?.code ?? payload?.code) || reason || 'LIBRARY_REQUEST_FAILED'
    return new LibraryAPIError(message, {
      status: error.response?.status,
      code,
      reason,
      metadata: nested?.metadata ?? payload?.metadata,
    })
  }
  const candidate = record(error)
  const reason = text(candidate?.reason)
  return new LibraryAPIError(text(candidate?.message) || fallback, {
    status: finiteNonNegative(candidate?.status),
    code: text(candidate?.code) || reason || 'LIBRARY_REQUEST_FAILED',
    reason,
    metadata: candidate?.metadata,
  })
}

function responseHeader(headers: unknown, name: string): string {
  if (!headers || typeof headers !== 'object') return ''
  const candidate = headers as {
    get?: (headerName: string) => unknown
    [key: string]: unknown
  }
  let raw: unknown
  try {
    raw = typeof candidate.get === 'function'
      ? candidate.get(name)
      : candidate[name] ?? candidate[name.toLowerCase()]
  } catch {
    return ''
  }
  return typeof raw === 'string' ? raw : ''
}

export function libraryDownloadFilename(contentDisposition: string, fallback: string): string {
  const safeFallback = text(fallback) || 'download'
  const extended = /(?:^|;)\s*filename\*\s*=\s*UTF-8''([^;]+)/i.exec(contentDisposition)?.[1]
  if (extended) {
    try {
      return decodeURIComponent(extended.trim().replace(/^"|"$/g, '')) || safeFallback
    } catch {
      // Fall through to the plain filename parameter.
    }
  }
  const quoted = /(?:^|;)\s*filename\s*=\s*"((?:\\.|[^"])*)"/i.exec(contentDisposition)?.[1]
  if (quoted) return quoted.replace(/\\(.)/g, '$1').trim() || safeFallback
  const plain = /(?:^|;)\s*filename\s*=\s*([^;]+)/i.exec(contentDisposition)?.[1]
  return plain?.trim() || safeFallback
}

function throwIfAborted(signal?: AbortSignal): void {
  if (!signal?.aborted) return
  throw new DOMException('The operation was aborted.', 'AbortError')
}

export function isLibraryAbortError(error: unknown): boolean {
  return (error instanceof DOMException && error.name === 'AbortError')
    || (axios.isAxiosError(error) && error.code === 'ERR_CANCELED')
}

export async function listLibraryFiles(
  query: LibraryFileQuery,
  signal?: AbortSignal,
): Promise<LibraryFilePage> {
  throwIfAborted(signal)
  try {
    const { data } = await apiClient.get<unknown>('/library/files', {
      signal,
      params: {
        ...(query.q?.trim() ? { q: query.q.trim() } : {}),
        category: query.category,
        source: query.source,
        type: query.type,
        sort: query.sort,
        page: query.page,
        pageSize: query.pageSize,
      },
    })
    throwIfAborted(signal)
    const payload = record(unwrappedPayload(data))
    if (!payload) {
      throw new LibraryAPIError('The file list response was invalid.', {
        code: 'INVALID_FILE_LIST_RESPONSE',
      })
    }
    const rawFiles = Array.isArray(payload.files)
      ? payload.files
      : Array.isArray(payload.items)
        ? payload.items
        : []
    return {
      files: rawFiles.map(normalizeLibraryFile).filter((file): file is LibraryFile => file !== null),
      total: finiteNonNegative(payload.total, rawFiles.length),
      page: Math.max(1, finiteNonNegative(payload.page, query.page)),
      pageSize: Math.max(1, finiteNonNegative(
        payload.page_size ?? payload.pageSize,
        query.pageSize,
      )),
    }
  } catch (error) {
    if (isLibraryAbortError(error)) throw error
    throw normalizeError(error, 'Unable to load library files.')
  }
}

export async function getLibraryFile(id: string, signal?: AbortSignal): Promise<LibraryFile> {
  const normalizedId = id.trim()
  if (!normalizedId) throw new LibraryAPIError('A file ID is required.', { code: 'FILE_ID_REQUIRED' })
  try {
    const { data } = await apiClient.get<unknown>(
      `/library/files/${encodeURIComponent(normalizedId)}`,
      { signal },
    )
    const responsePayload = unwrappedPayload(data)
    const payload = record(responsePayload)?.file ?? responsePayload
    const file = normalizeLibraryFile(payload)
    if (!file) throw new LibraryAPIError('The file response was invalid.', { code: 'INVALID_FILE_RESPONSE' })
    return file
  } catch (error) {
    if (isLibraryAbortError(error)) throw error
    throw normalizeError(error, 'Unable to load the file.')
  }
}

export async function uploadLibraryFile(
  file: File,
  options: LibraryUploadOptions = {},
): Promise<LibraryFile> {
  if (!(file instanceof File) || file.size <= 0) {
    throw new LibraryAPIError('A non-empty file is required.', { code: 'EMPTY_FILE' })
  }
  const body = new FormData()
  body.append('file', file, file.name)
  try {
    const { data } = await apiClient.post<unknown>('/library/files/upload', body, {
      signal: options.signal,
      timeout: 0,
      headers: { 'Content-Type': undefined },
      onUploadProgress: (event: AxiosProgressEvent) => {
        const total = event.total && event.total > 0 ? event.total : file.size
        const progress = total > 0 ? Math.round((event.loaded / total) * 100) : 0
        options.onProgress?.(Math.max(0, Math.min(100, progress)))
      },
    })
    const responsePayload = unwrappedPayload(data)
    const payload = record(responsePayload)?.file ?? responsePayload
    const uploaded = normalizeLibraryFile(payload)
    if (!uploaded) {
      throw new LibraryAPIError('The upload response was invalid.', {
        code: 'INVALID_UPLOAD_RESPONSE',
      })
    }
    return uploaded
  } catch (error) {
    if (isLibraryAbortError(error)) throw error
    throw normalizeError(error, 'Unable to upload the file.')
  }
}

async function getLibraryBlob(
  id: string,
  endpoint: 'thumbnail' | 'preview' | 'download',
  signal?: AbortSignal,
  fallbackFilename?: string,
): Promise<LibraryDownload> {
  const normalizedId = id.trim()
  if (!normalizedId) throw new LibraryAPIError('A file ID is required.', { code: 'FILE_ID_REQUIRED' })
  try {
    const { data, headers } = await apiClient.get<Blob>(
      `/library/files/${encodeURIComponent(normalizedId)}/${endpoint}`,
      { signal, responseType: 'blob' },
    )
    if (!(data instanceof Blob)) {
      throw new LibraryAPIError('The file content response was invalid.', {
        code: 'INVALID_FILE_CONTENT',
      })
    }
    return {
      blob: data,
      filename: libraryDownloadFilename(
        responseHeader(headers, 'content-disposition'),
        fallbackFilename || (endpoint === 'download' ? 'download' : endpoint),
      ),
    }
  } catch (error) {
    if (isLibraryAbortError(error)) throw error
    throw normalizeError(error, 'Unable to load the file content.')
  }
}

export const getLibraryThumbnail = (id: string, signal?: AbortSignal) => (
  getLibraryBlob(id, 'thumbnail', signal).then(({ blob }) => blob)
)

export const getLibraryPreview = (id: string, signal?: AbortSignal) => (
  getLibraryBlob(id, 'preview', signal).then(({ blob }) => blob)
)

export const downloadLibraryFile = (
  id: string,
  signal?: AbortSignal,
  fallbackFilename?: string,
) => (
  getLibraryBlob(id, 'download', signal, fallbackFilename)
)

export async function deleteLibraryFile(id: string, signal?: AbortSignal): Promise<void> {
  const normalizedId = id.trim()
  if (!normalizedId) throw new LibraryAPIError('A file ID is required.', { code: 'FILE_ID_REQUIRED' })
  try {
    await apiClient.delete(`/library/files/${encodeURIComponent(normalizedId)}`, { signal })
  } catch (error) {
    if (isLibraryAbortError(error)) throw error
    throw normalizeError(error, 'Unable to delete the file.')
  }
}

export async function batchDownloadLibraryFiles(
  fileIds: string[],
  signal?: AbortSignal,
): Promise<LibraryDownload> {
  const ids = [...new Set(fileIds.map((id) => id.trim()).filter(Boolean))]
  if (ids.length === 0) {
    throw new LibraryAPIError('Select at least one file.', { code: 'FILE_IDS_REQUIRED' })
  }
  try {
    const { data, headers } = await apiClient.post<Blob>(
      '/library/files/batch-download',
      { fileIds: ids },
      { signal, responseType: 'blob' },
    )
    if (!(data instanceof Blob)) {
      throw new LibraryAPIError('The archive response was invalid.', {
        code: 'INVALID_ARCHIVE_RESPONSE',
      })
    }
    return {
      blob: data,
      filename: libraryDownloadFilename(
        responseHeader(headers, 'content-disposition'),
        ids.length === 1 ? 'download' : 'library-files.zip',
      ),
    }
  } catch (error) {
    if (isLibraryAbortError(error)) throw error
    throw normalizeError(error, 'Unable to download the selected files.')
  }
}

const libraryDownloadPathPattern = /^\/api\/v1\/library\/download\/[A-Za-z0-9_-]{22}$/

function normalizeLibraryDownloadPath(value: unknown): LibraryDownloadPath | null {
  const candidate = text(value)
  if (!libraryDownloadPathPattern.test(candidate)) return null

  // The exact anchored pattern already excludes these, but keep the URL checks
  // explicit so future path-contract edits cannot silently admit URL metadata.
  const parsed = new URL(candidate, 'https://library-download.invalid')
  if (parsed.search || parsed.hash || parsed.pathname !== candidate) return null
  return candidate as LibraryDownloadPath
}

/**
 * Exchanges the authenticated selection for safe metadata pointing at a
 * cookie-backed, one-time download. The JSON response never carries the
 * credential; only a fixed-shape, non-secret handle is exposed to JavaScript.
 */
export async function createLibraryDownloadTicket(
  fileIds: string[],
  signal?: AbortSignal,
): Promise<LibraryDownloadTicket> {
  const ids = [...new Set(fileIds.map((id) => id.trim()).filter(Boolean))]
  if (ids.length === 0) {
    throw new LibraryAPIError('Select at least one file.', { code: 'FILE_IDS_REQUIRED' })
  }
  try {
    const { data } = await apiClient.post<unknown>(
      '/library/files/download-ticket',
      { fileIds: ids },
      { signal },
    )
    const payload = record(unwrappedPayload(data))
    const downloadPath = normalizeLibraryDownloadPath(
      payload?.download_path ?? payload?.downloadPath,
    )
    const filename = text(payload?.filename)
    const fileCount = finiteNonNegative(payload?.file_count ?? payload?.fileCount)
    const totalBytes = finiteNonNegative(payload?.total_bytes ?? payload?.totalBytes)
    const expiresIn = finiteNonNegative(payload?.expires_in ?? payload?.expiresIn)
    if (
      !downloadPath
      || !filename
      || !Number.isInteger(fileCount)
      || fileCount <= 0
      || expiresIn <= 0
    ) {
      throw new LibraryAPIError('The download ticket response was invalid.', {
        code: 'INVALID_DOWNLOAD_TICKET_RESPONSE',
      })
    }
    return { downloadPath, filename, fileCount, totalBytes, expiresIn }
  } catch (error) {
    if (isLibraryAbortError(error)) throw error
    throw normalizeError(error, 'Unable to prepare the download.')
  }
}

export async function getLibraryStorage(signal?: AbortSignal): Promise<LibraryStorageUsage> {
  try {
    const { data } = await apiClient.get<unknown>('/library/storage', { signal })
    const payload = record(unwrappedPayload(data))
    if (!payload) {
      throw new LibraryAPIError('The storage response was invalid.', {
        code: 'INVALID_STORAGE_RESPONSE',
      })
    }
    const usedBytes = finiteNonNegative(payload.used_bytes ?? payload.usedBytes)
    const limitBytes = finiteNonNegative(payload.limit_bytes ?? payload.limitBytes)
    const calculatedRemaining = limitBytes > 0 ? Math.max(0, limitBytes - usedBytes) : 0
    return {
      usedBytes,
      limitBytes,
      remainingBytes: finiteNonNegative(
        payload.remaining_bytes ?? payload.remainingBytes,
        calculatedRemaining,
      ),
      overLimit: boolean(payload.over_limit ?? payload.overLimit, limitBytes > 0 && usedBytes > limitBytes),
      unlimited: limitBytes === 0,
    }
  } catch (error) {
    if (isLibraryAbortError(error)) throw error
    throw normalizeError(error, 'Unable to load storage usage.')
  }
}

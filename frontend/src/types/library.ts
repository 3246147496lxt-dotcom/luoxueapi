export type LibraryCategory = 'all' | 'image' | 'file'

export type LibrarySource = 'all' | 'uploaded' | 'generated'

export type LibraryFileSource = Exclude<LibrarySource, 'all'>

export type LibraryFileType =
  | 'all'
  | 'image'
  | 'pdf'
  | 'document'
  | 'spreadsheet'
  | 'presentation'
  | 'other'

export type LibrarySort =
  | 'updated_desc'
  | 'updated_asc'
  | 'name_asc'
  | 'name_desc'
  | 'size_asc'
  | 'size_desc'

export type LibraryViewMode = 'grid' | 'list'

export interface LibraryFile {
  id: string
  name: string
  mimeType: string
  extension?: string
  size: number
  source: LibraryFileSource
  type: Exclude<LibraryFileType, 'all'>
  category: Exclude<LibraryCategory, 'all'>
  status: 'ready'
  previewable?: boolean
  createdAt: string
  updatedAt: string
  width?: number
  height?: number
  pageCount?: number
}

export interface LibraryFileQuery {
  q?: string
  category: LibraryCategory
  source: LibrarySource
  type: LibraryFileType
  sort: LibrarySort
  page: number
  pageSize: number
}

export interface LibraryFilePage {
  files: LibraryFile[]
  total: number
  page: number
  pageSize: number
}

export interface LibraryStorageUsage {
  usedBytes: number
  limitBytes: number
  remainingBytes?: number
  overLimit?: boolean
  unlimited?: boolean
}

export interface LibraryDownload {
  blob: Blob
  filename: string
}

export type LibraryDownloadPath = `/api/v1/library/download/${string}`

/**
 * Safe metadata for a cookie-backed, one-time download. The path contains an
 * opaque public handle; the download credential remains in an HttpOnly cookie.
 */
export interface LibraryDownloadTicket {
  downloadPath: LibraryDownloadPath
  filename: string
  fileCount: number
  totalBytes: number
  expiresIn: number
}

export interface LibraryUploadOptions {
  signal?: AbortSignal
  onProgress?: (progress: number) => void
}

export type LibraryUploadState = 'queued' | 'uploading' | 'processing' | 'ready' | 'error'

export interface LibraryUploadItem {
  key: string
  file: File
  progress: number
  state: LibraryUploadState
  errorCode?: string
  errorMessage?: string
}

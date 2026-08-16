import { beforeEach, describe, expect, it, vi } from 'vitest'

const clientMocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  delete: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get: clientMocks.get,
    post: clientMocks.post,
    delete: clientMocks.delete,
  },
}))

import {
  LibraryAPIError,
  batchDownloadLibraryFiles,
  createLibraryDownloadTicket,
  deleteLibraryFile,
  downloadLibraryFile,
  getLibraryFile,
  getLibraryPreview,
  getLibraryStorage,
  getLibraryThumbnail,
  isLibraryAbortError,
  libraryDownloadFilename,
  listLibraryFiles,
  normalizeLibraryFile,
  uploadLibraryFile,
} from '@/api/library'
import type { LibraryFileQuery } from '@/types/library'

const defaultQuery: LibraryFileQuery = {
  q: '',
  category: 'all',
  source: 'all',
  type: 'all',
  sort: 'updated_desc',
  page: 1,
  pageSize: 40,
}

const downloadHandle = 'ABCdef0123456789_-abcd'
const downloadPath = `/api/v1/library/download/${downloadHandle}`

function imagePayload(overrides: Record<string, unknown> = {}) {
  return {
    id: 'image-1',
    name: 'snow.png',
    mime_type: 'image/png',
    size: 4096,
    source: 'generated',
    created_at: '2026-08-13T10:00:00Z',
    updated_at: '2026-08-14T10:00:00Z',
    width: 1280,
    height: 720,
    ...overrides,
  }
}

beforeEach(() => {
  vi.resetAllMocks()
})

describe('library API contract', () => {
  it('normalizes backend naming and infers a useful file type when metadata is absent', () => {
    expect(normalizeLibraryFile(imagePayload())).toEqual({
      id: 'image-1',
      name: 'snow.png',
      mimeType: 'image/png',
      extension: 'png',
      size: 4096,
      source: 'generated',
      type: 'image',
      category: 'image',
      status: 'ready',
      previewable: true,
      createdAt: '2026-08-13T10:00:00Z',
      updatedAt: '2026-08-14T10:00:00Z',
      width: 1280,
      height: 720,
    })

    expect(normalizeLibraryFile({
      fileId: 'sheet-1',
      originalName: 'budget.xlsx',
      byteSize: '5120',
      mimeType: 'application/octet-stream',
      source: 'unexpected-source',
      createdAt: '2026-08-12T08:00:00Z',
    })).toMatchObject({
      id: 'sheet-1',
      name: 'budget.xlsx',
      size: 5120,
      source: 'uploaded',
      type: 'spreadsheet',
      category: 'file',
      updatedAt: '2026-08-12T08:00:00Z',
    })

    expect(normalizeLibraryFile({ id: '', name: 'invalid.txt', size: 10 })).toBeNull()
    expect(normalizeLibraryFile({ id: 'invalid', name: '', size: 10 })).toBeNull()
    expect(normalizeLibraryFile({ id: 'invalid', name: 'bad.txt', size: -1 })).toBeNull()
  })

  it('consumes the unwrapped files page, trims search, and drops malformed rows', async () => {
    const controller = new AbortController()
    clientMocks.get.mockResolvedValueOnce({
      data: {
        files: [
          imagePayload(),
          { id: '', name: 'broken.pdf', size: 100 },
        ],
        total: '7',
        page: 2,
        page_size: 25,
      },
    })

    await expect(listLibraryFiles({
      q: '  snow  ',
      category: 'image',
      source: 'generated',
      type: 'image',
      sort: 'name_asc',
      page: 2,
      pageSize: 25,
    }, controller.signal)).resolves.toMatchObject({
      total: 7,
      page: 2,
      pageSize: 25,
      files: [{ id: 'image-1', type: 'image' }],
    })

    expect(clientMocks.get).toHaveBeenCalledWith('/library/files', {
      signal: controller.signal,
      params: {
        q: 'snow',
        category: 'image',
        source: 'generated',
        type: 'image',
        sort: 'name_asc',
        page: 2,
        pageSize: 25,
      },
    })
  })

  it('tolerates a raw Go envelope and the generic items page during mixed deployments', async () => {
    clientMocks.get.mockResolvedValueOnce({
      data: {
        code: 0,
        message: 'success',
        data: {
          items: [imagePayload()],
          total: 1,
          page: 1,
          page_size: 40,
        },
      },
    })

    await expect(listLibraryFiles(defaultQuery)).resolves.toMatchObject({
      files: [{ id: 'image-1' }],
      total: 1,
    })
  })

  it('rejects a raw non-success envelope with its Go business reason', async () => {
    clientMocks.get.mockResolvedValueOnce({
      data: {
        code: 507,
        message: 'Library storage limit exceeded',
        reason: 'LIBRARY_STORAGE_LIMIT_EXCEEDED',
        metadata: { limit_bytes: '4096' },
      },
    })

    await expect(listLibraryFiles(defaultQuery)).rejects.toMatchObject({
      status: 507,
      code: 'LIBRARY_STORAGE_LIMIT_EXCEEDED',
      reason: 'LIBRARY_STORAGE_LIMIT_EXCEEDED',
      metadata: { limit_bytes: '4096' },
    })
  })

  it('accepts a file-wrapped detail payload during a mixed deployment', async () => {
    clientMocks.get.mockResolvedValueOnce({
      data: {
        code: 0,
        message: 'success',
        data: { file: imagePayload() },
      },
    })

    await expect(getLibraryFile('image-1')).resolves.toMatchObject({
      id: 'image-1',
      name: 'snow.png',
    })
  })

  it('fails fast for an already-aborted list request', async () => {
    const controller = new AbortController()
    controller.abort()

    const error = await listLibraryFiles(defaultQuery, controller.signal)
      .catch((reason: unknown) => reason)

    expect(isLibraryAbortError(error)).toBe(true)
    expect(clientMocks.get).not.toHaveBeenCalled()
  })

  it('uploads multipart data without forcing a boundary and clamps progress', async () => {
    const progress = vi.fn()
    const file = new File(['snow'], 'snow.png', { type: 'image/png' })
    clientMocks.post.mockImplementationOnce(async (
      path: string,
      body: FormData,
      config: {
        timeout: number
        headers: Record<string, unknown>
        onUploadProgress: (event: { loaded: number; total?: number }) => void
      },
    ) => {
      expect(path).toBe('/library/files/upload')
      expect(body).toBeInstanceOf(FormData)
      expect(body.get('file')).toBeInstanceOf(File)
      expect((body.get('file') as File).name).toBe('snow.png')
      expect(config.timeout).toBe(0)
      expect(config.headers).toEqual({ 'Content-Type': undefined })
      config.onUploadProgress({ loaded: 2, total: 4 })
      config.onUploadProgress({ loaded: 8, total: 4 })
      return { data: { file: imagePayload() } }
    })

    await expect(uploadLibraryFile(file, { onProgress: progress }))
      .resolves.toMatchObject({ id: 'image-1', name: 'snow.png' })
    expect(progress.mock.calls.map(([value]) => value)).toEqual([50, 100])
  })

  it('encodes detail and content paths and requires Blob responses', async () => {
    const blob = new Blob(['content'], { type: 'application/pdf' })
    clientMocks.get
      .mockResolvedValueOnce({ data: imagePayload({ id: 'folder/file 1' }) })
      .mockResolvedValueOnce({ data: blob, headers: {} })
      .mockResolvedValueOnce({ data: blob, headers: {} })
      .mockResolvedValueOnce({
        data: blob,
        headers: {
          'content-disposition': "attachment; filename=snow.png; filename*=UTF-8''%E9%9B%AA.png",
        },
      })
      .mockResolvedValueOnce({ data: 'not-a-blob' })

    await expect(getLibraryFile(' folder/file 1 ')).resolves.toMatchObject({
      id: 'folder/file 1',
    })
    await expect(getLibraryThumbnail('folder/file 1')).resolves.toBe(blob)
    await expect(getLibraryPreview('folder/file 1')).resolves.toBe(blob)
    await expect(downloadLibraryFile('folder/file 1', undefined, 'fallback.pdf')).resolves.toEqual({
      blob,
      filename: '雪.png',
    })
    await expect(getLibraryPreview('folder/file 1')).rejects.toMatchObject({
      name: 'LibraryAPIError',
      code: 'INVALID_FILE_CONTENT',
    })

    expect(clientMocks.get.mock.calls.map(([path]) => path)).toEqual([
      '/library/files/folder%2Ffile%201',
      '/library/files/folder%2Ffile%201/thumbnail',
      '/library/files/folder%2Ffile%201/preview',
      '/library/files/folder%2Ffile%201/download',
      '/library/files/folder%2Ffile%201/preview',
    ])
    expect(clientMocks.get.mock.calls.slice(1).every(([, config]) => (
      config.responseType === 'blob'
    ))).toBe(true)
  })

  it('uses the known file name when a download response omits Content-Disposition', async () => {
    const blob = new Blob(['content'])
    clientMocks.get.mockResolvedValueOnce({ data: blob, headers: {} })

    await expect(downloadLibraryFile('image-1', undefined, 'snow.png')).resolves.toEqual({
      blob,
      filename: 'snow.png',
    })
  })

  it('parses RFC 5987 and quoted download filenames with safe fallbacks', () => {
    expect(libraryDownloadFilename(
      "attachment; filename=fallback.pdf; filename*=UTF-8''%E8%B5%84%E6%96%99.pdf",
      'download',
    )).toBe('资料.pdf')
    expect(libraryDownloadFilename('attachment; filename="report \\"final\\".pdf"', 'download'))
      .toBe('report "final".pdf')
    expect(libraryDownloadFilename('', 'library-files.zip')).toBe('library-files.zip')
  })

  it('deduplicates batch IDs, deletes encoded IDs, and normalizes storage usage', async () => {
    const archive = new Blob(['zip'], { type: 'application/zip' })
    const controller = new AbortController()
    clientMocks.delete.mockResolvedValueOnce({ data: undefined })
    clientMocks.post.mockResolvedValueOnce({
      data: archive,
      headers: { 'content-disposition': 'attachment; filename="library-files.zip"' },
    })
    clientMocks.get.mockResolvedValueOnce({
      data: {
        used_bytes: '4096',
        limit_bytes: 16_384,
        remaining_bytes: 12_288,
        over_limit: false,
      },
    })

    await expect(deleteLibraryFile(' folder/file 1 ', controller.signal)).resolves.toBeUndefined()
    await expect(batchDownloadLibraryFiles(
      [' image-1 ', 'document-1', 'image-1', ''],
      controller.signal,
    )).resolves.toEqual({ blob: archive, filename: 'library-files.zip' })
    await expect(getLibraryStorage(controller.signal)).resolves.toEqual({
      usedBytes: 4096,
      limitBytes: 16_384,
      remainingBytes: 12_288,
      overLimit: false,
      unlimited: false,
    })

    expect(clientMocks.delete).toHaveBeenCalledWith(
      '/library/files/folder%2Ffile%201',
      { signal: controller.signal },
    )
    expect(clientMocks.post).toHaveBeenCalledWith(
      '/library/files/batch-download',
      { fileIds: ['image-1', 'document-1'] },
      { signal: controller.signal, responseType: 'blob' },
    )
    expect(clientMocks.get).toHaveBeenCalledWith(
      '/library/storage',
      { signal: controller.signal },
    )
  })

  it('represents a zero storage limit as unlimited instead of full', async () => {
    clientMocks.get.mockResolvedValueOnce({
      data: { used_bytes: 8192, limit_bytes: 0, remaining_bytes: 0, over_limit: false },
    })

    await expect(getLibraryStorage()).resolves.toEqual({
      usedBytes: 8192,
      limitBytes: 0,
      remainingBytes: 0,
      overLimit: false,
      unlimited: true,
    })
  })

  it('creates a deduplicated one-time native download ticket', async () => {
    const controller = new AbortController()
    clientMocks.post.mockResolvedValueOnce({
      data: {
        download_path: downloadPath,
        filename: '资料.zip',
        file_count: 2,
        total_bytes: 8192,
        expires_in: 60,
      },
    })

    const ticket = await createLibraryDownloadTicket(
      [' image-1 ', 'document-1', 'image-1', ''],
      controller.signal,
    )
    expect(ticket).toEqual({
      downloadPath,
      filename: '资料.zip',
      fileCount: 2,
      totalBytes: 8192,
      expiresIn: 60,
    })
    expect(clientMocks.post).toHaveBeenCalledWith(
      '/library/files/download-ticket',
      { fileIds: ['image-1', 'document-1'] },
      { signal: controller.signal },
    )
    expect(ticket.downloadPath).not.toMatch(/[?#]/)
    expect(ticket).not.toHaveProperty('code')
    expect(ticket).not.toHaveProperty('secret')
  })

  it.each([
    'https://evil.example/api/v1/library/download/ABCdef0123456789_-abcd',
    '//evil.example/api/v1/library/download/ABCdef0123456789_-abcd',
    '/api/v1/library/download/ABCdef0123456789_-abc',
    '/api/v1/library/download/ABCdef0123456789_-abcde',
    '/api/v1/library/download/ABCdef0123456789_+abcd',
    '/api/v1/library/download/ABCdef0123456789_-abcd/extra',
    '/api/v1/library/download/ABCdef0123456789_-abcd?code=secret',
    '/api/v1/library/download/ABCdef0123456789_-abcd#secret',
  ])('rejects unsafe or malformed download path %s', async (unsafePath) => {
    clientMocks.post.mockResolvedValueOnce({
      data: {
        download_path: unsafePath,
        filename: 'file.pdf',
        file_count: 1,
        total_bytes: 1,
        expires_in: 60,
      },
    })
    await expect(createLibraryDownloadTicket(['image-1'])).rejects.toMatchObject({
      code: 'INVALID_DOWNLOAD_TICKET_RESPONSE',
    })
  })

  it('preserves structured server errors for precise upload feedback', async () => {
    clientMocks.get.mockRejectedValueOnce({
      isAxiosError: true,
      message: 'Request failed',
      response: {
        status: 413,
        data: {
          code: 413,
          reason: 'LIBRARY_STORAGE_LIMIT_EXCEEDED',
          message: '资料库存储空间不足',
          metadata: { limit_bytes: 1024 },
        },
      },
    })

    const error = await listLibraryFiles(defaultQuery).catch((reason: unknown) => reason)

    expect(error).toBeInstanceOf(LibraryAPIError)
    expect(error).toMatchObject({
      status: 413,
      code: 'LIBRARY_STORAGE_LIMIT_EXCEEDED',
      reason: 'LIBRARY_STORAGE_LIMIT_EXCEEDED',
      message: '资料库存储空间不足',
      metadata: { limit_bytes: 1024 },
    })
  })
})

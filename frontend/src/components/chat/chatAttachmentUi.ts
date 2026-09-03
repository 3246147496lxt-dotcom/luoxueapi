import type { ChatAttachment, ChatAttachmentKind } from '@/types/chat'

export const CHAT_ATTACHMENT_ACCEPT = [
  '.jpg',
  '.jpeg',
  '.png',
  '.webp',
  '.docx',
].join(',')

export const CHAT_ATTACHMENT_MAX_COUNT = 4
export const CHAT_ATTACHMENT_TOTAL_MAX_BYTES = 20 * 1024 * 1024
export const CHAT_ATTACHMENT_IMAGE_MAX_BYTES = 10 * 1024 * 1024
export const CHAT_ATTACHMENT_DOCUMENT_MAX_BYTES = 20 * 1024 * 1024

export type ChatAttachmentDraftState = 'uploading' | 'processing' | 'ready' | 'error'

export interface ChatAttachmentDraft {
  key: string
  file: File
  kind: ChatAttachmentKind
  source?: 'upload' | 'library'
  libraryFileId?: string
  state: ChatAttachmentDraftState
  progress: number
  previewUrl?: string
  attachment?: ChatAttachment
  errorKey?: string
  errorArgs?: Record<string, string | number>
}

export interface ChatAttachmentValidationError {
  key: string
  args?: Record<string, string | number>
}

const IMAGE_EXTENSIONS = new Set(['jpg', 'jpeg', 'png', 'webp'])
const DOCUMENT_EXTENSIONS = new Set(['docx'])
const IMAGE_MIME_TYPES = new Set(['image/jpeg', 'image/png', 'image/webp'])
const DOCUMENT_MIME_TYPES = new Set([
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
])
const PASTED_IMAGE_EXTENSIONS: Readonly<Record<string, string>> = {
  'image/jpeg': 'jpg',
  'image/png': 'png',
  'image/webp': 'webp',
}
const PASTED_IMAGE_MIME_ALIASES: Readonly<Record<string, string>> = {
  'image/jpg': 'image/jpeg',
  'image/pjpeg': 'image/jpeg',
  'image/x-png': 'image/png',
}
const PASTED_IMAGE_EXTENSION_MIME_TYPES: Readonly<Record<string, string>> = {
  jpg: 'image/jpeg',
  jpeg: 'image/jpeg',
  png: 'image/png',
  webp: 'image/webp',
}

function normalizedMimeType(value: string): string {
  return value.split(';', 1)[0]?.trim().toLowerCase() ?? ''
}

function supportedPastedImageMimeType(value: string): string | null {
  const mimeType = normalizedMimeType(value)
  if (mimeType in PASTED_IMAGE_EXTENSIONS) return mimeType
  return PASTED_IMAGE_MIME_ALIASES[mimeType] ?? null
}

function pastedImageMimeType(file: File, itemMimeType: string): string | null {
  const fileMimeType = normalizedMimeType(file.type)
  const itemType = normalizedMimeType(itemMimeType)
  const supportedFileType = supportedPastedImageMimeType(fileMimeType)
  if (supportedFileType) return supportedFileType
  const supportedItemType = supportedPastedImageMimeType(itemType)
  if (supportedItemType) return supportedItemType

  if ([fileMimeType, itemType].some((type) => type.startsWith('image/'))) return null
  const extension = file.name.split('.').pop()?.toLowerCase() ?? ''
  return PASTED_IMAGE_EXTENSION_MIME_TYPES[extension] ?? null
}

function normalizedPastedImageFile(
  file: File,
  itemMimeType: string,
  timestamp: number,
  index: number,
): File {
  const mimeType = pastedImageMimeType(file, itemMimeType)
  if (!mimeType) return file
  const extension = PASTED_IMAGE_EXTENSIONS[mimeType]!
  const originalExtension = file.name.split('.').pop()?.toLowerCase() ?? ''
  const originalNameMatches = PASTED_IMAGE_EXTENSION_MIME_TYPES[originalExtension] === mimeType
  const originalTypeMatches = file.type.toLowerCase() === mimeType
  if (originalNameMatches && originalTypeMatches) return file

  const sequence = index > 0 ? `-${index + 1}` : ''
  const name = originalNameMatches
    ? file.name
    : `pasted-image-${timestamp}${sequence}.${extension}`
  return new File([file], name, {
    type: mimeType,
    lastModified: file.lastModified || timestamp,
  })
}

function clipboardItemFile(item: DataTransferItem): File | null {
  try {
    return item.getAsFile()
  } catch {
    return null
  }
}

export function pastedChatImageFiles(data: DataTransfer | null): File[] {
  if (!data) return []

  // Chromium may expose one physical clipboard image twice: once through a
  // DataTransferItem and again through DataTransfer.files. Those two File
  // wrappers do not reliably share a name or lastModified value, so metadata
  // equality cannot identify the duplicate. Pair both views by their file
  // slot instead, prefer the item (it carries the most useful MIME hint), and
  // fall back to the corresponding File when getAsFile() is unavailable.
  const fileItems = Array.from(data.items ?? []).filter((item) => item.kind === 'file')
  const fileList = Array.from(data.files ?? [])
  const candidates = fileItems.flatMap((item, index) => {
    const file = clipboardItemFile(item) ?? fileList[index]
    return file ? [{ file, itemMimeType: item.type || file.type }] : []
  })
  for (const file of fileList.slice(fileItems.length)) {
    candidates.push({ file, itemMimeType: file.type })
  }
  const timestamp = Date.now()

  return candidates.flatMap(({ file, itemMimeType }, index) => {
    if (!pastedImageMimeType(file, itemMimeType)) return []
    return [normalizedPastedImageFile(file, itemMimeType, timestamp, index)]
  })
}

export function attachmentFileKind(file: File): ChatAttachmentKind | null {
  const extension = file.name.split('.').pop()?.toLowerCase() ?? ''
  const mimeType = file.type.toLowerCase()
  if (IMAGE_EXTENSIONS.has(extension) && (!mimeType || IMAGE_MIME_TYPES.has(mimeType))) {
    return 'image'
  }
  if (
    DOCUMENT_EXTENSIONS.has(extension)
    && (!mimeType || DOCUMENT_MIME_TYPES.has(mimeType))
  ) {
    return 'document'
  }
  return null
}

export function validateAttachmentFile(
  file: File,
  options: {
    supportsVision: boolean
    otherTotalBytes: number
    otherDocumentCount: number
  },
): { kind: ChatAttachmentKind } | ChatAttachmentValidationError {
  const kind = attachmentFileKind(file)
  if (!kind) return { key: 'chat.attachments.errors.type' }
  if (file.size <= 0) return { key: 'chat.attachments.errors.empty' }
  if (kind === 'image' && !options.supportsVision) {
    return { key: 'chat.attachments.errors.visionUnsupported' }
  }
  if (kind === 'document' && options.otherDocumentCount >= 1) {
    return { key: 'chat.attachments.errors.oneDocument' }
  }
  const fileLimit = kind === 'image'
    ? CHAT_ATTACHMENT_IMAGE_MAX_BYTES
    : CHAT_ATTACHMENT_DOCUMENT_MAX_BYTES
  if (file.size > fileLimit) {
    return {
      key: kind === 'image'
        ? 'chat.attachments.errors.imageTooLarge'
        : 'chat.attachments.errors.documentTooLarge',
      args: { size: Math.round(fileLimit / 1024 / 1024) },
    }
  }
  if (options.otherTotalBytes + file.size > CHAT_ATTACHMENT_TOTAL_MAX_BYTES) {
    return {
      key: 'chat.attachments.errors.totalTooLarge',
      args: { size: Math.round(CHAT_ATTACHMENT_TOTAL_MAX_BYTES / 1024 / 1024) },
    }
  }
  return { kind }
}

export function formatAttachmentBytes(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '0 KB'
  if (value < 1024 * 1024) return `${Math.max(1, Math.round(value / 1024))} KB`
  return `${(value / 1024 / 1024).toFixed(value >= 10 * 1024 * 1024 ? 0 : 1)} MB`
}

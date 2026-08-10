import type { ChatAttachment, ChatAttachmentKind } from '@/types/chat'

export const CHAT_ATTACHMENT_ACCEPT = [
  '.jpg',
  '.jpeg',
  '.png',
  '.webp',
  '.pdf',
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
const DOCUMENT_EXTENSIONS = new Set(['pdf', 'docx'])
const IMAGE_MIME_TYPES = new Set(['image/jpeg', 'image/png', 'image/webp'])
const DOCUMENT_MIME_TYPES = new Set([
  'application/pdf',
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
])

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

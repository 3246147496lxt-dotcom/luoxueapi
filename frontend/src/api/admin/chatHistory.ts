/**
 * Read-only administrator access to a user's web-chat history.
 *
 * Conversation metadata is intentionally separate from content access. Calling
 * `getConversation` causes the server to write a dedicated content-access audit
 * record, including when older messages are requested.
 */

import { apiClient } from '../client'

export interface AdminChatConversationSummary {
  id: string
  model: string
  message_count: number
  status: string
  created_at: string
  updated_at: string
}

export interface AdminChatConversationListPage {
  items: AdminChatConversationSummary[]
  next_cursor?: string
  has_more: boolean
}

export interface AdminChatConversation {
  id: string
  title: string
  model: string
  status: string
  created_at: string
  updated_at: string
}

export interface AdminChatMessage {
  id: string
  position: number
  role: string
  content: string
  status: string
  requested_model?: string
  finish_reason?: string
  error_code?: string
  error_message?: string
  attempt_id?: string
  receipt_id?: string
  created_at: string
  completed_at?: string
}

export interface AdminChatConversationDetail {
  conversation: AdminChatConversation
  messages: AdminChatMessage[]
  next_before_position?: number
  has_more: boolean
}

export interface AdminChatConversationListParams {
  cursor?: string
  limit?: number
}

export interface AdminChatConversationDetailParams {
  before_position?: number
  limit?: number
}

export interface AdminChatRequestOptions {
  signal?: AbortSignal
}

type UnknownRecord = Record<string, unknown>

const asRecord = (value: unknown): UnknownRecord | null => (
  value !== null && typeof value === 'object' && !Array.isArray(value)
    ? value as UnknownRecord
    : null
)

const unwrapPayload = (value: unknown): UnknownRecord => {
  let current = value

  for (let depth = 0; depth < 3; depth += 1) {
    const record = asRecord(current)
    if (!record) return {}

    const nested = asRecord(record.data)
    const alreadyUnwrapped = 'items' in record || 'messages' in record || 'conversation' in record
    if (!nested || alreadyUnwrapped) return record
    current = nested
  }

  return asRecord(current) ?? {}
}

const normalizeConversationListPage = (value: unknown): AdminChatConversationListPage => {
  const payload = unwrapPayload(value)
  const rawItems = Array.isArray(payload.items)
    ? payload.items
    : Array.isArray(payload.conversations)
      ? payload.conversations
      : []
  const nextCursor = typeof payload.next_cursor === 'string' && payload.next_cursor.trim()
    ? payload.next_cursor
    : undefined

  return {
    items: rawItems as AdminChatConversationSummary[],
    has_more: payload.has_more === true,
    ...(nextCursor ? { next_cursor: nextCursor } : {})
  }
}

const normalizeConversationDetail = (value: unknown): AdminChatConversationDetail => {
  const payload = unwrapPayload(value)
  const conversation = asRecord(payload.conversation) ?? {}
  const rawMessages = Array.isArray(payload.messages) ? payload.messages : []
  const nextBeforePosition = typeof payload.next_before_position === 'number'
    && Number.isSafeInteger(payload.next_before_position)
    && payload.next_before_position > 0
    ? payload.next_before_position
    : undefined

  return {
    conversation: conversation as unknown as AdminChatConversation,
    messages: rawMessages as AdminChatMessage[],
    has_more: payload.has_more === true,
    ...(nextBeforePosition ? { next_before_position: nextBeforePosition } : {})
  }
}

const userConversationPath = (userId: number | string): string => (
  `/admin/users/${encodeURIComponent(String(userId))}/chat/conversations`
)

export async function listUserConversations(
  userId: number | string,
  params: AdminChatConversationListParams = {},
  options: AdminChatRequestOptions = {}
): Promise<AdminChatConversationListPage> {
  const { data } = await apiClient.get(userConversationPath(userId), {
    params,
    signal: options.signal
  })

  return normalizeConversationListPage(data)
}

export async function getUserConversation(
  userId: number | string,
  conversationId: string,
  params: AdminChatConversationDetailParams = {},
  options: AdminChatRequestOptions = {}
): Promise<AdminChatConversationDetail> {
  const { data } = await apiClient.get(
    `${userConversationPath(userId)}/${encodeURIComponent(conversationId)}`,
    {
      params,
      signal: options.signal
    }
  )

  return normalizeConversationDetail(data)
}

export const adminChatHistoryAPI = {
  listUserConversations,
  getUserConversation
}

export default adminChatHistoryAPI

import type {
  ChatActivity,
  ChatActivityError,
  ChatActivityEvent,
  ChatActivityEventType,
  ChatActivityItem,
  ChatActivityStatus,
  ChatActivitySummaryPart,
  ChatReasoningEffort,
  ReasoningMode,
} from '@/types/chat'

const ACTIVITY_EVENT_TYPES = new Set<ChatActivityEventType>([
  'response.created',
  'response.output_item.added',
  'response.output_item.done',
  'response.reasoning_summary_part.added',
  'response.reasoning_summary_text.delta',
  'response.reasoning_summary_text.done',
  'response.reasoning_summary_part.done',
  'response.completed',
  'response.incomplete',
  'response.failed',
])

const ACTIVITY_STATUSES = new Set<ChatActivityStatus>([
  'pending',
  'streaming',
  'completed',
  'incomplete',
  'failed',
  'stopped',
  'disconnected',
])

const REASONING_EFFORTS = new Set<ChatReasoningEffort>([
  'low',
  'medium',
  'high',
  'xhigh',
])

const OFFICIAL_RESPONSE_TERMINAL_STATUSES = new Set<ChatActivityStatus>([
  'completed',
  'incomplete',
  'failed',
])

const IMMEDIATE_EVENT_TYPES = new Set<ChatActivityEventType>([
  'response.created',
  'response.output_item.added',
  'response.output_item.done',
  'response.reasoning_summary_part.added',
  'response.reasoning_summary_text.done',
  'response.reasoning_summary_part.done',
  'response.completed',
  'response.incomplete',
  'response.failed',
])

const SEQUENCED_BOUNDARY_EVENT_TYPES = new Set<ChatActivityEventType>([
  'response.output_item.done',
  'response.reasoning_summary_text.done',
  'response.reasoning_summary_part.done',
  'response.completed',
  'response.incomplete',
  'response.failed',
])

const DEFAULT_REORDER_BUFFER_SIZE = 16
const DEFAULT_REORDER_WAIT_MS = 25
const DEFAULT_NOTIFICATION_BATCH_MS = 40

interface NormalizeChatActivitiesOptions {
  fallbackStartedAt?: number
  markDisconnected?: boolean
}

interface MergeChatActivitiesOptions {
  incomingIsServer?: boolean
}

export interface ChatActivityStateMachineOptions {
  onActivity?: (activity: ChatActivity) => void
  reasoningMode?: ReasoningMode
  reasoningEffort?: ChatReasoningEffort
  now?: () => number
  reorderBufferSize?: number
  reorderWaitMs?: number
  notificationBatchMs?: number
}

function asRecord(value: unknown): Record<string, unknown> | null {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
    ? value as Record<string, unknown>
    : null
}

function nonEmptyString(value: unknown): string | null {
  return typeof value === 'string' && value.trim() ? value.trim() : null
}

function nonNegativeInteger(value: unknown): number | null {
  return typeof value === 'number'
    && Number.isSafeInteger(value)
    && value >= 0
    ? value
    : null
}

function optionalTimestamp(value: unknown): number | null {
  if (typeof value === 'string' && value.trim()) {
    const numeric = Number(value)
    if (Number.isFinite(numeric) && numeric >= 0) return numeric
    const parsed = Date.parse(value)
    return Number.isFinite(parsed) ? parsed : null
  }
  if (typeof value !== 'number' || !Number.isFinite(value) || value < 0) return null
  return value
}

function activityStatus(value: unknown, fallback: ChatActivityStatus): ChatActivityStatus {
  if (value === 'in_progress') return 'streaming'
  if (value === 'interrupted') return 'stopped'
  return ACTIVITY_STATUSES.has(value as ChatActivityStatus)
    ? value as ChatActivityStatus
    : fallback
}

function reasoningMode(value: unknown): ReasoningMode | undefined {
  return value === 'standard' || value === 'pro' ? value : undefined
}

function reasoningEffort(value: unknown): ChatReasoningEffort | undefined {
  return REASONING_EFFORTS.has(value as ChatReasoningEffort)
    ? value as ChatReasoningEffort
    : undefined
}

function enforceReasoningMetadataInvariant(activity: ChatActivity): ChatActivity {
  if (activity.reasoningMode === 'pro') delete activity.reasoningEffort
  return activity
}

function normalizedError(value: unknown): ChatActivityError | undefined {
  const candidate = asRecord(value)
  if (!candidate) return undefined
  const code = nonEmptyString(candidate.code ?? candidate.type)
  const message = nonEmptyString(candidate.message)
  return code || message
    ? { ...(code ? { code } : {}), ...(message ? { message } : {}) }
    : undefined
}

function disconnectedStatus(status: ChatActivityStatus, enabled: boolean): ChatActivityStatus {
  return enabled && (status === 'pending' || status === 'streaming')
    ? 'disconnected'
    : status
}

export function isChatActivityEventType(value: unknown): value is ChatActivityEventType {
  return typeof value === 'string' && ACTIVITY_EVENT_TYPES.has(value as ChatActivityEventType)
}

export function isTerminalChatActivityStatus(status: ChatActivityStatus): boolean {
  return status !== 'pending' && status !== 'streaming'
}

export function chatActivityKey(responseId: string): string {
  return responseId || 'openai_responses:unknown'
}

export function chatActivityItemKey(
  responseId: string,
  itemId: string,
  outputIndex: number,
): string {
  return JSON.stringify([responseId, itemId, outputIndex])
}

export function chatActivityPartKey(
  responseId: string,
  itemId: string,
  outputIndex: number,
  summaryIndex: number,
): string {
  return JSON.stringify([responseId, itemId, outputIndex, summaryIndex])
}

export function chatActivityPartHasText(part: ChatActivitySummaryPart): boolean {
  if (/\S/u.test(part.text)) return true
  return part.streamingTextChunks?.some((chunk) => /\S/u.test(chunk)) === true
}

export function chatActivityPartText(part: ChatActivitySummaryPart): string {
  const chunks = part.streamingTextChunks
  if (!chunks || chunks.length === 0) return part.text
  return [part.text, ...chunks].join('')
}

export function finalizeChatActivityPartText(part: ChatActivitySummaryPart): void {
  if (!part.streamingTextChunks || part.streamingTextChunks.length === 0) return
  part.text = chatActivityPartText(part)
  delete part.streamingTextChunks
}

function clonePart(part: ChatActivitySummaryPart): ChatActivitySummaryPart {
  return {
    ...part,
    ...(part.streamingTextChunks
      ? { streamingTextChunks: [...part.streamingTextChunks] }
      : {}),
  }
}

function cloneItem(item: ChatActivityItem): ChatActivityItem {
  return { ...item, parts: item.parts.map(clonePart) }
}

export function cloneChatActivity(activity: ChatActivity): ChatActivity {
  return {
    ...activity,
    ...(activity.error ? { error: { ...activity.error } } : {}),
    items: activity.items.map(cloneItem),
  }
}

function normalizePart(
  value: unknown,
  responseId: string,
  itemId: string,
  outputIndex: number,
  fallbackStartedAt: number,
  fallbackStatus: ChatActivityStatus,
  summaryIndexFallback: number,
  markDisconnected: boolean,
): ChatActivitySummaryPart | null {
  const candidate = asRecord(value)
  if (!candidate) return null
  if (candidate.type !== undefined && candidate.type !== 'summary_text') return null
  const summaryIndex = nonNegativeInteger(candidate.summaryIndex ?? candidate.summary_index)
    ?? summaryIndexFallback
  const rawText = candidate.text ?? candidate.summary_text
  if (rawText !== undefined && typeof rawText !== 'string') return null
  const rawStreamingTextChunks = candidate.streamingTextChunks ?? candidate.streaming_text_chunks
  if (
    rawStreamingTextChunks !== undefined
    && (
      !Array.isArray(rawStreamingTextChunks)
      || rawStreamingTextChunks.some((chunk) => typeof chunk !== 'string')
    )
  ) return null
  const startedAt = optionalTimestamp(candidate.startedAt ?? candidate.started_at)
    ?? fallbackStartedAt
  const updatedAt = optionalTimestamp(candidate.updatedAt ?? candidate.updated_at)
    ?? startedAt
  const completedAt = optionalTimestamp(candidate.completedAt ?? candidate.completed_at)
  const status = disconnectedStatus(
    activityStatus(candidate.status, completedAt === null ? fallbackStatus : 'completed'),
    markDisconnected,
  )
  const part: ChatActivitySummaryPart = {
    key: chatActivityPartKey(responseId, itemId, outputIndex, summaryIndex),
    itemId,
    outputIndex,
    summaryIndex,
    text: typeof rawText === 'string' ? rawText : '',
    status,
    startedAt,
    updatedAt: Math.max(startedAt, updatedAt),
  }
  if (Array.isArray(rawStreamingTextChunks)) {
    const chunks = rawStreamingTextChunks.filter((chunk) => chunk.length > 0)
    if (chunks.length > 0) part.streamingTextChunks = [...chunks]
  }
  if (markDisconnected || isTerminalChatActivityStatus(status)) {
    finalizeChatActivityPartText(part)
  }
  if (completedAt !== null) part.completedAt = completedAt
  const lastSequenceNumber = nonNegativeInteger(
    candidate.lastSequenceNumber ?? candidate.last_sequence_number,
  )
  if (lastSequenceNumber !== null) part.lastSequenceNumber = lastSequenceNumber
  return part
}

function preferredSnapshot<T extends {
  status: ChatActivityStatus
  updatedAt: number
  lastSequenceNumber?: number
}>(
  existing: T,
  incoming: T,
  incomingIsServer: boolean,
): T {
  // Only the server is authoritative enough to replace a newer local
  // snapshot with an older terminal response. IndexedDB writes can arrive
  // from stale tabs, so local merges must still honor sequence/timestamp
  // ordering even when that stale snapshot says completed/failed.
  if (
    incomingIsServer
    && (
      OFFICIAL_RESPONSE_TERMINAL_STATUSES.has(incoming.status)
      || isTerminalChatActivityStatus(incoming.status)
    )
  ) return incoming
  if (
    isTerminalChatActivityStatus(existing.status)
    && !isTerminalChatActivityStatus(incoming.status)
  ) return existing
  const existingSequence = existing.lastSequenceNumber ?? -1
  const incomingSequence = incoming.lastSequenceNumber ?? -1
  if (incomingSequence !== existingSequence) {
    return incomingSequence > existingSequence ? incoming : existing
  }
  if (incoming.updatedAt !== existing.updatedAt) {
    return incoming.updatedAt > existing.updatedAt ? incoming : existing
  }
  if (isTerminalChatActivityStatus(incoming.status) && !isTerminalChatActivityStatus(existing.status)) {
    return incoming
  }
  return incoming
}

function mergeParts(
  existing: ChatActivitySummaryPart[],
  incoming: ChatActivitySummaryPart[],
  incomingIsServer: boolean,
): ChatActivitySummaryPart[] {
  const byKey = new Map(existing.map((part) => [part.key, clonePart(part)]))
  for (const candidate of incoming) {
    const current = byKey.get(candidate.key)
    byKey.set(
      candidate.key,
      current
        ? clonePart(preferredSnapshot(current, candidate, incomingIsServer))
        : clonePart(candidate),
    )
  }
  return [...byKey.values()].sort((left, right) => (
    left.summaryIndex - right.summaryIndex || left.key.localeCompare(right.key)
  ))
}

function normalizeItem(
  value: unknown,
  responseId: string,
  fallbackStartedAt: number,
  fallbackStatus: ChatActivityStatus,
  markDisconnected: boolean,
): ChatActivityItem | null {
  const candidate = asRecord(value)
  if (!candidate) return null
  if (candidate.type !== undefined && candidate.type !== 'reasoning') return null
  const itemId = nonEmptyString(candidate.itemId ?? candidate.item_id ?? candidate.id)
  const outputIndex = nonNegativeInteger(candidate.outputIndex ?? candidate.output_index)
  if (!itemId || outputIndex === null) return null
  const startedAt = optionalTimestamp(candidate.startedAt ?? candidate.started_at)
    ?? fallbackStartedAt
  const updatedAt = optionalTimestamp(candidate.updatedAt ?? candidate.updated_at)
    ?? startedAt
  const completedAt = optionalTimestamp(candidate.completedAt ?? candidate.completed_at)
  const status = disconnectedStatus(
    activityStatus(candidate.status, completedAt === null ? fallbackStatus : 'completed'),
    markDisconnected,
  )
  const partsValue = candidate.parts ?? candidate.summary ?? candidate.summary_parts
  const partsByKey = new Map<string, ChatActivitySummaryPart>()
  if (Array.isArray(partsValue)) {
    for (let index = 0; index < partsValue.length; index += 1) {
      const part = normalizePart(
        partsValue[index],
        responseId,
        itemId,
        outputIndex,
        startedAt,
        status,
        index,
        markDisconnected,
      )
      if (!part) continue
      const existing = partsByKey.get(part.key)
      partsByKey.set(part.key, existing ? preferredSnapshot(existing, part, false) : part)
    }
  }
  const item: ChatActivityItem = {
    key: chatActivityItemKey(responseId, itemId, outputIndex),
    itemId,
    outputIndex,
    status,
    parts: [...partsByKey.values()].sort((left, right) => (
      left.summaryIndex - right.summaryIndex || left.key.localeCompare(right.key)
    )),
    startedAt,
    updatedAt: Math.max(startedAt, updatedAt),
  }
  if (completedAt !== null) item.completedAt = completedAt
  const lastSequenceNumber = nonNegativeInteger(
    candidate.lastSequenceNumber ?? candidate.last_sequence_number,
  )
  if (lastSequenceNumber !== null) item.lastSequenceNumber = lastSequenceNumber
  return item
}

function mergeItems(
  existing: ChatActivityItem[],
  incoming: ChatActivityItem[],
  incomingIsServer: boolean,
): ChatActivityItem[] {
  const byKey = new Map(existing.map((item) => [item.key, cloneItem(item)]))
  for (const candidate of incoming) {
    const current = byKey.get(candidate.key)
    if (!current) {
      byKey.set(candidate.key, cloneItem(candidate))
      continue
    }
    const preferred = preferredSnapshot(current, candidate, incomingIsServer)
    byKey.set(candidate.key, {
      ...preferred,
      parts: mergeParts(current.parts, candidate.parts, incomingIsServer),
    })
  }
  return [...byKey.values()].sort((left, right) => (
    left.outputIndex - right.outputIndex || left.key.localeCompare(right.key)
  ))
}

function normalizeActivity(
  value: unknown,
  options: NormalizeChatActivitiesOptions,
): ChatActivity | null {
  const candidate = asRecord(value)
  if (!candidate) return null
  const rawResponseId = candidate.responseId ?? candidate.response_id ?? candidate.id
  const responseId = typeof rawResponseId === 'string' ? rawResponseId.trim() : null
  if (responseId === null) return null
  const startedAt = optionalTimestamp(candidate.startedAt ?? candidate.started_at)
    ?? options.fallbackStartedAt
    ?? 0
  const updatedAt = optionalTimestamp(candidate.updatedAt ?? candidate.updated_at)
    ?? startedAt
  const completedAt = optionalTimestamp(candidate.completedAt ?? candidate.completed_at)
  const status = disconnectedStatus(
    activityStatus(candidate.status, completedAt === null ? 'streaming' : 'completed'),
    options.markDisconnected === true,
  )
  const itemsByKey = new Map<string, ChatActivityItem>()
  if (Array.isArray(candidate.items)) {
    for (const itemValue of candidate.items) {
      const item = normalizeItem(
        itemValue,
        responseId,
        startedAt,
        status,
        options.markDisconnected === true,
      )
      if (!item) continue
      const existing = itemsByKey.get(item.key)
      if (!existing) itemsByKey.set(item.key, item)
      else {
        const preferred = preferredSnapshot(existing, item, false)
        itemsByKey.set(item.key, {
          ...preferred,
          parts: mergeParts(existing.parts, item.parts, false),
        })
      }
    }
  }
  const items = [...itemsByKey.values()].sort((left, right) => (
    left.outputIndex - right.outputIndex || left.key.localeCompare(right.key)
  ))
  // A response.created record by itself is transport state, not a displayable activity.
  if (items.length === 0) return null
  const activity: ChatActivity = {
    key: chatActivityKey(responseId),
    responseId,
    status,
    items,
    startedAt,
    updatedAt: Math.max(startedAt, updatedAt),
  }
  const mode = reasoningMode(candidate.reasoningMode ?? candidate.reasoning_mode)
  const effort = reasoningEffort(candidate.reasoningEffort ?? candidate.reasoning_effort)
  const error = normalizedError(candidate.error)
  if (mode) activity.reasoningMode = mode
  if (effort && mode !== 'pro') activity.reasoningEffort = effort
  if (completedAt !== null) activity.completedAt = completedAt
  if (error) activity.error = error
  const lastSequenceNumber = nonNegativeInteger(
    candidate.lastSequenceNumber ?? candidate.last_sequence_number,
  )
  if (lastSequenceNumber !== null) activity.lastSequenceNumber = lastSequenceNumber
  return enforceReasoningMetadataInvariant(activity)
}

function flatActivityStatus(value: unknown, markDisconnected: boolean): ChatActivityStatus | null {
  let status: ChatActivityStatus
  switch (value) {
    case 'in_progress':
    case 'pending':
    case 'streaming':
      status = value === 'pending' ? 'pending' : 'streaming'
      break
    case 'completed':
    case 'incomplete':
    case 'failed':
    case 'stopped':
    case 'disconnected':
      status = value
      break
    case 'interrupted':
      status = 'stopped'
      break
    default:
      return null
  }
  return disconnectedStatus(status, markDisconnected)
}

function aggregateStatuses(statuses: ChatActivityStatus[]): ChatActivityStatus {
  if (statuses.includes('failed')) return 'failed'
  if (statuses.includes('incomplete')) return 'incomplete'
  if (statuses.includes('streaming')) return 'streaming'
  if (statuses.includes('pending')) return 'pending'
  if (statuses.includes('stopped')) return 'stopped'
  if (statuses.includes('disconnected')) return 'disconnected'
  return 'completed'
}

function normalizeFlatChatActivities(
  value: unknown[],
  options: NormalizeChatActivitiesOptions,
): ChatActivity[] {
  const activities = new Map<string, ChatActivity>()
  for (const raw of value) {
    const candidate = asRecord(raw)
    if (
      !candidate
      || candidate.source !== 'openai_responses'
      || candidate.activity_type !== 'reasoning_summary'
    ) continue
    const rawResponseId = candidate.response_id ?? candidate.responseId
    const responseId = typeof rawResponseId === 'string' ? rawResponseId.trim() : ''
    const itemId = nonEmptyString(candidate.item_id ?? candidate.itemId)
    const outputIndex = nonNegativeInteger(candidate.output_index ?? candidate.outputIndex)
    const summaryIndex = nonNegativeInteger(candidate.summary_index ?? candidate.summaryIndex)
    const status = flatActivityStatus(candidate.status, options.markDisconnected === true)
    if (
      !itemId
      || outputIndex === null
      || summaryIndex === null
      || !status
      || typeof candidate.text !== 'string'
    ) continue
    const startedAt = optionalTimestamp(candidate.started_at ?? candidate.startedAt)
      ?? options.fallbackStartedAt
      ?? 0
    const updatedAt = optionalTimestamp(candidate.updated_at ?? candidate.updatedAt)
      ?? startedAt
    const completedAt = optionalTimestamp(candidate.completed_at ?? candidate.completedAt)
    const lastSequenceNumber = nonNegativeInteger(
      candidate.sequence_end
        ?? candidate.sequenceEnd
        ?? candidate.last_sequence_number
        ?? candidate.lastSequenceNumber,
    )
    const activityKey = chatActivityKey(responseId)
    let activity = activities.get(activityKey)
    if (!activity) {
      activity = {
        key: activityKey,
        responseId,
        status,
        items: [],
        startedAt,
        updatedAt: Math.max(startedAt, updatedAt),
      }
      activities.set(activityKey, activity)
    } else {
      activity.startedAt = Math.min(activity.startedAt, startedAt)
      activity.updatedAt = Math.max(activity.updatedAt, updatedAt)
    }
    const mode = reasoningMode(candidate.reasoning_mode ?? candidate.reasoningMode)
    const effort = reasoningEffort(candidate.reasoning_effort ?? candidate.reasoningEffort)
    if (mode) activity.reasoningMode = mode
    if (effort && activity.reasoningMode !== 'pro') activity.reasoningEffort = effort
    enforceReasoningMetadataInvariant(activity)
    if (completedAt !== null) {
      activity.completedAt = Math.max(activity.completedAt ?? 0, completedAt)
    }
    if (lastSequenceNumber !== null) {
      activity.lastSequenceNumber = Math.max(
        activity.lastSequenceNumber ?? -1,
        lastSequenceNumber,
      )
    }

    const itemKey = chatActivityItemKey(responseId, itemId, outputIndex)
    let item = activity.items.find((existing) => existing.key === itemKey)
    if (!item) {
      item = {
        key: itemKey,
        itemId,
        outputIndex,
        status,
        parts: [],
        startedAt,
        updatedAt: Math.max(startedAt, updatedAt),
      }
      activity.items.push(item)
    } else {
      item.startedAt = Math.min(item.startedAt, startedAt)
      item.updatedAt = Math.max(item.updatedAt, updatedAt)
    }
    if (completedAt !== null) item.completedAt = Math.max(item.completedAt ?? 0, completedAt)
    if (lastSequenceNumber !== null) {
      item.lastSequenceNumber = Math.max(item.lastSequenceNumber ?? -1, lastSequenceNumber)
    }

    const part: ChatActivitySummaryPart = {
      key: chatActivityPartKey(responseId, itemId, outputIndex, summaryIndex),
      itemId,
      outputIndex,
      summaryIndex,
      text: candidate.text,
      status,
      startedAt,
      updatedAt: Math.max(startedAt, updatedAt),
    }
    if (completedAt !== null) part.completedAt = completedAt
    if (lastSequenceNumber !== null) part.lastSequenceNumber = lastSequenceNumber
    const existingPartIndex = item.parts.findIndex(({ key }) => key === part.key)
    if (existingPartIndex < 0) item.parts.push(part)
    else {
      item.parts[existingPartIndex] = preferredSnapshot(
        item.parts[existingPartIndex]!,
        part,
        true,
      )
    }
  }

  const result = [...activities.values()]
  for (const activity of result) {
    activity.items.sort((left, right) => (
      left.outputIndex - right.outputIndex || left.key.localeCompare(right.key)
    ))
    for (const item of activity.items) {
      item.parts.sort((left, right) => (
        left.summaryIndex - right.summaryIndex || left.key.localeCompare(right.key)
      ))
      item.status = aggregateStatuses(item.parts.map(({ status }) => status))
      if (!isTerminalChatActivityStatus(item.status)) delete item.completedAt
    }
    activity.status = aggregateStatuses(activity.items.map(({ status }) => status))
    if (!isTerminalChatActivityStatus(activity.status)) delete activity.completedAt
  }
  return result.sort((left, right) => (
    left.startedAt - right.startedAt || left.key.localeCompare(right.key)
  ))
}

export function normalizeChatActivities(
  value: unknown,
  options: NormalizeChatActivitiesOptions = {},
): ChatActivity[] {
  if (!Array.isArray(value)) return []
  const byKey = new Map<string, ChatActivity>()
  for (const candidate of value) {
    const activity = normalizeActivity(candidate, options)
    if (!activity) continue
    const existing = byKey.get(activity.key)
    if (!existing) byKey.set(activity.key, activity)
    else {
      const preferred = preferredSnapshot(existing, activity, false)
      byKey.set(activity.key, {
        ...preferred,
        items: mergeItems(existing.items, activity.items, false),
      })
    }
  }
  for (const activity of normalizeFlatChatActivities(value, options)) {
    const existing = byKey.get(activity.key)
    if (!existing) byKey.set(activity.key, activity)
    else {
      const preferred = preferredSnapshot(existing, activity, false)
      byKey.set(activity.key, {
        ...preferred,
        items: mergeItems(existing.items, activity.items, false),
      })
    }
  }
  return [...byKey.values()].map(enforceReasoningMetadataInvariant).sort((left, right) => (
    left.startedAt - right.startedAt || left.key.localeCompare(right.key)
  ))
}

export function mergeChatActivities(
  existing: readonly ChatActivity[] | undefined,
  incoming: readonly ChatActivity[] | undefined,
  options: MergeChatActivitiesOptions = {},
): ChatActivity[] {
  const byKey = new Map((existing ?? []).map((activity) => [
    activity.key,
    cloneChatActivity(activity),
  ]))
  for (const candidate of incoming ?? []) {
    const current = byKey.get(candidate.key)
    if (!current) {
      byKey.set(candidate.key, cloneChatActivity(candidate))
      continue
    }
    const preferred = preferredSnapshot(
      current,
      candidate,
      options.incomingIsServer === true,
    )
    byKey.set(candidate.key, {
      ...preferred,
      ...(preferred.error ? { error: { ...preferred.error } } : {}),
      items: mergeItems(
        current.items,
        candidate.items,
        options.incomingIsServer === true,
      ),
    })
  }
  return [...byKey.values()].map(enforceReasoningMetadataInvariant).sort((left, right) => (
    left.startedAt - right.startedAt || left.key.localeCompare(right.key)
  ))
}

function responseRecord(payload: Record<string, unknown>): Record<string, unknown> | null {
  return asRecord(payload.response)
}

function responseIdFromPayload(
  payload: Record<string, unknown>,
  fallback: string | null,
): string | null {
  return nonEmptyString(
    payload.response_id
      ?? payload.responseId
      ?? responseRecord(payload)?.id,
  ) ?? fallback
}

function sequenceNumber(payload: Record<string, unknown>): number | null {
  return nonNegativeInteger(payload.sequence_number ?? payload.sequenceNumber)
}

function payloadTimestamp(payload: Record<string, unknown>, fallback: number): number {
  const response = responseRecord(payload)
  return optionalTimestamp(
    payload.created_at
      ?? payload.createdAt
      ?? response?.created_at
      ?? response?.createdAt,
  ) ?? fallback
}

function itemIdentity(payload: Record<string, unknown>): {
  item: Record<string, unknown> | null
  itemId: string | null
  outputIndex: number | null
} {
  const item = asRecord(payload.item)
  return {
    item,
    itemId: nonEmptyString(payload.item_id ?? payload.itemId ?? item?.id),
    outputIndex: nonNegativeInteger(payload.output_index ?? payload.outputIndex),
  }
}

function summaryIdentity(payload: Record<string, unknown>): {
  itemId: string | null
  outputIndex: number | null
  summaryIndex: number | null
} {
  return {
    itemId: nonEmptyString(payload.item_id ?? payload.itemId),
    outputIndex: nonNegativeInteger(payload.output_index ?? payload.outputIndex),
    summaryIndex: nonNegativeInteger(payload.summary_index ?? payload.summaryIndex),
  }
}

export class ChatActivityStateMachine {
  private readonly activities = new Map<string, ChatActivity>()
  private readonly visibleActivityKeys = new Set<string>()
  private readonly authoritativeTextDoneKeys = new Set<string>()
  /**
   * Keep streaming deltas as chunks until a consumer can actually observe
   * them. Appending to `part.text` for every SSE event repeatedly copies the
   * whole accumulated string and becomes quadratic for long summaries.
   */
  private readonly pendingTextChunks = new Map<string, string[]>()
  private readonly outputItemTypes = new Map<string, string>()
  private readonly pendingEvents = new Map<number, ChatActivityEvent>()
  private readonly pendingActivityNotifications = new Map<string, ChatActivity>()
  private readonly onActivity?: (activity: ChatActivity) => void
  private readonly defaultReasoningMode?: ReasoningMode
  private readonly defaultReasoningEffort?: ChatReasoningEffort
  private readonly now: () => number
  private readonly reorderBufferSize: number
  private readonly reorderWaitMs: number
  private readonly notificationBatchMs: number
  private currentResponseId: string | null = null
  private lastAppliedSequence: number | null = null
  private reorderTimer: ReturnType<typeof setTimeout> | null = null
  private notificationTimer: ReturnType<typeof setTimeout> | null = null

  constructor(options: ChatActivityStateMachineOptions = {}) {
    this.onActivity = options.onActivity
    this.defaultReasoningMode = options.reasoningMode
    this.defaultReasoningEffort = options.reasoningMode === 'pro'
      ? undefined
      : options.reasoningEffort
    this.now = options.now ?? Date.now
    this.reorderBufferSize = Math.max(1, options.reorderBufferSize ?? DEFAULT_REORDER_BUFFER_SIZE)
    this.reorderWaitMs = Math.max(0, options.reorderWaitMs ?? DEFAULT_REORDER_WAIT_MS)
    this.notificationBatchMs = Math.max(
      0,
      options.notificationBatchMs ?? DEFAULT_NOTIFICATION_BATCH_MS,
    )
  }

  consume(event: ChatActivityEvent): void {
    if (!isChatActivityEventType(event.eventType)) return
    const payload = asRecord(event.payload)
    if (!payload) return
    const sequence = sequenceNumber(payload)
    if (sequence === null) {
      if (IMMEDIATE_EVENT_TYPES.has(event.eventType)) this.flushPendingEvents()
      this.applyEvent(event, null)
      return
    }
    if (
      (this.lastAppliedSequence !== null && sequence <= this.lastAppliedSequence)
      || this.pendingEvents.has(sequence)
    ) return

    if (this.lastAppliedSequence === null) {
      this.applySequencedEvent(sequence, event)
      return
    }

    this.pendingEvents.set(sequence, event)
    this.drainConsecutiveEvents()
    if (!this.pendingEvents.has(sequence)) return

    // Lifecycle boundaries are authoritative and must not sit behind sequence
    // numbers belonging to public stream events that the private Activity
    // envelope intentionally omits. Flush all already-buffered deltas in
    // sequence order, then apply the done/terminal event synchronously.
    if (SEQUENCED_BOUNDARY_EVENT_TYPES.has(event.eventType)) {
      this.flushPendingEvents()
      return
    }
    if (this.pendingEvents.size >= this.reorderBufferSize) {
      this.flushPendingEvents()
      return
    }
    this.scheduleReorderFlush()
  }

  flush(): void {
    this.flushPendingEvents()
    this.flushActivityNotifications()
  }

  stop(): void {
    this.terminateOpenActivities('stopped')
  }

  disconnect(): void {
    this.terminateOpenActivities('disconnected')
  }

  dispose(): void {
    this.flush()
    this.clearReorderTimer()
    this.clearNotificationTimer()
  }

  snapshots(): ChatActivity[] {
    this.commitAllPendingTextChunks()
    return [...this.activities.values()]
      .filter((activity) => this.visibleActivityKeys.has(activity.key))
      .sort((left, right) => left.startedAt - right.startedAt || left.key.localeCompare(right.key))
      .map(cloneChatActivity)
  }

  private flushPendingEvents(): void {
    this.clearReorderTimer()
    const pending = [...this.pendingEvents.entries()].sort(([left], [right]) => left - right)
    this.pendingEvents.clear()
    for (const [sequence, event] of pending) {
      if (this.lastAppliedSequence !== null && sequence <= this.lastAppliedSequence) continue
      this.applySequencedEvent(sequence, event)
    }
  }

  private scheduleReorderFlush(): void {
    if (this.reorderTimer !== null) return
    this.reorderTimer = setTimeout(() => {
      this.reorderTimer = null
      this.flushPendingEvents()
    }, this.reorderWaitMs)
  }

  private clearReorderTimer(): void {
    if (this.reorderTimer === null) return
    clearTimeout(this.reorderTimer)
    this.reorderTimer = null
  }

  private scheduleActivityNotification(): void {
    if (this.notificationTimer !== null) return
    this.notificationTimer = setTimeout(() => {
      this.notificationTimer = null
      this.flushActivityNotifications()
    }, this.notificationBatchMs)
  }

  private clearNotificationTimer(): void {
    if (this.notificationTimer === null) return
    clearTimeout(this.notificationTimer)
    this.notificationTimer = null
  }

  private flushActivityNotifications(): void {
    this.clearNotificationTimer()
    const pending = [...this.pendingActivityNotifications.values()].sort((left, right) => (
      left.startedAt - right.startedAt || left.key.localeCompare(right.key)
    ))
    this.pendingActivityNotifications.clear()
    for (const activity of pending) {
      this.commitPendingTextChunks(activity)
      this.onActivity?.(cloneChatActivity(activity))
    }
  }

  private appendTextChunk(part: ChatActivitySummaryPart, delta: string): void {
    if (!delta) return
    const chunks = this.pendingTextChunks.get(part.key)
    if (chunks) chunks.push(delta)
    else this.pendingTextChunks.set(part.key, [delta])
  }

  private discardPendingText(part: ChatActivitySummaryPart): void {
    this.pendingTextChunks.delete(part.key)
  }

  private commitPartTextChunks(part: ChatActivitySummaryPart): void {
    const chunks = this.pendingTextChunks.get(part.key)
    if (!chunks || chunks.length === 0) return
    this.pendingTextChunks.delete(part.key)
    const segment = chunks.length === 1 ? chunks[0]! : chunks.join('')
    if (!segment) return
    if (part.streamingTextChunks) part.streamingTextChunks.push(segment)
    else part.streamingTextChunks = [segment]
  }

  private commitPendingTextChunks(activity: ChatActivity): void {
    if (this.pendingTextChunks.size === 0) return
    for (const item of activity.items) {
      for (const part of item.parts) this.commitPartTextChunks(part)
    }
  }

  private commitAllPendingTextChunks(): void {
    if (this.pendingTextChunks.size === 0) return
    for (const activity of this.activities.values()) this.commitPendingTextChunks(activity)
  }

  private replacePartText(part: ChatActivitySummaryPart, text: string): void {
    this.discardPendingText(part)
    part.text = text
    delete part.streamingTextChunks
  }

  private finalizePartText(part: ChatActivitySummaryPart): void {
    this.commitPartTextChunks(part)
    finalizeChatActivityPartText(part)
  }

  private terminateOpenActivities(status: 'stopped' | 'disconnected'): void {
    this.flushPendingEvents()
    const now = this.now()
    for (const activity of this.activities.values()) {
      if (
        !this.visibleActivityKeys.has(activity.key)
        || isTerminalChatActivityStatus(activity.status)
      ) continue
      activity.status = status
      activity.updatedAt = Math.max(activity.updatedAt, now)
      activity.completedAt = now
      for (const item of activity.items) {
        if (!isTerminalChatActivityStatus(item.status)) item.status = status
        item.updatedAt = Math.max(item.updatedAt, now)
        item.completedAt = item.completedAt ?? now
        for (const part of item.parts) {
          this.finalizePartText(part)
          if (!isTerminalChatActivityStatus(part.status)) part.status = status
          part.updatedAt = Math.max(part.updatedAt, now)
          part.completedAt = part.completedAt ?? now
        }
      }
      this.notifyActivity(activity, true)
    }
    this.flushActivityNotifications()
  }

  private drainConsecutiveEvents(): void {
    if (this.lastAppliedSequence === null) return
    let nextSequence = this.lastAppliedSequence + 1
    while (this.pendingEvents.has(nextSequence)) {
      const event = this.pendingEvents.get(nextSequence)!
      this.pendingEvents.delete(nextSequence)
      this.applySequencedEvent(nextSequence, event)
      nextSequence += 1
    }
    if (this.pendingEvents.size === 0) this.clearReorderTimer()
  }

  private applySequencedEvent(sequence: number, event: ChatActivityEvent): void {
    this.applyEvent(event, sequence)
    this.lastAppliedSequence = sequence
    this.drainConsecutiveEvents()
  }

  private ensureActivity(
    responseId: string,
    payload: Record<string, unknown>,
    sequence: number | null,
  ): ChatActivity {
    const now = this.now()
    const existing = this.activities.get(responseId)
    if (existing) {
      this.applyReasoningConfiguration(existing, payload)
      if (sequence !== null) existing.lastSequenceNumber = sequence
      existing.updatedAt = Math.max(existing.updatedAt, now)
      return existing
    }
    const startedAt = payloadTimestamp(payload, now)
    const activity: ChatActivity = {
      key: chatActivityKey(responseId),
      responseId,
      status: 'pending',
      items: [],
      startedAt,
      updatedAt: startedAt,
    }
    if (this.defaultReasoningMode) activity.reasoningMode = this.defaultReasoningMode
    if (this.defaultReasoningEffort) activity.reasoningEffort = this.defaultReasoningEffort
    if (sequence !== null) activity.lastSequenceNumber = sequence
    this.applyReasoningConfiguration(activity, payload)
    this.activities.set(responseId, activity)
    return activity
  }

  private applyReasoningConfiguration(
    activity: ChatActivity,
    payload: Record<string, unknown>,
  ): void {
    const response = responseRecord(payload)
    const reasoning = asRecord(response?.reasoning ?? payload.reasoning)
    const mode = reasoningMode(
      payload.reasoning_mode
        ?? payload.reasoningMode
        ?? reasoning?.mode,
    )
    const effort = reasoningEffort(
      payload.reasoning_effort
        ?? payload.reasoningEffort
        ?? reasoning?.effort,
    )
    // The request has already passed the product capability gate. Treat its
    // configuration as authoritative so an unexpected upstream echo cannot
    // relabel a Standard request as Pro (or silently downgrade Pro in the UI).
    // Event metadata remains a fallback for restored/direct parser use where
    // no request configuration was supplied.
    if (!this.defaultReasoningMode && mode) activity.reasoningMode = mode
    if (
      activity.reasoningMode !== 'pro'
      && !this.defaultReasoningEffort
      && effort
    ) {
      activity.reasoningEffort = effort
    }
    enforceReasoningMetadataInvariant(activity)
  }

  private ensureItem(
    activity: ChatActivity,
    itemId: string,
    outputIndex: number,
    sequence: number | null,
  ): ChatActivityItem {
    const key = chatActivityItemKey(activity.responseId, itemId, outputIndex)
    const existing = activity.items.find((item) => item.key === key)
    const now = this.now()
    if (existing) {
      existing.updatedAt = Math.max(existing.updatedAt, now)
      if (sequence !== null) existing.lastSequenceNumber = sequence
      return existing
    }
    const item: ChatActivityItem = {
      key,
      itemId,
      outputIndex,
      status: 'pending',
      parts: [],
      startedAt: now,
      updatedAt: now,
    }
    if (sequence !== null) item.lastSequenceNumber = sequence
    activity.items.push(item)
    activity.items.sort((left, right) => (
      left.outputIndex - right.outputIndex || left.key.localeCompare(right.key)
    ))
    return item
  }

  private ensurePart(
    activity: ChatActivity,
    item: ChatActivityItem,
    summaryIndex: number,
    sequence: number | null,
  ): ChatActivitySummaryPart {
    const key = chatActivityPartKey(
      activity.responseId,
      item.itemId,
      item.outputIndex,
      summaryIndex,
    )
    const existing = item.parts.find((part) => part.key === key)
    const now = this.now()
    if (existing) {
      existing.updatedAt = Math.max(existing.updatedAt, now)
      if (sequence !== null) existing.lastSequenceNumber = sequence
      return existing
    }
    const part: ChatActivitySummaryPart = {
      key,
      itemId: item.itemId,
      outputIndex: item.outputIndex,
      summaryIndex,
      text: '',
      status: 'pending',
      startedAt: now,
      updatedAt: now,
    }
    if (sequence !== null) part.lastSequenceNumber = sequence
    item.parts.push(part)
    item.parts.sort((left, right) => (
      left.summaryIndex - right.summaryIndex || left.key.localeCompare(right.key)
    ))
    this.visibleActivityKeys.add(activity.key)
    return part
  }

  private markStreaming(
    activity: ChatActivity,
    item: ChatActivityItem,
    part?: ChatActivitySummaryPart,
  ): void {
    const now = this.now()
    if (!isTerminalChatActivityStatus(activity.status)) activity.status = 'streaming'
    if (!isTerminalChatActivityStatus(item.status)) item.status = 'streaming'
    if (part && !isTerminalChatActivityStatus(part.status)) part.status = 'streaming'
    activity.updatedAt = Math.max(activity.updatedAt, now)
    item.updatedAt = Math.max(item.updatedAt, now)
    if (part) part.updatedAt = Math.max(part.updatedAt, now)
  }

  private ingestSummaryArray(
    activity: ChatActivity,
    item: ChatActivityItem,
    itemRecord: Record<string, unknown>,
    sequence: number | null,
    completed: boolean,
  ): void {
    if (!Array.isArray(itemRecord.summary)) return
    for (let index = 0; index < itemRecord.summary.length; index += 1) {
      const summary = asRecord(itemRecord.summary[index])
      if (!summary || summary.type !== 'summary_text' || typeof summary.text !== 'string') continue
      const summaryIndex = nonNegativeInteger(summary.summary_index ?? summary.summaryIndex) ?? index
      const part = this.ensurePart(activity, item, summaryIndex, sequence)
      if (!this.authoritativeTextDoneKeys.has(part.key)) {
        this.replacePartText(part, summary.text)
      }
      if (completed) {
        part.status = 'completed'
        part.completedAt = this.now()
      }
      this.markStreaming(activity, item, part)
    }
  }

  private ingestTerminalReasoningOutput(
    activity: ChatActivity,
    response: Record<string, unknown> | null,
    sequence: number | null,
  ): void {
    if (!Array.isArray(response?.output)) return
    for (let outputIndex = 0; outputIndex < response.output.length; outputIndex += 1) {
      const itemRecord = asRecord(response.output[outputIndex])
      if (!itemRecord || itemRecord.type !== 'reasoning') continue
      const itemId = nonEmptyString(itemRecord.id)
      if (!itemId || !Array.isArray(itemRecord.summary)) continue

      const summaries: Array<{
        summaryIndex: number
        text: string
      }> = []
      for (let index = 0; index < itemRecord.summary.length; index += 1) {
        const summary = asRecord(itemRecord.summary[index])
        if (!summary || summary.type !== 'summary_text' || typeof summary.text !== 'string') {
          continue
        }
        summaries.push({
          summaryIndex: nonNegativeInteger(summary.summary_index ?? summary.summaryIndex) ?? index,
          text: summary.text,
        })
      }
      if (summaries.length === 0) continue

      const itemKey = chatActivityItemKey(activity.responseId, itemId, outputIndex)
      this.outputItemTypes.set(itemKey, 'reasoning')
      const item = this.ensureItem(activity, itemId, outputIndex, sequence)
      for (const summary of summaries) {
        const part = this.ensurePart(activity, item, summary.summaryIndex, sequence)
        // The terminal response is the complete canonical snapshot. It may
        // correct a lifecycle text.done payload, so it supersedes both queued
        // deltas and earlier authoritative text for the same stable part key.
        this.replacePartText(part, summary.text)
        this.authoritativeTextDoneKeys.add(part.key)
      }
    }
  }

  private notifyActivity(activity: ChatActivity, immediate: boolean): void {
    if (!this.visibleActivityKeys.has(activity.key)) return
    if (!this.onActivity) return
    if (immediate) {
      this.pendingActivityNotifications.delete(activity.key)
      if (this.pendingActivityNotifications.size === 0) this.clearNotificationTimer()
      this.commitPendingTextChunks(activity)
      this.onActivity(cloneChatActivity(activity))
      return
    }
    this.pendingActivityNotifications.set(activity.key, activity)
    this.scheduleActivityNotification()
  }

  private applyEvent(event: ChatActivityEvent, sequence: number | null): void {
    const payload = asRecord(event.payload)
    if (!payload) return
    const responseId = responseIdFromPayload(payload, this.currentResponseId)

    if (event.eventType === 'response.created') {
      if (!responseId) return
      this.currentResponseId = responseId
      this.ensureActivity(responseId, payload, sequence)
      return
    }

    if (!responseId) return
    this.currentResponseId = responseId
    const activity = this.ensureActivity(responseId, payload, sequence)
    if (sequence !== null) activity.lastSequenceNumber = sequence

    if (
      event.eventType === 'response.completed'
      || event.eventType === 'response.incomplete'
      || event.eventType === 'response.failed'
    ) {
      const status: ChatActivityStatus = event.eventType === 'response.completed'
        ? 'completed'
        : event.eventType === 'response.incomplete'
          ? 'incomplete'
          : 'failed'
      const response = responseRecord(payload)
      this.ingestTerminalReasoningOutput(activity, response, sequence)
      const now = this.now()
      if (status === 'failed') {
        const error = normalizedError(response?.error ?? payload.error)
        if (error) activity.error = error
      }
      for (const item of activity.items) {
        for (const part of item.parts) {
          this.finalizePartText(part)
          if (status !== 'completed' || !isTerminalChatActivityStatus(part.status)) {
            part.status = status
          }
          part.updatedAt = Math.max(part.updatedAt, now)
          part.completedAt = part.completedAt ?? now
        }
        const itemTerminal = status === 'completed' && isTerminalChatActivityStatus(item.status)
          ? item.status
          : status
        item.status = item.parts.length > 0
          ? aggregateStatuses([itemTerminal, ...item.parts.map((part) => part.status)])
          : itemTerminal
        item.updatedAt = Math.max(item.updatedAt, now)
        item.completedAt = item.completedAt ?? now
      }
      const activityTerminal = status === 'completed'
        && isTerminalChatActivityStatus(activity.status)
        ? activity.status
        : status
      activity.status = activity.items.length > 0
        ? aggregateStatuses([activityTerminal, ...activity.items.map((item) => item.status)])
        : activityTerminal
      activity.updatedAt = Math.max(activity.updatedAt, now)
      activity.completedAt = now
      this.notifyActivity(activity, true)
      return
    }

    if (
      event.eventType === 'response.output_item.added'
      || event.eventType === 'response.output_item.done'
    ) {
      const identity = itemIdentity(payload)
      if (!identity.item || !identity.itemId || identity.outputIndex === null) return
      const itemKey = chatActivityItemKey(responseId, identity.itemId, identity.outputIndex)
      const itemType = nonEmptyString(identity.item.type)
      if (itemType) this.outputItemTypes.set(itemKey, itemType)
      if (itemType !== 'reasoning') return
      const item = this.ensureItem(
        activity,
        identity.itemId,
        identity.outputIndex,
        sequence,
      )
      const completed = event.eventType === 'response.output_item.done'
      this.ingestSummaryArray(activity, item, identity.item, sequence, completed)
      if (completed) {
        const now = this.now()
        item.status = 'completed'
        item.completedAt = now
        item.updatedAt = Math.max(item.updatedAt, now)
        for (const part of item.parts) {
          this.finalizePartText(part)
          if (!isTerminalChatActivityStatus(part.status)) part.status = 'completed'
          part.completedAt = part.completedAt ?? now
          part.updatedAt = Math.max(part.updatedAt, now)
        }
      } else {
        this.markStreaming(activity, item)
      }
      this.notifyActivity(activity, true)
      return
    }

    const identity = summaryIdentity(payload)
    if (!identity.itemId || identity.outputIndex === null || identity.summaryIndex === null) return
    const itemKey = chatActivityItemKey(responseId, identity.itemId, identity.outputIndex)
    if (this.outputItemTypes.get(itemKey) !== 'reasoning') return
    const partRecord = asRecord(payload.part)
    if (partRecord?.type !== undefined && partRecord.type !== 'summary_text') return
    if (
      (event.eventType === 'response.reasoning_summary_part.added'
        || event.eventType === 'response.reasoning_summary_part.done')
      && partRecord?.type !== 'summary_text'
    ) return
    const item = this.ensureItem(activity, identity.itemId, identity.outputIndex, sequence)
    const part = this.ensurePart(activity, item, identity.summaryIndex, sequence)

    if (event.eventType === 'response.reasoning_summary_part.added') {
      if (
        !this.authoritativeTextDoneKeys.has(part.key)
        && typeof partRecord?.text === 'string'
      ) {
        this.replacePartText(part, partRecord.text)
      }
      this.markStreaming(activity, item, part)
      this.notifyActivity(activity, true)
      return
    }

    if (event.eventType === 'response.reasoning_summary_text.delta') {
      if (typeof payload.delta !== 'string') return
      if (this.authoritativeTextDoneKeys.has(part.key)) return
      this.appendTextChunk(part, payload.delta)
      this.markStreaming(activity, item, part)
      this.notifyActivity(activity, false)
      return
    }

    if (event.eventType === 'response.reasoning_summary_text.done') {
      if (typeof payload.text !== 'string') return
      this.replacePartText(part, payload.text)
      part.status = 'completed'
      part.completedAt = this.now()
      this.authoritativeTextDoneKeys.add(part.key)
      this.markStreaming(activity, item, part)
      this.notifyActivity(activity, true)
      return
    }

    if (event.eventType === 'response.reasoning_summary_part.done') {
      if (
        !this.authoritativeTextDoneKeys.has(part.key)
        && typeof partRecord?.text === 'string'
      ) {
        this.replacePartText(part, partRecord.text)
      } else {
        this.finalizePartText(part)
      }
      const partDoneStatus = activityStatus(payload.status ?? partRecord?.status, 'completed')
      part.status = partDoneStatus === 'pending' || partDoneStatus === 'streaming'
        ? 'completed'
        : partDoneStatus
      part.completedAt = this.now()
      part.updatedAt = Math.max(part.updatedAt, part.completedAt)
      this.markStreaming(activity, item, part)
      this.notifyActivity(activity, true)
    }
  }
}

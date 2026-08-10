/**
 * Admin Usage API endpoints
 * Handles admin-level usage logs and statistics retrieval
 */

import { apiClient } from '../client'
import type {
  AdminUsageLog,
  UsageQueryParams,
  PaginatedResponse,
  UsageRequestType,
  UsageSource,
} from '@/types'
import type { EndpointStat } from '@/types'

// ==================== Types ====================

export interface AdminUsageStatsResponse {
  total_requests: number
  total_input_tokens: number
  total_output_tokens: number
  total_cache_tokens: number
  total_cache_creation_tokens: number
  total_cache_read_tokens: number
  total_tokens: number
  total_cost: number
  total_actual_cost: number
  total_account_cost: number
  average_duration_ms: number
  endpoints?: EndpointStat[]
  upstream_endpoints?: EndpointStat[]
  endpoint_paths?: EndpointStat[]
}

export interface SimpleUser {
  id: number
  email: string
  deleted: boolean
}

export interface SimpleApiKey {
  id: number
  name: string
  user_id: number
}

export interface UsageCleanupFilters {
  start_time: string
  end_time: string
  user_id?: number
  api_key_id?: number
  account_id?: number
  group_id?: number
  model?: string | null
  request_type?: UsageRequestType | null
  stream?: boolean | null
  billing_type?: number | null
}

export interface UsageCleanupTask {
  id: number
  status: string
  filters: UsageCleanupFilters
  created_by: number
  deleted_rows: number
  error_message?: string | null
  canceled_by?: number | null
  canceled_at?: string | null
  started_at?: string | null
  finished_at?: string | null
  created_at: string
  updated_at: string
}

export interface CreateUsageCleanupTaskRequest {
  start_date: string
  end_date: string
  user_id?: number
  api_key_id?: number
  account_id?: number
  group_id?: number
  model?: string | null
  request_type?: UsageRequestType | null
  stream?: boolean | null
  billing_type?: number | null
  timezone?: string
}

export interface AdminUsageQueryParams extends UsageQueryParams {
  user_id?: number
  exact_total?: boolean
  billing_mode?: string
  sort_by?: string
  sort_order?: 'asc' | 'desc'
  // 错误请求 tab 专属筛选(仅传给错误列表接口;共用同一 filters 对象)
  error_phase?: string | null
  error_category?: string | null
  status_code?: number | null
}

export interface BillingReceiptTokenSummary {
  input_tokens: number
  output_tokens: number
  cache_tokens: number
  cache_read_tokens?: number
  cache_creation_tokens?: number
}

export interface AdminBillingReceipt {
  id: number | string
  /** Stable UI identity. Attempts without a persisted receipt use their request ID. */
  row_key: string
  /** Empty when the request never produced a persisted billing receipt. */
  receipt_id: string
  request_id?: string | null
  user_id: number
  user?: {
    id?: number
    username?: string | null
    email?: string
    deleted?: boolean
  } | null
  username?: string | null
  user_username?: string | null
  user_email?: string | null
  source?: UsageSource | null
  requested_model?: string | null
  actual_model: string
  model?: string | null
  tokens: BillingReceiptTokenSummary
  gross_cost: number
  charged_amount: number
  balance_before?: number | null
  balance_after?: number | null
  status: string
  overdraft?: boolean
  failure_code?: string | null
  failure_reason?: string | null
  created_at: string
}

export interface BillingReceiptQueryParams {
  page?: number
  page_size?: number
  user_id?: number
  model?: string
  receipt_id?: string
  status?: string
  source?: UsageSource
  start_date?: string
  end_date?: string
}

// ==================== API Functions ====================

/**
 * List all usage logs with optional filters (admin only)
 * @param params - Query parameters for filtering and pagination
 * @returns Paginated list of usage logs
 */
export async function list(
  params: AdminUsageQueryParams,
  options?: { signal?: AbortSignal }
): Promise<PaginatedResponse<AdminUsageLog>> {
  const { data } = await apiClient.get<PaginatedResponse<AdminUsageLog>>('/admin/usage', {
    params,
    signal: options?.signal
  })
  return data
}

function asFiniteNumber(value: unknown, fallback = 0): number {
  const parsed = typeof value === 'number' ? value : Number(value)
  return Number.isFinite(parsed) ? parsed : fallback
}

function normalizeBillingReceipt(raw: Record<string, any>, index = 0): AdminBillingReceipt {
  const tokenData = raw.tokens && typeof raw.tokens === 'object' ? raw.tokens : {}
  const id = raw.id ?? raw.receipt_id ?? ''
  const rawReceiptId = raw.receipt_id ?? id
  const receiptIdValue = rawReceiptId == null ? '' : String(rawReceiptId).trim()
  // The backend emits id=0 for request attempts that have no persisted ledger row.
  // Keep that absence explicit instead of presenting a fictional "receipt 0".
  const receiptId = receiptIdValue === '0' ? '' : receiptIdValue
  const requestId = raw.request_id == null ? null : String(raw.request_id)
  const createdAt = String(raw.created_at ?? '')
  const rowKey = receiptId
    ? `receipt:${receiptId}`
    : requestId
      ? `request:${requestId}`
      : `attempt:${asFiniteNumber(raw.user_id ?? raw.user?.id)}:${createdAt}:${index}`
  const requestedModel = raw.requested_model ?? raw.model ?? null
  const actualModel = raw.actual_model ?? raw.upstream_model ?? raw.model ?? requestedModel ?? ''

  return {
    id,
    row_key: rowKey,
    receipt_id: receiptId,
    request_id: requestId,
    user_id: asFiniteNumber(raw.user_id ?? raw.user?.id),
    user: raw.user ?? null,
    username: raw.username ?? null,
    user_username: raw.user_username ?? null,
    user_email: raw.user_email ?? raw.email ?? raw.user?.email ?? null,
    source: raw.source ?? 'web_chat',
    requested_model: requestedModel,
    actual_model: String(actualModel),
    model: raw.model ?? requestedModel,
    tokens: {
      input_tokens: asFiniteNumber(raw.input_tokens ?? tokenData.input_tokens ?? tokenData.input),
      output_tokens: asFiniteNumber(raw.output_tokens ?? tokenData.output_tokens ?? tokenData.output),
      cache_tokens: asFiniteNumber(
        raw.cache_tokens
          ?? tokenData.cache_tokens
          ?? tokenData.cache
          ?? (
            asFiniteNumber(raw.cache_read_tokens ?? tokenData.cache_read_tokens)
            + asFiniteNumber(raw.cache_creation_tokens ?? tokenData.cache_creation_tokens)
          )
      ),
      cache_read_tokens: asFiniteNumber(raw.cache_read_tokens ?? tokenData.cache_read_tokens),
      cache_creation_tokens: asFiniteNumber(raw.cache_creation_tokens ?? tokenData.cache_creation_tokens),
    },
    gross_cost: asFiniteNumber(
      raw.gross_cost
        ?? raw.gross_amount
        ?? raw.total_cost
        ?? raw.actual_cost
        ?? raw.charged_amount
    ),
    charged_amount: asFiniteNumber(
      raw.charged_amount ?? raw.actual_cost ?? raw.gross_cost ?? raw.total_cost
    ),
    balance_before: raw.balance_before == null ? null : asFiniteNumber(raw.balance_before),
    balance_after: raw.balance_after == null ? null : asFiniteNumber(raw.balance_after),
    status: String(raw.status ?? raw.billing_status ?? 'pending'),
    overdraft: Boolean(raw.overdraft),
    failure_code: raw.failure_code ?? null,
    failure_reason: raw.failure_reason ?? raw.error_message ?? raw.billing_error ?? null,
    created_at: createdAt,
  }
}

/**
 * List request-level Web Chat billing receipts (admin only).
 */
export async function listBillingReceipts(
  params: BillingReceiptQueryParams,
  options?: { signal?: AbortSignal }
): Promise<PaginatedResponse<AdminBillingReceipt>> {
  const { data } = await apiClient.get<
    PaginatedResponse<Record<string, any>> | {
      data?: PaginatedResponse<Record<string, any>>
      receipts?: Record<string, any>[]
      items?: Record<string, any>[]
      total?: number
      page?: number
      page_size?: number
      pages?: number
    }
  >('/admin/billing/receipts', {
    params,
    signal: options?.signal,
  })

  const outer = data as any
  const payload = outer?.data && typeof outer.data === 'object' ? outer.data : outer
  const items = Array.isArray(payload?.items)
    ? payload.items
    : (Array.isArray(payload?.receipts) ? payload.receipts : [])
  const page = asFiniteNumber(payload?.page, params.page ?? 1)
  const pageSize = asFiniteNumber(payload?.page_size, params.page_size ?? 20)
  const total = asFiniteNumber(payload?.total, items.length)

  return {
    items: items.map((item: Record<string, any>, index: number) => normalizeBillingReceipt(item, index)),
    total,
    page,
    page_size: pageSize,
    pages: asFiniteNumber(payload?.pages, pageSize > 0 ? Math.ceil(total / pageSize) : 0),
  }
}

/**
 * Get usage statistics with optional filters (admin only)
 * @param params - Query parameters for filtering
 * @returns Usage statistics
 */
export async function getStats(params: {
  user_id?: number
  api_key_id?: number
  account_id?: number
  group_id?: number
  model?: string
  request_type?: UsageRequestType
  stream?: boolean
  period?: string
  start_date?: string
  end_date?: string
  timezone?: string
  nocache?: number
}): Promise<AdminUsageStatsResponse> {
  const { data } = await apiClient.get<AdminUsageStatsResponse>('/admin/usage/stats', {
    params
  })
  return data
}

/**
 * Search users by email keyword (admin only)
 * @param keyword - Email keyword to search
 * @returns List of matching users (max 30)
 */
export async function searchUsers(keyword: string): Promise<SimpleUser[]> {
  const { data } = await apiClient.get<SimpleUser[]>('/admin/usage/search-users', {
    params: { q: keyword }
  })
  return data
}

/**
 * Search API keys by user ID and/or keyword (admin only)
 * @param userId - Optional user ID to filter by
 * @param keyword - Optional keyword to search in key name
 * @returns List of matching API keys (max 30)
 */
export async function searchApiKeys(userId?: number, keyword?: string): Promise<SimpleApiKey[]> {
  const params: Record<string, unknown> = {}
  if (userId !== undefined) {
    params.user_id = userId
  }
  if (keyword) {
    params.q = keyword
  }
  const { data } = await apiClient.get<SimpleApiKey[]>('/admin/usage/search-api-keys', {
    params
  })
  return data
}

/**
 * List usage cleanup tasks (admin only)
 * @param params - Query parameters for pagination
 * @returns Paginated list of cleanup tasks
 */
export async function listCleanupTasks(
  params: { page?: number; page_size?: number },
  options?: { signal?: AbortSignal }
): Promise<PaginatedResponse<UsageCleanupTask>> {
  const { data } = await apiClient.get<PaginatedResponse<UsageCleanupTask>>('/admin/usage/cleanup-tasks', {
    params,
    signal: options?.signal
  })
  return data
}

/**
 * Create a usage cleanup task (admin only)
 * @param payload - Cleanup task parameters
 * @returns Created cleanup task
 */
export async function createCleanupTask(payload: CreateUsageCleanupTaskRequest): Promise<UsageCleanupTask> {
  const { data } = await apiClient.post<UsageCleanupTask>('/admin/usage/cleanup-tasks', payload)
  return data
}

/**
 * Cancel a usage cleanup task (admin only)
 * @param taskId - Task ID to cancel
 */
export async function cancelCleanupTask(taskId: number): Promise<{ id: number; status: string }> {
  const { data } = await apiClient.post<{ id: number; status: string }>(
    `/admin/usage/cleanup-tasks/${taskId}/cancel`
  )
  return data
}

export const adminUsageAPI = {
  list,
  listBillingReceipts,
  getStats,
  searchUsers,
  searchApiKeys,
  listCleanupTasks,
  createCleanupTask,
  cancelCleanupTask
}

export default adminUsageAPI

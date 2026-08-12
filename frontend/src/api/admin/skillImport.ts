/**
 * Durable Skill collection sources, schedules, and execution runs.
 *
 * Runs are intentionally split from their items and events so the 2-second
 * status poll remains small even when a catalog contains thousands of Skills.
 */
import { apiClient } from '../client'

export type SkillImportAdapter = 'skills_sh' | 'github' | 'well_known' | 'manifest'
export type SkillImportMode = 'review' | 'auto_publish' | 'dry_run'
export type SkillImportPublishPolicy = 'review' | 'auto_publish'
export type SkillImportMetadataPolicy = 'create_only' | 'refresh'

/** Includes the canonical backend values and accepted aliases during rollout. */
export type SkillImportRunStatus =
  | 'queued'
  | 'discovering'
  | 'preparing'
  | 'running'
  | 'waiting_retry'
  | 'ready'
  | 'awaiting_review'
  | 'publishing'
  | 'succeeded'
  | 'partial'
  | 'partial_succeeded'
  | 'failed'
  | 'canceled'
  | 'cancelled'

export type SkillImportItemStatus =
  | 'pending'
  | 'fetching'
  | 'prepared'
  | 'queued'
  | 'processing'
  | 'ready'
  | 'unchanged'
  | 'blocked'
  | 'failed'
  | 'published'
  | 'skipped'
  | 'canceled'
  | 'cancelled'

export interface SkillImportSource {
  id: number
  name: string
  adapter: SkillImportAdapter
  namespace: string
  base_url: string
  source_config: Record<string, unknown>
  catalog_priority: number
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface CreateSkillImportSourceRequest {
  name: string
  adapter: SkillImportAdapter
  namespace: string
  base_url: string
  source_config: Record<string, unknown>
  catalog_priority: number
  enabled: boolean
}

export type UpdateSkillImportSourceRequest = CreateSkillImportSourceRequest

export interface SkillImportAutoPublishGate {
  require_all_valid?: boolean
  max_blocked_items?: number
  max_failed_items?: number
  allow_license_unverified?: boolean
}

export interface SkillImportRunConfig {
  safe_gate?: boolean
  concurrency?: number
  auto_publish_gate?: SkillImportAutoPublishGate
  [key: string]: unknown
}

export interface SkillImportSelection extends Record<string, unknown> {
  start_rank: number
  limit: number
  page_size?: number
}

export interface SkillImportSchedule {
  id: number
  source_id: number
  source?: SkillImportSource
  name: string
  enabled: boolean
  cron_expression: string
  timezone: string
  selection: SkillImportSelection
  run_config: SkillImportRunConfig
  publish_policy: SkillImportPublishPolicy
  metadata_policy: SkillImportMetadataPolicy
  next_run_at?: string | null
  last_run_at?: string | null
  last_run_id?: number | null
  created_at: string
  updated_at: string
}

export interface CreateSkillImportScheduleRequest {
  source_id: number
  name: string
  enabled: boolean
  cron_expression: string
  timezone: string
  selection: SkillImportSelection
  run_config: SkillImportRunConfig
  publish_policy: SkillImportPublishPolicy
  metadata_policy: SkillImportMetadataPolicy
}

export type UpdateSkillImportScheduleRequest = CreateSkillImportScheduleRequest

export interface SkillImportRunCounts {
  requested: number
  discovered: number
  prepared: number
  created: number
  updated: number
  unchanged: number
  skipped: number
  blocked: number
  failed: number
  published: number
}

export interface SkillImportRunProgress {
  run_id: number
  status: SkillImportRunStatus
  counts: SkillImportRunCounts
  total_items: number
  percent: number
  updated_at: string
}

export interface SkillImportRun {
  id: number
  source_id: number
  source?: SkillImportSource
  schedule_id?: number | null
  schedule?: SkillImportSchedule
  parent_run_id?: number | null
  trigger_type: 'manual' | 'scheduled' | 'retry' | 'bootstrap' | string
  mode: SkillImportMode
  status: SkillImportRunStatus
  request_config: Record<string, unknown>
  snapshot?: Record<string, unknown>
  snapshot_sha256?: string
  scheduled_for?: string | null
  counts: SkillImportRunCounts
  progress?: SkillImportRunProgress
  cancel_requested_at?: string | null
  attempt_count: number
  next_attempt_at?: string | null
  last_error_code?: string
  last_error_message?: string
  started_at?: string | null
  finished_at?: string | null
  created_at: string
  updated_at: string
}

export interface SkillImportStableKey {
  source_id: number
  namespace: string
  external_id: string
}

export interface SkillImportValidationIssue {
  code: string
  message: string
  path?: string
}

export interface SkillImportRunItem {
  id: number
  run_id: number
  stable_key: SkillImportStableKey
  rank?: number | null
  market_slug: string
  status: SkillImportItemStatus
  upstream_name: string
  origin_url: string
  source_revision?: string
  source_content_sha256?: string
  package_sha256?: string
  desired_skill?: {
    slug: string
    display_name: string
    summary: string
    category: string
    catalog_source_priority?: number
    catalog_source_rank?: number | null
  }
  validation_report?: {
    valid: boolean
    errors: SkillImportValidationIssue[]
    warnings: SkillImportValidationIssue[]
  }
  license_unverified: boolean
  excluded_files: string[]
  warnings: SkillImportValidationIssue[]
  skill_id?: number | null
  version_id?: number | null
  attempt_count: number
  next_attempt_at?: string | null
  error_code?: string
  error_message?: string
  started_at?: string | null
  completed_at?: string | null
  created_at: string
  updated_at: string
}

export interface SkillImportEvent {
  id: number
  run_id: number
  run_item_id?: number | null
  level: 'debug' | 'info' | 'warn' | 'warning' | 'error'
  event_type: string
  message: string
  payload?: Record<string, unknown>
  created_at: string
}

export interface SkillImportListParams {
  page?: number
  page_size?: number
  status?: string
  search?: string
  source_id?: number
}

export interface SkillImportItemListParams extends SkillImportListParams {
  status?: SkillImportItemStatus | 'all'
}

export interface SkillImportListResponse<T> {
  items: T[]
  total: number
  page: number
  page_size: number
  pages: number
}

export interface CreateSkillImportRunRequest {
  source_id: number
  mode: SkillImportMode
  selection: SkillImportSelection
  run_config: SkillImportRunConfig
  publish_policy?: SkillImportPublishPolicy
  metadata_policy?: SkillImportMetadataPolicy
}

export interface UploadSkillImportRunRequest extends CreateSkillImportRunRequest {
  file: File
}

export interface PublishSkillImportRunRequest {
  item_ids?: number[]
}

export interface PublishSkillImportRunResult {
  run_id: number
  status: SkillImportRunStatus
  items: Array<{ run_item_id: number; skill_id: number; version_id: number }>
  counts: SkillImportRunCounts
  published_at: string
}

type ListPayload<T> =
  | T[]
  | SkillImportListResponse<T>
  | { items?: T[]; total?: number; page?: number; page_size?: number; pages?: number }

function normalizeList<T>(payload: ListPayload<T>, params?: SkillImportListParams): SkillImportListResponse<T> {
  if (Array.isArray(payload)) {
    const pageSize = params?.page_size ?? Math.max(payload.length, 1)
    return { items: payload, total: payload.length, page: params?.page ?? 1, page_size: pageSize, pages: 1 }
  }
  const items = payload.items ?? []
  const total = payload.total ?? items.length
  const pageSize = payload.page_size ?? params?.page_size ?? Math.max(items.length, 1)
  return {
    items,
    total,
    page: payload.page ?? params?.page ?? 1,
    page_size: pageSize,
    pages: payload.pages ?? Math.max(1, Math.ceil(total / pageSize)),
  }
}

function mutationConfig(idempotencyKey: string) {
  return { headers: { 'Idempotency-Key': idempotencyKey } }
}

export async function listSkillImportSources(
  params?: SkillImportListParams,
): Promise<SkillImportListResponse<SkillImportSource>> {
  const { data } = await apiClient.get<ListPayload<SkillImportSource>>('/admin/skill-import/sources', { params })
  return normalizeList(data, params)
}

export async function createSkillImportSource(
  request: CreateSkillImportSourceRequest,
  idempotencyKey: string,
): Promise<SkillImportSource> {
  const { data } = await apiClient.post<SkillImportSource>(
    '/admin/skill-import/sources',
    request,
    mutationConfig(idempotencyKey),
  )
  return data
}

export async function updateSkillImportSource(
  id: number,
  request: UpdateSkillImportSourceRequest,
): Promise<SkillImportSource> {
  const { data } = await apiClient.put<SkillImportSource>(`/admin/skill-import/sources/${id}`, request)
  return data
}

export async function listSkillImportSchedules(
  params?: SkillImportListParams,
): Promise<SkillImportListResponse<SkillImportSchedule>> {
  const { data } = await apiClient.get<ListPayload<SkillImportSchedule>>('/admin/skill-import/schedules', { params })
  return normalizeList(data, params)
}

export async function createSkillImportSchedule(
  request: CreateSkillImportScheduleRequest,
  idempotencyKey: string,
): Promise<SkillImportSchedule> {
  const { data } = await apiClient.post<SkillImportSchedule>(
    '/admin/skill-import/schedules',
    request,
    mutationConfig(idempotencyKey),
  )
  return data
}

export async function updateSkillImportSchedule(
  id: number,
  request: UpdateSkillImportScheduleRequest,
): Promise<SkillImportSchedule> {
  const { data } = await apiClient.put<SkillImportSchedule>(`/admin/skill-import/schedules/${id}`, request)
  return data
}

export async function runSkillImportSchedule(id: number, idempotencyKey: string): Promise<SkillImportRun> {
  const { data } = await apiClient.post<SkillImportRun>(
    `/admin/skill-import/schedules/${id}/run`,
    {},
    mutationConfig(idempotencyKey),
  )
  return data
}

export async function listSkillImportRuns(
  params?: SkillImportListParams,
): Promise<SkillImportListResponse<SkillImportRun>> {
  const { data } = await apiClient.get<ListPayload<SkillImportRun>>('/admin/skill-import/runs', { params })
  return normalizeList(data, params)
}

export async function createSkillImportRun(
  request: CreateSkillImportRunRequest,
  idempotencyKey: string,
): Promise<SkillImportRun> {
  const { data } = await apiClient.post<SkillImportRun>(
    '/admin/skill-import/runs',
    request,
    mutationConfig(idempotencyKey),
  )
  return data
}

export async function uploadSkillImportRun(
  request: UploadSkillImportRunRequest,
  idempotencyKey: string,
): Promise<SkillImportRun> {
  const formData = new FormData()
  formData.append('file', request.file)
  formData.append('source_id', String(request.source_id))
  formData.append('mode', request.mode)
  formData.append('selection', JSON.stringify(request.selection))
  formData.append('run_config', JSON.stringify(request.run_config))
  if (request.publish_policy) formData.append('publish_policy', request.publish_policy)
  if (request.metadata_policy) formData.append('metadata_policy', request.metadata_policy)
  const { data } = await apiClient.post<SkillImportRun>(
    '/admin/skill-import/runs/upload',
    formData,
    {
      headers: {
        'Idempotency-Key': idempotencyKey,
      },
    },
  )
  return data
}

export async function getSkillImportRun(id: number): Promise<SkillImportRun> {
  const { data } = await apiClient.get<SkillImportRun | { run: SkillImportRun; progress?: SkillImportRunProgress }>(
    `/admin/skill-import/runs/${id}`,
  )
  if ('run' in data) return { ...data.run, progress: data.progress }
  return data
}

export async function listSkillImportItems(
  runID: number,
  params?: SkillImportItemListParams,
): Promise<SkillImportListResponse<SkillImportRunItem>> {
  const { data } = await apiClient.get<ListPayload<SkillImportRunItem>>(
    `/admin/skill-import/runs/${runID}/items`,
    { params: { ...params, status: params?.status === 'all' ? undefined : params?.status } },
  )
  return normalizeList(data, params)
}

export async function listSkillImportEvents(
  runID: number,
  params?: SkillImportListParams,
): Promise<SkillImportListResponse<SkillImportEvent>> {
  const { data } = await apiClient.get<ListPayload<SkillImportEvent>>(
    `/admin/skill-import/runs/${runID}/events`,
    { params },
  )
  return normalizeList(data, params)
}

async function mutateRun(
  id: number,
  action: 'cancel' | 'retry-failed',
  idempotencyKey: string,
): Promise<SkillImportRun> {
  const { data } = await apiClient.post<SkillImportRun>(
    `/admin/skill-import/runs/${id}/${action}`,
    {},
    mutationConfig(idempotencyKey),
  )
  return data
}

export function cancelSkillImportRun(id: number, idempotencyKey: string): Promise<SkillImportRun> {
  return mutateRun(id, 'cancel', idempotencyKey)
}

export function retryFailedSkillImportRun(id: number, idempotencyKey: string): Promise<SkillImportRun> {
  return mutateRun(id, 'retry-failed', idempotencyKey)
}

export async function publishSkillImportRun(
  id: number,
  request: PublishSkillImportRunRequest,
  idempotencyKey: string,
): Promise<PublishSkillImportRunResult> {
  const { data } = await apiClient.post<PublishSkillImportRunResult>(
    `/admin/skill-import/runs/${id}/publish`,
    request,
    mutationConfig(idempotencyKey),
  )
  return data
}

export const skillImportAPI = {
  listSources: listSkillImportSources,
  createSource: createSkillImportSource,
  updateSource: updateSkillImportSource,
  listSchedules: listSkillImportSchedules,
  createSchedule: createSkillImportSchedule,
  updateSchedule: updateSkillImportSchedule,
  runSchedule: runSkillImportSchedule,
  listRuns: listSkillImportRuns,
  createRun: createSkillImportRun,
  uploadRun: uploadSkillImportRun,
  getRun: getSkillImportRun,
  listItems: listSkillImportItems,
  listEvents: listSkillImportEvents,
  cancelRun: cancelSkillImportRun,
  retryFailed: retryFailedSkillImportRun,
  publishRun: publishSkillImportRun,
}

export default skillImportAPI

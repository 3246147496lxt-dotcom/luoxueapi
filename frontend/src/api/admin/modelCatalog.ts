/**
 * Admin model catalog API.
 *
 * Catalog records are deliberately separate from channel configuration: channel
 * pricing remains the source of truth, while this resource only controls the
 * public presentation and publication lifecycle.
 */
import { apiClient } from '../client'

export type ModelCatalogStatus = 'draft' | 'published' | 'archived'

export interface ModelCatalogPricingPreview {
  label: string
  billing_mode: 'token' | 'per_request' | 'image' | string
  currency: string
  unit: string
  input_price: number | null
  output_price: number | null
  cache_write_price: number | null
  cache_read_price: number | null
  image_input_price: number | null
  image_output_price: number | null
  per_request_price: number | null
  intervals: Array<{
    min_tokens: number
    max_tokens: number | null
    tier_label: string
    input_price: number | null
    output_price: number | null
    cache_write_price: number | null
    cache_read_price: number | null
    per_request_price: number | null
  }>
  peak_rate: {
    enabled: boolean
    start: string | null
    end: string | null
    multiplier: number | null
  }
}

export interface ModelCatalogGroupOption {
  id: number
  name: string
  platform: string
  channel_id: number
  channel_name: string
  rate_multiplier: number
  peak_rate_enabled: boolean
  peak_start: string | null
  peak_end: string | null
  peak_rate_multiplier: number | null
  pricing: ModelCatalogPricingPreview
}

export interface ModelCatalogPublicGroup {
  id: number
  name: string
  platform: string
  rate_multiplier: number
  peak_rate_enabled: boolean
  peak_start: string | null
  peak_end: string | null
  peak_rate_multiplier: number | null
}

export interface ModelCatalogValidation {
  valid: boolean
  errors: string[]
}

export interface AdminModelCatalogModel {
  id: number
  slug: string
  model: string
  platform: string
  metadata_model_id: string | null
  display_name_zh: string
  display_name_en: string
  summary_zh: string
  summary_en: string
  provider: string
  logo_key: string
  category: string
  tags: string[]
  capabilities: string[]
  context_window: number | null
  max_output_tokens: number | null
  public_group_id: number | null
  public_group: ModelCatalogPublicGroup | null
  featured: boolean
  sort_order: number
  status: ModelCatalogStatus
  pricing?: ModelCatalogPricingPreview | null
  validation: ModelCatalogValidation
  published_at: string | null
  created_at: string
  updated_at: string
}

export interface ModelCatalogCandidate {
  model: string
  platform: string
  provider: string
  logo_key: string
  category: string
  context_window: number | null
  max_output_tokens: number | null
  capabilities: string[]
  group_options: ModelCatalogGroupOption[]
}

export interface ModelCatalogListResponse {
  items: AdminModelCatalogModel[]
  total: number
}

export interface ModelCatalogCandidateResponse {
  items: ModelCatalogCandidate[]
  total: number
}

export interface CreateModelCatalogRequest {
  slug?: string
  model: string
  platform: string
  metadata_model_id?: string | null
  display_name_zh?: string
  display_name_en?: string
  summary_zh?: string
  summary_en?: string
  provider?: string
  logo_key?: string
  category?: string
  tags?: string[]
  capabilities?: string[]
  context_window?: number | null
  max_output_tokens?: number | null
  public_group_id?: number | null
  featured?: boolean
  sort_order?: number
}

// PUT is a full replacement on the backend. Keep the required identity fields
// in the client contract so callers cannot accidentally clear omitted fields.
export type UpdateModelCatalogRequest = CreateModelCatalogRequest

function normalizeList<T>(payload: T[] | { items?: T[]; total?: number; candidates?: T[] }): {
  items: T[]
  total: number
} {
  if (Array.isArray(payload)) return { items: payload, total: payload.length }
  const items = payload.items ?? payload.candidates ?? []
  return { items, total: payload.total ?? items.length }
}

export async function list(params?: {
  status?: ModelCatalogStatus | 'all'
  search?: string
}): Promise<ModelCatalogListResponse> {
  const { data } = await apiClient.get<
    AdminModelCatalogModel[] | ModelCatalogListResponse
  >('/admin/model-catalog', {
    params: {
      status: params?.status === 'all' ? undefined : params?.status,
      search: params?.search || undefined,
    },
  })
  return normalizeList(data)
}

export async function candidates(params?: {
  search?: string
  platform?: string
}): Promise<ModelCatalogCandidateResponse> {
  const { data } = await apiClient.get<
    ModelCatalogCandidate[] |
      ModelCatalogCandidateResponse |
      { candidates: ModelCatalogCandidate[]; total?: number }
  >('/admin/model-catalog/candidates', {
    params: {
      search: params?.search || undefined,
      platform: params?.platform || undefined,
    },
  })
  return normalizeList(data)
}

export async function create(
  request: CreateModelCatalogRequest,
): Promise<AdminModelCatalogModel> {
  const { data } = await apiClient.post<AdminModelCatalogModel>(
    '/admin/model-catalog',
    request,
  )
  return data
}

export async function getById(id: number): Promise<AdminModelCatalogModel> {
  const { data } = await apiClient.get<AdminModelCatalogModel>(
    `/admin/model-catalog/${id}`,
  )
  return data
}

export async function update(
  id: number,
  request: UpdateModelCatalogRequest,
): Promise<AdminModelCatalogModel> {
  const { data } = await apiClient.put<AdminModelCatalogModel>(
    `/admin/model-catalog/${id}`,
    request,
  )
  return data
}

export async function publish(id: number): Promise<AdminModelCatalogModel> {
  const { data } = await apiClient.post<AdminModelCatalogModel>(
    `/admin/model-catalog/${id}/publish`,
  )
  return data
}

export async function unpublish(id: number): Promise<AdminModelCatalogModel> {
  const { data } = await apiClient.post<AdminModelCatalogModel>(
    `/admin/model-catalog/${id}/unpublish`,
  )
  return data
}

const modelCatalogAPI = {
  list,
  candidates,
  create,
  getById,
  update,
  publish,
  unpublish,
}

export default modelCatalogAPI

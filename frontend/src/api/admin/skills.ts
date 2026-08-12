/**
 * Admin Skill market API.
 *
 * Skill metadata and immutable package versions have separate lifecycles. A
 * draft can be edited freely, while publish/activate/yank/archive remain
 * explicit server-side operations.
 */
import { apiClient } from '../client'

export type SkillStatus = 'draft' | 'published' | 'archived'
export type SkillVersionStatus = 'available' | 'active' | 'yanked'

export interface SkillValidationIssue {
  code: string
  message: string
  path?: string
}

export interface SkillValidationReport {
  valid: boolean
  errors: SkillValidationIssue[]
  warnings: SkillValidationIssue[]
}

export interface SkillFileManifestEntry {
  path: string
  byte_size: number
  sha256: string
}

export interface AdminSkillVersion {
  id: number
  skill_id: number
  version: string
  changelog: string
  manifest_name: string
  manifest_description: string
  sha256: string
  byte_size: number
  unpacked_size: number
  file_count: number
  file_manifest: SkillFileManifestEntry[]
  validation_report: SkillValidationReport
  status: SkillVersionStatus
  yanked_at: string | null
  created_at: string
  download_count: number
}

export interface AdminSkill {
  id: number
  slug: string
  display_name: string
  summary: string
  description: string
  category: string
  tags: string[]
  icon: string
  origin_url?: string
  source_url?: string
  example_prompts?: string[]
  risk_notes?: string
  status: SkillStatus
  featured: boolean
  sort_order: number
  catalog_source_priority: number
  catalog_source_rank?: number | null
  current_version_id: number | null
  current_version: AdminSkillVersion | null
  latest_version?: AdminSkillVersion | null
  download_count: number
  published_at: string | null
  archived_at: string | null
  created_at: string
  updated_at: string
  versions?: AdminSkillVersion[]
}

export interface SkillListResponse {
  items: AdminSkill[]
  total: number
  page: number
  page_size: number
}

export interface SkillListParams {
  status?: SkillStatus | 'all'
  search?: string
  featured?: boolean | 'all'
  page?: number
  page_size?: number
}

export interface CreateSkillRequest {
  slug: string
  display_name: string
  summary: string
  description: string
  category: string
  tags: string[]
  icon: string
  origin_url?: string
  source_url?: string
  example_prompts?: string[]
  risk_notes?: string
  featured: boolean
  sort_order: number
  catalog_source_priority?: number
  catalog_source_rank?: number | null
}

export type UpdateSkillRequest = CreateSkillRequest

export interface UploadSkillVersionRequest {
  file: File
  version: string
  changelog: string
}

export interface SkillMarketplaceConfig {
  enabled: boolean
}

type SkillListPayload =
  | AdminSkill[]
  | SkillListResponse
  | { items?: AdminSkill[]; total?: number; page?: number; page_size?: number }

function normalizeList(payload: SkillListPayload, params?: SkillListParams): SkillListResponse {
  if (Array.isArray(payload)) {
    return {
      items: payload,
      total: payload.length,
      page: params?.page ?? 1,
      page_size: params?.page_size ?? payload.length,
    }
  }

  const items = payload.items ?? []
  return {
    items,
    total: payload.total ?? items.length,
    page: payload.page ?? params?.page ?? 1,
    page_size: payload.page_size ?? params?.page_size ?? items.length,
  }
}

export async function listSkills(params?: SkillListParams): Promise<SkillListResponse> {
  const { data } = await apiClient.get<SkillListPayload>('/admin/skills', {
    params: {
      status: params?.status === 'all' ? undefined : params?.status,
      search: params?.search?.trim() || undefined,
      featured: params?.featured === 'all' ? undefined : params?.featured,
      page: params?.page,
      page_size: params?.page_size,
    },
  })
  return normalizeList(data, params)
}

export async function getSkill(id: number): Promise<AdminSkill> {
  const { data } = await apiClient.get<AdminSkill>(`/admin/skills/${id}`)
  return data
}

export async function createSkill(request: CreateSkillRequest): Promise<AdminSkill> {
  const { data } = await apiClient.post<AdminSkill>('/admin/skills', request)
  return data
}

export async function updateSkill(
  id: number,
  request: UpdateSkillRequest,
): Promise<AdminSkill> {
  const { data } = await apiClient.put<AdminSkill>(`/admin/skills/${id}`, request)
  return data
}

export async function uploadSkillVersion(
  id: number,
  request: UploadSkillVersionRequest,
): Promise<AdminSkillVersion> {
  const formData = new FormData()
  formData.append('file', request.file)
  formData.append('version', request.version.trim())
  formData.append('changelog', request.changelog.trim())

  const { data } = await apiClient.post<AdminSkillVersion>(
    `/admin/skills/${id}/versions`,
    formData,
    { headers: { 'Content-Type': 'multipart/form-data' } },
  )
  return data
}

export async function publishSkill(id: number, versionId: number): Promise<AdminSkill> {
  const { data } = await apiClient.post<AdminSkill>(`/admin/skills/${id}/publish`, {
    version_id: versionId,
  })
  return data
}

export async function activateSkillVersion(
  id: number,
  versionId: number,
): Promise<AdminSkill> {
  const { data } = await apiClient.post<AdminSkill>(
    `/admin/skills/${id}/versions/${versionId}/activate`,
  )
  return data
}

export async function yankSkillVersion(
  id: number,
  versionId: number,
): Promise<AdminSkill> {
  const { data } = await apiClient.post<AdminSkill>(
    `/admin/skills/${id}/versions/${versionId}/yank`,
  )
  return data
}

export async function archiveSkill(id: number): Promise<AdminSkill> {
  const { data } = await apiClient.post<AdminSkill>(`/admin/skills/${id}/archive`)
  return data
}

export async function getSkillMarketplaceConfig(): Promise<SkillMarketplaceConfig> {
  const { data } = await apiClient.get<SkillMarketplaceConfig>('/admin/skills/config')
  return data
}

export async function updateSkillMarketplaceConfig(
  enabled: boolean,
): Promise<SkillMarketplaceConfig> {
  const { data } = await apiClient.put<SkillMarketplaceConfig>('/admin/skills/config', { enabled })
  return data
}

export const skillsAPI = {
  list: listSkills,
  getById: getSkill,
  create: createSkill,
  update: updateSkill,
  uploadVersion: uploadSkillVersion,
  publish: publishSkill,
  activateVersion: activateSkillVersion,
  yankVersion: yankSkillVersion,
  archive: archiveSkill,
  getConfig: getSkillMarketplaceConfig,
  updateConfig: updateSkillMarketplaceConfig,
}

export default skillsAPI

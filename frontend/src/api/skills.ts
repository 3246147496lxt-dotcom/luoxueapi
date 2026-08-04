import { apiClient } from './client'
import { buildApiUrl } from './url'

export interface SkillFileManifestEntry {
  path: string
  byte_size: number
  sha256: string
}

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

export interface PublicSkillVersion {
  version: string
  changelog: string
  manifest_name: string
  manifest_description: string
  skill_md: string
  sha256: string
  byte_size: number
  unpacked_size: number
  file_count: number
  file_manifest: SkillFileManifestEntry[]
  validation_report: SkillValidationReport
  created_at: string
  download_count: number
}

export interface PublicSkill {
  slug: string
  display_name: string
  summary: string
  description: string
  category: string
  tags: string[]
  icon: string
  example_prompts: string[]
  risk_notes: string
  featured: boolean
  current_version: PublicSkillVersion | null
  download_count: number
  published_at: string
  updated_at: string
  versions: PublicSkillVersion[]
  source_url?: string
  source_repository?: string
  repository_stars?: number | null
}

export interface SkillCategory {
  slug: string
  label: string
  count: number | null
}

export interface SkillCatalogQuery {
  search?: string
  category?: string
  featured?: boolean
  page?: number
  page_size?: number
}

export interface SkillCatalogResponse {
  items: PublicSkill[]
  total: number
  page: number
  page_size: number
  categories: SkillCategory[]
}

type UnknownRecord = Record<string, unknown>

function record(value: unknown): UnknownRecord {
  return value && typeof value === 'object' ? value as UnknownRecord : {}
}

function stringValue(value: unknown): string {
  return typeof value === 'string' ? value.trim() : ''
}

function numberValue(value: unknown): number {
  return typeof value === 'number' && Number.isFinite(value) ? value : 0
}

function optionalNumberValue(value: unknown): number | null {
  return typeof value === 'number' && Number.isFinite(value) && value >= 0 ? value : null
}

function stringArray(value: unknown): string[] {
  if (!Array.isArray(value)) return []
  return value
    .map(stringValue)
    .filter(Boolean)
}

function normalizeValidationIssues(value: unknown): SkillValidationIssue[] {
  if (!Array.isArray(value)) return []
  return value
    .map((entry) => {
      const issue = record(entry)
      return {
        code: stringValue(issue.code),
        message: stringValue(issue.message),
        path: stringValue(issue.path) || undefined,
      }
    })
    .filter((issue) => Boolean(issue.code || issue.message))
}

function normalizeValidationReport(value: unknown): SkillValidationReport {
  const report = record(value)
  return {
    valid: report.valid === true,
    errors: normalizeValidationIssues(report.errors),
    warnings: normalizeValidationIssues(report.warnings),
  }
}

export function normalizePublicSkillVersion(value: unknown): PublicSkillVersion {
  const source = record(value)
  const manifest = Array.isArray(source.file_manifest) ? source.file_manifest : []

  return {
    version: stringValue(source.version),
    changelog: stringValue(source.changelog),
    manifest_name: stringValue(source.manifest_name),
    manifest_description: stringValue(source.manifest_description),
    skill_md: stringValue(source.skill_md),
    sha256: stringValue(source.sha256),
    byte_size: numberValue(source.byte_size),
    unpacked_size: numberValue(source.unpacked_size),
    file_count: numberValue(source.file_count),
    file_manifest: manifest
      .map((entry) => {
        const item = record(entry)
        return {
          path: stringValue(item.path),
          byte_size: numberValue(item.byte_size),
          sha256: stringValue(item.sha256),
        }
      })
      .filter((entry) => Boolean(entry.path)),
    validation_report: normalizeValidationReport(source.validation_report),
    created_at: stringValue(source.created_at) || stringValue(source.released_at),
    download_count: numberValue(source.download_count),
  }
}

export function normalizePublicSkill(value: unknown): PublicSkill {
  const source = record(value)
  const rawCurrentVersion = source.current_version
  const currentVersion = typeof rawCurrentVersion === 'string'
    ? normalizePublicSkillVersion({ version: rawCurrentVersion })
    : rawCurrentVersion && typeof rawCurrentVersion === 'object'
      ? normalizePublicSkillVersion(rawCurrentVersion)
      : null
  const rawVersions = Array.isArray(source.versions) ? source.versions : []

  return {
    slug: stringValue(source.slug),
    display_name: stringValue(source.display_name) || stringValue(source.slug),
    summary: stringValue(source.summary),
    description: stringValue(source.description),
    category: stringValue(source.category),
    tags: stringArray(source.tags),
    icon: stringValue(source.icon),
    example_prompts: stringArray(source.example_prompts),
    risk_notes: stringValue(source.risk_notes),
    featured: source.featured === true,
    current_version: currentVersion?.version ? currentVersion : null,
    download_count: numberValue(source.download_count),
    published_at: stringValue(source.published_at),
    updated_at: stringValue(source.updated_at),
    versions: rawVersions
      .map(normalizePublicSkillVersion)
      .filter((version) => Boolean(version.version)),
    source_url: stringValue(source.source_url) || stringValue(source.repository_url) || undefined,
    source_repository: stringValue(source.source_repository) || stringValue(source.repository_name) || undefined,
    repository_stars: optionalNumberValue(
      source.repository_stars ?? source.github_stars ?? source.source_stars,
    ),
  }
}

function normalizeCategory(value: unknown): SkillCategory | null {
  if (typeof value === 'string') {
    const slug = value.trim()
    return slug ? { slug, label: slug, count: null } : null
  }

  const source = record(value)
  const slug = stringValue(source.slug) || stringValue(source.value) || stringValue(source.name)
  if (!slug) return null
  const rawCount = source.count
  return {
    slug,
    label: stringValue(source.label) || stringValue(source.name) || slug,
    count: typeof rawCount === 'number' && Number.isFinite(rawCount) ? rawCount : null,
  }
}

export function normalizeSkillCatalogResponse(
  value: unknown,
  query: SkillCatalogQuery = {},
): SkillCatalogResponse {
  const source = record(value)
  const rawItems = Array.isArray(source.items) ? source.items : Array.isArray(value) ? value : []
  const items = rawItems
    .map(normalizePublicSkill)
    .filter((skill) => Boolean(skill.slug))
  const rawCategories = Array.isArray(source.categories) ? source.categories : []

  return {
    items,
    total: typeof source.total === 'number' ? source.total : items.length,
    page: typeof source.page === 'number' ? source.page : query.page ?? 1,
    page_size: typeof source.page_size === 'number'
      ? source.page_size
      : query.page_size ?? items.length,
    categories: rawCategories
      .map(normalizeCategory)
      .filter((category): category is SkillCategory => category !== null),
  }
}

export async function listPublicSkills(
  query: SkillCatalogQuery = {},
  options?: { signal?: AbortSignal },
): Promise<SkillCatalogResponse> {
  const { data } = await apiClient.get<unknown>('/catalog/skills', {
    signal: options?.signal,
    params: {
      search: query.search?.trim() || undefined,
      category: query.category?.trim() || undefined,
      featured: query.featured,
      page: query.page,
      page_size: query.page_size,
    },
  })
  return normalizeSkillCatalogResponse(data, query)
}

export async function getPublicSkill(
  slug: string,
  options?: { signal?: AbortSignal },
): Promise<PublicSkill> {
  const { data } = await apiClient.get<unknown>(`/catalog/skills/${encodeURIComponent(slug)}`, {
    signal: options?.signal,
  })
  return normalizePublicSkill(data)
}

export async function getPublicSkillVersions(
  slug: string,
  options?: { signal?: AbortSignal },
): Promise<PublicSkillVersion[]> {
  const { data } = await apiClient.get<unknown>(
    `/catalog/skills/${encodeURIComponent(slug)}/versions`,
    { signal: options?.signal },
  )
  const source = record(data)
  const versions = Array.isArray(data) ? data : Array.isArray(source.items) ? source.items : []
  return versions
    .map(normalizePublicSkillVersion)
    .filter((version) => Boolean(version.version))
}

export function getSkillVersionDownloadURL(slug: string, version: string): string {
  return buildApiUrl(
    `/catalog/skills/${encodeURIComponent(slug)}/versions/${encodeURIComponent(version)}/download`,
  )
}

export const publicSkillsAPI = {
  list: listPublicSkills,
  get: getPublicSkill,
  versions: getPublicSkillVersions,
  versionDownloadURL: getSkillVersionDownloadURL,
}

export default publicSkillsAPI

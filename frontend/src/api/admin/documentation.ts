import { apiClient } from '../client'

export type DocumentationNoteTone = 'info' | 'warning'
export type DocumentationNotePlacement = 'before-image' | 'after-image'
export type DocumentationIconKey =
  | 'key'
  | 'client'
  | 'api'
  | 'wallet'
  | 'image'
  | 'help'
  | 'book'
  | 'code'
  | 'terminal'

export interface DocumentationNote {
  tone: DocumentationNoteTone
  text: string
}

export interface DocumentationCode {
  label: string
  value: string
}

export interface DocumentationImage {
  src: string
  alt: string
  caption?: string
}

export interface DocumentationLink {
  label: string
  href: string
}

export interface DocumentationStep {
  title: string
  description: string
  note?: DocumentationNote
  note_placement?: DocumentationNotePlacement
  code?: DocumentationCode
  image?: DocumentationImage
  link?: DocumentationLink
}

export interface DocumentationTutorial {
  id: string
  tab_label: string
  icon: DocumentationIconKey
  icon_svg?: string
  description: string
  steps: DocumentationStep[]
}

export interface DocumentationContent {
  schema_version: 1
  tutorials: DocumentationTutorial[]
  /**
   * Public-site presentation settings are owned by the backend seed. The
   * structured editor does not expose them yet, but must round-trip them so a
   * content-only save can never erase branding or navigation configuration.
   */
  site_config?: Record<string, unknown>
}

export interface DocumentationSnapshot {
  content: DocumentationContent
  version?: number | string | null
  updated_at?: string | null
  published_at?: string | null
}

export interface AdminDocumentationState {
  draft: DocumentationSnapshot
  published: DocumentationSnapshot | null
}

export interface DocumentationRevision {
  id: number | string
  version?: number | string | null
  created_at?: string | null
  published_at?: string | null
  restored_at?: string | null
}

export interface DocumentationRevisionList {
  items: DocumentationRevision[]
}

type SnapshotLike = DocumentationSnapshot | DocumentationContent
type StateLike =
  | AdminDocumentationState
  | DocumentationContent
  | DocumentationSnapshot
  | {
      draft?: SnapshotLike | null
      published?: SnapshotLike | null
      content?: DocumentationContent
      version?: number | string | null
      updated_at?: string | null
      published_at?: string | null
    }

const EMPTY_CONTENT: DocumentationContent = { schema_version: 1, tutorials: [] }

function isContent(value: unknown): value is DocumentationContent {
  if (!value || typeof value !== 'object') return false
  const candidate = value as Partial<DocumentationContent>
  return candidate.schema_version === 1 && Array.isArray(candidate.tutorials)
}

function normalizeSnapshot(
  value: SnapshotLike | null | undefined,
  fallback: DocumentationContent = EMPTY_CONTENT,
): DocumentationSnapshot {
  if (isContent(value)) return { content: value }
  if (value && typeof value === 'object' && isContent((value as DocumentationSnapshot).content)) {
    const snapshot = value as DocumentationSnapshot
    return {
      ...snapshot,
      published_at: snapshot.published_at ?? snapshot.updated_at ?? null,
    }
  }
  return { content: fallback }
}

function normalizeState(payload: StateLike): AdminDocumentationState {
  if (isContent(payload)) {
    return { draft: { content: payload }, published: null }
  }

  if (payload && typeof payload === 'object' && ('draft' in payload || 'published' in payload)) {
    const state = payload as Extract<StateLike, { draft?: SnapshotLike | null }>
    const draft = normalizeSnapshot(state.draft, EMPTY_CONTENT)
    return {
      draft,
      published: state.published ? normalizeSnapshot(state.published, draft.content) : null,
    }
  }

  return { draft: normalizeSnapshot(payload as SnapshotLike), published: null }
}

export async function getDocumentation(): Promise<AdminDocumentationState> {
  const { data } = await apiClient.get<StateLike>('/admin/documentation')
  return normalizeState(data)
}

export async function saveDocumentationDraft(
  content: DocumentationContent,
): Promise<DocumentationSnapshot> {
  const { data } = await apiClient.put<SnapshotLike | { draft?: SnapshotLike }>(
    '/admin/documentation/draft',
    { content },
  )
  if (data && typeof data === 'object' && 'draft' in data && data.draft) {
    return normalizeSnapshot(data.draft, content)
  }
  return normalizeSnapshot(data as SnapshotLike, content)
}

export async function publishDocumentation(): Promise<AdminDocumentationState> {
  const { data } = await apiClient.post<StateLike>('/admin/documentation/publish')
  return normalizeState(data)
}

export async function listDocumentationRevisions(): Promise<DocumentationRevisionList> {
  const { data } = await apiClient.get<
    DocumentationRevision[] | DocumentationRevisionList
  >('/admin/documentation/revisions')
  return Array.isArray(data) ? { items: data } : { items: data?.items ?? [] }
}

export async function restoreDocumentationRevision(
  id: number | string,
): Promise<DocumentationSnapshot> {
  const { data } = await apiClient.post<SnapshotLike | { draft?: SnapshotLike }>(
    `/admin/documentation/revisions/${encodeURIComponent(String(id))}/restore`,
  )
  if (data && typeof data === 'object' && 'draft' in data && data.draft) {
    return normalizeSnapshot(data.draft)
  }
  return normalizeSnapshot(data as SnapshotLike)
}

export async function uploadDocumentationAsset(file: File): Promise<{ url: string }> {
  const formData = new FormData()
  formData.append('file', file)
  const { data } = await apiClient.post<{ url: string }>(
    '/admin/documentation/assets',
    formData,
    {
      // apiClient defaults to JSON. Override it here so Axios keeps the File in
      // FormData; the browser adapter will attach the multipart boundary.
      headers: { 'Content-Type': 'multipart/form-data' },
    },
  )
  return data
}

const documentationAPI = {
  get: getDocumentation,
  saveDraft: saveDocumentationDraft,
  publish: publishDocumentation,
  revisions: listDocumentationRevisions,
  restoreRevision: restoreDocumentationRevision,
  uploadAsset: uploadDocumentationAsset,
}

export default documentationAPI

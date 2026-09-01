import { apiClient } from './client'
import type {
  Project,
  ProjectConversation,
  ProjectFile,
  ProjectInput,
  ProjectMemoryMode,
} from '@/types/projects'

type UnknownRecord = Record<string, unknown>

function record(value: unknown): UnknownRecord | null {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
    ? value as UnknownRecord
    : null
}

function text(value: unknown, fallback = ''): string {
  return typeof value === 'string' ? value.trim() || fallback : fallback
}

function timestamp(value: unknown): number {
  if (typeof value === 'number' && Number.isFinite(value)) return value
  const parsed = Date.parse(typeof value === 'string' ? value : '')
  return Number.isFinite(parsed) ? parsed : Date.now()
}

function normalizeMemoryMode(value: unknown): ProjectMemoryMode {
  return value === 'project_only' || value === 'project-only' ? 'project-only' : 'default'
}

function normalizeIcon(value: unknown): string {
  const icon = text(value)
  return icon === 'folder' || !icon ? '📁' : icon
}

function normalizeConversation(value: unknown): ProjectConversation | null {
  const item = record(value)
  const id = text(item?.id ?? item?.conversation_id ?? item?.conversationId)
  if (!id) return null
  return {
    id,
    title: text(item?.title, 'New chat'),
    model: text(item?.model) || undefined,
    messageCount: typeof item?.message_count === 'number'
      ? item.message_count
      : typeof item?.messageCount === 'number' ? item.messageCount : undefined,
    createdAt: text(item?.created_at ?? item?.createdAt) || undefined,
    updatedAt: text(item?.updated_at ?? item?.updatedAt) || undefined,
  }
}

function normalizeFile(value: unknown): ProjectFile | null {
  const item = record(value)
  const id = text(item?.id ?? item?.file_id ?? item?.fileId)
  const name = text(item?.name ?? item?.original_name ?? item?.originalName, 'Untitled file')
  if (!id) return null
  const rawSize = Number(item?.size ?? item?.byte_size ?? item?.byteSize ?? 0)
  return {
    id,
    name,
    size: Number.isFinite(rawSize) && rawSize >= 0 ? rawSize : 0,
    mimeType: text(item?.mime_type ?? item?.mimeType) || undefined,
    createdAt: text(item?.created_at ?? item?.createdAt) || undefined,
  }
}

export function normalizeProject(value: unknown): Project {
  const item = record(value)
  const conversations = (Array.isArray(item?.conversations) ? item.conversations : [])
    .map(normalizeConversation)
    .filter((conversation): conversation is ProjectConversation => conversation !== null)
  const explicitIds = Array.isArray(item?.conversationIds)
    ? item.conversationIds.map((id) => String(id)).filter(Boolean)
    : []
  const files = (Array.isArray(item?.files) ? item.files : [])
    .map(normalizeFile)
    .filter((file): file is ProjectFile => file !== null)
  return {
    id: text(item?.id),
    name: text(item?.name, 'Untitled project'),
    icon: normalizeIcon(item?.icon),
    color: text(item?.color, '#7c3aed'),
    memoryMode: normalizeMemoryMode(item?.memory_mode ?? item?.memoryMode),
    instructions: typeof item?.instructions === 'string' ? item.instructions : '',
    conversationIds: explicitIds.length > 0
      ? explicitIds
      : conversations.map(({ id }) => id),
    conversations,
    files,
    createdAt: timestamp(item?.created_at ?? item?.createdAt),
    updatedAt: timestamp(item?.updated_at ?? item?.updatedAt),
    localOnly: item?.localOnly === true,
  }
}

function requireProject(value: unknown): Project {
  const project = normalizeProject(value)
  if (!project.id) {
    throw new Error('The project response is invalid.')
  }
  return project
}

function payload(input: Partial<ProjectInput>): Record<string, unknown> {
  const body: Record<string, unknown> = {}
  if (input.name !== undefined) body.name = input.name
  if (input.icon !== undefined) body.icon = input.icon
  if (input.color !== undefined) body.color = input.color
  if (input.instructions !== undefined) body.instructions = input.instructions
  if (input.memoryMode !== undefined) {
    body.memory_mode = input.memoryMode === 'project-only' ? 'project_only' : input.memoryMode
  }
  return body
}

function unwrap(value: unknown): unknown {
  const item = record(value)
  return item && 'code' in item && item.code === 0 ? item.data : value
}

export async function listProjects(signal?: AbortSignal): Promise<Project[]> {
  const response = await apiClient.get('/projects', { signal })
  const data = unwrap(response.data)
  const object = record(data)
  const items = Array.isArray(data) ? data : Array.isArray(object?.projects) ? object.projects : []
  return items.map(normalizeProject).filter(({ id }) => Boolean(id))
}

export async function createProject(input: ProjectInput): Promise<Project> {
  const response = await apiClient.post('/projects', payload(input))
  return requireProject(unwrap(response.data))
}

export async function updateProject(id: string, input: Partial<ProjectInput>): Promise<Project> {
  const response = await apiClient.patch(`/projects/${encodeURIComponent(id)}`, payload(input))
  return requireProject(unwrap(response.data))
}

export async function getProject(id: string, signal?: AbortSignal): Promise<Project> {
  const response = await apiClient.get(`/projects/${encodeURIComponent(id)}`, { signal })
  return requireProject(unwrap(response.data))
}

export async function deleteProject(id: string): Promise<void> {
  await apiClient.delete(`/projects/${encodeURIComponent(id)}`)
}

export async function moveConversation(conversationId: string, projectId: string | null): Promise<void> {
  await apiClient.patch(`/chat/conversations/${encodeURIComponent(conversationId)}/project`, { project_id: projectId })
}

export async function addProjectFile(projectId: string, fileId: string): Promise<ProjectFile | null> {
  const response = await apiClient.post(`/projects/${encodeURIComponent(projectId)}/files`, { file_id: fileId })
  return normalizeFile(unwrap(response.data))
}

export async function removeProjectFile(projectId: string, fileId: string): Promise<void> {
  await apiClient.delete(`/projects/${encodeURIComponent(projectId)}/files/${encodeURIComponent(fileId)}`)
}

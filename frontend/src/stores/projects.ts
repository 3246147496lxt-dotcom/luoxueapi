import { computed, ref, watch } from 'vue'
import { defineStore } from 'pinia'
import {
  addProjectFile,
  createProject as createProjectRequest,
  deleteProject as deleteProjectRequest,
  getProject as getProjectRequest,
  listProjects,
  moveConversation,
  normalizeProject,
  removeProjectFile,
  updateProject as updateProjectRequest,
} from '@/api/projects'
import type {
  Project,
  ProjectConversation,
  ProjectFile,
  ProjectInput,
  ProjectMemoryMode,
} from '@/types/projects'
import { useAuthStore } from './auth'

const STORAGE_KEY = 'sub2api.projects.v2'
const DEFAULT_COLOR = '#7c3aed'
const DEFAULT_ICON = '📁'

function localId(): string {
  try {
    return `project-local-${crypto.randomUUID()}`
  } catch {
    return `project-local-${Date.now()}-${Math.random().toString(36).slice(2)}`
  }
}

function storageKey(userId: unknown): string | null {
  const normalized = String(userId ?? '').trim()
  return normalized ? `${STORAGE_KEY}:${normalized}` : null
}

function readLocal(userId: unknown): Project[] {
  const key = storageKey(userId)
  if (!key || typeof localStorage === 'undefined') return []
  try {
    const parsed = JSON.parse(localStorage.getItem(key) || '[]')
    return Array.isArray(parsed)
      ? parsed.map(normalizeProject).filter(({ id }) => Boolean(id))
      : []
  } catch {
    return []
  }
}

function persist(userId: unknown, projects: Project[]): void {
  const key = storageKey(userId)
  if (!key || typeof localStorage === 'undefined') return
  try {
    localStorage.setItem(key, JSON.stringify(projects))
  } catch {
    // Local persistence is an offline enhancement; server state remains source
    // of truth when storage is unavailable (private browsing, quota, etc.).
  }
}

function normalizeLocalProject(input: ProjectInput): Project {
  const now = Date.now()
  return {
    id: localId(),
    name: input.name.trim() || '未命名项目',
    icon: input.icon?.trim() || DEFAULT_ICON,
    color: input.color?.trim() || DEFAULT_COLOR,
    memoryMode: input.memoryMode || 'default',
    instructions: input.instructions ?? '',
    conversationIds: [],
    conversations: [],
    files: [],
    createdAt: now,
    updatedAt: now,
    localOnly: true,
  }
}

function mergeConversation(project: Project, conversation: ProjectConversation): void {
  if (!project.conversationIds.includes(conversation.id)) {
    project.conversationIds = [...project.conversationIds, conversation.id]
  }
  const index = project.conversations.findIndex(({ id }) => id === conversation.id)
  if (index >= 0) project.conversations[index] = { ...project.conversations[index], ...conversation }
  else project.conversations = [...project.conversations, conversation]
}

export const useProjectsStore = defineStore('projects', () => {
  const authStore = useAuthStore()
  const projects = ref<Project[]>(readLocal(authStore.user?.id))
  const loading = ref(false)
  const initialized = ref(false)
  const error = ref('')
  const activeProjectId = ref<string | null>(null)
  let loadPromise: Promise<void> | null = null
  let loadPromiseUserKey: string | null = null
  let loadSequence = 0
  const activeProject = computed(() => (
    projects.value.find((project) => project.id === activeProjectId.value) ?? null
  ))

  function save(): void {
    persist(authStore.user?.id, projects.value)
  }

  async function load(signal?: AbortSignal): Promise<void> {
    const userId = authStore.user?.id
    const userKey = storageKey(userId)
    if (loadPromise && loadPromiseUserKey === userKey) return loadPromise
    const sequence = ++loadSequence
    const run = (async () => {
      if (userId === null || userId === undefined) {
        projects.value = []
        initialized.value = true
        return
      }
      loading.value = true
      error.value = ''
      try {
        // A successful empty response must clear stale remote projects. Keep
        // this account's local-only drafts so an offline-created project is not
        // lost when the first online refresh returns an empty server list.
        const remoteProjects = await listProjects(signal)
        if (sequence !== loadSequence || storageKey(authStore.user?.id) !== userKey) return
        const localDrafts = projects.value.filter(({ localOnly }) => localOnly)
        projects.value = [...localDrafts, ...remoteProjects]
        if (activeProjectId.value && !projects.value.some(({ id }) => id === activeProjectId.value)) {
          activeProjectId.value = null
        }
        save()
      } catch (cause) {
        if (sequence !== loadSequence || storageKey(authStore.user?.id) !== userKey) return
        error.value = cause instanceof Error ? cause.message : '项目暂时无法同步，已显示本地数据。'
      } finally {
        if (sequence === loadSequence) {
          initialized.value = true
          loading.value = false
        }
      }
    })()
    loadPromise = run
    loadPromiseUserKey = userKey
    try {
      await run
    } finally {
      if (loadPromise === run) loadPromise = null
      if (loadPromise === null) loadPromiseUserKey = null
    }
  }

  async function get(id: string, signal?: AbortSignal): Promise<Project | null> {
    const normalizedId = id.trim()
    if (!normalizedId) return null
    try {
      const remote = await getProjectRequest(normalizedId, signal)
      const index = projects.value.findIndex(({ id: projectId }) => projectId === normalizedId)
      if (index >= 0) projects.value[index] = { ...projects.value[index], ...remote, localOnly: false }
      else projects.value.push(remote)
      save()
      return remote
    } catch (cause) {
      error.value = cause instanceof Error ? cause.message : '项目详情暂时无法加载。'
      return projects.value.find(({ id: projectId }) => projectId === normalizedId) ?? null
    }
  }

  async function create(input: ProjectInput): Promise<Project> {
    const local = normalizeLocalProject(input)
    projects.value.unshift(local)
    save()
    try {
      const remote = await createProjectRequest(input)
      const index = projects.value.findIndex(({ id }) => id === local.id)
      const persistedProject = { ...local, ...remote, localOnly: false }
      if (index >= 0) projects.value[index] = persistedProject
      save()
      return persistedProject
    } catch (cause) {
      error.value = cause instanceof Error ? cause.message : '项目已保存在本地，稍后可重试同步。'
      return local
    }
  }

  async function update(id: string, input: Partial<ProjectInput>): Promise<void> {
    const index = projects.value.findIndex(({ id: projectId }) => projectId === id)
    if (index < 0) return
    const current = projects.value[index]!
    projects.value[index] = { ...current, ...input, updatedAt: Date.now() }
    save()
    if (current.localOnly) return
    try {
      const remote = await updateProjectRequest(id, input)
      projects.value[index] = { ...projects.value[index]!, ...remote, localOnly: false }
      save()
    } catch (cause) {
      error.value = cause instanceof Error ? cause.message : '项目更改未同步。'
    }
  }

  async function remove(id: string): Promise<boolean> {
    const index = projects.value.findIndex(({ id: projectId }) => projectId === id)
    if (index < 0) return false
    const removed = projects.value[index]!
    projects.value.splice(index, 1)
    if (activeProjectId.value === id) activeProjectId.value = null
    save()
    if (removed.localOnly) return true
    try {
      await deleteProjectRequest(id)
      return true
    } catch (cause) {
      projects.value.splice(index, 0, removed)
      save()
      error.value = cause instanceof Error ? cause.message : '项目删除失败。'
      return false
    }
  }

  function select(id: string | null): void {
    activeProjectId.value = id
  }

  async function addConversation(projectId: string, conversationId: string, conversation?: ProjectConversation): Promise<boolean> {
    let project = projects.value.find(({ id }) => id === projectId)
    if (!project) {
      await load()
      project = projects.value.find(({ id }) => id === projectId)
    }
    if (!project) {
      await get(projectId)
      project = projects.value.find(({ id }) => id === projectId)
    }
    if (!project || !conversationId.trim()) return false
    if (!project.localOnly) {
      try {
        await moveConversation(conversationId, projectId)
      } catch (cause) {
        error.value = cause instanceof Error ? cause.message : '聊天移入项目失败。'
        return false
      }
    }
    mergeConversation(project, conversation ?? { id: conversationId, title: 'New chat' })
    project.updatedAt = Date.now()
    save()
    return true
  }

  async function removeConversation(projectId: string, conversationId: string): Promise<boolean> {
    const project = projects.value.find(({ id }) => id === projectId)
    if (!project) return false
    if (!project.localOnly) {
      try {
        await moveConversation(conversationId, null)
      } catch (cause) {
        error.value = cause instanceof Error ? cause.message : '聊天移出项目失败。'
        return false
      }
    }
    project.conversationIds = project.conversationIds.filter((id) => id !== conversationId)
    project.conversations = project.conversations.filter(({ id }) => id !== conversationId)
    project.updatedAt = Date.now()
    save()
    return true
  }

  async function addFiles(projectId: string, files: ProjectFile[]): Promise<ProjectFile[]> {
    const project = projects.value.find(({ id }) => id === projectId)
    if (!project) return []
    const accepted: ProjectFile[] = []
    for (const file of files) {
      if (!file.id || project.files.some(({ id }) => id === file.id)) continue
      if (!project.localOnly) {
        try {
          const linked = await addProjectFile(projectId, file.id)
          if (!linked) {
            error.value = '项目文件响应无效，文件未关联。'
            continue
          }
          accepted.push({ ...file, ...linked })
          continue
        } catch (cause) {
          error.value = cause instanceof Error ? cause.message : '文件关联项目失败。'
          continue
        }
      }
      accepted.push(file)
    }
    if (accepted.length > 0) {
      project.files = [...project.files, ...accepted]
      project.updatedAt = Date.now()
      save()
    }
    return accepted
  }

  async function removeFile(projectId: string, fileId: string): Promise<boolean> {
    const project = projects.value.find(({ id }) => id === projectId)
    if (!project) return false
    if (!project.localOnly) {
      try {
        await removeProjectFile(projectId, fileId)
      } catch (cause) {
        error.value = cause instanceof Error ? cause.message : '文件移出项目失败。'
        return false
      }
    }
    project.files = project.files.filter(({ id }) => id !== fileId)
    project.updatedAt = Date.now()
    save()
    return true
  }

  function setMemoryMode(id: string, mode: ProjectMemoryMode): Promise<void> {
    return update(id, { memoryMode: mode })
  }

  // Switch the cache namespace immediately on login/logout, before any route
  // component can render the previous account's project rows.
  watch(() => authStore.user?.id, (userId) => {
    loadSequence += 1
    loadPromise = null
    loadPromiseUserKey = null
    projects.value = readLocal(userId)
    activeProjectId.value = null
    initialized.value = false
    error.value = ''
    if (userId !== null && userId !== undefined) void load()
  }, { flush: 'sync' })

  return {
    projects,
    loading,
    initialized,
    error,
    activeProjectId,
    activeProject,
    load,
    get,
    create,
    update,
    remove,
    select,
    addConversation,
    removeConversation,
    addFiles,
    removeFile,
    setMemoryMode,
  }
})

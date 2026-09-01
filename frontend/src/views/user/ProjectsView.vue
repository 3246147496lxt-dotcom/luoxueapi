<template>
  <AppLayout variant="chat" shell-mode="chat">
    <div class="projects-shell">
      <ChatHistoryPanel
        shell
        active-section="projects"
        class="projects-shell__history"
        :conversations="historyConversations"
        :projects="projectsStore.projects"
        :active-id="chatStore.activeConversationId"
        @new="startNewChat"
        @select="openConversation"
      />

      <main class="projects-content" tabindex="-1">
        <header class="projects-topbar">
          <div class="projects-breadcrumbs">
            <button
              v-if="selectedProject"
              type="button"
              class="projects-back"
              @click="goBack"
            >
              <Icon name="arrowLeft" size="sm" aria-hidden="true" />
              <span>{{ t('projects.allProjects') }}</span>
            </button>
            <span v-else class="projects-kicker">{{ t('projects.eyebrow') }}</span>
          </div>
          <div class="projects-topbar__actions">
            <button class="projects-button projects-button--primary" type="button" @click="openCreate">
              <Icon name="plus" size="sm" aria-hidden="true" />
              <span>{{ t('projects.newProject') }}</span>
            </button>
          </div>
        </header>

        <div v-if="projectsStore.error" class="projects-notice" role="alert">
          <Icon name="exclamationTriangle" size="sm" aria-hidden="true" />
          <span>{{ projectsStore.error }}</span>
          <button type="button" @click="retryLoad">{{ t('projects.retry') }}</button>
        </div>

        <div v-if="projectsStore.loading && !projectsStore.initialized" class="projects-loading" aria-live="polite">
          <span v-for="index in 3" :key="index" class="projects-skeleton-card" aria-hidden="true" />
          <span class="sr-only">{{ t('projects.loading') }}</span>
        </div>

        <template v-else-if="selectedProject">
          <section class="project-hero" :style="{ '--project-accent': projectAccent(selectedProject) }">
            <div class="project-hero__identity">
              <span class="project-hero__icon" aria-hidden="true">{{ selectedProject.icon }}</span>
              <div>
                <p class="projects-kicker">{{ t('projects.projectLabel') }}</p>
                <h1>{{ selectedProject.name }}</h1>
                <p class="project-hero__meta">
                  {{ projectConversations.length }} {{ t('projects.chats') }}
                  <span aria-hidden="true">·</span>
                  {{ selectedProject.files.length }} {{ t('projects.files') }}
                </p>
              </div>
            </div>
            <div class="project-hero__actions">
              <button type="button" class="projects-button projects-button--quiet" @click="openEdit(selectedProject)">
                <Icon name="edit" size="sm" aria-hidden="true" /> {{ t('projects.edit') }}
              </button>
              <button type="button" class="projects-button projects-button--primary" @click="startProjectChat">
                <Icon name="plus" size="sm" aria-hidden="true" /> {{ t('projects.newChat') }}
              </button>
            </div>
          </section>

          <div class="project-layout">
            <section class="project-surface project-surface--chats">
              <div class="project-surface__heading">
                <div>
                  <p class="projects-kicker">{{ t('projects.activityLabel') }}</p>
                  <h2>{{ t('projects.chats') }}</h2>
                </div>
                <button type="button" class="projects-button projects-button--quiet" @click="showChatPicker = !showChatPicker">
                  <Icon name="plus" size="sm" aria-hidden="true" /> {{ t('projects.addChat') }}
                </button>
              </div>
              <div v-if="showChatPicker" class="project-picker" role="listbox" :aria-label="t('projects.addChat')">
                <button
                  v-for="conversation in availableConversations"
                  :key="conversation.id"
                  type="button"
                  role="option"
                  @click="addConversation(conversation.id)"
                >
                  <span>{{ conversation.title }}</span>
                  <Icon name="plus" size="sm" aria-hidden="true" />
                </button>
                <p v-if="availableConversations.length === 0" class="project-empty-inline">{{ t('projects.noAvailableChats') }}</p>
              </div>
              <ul v-if="projectConversations.length" class="project-chat-list">
                <li v-for="conversation in projectConversations" :key="conversation.id">
                  <button type="button" class="project-chat-list__open" @click="openConversation(conversation.id)">
                    <span class="project-chat-list__avatar" aria-hidden="true">✦</span>
                    <span class="project-chat-list__copy">
                      <strong>{{ conversation.title }}</strong>
                      <small>{{ formatConversationDate(conversation.updatedAt) }}</small>
                    </span>
                  </button>
                  <button
                    type="button"
                    class="project-icon-button"
                    :aria-label="t('projects.removeChat')"
                    :title="t('projects.removeChat')"
                    @click="removeConversation(conversation.id)"
                  >
                    <Icon name="x" size="sm" aria-hidden="true" />
                  </button>
                </li>
              </ul>
              <div v-else class="project-empty-state">
                <span class="project-empty-state__mark" aria-hidden="true">☼</span>
                <p>{{ t('projects.noChats') }}</p>
                <button type="button" class="projects-button projects-button--quiet" @click="startProjectChat">
                  {{ t('projects.newChat') }}
                </button>
              </div>
            </section>

            <aside class="project-layout__aside">
              <section class="project-surface">
                <div class="project-surface__heading project-surface__heading--compact">
                  <div>
                    <p class="projects-kicker">{{ t('projects.settingsLabel') }}</p>
                    <h2>{{ t('projects.instructions') }}</h2>
                  </div>
                  <span class="project-status-dot" :class="{ 'project-status-dot--set': instructionsDraft.trim() }" aria-hidden="true" />
                </div>
                <textarea
                  v-model="instructionsDraft"
                  class="project-instructions"
                  :placeholder="t('projects.instructionsPlaceholder')"
                  :aria-label="t('projects.instructions')"
                  @blur="saveInstructions"
                />
                <p class="project-help">{{ t('projects.instructionsHelp') }}</p>
              </section>

              <section class="project-surface">
                <div class="project-surface__heading project-surface__heading--compact">
                  <div>
                    <p class="projects-kicker">{{ t('projects.settingsLabel') }}</p>
                    <h2>{{ t('projects.memory') }}</h2>
                  </div>
                </div>
                <p class="project-help project-help--top">{{ t('projects.memoryDescription') }}</p>
                <label class="project-select-label">
                  <span class="sr-only">{{ t('projects.memory') }}</span>
                  <select :value="selectedProject.memoryMode" @change="changeMemory">
                    <option value="default">{{ t('projects.memoryDefault') }}</option>
                    <option value="project-only">{{ t('projects.memoryProjectOnly') }}</option>
                  </select>
                </label>
              </section>

              <section class="project-surface">
                <div class="project-surface__heading project-surface__heading--compact">
                  <div>
                    <p class="projects-kicker">{{ t('projects.sourcesLabel') }}</p>
                    <h2>{{ t('projects.files') }}</h2>
                  </div>
                  <span class="project-count">{{ selectedProject.files.length }}</span>
                </div>
                <ul v-if="selectedProject.files.length" class="project-file-list">
                  <li v-for="file in selectedProject.files" :key="file.id">
                    <span class="project-file-list__icon" aria-hidden="true">↗</span>
                    <span class="project-file-list__name" :title="file.name">{{ file.name }}</span>
                    <button type="button" class="project-icon-button" :aria-label="t('projects.removeFile')" @click="removeFile(file.id)">
                      <Icon name="x" size="sm" aria-hidden="true" />
                    </button>
                  </li>
                </ul>
                <p v-else class="project-help">{{ t('projects.noFiles') }}</p>
                <label class="projects-button projects-button--quiet project-upload">
                  <Icon name="paperclip" size="sm" aria-hidden="true" /> {{ t('projects.addFiles') }}
                  <input type="file" multiple @change="addFiles" />
                </label>
                <p class="project-help">{{ t('projects.sourcesHelp') }}</p>
              </section>
            </aside>
          </div>
        </template>

        <template v-else>
          <section class="projects-intro">
            <div>
              <p class="projects-kicker">{{ t('projects.eyebrow') }}</p>
              <h1>{{ t('projects.title') }}</h1>
              <p>{{ t('projects.subtitle') }}</p>
            </div>
            <span class="projects-intro__glyph" aria-hidden="true">⌘</span>
          </section>
          <section v-if="projectsStore.projects.length" class="projects-grid" :aria-label="t('projects.title')">
            <article
              v-for="project in projectsStore.projects"
              :key="project.id"
              class="project-card"
              :style="{ '--project-accent': projectAccent(project) }"
            >
              <button type="button" class="project-card__open" @click="openProject(project.id)">
                <span class="project-card__topline">
                  <span class="project-card__icon" aria-hidden="true">{{ project.icon }}</span>
                  <span class="project-card__arrow" aria-hidden="true">↗</span>
                </span>
                <span class="project-card__name">{{ project.name }}</span>
                <span class="project-card__meta">
                  {{ project.conversationIds.length }} {{ t('projects.chats') }}
                  <span aria-hidden="true">·</span>
                  {{ project.files.length }} {{ t('projects.files') }}
                </span>
              </button>
              <div class="project-card__actions">
                <button type="button" :aria-label="t('projects.rename')" @click="openEdit(project)">
                  <Icon name="edit" size="sm" aria-hidden="true" />
                </button>
                <button type="button" :aria-label="t('projects.delete')" @click="removeProject(project.id)">
                  <Icon name="trash" size="sm" aria-hidden="true" />
                </button>
              </div>
            </article>
          </section>
          <section v-else class="projects-empty">
            <span class="projects-empty__icon" aria-hidden="true">✦</span>
            <h2>{{ t('projects.emptyTitle') }}</h2>
            <p>{{ t('projects.emptyDescription') }}</p>
            <button class="projects-button projects-button--primary" type="button" @click="openCreate">
              <Icon name="plus" size="sm" aria-hidden="true" /> {{ t('projects.createFirst') }}
            </button>
          </section>
        </template>
      </main>

      <div v-if="editorOpen" class="projects-modal" role="dialog" aria-modal="true" @click.self="closeEditor">
        <form class="projects-dialog" @submit.prevent="saveProject">
          <div class="projects-dialog__header">
            <div>
              <p class="projects-kicker">{{ editingId ? t('projects.editProject') : t('projects.newProject') }}</p>
              <h2>{{ editingId ? t('projects.editProject') : t('projects.newProject') }}</h2>
            </div>
            <button type="button" class="project-icon-button" :aria-label="t('common.cancel')" @click="closeEditor">
              <Icon name="x" size="sm" aria-hidden="true" />
            </button>
          </div>
          <label class="projects-field">
            <span>{{ t('projects.name') }}</span>
            <input v-model="draft.name" required maxlength="200" autocomplete="off" autofocus />
          </label>
          <div class="projects-field-row">
            <label class="projects-field">
              <span>{{ t('projects.icon') }}</span>
              <input v-model="draft.icon" maxlength="4" inputmode="text" />
            </label>
            <label class="projects-field">
              <span>{{ t('projects.color') }}</span>
              <input v-model="draft.color" type="color" />
            </label>
          </div>
          <label class="projects-field">
            <span>{{ t('projects.instructions') }}</span>
            <textarea v-model="draft.instructions" :placeholder="t('projects.instructionsPlaceholder')" rows="4" />
          </label>
          <label class="projects-field">
            <span>{{ t('projects.memory') }}</span>
            <select v-model="draft.memoryMode">
              <option value="default">{{ t('projects.memoryDefault') }}</option>
              <option value="project-only">{{ t('projects.memoryProjectOnly') }}</option>
            </select>
          </label>
          <div class="projects-dialog__actions">
            <button type="button" class="projects-button projects-button--quiet" @click="closeEditor">{{ t('common.cancel') }}</button>
            <button type="submit" class="projects-button projects-button--primary" :disabled="saving">
              {{ saving ? t('projects.saving') : t('common.save') }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import ChatHistoryPanel from '@/components/chat/ChatHistoryPanel.vue'
import Icon from '@/components/icons/Icon.vue'
import { useChatStore } from '@/stores/chat'
import { useLibraryStore } from '@/stores/library'
import { useProjectsStore } from '@/stores/projects'
import type { Project, ProjectConversation, ProjectMemoryMode } from '@/types/projects'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const chatStore = useChatStore()
const libraryStore = useLibraryStore()
const projectsStore = useProjectsStore()

const selectedProjectId = ref<string | null>(typeof route.params.projectId === 'string' ? route.params.projectId : null)
const editorOpen = ref(false)
const editingId = ref<string | null>(null)
const saving = ref(false)
const showChatPicker = ref(false)
const instructionsDraft = ref('')
const draft = ref({
  name: '',
  icon: '📁',
  color: '#7c3aed',
  instructions: '',
  memoryMode: 'default' as ProjectMemoryMode,
})

const selectedProject = computed(() => (
  projectsStore.projects.find(({ id }) => id === selectedProjectId.value) ?? null
))
const historyConversations = computed(() => chatStore.conversations)
const projectConversations = computed<ProjectConversation[]>(() => {
  if (!selectedProject.value) return []
  const localById = new Map(chatStore.conversations.map((conversation) => [conversation.id, conversation]))
  const result: ProjectConversation[] = []
  for (const conversation of selectedProject.value.conversations) {
    const local = localById.get(conversation.id)
    result.push(local ? {
      id: local.id,
      title: local.title,
      model: local.model,
      updatedAt: new Date(local.updatedAt).toISOString(),
    } : conversation)
  }
  for (const conversation of chatStore.conversations) {
    if (selectedProject.value.conversationIds.includes(conversation.id) && !result.some(({ id }) => id === conversation.id)) {
      result.push({ id: conversation.id, title: conversation.title, model: conversation.model, updatedAt: new Date(conversation.updatedAt).toISOString() })
    }
  }
  return result
})
const availableConversations = computed(() => chatStore.conversations.filter((conversation) => (
  !selectedProject.value?.conversationIds.includes(conversation.id)
)))

watch(selectedProject, (project) => {
  instructionsDraft.value = project?.instructions ?? ''
  showChatPicker.value = false
}, { immediate: true })

watch(() => route.params.projectId, (value) => {
  selectedProjectId.value = typeof value === 'string' ? value : null
  if (selectedProjectId.value) void projectsStore.get(selectedProjectId.value)
})

onMounted(async () => {
  await projectsStore.load()
  if (selectedProjectId.value) await projectsStore.get(selectedProjectId.value)
})

function projectAccent(project: Project): string {
  return /^#[\da-f]{3,8}$/i.test(project.color) ? project.color : '#7c3aed'
}

function formatConversationDate(value?: string): string {
  if (!value) return t('projects.recentlyUpdated')
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return t('projects.recentlyUpdated')
  return new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric' }).format(date)
}

function openCreate(): void {
  editingId.value = null
  draft.value = { name: '', icon: '📁', color: '#7c3aed', instructions: '', memoryMode: 'default' }
  editorOpen.value = true
}

function openEdit(project: Project): void {
  editingId.value = project.id
  draft.value = {
    name: project.name,
    icon: project.icon,
    color: projectAccent(project),
    instructions: project.instructions,
    memoryMode: project.memoryMode,
  }
  editorOpen.value = true
}

function closeEditor(): void {
  if (!saving.value) editorOpen.value = false
}

async function saveProject(): Promise<void> {
  saving.value = true
  try {
    if (editingId.value) {
      await projectsStore.update(editingId.value, draft.value)
    } else {
      const project = await projectsStore.create(draft.value)
      selectedProjectId.value = project.id
      await router.push(`/projects/${encodeURIComponent(project.id)}`)
    }
    editorOpen.value = false
  } finally {
    saving.value = false
  }
}

async function removeProject(id: string): Promise<void> {
  if (typeof window !== 'undefined' && window.confirm(t('projects.deleteConfirm'))) {
    const removed = await projectsStore.remove(id)
    if (removed && selectedProjectId.value === id) await goBack()
  }
}

async function openProject(id: string): Promise<void> {
  selectedProjectId.value = id
  await router.push(`/projects/${encodeURIComponent(id)}`)
  await projectsStore.get(id)
}

async function goBack(): Promise<void> {
  selectedProjectId.value = null
  await router.push('/projects')
}

function startNewChat(): void {
  void router.push({ path: '/chat', query: { conversation: 'new' } })
}

function startProjectChat(): void {
  if (!selectedProject.value) return startNewChat()
  void router.push({ path: '/chat', query: { conversation: 'new', project: selectedProject.value.id } })
}

function openConversation(id: string): void {
  void router.push({ path: '/chat', query: { conversation: id } })
}

async function saveInstructions(): Promise<void> {
  if (selectedProject.value && instructionsDraft.value !== selectedProject.value.instructions) {
    await projectsStore.update(selectedProject.value.id, { instructions: instructionsDraft.value })
  }
}

async function changeMemory(event: Event): Promise<void> {
  const value = (event.target as HTMLSelectElement).value as ProjectMemoryMode
  if (selectedProject.value && value !== selectedProject.value.memoryMode) {
    await projectsStore.setMemoryMode(selectedProject.value.id, value)
  }
}

async function addConversation(id: string): Promise<void> {
  if (!selectedProject.value) return
  const conversation = chatStore.conversations.find(({ id: conversationId }) => conversationId === id)
  if (await projectsStore.addConversation(selectedProject.value.id, id, conversation ? {
    id: conversation.id,
    title: conversation.title,
    model: conversation.model,
    updatedAt: new Date(conversation.updatedAt).toISOString(),
  } : undefined)) showChatPicker.value = false
}

async function removeConversation(id: string): Promise<void> {
  if (selectedProject.value) await projectsStore.removeConversation(selectedProject.value.id, id)
}

async function addFiles(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files ?? [])
  input.value = ''
  if (!selectedProject.value || files.length === 0) return
  const uploaded = await libraryStore.uploadFiles(files)
  await projectsStore.addFiles(selectedProject.value.id, uploaded.map((file) => ({
    id: file.id,
    name: file.name,
    size: file.size,
    mimeType: file.mimeType,
    createdAt: file.createdAt,
  })))
}

async function removeFile(id: string): Promise<void> {
  if (selectedProject.value) await projectsStore.removeFile(selectedProject.value.id, id)
}

async function retryLoad(): Promise<void> {
  await projectsStore.load()
  if (selectedProjectId.value) await projectsStore.get(selectedProjectId.value)
}
</script>

<style scoped>
.projects-shell {
  --projects-ink: #202022;
  --projects-muted: #78777c;
  --projects-line: rgba(32, 32, 34, .11);
  --projects-panel: rgba(255, 255, 255, .86);
  display: flex;
  min-height: calc(100vh - 26px);
  overflow: hidden;
  border-radius: 18px;
  background:
    radial-gradient(circle at 90% 0%, rgba(124, 58, 237, .08), transparent 28rem),
    #f7f7f8;
  color: var(--projects-ink);
}
.projects-shell__history { flex: 0 0 260px; }
.projects-content { flex: 1; min-width: 0; max-width: 1240px; margin: 0 auto; padding: 30px clamp(20px, 4vw, 62px) 64px; outline: none; }
.projects-topbar { display: flex; align-items: center; justify-content: space-between; gap: 20px; min-height: 44px; }
.projects-breadcrumbs { display: flex; align-items: center; min-height: 40px; }
.projects-kicker { margin: 0; color: var(--projects-muted); font-size: 11px; font-weight: 700; letter-spacing: .12em; line-height: 1.2; text-transform: uppercase; }
.projects-back { display: inline-flex; align-items: center; gap: 7px; padding: 8px 0; border: 0; background: transparent; color: var(--projects-muted); cursor: pointer; font-size: 13px; }
.projects-back:hover { color: var(--projects-ink); }
.projects-topbar__actions, .project-hero__actions { display: flex; align-items: center; gap: 8px; }
.projects-button { display: inline-flex; align-items: center; justify-content: center; gap: 8px; min-height: 38px; padding: 8px 13px; border: 1px solid transparent; border-radius: 9px; cursor: pointer; font: inherit; font-size: 13px; font-weight: 650; transition: background .16s ease, border-color .16s ease, transform .16s ease; }
.projects-button:hover:not(:disabled) { transform: translateY(-1px); }
.projects-button:focus-visible, .project-icon-button:focus-visible, .project-card__open:focus-visible, .project-chat-list__open:focus-visible, .project-picker button:focus-visible, .projects-field input:focus-visible, .projects-field textarea:focus-visible, .projects-field select:focus-visible { outline: 2px solid #7c3aed; outline-offset: 2px; }
.projects-button:disabled { cursor: wait; opacity: .55; }
.projects-button--primary { background: #242326; color: #fff; box-shadow: 0 4px 12px rgba(35, 33, 39, .14); }
.projects-button--primary:hover:not(:disabled) { background: #111014; }
.projects-button--quiet { border-color: var(--projects-line); background: rgba(255, 255, 255, .5); color: #4e4d53; }
.projects-button--quiet:hover:not(:disabled) { border-color: rgba(32, 32, 34, .2); background: #fff; }
.projects-notice { display: flex; align-items: center; gap: 9px; margin-top: 18px; padding: 10px 13px; border: 1px solid rgba(180, 125, 0, .2); border-radius: 10px; background: #fff8e3; color: #735500; font-size: 13px; }
.projects-notice button { margin-left: auto; border: 0; background: transparent; color: inherit; cursor: pointer; font-weight: 700; text-decoration: underline; }
.projects-loading { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 17px; margin-top: 46px; }
.projects-skeleton-card { height: 158px; border-radius: 14px; background: linear-gradient(90deg, #eeeef0 25%, #f8f8f9 40%, #eeeef0 60%); background-size: 220% 100%; animation: projects-shimmer 1.25s linear infinite; }
@keyframes projects-shimmer { to { background-position: -220% 0; } }
.projects-intro { display: flex; align-items: flex-end; justify-content: space-between; gap: 30px; margin: 48px 0 30px; }
.projects-intro h1, .project-hero h1 { margin: 8px 0 0; color: var(--projects-ink); font-size: clamp(28px, 4vw, 42px); font-weight: 680; letter-spacing: -.045em; line-height: 1.05; }
.projects-intro p:not(.projects-kicker) { max-width: 570px; margin: 12px 0 0; color: var(--projects-muted); font-size: 15px; line-height: 1.55; }
.projects-intro__glyph { display: grid; place-items: center; width: 74px; height: 74px; border: 1px solid rgba(124, 58, 237, .2); border-radius: 22px; background: rgba(124, 58, 237, .08); color: #7c3aed; font-size: 32px; transform: rotate(-8deg); }
.projects-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(228px, 1fr)); gap: 17px; }
.project-card { position: relative; min-height: 158px; overflow: hidden; border: 1px solid var(--projects-line); border-top: 3px solid var(--project-accent); border-radius: 14px; background: var(--projects-panel); box-shadow: 0 8px 26px rgba(26, 24, 31, .035); transition: box-shadow .18s ease, transform .18s ease; }
.project-card:hover { box-shadow: 0 14px 32px rgba(26, 24, 31, .08); transform: translateY(-2px); }
.project-card__open { display: flex; flex-direction: column; width: 100%; height: 100%; min-height: 158px; padding: 18px; border: 0; background: transparent; color: inherit; cursor: pointer; text-align: left; }
.project-card__topline { display: flex; align-items: center; justify-content: space-between; }
.project-card__icon { font-size: 28px; line-height: 1; }
.project-card__arrow { color: var(--projects-muted); font-size: 18px; opacity: .65; }
.project-card__name { margin-top: 23px; overflow: hidden; font-size: 16px; font-weight: 680; text-overflow: ellipsis; white-space: nowrap; }
.project-card__meta, .project-hero__meta { display: flex; align-items: center; gap: 7px; margin-top: 7px; color: var(--projects-muted); font-size: 12px; }
.project-card__actions { position: absolute; right: 10px; bottom: 10px; display: flex; gap: 2px; opacity: 0; transition: opacity .16s ease; }
.project-card:hover .project-card__actions, .project-card:focus-within .project-card__actions { opacity: 1; }
.project-card__actions button, .project-icon-button { display: inline-grid; place-items: center; width: 32px; height: 32px; padding: 0; border: 0; border-radius: 8px; background: transparent; color: var(--projects-muted); cursor: pointer; }
.project-card__actions button:hover, .project-icon-button:hover { background: rgba(32, 32, 34, .07); color: var(--projects-ink); }
.projects-empty { display: grid; justify-items: center; margin: 70px auto 0; padding: 70px 20px; border: 1px dashed rgba(32, 32, 34, .18); border-radius: 16px; text-align: center; }
.projects-empty__icon { display: grid; place-items: center; width: 58px; height: 58px; border-radius: 18px; background: rgba(124, 58, 237, .09); color: #7c3aed; font-size: 28px; }
.projects-empty h2 { margin: 19px 0 0; font-size: 20px; letter-spacing: -.02em; }
.projects-empty p { max-width: 360px; margin: 9px 0 20px; color: var(--projects-muted); font-size: 14px; line-height: 1.5; }
.project-hero { display: flex; align-items: center; justify-content: space-between; gap: 22px; margin: 42px 0 28px; padding: 24px 26px; border: 1px solid rgba(32, 32, 34, .09); border-radius: 18px; background: linear-gradient(110deg, rgba(255,255,255,.92), rgba(255,255,255,.65)); box-shadow: inset 0 3px 0 var(--project-accent), 0 10px 28px rgba(26, 24, 31, .04); }
.project-hero__identity { display: flex; align-items: center; gap: 16px; min-width: 0; }
.project-hero__icon { display: grid; flex: 0 0 auto; place-items: center; width: 58px; height: 58px; border-radius: 17px; background: color-mix(in srgb, var(--project-accent) 12%, white); font-size: 28px; }
.project-hero__meta { margin-top: 9px; }
.project-layout { display: grid; grid-template-columns: minmax(0, 1.45fr) minmax(300px, .8fr); gap: 17px; align-items: start; }
.project-layout__aside { display: grid; gap: 17px; }
.project-surface { min-width: 0; padding: 20px; border: 1px solid var(--projects-line); border-radius: 14px; background: var(--projects-panel); box-shadow: 0 8px 26px rgba(26, 24, 31, .035); }
.project-surface--chats { min-height: 420px; }
.project-surface__heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 15px; margin-bottom: 18px; }
.project-surface__heading--compact { margin-bottom: 14px; }
.project-surface h2 { margin: 6px 0 0; font-size: 18px; letter-spacing: -.025em; }
.project-status-dot { display: block; width: 8px; height: 8px; margin-top: 8px; border-radius: 50%; background: #d8d8da; }
.project-status-dot--set { background: #7c3aed; box-shadow: 0 0 0 4px rgba(124, 58, 237, .1); }
.project-count { display: inline-grid; place-items: center; min-width: 26px; height: 26px; padding: 0 7px; border-radius: 99px; background: rgba(32, 32, 34, .06); color: var(--projects-muted); font-size: 12px; }
.project-chat-list, .project-file-list { list-style: none; margin: 0; padding: 0; }
.project-chat-list li { display: flex; align-items: center; gap: 8px; border-top: 1px solid var(--projects-line); }
.project-chat-list__open { display: flex; align-items: center; flex: 1; min-width: 0; gap: 11px; padding: 13px 6px; border: 0; background: transparent; color: inherit; cursor: pointer; text-align: left; }
.project-chat-list__avatar { display: grid; flex: 0 0 auto; place-items: center; width: 30px; height: 30px; border-radius: 9px; background: rgba(124, 58, 237, .1); color: #7c3aed; font-size: 13px; }
.project-chat-list__copy { display: grid; min-width: 0; gap: 4px; }
.project-chat-list__copy strong { overflow: hidden; font-size: 14px; font-weight: 600; text-overflow: ellipsis; white-space: nowrap; }
.project-chat-list__copy small { color: var(--projects-muted); font-size: 11px; }
.project-picker { display: grid; gap: 6px; margin: -4px 0 14px; padding: 8px; border: 1px solid var(--projects-line); border-radius: 10px; background: rgba(247, 247, 248, .8); }
.project-picker button { display: flex; align-items: center; justify-content: space-between; gap: 10px; padding: 9px 10px; border: 0; border-radius: 7px; background: transparent; color: inherit; cursor: pointer; font: inherit; font-size: 13px; text-align: left; }
.project-picker button:hover { background: rgba(124, 58, 237, .08); }
.project-empty-inline { margin: 4px; color: var(--projects-muted); font-size: 12px; }
.project-empty-state { display: grid; justify-items: center; padding: 78px 20px 42px; color: var(--projects-muted); text-align: center; }
.project-empty-state__mark { color: #7c3aed; font-size: 28px; }
.project-empty-state p { margin: 9px 0 14px; font-size: 13px; }
.project-instructions { display: block; width: 100%; min-height: 118px; resize: vertical; padding: 11px 12px; border: 1px solid var(--projects-line); border-radius: 9px; background: rgba(255, 255, 255, .62); color: var(--projects-ink); font: inherit; font-size: 13px; line-height: 1.5; }
.project-help { margin: 9px 0 0; color: var(--projects-muted); font-size: 11px; line-height: 1.45; }
.project-help--top { margin: 0 0 12px; }
.project-select-label select { width: 100%; padding: 10px 11px; border: 1px solid var(--projects-line); border-radius: 9px; background: rgba(255, 255, 255, .62); color: var(--projects-ink); font: inherit; font-size: 13px; }
.project-file-list { margin-bottom: 13px; }
.project-file-list li { display: flex; align-items: center; gap: 8px; min-width: 0; padding: 8px 0; border-top: 1px solid var(--projects-line); }
.project-file-list__icon { color: #7c3aed; font-size: 13px; }
.project-file-list__name { flex: 1; min-width: 0; overflow: hidden; color: #454449; font-size: 12px; text-overflow: ellipsis; white-space: nowrap; }
.project-upload { width: 100%; }
.project-upload input { display: none; }
.projects-modal { position: fixed; z-index: 70; inset: 0; display: grid; place-items: center; padding: 20px; background: rgba(16, 14, 20, .48); backdrop-filter: blur(4px); }
.projects-dialog { width: min(480px, 100%); max-height: min(720px, calc(100vh - 40px)); overflow: auto; padding: 25px; border: 1px solid rgba(255, 255, 255, .25); border-radius: 18px; background: #fff; box-shadow: 0 24px 80px rgba(13, 11, 17, .25); }
.projects-dialog__header { display: flex; align-items: flex-start; justify-content: space-between; gap: 15px; margin-bottom: 23px; }
.projects-dialog h2 { margin: 7px 0 0; font-size: 23px; letter-spacing: -.035em; }
.projects-field { display: grid; gap: 7px; margin-top: 15px; color: #4b4a50; font-size: 12px; font-weight: 650; }
.projects-field input, .projects-field textarea, .projects-field select { width: 100%; padding: 10px 11px; border: 1px solid #dedde1; border-radius: 9px; background: #fff; color: #202022; font: inherit; font-size: 13px; font-weight: 400; }
.projects-field input[type='color'] { height: 40px; padding: 4px; cursor: pointer; }
.projects-field textarea { min-height: 92px; resize: vertical; line-height: 1.45; }
.projects-field-row { display: grid; grid-template-columns: 1fr 110px; gap: 13px; }
.projects-dialog__actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 25px; }
@media (prefers-reduced-motion: reduce) { .projects-button, .project-card, .project-card__actions { transition: none; } .projects-skeleton-card { animation: none; } }
@media (prefers-color-scheme: dark) { .projects-shell { --projects-ink: #f0eff2; --projects-muted: #a3a0aa; --projects-line: rgba(255,255,255,.12); --projects-panel: rgba(35,33,39,.86); background: radial-gradient(circle at 90% 0%, rgba(124,58,237,.18), transparent 28rem), #19181c; } .projects-intro h1, .project-hero h1 { color: var(--projects-ink); } .project-hero { background: linear-gradient(110deg, rgba(40,38,45,.95), rgba(34,32,39,.75)); border-color: var(--projects-line); } .project-card__name, .project-file-list__name { color: var(--projects-ink); } .project-instructions, .project-select-label select { background: rgba(19,18,22,.45); color: var(--projects-ink); } .projects-empty { border-color: var(--projects-line); } .projects-dialog { background: #29272d; } .projects-dialog h2, .projects-field, .projects-field input, .projects-field textarea, .projects-field select { color: #f0eff2; } .projects-field input, .projects-field textarea, .projects-field select { border-color: rgba(255,255,255,.16); background: #211f24; } }
@media (max-width: 980px) { .projects-layout { grid-template-columns: 1fr; } .project-layout { grid-template-columns: 1fr; } .project-layout__aside { grid-template-columns: repeat(3, minmax(0, 1fr)); } }
@media (max-width: 800px) { .projects-shell { min-height: 100vh; border-radius: 0; } .projects-shell__history { display: none; } .projects-content { padding: 22px 17px 44px; } .projects-intro { margin-top: 34px; } .projects-intro__glyph { width: 56px; height: 56px; border-radius: 17px; font-size: 24px; } .project-hero { align-items: flex-start; flex-direction: column; padding: 20px; } .project-hero__actions { width: 100%; } .project-hero__actions .projects-button { flex: 1; } .project-layout__aside { grid-template-columns: 1fr; } .projects-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 11px; } .project-card, .project-card__open { min-height: 145px; } .project-card__open { padding: 14px; } .project-card__actions { opacity: 1; } .projects-field-row { grid-template-columns: 1fr 90px; } }
@media (max-width: 450px) { .projects-grid { grid-template-columns: 1fr; } .projects-topbar__actions .projects-button span { display: none; } .projects-topbar__actions .projects-button { width: 38px; padding: 0; } .projects-intro { align-items: flex-start; } .projects-intro__glyph { display: none; } }
</style>

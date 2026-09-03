<template>
  <AppLayout variant="chat" shell-mode="chat">
    <div class="projects-workspace">
      <ChatHistoryPanel
        v-show="!mobileHistoryLayout && !narrowSidebar"
        shell
        active-section="projects"
        class="projects-workspace__history projects-workspace__history--desktop"
        :conversations="historyConversations"
        :projects="projectsStore.projects"
        :active-id="chatStore.activeConversationId"
        :search-query="historySearchQuery"
        :searching="chatStore.searchingHistory"
        :has-more="chatStore.conversationsHaveMore"
        :loading-more="chatStore.loadingConversationPage"
        @new="startNewChat"
        @select="openConversation"
        @rename="chatStore.renameConversation"
        @delete="confirmHistoryDelete"
        @clear="confirmHistoryClear"
        @update:search-query="updateHistorySearch"
        @search="runHistorySearch"
        @load-more="chatStore.loadConversationPage()"
      />

      <Transition name="projects-drawer">
        <div
          v-if="historyOpen && mobileHistoryLayout"
          ref="historyDrawerRef"
          class="projects-workspace__drawer"
          role="dialog"
          aria-modal="true"
          :aria-label="t('chat.history.title')"
          tabindex="-1"
          @keydown="onHistoryDrawerKeydown"
        >
          <button
            type="button"
            class="projects-workspace__scrim"
            :aria-label="t('chat.actions.closeHistory')"
            tabindex="-1"
            @click="closeHistory()"
          ></button>
          <ChatHistoryPanel
            shell
            mobile
            active-section="projects"
            :conversations="historyConversations"
            :projects="projectsStore.projects"
            :active-id="chatStore.activeConversationId"
            :search-query="historySearchQuery"
            :searching="chatStore.searchingHistory"
            :has-more="chatStore.conversationsHaveMore"
            :loading-more="chatStore.loadingConversationPage"
            @new="startNewChat"
            @close="closeHistory"
            @select="openConversation"
            @rename="chatStore.renameConversation"
            @delete="confirmHistoryDelete"
            @clear="confirmHistoryClear"
            @update:search-query="updateHistorySearch"
            @search="runHistorySearch"
            @load-more="chatStore.loadConversationPage()"
          />
        </div>
      </Transition>

      <WorkspaceSidebarOverlayLayer
        v-if="narrowSidebar"
        active
        :open="narrowSidebarOpen"
        :label="t('chat.history.title')"
        return-focus-id="workspace-sidebar-overlay-trigger"
        @close="closeHistory()"
      >
        <ChatHistoryPanel
          shell
          overlay
          active-section="projects"
          sidebar-id="workspace-chat-sidebar-overlay"
          :conversations="historyConversations"
          :projects="projectsStore.projects"
          :active-id="chatStore.activeConversationId"
          :search-query="historySearchQuery"
          :searching="chatStore.searchingHistory"
          :has-more="chatStore.conversationsHaveMore"
          :loading-more="chatStore.loadingConversationPage"
          @new="startNewChat"
          @close="closeHistory"
          @select="openConversation"
          @rename="chatStore.renameConversation"
          @delete="confirmHistoryDelete"
          @clear="confirmHistoryClear"
          @update:search-query="updateHistorySearch"
          @search="runHistorySearch"
          @load-more="chatStore.loadConversationPage()"
        />
      </WorkspaceSidebarOverlayLayer>

      <main
        ref="mainRef"
        class="projects-main"
        :class="{ 'projects-main--detail': routeHasProject }"
        :aria-hidden="historyModalActive || editorOpen ? 'true' : undefined"
        :inert="historyModalActive || editorOpen ? true : undefined"
        tabindex="-1"
      >
        <button
          ref="historyTriggerRef"
          type="button"
          class="projects-mobile-navigation"
          :aria-label="t('chat.history.title')"
          :title="t('chat.history.title')"
          @click="openHistory"
        >
          <Icon name="menu" size="sm" aria-hidden="true" />
        </button>

        <div class="projects-page">
          <div v-if="showProjectsNotice" class="projects-notice" role="alert">
            <Icon name="exclamationTriangle" size="sm" aria-hidden="true" />
            <span>{{ projectsStore.error }}</span>
            <button type="button" @click="retryLoad">{{ t('projects.retry') }}</button>
          </div>

          <template v-if="routeHasProject">
            <template v-if="selectedProject">
              <ProjectDetailHeader
                v-model:active-tab="activeTab"
                v-model:prompt="projectPrompt"
                :project="selectedProject"
                @submit="startProjectChat"
                @add-chat="showChatPicker = !showChatPicker"
                @settings="openEdit(selectedProject)"
                @share="shareProject"
                @delete="removeProject(selectedProject.id)"
                @back="goBack"
                @work="goToWork"
              />

              <section
                class="projects-detail-content"
                :class="{ 'projects-detail-content--sources': activeTab === 'sources' }"
                :aria-live="projectsStore.loading ? 'polite' : undefined"
              >
                <ProjectChatsPanel
                  v-if="activeTab === 'chats'"
                  :conversations="projectConversations"
                  :available-conversations="availableConversations"
                  :picker-open="showChatPicker"
                  @open="openConversation"
                  @remove="removeConversation"
                  @add="addConversation"
                  @update:picker-open="showChatPicker = $event"
                />
                <ProjectSourcesPanel
                  v-else
                  :files="selectedProject.files"
                  :uploading="uploadingSources"
                  @add-files="addFiles"
                  @remove="removeFile"
                />
              </section>
            </template>

            <div v-else-if="projectsStore.loading || !projectsStore.initialized" class="projects-detail-loading" aria-live="polite">
              <span class="projects-detail-loading__title" aria-hidden="true"></span>
              <span class="projects-detail-loading__composer" aria-hidden="true"></span>
              <span class="sr-only">{{ t('projects.loading') }}</span>
            </div>

            <section v-else class="projects-missing">
              <span aria-hidden="true">📁</span>
              <h1>{{ t('projects.projectNotFound') }}</h1>
              <p>{{ t('projects.projectNotFoundDescription') }}</p>
              <button type="button" @click="goBack">{{ t('projects.allProjects') }}</button>
            </section>
          </template>

          <ProjectsDirectory
            v-else
            v-model:search="projectSearch"
            v-model:filter="projectFilter"
            :projects="projectsStore.projects"
            :loading="projectsStore.loading && !projectsStore.initialized"
            @create="openCreate"
            @open="openProject"
            @edit="openEdit"
            @delete="removeProject"
          />
        </div>
      </main>

      <BaseDialog
        :show="editorOpen"
        :title="editingId ? t('projects.projectSettings') : t('projects.createProject')"
        width="normal"
        :variant="editingId ? 'default' : 'project-create'"
        :close-button-label="t('common.close')"
        :close-on-click-outside="!saving"
        :close-on-escape="!saving"
        :show-close-button="!saving"
        @close="closeEditor"
      >
        <form
          id="project-editor-form"
          class="projects-editor"
          :class="{ 'projects-create-editor': !editingId }"
          @submit.prevent="saveProject"
        >
          <template v-if="!editingId">
            <label class="projects-create-field">
              <span>{{ t('projects.projectName') }}</span>
              <span class="projects-create-input-wrap">
                <button
                  type="button"
                  class="projects-create-icon-trigger"
                  :aria-label="projectIconTriggerLabel"
                  aria-haspopup="menu"
                  :aria-expanded="iconMenuOpen"
                  @click="toggleIconMenu"
                >
                  <Icon
                    v-if="usesDefaultDraftIcon"
                    name="projectDefault"
                    size="md"
                    aria-hidden="true"
                  />
                  <span v-else aria-hidden="true">{{ draft.icon }}</span>
                </button>
                <input
                  v-model="draft.name"
                  required
                  maxlength="200"
                  autocomplete="off"
                  :placeholder="t('projects.projectNamePlaceholder')"
                />

                <div
                  v-if="iconMenuOpen"
                  class="projects-create-icon-menu"
                  role="menu"
                  :aria-label="t('projects.iconMenuLabel')"
                  @click.stop
                >
                  <div class="projects-create-color-grid" role="group" :aria-label="t('projects.color')">
                    <button
                      v-for="color in projectColors"
                      :key="color.value"
                      type="button"
                      class="projects-create-color-swatch"
                      :class="{ 'is-selected': draft.color === color.value }"
                      role="menuitemradio"
                      :aria-label="t(color.labelKey)"
                      :aria-checked="draft.color === color.value"
                      :style="{ '--swatch-color': color.value }"
                      @click="selectProjectColor(color.value)"
                    ></button>
                  </div>
                  <button
                    type="button"
                    class="projects-create-custom-color"
                    @click="openCustomColorPicker"
                  >
                    <span class="projects-create-custom-color__wheel" aria-hidden="true"></span>
                    {{ t('projects.customColor') }}
                  </button>
                  <input
                    ref="customColorInput"
                    class="projects-create-visually-hidden"
                    type="color"
                    :value="draft.color"
                    :aria-label="t('projects.customColor')"
                    @input="applyCustomColor"
                  >
                  <div class="projects-create-icon-divider" aria-hidden="true"></div>
                  <div class="projects-create-icon-grid" role="group" :aria-label="t('projects.iconMenuLabel')">
                    <button
                      v-for="icon in projectIconOptions"
                      :key="icon.value"
                      type="button"
                      class="projects-create-icon-option"
                      :class="{ 'is-selected': draft.icon === icon.value }"
                      role="menuitemradio"
                      :aria-label="t(icon.labelKey)"
                      :aria-checked="draft.icon === icon.value"
                      @click="selectProjectIcon(icon.value)"
                    >
                      <span aria-hidden="true">{{ icon.glyph }}</span>
                    </button>
                  </div>
                </div>
              </span>
            </label>

            <aside class="projects-create-info">
              <Icon name="lightbulb" size="md" aria-hidden="true" />
              <p>{{ t('projects.createInfo') }}</p>
            </aside>
          </template>

          <template v-else>
            <label class="projects-field">
              <span>{{ t('projects.name') }}</span>
              <input v-model="draft.name" required maxlength="200" autocomplete="off" />
            </label>

            <div class="projects-field-row">
              <label class="projects-field projects-field--icon">
                <span>{{ t('projects.icon') }}</span>
                <input v-model="draft.icon" maxlength="4" inputmode="text" />
              </label>
              <label class="projects-field projects-field--color">
                <span>{{ t('projects.color') }}</span>
                <span class="projects-color-control">
                  <input v-model="draft.color" type="color" />
                  <span>{{ draft.color }}</span>
                </span>
              </label>
            </div>

            <label class="projects-field">
              <span>{{ t('projects.instructions') }}</span>
              <textarea
                v-model="draft.instructions"
                :placeholder="t('projects.instructionsPlaceholder')"
                rows="5"
              ></textarea>
              <small>{{ t('projects.instructionsHelp') }}</small>
            </label>

            <label class="projects-field">
              <span>{{ t('projects.memory') }}</span>
              <select v-model="draft.memoryMode">
                <option value="default">{{ t('projects.memoryDefault') }}</option>
                <option value="project-only">{{ t('projects.memoryProjectOnly') }}</option>
              </select>
              <small>{{ t('projects.memoryDescription') }}</small>
            </label>
          </template>
        </form>

        <template #footer>
          <template v-if="!editingId">
            <div class="projects-create-footer">
              <div class="projects-create-memory-wrap">
                <button
                  type="button"
                  class="projects-create-memory-trigger"
                  aria-haspopup="menu"
                  :aria-expanded="memoryMenuOpen"
                  @click="toggleMemoryMenu"
                >
                  <span>{{ memoryModeLabel }}</span>
                  <Icon name="chevronDown" size="sm" aria-hidden="true" />
                </button>
                <div
                  v-if="memoryMenuOpen"
                  class="projects-create-memory-menu"
                  role="menu"
                  :aria-label="t('projects.memoryRangeLabel')"
                  @click.stop
                >
                  <button
                    v-for="option in memoryOptions"
                    :key="option.value"
                    type="button"
                    class="projects-create-memory-option"
                    :class="{ 'is-selected': draft.memoryMode === option.value }"
                    role="menuitemradio"
                    :aria-checked="draft.memoryMode === option.value"
                    @click="selectMemoryMode(option.value)"
                  >
                    <span class="projects-create-memory-copy">
                      <strong>{{ option.label }}</strong>
                      <small>{{ option.description }}</small>
                    </span>
                    <span v-if="draft.memoryMode === option.value" class="projects-create-memory-check" aria-hidden="true">✓</span>
                  </button>
                </div>
              </div>
              <button
                type="submit"
                form="project-editor-form"
                class="projects-create-submit"
                :disabled="saving || !draft.name.trim()"
              >
                {{ saving ? t('projects.saving') : t('projects.createProject') }}
              </button>
            </div>
          </template>
          <template v-else>
            <button type="button" class="projects-dialog-button" :disabled="saving" @click="closeEditor">
              {{ t('common.cancel') }}
            </button>
            <button
              type="submit"
              form="project-editor-form"
              class="projects-dialog-button projects-dialog-button--primary"
              :disabled="saving || !draft.name.trim()"
            >
              {{ saving ? t('projects.saving') : t('common.save') }}
            </button>
          </template>
        </template>
      </BaseDialog>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ChatHistoryPanel from '@/components/chat/ChatHistoryPanel.vue'
import Icon from '@/components/icons/Icon.vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import WorkspaceSidebarOverlayLayer from '@/components/layout/WorkspaceSidebarOverlayLayer.vue'
import { useWorkspaceResponsiveState } from '@/components/layout/workspaceResponsive'
import ProjectChatsPanel from '@/components/projects/ProjectChatsPanel.vue'
import ProjectDetailHeader from '@/components/projects/ProjectDetailHeader.vue'
import ProjectSourcesPanel from '@/components/projects/ProjectSourcesPanel.vue'
import ProjectsDirectory from '@/components/projects/ProjectsDirectory.vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useChatStore } from '@/stores/chat'
import { useLibraryStore } from '@/stores/library'
import { useProjectsStore } from '@/stores/projects'
import type { Project, ProjectConversation, ProjectMemoryMode } from '@/types/projects'

type ProjectTab = 'chats' | 'sources'
type ProjectFilter = 'all' | 'mine' | 'shared'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()
const chatStore = useChatStore()
const libraryStore = useLibraryStore()
const projectsStore = useProjectsStore()
const { mobileDrawer: mobileHistoryLayout, narrowSidebar } = useWorkspaceResponsiveState()

const mainRef = ref<HTMLElement | null>(null)
const historyDrawerRef = ref<HTMLElement | null>(null)
const historyTriggerRef = ref<HTMLButtonElement | null>(null)
const selectedProjectId = ref<string | null>(routeProjectId())
const activeTab = ref<ProjectTab>('chats')
const projectPrompt = ref('')
const projectSearch = ref('')
const projectFilter = ref<ProjectFilter>('all')
const editorOpen = ref(false)
const editingId = ref<string | null>(null)
const saving = ref(false)
const showChatPicker = ref(false)
const uploadingSources = ref(false)
const historyOpen = ref(false)
const historySearchQuery = ref('')
const draft = ref({
  name: '',
  icon: '📁',
  color: '#0d0d0d',
  instructions: '',
  memoryMode: 'default' as ProjectMemoryMode,
})

const iconMenuOpen = ref(false)
const memoryMenuOpen = ref(false)
const customColorInput = ref<HTMLInputElement | null>(null)

const projectColors = [
  { value: '#0d0d0d', labelKey: 'projects.colorDefault' },
  { value: '#ef4444', labelKey: 'projects.colorRed' },
  { value: '#f97316', labelKey: 'projects.colorOrange' },
  { value: '#eab308', labelKey: 'projects.colorYellow' },
  { value: '#22c55e', labelKey: 'projects.colorGreen' },
  { value: '#3b82f6', labelKey: 'projects.colorBlue' },
  { value: '#8b5cf6', labelKey: 'projects.colorPurple' },
  { value: '#ec4899', labelKey: 'projects.colorPink' },
] as const

const projectIconOptions = [
  { value: '📁', glyph: '▱', labelKey: 'projects.iconFolder' },
  { value: '💵', glyph: '$', labelKey: 'projects.iconMoney' },
  { value: '📚', glyph: '▤', labelKey: 'projects.iconBooks' },
  { value: '✏️', glyph: '✎', labelKey: 'projects.iconPencil' },
  { value: '📝', glyph: '▧', labelKey: 'projects.iconWriting' },
  { value: '</>', glyph: '</>', labelKey: 'projects.iconCode' },
  { value: '⌘', glyph: '⌘', labelKey: 'projects.iconTerminal' },
  { value: '🎵', glyph: '♫', labelKey: 'projects.iconMusic' },
  { value: '🍿', glyph: '◉', labelKey: 'projects.iconPopcorn' },
  { value: '🩺', glyph: '+', labelKey: 'projects.iconHealth' },
  { value: '✈️', glyph: '✈', labelKey: 'projects.iconPlane' },
  { value: '🌐', glyph: '◎', labelKey: 'projects.iconGlobe' },
  { value: '🔧', glyph: '⌕', labelKey: 'projects.iconTool' },
  { value: '🧪', glyph: '◇', labelKey: 'projects.iconFlask' },
  { value: '♥️', glyph: '♥', labelKey: 'projects.iconHeart' },
  { value: '🌱', glyph: '❧', labelKey: 'projects.iconPlant' },
  { value: '✓', glyph: '✓', labelKey: 'projects.iconCheck' },
] as const

const memoryOptions = computed(() => [
  {
    value: 'default' as const,
    label: t('projects.memoryDefault'),
    description: t('projects.memoryDefaultDescription'),
  },
  {
    value: 'project-only' as const,
    label: t('projects.memoryProjectOnly'),
    description: t('projects.memoryProjectOnlyDescription'),
  },
])

let historySearchTimer: ReturnType<typeof setTimeout> | null = null
let loadSequence = 0
let disposed = false

const routeHasProject = computed(() => selectedProjectId.value !== null)
const showProjectsNotice = computed(() => (
  Boolean(projectsStore.error)
  && (
    routeHasProject.value
    || projectsStore.projects.length > 0
    || projectSearch.value.trim() !== ''
    || projectFilter.value !== 'all'
  )
))
const usesDefaultDraftIcon = computed(() => (
  !draft.value.icon || draft.value.icon === 'folder' || draft.value.icon === '📁'
))
const memoryModeLabel = computed(() => (
  memoryOptions.value.find(({ value }) => value === draft.value.memoryMode)?.label
  ?? t('projects.memoryDefault')
))
const projectColorLabel = computed(() => (
  draft.value.color === '#0d0d0d'
    ? t('projects.defaultColorLabel')
    : (() => {
        const color = projectColors.find(({ value }) => value === draft.value.color)
        return color ? t(color.labelKey) : draft.value.color
      })()
))
const projectIconTriggerLabel = computed(() => (
  t('projects.iconTrigger', {
    icon: usesDefaultDraftIcon.value ? t('projects.defaultFolderIcon') : draft.value.icon,
    color: projectColorLabel.value,
  })
))
const selectedProject = computed(() => (
  projectsStore.projects.find(({ id }) => id === selectedProjectId.value) ?? null
))
const historyConversations = computed(() => (
  historySearchQuery.value.trim() ? chatStore.searchResults : chatStore.conversations
))
const narrowSidebarOpen = computed(() => appStore.workspaceNarrowSidebarOpen)
const historyModalActive = computed(() => (
  (historyOpen.value && mobileHistoryLayout.value)
  || (narrowSidebar.value && narrowSidebarOpen.value)
))
const projectConversations = computed<ProjectConversation[]>(() => {
  if (!selectedProject.value) return []
  const localById = new Map(chatStore.conversations.map((conversation) => [conversation.id, conversation]))
  const merged = new Map<string, ProjectConversation>()

  for (const conversation of selectedProject.value.conversations) {
    const local = localById.get(conversation.id)
    merged.set(conversation.id, local ? toProjectConversation(local) : conversation)
  }
  for (const id of selectedProject.value.conversationIds) {
    const local = localById.get(id)
    if (local) merged.set(id, toProjectConversation(local))
  }

  return [...merged.values()].sort((left, right) => (
    dateTimestamp(right.updatedAt) - dateTimestamp(left.updatedAt)
  ))
})
const availableConversations = computed<ProjectConversation[]>(() => {
  const selectedIds = new Set(selectedProject.value?.conversationIds ?? [])
  return chatStore.conversations
    .filter(({ id }) => !selectedIds.has(id))
    .map(toProjectConversation)
    .sort((left, right) => dateTimestamp(right.updatedAt) - dateTimestamp(left.updatedAt))
})

watch(() => route.params.projectId, async () => {
  const id = routeProjectId()
  selectedProjectId.value = id
  projectsStore.select(id)
  activeTab.value = 'chats'
  projectPrompt.value = ''
  showChatPicker.value = false
  if (id && projectsStore.initialized) await projectsStore.get(id)
})

watch(selectedProject, () => {
  showChatPicker.value = false
})

watch(mobileHistoryLayout, (mobile) => {
  if (!mobile && historyOpen.value) void closeHistory(mainRef.value)
})

watch(() => authStore.user?.id, async (userId) => {
  const sequence = ++loadSequence
  await Promise.all([
    projectsStore.load(),
    chatStore.hydrate(userId),
  ])
  if (disposed || sequence !== loadSequence) return

  if (userId !== null && userId !== undefined) {
    await chatStore.syncHistory()
    if (disposed || sequence !== loadSequence) return
    await chatStore.loadConversationPage(true)
  }
  if (disposed || sequence !== loadSequence) return

  const id = selectedProjectId.value
  projectsStore.select(id)
  if (id) await projectsStore.get(id)
}, { immediate: true, flush: 'sync' })

onBeforeUnmount(() => {
  disposed = true
  if (historySearchTimer) clearTimeout(historySearchTimer)
  document.removeEventListener('pointerdown', handleProjectEditorPointerdown, true)
  document.removeEventListener('keydown', handleProjectEditorEscape)
})

onMounted(() => {
  document.addEventListener('pointerdown', handleProjectEditorPointerdown, true)
  document.addEventListener('keydown', handleProjectEditorEscape)
})

function routeProjectId(): string | null {
  return typeof route.params.projectId === 'string' && route.params.projectId.trim()
    ? route.params.projectId
    : null
}

function dateTimestamp(value?: string): number {
  if (!value) return 0
  const timestamp = new Date(value).getTime()
  return Number.isFinite(timestamp) ? timestamp : 0
}

function toProjectConversation(conversation: {
  id: string
  title: string
  model?: string
  updatedAt: string | number | Date
}): ProjectConversation {
  const updatedAt = new Date(conversation.updatedAt)
  return {
    id: conversation.id,
    title: conversation.title,
    model: conversation.model,
    updatedAt: Number.isNaN(updatedAt.getTime()) ? undefined : updatedAt.toISOString(),
  }
}

function projectAccent(project: Project): string {
  if (/^#[\da-f]{3,8}$/i.test(project.color)) return project.color
  const namedColors: Record<string, string> = {
    gray: '#0d0d0d',
    black: '#0d0d0d',
    red: '#ef4444',
    orange: '#f97316',
    yellow: '#eab308',
    green: '#22c55e',
    blue: '#3b82f6',
    purple: '#8b5cf6',
    pink: '#ec4899',
  }
  return namedColors[project.color.trim().toLowerCase()] ?? '#0d0d0d'
}

function openCreate(): void {
  editingId.value = null
  draft.value = { name: '', icon: '📁', color: '#0d0d0d', instructions: '', memoryMode: 'default' }
  iconMenuOpen.value = false
  memoryMenuOpen.value = false
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
  iconMenuOpen.value = false
  memoryMenuOpen.value = false
  editorOpen.value = true
}

function closeEditor(): void {
  if (!saving.value) {
    iconMenuOpen.value = false
    memoryMenuOpen.value = false
    editorOpen.value = false
  }
}

function toggleIconMenu(): void {
  if (saving.value) return
  iconMenuOpen.value = !iconMenuOpen.value
  if (iconMenuOpen.value) memoryMenuOpen.value = false
}

function selectProjectIcon(value: string): void {
  draft.value.icon = value
  iconMenuOpen.value = false
}

function selectProjectColor(value: string): void {
  draft.value.color = value
}

function openCustomColorPicker(): void {
  customColorInput.value?.click()
}

function applyCustomColor(event: Event): void {
  const input = event.target
  if (input instanceof HTMLInputElement && input.value) draft.value.color = input.value
}

function toggleMemoryMenu(): void {
  if (saving.value) return
  memoryMenuOpen.value = !memoryMenuOpen.value
  if (memoryMenuOpen.value) iconMenuOpen.value = false
}

function selectMemoryMode(value: ProjectMemoryMode): void {
  draft.value.memoryMode = value
  memoryMenuOpen.value = false
}

function handleProjectEditorPointerdown(event: PointerEvent): void {
  if (!iconMenuOpen.value && !memoryMenuOpen.value) return
  const target = event.target
  if (!(target instanceof Node)) return
  if ((target as Element).closest('.projects-create-input-wrap, .projects-create-memory-wrap')) return
  iconMenuOpen.value = false
  memoryMenuOpen.value = false
}

function handleProjectEditorEscape(event: KeyboardEvent): void {
  if (event.key !== 'Escape') return
  if (iconMenuOpen.value || memoryMenuOpen.value) {
    event.preventDefault()
    // BaseDialog also listens on document. Stop listeners registered after
    // this project-level handler so the first Escape only closes the menu;
    // a second Escape can then dismiss the dialog itself.
    event.stopImmediatePropagation()
    iconMenuOpen.value = false
    memoryMenuOpen.value = false
  }
}

async function saveProject(): Promise<void> {
  if (saving.value || !draft.value.name.trim()) return
  saving.value = true
  try {
    if (editingId.value) {
      await projectsStore.update(editingId.value, { ...draft.value, name: draft.value.name.trim() })
    } else {
      const project = await projectsStore.create({ ...draft.value, name: draft.value.name.trim() })
      await router.push(`/projects/${encodeURIComponent(project.id)}`)
    }
    editorOpen.value = false
  } finally {
    saving.value = false
  }
}

async function removeProject(id: string): Promise<void> {
  if (typeof window !== 'undefined' && !window.confirm(t('projects.deleteConfirm'))) return
  const removed = await projectsStore.remove(id)
  if (removed && selectedProjectId.value === id) await goBack()
}

async function openProject(id: string): Promise<void> {
  await router.push(`/projects/${encodeURIComponent(id)}`)
}

async function goBack(): Promise<void> {
  selectedProjectId.value = null
  projectsStore.select(null)
  await router.push('/projects')
  await nextTick()
  mainRef.value?.focus({ preventScroll: true })
}

async function goToWork(): Promise<void> {
  await router.push('/dashboard')
}

async function startNewChat(): Promise<void> {
  chatStore.selectConversation(null)
  await closeHistory(null)
  await router.push({ path: '/chat', query: { conversation: 'new' } })
}

async function startProjectChat(prompt = projectPrompt.value): Promise<void> {
  if (!selectedProject.value) return startNewChat()
  const normalizedPrompt = prompt.trim()
  chatStore.selectConversation(null)
  await closeHistory(null)
  await router.push({
    path: '/chat',
    query: {
      conversation: 'new',
      project: selectedProject.value.id,
      ...(normalizedPrompt ? { prompt: normalizedPrompt } : {}),
    },
  })
}

async function openConversation(id: string): Promise<void> {
  chatStore.selectConversation(id)
  await closeHistory(null)
  await router.push({ path: '/chat', query: { conversation: id } })
}

async function addConversation(id: string): Promise<void> {
  if (!selectedProject.value) return
  const conversation = chatStore.conversations.find(({ id: conversationId }) => conversationId === id)
  const added = await projectsStore.addConversation(
    selectedProject.value.id,
    id,
    conversation ? toProjectConversation(conversation) : undefined,
  )
  if (added) showChatPicker.value = false
}

async function removeConversation(id: string): Promise<void> {
  if (selectedProject.value) await projectsStore.removeConversation(selectedProject.value.id, id)
}

async function addFiles(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files ?? [])
  input.value = ''
  if (!selectedProject.value || files.length === 0 || uploadingSources.value) return
  uploadingSources.value = true
  try {
    const uploaded = await libraryStore.uploadFiles(files)
    await projectsStore.addFiles(selectedProject.value.id, uploaded.map((file) => ({
      id: file.id,
      name: file.name,
      size: file.size,
      mimeType: file.mimeType,
      createdAt: file.createdAt,
    })))
  } finally {
    uploadingSources.value = false
  }
}

async function removeFile(id: string): Promise<void> {
  if (selectedProject.value) await projectsStore.removeFile(selectedProject.value.id, id)
}

async function shareProject(): Promise<void> {
  if (!selectedProject.value || typeof window === 'undefined') return

  const shareData = {
    title: selectedProject.value.name,
    url: window.location.href,
  }

  try {
    if (typeof navigator.share === 'function') {
      await navigator.share(shareData)
      return
    }
    if (!navigator.clipboard?.writeText) throw new Error('Clipboard unavailable')
    await navigator.clipboard.writeText(shareData.url)
    appStore.showSuccess(t('projects.shareCopied'))
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') return
    appStore.showError(t('projects.shareFailed'))
  }
}

async function retryLoad(): Promise<void> {
  await projectsStore.load()
  if (selectedProjectId.value) await projectsStore.get(selectedProjectId.value)
}

function updateHistorySearch(value: string): void {
  historySearchQuery.value = value
  if (historySearchTimer) clearTimeout(historySearchTimer)
  historySearchTimer = setTimeout(() => {
    historySearchTimer = null
    void runHistorySearch()
  }, 250)
}

async function runHistorySearch(): Promise<void> {
  if (historySearchTimer) clearTimeout(historySearchTimer)
  historySearchTimer = null
  await chatStore.searchHistory(historySearchQuery.value)
}

async function openHistory(): Promise<void> {
  if (mobileHistoryLayout.value) {
    historyOpen.value = true
    await nextTick()
    const first = historyDrawerRef.value?.querySelector<HTMLElement>(
      'a[href], button:not([disabled]), input:not([disabled]), [tabindex]:not([tabindex="-1"])',
    )
    ;(first ?? historyDrawerRef.value)?.focus({ preventScroll: true })
    return
  }
  if (narrowSidebar.value) appStore.setWorkspaceNarrowSidebarOpen(true)
}

async function closeHistory(focusTarget: HTMLElement | null = historyTriggerRef.value): Promise<void> {
  const mobileWasOpen = historyOpen.value
  if (mobileWasOpen) historyOpen.value = false
  if (narrowSidebarOpen.value) appStore.setWorkspaceNarrowSidebarOpen(false)
  await nextTick()
  if (mobileWasOpen) focusTarget?.focus({ preventScroll: true })
}

function onHistoryDrawerKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape') {
    event.preventDefault()
    void closeHistory()
    return
  }
  if (event.key !== 'Tab') return
  const controls = Array.from(historyDrawerRef.value?.querySelectorAll<HTMLElement>(
    'a[href]:not([tabindex="-1"]), button:not([disabled]):not([tabindex="-1"]), input:not([disabled]), [tabindex]:not([tabindex="-1"])',
  ) ?? []).filter((element) => element.getClientRects().length > 0)
  if (controls.length === 0) return
  const first = controls[0]
  const last = controls[controls.length - 1]
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault()
    last?.focus({ preventScroll: true })
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first?.focus({ preventScroll: true })
  }
}

function confirmHistoryDelete(id: string): void {
  const conversation = historyConversations.value.find((item) => item.id === id)
  const title = conversation?.title.trim() || t('chat.actions.newChat')
  if (typeof window === 'undefined' || window.confirm(
    `${t('chat.confirm.deleteDescriptionPrefix')}${title}${t('chat.confirm.deleteDescriptionSuffix')}`,
  )) chatStore.deleteConversation(id)
}

function confirmHistoryClear(): void {
  if (typeof window === 'undefined' || window.confirm(t('chat.confirm.clearDescription'))) {
    chatStore.clearConversations()
  }
}
</script>

<style scoped>
.projects-workspace {
  --projects-page: #fff;
  --projects-surface: #f4f4f4;
  --projects-surface-hover: #ececec;
  --projects-text: #0d0d0d;
  --projects-text-secondary: #5d5d5d;
  --projects-text-muted: #7d7d7d;
  --projects-border: rgba(0, 0, 0, 0.1);
  --projects-border-strong: rgba(0, 0, 0, 0.14);
  --projects-popover: #fff;
  --projects-danger: #d00e17;
  display: flex;
  width: 100%;
  height: 100%;
  min-height: 0;
  overflow: hidden;
  color: var(--projects-text);
  background: var(--projects-page);
}
:global(html.dark .projects-workspace) {
  --projects-page: #000;
  --projects-surface: #212121;
  --projects-surface-hover: #2f2f2f;
  --projects-text: #f2f2f2;
  --projects-text-secondary: #b4b4b4;
  --projects-text-muted: #8f8f8f;
  --projects-border: rgba(255, 255, 255, 0.1);
  --projects-border-strong: rgba(255, 255, 255, 0.15);
  --projects-popover: #2f2f2f;
  --projects-danger: #ff6767;
}
.projects-workspace__history { flex: 0 0 auto; }
.projects-workspace__history :deep(.app-mode-switch) { display: none; }
.projects-main { position: relative; flex: 1; min-width: 0; min-height: 0; overflow-x: hidden; overflow-y: auto; color: var(--projects-text); background: var(--projects-page); outline: none; }
.projects-page { width: min(100%, 800px); min-height: 100%; margin: 0 auto; padding: 116px 16px 72px; }
.projects-main--detail .projects-page { width: 100%; max-width: none; padding: 0 0 96px; }
.projects-mobile-navigation { display: none; }
.projects-notice { display: flex; min-height: 42px; align-items: center; gap: 9px; margin-bottom: 18px; border: 1px solid color-mix(in srgb, #b7791f 32%, transparent); border-radius: 12px; padding: 8px 12px; color: var(--projects-text); background: color-mix(in srgb, #f6ad55 13%, var(--projects-page)); font-size: 13px; }
.projects-notice span { min-width: 0; flex: 1; }
.projects-notice button { border: 0; padding: 5px; color: inherit; background: transparent; font: inherit; font-weight: 600; cursor: pointer; }
.projects-detail-content { width: min(100%, 800px); margin: 18px auto 0; padding: 0 16px; }
.projects-detail-loading { display: grid; gap: 24px; padding-top: 72px; }
.projects-detail-loading span { display: block; overflow: hidden; background: var(--projects-surface); }
.projects-detail-loading__title { width: 280px; height: 36px; border-radius: 10px; }
.projects-detail-loading__composer { width: 100%; height: 52px; border-radius: 28px; }
.projects-missing { display: grid; min-height: 520px; place-items: center; align-content: center; text-align: center; }
.projects-missing > span { font-size: 34px; filter: grayscale(1); }
.projects-missing h1 { margin: 18px 0 0; font-size: 22px; font-weight: 600; }
.projects-missing p { margin: 8px 0 20px; color: var(--projects-text-secondary); font-size: 14px; }
.projects-missing button,
.projects-dialog-button { min-height: 38px; border: 1px solid var(--projects-border-strong); border-radius: 999px; padding: 0 16px; color: var(--projects-text); background: transparent; font: inherit; font-size: 14px; font-weight: 500; cursor: pointer; }
.projects-missing button:hover,
.projects-dialog-button:hover:not(:disabled) { background: var(--projects-surface-hover); }
.projects-editor { display: grid; gap: 18px; color: var(--projects-text); }
.projects-create-editor {
  display: block;
  position: relative;
  color: #0d0d0d;
}
.projects-create-field {
  display: block;
  color: #0d0d0d;
  font-size: 14px;
  font-weight: 400;
  line-height: 20px;
}
.projects-create-input-wrap {
  position: relative;
  display: block;
  height: 36px;
  margin-top: 8px;
}
.projects-create-input-wrap > input {
  box-sizing: border-box;
  display: block;
  width: 100%;
  height: 36px;
  border: 1px solid #0d0d0d;
  border-radius: 8px;
  padding: 8px 12px 8px 36px;
  color: #0d0d0d;
  background: #fff;
  font: inherit;
  font-size: 14px;
  line-height: 20px;
  outline: none;
}
.projects-create-input-wrap > input::placeholder { color: #8f8f8f; opacity: 1; }
.projects-create-input-wrap > input:focus-visible {
  outline: none;
}
.projects-create-icon-trigger {
  position: absolute;
  z-index: 2;
  top: 0;
  left: 0;
  display: grid;
  width: 36px;
  height: 36px;
  place-items: center;
  border: 0;
  border-radius: 8px 0 0 8px;
  padding: 0;
  color: #8f8f8f;
  background: transparent;
  cursor: pointer;
}
.projects-create-icon-trigger:hover,
.projects-create-icon-trigger[aria-expanded='true'] { color: #0d0d0d; background: #f3f3f3; }
.projects-create-icon-trigger:focus { outline: none; }
.projects-create-icon-trigger:focus-visible { outline: 2px solid #0d0d0d; outline-offset: 1px; }
.projects-create-icon-trigger > span:not(.projects-create-color-dot) {
  display: grid;
  min-width: 20px;
  place-items: center;
  font-size: 18px;
  line-height: 20px;
}
.projects-create-icon-trigger svg { width: 20px; height: 20px; stroke-width: 1.6; }
.projects-create-info {
  display: flex;
  min-height: 56px;
  box-sizing: border-box;
  align-items: center;
  gap: 12px;
  margin-top: 16px;
  border-radius: 12px;
  padding: 12px;
  color: #5d5d5d;
  background: #f3f3f3;
}
.projects-create-info svg { width: 20px; height: 24px; flex: 0 0 20px; stroke-width: 1.5; }
.projects-create-info p {
  min-width: 0;
  margin: 0;
  font-size: 12px;
  font-weight: 400;
  line-height: 16px;
}
.projects-create-footer {
  position: relative;
  display: flex;
  width: 100%;
  align-items: center;
  justify-content: space-between;
}
.projects-create-memory-wrap { position: relative; flex: 0 0 auto; }
.projects-create-memory-trigger {
  display: inline-flex;
  width: 104px;
  height: 36px;
  align-items: center;
  justify-content: space-between;
  border: 0;
  border-radius: 8px;
  padding: 0 12px;
  color: #0d0d0d;
  background: transparent;
  font: inherit;
  font-size: 14px;
  font-weight: 400;
  line-height: 20px;
  cursor: pointer;
}
.projects-create-memory-trigger:hover,
.projects-create-memory-trigger[aria-expanded='true'] { background: #f3f3f3; }
.projects-create-memory-trigger:focus { outline: none; }
.projects-create-memory-trigger:focus-visible { outline: 2px solid #0d0d0d; outline-offset: 1px; }
.projects-create-memory-trigger svg { width: 16px; height: 16px; stroke-width: 1.8; }
.projects-create-submit {
  display: inline-flex;
  min-width: 82px;
  height: 36px;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 999px;
  padding: 0 12px;
  color: #fff;
  background: #0d0d0d;
  font: inherit;
  font-size: 14px;
  font-weight: 500;
  line-height: 20px;
  cursor: pointer;
}
.projects-create-submit:hover:not(:disabled) { background: #2f2f2f; }
.projects-create-submit:disabled { color: #fff; background: #c4c4c4; opacity: 1; cursor: not-allowed; }
.projects-create-icon-menu,
.projects-create-memory-menu {
  position: absolute;
  z-index: 80;
  border: 1px solid rgb(0 0 0 / 0.08);
  border-radius: 16px;
  color: #0d0d0d;
  background: #fff;
  box-shadow: 0 8px 12px rgb(0 0 0 / 0.08), 0 0 1px rgb(0 0 0 / 0.62);
}
.projects-create-icon-menu {
  top: 40px;
  left: -4px;
  display: grid;
  width: 260px;
  box-sizing: border-box;
  height: 418px;
  overflow-y: auto;
  padding: 12px;
}
.projects-create-color-grid {
  display: grid;
  grid-template-columns: repeat(8, 20px);
  justify-content: space-between;
  gap: 8px;
  min-height: 28px;
}
.projects-create-color-swatch {
  display: grid;
  width: 20px;
  height: 20px;
  place-items: center;
  border: 0;
  border-radius: 50%;
  padding: 0;
  background: var(--swatch-color);
  box-shadow: inset 0 0 0 1px rgb(0 0 0 / 0.14);
  cursor: pointer;
}
.projects-create-color-swatch.is-selected { outline: 2px solid #0d0d0d; outline-offset: 2px; }
.projects-create-custom-color {
  display: flex;
  min-height: 40px;
  align-items: center;
  gap: 10px;
  margin-top: 8px;
  border: 0;
  border-radius: 8px;
  padding: 0 4px;
  color: #5d5d5d;
  background: transparent;
  font: inherit;
  font-size: 12px;
  text-align: left;
  cursor: pointer;
}
.projects-create-custom-color:hover { background: #f3f3f3; color: #0d0d0d; }
.projects-create-custom-color__wheel {
  width: 20px;
  height: 20px;
  border: 1px solid rgb(0 0 0 / 0.18);
  border-radius: 50%;
  background: conic-gradient(#ef4444, #eab308, #22c55e, #3b82f6, #8b5cf6, #ec4899, #ef4444);
}
.projects-create-icon-divider { height: 1px; margin: 4px 0 8px; background: rgb(0 0 0 / 0.1); }
.projects-create-icon-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(36px, 1fr));
  gap: 4px;
}
.projects-create-icon-option {
  display: grid;
  width: 36px;
  height: 36px;
  place-items: center;
  border: 0;
  border-radius: 8px;
  padding: 0;
  color: #5d5d5d;
  background: transparent;
  font: inherit;
  font-size: 17px;
  cursor: pointer;
}
.projects-create-icon-option:hover,
.projects-create-icon-option.is-selected { color: #0d0d0d; background: #f3f3f3; }
.projects-create-memory-menu {
  top: calc(100% + 6px);
  left: -4px;
  display: grid;
  width: 320px;
  box-sizing: border-box;
  padding: 6px 0;
}
.projects-create-memory-option {
  display: flex;
  min-height: 52px;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin: 0 6px;
  border: 0;
  border-radius: 10px;
  padding: 8px 10px;
  color: #0d0d0d;
  background: transparent;
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.projects-create-memory-option:last-child { min-height: 68px; }
.projects-create-memory-option:hover,
.projects-create-memory-option.is-selected { background: #f3f3f3; }
.projects-create-memory-copy { display: grid; min-width: 0; gap: 2px; }
.projects-create-memory-copy strong { font-size: 14px; font-weight: 400; line-height: 20px; }
.projects-create-memory-copy small { color: #5d5d5d; font-size: 12px; font-weight: 400; line-height: 16px; }
.projects-create-memory-check { flex: 0 0 auto; font-size: 16px; }
.projects-create-visually-hidden {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip: rect(0 0 0 0);
  clip-path: inset(50%);
  white-space: nowrap;
}
.projects-field { display: grid; gap: 7px; color: var(--projects-text); font-size: 13px; font-weight: 600; }
.projects-field small { color: var(--projects-text-secondary); font-size: 12px; font-weight: 400; line-height: 1.45; }
.projects-field-row { display: grid; grid-template-columns: 112px minmax(0, 1fr); gap: 14px; }
.projects-field input,
.projects-field textarea,
.projects-field select { width: 100%; border: 1px solid var(--projects-border-strong); border-radius: 10px; padding: 10px 12px; color: var(--projects-text); background: var(--projects-page); font: inherit; font-size: 14px; font-weight: 400; line-height: 1.45; }
.projects-field textarea { min-height: 112px; resize: vertical; }
.projects-field select { min-height: 42px; }
.projects-field--icon input { text-align: center; font-size: 22px; }
.projects-color-control { display: flex; min-width: 0; align-items: center; gap: 10px; }
.projects-color-control input { width: 42px; height: 42px; flex: 0 0 42px; padding: 4px; cursor: pointer; }
.projects-color-control span { overflow: hidden; color: var(--projects-text-secondary); font-size: 12px; font-weight: 400; text-overflow: ellipsis; }
.projects-dialog-button { min-width: 82px; }
.projects-dialog-button--primary { border-color: var(--projects-text); color: var(--projects-page); background: var(--projects-text); }
.projects-dialog-button--primary:hover:not(:disabled) { opacity: 0.88; background: var(--projects-text); }
.projects-dialog-button:disabled { opacity: 0.5; cursor: wait; }
:global(html.dark) .projects-create-editor,
:global(html.dark) .projects-create-field { color: #fff; }
:global(html.dark) .projects-create-input-wrap > input { border-color: #fff; color: #fff; background: #212121; }
:global(html.dark) .projects-create-input-wrap > input::placeholder { color: #afafaf; }
:global(html.dark) .projects-create-info { color: #b4b4b4; background: #2f2f2f; }
:global(html.dark) .projects-create-memory-trigger,
:global(html.dark) .projects-create-memory-option,
:global(html.dark) .projects-create-icon-menu,
:global(html.dark) .projects-create-memory-menu { color: #fff; background: #353535; }
:global(html.dark) .projects-create-icon-menu,
:global(html.dark) .projects-create-memory-menu { border-color: rgb(255 255 255 / 0.12); }
:global(html.dark) .projects-create-memory-trigger:hover,
:global(html.dark) .projects-create-memory-trigger[aria-expanded='true'],
:global(html.dark) .projects-create-memory-option:hover,
:global(html.dark) .projects-create-memory-option.is-selected,
:global(html.dark) .projects-create-icon-option:hover,
:global(html.dark) .projects-create-icon-option.is-selected,
:global(html.dark) .projects-create-icon-trigger:hover,
:global(html.dark) .projects-create-icon-trigger[aria-expanded='true'] { background: #424242; }
:global(html.dark) .projects-create-memory-copy small { color: #b4b4b4; }
:global(html.dark) .projects-create-custom-color,
:global(html.dark) .projects-create-icon-option { color: #b4b4b4; }
:global(html.dark) .projects-create-custom-color:hover,
:global(html.dark) .projects-create-icon-option:hover,
:global(html.dark) .projects-create-icon-option.is-selected {
  color: #fff;
  background: #424242;
}
:global(html.dark) .projects-create-submit:disabled { color: #f2f2f2; background: #4a4a4a; }
.projects-workspace__drawer { position: fixed; z-index: 45; inset: 0; display: flex; }
.projects-workspace__scrim { position: absolute; inset: 0; border: 0; background: rgb(0 0 0 / 0.45); backdrop-filter: blur(2px); }
.projects-workspace__drawer :deep(.chat-history) { position: relative; z-index: 1; box-shadow: 18px 0 40px rgb(0 0 0 / 0.22); }
.projects-drawer-enter-active,
.projects-drawer-leave-active { transition: opacity 180ms ease; }
.projects-drawer-enter-from,
.projects-drawer-leave-to { opacity: 0; }
.projects-main :deep(button:focus-visible),
.projects-main :deep(input:focus-visible),
.projects-main :deep(textarea:focus-visible),
.projects-main :deep(select:focus-visible),
.projects-editor :is(input, textarea, select):focus-visible,
.projects-dialog-button:focus-visible { outline: 2px solid var(--projects-text-muted); outline-offset: 2px; }
@media (max-width: 767px) {
  .projects-main { overflow-y: scroll; }
  .projects-page { width: 100%; padding: 16px 16px 80px; }
  .projects-main--detail .projects-page { padding-bottom: 152px; }
  .projects-main--detail .projects-detail-content { margin-top: 10px; }
  .projects-main--detail .projects-detail-content--sources { margin-top: 18px; }
  .projects-main:not(.projects-main--detail) :deep(.projects-directory h1) { padding-left: 36px; }
  .projects-field-row { grid-template-columns: 92px minmax(0, 1fr); }
}
@media (max-width: 767px) and (hover: none) and (pointer: coarse) {
  .projects-mobile-navigation { position: absolute; z-index: 12; top: 8px; left: 8px; display: grid; width: 40px; height: 40px; place-items: center; border: 0; border-radius: 10px; padding: 0; color: var(--projects-text); background: transparent; cursor: pointer; }
  .projects-mobile-navigation:hover { background: var(--projects-surface-hover); }
  .projects-workspace__history--desktop { display: none; }
}
@media (prefers-reduced-motion: reduce) {
  .projects-drawer-enter-active,
  .projects-drawer-leave-active { transition: none; }
}
</style>

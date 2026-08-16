<template>
  <AppLayout variant="chat" shell-mode="chat">
    <div class="library-workspace">
      <ChatHistoryPanel
        v-show="!narrowSidebar"
        shell
        active-section="library"
        class="library-workspace__history library-workspace__history--desktop"
        :conversations="historyConversations"
        :active-id="chatStore.activeConversationId"
        :search-query="historySearchQuery"
        :searching="chatStore.searchingHistory"
        :has-more="chatStore.conversationsHaveMore"
        :loading-more="chatStore.loadingConversationPage"
        @new="startNewConversation"
        @select="selectConversation"
        @rename="chatStore.renameConversation"
        @delete="confirmChatDelete"
        @clear="confirmChatClear"
        @update:search-query="updateHistorySearch"
        @search="runHistorySearch"
        @load-more="chatStore.loadConversationPage()"
      />

      <Transition name="library-drawer">
        <div
          v-if="historyOpen && mobileHistoryLayout"
          ref="historyDrawerRef"
          class="library-workspace__drawer"
          role="dialog"
          aria-modal="true"
          :aria-label="t('chat.history.title')"
          tabindex="-1"
          @keydown="onHistoryDrawerKeydown"
        >
          <button
            type="button"
            class="library-workspace__scrim"
            :aria-label="t('chat.actions.closeHistory')"
            tabindex="-1"
            @click="closeHistory()"
          ></button>
          <ChatHistoryPanel
            shell
            mobile
            active-section="library"
            :conversations="historyConversations"
            :active-id="chatStore.activeConversationId"
            :search-query="historySearchQuery"
            :searching="chatStore.searchingHistory"
            :has-more="chatStore.conversationsHaveMore"
            :loading-more="chatStore.loadingConversationPage"
            @new="startNewConversation"
            @close="closeHistory"
            @select="selectConversation"
            @rename="chatStore.renameConversation"
            @delete="confirmChatDelete"
            @clear="confirmChatClear"
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
          active-section="library"
          sidebar-id="workspace-library-sidebar-overlay"
          :conversations="historyConversations"
          :active-id="chatStore.activeConversationId"
          :search-query="historySearchQuery"
          :searching="chatStore.searchingHistory"
          :has-more="chatStore.conversationsHaveMore"
          :loading-more="chatStore.loadingConversationPage"
          @new="startNewConversation"
          @close="closeHistory"
          @select="selectConversation"
          @rename="chatStore.renameConversation"
          @delete="confirmChatDelete"
          @clear="confirmChatClear"
          @update:search-query="updateHistorySearch"
          @search="runHistorySearch"
          @load-more="chatStore.loadConversationPage()"
        />
      </WorkspaceSidebarOverlayLayer>

      <main
        v-show="!previewFile"
        ref="mainRef"
        class="library-main"
        :aria-hidden="historyModalActive || previewFile ? 'true' : undefined"
        :inert="historyModalActive || previewFile ? true : undefined"
        @dragenter="onDragEnter"
        @dragover="onDragOver"
        @dragleave="onDragLeave"
        @drop="onDrop"
      >
        <div class="library-mobile-actions">
          <button
            ref="historyTriggerRef"
            type="button"
            class="library-icon-button"
            :aria-label="t('chat.history.title')"
            :title="t('chat.history.title')"
            @click="openHistory"
          >
            <Icon name="menu" size="sm" />
          </button>
        </div>

        <div v-if="dragActive" class="library-drop-overlay" aria-hidden="true">
          <div>
            <Icon name="upload" size="lg" />
            <strong>{{ t('library.upload.dropTitle') }}</strong>
            <span>{{ t('library.upload.dropDescription') }}</span>
          </div>
        </div>

        <section class="library-content" :aria-label="t('library.title')">
          <header class="library-header">
            <h1>{{ t('library.title') }}</h1>
            <div class="library-header__actions">
              <label class="library-search">
                <Icon name="librarySearch" size="sm" aria-hidden="true" />
                <span class="sr-only">{{ t('library.searchLabel') }}</span>
                <input
                  v-model="searchDraft"
                  class="library-search__input"
                  type="text"
                  autocomplete="off"
                  :placeholder="t('library.search')"
                  :aria-label="t('library.searchLabel')"
                />
                <span v-if="libraryStore.loading && libraryStore.initialized" class="library-search__spinner" aria-hidden="true"></span>
              </label>

              <div ref="newMenuAnchorRef" class="library-new">
                <button
                  type="button"
                  class="library-new__trigger"
                  :aria-expanded="newMenuOpen"
                  aria-haspopup="menu"
                  @click="newMenuOpen = !newMenuOpen"
                >
                  <span>{{ t('library.new') }}</span>
                  <Icon name="chevronDown" size="xs" aria-hidden="true" />
                </button>
                <Transition name="library-menu">
                  <div v-if="newMenuOpen" class="library-new__menu" role="menu">
                    <button type="button" role="menuitem" @click="chooseUpload">
                      <Icon name="upload" size="sm" aria-hidden="true" />
                      {{ t('library.uploadFile') }}
                    </button>
                  </div>
                </Transition>
              </div>
            </div>
          </header>

          <div class="library-toolbar">
            <div class="library-categories" role="tablist" :aria-label="t('library.title')">
              <button
                v-for="option in categoryOptions"
                :key="option.value"
                type="button"
                role="tab"
                :aria-selected="libraryStore.category === option.value"
                :class="{ 'library-categories__button--active': libraryStore.category === option.value }"
                @click="changeCategory(option.value)"
              >
                {{ t(option.label) }}
              </button>
            </div>

            <div class="library-toolbar__tools">
              <button
                ref="filterAnchorRef"
                type="button"
                class="library-icon-button library-icon-button--filter"
                :class="{ 'library-icon-button--active': filtersActive || filterOpen }"
                :aria-label="t('library.toolbar.filter')"
                :aria-expanded="filterOpen"
                @click="filterOpen = !filterOpen"
              >
                <Icon name="libraryFilter" size="md" />
                <span v-if="filtersActive" class="library-filter-dot"></span>
              </button>
              <span class="library-toolbar__divider" aria-hidden="true"></span>
              <button
                type="button"
                class="library-icon-button library-icon-button--round"
                :class="{ 'library-icon-button--active': libraryStore.currentView === 'grid' }"
                :aria-label="t('library.views.grid')"
                aria-describedby="library-grid-view-tooltip"
                :aria-pressed="libraryStore.currentView === 'grid'"
                @click="libraryStore.setView('grid')"
              >
                <Icon name="libraryGrid" size="md" />
                <span id="library-grid-view-tooltip" class="library-icon-tooltip" role="tooltip">
                  {{ t('library.views.gridTooltip') }}
                </span>
              </button>
              <button
                type="button"
                class="library-icon-button library-icon-button--round"
                :class="{ 'library-icon-button--active': libraryStore.currentView === 'list' }"
                :aria-label="t('library.views.list')"
                aria-describedby="library-list-view-tooltip"
                :aria-pressed="libraryStore.currentView === 'list'"
                @click="libraryStore.setView('list')"
              >
                <Icon name="libraryList" size="md" />
                <span id="library-list-view-tooltip" class="library-icon-tooltip" role="tooltip">
                  {{ t('library.views.listTooltip') }}
                </span>
              </button>
            </div>
          </div>

          <Transition name="library-selection">
            <div v-if="libraryStore.selectedFileIds.size" class="library-selection-bar" role="toolbar">
              <strong>{{ t('library.actions.selectedWithSize', {
                count: libraryStore.selectedFileIds.size,
                size: formatLibraryBytes(selectedDownloadBytes, locale),
              }) }}</strong>
              <span></span>
              <button type="button" :disabled="downloadBusy" @click="downloadSelected">
                <Icon name="download" size="sm" />
                {{ t('library.actions.download') }}
              </button>
              <button type="button" class="library-selection-bar__danger" @click="confirmSelectedDelete">
                <Icon name="trash" size="sm" />
                {{ t('library.actions.delete') }}
              </button>
              <button type="button" @click="libraryStore.clearSelection()">
                {{ t('library.actions.cancelSelection') }}
              </button>
            </div>
          </Transition>

          <div
            v-if="libraryStore.error && libraryStore.files.length > 0"
            class="library-refresh-error"
            role="alert"
          >
            <Icon name="exclamationCircle" size="sm" aria-hidden="true" />
            <span>{{ libraryStore.error }}</span>
            <button type="button" @click="libraryStore.load()">{{ t('library.actions.retry') }}</button>
          </div>

          <div v-if="initialLoading" class="library-skeleton" aria-hidden="true">
            <span class="library-skeleton__head"></span>
            <span v-for="index in 8" :key="index" class="library-skeleton__row"></span>
          </div>

          <div v-else-if="libraryStore.error && libraryStore.files.length === 0" class="library-state" role="alert">
            <Icon name="exclamationCircle" size="lg" aria-hidden="true" />
            <h2>{{ t('library.loadError.title') }}</h2>
            <p>{{ libraryStore.error || t('library.loadError.description') }}</p>
            <button type="button" @click="reloadLibrary">
              <Icon name="refresh" size="sm" />
              {{ t('library.actions.retry') }}
            </button>
          </div>

          <div v-else-if="libraryStore.files.length === 0" class="library-state">
            <Icon name="chatSidebarLibrary" size="xl" aria-hidden="true" />
            <h2>{{ hasActiveQuery ? t('library.empty.searchTitle') : t('library.empty.title') }}</h2>
            <p>{{ hasActiveQuery ? t('library.empty.searchDescription') : t('library.empty.description') }}</p>
            <button v-if="!hasActiveQuery" type="button" @click="openFileDialog">
              <Icon name="upload" size="sm" />
              {{ t('library.empty.upload') }}
            </button>
          </div>

          <template v-else>
            <div
              v-if="libraryStore.currentView === 'list'"
              class="library-list"
              role="table"
              :aria-label="t('library.title')"
            >
              <div class="library-list__header-group" role="rowgroup">
                <div class="library-list__header" role="row">
                  <span role="columnheader">{{ t('library.table.name') }}</span>
                  <span role="columnheader">{{ t('library.table.updatedAt') }}</span>
                  <span role="columnheader">{{ t('library.table.size') }}</span>
                  <span role="columnheader"><span class="sr-only">{{ t('library.table.actionsColumn') }}</span></span>
                </div>
              </div>
              <div class="library-list__rows" role="rowgroup">
                <article
                  v-for="file in libraryStore.files"
                  :key="file.id"
                  class="library-list__row"
                  :class="{ 'library-list__row--selected': libraryStore.selectedFileIds.has(file.id) }"
                  role="row"
                >
                  <div class="library-file-name" role="cell">
                    <label
                      class="library-select-control"
                      :class="{ 'library-select-control--visible': selectionMode || libraryStore.selectedFileIds.has(file.id) }"
                      @click.stop
                    >
                      <input
                        type="checkbox"
                        :checked="libraryStore.selectedFileIds.has(file.id)"
                        :aria-label="t('library.table.select', { name: file.name })"
                        @change="toggleFromInput(file.id, $event)"
                      />
                      <span><Icon name="check" size="xs" /></span>
                    </label>
                    <button
                      type="button"
                      class="library-list__open"
                      :aria-label="t('library.actions.previewNamed', { name: file.name })"
                      @click="openPreview(file)"
                    >
                      <span class="library-list__visual"><LibraryFileVisual :file="file" compact /></span>
                      <strong :title="file.name">{{ file.name }}</strong>
                    </button>
                  </div>
                  <time
                    role="cell"
                    :datetime="modifiedTimestamp(file) || undefined"
                    :title="modifiedTimeTooltip(file)"
                  >{{ modifiedTimeLabel(file) }}</time>
                  <span role="cell">{{ formatLibraryBytes(file.size, locale) }}</span>
                  <div class="library-list__actions-cell" role="cell">
                    <button
                      :ref="(element) => setMenuAnchor(file.id, element)"
                      type="button"
                      class="library-row-menu"
                      :aria-label="t('library.table.actions', { name: file.name })"
                      :title="t('library.table.actions', { name: file.name })"
                      @click="toggleFileMenu(file)"
                    >
                      <Icon name="more" size="sm" />
                    </button>
                  </div>
                </article>
              </div>
            </div>

            <ul v-else class="library-grid">
              <li
                v-for="file in libraryStore.files"
                :key="file.id"
                :class="{ 'library-grid__item--selected': libraryStore.selectedFileIds.has(file.id) }"
              >
                <button class="library-grid__preview" type="button" @click="openPreview(file)">
                  <span class="library-grid__visual"><LibraryFileVisual :file="file" /></span>
                  <strong :title="file.name">{{ file.name }}</strong>
                  <span>{{ formatLibraryDate(file.updatedAt, locale) }}</span>
                </button>
                <label
                  class="library-select-control library-grid__select"
                  :class="{ 'library-select-control--visible': selectionMode || libraryStore.selectedFileIds.has(file.id) }"
                >
                  <input
                    type="checkbox"
                    :checked="libraryStore.selectedFileIds.has(file.id)"
                    :aria-label="t('library.table.select', { name: file.name })"
                    @change="toggleFromInput(file.id, $event)"
                  />
                  <span><Icon name="check" size="xs" /></span>
                </label>
                <button
                  :ref="(element) => setMenuAnchor(file.id, element)"
                  type="button"
                  class="library-grid__menu"
                  :aria-label="t('library.table.actions', { name: file.name })"
                  :title="t('library.table.actions', { name: file.name })"
                  @click="toggleFileMenu(file)"
                >
                  <Icon name="more" size="sm" />
                </button>
              </li>
            </ul>

            <button
              v-if="libraryStore.hasMore"
              type="button"
              class="library-load-more"
              :disabled="libraryStore.loading"
              @click="libraryStore.loadMore()"
            >
              {{ libraryStore.loading ? t('library.actions.loadingMore') : t('library.actions.loadMore') }}
            </button>
          </template>

          <div v-if="libraryStore.storageUsage" class="library-storage" :aria-label="t('library.storage.label')">
            <span v-if="libraryStore.storageUsage.limitBytes === 0">
              {{ formatLibraryBytes(libraryStore.storageUsage.usedBytes, locale) }}
              · {{ t('common.unlimited') }}
            </span>
            <template v-else>
              <span>{{ t('library.storage.usage', {
                used: formatLibraryBytes(libraryStore.storageUsage.usedBytes, locale),
                limit: formatLibraryBytes(libraryStore.storageUsage.limitBytes, locale),
              }) }}</span>
              <span class="library-storage__track" aria-hidden="true">
                <span :style="{ width: `${storagePercent}%` }"></span>
              </span>
            </template>
          </div>
        </section>

        <input
          ref="fileInputRef"
          class="sr-only"
          type="file"
          multiple
          hidden
          @change="onFileInput"
        />
      </main>

      <LibraryFilePreviewDialog
        :show="Boolean(previewFile)"
        :file="previewFile"
        :downloading="downloadBusy"
        @close="closePreview"
        @download="downloadOne"
        @remove="requestPreviewDelete"
      />
    </div>

    <LibraryUploadTray
      :uploads="libraryStore.uploads"
      @dismiss="libraryStore.dismissSettledUploads()"
      @retry="retryUpload"
    />

    <LibraryFilterPopover
      :show="filterOpen"
      :anchor="filterAnchorRef"
      :source="libraryStore.source"
      :type="libraryStore.fileType"
      :sort="libraryStore.sort"
      @close="filterOpen = false"
      @apply="applyFilters"
    />

    <LibraryFileMenu
      :show="Boolean(menuFile)"
      :file="menuFile"
      :anchor="menuFile ? menuAnchors.get(menuFile.id) ?? null : null"
      @close="menuFile = null"
      @preview="openPreview"
      @download="downloadOne"
      @delete="confirmFileDelete"
    />

    <BaseDialog
      :show="deleteFiles.length > 0"
      :title="deleteFiles.length > 1 ? t('library.deleteConfirm.multipleTitle') : t('library.deleteConfirm.singleTitle')"
      width="narrow"
      variant="workspace-confirm"
      :description-id="'library-delete-confirm-description'"
      @close="deleteFiles = []"
    >
      <p id="library-delete-confirm-description" class="library-confirm-copy">
        {{ deleteFiles.length > 1
          ? t('library.deleteConfirm.multipleDescription', { count: deleteFiles.length })
          : t('library.deleteConfirm.singleDescription', { name: deleteFiles[0]?.name ?? '' }) }}
      </p>
      <template #footer>
        <button type="button" class="library-confirm-button" @click="deleteFiles = []">
          {{ t('library.deleteConfirm.cancel') }}
        </button>
        <button type="button" class="library-confirm-button library-confirm-button--danger" :disabled="deleteBusy" @click="applyFileDelete">
          {{ t('library.deleteConfirm.confirm') }}
        </button>
      </template>
    </BaseDialog>

    <BaseDialog
      :show="chatConfirmation !== null"
      :title="chatConfirmation?.kind === 'clear' ? t('chat.confirm.clearTitle') : t('chat.confirm.deleteTitle')"
      width="narrow"
      variant="workspace-confirm"
      @close="chatConfirmation = null"
    >
      <p class="library-confirm-copy">
        {{ chatConfirmation?.kind === 'clear'
          ? t('chat.confirm.clearDescription')
          : t('chat.confirm.deleteDescriptionPrefix') + (chatConfirmation?.title ?? '') + t('chat.confirm.deleteDescriptionSuffix') }}
      </p>
      <template #footer>
        <button type="button" class="library-confirm-button" @click="chatConfirmation = null">{{ t('common.cancel') }}</button>
        <button type="button" class="library-confirm-button library-confirm-button--danger" @click="applyChatConfirmation">{{ t('common.delete') }}</button>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import WorkspaceSidebarOverlayLayer from '@/components/layout/WorkspaceSidebarOverlayLayer.vue'
import { useWorkspaceResponsiveState } from '@/components/layout/workspaceResponsive'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import ChatHistoryPanel from '@/components/chat/ChatHistoryPanel.vue'
import LibraryFileMenu from '@/components/library/LibraryFileMenu.vue'
import LibraryFilePreviewDialog from '@/components/library/LibraryFilePreviewDialog.vue'
import LibraryFileVisual from '@/components/library/LibraryFileVisual.vue'
import LibraryFilterPopover from '@/components/library/LibraryFilterPopover.vue'
import LibraryUploadTray from '@/components/library/LibraryUploadTray.vue'
import { formatLibraryBytes, formatLibraryDate, prepareLibraryBrowserDownload } from '@/components/library/libraryFileUi'
import { createLibraryDownloadTicket } from '@/api/library'
import { buildApiUrl } from '@/api/client'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useChatStore } from '@/stores/chat'
import { useLibraryStore } from '@/stores/library'
import {
  formatLibraryFileTime,
  formatLibraryFileTimeTooltip,
  libraryFileModifiedTimestamp,
} from '@/utils/libraryFileTime'
import type {
  LibraryCategory,
  LibraryFile,
  LibraryFileType,
  LibrarySort,
  LibrarySource,
} from '@/types/library'

type ChatConfirmation = { kind: 'delete'; conversationId: string; title: string } | { kind: 'clear' }

const libraryDownloadProbePath = '/api/v1/library/download/AAAAAAAAAAAAAAAAAAAAAA'

const { t, locale } = useI18n()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()
const chatStore = useChatStore()
const libraryStore = useLibraryStore()
const { mobileDrawer: mobileHistoryLayout, narrowSidebar } = useWorkspaceResponsiveState()

const mainRef = ref<HTMLElement | null>(null)
const historyDrawerRef = ref<HTMLElement | null>(null)
const historyTriggerRef = ref<HTMLButtonElement | null>(null)
const fileInputRef = ref<HTMLInputElement | null>(null)
const filterAnchorRef = ref<HTMLButtonElement | null>(null)
const newMenuAnchorRef = ref<HTMLElement | null>(null)
const searchDraft = ref('')
const filterOpen = ref(false)
const newMenuOpen = ref(false)
const historyOpen = ref(false)
const historySearchQuery = ref('')
const menuFile = ref<LibraryFile | null>(null)
const previewFile = ref<LibraryFile | null>(null)
const deleteFiles = ref<LibraryFile[]>([])
const downloadBusy = ref(false)
const deleteBusy = ref(false)
const dragActive = ref(false)
const chatConfirmation = ref<ChatConfirmation | null>(null)
const menuAnchors = new Map<string, HTMLElement>()
let searchTimer: ReturnType<typeof setTimeout> | null = null
let historySearchTimer: ReturnType<typeof setTimeout> | null = null
let dragDepth = 0
let disposed = false
let previewSidebarCollapsedBefore: boolean | null = null

const categoryOptions = [
  { value: 'all', label: 'library.categories.all' },
  { value: 'image', label: 'library.categories.image' },
  { value: 'file', label: 'library.categories.file' },
] as const

const selectedDownloadBytes = computed(() => libraryStore.selectedFiles.reduce(
  (total, file) => total + Math.max(0, file.size),
  0,
))
const historyConversations = computed(() => (
  historySearchQuery.value.trim() ? chatStore.searchResults : chatStore.conversations
))
const narrowSidebarOpen = computed(() => appStore.workspaceNarrowSidebarOpen)
const historyModalActive = computed(() => (
  (historyOpen.value && mobileHistoryLayout.value)
  || (narrowSidebar.value && narrowSidebarOpen.value)
))
const initialLoading = computed(() => libraryStore.loading && !libraryStore.initialized)
const selectionMode = computed(() => libraryStore.selectedFileIds.size > 0)
const filtersActive = computed(() => (
  libraryStore.source !== 'all'
  || libraryStore.fileType !== 'all'
  || libraryStore.sort !== 'updated_desc'
))
const hasActiveQuery = computed(() => (
  Boolean(libraryStore.searchKeyword)
  || libraryStore.category !== 'all'
  || filtersActive.value
))
const storagePercent = computed(() => {
  const usage = libraryStore.storageUsage
  if (!usage || usage.limitBytes <= 0) return 0
  return Math.min(100, Math.max(0, Math.round((usage.usedBytes / usage.limitBytes) * 100)))
})

function modifiedTimestamp(file: LibraryFile): string {
  return libraryFileModifiedTimestamp(file)
}

function modifiedTimeLabel(file: LibraryFile): string {
  return formatLibraryFileTime(modifiedTimestamp(file), { locale: locale.value })
}

function modifiedTimeTooltip(file: LibraryFile): string {
  return formatLibraryFileTimeTooltip(modifiedTimestamp(file), { locale: locale.value })
}

watch(searchDraft, (value) => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    libraryStore.setSearchKeyword(value.trim())
    void libraryStore.load()
  }, 280)
})

watch(mobileHistoryLayout, (mobile) => {
  if (!mobile && historyOpen.value) void closeHistory(mainRef.value)
})

watch(() => authStore.user?.id, async (userId) => {
  await chatStore.hydrate(userId)
  if (disposed || !userId) return
  await chatStore.syncHistory()
  if (!disposed) await chatStore.loadConversationPage(true)
}, { immediate: true, flush: 'sync' })

onMounted(() => {
  document.addEventListener('pointerdown', onDocumentPointerDown, true)
  void Promise.all([libraryStore.load(), libraryStore.fetchStorage()])
})

onBeforeUnmount(() => {
  disposed = true
  if (searchTimer) clearTimeout(searchTimer)
  if (historySearchTimer) clearTimeout(historySearchTimer)
  document.removeEventListener('pointerdown', onDocumentPointerDown, true)
  libraryStore.clearSelection()
  restorePreviewSidebar()
})

function onDocumentPointerDown(event: PointerEvent): void {
  if (!newMenuOpen.value) return
  const target = event.target as Node | null
  if (!target || newMenuAnchorRef.value?.contains(target)) return
  newMenuOpen.value = false
}

function changeCategory(category: LibraryCategory): void {
  if (libraryStore.category === category) return
  libraryStore.setCategory(category)
  void libraryStore.load()
}

function applyFilters(value: { source: LibrarySource; type: LibraryFileType; sort: LibrarySort }): void {
  libraryStore.setFilters({ source: value.source, type: value.type })
  libraryStore.setSort(value.sort)
  filterOpen.value = false
  void libraryStore.load()
}

function reloadLibrary(): void {
  void Promise.all([libraryStore.load(), libraryStore.fetchStorage()])
}

function chooseUpload(): void {
  newMenuOpen.value = false
  openFileDialog()
}

function openFileDialog(): void {
  fileInputRef.value?.click()
}

function onFileInput(event: Event): void {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files ?? [])
  input.value = ''
  if (files.length) void uploadFiles(files)
}

async function uploadFiles(files: File[]): Promise<void> {
  await libraryStore.uploadFiles(files)
}

async function retryUpload(key: string): Promise<void> {
  await libraryStore.retryUpload(key)
}

function isFileDrag(event: DragEvent): boolean {
  const transfer = event.dataTransfer
  return Boolean(transfer && (Array.from(transfer.types ?? []).includes('Files') || transfer.files.length))
}

function onDragEnter(event: DragEvent): void {
  if (!isFileDrag(event)) return
  event.preventDefault()
  dragDepth += 1
  dragActive.value = true
  if (event.dataTransfer) event.dataTransfer.dropEffect = 'copy'
}

function onDragOver(event: DragEvent): void {
  if (!isFileDrag(event)) return
  event.preventDefault()
  if (event.dataTransfer) event.dataTransfer.dropEffect = 'copy'
}

function onDragLeave(event: DragEvent): void {
  if (!dragActive.value) return
  if (event.relatedTarget === null) dragDepth = 0
  else dragDepth = Math.max(0, dragDepth - 1)
  if (dragDepth === 0) dragActive.value = false
}

function onDrop(event: DragEvent): void {
  if (!isFileDrag(event)) return
  event.preventDefault()
  const files = Array.from(event.dataTransfer?.files ?? [])
  dragDepth = 0
  dragActive.value = false
  if (files.length) void uploadFiles(files)
}

function toggleFromInput(id: string, event: Event): void {
  libraryStore.toggleSelection(id, (event.target as HTMLInputElement).checked)
}

function setMenuAnchor(id: string, element: unknown): void {
  if (element instanceof HTMLElement) menuAnchors.set(id, element)
  else menuAnchors.delete(id)
}

function toggleFileMenu(file: LibraryFile): void {
  menuFile.value = menuFile.value?.id === file.id ? null : file
}

function openPreview(file: LibraryFile): void {
  if (menuFile.value?.id === file.id) {
    menuAnchors.get(file.id)?.focus({ preventScroll: true })
  }
  menuFile.value = null
  filterOpen.value = false
  newMenuOpen.value = false
  if (!narrowSidebar.value && previewSidebarCollapsedBefore === null) {
    previewSidebarCollapsedBefore = appStore.sidebarCollapsed
    if (!appStore.sidebarCollapsed) appStore.setSidebarCollapsed(true)
  }
  previewFile.value = file
}

function restorePreviewSidebar(): void {
  if (previewSidebarCollapsedBefore === null) return
  const collapsed = previewSidebarCollapsedBefore
  previewSidebarCollapsedBefore = null
  appStore.setSidebarCollapsed(collapsed)
}

function closePreview(): void {
  previewFile.value = null
  restorePreviewSidebar()
}

function requestPreviewDelete(file: LibraryFile): void {
  closePreview()
  deleteFiles.value = [file]
}

async function downloadOne(file: LibraryFile): Promise<void> {
  if (downloadBusy.value) return
  downloadBusy.value = true
  let preparedDownload: ReturnType<typeof prepareLibraryBrowserDownload> | undefined
  try {
    preparedDownload = prepareLibraryBrowserDownload(buildApiUrl(libraryDownloadProbePath))
    const ticket = await createLibraryDownloadTicket([file.id])
    preparedDownload.start(buildApiUrl(ticket.downloadPath))
    preparedDownload = undefined
  } catch {
    preparedDownload?.cancel()
    appStore.showError(t('library.errors.downloadFailed'))
  } finally {
    downloadBusy.value = false
  }
}

async function downloadSelected(): Promise<void> {
  if (downloadBusy.value) return
  const files = libraryStore.selectedFiles
  if (files.length === 0) return
  if (files.length === 1 && files[0]) {
    await downloadOne(files[0])
    return
  }
  downloadBusy.value = true
  let preparedDownload: ReturnType<typeof prepareLibraryBrowserDownload> | undefined
  try {
    preparedDownload = prepareLibraryBrowserDownload(buildApiUrl(libraryDownloadProbePath))
    const ticket = await createLibraryDownloadTicket(files.map(({ id }) => id))
    preparedDownload.start(buildApiUrl(ticket.downloadPath))
    preparedDownload = undefined
  } catch {
    preparedDownload?.cancel()
    appStore.showError(t('library.errors.downloadFailed'))
  } finally {
    downloadBusy.value = false
  }
}

function confirmFileDelete(file: LibraryFile): void {
  deleteFiles.value = [file]
}

function confirmSelectedDelete(): void {
  deleteFiles.value = [...libraryStore.selectedFiles]
}

async function applyFileDelete(): Promise<void> {
  if (deleteBusy.value || deleteFiles.value.length === 0) return
  deleteBusy.value = true
  const targets = [...deleteFiles.value]
  deleteFiles.value = []
  try {
    const result = await libraryStore.removeFiles(targets.map(({ id }) => id))
    if (result.failed.length > 0) appStore.showError(t('library.errors.deleteFailed'))
  } finally {
    deleteBusy.value = false
  }
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

async function startNewConversation(): Promise<void> {
  chatStore.selectConversation(null)
  await router.push({ path: '/chat', query: { conversation: 'new' } })
}

async function selectConversation(id: string): Promise<void> {
  chatStore.selectConversation(id)
  await router.push('/chat')
}

async function openHistory(): Promise<void> {
  if (!mobileHistoryLayout.value) return
  historyOpen.value = true
  await nextTick()
  const first = historyDrawerRef.value?.querySelector<HTMLElement>('a[href], button:not([disabled]), input:not([disabled]), [tabindex]:not([tabindex="-1"])')
  ;(first ?? historyDrawerRef.value)?.focus()
}

async function closeHistory(focusTarget: HTMLElement | null = historyTriggerRef.value): Promise<void> {
  const mobileWasOpen = historyOpen.value
  if (mobileWasOpen) historyOpen.value = false
  if (narrowSidebarOpen.value) appStore.setWorkspaceNarrowSidebarOpen(false)
  await nextTick()
  if (mobileWasOpen) focusTarget?.focus()
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
    last?.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first?.focus()
  }
}

function confirmChatDelete(id: string): void {
  const conversation = historyConversations.value.find((item) => item.id === id)
    ?? chatStore.conversations.find((item) => item.id === id)
  chatConfirmation.value = {
    kind: 'delete',
    conversationId: id,
    title: conversation?.title.trim() || t('chat.actions.newChat'),
  }
}

function confirmChatClear(): void {
  chatConfirmation.value = { kind: 'clear' }
}

function applyChatConfirmation(): void {
  const action = chatConfirmation.value
  chatConfirmation.value = null
  if (!action) return
  if (action.kind === 'clear') chatStore.clearConversations()
  else chatStore.deleteConversation(action.conversationId)
}
</script>

<style scoped>
.library-workspace {
  display: flex;
  width: 100%;
  height: 100%;
  min-height: 0;
  overflow: hidden;
  color: var(--workspace-text);
  background: var(--workspace-canvas);
}

.library-main {
  --library-page-canvas: #fff;
  --library-control-active: #f3f3f3;
  --library-control-hover: rgb(0 0 0 / 0.05);
  --library-search-border: rgb(0 0 0 / 0.1);
  --library-search-border-focus: rgb(0 0 0 / 0.15);
  position: relative;
  min-width: 0;
  min-height: 0;
  flex: 1;
  overflow-x: hidden;
  overflow-y: auto;
  background: var(--library-page-canvas);
  font-family: inherit;
  overscroll-behavior: contain;
}

.library-content {
  box-sizing: border-box;
  width: min(100%, 800px);
  min-height: 100%;
  margin: 0 auto;
  padding: 116px 16px 48px 48px;
}

.library-header,
.library-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.library-header h1 {
  display: flex;
  height: 36px;
  min-width: 0;
  flex: 1;
  align-items: center;
  margin: 0;
  color: var(--workspace-text);
  font-size: 28px;
  font-weight: 400;
  line-height: 34px;
  letter-spacing: normal;
}

.library-header__actions {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 12px;
}

.library-search {
  position: relative;
  display: block;
  box-sizing: border-box;
  width: 240px;
  height: 36px;
  color: var(--workspace-identity-text-tertiary);
}

.library-search > svg {
  position: absolute;
  z-index: 1;
  top: 50%;
  left: 12px;
  pointer-events: none;
  transform: translateY(-50%);
}

.library-search input {
  box-sizing: border-box;
  width: 100%;
  height: 100%;
  min-width: 0;
  appearance: none;
  border: 1px solid var(--library-search-border);
  border-radius: 999px;
  outline: 0;
  padding: 8px 12px 8px 36px;
  color: var(--workspace-text);
  background: var(--library-page-canvas);
  font: inherit;
  font-size: 14px;
  line-height: 20px;
  transition: border-color 120ms ease;
}

.library-search input:focus { border-color: var(--library-search-border-focus); }
.library-search input::placeholder { color: var(--workspace-text-muted); opacity: 1; }

.library-search__spinner {
  position: absolute;
  z-index: 1;
  top: 11px;
  right: 12px;
  width: 14px;
  height: 14px;
  border: 1.5px solid var(--workspace-border-strong);
  border-top-color: var(--workspace-text-secondary);
  border-radius: 50%;
  animation: library-spin 700ms linear infinite;
}

.library-new { position: relative; }

.library-new__trigger {
  display: inline-flex;
  box-sizing: border-box;
  min-width: 76px;
  height: 36px;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border: 1px solid transparent;
  border-radius: 999px;
  padding: 0 12px;
  color: var(--library-page-canvas);
  background: var(--workspace-text);
  font: inherit;
  font-size: 14px;
  font-weight: 500;
  line-height: 20px;
  cursor: pointer;
}

.library-new__menu {
  position: absolute;
  z-index: 30;
  top: calc(100% + 7px);
  right: 0;
  width: 166px;
  border: 1px solid var(--workspace-popover-border);
  border-radius: 12px;
  padding: 5px;
  background: var(--workspace-popover-surface);
  box-shadow: var(--workspace-popover-shadow);
}

.library-new__menu button {
  display: flex;
  width: 100%;
  min-height: 38px;
  align-items: center;
  gap: 10px;
  border: 0;
  border-radius: 8px;
  padding: 0 9px;
  color: var(--workspace-popover-text);
  background: transparent;
  font: inherit;
  font-size: 14px;
  cursor: pointer;
}

.library-new__menu button:hover { background: var(--workspace-popover-hover); }

.library-toolbar {
  height: 60px;
  margin-top: 40px;
}

.library-categories {
  display: flex;
  align-items: center;
  gap: 0;
}

.library-categories button {
  box-sizing: border-box;
  height: 36px;
  border: 1px solid transparent;
  border-radius: 999px;
  padding: 0 16px;
  color: var(--workspace-text-secondary);
  background: transparent;
  font: inherit;
  font-size: 14px;
  font-weight: 500;
  line-height: 20px;
  cursor: pointer;
  transition: color 120ms ease, background-color 120ms ease;
}

.library-categories button:hover { color: var(--workspace-text); background: var(--library-control-active); }
.library-categories button.library-categories__button--active { color: var(--workspace-text); background: var(--library-control-active); }

.library-toolbar__tools {
  display: flex;
  align-items: center;
  gap: 8px;
}

.library-toolbar__divider {
  width: 1px;
  height: 20px;
  margin: 0 4px;
  background: var(--workspace-divider);
}

.library-icon-button {
  position: relative;
  display: grid;
  width: 36px;
  height: 36px;
  place-items: center;
  border: 0;
  border-radius: 999px;
  padding: 0;
  color: var(--workspace-identity-text-tertiary);
  background: transparent;
  cursor: pointer;
}

.library-icon-button--filter {
  transition:
    color 150ms cubic-bezier(0.4, 0, 0.2, 1),
    background-color 150ms cubic-bezier(0.4, 0, 0.2, 1);
}

.library-icon-button--round { border-radius: 50%; }
.library-icon-button--active { color: var(--workspace-text); background: var(--library-control-active); }
.library-filter-dot { position: absolute; top: 6px; right: 6px; width: 5px; height: 5px; border-radius: 50%; background: var(--workspace-text); }

.library-icon-tooltip {
  position: absolute;
  z-index: 50;
  bottom: calc(100% + 8px);
  left: 50%;
  width: max-content;
  max-width: 20rem;
  visibility: hidden;
  border: 1px solid rgb(255 255 255 / 0.05);
  border-radius: 999px;
  padding: 5px 12px;
  color: #fff;
  background: #1b1b1b;
  box-shadow: 0 8px 18px rgb(15 23 42 / 0.2);
  font-size: 14px;
  font-weight: 600;
  line-height: 18px;
  letter-spacing: -0.15px;
  text-align: center;
  white-space: pre-wrap;
  opacity: 0;
  pointer-events: none;
  user-select: none;
  transform: translateX(-50%);
  transition:
    opacity 150ms cubic-bezier(0.4, 0, 0.2, 1),
    visibility 0s linear 150ms;
}

@media (hover: hover) and (pointer: fine) {
  .library-icon-button:hover { background: var(--library-control-hover); }
  .library-icon-button--active:hover { background: var(--library-control-active); }
  .library-icon-button--filter:hover { color: var(--workspace-text); }
  .library-icon-button:hover .library-icon-tooltip {
    visibility: visible;
    opacity: 1;
    transition-delay: 250ms, 250ms;
  }
}

.library-icon-button:focus-visible .library-icon-tooltip {
  visibility: visible;
  opacity: 1;
  transition-delay: 0ms, 0ms;
}

.library-selection-bar {
  position: sticky;
  z-index: 10;
  top: 12px;
  display: flex;
  min-height: 46px;
  align-items: center;
  gap: 4px;
  margin: 18px 0 -64px;
  border: 1px solid var(--workspace-popover-border);
  border-radius: 14px;
  padding: 5px 7px 5px 14px;
  background: var(--workspace-popover-surface);
  box-shadow: var(--workspace-popover-shadow);
}

.library-selection-bar strong { font-size: 13px; font-weight: 600; }
.library-selection-bar > span { flex: 1; }
.library-selection-bar button { display: inline-flex; min-height: 34px; align-items: center; gap: 6px; border: 0; border-radius: 8px; padding: 0 9px; color: var(--workspace-popover-text); background: transparent; font: inherit; font-size: 13px; cursor: pointer; }
.library-selection-bar button:hover { background: var(--workspace-popover-hover); }
.library-selection-bar button:disabled { opacity: 0.5; cursor: wait; }
.library-selection-bar button.library-selection-bar__danger { color: var(--workspace-menu-danger); }

.library-refresh-error {
  display: flex;
  min-height: 42px;
  align-items: center;
  gap: 9px;
  margin-top: 18px;
  border-radius: 9px;
  padding: 6px 8px 6px 11px;
  color: var(--workspace-text-secondary);
  background: var(--workspace-surface-subtle);
  font-size: 12px;
}

.library-refresh-error > span { min-width: 0; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.library-refresh-error button { border: 0; border-radius: 7px; padding: 6px 8px; color: var(--workspace-text); background: transparent; font: inherit; font-weight: 600; cursor: pointer; }
.library-refresh-error button:hover { background: var(--workspace-hover); }

.library-list { margin-top: 9px; }
.library-list__header,
.library-list__row {
  display: grid;
  box-sizing: border-box;
  grid-template-columns: minmax(0, 1fr) 160px 88px 64px;
  align-items: center;
  column-gap: 16px;
}
.library-list__header {
  height: 42px;
  padding: 12px 8px 12px 0;
  color: var(--workspace-text-secondary);
  font-size: 14px;
  font-weight: 400;
  line-height: 20px;
}
.library-list__row {
  position: relative;
  height: 60px;
  border-bottom: 1px solid var(--workspace-divider);
  border-radius: 0;
  padding: 10px 8px 10px 0;
  color: var(--workspace-text-secondary);
  font-size: 14px;
  font-weight: 400;
  line-height: 20px;
}
.library-list__header > span:nth-child(3),
.library-list__row > span[role='cell'] { box-sizing: border-box; padding-left: 16px; }
.library-list__row--selected { background: var(--library-control-active); }
.library-file-name { display: flex; min-width: 0; align-items: center; color: var(--workspace-text); }
.library-file-name strong { min-width: 0; flex: 1; overflow: hidden; font-size: 14px; font-weight: 400; line-height: 20px; text-overflow: ellipsis; white-space: nowrap; }
.library-list__open { display: flex; min-width: 0; min-height: 32px; flex: 1; align-items: center; gap: 12px; border: 0; border-radius: 7px; padding: 0; color: inherit; background: transparent; font: inherit; text-align: left; cursor: pointer; }
.library-list__visual { width: 32px; height: 32px; flex: 0 0 32px; }
.library-list__actions-cell { display: grid; place-items: center; }

.library-file-name > .library-select-control {
  position: absolute;
  top: 0;
  left: -58px;
  width: 44px;
  height: 59px;
}

.library-select-control { position: relative; display: grid; width: 44px; height: 44px; flex: 0 0 44px; place-items: center; opacity: 0; cursor: pointer; transition: opacity 100ms ease; }
.library-select-control--visible,
.library-select-control:focus-within,
.library-list__row:hover .library-select-control,
.library-grid li:hover .library-select-control { opacity: 1; }
.library-select-control input { position: absolute; inset: 0; width: 100%; height: 100%; margin: 0; opacity: 0; cursor: pointer; }
.library-select-control > span { display: grid; width: 16px; height: 16px; place-items: center; border: 1px solid var(--workspace-border-strong); border-radius: 4px; color: transparent; background: var(--workspace-canvas); }
.library-select-control input:checked + span { border-color: var(--workspace-text); color: var(--workspace-canvas); background: var(--workspace-text); }
.library-select-control input:focus-visible + span { outline: 2px solid var(--workspace-text-muted); outline-offset: 2px; }

.library-row-menu,
.library-grid__menu { display: grid; width: 34px; height: 34px; place-items: center; border: 0; border-radius: 8px; padding: 0; color: var(--workspace-text-muted); background: transparent; cursor: pointer; opacity: 0; }
.library-list__row:hover .library-row-menu,
.library-row-menu:focus-visible,
.library-grid li:hover .library-grid__menu,
.library-grid__menu:focus-visible { opacity: 1; }
.library-row-menu:hover,
.library-grid__menu:hover { color: var(--workspace-text); background: var(--workspace-selected); }

.library-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(160px, 1fr)); gap: 20px 14px; margin: 30px 0 0; padding: 0; list-style: none; }
.library-grid li { position: relative; min-width: 0; border-radius: 12px; padding: 5px; }
.library-grid li:hover,
.library-grid__item--selected { background: var(--workspace-hover); }
.library-grid__preview { display: grid; width: 100%; min-width: 0; border: 0; padding: 0; color: inherit; background: transparent; text-align: left; cursor: pointer; }
.library-grid__visual { display: block; width: 100%; aspect-ratio: 4 / 3; overflow: hidden; border-radius: 10px; background: var(--workspace-surface-subtle); }
.library-grid__preview strong { margin: 9px 5px 0; overflow: hidden; color: var(--workspace-text); font-size: 13px; font-weight: 500; line-height: 20px; text-overflow: ellipsis; white-space: nowrap; }
.library-grid__preview > span:last-child { margin: 0 5px; color: var(--workspace-text-muted); font-size: 11px; line-height: 17px; }
.library-grid__select { position: absolute; top: 0; left: 0; }
.library-grid__menu { position: absolute; top: 9px; right: 9px; color: var(--workspace-text); background: color-mix(in srgb, var(--workspace-canvas) 82%, transparent); backdrop-filter: blur(5px); }

.library-skeleton { display: grid; gap: 0; margin-top: 9px; }
.library-skeleton span { display: block; border-bottom: 1px solid var(--workspace-divider); background: linear-gradient(90deg, transparent, color-mix(in srgb, var(--workspace-surface-subtle) 80%, transparent), transparent); background-size: 240% 100%; animation: library-shimmer 1.2s ease-in-out infinite; }
.library-skeleton__head { height: 42px; }
.library-skeleton__row { height: 60px; }

.library-state { display: grid; min-height: 370px; place-items: center; align-content: center; padding: 36px 20px; text-align: center; }
.library-state > svg { margin-bottom: 20px; color: var(--workspace-text-muted); }
.library-state h2 { margin: 0; font-size: 18px; font-weight: 600; letter-spacing: -0.01em; }
.library-state p { max-width: 44ch; margin: 7px 0 0; color: var(--workspace-text-secondary); font-size: 14px; line-height: 1.55; }
.library-state button,
.library-load-more { display: inline-flex; min-height: 38px; align-items: center; gap: 7px; margin-top: 20px; border: 1px solid var(--workspace-border-strong); border-radius: 999px; padding: 0 15px; color: var(--workspace-text); background: transparent; font: inherit; font-size: 13px; font-weight: 500; cursor: pointer; }
.library-state button:hover,
.library-load-more:hover { background: var(--workspace-hover); }
.library-load-more { display: flex; margin: 22px auto 0; }
.library-load-more:disabled { opacity: 0.55; cursor: wait; }

.library-storage { display: flex; align-items: center; justify-content: flex-end; gap: 10px; margin-top: 22px; color: var(--workspace-text-muted); font-size: 11px; }
.library-storage__track { display: block; width: 68px; height: 3px; overflow: hidden; border-radius: 999px; background: var(--workspace-border); }
.library-storage__track span { display: block; height: 100%; border-radius: inherit; background: var(--workspace-text-muted); }

.library-drop-overlay { position: fixed; z-index: 50; inset: 12px; display: grid; place-items: center; border: 1.5px dashed var(--workspace-border-strong); border-radius: 18px; background: color-mix(in srgb, var(--workspace-canvas) 88%, transparent); backdrop-filter: blur(9px); pointer-events: none; }
.library-drop-overlay > div { display: grid; place-items: center; gap: 7px; color: var(--workspace-text); }
.library-drop-overlay strong { margin-top: 8px; font-size: 17px; font-weight: 600; }
.library-drop-overlay span { color: var(--workspace-text-secondary); font-size: 13px; }

.library-mobile-actions { display: none; }
.library-confirm-copy { margin: 0; color: var(--workspace-confirm-text-secondary); font-size: 14px; line-height: 1.55; }
.library-confirm-button { min-height: 40px; border: 1px solid var(--workspace-confirm-cancel-border); border-radius: 999px; padding: 0 18px; color: var(--workspace-confirm-text); background: var(--workspace-confirm-surface); font: inherit; font-size: 14px; font-weight: 600; cursor: pointer; }
.library-confirm-button--danger { border: 0; color: #fff; background: var(--workspace-confirm-danger); }
.library-confirm-button:disabled { opacity: 0.55; cursor: wait; }

.library-main :deep(button:focus-visible),
.library-main :deep(input:focus-visible) { outline: 2px solid var(--workspace-text-muted); outline-offset: 2px; }
.library-main .library-icon-button:focus-visible { outline-color: var(--workspace-text); }
.library-main :deep(.library-search__input:focus-visible) { outline: none; outline-offset: 0; }

.library-menu-enter-active,
.library-menu-leave-active,
.library-selection-enter-active,
.library-selection-leave-active { transition: opacity 120ms ease, transform 120ms ease; }
.library-menu-enter-from,
.library-menu-leave-to { opacity: 0; transform: translateY(-3px) scale(0.98); }
.library-selection-enter-from,
.library-selection-leave-to { opacity: 0; transform: translateY(-4px); }

@keyframes library-spin { to { transform: rotate(1turn); } }
@keyframes library-shimmer { to { background-position: -240% 0; } }

@media (max-width: 900px) {
  .library-content { width: min(100%, 800px); padding-right: 16px; padding-left: 32px; }
  .library-list__header,
  .library-list__row { grid-template-columns: minmax(0, 1fr) 138px 78px 36px; column-gap: 10px; }
  .library-file-name > .library-select-control { left: -42px; }
}

@media (max-width: 700px) {
  .library-content { width: 100%; min-height: calc(100% - 54px); padding: 24px 16px; }
  .library-header { align-items: flex-start; flex-wrap: wrap; }
  .library-header__actions { width: 100%; }
  .library-search { width: auto; min-width: 0; flex: 1; }
  .library-toolbar { height: 52px; margin-top: 24px; }
  .library-categories { min-width: 0; }
  .library-categories button { padding: 0 10px; }
  .library-toolbar__divider { margin: 0 3px; }
  .library-list { margin-top: 20px; }
  .library-list__header,
  .library-list__row { grid-template-columns: minmax(0, 1fr) 105px 36px; }
  .library-list__header span:nth-child(3),
  .library-list__row > span[role='cell'] { display: none; }
  .library-list__header span:nth-child(4) { grid-column: 3; }
  .library-file-name { gap: 7px; }
  .library-list__visual { width: 32px; height: 32px; flex-basis: 32px; }
  .library-file-name > .library-select-control { position: relative; top: auto; left: auto; width: 44px; height: 44px; flex: 0 0 44px; }
  .library-grid { grid-template-columns: repeat(auto-fill, minmax(132px, 1fr)); gap: 15px 9px; margin-top: 20px; }
  .library-selection-bar { overflow-x: auto; margin-top: 14px; margin-bottom: -60px; white-space: nowrap; scrollbar-width: none; }
  .library-selection-bar::-webkit-scrollbar { display: none; }
  .library-selection-bar button { flex: 0 0 auto; }
  .library-storage { justify-content: center; }
}

@media (max-width: 767px) and (hover: none) and (pointer: coarse) {
  .library-mobile-actions { display: flex; height: 54px; align-items: center; padding: 0 10px; }
  .library-mobile-actions .library-icon-button { width: 44px; height: 44px; }
  .library-workspace__history--desktop { display: none; }
  .library-workspace__drawer { position: fixed; z-index: 45; inset: 0; display: flex; }
  .library-workspace__scrim { position: absolute; inset: 0; border: 0; background: rgb(15 23 42 / 0.36); backdrop-filter: blur(2px); }
  .library-workspace__drawer :deep(.chat-history) { position: relative; z-index: 1; box-shadow: 18px 0 40px rgb(15 23 42 / 0.18); }
  .library-select-control,
  .library-row-menu,
  .library-grid__menu { opacity: 1; }
}

@media (max-width: 420px) {
  .library-header__actions { align-items: stretch; }
  .library-new__trigger { padding: 0 13px; }
  .library-toolbar { align-items: flex-start; gap: 8px; }
  .library-toolbar__tools { gap: 0; }
  .library-list__header,
  .library-list__row { grid-template-columns: minmax(0, 1fr) 36px; }
  .library-list__header span:nth-child(2),
  .library-list__row time { display: none; }
  .library-list__header span:nth-child(4) { grid-column: 2; }
}

@media (prefers-reduced-motion: reduce) {
  .library-search__spinner,
  .library-skeleton span { animation: none; }
  .library-icon-button--filter,
  .library-icon-tooltip,
  .library-menu-enter-active,
  .library-menu-leave-active,
  .library-selection-enter-active,
  .library-selection-leave-active { transition: none; }
}

:global(html.dark) .library-main {
  --library-page-canvas: #000;
  --library-control-active: #2f2f2f;
  --library-control-hover: #2f2f2f;
  --library-search-border: rgb(255 255 255 / 0.1);
  --library-search-border-focus: rgb(255 255 255 / 0.15);
}
</style>

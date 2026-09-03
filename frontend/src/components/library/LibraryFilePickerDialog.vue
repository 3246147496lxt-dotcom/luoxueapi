<template>
  <BaseDialog
    :show="show"
    :title="t('library.picker.title')"
    width="wide"
    :close-on-click-outside="true"
    @close="emit('close')"
  >
    <div class="library-picker">
      <p class="library-picker__description">{{ t('library.picker.description') }}</p>

      <div class="library-picker__controls">
        <label class="library-picker__search">
          <Icon name="search" size="sm" aria-hidden="true" />
          <span class="sr-only">{{ t('library.searchLabel') }}</span>
          <input
            v-model="searchDraft"
            type="search"
            :placeholder="t('library.search')"
            :aria-label="t('library.searchLabel')"
          />
        </label>

        <div class="library-picker__categories" role="tablist" :aria-label="t('library.title')">
          <button
            v-for="option in categories"
            :key="option.value"
            type="button"
            role="tab"
            :aria-selected="category === option.value"
            :class="{ 'library-picker__category--active': category === option.value }"
            @click="setCategory(option.value)"
          >
            {{ t(option.label) }}
          </button>
        </div>
      </div>

      <div class="library-picker__results" :aria-busy="loading">
        <div v-if="loading && files.length === 0" class="library-picker__skeleton" aria-hidden="true">
          <span v-for="index in 8" :key="index"></span>
        </div>

        <div v-else-if="error" class="library-picker__state" role="alert">
          <Icon name="exclamationCircle" size="lg" aria-hidden="true" />
          <p>{{ error }}</p>
          <button type="button" @click="load()">{{ t('library.actions.retry') }}</button>
        </div>

        <div v-else-if="files.length === 0" class="library-picker__state">
          <Icon name="chatSidebarLibrary" size="lg" aria-hidden="true" />
          <p>{{ t('library.picker.empty') }}</p>
        </div>

        <ul
          v-else
          id="library-picker-results"
          class="library-picker__grid"
          role="list"
          :aria-label="t('library.picker.resultsLabel')"
        >
          <li v-for="file in files" :key="file.id">
            <button
              type="button"
              :aria-pressed="selectedIds.has(file.id)"
              :aria-disabled="!canSelect(file)"
              :class="{
                'library-picker__file--selected': selectedIds.has(file.id),
                'library-picker__file--disabled': !canSelect(file),
              }"
              :title="selectionTitle(file)"
              @click="toggle(file)"
            >
              <span class="library-picker__visual">
                <LibraryFileVisual :file="file" />
                <span v-if="selectedIds.has(file.id)" class="library-picker__check">
                  <Icon name="check" size="xs" aria-hidden="true" />
                </span>
              </span>
              <strong>{{ file.name }}</strong>
              <span class="library-picker__meta">
                <span>{{ t(`library.filters.types.${file.type}`) }}</span>
                <span aria-hidden="true">·</span>
                <span>{{ formatLibraryBytes(file.size, locale) }}</span>
              </span>
            </button>
          </li>
        </ul>

        <button
          v-if="hasMore"
          type="button"
          class="library-picker__load-more"
          :disabled="loading"
          aria-controls="library-picker-results"
          @click="loadMore"
        >
          {{ loading ? t('library.actions.loadingMore') : t('library.actions.loadMore') }}
        </button>

        <p v-if="notice" class="library-picker__notice" role="status">{{ notice }}</p>
      </div>
    </div>

    <template #footer>
      <button type="button" class="library-picker__cancel" @click="emit('close')">
        {{ t('library.picker.cancel') }}
      </button>
      <button
        type="button"
        class="library-picker__confirm"
        :disabled="selectedFiles.length === 0 || Boolean(selectionError)"
        @click="confirm"
      >
        {{ selectedFiles.length === 1
          ? t('library.picker.addOne')
          : t('library.picker.add', { count: selectedFiles.length }) }}
      </button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { isLibraryAbortError, listLibraryFiles } from '@/api/library'
import { CHAT_ATTACHMENT_TOTAL_MAX_BYTES } from '@/components/chat/chatAttachmentUi'
import type { LibraryCategory, LibraryFile } from '@/types/library'
import LibraryFileVisual from './LibraryFileVisual.vue'
import { formatLibraryBytes } from './libraryFileUi'

const props = withDefaults(defineProps<{
  show: boolean
  excludedIds?: string[]
  maxSelection?: number
  maxSelectionBytes?: number
  totalLimitBytes?: number
  supportsVision?: boolean
}>(), {
  excludedIds: () => [],
  maxSelection: 4,
  maxSelectionBytes: CHAT_ATTACHMENT_TOTAL_MAX_BYTES,
  totalLimitBytes: CHAT_ATTACHMENT_TOTAL_MAX_BYTES,
  supportsVision: true,
})

const emit = defineEmits<{
  close: []
  select: [files: LibraryFile[]]
}>()

const { t, locale } = useI18n()
const searchDraft = ref('')
const searchKeyword = ref('')
const category = ref<LibraryCategory>('all')
const files = ref<LibraryFile[]>([])
const total = ref(0)
const page = ref(1)
const selectedIds = ref<Set<string>>(new Set())
const loading = ref(false)
const error = ref('')
const notice = ref('')
let controller: AbortController | null = null
let searchTimer: ReturnType<typeof setTimeout> | undefined
let noticeTimer: ReturnType<typeof setTimeout> | undefined
let requestSequence = 0
const PAGE_SIZE = 60

const categories = [
  { value: 'all', label: 'library.categories.all' },
  { value: 'image', label: 'library.categories.image' },
  { value: 'file', label: 'library.categories.file' },
] as const
const excluded = computed(() => new Set(props.excludedIds))
const selectedFiles = computed(() => files.value.filter(({ id }) => selectedIds.value.has(id)))
const selectedBytes = computed(() => selectedFiles.value.reduce((total, file) => total + file.size, 0))
const hasMore = computed(() => files.value.length < total.value)
const selectionByteLimit = computed(() => (
  Number.isFinite(props.maxSelectionBytes) && props.maxSelectionBytes >= 0
    ? props.maxSelectionBytes
    : 0
))
const selectionError = computed(() => {
  if (selectedFiles.value.length > props.maxSelection) {
    return t('library.picker.limit', { count: props.maxSelection })
  }
  if (selectedFiles.value.some((file) => excluded.value.has(file.id))) {
    return t('library.picker.alreadySelected')
  }
  if (!props.supportsVision && selectedFiles.value.some((file) => file.type === 'image')) {
    return t('chat.attachments.errors.visionUnsupported')
  }
  if (selectedBytes.value > selectionByteLimit.value) {
    return t('chat.attachments.errors.totalTooLarge', {
      size: Math.round(props.totalLimitBytes / 1024 / 1024),
    })
  }
  return ''
})

watch(searchDraft, (value) => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    searchKeyword.value = value.trim()
    void load()
  }, 280)
})

watch(() => props.show, (visible) => {
  if (!visible) {
    controller?.abort()
    return
  }
  searchDraft.value = ''
  searchKeyword.value = ''
  category.value = 'all'
  files.value = []
  total.value = 0
  page.value = 1
  selectedIds.value = new Set()
  notice.value = ''
  void load()
})

async function load(options: { append?: boolean } = {}): Promise<void> {
  const append = options.append === true
  const nextPage = append ? page.value + 1 : 1
  controller?.abort()
  const nextController = new AbortController()
  const sequence = ++requestSequence
  controller = nextController
  loading.value = true
  error.value = ''
  try {
    const result = await listLibraryFiles({
      q: searchKeyword.value,
      category: category.value,
      source: 'all',
      type: 'all',
      sort: 'updated_desc',
      page: nextPage,
      pageSize: PAGE_SIZE,
    }, nextController.signal)
    if (sequence !== requestSequence || nextController.signal.aborted) return
    files.value = append
      ? [
          ...files.value,
          ...result.files.filter((file) => !files.value.some(({ id }) => id === file.id)),
        ]
      : result.files
    total.value = result.total
    page.value = result.page
    if (!append) {
      const visible = new Set(files.value.map(({ id }) => id))
      selectedIds.value = new Set([...selectedIds.value].filter((id) => visible.has(id)))
    }
  } catch (requestError) {
    if (isLibraryAbortError(requestError) || sequence !== requestSequence) return
    const message = requestError instanceof Error
      ? requestError.message
      : t('library.errors.generic')
    if (append && files.value.length > 0) showNotice(message)
    else error.value = message
  } finally {
    if (sequence === requestSequence) loading.value = false
  }
}

function loadMore(): void {
  if (loading.value || !hasMore.value) return
  void load({ append: true })
}

function setCategory(value: LibraryCategory): void {
  if (category.value === value) return
  category.value = value
  void load()
}

function canSelect(file: LibraryFile): boolean {
  if (excluded.value.has(file.id)) return false
  if (file.type === 'image' && !props.supportsVision) return false
  if (selectedIds.value.has(file.id)) return true
  if (selectedIds.value.size >= props.maxSelection) return false
  return file.size <= selectionByteLimit.value - selectedBytes.value
}

function selectionTitle(file: LibraryFile): string {
  if (excluded.value.has(file.id)) return t('library.picker.alreadySelected')
  if (file.type === 'image' && !props.supportsVision) {
    return t('chat.attachments.errors.visionUnsupported')
  }
  if (selectedIds.value.has(file.id)) return file.name
  if (selectedIds.value.size >= props.maxSelection) {
    return t('library.picker.limit', { count: props.maxSelection })
  }
  if (file.size > selectionByteLimit.value - selectedBytes.value) {
    return t('chat.attachments.errors.totalTooLarge', {
      size: Math.round(props.totalLimitBytes / 1024 / 1024),
    })
  }
  return file.name
}

function showNotice(message: string): void {
  notice.value = message
  if (noticeTimer) clearTimeout(noticeTimer)
  noticeTimer = setTimeout(() => { notice.value = '' }, 3600)
}

function toggle(file: LibraryFile): void {
  const next = new Set(selectedIds.value)
  if (next.has(file.id)) {
    next.delete(file.id)
    selectedIds.value = next
    return
  }
  if (!canSelect(file)) {
    showNotice(selectionTitle(file))
    return
  }
  next.add(file.id)
  selectedIds.value = next
}

function confirm(): void {
  if (selectedFiles.value.length === 0) return
  if (selectionError.value) {
    showNotice(selectionError.value)
    return
  }
  emit('select', selectedFiles.value)
  emit('close')
}

onBeforeUnmount(() => {
  controller?.abort()
  if (searchTimer) clearTimeout(searchTimer)
  if (noticeTimer) clearTimeout(noticeTimer)
})
</script>

<style scoped>
.library-picker {
  min-height: 460px;
}

.library-picker__description {
  margin: -3px 0 16px;
  color: var(--workspace-text-secondary);
  font-size: 14px;
}

.library-picker__controls {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 16px;
}

.library-picker__search {
  display: flex;
  width: min(320px, 50%);
  height: 40px;
  align-items: center;
  gap: 9px;
  border: 1px solid var(--workspace-border);
  border-radius: 999px;
  padding: 0 13px;
  color: var(--workspace-text-muted);
  background: var(--workspace-surface-subtle);
}

.library-picker__search:focus-within {
  border-color: var(--workspace-border-strong);
}

.library-picker__search input {
  min-width: 0;
  flex: 1;
  border: 0;
  color: var(--workspace-text);
  background: transparent;
  outline: none;
  font: inherit;
  font-size: 14px;
}

.library-picker__categories {
  display: flex;
  gap: 3px;
}

.library-picker__categories button {
  min-height: 34px;
  border: 0;
  border-radius: 999px;
  padding: 0 12px;
  color: var(--workspace-text-secondary);
  background: transparent;
  font: inherit;
  font-size: 13px;
  cursor: pointer;
}

.library-picker__categories button:hover,
.library-picker__categories .library-picker__category--active {
  color: var(--workspace-text);
  background: var(--workspace-selected);
}

.library-picker__results {
  position: relative;
  min-height: 350px;
}

.library-picker__grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.library-picker__file {
  width: 100%;
}

.library-picker__grid button {
  display: grid;
  width: 100%;
  min-width: 0;
  gap: 5px;
  border: 0;
  border-radius: 12px;
  padding: 7px;
  color: var(--workspace-text);
  background: transparent;
  text-align: left;
  cursor: pointer;
}

.library-picker__grid button:hover {
  background: var(--workspace-hover);
}

.library-picker__grid button:focus-visible {
  outline: 2px solid var(--workspace-text-muted);
  outline-offset: 2px;
}

.library-picker__grid button.library-picker__file--selected {
  background: var(--workspace-selected);
}

.library-picker__grid button.library-picker__file--disabled {
  opacity: 0.48;
  cursor: not-allowed;
}

.library-picker__visual {
  position: relative;
  display: block;
  aspect-ratio: 4 / 3;
  min-width: 0;
}

.library-picker__check {
  position: absolute;
  top: 7px;
  right: 7px;
  display: grid;
  width: 22px;
  height: 22px;
  place-items: center;
  border: 2px solid var(--workspace-canvas);
  border-radius: 50%;
  color: var(--workspace-canvas);
  background: var(--workspace-text);
}

.library-picker__grid strong,
.library-picker__meta {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.library-picker__grid strong {
  font-size: 13px;
  font-weight: 500;
}

.library-picker__meta {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 4px;
  color: var(--workspace-text-muted);
  font-size: 11px;
}

.library-picker__meta span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.library-picker__meta span:first-child { flex: 1; white-space: nowrap; }
.library-picker__meta span:nth-child(2),
.library-picker__meta span:last-child { flex: none; white-space: nowrap; }

.library-picker__state {
  display: grid;
  min-height: 320px;
  place-items: center;
  align-content: center;
  gap: 9px;
  color: var(--workspace-text-secondary);
  text-align: center;
}

.library-picker__state p { margin: 0; }
.library-picker__state button {
  border: 1px solid var(--workspace-border-strong);
  border-radius: 999px;
  padding: 7px 14px;
  color: var(--workspace-text);
  background: transparent;
  cursor: pointer;
}

.library-picker__skeleton {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.library-picker__skeleton span {
  aspect-ratio: 4 / 3;
  border-radius: 10px;
  background: var(--workspace-surface-subtle);
  animation: library-picker-pulse 1.2s ease-in-out infinite alternate;
}

.library-picker__notice {
  position: sticky;
  bottom: 8px;
  width: max-content;
  max-width: 100%;
  margin: 12px auto 0;
  border-radius: 999px;
  padding: 7px 12px;
  color: var(--workspace-text);
  background: var(--workspace-popup-surface);
  box-shadow: var(--workspace-popover-shadow);
  font-size: 12px;
}

.library-picker__load-more {
  display: flex;
  min-height: 44px;
  align-items: center;
  margin: 16px auto 0;
  border: 1px solid var(--workspace-border-strong);
  border-radius: 999px;
  padding: 0 16px;
  color: var(--workspace-text);
  background: transparent;
  font: inherit;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
}

.library-picker__load-more:hover { background: var(--workspace-hover); }
.library-picker__load-more:disabled { opacity: 0.55; cursor: wait; }

.library-picker__cancel,
.library-picker__confirm {
  min-height: 38px;
  border-radius: 999px;
  padding: 0 16px;
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
}

.library-picker__cancel {
  border: 1px solid var(--workspace-border-strong);
  color: var(--workspace-text);
  background: transparent;
}

.library-picker__confirm {
  border: 0;
  color: var(--workspace-canvas);
  background: var(--workspace-text);
}

.library-picker__confirm:disabled { opacity: 0.45; cursor: not-allowed; }

@keyframes library-picker-pulse { to { opacity: 0.55; } }

@media (max-width: 720px) {
  .library-picker { min-height: 60vh; }
  .library-picker__controls { align-items: stretch; flex-direction: column; }
  .library-picker__search { width: 100%; }
  .library-picker__grid,
  .library-picker__skeleton { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}

@media (pointer: coarse) {
  .library-picker__search,
  .library-picker__categories button,
  .library-picker__cancel,
  .library-picker__confirm { min-height: 44px; }
}

@media (prefers-reduced-motion: reduce) {
  .library-picker__skeleton span { animation: none; }
}
</style>

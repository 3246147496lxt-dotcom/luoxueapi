<template>
  <Transition name="library-viewer">
    <section
      v-if="show && file"
      ref="viewerRef"
      class="library-viewer"
      role="dialog"
      :aria-label="file.name"
      tabindex="-1"
    >
      <header class="library-viewer__header">
        <button
          ref="closeButtonRef"
          type="button"
          class="library-viewer__icon-button library-viewer__close"
          :aria-label="t('library.preview.closeViewer')"
          @click="emit('close')"
        >
          <Icon name="x" size="md" aria-hidden="true" />
        </button>

        <nav class="library-viewer__breadcrumb" :aria-label="t('library.preview.breadcrumb')">
          <button type="button" @click="emit('close')">{{ t('library.title') }}</button>
          <span aria-hidden="true">/</span>
          <strong :title="file.name">{{ file.name }}</strong>
        </nav>

        <div class="library-viewer__actions" role="toolbar" :aria-label="t('library.preview.toolbarLabel')">
          <button
            type="button"
            class="library-viewer__pill library-viewer__pill--danger"
            :aria-label="t('library.actions.deleteNamed', { name: file.name })"
            @click="emit('remove', file)"
          >
            <Icon name="trash" size="sm" aria-hidden="true" />
            <span>{{ t('library.actions.delete') }}</span>
          </button>

          <button
            type="button"
            class="library-viewer__icon-button"
            :disabled="downloading"
            :aria-label="t('library.actions.downloadNamed', { name: file.name })"
            @click="emit('download', file)"
          >
            <Icon name="download" size="md" aria-hidden="true" />
          </button>

          <button
            v-if="file.type === 'image'"
            type="button"
            class="library-viewer__icon-button"
            :aria-label="actualSize ? t('library.preview.fitToWindow') : t('library.preview.actualSize')"
            :aria-pressed="actualSize"
            @click="actualSize = !actualSize"
          >
            <Icon name="maximize" size="md" aria-hidden="true" />
          </button>
        </div>
      </header>

      <div class="library-viewer__body">
        <div
          v-if="canPreview"
          class="library-viewer__stage"
          :class="{ 'library-viewer__stage--actual': actualSize && file.type === 'image' }"
        >
          <div v-if="loading" class="library-viewer__loading" role="status">
            <span aria-hidden="true"></span>
            {{ t('library.preview.loading') }}
          </div>

          <img
            v-if="previewUrl && file.type === 'image' && !loadFailed"
            :class="{ 'library-viewer__asset--loading': loading }"
            :src="previewUrl"
            :alt="t('library.preview.imageAlt', { name: file.name })"
            @load="previewLoaded"
            @error="previewFailed"
          />
          <iframe
            v-else-if="previewUrl && file.type === 'pdf' && !loadFailed"
            :class="{ 'library-viewer__asset--loading': loading }"
            :src="previewUrl"
            :title="t('library.preview.pdfTitle', { name: file.name })"
            sandbox=""
            referrerpolicy="no-referrer"
            @load="previewLoaded"
          ></iframe>

          <div v-if="!loading && loadFailed" class="library-viewer__failure" role="alert">
            <Icon name="exclamationCircle" size="lg" aria-hidden="true" />
            <strong>{{ t('library.errors.previewFailed') }}</strong>
            <button type="button" @click="loadPreview">{{ t('library.actions.retry') }}</button>
          </div>
        </div>

        <div v-else class="library-viewer__unavailable">
          <div class="library-viewer__file-icon">
            <Icon name="document" size="xl" aria-hidden="true" />
            <span>{{ libraryFileExtension(file) }}</span>
          </div>
          <h2>{{ t('library.preview.unavailableTitle') }}</h2>
          <p>{{ t('library.preview.unavailableDescription') }}</p>
          <button type="button" :disabled="downloading" @click="emit('download', file)">
            <Icon name="download" size="sm" aria-hidden="true" />
            {{ t('library.preview.download') }}
          </button>
        </div>
      </div>
    </section>
  </Transition>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { getLibraryPreview, isLibraryAbortError } from '@/api/library'
import type { LibraryFile } from '@/types/library'
import { libraryFileCanPreview, libraryFileExtension } from './libraryFileUi'

const props = withDefaults(defineProps<{
  show: boolean
  file: LibraryFile | null
  downloading?: boolean
}>(), {
  downloading: false,
})

const emit = defineEmits<{
  close: []
  download: [file: LibraryFile]
  remove: [file: LibraryFile]
}>()

const { t } = useI18n()
const viewerRef = ref<HTMLElement | null>(null)
const closeButtonRef = ref<HTMLButtonElement | null>(null)
const loading = ref(false)
const loadFailed = ref(false)
const previewUrl = ref('')
const actualSize = ref(false)
const canPreview = computed(() => Boolean(props.file && libraryFileCanPreview(props.file)))
let controller: AbortController | null = null
let previousActiveElement: HTMLElement | null = null
let viewerActive = false

function releasePreview(): void {
  controller?.abort()
  controller = null
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
  previewUrl.value = ''
  loading.value = false
  loadFailed.value = false
  actualSize.value = false
}

async function loadPreview(): Promise<void> {
  releasePreview()
  const file = props.file
  if (!props.show || !file || !libraryFileCanPreview(file)) return
  const nextController = new AbortController()
  controller = nextController
  loading.value = true
  try {
    const blob = await getLibraryPreview(file.id, nextController.signal)
    if (nextController.signal.aborted || controller !== nextController) return
    previewUrl.value = URL.createObjectURL(blob)
  } catch (error) {
    if (!isLibraryAbortError(error)) {
      loadFailed.value = true
      loading.value = false
    }
  } finally {
    if (controller === nextController) controller = null
  }
}

function previewLoaded(): void {
  loading.value = false
  loadFailed.value = false
}

function previewFailed(): void {
  loading.value = false
  loadFailed.value = true
}

function handleDocumentKeydown(event: KeyboardEvent): void {
  if (!viewerActive || event.key !== 'Escape') return
  event.preventDefault()
  emit('close')
}

async function activateViewer(): Promise<void> {
  if (viewerActive || typeof document === 'undefined') return
  viewerActive = true
  previousActiveElement = document.activeElement instanceof HTMLElement
    ? document.activeElement
    : null
  document.addEventListener('keydown', handleDocumentKeydown)
  await nextTick()
  if (viewerActive) closeButtonRef.value?.focus({ preventScroll: true })
}

function deactivateViewer(restoreFocus: boolean): void {
  if (!viewerActive || typeof document === 'undefined') return
  viewerActive = false
  document.removeEventListener('keydown', handleDocumentKeydown)
  const returnTarget = previousActiveElement
  previousActiveElement = null
  if (restoreFocus && returnTarget?.isConnected) {
    void nextTick(() => returnTarget.focus({ preventScroll: true }))
  }
}

watch(
  () => [props.show, props.file?.id, props.file?.updatedAt],
  () => void loadPreview(),
  { immediate: true },
)

watch(
  () => props.show && Boolean(props.file),
  (visible) => {
    if (visible) void activateViewer()
    else deactivateViewer(true)
  },
  { immediate: true, flush: 'post' },
)

onBeforeUnmount(() => {
  releasePreview()
  deactivateViewer(false)
})
</script>

<style scoped>
.library-viewer {
  --library-viewer-canvas: #fcfcfc;
  --library-viewer-surface: #fff;
  --library-viewer-hover: rgb(0 0 0 / 0.05);
  display: flex;
  min-width: 0;
  min-height: 0;
  flex: 1;
  flex-direction: column;
  color: var(--workspace-text);
  background: var(--library-viewer-canvas);
}

.library-viewer__header {
  position: relative;
  z-index: 10;
  display: flex;
  height: 56px;
  flex: 0 0 56px;
  align-items: center;
  gap: 8px;
  border-bottom: 1px solid var(--workspace-divider);
  padding: 0 16px 0 8px;
  background: var(--library-viewer-canvas);
}

.library-viewer__icon-button {
  position: relative;
  display: grid;
  width: 36px;
  height: 36px;
  flex: 0 0 36px;
  place-items: center;
  border: 0;
  border-radius: 8px;
  padding: 0;
  color: var(--workspace-text);
  background: transparent;
  cursor: pointer;
}

.library-viewer__icon-button:hover:not(:disabled) { background: var(--library-viewer-hover); }
.library-viewer__icon-button:disabled { opacity: 0.5; cursor: wait; }
.library-viewer__icon-button:focus-visible,
.library-viewer__pill:focus-visible,
.library-viewer__breadcrumb button:focus-visible,
.library-viewer__unavailable button:focus-visible {
  outline: 2px solid var(--workspace-text);
  outline-offset: 2px;
}

.library-viewer__breadcrumb {
  display: flex;
  min-width: 0;
  flex: 1;
  align-items: center;
  gap: 10px;
  font-size: 14px;
  font-weight: 400;
  line-height: 18px;
}

.library-viewer__breadcrumb button {
  flex: 0 0 auto;
  border: 0;
  border-radius: 5px;
  padding: 2px 0;
  color: var(--workspace-identity-text-tertiary);
  background: transparent;
  font: inherit;
  cursor: pointer;
}

.library-viewer__breadcrumb button:hover { color: var(--workspace-text); }
.library-viewer__breadcrumb > span { color: var(--workspace-identity-text-tertiary); }
.library-viewer__breadcrumb strong {
  min-width: 0;
  overflow: hidden;
  color: var(--workspace-text);
  font: inherit;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.library-viewer__actions {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 8px;
}

.library-viewer__pill {
  display: inline-flex;
  height: 36px;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border: 1px solid var(--workspace-border-strong);
  border-radius: 999px;
  padding: 0 12px;
  color: var(--workspace-text);
  background: var(--library-viewer-surface);
  font: inherit;
  font-size: 14px;
  font-weight: 500;
  line-height: 20px;
  cursor: pointer;
}

.library-viewer__pill:hover { background: var(--library-viewer-hover); }
.library-viewer__pill--danger:hover { color: var(--workspace-menu-danger); }

.library-viewer__body {
  min-width: 0;
  min-height: 0;
  flex: 1;
  overflow: hidden auto;
  padding: 36px 16px;
  overscroll-behavior: contain;
}

.library-viewer__stage {
  position: relative;
  display: flex;
  width: 100%;
  height: 100%;
  min-height: 360px;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border-radius: 4px;
}

.library-viewer__stage--actual {
  align-items: flex-start;
  justify-content: flex-start;
  overflow: auto;
}

.library-viewer__stage img {
  display: block;
  max-width: 100%;
  max-height: 100%;
  border: 1px solid rgb(0 0 0 / 0.05);
  object-fit: contain;
  box-shadow: 0 20px 25px -5px rgb(0 0 0 / 0.1), 0 8px 10px -6px rgb(0 0 0 / 0.1);
}

.library-viewer__stage--actual img {
  max-width: none;
  max-height: none;
}

.library-viewer__stage iframe {
  display: block;
  width: min(100%, 1000px);
  height: 100%;
  min-height: 620px;
  border: 1px solid rgb(0 0 0 / 0.08);
  background: #fff;
  box-shadow: 0 20px 25px -5px rgb(0 0 0 / 0.1), 0 8px 10px -6px rgb(0 0 0 / 0.1);
}

.library-viewer__asset--loading { opacity: 0; }

.library-viewer__loading,
.library-viewer__failure {
  position: absolute;
  z-index: 1;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: var(--workspace-text-secondary);
  font-size: 14px;
}

.library-viewer__loading span {
  width: 20px;
  height: 20px;
  border: 2px solid var(--workspace-border-strong);
  border-top-color: var(--workspace-text);
  border-radius: 50%;
  animation: library-viewer-spin 800ms linear infinite;
}

.library-viewer__failure {
  flex-direction: column;
  text-align: center;
}

.library-viewer__failure button,
.library-viewer__unavailable > button {
  display: inline-flex;
  min-height: 38px;
  align-items: center;
  justify-content: center;
  gap: 7px;
  border: 1px solid var(--workspace-border-strong);
  border-radius: 999px;
  padding: 0 15px;
  color: var(--workspace-text);
  background: transparent;
  font: inherit;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
}

.library-viewer__failure button:hover,
.library-viewer__unavailable > button:hover { background: var(--library-viewer-hover); }

.library-viewer__unavailable {
  display: grid;
  min-height: 100%;
  place-items: center;
  align-content: center;
  text-align: center;
}

.library-viewer__file-icon {
  position: relative;
  display: grid;
  width: 88px;
  height: 104px;
  place-items: center;
  border-radius: 12px;
  color: var(--workspace-text-secondary);
  background: var(--workspace-surface-subtle);
}

.library-viewer__file-icon span {
  position: absolute;
  right: 8px;
  bottom: 8px;
  font-size: 10px;
  font-weight: 700;
}

.library-viewer__unavailable h2 {
  margin: 22px 0 5px;
  font-size: 17px;
  font-weight: 600;
}

.library-viewer__unavailable p {
  max-width: 46ch;
  margin: 0 0 20px;
  color: var(--workspace-text-secondary);
  font-size: 14px;
  line-height: 1.55;
}

@keyframes library-viewer-spin { to { transform: rotate(1turn); } }

.library-viewer-enter-active,
.library-viewer-leave-active { transition: opacity 160ms ease; }
.library-viewer-enter-from,
.library-viewer-leave-to { opacity: 0; }

@media (max-width: 700px) {
  .library-viewer__header { padding-right: 8px; }
  .library-viewer__breadcrumb { gap: 7px; }
  .library-viewer__actions { gap: 2px; }
  .library-viewer__icon-button { width: 44px; height: 44px; flex-basis: 44px; }
  .library-viewer__pill { width: 44px; height: 44px; padding: 0; }
  .library-viewer__pill > span { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0 0 0 0); }
  .library-viewer__body { padding: 20px 12px calc(20px + env(safe-area-inset-bottom)); }
  .library-viewer__stage { min-height: 280px; }
  .library-viewer__stage iframe { min-height: 520px; }
}

@media (max-width: 460px) {
  .library-viewer__breadcrumb button,
  .library-viewer__breadcrumb > span { display: none; }
}

@media (prefers-reduced-motion: reduce) {
  .library-viewer,
  .library-viewer__loading span { animation: none; transition: none; }
}

:global(html.dark) .library-viewer {
  --library-viewer-canvas: #000;
  --library-viewer-surface: #171717;
  --library-viewer-hover: rgb(255 255 255 / 0.1);
}

:global(html.dark) .library-viewer__stage img { border-color: rgb(255 255 255 / 0.1); }
</style>

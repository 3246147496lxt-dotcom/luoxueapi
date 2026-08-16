<template>
  <aside
    v-if="uploads.length > 0"
    class="library-upload-tray"
    role="region"
    :aria-label="title"
    :aria-busy="activeCount > 0 ? 'true' : 'false'"
  >
    <header class="library-upload-tray__header">
      <h2>{{ title }}</h2>

      <div class="library-upload-tray__summary" aria-hidden="true">
        <svg
          class="library-upload-tray__summary-ring"
          :class="{ 'library-upload-tray__summary-ring--complete': allSettled }"
          viewBox="0 0 20 20"
        >
          <circle class="library-upload-tray__summary-track" cx="10" cy="10" r="8" />
          <circle
            class="library-upload-tray__summary-value"
            cx="10"
            cy="10"
            r="8"
            :style="{ strokeDashoffset: String(progressRingOffset) }"
          />
        </svg>
        <strong>{{ settledCount }}/{{ uploads.length }}</strong>
      </div>

      <button
        type="button"
        class="library-upload-tray__icon-button"
        :aria-label="collapsed ? t('library.uploadTray.expand') : t('library.uploadTray.collapse')"
        :aria-expanded="!collapsed"
        :aria-controls="bodyId"
        @click="collapsed = !collapsed"
      >
        <Icon
          name="chevronDown"
          size="xs"
          aria-hidden="true"
          :class="{ 'library-upload-tray__chevron--collapsed': collapsed }"
        />
      </button>

      <button
        v-if="allSettled"
        type="button"
        class="library-upload-tray__icon-button"
        :aria-label="t('library.uploadTray.dismiss')"
        @click="emit('dismiss')"
      >
        <Icon name="x" size="xs" aria-hidden="true" />
      </button>
    </header>

    <p class="library-upload-tray__sr-only" aria-live="polite" aria-atomic="true">
      {{ t('library.uploadTray.summary', { completed: settledCount, total: uploads.length }) }}
    </p>

    <div
      v-show="!collapsed"
      :id="bodyId"
      class="library-upload-tray__body"
      role="list"
      :aria-hidden="collapsed ? 'true' : undefined"
    >
      <article
        v-for="upload in uploads"
        :key="upload.key"
        class="library-upload-tray__row"
        :class="`library-upload-tray__row--${upload.state}`"
        role="listitem"
      >
        <span
          class="library-upload-tray__state"
          :class="`library-upload-tray__state--${upload.state}`"
          :role="isActiveState(upload.state) ? 'progressbar' : undefined"
          :aria-label="isActiveState(upload.state)
            ? t('library.uploadTray.progressLabel', { name: upload.file.name })
            : undefined"
          :aria-valuemin="isActiveState(upload.state) ? 0 : undefined"
          :aria-valuemax="isActiveState(upload.state) ? 100 : undefined"
          :aria-valuenow="isActiveState(upload.state) ? upload.progress : undefined"
          :aria-valuetext="isActiveState(upload.state) ? uploadStatus(upload) : undefined"
        >
          <span
            v-if="isActiveState(upload.state)"
            class="library-upload-tray__spinner"
            aria-hidden="true"
          ></span>
          <svg
            v-else-if="upload.state === 'ready'"
            class="library-upload-tray__ready-check"
            viewBox="0 0 12 12"
            aria-hidden="true"
          >
            <path d="M2.25 6.375l3 3 4.5-6.75" />
          </svg>
          <Icon v-else name="exclamationCircle" size="sm" aria-hidden="true" />
        </span>

        <div class="library-upload-tray__copy">
          <strong :title="upload.file.name">{{ upload.file.name }}</strong>
          <span :title="upload.state === 'error' ? uploadStatus(upload) : undefined">
            {{ uploadStatus(upload) }}
          </span>
        </div>

        <button
          v-if="upload.state === 'error'"
          type="button"
          class="library-upload-tray__retry"
          :aria-label="t('library.uploadTray.retryNamed', { name: upload.file.name })"
          @click="emit('retry', upload.key)"
        >
          {{ t('library.uploadTray.retry') }}
        </button>
      </article>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { LibraryUploadItem, LibraryUploadState } from '@/types/library'

const props = defineProps<{
  uploads: LibraryUploadItem[]
}>()

const emit = defineEmits<{
  dismiss: []
  retry: [key: string]
}>()

const { t } = useI18n()
const bodyId = 'library-upload-tray-body'
const ringCircumference = 2 * Math.PI * 8
const collapsed = ref(false)

function isActiveState(state: LibraryUploadState): boolean {
  return state === 'queued' || state === 'uploading' || state === 'processing'
}

const activeCount = computed(() => props.uploads.filter(({ state }) => (
  isActiveState(state)
)).length)
const settledCount = computed(() => props.uploads.length - activeCount.value)
const allSettled = computed(() => props.uploads.length > 0 && activeCount.value === 0)
const title = computed(() => t('library.uploadTray.title', { count: props.uploads.length }))
const progressRingOffset = computed(() => {
  if (props.uploads.length === 0) return ringCircumference
  return ringCircumference * (1 - (settledCount.value / props.uploads.length))
})

watch(
  () => [props.uploads.length, activeCount.value] as const,
  ([nextLength, nextActive], [previousLength, previousActive]) => {
    if (nextLength > previousLength || nextActive > previousActive) collapsed.value = false
  },
)

function uploadErrorStatus(upload: LibraryUploadItem): string {
  const code = (upload.errorCode ?? '').toUpperCase()
  if (code.includes('EMPTY')) return t('library.errors.emptyFile')
  if (code.includes('TOO_LARGE') || code.includes('SIZE_LIMIT')) {
    return t('library.errors.tooLarge')
  }
  if (code.includes('UNSUPPORTED') || code.includes('INVALID_TYPE')) {
    return t('library.errors.unsupportedType')
  }
  if (code.includes('STORAGE') || code.includes('QUOTA')) {
    return t('library.errors.storageFull')
  }
  if (code.includes('RATE_LIMIT')) return t('library.uploadTray.errors.rateLimited')
  return upload.errorMessage || t('library.uploadTray.status.error')
}

function uploadStatus(upload: LibraryUploadItem): string {
  switch (upload.state) {
    case 'queued':
      return t('library.uploadTray.status.queued')
    case 'uploading':
      return t('library.uploadTray.status.uploading', { progress: upload.progress })
    case 'processing':
      return t('library.uploadTray.status.processing')
    case 'ready':
      return t('library.uploadTray.status.ready')
    case 'error':
      return uploadErrorStatus(upload)
  }
}
</script>

<style scoped>
.library-upload-tray {
  position: fixed;
  z-index: 42;
  inset-inline-end: 16px;
  bottom: 16px;
  display: grid;
  box-sizing: border-box;
  width: min(420px, calc(100vw - 32px));
  max-height: min(372px, calc(100dvh - 32px));
  overflow: hidden;
  border: 0;
  border-radius: 16px;
  color: var(--workspace-text);
  background: var(--workspace-popover-surface);
  box-shadow:
    inset 0 0 0 1px var(--workspace-popover-border),
    var(--workspace-popover-shadow);
  font-family: -apple-system-body, ui-sans-serif, -apple-system, system-ui, "Segoe UI", Helvetica, Arial, sans-serif;
}

.library-upload-tray__header {
  display: flex;
  box-sizing: border-box;
  min-width: 0;
  min-height: 52px;
  align-items: center;
  gap: 4px;
  padding: 8px 10px 8px 20px;
}

.library-upload-tray__header h2 {
  min-width: 0;
  flex: 1;
  margin: 0;
  overflow: hidden;
  font-size: 14px;
  font-weight: 600;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.library-upload-tray__summary {
  display: inline-flex;
  min-width: 50px;
  align-items: center;
  justify-content: flex-end;
  gap: 7px;
  color: var(--workspace-text);
}

.library-upload-tray__summary strong {
  min-width: 25px;
  font-size: 13px;
  font-weight: 600;
  line-height: 18px;
  text-align: start;
}

.library-upload-tray__summary-ring {
  width: 18px;
  height: 18px;
  overflow: visible;
  transform: rotate(-90deg);
}

.library-upload-tray__summary-ring circle {
  fill: none;
  stroke-width: 2;
}

.library-upload-tray__summary-track { stroke: var(--workspace-border); }
.library-upload-tray__summary-value {
  stroke: var(--workspace-text);
  stroke-dasharray: 50.2655;
  stroke-linecap: round;
  transition: stroke-dashoffset 180ms cubic-bezier(0.2, 0, 0, 1);
}

.library-upload-tray__icon-button {
  display: grid;
  width: 36px;
  height: 36px;
  flex: 0 0 36px;
  place-items: center;
  border: 0;
  border-radius: 999px;
  padding: 0;
  color: var(--workspace-text-secondary);
  background: transparent;
  cursor: pointer;
}

.library-upload-tray__icon-button svg { transition: transform 150ms cubic-bezier(0.2, 0, 0, 1); }
.library-upload-tray__icon-button .library-upload-tray__chevron--collapsed { transform: rotate(180deg); }

.library-upload-tray__body {
  min-height: 0;
  overflow-y: auto;
  border-top: 1px solid var(--workspace-divider);
  overscroll-behavior: contain;
  scrollbar-width: thin;
}

.library-upload-tray__row {
  display: flex;
  box-sizing: border-box;
  min-width: 0;
  min-height: 62px;
  align-items: center;
  gap: 12px;
  padding: 10px 20px;
}

.library-upload-tray__row + .library-upload-tray__row {
  border-top: 1px solid var(--workspace-divider);
}

.library-upload-tray__state {
  display: grid;
  box-sizing: border-box;
  width: 36px;
  height: 36px;
  flex: 0 0 36px;
  place-items: center;
  border: 1px solid var(--workspace-border);
  border-radius: 9px;
  color: var(--workspace-text-secondary);
  background: var(--workspace-card-surface);
}

.library-upload-tray__state--ready {
  border-color: transparent;
  color: #fff;
  background: var(--workspace-work-success);
}

.library-upload-tray__ready-check {
  width: 12px;
  height: 12px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.25;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.library-upload-tray__state--error {
  color: var(--workspace-menu-danger);
  background: var(--workspace-menu-danger-surface);
}

.library-upload-tray__spinner {
  box-sizing: border-box;
  width: 17px;
  height: 17px;
  border: 2px solid color-mix(in srgb, var(--workspace-text-muted) 28%, transparent);
  border-top-color: var(--workspace-text-secondary);
  border-radius: 50%;
  animation: library-upload-tray-spin 760ms linear infinite;
}

.library-upload-tray__state--queued .library-upload-tray__spinner {
  border-color: var(--workspace-border-strong);
  animation: none;
}

.library-upload-tray__copy {
  display: grid;
  min-width: 0;
  flex: 1;
  gap: 1px;
}

.library-upload-tray__copy strong,
.library-upload-tray__copy span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.library-upload-tray__copy strong {
  font-size: 14px;
  font-weight: 500;
  line-height: 20px;
}

.library-upload-tray__copy span {
  color: var(--workspace-text-secondary);
  font-size: 12px;
  line-height: 17px;
}

.library-upload-tray__row--error .library-upload-tray__copy span {
  color: var(--workspace-menu-danger);
}

.library-upload-tray__retry {
  min-height: 36px;
  flex: 0 0 auto;
  border: 0;
  border-radius: 8px;
  padding: 0 10px;
  color: var(--workspace-text);
  background: transparent;
  font: inherit;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.library-upload-tray__sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
}

.library-upload-tray button:focus-visible {
  outline: 2px solid var(--workspace-text);
  outline-offset: 2px;
}

@media (hover: hover) and (pointer: fine) {
  .library-upload-tray__icon-button:hover,
  .library-upload-tray__retry:hover { background: var(--workspace-popover-hover); }
}

@media (max-width: 640px) {
  .library-upload-tray {
    inset-inline: 12px;
    bottom: max(12px, env(safe-area-inset-bottom));
    width: auto;
    max-height: min(360px, calc(100dvh - 24px - env(safe-area-inset-bottom)));
  }

  .library-upload-tray__header { padding-inline: 16px 8px; }
  .library-upload-tray__row { padding-inline: 16px; }
  .library-upload-tray__icon-button { width: 44px; height: 44px; flex-basis: 44px; }
  .library-upload-tray__retry { min-height: 44px; }
}

@media (forced-colors: active) {
  .library-upload-tray { outline: 1px solid CanvasText; outline-offset: -1px; }
  .library-upload-tray__state--ready,
  .library-upload-tray__state--error { border-color: CanvasText; }
}

@media (prefers-reduced-motion: reduce) {
  .library-upload-tray__summary-value,
  .library-upload-tray__icon-button svg { transition: none; }
  .library-upload-tray__spinner { animation: none; }
}

@keyframes library-upload-tray-spin {
  to { transform: rotate(1turn); }
}
</style>

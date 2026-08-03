<template>
  <section class="skill-uploader" aria-labelledby="skill-uploader-title">
    <div class="skill-uploader__heading">
      <div>
        <h3 id="skill-uploader-title">{{ t('admin.skills.editor.package.uploadTitle') }}</h3>
        <p>{{ t('admin.skills.editor.package.uploadHint') }}</p>
      </div>
      <Icon name="upload" size="md" aria-hidden="true" />
    </div>

    <div
      class="skill-uploader__dropzone"
      :class="{
        'skill-uploader__dropzone--active': dragging,
        'skill-uploader__dropzone--selected': selectedFile,
        'skill-uploader__dropzone--disabled': disabled || uploading,
      }"
      @dragenter.prevent="dragging = true"
      @dragover.prevent="dragging = true"
      @dragleave.prevent="dragging = false"
      @drop.prevent="handleDrop"
    >
      <input
        ref="fileInput"
        class="sr-only"
        type="file"
        accept=".zip,application/zip"
        :disabled="disabled || uploading"
        @change="handleFileChange"
      />
      <div class="skill-uploader__file-icon" aria-hidden="true">
        <Icon :name="selectedFile ? 'document' : 'upload'" size="lg" />
      </div>
      <div class="skill-uploader__drop-copy">
        <p v-if="selectedFile" class="skill-uploader__file-name">
          {{ t('admin.skills.editor.package.selectedFile', {
            name: selectedFile.name,
            size: formatBytes(selectedFile.size),
          }) }}
        </p>
        <p v-else>{{ t('admin.skills.editor.package.dropHint') }}</p>
        <button
          type="button"
          class="skill-uploader__choose"
          :disabled="disabled || uploading"
          @click="fileInput?.click()"
        >
          {{ selectedFile
            ? t('admin.skills.editor.package.replaceFile')
            : t('admin.skills.editor.package.chooseFile') }}
        </button>
      </div>
    </div>

    <div class="skill-uploader__fields">
      <div>
        <label for="skill-package-version" class="input-label">
          {{ t('admin.skills.editor.package.version') }}
        </label>
        <input
          id="skill-package-version"
          v-model="version"
          class="input"
          type="text"
          autocomplete="off"
          :disabled="disabled || uploading"
          :placeholder="t('admin.skills.editor.package.versionPlaceholder')"
        />
      </div>
      <div class="skill-uploader__changelog">
        <label for="skill-package-changelog" class="input-label">
          {{ t('admin.skills.editor.package.changelog') }}
        </label>
        <textarea
          id="skill-package-changelog"
          v-model="changelog"
          class="input"
          rows="3"
          :disabled="disabled || uploading"
          :placeholder="t('admin.skills.editor.package.changelogPlaceholder')"
        ></textarea>
      </div>
    </div>

    <div v-if="validationErrors.length" class="skill-uploader__errors" role="alert">
      <p v-for="message in validationErrors" :key="message">
        <Icon name="exclamationCircle" size="sm" aria-hidden="true" />
        {{ message }}
      </p>
    </div>

    <div class="skill-uploader__footer">
      <p v-if="disabled" class="skill-uploader__save-first">
        {{ t('admin.skills.editor.package.saveFirst') }}
      </p>
      <button
        type="button"
        class="btn btn-secondary"
        :disabled="disabled || uploading"
        data-testid="upload-skill-version"
        @click="submit"
      >
        <Icon name="shield" size="sm" aria-hidden="true" />
        <span class="ml-1.5">
          {{ uploading
            ? t('admin.skills.editor.package.uploading')
            : t('admin.skills.editor.package.upload') }}
        </span>
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UploadSkillVersionRequest } from '@/api/admin/skills'
import Icon from '@/components/icons/Icon.vue'

defineProps<{
  disabled?: boolean
  uploading?: boolean
}>()

const emit = defineEmits<{
  upload: [request: UploadSkillVersionRequest]
}>()

const { t } = useI18n()
const fileInput = ref<HTMLInputElement | null>(null)
const selectedFile = ref<File | null>(null)
const version = ref('')
const changelog = ref('')
const dragging = ref(false)
const validationErrors = ref<string[]>([])

function chooseFile(file: File | null): void {
  validationErrors.value = []
  if (!file) {
    selectedFile.value = null
    return
  }
  if (!file.name.toLowerCase().endsWith('.zip')) {
    selectedFile.value = null
    validationErrors.value = [t('admin.skills.editor.package.zipRequired')]
    return
  }
  selectedFile.value = file
}

function handleFileChange(event: Event): void {
  chooseFile((event.target as HTMLInputElement).files?.[0] ?? null)
}

function handleDrop(event: DragEvent): void {
  dragging.value = false
  chooseFile(event.dataTransfer?.files?.[0] ?? null)
}

function submit(): void {
  const errors: string[] = []
  if (!selectedFile.value) errors.push(t('admin.skills.editor.package.fileRequired'))
  if (!version.value.trim()) errors.push(t('admin.skills.editor.package.versionRequired'))
  if (!changelog.value.trim()) errors.push(t('admin.skills.editor.package.changelogRequired'))
  validationErrors.value = errors
  if (errors.length || !selectedFile.value) return

  emit('upload', {
    file: selectedFile.value,
    version: version.value.trim(),
    changelog: changelog.value.trim(),
  })
}

function reset(): void {
  selectedFile.value = null
  version.value = ''
  changelog.value = ''
  validationErrors.value = []
  if (fileInput.value) fileInput.value.value = ''
}

function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  const unitIndex = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  const value = bytes / Math.pow(1024, unitIndex)
  return `${value >= 10 || unitIndex === 0 ? value.toFixed(0) : value.toFixed(1)} ${units[unitIndex]}`
}

defineExpose({ reset })
</script>

<style scoped>
.skill-uploader {
  display: grid;
  min-width: 0;
  gap: 15px;
}

.skill-uploader__heading {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  color: var(--lx-clay-text-muted);
}

.skill-uploader__heading h3 {
  margin: 0;
  color: var(--lx-clay-text);
  font-size: 0.9375rem;
  font-weight: 800;
}

.skill-uploader__heading p {
  max-width: 66ch;
  margin: 4px 0 0;
  color: var(--lx-clay-text-muted);
  font-size: 0.75rem;
  line-height: 1.55;
}

.skill-uploader__dropzone {
  display: flex;
  min-height: 126px;
  align-items: center;
  justify-content: center;
  gap: 14px;
  padding: 20px;
  border: 1px dashed var(--lx-clay-border-strong);
  border-radius: var(--lx-clay-radius-form);
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-surface-soft);
  transition:
    border-color 150ms ease,
    background-color 150ms ease;
}

.skill-uploader__dropzone--active,
.skill-uploader__dropzone--selected {
  border-color: var(--lx-clay-accent);
  background: var(--lx-clay-accent-soft);
}

.skill-uploader__dropzone--disabled {
  opacity: 0.64;
}

.skill-uploader__file-icon {
  display: flex;
  width: 44px;
  height: 44px;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border-radius: var(--lx-clay-radius-control);
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-flat);
}

html.dark .skill-uploader__file-icon {
  color: var(--lx-clay-accent);
}

.skill-uploader__drop-copy {
  min-width: 0;
}

.skill-uploader__drop-copy p {
  max-width: 58ch;
  margin: 0;
  font-size: 0.8125rem;
  font-weight: 700;
  line-height: 1.5;
  overflow-wrap: anywhere;
}

.skill-uploader__choose {
  margin-top: 7px;
  padding: 0;
  border: 0;
  color: var(--lx-clay-accent-deep);
  background: transparent;
  font: inherit;
  font-size: 0.75rem;
  font-weight: 800;
  cursor: pointer;
}

html.dark .skill-uploader__choose {
  color: var(--lx-clay-accent);
}

.skill-uploader__choose:disabled {
  cursor: not-allowed;
}

.skill-uploader__fields {
  display: grid;
  grid-template-columns: minmax(160px, 0.36fr) minmax(0, 1fr);
  gap: 14px;
}

.skill-uploader__fields label {
  display: block;
  margin-bottom: 6px;
}

.skill-uploader__changelog textarea {
  resize: vertical;
}

.skill-uploader__errors {
  display: grid;
  gap: 5px;
  color: var(--lx-clay-danger);
  font-size: 0.75rem;
  font-weight: 700;
}

.skill-uploader__errors p {
  display: flex;
  align-items: flex-start;
  gap: 7px;
  margin: 0;
}

.skill-uploader__footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 14px;
}

.skill-uploader__save-first {
  margin: 0 auto 0 0;
  color: var(--lx-clay-text-muted);
  font-size: 0.75rem;
}

@media (max-width: 640px) {
  .skill-uploader__dropzone {
    align-items: flex-start;
    justify-content: flex-start;
  }

  .skill-uploader__fields {
    grid-template-columns: minmax(0, 1fr);
  }

  .skill-uploader__footer {
    align-items: stretch;
    flex-direction: column;
  }

  .skill-uploader__footer .btn {
    width: 100%;
  }
}

@media (prefers-reduced-motion: reduce) {
  .skill-uploader__dropzone {
    transition-duration: 0.01ms;
  }
}
</style>

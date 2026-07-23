<template>
  <div class="space-y-4">
    <div>
      <label class="input-label">Service Account JSON</label>
      <input
        ref="fileInput"
        type="file"
        accept="application/json,.json"
        class="hidden"
        @change="handleFileChange"
      />
      <div
        :class="[
          'rounded-lg border-2 border-dashed px-4 py-5 transition-colors',
          dragActive
            ? 'border-sky-500 bg-sky-50 dark:border-sky-500 dark:bg-sky-900/20'
            : 'border-gray-300 bg-gray-50 hover:border-sky-400 hover:bg-sky-50/60 dark:border-dark-500 dark:bg-dark-700/40 dark:hover:border-sky-600 dark:hover:bg-sky-900/10'
        ]"
        @dragenter.prevent="dragActive = true"
        @dragover.prevent="dragActive = true"
        @dragleave.prevent="dragActive = false"
        @drop.prevent="handleDrop"
      >
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div class="min-w-0">
            <div class="flex items-center gap-2 text-sm font-medium text-gray-900 dark:text-white">
              <Icon name="upload" size="sm" />
              <span>{{ clientEmail ? t('admin.accounts.vertexSaJsonLoaded') : t('admin.accounts.vertexSaJsonDrop') }}</span>
            </div>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ clientEmail ? t('admin.accounts.vertexSaJsonKeyHidden') : t('admin.accounts.vertexSaJsonDropHint') }}
            </p>
          </div>
          <button type="button" class="btn btn-secondary shrink-0" @click="fileInput?.click()">
            <Icon name="upload" size="sm" />
            {{ t('admin.accounts.vertexSaJsonSelectBtn') }}
          </button>
        </div>
        <div
          v-if="clientEmail"
          class="mt-3 rounded-md border border-sky-200 bg-white px-3 py-2 text-xs text-sky-900 dark:border-sky-800/50 dark:bg-dark-800 dark:text-sky-200"
        >
          <div class="truncate">Project ID: <span class="font-mono">{{ projectId }}</span></div>
          <div class="truncate">Client Email: <span class="font-mono">{{ clientEmail }}</span></div>
        </div>
      </div>
      <p class="input-hint">{{ t('admin.accounts.vertexSaJsonUploadHint') }}</p>
    </div>

    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
      <div>
        <label class="input-label">Project ID</label>
        <input
          :value="projectId"
          type="text"
          class="input font-mono"
          readonly
          :placeholder="t('admin.accounts.vertexProjectIdPlaceholder')"
        />
      </div>
      <div>
        <label class="input-label">Location</label>
        <select v-model="selectedLocation" required class="input font-mono">
          <optgroup
            v-for="group in VERTEX_LOCATION_OPTIONS"
            :key="group.label"
            :label="group.label"
          >
            <option
              v-for="option in group.options"
              :key="option.value"
              :value="option.value"
            >
              {{ option.label }}
            </option>
          </optgroup>
        </select>
        <p class="input-hint">{{ t('admin.accounts.vertexLocationHint') }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { VERTEX_LOCATION_OPTIONS } from '@/constants/account'

const props = defineProps<{
  projectId: string
  clientEmail: string
  location: string
}>()
const emit = defineEmits<{
  'update:location': [value: string]
  'file-selected': [file: File]
}>()

const { t } = useI18n()
const fileInput = ref<HTMLInputElement | null>(null)
const dragActive = ref(false)
const selectedLocation = computed({
  get: () => props.location,
  set: (value) => emit('update:location', value),
})

const handleFileChange = (event: Event) => {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (file) emit('file-selected', file)
  input.value = ''
}

const handleDrop = (event: DragEvent) => {
  dragActive.value = false
  const file = event.dataTransfer?.files?.[0]
  if (file) emit('file-selected', file)
}
</script>

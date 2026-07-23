<template>
  <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
    <div class="mb-3 flex items-center justify-between">
      <div>
        <label class="input-label mb-0">{{ t('admin.accounts.grokCustomBaseUrl.title') }}</label>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.grokCustomBaseUrl.hint') }}
        </p>
      </div>
      <button
        type="button"
        data-testid="grok-custom-base-url-toggle"
        @click="baseUrlEnabled = !baseUrlEnabled"
        :class="[
          'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
          baseUrlEnabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
        ]"
      >
        <span
          :class="[
            'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
            baseUrlEnabled ? 'translate-x-5' : 'translate-x-0'
          ]"
        />
      </button>
    </div>
    <div v-if="baseUrlEnabled">
      <input
        v-model="baseUrl"
        type="text"
        class="input"
        data-testid="grok-custom-base-url-input"
        :placeholder="t('admin.accounts.grokCustomBaseUrl.placeholder')"
      />
    </div>
  </div>

  <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
    <div class="mb-3 flex items-center justify-between">
      <div>
        <label class="input-label mb-0">{{ t('admin.accounts.headerOverride.title') }}</label>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.headerOverride.hint') }}
        </p>
      </div>
      <button
        type="button"
        @click="headerEnabled = !headerEnabled"
        :class="[
          'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
          headerEnabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
        ]"
      >
        <span
          :class="[
            'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
            headerEnabled ? 'translate-x-5' : 'translate-x-0'
          ]"
        />
      </button>
    </div>

    <div v-if="headerEnabled" class="space-y-3">
      <div class="rounded-lg bg-blue-50 p-3 dark:bg-blue-900/20">
        <p class="text-xs text-blue-700 dark:text-blue-400">
          <Icon name="exclamationCircle" size="sm" class="mr-1 inline" :stroke-width="2" />
          {{ t('admin.accounts.headerOverride.info') }}
        </p>
      </div>

      <HeaderOverrideEditor :rows="rows" @update:rows="rows = $event" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import HeaderOverrideEditor from '@/components/account/HeaderOverrideEditor.vue'
import Icon from '@/components/icons/Icon.vue'
import type { HeaderOverrideRow } from '@/components/account/credentialsBuilder'

const props = defineProps<{
  baseUrlEnabled: boolean
  baseUrl: string
  headerEnabled: boolean
  rows: HeaderOverrideRow[]
}>()
const emit = defineEmits<{
  'update:baseUrlEnabled': [value: boolean]
  'update:baseUrl': [value: string]
  'update:headerEnabled': [value: boolean]
  'update:rows': [value: HeaderOverrideRow[]]
}>()

const { t } = useI18n()
const baseUrlEnabled = computed({
  get: () => props.baseUrlEnabled,
  set: (value) => emit('update:baseUrlEnabled', value),
})
const baseUrl = computed({
  get: () => props.baseUrl,
  set: (value) => emit('update:baseUrl', value),
})
const headerEnabled = computed({
  get: () => props.headerEnabled,
  set: (value) => emit('update:headerEnabled', value),
})
const rows = computed({
  get: () => props.rows,
  set: (value) => emit('update:rows', value),
})
</script>

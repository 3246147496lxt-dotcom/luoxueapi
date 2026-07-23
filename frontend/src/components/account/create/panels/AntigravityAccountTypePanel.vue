<template>
  <div>
    <label class="input-label">{{ t('admin.accounts.accountType') }}</label>
    <div class="mt-2 grid grid-cols-2 gap-3">
      <button
        type="button"
        @click="accountKind = 'oauth'"
        :class="[
          'flex items-center gap-3 rounded-lg border-2 p-3 text-left transition-all',
          accountKind === 'oauth'
            ? 'border-purple-500 bg-purple-50 dark:bg-purple-900/20'
            : 'border-gray-200 hover:border-purple-300 dark:border-dark-600 dark:hover:border-purple-700'
        ]"
      >
        <div
          :class="[
            'flex h-8 w-8 shrink-0 items-center justify-center rounded-lg',
            accountKind === 'oauth'
              ? 'bg-purple-500 text-white'
              : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
          ]"
        >
          <Icon name="key" size="sm" />
        </div>
        <div>
          <span class="block text-sm font-medium text-gray-900 dark:text-white">OAuth</span>
          <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.types.antigravityOauth') }}</span>
        </div>
      </button>

      <button
        type="button"
        @click="accountKind = 'upstream'"
        :class="[
          'flex items-center gap-3 rounded-lg border-2 p-3 text-left transition-all',
          accountKind === 'upstream'
            ? 'border-purple-500 bg-purple-50 dark:bg-purple-900/20'
            : 'border-gray-200 hover:border-purple-300 dark:border-dark-600 dark:hover:border-purple-700'
        ]"
      >
        <div
          :class="[
            'flex h-8 w-8 shrink-0 items-center justify-center rounded-lg',
            accountKind === 'upstream'
              ? 'bg-purple-500 text-white'
              : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
          ]"
        >
          <Icon name="cloud" size="sm" />
        </div>
        <div>
          <span class="block text-sm font-medium text-gray-900 dark:text-white">API Key</span>
          <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.types.antigravityApikey') }}</span>
        </div>
      </button>
    </div>
  </div>

  <div v-if="accountKind === 'oauth'">
    <label class="input-label">{{ t('admin.accounts.antigravityProjectIdLabel') }}</label>
    <input
      v-model="projectId"
      data-testid="antigravity-project-id-input"
      type="text"
      class="input font-mono"
      :placeholder="t('admin.accounts.antigravityProjectIdPlaceholder')"
    />
    <p class="input-hint">{{ t('admin.accounts.antigravityProjectIdHint') }}</p>
  </div>

  <div v-if="accountKind === 'upstream'" class="space-y-4">
    <div>
      <label class="input-label">{{ t('admin.accounts.upstream.baseUrl') }}</label>
      <input
        v-model="baseUrl"
        type="text"
        required
        class="input"
        placeholder="https://cloudcode-pa.googleapis.com"
      />
      <p class="input-hint">{{ t('admin.accounts.upstream.baseUrlHint') }}</p>
    </div>
    <div>
      <label class="input-label">{{ t('admin.accounts.upstream.apiKey') }}</label>
      <input
        v-model="apiKey"
        type="password"
        required
        class="input font-mono"
        placeholder="sk-..."
      />
      <p class="input-hint">{{ t('admin.accounts.upstream.apiKeyHint') }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { AntigravityAccountKind } from '../formDraft'

const props = defineProps<{
  accountKind: AntigravityAccountKind
  projectId: string
  baseUrl: string
  apiKey: string
}>()
const emit = defineEmits<{
  'update:accountKind': [value: AntigravityAccountKind]
  'update:projectId': [value: string]
  'update:baseUrl': [value: string]
  'update:apiKey': [value: string]
}>()

const { t } = useI18n()
const accountKind = computed({
  get: () => props.accountKind,
  set: (value) => emit('update:accountKind', value),
})
const projectId = computed({
  get: () => props.projectId,
  set: (value) => emit('update:projectId', value),
})
const baseUrl = computed({
  get: () => props.baseUrl,
  set: (value) => emit('update:baseUrl', value),
})
const apiKey = computed({
  get: () => props.apiKey,
  set: (value) => emit('update:apiKey', value),
})
</script>

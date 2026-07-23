<template>
  <div>
    <label class="input-label">{{ t('admin.accounts.accountType') }}</label>
    <div class="mt-2 grid grid-cols-2 gap-3 sm:grid-cols-4" data-tour="account-form-type">
      <button
        type="button"
        @click="selectCategory('oauth-based')"
        :class="[
          'flex items-center gap-3 rounded-lg border-2 p-3 text-left transition-all',
          selected === 'oauth-based'
            ? 'border-orange-500 bg-orange-50 dark:bg-orange-900/20'
            : 'border-gray-200 hover:border-orange-300 dark:border-dark-600 dark:hover:border-orange-700'
        ]"
      >
        <div
          :class="[
            'flex h-8 w-8 shrink-0 items-center justify-center rounded-lg',
            selected === 'oauth-based'
              ? 'bg-orange-500 text-white'
              : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
          ]"
        >
          <Icon name="sparkles" size="sm" />
        </div>
        <div>
          <span class="block text-sm font-medium text-gray-900 dark:text-white">{{
            t('admin.accounts.claudeCode')
          }}</span>
          <span class="text-xs text-gray-500 dark:text-gray-400">{{
            t('admin.accounts.oauthSetupToken')
          }}</span>
        </div>
      </button>

      <button
        type="button"
        @click="selectCategory('apikey')"
        :class="[
          'flex items-center gap-3 rounded-lg border-2 p-3 text-left transition-all',
          selected === 'apikey'
            ? 'border-purple-500 bg-purple-50 dark:bg-purple-900/20'
            : 'border-gray-200 hover:border-purple-300 dark:border-dark-600 dark:hover:border-purple-700'
        ]"
      >
        <div
          :class="[
            'flex h-8 w-8 shrink-0 items-center justify-center rounded-lg',
            selected === 'apikey'
              ? 'bg-purple-500 text-white'
              : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
          ]"
        >
          <Icon name="key" size="sm" />
        </div>
        <div>
          <span class="block text-sm font-medium text-gray-900 dark:text-white">{{
            t('admin.accounts.claudeConsole')
          }}</span>
          <span class="text-xs text-gray-500 dark:text-gray-400">{{
            t('admin.accounts.apiKey')
          }}</span>
        </div>
      </button>

      <button
        type="button"
        @click="selectCategory('bedrock')"
        :class="[
          'flex items-center gap-3 rounded-lg border-2 p-3 text-left transition-all',
          selected === 'bedrock'
            ? 'border-amber-500 bg-amber-50 dark:bg-amber-900/20'
            : 'border-gray-200 hover:border-amber-300 dark:border-dark-600 dark:hover:border-amber-700'
        ]"
      >
        <div
          :class="[
            'flex h-8 w-8 shrink-0 items-center justify-center rounded-lg',
            selected === 'bedrock'
              ? 'bg-amber-500 text-white'
              : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
          ]"
        >
          <Icon name="cloud" size="sm" />
        </div>
        <div>
          <span class="block text-sm font-medium text-gray-900 dark:text-white">{{
            t('admin.accounts.bedrockLabel')
          }}</span>
          <span class="text-xs text-gray-500 dark:text-gray-400">{{
            t('admin.accounts.bedrockDesc')
          }}</span>
        </div>
      </button>

      <button
        type="button"
        @click="selectCategory('service_account')"
        :class="[
          'flex items-center gap-3 rounded-lg border-2 p-3 text-left transition-all',
          selected === 'service_account'
            ? 'border-sky-500 bg-sky-50 dark:bg-sky-900/20'
            : 'border-gray-200 hover:border-sky-300 dark:border-dark-600 dark:hover:border-sky-700'
        ]"
      >
        <div
          :class="[
            'flex h-8 w-8 shrink-0 items-center justify-center rounded-lg',
            selected === 'service_account'
              ? 'bg-sky-500 text-white'
              : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
          ]"
        >
          <Icon name="cloud" size="sm" />
        </div>
        <div>
          <span class="block text-sm font-medium text-gray-900 dark:text-white">Vertex</span>
          <span class="text-xs text-gray-500 dark:text-gray-400">Service Account</span>
        </div>
      </button>
    </div>

    <div
      v-if="selected === 'service_account'"
      class="mt-3 rounded-lg border border-sky-200 bg-sky-50 px-3 py-2 text-xs text-sky-800 dark:border-sky-800/40 dark:bg-sky-900/20 dark:text-sky-200"
    >
      <p>{{ t('admin.accounts.vertexAnthropicHint') }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { CreateAccountCategory } from '../formDraft'

const props = defineProps<{ modelValue: CreateAccountCategory }>()
const emit = defineEmits<{
  'update:modelValue': [value: CreateAccountCategory]
}>()

const { t } = useI18n()
const selected = computed(() => props.modelValue)
const selectCategory = (value: CreateAccountCategory) => emit('update:modelValue', value)
</script>

<template>
  <div>
    <label class="input-label">{{ t('admin.accounts.accountType') }}</label>
    <div class="mt-2 grid grid-cols-2 gap-3" data-tour="account-form-type">
      <button
        type="button"
        @click="selectCategory('oauth-based')"
        :class="[
          'flex items-center gap-3 rounded-lg border-2 p-3 text-left transition-all',
          selected === 'oauth-based'
            ? 'border-green-500 bg-green-50 dark:bg-green-900/20'
            : 'border-gray-200 hover:border-green-300 dark:border-dark-600 dark:hover:border-green-700'
        ]"
      >
        <div
          :class="[
            'flex h-8 w-8 shrink-0 items-center justify-center rounded-lg',
            selected === 'oauth-based'
              ? 'bg-green-500 text-white'
              : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
          ]"
        >
          <Icon name="key" size="sm" />
        </div>
        <div>
          <span class="block text-sm font-medium text-gray-900 dark:text-white">OAuth</span>
          <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.types.chatgptOauth') }}</span>
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
          <span class="block text-sm font-medium text-gray-900 dark:text-white">API Key</span>
          <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.types.responsesApi') }}</span>
        </div>
      </button>
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

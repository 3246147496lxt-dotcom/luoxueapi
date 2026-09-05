<template>
  <div>
    <label class="input-label">{{ t('admin.accounts.platform') }}</label>
    <div
      class="mt-2 flex flex-wrap rounded-lg bg-gray-100 p-1 dark:bg-dark-700"
      data-tour="account-form-platform"
    >
      <button
        v-for="option in platformOptions"
        :key="option.platform"
        type="button"
        :data-testid="`account-platform-${option.platform}`"
        @click="emit('select', option.platform)"
        :class="[
          'flex flex-1 items-center justify-center gap-2 rounded-md px-4 py-2.5 text-sm font-medium transition-all',
          props.platform === option.platform
            ? option.activeClass
            : 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-200'
        ]"
      >
        <PlatformIcon :platform="option.platform" size="sm" />
        {{ option.label }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { AccountPlatform } from '@/types'
import PlatformIcon from '@/components/common/PlatformIcon.vue'

const props = defineProps<{ platform: AccountPlatform }>()
const emit = defineEmits<{ select: [platform: AccountPlatform] }>()
const { t } = useI18n()

const platformOptions: ReadonlyArray<{
  platform: AccountPlatform
  label: string
  activeClass: string
}> = [
  {
    platform: 'anthropic',
    label: 'Anthropic',
    activeClass: 'bg-white text-orange-600 shadow-sm dark:bg-dark-600 dark:text-orange-400',
  },
  {
    platform: 'openai',
    label: 'OpenAI',
    activeClass: 'bg-white text-green-600 shadow-sm dark:bg-dark-600 dark:text-green-400',
  },
  {
    platform: 'gemini',
    label: 'Gemini',
    activeClass: 'bg-white text-blue-600 shadow-sm dark:bg-dark-600 dark:text-blue-400',
  },
  {
    platform: 'antigravity',
    label: 'Antigravity',
    activeClass: 'bg-white text-purple-600 shadow-sm dark:bg-dark-600 dark:text-purple-400',
  },
  {
    platform: 'grok',
    label: 'Grok',
    activeClass: 'bg-white text-zinc-900 shadow-sm dark:bg-dark-600 dark:text-zinc-100',
  },
  {
    platform: 'deepseek',
    label: 'DeepSeek',
    activeClass: 'bg-white text-sky-600 shadow-sm dark:bg-dark-600 dark:text-sky-400',
  },
  {
    platform: 'zhipu',
    label: 'GLM',
    activeClass: 'bg-white text-indigo-600 shadow-sm dark:bg-dark-600 dark:text-indigo-400',
  },
]
</script>

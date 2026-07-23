<template>
  <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
    <label class="input-label">{{ t('admin.accounts.modelRestriction') }}</label>

    <div>
      <div class="mb-3 rounded-lg bg-purple-50 p-3 dark:bg-purple-900/20">
        <p class="text-xs text-purple-700 dark:text-purple-400">
          {{ t('admin.accounts.mapRequestModels') }}
        </p>
      </div>

      <div v-if="mappings.length > 0" class="mb-3 space-y-2">
        <div
          v-for="(mapping, index) in mappings"
          :key="getMappingKey(mapping)"
          class="space-y-1"
        >
          <div class="flex items-center gap-2">
            <input
              :value="mapping.from"
              type="text"
              :class="[
                'input flex-1',
                !isValidWildcardPattern(mapping.from) ? 'border-red-500 dark:border-red-500' : ''
              ]"
              :placeholder="t('admin.accounts.requestModel')"
              @input="updateMapping(index, 'from', $event)"
            />
            <svg class="h-4 w-4 flex-shrink-0 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3" />
            </svg>
            <input
              :value="mapping.to"
              type="text"
              :class="[
                'input flex-1',
                mapping.to.includes('*') ? 'border-red-500 dark:border-red-500' : ''
              ]"
              :placeholder="t('admin.accounts.actualModel')"
              @input="updateMapping(index, 'to', $event)"
            />
            <button
              type="button"
              @click="emit('remove', index)"
              class="rounded-lg p-2 text-red-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20"
            >
              <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                />
              </svg>
            </button>
          </div>
          <p v-if="!isValidWildcardPattern(mapping.from)" class="text-xs text-red-500">
            {{ t('admin.accounts.wildcardOnlyAtEnd') }}
          </p>
          <p v-if="mapping.to.includes('*')" class="text-xs text-red-500">
            {{ t('admin.accounts.targetNoWildcard') }}
          </p>
        </div>
      </div>

      <button
        type="button"
        @click="emit('add')"
        class="mb-3 w-full rounded-lg border-2 border-dashed border-gray-300 px-4 py-2 text-gray-600 transition-colors hover:border-gray-400 hover:text-gray-700 dark:border-dark-500 dark:text-gray-400 dark:hover:border-dark-400 dark:hover:text-gray-300"
      >
        <svg class="mr-1 inline h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        {{ t('admin.accounts.addMapping') }}
      </button>

      <div class="flex flex-wrap gap-2">
        <button
          v-for="preset in presets"
          :key="preset.label"
          type="button"
          @click="emit('add-preset', preset.from, preset.to)"
          :class="['rounded-lg px-3 py-1 text-xs transition-colors', preset.color]"
        >
          + {{ preset.label }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { isValidWildcardPattern } from '@/composables/useModelWhitelist'
import { createStableObjectKeyResolver } from '@/utils/stableObjectKey'
import type { AccountModelMappingDraft } from '../formDraft'

defineProps<{
  mappings: AccountModelMappingDraft[]
  presets: Array<{ label: string; from: string; to: string; color: string }>
}>()
const emit = defineEmits<{
  add: []
  remove: [index: number]
  'update-mapping': [index: number, field: keyof AccountModelMappingDraft, value: string]
  'add-preset': [from: string, to: string]
}>()

const { t } = useI18n()
const getMappingKey = createStableObjectKeyResolver<AccountModelMappingDraft>(
  'create-antigravity-model-mapping',
)
const updateMapping = (
  index: number,
  field: keyof AccountModelMappingDraft,
  event: Event,
) => {
  emit('update-mapping', index, field, (event.target as HTMLInputElement).value)
}
</script>

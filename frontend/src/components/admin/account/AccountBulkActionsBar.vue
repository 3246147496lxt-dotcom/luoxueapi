<template>
  <div class="mb-4 flex flex-col gap-3 rounded-lg bg-primary-50 p-3 sm:flex-row sm:items-center sm:justify-between dark:bg-primary-900/20">
    <div class="flex flex-wrap items-center gap-2">
      <span class="text-sm font-medium text-primary-900 dark:text-primary-100">
        {{ t('admin.accounts.bulkActions.selected', { count: selectedIds.length }) }}
      </span>
      <button
        @click="$emit('select-page')"
        class="min-h-11 text-xs font-medium text-primary-700 hover:text-primary-800 dark:text-primary-300 dark:hover:text-primary-200"
      >
        {{ t('admin.accounts.bulkActions.selectCurrentPage') }}
      </button>
      <span class="text-gray-300 dark:text-primary-800">•</span>
      <button
        @click="$emit('clear')"
        class="min-h-11 text-xs font-medium text-primary-700 hover:text-primary-800 dark:text-primary-300 dark:hover:text-primary-200"
      >
        {{ t('admin.accounts.bulkActions.clear') }}
      </button>
    </div>
    <div class="flex flex-wrap gap-2">
      <button @click="$emit('delete')" class="btn btn-danger btn-sm min-h-11">{{ t('admin.accounts.bulkActions.delete') }}</button>
      <button @click="$emit('reset-status')" class="btn btn-secondary btn-sm min-h-11">{{ t('admin.accounts.bulkActions.resetStatus') }}</button>
      <button @click="$emit('refresh-token')" class="btn btn-secondary btn-sm min-h-11">{{ t('admin.accounts.bulkActions.refreshToken') }}</button>
      <button @click="$emit('probe-upstream-billing')" class="btn btn-secondary btn-sm min-h-11">{{ t('admin.accounts.bulkActions.probeUpstreamBilling') }}</button>
      <button @click="$emit('toggle-schedulable', true)" class="btn btn-success btn-sm min-h-11">{{ t('admin.accounts.bulkActions.enableScheduling') }}</button>
      <button @click="$emit('toggle-schedulable', false)" class="btn btn-warning btn-sm min-h-11">{{ t('admin.accounts.bulkActions.disableScheduling') }}</button>
      <button @click="$emit('edit-selected')" class="btn btn-secondary btn-sm min-h-11">{{ t('admin.accounts.bulkActions.edit') }}</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{ selectedIds: number[] }>()
defineEmits([
  'delete',
  'edit-selected',
  'clear',
  'select-page',
  'toggle-schedulable',
  'reset-status',
  'refresh-token',
  'probe-upstream-billing'
])

const { t } = useI18n()
</script>

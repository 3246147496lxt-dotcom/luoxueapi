<template>
  <div
    class="grid w-full min-w-0 grid-cols-[minmax(0,1fr)_auto] items-center gap-3 sm:grid-cols-[minmax(0,1fr)_10rem_auto] lg:flex lg:flex-wrap"
  >
    <SearchInput
      :model-value="searchQuery"
      :placeholder="t('admin.accounts.searchAccounts')"
      class="col-span-2 min-w-0 sm:col-span-1 lg:order-1 lg:w-64"
      data-testid="account-search-filter"
      @update:model-value="$emit('update:searchQuery', $event)"
      @search="$emit('change')"
    />

    <Select
      :model-value="filters.status"
      class="w-full min-w-0 lg:order-4 lg:w-40"
      data-testid="account-status-filter"
      :options="sOpts"
      @update:model-value="updateStatus"
      @change="$emit('change')"
    />

    <button
      type="button"
      class="btn btn-secondary min-h-11 min-w-11 justify-center px-3 lg:hidden"
      data-testid="account-filters-toggle"
      :aria-expanded="secondaryFiltersOpen"
      aria-controls="account-secondary-filters"
      @click="secondaryFiltersOpen = !secondaryFiltersOpen"
    >
      <Icon name="filter" size="sm" />
      <span>{{ t('common.filter') }}</span>
      <span
        v-if="activeSecondaryFilterCount > 0"
        class="inline-flex min-w-5 items-center justify-center rounded-full bg-primary-100 px-1.5 text-xs font-semibold text-primary-700 dark:bg-primary-900/40 dark:text-primary-200"
      >
        {{ activeSecondaryFilterCount }}
      </span>
    </button>

    <div
      id="account-secondary-filters"
      data-testid="account-secondary-filters"
      :class="[
        'col-span-2 min-w-0 gap-3 sm:col-span-3 sm:grid-cols-2',
        secondaryFiltersOpen ? 'grid' : 'hidden',
        'lg:contents'
      ]"
    >
      <Select
        :model-value="filters.platform"
        class="w-full min-w-0 lg:order-2 lg:w-40"
        data-testid="account-platform-filter"
        :options="pOpts"
        @update:model-value="updatePlatform"
        @change="$emit('change')"
      />
      <Select
        :model-value="filters.type"
        class="w-full min-w-0 lg:order-3 lg:w-40"
        data-testid="account-type-filter"
        :options="tOpts"
        @update:model-value="updateType"
        @change="$emit('change')"
      />
      <Select
        :model-value="filters.privacy_mode"
        class="w-full min-w-0 lg:order-5 lg:w-40"
        data-testid="account-privacy-filter"
        :options="privacyOpts"
        @update:model-value="updatePrivacyMode"
        @change="$emit('change')"
      />
      <Select
        :model-value="filters.group"
        class="w-full min-w-0 lg:order-6 lg:w-40"
        data-testid="account-group-filter"
        :options="gOpts"
        @update:model-value="updateGroup"
        @change="$emit('change')"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Icon from '@/components/icons/Icon.vue'
import type { AdminGroup } from '@/types'

const props = defineProps<{ searchQuery: string; filters: Record<string, any>; groups?: AdminGroup[] }>()
const emit = defineEmits(['update:searchQuery', 'update:filters', 'change'])
const { t } = useI18n()

const secondaryFiltersOpen = ref(false)
const activeSecondaryFilterCount = computed(() =>
  ['platform', 'type', 'privacy_mode', 'group'].filter(key => Boolean(props.filters[key])).length
)

const updatePlatform = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, platform: value }) }
const updateType = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, type: value }) }
const updateStatus = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, status: value }) }
const updatePrivacyMode = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, privacy_mode: value }) }
const updateGroup = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, group: value }) }
const pOpts = computed(() => [{ value: '', label: t('admin.accounts.allPlatforms') }, { value: 'anthropic', label: 'Anthropic' }, { value: 'openai', label: 'OpenAI' }, { value: 'gemini', label: 'Gemini' }, { value: 'antigravity', label: 'Antigravity' }, { value: 'grok', label: 'Grok' }])
const tOpts = computed(() => [{ value: '', label: t('admin.accounts.allTypes') }, { value: 'oauth', label: t('admin.accounts.oauthType') }, { value: 'setup-token', label: t('admin.accounts.setupToken') }, { value: 'apikey', label: t('admin.accounts.apiKey') }, { value: 'bedrock', label: 'AWS Bedrock' }])
const sOpts = computed(() => [{ value: '', label: t('admin.accounts.allStatus') }, { value: 'active', label: t('admin.accounts.status.active') }, { value: 'inactive', label: t('admin.accounts.status.inactive') }, { value: 'error', label: t('admin.accounts.status.error') }, { value: 'rate_limited', label: t('admin.accounts.status.rateLimited') }, { value: 'temp_unschedulable', label: t('admin.accounts.status.tempUnschedulable') }, { value: 'unschedulable', label: t('admin.accounts.status.unschedulable') }, { value: 'overloaded', label: t('admin.accounts.status.overloaded') }, { value: 'expired', label: t('admin.proxies.expired') }, { value: 'quota_exhausted', label: t('admin.accounts.status.quotaExceeded') }])
const privacyOpts = computed(() => [
  { value: '', label: t('admin.accounts.allPrivacyModes') },
  { value: '__unset__', label: t('admin.accounts.privacyUnset') },
  { value: 'training_off', label: 'Privacy' },
  { value: 'training_set_cf_blocked', label: 'CF' },
  { value: 'training_set_failed', label: 'Fail' }
])
const gOpts = computed(() => [
  { value: '', label: t('admin.accounts.allGroups') },
  { value: 'ungrouped', label: t('admin.accounts.ungroupedGroup') },
  ...(props.groups || []).map(g => ({ value: String(g.id), label: g.name }))
])
</script>

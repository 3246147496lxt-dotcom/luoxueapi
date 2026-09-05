<template>
  <div ref="rootRef" class="account-filter-stack">
    <div class="account-filter-row">
      <SearchInput
        :model-value="searchQuery"
        :placeholder="t('admin.accounts.searchAccounts')"
        class="account-filter-search"
        data-testid="account-search-filter"
        @update:model-value="$emit('update:searchQuery', $event)"
        @search="$emit('change')"
      />

      <button
        type="button"
        class="btn btn-secondary account-filter-toggle"
        data-testid="account-filters-toggle"
        :aria-expanded="secondaryFiltersOpen"
        aria-controls="account-secondary-filters"
        @click="secondaryFiltersOpen = !secondaryFiltersOpen"
      >
        <Icon name="filter" size="sm" />
        <span>{{ t('admin.accounts.advancedFilters') }}</span>
        <span
          v-if="activeFilterChips.length"
          class="account-filter-count"
        >
          {{ activeFilterChips.length }}
        </span>
      </button>
    </div>

    <div
      v-if="activeFilterChips.length"
      class="account-applied-filters"
      data-testid="account-applied-filters"
    >
      <span class="account-applied-filters__label">
        {{ t('admin.accounts.appliedFilters') }}
      </span>
      <button
        v-for="chip in activeFilterChips"
        :key="chip.key"
        type="button"
        :class="['account-filter-chip', filterChipClass(chip)]"
        :aria-label="t('admin.accounts.removeFilter', { filter: chip.label })"
        @click="clearFilter(chip.key)"
      >
        <span>{{ chip.label }}</span>
        <Icon name="x" size="xs" aria-hidden="true" />
      </button>
      <button
        type="button"
        class="account-applied-filters__clear"
        @click="clearAllFilters"
      >
        {{ t('admin.accounts.clearAllFilters') }}
      </button>
    </div>

    <div
      v-if="secondaryFiltersOpen"
      id="account-secondary-filters"
      class="account-filter-popover"
      data-testid="account-secondary-filters"
      @keydown.esc.stop.prevent="secondaryFiltersOpen = false"
    >
      <div class="account-filter-popover__header">
        <span>{{ t('admin.accounts.advancedFilters') }}</span>
        <button
          type="button"
          class="account-filter-popover__close"
          :aria-label="t('common.close')"
          @click="secondaryFiltersOpen = false"
        >
          <Icon name="x" size="sm" aria-hidden="true" />
        </button>
      </div>
      <div class="account-filter-popover__grid">
        <Select
          :model-value="filters.platform"
          data-testid="account-platform-filter"
          :options="pOpts"
          @update:model-value="updatePlatform"
          @change="$emit('change')"
        />
        <Select
          :model-value="filters.type"
          data-testid="account-type-filter"
          :options="tOpts"
          @update:model-value="updateType"
          @change="$emit('change')"
        />
        <Select
          :model-value="filters.status"
          data-testid="account-status-filter"
          :options="sOpts"
          @update:model-value="updateStatus"
          @change="$emit('change')"
        />
        <Select
          :model-value="filters.privacy_mode"
          data-testid="account-privacy-filter"
          :options="privacyOpts"
          @update:model-value="updatePrivacyMode"
          @change="$emit('change')"
        />
        <Select
          :model-value="filters.group"
          class="sm:col-span-2"
          data-testid="account-group-filter"
          :options="gOpts"
          @update:model-value="updateGroup"
          @change="$emit('change')"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Icon from '@/components/icons/Icon.vue'
import type { AdminGroup } from '@/types'

type FilterKey = 'platform' | 'type' | 'status' | 'privacy_mode' | 'group'
type FilterValue = string | number | boolean | null
type FilterOption = { value: FilterValue; label: string }
type FilterChip = { key: FilterKey; label: string; value: string }

const props = defineProps<{
  searchQuery: string
  filters: Record<string, any>
  groups?: AdminGroup[]
}>()
const emit = defineEmits(['update:searchQuery', 'update:filters', 'change'])
const { t } = useI18n()

const rootRef = ref<HTMLElement | null>(null)
const secondaryFiltersOpen = ref(false)

const pOpts = computed<FilterOption[]>(() => [
  { value: '', label: t('admin.accounts.allPlatforms') },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'openai', label: 'OpenAI' },
  { value: 'gemini', label: 'Gemini' },
  { value: 'antigravity', label: 'Antigravity' },
  { value: 'grok', label: 'Grok' },
  { value: 'zhipu', label: 'Zhipu GLM' },
  { value: 'deepseek', label: 'DeepSeek' }
])
const tOpts = computed<FilterOption[]>(() => [
  { value: '', label: t('admin.accounts.allTypes') },
  { value: 'oauth', label: t('admin.accounts.oauthType') },
  { value: 'setup-token', label: t('admin.accounts.setupToken') },
  { value: 'apikey', label: t('admin.accounts.apiKey') },
  { value: 'bedrock', label: 'AWS Bedrock' }
])
const sOpts = computed<FilterOption[]>(() => [
  { value: '', label: t('admin.accounts.allStatus') },
  { value: 'active', label: t('admin.accounts.status.active') },
  { value: 'inactive', label: t('admin.accounts.status.inactive') },
  { value: 'error', label: t('admin.accounts.status.error') },
  { value: 'rate_limited', label: t('admin.accounts.status.rateLimited') },
  { value: 'temp_unschedulable', label: t('admin.accounts.status.tempUnschedulable') },
  { value: 'unschedulable', label: t('admin.accounts.status.unschedulable') },
  { value: 'overloaded', label: t('admin.accounts.status.overloaded') },
  { value: 'expired', label: t('admin.proxies.expired') },
  { value: 'quota_exhausted', label: t('admin.accounts.status.quotaExceeded') }
])
const privacyOpts = computed<FilterOption[]>(() => [
  { value: '', label: t('admin.accounts.allPrivacyModes') },
  { value: '__unset__', label: t('admin.accounts.privacyUnset') },
  { value: 'training_off', label: 'Privacy' },
  { value: 'training_set_cf_blocked', label: 'CF' },
  { value: 'training_set_failed', label: 'Fail' }
])
const gOpts = computed<FilterOption[]>(() => [
  { value: '', label: t('admin.accounts.allGroups') },
  { value: 'ungrouped', label: t('admin.accounts.ungroupedGroup') },
  ...(props.groups || []).map(group => ({ value: String(group.id), label: group.name }))
])

const selectedLabel = (options: FilterOption[], value: unknown): string => {
  const normalizedValue = String(value ?? '')
  return options.find(option => String(option.value ?? '') === normalizedValue)?.label || normalizedValue
}

const activeFilterChips = computed<FilterChip[]>(() => {
  const definitions: Array<[FilterKey, FilterOption[]]> = [
    ['platform', pOpts.value],
    ['type', tOpts.value],
    ['status', sOpts.value],
    ['privacy_mode', privacyOpts.value],
    ['group', gOpts.value]
  ]

  return definitions.flatMap(([key, options]) => {
    const value = String(props.filters[key] ?? '')
    if (!value) return []
    return [{ key, value, label: selectedLabel(options, value) }]
  })
})

const updateFilter = (key: FilterKey, value: FilterValue) => {
  emit('update:filters', { ...props.filters, [key]: value ?? '' })
}
const updatePlatform = (value: FilterValue) => updateFilter('platform', value)
const updateType = (value: FilterValue) => updateFilter('type', value)
const updateStatus = (value: FilterValue) => updateFilter('status', value)
const updatePrivacyMode = (value: FilterValue) => updateFilter('privacy_mode', value)
const updateGroup = (value: FilterValue) => updateFilter('group', value)

const clearFilter = (key: FilterKey) => {
  updateFilter(key, '')
  emit('change')
}

const clearAllFilters = () => {
  emit('update:filters', {
    ...props.filters,
    platform: '',
    type: '',
    status: '',
    privacy_mode: '',
    group: ''
  })
  emit('change')
}

const filterChipClass = (chip: FilterChip): string => {
  if (chip.key !== 'platform') return 'account-filter-chip--neutral'
  if (chip.value === 'openai') return 'account-filter-chip--openai'
  if (chip.value === 'anthropic') return 'account-filter-chip--anthropic'
  if (chip.value === 'gemini') return 'account-filter-chip--gemini'
  if (chip.value === 'antigravity') return 'account-filter-chip--antigravity'
  if (chip.value === 'zhipu') return 'account-filter-chip--zhipu'
  if (chip.value === 'deepseek') return 'account-filter-chip--deepseek'
  return 'account-filter-chip--grok'
}

const handleDocumentClick = (event: MouseEvent) => {
  if (!secondaryFiltersOpen.value) return
  const target = event.target
  if (target instanceof Node && !rootRef.value?.contains(target)) {
    secondaryFiltersOpen.value = false
  }
}

const handleEscape = (event: KeyboardEvent) => {
  if (event.key === 'Escape') secondaryFiltersOpen.value = false
}

onMounted(() => {
  document.addEventListener('click', handleDocumentClick)
  document.addEventListener('keydown', handleEscape)
})

onUnmounted(() => {
  document.removeEventListener('click', handleDocumentClick)
  document.removeEventListener('keydown', handleEscape)
})
</script>

<style scoped>
.account-filter-stack {
  position: relative;
  display: flex;
  min-width: 0;
  flex: 1 1 auto;
  flex-direction: column;
  gap: 10px;
}

.account-filter-row {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 12px;
}

.account-filter-search {
  min-width: 0;
  flex: 1 1 auto;
}

.account-filter-search :deep(.input) {
  height: 44px;
  border-radius: 12px;
}

.account-filter-toggle {
  min-width: max-content;
  min-height: 44px;
  gap: 8px;
  padding-inline: 14px;
  border-radius: 12px;
}

.account-filter-count {
  display: inline-flex;
  min-width: 18px;
  height: 18px;
  align-items: center;
  justify-content: center;
  padding-inline: 5px;
  border-radius: 999px;
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
  font-size: 0.6875rem;
  font-weight: 700;
}

.account-applied-filters {
  display: flex;
  min-height: 22px;
  flex-wrap: wrap;
  align-items: center;
  gap: 7px;
}

.account-applied-filters__label {
  color: var(--lx-clay-text-muted);
  font-size: 0.6875rem;
  font-weight: 700;
}

.account-filter-chip {
  display: inline-flex;
  min-height: 26px;
  align-items: center;
  gap: 5px;
  padding: 3px 8px;
  border: 1px solid transparent;
  border-radius: 8px;
  font-size: 0.75rem;
  font-weight: 600;
}

.account-filter-chip--openai {
  border-color: rgb(167 243 208);
  color: rgb(4 120 87);
  background: rgb(236 253 245);
}

.account-filter-chip--anthropic {
  border-color: rgb(254 215 170);
  color: rgb(194 65 12);
  background: rgb(255 247 237);
}

.account-filter-chip--gemini {
  border-color: rgb(191 219 254);
  color: rgb(29 78 216);
  background: rgb(239 246 255);
}

.account-filter-chip--antigravity {
  border-color: rgb(221 214 254);
  color: rgb(109 40 217);
  background: rgb(245 243 255);
}

.account-filter-chip--deepseek {
  border-color: color-mix(in srgb, var(--lx-clay-info) 24%, var(--lx-clay-border));
  color: var(--lx-clay-info-deep);
  background: var(--lx-clay-info-soft);
}

.account-filter-chip--zhipu {
  border-color: rgb(199 210 254);
  color: rgb(67 56 202);
  background: rgb(238 242 255);
}

.account-filter-chip--grok,
.account-filter-chip--neutral {
  border-color: var(--lx-clay-border);
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-surface-soft);
}

.account-applied-filters__clear {
  color: var(--lx-clay-text-muted);
  font-size: 0.75rem;
  text-decoration: underline;
  text-underline-offset: 2px;
}

.account-filter-popover {
  position: absolute;
  z-index: 50;
  top: 52px;
  right: 0;
  width: min(36rem, calc(100vw - 2rem));
  padding: 14px;
  border: 1px solid var(--lx-clay-border);
  border-radius: 12px;
  background: var(--lx-clay-surface);
  box-shadow: 0 18px 42px rgb(31 23 47 / 0.15);
}

.account-filter-popover__header {
  display: flex;
  min-height: 28px;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
  color: var(--lx-clay-text);
  font-size: 0.8125rem;
  font-weight: 700;
}

.account-filter-popover__close {
  display: inline-flex;
  width: 28px;
  height: 28px;
  align-items: center;
  justify-content: center;
  border-radius: 7px;
  color: var(--lx-clay-text-muted);
}

.account-filter-popover__close:hover {
  color: var(--lx-clay-text);
  background: var(--lx-clay-recessed);
}

.account-filter-popover__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

@media (max-width: 639px) {
  .account-filter-row {
    align-items: stretch;
  }

  .account-filter-toggle {
    padding-inline: 12px;
  }

  .account-filter-popover {
    left: 0;
    right: auto;
    width: 100%;
  }

  .account-filter-popover__grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .account-filter-popover__grid :deep(.sm\:col-span-2) {
    grid-column: auto;
  }
}
</style>

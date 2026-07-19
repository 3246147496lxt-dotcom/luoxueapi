<template>
  <div
    class="key-table-shell"
    data-test="api-key-table"
  >
    <table :class="['key-table', compact && 'key-table-compact']">
      <thead>
        <tr>
          <th class="w-[170px]">
            <SortHeader column="name" :label="t('common.name')" />
          </th>
          <th class="w-[220px]">
            <SortHeader column="status" :label="t('common.status')" />
          </th>
          <th class="w-[180px]">{{ t('keys.group') }}</th>
          <th class="w-[250px]">{{ t('keys.apiKey') }}</th>
          <th v-if="isVisible('current_concurrency')" class="w-[130px]">
            <SortHeader column="current_concurrency" :label="t('keys.currentConcurrency')" />
          </th>
          <th v-if="isVisible('rate_limit')" class="w-[270px]">{{ t('keys.rateLimitColumn') }}</th>
          <th v-if="isVisible('expires_at')" class="w-[190px]">
            <SortHeader column="expires_at" :label="t('keys.expiresAt')" />
          </th>
          <th v-if="isVisible('created_at')" class="w-[190px]">
            <SortHeader column="created_at" :label="t('keys.created')" />
          </th>
          <th v-if="isVisible('last_used_at')" class="w-[190px]">
            <SortHeader column="last_used_at" :label="t('keys.lastUsedAt')" />
          </th>
          <th v-if="isVisible('last_used_ip')" class="w-[170px]">{{ t('keys.lastUsedIP') }}</th>
          <th v-if="isVisible('id')" class="w-[100px]">
            <SortHeader column="id" :label="t('keys.id')" />
          </th>
          <th class="w-[150px]">{{ t('keys.ipRestriction') }}</th>
          <th v-if="showActions" class="sticky-actions w-[280px] text-center">
            {{ t('common.actions') }}
          </th>
        </tr>
      </thead>

      <tbody>
        <tr
          v-for="row in apiKeys"
          :key="row.id"
          :data-key-id="row.id"
          :data-test="`api-key-table-row-${row.id}`"
        >
          <td>
            <div class="max-w-[170px] truncate font-medium text-gray-950 dark:text-white" :title="row.name">
              {{ row.name }}
            </div>
          </td>

          <td>
            <div class="flex min-w-[220px] flex-col gap-1.5">
              <div class="flex min-h-11 items-center gap-2">
                <button
                  type="button"
                  role="switch"
                  :aria-checked="row.status === 'active'"
                  :aria-busy="isStatusUpdating(row.id)"
                  :aria-label="`${row.name} · ${row.status === 'active' ? t('keys.disable') : t('keys.enable')}`"
                  :disabled="isStatusUpdating(row.id)"
                  :data-test="`key-table-status-switch-${row.id}`"
                  class="group relative -m-1 flex h-11 w-14 shrink-0 items-center justify-center rounded-full outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 disabled:cursor-wait disabled:opacity-60 dark:focus-visible:ring-offset-dark-800"
                  @click="emit('toggle-status', row)"
                >
                  <span
                    :class="[
                      'relative block h-[22px] w-10 rounded-full transition-colors duration-200 motion-reduce:transition-none',
                      row.status === 'active' ? 'bg-primary-600 dark:bg-primary-500' : 'bg-gray-300 dark:bg-dark-500'
                    ]"
                  >
                    <span
                      :class="[
                        'absolute left-0 top-[3px] h-4 w-4 rounded-full bg-white shadow-sm transition-transform duration-200 motion-reduce:transition-none',
                        row.status === 'active' ? 'translate-x-[21px]' : 'translate-x-[3px]'
                      ]"
                    />
                  </span>
                </button>
                <span :class="['h-2 w-2 shrink-0 rounded-full', statusMeta(row).dotClass]" aria-hidden="true" />
                <span :class="['whitespace-nowrap text-sm font-semibold', statusMeta(row).textClass]">
                  {{ t(`keys.status.${row.status}`) }}
                </span>
              </div>

              <div class="pl-1 text-xs tabular-nums text-gray-500 dark:text-gray-400">
                <template v-if="quotaValue(row) > 0">
                  <CreditAmount class="font-medium text-gray-700 dark:text-gray-200" :value="quotaUsedValue(row).toFixed(2)" icon-size="xs" />
                  <span class="mx-1.5 text-gray-300 dark:text-dark-500">/</span>
                  <CreditAmount :value="quotaValue(row).toFixed(2)" icon-size="xs" />
                </template>
                <span v-else>{{ t('keys.unlimitedQuota') }}</span>
              </div>

              <div v-if="quotaValue(row) > 0" class="h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
                <div
                  :class="['h-full rounded-full transition-[width] duration-300 motion-reduce:transition-none', meterColor(quotaPercent(row))]"
                  :style="{ width: `${quotaPercent(row)}%` }"
                  role="progressbar"
                  :aria-label="`${row.name} · ${t('keys.quotaUsage')}`"
                  aria-valuemin="0"
                  aria-valuemax="100"
                  :aria-valuenow="Math.round(quotaPercent(row))"
                  :data-test="`key-table-quota-progress-${row.id}`"
                />
              </div>

              <div class="flex flex-wrap gap-x-3 gap-y-1 pl-1 text-[11px] tabular-nums text-gray-500 dark:text-gray-400">
                <span class="inline-flex items-center gap-1">
                  <span>{{ t('keys.today') }}</span>
                  <CreditAmount v-if="usageFor(row)" :value="usageFor(row)?.today_actual_cost.toFixed(4) || '0.0000'" icon-size="xs" />
                  <span v-else>—</span>
                </span>
                <span class="inline-flex items-center gap-1">
                  <span>{{ t('keys.total') }}</span>
                  <CreditAmount v-if="usageFor(row)" :value="usageFor(row)?.total_actual_cost.toFixed(4) || '0.0000'" icon-size="xs" />
                  <span v-else>—</span>
                </span>
              </div>
            </div>
          </td>

          <td>
            <button
              type="button"
              class="group/dropdown flex min-h-11 max-w-[205px] items-center gap-2 rounded-xl px-2 text-left transition-colors hover:bg-gray-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:hover:bg-dark-700"
              :title="t('keys.clickToChangeGroup')"
              @click.stop="emit('change-group', row, $event)"
            >
              <GroupBadge
                v-if="row.group"
                :name="row.group.name"
                :platform="row.group.platform"
                :subscription-type="row.group.subscription_type"
                :rate-multiplier="row.group.rate_multiplier"
                :user-rate-multiplier="row.group ? userGroupRates[row.group.id] ?? null : null"
                :peak-rate-enabled="row.group.peak_rate_enabled"
                :peak-start="row.group.peak_start"
                :peak-end="row.group.peak_end"
                :peak-rate-multiplier="row.group.peak_rate_multiplier"
              />
              <span v-else class="text-sm text-gray-500 dark:text-gray-400">{{ t('keys.noGroup') }}</span>
              <Icon name="sort" size="sm" class="shrink-0 text-gray-400" />
            </button>
          </td>

          <td>
            <div class="flex min-h-11 min-w-[245px] max-w-[260px] items-center rounded-full bg-gray-100 pl-4 pr-1 dark:bg-dark-700">
              <code class="min-w-0 flex-1 truncate font-mono text-sm text-gray-800 dark:text-gray-100">
                {{ isRevealed(row.id) ? row.key : maskApiKey(row.key) }}
              </code>
              <button
                type="button"
                class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full text-gray-500 transition-colors hover:bg-white hover:text-gray-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:hover:bg-dark-600 dark:hover:text-white"
                :title="isRevealed(row.id) ? t('keys.hideKey') : t('keys.revealKey')"
                :aria-label="isRevealed(row.id) ? t('keys.hideKey') : t('keys.revealKey')"
                :data-test="`key-table-reveal-${row.id}`"
                @click="toggleReveal(row.id)"
              >
                <Icon :name="isRevealed(row.id) ? 'eyeOff' : 'eye'" size="md" />
              </button>
              <button
                type="button"
                class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full text-gray-500 transition-colors hover:bg-white hover:text-gray-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:hover:bg-dark-600 dark:hover:text-white"
                :class="copiedKeyId === row.id && 'text-primary-600 dark:text-primary-400'"
                :title="copiedKeyId === row.id ? t('keys.copied') : t('keys.copyToClipboard')"
                :aria-label="copiedKeyId === row.id ? t('keys.copied') : t('keys.copyToClipboard')"
                :data-test="`key-table-copy-${row.id}`"
                @click="emit('copy-key', row)"
              >
                <Icon :name="copiedKeyId === row.id ? 'check' : 'copy'" size="md" />
              </button>
            </div>
          </td>

          <td v-if="isVisible('current_concurrency')">
            <span :class="['table-pill tabular-nums', finiteNumber(row.current_concurrency) > 0 && 'table-pill-active']">
              {{ finiteNumber(row.current_concurrency) }}
            </span>
          </td>

          <td v-if="isVisible('rate_limit')">
            <div v-if="rateLimits(row).length" class="min-w-[245px] space-y-1.5">
              <div v-for="limit in rateLimits(row)" :key="limit.label" class="flex items-center gap-2 text-xs">
                <span class="w-6 font-semibold text-gray-500 dark:text-gray-400">{{ limit.label }}</span>
                <div class="h-1.5 flex-1 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
                  <div :class="['h-full rounded-full', meterColor(limit.percent)]" :style="{ width: `${limit.percent}%` }" />
                </div>
                <CreditAmount
                  class="w-[110px] justify-end text-right text-gray-600 dark:text-gray-300"
                  :value="`${limit.used.toFixed(2)} / ${limit.total.toFixed(2)}`"
                  icon-size="xs"
                />
              </div>
              <button
                v-if="hasRateLimitUsage(row)"
                type="button"
                class="inline-flex min-h-11 items-center gap-1.5 rounded-xl px-2 text-xs font-medium text-gray-600 hover:bg-gray-100 hover:text-primary-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:text-gray-300 dark:hover:bg-dark-700 dark:hover:text-primary-300"
                @click="emit('reset-rate-limit', row)"
              >
                <Icon name="refresh" size="sm" />
                {{ t('keys.resetUsage') }}
              </button>
            </div>
            <span v-else class="text-sm text-gray-500 dark:text-gray-400">{{ t('keys.noRateLimit') }}</span>
          </td>

          <td v-if="isVisible('expires_at')" class="whitespace-nowrap text-sm tabular-nums">
            <span :class="isExpired(row) ? 'text-red-600 dark:text-red-400' : 'text-gray-700 dark:text-gray-200'">
              {{ row.expires_at ? formatDateTime(row.expires_at) : t('keys.noExpiration') }}
            </span>
          </td>

          <td v-if="isVisible('created_at')" class="whitespace-nowrap text-sm tabular-nums text-gray-700 dark:text-gray-200">
            {{ formatDateTime(row.created_at) }}
          </td>

          <td v-if="isVisible('last_used_at')" class="whitespace-nowrap text-sm tabular-nums text-gray-700 dark:text-gray-200">
            {{ row.last_used_at ? formatDateTime(row.last_used_at) : '-' }}
          </td>

          <td v-if="isVisible('last_used_ip')" class="font-mono text-sm text-gray-700 dark:text-gray-200">
            {{ row.last_used_ip || '-' }}
          </td>

          <td v-if="isVisible('id')" class="font-mono text-sm text-gray-500 dark:text-gray-400">#{{ row.id }}</td>

          <td>
            <span v-if="hasIpRestriction(row)" class="flex min-w-[120px] flex-col gap-1 text-xs text-gray-600 dark:text-gray-300">
              <span>{{ t('keys.whitelistCount', { count: row.ip_whitelist?.length ?? 0 }) }}</span>
              <span>{{ t('keys.blacklistCount', { count: row.ip_blacklist?.length ?? 0 }) }}</span>
            </span>
            <span v-else class="table-pill">{{ t('keys.noIpRestriction') }}</span>
          </td>

          <td v-if="showActions" class="sticky-actions">
            <div class="flex min-w-[250px] items-center justify-end gap-1.5">
              <button type="button" class="table-action" @click="emit('use-key', row)">
                <Icon name="terminal" size="sm" />
                {{ t('keys.useKey') }}
              </button>
              <button
                v-if="showCcsImport"
                type="button"
                class="table-action table-action-icon"
                :title="t('keys.importToCcSwitch')"
                :aria-label="t('keys.importToCcSwitch')"
                @click="emit('import-ccs', row)"
              >
                <Icon name="upload" size="sm" />
              </button>
              <button type="button" class="table-action" @click="emit('edit', row)">
                <Icon name="edit" size="sm" />
                {{ t('common.edit') }}
              </button>
              <button type="button" class="table-action table-action-danger" @click="emit('delete', row)">
                <Icon name="trash" size="sm" />
                {{ t('common.delete') }}
              </button>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { defineComponent, h, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import CreditAmount from '@/components/common/CreditAmount.vue'
import Icon from '@/components/icons/Icon.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import type { ApiKey } from '@/types'
import type { BatchApiKeyUsageStats } from '@/api/usage'
import { formatDateTime } from '@/utils/format'
import { maskApiKey } from '@/utils/maskApiKey'

interface Props {
  apiKeys: ApiKey[]
  usageStats: Record<string, BatchApiKeyUsageStats>
  userGroupRates: Record<number, number>
  visibleColumns: string[]
  showActions?: boolean
  showCcsImport?: boolean
  compact?: boolean
  copiedKeyId?: number | null
  statusUpdatingIds?: number[]
  sortBy: string
  sortOrder: 'asc' | 'desc'
  now?: Date
}

const props = withDefaults(defineProps<Props>(), {
  showActions: true,
  showCcsImport: true,
  compact: false,
  copiedKeyId: null,
  statusUpdatingIds: () => [],
  now: () => new Date()
})

const emit = defineEmits<{
  (event: 'copy-key', key: ApiKey): void
  (event: 'toggle-status', key: ApiKey): void
  (event: 'change-group', key: ApiKey, mouseEvent: MouseEvent): void
  (event: 'use-key', key: ApiKey): void
  (event: 'import-ccs', key: ApiKey): void
  (event: 'edit', key: ApiKey): void
  (event: 'delete', key: ApiKey): void
  (event: 'reset-rate-limit', key: ApiKey): void
  (event: 'sort', key: string, order: 'asc' | 'desc'): void
}>()

const { t } = useI18n()
const revealedKeyIds = reactive(new Set<number>())

const SortHeader = defineComponent({
  props: {
    column: { type: String, required: true },
    label: { type: String, required: true }
  },
  setup(componentProps) {
    return () => h(
      'button',
      {
        type: 'button',
        class: 'inline-flex min-h-11 items-center gap-1.5 rounded-lg px-1 text-left font-semibold text-gray-500 transition-colors hover:text-gray-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:text-gray-400 dark:hover:text-white',
        'aria-label': componentProps.label,
        onClick: () => {
          const nextOrder = props.sortBy === componentProps.column && props.sortOrder === 'asc' ? 'desc' : 'asc'
          emit('sort', componentProps.column, nextOrder)
        }
      },
      [
        componentProps.label,
        props.sortBy === componentProps.column
          ? h(Icon, { name: props.sortOrder === 'asc' ? 'chevronUp' : 'chevronDown', size: 'sm' })
          : h(Icon, { name: 'sort', size: 'sm', class: 'opacity-50' })
      ]
    )
  }
})

const finiteNumber = (value: unknown) => typeof value === 'number' && Number.isFinite(value) ? value : 0
const isVisible = (key: string) => props.visibleColumns.includes(key)
const isStatusUpdating = (id: number) => props.statusUpdatingIds.includes(id)
const isRevealed = (id: number) => revealedKeyIds.has(id)
const toggleReveal = (id: number) => {
  if (revealedKeyIds.has(id)) revealedKeyIds.delete(id)
  else revealedKeyIds.add(id)
}
const usageFor = (key: ApiKey) => {
  const usage = props.usageStats[String(key.id)]
  if (!usage) return null
  return {
    today_actual_cost: finiteNumber(usage?.today_actual_cost),
    total_actual_cost: finiteNumber(usage?.total_actual_cost)
  }
}
const quotaValue = (key: ApiKey) => finiteNumber(key.quota)
const quotaUsedValue = (key: ApiKey) => finiteNumber(key.quota_used)
const quotaPercent = (key: ApiKey) => {
  if (quotaValue(key) <= 0) return 0
  return Math.min(Math.max((quotaUsedValue(key) / quotaValue(key)) * 100, 0), 100)
}
const isExpired = (key: ApiKey) => Boolean(key.expires_at && new Date(key.expires_at) < props.now)
const hasIpRestriction = (key: ApiKey) => (key.ip_whitelist?.length ?? 0) > 0 || (key.ip_blacklist?.length ?? 0) > 0
const statusMeta = (key: ApiKey) => {
  switch (key.status) {
    case 'active':
      return { dotClass: 'bg-primary-600 dark:bg-primary-400', textClass: 'text-primary-700 dark:text-primary-300' }
    case 'quota_exhausted':
      return { dotClass: 'bg-amber-500', textClass: 'text-amber-700 dark:text-amber-300' }
    case 'expired':
      return { dotClass: 'bg-red-500', textClass: 'text-red-700 dark:text-red-300' }
    default:
      return { dotClass: 'bg-gray-400 dark:bg-dark-400', textClass: 'text-gray-600 dark:text-gray-300' }
  }
}
const rateLimits = (key: ApiKey) => [
  { label: '5h', used: finiteNumber(key.usage_5h), total: finiteNumber(key.rate_limit_5h) },
  { label: '1d', used: finiteNumber(key.usage_1d), total: finiteNumber(key.rate_limit_1d) },
  { label: '7d', used: finiteNumber(key.usage_7d), total: finiteNumber(key.rate_limit_7d) }
].filter((limit) => limit.total > 0).map((limit) => ({
  ...limit,
  percent: Math.min(Math.max((limit.used / limit.total) * 100, 0), 100)
}))
const hasRateLimitUsage = (key: ApiKey) =>
  finiteNumber(key.usage_5h) > 0 || finiteNumber(key.usage_1d) > 0 || finiteNumber(key.usage_7d) > 0
const meterColor = (percent: number) => {
  if (percent >= 100) return 'bg-red-500'
  if (percent >= 80) return 'bg-amber-500'
  return 'bg-primary-600 dark:bg-primary-400'
}
</script>

<style scoped>
.key-table-shell {
  max-width: 100%;
  overflow-x: auto;
  border-top: 1px solid rgb(229 231 235);
  border-bottom: 1px solid rgb(229 231 235);
  background: white;
  scrollbar-color: rgb(203 213 225) transparent;
  scrollbar-width: thin;
}

:global(.dark) .key-table-shell {
  border-color: rgb(55 65 81);
  background: rgb(31 41 55);
  scrollbar-color: rgb(75 85 99) transparent;
}

.key-table {
  width: max-content;
  min-width: 100%;
  border-collapse: separate;
  border-spacing: 0;
  table-layout: fixed;
}

.key-table th,
.key-table td {
  padding: 0.9rem 1rem;
  text-align: left;
  vertical-align: middle;
}

.key-table th {
  height: 4.5rem;
  border-bottom: 1px solid rgb(229 231 235);
  color: rgb(107 114 128);
  font-size: 0.875rem;
  font-weight: 600;
  white-space: nowrap;
}

.key-table td {
  min-height: 6.5rem;
  border-bottom: 1px solid rgb(229 231 235);
}

.key-table-compact th {
  height: 3.75rem;
  padding-block: 0.625rem;
}

.key-table-compact td {
  padding-block: 0.625rem;
}

.key-table tbody tr:last-child td {
  border-bottom: 0;
}

.key-table tbody tr {
  transition: background-color 160ms ease;
}

.key-table tbody tr:hover td {
  background: rgb(249 250 251);
}

:global(.dark) .key-table th,
:global(.dark) .key-table td {
  border-color: rgb(55 65 81);
}

:global(.dark) .key-table tbody tr:hover td {
  background: rgb(17 24 39 / 0.45);
}

.sticky-actions {
  position: sticky;
  right: 0;
  z-index: 10;
  border-left: 1px solid rgb(229 231 235);
  background: white;
  box-shadow: -10px 0 20px -18px rgb(15 23 42 / 0.45);
}

:global(.dark) .sticky-actions {
  border-left-color: rgb(55 65 81);
  background: rgb(31 41 55);
}

.key-table tbody tr:hover .sticky-actions {
  background: rgb(249 250 251);
}

:global(.dark) .key-table tbody tr:hover .sticky-actions {
  background: rgb(24 33 47);
}

.table-pill {
  display: inline-flex;
  min-height: 2rem;
  align-items: center;
  border-radius: 9999px;
  border: 1px solid rgb(209 213 219);
  padding-inline: 0.75rem;
  color: rgb(75 85 99);
  font-size: 0.8125rem;
  white-space: nowrap;
}

.table-pill-active {
  border-color: rgb(153 246 228);
  background: rgb(240 253 250);
  color: rgb(15 118 110);
}

:global(.dark) .table-pill {
  border-color: rgb(75 85 99);
  color: rgb(209 213 219);
}

:global(.dark) .table-pill-active {
  border-color: rgb(17 94 89);
  background: rgb(19 78 74 / 0.35);
  color: rgb(94 234 212);
}

.table-action {
  display: inline-flex;
  min-height: 2.75rem;
  align-items: center;
  gap: 0.375rem;
  border-radius: 9999px;
  background: rgb(243 244 246);
  padding-inline: 0.65rem;
  color: rgb(75 85 99);
  font-size: 0.8125rem;
  font-weight: 600;
  white-space: nowrap;
  transition: background-color 160ms ease, color 160ms ease;
}

.table-action:hover {
  background: rgb(240 253 250);
  color: rgb(15 118 110);
}

.table-action:focus-visible {
  outline: 2px solid rgb(20 184 166);
  outline-offset: 2px;
}

.table-action-danger {
  color: rgb(220 38 38);
}

.table-action-icon {
  width: 2.75rem;
  justify-content: center;
  padding-inline: 0;
}

.table-action-danger:hover {
  background: rgb(254 242 242);
  color: rgb(185 28 28);
}

:global(.dark) .table-action {
  background: rgb(55 65 81);
  color: rgb(209 213 219);
}

:global(.dark) .table-action:hover {
  background: rgb(19 78 74 / 0.4);
  color: rgb(94 234 212);
}

:global(.dark) .table-action-danger {
  color: rgb(248 113 113);
}

:global(.dark) .table-action-danger:hover {
  background: rgb(127 29 29 / 0.35);
  color: rgb(252 165 165);
}

@media (prefers-reduced-motion: reduce) {
  .key-table tbody tr,
  .table-action {
    transition: none;
  }
}
</style>

<template>
  <article
    class="overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-[0_12px_30px_rgba(15,23,42,0.045)] dark:border-dark-600 dark:bg-dark-800"
    :data-key-id="apiKey.id"
    :data-test="`api-key-card-${apiKey.id}`"
  >
    <dl class="px-4 py-4 sm:px-6 sm:py-5">
      <div class="key-row key-row-compact">
        <dt class="key-label">{{ t('common.name') }}</dt>
        <dd class="min-w-0 text-right text-base font-semibold text-gray-950 dark:text-white">
          <span class="break-words">{{ apiKey.name }}</span>
        </dd>
      </div>

      <div class="key-row key-row-status">
        <dt class="key-label">{{ t('common.status') }}</dt>
        <dd class="min-w-0">
          <div class="flex flex-col items-end gap-2.5">
            <div class="flex min-h-11 items-center justify-end gap-2">
              <button
                type="button"
                role="switch"
                :aria-checked="isActive"
                :aria-busy="statusUpdating"
                :aria-label="`${apiKey.name} · ${isActive ? t('keys.disable') : t('keys.enable')}`"
                :disabled="statusUpdating"
                :data-test="`key-status-switch-${apiKey.id}`"
                class="group relative -m-1 flex h-11 w-14 items-center justify-center rounded-full outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 disabled:cursor-wait disabled:opacity-60 dark:focus-visible:ring-offset-dark-800"
                @click="emit('toggle-status', apiKey)"
              >
                <span
                  :class="[
                    'relative block h-[22px] w-10 rounded-full transition-colors duration-200 motion-reduce:transition-none',
                    isActive ? 'bg-primary-600 dark:bg-primary-500' : 'bg-gray-300 dark:bg-dark-500'
                  ]"
                >
                  <span
                    :class="[
                      'absolute left-0 top-[3px] h-4 w-4 rounded-full bg-white shadow-sm transition-transform duration-200 motion-reduce:transition-none',
                      isActive ? 'translate-x-[21px]' : 'translate-x-[3px]'
                    ]"
                  />
                </span>
              </button>
              <span :class="['h-2 w-2 rounded-full', statusMeta.dotClass]" aria-hidden="true" />
              <span :class="['text-sm font-semibold', statusMeta.textClass]">
                {{ t(`keys.status.${apiKey.status}`) }}
              </span>
            </div>

            <div class="flex w-full max-w-[340px] flex-wrap justify-end gap-x-3 gap-y-1 text-xs tabular-nums text-gray-500 dark:text-gray-400">
              <span class="inline-flex items-center gap-1">
                <span>{{ t('keys.today') }}</span>
                <CreditAmount v-if="hasUsage" :value="todayCost.toFixed(4)" icon-size="xs" />
                <span v-else>—</span>
              </span>
              <span class="inline-flex items-center gap-1">
                <span>{{ t('keys.total') }}</span>
                <CreditAmount v-if="hasUsage" :value="totalCost.toFixed(4)" icon-size="xs" />
                <span v-else>—</span>
              </span>
            </div>

            <div class="w-full max-w-[340px] text-right text-xs tabular-nums text-gray-500 dark:text-gray-400">
              <template v-if="quotaValue > 0">
                <CreditAmount class="font-semibold text-gray-700 dark:text-gray-200" :value="quotaUsedValue.toFixed(2)" icon-size="xs" />
                <span class="mx-1.5 text-gray-300 dark:text-dark-500">/</span>
                <CreditAmount :value="quotaValue.toFixed(2)" icon-size="xs" />
              </template>
              <span v-else>{{ t('keys.unlimitedQuota') }}</span>
            </div>

            <div
              v-if="quotaValue > 0"
              class="w-full max-w-[340px]"
              role="progressbar"
              :aria-label="t('keys.quotaUsage')"
              aria-valuemin="0"
              aria-valuemax="100"
              :aria-valuenow="Math.round(quotaPercent)"
              :data-test="`key-quota-progress-${apiKey.id}`"
            >
              <div class="h-1.5 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
                <div
                  :class="['h-full rounded-full transition-[width] duration-300 motion-reduce:transition-none', quotaColorClass]"
                  :style="{ width: `${quotaPercent}%` }"
                />
              </div>
            </div>
          </div>
        </dd>
      </div>

      <div class="key-row">
        <dt class="key-label">{{ t('keys.group') }}</dt>
        <dd class="group/dropdown flex min-w-0 justify-end">
          <button
            type="button"
            class="flex min-h-11 max-w-full items-center gap-2 rounded-xl px-2 text-left transition-colors hover:bg-gray-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:hover:bg-dark-700"
            :title="t('keys.clickToChangeGroup')"
            @click.stop="emit('change-group', apiKey, $event)"
          >
            <GroupBadge
              v-if="apiKey.group"
              :name="apiKey.group.name"
              :platform="apiKey.group.platform"
              :subscription-type="apiKey.group.subscription_type"
              :rate-multiplier="apiKey.group.rate_multiplier"
              :user-rate-multiplier="userGroupRate"
              :peak-rate-enabled="apiKey.group.peak_rate_enabled"
              :peak-start="apiKey.group.peak_start"
              :peak-end="apiKey.group.peak_end"
              :peak-rate-multiplier="apiKey.group.peak_rate_multiplier"
            />
            <span v-else class="text-sm text-gray-500 dark:text-gray-400">{{ t('keys.noGroup') }}</span>
            <Icon name="sort" size="sm" class="shrink-0 text-gray-400" />
          </button>
        </dd>
      </div>

      <div class="key-row">
        <dt class="key-label">{{ t('keys.apiKey') }}</dt>
        <dd class="flex min-w-0 justify-end">
          <div class="flex min-h-11 min-w-0 max-w-full items-center rounded-full bg-gray-100 pl-4 pr-1 dark:bg-dark-700">
            <code class="min-w-0 truncate font-mono text-sm text-gray-800 dark:text-gray-100">
              {{ revealKey ? apiKey.key : maskApiKey(apiKey.key) }}
            </code>
            <button
              type="button"
              class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full text-gray-500 transition-colors hover:bg-white hover:text-gray-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:hover:bg-dark-600 dark:hover:text-white"
              :title="revealKey ? t('keys.hideKey') : t('keys.revealKey')"
              :aria-label="revealKey ? t('keys.hideKey') : t('keys.revealKey')"
              :data-test="`key-reveal-${apiKey.id}`"
              @click="revealKey = !revealKey"
            >
              <Icon :name="revealKey ? 'eyeOff' : 'eye'" size="md" />
            </button>
            <button
              type="button"
              class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full text-gray-500 transition-colors hover:bg-white hover:text-gray-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:hover:bg-dark-600 dark:hover:text-white"
              :class="copied && 'text-primary-600 dark:text-primary-400'"
              :title="copied ? t('keys.copied') : t('keys.copyToClipboard')"
              :aria-label="copied ? t('keys.copied') : t('keys.copyToClipboard')"
              :data-test="`key-copy-${apiKey.id}`"
              @click="emit('copy-key', apiKey)"
            >
              <Icon :name="copied ? 'check' : 'copy'" size="md" />
            </button>
          </div>
        </dd>
      </div>

      <div class="key-row">
        <dt class="key-label">{{ t('keys.ipRestriction') }}</dt>
        <dd class="text-right text-sm text-gray-700 dark:text-gray-200">
          <span v-if="hasIpRestriction" class="inline-flex flex-wrap justify-end gap-x-2 gap-y-1">
            <span>{{ t('keys.whitelistCount', { count: whitelistCount }) }}</span>
            <span>{{ t('keys.blacklistCount', { count: blacklistCount }) }}</span>
          </span>
          <span v-else class="key-pill">{{ t('keys.noIpRestriction') }}</span>
        </dd>
      </div>

      <div v-if="isVisible('current_concurrency')" class="key-row">
        <dt class="key-label">{{ t('keys.currentConcurrency') }}</dt>
        <dd class="text-right">
          <span :class="['key-pill tabular-nums', apiKey.current_concurrency > 0 && 'key-pill-active']">
            {{ apiKey.current_concurrency ?? 0 }}
          </span>
        </dd>
      </div>

      <div v-if="isVisible('rate_limit')" class="key-row key-row-rate-limit">
        <dt class="key-label">{{ t('keys.rateLimitUsage') }}</dt>
        <dd class="min-w-0">
          <div v-if="rateLimits.length" class="grid gap-2 sm:grid-cols-3">
            <div v-for="limit in rateLimits" :key="limit.label" class="rate-limit-card">
              <div class="flex items-center justify-between gap-3 text-xs">
                <span class="font-semibold text-gray-600 dark:text-gray-300">{{ limit.label }}</span>
                <CreditAmount
                  class="text-gray-700 dark:text-gray-200"
                  :value="`${limit.used.toFixed(2)} / ${limit.total.toFixed(2)}`"
                  icon-size="xs"
                />
              </div>
              <div
                class="mt-2 h-1 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600"
                role="progressbar"
                :aria-label="`${limit.label} ${t('keys.rateLimitUsage')}`"
                aria-valuemin="0"
                aria-valuemax="100"
                :aria-valuenow="Math.round(limit.percent)"
              >
                <div
                  :class="['h-full rounded-full', meterColor(limit.percent)]"
                  :style="{ width: `${limit.percent}%` }"
                />
              </div>
              <div v-if="limit.resetAt" class="mt-1.5 text-[11px] tabular-nums text-gray-500 dark:text-gray-400">
                {{ t('keys.resetsIn', { time: formatResetTime(limit.resetAt) }) }}
              </div>
            </div>
          </div>
          <div v-else class="text-right text-sm text-gray-500 dark:text-gray-400">{{ t('keys.noRateLimit') }}</div>
          <div v-if="hasRateLimitUsage" class="mt-2 flex justify-end">
            <button
              type="button"
              class="inline-flex min-h-11 items-center gap-2 rounded-xl px-3 text-sm font-medium text-gray-600 transition-colors hover:bg-gray-100 hover:text-primary-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:text-gray-300 dark:hover:bg-dark-700 dark:hover:text-primary-300"
              @click="emit('reset-rate-limit', apiKey)"
            >
              <Icon name="refresh" size="sm" />
              {{ t('keys.resetUsage') }}
            </button>
          </div>
        </dd>
      </div>

      <div v-if="isVisible('created_at')" class="key-row">
        <dt class="key-label">{{ t('keys.created') }}</dt>
        <dd class="text-right text-sm tabular-nums text-gray-700 dark:text-gray-200">
          {{ formatDateTime(apiKey.created_at) }}
        </dd>
      </div>

      <div v-if="isVisible('expires_at')" class="key-row">
        <dt class="key-label">{{ t('keys.expiresAt') }}</dt>
        <dd :class="['text-right text-sm tabular-nums', isExpired ? 'text-red-600 dark:text-red-400' : 'text-gray-700 dark:text-gray-200']">
          {{ apiKey.expires_at ? formatDateTime(apiKey.expires_at) : t('keys.noExpiration') }}
        </dd>
      </div>

      <div v-if="isVisible('last_used_at')" class="key-row">
        <dt class="key-label">{{ t('keys.lastUsedAt') }}</dt>
        <dd class="text-right text-sm tabular-nums text-gray-700 dark:text-gray-200">
          {{ apiKey.last_used_at ? formatDateTime(apiKey.last_used_at) : '-' }}
        </dd>
      </div>

      <div v-if="isVisible('last_used_ip')" class="key-row">
        <dt class="key-label">{{ t('keys.lastUsedIP') }}</dt>
        <dd class="min-w-0 truncate text-right font-mono text-sm text-gray-700 dark:text-gray-200">
          {{ apiKey.last_used_ip || '-' }}
        </dd>
      </div>

      <div v-if="isVisible('id')" class="key-row key-row-last">
        <dt class="key-label">{{ t('keys.id') }}</dt>
        <dd class="text-right font-mono text-sm text-gray-500 dark:text-gray-400">#{{ apiKey.id }}</dd>
      </div>
    </dl>

    <div
      v-if="showActions"
      class="flex flex-wrap items-center justify-end gap-1 border-t border-gray-100 px-3 py-2 dark:border-dark-700 sm:px-5"
      :data-test="`key-actions-${apiKey.id}`"
    >
      <button type="button" class="card-action" @click="emit('use-key', apiKey)">
        <Icon name="terminal" size="sm" />
        {{ t('keys.useKey') }}
      </button>
      <button
        v-if="showCcsImport"
        type="button"
        class="card-action"
        @click="emit('import-ccs', apiKey)"
      >
        <Icon name="upload" size="sm" />
        {{ t('keys.importToCcSwitch') }}
      </button>
      <button type="button" class="card-action" @click="emit('edit', apiKey)">
        <Icon name="edit" size="sm" />
        {{ t('common.edit') }}
      </button>
      <button type="button" class="card-action card-action-danger" @click="emit('delete', apiKey)">
        <Icon name="trash" size="sm" />
        {{ t('common.delete') }}
      </button>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import CreditAmount from '@/components/common/CreditAmount.vue'
import Icon from '@/components/icons/Icon.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import type { ApiKey } from '@/types'
import type { BatchApiKeyUsageStats } from '@/api/usage'
import { formatDateTime } from '@/utils/format'
import { maskApiKey } from '@/utils/maskApiKey'

interface Props {
  apiKey: ApiKey
  usage?: BatchApiKeyUsageStats
  userGroupRate?: number | null
  visibleColumns: string[]
  showActions?: boolean
  showCcsImport?: boolean
  copied?: boolean
  statusUpdating?: boolean
  now?: Date
}

const props = withDefaults(defineProps<Props>(), {
  usage: undefined,
  userGroupRate: null,
  showActions: true,
  showCcsImport: true,
  copied: false,
  statusUpdating: false,
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
}>()

const { t } = useI18n()
const revealKey = ref(false)

const finiteNumber = (value: unknown) =>
  typeof value === 'number' && Number.isFinite(value) ? value : 0

const isActive = computed(() => props.apiKey.status === 'active')
const isExpired = computed(() => Boolean(props.apiKey.expires_at && new Date(props.apiKey.expires_at) < props.now))
const whitelistCount = computed(() => props.apiKey.ip_whitelist?.length ?? 0)
const blacklistCount = computed(() => props.apiKey.ip_blacklist?.length ?? 0)
const hasIpRestriction = computed(() => whitelistCount.value > 0 || blacklistCount.value > 0)
const hasUsage = computed(() => props.usage !== undefined)
const todayCost = computed(() => finiteNumber(props.usage?.today_actual_cost))
const totalCost = computed(() => finiteNumber(props.usage?.total_actual_cost))
const quotaValue = computed(() => finiteNumber(props.apiKey.quota))
const quotaUsedValue = computed(() => finiteNumber(props.apiKey.quota_used))
const quotaPercent = computed(() => {
  if (quotaValue.value <= 0) return 0
  return Math.min(Math.max((quotaUsedValue.value / quotaValue.value) * 100, 0), 100)
})
const quotaColorClass = computed(() => meterColor(quotaPercent.value))

const statusMeta = computed(() => {
  switch (props.apiKey.status) {
    case 'active':
      return { dotClass: 'bg-primary-600 dark:bg-primary-400', textClass: 'text-primary-700 dark:text-primary-300' }
    case 'quota_exhausted':
      return { dotClass: 'bg-amber-500', textClass: 'text-amber-700 dark:text-amber-300' }
    case 'expired':
      return { dotClass: 'bg-red-500', textClass: 'text-red-700 dark:text-red-300' }
    default:
      return { dotClass: 'bg-gray-400 dark:bg-dark-400', textClass: 'text-gray-600 dark:text-gray-300' }
  }
})

const rateLimits = computed(() => [
  { label: '5h', used: finiteNumber(props.apiKey.usage_5h), total: finiteNumber(props.apiKey.rate_limit_5h), resetAt: props.apiKey.reset_5h_at },
  { label: '1d', used: finiteNumber(props.apiKey.usage_1d), total: finiteNumber(props.apiKey.rate_limit_1d), resetAt: props.apiKey.reset_1d_at },
  { label: '7d', used: finiteNumber(props.apiKey.usage_7d), total: finiteNumber(props.apiKey.rate_limit_7d), resetAt: props.apiKey.reset_7d_at }
].filter((limit) => limit.total > 0).map((limit) => ({
  ...limit,
  percent: Math.min(Math.max((limit.used / limit.total) * 100, 0), 100)
})))

const hasRateLimitUsage = computed(() =>
  finiteNumber(props.apiKey.usage_5h) > 0 ||
  finiteNumber(props.apiKey.usage_1d) > 0 ||
  finiteNumber(props.apiKey.usage_7d) > 0
)

const isVisible = (key: string) => props.visibleColumns.includes(key)

function meterColor(percent: number) {
  if (percent >= 100) return 'bg-red-500'
  if (percent >= 80) return 'bg-amber-500'
  return 'bg-primary-600 dark:bg-primary-400'
}

function formatResetTime(resetAt: string | null) {
  if (!resetAt) return ''
  const diff = new Date(resetAt).getTime() - props.now.getTime()
  if (diff <= 0) return t('keys.resetNow')
  const days = Math.floor(diff / 86400000)
  const hours = Math.floor((diff % 86400000) / 3600000)
  const minutes = Math.floor((diff % 3600000) / 60000)
  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${minutes}m`
}
</script>

<style scoped>
.key-row {
  display: grid;
  grid-template-columns: minmax(6.25rem, 0.42fr) minmax(0, 1fr);
  align-items: center;
  gap: 1rem;
  min-height: 3.25rem;
  border-bottom: 1px dashed rgb(229 231 235);
}

:global(.dark) .key-row {
  border-bottom-color: rgb(55 65 81);
}

.key-row-status,
.key-row-rate-limit {
  align-items: start;
  padding-block: 0.75rem;
}

.key-row-last {
  border-bottom: 0;
}

.key-label {
  font-size: 0.875rem;
  font-weight: 600;
  color: rgb(75 85 99);
}

:global(.dark) .key-label {
  color: rgb(209 213 219);
}

.key-pill {
  display: inline-flex;
  min-height: 1.75rem;
  align-items: center;
  border-radius: 9999px;
  border: 1px solid rgb(209 213 219);
  padding-inline: 0.75rem;
  color: rgb(75 85 99);
}

.key-pill-active {
  border-color: rgb(153 246 228);
  background: rgb(240 253 250);
  color: rgb(15 118 110);
}

:global(.dark) .key-pill {
  border-color: rgb(75 85 99);
  color: rgb(209 213 219);
}

:global(.dark) .key-pill-active {
  border-color: rgb(17 94 89);
  background: rgb(19 78 74 / 0.35);
  color: rgb(94 234 212);
}

.rate-limit-card {
  min-width: 0;
  border-radius: 0.75rem;
  background: rgb(249 250 251);
  padding: 0.625rem;
}

:global(.dark) .rate-limit-card {
  background: rgb(31 41 55 / 0.8);
}

.card-action {
  display: inline-flex;
  min-height: 2.75rem;
  align-items: center;
  gap: 0.5rem;
  border-radius: 0.75rem;
  padding-inline: 0.75rem;
  font-size: 0.875rem;
  font-weight: 500;
  color: rgb(75 85 99);
  transition: background-color 150ms, color 150ms;
}

.card-action:hover {
  background: rgb(243 244 246);
  color: rgb(15 118 110);
}

.card-action:focus-visible {
  outline: 2px solid rgb(20 184 166);
  outline-offset: 2px;
}

.card-action-danger:hover {
  background: rgb(254 242 242);
  color: rgb(220 38 38);
}

:global(.dark) .card-action {
  color: rgb(209 213 219);
}

:global(.dark) .card-action:hover {
  background: rgb(31 41 55);
  color: rgb(94 234 212);
}

:global(.dark) .card-action-danger:hover {
  background: rgb(127 29 29 / 0.25);
  color: rgb(248 113 113);
}

</style>

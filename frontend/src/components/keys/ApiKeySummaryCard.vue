<template>
  <article
    class="api-key-summary-card space-y-5 rounded-[24px] border border-gray-100 bg-white p-5 shadow-sm dark:border-white/5 dark:bg-[#251e2f]"
    :data-key-id="apiKey.id"
    :data-test="`api-key-summary-card-${apiKey.id}`"
  >
    <div class="flex items-start justify-between">
      <div class="min-w-0">
        <h3
          class="truncate text-sm font-black text-gray-900 dark:text-white"
          :title="apiKey.name"
        >
          {{ apiKey.name }}
        </h3>
        <p class="mt-0.5 text-[9px] font-black uppercase text-gray-400">
          {{ t('keys.id') }}: #{{ apiKey.id }}
        </p>
      </div>

      <div class="flex shrink-0 items-center gap-2">
        <span :class="['text-[10px] font-bold uppercase', statusMeta.textClass]">
          {{ t(`keys.status.${apiKey.status}`) }}
        </span>
        <button
          type="button"
          role="switch"
          :aria-checked="isActive"
          :aria-busy="statusUpdating"
          :aria-label="`${apiKey.name} · ${isActive ? t('keys.disable') : t('keys.enable')}`"
          :disabled="statusUpdating"
          :data-test="`key-summary-status-switch-${apiKey.id}`"
          class="group relative flex h-11 w-11 shrink-0 items-center justify-center rounded-full outline-none focus-visible:ring-2 focus-visible:ring-primary-500 disabled:cursor-wait disabled:opacity-60"
          @click.stop="emit('toggle-status', apiKey)"
        >
          <div
            :class="[
              'relative h-[22px] w-[40px] rounded-full transition-colors',
              isActive ? 'bg-primary-600' : 'bg-gray-300 dark:bg-white/15'
            ]"
            aria-hidden="true"
          >
            <div
              :class="[
                'absolute top-[2px] h-[18px] w-[18px] rounded-full bg-white shadow-sm transition-[left,right]',
                isActive ? 'right-[2px]' : 'left-[2px]'
              ]"
            />
          </div>
        </button>
      </div>
    </div>

    <div class="space-y-2.5">
      <button
        type="button"
        :class="[
          'api-key-summary-card__group group/dropdown flex h-11 w-full items-center justify-between rounded-xl border px-4 transition-transform active:scale-95',
          apiKey.group
            ? 'api-key-summary-card__group--assigned'
            : 'api-key-summary-card__group--unassigned',
          apiKey.group && `api-key-summary-card__group--${apiKey.group.platform}`
        ]"
        :data-platform="apiKey.group?.platform"
        :title="t('keys.clickToChangeGroup')"
        :aria-label="`${t('keys.group')}: ${apiKey.group?.name || t('keys.noGroup')}`"
        @click.stop="emit('change-group', apiKey, $event)"
      >
        <div class="flex min-w-0 items-center gap-2">
          <PlatformIcon
            v-if="apiKey.group"
            :platform="apiKey.group.platform"
            size="md"
            class="api-key-summary-card__group-icon shrink-0"
            aria-hidden="true"
          />
          <span class="api-key-summary-card__group-label truncate text-xs font-black">
            {{ apiKey.group?.name || t('keys.noGroup') }}
          </span>
        </div>
        <KeysLucideIcon
          name="chevronRight"
          :size="16"
          class="api-key-summary-card__group-chevron shrink-0"
          aria-hidden="true"
        />
      </button>

      <div class="api-key-summary-card__credential recessed-well flex h-11 items-center gap-3 rounded-xl px-4">
        <code class="min-w-0 flex-1 truncate font-mono text-[11px] text-gray-500 dark:text-gray-400">
          {{ revealKey ? apiKey.key : maskApiKey(apiKey.key) }}
        </code>
        <div class="flex">
          <button
            type="button"
            class="flex h-11 w-11 items-center justify-center rounded-lg text-gray-400 transition-colors active:text-primary-600 focus-visible:bg-primary-50 focus-visible:outline-none dark:focus-visible:bg-primary-900/20"
            :title="revealKey ? t('keys.hideKey') : t('keys.revealKey')"
            :aria-label="revealKey ? t('keys.hideKey') : t('keys.revealKey')"
            :aria-pressed="revealKey"
            :data-test="`key-summary-reveal-${apiKey.id}`"
            @click.stop="revealKey = !revealKey"
          >
            <KeysLucideIcon :name="revealKey ? 'eyeOff' : 'eye'" :size="18" />
          </button>
          <button
            type="button"
            :class="[
              'flex h-11 w-11 items-center justify-center rounded-lg text-gray-400 transition-colors active:text-primary-600 focus-visible:bg-primary-50 focus-visible:outline-none dark:focus-visible:bg-primary-900/20',
              copied && 'text-primary-600 dark:text-primary-400'
            ]"
            :title="copied ? t('keys.copied') : t('keys.copyToClipboard')"
            :aria-label="copied ? t('keys.copied') : t('keys.copyToClipboard')"
            :data-test="`key-summary-copy-${apiKey.id}`"
            @click.stop="emit('copy-key', apiKey)"
          >
            <KeysLucideIcon :name="copied ? 'check' : 'copy'" :size="18" />
          </button>
        </div>
      </div>
    </div>

    <div class="space-y-4">
      <div class="space-y-1.5" :aria-label="t('keys.quota')">
        <div class="flex justify-between gap-3 text-[10px] font-black uppercase">
          <span class="min-w-0 text-gray-400">
            {{ t('keys.workspaceQuotaProgress') }}
            <template v-if="quotaValue > 0">({{ quotaPercent.toFixed(1) }}%)</template>
          </span>
          <CreditAmount
            v-if="quotaValue > 0"
            class="shrink-0 text-gray-700 dark:text-gray-300"
            :value="`${quotaUsedValue.toFixed(1)} / ${quotaValue.toFixed(1)}`"
            icon-size="xs"
          />
          <span v-else class="shrink-0 text-gray-700 dark:text-gray-300">
            {{ t('keys.unlimitedQuota') }}
          </span>
        </div>
        <div
          v-if="quotaValue > 0"
          class="h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-white/5"
          role="progressbar"
          :aria-label="t('keys.workspaceQuotaProgress')"
          aria-valuemin="0"
          aria-valuemax="100"
          :aria-valuenow="Math.round(quotaPercent)"
        >
          <div
            class="h-full bg-primary-500 transition-[width] duration-300 motion-reduce:transition-none"
            :style="{ width: `${quotaPercent}%` }"
          />
        </div>
      </div>

      <div class="grid grid-cols-2 gap-x-6 gap-y-3 pt-1">
        <div class="flex min-w-0 flex-col gap-0.5">
          <span class="text-[9px] font-black uppercase text-gray-400">
            {{ t('keys.workspaceCardTodayUsage') }}
          </span>
          <CreditAmount
            v-if="hasUsage"
            class="text-xs font-black text-gray-900 dark:text-white"
            :value="todayCost.toFixed(2)"
            icon-size="xs"
          />
          <span v-else class="text-xs font-black text-gray-900 dark:text-white" :aria-label="t('common.notAvailable')">—</span>
        </div>
        <div class="flex min-w-0 flex-col items-end gap-0.5">
          <span class="text-right text-[9px] font-black uppercase text-gray-400">
            {{ t('keys.workspaceCardThirtyDayUsage') }}
          </span>
          <CreditAmount
            v-if="hasUsage"
            class="justify-end text-xs font-black text-gray-900 dark:text-white"
            :value="totalCost.toFixed(2)"
            icon-size="xs"
          />
          <span v-else class="text-xs font-black text-gray-900 dark:text-white" :aria-label="t('common.notAvailable')">—</span>
        </div>
        <div v-if="isVisible('current_concurrency')" class="flex min-w-0 flex-col gap-0.5">
          <span class="text-[9px] font-black uppercase text-gray-400">
            {{ t('keys.currentConcurrency') }}
          </span>
          <span class="text-xs font-black text-gray-900 dark:text-white">
            {{ finiteNumber(apiKey.current_concurrency) }} / {{ t('common.unlimited') }}
          </span>
        </div>
        <div
          v-if="isVisible('expires_at')"
          class="flex min-w-0 flex-col items-end gap-0.5"
        >
          <span class="text-right text-[9px] font-black uppercase text-gray-400">
            {{ t('keys.expiresAt') }}
          </span>
          <span
            :class="[
              'max-w-full truncate text-xs font-black uppercase text-gray-900 dark:text-white',
              isExpired && 'text-red-600 dark:text-red-400'
            ]"
            :title="expiryLabel"
          >
            {{ expiryLabel }}
          </span>
        </div>
      </div>
    </div>

    <button
      type="button"
      class="api-key-summary-card__details transition-clay flex h-11 w-full items-center justify-center gap-2 rounded-xl bg-primary-50 text-sm font-black text-primary-700 active:scale-95 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:bg-primary-900/10 dark:text-primary-400"
      :data-test="`key-summary-open-details-${apiKey.id}`"
      @click.stop="emit('open-details', apiKey)"
    >
      <span>{{ t('keys.viewDetailsAndActions') }}</span>
      <KeysLucideIcon name="chevronRight" :size="16" aria-hidden="true" />
    </button>
  </article>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import CreditAmount from '@/components/common/CreditAmount.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import KeysLucideIcon from '@/components/keys/KeysLucideIcon.vue'
import type { BatchApiKeyUsageStats } from '@/api/usage'
import type { ApiKey } from '@/types'
import { formatDateTime } from '@/utils/format'
import { maskApiKey } from '@/utils/maskApiKey'

interface Props {
  apiKey: ApiKey
  usage?: BatchApiKeyUsageStats
  userGroupRate?: number | null
  copied?: boolean
  statusUpdating?: boolean
  now?: Date
  visibleColumns?: string[]
}

const props = withDefaults(defineProps<Props>(), {
  usage: undefined,
  userGroupRate: null,
  copied: false,
  statusUpdating: false,
  now: () => new Date(),
  visibleColumns: undefined
})

const emit = defineEmits<{
  (event: 'open-details', key: ApiKey): void
  (event: 'copy-key', key: ApiKey): void
  (event: 'toggle-status', key: ApiKey): void
  (event: 'change-group', key: ApiKey, mouseEvent: MouseEvent): void
}>()

const { t } = useI18n()
const revealKey = ref(false)
const isVisible = (key: string) => !props.visibleColumns || props.visibleColumns.includes(key)

const finiteNumber = (value: unknown) =>
  typeof value === 'number' && Number.isFinite(value) ? value : 0

const isActive = computed(() => props.apiKey.status === 'active')
const isExpired = computed(() => Boolean(
  props.apiKey.expires_at && new Date(props.apiKey.expires_at).getTime() < props.now.getTime()
))
const hasUsage = computed(() => props.usage !== undefined)
const todayCost = computed(() => finiteNumber(props.usage?.today_actual_cost))
const totalCost = computed(() => finiteNumber(props.usage?.total_actual_cost))
const quotaValue = computed(() => finiteNumber(props.apiKey.quota))
const quotaUsedValue = computed(() => finiteNumber(props.apiKey.quota_used))
const quotaPercent = computed(() => {
  if (quotaValue.value <= 0) return 0
  const remaining = Math.max(quotaValue.value - quotaUsedValue.value, 0)
  return Math.min(Math.max((remaining / quotaValue.value) * 100, 0), 100)
})
const expiryLabel = computed(() => (
  props.apiKey.expires_at ? formatDateTime(props.apiKey.expires_at) : t('keys.noExpiration')
))
const statusMeta = computed(() => {
  switch (props.apiKey.status) {
    case 'active':
      return { textClass: 'text-emerald-600' }
    case 'quota_exhausted':
      return { textClass: 'text-amber-600 dark:text-amber-400' }
    case 'expired':
      return { textClass: 'text-red-600 dark:text-red-400' }
    default:
      return { textClass: 'text-gray-400' }
  }
})
</script>

<style scoped>
.api-key-summary-card {
  min-width: 0;
  color: #332f3a;
  font-family: var(
    --lx-clay-font-ui,
    "DM Sans",
    "PingFang SC",
    "Microsoft YaHei",
    system-ui,
    sans-serif
  );
  font-variant-numeric: tabular-nums;
}

.api-key-summary-card__credential {
  background-color: #f9f8fd;
  box-shadow:
    inset 6px 6px 12px rgb(91 80 112 / 0.08),
    inset -6px -6px 12px rgb(255 255 255 / 0.7);
}

.api-key-summary-card__group {
  color: var(--summary-group-foreground);
  border-color: var(--summary-group-border);
  background: var(--summary-group-background);
}

.api-key-summary-card__group--assigned {
  --summary-group-background: rgb(237 233 254 / 45%);
  --summary-group-border: #ddd6fe;
  --summary-group-foreground: #5b21b6;
  --summary-group-icon: #7c3aed;
  --summary-group-accent: #a78bfa;
}

.api-key-summary-card__group--unassigned {
  --summary-group-background: #f3f4f6;
  --summary-group-border: #e5e7eb;
  --summary-group-foreground: #4b5563;
  --summary-group-icon: #6b7280;
  --summary-group-accent: #9ca3af;
}

.api-key-summary-card__group--openai {
  --summary-group-background: #ecfdf5;
  --summary-group-border: #d1fae5;
  --summary-group-foreground: #065f46;
  --summary-group-icon: #059669;
  --summary-group-accent: #34d399;
}

.api-key-summary-card__group--anthropic {
  --summary-group-background: #fffbeb;
  --summary-group-border: #fde68a;
  --summary-group-foreground: #92400e;
  --summary-group-icon: #d97706;
  --summary-group-accent: #f59e0b;
}

.api-key-summary-card__group--gemini {
  --summary-group-background: #f0f9ff;
  --summary-group-border: #bae6fd;
  --summary-group-foreground: #075985;
  --summary-group-icon: #0284c7;
  --summary-group-accent: #38bdf8;
}

.api-key-summary-card__group--antigravity {
  --summary-group-background: #fdf4ff;
  --summary-group-border: #f5d0fe;
  --summary-group-foreground: #86198f;
  --summary-group-icon: #c026d3;
  --summary-group-accent: #e879f9;
}

.api-key-summary-card__group--grok {
  --summary-group-background: #f4f4f5;
  --summary-group-border: #e4e4e7;
  --summary-group-foreground: #3f3f46;
  --summary-group-icon: #52525b;
  --summary-group-accent: #a1a1aa;
}

.api-key-summary-card__group-icon {
  color: var(--summary-group-icon);
}

.api-key-summary-card__group-chevron {
  color: var(--summary-group-accent);
}

.api-key-summary-card__details {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

:global(.dark) .api-key-summary-card {
  color: #f8f5fc;
}

:global(.dark) .api-key-summary-card__credential {
  background-color: #120f18;
  box-shadow:
    inset 4px 4px 10px rgb(0 0 0 / 0.3),
    inset -4px -4px 10px rgb(255 255 255 / 0.02);
}

:global(.dark) .api-key-summary-card__group--assigned {
  --summary-group-background: rgb(76 29 149 / 18%);
  --summary-group-border: rgb(139 92 246 / 30%);
  --summary-group-foreground: #c4b5fd;
  --summary-group-icon: #a78bfa;
  --summary-group-accent: #a78bfa;
}

:global(.dark) .api-key-summary-card__group--unassigned {
  --summary-group-background: rgb(255 255 255 / 5%);
  --summary-group-border: rgb(255 255 255 / 10%);
  --summary-group-foreground: #d1d5db;
  --summary-group-icon: #9ca3af;
  --summary-group-accent: #9ca3af;
}

:global(.dark) .api-key-summary-card__group--openai {
  --summary-group-background: rgb(6 78 59 / 20%);
  --summary-group-border: rgb(16 185 129 / 30%);
  --summary-group-foreground: #6ee7b7;
  --summary-group-icon: #34d399;
  --summary-group-accent: #34d399;
}

:global(.dark) .api-key-summary-card__group--anthropic {
  --summary-group-background: rgb(120 53 15 / 20%);
  --summary-group-border: rgb(245 158 11 / 30%);
  --summary-group-foreground: #fcd34d;
  --summary-group-icon: #fbbf24;
  --summary-group-accent: #fbbf24;
}

:global(.dark) .api-key-summary-card__group--gemini {
  --summary-group-background: rgb(7 89 133 / 20%);
  --summary-group-border: rgb(14 165 233 / 30%);
  --summary-group-foreground: #7dd3fc;
  --summary-group-icon: #38bdf8;
  --summary-group-accent: #38bdf8;
}

:global(.dark) .api-key-summary-card__group--antigravity {
  --summary-group-background: rgb(112 26 117 / 20%);
  --summary-group-border: rgb(217 70 239 / 30%);
  --summary-group-foreground: #f0abfc;
  --summary-group-icon: #e879f9;
  --summary-group-accent: #e879f9;
}

:global(.dark) .api-key-summary-card__group--grok {
  --summary-group-background: rgb(63 63 70 / 50%);
  --summary-group-border: rgb(161 161 170 / 30%);
  --summary-group-foreground: #e4e4e7;
  --summary-group-icon: #d4d4d8;
  --summary-group-accent: #a1a1aa;
}

@media (prefers-reduced-motion: reduce) {
  .api-key-summary-card__details {
    transition-duration: 0.01ms;
  }
}
</style>

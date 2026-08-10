<template>
  <div
    class="key-workspace-list"
    data-test="api-key-workspace-list"
  >
    <table
      :class="['workspace-table', compact && 'workspace-table--compact']"
      role="grid"
      :aria-label="t('keys.title')"
    >
      <colgroup>
        <col class="workspace-col-status" />
        <col class="workspace-col-key" />
        <col class="workspace-col-group" />
        <col class="workspace-col-quota" />
        <col class="workspace-col-usage" />
      </colgroup>

      <thead>
        <tr>
          <th scope="col"><span class="sr-only">{{ t('common.status') }}</span></th>
          <th scope="col">{{ t('keys.workspaceKeyInfo') }}</th>
          <th scope="col">{{ t('keys.workspaceAssignedGroup') }}</th>
          <th scope="col">{{ t('keys.workspaceAvailableQuota') }}</th>
          <th scope="col">{{ t('keys.usage') }} (Today)</th>
        </tr>
      </thead>

      <tbody>
        <tr
          v-for="row in apiKeys"
          :key="row.id"
          :class="[
            'workspace-row cursor-pointer outline-none',
            isSelected(row.id) && 'workspace-row--selected'
          ]"
          tabindex="0"
          :aria-selected="isSelected(row.id)"
          :data-key-id="row.id"
          :data-test="`api-key-workspace-row-${row.id}`"
          @click="emit('select', row)"
          @keydown.enter.self.prevent="emit('select', row)"
          @keydown.space.self.prevent="emit('select', row)"
        >
          <td class="workspace-status-cell">
            <button
              type="button"
              role="switch"
              :aria-checked="row.status === 'active'"
              :aria-busy="isStatusUpdating(row.id)"
              :aria-label="`${row.name} · ${row.status === 'active' ? t('keys.disable') : t('keys.enable')}`"
              :title="t(`keys.status.${row.status}`)"
              :disabled="isStatusUpdating(row.id)"
              :data-test="`key-workspace-status-switch-${row.id}`"
              class="workspace-status-switch"
              @click.stop="emit('toggle-status', row)"
            >
              <span
                :class="[
                  'workspace-status-track',
                  row.status === 'active' && 'workspace-status-track--active'
                ]"
                aria-hidden="true"
              >
                <span
                  :class="['workspace-status-thumb', row.status === 'active' && 'workspace-status-thumb--active']"
                />
              </span>
            </button>
          </td>

          <td class="workspace-key-cell">
            <div class="workspace-key-copy">
              <div class="workspace-key-name" :title="row.name">
                {{ row.name }}
              </div>
              <code class="workspace-key-token">{{ maskApiKey(row.key) }}</code>
              <!-- Kept for the existing event contract; desktop copy/reveal controls live in the inspector. -->
              <button
                type="button"
                class="sr-only"
                tabindex="-1"
                aria-hidden="true"
                :title="copiedKeyId === row.id ? t('keys.copied') : t('keys.copyToClipboard')"
                :data-test="`key-workspace-copy-${row.id}`"
                @click.stop="emit('copy-key', row)"
              />
            </div>
          </td>

          <td class="workspace-group-cell">
            <button
              type="button"
              :class="[
                'workspace-group-button',
                row.group ? 'workspace-group-button--assigned' : 'workspace-group-button--unassigned',
                row.group && `workspace-group-button--${row.group.platform}`
              ]"
              :data-platform="row.group?.platform"
              :title="t('keys.clickToChangeGroup')"
              :aria-label="`${row.name} · ${t('keys.clickToChangeGroup')}`"
              @click.stop="emit('change-group', row, $event)"
            >
              <PlatformIcon
                v-if="row.group"
                :platform="row.group.platform"
                size="xs"
                aria-hidden="true"
              />
              <span>{{ row.group?.name || t('keys.workspaceUnassignedGroup') }}</span>
              <KeysLucideIcon v-if="row.group" name="chevronDown" :size="12" class="workspace-group-chevron" aria-hidden="true" />
            </button>
          </td>

          <td class="workspace-quota-cell">
            <div v-if="quotaValue(row) > 0" class="workspace-quota">
              <div class="workspace-quota-labels">
                <span class="workspace-quota-percent">{{ remainingPercent(row).toFixed(1) }}%</span>
                <span class="workspace-quota-values">
                  <CreditAmount :value="formatQuotaAmount(remainingQuota(row))" icon-size="xs" />
                  <span aria-hidden="true">/</span>
                  <CreditAmount :value="formatQuotaAmount(quotaValue(row))" icon-size="xs" />
                </span>
              </div>
              <div
                class="workspace-quota-progress"
                role="progressbar"
                :aria-label="`${row.name} · ${t('keys.quota')}`"
                aria-valuemin="0"
                aria-valuemax="100"
                :aria-valuenow="Math.round(remainingPercent(row))"
              >
                <div
                  class="workspace-quota-meter"
                  :style="{ width: `${remainingPercent(row)}%` }"
                />
              </div>
            </div>
            <span v-else class="workspace-quota-unlimited">
              {{ t('keys.unlimitedQuota') }}
            </span>
          </td>

          <td class="workspace-usage-cell">
            <template v-if="usageFor(row)">
              <CreditAmount
                class="workspace-usage-today"
                :value="usageFor(row)!.today.toFixed(2)"
                icon-size="xs"
              />
              <div class="workspace-usage-total">
                <span>{{ t('keys.workspaceThirtyDayShort') }}:</span>
                <CreditAmount :value="usageFor(row)!.total.toFixed(2)" icon-size="xs" />
              </div>
            </template>
            <span v-else class="workspace-usage-empty">&mdash;</span>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import CreditAmount from '@/components/common/CreditAmount.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import KeysLucideIcon from '@/components/keys/KeysLucideIcon.vue'
import type { BatchApiKeyUsageStats } from '@/api/usage'
import type { ApiKey } from '@/types'
import { maskApiKey } from '@/utils/maskApiKey'

interface Props {
  apiKeys: ApiKey[]
  usageStats: Record<string, BatchApiKeyUsageStats>
  userGroupRates: Record<number, number>
  selectedKeyId: number | null
  copiedKeyId?: number | null
  statusUpdatingIds?: number[]
  compact?: boolean
  now?: Date
}

const props = withDefaults(defineProps<Props>(), {
  copiedKeyId: null,
  statusUpdatingIds: () => [],
  compact: false,
  now: () => new Date()
})

const emit = defineEmits<{
  (event: 'select', key: ApiKey): void
  (event: 'copy-key', key: ApiKey): void
  (event: 'toggle-status', key: ApiKey): void
  (event: 'change-group', key: ApiKey, mouseEvent: MouseEvent): void
}>()

const { t } = useI18n()

const finiteNumber = (value: unknown) =>
  typeof value === 'number' && Number.isFinite(value) ? value : 0

const isSelected = (id: number) => props.selectedKeyId === id
const isStatusUpdating = (id: number) => props.statusUpdatingIds.includes(id)

const quotaValue = (key: ApiKey) => finiteNumber(key.quota)
const quotaUsedValue = (key: ApiKey) => finiteNumber(key.quota_used)
const remainingQuota = (key: ApiKey) => Math.max(quotaValue(key) - quotaUsedValue(key), 0)
const formatQuotaAmount = (value: number) => String(Number(value.toFixed(2)))
const remainingPercent = (key: ApiKey) => {
  if (quotaValue(key) <= 0) return 0
  return Math.min(Math.max((remainingQuota(key) / quotaValue(key)) * 100, 0), 100)
}

const usageFor = (key: ApiKey) => {
  const usage = props.usageStats[String(key.id)]
  if (!usage) return null
  return {
    today: finiteNumber(usage.today_actual_cost),
    total: finiteNumber(usage.total_actual_cost)
  }
}
</script>

<style scoped>
.key-workspace-list {
  width: 100%;
  height: 100%;
  max-width: 100%;
  overflow: auto;
  background: var(--workspace-card-surface);
  scrollbar-color: var(--workspace-border-strong) transparent;
  scrollbar-width: thin;
}

.key-workspace-list::-webkit-scrollbar {
  width: 5px;
  height: 5px;
}

.key-workspace-list::-webkit-scrollbar-thumb {
  border-radius: 10px;
  background: var(--workspace-border-strong);
}

.workspace-table {
  width: 100%;
  border-collapse: collapse;
  color: var(--workspace-text);
  text-align: left;
  font-variant-numeric: tabular-nums;
}

.workspace-col-status { width: 10.5%; }
.workspace-col-key { width: 23.5%; }
.workspace-col-group { width: 23.5%; }
.workspace-col-quota { width: 25.5%; }
.workspace-col-usage { width: 17%; }

.workspace-table thead {
  position: sticky;
  top: 0;
  z-index: 10;
  border-bottom: 1px solid var(--workspace-border);
  background: color-mix(in srgb, var(--workspace-card-surface) 95%, transparent);
  backdrop-filter: blur(4px);
  -webkit-backdrop-filter: blur(4px);
}

.workspace-table th {
  height: 3.5rem;
  padding: 1.25rem 1rem;
  color: var(--workspace-text-muted);
  font-size: 0.625rem;
  font-weight: 900;
  letter-spacing: 0;
  line-height: 1rem;
  text-transform: uppercase;
  white-space: nowrap;
}

.workspace-table th:first-child {
  padding-left: 2rem;
}

.workspace-table th:last-child {
  padding-right: 2rem;
  text-align: right;
}

.workspace-row {
  height: 5.75rem;
  background: var(--workspace-card-surface);
  transition: background-color 300ms cubic-bezier(0.4, 0, 0.2, 1);
}

.workspace-table tbody tr + tr {
  border-top: 1px solid var(--workspace-border);
}

.workspace-row:hover {
  background: var(--workspace-hover);
}

.workspace-row--selected,
.workspace-row--selected:hover {
  background: var(--workspace-selected);
}

.workspace-row:focus-visible {
  outline: 2px solid #7c3aed;
  outline-offset: -2px;
}

.workspace-table td {
  padding: 1.5rem 1rem;
  vertical-align: middle;
}

.workspace-status-cell {
  padding-left: 2rem !important;
  padding-right: 0 !important;
}

.workspace-row--selected > .workspace-status-cell {
  box-shadow: inset 4px 0 0 var(--workspace-border-strong);
}

.workspace-usage-cell {
  padding-right: 2rem !important;
  text-align: right;
}

.workspace-status-switch {
  position: relative;
  display: flex;
  width: 2.75rem;
  height: 2.75rem;
  align-items: center;
  justify-content: center;
  border-radius: 9999px;
  outline: none;
}

.workspace-status-switch:focus-visible,
.workspace-group-button:focus-visible {
  outline: 2px solid #8b5cf6;
  outline-offset: 2px;
}

.workspace-status-switch:disabled {
  cursor: wait;
  opacity: 0.6;
}

.workspace-status-track {
  position: relative;
  display: block;
  width: 2.5rem;
  height: 1.375rem;
  border-radius: 9999px;
  background: var(--workspace-border-strong);
  transition: background-color 200ms ease;
}

.workspace-status-track--active {
  background: #7c3aed;
}

.workspace-status-thumb {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 1.125rem;
  height: 1.125rem;
  border-radius: 9999px;
  background: var(--workspace-light-surface);
  box-shadow: 0 1px 2px 0 rgb(0 0 0 / 5%);
  transition: transform 200ms ease;
}

.workspace-status-thumb--active {
  transform: translateX(18px);
}

.workspace-key-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 0.25rem;
}

.workspace-key-name,
.workspace-key-token {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workspace-key-name {
  color: var(--workspace-text);
  font-size: 0.875rem;
  font-weight: 700;
  line-height: 1.25rem;
}

.workspace-key-token {
  color: var(--workspace-text-muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.6875rem;
  letter-spacing: 0;
  line-height: 1rem;
}

.workspace-group-button {
  display: inline-flex;
  max-width: 100%;
  min-height: 1.75rem;
  align-items: center;
  gap: 0.375rem;
  border-radius: 0.75rem;
  padding: 0.375rem 0.75rem;
  font-size: 0.625rem;
  line-height: 1rem;
  text-align: left;
  white-space: nowrap;
  transition: background-color 150ms ease;
}

.workspace-group-button > span {
  overflow: hidden;
  text-overflow: ellipsis;
}

.workspace-group-button--assigned {
  --workspace-group-background: rgb(237 233 254 / 45%);
  --workspace-group-background-hover: #ede9fe;
  --workspace-group-border: #ddd6fe;
  --workspace-group-foreground: #6d28d9;
  --workspace-group-accent: #a78bfa;

  border: 1px solid var(--workspace-group-border);
  color: var(--workspace-group-foreground);
  background: var(--workspace-group-background);
  font-weight: 900;
}

.workspace-group-button--assigned:hover {
  background: var(--workspace-group-background-hover);
}

.workspace-group-button--openai {
  --workspace-group-background: rgb(209 250 229 / 40%);
  --workspace-group-background-hover: #d1fae5;
  --workspace-group-border: #d1fae5;
  --workspace-group-foreground: #047857;
  --workspace-group-accent: #34d399;
}

.workspace-group-button--anthropic {
  --workspace-group-background: rgb(254 243 199 / 40%);
  --workspace-group-background-hover: #fef3c7;
  --workspace-group-border: #fde68a;
  --workspace-group-foreground: #b45309;
  --workspace-group-accent: #f59e0b;
}

.workspace-group-button--gemini {
  --workspace-group-background: rgb(224 242 254 / 50%);
  --workspace-group-background-hover: #e0f2fe;
  --workspace-group-border: #bae6fd;
  --workspace-group-foreground: #0369a1;
  --workspace-group-accent: #38bdf8;
}

.workspace-group-button--antigravity {
  --workspace-group-background: rgb(250 232 255 / 50%);
  --workspace-group-background-hover: #fae8ff;
  --workspace-group-border: #f5d0fe;
  --workspace-group-foreground: #a21caf;
  --workspace-group-accent: #e879f9;
}

.workspace-group-button--grok {
  --workspace-group-background: rgb(244 244 245 / 70%);
  --workspace-group-background-hover: #e4e4e7;
  --workspace-group-border: #e4e4e7;
  --workspace-group-foreground: #3f3f46;
  --workspace-group-accent: #a1a1aa;
}

.workspace-group-button--unassigned {
  border: 1px solid var(--workspace-border);
  color: var(--workspace-text-secondary);
  background: var(--workspace-surface-subtle);
  font-weight: 700;
}

.workspace-group-chevron {
  margin-left: 0.125rem;
  color: var(--workspace-group-accent);
}

.workspace-quota {
  width: 8rem;
  max-width: 100%;
}

.workspace-quota-labels {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0;
  margin-bottom: 0.375rem;
  font-size: 0.625rem;
  font-weight: 700;
  line-height: 1rem;
}

.workspace-quota-percent {
  flex: 0 0 auto;
  color: #7c3aed;
}

.workspace-quota-values {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.125rem;
  overflow: hidden;
  color: var(--workspace-text-muted);
  white-space: nowrap;
}

.workspace-quota-progress {
  height: 0.375rem;
  overflow: hidden;
  border-radius: 9999px;
  background: var(--workspace-surface-subtle);
}

.workspace-quota-meter {
  height: 100%;
  border-radius: 9999px;
  background: #8b5cf6;
  transition: width 300ms ease;
}

.workspace-quota-unlimited {
  color: var(--workspace-text-muted);
  font-size: 0.75rem;
  font-weight: 700;
}

.workspace-usage-today {
  justify-content: flex-end;
  color: var(--workspace-text);
  font-size: 0.875rem;
  font-weight: 900;
  line-height: 1.25rem;
}

.workspace-usage-total {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.25rem;
  margin-top: 0.125rem;
  color: var(--workspace-text-muted);
  font-size: 0.625rem;
  font-weight: 700;
  letter-spacing: 0;
  line-height: 1rem;
  text-transform: uppercase;
}

.workspace-usage-empty {
  color: var(--workspace-text-muted);
  font-size: 0.875rem;
  font-weight: 700;
}

.workspace-table--compact .workspace-row {
  height: 4.5rem;
}

.workspace-table--compact td {
  padding-top: 0.875rem;
  padding-bottom: 0.875rem;
}

:global(.dark) .key-workspace-list,
:global(.dark) .workspace-row {
  background: var(--workspace-card-surface);
}

:global(.dark) .workspace-table {
  color: var(--workspace-text);
}

:global(.dark) .workspace-table thead {
  border-bottom-color: var(--workspace-border);
  background: color-mix(in srgb, var(--workspace-card-surface) 95%, transparent);
}

:global(.dark) .workspace-table th {
  color: var(--workspace-text-muted);
}

:global(.dark) .workspace-table tbody tr + tr {
  border-top-color: var(--workspace-border);
}

:global(.dark) .workspace-row:hover {
  background: var(--workspace-hover);
}

:global(.dark) .workspace-row--selected,
:global(.dark) .workspace-row--selected:hover {
  background: var(--workspace-selected);
}

:global(.dark) .workspace-key-name,
:global(.dark) .workspace-usage-today {
  color: var(--workspace-text);
}

:global(.dark) .workspace-status-track {
  background: var(--workspace-border-strong);
}

:global(.dark) .workspace-status-track--active {
  background: #7c3aed;
}

:global(.dark) .workspace-quota-progress {
  background: var(--workspace-surface-subtle);
}

:global(.dark) .workspace-group-button--assigned {
  --workspace-group-background: rgb(76 29 149 / 18%);
  --workspace-group-background-hover: rgb(91 33 182 / 28%);
  --workspace-group-border: rgb(139 92 246 / 30%);
  --workspace-group-foreground: #c4b5fd;
  --workspace-group-accent: #a78bfa;
}

:global(.dark) .workspace-group-button--openai {
  --workspace-group-background: rgb(6 78 59 / 20%);
  --workspace-group-background-hover: rgb(6 95 70 / 32%);
  --workspace-group-border: rgb(16 185 129 / 30%);
  --workspace-group-foreground: #6ee7b7;
  --workspace-group-accent: #34d399;
}

:global(.dark) .workspace-group-button--anthropic {
  --workspace-group-background: rgb(120 53 15 / 20%);
  --workspace-group-background-hover: rgb(146 64 14 / 30%);
  --workspace-group-border: rgb(245 158 11 / 30%);
  --workspace-group-foreground: #fcd34d;
  --workspace-group-accent: #fbbf24;
}

:global(.dark) .workspace-group-button--gemini {
  --workspace-group-background: rgb(7 89 133 / 20%);
  --workspace-group-background-hover: rgb(3 105 161 / 30%);
  --workspace-group-border: rgb(14 165 233 / 30%);
  --workspace-group-foreground: #7dd3fc;
  --workspace-group-accent: #38bdf8;
}

:global(.dark) .workspace-group-button--antigravity {
  --workspace-group-background: rgb(112 26 117 / 20%);
  --workspace-group-background-hover: rgb(134 25 143 / 30%);
  --workspace-group-border: rgb(217 70 239 / 30%);
  --workspace-group-foreground: #f0abfc;
  --workspace-group-accent: #e879f9;
}

:global(.dark) .workspace-group-button--grok {
  --workspace-group-background: rgb(63 63 70 / 50%);
  --workspace-group-background-hover: rgb(82 82 91 / 60%);
  --workspace-group-border: rgb(161 161 170 / 30%);
  --workspace-group-foreground: #e4e4e7;
  --workspace-group-accent: #a1a1aa;
}

@media (prefers-reduced-motion: reduce) {
  .workspace-row,
  .workspace-status-track,
  .workspace-status-thumb,
  .workspace-group-button,
  .workspace-quota-meter {
    transition: none;
  }
}
</style>

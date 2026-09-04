<template>
  <div
    class="key-workspace-list"
    data-test="api-key-workspace-list"
  >
    <table
      :class="['workspace-table', compact && 'workspace-table--compact']"
      :aria-label="t('keys.title')"
    >
      <caption class="sr-only">{{ t('keys.title') }}</caption>
      <colgroup>
        <col class="workspace-col-name" />
        <col class="workspace-col-key" />
        <col class="workspace-col-group" />
        <col class="workspace-col-created" />
        <col class="workspace-col-last-used" />
        <col v-if="showActions" class="workspace-col-actions" />
      </colgroup>

      <thead>
        <tr>
          <th scope="col">{{ t('common.name') }}</th>
          <th scope="col">{{ t('keys.key') }}</th>
          <th scope="col">{{ t('keys.group') }}</th>
          <th scope="col">{{ t('keys.created') }}</th>
          <th scope="col">{{ t('keys.lastUsedAt') }}</th>
          <th v-if="showActions" scope="col" class="workspace-actions-heading">
            {{ t('common.actions') }}
          </th>
        </tr>
      </thead>

      <tbody>
        <tr
          v-for="row in apiKeys"
          :key="row.id"
          class="workspace-row"
          :data-key-id="row.id"
          :data-test="`api-key-workspace-row-${row.id}`"
        >
          <td class="workspace-name-cell">
            <span class="workspace-key-name" :title="row.name || undefined">
              {{ row.name || '—' }}
            </span>
          </td>

          <td class="workspace-key-cell">
            <div class="workspace-key-value">
              <code
                class="workspace-key-token"
                :title="maskApiKey(row.key)"
              >{{ maskApiKey(row.key) }}</code>
              <button
                type="button"
                class="workspace-copy-button"
                :class="isCopied(row.id) && 'workspace-copy-button--copied'"
                :title="isCopied(row.id) ? t('keys.copied') : t('keys.copyToClipboard')"
                :aria-label="isCopied(row.id) ? t('keys.copied') : t('keys.copyToClipboard')"
                :data-test="`key-workspace-copy-${row.id}`"
                @click.stop="emit('copy-key', row)"
              >
                <KeysLucideIcon
                  :name="isCopied(row.id) ? 'check' : 'copy'"
                  :size="16"
                  aria-hidden="true"
                />
              </button>
            </div>
          </td>

          <td class="workspace-group-cell">
            <button
              type="button"
              :class="[
                'workspace-group-button workspace-group-chip',
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
                class="workspace-group-icon"
                aria-hidden="true"
              />
              <span class="workspace-group-label">
                {{ row.group?.name || t('keys.workspaceUnassignedGroup') }}
              </span>
            </button>
          </td>

          <td class="workspace-time-cell">
            <time :datetime="row.created_at || undefined">
              {{ formatTableDate(row.created_at) }}
            </time>
          </td>

          <td class="workspace-time-cell">
            <time :datetime="row.last_used_at || undefined">
              {{ formatTableDate(row.last_used_at) }}
            </time>
          </td>

          <td v-if="showActions" class="workspace-actions-cell">
            <div class="workspace-action-list">
              <button
                type="button"
                class="workspace-action-button workspace-action-use"
                :title="t('keys.useKey')"
                :aria-label="t('keys.useKey')"
                :data-action="`use-key-${row.id}`"
                :data-test="`key-workspace-use-${row.id}`"
                @click.stop="emit('use-key', row)"
              >
                <KeysLucideIcon name="terminal" :size="16" aria-hidden="true" />
              </button>
              <button
                type="button"
                class="workspace-action-button workspace-action-edit"
                :title="t('common.edit')"
                :aria-label="t('common.edit')"
                :data-action="`edit-key-${row.id}`"
                :data-test="`key-workspace-edit-${row.id}`"
                @click.stop="emit('edit', row)"
              >
                <KeysLucideIcon name="penLine" :size="16" aria-hidden="true" />
              </button>
              <button
                v-if="showCcsImport"
                type="button"
                class="workspace-action-button workspace-action-import"
                :title="t('keys.workspaceImportCcs')"
                :aria-label="t('keys.workspaceImportCcs')"
                :data-action="`import-ccs-${row.id}`"
                :data-test="`key-workspace-import-${row.id}`"
                @click.stop="emit('import-ccs', row)"
              >
                <KeysLucideIcon name="uploadCloud" :size="16" aria-hidden="true" />
              </button>
              <button
                type="button"
                class="workspace-action-button workspace-action-delete"
                :title="t('common.delete')"
                :aria-label="t('common.delete')"
                :data-action="`delete-key-${row.id}`"
                :data-test="`key-workspace-delete-${row.id}`"
                @click.stop="emit('delete', row)"
              >
                <KeysLucideIcon name="trash2" :size="16" aria-hidden="true" />
              </button>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import KeysLucideIcon from '@/components/keys/KeysLucideIcon.vue'
import type { BatchApiKeyUsageStats } from '@/api/usage'
import type { ApiKey } from '@/types'
import { maskApiKey } from '@/utils/maskApiKey'

/**
 * The workspace list intentionally keeps the legacy data props optional. The
 * page used to render quota/status details here; callers can continue passing
 * those values while the compact six-column table ignores them.
 */
interface Props {
  apiKeys: ApiKey[]
  usageStats?: Record<string, BatchApiKeyUsageStats>
  userGroupRates?: Record<number, number>
  selectedKeyId?: number | null
  copiedKeyId?: number | null
  statusUpdatingIds?: number[]
  compact?: boolean
  now?: Date
  showActions?: boolean
  showCcsImport?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  usageStats: () => ({}),
  userGroupRates: () => ({}),
  selectedKeyId: null,
  copiedKeyId: null,
  statusUpdatingIds: () => [],
  compact: false,
  now: () => new Date(),
  showActions: true,
  showCcsImport: true,
})

const emit = defineEmits<{
  (event: 'copy-key', key: ApiKey): void
  (event: 'change-group', key: ApiKey, mouseEvent: MouseEvent): void
  (event: 'use-key', key: ApiKey): void
  (event: 'edit', key: ApiKey): void
  (event: 'import-ccs', key: ApiKey): void
  (event: 'delete', key: ApiKey): void
}>()

const { t } = useI18n()

const isCopied = (id: number) => props.copiedKeyId === id

/** Keep dates compact and predictable in the dense table, matching the design. */
const formatTableDate = (value: string | null | undefined): string => {
  if (!value) return '—'

  // API timestamps are ISO strings. Preserve their calendar date instead of
  // allowing a user's local timezone to move a midnight UTC value backwards.
  const isoDate = value.slice(0, 10)
  if (/^\d{4}-\d{2}-\d{2}$/.test(isoDate)) return isoDate

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'

  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

</script>

<style scoped>
.key-workspace-list {
  width: 100%;
  height: 100%;
  min-width: 0;
  overflow: auto;
  border: 1px solid var(--workspace-border);
  border-radius: 15px;
  background: var(--workspace-card-surface);
  scrollbar-color: var(--workspace-border-strong) transparent;
  scrollbar-width: thin;
}

.key-workspace-list::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

.key-workspace-list::-webkit-scrollbar-thumb {
  border-radius: 999px;
  background: var(--workspace-border-strong);
}

.workspace-table {
  width: 100%;
  min-width: 860px;
  border-collapse: collapse;
  table-layout: fixed;
  color: var(--workspace-text);
  font-variant-numeric: tabular-nums;
  text-align: left;
}

/* Keep the credential and group fields roomy; reserve a compact action rail. */
.workspace-col-name { width: 20%; }
.workspace-col-key { width: 16%; }
.workspace-col-group { width: 14%; }
.workspace-col-created { width: 15%; }
.workspace-col-last-used { width: 15%; }
.workspace-col-actions { width: 20%; }

.workspace-table thead {
  position: sticky;
  top: 0;
  z-index: 1;
  background: var(--workspace-surface-subtle);
}

.workspace-table th {
  height: 48px;
  padding: 0 16px;
  border-bottom: 1px solid var(--workspace-border);
  color: var(--workspace-text-secondary);
  font-size: 0.75rem;
  font-weight: 600;
  line-height: 1.1rem;
  white-space: nowrap;
}

.workspace-table th:first-child,
.workspace-table td:first-child {
  padding-left: 24px;
}

.workspace-table th:last-child,
.workspace-table td:last-child {
  padding-right: 24px;
}

/* Match the header's visual starts to the tighter body cells. The key and
   group columns intentionally meet more closely; dates and actions keep
   balanced gutters. */
.workspace-table th:nth-child(2) {
  padding-left: 16px !important;
  padding-right: 8px !important;
}

.workspace-table th:nth-child(3) {
  padding-left: 6px !important;
  padding-right: 16px !important;
}

.workspace-table th:nth-child(4),
.workspace-table th:nth-child(5),
.workspace-table th:last-child {
  padding-left: 12px !important;
  padding-right: 12px !important;
}

.workspace-actions-heading,
.workspace-actions-cell {
  text-align: center;
}

.workspace-row {
  height: 64px;
  background: var(--workspace-card-surface);
  transition: background-color 150ms ease;
}

.workspace-table tbody tr + tr td {
  border-top: 1px solid var(--workspace-border);
}

.workspace-table tbody tr:last-child td {
  border-bottom: 0;
}

.workspace-row:hover {
  background: var(--workspace-hover);
}

.workspace-table td {
  min-width: 0;
  padding: 0 16px;
  vertical-align: middle;
}

.workspace-key-name,
.workspace-key-token,
.workspace-group-label,
.workspace-time-cell time {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workspace-key-name {
  display: block;
  color: var(--workspace-text);
  font-size: 0.875rem;
  font-weight: 600;
  line-height: 1.25rem;
}

.workspace-key-value {
  display: flex;
  width: max-content;
  min-width: 0;
  max-width: 100%;
  align-items: center;
  gap: 4px;
}

.workspace-key-cell {
  padding-left: 16px !important;
  padding-right: 8px !important;
}

.workspace-key-token {
  display: block;
  min-width: 0;
  flex: 0 1 auto;
  max-width: calc(100% - 36px);
  color: var(--workspace-text-secondary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.75rem;
  line-height: 1.1rem;
}

.workspace-copy-button,
.workspace-action-button {
  display: inline-grid;
  place-items: center;
  flex: 0 0 auto;
  width: 32px;
  height: 32px;
  border: 1px solid transparent;
  border-radius: 8px;
  color: var(--workspace-text-muted);
  background: transparent;
  outline: none;
  transition: color 150ms ease, border-color 150ms ease, background-color 150ms ease;
}

.workspace-copy-button:hover,
.workspace-copy-button:focus-visible,
.workspace-action-button:hover,
.workspace-action-button:focus-visible {
  border-color: var(--workspace-border);
  color: var(--workspace-text);
  background: var(--workspace-hover);
}

.workspace-copy-button:focus-visible,
.workspace-action-button:focus-visible,
.workspace-group-button:focus-visible {
  outline: 2px solid var(--workspace-text-muted);
  outline-offset: 2px;
}

.workspace-copy-button--copied {
  color: var(--workspace-text-secondary);
}

.workspace-group-cell {
  /* The group is a separate field, but its chip should begin close enough
     to the key column to read as the next related value. */
  padding-left: 6px !important;
  padding-right: 12px !important;
}

.workspace-group-button {
  --group-chip-border: var(--workspace-border);
  --group-chip-text: var(--workspace-text-secondary);
  --group-chip-surface: var(--workspace-surface-subtle);
  display: inline-flex;
  min-width: 0;
  max-width: 100%;
  height: 24px;
  min-height: 24px;
  align-items: center;
  gap: 5px;
  border: 1px solid var(--group-chip-border);
  border-radius: 999px;
  padding: 0 8px;
  color: var(--group-chip-text);
  background: var(--group-chip-surface);
  font-size: 0.6875rem;
  font-weight: 500;
  line-height: 1rem;
  text-align: left;
  white-space: nowrap;
  outline: none;
  transition: border-color 150ms ease, background-color 150ms ease, color 150ms ease;
}

.workspace-group-button:hover {
  --group-chip-border: var(--workspace-border-strong);
  --group-chip-text: var(--workspace-text);
  --group-chip-surface: var(--workspace-hover);
}

/* Keep provider identity visible without making the table visually noisy. */
.workspace-group-button--assigned,
.workspace-group-button--unassigned {
  --group-chip-border: var(--workspace-border);
  --group-chip-text: var(--workspace-text-secondary);
  --group-chip-surface: var(--workspace-surface-subtle);
}

.workspace-group-button--openai {
  --group-chip-border: #bbf7d0;
  --group-chip-text: #15803d;
  --group-chip-surface: #f0fdf4;
}

.workspace-group-button--anthropic {
  --group-chip-border: #fed7aa;
  --group-chip-text: #c2410c;
  --group-chip-surface: #fff7ed;
}

.workspace-group-button--gemini {
  --group-chip-border: #bfdbfe;
  --group-chip-text: #1d4ed8;
  --group-chip-surface: #eff6ff;
}

.workspace-group-button--antigravity {
  --group-chip-border: #e9d5ff;
  --group-chip-text: #7e22ce;
  --group-chip-surface: #faf5ff;
}

.workspace-group-button--grok {
  --group-chip-border: #d4d4d8;
  --group-chip-text: #3f3f46;
  --group-chip-surface: #f4f4f5;
}

.workspace-group-button--deepseek {
  --group-chip-border: #bae6fd;
  --group-chip-text: #0369a1;
  --group-chip-surface: #f0f9ff;
}

/* The provider tone remains legible in the dark workspace theme. */
:global(html.dark .workspace-group-button--openai) {
  --group-chip-border: rgb(74 222 128 / 0.34);
  --group-chip-text: #86efac;
  --group-chip-surface: rgb(20 83 45 / 0.34);
}

:global(html.dark .workspace-group-button--anthropic) {
  --group-chip-border: rgb(251 146 60 / 0.34);
  --group-chip-text: #fdba74;
  --group-chip-surface: rgb(124 45 18 / 0.32);
}

:global(html.dark .workspace-group-button--gemini) {
  --group-chip-border: rgb(96 165 250 / 0.34);
  --group-chip-text: #93c5fd;
  --group-chip-surface: rgb(30 58 138 / 0.34);
}

:global(html.dark .workspace-group-button--antigravity) {
  --group-chip-border: rgb(192 132 252 / 0.34);
  --group-chip-text: #d8b4fe;
  --group-chip-surface: rgb(88 28 135 / 0.34);
}

:global(html.dark .workspace-group-button--grok) {
  --group-chip-border: rgb(161 161 170 / 0.34);
  --group-chip-text: #d4d4d8;
  --group-chip-surface: rgb(63 63 70 / 0.48);
}

:global(html.dark .workspace-group-button--deepseek) {
  --group-chip-border: rgb(56 189 248 / 0.34);
  --group-chip-text: #7dd3fc;
  --group-chip-surface: rgb(8 47 73 / 0.38);
}

.workspace-group-button--assigned:hover,
.workspace-group-button--unassigned:hover {
  --group-chip-border: var(--workspace-border-strong);
  --group-chip-text: var(--workspace-text);
  --group-chip-surface: var(--workspace-hover);
}

.workspace-group-button--openai:hover {
  --group-chip-border: #86efac;
  --group-chip-text: #166534;
  --group-chip-surface: #dcfce7;
}

.workspace-group-button--anthropic:hover {
  --group-chip-border: #fdba74;
  --group-chip-text: #9a3412;
  --group-chip-surface: #ffedd5;
}

.workspace-group-button--gemini:hover {
  --group-chip-border: #93c5fd;
  --group-chip-text: #1e40af;
  --group-chip-surface: #dbeafe;
}

.workspace-group-button--antigravity:hover {
  --group-chip-border: #d8b4fe;
  --group-chip-text: #6b21a8;
  --group-chip-surface: #f3e8ff;
}

.workspace-group-button--grok:hover {
  --group-chip-border: #a1a1aa;
  --group-chip-text: #27272a;
  --group-chip-surface: #e4e4e7;
}

.workspace-group-button--deepseek:hover {
  --group-chip-border: #7dd3fc;
  --group-chip-text: #075985;
  --group-chip-surface: #e0f2fe;
}

:global(html.dark .workspace-group-button--openai:hover) {
  --group-chip-border: rgb(134 239 172 / 0.52);
  --group-chip-text: #bbf7d0;
  --group-chip-surface: rgb(22 101 52 / 0.5);
}

:global(html.dark .workspace-group-button--anthropic:hover) {
  --group-chip-border: rgb(253 186 116 / 0.52);
  --group-chip-text: #fed7aa;
  --group-chip-surface: rgb(154 52 18 / 0.5);
}

:global(html.dark .workspace-group-button--gemini:hover) {
  --group-chip-border: rgb(147 197 253 / 0.52);
  --group-chip-text: #bfdbfe;
  --group-chip-surface: rgb(30 64 175 / 0.5);
}

:global(html.dark .workspace-group-button--antigravity:hover) {
  --group-chip-border: rgb(216 180 254 / 0.52);
  --group-chip-text: #e9d5ff;
  --group-chip-surface: rgb(107 33 168 / 0.5);
}

:global(html.dark .workspace-group-button--grok:hover) {
  --group-chip-border: rgb(212 212 216 / 0.52);
  --group-chip-text: #f4f4f5;
  --group-chip-surface: rgb(82 82 91 / 0.64);
}

:global(html.dark .workspace-group-button--deepseek:hover) {
  --group-chip-border: rgb(125 211 252 / 0.52);
  --group-chip-text: #bae6fd;
  --group-chip-surface: rgb(7 89 133 / 0.5);
}

.workspace-group-label {
  min-width: 0;
}

.workspace-time-cell {
  overflow: hidden;
  padding-left: 12px !important;
  padding-right: 12px !important;
  color: var(--workspace-text-secondary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.6875rem;
  line-height: 1rem;
  white-space: nowrap;
}

.workspace-action-list {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
}

.workspace-actions-cell {
  padding-right: 12px !important;
  padding-left: 12px !important;
}

.workspace-table--compact .workspace-row {
  height: 56px;
}

.workspace-table--compact td {
  padding-top: 0.5rem;
  padding-bottom: 0.5rem;
}

@media (max-width: 960px) {
  .workspace-table {
    min-width: 820px;
  }

  .workspace-table th:first-child,
  .workspace-table td:first-child {
    padding-left: 18px;
  }

  .workspace-table th:last-child,
  .workspace-table td:last-child {
    padding-right: 18px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .workspace-row,
  .workspace-copy-button,
  .workspace-action-button,
  .workspace-group-button {
    transition: none;
  }
}
</style>

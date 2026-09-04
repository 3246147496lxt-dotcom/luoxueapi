<template>
  <article
    class="api-key-summary-card"
    :data-key-id="apiKey.id"
    :data-test="`api-key-summary-card-${apiKey.id}`"
  >
    <div class="api-key-summary-card__head">
      <h3
        class="api-key-summary-card__name"
        :title="apiKey.name"
      >
        {{ apiKey.name }}
      </h3>
      <!-- Keep actions independent and visible on the card itself. -->
      <div
        class="api-key-summary-card__actions"
        role="group"
        :aria-label="`${apiKey.name} · ${t('common.actions')}`"
      >
        <button
          type="button"
          class="api-key-summary-card__action"
          :data-test="`key-summary-use-${apiKey.id}`"
          :aria-label="t('keys.useKey')"
          :title="t('keys.useKey')"
          @click.stop="emit('use-key', apiKey)"
        >
          <KeysLucideIcon name="terminal" :size="17" aria-hidden="true" />
        </button>
        <button
          type="button"
          class="api-key-summary-card__action"
          :data-test="`key-summary-edit-${apiKey.id}`"
          :aria-label="t('common.edit')"
          :title="t('common.edit')"
          @click.stop="emit('edit', apiKey)"
        >
          <KeysLucideIcon name="penLine" :size="17" aria-hidden="true" />
        </button>
        <button
          v-if="showCcsImport"
          type="button"
          class="api-key-summary-card__action"
          :data-test="`key-summary-import-${apiKey.id}`"
          :aria-label="t('keys.workspaceImportCcs')"
          :title="t('keys.workspaceImportCcs')"
          @click.stop="emit('import-ccs', apiKey)"
        >
          <KeysLucideIcon name="uploadCloud" :size="17" aria-hidden="true" />
        </button>
        <button
          type="button"
          class="api-key-summary-card__action"
          :data-test="`key-summary-delete-${apiKey.id}`"
          :aria-label="t('common.delete')"
          :title="t('common.delete')"
          @click.stop="emit('delete', apiKey)"
        >
          <KeysLucideIcon name="trash2" :size="17" aria-hidden="true" />
        </button>
      </div>
    </div>

    <div class="api-key-summary-card__field">
      <span class="api-key-summary-card__label">{{ t('keys.key') }}</span>
      <div class="api-key-summary-card__credential">
        <code :title="maskApiKey(apiKey.key)">{{ maskApiKey(apiKey.key) }}</code>
        <button
          type="button"
          class="api-key-summary-card__copy"
          :class="copied && 'is-copied'"
          :data-test="`key-summary-copy-${apiKey.id}`"
          :aria-label="copied ? t('keys.copied') : t('keys.copyToClipboard')"
          :title="copied ? t('keys.copied') : t('keys.copyToClipboard')"
          @click.stop="emit('copy-key', apiKey)"
        >
          <KeysLucideIcon :name="copied ? 'check' : 'copy'" :size="17" aria-hidden="true" />
        </button>
      </div>
    </div>

    <div class="api-key-summary-card__field api-key-summary-card__group-field">
      <span class="api-key-summary-card__label">{{ t('keys.group') }}</span>
      <button
        type="button"
        :class="[
          'api-key-summary-card__group',
          apiKey.group ? 'api-key-summary-card__group--assigned' : 'api-key-summary-card__group--unassigned',
          apiKey.group && `api-key-summary-card__group--${apiKey.group.platform}`
        ]"
        :data-platform="apiKey.group?.platform"
        :title="t('keys.clickToChangeGroup')"
        :aria-label="`${t('keys.group')}: ${apiKey.group?.name || t('keys.noGroup')}`"
        @click.stop="emit('change-group', apiKey, $event)"
      >
        <PlatformIcon
          v-if="apiKey.group"
          :platform="apiKey.group.platform"
          size="md"
          class="api-key-summary-card__group-icon"
          aria-hidden="true"
        />
        <span class="api-key-summary-card__group-name">
          {{ apiKey.group?.name || t('keys.noGroup') }}
        </span>
      </button>
    </div>

    <div class="api-key-summary-card__time-grid">
      <div class="api-key-summary-card__time-item">
        <span class="api-key-summary-card__label">{{ t('keys.created') }}</span>
        <time
          class="api-key-summary-card__time-value"
          :datetime="apiKey.created_at"
        >
          {{ formatKeyDate(apiKey.created_at) }}
        </time>
      </div>
      <div class="api-key-summary-card__time-item">
        <span class="api-key-summary-card__label">{{ t('keys.lastUsedAt') }}</span>
        <time
          v-if="apiKey.last_used_at"
          class="api-key-summary-card__time-value"
          :datetime="apiKey.last_used_at"
        >
          {{ formatKeyDate(apiKey.last_used_at) }}
        </time>
        <span v-else class="api-key-summary-card__time-value" aria-label="—">—</span>
      </div>
    </div>

  </article>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import type { BatchApiKeyUsageStats } from '@/api/usage'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import KeysLucideIcon from '@/components/keys/KeysLucideIcon.vue'
import type { ApiKey } from '@/types'
import { maskApiKey } from '@/utils/maskApiKey'

interface Props {
  apiKey: ApiKey
  // Compatibility props accepted by the previous card API. The simplified
  // visual intentionally does not render usage, quota, status, or tier data.
  usage?: BatchApiKeyUsageStats
  userGroupRate?: number | null
  copied?: boolean
  statusUpdating?: boolean
  now?: Date
  visibleColumns?: string[]
  showCcsImport?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  usage: undefined,
  userGroupRate: null,
  copied: false,
  statusUpdating: false,
  now: () => new Date(),
  visibleColumns: undefined,
  showCcsImport: true
})

const emit = defineEmits<{
  (event: 'copy-key', key: ApiKey): void
  (event: 'toggle-status', key: ApiKey): void
  (event: 'change-group', key: ApiKey, mouseEvent: MouseEvent): void
  (event: 'use-key', key: ApiKey): void
  (event: 'edit', key: ApiKey): void
  (event: 'import-ccs', key: ApiKey): void
  (event: 'delete', key: ApiKey): void
}>()

// Keep compatibility props in the component contract for callers that still
// pass them during the KeysView migration.
void props

const { t } = useI18n()

const formatKeyDate = (value: string | null | undefined): string => {
  if (!value) return '—'
  const isoDate = value.slice(0, 10)
  if (/^\d{4}-\d{2}-\d{2}$/.test(isoDate)) return isoDate
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return date.toISOString().slice(0, 10)
}
</script>

<style scoped>
.api-key-summary-card {
  width: 100%;
  min-width: 0;
  padding: 16px;
  border: 1px solid var(--workspace-border);
  border-radius: 12px;
  color: var(--workspace-text);
  background: var(--workspace-card-surface);
  font-variant-numeric: tabular-nums;
  -webkit-font-smoothing: antialiased;
}

.api-key-summary-card:hover {
  border-color: var(--workspace-border-strong);
}

.api-key-summary-card__head {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.api-key-summary-card__name {
  min-width: 0;
  margin: 0;
  overflow: hidden;
  color: var(--workspace-text);
  font-size: 14px;
  font-weight: 600;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.api-key-summary-card__actions {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 4px;
}

.api-key-summary-card__action,
.api-key-summary-card__copy {
  display: inline-grid;
  place-items: center;
  border: 1px solid transparent;
  color: var(--workspace-text-muted);
  background: transparent;
  outline: none;
  transition: color 150ms ease, border-color 150ms ease, background-color 150ms ease;
}

.api-key-summary-card__action {
  width: 40px;
  height: 40px;
  flex: 0 0 40px;
  border-radius: 8px;
}

.api-key-summary-card__action:hover,
.api-key-summary-card__copy:hover {
  border-color: var(--workspace-border);
  color: var(--workspace-text);
  background: var(--workspace-hover);
}

.api-key-summary-card__action:focus-visible,
.api-key-summary-card__copy:focus-visible,
.api-key-summary-card__group:focus-visible {
  outline: 2px solid var(--workspace-work-accent);
  outline-offset: 2px;
}

.api-key-summary-card__field {
  min-width: 0;
  margin-top: 14px;
}

.api-key-summary-card__label {
  display: block;
  margin-bottom: 5px;
  color: var(--workspace-text-muted);
  font-size: 11px;
  font-weight: 400;
  line-height: 16px;
}

.api-key-summary-card__credential {
  min-width: 0;
  min-height: 42px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 6px 0 11px;
  border: 1px solid var(--workspace-border);
  border-radius: 8px;
  background: var(--workspace-surface-subtle);
}

.api-key-summary-card__credential code {
  min-width: 0;
  flex: 1 1 auto;
  overflow: hidden;
  color: var(--workspace-text-secondary);
  font-family: var(--workspace-font-mono);
  font-size: 11px;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.api-key-summary-card__copy {
  width: 36px;
  height: 36px;
  flex: 0 0 36px;
  border-radius: 7px;
}

.api-key-summary-card__copy.is-copied {
  color: var(--workspace-work-accent);
}

.api-key-summary-card__group-field {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
}

.api-key-summary-card__group-field .api-key-summary-card__label {
  flex: 0 0 auto;
  margin-bottom: 0;
}

.api-key-summary-card__group {
  --group-chip-border: var(--workspace-border);
  --group-chip-text: var(--workspace-text-secondary);
  --group-chip-surface: var(--workspace-surface-subtle);
  width: max-content;
  min-width: 0;
  min-height: 36px;
  max-width: min(100%, 18rem);
  display: inline-flex;
  flex: 0 1 auto;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 0 10px;
  border: 1px solid var(--group-chip-border);
  border-radius: 999px;
  color: var(--group-chip-text);
  background: var(--group-chip-surface);
  text-align: left;
  outline: none;
  transition: color 150ms ease, border-color 150ms ease, background-color 150ms ease;
}

.api-key-summary-card__group:hover {
  --group-chip-border: var(--workspace-border-strong);
  --group-chip-text: var(--workspace-text);
  --group-chip-surface: var(--workspace-hover);
}

.api-key-summary-card__group-name {
  min-width: 0;
  overflow: hidden;
  font-size: 12px;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Provider-specific tones make the group immediately scannable on mobile. */
.api-key-summary-card__group--assigned,
.api-key-summary-card__group--unassigned {
  --group-chip-border: var(--workspace-border);
  --group-chip-text: var(--workspace-text-secondary);
  --group-chip-surface: var(--workspace-surface-subtle);
}

.api-key-summary-card__group--openai {
  --group-chip-border: #bbf7d0;
  --group-chip-text: #15803d;
  --group-chip-surface: #f0fdf4;
}

.api-key-summary-card__group--anthropic {
  --group-chip-border: #fed7aa;
  --group-chip-text: #c2410c;
  --group-chip-surface: #fff7ed;
}

.api-key-summary-card__group--gemini {
  --group-chip-border: #bfdbfe;
  --group-chip-text: #1d4ed8;
  --group-chip-surface: #eff6ff;
}

.api-key-summary-card__group--antigravity {
  --group-chip-border: #e9d5ff;
  --group-chip-text: #7e22ce;
  --group-chip-surface: #faf5ff;
}

.api-key-summary-card__group--grok {
  --group-chip-border: #d4d4d8;
  --group-chip-text: #3f3f46;
  --group-chip-surface: #f4f4f5;
}

.api-key-summary-card__group--deepseek {
  --group-chip-border: #bae6fd;
  --group-chip-text: #0369a1;
  --group-chip-surface: #f0f9ff;
}

:global(html.dark .api-key-summary-card__group--openai) {
  --group-chip-border: rgb(74 222 128 / 0.34);
  --group-chip-text: #86efac;
  --group-chip-surface: rgb(20 83 45 / 0.34);
}

:global(html.dark .api-key-summary-card__group--anthropic) {
  --group-chip-border: rgb(251 146 60 / 0.34);
  --group-chip-text: #fdba74;
  --group-chip-surface: rgb(124 45 18 / 0.32);
}

:global(html.dark .api-key-summary-card__group--gemini) {
  --group-chip-border: rgb(96 165 250 / 0.34);
  --group-chip-text: #93c5fd;
  --group-chip-surface: rgb(30 58 138 / 0.34);
}

:global(html.dark .api-key-summary-card__group--antigravity) {
  --group-chip-border: rgb(192 132 252 / 0.34);
  --group-chip-text: #d8b4fe;
  --group-chip-surface: rgb(88 28 135 / 0.34);
}

:global(html.dark .api-key-summary-card__group--grok) {
  --group-chip-border: rgb(161 161 170 / 0.34);
  --group-chip-text: #d4d4d8;
  --group-chip-surface: rgb(63 63 70 / 0.48);
}

:global(html.dark .api-key-summary-card__group--deepseek) {
  --group-chip-border: rgb(56 189 248 / 0.34);
  --group-chip-text: #7dd3fc;
  --group-chip-surface: rgb(8 47 73 / 0.38);
}

.api-key-summary-card__group--assigned:hover,
.api-key-summary-card__group--unassigned:hover {
  --group-chip-border: var(--workspace-border-strong);
  --group-chip-text: var(--workspace-text);
  --group-chip-surface: var(--workspace-hover);
}

.api-key-summary-card__group--openai:hover {
  --group-chip-border: #86efac;
  --group-chip-text: #166534;
  --group-chip-surface: #dcfce7;
}

.api-key-summary-card__group--anthropic:hover {
  --group-chip-border: #fdba74;
  --group-chip-text: #9a3412;
  --group-chip-surface: #ffedd5;
}

.api-key-summary-card__group--gemini:hover {
  --group-chip-border: #93c5fd;
  --group-chip-text: #1e40af;
  --group-chip-surface: #dbeafe;
}

.api-key-summary-card__group--antigravity:hover {
  --group-chip-border: #d8b4fe;
  --group-chip-text: #6b21a8;
  --group-chip-surface: #f3e8ff;
}

.api-key-summary-card__group--grok:hover {
  --group-chip-border: #a1a1aa;
  --group-chip-text: #27272a;
  --group-chip-surface: #e4e4e7;
}

.api-key-summary-card__group--deepseek:hover {
  --group-chip-border: #7dd3fc;
  --group-chip-text: #075985;
  --group-chip-surface: #e0f2fe;
}

:global(html.dark .api-key-summary-card__group--openai:hover) {
  --group-chip-border: rgb(134 239 172 / 0.52);
  --group-chip-text: #bbf7d0;
  --group-chip-surface: rgb(22 101 52 / 0.5);
}

:global(html.dark .api-key-summary-card__group--anthropic:hover) {
  --group-chip-border: rgb(253 186 116 / 0.52);
  --group-chip-text: #fed7aa;
  --group-chip-surface: rgb(154 52 18 / 0.5);
}

:global(html.dark .api-key-summary-card__group--gemini:hover) {
  --group-chip-border: rgb(147 197 253 / 0.52);
  --group-chip-text: #bfdbfe;
  --group-chip-surface: rgb(30 64 175 / 0.5);
}

:global(html.dark .api-key-summary-card__group--antigravity:hover) {
  --group-chip-border: rgb(216 180 254 / 0.52);
  --group-chip-text: #e9d5ff;
  --group-chip-surface: rgb(107 33 168 / 0.5);
}

:global(html.dark .api-key-summary-card__group--grok:hover) {
  --group-chip-border: rgb(212 212 216 / 0.52);
  --group-chip-text: #f4f4f5;
  --group-chip-surface: rgb(82 82 91 / 0.64);
}

:global(html.dark .api-key-summary-card__group--deepseek:hover) {
  --group-chip-border: rgb(125 211 252 / 0.52);
  --group-chip-text: #bae6fd;
  --group-chip-surface: rgb(7 89 133 / 0.5);
}

.api-key-summary-card__time-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  margin-top: 12px;
  padding-top: 11px;
  border-top: 1px solid var(--workspace-divider);
}

.api-key-summary-card__time-item {
  min-width: 0;
}

.api-key-summary-card__time-item .api-key-summary-card__label {
  margin-bottom: 4px;
}

.api-key-summary-card__time-value {
  display: block;
  min-width: 0;
  overflow: hidden;
  color: var(--workspace-text-secondary);
  font-family: var(--workspace-font-mono);
  font-size: 11px;
  line-height: 16px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (prefers-reduced-motion: reduce) {
  .api-key-summary-card__action,
  .api-key-summary-card__copy,
  .api-key-summary-card__group {
    transition: none;
  }
}
</style>

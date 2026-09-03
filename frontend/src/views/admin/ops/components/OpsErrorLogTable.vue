<template>
  <div class="flex h-full min-h-0 flex-col">
    <div class="flex min-h-0 flex-1 flex-col overflow-hidden" :class="flat ? '' : 'card'">
      <IpGeoBatchToolbar :ips="rows.map((r) => r.client_ip)" @failed="emit('ipGeoBatchFailed')" />

      <DataTable
        class="ops-error-event-table"
        :columns="columns"
        :data="rows"
        :loading="loading"
        :row-aria-label="errorRowAriaLabel"
        clickable-rows
        server-side-sort
        default-sort-key="created_at"
        default-sort-order="desc"
        @sort="onSort"
        @rowClick="(row) => emit('openErrorDetail', row.id)"
      >
        <template #cell-created_at="{ row }">
          <div class="ops-error-event-table__time">
            <span>{{ formatDateTime(row.created_at) }}</span>
            <span class="ops-error-event-table__event-id" :title="eventIdentifier(row)">
              #{{ row.id }}
            </span>
          </div>
        </template>

        <template #cell-type="{ row }">
          <div class="ops-error-event-table__classification">
            <span class="ops-error-event-table__badge" :class="getPhaseBadge(row).className">
              {{ getPhaseBadge(row).label }}
            </span>
            <span v-if="row.error_owner" class="ops-error-event-table__owner">
              {{ ownerLabel(row.error_owner) }}
            </span>
          </div>
        </template>

        <template #cell-endpoint="{ row }">
          <div class="max-w-[320px] space-y-1 text-xs">
            <div class="break-all text-gray-700 dark:text-gray-300">
              <span class="font-medium text-gray-500 dark:text-gray-400">{{ t('usage.inbound') }}:</span>
              <span class="ml-1">{{ row.inbound_endpoint?.trim() || '-' }}</span>
            </div>
            <div v-if="row.upstream_endpoint" class="break-all text-gray-700 dark:text-gray-300">
              <span class="font-medium text-gray-500 dark:text-gray-400">{{ t('usage.upstream') }}:</span>
              <span class="ml-1">{{ row.upstream_endpoint?.trim() || '-' }}</span>
            </div>
          </div>
        </template>

        <template #cell-platform="{ row }">
          <span class="text-sm text-gray-900 dark:text-white">{{ row.platform || '-' }}</span>
        </template>

        <template #cell-model="{ row }">
          <div v-if="hasModelMapping(row)" class="space-y-0.5 text-xs">
            <div class="break-all font-medium text-gray-900 dark:text-white">{{ row.requested_model }}</div>
            <div class="break-all text-gray-500 dark:text-gray-400"><span class="mr-0.5">↳</span>{{ row.upstream_model }}</div>
          </div>
          <span v-else-if="displayModel(row)" class="text-sm font-medium text-gray-900 dark:text-white">{{ displayModel(row) }}</span>
          <span v-else class="text-sm text-gray-400 dark:text-gray-500">-</span>
        </template>

        <template #cell-group="{ row }">
          <span
            v-if="row.group_id"
            class="inline-flex items-center rounded px-2 py-0.5 text-xs font-medium bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-200"
            :title="t('admin.ops.errorLog.id') + ' ' + row.group_id"
          >
            {{ row.group_name || '#' + row.group_id }}
          </span>
          <span v-else class="text-sm text-gray-400 dark:text-gray-500">-</span>
        </template>

        <template #cell-user="{ row }">
          <div v-if="row.user_id" class="text-sm">
            <button
              v-if="userClickable && row.user_email"
              class="font-medium text-primary-600 underline decoration-dashed underline-offset-2 transition-colors hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
              :title="t('admin.usage.clickToViewBalance')"
              @click.stop="emit('userClick', row.user_id, row.user_email)"
            >
              {{ row.user_email }}
            </button>
            <span v-else class="font-medium text-gray-900 dark:text-white">{{ row.user_email || '-' }}</span>
            <span class="ml-1 text-gray-500 dark:text-gray-400">#{{ row.user_id }}</span>
          </div>
          <!-- 认证失败行 user_id 为空:回退显示已删除 KEY 所有者(归因快照,与详情弹窗一致) -->
          <div v-else-if="row.deleted_key_owner_user_id" class="text-sm">
            <button
              v-if="userClickable && row.deleted_key_owner_email"
              class="font-medium text-primary-600 underline decoration-dashed underline-offset-2 transition-colors hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
              :title="t('admin.usage.clickToViewBalance')"
              @click.stop="emit('userClick', row.deleted_key_owner_user_id, row.deleted_key_owner_email ?? undefined)"
            >
              {{ row.deleted_key_owner_email }}
            </button>
            <span v-else class="font-medium text-gray-900 dark:text-white">{{ row.deleted_key_owner_email || '-' }}</span>
            <span class="ml-1 text-gray-500 dark:text-gray-400">#{{ row.deleted_key_owner_user_id }}</span>
          </div>
          <span v-else class="text-sm text-gray-400 dark:text-gray-500">-</span>
        </template>

        <template #cell-api_key="{ row }">
          <div v-if="row.api_key_id || row.api_key_name" class="text-sm">
            <span class="text-gray-900 dark:text-white">{{ row.api_key_name || '#' + row.api_key_id }}</span>
            <span
              v-if="row.api_key_deleted"
              class="ml-1 inline-flex items-center rounded px-1 py-px text-[10px] font-medium leading-tight bg-rose-100 text-rose-600 ring-1 ring-inset ring-rose-200 dark:bg-rose-500/20 dark:text-rose-400 dark:ring-rose-500/30"
            >{{ t('admin.ops.errorLog.keyDeletedBadge') }}</span>
          </div>
          <span v-else class="text-sm text-gray-400 dark:text-gray-500">-</span>
        </template>

        <template #cell-account="{ row }">
          <span
            v-if="row.account_id"
            class="text-sm text-gray-900 dark:text-white"
            :title="t('admin.ops.errorLog.accountId') + ' ' + row.account_id"
          >{{ row.account_name || '#' + row.account_id }}</span>
          <span v-else class="text-sm text-gray-400 dark:text-gray-500">-</span>
        </template>

        <template #cell-category="{ row }">
          <span
            class="ops-error-event-table__badge"
            :class="errorCategoryBadgeClass(mapErrorCategory(row.phase, row.type))"
          >
            {{ t('usage.errors.categories.' + mapErrorCategory(row.phase, row.type)) }}
          </span>
        </template>

        <template #cell-status="{ row }">
          <div class="ops-error-event-table__status-stack">
            <span class="ops-error-event-table__status" :class="getStatusClass(row.status_code)">
              {{ row.status_code }}
            </span>
            <span
              v-if="normalizedErrorPriority(row.severity)"
              class="ops-error-event-table__priority"
              :class="getSeverityClass(normalizedErrorPriority(row.severity))"
            >{{ normalizedErrorPriority(row.severity) }}</span>
            <span v-else-if="row.severity" class="ops-error-event-table__priority ops-error-event-table__priority--neutral">
              {{ row.severity }}
            </span>
          </div>
        </template>

        <template #cell-message="{ row }">
          <div class="ops-error-event-table__message-cell">
            <span
              v-if="row.message"
              class="ops-error-event-table__message"
              :title="row.message"
            >{{ formatSmartMessage(row.message) || '-' }}</span>
            <span v-else class="text-sm text-gray-400 dark:text-gray-500">-</span>
            <div class="ops-error-event-table__request-meta">
              <span v-if="eventIdentifier(row)" class="ops-error-event-table__request-id" :title="eventIdentifier(row)">
                {{ eventIdentifier(row) }}
              </span>
              <span v-if="formatRequestType(row.request_type)" class="ops-error-event-table__request-type">
                {{ formatRequestType(row.request_type) }}
              </span>
            </div>
          </div>
        </template>

        <template #cell-user_agent="{ row }">
          <span
            v-if="row.user_agent"
            class="block max-w-[320px] truncate text-sm text-gray-600 dark:text-gray-400"
            :title="row.user_agent"
          >{{ row.user_agent }}</span>
          <span v-else class="text-sm text-gray-400 dark:text-gray-500">-</span>
        </template>

        <template #cell-client_ip="{ row }">
          <div @click.stop>
            <div v-if="row.client_ip">
              <span class="text-sm font-mono text-gray-600 dark:text-gray-400">{{ row.client_ip }}</span>
              <IpGeoCell :ip="row.client_ip" />
            </div>
            <span v-else class="text-sm text-gray-400 dark:text-gray-500">-</span>
          </div>
        </template>

        <template #cell-actions="{ row }">
          <button
            type="button"
            class="ops-error-event-table__detail-button"
            :title="t('admin.ops.errorLog.details')"
            :aria-label="`${t('admin.ops.errorLog.details')} #${row.id}`"
            @click.stop="emit('openErrorDetail', row.id)"
          >
            <Icon name="chevronRight" size="sm" :stroke-width="2" />
          </button>
        </template>

        <template #empty><EmptyState :message="t('admin.ops.errorLog.noErrors')" /></template>
      </DataTable>
    </div>

    <div class="flex-shrink-0">
      <Pagination
        v-if="total > 0"
        :total="total"
        :page="page"
        :page-size="pageSize"
        @update:page="emit('update:page', $event)"
        @update:pageSize="emit('update:pageSize', $event)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Pagination from '@/components/common/Pagination.vue'
import IpGeoCell from '@/components/common/IpGeoCell.vue'
import IpGeoBatchToolbar from '@/components/common/IpGeoBatchToolbar.vue'
import Icon from '@/components/icons/Icon.vue'
import type { OpsErrorLog } from '@/api/admin/ops'
import type { Column } from '@/components/common/types'
import { getSeverityClass, formatDateTime } from '../utils/opsFormatters'
import { mapErrorCategory } from '@/utils/errorCategory'
import { mapErrorSortKey, statusCodeBadgeClass } from '@/utils/errorBadges'
import {
  errorCategoryBadgeClass,
  errorOwnerLabelKey,
  errorPhasePresentation,
  normalizedErrorPriority
} from '../utils/errorPresentation'

const { t } = useI18n()

const errorRowAriaLabel = (row: OpsErrorLog) => {
  const summary = formatSmartMessage(row.message) || row.request_id || row.client_request_id || `#${row.id}`
  const phase = getPhaseBadge(row).label
  const category = t('usage.errors.categories.' + mapErrorCategory(row.phase, row.type))
  return `${t('admin.ops.errorLog.details')}: ${row.status_code || '-'} · ${phase} · ${category} · ${summary}`
}

// 事件优先：先让管理员判断「何时、影响级别、阶段、分类、发生了什么」，
// 再按需横向查看身份与技术上下文。key 集合保持不变，兼容 UsageView 的列偏好。
const allColumns = computed<Column[]>(() => [
  { key: 'created_at', label: t('admin.ops.errorLog.timeId'), sortable: true, class: 'ops-error-event-table__col-time' },
  { key: 'status', label: t('admin.ops.errorLog.status'), sortable: true, class: 'ops-error-event-table__col-status' },
  { key: 'type', label: t('admin.ops.errorLog.phase'), class: 'ops-error-event-table__col-phase' },
  { key: 'category', label: t('usage.errors.category'), class: 'ops-error-event-table__col-category' },
  { key: 'message', label: t('admin.ops.errorLog.message'), class: 'ops-error-event-table__col-message' },
  { key: 'user', label: t('admin.ops.errorLog.user'), class: 'ops-error-event-table__col-user' },
  { key: 'model', label: t('admin.ops.errorLog.model'), sortable: true },
  { key: 'platform', label: t('admin.ops.errorLog.platform') },
  { key: 'endpoint', label: t('admin.ops.errorLog.endpoint') },
  { key: 'api_key', label: t('admin.ops.errorLog.apiKey') },
  { key: 'account', label: t('admin.ops.errorLog.account') },
  { key: 'group', label: t('admin.ops.errorLog.group') },
  { key: 'user_agent', label: t('usage.userAgent') },
  { key: 'client_ip', label: t('admin.ops.errorLog.ip') },
  { key: 'actions', label: t('admin.ops.errorLog.action'), class: 'ops-error-event-table__col-actions' },
])

// 传入 visibleColumnKeys 时按其过滤(列设置);未传则全量(Ops 弹窗等使用方)
const columns = computed<Column[]>(() =>
  props.visibleColumnKeys
    ? allColumns.value.filter((c) => props.visibleColumnKeys!.includes(c.key))
    : allColumns.value
)

function hasModelMapping(log: OpsErrorLog): boolean {
  const requested = String(log.requested_model || '').trim()
  const upstream = String(log.upstream_model || '').trim()
  return !!requested && !!upstream && requested !== upstream
}

function displayModel(log: OpsErrorLog): string {
  const upstream = String(log.upstream_model || '').trim()
  if (upstream) return upstream
  const requested = String(log.requested_model || '').trim()
  if (requested) return requested
  return String(log.model || '').trim()
}

function formatRequestType(type: number | null | undefined): string {
  switch (type) {
    case 1: return t('admin.ops.errorLog.requestTypeSync')
    case 2: return t('admin.ops.errorLog.requestTypeStream')
    case 3: return t('admin.ops.errorLog.requestTypeWs')
    default: return ''
  }
}

// 徽章配色对齐用量明细(UsageTable)的 bg-X-100/text-X-800 体系
function getPhaseBadge(log: OpsErrorLog): { label: string; className: string } {
  const presentation = errorPhasePresentation(log)
  return {
    label: presentation.labelKey ? t(presentation.labelKey) : (presentation.fallback || t('common.unknown')),
    className: presentation.className
  }
}

function eventIdentifier(log: OpsErrorLog): string {
  return String(log.request_id || log.client_request_id || '').trim()
}

function ownerLabel(owner?: string | null): string {
  const value = String(owner || '').trim().toLowerCase()
  const labelKey = errorOwnerLabelKey(value)
  return labelKey ? t(labelKey) : (value || t('common.unknown'))
}

interface Props {
  rows: OpsErrorLog[]
  total: number
  loading: boolean
  page: number
  pageSize: number
  /** 用户邮箱可点击(emit userClick),仅在有弹窗承接的使用方开启 */
  userClickable?: boolean
  /** 列设置:仅显示这些 key 的列;不传则全量 */
  visibleColumnKeys?: string[]
  /** 嵌入统一卡片内使用：去掉自身卡片外观 */
  flat?: boolean
}

interface Emits {
  (e: 'openErrorDetail', id: number): void
  (e: 'update:page', value: number): void
  (e: 'update:pageSize', value: number): void
  (e: 'ipGeoBatchFailed'): void
  (e: 'sort', sortBy: string, sortOrder: 'asc' | 'desc'): void
  (e: 'userClick', userId: number, email?: string): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

function onSort(key: string, order: 'asc' | 'desc') {
  emit('sort', mapErrorSortKey(key), order)
}

const getStatusClass = statusCodeBadgeClass

function formatSmartMessage(msg: string): string {
  if (!msg) return ''

  if (msg.startsWith('{') || msg.startsWith('[')) {
    try {
      const obj = JSON.parse(msg)
      if (obj?.error?.message) return String(obj.error.message)
      if (obj?.message) return String(obj.message)
      if (obj?.detail) return String(obj.detail)
      if (typeof obj === 'object') return JSON.stringify(obj).substring(0, 150)
    } catch {
      // ignore parse error
    }
  }

  if (msg.includes('context deadline exceeded')) return t('admin.ops.errorLog.commonErrors.contextDeadlineExceeded')
  if (msg.includes('connection refused')) return t('admin.ops.errorLog.commonErrors.connectionRefused')
  if (msg.toLowerCase().includes('rate limit')) return t('admin.ops.errorLog.commonErrors.rateLimit')

  return msg.length > 200 ? msg.substring(0, 200) + '...' : msg
}
</script>

<style scoped>
.ops-error-event-table__time,
.ops-error-event-table__classification,
.ops-error-event-table__status-stack,
.ops-error-event-table__request-meta {
  display: flex;
  align-items: center;
}

.ops-error-event-table__time {
  min-width: 7.75rem;
  flex-direction: column;
  align-items: flex-start;
  gap: 0.2rem;
  color: var(--lx-clay-text-secondary);
  font-size: 0.8125rem;
  font-variant-numeric: tabular-nums;
}

.ops-error-event-table__event-id,
.ops-error-event-table__request-id {
  max-width: 15rem;
  overflow: hidden;
  color: var(--lx-clay-text-subtle);
  font-family: var(--lx-clay-font-mono);
  font-size: 0.6875rem;
  line-height: 1.25rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-error-event-table__classification,
.ops-error-event-table__status-stack {
  gap: 0.4rem;
}

.ops-error-event-table__classification {
  min-width: 6.75rem;
  flex-direction: column;
  align-items: flex-start;
}

.ops-error-event-table__badge,
.ops-error-event-table__status,
.ops-error-event-table__priority,
.ops-error-event-table__request-type {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 0.5rem;
  font-weight: 700;
  line-height: 1rem;
  white-space: nowrap;
}

.ops-error-event-table__badge,
.ops-error-event-table__status {
  min-height: 1.75rem;
  padding: 0.25rem 0.55rem;
  font-size: 0.75rem;
}

.ops-error-event-table__status {
  min-width: 3rem;
  font-variant-numeric: tabular-nums;
}

.ops-error-event-table__priority,
.ops-error-event-table__request-type {
  min-height: 1.25rem;
  padding: 0.125rem 0.4rem;
  font-size: 0.625rem;
}

.ops-error-event-table__priority--neutral,
.ops-error-event-table__request-type {
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-recessed);
}

.ops-error-event-table__owner {
  color: var(--lx-clay-text-subtle);
  font-size: 0.6875rem;
  line-height: 1rem;
}

.ops-error-event-table__message-cell {
  width: min(30rem, 36vw);
  min-width: 18rem;
  white-space: normal;
}

.ops-error-event-table__message {
  display: -webkit-box;
  overflow: hidden;
  color: var(--lx-clay-text);
  font-size: 0.8125rem;
  font-weight: 600;
  line-height: 1.25rem;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.ops-error-event-table__request-meta {
  min-width: 0;
  gap: 0.45rem;
  margin-top: 0.35rem;
}

.ops-error-event-table__detail-button {
  display: inline-flex;
  width: 2.75rem;
  height: 2.75rem;
  align-items: center;
  justify-content: center;
  border-radius: var(--lx-clay-radius-control);
  color: var(--lx-clay-text-muted);
  transition: color 150ms ease, background-color 150ms ease, transform 150ms ease;
}

.ops-error-event-table__detail-button:hover {
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

.ops-error-event-table__detail-button:active {
  transform: scale(0.96);
}

.ops-error-event-table__detail-button:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 30%, transparent);
  outline-offset: 2px;
}

:deep(.ops-error-event-table .ops-error-event-table__col-time) {
  position: sticky;
  left: 0;
  z-index: 2;
  background: var(--lx-clay-surface);
}

:deep(.ops-error-event-table thead .ops-error-event-table__col-time) {
  z-index: 4;
  background: var(--lx-clay-recessed);
}

:deep(.ops-error-event-table .data-table-row:hover .ops-error-event-table__col-time),
:deep(.ops-error-event-table .data-table-row:focus .ops-error-event-table__col-time) {
  background: color-mix(in srgb, var(--lx-clay-surface) 90%, var(--lx-clay-accent-soft));
}

:deep(.ops-error-event-table .ops-error-event-table__col-actions) {
  min-width: 4rem;
  text-align: center;
}

@media (max-width: 1023px) {
  .ops-error-event-table__message-cell {
    width: auto;
    min-width: 0;
    text-align: right;
  }

  .ops-error-event-table__request-meta {
    justify-content: flex-end;
  }

  .ops-error-event-table__classification,
  .ops-error-event-table__status-stack,
  .ops-error-event-table__time {
    min-width: 0;
    align-items: flex-end;
  }
}

@media (prefers-reduced-motion: reduce) {
  .ops-error-event-table__detail-button {
    transition-duration: 0.01ms;
  }
}
</style>

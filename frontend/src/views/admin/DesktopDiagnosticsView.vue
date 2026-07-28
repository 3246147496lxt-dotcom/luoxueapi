<template>
  <AppLayout variant="home-clay">
    <TablePageLayout>
      <template #header>
        <AdminPageHeader
          :title="t('admin.desktopDiagnostics.title')"
          :description="t('admin.desktopDiagnostics.description')"
        >
          <template #meta>
            <span class="diagnostic-count">{{ pagination.total }}</span>
          </template>
          <template #secondary-actions>
            <button
              type="button"
              class="btn btn-secondary"
              :disabled="loading"
              @click="loadDiagnostics"
            >
              <Icon name="refresh" size="sm" />
              <span>{{ t('common.refresh') }}</span>
            </button>
          </template>
        </AdminPageHeader>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="diagnostics" :loading="loading" row-key="id">
          <template #cell-created_at="{ value }">
            <span class="whitespace-nowrap text-gray-700 dark:text-gray-200">
              {{ formatDateTime(value) }}
            </span>
          </template>

          <template #cell-owner="{ row }">
            <div class="min-w-0 max-w-[220px]">
              <div class="font-medium text-gray-900 dark:text-white">
                {{ t('admin.desktopDiagnostics.labels.user', { id: row.user_id }) }}
              </div>
              <div class="mt-0.5 truncate font-mono text-xs text-gray-500 dark:text-dark-300" :title="row.device_id">
                {{ t('admin.desktopDiagnostics.labels.device', { id: shortID(row.device_id) }) }}
              </div>
            </div>
          </template>

          <template #cell-environment="{ row }">
            <div class="whitespace-nowrap">
              <div class="text-gray-900 dark:text-white">
                {{ t('admin.desktopDiagnostics.labels.appVersion', { version: row.app_version }) }}
              </div>
              <div class="mt-0.5 text-xs text-gray-500 dark:text-dark-300">
                macOS {{ row.os_version }} · {{ architectureLabel(row.architecture) }}
              </div>
            </div>
          </template>

          <template #cell-local_status="{ row }">
            <div class="flex min-w-[148px] flex-col items-start gap-1.5">
              <span :class="statusBadgeClass(row.gateway_status)">
                {{ t('admin.desktopDiagnostics.labels.gateway') }} · {{ statusLabel(row.gateway_status) }}
              </span>
              <span :class="statusBadgeClass(row.codex_config_status)">
                {{ t('admin.desktopDiagnostics.labels.codex') }} · {{ statusLabel(row.codex_config_status) }}
              </span>
            </div>
          </template>

          <template #cell-request_sample_count="{ row }">
            <div class="whitespace-nowrap">
              <div class="font-medium text-gray-900 dark:text-white">
                {{ t('admin.desktopDiagnostics.labels.samples', { count: row.request_sample_count }) }}
              </div>
              <div
                class="mt-0.5 text-xs"
                :class="row.request_error_count > 0 ? 'text-red-600 dark:text-red-400' : 'text-gray-500 dark:text-dark-300'"
              >
                {{ row.request_error_count > 0
                  ? t('admin.desktopDiagnostics.labels.errors', { count: row.request_error_count })
                  : t('admin.desktopDiagnostics.labels.noErrors') }}
              </div>
            </div>
          </template>

          <template #cell-expires_at="{ row }">
            <div class="whitespace-nowrap">
              <span :class="retentionBadgeClass(row.expires_at)">
                {{ retentionLabel(row.expires_at) }}
              </span>
              <div class="mt-1 text-xs text-gray-500 dark:text-dark-300">
                {{ t('admin.desktopDiagnostics.labels.expiresAt', { time: formatDateTime(row.expires_at) }) }}
              </div>
            </div>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex min-w-[76px] items-center justify-end gap-1">
              <button
                type="button"
                class="diagnostic-icon-button"
                :title="t('admin.desktopDiagnostics.actions.view')"
                :aria-label="t('admin.desktopDiagnostics.actions.view')"
                :disabled="detailLoading"
                @click="openDetail(row)"
              >
                <Icon name="eye" size="sm" />
              </button>
              <button
                type="button"
                class="diagnostic-icon-button"
                :title="t('admin.desktopDiagnostics.actions.download')"
                :aria-label="t('admin.desktopDiagnostics.actions.download')"
                :disabled="downloadingID === row.id"
                @click="downloadDiagnostic(row.id)"
              >
                <Icon name="download" size="sm" />
              </button>
            </div>
          </template>

          <template #empty>
            <div class="flex flex-col items-center px-6 py-16 text-center">
              <Icon name="activity" size="xl" class="text-gray-400 dark:text-dark-400" />
              <h2 class="mt-4 text-base font-semibold text-gray-900 dark:text-white">
                {{ t('admin.desktopDiagnostics.emptyTitle') }}
              </h2>
              <p class="mt-1 max-w-md text-sm text-gray-500 dark:text-dark-300">
                {{ t('admin.desktopDiagnostics.emptyDescription') }}
              </p>
            </div>
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <BaseDialog
      :show="detailVisible"
      :title="t('admin.desktopDiagnostics.detail.title')"
      width="wide"
      @close="closeDetail"
    >
      <div v-if="detailLoading" class="flex min-h-64 items-center justify-center gap-3 text-sm text-gray-500 dark:text-dark-300">
        <span class="h-5 w-5 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></span>
        {{ t('admin.desktopDiagnostics.detail.loading') }}
      </div>

      <div v-else-if="selectedDetail" class="space-y-5">
        <div class="diagnostic-audit-notice">
          <Icon name="shield" size="sm" class="mt-0.5 flex-shrink-0" />
          <p>{{ t('admin.desktopDiagnostics.detail.auditNotice') }}</p>
        </div>

        <section aria-labelledby="diagnostic-overview-title">
          <h4 id="diagnostic-overview-title" class="diagnostic-section-title">
            {{ t('admin.desktopDiagnostics.detail.overview') }}
          </h4>
          <dl class="diagnostic-summary-grid">
            <div>
              <dt>{{ t('admin.desktopDiagnostics.columns.owner') }}</dt>
              <dd>{{ t('admin.desktopDiagnostics.labels.user', { id: selectedDetail.user_id }) }}</dd>
            </div>
            <div>
              <dt>{{ t('admin.desktopDiagnostics.columns.environment') }}</dt>
              <dd>macOS {{ selectedDetail.os_version }} · {{ architectureLabel(selectedDetail.architecture) }}</dd>
            </div>
            <div>
              <dt>{{ t('admin.desktopDiagnostics.columns.createdAt') }}</dt>
              <dd>{{ formatDateTime(selectedDetail.created_at) }}</dd>
            </div>
            <div>
              <dt>{{ t('admin.desktopDiagnostics.columns.retention') }}</dt>
              <dd>{{ formatDateTime(selectedDetail.expires_at) }}</dd>
            </div>
          </dl>
        </section>

        <section aria-labelledby="diagnostic-payload-title">
          <h4 id="diagnostic-payload-title" class="diagnostic-section-title">
            {{ t('admin.desktopDiagnostics.detail.payload') }}
          </h4>
          <pre class="diagnostic-payload"><code>{{ formattedDiagnostic }}</code></pre>
        </section>
      </div>

      <template #footer>
        <button type="button" class="btn btn-secondary" @click="closeDetail">
          {{ t('admin.desktopDiagnostics.actions.close') }}
        </button>
        <button
          v-if="selectedDetail"
          type="button"
          class="btn btn-primary"
          :disabled="downloadingID === selectedDetail.id"
          @click="downloadDiagnostic(selectedDetail.id)"
        >
          <Icon name="download" size="sm" />
          <span>{{ t('admin.desktopDiagnostics.actions.download') }}</span>
        </button>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { DesktopDiagnosticDetail, DesktopDiagnosticMetadata } from '@/api/admin'
import type { Column } from '@/components/common/types'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import AppLayout from '@/components/layout/AppLayout.vue'
import AdminPageHeader from '@/components/layout/AdminPageHeader.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import Icon from '@/components/icons/Icon.vue'

const { locale, t } = useI18n()
const appStore = useAppStore()

const diagnostics = ref<DesktopDiagnosticMetadata[]>([])
const loading = ref(false)
const detailLoading = ref(false)
const detailVisible = ref(false)
const selectedDetail = ref<DesktopDiagnosticDetail | null>(null)
const downloadingID = ref<string | null>(null)
const pagination = reactive({ page: 1, page_size: getPersistedPageSize(), total: 0 })
let listController: AbortController | null = null
let detailController: AbortController | null = null

const columns = computed<Column[]>(() => [
  { key: 'created_at', label: t('admin.desktopDiagnostics.columns.createdAt') },
  { key: 'owner', label: t('admin.desktopDiagnostics.columns.owner') },
  { key: 'environment', label: t('admin.desktopDiagnostics.columns.environment') },
  { key: 'local_status', label: t('admin.desktopDiagnostics.columns.status') },
  { key: 'request_sample_count', label: t('admin.desktopDiagnostics.columns.requests') },
  { key: 'expires_at', label: t('admin.desktopDiagnostics.columns.retention') },
  { key: 'actions', label: t('admin.desktopDiagnostics.columns.actions') },
])

const formattedDiagnostic = computed(() => (
  selectedDetail.value ? JSON.stringify(selectedDetail.value.diagnostic, null, 2) : ''
))

function formatDateTime(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat(locale.value, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

function shortID(value: string): string {
  return value.length > 14 ? `${value.slice(0, 8)}…${value.slice(-4)}` : value
}

function architectureLabel(value: string): string {
  if (value === 'arm64') return 'Apple Silicon'
  if (value === 'x86_64') return 'Intel'
  return value
}

function statusLabel(value: string): string {
  const key = `admin.desktopDiagnostics.statuses.${value}`
  const translated = t(key)
  return translated === key ? value : translated
}

function statusBadgeClass(value: string): string[] {
  const healthy = value === 'running' || value === 'managed'
  const failed = value === 'error'
  return [
    'inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium',
    healthy
      ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300'
      : failed
        ? 'bg-red-50 text-red-700 dark:bg-red-950/40 dark:text-red-300'
        : 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-200',
  ]
}

function retentionState(expiresAt: string): 'available' | 'expiringSoon' | 'expired' {
  const remaining = new Date(expiresAt).getTime() - Date.now()
  if (remaining <= 0) return 'expired'
  if (remaining <= 24 * 60 * 60 * 1000) return 'expiringSoon'
  return 'available'
}

function retentionLabel(expiresAt: string): string {
  return t(`admin.desktopDiagnostics.labels.${retentionState(expiresAt)}`)
}

function retentionBadgeClass(expiresAt: string): string[] {
  const state = retentionState(expiresAt)
  return [
    'inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium',
    state === 'available'
      ? 'bg-sky-50 text-sky-700 dark:bg-sky-950/40 dark:text-sky-300'
      : state === 'expiringSoon'
        ? 'bg-amber-50 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300'
        : 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-dark-300',
  ]
}

async function loadDiagnostics(): Promise<void> {
  listController?.abort()
  const controller = new AbortController()
  listController = controller
  loading.value = true
  try {
    const result = await adminAPI.desktopDiagnostics.list({
      page: pagination.page,
      page_size: pagination.page_size,
    })
    if (controller.signal.aborted || listController !== controller) return
    diagnostics.value = result.items ?? []
    pagination.total = result.total ?? 0
  } catch (error: unknown) {
    if (controller.signal.aborted) return
    appStore.showError(extractApiErrorMessage(error, t('admin.desktopDiagnostics.loadFailed')))
  } finally {
    if (listController === controller) {
      listController = null
      loading.value = false
    }
  }
}

async function openDetail(item: DesktopDiagnosticMetadata): Promise<void> {
  detailController?.abort()
  const controller = new AbortController()
  detailController = controller
  selectedDetail.value = null
  detailVisible.value = true
  detailLoading.value = true
  try {
    const detail = await adminAPI.desktopDiagnostics.get(item.id)
    if (controller.signal.aborted || detailController !== controller) return
    selectedDetail.value = detail
  } catch (error: unknown) {
    if (controller.signal.aborted) return
    detailVisible.value = false
    appStore.showError(extractApiErrorMessage(error, t('admin.desktopDiagnostics.detailFailed')))
  } finally {
    if (detailController === controller) {
      detailController = null
      detailLoading.value = false
    }
  }
}

function closeDetail(): void {
  detailController?.abort()
  detailController = null
  detailVisible.value = false
  detailLoading.value = false
  selectedDetail.value = null
}

async function downloadDiagnostic(id: string): Promise<void> {
  downloadingID.value = id
  try {
    const blob = await adminAPI.desktopDiagnostics.download(id)
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `desktop-diagnostic-${id}.json`
    link.click()
    window.setTimeout(() => URL.revokeObjectURL(url), 0)
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.desktopDiagnostics.downloadFailed')))
  } finally {
    downloadingID.value = null
  }
}

function handlePageChange(page: number): void {
  pagination.page = page
  void loadDiagnostics()
}

function handlePageSizeChange(pageSize: number): void {
  pagination.page_size = pageSize
  pagination.page = 1
  void loadDiagnostics()
}

onMounted(() => void loadDiagnostics())
onBeforeUnmount(() => {
  listController?.abort()
  detailController?.abort()
})
</script>

<style scoped>
.diagnostic-count {
  @apply inline-flex min-w-7 items-center justify-center rounded-md bg-gray-100 px-2 py-0.5 text-xs font-semibold text-gray-600 dark:bg-dark-700 dark:text-dark-200;
}

.diagnostic-icon-button {
  @apply inline-flex h-8 w-8 flex-none items-center justify-center rounded-md text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 disabled:cursor-not-allowed disabled:opacity-40 dark:text-dark-300 dark:hover:bg-dark-700 dark:hover:text-white;
}

.diagnostic-audit-notice {
  @apply flex gap-2.5 rounded-lg border border-sky-200 bg-sky-50 px-4 py-3 text-sm leading-5 text-sky-800 dark:border-sky-900/70 dark:bg-sky-950/30 dark:text-sky-200;
}

.diagnostic-section-title {
  @apply mb-2 text-sm font-semibold text-gray-900 dark:text-white;
}

.diagnostic-summary-grid {
  @apply grid grid-cols-1 gap-px overflow-hidden rounded-lg border border-gray-200 bg-gray-200 sm:grid-cols-2 dark:border-dark-700 dark:bg-dark-700;
}

.diagnostic-summary-grid > div {
  @apply min-w-0 bg-white px-4 py-3 dark:bg-dark-800;
}

.diagnostic-summary-grid dt {
  @apply text-xs font-medium text-gray-500 dark:text-dark-300;
}

.diagnostic-summary-grid dd {
  @apply mt-1 break-words text-sm text-gray-900 dark:text-white;
}

.diagnostic-payload {
  @apply max-h-[42vh] overflow-auto rounded-lg border border-gray-200 bg-gray-950 p-4 text-xs leading-5 text-gray-100 dark:border-dark-600;
  tab-size: 2;
}
</style>

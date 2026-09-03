<template>
  <AppLayout variant="home-clay" content-mode="workbench">
    <div
      class="usage-workbench"
      data-admin-page-kind="ops"
      data-testid="usage-workbench"
    >
      <header class="usage-workbench__header">
        <AdminPageHeader
          class="usage-workbench__page-header"
          :title="t('admin.usage.title')"
          :description="t('admin.usage.description')"
        >
          <template #secondary-actions>
            <button
              type="button"
              class="usage-workbench__header-button usage-workbench__header-button--secondary"
              :disabled="loading"
              @click="refreshData"
            >
              <Icon name="refresh" size="sm" aria-hidden="true" />
              <span>{{ t('common.refresh') }}</span>
            </button>
          </template>
          <template v-if="activeTab === 'usage'" #primary-actions>
            <button
              type="button"
              class="usage-workbench__header-button usage-workbench__header-button--primary"
              :disabled="exporting"
              @click="exportToExcel"
            >
              <Icon name="download" size="sm" aria-hidden="true" />
              <span>{{ t('usage.exportExcel') }}</span>
            </button>
          </template>
        </AdminPageHeader>

        <UsageStatsCards v-if="activeTab === 'usage'" :stats="usageStats" credit-mode />
      </header>

      <div class="usage-workbench__body">
        <aside
          class="usage-workbench__filter-rail"
          data-testid="usage-filter-rail"
          :aria-label="t('admin.usage.workspace.filtersLabel')"
        >
          <div class="usage-workbench__filter-scroll">
            <section class="usage-workbench__filter-section">
              <h2 class="usage-workbench__filter-heading">
                {{ t('admin.usage.workspace.periodTitle') }}
              </h2>
              <div class="usage-workbench__scope-field">
                <span class="usage-workbench__scope-label">
                  {{ t('admin.dashboard.timeRange') }}
                </span>
                <DateRangePicker
                  v-model:start-date="startDate"
                  v-model:end-date="endDate"
                  @change="onDateRangeChange"
                />
              </div>
              <div v-if="activeTab === 'usage'" class="usage-workbench__scope-field">
                <span class="usage-workbench__scope-label">
                  {{ t('admin.dashboard.granularity') }}
                </span>
                <Select v-model="granularity" :options="granularityOptions" @change="loadChartData" />
              </div>
            </section>

            <section v-if="activeTab !== 'billing'" class="usage-workbench__filter-section">
              <h2 class="usage-workbench__filter-heading">
                {{ t('admin.usage.workspace.auditObjectsTitle') }}
              </h2>
              <UsageFilters
                v-model="filters"
                ref="usageFiltersRef"
                flat
                layout="rail"
                :show-actions="false"
                :mode="usageFilterMode"
                :start-date="startDate"
                :end-date="endDate"
                :exporting="exporting"
                :model-options="modelNameOptions"
                @change="applyFilters"
                @refresh="refreshData"
                @reset="resetFilters"
                @cleanup="openCleanupDialog"
                @export="exportToExcel"
              />
            </section>

            <section class="usage-workbench__filter-section">
              <h2 class="usage-workbench__filter-heading">
                {{ t('admin.usage.workspace.conditionsTitle') }}
              </h2>
              <p class="usage-workbench__filter-status">
                {{ activeFilterCount > 0
                  ? t('admin.usage.workspace.activeFilters', { count: activeFilterCount })
                  : t('admin.usage.workspace.noActiveFilters') }}
              </p>
            </section>
          </div>

          <footer class="usage-workbench__filter-actions">
            <button type="button" class="btn btn-primary w-full" @click="applyFilters">
              {{ t('admin.usage.workspace.applyFilters') }}
            </button>
            <button type="button" class="btn btn-secondary w-full" @click="resetFilters">
              {{ t('admin.usage.workspace.resetLedger') }}
            </button>
            <div v-if="activeTab === 'usage'" class="usage-workbench__danger-zone">
              <span class="usage-workbench__danger-label">
                {{ t('admin.usage.workspace.dangerTitle') }}
              </span>
              <button type="button" class="btn btn-danger w-full" @click="openCleanupDialog">
                {{ t('admin.usage.cleanup.button') }}
              </button>
            </div>
          </footer>
        </aside>

        <section
          class="usage-workbench__evidence"
          data-testid="usage-evidence-area"
          :aria-label="t('admin.usage.workspace.evidenceLabel')"
        >
          <section class="usage-workbench__surface" :aria-label="activeTabMeta.label">
            <header
              class="usage-workbench__toolbar"
              data-testid="usage-tab-toolbar"
            >
              <nav
                class="usage-workbench__tabs"
                role="tablist"
                :aria-label="t('admin.usage.workspace.tabsLabel')"
              >
                <button
                  v-for="tab in detailTabs"
                  :key="tab.key"
                  :id="'usage-tab-' + tab.key"
                  type="button"
                  role="tab"
                  data-testid="usage-detail-tab"
                  class="usage-workbench__tab"
                  :class="{ 'usage-workbench__tab--active': activeTab === tab.key }"
                  :aria-current="activeTab === tab.key ? 'page' : undefined"
                  :aria-selected="activeTab === tab.key"
                  :aria-controls="'usage-panel-' + tab.key"
                  @click="switchTab(tab.key)"
                >
                  {{ tab.label }}
                </button>
              </nav>

              <div class="usage-workbench__toolbar-actions">
                <span v-if="activeTab === 'usage' && pagination.total > 0" class="usage-workbench__result-count">
                  {{ t('admin.usage.workspace.resultCount', { count: pagination.total.toLocaleString() }) }}
                </span>
                <span v-else-if="activeTab === 'errors' && errTotal > 0" class="usage-workbench__result-count usage-workbench__result-count--danger">
                  {{ t('admin.usage.workspace.resultCount', { count: errTotal.toLocaleString() }) }}
                </span>

                <div
                  v-if="activeTab !== 'ranking' && activeTab !== 'billing'"
                  ref="columnDropdownRef"
                  class="usage-workbench__column-control"
                >
                  <button
                    type="button"
                    class="usage-workbench__tool-button"
                    :title="t('admin.users.columnSettings')"
                    :aria-expanded="showColumnDropdown"
                    @click="showColumnDropdown = !showColumnDropdown"
                  >
                    <Icon name="slidersHorizontal" size="sm" aria-hidden="true" />
                    <span>{{ t('admin.users.columnSettings') }}</span>
                  </button>
                  <div v-if="showColumnDropdown" class="usage-workbench__column-menu">
                    <button
                      v-for="col in currentToggleableColumns"
                      :key="col.key"
                      type="button"
                      class="usage-workbench__column-option"
                      @click="toggleCurrentColumn(col.key)"
                    >
                      <span>{{ col.label }}</span>
                      <Icon
                        v-if="isCurrentColumnVisible(col.key)"
                        name="check"
                        size="sm"
                        class="text-primary-500"
                        :stroke-width="2"
                      />
                    </button>
                  </div>
                </div>

                <div v-if="activeTab === 'usage'" class="usage-workbench__density" role="group" :aria-label="t('admin.usage.workspace.densityLabel')">
                  <button
                    type="button"
                    class="usage-workbench__density-button"
                    :class="{ 'usage-workbench__density-button--active': tableDensity === 'compact' }"
                    :aria-pressed="tableDensity === 'compact'"
                    :title="t('admin.usage.workspace.compactView')"
                    @click="tableDensity = 'compact'"
                  >
                    <Icon name="grid" size="sm" aria-hidden="true" />
                  </button>
                  <button
                    type="button"
                    class="usage-workbench__density-button"
                    :class="{ 'usage-workbench__density-button--active': tableDensity === 'comfortable' }"
                    :aria-pressed="tableDensity === 'comfortable'"
                    :title="t('admin.usage.workspace.comfortableView')"
                    @click="tableDensity = 'comfortable'"
                  >
                    <Icon name="menu" size="sm" aria-hidden="true" />
                  </button>
                </div>
              </div>
            </header>

            <div
              v-if="activeTab === 'usage'"
              :id="'usage-panel-' + activeTab"
              class="usage-workbench__panel usage-workbench__panel--usage"
              role="tabpanel"
              :aria-labelledby="'usage-tab-' + activeTab"
              :class="{ 'usage-workbench__panel--comfortable': tableDensity === 'comfortable' }"
            >
              <UsageTable
                flat
                audit-layout
                :data="usageLogs"
                :loading="loading"
                :columns="visibleColumns"
                :server-side-sort="true"
                :default-sort-key="'created_at'"
                :default-sort-order="'desc'"
                credit-mode
                show-upstream-model-audit
                @sort="handleSort"
                @userClick="handleUserClick"
                @ipGeoBatchFailed="handleIpGeoBatchFailed"
              />
              <Pagination
                v-if="pagination.total > 0"
                :page="pagination.page"
                :total="pagination.total"
                :page-size="pagination.page_size"
                @update:page="handlePageChange"
                @update:pageSize="handlePageSizeChange"
              />
            </div>
            <div v-else-if="activeTab === 'errors'" :id="'usage-panel-' + activeTab" class="usage-workbench__panel" role="tabpanel" :aria-labelledby="'usage-tab-' + activeTab">
              <OpsErrorLogTable
                flat
                :rows="errRows"
                :total="errTotal"
                :loading="errLoading"
                :page="errPage"
                :page-size="errPageSize"
                :visible-column-keys="errVisibleColumnKeys"
                user-clickable
                @userClick="handleUserClick"
                @openErrorDetail="openError"
                @sort="onErrSort"
                @update:page="onErrPage"
                @update:pageSize="onErrPageSize"
                @ipGeoBatchFailed="handleIpGeoBatchFailed"
              />
            </div>
            <!-- 懒挂载：首次切到该 tab 才请求排行数据，之后随筛选自动刷新 -->
            <div v-else-if="activeTab === 'ranking' && rankingMounted" :id="'usage-panel-' + activeTab" class="usage-workbench__panel" role="tabpanel" :aria-labelledby="'usage-tab-' + activeTab">
              <UserTokenRanking
                ref="rankingRef"
                :start-date="startDate"
                :end-date="endDate"
                :filters="breakdownFilters"
                :model="filters.model"
                :initial-sort-by="rankingInitialSortBy"
                @select-user="handleRankingSelectUser"
              />
            </div>
            <div v-else-if="activeTab === 'billing' && billingMounted" :id="'usage-panel-' + activeTab" class="usage-workbench__panel" role="tabpanel" :aria-labelledby="'usage-tab-' + activeTab">
              <AdminBillingReceiptsPanel
                ref="billingReceiptsRef"
                :start-date="startDate"
                :end-date="endDate"
              />
            </div>
          </section>

          <details
            v-if="activeTab === 'usage'"
            class="usage-workbench__analytics"
            data-testid="usage-analytics"
            open
          >
            <summary class="usage-workbench__analytics-summary">
              <span class="usage-workbench__analytics-heading-wrap">
                <Icon name="chart" size="sm" aria-hidden="true" />
                <span id="usage-analytics-heading" class="usage-workbench__analytics-title">
                  {{ t('admin.usage.workspace.analyticsTitle') }}
                </span>
              </span>
              <Icon name="chevronDown" size="sm" class="usage-workbench__analytics-chevron" aria-hidden="true" />
            </summary>
            <p class="usage-workbench__analytics-description">
              {{ t('admin.usage.workspace.analyticsDescription') }}
            </p>
            <div class="usage-workbench__chart-grid" aria-labelledby="usage-analytics-heading">
              <TokenUsageTrend
                class="usage-workbench__chart-wide"
                variant="home-clay"
                :trend-data="trendData"
                :loading="chartsLoading"
                credit-mode
              />
              <ModelDistributionChart
                variant="home-clay"
                v-model:source="modelDistributionSource"
                v-model:metric="modelDistributionMetric"
                :model-stats="requestedModelStats"
                :upstream-model-stats="upstreamModelStats"
                :mapping-model-stats="mappingModelStats"
                :loading="modelStatsLoading"
                :show-source-toggle="true"
                :show-metric-toggle="true"
                credit-mode
                :start-date="startDate"
                :end-date="endDate"
                :filters="breakdownFilters"
              />
              <GroupDistributionChart
                v-model:metric="groupDistributionMetric"
                :group-stats="groupStats"
                :loading="chartsLoading"
                :show-metric-toggle="true"
                credit-mode
                :start-date="startDate"
                :end-date="endDate"
                :filters="breakdownFilters"
              />
              <EndpointDistributionChart
                v-model:source="endpointDistributionSource"
                v-model:metric="endpointDistributionMetric"
                :endpoint-stats="inboundEndpointStats"
                :upstream-endpoint-stats="upstreamEndpointStats"
                :endpoint-path-stats="endpointPathStats"
                :loading="endpointStatsLoading"
                :show-source-toggle="true"
                :show-metric-toggle="true"
                credit-mode
                :title="t('usage.endpointDistribution')"
                :start-date="startDate"
                :end-date="endDate"
                :filters="breakdownFilters"
              />
            </div>
          </details>
        </section>
      </div>

      <OpsErrorDetailModal v-model:show="showErrorModal" :error-id="selectedErrorId" :error-type="'request'" />
    </div>
  </AppLayout>
  <UsageExportProgress :show="exportProgress.show" :progress="exportProgress.progress" :current="exportProgress.current" :total="exportProgress.total" :estimated-time="exportProgress.estimatedTime" @cancel="cancelExport" />
  <UsageCleanupDialog
    :show="cleanupDialogVisible"
    :filters="filters"
    :start-date="startDate"
    :end-date="endDate"
    @close="cleanupDialogVisible = false"
  />
  <!-- Balance history modal triggered from usage table user click -->
  <UserBalanceHistoryModal
    :show="showBalanceHistoryModal"
    :user="balanceHistoryUser"
    :hide-actions="true"
    @close="showBalanceHistoryModal = false; balanceHistoryUser = null"
  />
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { saveAs } from 'file-saver'
import { useRoute } from 'vue-router'
import { useAppStore } from '@/stores/app'; import { adminAPI } from '@/api/admin'; import { adminUsageAPI } from '@/api/admin/usage'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { formatReasoningEffort } from '@/utils/format'
import { resolveUsageRequestType, requestTypeToLegacyStream } from '@/utils/usageRequestType'
import AppLayout from '@/components/layout/AppLayout.vue'; import Pagination from '@/components/common/Pagination.vue'; import Select from '@/components/common/Select.vue'; import DateRangePicker from '@/components/common/DateRangePicker.vue'
import AdminPageHeader from '@/components/layout/AdminPageHeader.vue'
import UsageStatsCards from '@/components/shared-domain/usage/UsageStatsCards.vue'; import UsageFilters from '@/components/admin/usage/UsageFilters.vue'
import UsageTable from '@/components/shared-domain/usage/UsageTable.vue'; import UsageExportProgress from '@/components/admin/usage/UsageExportProgress.vue'
import UserTokenRanking from '@/components/admin/usage/UserTokenRanking.vue'
import AdminBillingReceiptsPanel from '@/components/admin/usage/AdminBillingReceiptsPanel.vue'
import UsageCleanupDialog from '@/components/admin/usage/UsageCleanupDialog.vue'
import UserBalanceHistoryModal from '@/components/admin/user/UserBalanceHistoryModal.vue'
import OpsErrorLogTable from '@/views/admin/ops/components/OpsErrorLogTable.vue'
import OpsErrorDetailModal from '@/views/admin/ops/components/OpsErrorDetailModal.vue'
import { listErrorLogs } from '@/api/admin/ops'
import type { OpsErrorLog } from '@/api/admin/ops'
import ModelDistributionChart from '@/components/charts/ModelDistributionChart.vue'; import GroupDistributionChart from '@/components/charts/GroupDistributionChart.vue'; import TokenUsageTrend from '@/components/charts/TokenUsageTrend.vue'
import EndpointDistributionChart from '@/components/charts/EndpointDistributionChart.vue'
import Icon from '@/components/icons/Icon.vue'
import type { AdminUsageLog, TrendDataPoint, ModelStat, GroupStat, EndpointStat, AdminUser } from '@/types'; import type { AdminUsageStatsResponse, AdminUsageQueryParams } from '@/api/admin/usage'

const { t } = useI18n()
const appStore = useAppStore()
type DistributionMetric = 'tokens' | 'actual_cost'
type EndpointSource = 'inbound' | 'upstream' | 'path'
type ModelDistributionSource = 'requested' | 'upstream' | 'mapping'
const RANKING_SORT_KEYS = ['requests', 'input_tokens', 'output_tokens', 'cache_tokens', 'total_tokens', 'actual_cost'] as const
type RankingSortKey = typeof RANKING_SORT_KEYS[number]
const route = useRoute()
const usageStats = ref<AdminUsageStatsResponse | null>(null); const usageLogs = ref<AdminUsageLog[]>([]); const loading = ref(false); const exporting = ref(false)
const trendData = ref<TrendDataPoint[]>([]); const requestedModelStats = ref<ModelStat[]>([]); const upstreamModelStats = ref<ModelStat[]>([]); const mappingModelStats = ref<ModelStat[]>([]); const groupStats = ref<GroupStat[]>([]); const chartsLoading = ref(false); const modelStatsLoading = ref(false); const granularity = ref<'day' | 'hour'>('hour')
const modelDistributionMetric = ref<DistributionMetric>('tokens')
const modelDistributionSource = ref<ModelDistributionSource>('requested')
const loadedModelSources = reactive<Record<ModelDistributionSource, boolean>>({
  requested: false,
  upstream: false,
  mapping: false,
})
const groupDistributionMetric = ref<DistributionMetric>('tokens')
const endpointDistributionMetric = ref<DistributionMetric>('tokens')
const endpointDistributionSource = ref<EndpointSource>('inbound')
const inboundEndpointStats = ref<EndpointStat[]>([])
const upstreamEndpointStats = ref<EndpointStat[]>([])
const endpointPathStats = ref<EndpointStat[]>([])
const endpointStatsLoading = ref(false)
let abortController: AbortController | null = null; let exportAbortController: AbortController | null = null
let chartReqSeq = 0
let statsReqSeq = 0
let modelStatsReqSeq = 0
let errorReqSeq = 0
const exportProgress = reactive({ show: false, progress: 0, current: 0, total: 0, estimatedTime: '' })
const cleanupDialogVisible = ref(false)
// Balance history modal state
const showBalanceHistoryModal = ref(false)
const balanceHistoryUser = ref<AdminUser | null>(null)

const breakdownFilters = computed(() => {
  const f: Record<string, any> = {}
  if (filters.value.user_id) f.user_id = filters.value.user_id
  if (filters.value.api_key_id) f.api_key_id = filters.value.api_key_id
  if (filters.value.account_id) f.account_id = filters.value.account_id
  if (filters.value.group_id) f.group_id = filters.value.group_id
  if (filters.value.request_type != null) f.request_type = filters.value.request_type
  if (filters.value.billing_type != null) f.billing_type = filters.value.billing_type
  return f
})

const modelNameOptions = computed(() =>
  Array.from(new Set(requestedModelStats.value.map((m) => m.model).filter(Boolean))).sort()
)

const handleUserClick = async (userId: number) => {
  try {
    const user = await adminAPI.users.getById(userId, true)
    balanceHistoryUser.value = user
    showBalanceHistoryModal.value = true
  } catch {
    appStore.showError(t('admin.usage.failedToLoadUser'))
  }
}

// Drill down from the per-user token ranking: scope the whole usage view to
// that user and jump to the usage-detail tab so the drill-down is visible.
const handleRankingSelectUser = (userId: number, email: string) => {
  filters.value = { ...filters.value, user_id: userId }
  usageFiltersRef.value?.setUserKeyword?.(email || '')
  activeTab.value = 'usage'
  applyFilters()
}

const granularityOptions = computed(() => [{ value: 'day', label: t('admin.dashboard.day') }, { value: 'hour', label: t('admin.dashboard.hour') }])
// Use local timezone to avoid UTC timezone issues
const formatLD = (d: Date) => {
  const year = d.getFullYear()
  const month = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}
const getLast24HoursRangeDates = (): { start: string; end: string } => {
  const end = new Date()
  const start = new Date(end.getTime() - 24 * 60 * 60 * 1000)
  return {
    start: formatLD(start),
    end: formatLD(end)
  }
}
const getGranularityForRange = (start: string, end: string): 'day' | 'hour' => {
  const startTime = new Date(`${start}T00:00:00`).getTime()
  const endTime = new Date(`${end}T00:00:00`).getTime()
  const daysDiff = Math.ceil((endTime - startTime) / (1000 * 60 * 60 * 24))
  return daysDiff <= 1 ? 'hour' : 'day'
}
const defaultRange = getLast24HoursRangeDates()
const startDate = ref(defaultRange.start); const endDate = ref(defaultRange.end)
const filters = ref<AdminUsageQueryParams>({ user_id: undefined, model: undefined, group_id: undefined, request_type: undefined, billing_type: null, start_date: startDate.value, end_date: endDate.value })
const activeFilterCount = computed(() =>
  Object.entries(filters.value).filter(([key, value]) =>
    key !== 'start_date'
    && key !== 'end_date'
    && value !== undefined
    && value !== null
    && value !== ''
  ).length
)
const pagination = reactive({ page: 1, page_size: getPersistedPageSize(), total: 0 })
const sortState = reactive({
  sort_by: 'created_at',
  sort_order: 'desc' as 'asc' | 'desc'
})
const rankingInitialSortBy = ref<RankingSortKey>('total_tokens')

const getSingleQueryValue = (value: string | null | Array<string | null> | undefined): string | undefined => {
  if (Array.isArray(value)) return value.find((item): item is string => typeof item === 'string' && item.length > 0)
  return typeof value === 'string' && value.length > 0 ? value : undefined
}

const getNumericQueryValue = (value: string | null | Array<string | null> | undefined): number | undefined => {
  const raw = getSingleQueryValue(value)
  if (!raw) return undefined
  const parsed = Number(raw)
  return Number.isFinite(parsed) ? parsed : undefined
}

const getRankingSortQueryValue = (
  value: string | null | Array<string | null> | undefined
): RankingSortKey | undefined => {
  const raw = getSingleQueryValue(value)
  return RANKING_SORT_KEYS.find((key) => key === raw)
}

const applyRouteQueryFilters = () => {
  const queryStartDate = getSingleQueryValue(route.query.start_date)
  const queryEndDate = getSingleQueryValue(route.query.end_date)
  const queryUserId = getNumericQueryValue(route.query.user_id)
  const queryTab = getSingleQueryValue(route.query.tab)
  const queryRankingSort = getRankingSortQueryValue(route.query.sort_by)

  if (queryStartDate) {
    startDate.value = queryStartDate
  }
  if (queryEndDate) {
    endDate.value = queryEndDate
  }

  filters.value = {
    ...filters.value,
    user_id: queryUserId,
    start_date: startDate.value,
    end_date: endDate.value
  }
  granularity.value = getGranularityForRange(startDate.value, endDate.value)
  rankingInitialSortBy.value = queryRankingSort ?? 'total_tokens'

  if (queryTab === 'errors') {
    activeTab.value = 'errors'
  } else if (queryTab === 'ranking') {
    activeTab.value = 'ranking'
    rankingMounted.value = true
  } else if (queryTab === 'billing') {
    activeTab.value = 'billing'
    billingMounted.value = true
  }
}

const onDateRangeChange = (range: { startDate: string; endDate: string; preset: string | null }) => {
  startDate.value = range.startDate
  endDate.value = range.endDate
  filters.value = {
    ...filters.value,
    start_date: range.startDate,
    end_date: range.endDate
  }
  granularity.value = getGranularityForRange(range.startDate, range.endDate)
  applyFilters()
}

const buildUsageListParams = (
  page: number,
  pageSize: number,
  exactTotal: boolean
): AdminUsageQueryParams => {
  const requestType = filters.value.request_type
  const legacyStream = requestType ? requestTypeToLegacyStream(requestType) : filters.value.stream
  return {
    page,
    page_size: pageSize,
    exact_total: exactTotal,
    ...filters.value,
    stream: legacyStream === null ? undefined : legacyStream,
    sort_by: sortState.sort_by,
    sort_order: sortState.sort_order
  }
}

const loadLogs = async () => {
  abortController?.abort(); const c = new AbortController(); abortController = c; loading.value = true
  try {
    const res = await adminAPI.usage.list(
      buildUsageListParams(pagination.page, pagination.page_size, false),
      { signal: c.signal }
    )
    if(!c.signal.aborted) { usageLogs.value = res.items; pagination.total = res.total }
  } catch (error: any) { if(error?.name !== 'AbortError') console.error('Failed to load usage logs:', error) } finally { if(abortController === c) loading.value = false }
}
const loadStats = async (force = false) => {
  const seq = ++statsReqSeq
  endpointStatsLoading.value = true
  try {
    const requestType = filters.value.request_type
    const legacyStream = requestType ? requestTypeToLegacyStream(requestType) : filters.value.stream
    const s = await adminAPI.usage.getStats({
      ...filters.value,
      stream: legacyStream === null ? undefined : legacyStream,
      ...(force ? { nocache: 1 } : {}),
    })
    if (seq !== statsReqSeq) return
    usageStats.value = s
    inboundEndpointStats.value = s.endpoints || []
    upstreamEndpointStats.value = s.upstream_endpoints || []
    endpointPathStats.value = s.endpoint_paths || []
  } catch (error) {
    if (seq !== statsReqSeq) return
    console.error('Failed to load usage stats:', error)
    inboundEndpointStats.value = []
    upstreamEndpointStats.value = []
    endpointPathStats.value = []
  } finally {
    if (seq === statsReqSeq) endpointStatsLoading.value = false
  }
}

// 失效模型统计缓存:仅标记需要重取,保留旧数据直到新数据到达(避免刷新时图表闪空)。
const invalidateModelStatsCache = () => {
  loadedModelSources.requested = false
  loadedModelSources.upstream = false
  loadedModelSources.mapping = false
}

const loadModelStats = async (source: ModelDistributionSource, force = false) => {
  if (!force && loadedModelSources[source]) {
    return
  }

  const seq = ++modelStatsReqSeq
  modelStatsLoading.value = true
  try {
    const requestType = filters.value.request_type
    const legacyStream = requestType ? requestTypeToLegacyStream(requestType) : filters.value.stream
    const baseParams = {
      start_date: filters.value.start_date || startDate.value,
      end_date: filters.value.end_date || endDate.value,
      user_id: filters.value.user_id,
      model: filters.value.model,
      api_key_id: filters.value.api_key_id,
      account_id: filters.value.account_id,
      group_id: filters.value.group_id,
      request_type: requestType,
      stream: legacyStream === null ? undefined : legacyStream,
      billing_type: filters.value.billing_type,
      upstream_model_mismatch: filters.value.upstream_model_mismatch,
    }

    const response = await adminAPI.dashboard.getModelStats({ ...baseParams, model_source: source })

    if (seq !== modelStatsReqSeq) return

    const models = response.models || []
    if (source === 'requested') {
      requestedModelStats.value = models
    } else if (source === 'upstream') {
      upstreamModelStats.value = models
    } else {
      mappingModelStats.value = models
    }
    loadedModelSources[source] = true
  } catch (error) {
    if (seq !== modelStatsReqSeq) return
    console.error('Failed to load model stats:', error)
    if (source === 'requested') {
      requestedModelStats.value = []
    } else if (source === 'upstream') {
      upstreamModelStats.value = []
    } else {
      mappingModelStats.value = []
    }
    loadedModelSources[source] = false
  } finally {
    if (seq === modelStatsReqSeq) modelStatsLoading.value = false
  }
}

const loadChartData = async () => {
  const seq = ++chartReqSeq
  chartsLoading.value = true
  try {
    const requestType = filters.value.request_type
    const legacyStream = requestType ? requestTypeToLegacyStream(requestType) : filters.value.stream
    const snapshot = await adminAPI.dashboard.getSnapshotV2({
      start_date: filters.value.start_date || startDate.value,
      end_date: filters.value.end_date || endDate.value,
      granularity: granularity.value,
      user_id: filters.value.user_id,
      model: filters.value.model,
      api_key_id: filters.value.api_key_id,
      account_id: filters.value.account_id,
      group_id: filters.value.group_id,
      request_type: requestType,
      stream: legacyStream === null ? undefined : legacyStream,
      billing_type: filters.value.billing_type,
      upstream_model_mismatch: filters.value.upstream_model_mismatch,
      include_stats: false,
      include_trend: true,
      include_model_stats: false,
      include_group_stats: true,
      include_users_trend: false
    })
    if (seq !== chartReqSeq) return
    trendData.value = snapshot.trend || []
    groupStats.value = snapshot.groups || []
  } catch (error) { console.error('Failed to load chart data:', error) } finally { if (seq === chartReqSeq) chartsLoading.value = false }
}
const applyFilters = () => {
  pagination.page = 1
  invalidateModelStatsCache()
  loadLogs()
  loadStats()
  loadModelStats(modelDistributionSource.value, true)
  loadChartData()
  errPage.value = 1
  if (activeTab.value === 'errors') {
    loadAdminErrors()
  } else {
    errRows.value = []
  }
}
const refreshData = () => {
  invalidateModelStatsCache()
  loadLogs()
  loadStats(true)
  loadModelStats(modelDistributionSource.value, true)
  loadChartData()
  if (activeTab.value === 'errors') loadAdminErrors()
  if (rankingMounted.value) rankingRef.value?.reload()
  if (billingMounted.value) billingReceiptsRef.value?.reload()
}
const resetFilters = () => {
  const range = getLast24HoursRangeDates()
  startDate.value = range.start
  endDate.value = range.end
  filters.value = { start_date: startDate.value, end_date: endDate.value, request_type: undefined, billing_type: null, billing_mode: undefined }
  granularity.value = getGranularityForRange(startDate.value, endDate.value)
  applyFilters()
}
const handlePageChange = (p: number) => { pagination.page = p; loadLogs() }
const handlePageSizeChange = (s: number) => { pagination.page_size = s; pagination.page = 1; loadLogs() }
const handleSort = (key: string, order: 'asc' | 'desc') => {
  sortState.sort_by = key
  sortState.sort_order = order
  pagination.page = 1
  loadLogs()
}

const handleIpGeoBatchFailed = () => {
  appStore.showError(t('usage.ipGeo.batchFailed'))
}
const cancelExport = () => exportAbortController?.abort()
const openCleanupDialog = () => { cleanupDialogVisible.value = true }
const getRequestTypeLabel = (log: AdminUsageLog): string => {
  const requestType = resolveUsageRequestType(log)
  if (requestType === 'cyber') return t('usage.cyber')
  if (requestType === 'ws_v2') return t('usage.ws')
  if (requestType === 'stream') return t('usage.stream')
  if (requestType === 'sync') return t('usage.sync')
  return t('usage.unknown')
}

const exportToExcel = async () => {
  if (exporting.value) return; exporting.value = true; exportProgress.show = true
  const c = new AbortController(); exportAbortController = c
  try {
    let p = 1; let total = pagination.total; let exportedCount = 0
    const XLSX = await import('xlsx')
    const headers = [
      t('usage.time'), t('admin.usage.user'), t('usage.apiKeyFilter'),
      t('admin.usage.account'), t('usage.requestedModel'), t('usage.sentUpstreamModel'), t('usage.upstreamResponseModel'), t('usage.upstreamModelMismatch'), t('usage.reasoningEffort'), t('admin.usage.group'),
      t('usage.inboundEndpoint'), t('usage.upstreamEndpoint'),
      t('usage.type'),
      t('admin.usage.inputTokens'), t('admin.usage.outputTokens'),
      t('admin.usage.cacheReadTokens'), t('admin.usage.cacheCreationTokens'),
      t('admin.usage.inputCost'), t('admin.usage.outputCost'),
      t('admin.usage.cacheReadCost'), t('admin.usage.cacheCreationCost'),
      t('usage.rate'), t('usage.accountMultiplier'), t('usage.original'), t('usage.userBilled'), t('usage.accountBilled'),
      t('usage.firstToken'), t('usage.duration'),
      t('admin.usage.requestId'), t('usage.userAgent'), t('admin.usage.ipAddress')
    ]
    const ws = XLSX.utils.aoa_to_sheet([headers])
    while (true) {
      const res = await adminUsageAPI.list(
        buildUsageListParams(p, 100, true),
        { signal: c.signal }
      )
      if (c.signal.aborted) break; if (p === 1) { total = res.total; exportProgress.total = total }
      const rows = (res.items || []).map((log: AdminUsageLog) => [
        log.created_at, log.user?.email || '', log.api_key?.name || '', log.account?.name || '', log.model,
        log.upstream_model || log.model, log.upstream_response_model || '', log.upstream_model_mismatch == null ? '' : t(log.upstream_model_mismatch ? 'common.yes' : 'common.no'), formatReasoningEffort(log.reasoning_effort), log.group?.name || '',
        log.inbound_endpoint || '', log.upstream_endpoint || '', getRequestTypeLabel(log),
        log.input_tokens, log.output_tokens, log.cache_read_tokens, log.cache_creation_tokens,
        log.input_cost?.toFixed(6) || '0.000000', log.output_cost?.toFixed(6) || '0.000000',
        log.cache_read_cost?.toFixed(6) || '0.000000', log.cache_creation_cost?.toFixed(6) || '0.000000',
        log.rate_multiplier?.toPrecision(4) || '1.00', (log.account_rate_multiplier ?? 1).toPrecision(4),
        log.total_cost?.toFixed(6) || '0.000000', log.actual_cost?.toFixed(6) || '0.000000',
        ((log.account_stats_cost ?? log.total_cost) * (log.account_rate_multiplier ?? 1)).toFixed(6), log.first_token_ms ?? '', log.duration_ms,
        log.request_id || '', log.user_agent || '', log.ip_address || ''
      ])
      if (rows.length) {
        XLSX.utils.sheet_add_aoa(ws, rows, { origin: -1 })
      }
      exportedCount += rows.length
      exportProgress.current = exportedCount
      exportProgress.progress = total > 0 ? Math.min(100, Math.round(exportedCount / total * 100)) : 0
      if (exportedCount >= total || res.items.length < 100) break; p++
    }
    if(!c.signal.aborted) {
      const wb = XLSX.utils.book_new()
      XLSX.utils.book_append_sheet(wb, ws, 'Usage')
      saveAs(new Blob([XLSX.write(wb, { bookType: 'xlsx', type: 'array' })], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' }), `usage_${filters.value.start_date}_to_${filters.value.end_date}.xlsx`)
      appStore.showSuccess(t('usage.exportSuccess'))
    }
  } catch (error) { console.error('Failed to export:', error); appStore.showError('Export Failed') }
  finally { if(exportAbortController === c) { exportAbortController = null; exporting.value = false; exportProgress.show = false } }
}

// Column visibility
const ALWAYS_VISIBLE = ['user', 'created_at']
const DEFAULT_HIDDEN_COLUMNS = [
  'api_key',
  'reasoning_effort',
  'endpoint',
  'group',
  'stream',
  'billing_mode',
  'user_agent',
]
const HIDDEN_COLUMNS_KEY = 'usage-hidden-columns-v2'

const allColumns = computed(() => [
  {
    key: 'user',
    label: t('admin.usage.workspace.columns.auditSubject'),
    sortable: false,
    class: 'usage-audit-col--subject min-w-[220px] max-w-[240px]',
  },
  {
    key: 'model',
    label: t('admin.usage.workspace.columns.modelMapping'),
    sortable: true,
    class: 'usage-audit-col--model min-w-[150px]',
  },
  {
    key: 'tokens',
    label: t('admin.usage.workspace.columns.tokenPayload'),
    sortable: false,
    class: 'usage-audit-col--tokens text-right',
  },
  {
    key: 'cost',
    label: t('admin.usage.workspace.columns.actualCharge'),
    sortable: false,
    class: 'usage-audit-col--cost text-right',
  },
  {
    key: 'latency',
    label: t('admin.usage.workspace.columns.latencyResponse'),
    sortable: false,
    class: 'usage-audit-col--latency text-right',
  },
  {
    key: 'account',
    label: t('admin.usage.workspace.columns.channelAccount'),
    sortable: false,
    class: 'usage-audit-col--account max-w-[120px] text-right',
  },
  {
    key: 'created_at',
    label: t('admin.usage.workspace.columns.timeSerial'),
    sortable: true,
    class: 'usage-audit-col--time min-w-[140px]',
  },
  {
    key: 'ip_address',
    label: t('admin.usage.workspace.columns.ipGeo'),
    sortable: false,
    class: 'usage-audit-col--ip min-w-[120px]',
  },
  { key: 'api_key', label: t('usage.apiKeyFilter'), sortable: false },
  { key: 'reasoning_effort', label: t('usage.reasoningEffort'), sortable: false },
  { key: 'endpoint', label: t('usage.endpoint'), sortable: false },
  { key: 'group', label: t('admin.usage.group'), sortable: false },
  { key: 'stream', label: t('usage.type'), sortable: false },
  { key: 'billing_mode', label: t('admin.usage.billingMode'), sortable: false },
  { key: 'user_agent', label: t('usage.userAgent'), sortable: false },
])

const hiddenColumns = reactive<Set<string>>(new Set())

const toggleableColumns = computed(() =>
  allColumns.value.filter(col => !ALWAYS_VISIBLE.includes(col.key))
)

const visibleColumns = computed(() =>
  allColumns.value.filter(col =>
    ALWAYS_VISIBLE.includes(col.key) || !hiddenColumns.has(col.key)
  )
)

const isColumnVisible = (key: string) => !hiddenColumns.has(key)

const toggleColumn = (key: string) => {
  if (hiddenColumns.has(key)) {
    hiddenColumns.delete(key)
  } else {
    hiddenColumns.add(key)
  }
  try {
    localStorage.setItem(HIDDEN_COLUMNS_KEY, JSON.stringify([...hiddenColumns]))
  } catch (e) {
    console.error('Failed to save columns:', e)
  }
}

// ---- 错误请求 tab 列设置(与用量明细同机制,独立存储) ----
const ERR_ALWAYS_VISIBLE = ['user', 'status', 'created_at', 'actions']
const ERR_DEFAULT_HIDDEN_COLUMNS = ['user_agent']
const ERR_HIDDEN_COLUMNS_KEY = 'usage-error-hidden-columns'

// key 集合须与 OpsErrorLogTable 内部 allColumns 一致
const errAllColumns = computed(() => [
  { key: 'user', label: t('admin.ops.errorLog.user') },
  { key: 'api_key', label: t('admin.ops.errorLog.apiKey') },
  { key: 'account', label: t('admin.ops.errorLog.account') },
  { key: 'platform', label: t('admin.ops.errorLog.platform') },
  { key: 'model', label: t('admin.ops.errorLog.model') },
  { key: 'endpoint', label: t('admin.ops.errorLog.endpoint') },
  { key: 'group', label: t('admin.ops.errorLog.group') },
  { key: 'type', label: t('admin.ops.errorLog.type') },
  { key: 'category', label: t('usage.errors.category') },
  { key: 'status', label: t('admin.ops.errorLog.status') },
  { key: 'message', label: t('admin.ops.errorLog.message') },
  { key: 'created_at', label: t('admin.ops.errorLog.time') },
  { key: 'user_agent', label: t('usage.userAgent') },
  { key: 'client_ip', label: t('admin.ops.errorLog.ip') },
  { key: 'actions', label: t('admin.ops.errorLog.action') },
])

const errHiddenColumns = reactive<Set<string>>(new Set())

const errToggleableColumns = computed(() =>
  errAllColumns.value.filter(col => !ERR_ALWAYS_VISIBLE.includes(col.key))
)

const errVisibleColumnKeys = computed(() =>
  errAllColumns.value
    .filter(col => ERR_ALWAYS_VISIBLE.includes(col.key) || !errHiddenColumns.has(col.key))
    .map(col => col.key)
)

const toggleErrColumn = (key: string) => {
  if (errHiddenColumns.has(key)) {
    errHiddenColumns.delete(key)
  } else {
    errHiddenColumns.add(key)
  }
  try {
    localStorage.setItem(ERR_HIDDEN_COLUMNS_KEY, JSON.stringify([...errHiddenColumns]))
  } catch (e) {
    console.error('Failed to save error columns:', e)
  }
}

const loadSavedErrColumns = () => {
  try {
    const saved = localStorage.getItem(ERR_HIDDEN_COLUMNS_KEY)
    const keys = saved ? (JSON.parse(saved) as string[]) : ERR_DEFAULT_HIDDEN_COLUMNS
    keys.forEach((key) => errHiddenColumns.add(key))
  } catch {
    ERR_DEFAULT_HIDDEN_COLUMNS.forEach((key) => errHiddenColumns.add(key))
  }
}

// 列设置下拉按当前 tab 分发
const currentToggleableColumns = computed(() =>
  activeTab.value === 'errors' ? errToggleableColumns.value : toggleableColumns.value
)
const isCurrentColumnVisible = (key: string) =>
  activeTab.value === 'errors' ? !errHiddenColumns.has(key) : isColumnVisible(key)
const toggleCurrentColumn = (key: string) =>
  activeTab.value === 'errors' ? toggleErrColumn(key) : toggleColumn(key)

const loadSavedColumns = () => {
  try {
    const saved = localStorage.getItem(HIDDEN_COLUMNS_KEY)
    if (saved) {
      (JSON.parse(saved) as string[]).forEach((key) => {
        hiddenColumns.add(key)
      })
    } else {
      DEFAULT_HIDDEN_COLUMNS.forEach((key) => {
        hiddenColumns.add(key)
      })
    }
  } catch {
    DEFAULT_HIDDEN_COLUMNS.forEach((key) => {
      hiddenColumns.add(key)
    })
  }
}

// Detail tabs
type DetailTab = 'usage' | 'errors' | 'ranking' | 'billing'
const activeTab = ref<DetailTab>('usage')
const tableDensity = ref<'compact' | 'comfortable'>('compact')
const detailTabs = computed(() => [
  {
    key: 'usage' as const,
    label: t('usage.tabs.usage'),
    icon: 'document' as const,
    description: t('admin.usage.workspace.tabs.usageDescription'),
    detail: t('admin.usage.workspace.tabs.usageDetail'),
  },
  {
    key: 'errors' as const,
    label: t('usage.tabs.errors'),
    icon: 'exclamationTriangle' as const,
    description: t('admin.usage.workspace.tabs.errorsDescription'),
    detail: t('admin.usage.workspace.tabs.errorsDetail'),
  },
  {
    key: 'ranking' as const,
    label: t('usage.tabs.ranking'),
    icon: 'chart' as const,
    description: t('admin.usage.workspace.tabs.rankingDescription'),
    detail: t('admin.usage.workspace.tabs.rankingDetail'),
  },
  {
    key: 'billing' as const,
    label: t('usage.tabs.billing'),
    icon: 'coins' as const,
    description: t('admin.usage.workspace.tabs.billingDescription'),
    detail: t('admin.usage.workspace.tabs.billingDetail'),
  },
])
const activeTabMeta = computed(() =>
  detailTabs.value.find((tab) => tab.key === activeTab.value) ?? detailTabs.value[0]!
)
const usageFilterMode = computed(() =>
  activeTab.value === 'billing' ? 'usage' : activeTab.value
)
const usageFiltersRef = ref<InstanceType<typeof UsageFilters> | null>(null)
const rankingMounted = ref(false)
const rankingRef = ref<InstanceType<typeof UserTokenRanking> | null>(null)
const billingMounted = ref(false)
const billingReceiptsRef = ref<InstanceType<typeof AdminBillingReceiptsPanel> | null>(null)

const switchTab = (tab: DetailTab) => {
  activeTab.value = tab
  if (tab === 'errors' && errRows.value.length === 0) loadAdminErrors()
  if (tab === 'ranking') rankingMounted.value = true
  if (tab === 'billing') billingMounted.value = true
}

// Error tab state
const errRows = ref<OpsErrorLog[]>([])
const errLoading = ref(false)
const errPage = ref(1)
const errPageSize = ref(20)
const errTotal = ref(0)
const errSortBy = ref('created_at')
const errSortOrder = ref<'asc' | 'desc'>('desc')
const showErrorModal = ref(false)
const selectedErrorId = ref<number | null>(null)

// 注意：'YYYY-MM-DDT00:00:00' 无时区后缀，按本地时区解析后再转 UTC——与页面其它日期处理语义一致，刻意如此，勿改成 'T00:00:00Z'
const toRFC3339 = (d: string | undefined, endOfDay = false): string | undefined =>
  d ? new Date(d + (endOfDay ? 'T23:59:59.999' : 'T00:00:00')).toISOString() : undefined

const loadAdminErrors = async () => {
  const seq = ++errorReqSeq
  errLoading.value = true
  try {
    const resp = await listErrorLogs({
      page: errPage.value,
      page_size: errPageSize.value,
      view: 'all',
      start_time: toRFC3339(filters.value.start_date),
      end_time: toRFC3339(filters.value.end_date, true),
      user_id: filters.value.user_id ?? undefined,
      api_key_id: filters.value.api_key_id ?? undefined,
      account_id: filters.value.account_id ?? undefined,
      group_id: filters.value.group_id ?? undefined,
      model: filters.value.model || undefined,
      phase: filters.value.error_phase || undefined,
      category: filters.value.error_category || undefined,
      status_codes: filters.value.status_code != null ? String(filters.value.status_code) : undefined,
      sort_by: errSortBy.value,
      sort_order: errSortOrder.value,
    })
    if (seq !== errorReqSeq) return
    errRows.value = resp.items
    errTotal.value = resp.total
  } catch (error) {
    if (seq !== errorReqSeq) return
    console.error('Failed to load admin errors:', error)
    appStore.showError(t('usage.errors.failedToLoad'))
  } finally {
    if (seq === errorReqSeq) errLoading.value = false
  }
}

const onErrSort = (sortBy: string, sortOrder: 'asc' | 'desc') => {
  errSortBy.value = sortBy
  errSortOrder.value = sortOrder
  errPage.value = 1
  loadAdminErrors()
}
const onErrPage = (p: number) => { errPage.value = p; loadAdminErrors() }
const onErrPageSize = (s: number) => { errPageSize.value = s; errPage.value = 1; loadAdminErrors() }
const openError = (id: number) => { selectedErrorId.value = id; showErrorModal.value = true }

const showColumnDropdown = ref(false)
const columnDropdownRef = ref<HTMLElement | null>(null)

const handleColumnClickOutside = (event: MouseEvent) => {
  if (columnDropdownRef.value && !columnDropdownRef.value.contains(event.target as HTMLElement)) {
    showColumnDropdown.value = false
  }
}

onMounted(() => {
  applyRouteQueryFilters()
  loadLogs()
  loadStats()
  loadModelStats(modelDistributionSource.value, true)
  window.setTimeout(() => {
    void loadChartData()
  }, 120)
  if (activeTab.value === 'errors') {
    void loadAdminErrors()
  }
  loadSavedColumns()
  loadSavedErrColumns()
  document.addEventListener('click', handleColumnClickOutside)
})
onUnmounted(() => {
  abortController?.abort()
  exportAbortController?.abort()
  errorReqSeq += 1
  document.removeEventListener('click', handleColumnClickOutside)
})

watch(modelDistributionSource, (source) => {
  void loadModelStats(source)
})

defineExpose({ requestedModelStats, refreshData })
</script>

<style scoped>
.usage-workbench {
  display: flex;
  height: calc(100dvh - var(--app-shell-top-offset, 0px));
  min-height: 640px;
  min-width: 0;
  overflow: hidden;
  flex-direction: column;
  color: var(--lx-clay-text);
  background: var(--lx-clay-canvas);
  font-family: var(--lx-clay-font-ui);
}

.usage-workbench__header {
  position: relative;
  z-index: 30;
  flex: 0 0 auto;
  padding: 12px 32px;
  border-bottom: 1px solid var(--lx-clay-border);
  background: var(--lx-clay-surface);
}

.usage-workbench__page-header {
  gap: 20px;
}

.usage-workbench__page-header :deep(.admin-page-header__title) {
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-display);
  font-size: 20px;
  font-weight: 900;
  line-height: 1.2;
  letter-spacing: -0.02em;
}

.usage-workbench__page-header :deep(.admin-page-header__description) {
  margin-top: 4px;
  color: var(--lx-clay-text-secondary);
  font-size: 14px;
  line-height: 1.35;
}

.usage-workbench__page-header :deep(.admin-page-header__actions) {
  align-self: center;
}

.usage-workbench__header > :deep(.usage-stat-rail) {
  margin-top: 10px;
}

.usage-workbench__header-button {
  display: inline-flex;
  min-height: 38px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 1px solid var(--lx-clay-border);
  border-radius: 12px;
  padding: 0 15px;
  font-size: 14px;
  font-weight: 700;
  transition:
    border-color 150ms ease,
    background-color 150ms ease,
    color 150ms ease,
    transform 150ms ease;
}

.usage-workbench__header-button--secondary {
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-flat);
}

.usage-workbench__header-button--secondary:hover:not(:disabled) {
  border-color: var(--lx-clay-border-strong);
  color: var(--lx-clay-text);
  background: var(--lx-clay-recessed);
}

.usage-workbench__header-button--primary {
  border-color: var(--lx-clay-accent);
  color: var(--lx-clay-on-accent);
  background: var(--lx-clay-accent);
  box-shadow: var(--lx-clay-shadow-primary);
}

.usage-workbench__header-button--primary:hover:not(:disabled) {
  background: var(--lx-clay-accent-deep);
}

.usage-workbench__header-button:active:not(:disabled) {
  transform: scale(0.97);
}

.usage-workbench__header-button:disabled {
  cursor: not-allowed;
  opacity: 0.58;
}

.usage-workbench__body {
  display: grid;
  min-height: 0;
  min-width: 0;
  flex: 1 1 auto;
  grid-template-columns: 300px minmax(0, 1fr);
}

.usage-workbench__filter-rail {
  display: flex;
  min-height: 0;
  min-width: 0;
  overflow: hidden;
  flex-direction: column;
  border-right: 1px solid var(--lx-clay-border);
  background: var(--lx-clay-surface);
}

.usage-workbench__filter-scroll {
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: 20px;
  scrollbar-color: var(--lx-clay-border) transparent;
  scrollbar-width: thin;
}

.usage-workbench__filter-section + .usage-workbench__filter-section {
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid var(--lx-clay-border);
}

.usage-workbench__filter-heading,
.usage-workbench__danger-label {
  display: block;
  margin: 0 0 10px;
  color: var(--lx-clay-text-muted);
  font-family: var(--lx-clay-font-ui);
  font-size: 11px;
  font-weight: 900;
  line-height: 1.25;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

.usage-workbench__scope-field + .usage-workbench__scope-field {
  margin-top: 10px;
}

.usage-workbench__scope-label {
  display: block;
  margin-bottom: 6px;
  color: var(--lx-clay-text-secondary);
  font-size: 11px;
  font-weight: 600;
  line-height: 1.3;
}

.usage-workbench__scope-field :deep(.date-picker-trigger),
.usage-workbench__scope-field :deep(.select-control) {
  width: 100%;
  min-height: 38px;
  border-radius: 12px;
}

.usage-workbench__scope-field :deep(.date-picker-dropdown) {
  width: 100%;
  min-width: 0;
}

.usage-workbench__scope-field :deep(.date-picker-custom) {
  align-items: stretch;
  flex-direction: column;
}

.usage-workbench__scope-field :deep(.date-picker-separator) {
  display: none;
}

.usage-workbench__filter-status {
  margin: 0;
  color: var(--lx-clay-text-muted);
  font-size: 11px;
  font-style: italic;
  line-height: 1.5;
}

.usage-workbench__filter-actions {
  flex: 0 0 auto;
  padding: 16px 20px 20px;
  border-top: 1px solid var(--lx-clay-border);
  background: var(--lx-clay-surface);
}

.usage-workbench__filter-actions > .btn + .btn {
  margin-top: 9px;
}

.usage-workbench__danger-zone {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--lx-clay-border);
}

.usage-workbench__danger-label {
  color: var(--lx-clay-danger);
}

.usage-workbench__evidence {
  min-height: 0;
  min-width: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: 24px;
  background: var(--lx-clay-canvas);
  scrollbar-color: var(--lx-clay-border) transparent;
  scrollbar-width: thin;
}

.usage-workbench__surface {
  min-width: 0;
  overflow: clip;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-ops);
  background: var(--lx-clay-surface);
}

.usage-workbench__toolbar {
  position: sticky;
  top: 0;
  z-index: 20;
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 4px 24px;
  border-bottom: 1px solid var(--lx-clay-border);
  background: color-mix(in srgb, var(--lx-clay-surface) 96%, transparent);
  backdrop-filter: blur(10px);
}

.usage-workbench__tabs {
  display: flex;
  min-width: 0;
  gap: 24px;
  overflow-x: auto;
  scrollbar-width: none;
}

.usage-workbench__tabs::-webkit-scrollbar {
  display: none;
}

.usage-workbench__tab {
  position: relative;
  display: inline-flex;
  min-width: max-content;
  flex: 0 0 auto;
  align-items: center;
  border: 0;
  border-bottom: 2px solid transparent;
  border-radius: 0;
  padding: 16px 0;
  color: var(--lx-clay-text-muted) !important;
  background: transparent;
  font-size: 14px;
  font-weight: 700;
  line-height: 20px;
  transition:
    border-color 150ms ease,
    color 150ms ease;
}

.usage-workbench__tab:not(.usage-workbench__tab--active):hover {
  color: var(--lx-clay-text) !important;
}

.usage-workbench__tab--active,
.usage-workbench__tab--active:hover {
  border-bottom-color: var(--lx-clay-accent);
  color: var(--lx-clay-accent) !important;
}

.usage-workbench__toolbar-actions {
  display: flex;
  min-width: max-content;
  flex: 0 0 auto;
  align-items: center;
  gap: 10px;
}

.usage-workbench__result-count {
  display: inline-flex;
  min-height: 24px;
  align-items: center;
  border: 1px solid var(--lx-clay-border);
  border-radius: 999px;
  padding: 2px 9px;
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-recessed);
  font-size: 11px;
  font-weight: 700;
  line-height: 1.2;
  white-space: nowrap;
}

.usage-workbench__result-count--danger {
  border-color: color-mix(in srgb, var(--lx-clay-danger) 28%, transparent);
  color: var(--lx-clay-danger);
  background: var(--lx-clay-danger-soft);
}

.usage-workbench__column-control {
  position: relative;
}

.usage-workbench__tool-button {
  display: inline-flex;
  min-height: 32px;
  align-items: center;
  gap: 6px;
  border: 1px solid var(--lx-clay-border);
  border-radius: 8px;
  padding: 0 10px;
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-surface);
  font-size: 12px;
  font-weight: 700;
}

.usage-workbench__tool-button:hover {
  border-color: var(--lx-clay-border-strong);
  color: var(--lx-clay-text);
  background: var(--lx-clay-recessed);
}

.usage-workbench__density {
  display: inline-flex;
  gap: 2px;
  padding-left: 10px;
  border-left: 1px solid var(--lx-clay-border);
}

.usage-workbench__density-button {
  display: inline-flex;
  width: 32px;
  height: 32px;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  color: var(--lx-clay-text-subtle);
  background: transparent;
}

.usage-workbench__density-button:hover,
.usage-workbench__density-button--active {
  color: var(--lx-clay-text);
  background: var(--lx-clay-recessed);
}

.usage-workbench__column-menu {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  z-index: 50;
  width: 208px;
  max-height: 320px;
  overflow-y: auto;
  padding: 5px;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-table);
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-overlay);
}

.usage-workbench__column-option {
  display: flex;
  width: 100%;
  min-height: 40px;
  align-items: center;
  justify-content: space-between;
  border-radius: 9px;
  padding: 8px 10px;
  color: var(--lx-clay-text-secondary);
  text-align: left;
}

.usage-workbench__column-option:hover {
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

.usage-workbench__panel {
  min-width: 0;
  overflow: hidden;
}

.usage-workbench__panel :deep(.data-table-header) {
  background: color-mix(in srgb, var(--lx-clay-recessed) 62%, var(--lx-clay-surface));
}

.usage-workbench__panel :deep(.data-table-header-cell) {
  padding: 10px 16px !important;
  color: var(--lx-clay-text-secondary) !important;
  font-size: 11px !important;
  font-weight: 700 !important;
  letter-spacing: 0 !important;
  text-transform: none !important;
}

.usage-workbench__panel--usage :deep(.data-table-table) {
  border-collapse: separate;
  border-spacing: 0;
  font-family: var(--lx-clay-font-ui);
  font-size: 13px;
}

.usage-workbench__panel--usage :deep(.data-table-header),
.usage-workbench__panel--usage :deep(.data-table-header-cell) {
  background: #f9fafb !important;
}

.usage-workbench__panel--usage :deep(.data-table-header-cell) {
  color: #635f69 !important;
  font-size: 13px !important;
  font-weight: 700 !important;
}

.usage-workbench__panel--usage :deep(.usage-audit-col--account.data-table-cell) {
  text-align: left !important;
}

.dark .usage-workbench__panel--usage :deep(.data-table-header),
.dark .usage-workbench__panel--usage :deep(.data-table-header-cell) {
  color: var(--lx-clay-text-secondary) !important;
  background: var(--lx-clay-recessed) !important;
}

.usage-workbench__panel :deep(.data-table-cell) {
  padding: 10px 16px !important;
  color: var(--lx-clay-text);
  font-size: 13px !important;
}

.usage-workbench__panel--comfortable :deep(.data-table-cell) {
  padding-top: 14px !important;
  padding-bottom: 14px !important;
}

.usage-workbench__panel :deep(.pagination) {
  border-top: 1px solid var(--lx-clay-border);
  border-right: 0;
  border-bottom: 0;
  border-left: 0;
  border-radius: 0;
}

.usage-workbench__analytics {
  min-width: 0;
  margin-top: 24px;
  overflow: hidden;
  border: 1px dashed var(--lx-clay-border-strong);
  border-radius: var(--lx-clay-radius-ops);
  background: color-mix(in srgb, var(--lx-clay-surface) 72%, transparent);
}

.usage-workbench__analytics-summary {
  display: flex;
  min-height: 54px;
  cursor: pointer;
  list-style: none;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 0 20px;
  user-select: none;
}

.usage-workbench__analytics-summary::-webkit-details-marker {
  display: none;
}

.usage-workbench__analytics-heading-wrap {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--lx-clay-accent);
}

.usage-workbench__analytics-title {
  color: var(--lx-clay-text-secondary);
  font-size: 13px;
  font-weight: 900;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.usage-workbench__analytics-description {
  margin: -4px 20px 16px;
  color: var(--lx-clay-text-muted);
  font-size: 12px;
  line-height: 1.5;
}

.usage-workbench__analytics-chevron {
  color: var(--lx-clay-text-muted);
  transition: transform 150ms ease;
}

.usage-workbench__analytics[open] .usage-workbench__analytics-chevron {
  transform: rotate(180deg);
}

.usage-workbench__chart-grid {
  display: grid;
  min-width: 0;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
  padding: 0 20px 20px;
}

.usage-workbench__chart-wide {
  grid-column: 1 / -1;
}

@media (max-width: 1439px) {
  .usage-workbench__chart-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 1199px) {
  .usage-workbench__header {
    padding-right: 24px;
    padding-left: 24px;
  }

  .usage-workbench__body {
    grid-template-columns: 280px minmax(0, 1fr);
  }
}

@media (max-width: 1023px) {
  .usage-workbench__body {
    grid-template-columns: 260px minmax(0, 1fr);
  }

  .usage-workbench__evidence {
    padding: 18px;
  }

  .usage-workbench__toolbar {
    align-items: stretch;
    flex-direction: column;
    gap: 0;
    padding: 0 16px 10px;
  }

  .usage-workbench__tabs {
    min-height: 50px;
  }

  .usage-workbench__toolbar-actions {
    justify-content: flex-end;
  }

  .usage-workbench__chart-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (max-width: 767px) {
  .usage-workbench {
    height: auto;
    min-height: calc(100dvh - var(--app-shell-top-offset, 0px));
    overflow: visible;
  }

  .usage-workbench__header {
    padding: 16px;
  }

  .usage-workbench__page-header :deep(.admin-page-header__actions) {
    align-self: stretch;
  }

  .usage-workbench__header-button {
    flex: 1 1 auto;
  }

  .usage-workbench__body {
    display: flex;
    min-height: 0;
    flex-direction: column;
  }

  .usage-workbench__filter-rail {
    width: 100%;
    max-height: none;
    overflow: visible;
    border-right: 0;
    border-bottom: 1px solid var(--lx-clay-border);
  }

  .usage-workbench__filter-scroll {
    overflow: visible;
    padding: 18px 16px;
  }

  .usage-workbench__filter-actions {
    padding: 16px;
  }

  .usage-workbench__evidence {
    overflow: visible;
    padding: 16px;
  }

  .usage-workbench__toolbar-actions {
    min-width: 0;
    flex-wrap: wrap;
  }

  .usage-workbench__result-count {
    display: none;
  }

  .usage-workbench__tool-button span {
    display: none;
  }

  .usage-workbench__analytics {
    margin-top: 16px;
  }

  .usage-workbench__analytics-summary {
    padding: 0 16px;
  }

  .usage-workbench__analytics-description {
    margin-right: 16px;
    margin-left: 16px;
  }

  .usage-workbench__chart-grid {
    padding: 0 16px 16px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .usage-workbench__header-button,
  .usage-workbench__tab,
  .usage-workbench__analytics-chevron {
    transition: none;
  }
}

</style>

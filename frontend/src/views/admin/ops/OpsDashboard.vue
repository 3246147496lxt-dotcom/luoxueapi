<template>
  <component
    :is="isFullscreen ? 'main' : AppLayout"
    :class="isFullscreen ? 'ops-fullscreen-shell' : ''"
    :variant="isFullscreen ? undefined : 'home-clay'"
    :content-mode="isFullscreen ? undefined : 'workbench'"
  >
    <div
      data-admin-page-kind="ops"
      :class="['ops-dashboard-shell', { 'ops-dashboard-shell--fullscreen': isFullscreen }]"
      :aria-busy="loading"
    >
      <div
        v-if="errorMessage"
        class="ops-dashboard-alert"
        role="alert"
        aria-live="assertive"
        data-testid="ops-error-alert"
      >
        {{ errorMessage }}
      </div>

      <OpsDashboardSkeleton
        v-if="loading && !hasLoadedOnce"
        class="ops-dashboard-skeleton"
        :fullscreen="isFullscreen"
        role="status"
        aria-live="polite"
        aria-busy="true"
        :aria-label="t('common.loading')"
        data-testid="ops-dashboard-loading"
      />

      <OpsDashboardHeader
        v-else-if="opsEnabled"
        compact
        :overview="activeWorkspace === 'traffic' ? overview : null"
        :platform="platform"
        :group-id="groupId"
        :time-range="timeRange"
        :query-mode="queryMode"
        :loading="loading"
        :last-updated="lastUpdated"
        :thresholds="metricThresholds"
        :auto-refresh-enabled="autoRefreshEnabled"
        :auto-refresh-countdown="autoRefreshCountdown"
        :fullscreen="isFullscreen"
        :custom-start-time="customStartTime"
        :custom-end-time="customEndTime"
        :workspace="activeWorkspace"
        @update:time-range="onTimeRangeChange"
        @update:platform="onPlatformChange"
        @update:group="onGroupChange"
        @update:query-mode="onQueryModeChange"
        @update:custom-time-range="onCustomTimeRangeChange"
        @refresh="fetchData"
        @open-request-details="handleOpenRequestDetails"
        @open-error-details="openErrorDetails"
        @open-settings="showSettingsDialog = true"
        @open-alert-rules="showAlertRulesCard = true"
        @enter-fullscreen="enterFullscreen"
        @exit-fullscreen="exitFullscreen"
      />

      <div
        v-if="opsEnabled && !(loading && !hasLoadedOnce)"
        class="ops-dashboard-body"
        data-testid="ops-dashboard-body"
      >
        <OpsWorkspaceNav
          class="ops-workspace-nav-host"
          :model-value="activeWorkspace"
          :items="workspaceItems"
          :label="t('admin.ops.workspace.label')"
          @update:model-value="onWorkspaceChange"
        />

        <div
          v-show="activeWorkspace === 'traffic'"
          id="ops-workspace-panel-traffic"
          class="ops-workspace-view"
          role="tabpanel"
          aria-labelledby="ops-workspace-tab-traffic"
          data-testid="ops-workspace-panel-traffic"
        >
          <template v-if="visitedWorkspaces.traffic">
          <section class="ops-workspace-section ops-workspace-section--flush" aria-labelledby="ops-traffic-heading" data-testid="ops-traffic-section">
            <OpsWorkbenchShell
              :rail-label="t('admin.ops.trafficSectionTitle')"
              :evidence-label="t('admin.ops.trafficSectionDescription')"
              :default-mobile-rail-open="false"
            >
              <template #rail>
                <OpsTrafficInvestigationRail
                  :overview="overview"
                  :platform="platform"
                  :group-id="groupId"
                  :time-range="timeRange"
                  :loading="loading"
                  @update:platform="onPlatformChange"
                  @update:group="onGroupChange"
                  @update:time-range="onTimeRangeChange"
                  @select-signal="handleTrafficSignal"
                />
              </template>

              <template #evidence>
                <div class="ops-evidence-canvas">
                  <div class="ops-live-grid">
                    <div id="ops-traffic-evidence-throughput" class="ops-panel ops-panel--throughput min-w-0">
                      <OpsThroughputTrendChart
                        :points="throughputTrend?.points ?? []"
                        :overview="overview"
                        :by-platform="throughputTrend?.by_platform ?? []"
                        :top-groups="throughputTrend?.top_groups ?? []"
                        :loading="loadingTrend"
                        :time-range="timeRange"
                        :fullscreen="isFullscreen"
                        @select-platform="handleThroughputSelectPlatform"
                        @select-group="handleThroughputSelectGroup"
                        @open-details="handleOpenRequestDetails"
                      />
                    </div>
                  </div>

                  <div class="ops-performance-grid">
                    <div class="ops-panel ops-panel--latency min-w-0">
                      <OpsLatencyChart :latency-data="latencyHistogram" :loading="loadingLatency" />
                    </div>
                    <div v-if="showOpenAITokenStats" class="ops-model-panel min-w-0">
                      <OpsOpenAITokenStatsCard
                        :platform-filter="platform"
                        :group-id-filter="groupId"
                        :refresh-token="dashboardRefreshToken"
                      />
                    </div>
                  </div>
                </div>
              </template>
            </OpsWorkbenchShell>
          </section>
          </template>
        </div>

        <div
          v-show="activeWorkspace === 'incidents'"
          id="ops-workspace-panel-incidents"
          class="ops-workspace-view"
          role="tabpanel"
          aria-labelledby="ops-workspace-tab-incidents"
          data-testid="ops-workspace-panel-incidents"
        >
          <template v-if="visitedWorkspaces.incidents">
          <section
            class="ops-workspace-section ops-workspace-section--flush"
            aria-labelledby="ops-incidents-heading"
            data-testid="ops-incidents-section"
          >
            <h2 id="ops-incidents-heading" class="sr-only">
              {{ t('admin.ops.incidentsSectionTitle') }}
            </h2>
            <div data-testid="ops-alerts-section">
              <OpsAlertEventsCard
                :enabled="showAlertEvents"
                :platform-filter="platform"
                :group-id-filter="groupId"
                @update:platform="onPlatformChange"
                @update:group="onGroupChange"
                @view-related-logs="handleViewRelatedLogs"
              >
                <template #evidence>
                  <section class="ops-incident-evidence" aria-labelledby="ops-incidents-evidence-heading">
                    <header class="ops-section-heading ops-section-heading--evidence">
                      <div>
                        <h3 id="ops-incidents-evidence-heading">{{ t('admin.ops.incidentsSectionTitle') }}</h3>
                        <p>{{ t('admin.ops.incidentsSectionDescription') }}</p>
                      </div>
                      <span class="ops-section-signal ops-section-signal--risk">
                        <i aria-hidden="true"></i>
                        {{ t('admin.ops.incidentsSectionStatus') }}
                      </span>
                    </header>

                    <div class="ops-quality-grid">
                      <div class="ops-panel ops-panel--error-trend min-w-0">
                        <OpsErrorTrendChart
                          :points="errorTrend?.points ?? []"
                          :loading="loadingErrorTrend"
                          :time-range="timeRange"
                          @open-request-errors="openErrorDetails('request')"
                          @open-upstream-errors="openErrorDetails('upstream')"
                        />
                      </div>
                      <div class="ops-panel ops-panel--switch-rate min-w-0">
                        <OpsSwitchRateTrendChart
                          :points="switchTrend?.points ?? []"
                          :loading="loadingSwitchTrend"
                          :time-range="switchTrendTimeRange"
                          :fullscreen="isFullscreen"
                        />
                      </div>
                      <div class="ops-panel ops-panel--error-distribution min-w-0">
                        <OpsErrorDistributionChart
                          :data="errorDistribution"
                          :loading="loadingErrorDistribution"
                          @open-details="openErrorDetails('request')"
                        />
                      </div>
                    </div>
                  </section>
                </template>
              </OpsAlertEventsCard>
            </div>
          </section>
          </template>
        </div>

        <div
          v-show="activeWorkspace === 'diagnostics'"
          id="ops-workspace-panel-diagnostics"
          class="ops-workspace-view"
          role="tabpanel"
          aria-labelledby="ops-workspace-tab-diagnostics"
          data-testid="ops-workspace-panel-diagnostics"
        >
          <section
            v-if="visitedWorkspaces.diagnostics"
            class="ops-workspace-section ops-workspace-section--flush"
            aria-labelledby="ops-diagnostics-heading"
            data-testid="ops-log-section"
          >
            <h2 id="ops-diagnostics-heading" class="sr-only">
              {{ t('admin.ops.diagnosticsSectionTitle') }}
            </h2>
            <div class="ops-support-stack">
              <OpsSystemLogTable
                :platform-filter="platform"
                :refresh-token="dashboardRefreshToken"
                :investigation-preset="logInvestigationPreset"
                @clear-investigation="logInvestigationPreset = null"
                @open-request-details="handleOpenSystemLogRequestDetails"
              />
            </div>
          </section>
        </div>
      </div>

      <!-- Settings Dialog (hidden in fullscreen mode) -->
      <template v-if="!isFullscreen">
        <OpsSettingsDialog :show="showSettingsDialog" @close="showSettingsDialog = false" @saved="onSettingsSaved" />

        <BaseDialog :show="showAlertRulesCard" :title="t('admin.ops.alertRules.title')" width="extra-wide" @close="showAlertRulesCard = false">
          <OpsAlertRulesCard />
        </BaseDialog>

        <BaseDialog
          :show="showConcurrencyDialog"
          :title="t('admin.ops.concurrency.title')"
          width="wide"
          @close="showConcurrencyDialog = false"
        >
          <div class="ops-concurrency-dialog-body">
            <OpsConcurrencyCard
              :platform-filter="platform"
              :group-id-filter="groupId"
              :refresh-token="dashboardRefreshToken"
            />
          </div>
        </BaseDialog>

        <OpsErrorDetailsModal
          :show="showErrorDetails"
          :time-range="timeRange"
          :platform="platform"
          :group-id="groupId"
          :error-type="errorDetailsType"
          @update:show="showErrorDetails = $event"
          @openErrorDetail="openError"
        />

        <OpsErrorDetailModal v-model:show="showErrorModal" :error-id="selectedErrorId" :error-type="errorDetailsType" />

        <OpsRequestDetailsModal
          v-model="showRequestDetails"
          :time-range="timeRange"
          :preset="requestDetailsPreset"
          :platform="platform"
          :group-id="groupId"
          @openErrorDetail="openError"
        />
      </template>
    </div>
  </component>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useDebounceFn, useIntervalFn } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import {
  opsAPI,
  type OpsDashboardOverview,
  type OpsErrorDistributionResponse,
  type OpsErrorTrendResponse,
  type OpsLatencyHistogramResponse,
  type OpsThroughputTrendResponse,
  type OpsMetricThresholds,
  type OpsSystemLog
} from '@/api/admin/ops'
import { useAdminSettingsStore, useAppStore } from '@/stores'
import OpsDashboardHeader from './components/OpsDashboardHeader.vue'
import OpsDashboardSkeleton from './components/OpsDashboardSkeleton.vue'
import OpsConcurrencyCard from './components/OpsConcurrencyCard.vue'
import OpsErrorDetailModal from './components/OpsErrorDetailModal.vue'
import OpsErrorDistributionChart from './components/OpsErrorDistributionChart.vue'
import OpsErrorDetailsModal from './components/OpsErrorDetailsModal.vue'
import OpsErrorTrendChart from './components/OpsErrorTrendChart.vue'
import OpsLatencyChart from './components/OpsLatencyChart.vue'
import OpsThroughputTrendChart from './components/OpsThroughputTrendChart.vue'
import OpsSwitchRateTrendChart from './components/OpsSwitchRateTrendChart.vue'
import OpsAlertEventsCard from './components/OpsAlertEventsCard.vue'
import OpsOpenAITokenStatsCard from './components/OpsOpenAITokenStatsCard.vue'
import OpsSystemLogTable from './components/OpsSystemLogTable.vue'
import OpsRequestDetailsModal, { type OpsRequestDetailsPreset } from './components/OpsRequestDetailsModal.vue'
import OpsSettingsDialog from './components/OpsSettingsDialog.vue'
import OpsAlertRulesCard from './components/OpsAlertRulesCard.vue'
import OpsWorkspaceNav from './components/OpsWorkspaceNav.vue'
import OpsWorkbenchShell from './components/OpsWorkbenchShell.vue'
import OpsTrafficInvestigationRail from './components/OpsTrafficInvestigationRail.vue'
import type { OpsAlertLogContext, OpsLogInvestigationPreset } from './types'
import './OpsDashboard.clay.css'

const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const adminSettingsStore = useAdminSettingsStore()
const { t } = useI18n()

const opsEnabled = computed(() => adminSettingsStore.opsMonitoringEnabled)

type TimeRange = '5m' | '30m' | '1h' | '6h' | '24h' | 'custom'
const allowedTimeRanges = new Set<TimeRange>(['5m', '30m', '1h', '6h', '24h', 'custom'])

type QueryMode = 'auto' | 'raw' | 'preagg'
const allowedQueryModes = new Set<QueryMode>(['auto', 'raw', 'preagg'])

type OpsWorkspace = 'traffic' | 'incidents' | 'diagnostics'

const allowedWorkspaces = new Set<OpsWorkspace>(['traffic', 'incidents', 'diagnostics'])
const legacyWorkspaceMap: Record<string, OpsWorkspace> = {
  live: 'traffic',
  quality: 'incidents',
  alerts: 'incidents',
  logs: 'diagnostics'
}

const loading = ref(true)
const hasLoadedOnce = ref(false)
const errorMessage = ref('')
const lastUpdated = ref<Date | null>(new Date())

const timeRange = ref<TimeRange>('1h')
const platform = ref<string>('')
const groupId = ref<number | null>(null)
const queryMode = ref<QueryMode>('auto')
const customStartTime = ref<string | null>(null)
const customEndTime = ref<string | null>(null)
const activeWorkspace = ref<OpsWorkspace>('traffic')
const visitedWorkspaces = ref<Record<OpsWorkspace, boolean>>({
  traffic: false,
  incidents: false,
  diagnostics: false
})
const loadedWorkspaces = ref<Record<OpsWorkspace, boolean>>({
  traffic: false,
  incidents: false,
  diagnostics: false
})
const workspaceItems = computed(() => [
  { id: 'traffic', label: t('admin.ops.workspace.traffic') },
  { id: 'incidents', label: t('admin.ops.workspace.incidents') },
  { id: 'diagnostics', label: t('admin.ops.workspace.diagnostics') }
])
const switchTrendWindowHours = 5
const switchTrendTimeRange = `${switchTrendWindowHours}h`
const switchTrendWindowMs = switchTrendWindowHours * 60 * 60 * 1000

const QUERY_KEYS = {
  timeRange: 'tr',
  platform: 'platform',
  groupId: 'group_id',
  queryMode: 'mode',
  workspace: 'section',
  resource: 'resource',
  fullscreen: 'fullscreen',

  // Deep links
  openErrorDetails: 'open_error_details',
  errorType: 'error_type',
  alertRuleId: 'alert_rule_id',
  openAlertRules: 'open_alert_rules'
} as const

const STATE_QUERY_KEYS = [
  QUERY_KEYS.timeRange,
  QUERY_KEYS.platform,
  QUERY_KEYS.groupId,
  QUERY_KEYS.queryMode,
  QUERY_KEYS.workspace,
  QUERY_KEYS.resource
] as const

const TRANSIENT_QUERY_KEYS = [
  QUERY_KEYS.openErrorDetails,
  QUERY_KEYS.errorType,
  QUERY_KEYS.alertRuleId,
  QUERY_KEYS.openAlertRules
] as const

const isApplyingRouteQuery = ref(false)
const isSyncingRouteQuery = ref(false)
let activeRouteSyncs = 0
let routeReplaceQueue: Promise<void> = Promise.resolve()

// Fullscreen mode
const isFullscreen = computed(() => {
  const val = route.query[QUERY_KEYS.fullscreen]
  return val === '1' || val === 'true'
})

const OPS_FULLSCREEN_BODY_CLASS = 'admin-ops-fullscreen'
const OPS_OPTION_B_BODY_CLASS = 'admin-ops-option-b'
let previousSidebarCollapsed: boolean | null = null

function syncFullscreenBodyClass(enabled: boolean) {
  if (typeof document === 'undefined') return
  document.body.classList.toggle(OPS_FULLSCREEN_BODY_CLASS, enabled)
}

function syncOpsOptionBBodyClass(enabled: boolean) {
  if (typeof document === 'undefined') return
  document.body.classList.toggle(OPS_OPTION_B_BODY_CLASS, enabled)
}

watch(isFullscreen, syncFullscreenBodyClass, { immediate: true })

function exitFullscreen() {
  const nextQuery = { ...route.query }
  delete nextQuery[QUERY_KEYS.fullscreen]
  router.replace({ query: nextQuery })
}

function enterFullscreen() {
  const nextQuery = { ...route.query, [QUERY_KEYS.fullscreen]: '1' }
  router.replace({ query: nextQuery })
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && isFullscreen.value) {
    exitFullscreen()
  }
}

let dashboardFetchController: AbortController | null = null
let dashboardFetchSeq = 0

function isCanceledRequest(err: unknown): boolean {
  return (
    !!err &&
    typeof err === 'object' &&
    'code' in err &&
    (err as Record<string, unknown>).code === 'ERR_CANCELED'
  )
}

function abortDashboardFetch() {
  if (dashboardFetchController) {
    dashboardFetchController.abort()
    dashboardFetchController = null
  }
}

const readQueryString = (key: string): string => {
  const value = route.query[key]
  if (typeof value === 'string') return value
  if (Array.isArray(value) && typeof value[0] === 'string') return value[0]
  return ''
}

const readQueryNumber = (key: string): number | null => {
  const raw = readQueryString(key)
  if (!raw) return null
  const n = Number.parseInt(raw, 10)
  return Number.isFinite(n) ? n : null
}

function resolveWorkspace(value: string): OpsWorkspace | null {
  if (allowedWorkspaces.has(value as OpsWorkspace)) return value as OpsWorkspace
  return legacyWorkspaceMap[value] ?? null
}

async function normalizeInvalidWorkspaceQuery() {
  const requestedWorkspace = readQueryString(QUERY_KEYS.workspace)
  const requestedResource = readQueryString(QUERY_KEYS.resource)
  const nextQuery = { ...route.query }
  let changed = false

  if (requestedWorkspace) {
    const resolved = resolveWorkspace(requestedWorkspace)
    if (resolved && resolved !== requestedWorkspace) {
      nextQuery[QUERY_KEYS.workspace] = resolved
      changed = true
    } else if (!resolved) {
      delete nextQuery[QUERY_KEYS.workspace]
      changed = true
    }
  }

  if (requestedResource) {
    delete nextQuery[QUERY_KEYS.resource]
    changed = true
  }

  if (changed) await router.replace({ query: nextQuery })
}

const applyRouteQueryToState = () => {
  const requestedWorkspace = readQueryString(QUERY_KEYS.workspace)
  const resolvedWorkspace = resolveWorkspace(requestedWorkspace)
  const hasExplicitWorkspace = resolvedWorkspace !== null
  let nextWorkspace: OpsWorkspace = resolvedWorkspace ?? 'traffic'

  const nextTimeRange = readQueryString(QUERY_KEYS.timeRange)
  if (nextTimeRange && allowedTimeRanges.has(nextTimeRange as TimeRange)) {
    timeRange.value = nextTimeRange as TimeRange
  }

  platform.value = readQueryString(QUERY_KEYS.platform) || ''

  const groupIdRaw = readQueryNumber(QUERY_KEYS.groupId)
  groupId.value = typeof groupIdRaw === 'number' && groupIdRaw > 0 ? groupIdRaw : null

  const nextMode = readQueryString(QUERY_KEYS.queryMode)
  if (nextMode && allowedQueryModes.has(nextMode as QueryMode)) {
    queryMode.value = nextMode as QueryMode
  } else {
    const fallback = adminSettingsStore.opsQueryModeDefault || 'auto'
    queryMode.value = allowedQueryModes.has(fallback as QueryMode) ? (fallback as QueryMode) : 'auto'
  }

  // Deep links
  const openRules = readQueryString(QUERY_KEYS.openAlertRules)
  if (openRules === '1' || openRules === 'true') {
    showAlertRulesCard.value = true
    if (!hasExplicitWorkspace) nextWorkspace = 'incidents'
  }

  const ruleID = readQueryNumber(QUERY_KEYS.alertRuleId)
  if (typeof ruleID === 'number' && ruleID > 0) {
    showAlertRulesCard.value = true
    if (!hasExplicitWorkspace) nextWorkspace = 'incidents'
  }

  const openErr = readQueryString(QUERY_KEYS.openErrorDetails)
  if (openErr === '1' || openErr === 'true') {
    const typ = readQueryString(QUERY_KEYS.errorType)
    errorDetailsType.value = typ === 'upstream' ? 'upstream' : 'request'
    showErrorDetails.value = true
    if (!hasExplicitWorkspace) nextWorkspace = 'incidents'
  }

  activeWorkspace.value = nextWorkspace
  visitedWorkspaces.value[nextWorkspace] = true
}

const buildQueryFromState = () => {
  const next: Record<string, any> = { ...route.query }
  const queryKeysToReplace = [...STATE_QUERY_KEYS, ...TRANSIENT_QUERY_KEYS]

  queryKeysToReplace.forEach((k) => {
    delete next[k]
  })

  if (timeRange.value !== '1h') next[QUERY_KEYS.timeRange] = timeRange.value
  if (platform.value) next[QUERY_KEYS.platform] = platform.value
  if (typeof groupId.value === 'number' && groupId.value > 0) next[QUERY_KEYS.groupId] = String(groupId.value)
  if (queryMode.value !== 'auto') next[QUERY_KEYS.queryMode] = queryMode.value
  if (activeWorkspace.value !== 'traffic') next[QUERY_KEYS.workspace] = activeWorkspace.value

  return next
}

const replaceQueryFromState = async (force = false) => {
  if (isApplyingRouteQuery.value) return
  const nextQuery = buildQueryFromState()

  const curr = route.query as Record<string, any>
  const nextKeys = Object.keys(nextQuery)
  const currKeys = Object.keys(curr)
  const sameLength = nextKeys.length === currKeys.length
  const sameValues = sameLength && nextKeys.every((k) => String(curr[k] ?? '') === String(nextQuery[k] ?? ''))
  if (!force && sameValues) return

  activeRouteSyncs += 1
  isSyncingRouteQuery.value = true
  const queuedReplace = routeReplaceQueue.then(async () => {
    await router.replace({ query: nextQuery })
  })
  routeReplaceQueue = queuedReplace.then(
    () => undefined,
    () => undefined
  )
  try {
    await queuedReplace
  } finally {
    activeRouteSyncs = Math.max(0, activeRouteSyncs - 1)
    isSyncingRouteQuery.value = activeRouteSyncs > 0
  }
}

const syncQueryToRoute = useDebounceFn(replaceQueryFromState, 250)

const overview = ref<OpsDashboardOverview | null>(null)
const metricThresholds = ref<OpsMetricThresholds | null>(null)

const throughputTrend = ref<OpsThroughputTrendResponse | null>(null)
const loadingTrend = ref(false)

const switchTrend = ref<OpsThroughputTrendResponse | null>(null)
const loadingSwitchTrend = ref(false)

const latencyHistogram = ref<OpsLatencyHistogramResponse | null>(null)
const loadingLatency = ref(false)

const errorTrend = ref<OpsErrorTrendResponse | null>(null)
const loadingErrorTrend = ref(false)

const errorDistribution = ref<OpsErrorDistributionResponse | null>(null)
const loadingErrorDistribution = ref(false)

const selectedErrorId = ref<number | null>(null)
const showErrorModal = ref(false)

const showErrorDetails = ref(false)
const errorDetailsType = ref<'request' | 'upstream'>('request')

const showRequestDetails = ref(false)
const showConcurrencyDialog = ref(false)
const requestDetailsPreset = ref<OpsRequestDetailsPreset>({
  title: '',
  kind: 'all',
  sort: 'created_at_desc'
})

const showSettingsDialog = ref(false)
const showAlertRulesCard = ref(false)
const logInvestigationPreset = ref<OpsLogInvestigationPreset | null>(null)
let logInvestigationSeq = 0

function scrollWorkspaceIntoView(workspace: OpsWorkspace) {
  if (typeof document === 'undefined') return
  const panel = document.getElementById(`ops-workspace-panel-${workspace}`)
  if (typeof panel?.scrollIntoView === 'function') {
    panel.scrollIntoView({ behavior: 'auto', block: 'start' })
  }
}

let workspaceNavigationSeq = 0

async function onWorkspaceChange(value: string) {
  if (!allowedWorkspaces.has(value as OpsWorkspace)) return
  const nextWorkspace = value as OpsWorkspace
  if (activeWorkspace.value === nextWorkspace) return

  visitedWorkspaces.value[nextWorkspace] = true
  activeWorkspace.value = nextWorkspace
  workspaceNavigationSeq += 1
  const navigationSeq = workspaceNavigationSeq
  await replaceQueryFromState(true)
  if (navigationSeq !== workspaceNavigationSeq || activeWorkspace.value !== nextWorkspace) return
  if (!loadedWorkspaces.value[nextWorkspace]) {
    await fetchData({ refreshChildren: false })
  }
  await nextTick()
  scrollWorkspaceIntoView(nextWorkspace)
}

async function handleViewRelatedLogs(context: OpsAlertLogContext) {
  const firedAt = new Date(context.firedAt)
  if (!Number.isFinite(firedAt.getTime())) return

  const investigationRadiusMs = 30 * 60 * 1000
  logInvestigationSeq += 1
  logInvestigationPreset.value = {
    ...context,
    key: logInvestigationSeq,
    startTime: new Date(firedAt.getTime() - investigationRadiusMs).toISOString(),
    endTime: new Date(firedAt.getTime() + investigationRadiusMs).toISOString()
  }

  if (activeWorkspace.value !== 'diagnostics') {
    await onWorkspaceChange('diagnostics')
  } else {
    await nextTick()
    scrollWorkspaceIntoView('diagnostics')
  }
}

applyRouteQueryToState()

// Auto refresh settings
const showAlertEvents = ref(true)
const showOpenAITokenStats = ref(false)
const autoRefreshEnabled = ref(false)
const autoRefreshIntervalMs = ref(30000) // default 30 seconds
const autoRefreshCountdown = ref(0)

// Used to trigger child component refreshes in a single shared cadence.
const dashboardRefreshToken = ref(0)

// Countdown timer (drives auto refresh; updates every second)
const { pause: pauseCountdown, resume: resumeCountdown } = useIntervalFn(
  () => {
    if (!autoRefreshEnabled.value) return
    if (!opsEnabled.value) return
    if (loading.value) return

    if (autoRefreshCountdown.value <= 0) {
      // Fetch immediately when the countdown reaches 0.
      // fetchData() will reset the countdown to the full interval.
      fetchData()
      return
    }

    autoRefreshCountdown.value -= 1
  },
  1000,
  { immediate: false }
)

// Load ops dashboard presentation settings from backend.
async function loadDashboardAdvancedSettings() {
  try {
    const settings = await opsAPI.getAdvancedSettings()
    showAlertEvents.value = settings.display_alert_events
    showOpenAITokenStats.value = settings.display_openai_token_stats
    autoRefreshEnabled.value = settings.auto_refresh_enabled
    autoRefreshIntervalMs.value = settings.auto_refresh_interval_seconds * 1000
    autoRefreshCountdown.value = settings.auto_refresh_interval_seconds
  } catch (err) {
    console.error('[OpsDashboard] Failed to load dashboard advanced settings', err)
    showAlertEvents.value = true
    showOpenAITokenStats.value = false
    autoRefreshEnabled.value = false
    autoRefreshIntervalMs.value = 30000
    autoRefreshCountdown.value = 0
  }
}

function handleThroughputSelectPlatform(nextPlatform: string) {
  platform.value = nextPlatform || ''
  groupId.value = null
}

function handleThroughputSelectGroup(nextGroupId: number) {
  const id = Number.isFinite(nextGroupId) && nextGroupId > 0 ? nextGroupId : null
  groupId.value = id
}

function scrollToEvidence(id: string) {
  if (typeof document === 'undefined') return
  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'center' })
}

function handleTrafficSignal(signal: 'health' | 'sla' | 'errors' | 'throughput' | 'concurrency') {
  if (signal === 'sla') {
    handleOpenRequestDetails({
      title: t('admin.ops.requestDetails.title'),
      kind: 'error',
      sort: 'created_at_desc'
    })
    return
  }
  if (signal === 'errors') {
    openErrorDetails('request')
    return
  }
  if (signal === 'concurrency') {
    showConcurrencyDialog.value = true
    return
  }
  scrollToEvidence('ops-traffic-evidence-throughput')
}

function handleOpenRequestDetails(preset?: OpsRequestDetailsPreset) {
  const basePreset: OpsRequestDetailsPreset = {
    title: t('admin.ops.requestDetails.title'),
    kind: 'all',
    sort: 'created_at_desc'
  }

  requestDetailsPreset.value = { ...basePreset, ...(preset ?? {}) }
  if (!requestDetailsPreset.value.title) requestDetailsPreset.value.title = basePreset.title
  // Ensure only one modal visible at a time.
  showErrorDetails.value = false
  showErrorModal.value = false
  showRequestDetails.value = true
}

function handleOpenSystemLogRequestDetails(log: OpsSystemLog) {
  const createdAt = new Date(log.created_at)
  const radiusMs = 5 * 60 * 1000
  const hasValidTime = Number.isFinite(createdAt.getTime())
  handleOpenRequestDetails({
    title: t('admin.ops.requestDetails.title'),
    kind: 'all',
    sort: 'created_at_desc',
    request_id: log.request_id || undefined,
    platform: log.platform || undefined,
    start_time: hasValidTime ? new Date(createdAt.getTime() - radiusMs).toISOString() : undefined,
    end_time: hasValidTime ? new Date(createdAt.getTime() + radiusMs).toISOString() : undefined
  })
}

function openErrorDetails(kind: 'request' | 'upstream') {
  errorDetailsType.value = kind
  // Ensure only one modal visible at a time.
  showRequestDetails.value = false
  showErrorModal.value = false
  showErrorDetails.value = true
}

function onTimeRangeChange(v: string | number | boolean | null) {
  if (typeof v !== 'string') return
  if (!allowedTimeRanges.has(v as TimeRange)) return
  timeRange.value = v as TimeRange
}

function onCustomTimeRangeChange(startTime: string, endTime: string) {
  customStartTime.value = startTime
  customEndTime.value = endTime
}

async function onSettingsSaved() {
  await loadDashboardAdvancedSettings()
  loadThresholds()
  fetchData()
}

function onPlatformChange(v: string | number | boolean | null) {
  platform.value = typeof v === 'string' ? v : ''
}

function onGroupChange(v: string | number | boolean | null) {
  if (v === null) {
    groupId.value = null
    return
  }
  if (typeof v === 'number') {
    groupId.value = v > 0 ? v : null
    return
  }
  if (typeof v === 'string') {
    const n = Number.parseInt(v, 10)
    groupId.value = Number.isFinite(n) && n > 0 ? n : null
  }
}

function onQueryModeChange(v: string | number | boolean | null) {
  if (typeof v !== 'string') return
  if (!allowedQueryModes.has(v as QueryMode)) return
  queryMode.value = v as QueryMode
}

function openError(id: number) {
  selectedErrorId.value = id
  // Ensure only one modal visible at a time.
  showErrorDetails.value = false
  showRequestDetails.value = false
  showErrorModal.value = true
}

function buildApiParams() {
  const params: any = {
    platform: platform.value || undefined,
    group_id: groupId.value ?? undefined,
    mode: queryMode.value
  }

  if (timeRange.value === 'custom') {
    if (customStartTime.value && customEndTime.value) {
      params.start_time = customStartTime.value
      params.end_time = customEndTime.value
    } else {
      // Safety fallback: avoid sending time_range=custom (backend may not support it)
      params.time_range = '1h'
    }
  } else {
    params.time_range = timeRange.value
  }

  return params
}

function buildSwitchTrendParams() {
  const params: any = {
    platform: platform.value || undefined,
    group_id: groupId.value ?? undefined,
    mode: queryMode.value
  }
  const endTime = new Date()
  const startTime = new Date(endTime.getTime() - switchTrendWindowMs)
  params.start_time = startTime.toISOString()
  params.end_time = endTime.toISOString()
  return params
}

async function refreshOverviewWithCancel(fetchSeq: number, signal: AbortSignal) {
  if (!opsEnabled.value) return
  try {
    const data = await opsAPI.getDashboardOverview(buildApiParams(), { signal })
    if (fetchSeq !== dashboardFetchSeq) return
    overview.value = data
  } catch (err: any) {
    if (fetchSeq !== dashboardFetchSeq || isCanceledRequest(err)) return
    overview.value = null
    appStore.showError(err?.message || t('admin.ops.failedToLoadOverview'))
  }
}

async function refreshSwitchTrendWithCancel(fetchSeq: number, signal: AbortSignal) {
  if (!opsEnabled.value) return
  loadingSwitchTrend.value = true
  try {
    const data = await opsAPI.getThroughputTrend(buildSwitchTrendParams(), { signal })
    if (fetchSeq !== dashboardFetchSeq) return
    switchTrend.value = data
  } catch (err: any) {
    if (fetchSeq !== dashboardFetchSeq || isCanceledRequest(err)) return
    switchTrend.value = null
    appStore.showError(err?.message || t('admin.ops.failedToLoadSwitchTrend'))
  } finally {
    if (fetchSeq === dashboardFetchSeq) {
      loadingSwitchTrend.value = false
    }
  }
}

async function refreshThroughputTrendWithCancel(fetchSeq: number, signal: AbortSignal) {
  if (!opsEnabled.value) return
  loadingTrend.value = true
  try {
    const data = await opsAPI.getThroughputTrend(buildApiParams(), { signal })
    if (fetchSeq !== dashboardFetchSeq) return
    throughputTrend.value = data
  } catch (err: any) {
    if (fetchSeq !== dashboardFetchSeq || isCanceledRequest(err)) return
    throughputTrend.value = null
    appStore.showError(err?.message || t('admin.ops.failedToLoadThroughputTrend'))
  } finally {
    if (fetchSeq === dashboardFetchSeq) {
      loadingTrend.value = false
    }
  }
}

async function refreshLatencyHistogramWithCancel(fetchSeq: number, signal: AbortSignal) {
  if (!opsEnabled.value) return
  loadingLatency.value = true
  try {
    const data = await opsAPI.getLatencyHistogram(buildApiParams(), { signal })
    if (fetchSeq !== dashboardFetchSeq) return
    latencyHistogram.value = data
  } catch (err: any) {
    if (fetchSeq !== dashboardFetchSeq || isCanceledRequest(err)) return
    latencyHistogram.value = null
    appStore.showError(err?.message || t('admin.ops.failedToLoadLatencyHistogram'))
  } finally {
    if (fetchSeq === dashboardFetchSeq) {
      loadingLatency.value = false
    }
  }
}

async function refreshErrorTrendWithCancel(fetchSeq: number, signal: AbortSignal) {
  if (!opsEnabled.value) return
  loadingErrorTrend.value = true
  try {
    const data = await opsAPI.getErrorTrend(buildApiParams(), { signal })
    if (fetchSeq !== dashboardFetchSeq) return
    errorTrend.value = data
  } catch (err: any) {
    if (fetchSeq !== dashboardFetchSeq || isCanceledRequest(err)) return
    errorTrend.value = null
    appStore.showError(err?.message || t('admin.ops.failedToLoadErrorTrend'))
  } finally {
    if (fetchSeq === dashboardFetchSeq) {
      loadingErrorTrend.value = false
    }
  }
}

async function refreshErrorDistributionWithCancel(fetchSeq: number, signal: AbortSignal) {
  if (!opsEnabled.value) return
  loadingErrorDistribution.value = true
  try {
    const data = await opsAPI.getErrorDistribution(buildApiParams(), { signal })
    if (fetchSeq !== dashboardFetchSeq) return
    errorDistribution.value = data
  } catch (err: any) {
    if (fetchSeq !== dashboardFetchSeq || isCanceledRequest(err)) return
    errorDistribution.value = null
    appStore.showError(err?.message || t('admin.ops.failedToLoadErrorDistribution'))
  } finally {
    if (fetchSeq === dashboardFetchSeq) {
      loadingErrorDistribution.value = false
    }
  }
}

function isOpsDisabledError(err: unknown): boolean {
  return (
    !!err &&
    typeof err === 'object' &&
    'code' in err &&
    typeof (err as Record<string, unknown>).code === 'string' &&
    (err as Record<string, unknown>).code === 'OPS_DISABLED'
  )
}

interface FetchWorkspaceOptions {
  refreshChildren?: boolean
}

async function fetchData(options: FetchWorkspaceOptions = {}) {
  if (!opsEnabled.value) return

  abortDashboardFetch()
  dashboardFetchSeq += 1
  const fetchSeq = dashboardFetchSeq
  dashboardFetchController = new AbortController()

  loading.value = true
  errorMessage.value = ''
  try {
    const workspace = activeWorkspace.value
    const signal = dashboardFetchController.signal

    if (workspace === 'traffic') {
      await Promise.all([
        refreshOverviewWithCancel(fetchSeq, signal),
        refreshThroughputTrendWithCancel(fetchSeq, signal),
        refreshLatencyHistogramWithCancel(fetchSeq, signal)
      ])
    } else if (workspace === 'incidents') {
      await Promise.all([
        refreshErrorTrendWithCancel(fetchSeq, signal),
        refreshSwitchTrendWithCancel(fetchSeq, signal),
        refreshErrorDistributionWithCancel(fetchSeq, signal)
      ])
    }
    if (fetchSeq !== dashboardFetchSeq) return

    loadedWorkspaces.value[workspace] = true
    lastUpdated.value = new Date()

    // Newly visited child panels load themselves on mount. Manual/automatic
    // refreshes bump the shared token so already-mounted panels revalidate.
    if (options.refreshChildren !== false) {
      dashboardRefreshToken.value += 1
    }

    // Reset auto refresh countdown after successful fetch
    if (autoRefreshEnabled.value) {
      autoRefreshCountdown.value = Math.floor(autoRefreshIntervalMs.value / 1000)
    }

  } catch (err) {
    if (!isOpsDisabledError(err)) {
      console.error('[ops] failed to fetch dashboard data', err)
      errorMessage.value = t('admin.ops.failedToLoadData')
    }
  } finally {
    if (fetchSeq === dashboardFetchSeq) {
      loading.value = false
      hasLoadedOnce.value = true
    }
  }
}

watch(
  () => [timeRange.value, platform.value, groupId.value, queryMode.value] as const,
  () => {
    if (isApplyingRouteQuery.value) return
    loadedWorkspaces.value.traffic = false
    loadedWorkspaces.value.incidents = false
    if (opsEnabled.value) {
      fetchData({ refreshChildren: false })
    }
    syncQueryToRoute()
  }
)

watch(
  () => route.query,
  async () => {
    if (isSyncingRouteQuery.value) return

    const prevTimeRange = timeRange.value
    const prevPlatform = platform.value
    const prevGroupId = groupId.value
    const prevQueryMode = queryMode.value
    const prevWorkspace = activeWorkspace.value

    isApplyingRouteQuery.value = true
    applyRouteQueryToState()
    await nextTick()
    isApplyingRouteQuery.value = false

    const changed =
      prevTimeRange !== timeRange.value ||
      prevPlatform !== platform.value ||
      prevGroupId !== groupId.value ||
      prevQueryMode !== queryMode.value
    let workspaceFetchStarted = false
    if (changed) {
      loadedWorkspaces.value.traffic = false
      loadedWorkspaces.value.incidents = false
      if (opsEnabled.value) {
        workspaceFetchStarted = true
        await fetchData({ refreshChildren: false })
      }
    }
    if (prevWorkspace !== activeWorkspace.value) {
      if (!workspaceFetchStarted && !loadedWorkspaces.value[activeWorkspace.value] && opsEnabled.value) {
        await fetchData({ refreshChildren: false })
      }
      void nextTick(() => scrollWorkspaceIntoView(activeWorkspace.value))
    }
  }
)

onMounted(async () => {
  // Fullscreen mode: listen for ESC key
  window.addEventListener('keydown', handleKeydown)
  syncOpsOptionBBodyClass(true)
  previousSidebarCollapsed = appStore.sidebarCollapsed
  if (typeof appStore.setSidebarCollapsed === 'function') {
    appStore.setSidebarCollapsed(false)
  }

  await adminSettingsStore.fetch()
  if (!adminSettingsStore.opsMonitoringEnabled) {
    await router.replace('/admin/settings')
    return
  }

  await normalizeInvalidWorkspaceQuery()

  // Load thresholds configuration
  loadThresholds()

  // Load auto refresh settings
  await loadDashboardAdvancedSettings()

  if (opsEnabled.value && !loadedWorkspaces.value[activeWorkspace.value]) {
    await fetchData({ refreshChildren: false })
  }

  // Start auto refresh if enabled
  if (autoRefreshEnabled.value) {
    resumeCountdown()
  }
})

async function loadThresholds() {
  try {
    const thresholds = await opsAPI.getMetricThresholds()
    metricThresholds.value = thresholds || null
  } catch (err) {
    console.warn('[OpsDashboard] Failed to load thresholds', err)
    metricThresholds.value = null
  }
}

onUnmounted(() => {
  syncFullscreenBodyClass(false)
  syncOpsOptionBBodyClass(false)
  if (previousSidebarCollapsed !== null && typeof appStore.setSidebarCollapsed === 'function') {
    appStore.setSidebarCollapsed(previousSidebarCollapsed)
  }
  window.removeEventListener('keydown', handleKeydown)
  abortDashboardFetch()
  pauseCountdown()
})

// Watch auto refresh settings changes
watch(autoRefreshEnabled, (enabled) => {
  if (enabled) {
    autoRefreshCountdown.value = Math.floor(autoRefreshIntervalMs.value / 1000)
    resumeCountdown()
  } else {
    pauseCountdown()
    autoRefreshCountdown.value = 0
  }
})

// Reload auto refresh settings after settings dialog is closed
watch(showSettingsDialog, async (show) => {
  if (!show) {
    await loadDashboardAdvancedSettings()
  }
})
</script>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api'
import type { OpsDashboardOverview } from '@/api/admin/ops'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'

type TrafficSignal = 'health' | 'sla' | 'errors' | 'throughput' | 'concurrency'

const props = defineProps<{
  overview: OpsDashboardOverview | null
  platform: string
  groupId: number | null
  timeRange: string
  loading?: boolean
}>()

const emit = defineEmits<{
  'update:platform': [value: string]
  'update:group': [value: number | null]
  'update:time-range': [value: string]
  'select-signal': [value: TrafficSignal]
}>()

const { t } = useI18n()
const groups = ref<Array<{ id: number; name: string; platform: string }>>([])
const selectedSignal = ref<TrafficSignal>('health')

const platformOptions = computed(() => [
  { value: '', label: t('admin.ops.trafficRail.allPlatforms') },
  { value: 'openai', label: 'OpenAI' },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'gemini', label: 'Gemini' },
  { value: 'antigravity', label: 'Antigravity' },
  { value: 'grok', label: 'Grok' }
])

const groupOptions = computed(() => {
  const visible = props.platform
    ? groups.value.filter((group) => group.platform === props.platform)
    : groups.value
  return [
    { value: null, label: t('admin.ops.trafficRail.allGroups') },
    ...visible.map((group) => ({ value: group.id, label: group.name }))
  ]
})

const timeRangeOptions = computed(() => [
  { value: '5m', label: t('admin.ops.timeRange.5m') },
  { value: '30m', label: t('admin.ops.timeRange.30m') },
  { value: '1h', label: t('admin.ops.timeRange.1h') },
  { value: '6h', label: t('admin.ops.timeRange.6h') },
  { value: '24h', label: t('admin.ops.timeRange.24h') }
])

const healthScore = computed(() => {
  const value = props.overview?.health_score
  return typeof value === 'number' && Number.isFinite(value) ? value : null
})
const slaPercent = computed(() => {
  const value = props.overview?.sla
  return typeof value === 'number' && Number.isFinite(value) ? value * 100 : null
})
const errorPercent = computed(() => {
  const value = props.overview?.error_rate
  return typeof value === 'number' && Number.isFinite(value) ? value * 100 : null
})
const qps = computed(() => {
  const value = props.overview?.qps?.current
  return typeof value === 'number' && Number.isFinite(value) ? value : null
})
const tps = computed(() => {
  const value = props.overview?.tps?.current
  return typeof value === 'number' && Number.isFinite(value) ? value : null
})
const queueDepth = computed(() => {
  const value = props.overview?.system_metrics?.concurrency_queue_depth
  return typeof value === 'number' && Number.isFinite(value) ? value : null
})

const healthTone = computed(() => {
  if (healthScore.value == null) return 'neutral'
  if (healthScore.value >= 90) return 'success'
  return 'warning'
})
const slaTone = computed(() => {
  if (slaPercent.value == null) return 'neutral'
  if (slaPercent.value >= 95) return 'success'
  return 'danger'
})
const errorTone = computed(() => {
  if (errorPercent.value == null) return 'neutral'
  if (errorPercent.value <= 1) return 'success'
  return 'danger'
})
const queueTone = computed(() => {
  if (queueDepth.value == null) return 'neutral'
  return queueDepth.value > 0 ? 'warning' : 'success'
})

const healthSummary = computed(() => {
  if (healthScore.value == null) return t('admin.ops.idleStatus')
  if (healthScore.value >= 90) return t('admin.ops.healthyStatus')
  return t('admin.ops.riskyStatus')
})

function selectSignal(signal: TrafficSignal): void {
  selectedSignal.value = signal
  emit('select-signal', signal)
}

function updatePlatform(value: string | number | boolean | null): void {
  emit('update:platform', typeof value === 'string' ? value : '')
}

function updateGroup(value: string | number | boolean | null): void {
  if (value === null || value === '' || typeof value === 'boolean') {
    emit('update:group', null)
    return
  }
  const parsed = typeof value === 'number' ? value : Number.parseInt(value, 10)
  emit('update:group', Number.isFinite(parsed) && parsed > 0 ? parsed : null)
}

function updateTimeRange(value: string | number | boolean | null): void {
  if (typeof value === 'string' && value) emit('update:time-range', value)
}

watch(() => props.platform, (platform) => {
  if (!platform || props.groupId == null) return
  const selectedGroup = groups.value.find((group) => group.id === props.groupId)
  if (selectedGroup && selectedGroup.platform !== platform) emit('update:group', null)
})

onMounted(async () => {
  try {
    const result = await adminAPI.groups.getAll()
    groups.value = result.map((group) => ({ id: group.id, name: group.name, platform: group.platform }))
  } catch (error) {
    console.error('[OpsTrafficInvestigationRail] Failed to load groups', error)
    groups.value = []
  }
})
</script>

<template>
  <div class="ops-traffic-rail" data-testid="ops-traffic-investigation-rail">
    <section class="ops-traffic-rail__scope" aria-labelledby="ops-traffic-scope-heading">
      <div class="ops-traffic-rail__section-heading">
        <Icon name="filter" size="sm" aria-hidden="true" />
        <h3 id="ops-traffic-scope-heading">{{ t('admin.ops.trafficRail.currentScope') }}</h3>
      </div>
      <div class="ops-traffic-rail__filters">
        <div class="ops-traffic-rail__filter-field">
          <span class="sr-only">{{ t('admin.ops.errorLog.platform') }}</span>
          <Select
            :model-value="platform"
            :options="platformOptions"
            :aria-label="t('admin.ops.errorLog.platform')"
            @update:model-value="updatePlatform"
          />
        </div>
        <div class="ops-traffic-rail__filter-field">
          <span class="sr-only">{{ t('admin.ops.errorLog.group') }}</span>
          <Select
            :model-value="groupId"
            :options="groupOptions"
            :aria-label="t('admin.ops.errorLog.group')"
            @update:model-value="updateGroup"
          />
        </div>
        <div class="ops-traffic-rail__filter-field">
          <span class="sr-only">{{ t('admin.ops.systemLogs.timeRange') }}</span>
          <Select
            :model-value="timeRange"
            :options="timeRangeOptions"
            :aria-label="t('admin.ops.systemLogs.timeRange')"
            @update:model-value="updateTimeRange"
          />
        </div>
      </div>
    </section>

    <section class="ops-traffic-rail__signals" :aria-label="t('admin.ops.trafficRail.coreSignals')">
      <h3>{{ t('admin.ops.trafficRail.coreSignals') }}</h3>

      <button
        type="button"
        class="ops-traffic-signal ops-traffic-signal--health"
        data-testid="ops-traffic-signal-health"
        :aria-pressed="selectedSignal === 'health'"
        :class="[`ops-traffic-signal--${healthTone}`, { 'ops-traffic-signal--active': selectedSignal === 'health' }]"
        :disabled="loading"
        @click="selectSignal('health')"
      >
        <span class="ops-traffic-signal__label">
          {{ t('admin.ops.trafficRail.healthScore') }}
          <span class="ops-traffic-signal__badge">{{ healthSummary }}</span>
        </span>
        <strong>{{ healthScore ?? '—' }}</strong>
        <small>{{ t('admin.ops.trafficRail.healthBasis') }}</small>
      </button>

      <button
        type="button"
        class="ops-traffic-signal ops-traffic-signal--sla"
        data-testid="ops-traffic-signal-sla"
        :aria-pressed="selectedSignal === 'sla'"
        :class="[`ops-traffic-signal--${slaTone}`, { 'ops-traffic-signal--active': selectedSignal === 'sla' }]"
        :disabled="loading"
        @click="selectSignal('sla')"
      >
        <span class="ops-traffic-signal__label">
          {{ t('admin.ops.trafficRail.slaAttainment') }}
          <Icon v-if="slaTone === 'success'" name="checkCircle" size="sm" class="ops-traffic-signal__check" aria-hidden="true" />
          <i v-else aria-hidden="true"></i>
        </span>
        <strong>{{ slaPercent == null ? '—' : `${slaPercent.toFixed(3)}%` }}</strong>
        <small>{{ t('admin.ops.exceptions') }} {{ overview?.error_count_sla ?? '—' }} · {{ t('admin.ops.timeRange.1h') }}</small>
      </button>

      <button
        type="button"
        class="ops-traffic-signal ops-traffic-signal--errors"
        data-testid="ops-traffic-signal-errors"
        :aria-pressed="selectedSignal === 'errors'"
        :class="[`ops-traffic-signal--${errorTone}`, { 'ops-traffic-signal--active': selectedSignal === 'errors' }]"
        :disabled="loading"
        @click="selectSignal('errors')"
      >
        <span class="ops-traffic-signal__label">
          {{ t('admin.ops.trafficRail.requestErrorRate') }}
          <i aria-hidden="true"></i>
        </span>
        <strong>{{ errorPercent == null ? '—' : `${errorPercent.toFixed(2)}%` }}</strong>
        <small>{{ t('admin.ops.errorCount') }} {{ overview?.error_count_sla ?? '—' }} · {{ t('admin.ops.trafficRail.excludedBusinessLimits') }}</small>
      </button>

      <button
        type="button"
        class="ops-traffic-signal ops-traffic-signal--throughput"
        data-testid="ops-traffic-signal-throughput"
        :aria-pressed="selectedSignal === 'throughput'"
        :class="{ 'ops-traffic-signal--active': selectedSignal === 'throughput' }"
        :disabled="loading"
        @click="selectSignal('throughput')"
      >
        <span class="ops-traffic-signal__label">
          {{ t('admin.ops.trafficRail.realtimeThroughput') }}
        </span>
        <strong>{{ qps == null ? '—' : qps.toFixed(1) }} <em>QPS</em></strong>
        <small>{{ tps == null ? '—' : tps.toFixed(1) }} TPS (Token/s)</small>
      </button>

      <button
        type="button"
        class="ops-traffic-signal ops-traffic-signal--queue"
        data-testid="ops-traffic-signal-concurrency"
        :aria-pressed="selectedSignal === 'concurrency'"
        :class="[`ops-traffic-signal--${queueTone}`, { 'ops-traffic-signal--active': selectedSignal === 'concurrency' }]"
        :disabled="loading"
        @click="selectSignal('concurrency')"
      >
        <span class="ops-traffic-signal__label">
          {{ t('admin.ops.trafficRail.concurrencyQueue') }}
        </span>
        <strong>{{ queueDepth ?? '—' }}</strong>
        <small>{{ t('admin.ops.trafficRail.queueDepth', { count: queueDepth ?? '—' }) }}</small>
      </button>
    </section>

    <footer class="ops-traffic-rail__footer">
      <button type="button" @click="selectSignal('health')">
        <span>
          <Icon name="server" size="sm" aria-hidden="true" />
          {{ t('admin.ops.trafficRail.baseHealth') }}
        </span>
        <Icon name="chevronDown" size="sm" aria-hidden="true" />
      </button>
    </footer>
  </div>
</template>

<style scoped>
.ops-traffic-rail {
  min-width: 0;
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface);
}

.ops-traffic-rail__scope,
.ops-traffic-rail__signals {
  padding: 18px 16px;
}

.ops-traffic-rail__scope {
  border-bottom: 1px solid var(--lx-clay-border);
}

.ops-traffic-rail__section-heading {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--lx-clay-text-secondary);
}

.ops-traffic-rail__section-heading h3,
.ops-traffic-rail__signals > h3 {
  margin: 0;
  font-size: 0.75rem;
  font-weight: 800;
}

.ops-traffic-rail__filters {
  display: grid;
  gap: 8px;
  margin-top: 14px;
}

.ops-traffic-rail__filter-field {
  display: grid;
  min-width: 0;
  gap: 5px;
}

.ops-traffic-rail__filter-field > span {
  color: var(--lx-clay-text-muted);
  font-size: 0.6875rem;
  font-weight: 750;
}

.ops-traffic-rail__filter-field > * {
  min-width: 0;
}

.ops-traffic-rail__signals {
  display: grid;
  gap: 10px;
}

.ops-traffic-rail__signals > h3 {
  margin-bottom: 2px;
  color: var(--lx-clay-text-muted);
}

.ops-traffic-signal {
  display: grid;
  min-height: 96px;
  gap: 6px;
  padding: 13px 14px;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-ops);
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface);
  text-align: left;
  cursor: pointer;
  transition: border-color 160ms ease-out, background-color 160ms ease-out;
}

.ops-traffic-signal:hover:not(:disabled) {
  border-color: var(--lx-clay-border-strong);
  background: var(--lx-clay-surface-soft);
}

.ops-traffic-signal:disabled {
  cursor: wait;
  opacity: 0.68;
}

.ops-traffic-signal__label {
  color: var(--lx-clay-text-secondary);
  font-size: 0.75rem;
  font-weight: 800;
}

.ops-traffic-signal strong {
  color: var(--lx-clay-text);
  font-size: 1.5rem;
  font-variant-numeric: tabular-nums;
  font-weight: 900;
  letter-spacing: -0.03em;
  line-height: 1;
}

.ops-traffic-signal strong em {
  font-size: 0.6875rem;
  font-style: normal;
  font-weight: 750;
  letter-spacing: 0;
}

.ops-traffic-signal small {
  color: var(--lx-clay-text-muted);
  font-size: 0.6875rem;
  line-height: 1.4;
}

.ops-traffic-signal--success strong {
  color: var(--lx-clay-success);
}

.ops-traffic-signal--warning strong {
  color: var(--lx-clay-warning);
}

.ops-traffic-signal--danger strong {
  color: var(--lx-clay-danger);
}

@media (max-width: 860px) {
  .ops-traffic-rail__signals {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .ops-traffic-rail__signals > h3 {
    grid-column: 1 / -1;
  }
}

@media (max-width: 520px) {
  .ops-traffic-rail__signals {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (prefers-reduced-motion: reduce) {
  .ops-traffic-signal {
    transition-duration: 0.01ms;
  }
}

/* Superdesign Option B investigative rail. */
.ops-traffic-rail {
  display: flex;
  width: 100%;
  min-height: 0;
  flex: 1 1 auto;
  flex-direction: column;
  overflow-x: hidden;
  overflow-y: auto;
  background: #ffffff;
  scrollbar-width: none;
}

.ops-traffic-rail::-webkit-scrollbar {
  display: none;
}

.ops-traffic-rail__scope {
  flex: 0 0 auto;
  padding: 16px;
  border-bottom: 1px solid rgba(91, 80, 112, 0.14);
  background: #fcfcfc;
}

.ops-traffic-rail__section-heading,
.ops-traffic-rail__signals > h3 {
  color: #756e80;
  font-size: 11px;
  font-weight: 800;
  line-height: 16px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.ops-traffic-rail__section-heading {
  gap: 6px;
}

.ops-traffic-rail__filters {
  gap: 8px;
  margin-top: 12px;
}

.ops-traffic-rail__filter-field {
  gap: 0;
}

.ops-traffic-rail__filter-field :deep(.select-trigger) {
  min-height: 36px;
  padding: 0 12px;
  border: 1px solid rgba(91, 80, 112, 0.14);
  border-radius: 10px;
  color: #332f3a;
  background: #ffffff;
  box-shadow: none;
  font-size: 13px;
  font-weight: 500;
}

.ops-traffic-rail__filter-field :deep(.select-trigger:hover) {
  border-color: rgba(91, 80, 112, 0.23);
  background: #ffffff;
}

.ops-traffic-rail__signals {
  display: flex;
  min-height: 0;
  flex: 0 0 auto;
  flex-direction: column;
  gap: 4px;
  overflow: visible;
  padding: 12px;
}

.ops-traffic-rail__signals > h3 {
  flex: 0 0 auto;
  margin: 0;
  padding: 16px 8px 12px;
}

.ops-traffic-signal {
  display: flex;
  min-height: 0;
  flex: 0 0 auto;
  flex-direction: column;
  gap: 0;
  padding: 16px;
  border: 1px solid transparent;
  border-radius: 12px;
  color: #332f3a;
  background: transparent;
  box-shadow: none;
  text-align: left;
  transition: all 180ms cubic-bezier(0.4, 0, 0.2, 1);
}

.ops-traffic-signal:hover:not(:disabled) {
  border-color: transparent;
  background: #f4f1fa;
}

.ops-traffic-signal--active {
  border-color: rgba(91, 80, 112, 0.23);
  background: #efebf5;
}

.ops-traffic-signal--active:hover:not(:disabled) {
  border-color: rgba(91, 80, 112, 0.23);
  background: #f4f1fa;
}

.ops-traffic-signal:focus-visible {
  outline: 2px solid #7c3aed;
  outline-offset: 2px;
}

.ops-traffic-signal__label {
  display: flex;
  width: 100%;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  color: #6b7280;
  font-size: 12px;
  font-weight: 700;
  line-height: 16px;
}

.ops-traffic-signal__label i {
  width: 8px;
  height: 8px;
  flex: 0 0 auto;
  border-radius: 999px;
  background: #9ca3af;
}

.ops-traffic-signal__check {
  width: 14px;
  height: 14px;
  flex: 0 0 auto;
  color: #10b981;
}

.ops-traffic-signal__badge {
  padding: 2px 6px;
  border-radius: 4px;
  color: #6b7280;
  background: #f3f4f6;
  font-size: 10px;
  font-weight: 700;
  line-height: 14px;
  letter-spacing: -0.01em;
}

.ops-traffic-signal strong {
  margin-top: 4px;
  color: #1f2937;
  font-size: 20px;
  font-weight: 700;
  line-height: 28px;
  letter-spacing: -0.025em;
  font-variant-numeric: tabular-nums;
}

.ops-traffic-signal--health strong {
  font-size: 24px;
  font-weight: 900;
  line-height: 32px;
}

.ops-traffic-signal strong em {
  color: #9ca3af;
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
}

.ops-traffic-signal small {
  margin-top: 4px;
  color: #9ca3af;
  font-size: 11px;
  line-height: 1.5;
}

.ops-traffic-signal--success .ops-traffic-signal__label i {
  background: #10b981;
}

.ops-traffic-signal--warning .ops-traffic-signal__label i {
  background: #f59e0b;
}

.ops-traffic-signal--danger .ops-traffic-signal__label i {
  background: #f43f5e;
}

.ops-traffic-signal--success strong {
  color: #059669;
}

.ops-traffic-signal--warning strong {
  color: #d97706;
}

.ops-traffic-signal--danger strong {
  color: #e11d48;
}

.ops-traffic-signal--warning .ops-traffic-signal__badge {
  color: #b45309;
  background: #fffbeb;
}

.ops-traffic-signal--success .ops-traffic-signal__badge {
  color: #047857;
  background: #ecfdf5;
}

.ops-traffic-signal--errors.ops-traffic-signal--danger small {
  color: #fb7185;
  font-weight: 500;
}

.ops-traffic-rail__footer {
  flex: 0 0 auto;
  padding: 16px;
  border-top: 1px solid rgba(91, 80, 112, 0.14);
  background: #fcfcfc;
}

.ops-traffic-rail__footer button {
  display: flex;
  width: 100%;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  border: 0;
  color: #635f69;
  background: transparent;
  font-size: 12px;
  font-weight: 700;
}

.ops-traffic-rail__footer button > span {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

@media (max-width: 860px) {
  .ops-traffic-rail {
    overflow: visible;
  }

  .ops-traffic-rail__signals {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    overflow: visible;
  }
}
</style>

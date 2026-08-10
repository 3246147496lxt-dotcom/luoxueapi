<template>
  <section class="usage-stat-rail" :aria-label="t('admin.usage.workspace.summaryLabel')">
    <article
      class="usage-stat"
      :aria-label="`${t('usage.totalRequests')} ${stats?.total_requests?.toLocaleString() || '0'}，${t('usage.inSelectedRange')}`"
    >
      <div class="usage-stat__primary">
        <span class="usage-stat__label">{{ t('usage.totalRequests') }}</span>
        <span class="usage-stat__value">{{ stats?.total_requests?.toLocaleString() || '0' }}</span>
      </div>
      <p class="usage-stat__meta">{{ t('usage.inSelectedRange') }}</p>
    </article>
    <article
      ref="tokenStatRef"
      class="usage-stat usage-stat--has-tooltip"
      tabindex="0"
      :aria-describedby="tokenTooltipId"
      :aria-label="`${t('usage.totalTokens')} ${formatTokens(stats?.total_tokens || 0)}`"
      @mouseenter="hoveredTooltip = 'tokens'"
      @mouseleave="hoveredTooltip = null"
      @focus="focusedTooltip = 'tokens'"
      @blur="focusedTooltip = null"
    >
      <div class="usage-stat__primary">
        <span class="usage-stat__label">{{ t('usage.totalTokens') }}</span>
        <span class="usage-stat__value">{{ formatTokens(stats?.total_tokens || 0) }}</span>
      </div>
      <p class="usage-stat__meta">
        <span>{{ t('usage.in') }} {{ formatTokens(stats?.total_input_tokens || 0) }}</span>
        <span aria-hidden="true">/</span>
        <span>{{ t('usage.out') }} {{ formatTokens(stats?.total_output_tokens || 0) }}</span>
        <svg class="usage-stat__info" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      </p>
    </article>
    <article
      ref="costStatRef"
      class="usage-stat usage-stat--has-tooltip"
      tabindex="0"
      :aria-describedby="costTooltipId"
      :aria-label="`${t('usage.totalCost')} ${(stats?.total_actual_cost || 0).toFixed(4)}`"
      @mouseenter="hoveredTooltip = 'cost'"
      @mouseleave="hoveredTooltip = null"
      @focus="focusedTooltip = 'cost'"
      @blur="focusedTooltip = null"
    >
      <div class="usage-stat__primary">
        <span class="usage-stat__label">{{ t('usage.totalCost') }}</span>
        <CreditAmount
          v-if="creditMode"
          class="usage-stat__value usage-stat__value--success"
          :value="formatCompactCost(stats?.total_actual_cost || 0)"
          icon-size="xs"
          :label="`${t('usage.totalCost')} ${(stats?.total_actual_cost || 0).toFixed(4)}`"
        />
        <span v-else class="usage-stat__value usage-stat__value--success">
          ${{ formatCompactCost(stats?.total_actual_cost || 0) }}
        </span>
      </div>
      <p class="usage-stat__meta usage-stat__meta--cost">
        <template v-if="showAccountCost && totalAccountCost != null">
          <span>{{ t('usage.accountCost') }} ${{ formatCompactCost(totalAccountCost) }}</span>
          <span class="usage-stat__divider" aria-hidden="true">|</span>
        </template>
        <span>
          {{ t('usage.standardCost') }}
          <span :class="{ 'line-through': strikeStandardCost }">${{ formatCompactCost(stats?.total_cost || 0) }}</span>
        </span>
      </p>
    </article>
    <article
      class="usage-stat"
      :aria-label="`${t('usage.avgDuration')} ${formatDuration(stats?.average_duration_ms || 0)}，${t('usage.averageDurationHint')}`"
    >
      <div class="usage-stat__primary">
        <span class="usage-stat__label">{{ t('usage.avgDuration') }}</span>
        <span class="usage-stat__value">{{ formatDuration(stats?.average_duration_ms || 0) }}</span>
      </div>
      <p class="usage-stat__meta">{{ t('usage.averageDurationHint') }}</p>
    </article>
  </section>

  <Teleport to="body">
    <div
      :id="tokenTooltipId"
      ref="tokenTooltipRef"
      class="usage-stat__tooltip"
      :class="{ 'usage-stat__tooltip--visible': activeTooltip === 'tokens' && tooltipReady }"
      :style="tooltipPosition"
      role="tooltip"
      data-ui-portal="usage-stat-tooltip"
    >
      <strong>{{ t('usage.tokenDetails') }}</strong>
      <span>{{ t('usage.totalTokens') }}: {{ formatExactNumber(stats?.total_tokens || 0) }}</span>
      <span>{{ t('usage.in') }}: {{ formatExactNumber(stats?.total_input_tokens || 0) }}</span>
      <span>{{ t('usage.out') }}: {{ formatExactNumber(stats?.total_output_tokens || 0) }}</span>
      <span>{{ cacheLabel() }}: {{ formatExactNumber(stats?.total_cache_tokens || 0) }}</span>
      <span>{{ t('usage.cacheCreationTokensLabel') }}: {{ formatExactNumber(stats?.total_cache_creation_tokens || 0) }}</span>
      <span>{{ t('usage.cacheReadTokensLabel') }}: {{ formatExactNumber(stats?.total_cache_read_tokens || 0) }}</span>
    </div>
    <div
      :id="costTooltipId"
      ref="costTooltipRef"
      class="usage-stat__tooltip"
      :class="{ 'usage-stat__tooltip--visible': activeTooltip === 'cost' && tooltipReady }"
      :style="tooltipPosition"
      role="tooltip"
      data-ui-portal="usage-stat-tooltip"
    >
      <strong>{{ t('usage.costDetails') }}</strong>
      <span>{{ t('usage.totalCost') }}: {{ (stats?.total_actual_cost || 0).toFixed(4) }}</span>
      <span v-if="showAccountCost && totalAccountCost != null">
        {{ t('usage.accountCost') }}: ${{ totalAccountCost.toFixed(4) }}
      </span>
      <span>{{ t('usage.standardCost') }}: ${{ (stats?.total_cost || 0).toFixed(4) }}</span>
    </div>
  </Teleport>
</template>

<script lang="ts">
let usageStatTooltipInstanceCounter = 0
</script>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UsageStatsResponse } from '@/types'
import CreditAmount from '@/components/common/CreditAmount.vue'

interface UsageStats extends UsageStatsResponse {
  total_account_cost?: number
}

const props = withDefaults(defineProps<{
  stats: UsageStats | null
  showAccountCost?: boolean
  strikeStandardCost?: boolean
  creditMode?: boolean
}>(), {
  showAccountCost: true,
  strikeStandardCost: false,
  creditMode: false,
})

const { t } = useI18n()

type TooltipKind = 'tokens' | 'cost'

const tooltipInstanceId = ++usageStatTooltipInstanceCounter
const tokenTooltipId = `usage-token-summary-tooltip-${tooltipInstanceId}`
const costTooltipId = `usage-cost-summary-tooltip-${tooltipInstanceId}`
const tokenStatRef = ref<HTMLElement | null>(null)
const costStatRef = ref<HTMLElement | null>(null)
const tokenTooltipRef = ref<HTMLElement | null>(null)
const costTooltipRef = ref<HTMLElement | null>(null)
const hoveredTooltip = ref<TooltipKind | null>(null)
const focusedTooltip = ref<TooltipKind | null>(null)
const activeTooltip = computed(() => hoveredTooltip.value ?? focusedTooltip.value)
const tooltipReady = ref(false)
const tooltipPosition = ref({ left: '-9999px', top: '-9999px' })

const totalAccountCost = computed(() => {
  return props.stats?.total_account_cost ?? null
})
const showAccountCost = computed(() => props.showAccountCost)
const strikeStandardCost = computed(() => props.strikeStandardCost)
const creditMode = computed(() => props.creditMode)

const formatDuration = (ms: number) =>
  ms < 1000 ? `${ms.toFixed(0)}ms` : `${(ms / 1000).toFixed(2)}s`

const formatTokens = (value: number) => {
  if (value >= 1e9) return (value / 1e9).toFixed(2) + 'B'
  if (value >= 1e6) return (value / 1e6).toFixed(2) + 'M'
  if (value >= 1e3) return (value / 1e3).toFixed(2) + 'K'
  return value.toLocaleString()
}

const formatExactNumber = (value: number) => value.toLocaleString()

const formatCompactCost = (value: number) => {
  const absoluteValue = Math.abs(value)
  return absoluteValue > 0 && absoluteValue < 0.01 ? value.toFixed(4) : value.toFixed(2)
}

const cacheLabel = () => t('usage.cacheTotal')

function updateTooltipPosition() {
  const kind = activeTooltip.value
  if (!kind || typeof window === 'undefined') return

  const anchor = kind === 'tokens' ? tokenStatRef.value : costStatRef.value
  const tooltip = kind === 'tokens' ? tokenTooltipRef.value : costTooltipRef.value
  if (!anchor || !tooltip) return

  const anchorRect = anchor.getBoundingClientRect()
  const tooltipRect = tooltip.getBoundingClientRect()
  const viewportWidth = window.innerWidth || document.documentElement.clientWidth || 1024
  const viewportHeight = window.innerHeight || document.documentElement.clientHeight || 768
  const edge = 8
  const gap = 8
  const width = tooltipRect.width || 260
  const height = tooltipRect.height || (kind === 'tokens' ? 128 : 84)
  const centeredLeft = anchorRect.left + (anchorRect.width - width) / 2
  const left = Math.min(Math.max(centeredLeft, edge), Math.max(edge, viewportWidth - width - edge))
  const below = anchorRect.bottom + gap
  const top = below + height <= viewportHeight - edge
    ? below
    : Math.max(edge, anchorRect.top - height - gap)

  tooltipPosition.value = {
    left: `${Math.round(left)}px`,
    top: `${Math.round(top)}px`,
  }
  tooltipReady.value = true
}

watch(activeTooltip, async (kind) => {
  tooltipReady.value = false
  if (!kind) return
  await nextTick()
  updateTooltipPosition()
})

onMounted(() => {
  window.addEventListener('resize', updateTooltipPosition)
  window.addEventListener('scroll', updateTooltipPosition, { capture: true, passive: true })
})

onUnmounted(() => {
  window.removeEventListener('resize', updateTooltipPosition)
  window.removeEventListener('scroll', updateTooltipPosition, { capture: true })
})
</script>

<style scoped>
.usage-stat-rail {
  container-type: inline-size;
  display: flex;
  min-width: 0;
  overflow-x: auto;
  overflow-y: hidden;
  border: 1px solid var(--lx-clay-border);
  border-radius: 10px;
  gap: 1px;
  background: var(--lx-clay-border);
  overscroll-behavior-inline: contain;
  scroll-padding-inline: 16px;
  scroll-snap-type: x mandatory;
  scrollbar-width: none;
  -ms-overflow-style: none;
  -webkit-overflow-scrolling: touch;
}

.usage-stat-rail::-webkit-scrollbar {
  display: none;
}

.usage-stat {
  position: relative;
  display: flex;
  width: 210px;
  height: 54px;
  min-width: 210px;
  flex: 0 0 210px;
  flex-direction: column;
  justify-content: center;
  box-sizing: border-box;
  padding: 6px 16px;
  background: var(--lx-clay-surface);
  outline: none;
  scroll-snap-align: start;
}

.usage-stat:focus-visible {
  z-index: 2;
  box-shadow: inset 0 0 0 2px var(--lx-clay-accent);
}

.usage-stat__primary {
  display: flex;
  min-width: 0;
  align-items: baseline;
  gap: 8px;
}

.usage-stat__label,
.usage-stat__meta,
.usage-stat__value {
  margin: 0;
}

.usage-stat__label {
  flex: 0 0 auto;
  color: var(--lx-clay-text-muted);
  font-size: 11px;
  font-weight: 700;
  line-height: 1;
  white-space: nowrap;
}

.usage-stat__value {
  min-width: 0;
  color: var(--lx-clay-text);
  font-size: 18px;
  font-weight: 900;
  line-height: 1;
  letter-spacing: -0.015em;
}

.usage-stat__value--success {
  color: var(--lx-clay-success);
}

.usage-stat__meta {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 4px;
  margin-top: 2px;
  overflow: hidden;
  color: var(--lx-clay-text-muted);
  font-size: 10px;
  line-height: 1.2;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.usage-stat__meta--cost {
  color: var(--lx-clay-warning);
}

.usage-stat__divider {
  color: var(--lx-clay-border-strong);
}

.usage-stat__info {
  width: 12px;
  height: 12px;
  flex: 0 0 auto;
  color: var(--lx-clay-text-subtle);
}

.usage-stat__tooltip {
  position: fixed;
  z-index: 60;
  display: flex;
  width: max-content;
  min-width: 192px;
  max-width: 260px;
  flex-direction: column;
  gap: 4px;
  padding: 9px 10px;
  border: 1px solid color-mix(in srgb, var(--lx-clay-border) 78%, white);
  border-radius: 8px;
  color: #ffffff;
  background: var(--lx-clay-code-surface);
  box-shadow: var(--lx-clay-shadow-overlay);
  font-size: 10px;
  font-weight: 500;
  line-height: 1.35;
  opacity: 0;
  pointer-events: none;
  transform: translateY(-3px);
  transition:
    opacity 140ms ease,
    transform 140ms ease;
}

.usage-stat__tooltip strong {
  font-size: 11px;
  font-weight: 800;
}

.usage-stat__tooltip--visible {
  opacity: 1;
  transform: translateY(0);
}

@container (min-width: 760px) {
  .usage-stat {
    width: auto;
    min-width: 0;
    flex: 1 1 0;
  }
}

@media (min-width: 1100px) {
  .usage-stat-rail {
    overflow: visible;
  }
}

@media (prefers-reduced-motion: reduce) {
  .usage-stat__tooltip {
    transition: none;
  }
}
</style>

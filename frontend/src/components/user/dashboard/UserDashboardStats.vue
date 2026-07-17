<template>
  <!-- Core metrics: one continuous horizontal strip instead of separate statistic cards. -->
  <section
    class="card grid grid-cols-1 gap-px overflow-hidden bg-gray-100 sm:grid-cols-2 dark:bg-dark-700"
    :class="isSimple ? 'xl:grid-cols-3' : 'xl:grid-cols-4'"
  >
    <div v-if="!isSimple" class="min-w-0 bg-white p-5 md:p-6 dark:bg-dark-800">
      <div class="flex items-center justify-between gap-4">
        <p class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ t('dashboard.balance') }}</p>
        <span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">
          <Icon name="dollar" size="sm" :stroke-width="1.75" />
        </span>
      </div>
      <p class="mt-5 truncate text-3xl font-semibold tracking-tight text-gray-950 dark:text-white">
        ${{ formatBalance(balance) }}
      </p>
      <p class="mt-2 text-xs font-medium text-primary-700 dark:text-primary-300">{{ t('common.available') }}</p>
    </div>

    <div class="min-w-0 bg-white p-5 md:p-6 dark:bg-dark-800">
      <div class="flex items-center justify-between gap-4">
        <p class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ t('dashboard.apiKeys') }}</p>
        <span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">
          <Icon name="key" size="sm" :stroke-width="1.75" />
        </span>
      </div>
      <p class="mt-5 text-3xl font-semibold tracking-tight text-gray-950 dark:text-white">
        {{ stats?.total_api_keys || 0 }}
      </p>
      <p class="mt-2 text-xs font-medium text-primary-700 dark:text-primary-300">
        {{ stats?.active_api_keys || 0 }} {{ t('common.active') }}
      </p>
    </div>

    <div class="min-w-0 bg-white p-5 md:p-6 dark:bg-dark-800">
      <div class="flex items-center justify-between gap-4">
        <p class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ t('dashboard.todayRequests') }}</p>
        <span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">
          <Icon name="chart" size="sm" :stroke-width="1.75" />
        </span>
      </div>
      <p class="mt-5 text-3xl font-semibold tracking-tight text-gray-950 dark:text-white">
        {{ formatNumber(stats?.today_requests || 0) }}
      </p>
      <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">
        {{ t('common.total') }}: {{ formatNumber(stats?.total_requests || 0) }}
      </p>
    </div>

    <div class="min-w-0 bg-white p-5 md:p-6 dark:bg-dark-800">
      <div class="flex items-center justify-between gap-4">
        <p class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ t('dashboard.todayCost') }}</p>
        <span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-300">
          <Icon name="dollar" size="sm" :stroke-width="1.75" />
        </span>
      </div>
      <p class="mt-5 truncate text-3xl font-semibold tracking-tight text-gray-950 dark:text-white" :title="t('dashboard.actual')">
        ${{ formatCost(stats?.today_actual_cost || 0) }}
      </p>
      <div class="mt-2 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs">
        <span class="text-gray-500 dark:text-gray-400" :title="t('dashboard.standard')">
          {{ t('dashboard.standard') }} ${{ formatCost(stats?.today_cost || 0) }}
        </span>
        <span class="text-gray-300 dark:text-dark-600">·</span>
        <span class="text-primary-700 dark:text-primary-300" :title="t('dashboard.actual')">
          {{ t('common.total') }} ${{ formatCost(stats?.total_actual_cost || 0) }}
        </span>
        <span class="text-gray-400 dark:text-gray-500" :title="t('dashboard.standard')">
          / ${{ formatCost(stats?.total_cost || 0) }}
        </span>
      </div>
    </div>
  </section>

  <!-- Token composition and performance are paired as a primary instrument + summary. -->
  <div class="grid grid-cols-1 gap-5 xl:grid-cols-[minmax(0,1.7fr)_minmax(17rem,0.8fr)]">
    <section class="card p-5 md:p-6">
      <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div class="flex items-center gap-3">
          <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">
            <Icon name="cube" size="md" :stroke-width="1.75" />
          </span>
          <div>
            <p class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ t('dashboard.todayTokens') }}</p>
            <p class="mt-0.5 text-3xl font-semibold tracking-tight text-gray-950 dark:text-white">
              {{ formatTokens(stats?.today_tokens || 0) }}
            </p>
          </div>
        </div>
        <div class="sm:text-right">
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('dashboard.totalTokens') }}</p>
          <p class="mt-1 flex items-center gap-2 text-lg font-semibold text-gray-900 sm:justify-end dark:text-white">
            <Icon name="database" size="sm" :stroke-width="1.75" class="text-gray-400" />
            {{ formatTokens(stats?.total_tokens || 0) }}
          </p>
          <p class="mt-1 text-[11px] text-gray-400 dark:text-gray-500">
            {{ t('dashboard.input') }} {{ formatTokens(stats?.total_input_tokens || 0) }} ·
            {{ t('dashboard.output') }} {{ formatTokens(stats?.total_output_tokens || 0) }}
          </p>
        </div>
      </div>

      <div class="mt-8 grid gap-6 md:grid-cols-[8.5rem_minmax(0,1fr)] md:items-center">
        <div class="md:border-r md:border-gray-100 md:pr-6 dark:md:border-dark-700">
          <p class="text-5xl font-semibold tracking-tight text-gray-950 dark:text-white">
            {{ Math.round(todayTokenComposition.inputShare) }}<span class="ml-1 text-lg font-medium text-gray-400">%</span>
          </p>
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">{{ t('dashboard.input') }}</p>
        </div>

        <div class="min-w-0">
          <div
            class="flex h-4 w-full overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700"
            role="img"
            :aria-label="`${t('dashboard.input')} ${Math.round(todayTokenComposition.inputShare)}%, ${t('dashboard.output')} ${Math.round(todayTokenComposition.outputShare)}%, ${t('dashboard.cache')} ${Math.round(todayTokenComposition.cacheShare)}%`"
          >
            <span
              class="h-full bg-primary-500"
              :style="{ width: `${todayTokenComposition.inputShare}%` }"
              :title="`${t('dashboard.input')}: ${formatTokens(todayTokenComposition.input)}`"
            />
            <span
              class="h-full bg-amber-200 dark:bg-amber-400"
              :style="{ width: `${todayTokenComposition.outputShare}%` }"
              :title="`${t('dashboard.output')}: ${formatTokens(todayTokenComposition.output)}`"
            />
            <span
              class="h-full bg-gray-300 dark:bg-dark-500"
              :style="{ width: `${todayTokenComposition.cacheShare}%` }"
              :title="`${t('dashboard.cache')}: ${formatTokens(todayTokenComposition.cache)}`"
            />
          </div>

          <dl class="mt-5 grid grid-cols-1 gap-3 sm:grid-cols-3">
            <div class="min-w-0">
              <dt class="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                <span class="h-2 w-2 rounded-full bg-primary-500" />
                {{ t('dashboard.input') }}
              </dt>
              <dd class="mt-1 truncate font-mono text-sm font-semibold text-gray-900 dark:text-white">
                {{ formatTokens(todayTokenComposition.input) }}
              </dd>
            </div>
            <div class="min-w-0">
              <dt class="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                <span class="h-2 w-2 rounded-full bg-amber-200 dark:bg-amber-400" />
                {{ t('dashboard.output') }}
              </dt>
              <dd class="mt-1 truncate font-mono text-sm font-semibold text-gray-900 dark:text-white">
                {{ formatTokens(todayTokenComposition.output) }}
              </dd>
            </div>
            <div class="min-w-0">
              <dt class="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                <span class="h-2 w-2 rounded-full bg-gray-300 dark:bg-dark-500" />
                {{ t('dashboard.cache') }}
              </dt>
              <dd class="mt-1 truncate font-mono text-sm font-semibold text-gray-900 dark:text-white">
                {{ formatTokens(todayTokenComposition.cache) }}
              </dd>
            </div>
          </dl>
        </div>
      </div>
    </section>

    <section class="card overflow-hidden">
      <div class="flex items-center justify-between border-b border-primary-100 bg-primary-50/70 px-5 py-4 dark:border-primary-900/40 dark:bg-primary-950/30">
        <div>
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('dashboard.performance') }}</h3>
          <p class="mt-1 text-xs text-primary-700 dark:text-primary-300">RPM · TPM</p>
        </div>
        <span class="flex h-9 w-9 items-center justify-center rounded-full bg-white text-primary-700 dark:bg-dark-800 dark:text-primary-300">
          <Icon name="bolt" size="md" :stroke-width="1.75" />
        </span>
      </div>
      <dl class="divide-y divide-gray-100 dark:divide-dark-700">
        <div class="flex items-end justify-between gap-4 px-5 py-4">
          <dt class="text-sm text-gray-500 dark:text-gray-400">RPM</dt>
          <dd class="font-mono text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">
            {{ formatTokens(stats?.rpm || 0) }}
          </dd>
        </div>
        <div class="flex items-end justify-between gap-4 px-5 py-4">
          <dt class="text-sm text-gray-500 dark:text-gray-400">TPM</dt>
          <dd class="font-mono text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">
            {{ formatTokens(stats?.tpm || 0) }}
          </dd>
        </div>
        <div class="flex items-center justify-between gap-4 px-5 py-4">
          <dt>
            <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('dashboard.avgResponse') }}</p>
            <p class="mt-1 text-xs text-gray-400 dark:text-gray-500">{{ t('dashboard.averageTime') }}</p>
          </dt>
          <dd class="flex items-center gap-2 font-mono text-xl font-semibold text-gray-950 dark:text-white">
            <Icon name="clock" size="sm" :stroke-width="1.75" class="text-gray-400" />
            {{ formatDuration(stats?.average_duration_ms || 0) }}
          </dd>
        </div>
      </dl>
    </section>
  </div>

  <!-- Per-platform usage: a dense, scan-friendly list with dividers. -->
  <section v-if="!isSimple && platformCards.length > 0" class="card overflow-hidden">
    <div class="flex items-center justify-between gap-4 px-5 py-5 md:px-6">
      <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('dashboard.platformBreakdown') }}</h3>
      <span class="rounded-full bg-primary-50 px-3 py-1 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">
        {{ t('dashboard.platformCount', { count: sortedPlatforms.length }) }}
      </span>
    </div>

    <div class="divide-y divide-gray-100 border-t border-gray-100 dark:divide-dark-700 dark:border-dark-700">
      <div v-for="item in platformCards" :key="item.platform" class="px-5 py-5 md:px-6">
        <div class="grid gap-5 lg:grid-cols-[minmax(10rem,0.8fr)_minmax(0,2.2fr)] lg:items-center">
          <div class="min-w-0">
            <div class="flex items-center gap-2">
              <span
                class="h-2.5 w-2.5 shrink-0 rounded-full"
                :class="item.isOther ? 'bg-gray-300 dark:bg-dark-500' : 'bg-primary-400'"
              />
              <p class="truncate text-base font-semibold text-gray-950 dark:text-white">
                {{ item.isOther ? t('dashboard.platformOther') : platformLabel(item.platform) }}
              </p>
            </div>
            <p v-if="hasAnyLimit(item.quota) && !item.isOther" class="mt-2 text-xs text-gray-500 dark:text-gray-400">
              {{ t('dashboard.platformQuota.title') }}
            </p>
          </div>

          <dl class="grid grid-cols-2 gap-x-6 gap-y-4 sm:grid-cols-4">
            <div class="min-w-0">
              <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('common.total') }}</dt>
              <dd class="mt-1 truncate font-mono text-sm font-semibold text-primary-700 dark:text-primary-300" :title="t('dashboard.actual')">
                ${{ formatCost(item.total_actual_cost) }}
              </dd>
            </div>
            <div class="min-w-0">
              <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('dashboard.todayCost') }}</dt>
              <dd class="mt-1 truncate font-mono text-sm font-semibold text-gray-900 dark:text-white">
                ${{ formatCost(item.today_actual_cost) }}
              </dd>
            </div>
            <div class="min-w-0">
              <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('dashboard.requests') }}</dt>
              <dd class="mt-1 truncate font-mono text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ item.total_requests > 0 ? formatNumber(item.total_requests) : '-' }}
              </dd>
            </div>
            <div class="min-w-0">
              <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('dashboard.tokens') }}</dt>
              <dd class="mt-1 truncate font-mono text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ item.total_tokens > 0 ? formatTokens(item.total_tokens) : '-' }}
              </dd>
            </div>
          </dl>
        </div>

        <!-- Quota 区：仅当 quota 配置存在、非 __other__ 且至少有一个窗口配了 limit 时显示 -->
        <div
          v-if="hasAnyLimit(item.quota) && !item.isOther"
          class="mt-5 grid grid-cols-1 gap-4 border-t border-dashed border-gray-200 pt-4 sm:grid-cols-2 xl:grid-cols-3 dark:border-dark-600"
        >
          <template v-for="w in (['daily', 'weekly', 'monthly'] as const)" :key="w">
            <div v-if="quotaVal(item.quota, `${w}_limit_usd`) != null" class="min-w-0 space-y-2">
              <!-- limit=0：完全禁用 -->
              <template v-if="(quotaVal(item.quota, `${w}_limit_usd`) as number) === 0">
                <div class="flex items-center justify-between gap-3 text-xs">
                  <span class="truncate text-gray-600 dark:text-gray-300">{{ t(`dashboard.platformQuota.${w}`) }}</span>
                  <span class="shrink-0 font-mono text-red-500">{{ t('dashboard.platformQuota.disabled') }}</span>
                </div>
                <div class="h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-700">
                  <div class="h-full w-full rounded-full bg-red-500" />
                </div>
              </template>
              <!-- limit>0：正常用量进度条 -->
              <template v-else>
                <div class="flex items-center justify-between gap-3 text-xs">
                  <span class="truncate text-gray-600 dark:text-gray-300">{{ t(`dashboard.platformQuota.${w}`) }}</span>
                  <span class="shrink-0 font-mono text-gray-700 dark:text-gray-200">
                    ${{ formatUsd((quotaVal(item.quota, `${w}_usage_usd`) as number) ?? 0) }} / ${{ formatUsd(quotaVal(item.quota, `${w}_limit_usd`) as number) }}
                  </span>
                </div>
                <div class="h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-700">
                  <div
                    class="h-full rounded-full transition-all"
                    :class="quotaBarClass(calcPercent((quotaVal(item.quota, `${w}_usage_usd`) as number) ?? 0, quotaVal(item.quota, `${w}_limit_usd`) as number))"
                    :style="{ width: calcPercent((quotaVal(item.quota, `${w}_usage_usd`) as number) ?? 0, quotaVal(item.quota, `${w}_limit_usd`) as number) + '%' }"
                  />
                </div>
                <p v-if="quotaVal(item.quota, `${w}_window_resets_at`)" class="text-[10px] text-gray-400">
                  {{ t('dashboard.platformQuota.resetsAt', { time: formatResetTime(quotaVal(item.quota, `${w}_window_resets_at`) as string) }) }}
                </p>
              </template>
            </div>
          </template>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { UserDashboardStats as UserStatsType } from '@/api/usage'
import type { PlatformQuotaItem } from '@/types'

interface FusedPlatformCard {
  platform: string
  total_actual_cost: number
  today_actual_cost: number
  total_requests: number
  total_tokens: number
  isOther?: boolean
  quota?: PlatformQuotaItem
}

const props = defineProps<{
  stats: UserStatsType
  balance: number
  isSimple: boolean
  platformQuotas?: PlatformQuotaItem[] | null
}>()
const { t } = useI18n()

const PLATFORM_LABELS: Record<string, string> = {
  anthropic: 'Claude',
  openai: 'OpenAI',
  gemini: 'Gemini',
  antigravity: 'Antigravity'
}

const platformLabel = (p: string) => PLATFORM_LABELS[p] ?? p

const sortedPlatforms = computed(() => {
  const list = props.stats?.by_platform ?? []
  return [...list].sort((a, b) => b.total_actual_cost - a.total_actual_cost)
})

// 仅用于仪表展示：将今日 Token 拆成输入、输出与两类缓存的合计占比。
const todayTokenComposition = computed(() => {
  const input = Math.max(0, props.stats?.today_input_tokens ?? 0)
  const output = Math.max(0, props.stats?.today_output_tokens ?? 0)
  const cache = Math.max(
    0,
    (props.stats?.today_cache_creation_tokens ?? 0) + (props.stats?.today_cache_read_tokens ?? 0)
  )
  const total = input + output + cache
  const share = (value: number) => (total > 0 ? (value / total) * 100 : 0)

  return {
    input,
    output,
    cache,
    inputShare: share(input),
    outputShare: share(output),
    cacheShare: share(cache)
  }
})

// 处理"各平台之和 < 总值"的差值：后端按平台聚合时过滤了无法归属平台的行
// （group 与 account 都缺 platform）。这里把差值作为"其他"卡片显式展示，
// 避免 Row 1 总值与 Row 3 平台拆分加总对不上、用户困惑。
const OTHER_THRESHOLD = 0.0001
const platformCards = computed<FusedPlatformCard[]>(() => {
  // 建立 by_platform Map
  const byPlat = new Map<string, (typeof sortedPlatforms.value)[number]>()
  for (const item of props.stats?.by_platform ?? []) byPlat.set(item.platform, item)

  // 建立 quota Map
  const byQuota = new Map<string, PlatformQuotaItem>()
  for (const q of props.platformQuotas ?? []) byQuota.set(q.platform, q)

  // union 平台集合。后端 by_platform / quota 接口均不会返回 platform='__other__'，
  // 无需显式排除；__other__ 由下方差值补差逻辑单独追加。
  const platforms = new Set<string>([...byPlat.keys(), ...byQuota.keys()])

  const PLATFORM_ORDER = ['anthropic', 'openai', 'gemini', 'antigravity', 'grok']
  const cards: FusedPlatformCard[] = []

  for (const p of platforms) {
    const stat = byPlat.get(p)
    cards.push({
      platform: p,
      total_actual_cost: stat?.total_actual_cost ?? 0,
      today_actual_cost: stat?.today_actual_cost ?? 0,
      total_requests: stat?.total_requests ?? 0,
      total_tokens: stat?.total_tokens ?? 0,
      quota: byQuota.get(p),
    })
  }

  // 排序：按 PLATFORM_ORDER，未知平台按名称排序
  cards.sort((a, b) => {
    const ai = PLATFORM_ORDER.indexOf(a.platform)
    const bi = PLATFORM_ORDER.indexOf(b.platform)
    if (ai === -1 && bi === -1) return a.platform.localeCompare(b.platform)
    if (ai === -1) return 1
    if (bi === -1) return -1
    return ai - bi
  })

  // __other__ 补差逻辑：只对 by_platform 有 usage 数据的总和计算
  const total = props.stats?.total_actual_cost ?? 0
  const today = props.stats?.today_actual_cost ?? 0
  const sumTotal = cards.reduce((s, c) => s + c.total_actual_cost, 0)
  const sumToday = cards.reduce((s, c) => s + c.today_actual_cost, 0)
  const diffTotal = Math.max(0, total - sumTotal)
  const diffToday = Math.max(0, today - sumToday)

  if (diffTotal > OTHER_THRESHOLD || diffToday > OTHER_THRESHOLD) {
    cards.push({
      platform: '__other__',
      total_actual_cost: diffTotal,
      today_actual_cost: diffToday,
      total_requests: 0,
      total_tokens: 0,
      isOther: true,
    })
  }

  return cards
})

// Quota helpers

type QuotaWindow = 'daily' | 'weekly' | 'monthly'
type QuotaField = `${QuotaWindow}_limit_usd` | `${QuotaWindow}_usage_usd` | `${QuotaWindow}_window_resets_at`

function quotaVal(q: PlatformQuotaItem | undefined, key: QuotaField): PlatformQuotaItem[QuotaField] {
  return q?.[key]
}

function hasAnyLimit(q: PlatformQuotaItem | undefined): boolean {
  if (!q) return false
  return q.daily_limit_usd != null || q.weekly_limit_usd != null || q.monthly_limit_usd != null
}

function calcPercent(usage: number, limit: number): number {
  if (!limit || limit <= 0) return 0
  return Math.min(100, Math.max(0, Math.round((usage / limit) * 100)))
}

function quotaBarClass(p: number): string {
  if (p >= 95) return 'bg-red-500'
  if (p >= 75) return 'bg-amber-500'
  return 'bg-green-500'
}

// 与 formatBalance 一致使用 Intl.NumberFormat 做半偶舍入，避免 toFixed 在不同 JS 引擎
// 下偶发截断而非四舍五入（与后端展示精度不一致）。
const usdFormatter = new Intl.NumberFormat('en-US', {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
})
function formatUsd(n: number): string {
  if (!Number.isFinite(n)) return '0.00'
  return usdFormatter.format(n)
}

function formatResetTime(iso: string | null | undefined): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString(undefined, {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  })
}

const formatBalance = (b: number) =>
  new Intl.NumberFormat('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2
  }).format(b)

const formatNumber = (n: number) => n.toLocaleString()
const formatCost = (c: number) => c.toFixed(4)
const formatTokens = (t: number) => {
  if (t >= 1_000_000) return `${(t / 1_000_000).toFixed(1)}M`
  if (t >= 1000) return `${(t / 1000).toFixed(1)}K`
  return t.toString()
}
const formatDuration = (ms: number) => ms >= 1000 ? `${(ms / 1000).toFixed(2)}s` : `${ms.toFixed(0)}ms`
</script>

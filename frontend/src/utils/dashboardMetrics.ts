import type { TrendDataPoint } from '@/types'

export interface DashboardRangeMetrics {
  requests: number
  tokens: number
  actualCost: number
  minutes: number
  averageRpm: number
  averageTpm: number
}

export interface DashboardSparklineSeries {
  requests: number[]
  actualCost: number[]
  tokens: number[]
}

type DashboardTrendGranularity = 'day' | 'hour'

const MAX_SPARKLINE_POINTS = 48

function localDayStart(value: string): number {
  const timestamp = new Date(`${value}T00:00:00`).getTime()
  return Number.isFinite(timestamp) ? timestamp : Number.NaN
}

function sumFinite(values: number[]): number {
  return values.reduce((total, value) => total + (Number.isFinite(value) ? value : 0), 0)
}

function finiteUsageValue(value: number): number {
  return Number.isFinite(value) ? Math.max(0, value) : 0
}

function localDate(value: string): Date | null {
  const date = new Date(`${value}T00:00:00`)
  return Number.isFinite(date.getTime()) ? date : null
}

function formatLocalDay(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function formatLocalHour(date: Date): string {
  return `${formatLocalDay(date)} ${String(date.getHours()).padStart(2, '0')}:00`
}

function trendKey(value: string, granularity: DashboardTrendGranularity): string | null {
  if (granularity === 'day') {
    const match = value.match(/^(\d{4}-\d{2}-\d{2})/)
    return match?.[1] || null
  }

  const match = value.match(/^(\d{4}-\d{2}-\d{2})[ T](\d{2})/)
  return match ? `${match[1]} ${match[2]}:00` : null
}

function compactSparklineSeries(series: DashboardSparklineSeries): DashboardSparklineSeries {
  if (series.requests.length === 1) {
    return {
      requests: [series.requests[0], series.requests[0]],
      actualCost: [series.actualCost[0], series.actualCost[0]],
      tokens: [series.tokens[0], series.tokens[0]],
    }
  }
  if (series.requests.length <= MAX_SPARKLINE_POINTS) return series

  const bucketSize = Math.ceil(series.requests.length / MAX_SPARKLINE_POINTS)
  const compacted: DashboardSparklineSeries = {
    requests: [],
    actualCost: [],
    tokens: [],
  }

  for (let index = 0; index < series.requests.length; index += bucketSize) {
    compacted.requests.push(sumFinite(series.requests.slice(index, index + bucketSize)))
    compacted.actualCost.push(sumFinite(series.actualCost.slice(index, index + bucketSize)))
    compacted.tokens.push(sumFinite(series.tokens.slice(index, index + bucketSize)))
  }

  return compacted
}

/**
 * Build aligned, zero-filled series for the compact charts in the metric cards.
 * The backend only returns buckets that contain usage, so filling the selected
 * range preserves gaps and still produces a useful line when there is one
 * active bucket in an otherwise quiet period.
 */
export function buildDashboardSparklineSeries(
  trend: TrendDataPoint[],
  startDate: string,
  endDate: string,
  granularity: DashboardTrendGranularity,
  nowMs = Date.now(),
): DashboardSparklineSeries {
  const start = localDate(startDate)
  const end = localDate(endDate)
  const empty: DashboardSparklineSeries = { requests: [], actualCost: [], tokens: [] }
  if (!start || !end || start.getTime() > end.getTime()) return empty

  const byBucket = new Map<string, { requests: number; actualCost: number; tokens: number }>()
  for (const point of trend) {
    const key = trendKey(point.date, granularity)
    if (!key) continue
    const current = byBucket.get(key) || { requests: 0, actualCost: 0, tokens: 0 }
    current.requests += finiteUsageValue(point.requests)
    current.actualCost += finiteUsageValue(point.actual_cost)
    current.tokens += finiteUsageValue(point.total_tokens)
    byBucket.set(key, current)
  }

  const result: DashboardSparklineSeries = { requests: [], actualCost: [], tokens: [] }
  const cursor = new Date(start)
  let lastBucket = new Date(end)

  if (granularity === 'hour') {
    lastBucket.setHours(23, 0, 0, 0)
    const now = new Date(nowMs)
    now.setMinutes(0, 0, 0)
    if (now.getTime() >= cursor.getTime() && now.getTime() < lastBucket.getTime()) {
      lastBucket = now
    }
  }

  while (cursor.getTime() <= lastBucket.getTime()) {
    const key = granularity === 'hour' ? formatLocalHour(cursor) : formatLocalDay(cursor)
    const bucket = byBucket.get(key) || { requests: 0, actualCost: 0, tokens: 0 }
    result.requests.push(bucket.requests)
    result.actualCost.push(bucket.actualCost)
    result.tokens.push(bucket.tokens)

    if (granularity === 'hour') cursor.setHours(cursor.getHours() + 1)
    else cursor.setDate(cursor.getDate() + 1)
  }

  return compactSparklineSeries(result)
}

/**
 * Aggregate the selected dashboard range without mixing it with the lifetime
 * counters returned by /usage/dashboard/stats.
 */
export function aggregateDashboardRange(
  trend: TrendDataPoint[],
  startDate: string,
  endDate: string,
  nowMs = Date.now(),
): DashboardRangeMetrics {
  const requests = sumFinite(trend.map((point) => point.requests))
  const tokens = sumFinite(trend.map((point) => point.total_tokens))
  const actualCost = sumFinite(trend.map((point) => point.actual_cost))

  const startMs = localDayStart(startDate)
  const endStartMs = localDayStart(endDate)
  let endExclusiveMs = Number.NaN
  if (Number.isFinite(endStartMs)) {
    const endExclusive = new Date(endStartMs)
    endExclusive.setDate(endExclusive.getDate() + 1)
    endExclusiveMs = endExclusive.getTime()
  }

  let minutes = 1
  if (Number.isFinite(startMs) && Number.isFinite(endExclusiveMs)) {
    const effectiveEnd = Math.min(Math.max(nowMs, startMs), endExclusiveMs)
    minutes = Math.max(1, (effectiveEnd - startMs) / 60_000)
  }

  return {
    requests,
    tokens,
    actualCost,
    minutes,
    averageRpm: requests / minutes,
    averageTpm: tokens / minutes,
  }
}

import type { TrendDataPoint } from '@/types'

export type DashboardUsagePeriod = 'today' | 'week' | 'month' | 'thirtyDays'

export interface DashboardUsagePeriodQuery {
  startDate: string
  endDate: string
  granularity: 'day' | 'hour'
}

export interface DashboardUsagePoint {
  date: string
  requests: number
  credits: number
  cachedInputTokens: number
  uncachedInputTokens: number
  outputTokens: number
  totalTokens: number
}

function finiteNumber(value: unknown): number {
  const normalized = Number(value ?? 0)
  return Number.isFinite(normalized) ? normalized : 0
}

function formatLocalDate(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

export function resolveDashboardUsagePeriod(
  period: DashboardUsagePeriod,
  now = new Date(),
): DashboardUsagePeriodQuery {
  const end = new Date(now)
  end.setHours(0, 0, 0, 0)
  const start = new Date(end)

  if (period === 'week') {
    const weekday = end.getDay()
    start.setDate(end.getDate() + (weekday === 0 ? -6 : 1 - weekday))
  } else if (period === 'month') {
    start.setDate(1)
  } else if (period === 'thirtyDays') {
    start.setDate(end.getDate() - 29)
  }

  return {
    startDate: formatLocalDate(start),
    endDate: formatLocalDate(end),
    granularity: period === 'today' ? 'hour' : 'day',
  }
}

export function normalizeDashboardTrend(points: readonly TrendDataPoint[]): DashboardUsagePoint[] {
  const buckets = new Map<string, DashboardUsagePoint>()

  for (const point of points) {
    const key = String(point.date ?? '').trim()
    if (!key) continue

    const cachedInputTokens = finiteNumber(point.cache_read_tokens)
    const uncachedInputTokens = finiteNumber(point.input_tokens)
      + finiteNumber(point.cache_creation_tokens)
    const outputTokens = finiteNumber(point.output_tokens)
    const existing = buckets.get(key)
    const normalized: DashboardUsagePoint = {
      date: key,
      requests: finiteNumber(point.requests),
      credits: finiteNumber(point.actual_cost),
      cachedInputTokens,
      uncachedInputTokens,
      outputTokens,
      totalTokens: cachedInputTokens + uncachedInputTokens + outputTokens,
    }

    if (!existing) {
      buckets.set(key, normalized)
      continue
    }

    existing.requests += normalized.requests
    existing.credits += normalized.credits
    existing.cachedInputTokens += normalized.cachedInputTokens
    existing.uncachedInputTokens += normalized.uncachedInputTokens
    existing.outputTokens += normalized.outputTokens
    existing.totalTokens += normalized.totalTokens
  }

  return [...buckets.values()].sort((left, right) => {
    const leftTimestamp = Date.parse(left.date)
    const rightTimestamp = Date.parse(right.date)
    if (Number.isFinite(leftTimestamp) && Number.isFinite(rightTimestamp)) {
      return leftTimestamp - rightTimestamp
    }
    return left.date.localeCompare(right.date)
  })
}

function canonicalBucketKey(value: string, period: DashboardUsagePeriod): string {
  const match = value.match(/^(\d{4}-\d{2}-\d{2})(?:[ T](\d{1,2}))?/)
  if (!match) return value
  if (period !== 'today') return match[1]
  const hour = String(Number(match[2] ?? 0)).padStart(2, '0')
  return `${match[1]} ${hour}:00`
}

function emptyUsagePoint(date: string): DashboardUsagePoint {
  return {
    date,
    requests: 0,
    credits: 0,
    cachedInputTokens: 0,
    uncachedInputTokens: 0,
    outputTokens: 0,
    totalTokens: 0,
  }
}

export function buildDashboardUsageSeries(
  points: readonly TrendDataPoint[],
  period: DashboardUsagePeriod,
  now = new Date(),
): DashboardUsagePoint[] {
  const normalized = normalizeDashboardTrend(points)
  const existing = new Map<string, DashboardUsagePoint>()
  for (const point of normalized) {
    existing.set(canonicalBucketKey(point.date, period), point)
  }

  const range = resolveDashboardUsagePeriod(period, now)
  if (period === 'today') {
    const currentHour = Math.max(0, Math.min(23, now.getHours()))
    return Array.from({ length: currentHour + 1 }, (_, hour) => {
      const key = `${range.startDate} ${String(hour).padStart(2, '0')}:00`
      return existing.get(key) ?? emptyUsagePoint(key)
    })
  }

  const start = new Date(`${range.startDate}T00:00:00`)
  const end = new Date(`${range.endDate}T00:00:00`)
  const result: DashboardUsagePoint[] = []
  for (const cursor = new Date(start); cursor <= end; cursor.setDate(cursor.getDate() + 1)) {
    const key = formatLocalDate(cursor)
    result.push(existing.get(key) ?? emptyUsagePoint(key))
  }
  return result
}

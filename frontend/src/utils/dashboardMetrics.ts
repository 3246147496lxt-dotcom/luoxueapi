import type { TrendDataPoint } from '@/types'

export interface DashboardRangeMetrics {
  requests: number
  tokens: number
  actualCost: number
  minutes: number
  averageRpm: number
  averageTpm: number
}

function localDayStart(value: string): number {
  const timestamp = new Date(`${value}T00:00:00`).getTime()
  return Number.isFinite(timestamp) ? timestamp : Number.NaN
}

function sumFinite(values: number[]): number {
  return values.reduce((total, value) => total + (Number.isFinite(value) ? value : 0), 0)
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

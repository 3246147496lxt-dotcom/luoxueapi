import { describe, expect, it } from 'vitest'

import type { TrendDataPoint } from '@/types'
import { aggregateDashboardRange } from '../dashboardMetrics'

function trendPoint(
  date: string,
  requests: number,
  totalTokens: number,
  actualCost: number,
): TrendDataPoint {
  return {
    date,
    requests,
    input_tokens: 0,
    output_tokens: 0,
    cache_creation_tokens: 0,
    cache_read_tokens: 0,
    total_tokens: totalTokens,
    cost: actualCost,
    actual_cost: actualCost,
  }
}

describe('aggregateDashboardRange', () => {
  it('aggregates requests, tokens, and actual cost across the selected range', () => {
    const result = aggregateDashboardRange(
      [
        trendPoint('2026-07-10', 120, 12_000, 1.25),
        trendPoint('2026-07-11', 80, 8_000, 2.75),
      ],
      '2026-07-10',
      '2026-07-11',
      new Date('2026-07-12T00:00:00').getTime(),
    )

    expect(result).toEqual({
      requests: 200,
      tokens: 20_000,
      actualCost: 4,
      minutes: 2 * 24 * 60,
      averageRpm: 200 / (2 * 24 * 60),
      averageTpm: 20_000 / (2 * 24 * 60),
    })
  })

  it('includes the end date while capping the window at now', () => {
    const trend = [trendPoint('2026-07-10', 10, 100, 1)]

    const duringEndDate = aggregateDashboardRange(
      trend,
      '2026-07-10',
      '2026-07-11',
      new Date('2026-07-11T12:00:00').getTime(),
    )
    const afterEndDate = aggregateDashboardRange(
      trend,
      '2026-07-10',
      '2026-07-11',
      new Date('2026-07-20T00:00:00').getTime(),
    )

    expect(duringEndDate.minutes).toBe(36 * 60)
    expect(afterEndDate.minutes).toBe(48 * 60)
  })

  it('uses at least one minute for a zero-length window', () => {
    const result = aggregateDashboardRange(
      [trendPoint('2026-07-10', 12, 120, 1.2)],
      '2026-07-10',
      '2026-07-10',
      new Date('2026-07-10T00:00:00').getTime(),
    )

    expect(result.minutes).toBe(1)
    expect(result.averageRpm).toBe(12)
    expect(result.averageTpm).toBe(120)
  })
})

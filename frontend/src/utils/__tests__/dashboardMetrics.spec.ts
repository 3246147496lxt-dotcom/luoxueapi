import { describe, expect, it } from 'vitest'

import type { TrendDataPoint } from '@/types'
import { aggregateDashboardRange, buildDashboardSparklineSeries } from '../dashboardMetrics'

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

describe('buildDashboardSparklineSeries', () => {
  it('fills missing daily buckets so a quiet range still has a meaningful shape', () => {
    const result = buildDashboardSparklineSeries(
      [
        trendPoint('2026-07-10', 2, 20, 0.2),
        trendPoint('2026-07-12', 4, 40, 0.4),
      ],
      '2026-07-10',
      '2026-07-12',
      'day',
    )

    expect(result).toEqual({
      requests: [2, 0, 4],
      actualCost: [0.2, 0, 0.4],
      tokens: [20, 0, 40],
    })
  })

  it('fills hourly buckets through the current hour and normalizes invalid values', () => {
    const invalidPoint = trendPoint('2026-07-10 01:00', Number.NaN, Number.POSITIVE_INFINITY, -1)
    const validPoint = trendPoint('2026-07-10 02:00', 3, 30, 0.3)

    const result = buildDashboardSparklineSeries(
      [invalidPoint, validPoint],
      '2026-07-10',
      '2026-07-10',
      'hour',
      new Date('2026-07-10T02:30:00').getTime(),
    )

    expect(result).toEqual({
      requests: [0, 0, 3],
      actualCost: [0, 0, 0.3],
      tokens: [0, 0, 30],
    })
  })

  it('returns empty series for an invalid selected range', () => {
    expect(buildDashboardSparklineSeries([], '2026-07-12', '2026-07-10', 'day')).toEqual({
      requests: [],
      actualCost: [],
      tokens: [],
    })
  })

  it('duplicates a single selected bucket so a non-zero value remains visible as a line', () => {
    const result = buildDashboardSparklineSeries(
      [trendPoint('2026-07-10', 5, 50, 0.5)],
      '2026-07-10',
      '2026-07-10',
      'day',
    )

    expect(result).toEqual({
      requests: [5, 5],
      actualCost: [0.5, 0.5],
      tokens: [50, 50],
    })
  })
})

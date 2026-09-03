import { describe, expect, it } from 'vitest'
import type { TrendDataPoint } from '@/types'
import {
  buildDashboardUsageSeries,
  normalizeDashboardTrend,
  resolveDashboardUsagePeriod,
} from '../dashboardUsage'

describe('dashboardUsage', () => {
  it('resolves the four user-facing periods with local calendar arithmetic', () => {
    const now = new Date(2026, 7, 16, 12, 30)

    expect(resolveDashboardUsagePeriod('today', now)).toEqual({
      startDate: '2026-08-16',
      endDate: '2026-08-16',
      granularity: 'hour',
    })
    expect(resolveDashboardUsagePeriod('week', now).startDate).toBe('2026-08-10')
    expect(resolveDashboardUsagePeriod('month', now).startDate).toBe('2026-08-01')
    expect(resolveDashboardUsagePeriod('thirtyDays', now).startDate).toBe('2026-07-18')
  })

  it('sorts and merges buckets while keeping token tooltip parts equal to the total bar', () => {
    const point = (date: string, requests: number): TrendDataPoint => ({
      date,
      requests,
      input_tokens: 100,
      output_tokens: 25,
      cache_creation_tokens: 10,
      cache_read_tokens: 15,
      total_tokens: 150,
      cost: 1,
      actual_cost: 0.5,
    })

    const result = normalizeDashboardTrend([
      point('2026-08-16 09:00:00', 2),
      point('2026-08-16 08:00:00', 1),
      point('2026-08-16 09:00:00', 3),
    ])

    expect(result.map(item => item.date)).toEqual([
      '2026-08-16 08:00:00',
      '2026-08-16 09:00:00',
    ])
    expect(result[1]).toMatchObject({
      requests: 5,
      credits: 1,
      cachedInputTokens: 30,
      uncachedInputTokens: 220,
      outputTokens: 50,
      totalTokens: 300,
    })
  })

  it('fills missing elapsed time buckets so axes and columns remain stable', () => {
    const now = new Date(2026, 7, 16, 3, 30)
    const points: TrendDataPoint[] = [{
      date: '2026-08-16 02:00',
      requests: 2,
      input_tokens: 10,
      output_tokens: 5,
      cache_creation_tokens: 0,
      cache_read_tokens: 0,
      total_tokens: 15,
      cost: 1,
      actual_cost: 0.5,
    }]

    const result = buildDashboardUsageSeries(points, 'today', now)
    expect(result.map(point => point.date)).toEqual([
      '2026-08-16 00:00',
      '2026-08-16 01:00',
      '2026-08-16 02:00',
      '2026-08-16 03:00',
    ])
    expect(result.map(point => point.requests)).toEqual([0, 0, 2, 0])
  })
})

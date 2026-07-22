import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn()
}))

vi.mock('@/api/client', () => ({
  apiClient: { get, post },
  buildGatewayUrl: vi.fn((path: string) => path)
}))

import {
  getAccountPool,
  getProxyHealth,
  getProxyHealthDetail,
  listAlertEvents,
  reprobeProxyHealth
} from '@/api/admin/ops'

describe('ops resource health API', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
  })

  it('loads the bounded, read-only account pool through the dedicated ops endpoint', async () => {
    const response = {
      summary: { total_accounts: 3 },
      platforms: [],
      groups: [],
      anomalies: [],
      actionable_anomalies: [],
      auto_recovering_anomalies: [],
      group_counts_additive: false,
      collected_at: '2026-01-02T12:00:00Z'
    }
    get.mockResolvedValue({ data: response })
    const controller = new AbortController()

    await expect(getAccountPool('openai', 42, 20, { signal: controller.signal })).resolves.toEqual(response)
    expect(get).toHaveBeenCalledWith('/admin/ops/account-pool', {
      params: { platform: 'openai', group_id: 42, limit: 20 },
      signal: controller.signal
    })
  })

  it('uses safe proxy health list and detail endpoints without proxy pagination', async () => {
    const list = { summary: { total: 1 }, items: [], generated_at: '2026-01-02T12:00:00Z', data_status: 'complete' }
    const detail = { id: 7, name: 'Exit 7', health: 'healthy' }
    get.mockResolvedValueOnce({ data: list }).mockResolvedValueOnce({ data: detail })
    const controller = new AbortController()

    await expect(getProxyHealth({ health: 'degraded', protocol: 'socks5' }, { signal: controller.signal })).resolves.toEqual(list)
    await expect(getProxyHealthDetail(7, { signal: controller.signal })).resolves.toEqual(detail)

    expect(get).toHaveBeenNthCalledWith(1, '/admin/ops/proxy-health', {
      params: { health: 'degraded', protocol: 'socks5' },
      signal: controller.signal
    })
    expect(get).toHaveBeenNthCalledWith(2, '/admin/ops/proxy-health/7', {
      signal: controller.signal
    })
    expect(get).not.toHaveBeenCalledWith('/admin/proxies', expect.anything())
  })

  it('uses the dedicated non-mutating health reprobe action', async () => {
    const result = { id: 7, name: 'Exit 7', health: 'healthy', connectivity_stale: false, quality_stale: false }
    post.mockResolvedValue({ data: result })

    await expect(reprobeProxyHealth(7)).resolves.toEqual(result)
    expect(post).toHaveBeenCalledWith('/admin/ops/proxy-health/7/reprobe')
  })

  it('supports independently abortable unresolved alert loading', async () => {
    const response = [{ id: 9, severity: 'P0', status: 'firing' }]
    get.mockResolvedValue({ data: response })
    const controller = new AbortController()

    await expect(
      listAlertEvents(
        { limit: 100, status: 'firing', platform: 'openai', group_id: 42 },
        { signal: controller.signal }
      )
    ).resolves.toEqual(response)
    expect(get).toHaveBeenCalledWith('/admin/ops/alert-events', {
      params: { limit: 100, status: 'firing', platform: 'openai', group_id: 42 },
      signal: controller.signal
    })
  })
})

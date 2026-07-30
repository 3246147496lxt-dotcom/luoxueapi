import { beforeEach, describe, expect, it, vi } from 'vitest'

const client = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  delete: vi.fn(),
}))

vi.mock('../client', () => ({ apiClient: client }))

import {
  approveQuotaViewerPairing,
  getQuotaViewerPairingPreview,
  listQuotaViewerDevices,
  revokeQuotaViewerDevice,
} from '../quotaViewer'

describe('quota viewer authorization API', () => {
  beforeEach(() => {
    client.get.mockReset()
    client.post.mockReset()
    client.delete.mockReset()
  })

  it('encodes the one-time code in the dedicated read-only path', async () => {
    const preview = { user_code: 'ABCD-EFGH', scope: 'quota:read', read_only: true }
    client.get.mockResolvedValue({ data: preview })

    await expect(getQuotaViewerPairingPreview('ABCD/EFGH')).resolves.toBe(preview)
    expect(client.get).toHaveBeenCalledWith('/quota/authorizations/ABCD%2FEFGH')
  })

  it('approves without sending website credentials in the body', async () => {
    const device = { id: 'device-id', scope: 'quota:read' }
    client.post.mockResolvedValue({ data: device })

    await expect(approveQuotaViewerPairing('ABCD-EFGH')).resolves.toBe(device)
    expect(client.post).toHaveBeenCalledWith(
      '/quota/authorizations/ABCD-EFGH/approve',
    )
  })

  it('lists and revokes quota viewer devices through user-scoped endpoints', async () => {
    const device = {
      id: 'device/id',
      client_id: 'luoxue-quota-viewer',
      scope: 'quota:read',
      status: 'active',
    }
    client.get.mockResolvedValue({ data: [device] })
    client.delete.mockResolvedValue({ data: { ...device, status: 'revoked' } })

    await expect(listQuotaViewerDevices()).resolves.toEqual([device])
    await expect(revokeQuotaViewerDevice('device/id')).resolves.toMatchObject({
      status: 'revoked',
    })

    expect(client.get).toHaveBeenCalledWith('/quota/devices')
    expect(client.delete).toHaveBeenCalledWith('/quota/devices/device%2Fid')
  })
})

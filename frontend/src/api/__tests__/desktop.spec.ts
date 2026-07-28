import { beforeEach, describe, expect, it, vi } from 'vitest'

const client = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  patch: vi.fn(),
  delete: vi.fn(),
}))

vi.mock('../client', () => ({ apiClient: client }))

import {
  approvePairing,
  getPairingPreview,
  listDevices,
  renameDevice,
  revokeDevice,
} from '../desktop'

describe('desktop pairing API', () => {
  beforeEach(() => {
    client.get.mockReset()
    client.post.mockReset()
    client.patch.mockReset()
    client.delete.mockReset()
  })

  it('encodes the user code in the preview path', async () => {
    const preview = { user_code: 'ABCD-EFGH', requested_scopes: [] }
    client.get.mockResolvedValue({ data: preview })

    await expect(getPairingPreview('ABCD/EFGH')).resolves.toBe(preview)
    expect(client.get).toHaveBeenCalledWith('/desktop/authorizations/ABCD%2FEFGH')
  })

  it('approves through the dedicated authorization endpoint', async () => {
    const device = { id: 'device-id', name: 'Mac' }
    client.post.mockResolvedValue({ data: device })

    await expect(approvePairing('ABCD-EFGH')).resolves.toBe(device)
    expect(client.post).toHaveBeenCalledWith('/desktop/authorizations/ABCD-EFGH/approve')
  })

  it('lists, renames, and revokes devices through user-scoped endpoints', async () => {
    const device = { id: 'device/id', name: 'Studio Mac' }
    client.get.mockResolvedValue({ data: [device] })
    client.patch.mockResolvedValue({ data: device })
    client.delete.mockResolvedValue({ data: { ...device, status: 'revoked' } })

    await expect(listDevices()).resolves.toEqual([device])
    await expect(renameDevice('device/id', 'Studio Mac')).resolves.toBe(device)
    await expect(revokeDevice('device/id')).resolves.toMatchObject({ status: 'revoked' })

    expect(client.get).toHaveBeenCalledWith('/desktop/devices')
    expect(client.patch).toHaveBeenCalledWith('/desktop/devices/device%2Fid', { name: 'Studio Mac' })
    expect(client.delete).toHaveBeenCalledWith('/desktop/devices/device%2Fid')
  })
})

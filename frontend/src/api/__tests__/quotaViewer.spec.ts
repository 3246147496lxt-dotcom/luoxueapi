import { beforeEach, describe, expect, it, vi } from 'vitest'

const client = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  delete: vi.fn(),
}))

vi.mock('../client', () => ({
  apiClient: client,
  buildApiUrl: (path: string) => `/api/v1${path.startsWith('/api/v1') ? path.slice(7) : path}`,
}))

import {
  approveQuotaViewerPairing,
  getQuotaViewerPairingPreview,
  issueQuotaViewerInstallerDownload,
  listQuotaViewerDevices,
  resolveQuotaViewerInstallerDownloadURL,
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

  it('issues a login-gated one-time installer path without sending membership data', async () => {
    const release = {
      platform: 'macos',
      version: '2.0.0-rc.6',
      filename: 'Luoxue-Quota-Viewer_2.0.0-rc.6_macOS-universal-UNNOTARIZED.dmg',
      sha256: '63c7e3890ef45e882a99e609c73f3fa5e8452cc708bc4e1ed6f0278bdc519f92',
      size: 10_655_545,
      architecture: 'universal',
      signing_status: 'unsigned-unnotarized',
      expires_in: 60,
      download_path: `/api/v1/quota/releases/macos/latest/download?code=${'a'.repeat(43)}`,
    }
    client.post.mockResolvedValue({ data: release })

    await expect(issueQuotaViewerInstallerDownload('macos')).resolves.toEqual(release)
    expect(client.post).toHaveBeenCalledWith('/quota/releases/macos/latest/download')
  })

  it('accepts only the expected one-time API path and code', () => {
    const code = 'a'.repeat(43)
    expect(resolveQuotaViewerInstallerDownloadURL(
      'windows',
      `/api/v1/quota/releases/windows/latest/download?code=${code}`,
    )).toBe(`${window.location.origin}/api/v1/quota/releases/windows/latest/download?code=${code}`)

    expect(() => resolveQuotaViewerInstallerDownloadURL(
      'windows',
      `https://github.com/example/releases/download/file.exe?code=${code}`,
    )).toThrow('Unexpected quota viewer download path')
  })
})

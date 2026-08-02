import { apiClient, buildApiUrl } from './client'

export interface QuotaViewerPairingPreview {
  user_code: string
  client_id: string
  scope: string
  device_name: string
  platform: string
  architecture: string
  os_version?: string
  app_version?: string
  read_only: boolean
  expires_at: string
}

export interface QuotaViewerDevice {
  id: string
  client_id: string
  scope: string
  name: string
  platform: string
  architecture: string
  os_version?: string
  app_version?: string
  status: string
  approved_at?: string
  activated_at?: string
  last_seen_at?: string
}

export type QuotaViewerPlatform = 'macos' | 'windows'

export interface QuotaViewerInstallerRelease {
  platform: QuotaViewerPlatform
  version: string
  filename: string
  sha256: string
  size: number
  architecture: string
  signing_status: string
  download_path: string
  expires_in: number
}

export const QUOTA_VIEWER_RELEASES = Object.freeze({
  macos: {
    platform: 'macos',
    version: '2.0.0-rc.6',
    filename: 'Luoxue-Quota-Viewer_2.0.0-rc.6_macOS-universal-UNNOTARIZED.dmg',
    sha256: '63c7e3890ef45e882a99e609c73f3fa5e8452cc708bc4e1ed6f0278bdc519f92',
    sizeBytes: 10_655_545,
    architecture: 'universal',
    signingStatus: 'unsigned-unnotarized',
  },
  windows: {
    platform: 'windows',
    version: '2.0.0-rc.5',
    filename: 'Luoxue-Quota-Viewer_2.0.0-rc.5_windows-x64_NSIS-UNSIGNED.exe',
    sha256: '2f1a0b37f3720d02564d306bc5393e71edfbc2ea5d788b89f87feb9a2b4a27d1',
    sizeBytes: 5_110_179,
    architecture: 'x64',
    signingStatus: 'unsigned',
  },
} as const satisfies Record<QuotaViewerPlatform, {
  platform: QuotaViewerPlatform
  version: string
  filename: string
  sha256: string
  sizeBytes: number
  architecture: string
  signingStatus: string
}>)

export function isQuotaViewerPlatform(value: unknown): value is QuotaViewerPlatform {
  return value === 'macos' || value === 'windows'
}

export async function issueQuotaViewerInstallerDownload(
  platform: QuotaViewerPlatform,
): Promise<QuotaViewerInstallerRelease> {
  const { data } = await apiClient.post<QuotaViewerInstallerRelease>(
    `/quota/releases/${platform}/latest/download`,
  )
  return data
}

export function resolveQuotaViewerInstallerDownloadURL(
  platform: QuotaViewerPlatform,
  downloadPath: string,
): string {
  const base = typeof window === 'undefined' ? 'http://localhost' : window.location.origin
  const resolved = new URL(buildApiUrl(downloadPath), base)
  const expected = new URL(buildApiUrl(`/quota/releases/${platform}/latest/download`), base)

  if (resolved.origin !== expected.origin || resolved.pathname !== expected.pathname) {
    throw new Error('Unexpected quota viewer download path')
  }
  if (resolved.hash || resolved.searchParams.getAll('code').length !== 1) {
    throw new Error('Invalid quota viewer download code')
  }
  const code = resolved.searchParams.get('code')?.trim() || ''
  if (!/^[A-Za-z0-9_-]{43}$/.test(code)) {
    throw new Error('Invalid quota viewer download code')
  }
  for (const key of resolved.searchParams.keys()) {
    if (key !== 'code') throw new Error('Unexpected quota viewer download query')
  }
  return resolved.toString()
}

function authorizationPath(userCode: string): string {
  return `/quota/authorizations/${encodeURIComponent(userCode)}`
}

export async function getQuotaViewerPairingPreview(
  userCode: string,
): Promise<QuotaViewerPairingPreview> {
  const { data } = await apiClient.get<QuotaViewerPairingPreview>(
    authorizationPath(userCode),
  )
  return data
}

export async function approveQuotaViewerPairing(
  userCode: string,
): Promise<QuotaViewerDevice> {
  const { data } = await apiClient.post<QuotaViewerDevice>(
    `${authorizationPath(userCode)}/approve`,
  )
  return data
}

export async function listQuotaViewerDevices(): Promise<QuotaViewerDevice[]> {
  const { data } = await apiClient.get<QuotaViewerDevice[]>('/quota/devices')
  return data
}

export async function revokeQuotaViewerDevice(
  deviceId: string,
): Promise<QuotaViewerDevice> {
  const { data } = await apiClient.delete<QuotaViewerDevice>(
    `/quota/devices/${encodeURIComponent(deviceId)}`,
  )
  return data
}

export const quotaViewerAPI = {
  issueInstallerDownload: issueQuotaViewerInstallerDownload,
  getPairingPreview: getQuotaViewerPairingPreview,
  approvePairing: approveQuotaViewerPairing,
  listDevices: listQuotaViewerDevices,
  revokeDevice: revokeQuotaViewerDevice,
}

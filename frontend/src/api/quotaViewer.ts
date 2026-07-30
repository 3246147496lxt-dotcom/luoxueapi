import { apiClient } from './client'

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
  getPairingPreview: getQuotaViewerPairingPreview,
  approvePairing: approveQuotaViewerPairing,
  listDevices: listQuotaViewerDevices,
  revokeDevice: revokeQuotaViewerDevice,
}

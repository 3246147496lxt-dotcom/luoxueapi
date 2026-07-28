import { apiClient } from './client'

export interface DesktopPairingPreview {
  user_code: string
  device_name: string
  platform: string
  architecture: string
  os_version?: string
  app_version?: string
  requested_scopes: string[]
  expires_at: string
}

export interface DesktopDevice {
  id: string
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

function pairingPath(userCode: string): string {
  return `/desktop/authorizations/${encodeURIComponent(userCode)}`
}

export async function getPairingPreview(userCode: string): Promise<DesktopPairingPreview> {
  const { data } = await apiClient.get<DesktopPairingPreview>(pairingPath(userCode))
  return data
}

export async function approvePairing(userCode: string): Promise<DesktopDevice> {
  const { data } = await apiClient.post<DesktopDevice>(`${pairingPath(userCode)}/approve`)
  return data
}

export async function listDevices(): Promise<DesktopDevice[]> {
  const { data } = await apiClient.get<DesktopDevice[]>('/desktop/devices')
  return data
}

export async function renameDevice(deviceId: string, name: string): Promise<DesktopDevice> {
  const path = `/desktop/devices/${encodeURIComponent(deviceId)}`
  const { data } = await apiClient.patch<DesktopDevice>(path, { name })
  return data
}

export async function revokeDevice(deviceId: string): Promise<DesktopDevice> {
  const path = `/desktop/devices/${encodeURIComponent(deviceId)}`
  const { data } = await apiClient.delete<DesktopDevice>(path)
  return data
}

export const desktopAPI = {
  getPairingPreview,
  approvePairing,
  listDevices,
  renameDevice,
  revokeDevice,
}

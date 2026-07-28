import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'

export interface DesktopDiagnosticMetadata {
  id: string
  device_id: string
  user_id: number
  app_version: string
  platform: string
  architecture: string
  os_version: string
  gateway_status: 'running' | 'stopped' | 'error'
  codex_config_status: 'managed' | 'unmanaged' | 'missing' | 'error'
  request_sample_count: number
  request_error_count: number
  created_at: string
  expires_at: string
}

export interface DesktopDiagnosticUpload {
  app_version: string
  platform: string
  architecture: string
  os_version: string
  gateway: {
    status: 'running' | 'stopped' | 'error'
    port: number
    takeover_enabled: boolean
  }
  codex: {
    config_status: 'managed' | 'unmanaged' | 'missing' | 'error'
    installations: Array<{
      kind: string
      installed: boolean
      version: string
    }>
  }
  route: {
    group_id: number
    model: string
    available_route_count: number
  }
  requests: {
    sample_count: number
    success_count: number
    error_count: number
    average_duration_ms: number
    average_first_token_ms: number
    recent: Array<{
      occurred_at: string
      model: string
      status_code: number
      duration_ms: number
      first_token_ms: number
      request_id: string
    }>
  }
}

export interface DesktopDiagnosticDetail extends DesktopDiagnosticMetadata {
  diagnostic: DesktopDiagnosticUpload
}

export type DesktopDiagnosticListResponse = PaginatedResponse<DesktopDiagnosticMetadata>

export async function list(params: {
  page?: number
  page_size?: number
} = {}): Promise<DesktopDiagnosticListResponse> {
  const { data } = await apiClient.get('/admin/desktop/diagnostics', { params })
  return data
}

export async function get(id: string): Promise<DesktopDiagnosticDetail> {
  const { data } = await apiClient.get(`/admin/desktop/diagnostics/${encodeURIComponent(id)}`)
  return data
}

export async function download(id: string): Promise<Blob> {
  const { data } = await apiClient.get(
    `/admin/desktop/diagnostics/${encodeURIComponent(id)}/download`,
    { responseType: 'blob' },
  )
  return data
}

export const desktopDiagnosticsAPI = { list, get, download }

export default desktopDiagnosticsAPI

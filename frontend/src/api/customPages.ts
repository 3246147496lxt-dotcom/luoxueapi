import { apiClient } from './client'
import type { EmbeddedPageContext } from '@/utils/embedded-url'

export type CustomPageLaunchRequest = Partial<EmbeddedPageContext>

export interface CustomPageLaunchResponse {
  launch_url: string
  auth_mode: 'exchange_code'
  expires_in: number
}

export async function requestCustomPageLaunch(
  menuItemId: string,
  context: CustomPageLaunchRequest,
): Promise<CustomPageLaunchResponse> {
  const payload: CustomPageLaunchRequest = {
    ...(context.theme ? { theme: context.theme } : {}),
    ...(context.lang ? { lang: context.lang } : {}),
    ...(context.ui_mode ? { ui_mode: context.ui_mode } : {}),
    ...(context.src_host ? { src_host: context.src_host } : {}),
  }
  const { data } = await apiClient.post<CustomPageLaunchResponse>(
    `/user/custom-pages/${encodeURIComponent(menuItemId)}/launch`,
    payload,
  )
  return data
}

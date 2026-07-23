import axios from 'axios'
import type { ApiResponse } from '@/types'
import { getAPIBaseURL } from '@/api/url'
import {
  authSession,
  type AuthSessionRefreshResult,
  type AuthSessionSnapshot,
} from './authSession'

function unwrapRefreshResponse(value: unknown): AuthSessionRefreshResult {
  const candidate = value as ApiResponse<AuthSessionRefreshResult> | AuthSessionRefreshResult
  const data = candidate && typeof candidate === 'object' && 'code' in candidate
    ? candidate.code === 0
      ? candidate.data
      : null
    : candidate

  if (!data || typeof data.access_token !== 'string' || !data.access_token.trim()) {
    throw new Error('Token refresh failed')
  }
  return data
}

export async function requestAuthSessionRefresh(
  refreshToken: string,
): Promise<AuthSessionRefreshResult> {
  const response = await axios.post(
    `${getAPIBaseURL()}/auth/refresh`,
    { refresh_token: refreshToken },
    {
      headers: { 'Content-Type': 'application/json' },
      timeout: 30000,
      withCredentials: true,
    },
  )
  return unwrapRefreshResponse(response.data)
}

export function refreshAuthSession(): Promise<AuthSessionSnapshot> {
  return authSession.refresh(requestAuthSessionRefresh)
}

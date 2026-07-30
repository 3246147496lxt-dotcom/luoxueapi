/**
 * Axios HTTP Client Configuration
 * Base client with interceptors for authentication, token refresh, and error handling
 */

import axios, { AxiosInstance, AxiosError, InternalAxiosRequestConfig, AxiosResponse } from 'axios'
import type { ApiResponse } from '@/types'
import { getLocale } from '@/i18n'
import {
  ADMIN_UI_REQUEST_HEADER,
  USER_UI_REQUEST_HEADER,
  shouldMarkAdminUIRequest,
  shouldMarkUserUIRequest,
} from './adminUIRequest'
import { getAPIBaseURL } from './url'
import { AuthSessionChangedError, authSession } from '@/auth/authSession'
import { refreshAuthSession } from '@/auth/authRefresh'
import { sessionExpiredLoginURL } from '@/auth/quotaViewerAuthorizationRedirect'
export { buildApiUrl, buildGatewayUrl } from './url'

type AuthenticatedRequestConfig = InternalAxiosRequestConfig & {
  _retry?: boolean
  _authSessionAccessToken?: string | null
  _authSessionPrincipalId?: number | string | null
  _authSessionGeneration?: string | null
  _authRetryExpectedAccessToken?: string
  _authRetryExpectedPrincipalId?: number | string | null
  _authRetryExpectedGeneration?: string | null
}

// ==================== Axios Instance Configuration ====================

export const apiClient: AxiosInstance = axios.create({
  baseURL: getAPIBaseURL(),
  withCredentials: true,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

function authSessionChangedResponse() {
  return {
    status: 409,
    code: 'AUTH_SESSION_CHANGED',
    message: 'Authentication session changed. Please retry the request.'
  }
}

function normalizeAccessToken(value: unknown): string | null {
  return typeof value === 'string' && value.trim() ? value.trim() : null
}

function getSentBearerToken(headers: unknown): string | null {
  if (!headers || typeof headers !== 'object') return null

  const candidate = headers as {
    get?: (name: string) => unknown
    Authorization?: unknown
    authorization?: unknown
  }
  let authorization: unknown
  try {
    authorization = typeof candidate.get === 'function'
      ? candidate.get('Authorization')
      : candidate.Authorization ?? candidate.authorization
  } catch {
    return null
  }

  const values = Array.isArray(authorization) ? authorization : [authorization]
  for (const value of values) {
    if (typeof value !== 'string') continue
    const match = /^Bearer\s+(.+?)\s*$/i.exec(value.trim())
    const token = normalizeAccessToken(match?.[1])
    if (token) return token
  }
  return null
}

function shouldClearSessionAfterRefreshFailure(error: unknown): boolean {
  if (axios.isAxiosError(error)) {
    const status = error.response?.status
    // Network failures, rate limiting, and server outages are transient. Keep
    // the atomically stored token pair so a later request can retry refresh.
    return status === 400 || status === 401 || status === 403
  }

  const status = (error as { status?: number } | null)?.status
  if (typeof status === 'number') return status === 400 || status === 401 || status === 403

  // A syntactically successful but invalid refresh payload cannot be retried
  // safely with the same token family.
  return true
}

// ==================== Request Interceptor ====================

// Get user's timezone
const getUserTimezone = (): string => {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone
  } catch {
    return 'UTC'
  }
}

apiClient.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    // Attach the current token from the single in-memory session authority.
    const currentSession = authSession.getSnapshot()
    const currentGeneration = authSession.getGeneration()
    const token = currentSession.accessToken
    const principalId = currentSession.user?.id ?? null
    const authenticatedConfig = config as AuthenticatedRequestConfig
    if (
      (
        Object.prototype.hasOwnProperty.call(authenticatedConfig, '_authRetryExpectedAccessToken')
        && authenticatedConfig._authRetryExpectedAccessToken !== token
      )
      || (
        Object.prototype.hasOwnProperty.call(authenticatedConfig, '_authRetryExpectedPrincipalId')
        && authenticatedConfig._authRetryExpectedPrincipalId !== principalId
      )
      || (
        Object.prototype.hasOwnProperty.call(authenticatedConfig, '_authRetryExpectedGeneration')
        && authenticatedConfig._authRetryExpectedGeneration !== currentGeneration
      )
    ) {
      throw new AuthSessionChangedError()
    }
    // Preserve the dispatch-time identity, including an explicit null. The
    // response interceptor uses it to reject stale 401s after login/logout or
    // token rotation instead of replaying a write under a different session.
    authenticatedConfig._authSessionAccessToken = token
    authenticatedConfig._authSessionPrincipalId = principalId
    authenticatedConfig._authSessionGeneration = currentGeneration
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`
    }

    // Attach locale for backend translations
    if (config.headers) {
      config.headers['Accept-Language'] = getLocale()
    }

    // Attach timezone for all GET requests (backend may use it for default date ranges)
    if (config.method === 'get') {
      if (!config.params) {
        config.params = {}
      }
      config.params.timezone = getUserTimezone()
    }

    if (config.headers) {
      const requestURL = String(config.url || '')
      if (shouldMarkAdminUIRequest(requestURL)) {
        config.headers[ADMIN_UI_REQUEST_HEADER] = '1'
      }
      if (shouldMarkUserUIRequest(requestURL)) {
        config.headers[USER_UI_REQUEST_HEADER] = '1'
      }
    }

    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// ==================== Response Interceptor ====================

apiClient.interceptors.response.use(
  (response: AxiosResponse) => {
    const dispatchedConfig = response.config as AuthenticatedRequestConfig
    const dispatchedAccessToken = normalizeAccessToken(dispatchedConfig._authSessionAccessToken)
    if (dispatchedAccessToken) {
      // Re-read the canonical record at response time. A 2xx produced for an
      // old family must not reach callers that would patch the current session.
      // Same-family token rotation remains valid because access-token equality
      // is intentionally not required here.
      const currentSession = authSession.hydrate()
      const currentGeneration = authSession.getGeneration()
      const dispatchedPrincipalId = dispatchedConfig._authSessionPrincipalId ?? null
      const dispatchedGeneration = dispatchedConfig._authSessionGeneration ?? null
      if (
        currentGeneration !== dispatchedGeneration
        || (currentSession.user?.id ?? null) !== dispatchedPrincipalId
      ) {
        return Promise.reject(authSessionChangedResponse())
      }
    }

    // Unwrap standard API response format { code, message, data }
    const apiResponse = response.data as ApiResponse<unknown>
    if (apiResponse && typeof apiResponse === 'object' && 'code' in apiResponse) {
      if (apiResponse.code === 0) {
        // Success - return the data portion
        response.data = apiResponse.data
      } else {
        // API error
        const resp = apiResponse as unknown as Record<string, unknown>
        return Promise.reject({
          status: response.status,
          code: apiResponse.code,
          message: apiResponse.message || 'Unknown error',
          reason: resp.reason,
          metadata: resp.metadata,
        })
      }
    }
    return response
  },
  async (error: AxiosError<ApiResponse<unknown>> | AuthSessionChangedError) => {
    if (error instanceof AuthSessionChangedError) {
      return Promise.reject(authSessionChangedResponse())
    }

    // Request cancellation: keep the original axios cancellation error so callers can ignore it.
    // Otherwise we'd misclassify it as a generic "network error".
    if (error.code === 'ERR_CANCELED' || axios.isCancel(error)) {
      return Promise.reject(error)
    }

    const originalRequest = error.config as AuthenticatedRequestConfig

    // Handle common errors
    if (error.response) {
      const { status, data } = error.response
      const url = String(error.config?.url || '')

      // Validate `data` shape to avoid HTML error pages breaking our error handling.
      const apiData = (typeof data === 'object' && data !== null ? data : {}) as Record<string, any>

      // Ops monitoring disabled: treat as feature-flagged 404, and proactively redirect away
      // from ops pages to avoid broken UI states.
      if (status === 404 && apiData.message === 'Ops monitoring is disabled') {
        try {
          localStorage.setItem('ops_monitoring_enabled_cached', 'false')
        } catch {
          // ignore localStorage failures
        }
        try {
          window.dispatchEvent(new CustomEvent('ops-monitoring-disabled'))
        } catch {
          // ignore event failures
        }

        if (window.location.pathname.startsWith('/admin/ops')) {
          window.location.href = '/admin/settings'
        }

        return Promise.reject({
          status,
          code: 'OPS_DISABLED',
          message: apiData.message || error.message,
          url
        })
      }

      if (status === 423 && apiData.code === 'ADMIN_COMPLIANCE_ACK_REQUIRED') {
        try {
          window.dispatchEvent(new CustomEvent('admin-compliance-required', {
            detail: apiData.metadata || {}
          }))
        } catch {
          // ignore event failures
        }

        return Promise.reject({
          status,
          code: apiData.code,
          message: apiData.message || error.message,
          metadata: apiData.metadata,
        })
      }

      // 401: Try to refresh the token if we have a refresh token
      // This handles TOKEN_EXPIRED, INVALID_TOKEN, TOKEN_REVOKED, etc.
      if (status === 401 && !originalRequest._retry) {
        const currentSession = authSession.getSnapshot()
        const currentGeneration = authSession.getGeneration()
        const sentBearerToken = getSentBearerToken(originalRequest.headers)
        const dispatchedAccessToken = Object.prototype.hasOwnProperty.call(
          originalRequest,
          '_authSessionAccessToken',
        )
          ? normalizeAccessToken(originalRequest._authSessionAccessToken)
          : sentBearerToken
        const principalWasTracked = Object.prototype.hasOwnProperty.call(
          originalRequest,
          '_authSessionPrincipalId',
        )
        const dispatchedPrincipalId = principalWasTracked
          ? originalRequest._authSessionPrincipalId ?? null
          : null
        const generationWasTracked = Object.prototype.hasOwnProperty.call(
          originalRequest,
          '_authSessionGeneration',
        )
        const dispatchedGeneration = generationWasTracked
          ? originalRequest._authSessionGeneration ?? null
          : null

        if (
          dispatchedAccessToken !== currentSession.accessToken
          || (sentBearerToken !== null && sentBearerToken !== currentSession.accessToken)
          || (
            principalWasTracked
            && dispatchedPrincipalId !== (currentSession.user?.id ?? null)
          )
          || (generationWasTracked && dispatchedGeneration !== currentGeneration)
        ) {
          return Promise.reject(authSessionChangedResponse())
        }

        const isAuthEndpoint =
          url.includes('/auth/login') || url.includes('/auth/register') || url.includes('/auth/refresh')

        // If we have a refresh token and this is not an auth endpoint, try to refresh
        if (currentSession.refreshToken && !isAuthEndpoint) {
          originalRequest._retry = true

          try {
            // authSession owns the single-flight promise, so simultaneous 401s
            // all retry with the same rotated token pair.
            const refreshedSession = await refreshAuthSession()
            if (!refreshedSession.accessToken) throw new Error('Token refresh failed')

            const latestSession = authSession.getSnapshot()
            const latestGeneration = authSession.getGeneration()
            if (
              latestSession.accessToken !== refreshedSession.accessToken
              || latestSession.refreshToken !== refreshedSession.refreshToken
              || (latestSession.user?.id ?? null) !== (refreshedSession.user?.id ?? null)
              || latestGeneration !== currentGeneration
            ) {
              return Promise.reject(authSessionChangedResponse())
            }

            if (originalRequest.headers) {
              originalRequest.headers.Authorization = `Bearer ${refreshedSession.accessToken}`
            }
            originalRequest._authRetryExpectedAccessToken = refreshedSession.accessToken
            originalRequest._authRetryExpectedPrincipalId = refreshedSession.user?.id ?? null
            originalRequest._authRetryExpectedGeneration = latestGeneration
            return apiClient(originalRequest)
          } catch (refreshError) {
            if (refreshError instanceof AuthSessionChangedError) {
              // The user logged out or switched accounts while the stale 401
              // was refreshing. Never clear or redirect the newer session.
              return Promise.reject(authSessionChangedResponse())
            }

            if (shouldClearSessionAfterRefreshFailure(refreshError)) {
              // Fatal refresh rejection invalidates only this tab's observed
              // family. It never deletes durable state, so a concurrent login
              // in another tab cannot be erased by a compare-then-delete race.
              if (!authSession.invalidate(currentGeneration)) {
                return Promise.reject(authSessionChangedResponse())
              }

              sessionStorage.setItem('auth_expired', '1')

              if (!window.location.pathname.includes('/login')) {
                window.location.href = sessionExpiredLoginURL(window.location)
              }

              return Promise.reject({
                status: 401,
                code: 'TOKEN_REFRESH_FAILED',
                message: 'Session expired. Please log in again.'
              })
            }

            return Promise.reject({
              status: axios.isAxiosError(refreshError) ? (refreshError.response?.status ?? 0) : 0,
              code: 'TOKEN_REFRESH_TEMPORARILY_UNAVAILABLE',
              message: 'Token refresh is temporarily unavailable. Please try again.'
            })
          }
        }

        // No refresh token or is auth endpoint - clear auth and redirect
        const hasToken = !!currentSession.accessToken
        const headers = error.config?.headers as Record<string, unknown> | undefined
        const authHeader = headers?.Authorization ?? headers?.authorization
        const sentAuth =
          typeof authHeader === 'string'
            ? authHeader.trim() !== ''
            : Array.isArray(authHeader)
              ? authHeader.length > 0
              : !!authHeader

        if (!authSession.invalidate(currentGeneration)) {
          return Promise.reject(authSessionChangedResponse())
        }
        if ((hasToken || sentAuth) && !isAuthEndpoint) {
          sessionStorage.setItem('auth_expired', '1')
        }
        // Only redirect if not already on login page
        if (!window.location.pathname.includes('/login')) {
          window.location.href = sessionExpiredLoginURL(window.location)
        }
      }

      // Return structured error
      return Promise.reject({
        status,
        code: apiData.code,
        reason: apiData.reason,
        error: apiData.error,
        message: apiData.message || apiData.detail || error.message,
        metadata: apiData.metadata,
      })
    }

    // Network error
    return Promise.reject({
      status: 0,
      message: 'Network error. Please check your connection.'
    })
  }
)

export default apiClient

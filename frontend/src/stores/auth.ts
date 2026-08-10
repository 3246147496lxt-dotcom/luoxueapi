/**
 * Authentication Store
 *
 * Pinia exposes reactive UI state, while authSession is the sole authority for
 * access/refresh tokens and their persistence.
 */

import { defineStore } from 'pinia'
import { computed, onScopeDispose, readonly, ref } from 'vue'
import { authAPI, isTotp2FARequired, type LoginResponse } from '@/api/auth'
import {
  authSession,
  isAuthSessionChangedError,
  type AuthSessionSnapshot,
} from '@/auth/authSession'
import type { User, LoginRequest, RegisterRequest, AuthResponse } from '@/types'

const PENDING_AUTH_SESSION_KEY = 'pending_auth_session'
const TOKEN_REFRESH_BUFFER = 120 * 1000
const MAX_TIMEOUT_DELAY = 2_147_483_647

type PendingAuthTokenField = 'pending_auth_token' | 'pending_oauth_token'

interface PendingAuthSessionSummary {
  token: string
  token_field: PendingAuthTokenField
  provider: string
  redirect?: string
  adoption_required?: boolean
  suggested_display_name?: string
  suggested_avatar_url?: string
}

function normalizePendingAuthTokenField(value: unknown): PendingAuthTokenField {
  return value === 'pending_oauth_token' ? 'pending_oauth_token' : 'pending_auth_token'
}

function getPersistedPendingAuthSession(): PendingAuthSessionSummary | null {
  const raw = localStorage.getItem(PENDING_AUTH_SESSION_KEY)
  if (!raw) return null

  try {
    const parsed = JSON.parse(raw) as Partial<PendingAuthSessionSummary> | null
    const provider = typeof parsed?.provider === 'string' ? parsed.provider.trim() : ''
    if (!provider) {
      localStorage.removeItem(PENDING_AUTH_SESSION_KEY)
      return null
    }
    return {
      token: typeof parsed?.token === 'string' ? parsed.token : '',
      token_field: normalizePendingAuthTokenField(parsed?.token_field),
      provider,
      redirect: typeof parsed?.redirect === 'string' ? parsed.redirect : undefined,
      adoption_required: typeof parsed?.adoption_required === 'boolean'
        ? parsed.adoption_required
        : undefined,
      suggested_display_name: typeof parsed?.suggested_display_name === 'string'
        ? parsed.suggested_display_name
        : undefined,
      suggested_avatar_url: typeof parsed?.suggested_avatar_url === 'string'
        ? parsed.suggested_avatar_url
        : undefined,
    }
  } catch {
    localStorage.removeItem(PENDING_AUTH_SESSION_KEY)
    return null
  }
}

function persistPendingAuthSession(session: PendingAuthSessionSummary): void {
  localStorage.setItem(PENDING_AUTH_SESSION_KEY, JSON.stringify(session))
}

function clearPendingAuthSessionStorage(): void {
  localStorage.removeItem(PENDING_AUTH_SESSION_KEY)
}

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const token = ref<string | null>(null)
  const refreshTokenValue = ref<string | null>(null)
  const tokenExpiresAt = ref<number | null>(null)
  const runMode = ref<'standard' | 'simple'>('standard')
  const pendingAuthSession = ref<PendingAuthSessionSummary | null>(null)

  let tokenRefreshTimeoutId: ReturnType<typeof setTimeout> | null = null
  let scheduledTokenExpiry: number | null = null
  let activeUserRefresh: { token: string; promise: Promise<User> } | null = null

  const isAuthenticated = computed(() => !!token.value && !!user.value)
  const isAdmin = computed(() => user.value?.role === 'admin')
  const isSimpleMode = computed(() => runMode.value === 'simple')
  const hasPendingAuthSession = computed(() => pendingAuthSession.value !== null)

  function stopTokenRefresh(): void {
    if (tokenRefreshTimeoutId !== null) {
      clearTimeout(tokenRefreshTimeoutId)
      tokenRefreshTimeoutId = null
    }
    scheduledTokenExpiry = null
  }

  function scheduleTokenRefreshAt(expiresAtMs: number): void {
    if (scheduledTokenExpiry === expiresAtMs && tokenRefreshTimeoutId !== null) return
    stopTokenRefresh()
    scheduledTokenExpiry = expiresAtMs

    const refreshInMs = Math.max(0, expiresAtMs - Date.now() - TOKEN_REFRESH_BUFFER)
    if (refreshInMs <= 0) {
      void performTokenRefresh()
      return
    }

    tokenRefreshTimeoutId = setTimeout(() => {
      tokenRefreshTimeoutId = null
      void performTokenRefresh()
    }, Math.min(refreshInMs, MAX_TIMEOUT_DELAY))
  }

  function syncFromSession(session: AuthSessionSnapshot): void {
    token.value = session.accessToken
    refreshTokenValue.value = session.refreshToken
    tokenExpiresAt.value = session.expiresAt

    if (session.user) {
      const { run_mode: sessionRunMode, ...sessionUser } = session.user
      if (sessionRunMode) runMode.value = sessionRunMode
      user.value = sessionUser as User
    } else {
      user.value = null
      if (!session.accessToken) runMode.value = 'standard'
    }

    if (session.refreshToken && session.expiresAt !== null) {
      scheduleTokenRefreshAt(session.expiresAt)
    } else {
      stopTokenRefresh()
    }
  }

  const unsubscribeSession = authSession.subscribe(syncFromSession)
  onScopeDispose(unsubscribeSession)

  function checkAuth(): void {
    pendingAuthSession.value = getPersistedPendingAuthSession()
    const session = authSession.hydrate()
    syncFromSession(session)

    if (session.accessToken && session.user) {
      refreshUser().catch((error) => {
        console.error('Failed to refresh user on init:', error)
      })
      return
    }

    // Do not keep an incomplete or corrupt persisted login across reloads.
    if (session.accessToken || session.refreshToken || session.user) {
      clearAuth({ preservePendingAuthSession: true })
    }
  }

  async function performTokenRefresh(): Promise<void> {
    if (!refreshTokenValue.value) return

    try {
      const response = await authAPI.refreshToken()
      // Real authAPI calls already commit through authSession. Keeping this
      // patch makes mocked/custom implementations follow the same contract.
      authSession.patch({
        accessToken: response.access_token,
        refreshToken: response.refresh_token,
        expiresAt: Date.now() + response.expires_in * 1000,
      })
    } catch (error) {
      console.error('Token refresh failed:', error)
      // A later authenticated request will surface expiry and perform logout.
    }
  }

  function sessionUserFromResponse(response: AuthResponse): User {
    const { run_mode: _runMode, ...sessionUser } = response.user
    return sessionUser
  }

  function setAuthFromResponse(response: AuthResponse): void {
    const sessionUser = sessionUserFromResponse(response)
    authSession.replace({
      accessToken: response.access_token,
      refreshToken: response.refresh_token ?? null,
      expiresAt: response.expires_in
        ? Date.now() + response.expires_in * 1000
        : null,
      user: sessionUser,
    })
    if (response.user.run_mode) runMode.value = response.user.run_mode
    clearPendingAuthSession()
  }

  async function login(credentials: LoginRequest): Promise<LoginResponse> {
    try {
      const response = await authAPI.login(credentials)
      if (isTotp2FARequired(response)) return response
      setAuthFromResponse(response)
      return response
    } catch (error) {
      if (!isAuthSessionChangedError(error)) {
        clearAuth({ preservePendingAuthSession: pendingAuthSession.value !== null })
      }
      throw error
    }
  }

  async function login2FA(tempToken: string, totpCode: string): Promise<User> {
    try {
      const response = await authAPI.login2FA({ temp_token: tempToken, totp_code: totpCode })
      setAuthFromResponse(response)
      return user.value!
    } catch (error) {
      if (!isAuthSessionChangedError(error)) {
        clearAuth({ preservePendingAuthSession: pendingAuthSession.value !== null })
      }
      throw error
    }
  }

  async function register(userData: RegisterRequest): Promise<User> {
    try {
      const response = await authAPI.register(userData)
      setAuthFromResponse(response)
      return user.value!
    } catch (error) {
      if (!isAuthSessionChangedError(error)) {
        clearAuth({ preservePendingAuthSession: pendingAuthSession.value !== null })
      }
      throw error
    }
  }

  /** Set an OAuth/SSO access token while preserving its freshly stored refresh context. */
  async function setToken(newToken: string): Promise<User> {
    stopTokenRefresh()
    const oauthContext = authSession.getSnapshot()
    if (authSession.getGeneration() && oauthContext.accessToken === newToken) {
      authSession.patch({ accessToken: newToken, user: null })
    } else {
      authSession.replace({
        accessToken: newToken,
        refreshToken: oauthContext.refreshToken,
        expiresAt: oauthContext.expiresAt,
        user: null,
      })
    }

    try {
      const userData = await refreshUser()
      clearPendingAuthSession()
      return userData
    } catch (error) {
      if (!isAuthSessionChangedError(error)) {
        clearAuth({ preservePendingAuthSession: pendingAuthSession.value !== null })
      }
      throw error
    }
  }

  function setPendingAuthSession(session: PendingAuthSessionSummary | null): void {
    pendingAuthSession.value = session
    if (session) {
      persistPendingAuthSession(session)
      return
    }
    clearPendingAuthSessionStorage()
  }

  function clearPendingAuthSession(): void {
    setPendingAuthSession(null)
  }

  async function logout(): Promise<void> {
    const current = authSession.getSnapshot()
    const logoutSession = {
      accessToken: current.accessToken,
      refreshToken: current.refreshToken,
    }
    // Commit local logout before waiting for the network. A later login is a
    // newer family and must not be cleared when this revocation settles.
    clearAuth()
    try {
      await authAPI.logout(logoutSession)
    } catch (error) {
      console.warn('Logout API call failed after clearing local session', error)
    }
  }

  function refreshUser(): Promise<User> {
    const requestToken = token.value
    if (!requestToken) return Promise.reject(new Error('Not authenticated'))
    if (activeUserRefresh?.token === requestToken) return activeUserRefresh.promise

    const operation = authAPI.getCurrentUser()
      .then((response) => {
        const responseUser = response.data as User & { run_mode?: 'standard' | 'simple' }
        const { run_mode: _runMode, ...userData } = responseUser
        authSession.patch({ user: userData })
        if (responseUser.run_mode) runMode.value = responseUser.run_mode
        return userData
      })
      .catch((error) => {
        if (
          !isAuthSessionChangedError(error)
          && (error as { status?: number }).status === 401
        ) {
          clearAuth({ preservePendingAuthSession: pendingAuthSession.value !== null })
        }
        throw error
      })

    const record = { token: requestToken, promise: operation }
    record.promise = operation.finally(() => {
      if (activeUserRefresh === record) activeUserRefresh = null
    })
    activeUserRefresh = record
    return record.promise
  }

  function clearAuth(options?: { preservePendingAuthSession?: boolean }): void {
    activeUserRefresh = null
    stopTokenRefresh()
    authSession.clear()
    syncFromSession(authSession.getSnapshot())

    if (options?.preservePendingAuthSession) {
      pendingAuthSession.value = getPersistedPendingAuthSession()
      return
    }
    pendingAuthSession.value = null
    clearPendingAuthSessionStorage()
  }

  return {
    user,
    token: readonly(token),
    runMode: readonly(runMode),
    pendingAuthSession: readonly(pendingAuthSession),
    isAuthenticated,
    isAdmin,
    isSimpleMode,
    hasPendingAuthSession,
    login,
    login2FA,
    register,
    setToken,
    logout,
    checkAuth,
    refreshUser,
    setPendingAuthSession,
    clearPendingAuthSession,
  }
})

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import axios from 'axios'
import type { AxiosInstance, AxiosResponse } from 'axios'

// 需要在导入 client 之前设置 mock
vi.mock('@/i18n', () => ({
  getLocale: () => 'zh-CN',
}))

function sessionUser(id: number, username: string) {
  return {
    id,
    username,
    email: `${username}@example.com`,
    role: 'user' as const,
    balance: 0,
    concurrency: 1,
    status: 'active' as const,
    allowed_groups: null,
    balance_notify_enabled: false,
    balance_notify_threshold: null,
    balance_notify_extra_emails: [],
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  }
}

describe('API Client', () => {
  let apiClient: AxiosInstance

  beforeEach(async () => {
    localStorage.clear()
    sessionStorage.clear()
    window.history.replaceState({}, '', '/')
    // 每次测试重新导入以获取干净的模块状态
    vi.resetModules()
    const mod = await import('@/api/client')
    apiClient = mod.apiClient
  })

  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllEnvs()
  })

  // --- 请求拦截器 ---

  describe('请求拦截器', () => {
    it('规范化相对 API base，避免在回调页拼出相对 v1 路径', async () => {
      vi.resetModules()
      vi.stubEnv('VITE_API_BASE_URL', 'api/v1')

      const mod = await import('@/api/client')

      expect(mod.apiClient.defaults.baseURL).toBe('/api/v1')
      expect(mod.buildApiUrl('/auth/oauth/github/callback?code=abc')).toBe(
        '/api/v1/auth/oauth/github/callback?code=abc'
      )
    })

    it('自动附加 Authorization 头', async () => {
      localStorage.setItem('auth_token', 'my-jwt-token')

      // 拦截实际请求
      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 0, data: {} },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter

      await apiClient.get('/test')

      const config = adapter.mock.calls[0][0]
      expect(config.headers.get('Authorization')).toBe('Bearer my-jwt-token')
    })

    it('无 token 时不附加 Authorization 头', async () => {
      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 0, data: {} },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter

      await apiClient.get('/test')

      const config = adapter.mock.calls[0][0]
      expect(config.headers.get('Authorization')).toBeFalsy()
    })

    it('GET 请求自动附加 timezone 参数', async () => {
      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 0, data: {} },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter

      await apiClient.get('/test')

      const config = adapter.mock.calls[0][0]
      expect(config.params).toHaveProperty('timezone')
    })

    it('POST 请求不附加 timezone 参数', async () => {
      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 0, data: {} },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter

      await apiClient.post('/test', { foo: 'bar' })

      const config = adapter.mock.calls[0][0]
      expect(config.params?.timezone).toBeUndefined()
    })

    it('显式清除 JSON 默认头时保留 FormData 供浏览器生成 multipart boundary', async () => {
      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 0, data: {} },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter
      const formData = new FormData()
      formData.append('file', new Blob(['voice'], { type: 'audio/webm' }), 'voice.webm')

      await apiClient.post('/chat/transcriptions', formData, {
        headers: {
          'Content-Type': undefined,
          'Idempotency-Key': '11111111-2222-4333-8444-555555555555',
        },
      })

      const config = adapter.mock.calls[0][0]
      expect(config.data).toBe(formData)
      // Axios applies its generic POST fallback before invoking a custom test
      // adapter. The real browser adapter clears that fallback for FormData;
      // retaining the FormData object here proves JSON serialization did not
      // happen before that browser-specific boundary step.
      expect(config.headers.get('Idempotency-Key'))
        .toBe('11111111-2222-4333-8444-555555555555')
    })

    it('请求默认带 withCredentials 以支持跨域 cookie', async () => {
      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 0, data: {} },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter

      await apiClient.post('/auth/oauth/bind-token')

      const config = adapter.mock.calls[0][0]
      expect(config.withCredentials).toBe(true)
    })

    it('Admin API 在进入管理页面前也带 Admin UI 标记', async () => {
      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 0, data: {} },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter

      await apiClient.get('/admin/users')

      const config = adapter.mock.calls[0][0]
      expect(config.headers.get('X-Admin-UI-Request')).toBe('1')
    })

    it('管理页面调用共享 API 时带 Admin UI 标记', async () => {
      window.history.replaceState({}, '', '/admin/dashboard')
      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 0, data: {} },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter

      await apiClient.get('/groups/available')

      const config = adapter.mock.calls[0][0]
      expect(config.headers.get('X-Admin-UI-Request')).toBe('1')
    })

    it('普通用户页面调用共享 API 时不带 Admin UI 标记', async () => {
      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 0, data: {} },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter

      await apiClient.get('/groups/available')

      const config = adapter.mock.calls[0][0]
      expect(config.headers.get('X-Admin-UI-Request')).toBeFalsy()
    })

    it('用户侧 timing API 自动带 User UI 标记', async () => {
      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 0, data: {} },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter

      await apiClient.get('/auth/me')

      const config = adapter.mock.calls[0][0]
      expect(config.headers.get('X-User-UI-Request')).toBe('1')
      expect(config.headers.get('X-Admin-UI-Request')).toBeFalsy()
    })

    it('支付用户 API 带 User UI 标记，公开支付 API 不带', async () => {
      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 0, data: {} },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter

      await apiClient.get('/payment/plans')
      expect(adapter.mock.calls[0][0].headers.get('X-User-UI-Request')).toBe('1')

      await apiClient.post('/payment/public/orders/verify', {})
      expect(adapter.mock.calls[1][0].headers.get('X-User-UI-Request')).toBeFalsy()
    })

    it('管理页调用共享 API 时同时带 Admin 与 User UI 标记', async () => {
      window.history.replaceState({}, '', '/admin/dashboard')
      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 0, data: {} },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter

      await apiClient.get('/keys')

      const config = adapter.mock.calls[0][0]
      expect(config.headers.get('X-Admin-UI-Request')).toBe('1')
      expect(config.headers.get('X-User-UI-Request')).toBe('1')
    })
  })

  // --- 响应拦截器 ---

  describe('响应拦截器', () => {
    it('code=0 时解包 data 字段', async () => {
      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 0, data: { name: 'test' }, message: 'ok' },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter

      const response = await apiClient.get('/test')
      expect(response.data).toEqual({ name: 'test' })
    })

    it('code!=0 时拒绝并返回结构化错误', async () => {
      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 1001, message: '参数错误', data: null },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter

      await expect(apiClient.get('/test')).rejects.toEqual(
        expect.objectContaining({
          code: 1001,
          message: '参数错误',
        })
      )
    })

    it('部署与运营合规未确认时广播事件且保留登录态', async () => {
      localStorage.setItem('auth_token', 'admin-token')
      const listener = vi.fn()
      window.addEventListener('admin-compliance-required', listener)

      const adapter = vi.fn().mockRejectedValue({
        response: {
          status: 423,
          data: {
            code: 'ADMIN_COMPLIANCE_ACK_REQUIRED',
            message: 'administrator compliance acknowledgement is required',
            metadata: {
              version: 'v2026.06.10',
              document_path_zh: 'docs/legal/admin-compliance.zh.md',
              document_path_en: 'docs/legal/admin-compliance.en.md',
            },
          },
        },
        config: {
          url: '/admin/users',
          headers: { Authorization: 'Bearer admin-token' },
        },
        code: 'ERR_BAD_REQUEST',
      })
      apiClient.defaults.adapter = adapter

      await expect(apiClient.get('/admin/users')).rejects.toEqual(
        expect.objectContaining({
          status: 423,
          code: 'ADMIN_COMPLIANCE_ACK_REQUIRED',
          metadata: expect.objectContaining({
            version: 'v2026.06.10',
          }),
        })
      )

      expect(listener).toHaveBeenCalledTimes(1)
      expect((listener.mock.calls[0][0] as CustomEvent).detail).toEqual(
        expect.objectContaining({
          version: 'v2026.06.10',
        })
      )
      expect(localStorage.getItem('auth_token')).toBe('admin-token')

      window.removeEventListener('admin-compliance-required', listener)
    })

    it('旧客户端同用户重登后拒绝旧认证 2xx 且调用方不能 patch canonical', async () => {
      const { authSession } = await import('@/auth/authSession')
      const sameUser = sessionUser(1, 'same-user-2xx-fence')
      authSession.replace({
        accessToken: 'old-2xx-access',
        refreshToken: 'old-2xx-refresh',
        expiresAt: Date.now() + 900_000,
        user: sameUser,
      })

      let resolveResponse!: (value: unknown) => void
      const adapter = vi.fn((config: any) => new Promise<AxiosResponse>((resolve) => {
        resolveResponse = (value) => resolve({
          status: 200,
          data: { code: 0, data: value },
          headers: {},
          config,
          statusText: 'OK',
        })
      }))
      apiClient.defaults.adapter = adapter
      const patch = vi.spyOn(authSession, 'patch')
      const pending = apiClient.get('/auth/me').then((response) => {
        authSession.patch({ user: response.data as ReturnType<typeof sessionUser> })
      })
      await vi.waitFor(() => expect(adapter).toHaveBeenCalledTimes(1))

      const generation = authSession.getGeneration()
      const canonicalHead = localStorage.getItem('auth_session_head')
      const canonicalFamily = localStorage.getItem(`auth_session_family:${generation}`)
      localStorage.removeItem('auth_token')
      localStorage.removeItem('refresh_token')
      localStorage.removeItem('token_expires_at')
      localStorage.removeItem('auth_user')
      localStorage.setItem('auth_token', 'legacy-new-2xx-access')
      localStorage.setItem('refresh_token', 'legacy-new-2xx-refresh')
      localStorage.setItem('token_expires_at', String(Date.now() + 900_000))
      localStorage.setItem('auth_user', JSON.stringify(sameUser))
      resolveResponse({ ...sameUser, username: 'stale-profile' })

      await expect(pending).rejects.toEqual(expect.objectContaining({
        status: 409,
        code: 'AUTH_SESSION_CHANGED',
      }))
      expect(patch).not.toHaveBeenCalled()
      expect(authSession.getSnapshot().accessToken).toBeNull()
      expect(authSession.getGeneration()).toBeNull()
      expect(localStorage.getItem('auth_token')).toBe('legacy-new-2xx-access')
      expect(localStorage.getItem('refresh_token')).toBe('legacy-new-2xx-refresh')
      expect(localStorage.getItem('auth_session_head')).toBe(canonicalHead)
      expect(localStorage.getItem(`auth_session_family:${generation}`)).toBe(canonicalFamily)
    })

    it('同 family token rotation 后仍接受先前派发的认证 2xx', async () => {
      const { authSession } = await import('@/auth/authSession')
      const sameUser = sessionUser(1, 'same-family-2xx')
      authSession.replace({
        accessToken: 'before-rotation-access',
        refreshToken: 'before-rotation-refresh',
        expiresAt: Date.now() + 900_000,
        user: sameUser,
      })

      let resolveResponse!: () => void
      const adapter = vi.fn((config: any) => new Promise<AxiosResponse>((resolve) => {
        resolveResponse = () => resolve({
          status: 200,
          data: { code: 0, data: { ok: true } },
          headers: {},
          config,
          statusText: 'OK',
        })
      }))
      apiClient.defaults.adapter = adapter
      const pending = apiClient.get('/same-family-2xx')
      await vi.waitFor(() => expect(adapter).toHaveBeenCalledTimes(1))

      const generation = authSession.getGeneration()
      authSession.patch({
        accessToken: 'after-rotation-access',
        refreshToken: 'after-rotation-refresh',
      })
      resolveResponse()

      await expect(pending).resolves.toEqual(expect.objectContaining({ data: { ok: true } }))
      expect(authSession.getGeneration()).toBe(generation)
    })
  })

  // --- 401 Token 刷新 ---

  describe('401 Token 刷新', () => {
    it('无 refresh_token 时 401 仅失效本标签且不删除持久会话', async () => {
      localStorage.setItem('auth_token', 'expired-token')
      // 不设置 refresh_token

      // Mock window.location
      const originalLocation = window.location
      Object.defineProperty(window, 'location', {
        value: { ...originalLocation, pathname: '/dashboard', href: '/dashboard' },
        writable: true,
      })

      const adapter = vi.fn().mockRejectedValue({
        response: {
          status: 401,
          data: { code: 'TOKEN_EXPIRED', message: 'Token expired' },
        },
        config: {
          url: '/test',
          headers: { Authorization: 'Bearer expired-token' },
        },
        code: 'ERR_BAD_REQUEST',
      })
      apiClient.defaults.adapter = adapter

      await expect(apiClient.get('/test')).rejects.toBeDefined()

      const { authSession } = await import('@/auth/authSession')
      expect(authSession.getSnapshot().accessToken).toBeNull()
      expect(localStorage.getItem('auth_token')).toBe('expired-token')
      expect(sessionStorage.getItem('auth_expired')).toBe('1')

      // 恢复 location
      Object.defineProperty(window, 'location', {
        value: originalLocation,
        writable: true,
      })
    })

    it('额度授权深链会在 401 失效登录时安全保留完整回跳地址', async () => {
      localStorage.setItem('auth_token', 'expired-token')

      const originalLocation = window.location
      Object.defineProperty(window, 'location', {
        configurable: true,
        value: {
          ...originalLocation,
          pathname: '/quota-viewer/authorize',
          search: '?user_code=ABCD-EFGH',
          hash: '#confirm',
          href: '/quota-viewer/authorize?user_code=ABCD-EFGH#confirm',
        },
        writable: true,
      })

      try {
        const adapter = vi.fn().mockRejectedValue({
          response: {
            status: 401,
            data: { code: 'TOKEN_EXPIRED', message: 'Token expired' },
          },
          config: {
            url: '/quota/authorizations/ABCD-EFGH',
            headers: { Authorization: 'Bearer expired-token' },
          },
          code: 'ERR_BAD_REQUEST',
        })
        apiClient.defaults.adapter = adapter

        await expect(apiClient.get('/quota/authorizations/ABCD-EFGH')).rejects.toBeDefined()

        expect(window.location.href).toBe(
          '/login?redirect=%2Fquota-viewer%2Fauthorize%3Fuser_code%3DABCD-EFGH%23confirm',
        )
      } finally {
        Object.defineProperty(window, 'location', {
          configurable: true,
          value: originalLocation,
          writable: true,
        })
      }
    })

    it('并发 401 只刷新一次并用同一新 token 重试', async () => {
      localStorage.setItem('auth_token', 'expired-token')
      localStorage.setItem('refresh_token', 'refresh-once')

      let resolveRefresh!: (value: unknown) => void
      const refreshPost = vi.spyOn(axios, 'post').mockReturnValue(
        new Promise((resolve) => {
          resolveRefresh = resolve
        }) as ReturnType<typeof axios.post>
      )

      const adapter = vi.fn((config: any) => {
        if (!config._retry) {
          return Promise.reject({
            response: {
              status: 401,
              data: { code: 'TOKEN_EXPIRED', message: 'Token expired' },
            },
            config,
            code: 'ERR_BAD_REQUEST',
          })
        }
        return Promise.resolve({
          status: 200,
          data: { code: 0, data: { ok: true } },
          headers: {},
          config,
          statusText: 'OK',
        })
      })
      apiClient.defaults.adapter = adapter

      const first = apiClient.get('/concurrent/one')
      const second = apiClient.get('/concurrent/two')
      await vi.waitFor(() => expect(refreshPost).toHaveBeenCalledTimes(1))

      resolveRefresh({
        data: {
          code: 0,
          data: {
            access_token: 'rotated-access',
            refresh_token: 'rotated-refresh',
            expires_in: 900,
          },
        },
      })

      await expect(Promise.all([first, second])).resolves.toHaveLength(2)
      expect(refreshPost).toHaveBeenCalledTimes(1)
      expect(adapter).toHaveBeenCalledTimes(4)
      expect(localStorage.getItem('auth_token')).toBe('rotated-access')
      expect(localStorage.getItem('refresh_token')).toBe('rotated-refresh')
    })

    it('跨标签刷新 loser 复用新 access token 重试且不标记 auth_expired', async () => {
      localStorage.setItem('auth_token', 'expired-token')
      localStorage.setItem('refresh_token', 'old-refresh')
      localStorage.setItem('auth_user', JSON.stringify({ id: 1, username: 'same-user' }))

      let rejectRefresh!: (reason: unknown) => void
      const refreshPost = vi.spyOn(axios, 'post').mockReturnValue(
        new Promise((_resolve, reject) => {
          rejectRefresh = reject
        }) as ReturnType<typeof axios.post>
      )

      let retryAuthorization: string | null = null
      const adapter = vi.fn((config: any) => {
        if (!config._retry) {
          return Promise.reject({
            response: {
              status: 401,
              data: { code: 'TOKEN_EXPIRED', message: 'Token expired' },
            },
            config,
            code: 'ERR_BAD_REQUEST',
          })
        }

        retryAuthorization = typeof config.headers?.get === 'function'
          ? config.headers.get('Authorization')
          : config.headers?.Authorization
        return Promise.resolve({
          status: 200,
          data: { code: 0, data: { ok: true } },
          headers: {},
          config,
          statusText: 'OK',
        })
      })
      apiClient.defaults.adapter = adapter

      const pending = apiClient.get('/cross-tab-refresh-loser')
      await vi.waitFor(() => expect(refreshPost).toHaveBeenCalledTimes(1))

      const { createAuthSession } = await import('@/auth/authSession')
      const peerSession = createAuthSession({
        storage: localStorage,
        tabStorage: null,
        eventTarget: null,
        lockManager: null,
      })
      peerSession.hydrate()
      peerSession.patch({
        accessToken: 'other-tab-access',
        refreshToken: 'other-tab-refresh',
        expiresAt: Date.now() + 900_000,
        user: sessionUser(1, 'same-user'),
      })
      peerSession.dispose()
      window.dispatchEvent(new StorageEvent('storage', { key: 'auth_token' }))
      await Promise.resolve()

      rejectRefresh({
        isAxiosError: true,
        response: { status: 401, data: { code: 'REFRESH_ALREADY_ROTATED' } },
      })

      await expect(pending).resolves.toEqual(expect.objectContaining({ data: { ok: true } }))
      expect(adapter).toHaveBeenCalledTimes(2)
      expect(retryAuthorization).toBe('Bearer other-tab-access')
      expect(localStorage.getItem('auth_token')).toBe('other-tab-access')
      expect(localStorage.getItem('refresh_token')).toBe('other-tab-refresh')
      expect(sessionStorage.getItem('auth_expired')).toBeNull()
    })

    it('迟到的刷新失败遇到换用户时返回 AUTH_SESSION_CHANGED 且保留新会话', async () => {
      localStorage.setItem('auth_token', 'old-access')
      localStorage.setItem('refresh_token', 'old-refresh')
      localStorage.setItem('auth_user', JSON.stringify({ id: 1, username: 'old-user' }))

      let rejectRefresh!: (reason: unknown) => void
      vi.spyOn(axios, 'post').mockReturnValue(
        new Promise((_resolve, reject) => {
          rejectRefresh = reject
        }) as ReturnType<typeof axios.post>
      )

      const adapter = vi.fn((config: any) => Promise.reject({
        response: {
          status: 401,
          data: { code: 'TOKEN_EXPIRED', message: 'Token expired' },
        },
        config,
        code: 'ERR_BAD_REQUEST',
      }))
      apiClient.defaults.adapter = adapter

      const pending = apiClient.get('/stale-user-request')
      await vi.waitFor(() => expect(axios.post).toHaveBeenCalledTimes(1))

      localStorage.setItem('auth_token', 'new-user-access')
      localStorage.setItem('refresh_token', 'new-user-refresh')
      localStorage.setItem('token_expires_at', String(Date.now() + 900_000))
      localStorage.setItem('auth_user', JSON.stringify({ id: 2, username: 'new-user' }))
      window.dispatchEvent(new StorageEvent('storage', { key: 'auth_token' }))
      await Promise.resolve()

      rejectRefresh({
        isAxiosError: true,
        response: { status: 401, data: { code: 'REFRESH_REJECTED' } },
      })

      await expect(pending).rejects.toEqual(expect.objectContaining({
        status: 409,
        code: 'AUTH_SESSION_CHANGED',
      }))
      expect(adapter).toHaveBeenCalledTimes(1)
      expect(localStorage.getItem('auth_token')).toBe('new-user-access')
      expect(localStorage.getItem('refresh_token')).toBe('new-user-refresh')
      expect(sessionStorage.getItem('auth_expired')).toBeNull()
    })

    it.each([
      {
        label: '切换到有 refresh token 的新用户',
        initial: {
          accessToken: 'old-user-access',
          refreshToken: 'old-user-refresh',
          user: sessionUser(1, 'old-user'),
        },
        next: {
          accessToken: 'new-user-access',
          refreshToken: 'new-user-refresh',
          user: sessionUser(2, 'new-user'),
        },
      },
      {
        label: '切换到无 refresh token 的新用户',
        initial: {
          accessToken: 'old-user-access',
          refreshToken: 'old-user-refresh',
          user: sessionUser(1, 'old-user'),
        },
        next: {
          accessToken: 'new-user-access-no-refresh',
          refreshToken: null,
          user: sessionUser(2, 'new-user-no-refresh'),
        },
      },
      {
        label: '同用户已由 peer tab 轮换 token',
        initial: {
          accessToken: 'old-user-access',
          refreshToken: 'old-user-refresh',
          user: sessionUser(1, 'same-user'),
        },
        next: {
          accessToken: 'peer-rotated-access',
          refreshToken: 'peer-rotated-refresh',
          user: sessionUser(1, 'same-user'),
        },
      },
      {
        label: '匿名请求发出后登录',
        initial: {
          accessToken: null,
          refreshToken: null,
          user: null,
        },
        next: {
          accessToken: 'new-login-access',
          refreshToken: 'new-login-refresh',
          user: sessionUser(3, 'new-login'),
        },
      },
    ])('旧 POST 的 401 在$label后不刷新、不重放且不清新会话', async ({ initial, next }) => {
      const { authSession } = await import('@/auth/authSession')
      authSession.replace({
        ...initial,
        expiresAt: initial.accessToken ? Date.now() - 1 : null,
      })

      const refreshPost = vi.spyOn(axios, 'post').mockRejectedValue(
        new Error('stale request must not refresh the current session'),
      )
      let rejectRequest!: () => void
      const adapter = vi.fn((config: any) => new Promise<never>((_resolve, reject) => {
        rejectRequest = () => reject({
          response: {
            status: 401,
            data: { code: 'TOKEN_EXPIRED', message: 'Token expired' },
          },
          config,
          code: 'ERR_BAD_REQUEST',
        })
      }))
      apiClient.defaults.adapter = adapter

      const pending = apiClient.post('/sensitive-mutation', { operation: 'write' })
      const rejection = expect(pending).rejects.toEqual(expect.objectContaining({
        status: 409,
        code: 'AUTH_SESSION_CHANGED',
      }))
      await vi.waitFor(() => expect(adapter).toHaveBeenCalledTimes(1))

      authSession.replace({
        ...next,
        expiresAt: Date.now() + 900_000,
      })
      rejectRequest()

      await rejection
      expect(refreshPost).not.toHaveBeenCalled()
      expect(adapter).toHaveBeenCalledTimes(1)
      expect(authSession.getSnapshot()).toEqual(expect.objectContaining({
        accessToken: next.accessToken,
        refreshToken: next.refreshToken,
        user: next.user,
      }))
      expect(sessionStorage.getItem('auth_expired')).toBeNull()
    })

    it('fatal refresh 进入本标签失效边界前的新登录不会被删除', async () => {
      const { authSession } = await import('@/auth/authSession')
      authSession.replace({
        accessToken: 'old-access-before-fatal-refresh',
        refreshToken: 'old-refresh-before-fatal-refresh',
        expiresAt: Date.now() - 1,
        user: sessionUser(1, 'old-before-fatal-refresh'),
      })
      const oldGeneration = authSession.getGeneration()
      const invalidate = vi.spyOn(authSession, 'invalidate')

      const nextSession = {
        accessToken: 'new-access-after-fatal-refresh',
        refreshToken: 'new-refresh-after-fatal-refresh',
        expiresAt: Date.now() + 900_000,
        user: sessionUser(2, 'new-after-fatal-refresh'),
      }
      const refresh = vi.spyOn(authSession, 'refresh').mockImplementation(async () => {
        queueMicrotask(() => authSession.replace(nextSession))
        throw {
          isAxiosError: true,
          response: { status: 401, data: { code: 'REFRESH_REJECTED' } },
        }
      })
      const adapter = vi.fn((config: any) => Promise.reject({
        response: {
          status: 401,
          data: { code: 'TOKEN_EXPIRED', message: 'Token expired' },
        },
        config,
        code: 'ERR_BAD_REQUEST',
      }))
      apiClient.defaults.adapter = adapter

      await expect(apiClient.post('/fatal-refresh-race', { operation: 'write' })).rejects.toEqual(
        expect.objectContaining({
          status: 409,
          code: 'AUTH_SESSION_CHANGED',
        }),
      )

      expect(refresh).toHaveBeenCalledTimes(1)
      expect(invalidate).toHaveBeenCalledWith(oldGeneration)
      expect(adapter).toHaveBeenCalledTimes(1)
      expect(authSession.getSnapshot()).toEqual(nextSession)
      expect(localStorage.getItem('auth_token')).toBe(nextSession.accessToken)
      expect(localStorage.getItem('refresh_token')).toBe(nextSession.refreshToken)
      expect(sessionStorage.getItem('auth_expired')).toBeNull()
    })

    it('refresh flight 中同用户 logout 再 login 会换 family 且旧 POST 不重放', async () => {
      const { authSession } = await import('@/auth/authSession')
      const sameUser = sessionUser(1, 'same-user-new-family')
      authSession.replace({
        accessToken: 'old-family-access',
        refreshToken: 'old-family-refresh',
        expiresAt: Date.now() - 1,
        user: sameUser,
      })

      let resolveRefresh!: (value: unknown) => void
      const refreshPost = vi.spyOn(axios, 'post').mockReturnValue(
        new Promise((resolve) => {
          resolveRefresh = resolve
        }) as ReturnType<typeof axios.post>,
      )
      const adapter = vi.fn((config: any) => {
        if (config._retry) {
          return Promise.resolve({
            status: 200,
            data: { code: 0, data: { replayed: true } },
            headers: {},
            config,
            statusText: 'OK',
          })
        }
        return Promise.reject({
          response: {
            status: 401,
            data: { code: 'TOKEN_EXPIRED', message: 'Token expired' },
          },
          config,
          code: 'ERR_BAD_REQUEST',
        })
      })
      apiClient.defaults.adapter = adapter

      const pending = apiClient.post('/same-user-family-fence', { operation: 'write' })
      await vi.waitFor(() => expect(refreshPost).toHaveBeenCalledTimes(1))

      authSession.clear()
      authSession.replace({
        accessToken: 'new-family-access',
        refreshToken: 'new-family-refresh',
        expiresAt: Date.now() + 900_000,
        user: sameUser,
      })
      resolveRefresh({
        data: {
          code: 0,
          data: {
            access_token: 'stale-rotated-access',
            refresh_token: 'stale-rotated-refresh',
            expires_in: 900,
          },
        },
      })

      await expect(pending).rejects.toEqual(expect.objectContaining({
        status: 409,
        code: 'AUTH_SESSION_CHANGED',
      }))
      expect(adapter).toHaveBeenCalledTimes(1)
      expect(authSession.getSnapshot()).toEqual(expect.objectContaining({
        accessToken: 'new-family-access',
        refreshToken: 'new-family-refresh',
        user: expect.objectContaining({ id: sameUser.id }),
      }))
      expect(sessionStorage.getItem('auth_expired')).toBeNull()
    })

    it('旧客户端遗留 canonical 后同用户重登会 fail closed 且旧 POST 不 retry', async () => {
      const { authSession } = await import('@/auth/authSession')
      const sameUser = sessionUser(1, 'legacy-client-same-user')
      authSession.replace({
        accessToken: 'bound-old-access',
        refreshToken: 'bound-old-refresh',
        expiresAt: Date.now() - 1,
        user: sameUser,
      })
      const oldGeneration = authSession.getGeneration()
      const oldHead = localStorage.getItem('auth_session_head')
      const oldFamily = localStorage.getItem(`auth_session_family:${oldGeneration}`)

      let resolveRefresh!: (value: unknown) => void
      const refreshPost = vi.spyOn(axios, 'post').mockReturnValue(
        new Promise((resolve) => {
          resolveRefresh = resolve
        }) as ReturnType<typeof axios.post>,
      )
      const adapter = vi.fn((config: any) => {
        if (config._retry) {
          return Promise.resolve({
            status: 200,
            data: { code: 0, data: { replayed: true } },
            headers: {},
            config,
            statusText: 'OK',
          })
        }
        return Promise.reject({
          response: {
            status: 401,
            data: { code: 'TOKEN_EXPIRED', message: 'Token expired' },
          },
          config,
          code: 'ERR_BAD_REQUEST',
        })
      })
      apiClient.defaults.adapter = adapter

      const pending = apiClient.post('/legacy-client-family-fence', { operation: 'write' })
      await vi.waitFor(() => expect(refreshPost).toHaveBeenCalledTimes(1))

      // Simulate an older tab: logout removes only the four legacy keys, then
      // same-user login writes a new token tuple while leaving new metadata.
      localStorage.removeItem('auth_token')
      localStorage.removeItem('refresh_token')
      localStorage.removeItem('token_expires_at')
      localStorage.removeItem('auth_user')
      localStorage.setItem('auth_token', 'legacy-new-access')
      localStorage.setItem('refresh_token', 'legacy-new-refresh')
      localStorage.setItem('token_expires_at', String(Date.now() + 900_000))
      localStorage.setItem('auth_user', JSON.stringify(sameUser))
      expect(localStorage.getItem('auth_session_generation')).toBe(oldGeneration)
      expect(localStorage.getItem('auth_session_head')).toBe(oldHead)
      expect(localStorage.getItem(`auth_session_family:${oldGeneration}`)).toBe(oldFamily)

      resolveRefresh({
        data: {
          code: 0,
          data: {
            access_token: 'stale-old-family-access',
            refresh_token: 'stale-old-family-refresh',
            expires_in: 900,
          },
        },
      })

      await expect(pending).rejects.toEqual(expect.objectContaining({
        status: 409,
        code: 'AUTH_SESSION_CHANGED',
      }))
      expect(adapter).toHaveBeenCalledTimes(1)
      expect(authSession.getSnapshot().accessToken).toBeNull()
      expect(authSession.getSnapshot().refreshToken).toBeNull()
      expect(authSession.getGeneration()).toBeNull()
      expect(localStorage.getItem('auth_token')).toBe('legacy-new-access')
      expect(localStorage.getItem('refresh_token')).toBe('legacy-new-refresh')
      expect(localStorage.getItem('auth_session_generation')).toBe(oldGeneration)
      expect(localStorage.getItem('auth_session_head')).toBe(oldHead)
      expect(localStorage.getItem(`auth_session_family:${oldGeneration}`)).toBe(oldFamily)
      expect(sessionStorage.getItem('auth_expired')).toBeNull()
    })

    it.each([
      ['网络故障', undefined],
      ['服务端暂时不可用', 503],
      ['刷新限流', 429],
    ])('刷新遇到%s时保留原子 token 对以便稍后重试', async (_label, refreshStatus) => {
      localStorage.setItem('auth_token', 'expired-token')
      localStorage.setItem('refresh_token', 'retryable-refresh')
      localStorage.setItem('token_expires_at', '123456789')

      vi.spyOn(axios, 'post').mockRejectedValue({
        isAxiosError: true,
        message: 'refresh temporarily unavailable',
        response:
          refreshStatus === undefined
            ? undefined
            : { status: refreshStatus, data: { code: 'TEMPORARY_FAILURE' } },
      })

      const adapter = vi.fn().mockRejectedValue({
        response: {
          status: 401,
          data: { code: 'TOKEN_EXPIRED', message: 'Token expired' },
        },
        config: {
          url: '/retryable-refresh',
          headers: { Authorization: 'Bearer expired-token' },
        },
        code: 'ERR_BAD_REQUEST',
      })
      apiClient.defaults.adapter = adapter

      await expect(apiClient.get('/retryable-refresh')).rejects.toEqual(
        expect.objectContaining({
          code: 'TOKEN_REFRESH_TEMPORARILY_UNAVAILABLE',
        })
      )

      expect(localStorage.getItem('auth_token')).toBe('expired-token')
      expect(localStorage.getItem('refresh_token')).toBe('retryable-refresh')
      expect(localStorage.getItem('token_expires_at')).toBe('123456789')
      expect(sessionStorage.getItem('auth_expired')).toBeNull()
    })

    it.each([400, 401, 403])('刷新返回不可恢复的 %s 时仅失效本标签并保留持久 family', async (refreshStatus) => {
      window.history.replaceState({}, '', '/login')
      localStorage.setItem('auth_token', 'expired-token')
      localStorage.setItem('refresh_token', 'invalid-refresh')

      vi.spyOn(axios, 'post').mockRejectedValue({
        isAxiosError: true,
        message: 'refresh rejected',
        response: { status: refreshStatus, data: { code: 'REFRESH_REJECTED' } },
      })

      const adapter = vi.fn().mockRejectedValue({
        response: {
          status: 401,
          data: { code: 'TOKEN_EXPIRED', message: 'Token expired' },
        },
        config: {
          url: '/invalid-refresh',
          headers: { Authorization: 'Bearer expired-token' },
        },
        code: 'ERR_BAD_REQUEST',
      })
      apiClient.defaults.adapter = adapter

      await expect(apiClient.get('/invalid-refresh')).rejects.toEqual(
        expect.objectContaining({ code: 'TOKEN_REFRESH_FAILED' })
      )

      const { authSession } = await import('@/auth/authSession')
      expect(authSession.getSnapshot().accessToken).toBeNull()
      expect(authSession.getSnapshot().refreshToken).toBeNull()
      expect(localStorage.getItem('auth_token')).toBe('expired-token')
      expect(localStorage.getItem('refresh_token')).toBe('invalid-refresh')
      expect(localStorage.getItem('auth_session_generation')).toMatch(/^legacy-[0-9a-f]{64}$/)
      expect(sessionStorage.getItem('auth_expired')).toBe('1')
    })
  })

  // --- 网络错误 ---

  describe('网络错误', () => {
    it('网络错误返回 status 0 的错误', async () => {
      const adapter = vi.fn().mockRejectedValue({
        code: 'ERR_NETWORK',
        message: 'Network Error',
        config: { url: '/test' },
        // 没有 response
      })
      apiClient.defaults.adapter = adapter

      await expect(apiClient.get('/test')).rejects.toEqual(
        expect.objectContaining({
          status: 0,
          message: 'Network error. Please check your connection.',
        })
      )
    })
  })

  // --- 请求取消 ---

  describe('请求取消', () => {
    it('取消的请求保持原始取消错误', async () => {
      const source = axios.CancelToken.source()

      const adapter = vi.fn().mockRejectedValue(
        new axios.Cancel('Operation canceled')
      )
      apiClient.defaults.adapter = adapter

      await expect(
        apiClient.get('/test', { cancelToken: source.token })
      ).rejects.toBeDefined()
    })
  })
})

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  AUTH_SESSION_STORAGE_KEYS,
  AuthSessionChangedError,
  createAuthSession,
  type AuthSessionLockManager,
  type AuthSessionRefreshResult,
  type AuthSessionSnapshot,
} from '../authSession'

const FAMILY_RECORD_PREFIX = 'auth_session_family:'

function readHead(): Record<string, unknown> | null {
  const raw = localStorage.getItem(AUTH_SESSION_STORAGE_KEYS.head)
  return raw ? JSON.parse(raw) as Record<string, unknown> : null
}

function readFamily(generation: string | null): Record<string, unknown> | null {
  if (!generation) return null
  const raw = localStorage.getItem(`${FAMILY_RECORD_PREFIX}${generation}`)
  return raw ? JSON.parse(raw) as Record<string, unknown> : null
}

function writeCanonicalFamily(generation: string, snapshot: AuthSessionSnapshot): void {
  localStorage.setItem(`${FAMILY_RECORD_PREFIX}${generation}`, JSON.stringify({
    version: 1,
    generation,
    snapshot,
  }))
  localStorage.setItem(AUTH_SESSION_STORAGE_KEYS.generation, generation)
  if (snapshot.accessToken) localStorage.setItem('auth_token', snapshot.accessToken)
  else localStorage.removeItem('auth_token')
  if (snapshot.refreshToken) localStorage.setItem('refresh_token', snapshot.refreshToken)
  else localStorage.removeItem('refresh_token')
  if (snapshot.expiresAt !== null) {
    localStorage.setItem('token_expires_at', String(snapshot.expiresAt))
  } else {
    localStorage.removeItem('token_expires_at')
  }
  if (snapshot.user) localStorage.setItem('auth_user', JSON.stringify(snapshot.user))
  else localStorage.removeItem('auth_user')
  localStorage.setItem(AUTH_SESSION_STORAGE_KEYS.head, JSON.stringify({
    version: 1,
    state: 'active',
    generation,
  }))
}

const fakeUser = {
  id: 1,
  username: 'session-user',
  email: 'session@example.com',
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

function createSerialLockManager(): AuthSessionLockManager {
  let tail = Promise.resolve()

  return {
    async request<T>(_name: string, callback: () => Promise<T>): Promise<T> {
      const previous = tail
      let release!: () => void
      const held = new Promise<void>((resolve) => {
        release = resolve
      })
      tail = previous.then(() => held)

      await previous
      try {
        return await callback()
      } finally {
        release()
      }
    },
  }
}

function wrapStorage(onSetItem: (key: string) => void): Storage {
  return {
    get length() {
      return localStorage.length
    },
    clear() {
      localStorage.clear()
    },
    getItem(key: string) {
      return localStorage.getItem(key)
    },
    key(index: number) {
      return localStorage.key(index)
    },
    removeItem(key: string) {
      localStorage.removeItem(key)
    },
    setItem(key: string, value: string) {
      localStorage.setItem(key, value)
      onSetItem(key)
    },
  }
}

function createThrowingStorage(): Storage {
  const fail = (): never => {
    throw new Error('tab storage unavailable')
  }
  return {
    get length() {
      return fail()
    },
    clear: fail,
    getItem: fail,
    key: fail,
    removeItem: fail,
    setItem: fail,
  }
}

describe('authSession', () => {
  const sessions: Array<ReturnType<typeof createAuthSession>> = []

  beforeEach(() => {
    localStorage.clear()
    sessionStorage.clear()
  })

  afterEach(() => {
    sessions.forEach((session) => session.dispose())
    sessions.length = 0
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  function createSession(lockManager: AuthSessionLockManager | null = null) {
    const session = createAuthSession({ storage: localStorage, eventTarget: window, lockManager })
    sessions.push(session)
    return session
  }

  it('hydrates, replaces, patches, publishes and clears one persisted snapshot', () => {
    localStorage.setItem('auth_token', 'stored-access')
    localStorage.setItem('refresh_token', 'stored-refresh')
    localStorage.setItem('token_expires_at', '123456')
    localStorage.setItem('auth_user', JSON.stringify(fakeUser))
    const session = createSession()
    const subscriber = vi.fn()
    session.subscribe(subscriber)

    expect(session.hydrate()).toEqual({
      accessToken: 'stored-access',
      refreshToken: 'stored-refresh',
      expiresAt: 123456,
      user: fakeUser,
    })

    session.patch({ accessToken: 'patched-access' })
    expect(session.getSnapshot().accessToken).toBe('patched-access')
    expect(session.getSnapshot().refreshToken).toBe('stored-refresh')
    expect(localStorage.getItem('auth_token')).toBe('patched-access')

    session.replace({
      accessToken: 'replacement-access',
      refreshToken: null,
      expiresAt: null,
      user: fakeUser,
    })
    const replacementGeneration = session.getGeneration()
    expect(localStorage.getItem('refresh_token')).toBeNull()
    expect(subscriber).toHaveBeenCalled()

    session.clear()
    expect(session.getSnapshot()).toEqual({
      accessToken: null,
      refreshToken: null,
      expiresAt: null,
      user: null,
    })
    expect(localStorage.getItem('auth_user')).toBeNull()
    expect(readFamily(replacementGeneration)).toBeNull()
  })

  it('lazily migrates one legacy token pair to the same generation in every tab', () => {
    localStorage.setItem('auth_token', 'legacy-access')
    localStorage.setItem('refresh_token', 'legacy-refresh')
    localStorage.setItem('token_expires_at', '123456')
    localStorage.setItem('auth_user', JSON.stringify(fakeUser))

    const first = createSession()
    const second = createSession()
    first.hydrate()
    const firstGeneration = first.getGeneration()
    second.hydrate()

    expect(firstGeneration).toBe(
      'legacy-70f2ae333e93d5c296ac62c4455971a83534da771350f20ff0d334c4f665a345',
    )
    expect(second.getGeneration()).toBe(firstGeneration)
    expect(localStorage.getItem('auth_session_generation')).toBe(firstGeneration)
    expect(readHead()).toEqual({
      version: 1,
      state: 'active',
      generation: firstGeneration,
    })
    expect(readFamily(firstGeneration)).toEqual({
      version: 1,
      generation: firstGeneration,
      snapshot: {
        accessToken: 'legacy-access',
        refreshToken: 'legacy-refresh',
        expiresAt: 123456,
        user: fakeUser,
      },
    })
  })

  it.each([
    ['malformed JSON', '{not-json'],
    ['unsupported version', JSON.stringify({ version: 2, state: 'cleared' })],
    ['invalid schema', JSON.stringify({ version: 1, state: 'active', generation: '' })],
  ])('fails closed for a present canonical head with %s', (_label, rawHead) => {
    localStorage.setItem('auth_token', 'legacy-access')
    localStorage.setItem('refresh_token', 'legacy-refresh')
    localStorage.setItem('auth_user', JSON.stringify(fakeUser))
    localStorage.setItem(AUTH_SESSION_STORAGE_KEYS.head, rawHead)
    const session = createSession()

    expect(session.hydrate()).toEqual({
      accessToken: null,
      refreshToken: null,
      expiresAt: null,
      user: null,
    })
    expect(session.getGeneration()).toBeNull()
    expect(localStorage.getItem(AUTH_SESSION_STORAGE_KEYS.head)).toBe(rawHead)
    expect(localStorage.getItem('auth_session_generation')).toBeNull()
  })

  it('fails closed when an active head points to a malformed family record', () => {
    localStorage.setItem('auth_token', 'legacy-access')
    localStorage.setItem('refresh_token', 'legacy-refresh')
    localStorage.setItem('auth_user', JSON.stringify(fakeUser))
    localStorage.setItem('auth_session_generation', 'broken-family')
    localStorage.setItem(AUTH_SESSION_STORAGE_KEYS.head, JSON.stringify({
      version: 1,
      state: 'active',
      generation: 'broken-family',
    }))
    localStorage.setItem(`${FAMILY_RECORD_PREFIX}broken-family`, '{not-json')
    const session = createSession()

    expect(session.hydrate()).toEqual({
      accessToken: null,
      refreshToken: null,
      expiresAt: null,
      user: null,
    })
    expect(session.getGeneration()).toBeNull()

    session.clear()
    expect(readHead()).toEqual(expect.objectContaining({ version: 1, state: 'cleared' }))
    expect(readFamily('broken-family')).toBeNull()
  })

  it('fails closed when the head is missing but a protocol generation mirror remains', () => {
    localStorage.setItem('auth_token', 'residual-access')
    localStorage.setItem('refresh_token', 'residual-refresh')
    localStorage.setItem('auth_user', JSON.stringify(fakeUser))
    localStorage.setItem('auth_session_generation', 'orphan-generation')
    const session = createSession()

    expect(session.hydrate()).toEqual({
      accessToken: null,
      refreshToken: null,
      expiresAt: null,
      user: null,
    })
    expect(session.getGeneration()).toBeNull()
    expect(localStorage.getItem(AUTH_SESSION_STORAGE_KEYS.head)).toBeNull()
  })

  it('never authenticates a torn legacy mirror while canonical patch commit is pending', () => {
    let interceptAccessWrite = false
    let tornSnapshot: ReturnType<ReturnType<typeof createAuthSession>['getSnapshot']> | null = null
    const storage = wrapStorage((key) => {
      if (interceptAccessWrite && key === 'auth_token' && tornSnapshot === null) {
        tornSnapshot = observer.hydrate()
      }
    })
    const writer = createAuthSession({
      storage,
      tabStorage: null,
      eventTarget: null,
      lockManager: null,
    })
    const observer = createAuthSession({
      storage,
      tabStorage: null,
      eventTarget: null,
      lockManager: null,
    })
    sessions.push(writer, observer)

    writer.replace({
      accessToken: 'before-access',
      refreshToken: 'before-refresh',
      expiresAt: Date.now() - 1,
      user: fakeUser,
    })
    const generation = writer.getGeneration()
    observer.hydrate()

    interceptAccessWrite = true
    writer.patch({
      accessToken: 'after-access',
      refreshToken: 'after-refresh',
      expiresAt: Date.now() + 900_000,
    })
    interceptAccessWrite = false

    expect(tornSnapshot).toEqual({
      accessToken: null,
      refreshToken: null,
      expiresAt: null,
      user: null,
    })
    expect(observer.hydrate()).toEqual(expect.objectContaining({
      accessToken: 'after-access',
      refreshToken: 'after-refresh',
    }))
    expect(observer.getGeneration()).toBe(generation)
  })

  it('keeps one family record for patch and switches the head only for replace', () => {
    const session = createSession()
    session.replace({
      accessToken: 'first-access',
      refreshToken: 'first-refresh',
      expiresAt: Date.now() + 900_000,
      user: fakeUser,
    })
    const firstGeneration = session.getGeneration()
    const firstHead = localStorage.getItem(AUTH_SESSION_STORAGE_KEYS.head)

    session.patch({ accessToken: 'first-rotated-access' })

    expect(session.getGeneration()).toBe(firstGeneration)
    expect(localStorage.getItem(AUTH_SESSION_STORAGE_KEYS.head)).toBe(firstHead)
    expect(readFamily(firstGeneration)).toEqual(expect.objectContaining({
      generation: firstGeneration,
      snapshot: expect.objectContaining({ accessToken: 'first-rotated-access' }),
    }))
    expect(Array.from({ length: localStorage.length }, (_, index) => localStorage.key(index))
      .filter((key) => key?.startsWith(FAMILY_RECORD_PREFIX))).toHaveLength(1)

    session.replace({
      accessToken: 'second-access',
      refreshToken: 'second-refresh',
      expiresAt: Date.now() + 1_800_000,
      user: fakeUser,
    })
    const secondGeneration = session.getGeneration()

    expect(secondGeneration).not.toBe(firstGeneration)
    expect(readHead()).toEqual({
      version: 1,
      state: 'active',
      generation: secondGeneration,
    })
    expect(readFamily(firstGeneration)).toBeNull()
    expect(readFamily(secondGeneration)).toEqual(expect.objectContaining({
      generation: secondGeneration,
      snapshot: expect.objectContaining({ accessToken: 'second-access' }),
    }))
    expect(Array.from({ length: localStorage.length }, (_, index) => localStorage.key(index))
      .filter((key) => key?.startsWith(FAMILY_RECORD_PREFIX))).toHaveLength(1)
  })

  it('removes its unreferenced family when another head wins replace', () => {
    let replaceHead = false
    let losingGeneration: string | null = null
    const winnerGeneration = 'replace-winner-generation'
    const winnerSnapshot: AuthSessionSnapshot = {
      accessToken: 'replace-winner-access',
      refreshToken: 'replace-winner-refresh',
      expiresAt: Date.now() + 900_000,
      user: { ...fakeUser, id: 2, username: 'replace-winner-user' },
    }
    const storage = wrapStorage((key) => {
      if (!replaceHead || key !== AUTH_SESSION_STORAGE_KEYS.head) return
      replaceHead = false
      losingGeneration = (readHead()?.generation as string | undefined) ?? null
      writeCanonicalFamily(winnerGeneration, winnerSnapshot)
    })
    const writer = createAuthSession({
      storage,
      tabStorage: null,
      eventTarget: null,
      lockManager: null,
    })
    sessions.push(writer)

    replaceHead = true
    expect(() => writer.replace({
      accessToken: 'replace-loser-access',
      refreshToken: 'replace-loser-refresh',
      expiresAt: Date.now() + 450_000,
      user: fakeUser,
    })).toThrow(AuthSessionChangedError)

    expect(losingGeneration).not.toBeNull()
    expect(readFamily(losingGeneration)).toBeNull()
    expect(readHead()).toEqual({
      version: 1,
      state: 'active',
      generation: winnerGeneration,
    })
    expect(readFamily(winnerGeneration)).not.toBeNull()
  })

  it('rejects a stale patch without moving a head replaced by another tab', () => {
    let replaceDuringPatch = false
    const storage = wrapStorage((key) => {
      if (!replaceDuringPatch || key !== 'auth_token') return
      replaceDuringPatch = false
      writeCanonicalFamily('new-family-generation', {
        accessToken: 'new-family-access',
        refreshToken: 'new-family-refresh',
        expiresAt: Date.now() + 900_000,
        user: { ...fakeUser, id: 2, username: 'new-family-user' },
      })
    })
    const writer = createAuthSession({
      storage,
      tabStorage: null,
      eventTarget: null,
      lockManager: null,
    })
    const peer = createAuthSession({
      storage: localStorage,
      tabStorage: null,
      eventTarget: null,
      lockManager: null,
    })
    sessions.push(writer, peer)
    writer.replace({
      accessToken: 'old-family-access',
      refreshToken: 'old-family-refresh',
      expiresAt: Date.now() - 1,
      user: fakeUser,
    })
    const oldGeneration = writer.getGeneration()

    replaceDuringPatch = true
    expect(() => writer.patch({ accessToken: 'stale-patch-access' }))
      .toThrow(AuthSessionChangedError)

    const newGeneration = peer.getGeneration()
    expect(newGeneration).not.toBe(oldGeneration)
    expect(readFamily(oldGeneration)).toBeNull()
    expect(readHead()).toEqual({
      version: 1,
      state: 'active',
      generation: newGeneration,
    })
    expect(peer.hydrate()).toEqual(expect.objectContaining({
      accessToken: 'new-family-access',
      refreshToken: 'new-family-refresh',
      user: expect.objectContaining({ id: 2 }),
    }))
  })

  it('commits a tombstone before cleanup so residual mirrors cannot resurrect', () => {
    let denyCleanup = false
    const storage: Storage = {
      get length() {
        return localStorage.length
      },
      clear() {
        localStorage.clear()
      },
      getItem(key: string) {
        return localStorage.getItem(key)
      },
      key(index: number) {
        return localStorage.key(index)
      },
      removeItem(key: string) {
        if (denyCleanup && key !== AUTH_SESSION_STORAGE_KEYS.head) {
          throw new Error('cleanup denied')
        }
        localStorage.removeItem(key)
      },
      setItem(key: string, value: string) {
        localStorage.setItem(key, value)
      },
    }
    const writer = createAuthSession({ storage, tabStorage: null, eventTarget: null })
    const observer = createAuthSession({
      storage: localStorage,
      tabStorage: null,
      eventTarget: null,
      lockManager: null,
    })
    sessions.push(writer, observer)
    writer.replace({
      accessToken: 'residual-access',
      refreshToken: 'residual-refresh',
      expiresAt: Date.now() + 900_000,
      user: fakeUser,
    })
    const clearedGeneration = writer.getGeneration()

    denyCleanup = true
    writer.clear()

    expect(readHead()).toEqual(expect.objectContaining({ version: 1, state: 'cleared' }))
    expect(localStorage.getItem('auth_token')).toBe('residual-access')
    expect(readFamily(clearedGeneration)).not.toBeNull()
    expect(observer.hydrate()).toEqual({
      accessToken: null,
      refreshToken: null,
      expiresAt: null,
      user: null,
    })
  })

  it('does not let legacy migration overwrite a tombstone committed mid-migration', () => {
    localStorage.setItem('auth_token', 'legacy-access')
    localStorage.setItem('refresh_token', 'legacy-refresh')
    localStorage.setItem('auth_user', JSON.stringify(fakeUser))
    const clearer = createAuthSession({
      storage: localStorage,
      tabStorage: null,
      eventTarget: null,
      lockManager: null,
    })
    let clearDuringMigration = true
    let migratingGeneration: string | null = null
    const storage = wrapStorage((key) => {
      if (!clearDuringMigration || key !== AUTH_SESSION_STORAGE_KEYS.generation) return
      clearDuringMigration = false
      migratingGeneration = localStorage.getItem(AUTH_SESSION_STORAGE_KEYS.generation)
      clearer.clear()
    })
    const migrating = createAuthSession({
      storage,
      tabStorage: null,
      eventTarget: null,
      lockManager: null,
    })
    sessions.push(clearer, migrating)

    expect(migrating.hydrate()).toEqual({
      accessToken: null,
      refreshToken: null,
      expiresAt: null,
      user: null,
    })
    expect(readHead()).toEqual(expect.objectContaining({ version: 1, state: 'cleared' }))
    expect(migrating.getGeneration()).toBeNull()
    expect(migratingGeneration).not.toBeNull()
    expect(readFamily(migratingGeneration)).toBeNull()
  })

  it('synchronizes token rotation and logout from storage events', async () => {
    const session = createSession()
    session.hydrate()
    const subscriber = vi.fn()
    session.subscribe(subscriber)

    localStorage.setItem('auth_token', 'other-tab-access')
    localStorage.setItem('refresh_token', 'other-tab-refresh')
    localStorage.setItem('auth_user', JSON.stringify(fakeUser))
    window.dispatchEvent(new StorageEvent('storage', { key: 'auth_token' }))
    await Promise.resolve()

    expect(session.getSnapshot()).toEqual(expect.objectContaining({
      accessToken: 'other-tab-access',
      refreshToken: 'other-tab-refresh',
      user: fakeUser,
    }))

    localStorage.clear()
    window.dispatchEvent(new StorageEvent('storage', { key: null }))
    await Promise.resolve()
    expect(session.getSnapshot().accessToken).toBeNull()
    expect(subscriber).toHaveBeenCalledTimes(2)
  })

  it('deduplicates concurrent refresh callers into one request and one rotation', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-07-22T00:00:00Z'))
    const session = createSession()
    session.replace({
      accessToken: 'expired-access',
      refreshToken: 'refresh-once',
      expiresAt: Date.now() - 1,
      user: fakeUser,
    })

    let resolveRefresh!: (value: AuthSessionRefreshResult) => void
    const requester = vi.fn(() => new Promise<AuthSessionRefreshResult>((resolve) => {
      resolveRefresh = resolve
    }))

    const first = session.refresh(requester)
    const second = session.refresh(requester)
    expect(requester).toHaveBeenCalledTimes(1)
    expect(requester).toHaveBeenCalledWith('refresh-once')

    resolveRefresh({
      access_token: 'rotated-access',
      refresh_token: 'rotated-refresh',
      expires_in: 900,
    })

    const [firstResult, secondResult] = await Promise.all([first, second])
    expect(firstResult).toEqual(secondResult)
    expect(firstResult).toEqual(expect.objectContaining({
      accessToken: 'rotated-access',
      refreshToken: 'rotated-refresh',
      expiresAt: Date.now() + 900_000,
    }))
    expect(localStorage.getItem('auth_token')).toBe('rotated-access')
  })

  it('does not resurrect a session cleared while refresh is in flight', async () => {
    const session = createSession()
    session.replace({
      accessToken: 'expired-access',
      refreshToken: 'refresh-once',
      expiresAt: Date.now() - 1,
      user: fakeUser,
    })

    let resolveRefresh!: (value: AuthSessionRefreshResult) => void
    const pending = session.refresh(() => new Promise((resolve) => {
      resolveRefresh = resolve
    }))
    session.clear()
    resolveRefresh({ access_token: 'stale-access', refresh_token: 'stale-refresh' })

    await expect(pending).rejects.toBeInstanceOf(AuthSessionChangedError)
    expect(session.getSnapshot().accessToken).toBeNull()
  })

  it('does not replay an old refresh caller into a newly logged-in principal', async () => {
    const session = createSession()
    session.replace({
      accessToken: 'old-access',
      refreshToken: 'old-refresh',
      expiresAt: Date.now() - 1,
      user: fakeUser,
    })

    let resolveRefresh!: (value: AuthSessionRefreshResult) => void
    const pending = session.refresh(() => new Promise((resolve) => {
      resolveRefresh = resolve
    }))

    session.replace({
      accessToken: 'new-access',
      refreshToken: 'new-refresh',
      expiresAt: Date.now() + 900_000,
      user: { ...fakeUser, id: 2, username: 'different-user' },
    })
    resolveRefresh({ access_token: 'stale-access', refresh_token: 'stale-refresh' })

    await expect(pending).rejects.toBeInstanceOf(AuthSessionChangedError)
    expect(session.getSnapshot()).toEqual(expect.objectContaining({
      accessToken: 'new-access',
      refreshToken: 'new-refresh',
      user: expect.objectContaining({ id: 2 }),
    }))
  })

  it('reuses a same-user token rotation completed by another tab', async () => {
    const session = createSession()
    session.replace({
      accessToken: 'old-access',
      refreshToken: 'old-refresh',
      expiresAt: Date.now() - 1,
      user: fakeUser,
    })

    let resolveRefresh!: (value: AuthSessionRefreshResult) => void
    const pending = session.refresh(() => new Promise((resolve) => {
      resolveRefresh = resolve
    }))

    session.patch({
      accessToken: 'other-tab-access',
      refreshToken: 'other-tab-refresh',
      expiresAt: Date.now() + 900_000,
      user: { ...fakeUser, username: 'session-user-updated' },
    })
    resolveRefresh({ access_token: 'stale-access', refresh_token: 'stale-refresh' })

    await expect(pending).resolves.toEqual(expect.objectContaining({
      accessToken: 'other-tab-access',
      refreshToken: 'other-tab-refresh',
    }))
  })

  it('keeps a same-user storage rotation that arrives while refresh is in flight', async () => {
    const session = createSession()
    session.replace({
      accessToken: 'old-access',
      refreshToken: 'old-refresh',
      expiresAt: Date.now() - 1,
      user: fakeUser,
    })
    const peerSession = createSession()
    peerSession.hydrate()

    let resolveRefresh!: (value: AuthSessionRefreshResult) => void
    const pending = session.refresh(() => new Promise((resolve) => {
      resolveRefresh = resolve
    }))

    peerSession.patch({
      accessToken: 'other-tab-access',
      refreshToken: 'other-tab-refresh',
      expiresAt: Date.now() + 900_000,
      user: fakeUser,
    })
    window.dispatchEvent(new StorageEvent('storage', { key: 'auth_token' }))
    await Promise.resolve()

    resolveRefresh({ access_token: 'stale-access', refresh_token: 'stale-refresh' })

    await expect(pending).resolves.toEqual(expect.objectContaining({
      accessToken: 'other-tab-access',
      refreshToken: 'other-tab-refresh',
    }))
    expect(session.getSnapshot().accessToken).toBe('other-tab-access')
    expect(localStorage.getItem('auth_token')).toBe('other-tab-access')
  })

  it('reuses another tab rotation when this tab requester fails with 401', async () => {
    const session = createSession()
    session.replace({
      accessToken: 'old-access',
      refreshToken: 'old-refresh',
      expiresAt: Date.now() - 1,
      user: fakeUser,
    })
    const peerSession = createSession()
    peerSession.hydrate()

    let rejectRefresh!: (reason: unknown) => void
    const pending = session.refresh(() => new Promise((_resolve, reject) => {
      rejectRefresh = reject
    }))

    peerSession.patch({
      accessToken: 'other-tab-access',
      refreshToken: 'other-tab-refresh',
      expiresAt: Date.now() + 900_000,
      user: fakeUser,
    })
    window.dispatchEvent(new StorageEvent('storage', { key: 'auth_token' }))
    await Promise.resolve()

    rejectRefresh({ response: { status: 401 } })

    await expect(pending).resolves.toEqual(expect.objectContaining({
      accessToken: 'other-tab-access',
      refreshToken: 'other-tab-refresh',
    }))
  })

  it('does not clear a newly logged-in principal when a stale refresh fails late', async () => {
    const session = createSession()
    session.replace({
      accessToken: 'old-access',
      refreshToken: 'old-refresh',
      expiresAt: Date.now() - 1,
      user: fakeUser,
    })

    let rejectRefresh!: (reason: unknown) => void
    const pending = session.refresh(() => new Promise((_resolve, reject) => {
      rejectRefresh = reject
    }))

    session.replace({
      accessToken: 'new-user-access',
      refreshToken: 'new-user-refresh',
      expiresAt: Date.now() + 900_000,
      user: { ...fakeUser, id: 2, username: 'new-user' },
    })
    rejectRefresh({ response: { status: 401 } })

    await expect(pending).rejects.toBeInstanceOf(AuthSessionChangedError)
    expect(session.getSnapshot()).toEqual(expect.objectContaining({
      accessToken: 'new-user-access',
      refreshToken: 'new-user-refresh',
      user: expect.objectContaining({ id: 2 }),
    }))
  })

  it('rejects a stale refresh after logout and same-user login starts a new family', async () => {
    const session = createSession()
    session.replace({
      accessToken: 'old-access',
      refreshToken: 'old-refresh',
      expiresAt: Date.now() - 1,
      user: fakeUser,
    })
    const oldGeneration = session.getGeneration()

    let resolveRefresh!: (value: AuthSessionRefreshResult) => void
    const pending = session.refresh(() => new Promise((resolve) => {
      resolveRefresh = resolve
    }))

    session.clear()
    session.replace({
      accessToken: 'new-access',
      refreshToken: 'new-refresh',
      expiresAt: Date.now() + 900_000,
      user: { ...fakeUser },
    })
    const newGeneration = session.getGeneration()
    expect(newGeneration).not.toBe(oldGeneration)

    resolveRefresh({ access_token: 'stale-access', refresh_token: 'stale-refresh' })

    await expect(pending).rejects.toBeInstanceOf(AuthSessionChangedError)
    expect(session.getSnapshot()).toEqual(expect.objectContaining({
      accessToken: 'new-access',
      refreshToken: 'new-refresh',
      user: expect.objectContaining({ id: fakeUser.id }),
    }))
    expect(session.getGeneration()).toBe(newGeneration)
  })

  it('preserves generation for refresh patches and rotates it for explicit replace', () => {
    const session = createSession()
    session.replace({
      accessToken: 'first-access',
      refreshToken: 'first-refresh',
      expiresAt: Date.now() - 1,
      user: fakeUser,
    })
    const firstGeneration = session.getGeneration()

    session.patch({ accessToken: 'rotated-access', refreshToken: 'rotated-refresh' })
    expect(session.getGeneration()).toBe(firstGeneration)
    expect(localStorage.getItem('auth_session_generation')).toBe(firstGeneration)

    session.replace({
      accessToken: 'second-login-access',
      refreshToken: 'second-login-refresh',
      expiresAt: Date.now() + 900_000,
      user: fakeUser,
    })
    expect(session.getGeneration()).not.toBe(firstGeneration)
  })

  it('tab invalidation never deletes a newer durable login family', () => {
    const session = createSession()
    session.replace({
      accessToken: 'old-access',
      refreshToken: 'old-refresh',
      expiresAt: Date.now() - 1,
      user: fakeUser,
    })
    const oldGeneration = session.getGeneration()

    session.replace({
      accessToken: 'new-login-access',
      refreshToken: 'new-login-refresh',
      expiresAt: Date.now() + 900_000,
      user: fakeUser,
    })
    const newGeneration = session.getGeneration()

    expect(session.invalidate(oldGeneration)).toBe(false)
    expect(session.getSnapshot().accessToken).toBe('new-login-access')
    expect(localStorage.getItem('auth_token')).toBe('new-login-access')
    expect(localStorage.getItem('refresh_token')).toBe('new-login-refresh')
    expect(localStorage.getItem('auth_session_generation')).toBe(newGeneration)
  })

  it.each([
    ['without tab storage', null],
    ['when tab storage throws', createThrowingStorage()],
  ])('keeps an invalidated family empty across storage events %s', async (_label, tabStorage) => {
    const session = createAuthSession({
      storage: localStorage,
      tabStorage,
      eventTarget: window,
      lockManager: null,
    })
    const peer = createAuthSession({
      storage: localStorage,
      tabStorage: null,
      eventTarget: null,
      lockManager: null,
    })
    sessions.push(session, peer)
    session.replace({
      accessToken: 'invalidated-access',
      refreshToken: null,
      expiresAt: Date.now() - 1,
      user: fakeUser,
    })
    peer.hydrate()
    const invalidatedFamily = session.getGeneration()

    expect(session.invalidate(invalidatedFamily)).toBe(true)
    window.dispatchEvent(new StorageEvent('storage', {
      key: AUTH_SESSION_STORAGE_KEYS.head,
    }))
    await Promise.resolve()
    expect(session.getSnapshot().accessToken).toBeNull()

    peer.patch({ accessToken: 'same-family-rotation' })
    window.dispatchEvent(new StorageEvent('storage', {
      key: `${FAMILY_RECORD_PREFIX}${invalidatedFamily}`,
    }))
    await Promise.resolve()
    expect(session.getSnapshot().accessToken).toBeNull()
    expect(session.getGeneration()).toBeNull()

    peer.replace({
      accessToken: 'new-family-access',
      refreshToken: 'new-family-refresh',
      expiresAt: Date.now() + 900_000,
      user: fakeUser,
    })
    const newGeneration = peer.getGeneration()
    window.dispatchEvent(new StorageEvent('storage', {
      key: AUTH_SESSION_STORAGE_KEYS.head,
    }))
    await Promise.resolve()

    expect(newGeneration).not.toBe(invalidatedFamily)
    expect(session.getGeneration()).toBe(newGeneration)
    expect(session.getSnapshot()).toEqual(expect.objectContaining({
      accessToken: 'new-family-access',
      refreshToken: 'new-family-refresh',
    }))
  })

  it('serializes independent sessions with one shared lock and reuses one rotation', async () => {
    const lockManager = createSerialLockManager()
    const firstSession = createSession(lockManager)
    firstSession.replace({
      accessToken: 'expired-access',
      refreshToken: 'shared-refresh',
      expiresAt: Date.now() - 1,
      user: fakeUser,
    })
    const secondSession = createSession(lockManager)

    let resolveRefresh!: (value: AuthSessionRefreshResult) => void
    const requester = vi.fn(() => new Promise<AuthSessionRefreshResult>((resolve) => {
      resolveRefresh = resolve
    }))

    const first = firstSession.refresh(requester)
    const second = secondSession.refresh(requester)
    await vi.waitFor(() => expect(requester).toHaveBeenCalledTimes(1))

    resolveRefresh({
      access_token: 'shared-rotated-access',
      refresh_token: 'shared-rotated-refresh',
      expires_in: 900,
    })

    const [firstResult, secondResult] = await Promise.all([first, second])
    expect(requester).toHaveBeenCalledTimes(1)
    expect(firstResult).toEqual(secondResult)
    expect(firstResult).toEqual(expect.objectContaining({
      accessToken: 'shared-rotated-access',
      refreshToken: 'shared-rotated-refresh',
    }))
    expect(secondSession.getSnapshot()).toEqual(firstResult)
  })

  it('keeps its in-memory snapshot when storage is unavailable and hydrate is called', () => {
    const session = createAuthSession({ storage: null, eventTarget: null, lockManager: null })
    sessions.push(session)
    session.replace({
      accessToken: 'memory-access',
      refreshToken: 'memory-refresh',
      expiresAt: 123456,
      user: fakeUser,
    })

    expect(session.hydrate()).toEqual(expect.objectContaining({
      accessToken: 'memory-access',
      refreshToken: 'memory-refresh',
      user: fakeUser,
    }))
  })
})

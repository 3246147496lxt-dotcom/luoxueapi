import type { User } from '@/types'

export const AUTH_SESSION_STORAGE_KEYS = {
  accessToken: 'auth_token',
  refreshToken: 'refresh_token',
  expiresAt: 'token_expires_at',
  user: 'auth_user',
  generation: 'auth_session_generation',
  head: 'auth_session_head',
} as const

const AUTH_SESSION_INVALIDATED_GENERATION_KEY = 'auth_session_invalidated_generation'
const AUTH_SESSION_FAMILY_RECORD_PREFIX = 'auth_session_family:'

export type AuthSessionUser = User & { run_mode?: 'standard' | 'simple' }

export interface AuthSessionSnapshot {
  accessToken: string | null
  refreshToken: string | null
  expiresAt: number | null
  user: AuthSessionUser | null
}

export type AuthSessionPatch = Partial<AuthSessionSnapshot>

export interface AuthSessionInvalidationExpectation {
  generation: string | null
  accessToken: string | null
  refreshToken: string | null
  userId: number | null
}

export interface AuthSessionRefreshResult {
  access_token: string
  refresh_token?: string
  expires_in?: number
}

export type AuthSessionRefreshRequester = (
  refreshToken: string,
) => Promise<AuthSessionRefreshResult>

export type AuthSessionSubscriber = (snapshot: AuthSessionSnapshot) => void

export interface AuthSessionLockManager {
  request<T>(name: string, callback: () => Promise<T>): Promise<T>
}

export interface AuthSessionOptions {
  storage?: Storage | null
  tabStorage?: Storage | null
  eventTarget?: Pick<Window, 'addEventListener' | 'removeEventListener'> | null
  lockManager?: AuthSessionLockManager | null
}

interface StoredAuthSessionFamily {
  version: 1
  generation: string
  snapshot: AuthSessionSnapshot
}

type StoredAuthSessionHead =
  | { version: 1, state: 'active', generation: string }
  | { version: 1, state: 'cleared', tombstone: string }

type StoredSessionAuthority =
  | { kind: 'active', generation: string, snapshot: AuthSessionSnapshot }
  | { kind: 'cleared' }
  | {
    kind: 'legacy'
    generationPresent: boolean
    snapshot: AuthSessionSnapshot
    corrupt: boolean
  }
  | { kind: 'invalid' }

const EMPTY_SESSION: AuthSessionSnapshot = {
  accessToken: null,
  refreshToken: null,
  expiresAt: null,
  user: null,
}

const SESSION_STORAGE_KEY_SET = new Set<string>(Object.values(AUTH_SESSION_STORAGE_KEYS))
const AUTH_SESSION_REFRESH_LOCK_NAME = 'luoxueapi-auth-session-refresh'

function copySnapshot(snapshot: AuthSessionSnapshot): AuthSessionSnapshot {
  return { ...snapshot }
}

function normalizeToken(value: unknown): string | null {
  return typeof value === 'string' && value.trim() ? value.trim() : null
}

function normalizeExpiresAt(value: unknown): number | null {
  if (value === null || value === undefined || value === '') return null
  const parsed = typeof value === 'number' ? value : Number(value)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : null
}

function normalizeGeneration(value: unknown): string | null {
  return typeof value === 'string' && value.trim() ? value.trim() : null
}

function createSessionGeneration(): string {
  try {
    if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
      return crypto.randomUUID()
    }
    if (typeof crypto !== 'undefined' && typeof crypto.getRandomValues === 'function') {
      const bytes = crypto.getRandomValues(new Uint8Array(16))
      return Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('')
    }
  } catch {
    // Fall through to a non-secret uniqueness fallback for restricted browsers.
  }
  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`
}

const SHA256_ROUND_CONSTANTS = [
  0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5,
  0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
  0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3,
  0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
  0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc,
  0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
  0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7,
  0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
  0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13,
  0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
  0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3,
  0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
  0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5,
  0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
  0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208,
  0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
] as const

function rotateRight(value: number, bits: number): number {
  return (value >>> bits) | (value << (32 - bits))
}

function sha256Hex(value: string): string {
  const input = new TextEncoder().encode(value)
  const paddedLength = Math.ceil((input.length + 9) / 64) * 64
  const bytes = new Uint8Array(paddedLength)
  bytes.set(input)
  bytes[input.length] = 0x80
  const view = new DataView(bytes.buffer)
  const bitLength = input.length * 8
  view.setUint32(paddedLength - 8, Math.floor(bitLength / 0x1_0000_0000), false)
  view.setUint32(paddedLength - 4, bitLength >>> 0, false)

  const state = new Uint32Array([
    0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a,
    0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19,
  ])
  const schedule = new Uint32Array(64)

  for (let offset = 0; offset < paddedLength; offset += 64) {
    for (let index = 0; index < 16; index += 1) {
      schedule[index] = view.getUint32(offset + index * 4, false)
    }
    for (let index = 16; index < 64; index += 1) {
      const previous15 = schedule[index - 15]
      const previous2 = schedule[index - 2]
      const sigma0 = rotateRight(previous15, 7)
        ^ rotateRight(previous15, 18)
        ^ (previous15 >>> 3)
      const sigma1 = rotateRight(previous2, 17)
        ^ rotateRight(previous2, 19)
        ^ (previous2 >>> 10)
      schedule[index] = (
        schedule[index - 16]
        + sigma0
        + schedule[index - 7]
        + sigma1
      ) >>> 0
    }

    let [a, b, c, d, e, f, g, h] = state
    for (let index = 0; index < 64; index += 1) {
      const sum1 = rotateRight(e, 6) ^ rotateRight(e, 11) ^ rotateRight(e, 25)
      const choice = (e & f) ^ (~e & g)
      const temp1 = (h + sum1 + choice + SHA256_ROUND_CONSTANTS[index] + schedule[index]) >>> 0
      const sum0 = rotateRight(a, 2) ^ rotateRight(a, 13) ^ rotateRight(a, 22)
      const majority = (a & b) ^ (a & c) ^ (b & c)
      const temp2 = (sum0 + majority) >>> 0
      h = g
      g = f
      f = e
      e = (d + temp1) >>> 0
      d = c
      c = b
      b = a
      a = (temp1 + temp2) >>> 0
    }

    state[0] = (state[0] + a) >>> 0
    state[1] = (state[1] + b) >>> 0
    state[2] = (state[2] + c) >>> 0
    state[3] = (state[3] + d) >>> 0
    state[4] = (state[4] + e) >>> 0
    state[5] = (state[5] + f) >>> 0
    state[6] = (state[6] + g) >>> 0
    state[7] = (state[7] + h) >>> 0
  }

  return Array.from(state, (part) => part.toString(16).padStart(8, '0')).join('')
}

function serializeSessionMaterial(snapshot: AuthSessionSnapshot): string {
  const value = JSON.stringify([
    snapshot.accessToken,
    snapshot.refreshToken,
    snapshot.user?.id ?? null,
  ])
  return value
}

function createLegacySessionGeneration(snapshot: AuthSessionSnapshot): string {
  // Older releases persisted no trustworthy binding. Derive the same opaque
  // SHA-256 fence in every tab, so simultaneous lazy migration cannot
  // manufacture competing families for one normalized token tuple.
  return `legacy-${sha256Hex(serializeSessionMaterial(snapshot))}`
}

function hasSessionMaterial(snapshot: AuthSessionSnapshot): boolean {
  return Boolean(snapshot.accessToken || snapshot.refreshToken || snapshot.user)
}

function usersEqual(left: AuthSessionUser | null, right: AuthSessionUser | null): boolean {
  if (left === right) return true
  try {
    return JSON.stringify(left) === JSON.stringify(right)
  } catch {
    return false
  }
}

function sessionsEqual(left: AuthSessionSnapshot, right: AuthSessionSnapshot): boolean {
  return left.accessToken === right.accessToken
    && left.refreshToken === right.refreshToken
    && left.expiresAt === right.expiresAt
    && usersEqual(left.user, right.user)
}

function isSameAuthenticatedPrincipal(
  left: AuthSessionSnapshot,
  right: AuthSessionSnapshot,
): boolean {
  return left.user !== null
    && right.user !== null
    && left.user.id === right.user.id
}

function resolveBrowserStorage(): Storage | null {
  try {
    return typeof window === 'undefined' ? null : window.localStorage
  } catch {
    return null
  }
}

function resolveBrowserTabStorage(): Storage | null {
  try {
    return typeof window === 'undefined' ? null : window.sessionStorage
  } catch {
    return null
  }
}

function resolveBrowserEventTarget(): Pick<Window, 'addEventListener' | 'removeEventListener'> | null {
  return typeof window === 'undefined' ? null : window
}

function resolveBrowserLockManager(): AuthSessionLockManager | null {
  try {
    if (typeof navigator === 'undefined' || !navigator.locks) return null
    return {
      request: <T>(name: string, callback: () => Promise<T>) => (
        navigator.locks.request(name, () => callback()) as Promise<T>
      ),
    }
  } catch {
    return null
  }
}

export class AuthSessionChangedError extends Error {
  constructor() {
    super('Authentication session changed while token refresh was in flight')
    this.name = 'AuthSessionChangedError'
  }
}

export function isAuthSessionChangedError(error: unknown): boolean {
  if (error instanceof AuthSessionChangedError) return true
  if (!error || typeof error !== 'object') return false
  return (error as { code?: unknown }).code === 'AUTH_SESSION_CHANGED'
}

/**
 * Framework-neutral owner of the browser authentication session.
 *
 * All access/refresh token reads and writes go through this object. Pinia and
 * HTTP clients subscribe to it instead of keeping independent token copies.
 */
export function createAuthSession(options: AuthSessionOptions = {}) {
  const storage = options.storage === undefined ? resolveBrowserStorage() : options.storage
  const tabStorage = options.tabStorage === undefined
    ? resolveBrowserTabStorage()
    : options.tabStorage
  const eventTarget = options.eventTarget === undefined
    ? resolveBrowserEventTarget()
    : options.eventTarget
  const lockManager = options.lockManager === undefined
    ? resolveBrowserLockManager()
    : options.lockManager

  let snapshot = copySnapshot(EMPTY_SESSION)
  let hydrated = false
  let listeningForStorage = false
  let storageSyncScheduled = false
  let familyGeneration: string | null = null
  let invalidatedGeneration: string | null = null
  let refreshMutationVersion = 0
  let refreshFlight: Promise<AuthSessionSnapshot> | null = null
  const subscribers = new Set<AuthSessionSubscriber>()

  const notify = () => {
    const current = copySnapshot(snapshot)
    subscribers.forEach((subscriber) => {
      try {
        subscriber(current)
      } catch (error) {
        console.error('Auth session subscriber failed:', error)
      }
    })
  }

  const applySnapshot = (
    next: AuthSessionSnapshot,
    nextGeneration: string | null,
    shouldNotify = true,
  ) => {
    const normalized: AuthSessionSnapshot = {
      accessToken: normalizeToken(next.accessToken),
      refreshToken: normalizeToken(next.refreshToken),
      expiresAt: normalizeExpiresAt(next.expiresAt),
      user: next.user ?? null,
    }
    const changed = !sessionsEqual(snapshot, normalized)
    const refreshIdentityChanged = snapshot.accessToken !== normalized.accessToken
      || snapshot.refreshToken !== normalized.refreshToken
      || (snapshot.user?.id ?? null) !== (normalized.user?.id ?? null)
      || familyGeneration !== nextGeneration

    snapshot = normalized
    familyGeneration = normalizeGeneration(nextGeneration)
    hydrated = true
    if (refreshIdentityChanged) refreshMutationVersion += 1
    if (changed && shouldNotify) notify()
    return copySnapshot(snapshot)
  }

  const legacyMirrorEntries = (
    next: AuthSessionSnapshot,
    generation: string,
  ): ReadonlyArray<readonly [string, string | null]> => {
    let serializedUser: string | null = null
    if (next.user) serializedUser = JSON.stringify(next.user)
    return [
      // Mark a protocol write before touching pre-protocol keys. If an initial
      // family commit tears later, it cannot be mistaken for legacy state.
      [AUTH_SESSION_STORAGE_KEYS.generation, generation],
      [AUTH_SESSION_STORAGE_KEYS.accessToken, next.accessToken],
      [AUTH_SESSION_STORAGE_KEYS.refreshToken, next.refreshToken],
      [
        AUTH_SESSION_STORAGE_KEYS.expiresAt,
        next.expiresAt === null ? null : String(next.expiresAt),
      ],
      [AUTH_SESSION_STORAGE_KEYS.user, serializedUser],
    ]
  }

  const writeLegacyMirrors = (next: AuthSessionSnapshot, generation: string) => {
    if (!storage) return
    const entries = legacyMirrorEntries(next, generation)
    for (const [key, value] of entries) {
      if (value === null) storage.removeItem(key)
      else storage.setItem(key, value)
    }
  }

  const bestEffortWriteLegacyMirrors = (
    next: AuthSessionSnapshot,
    generation: string,
  ) => {
    if (!storage) return
    let entries: ReadonlyArray<readonly [string, string | null]>
    try {
      entries = legacyMirrorEntries(next, generation)
    } catch {
      return
    }
    for (const [key, value] of entries) {
      try {
        if (value === null) storage.removeItem(key)
        else storage.setItem(key, value)
      } catch {
        // The fixed head remains authoritative even when a compatibility
        // mirror cannot be repaired in this browser.
      }
    }
  }

  const bestEffortRemoveLegacyMirrors = () => {
    if (!storage) return
    for (const key of [
      AUTH_SESSION_STORAGE_KEYS.accessToken,
      AUTH_SESSION_STORAGE_KEYS.refreshToken,
      AUTH_SESSION_STORAGE_KEYS.expiresAt,
      AUTH_SESSION_STORAGE_KEYS.user,
      AUTH_SESSION_STORAGE_KEYS.generation,
    ]) {
      try {
        storage.removeItem(key)
      } catch {
        // A committed tombstone keeps the session empty even when cleanup is
        // denied or interrupted.
      }
    }
  }

  const readInvalidatedGeneration = (): string | null => {
    if (!tabStorage) return invalidatedGeneration
    try {
      const persisted = normalizeGeneration(
        tabStorage.getItem(AUTH_SESSION_INVALIDATED_GENERATION_KEY),
      )
      if (persisted) invalidatedGeneration = persisted
    } catch {
      // The page-lifetime fence remains authoritative when tab storage fails.
    }
    return invalidatedGeneration
  }

  const writeInvalidatedGeneration = (generation: string | null) => {
    invalidatedGeneration = normalizeGeneration(generation)
    if (!tabStorage) return
    try {
      if (invalidatedGeneration) {
        tabStorage.setItem(AUTH_SESSION_INVALIDATED_GENERATION_KEY, invalidatedGeneration)
      }
      else tabStorage.removeItem(AUTH_SESSION_INVALIDATED_GENERATION_KEY)
    } catch {
      // In-memory invalidation still protects this page lifetime.
    }
  }

  const parseHead = (raw: string | null): StoredAuthSessionHead | null => {
    if (raw === null) return null
    try {
      const candidate = JSON.parse(raw) as Partial<StoredAuthSessionHead> | null
      if (candidate?.version !== 1) return null
      if (candidate.state === 'active') {
        const generation = normalizeGeneration(candidate.generation)
        return generation ? { version: 1, state: 'active', generation } : null
      }
      if (candidate.state === 'cleared') {
        const tombstone = normalizeGeneration(candidate.tombstone)
        return tombstone ? { version: 1, state: 'cleared', tombstone } : null
      }
      return null
    } catch {
      return null
    }
  }

  const bestEffortRemoveUnreferencedFamily = (generation: string | null) => {
    const normalizedGeneration = normalizeGeneration(generation)
    if (!storage || !normalizedGeneration) return
    try {
      const rawHead = storage.getItem(AUTH_SESSION_STORAGE_KEYS.head)
      const head = parseHead(rawHead)
      // An unreadable head is not proof that a family is unowned.
      if (rawHead !== null && !head) return
      if (head?.state === 'active' && head.generation === normalizedGeneration) return

      // Re-read immediately before deletion. Generations are never reused, so
      // a family not referenced by a stable head cannot become active again.
      const confirmedRawHead = storage.getItem(AUTH_SESSION_STORAGE_KEYS.head)
      if (confirmedRawHead !== rawHead) return
      const confirmedHead = parseHead(confirmedRawHead)
      if (confirmedRawHead !== null && !confirmedHead) return
      if (
        confirmedHead?.state === 'active'
        && confirmedHead.generation === normalizedGeneration
      ) return
      storage.removeItem(`${AUTH_SESSION_FAMILY_RECORD_PREFIX}${normalizedGeneration}`)
    } catch {
      // Head/tombstone remains authoritative if credential cleanup is denied.
    }
  }

  const readActiveHeadGeneration = (): string | null => {
    if (!storage) return familyGeneration
    try {
      const head = parseHead(storage.getItem(AUTH_SESSION_STORAGE_KEYS.head))
      return head?.state === 'active' ? head.generation : familyGeneration
    } catch {
      return familyGeneration
    }
  }

  const parseFamily = (raw: string | null): StoredAuthSessionFamily | null => {
    if (raw === null) return null
    try {
      const candidate = JSON.parse(raw) as Partial<StoredAuthSessionFamily> | null
      const generation = normalizeGeneration(candidate?.generation)
      const candidateSnapshot = candidate?.snapshot
      if (
        candidate?.version !== 1
        || !generation
        || !candidateSnapshot
        || typeof candidateSnapshot !== 'object'
        || Array.isArray(candidateSnapshot)
      ) {
        return null
      }
      const candidateUser = candidateSnapshot.user
      if (
        candidateUser !== null
        && candidateUser !== undefined
        && (
          typeof candidateUser !== 'object'
          || Array.isArray(candidateUser)
        )
      ) {
        return null
      }
      const family: StoredAuthSessionFamily = {
        version: 1,
        generation,
        snapshot: {
          accessToken: normalizeToken(candidateSnapshot.accessToken),
          refreshToken: normalizeToken(candidateSnapshot.refreshToken),
          expiresAt: normalizeExpiresAt(candidateSnapshot.expiresAt),
          user: candidateUser ?? null,
        },
      }
      return hasSessionMaterial(family.snapshot) ? family : null
    } catch {
      return null
    }
  }

  const readLegacySnapshot = (): {
    generationPresent: boolean
    generation: string | null
    snapshot: AuthSessionSnapshot
    corrupt: boolean
  } => {
    if (!storage) {
      return {
        generationPresent: false,
        generation: null,
        snapshot: copySnapshot(EMPTY_SESSION),
        corrupt: false,
      }
    }

    const rawGeneration = storage.getItem(AUTH_SESSION_STORAGE_KEYS.generation)
    const rawUser = storage.getItem(AUTH_SESSION_STORAGE_KEYS.user)
    let user: AuthSessionUser | null = null
    let corrupt = false
    if (rawUser !== null) {
      try {
        const parsed = JSON.parse(rawUser) as unknown
        if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
          corrupt = true
        } else {
          user = parsed as AuthSessionUser
        }
      } catch {
        corrupt = true
      }
    }

    const generation = normalizeGeneration(rawGeneration)
    if (rawGeneration !== null && !generation) corrupt = true
    return {
      generationPresent: rawGeneration !== null,
      generation,
      snapshot: {
        accessToken: normalizeToken(storage.getItem(AUTH_SESSION_STORAGE_KEYS.accessToken)),
        refreshToken: normalizeToken(storage.getItem(AUTH_SESSION_STORAGE_KEYS.refreshToken)),
        expiresAt: normalizeExpiresAt(storage.getItem(AUTH_SESSION_STORAGE_KEYS.expiresAt)),
        user,
      },
      corrupt,
    }
  }

  const readStoredAuthority = (validateMirrors = true): StoredSessionAuthority => {
    if (!storage) {
      return familyGeneration && hasSessionMaterial(snapshot)
        ? { kind: 'active', generation: familyGeneration, snapshot: copySnapshot(snapshot) }
        : {
          kind: 'legacy',
          generationPresent: false,
          snapshot: copySnapshot(EMPTY_SESSION),
          corrupt: false,
        }
    }

    try {
      for (let attempt = 0; attempt < 3; attempt += 1) {
        const rawHeadBefore = storage.getItem(AUTH_SESSION_STORAGE_KEYS.head)
        const head = parseHead(rawHeadBefore)
        if (rawHeadBefore !== null && !head) {
          if (storage.getItem(AUTH_SESSION_STORAGE_KEYS.head) !== rawHeadBefore) continue
          return { kind: 'invalid' }
        }

        if (head?.state === 'cleared') {
          if (storage.getItem(AUTH_SESSION_STORAGE_KEYS.head) !== rawHeadBefore) continue
          return { kind: 'cleared' }
        }

        if (head?.state === 'active') {
          const family = parseFamily(
            storage.getItem(`${AUTH_SESSION_FAMILY_RECORD_PREFIX}${head.generation}`),
          )
          const legacy = validateMirrors ? readLegacySnapshot() : null
          if (storage.getItem(AUTH_SESSION_STORAGE_KEYS.head) !== rawHeadBefore) continue
          if (!family || family.generation !== head.generation) return { kind: 'invalid' }
          if (
            legacy
            && (
              legacy.corrupt
              || legacy.generation !== head.generation
              || !sessionsEqual(legacy.snapshot, family.snapshot)
            )
          ) {
            // Compatibility mirrors never replace the family record. A torn
            // mirror update or an old client mutation fails closed.
            return { kind: 'invalid' }
          }
          return {
            kind: 'active',
            generation: head.generation,
            snapshot: family.snapshot,
          }
        }

        const legacy = readLegacySnapshot()
        if (storage.getItem(AUTH_SESSION_STORAGE_KEYS.head) !== rawHeadBefore) continue
        return {
          kind: 'legacy',
          generationPresent: legacy.generationPresent,
          snapshot: legacy.snapshot,
          corrupt: legacy.corrupt,
        }
      }
    } catch {
      // Read failures are indistinguishable from a torn persistent session.
    }
    return { kind: 'invalid' }
  }

  const repairLegacyMirrorsFromHead = (): StoredSessionAuthority => {
    let authority = readStoredAuthority(false)
    for (let attempt = 0; attempt < 2; attempt += 1) {
      if (authority.kind === 'active') {
        bestEffortWriteLegacyMirrors(authority.snapshot, authority.generation)
      } else if (authority.kind === 'cleared') {
        bestEffortRemoveLegacyMirrors()
      } else {
        return authority
      }

      const latest = readStoredAuthority(false)
      const unchanged = authority.kind === latest.kind
        && (
          authority.kind !== 'active'
          || (
            latest.kind === 'active'
            && authority.generation === latest.generation
            && sessionsEqual(authority.snapshot, latest.snapshot)
          )
        )
      authority = latest
      if (unchanged) break
    }
    return authority
  }

  const commitLegacyMigration = (
    next: AuthSessionSnapshot,
    generation: string,
  ): StoredSessionAuthority => {
    if (!storage) return { kind: 'active', generation, snapshot: next }
    const family: StoredAuthSessionFamily = { version: 1, generation, snapshot: next }
    const head: StoredAuthSessionHead = { version: 1, state: 'active', generation }
    try {
      storage.setItem(
        `${AUTH_SESSION_FAMILY_RECORD_PREFIX}${generation}`,
        JSON.stringify(family),
      )
      if (storage.getItem(AUTH_SESSION_STORAGE_KEYS.head) !== null) {
        const winner = repairLegacyMirrorsFromHead()
        bestEffortRemoveUnreferencedFamily(generation)
        return winner
      }
      writeLegacyMirrors(next, generation)
      if (storage.getItem(AUTH_SESSION_STORAGE_KEYS.head) !== null) {
        const winner = repairLegacyMirrorsFromHead()
        bestEffortRemoveUnreferencedFamily(generation)
        return winner
      }
      storage.setItem(AUTH_SESSION_STORAGE_KEYS.head, JSON.stringify(head))
    } catch {
      bestEffortRemoveUnreferencedFamily(generation)
      return { kind: 'invalid' }
    }
    const committed = readStoredAuthority()
    if (committed.kind !== 'active' || committed.generation !== generation) {
      bestEffortRemoveUnreferencedFamily(generation)
    }
    return committed
  }

  const resolveStoredAuthority = (): StoredSessionAuthority => {
    const authority = readStoredAuthority()
    if (authority.kind !== 'legacy') return authority

    // A generation mirror proves that a head/family commit was attempted.
    // Never reinterpret such partial protocol state as a pre-protocol session.
    if (authority.generationPresent) return { kind: 'invalid' }
    if (authority.corrupt) {
      try {
        const tombstone: StoredAuthSessionHead = {
          version: 1,
          state: 'cleared',
          tombstone: createSessionGeneration(),
        }
        storage?.setItem(AUTH_SESSION_STORAGE_KEYS.head, JSON.stringify(tombstone))
        bestEffortRemoveLegacyMirrors()
      } catch {
        // Fail closed even if corrupt pre-protocol data cannot be cleaned up.
      }
      return { kind: 'invalid' }
    }
    if (!hasSessionMaterial(authority.snapshot)) return authority

    // One-time migration is only legal when both the fixed head and generation
    // mirror have never existed. SHA-256 gives simultaneous tabs one family ID.
    return commitLegacyMigration(
      authority.snapshot,
      createLegacySessionGeneration(authority.snapshot),
    )
  }

  const applyStoredAuthority = (authority: StoredSessionAuthority): AuthSessionSnapshot => {
    if (authority.kind !== 'active') {
      if (authority.kind === 'cleared') writeInvalidatedGeneration(null)
      return applySnapshot(EMPTY_SESSION, null)
    }

    const invalidated = readInvalidatedGeneration()
    if (invalidated === authority.generation) {
      return applySnapshot(EMPTY_SESSION, null)
    }
    if (invalidated) writeInvalidatedGeneration(null)
    return applySnapshot(authority.snapshot, authority.generation)
  }

  const syncFromStorage = () => {
    storageSyncScheduled = false
    if (!storage) return
    applyStoredAuthority(resolveStoredAuthority())
  }

  const handleStorage = (event: Event) => {
    const storageEvent = event as StorageEvent
    if (
      storageEvent.key !== null
      && !SESSION_STORAGE_KEY_SET.has(storageEvent.key)
      && !storageEvent.key.startsWith(AUTH_SESSION_FAMILY_RECORD_PREFIX)
    ) return
    if (storageEvent.storageArea && storage && storageEvent.storageArea !== storage) return
    if (storageSyncScheduled) return
    storageSyncScheduled = true
    queueMicrotask(syncFromStorage)
  }

  const ensureStorageSync = () => {
    if (listeningForStorage || !eventTarget) return
    eventTarget.addEventListener('storage', handleStorage)
    listeningForStorage = true
  }

  const hydrate = (): AuthSessionSnapshot => {
    ensureStorageSync()
    if (!storage) {
      hydrated = true
      return copySnapshot(snapshot)
    }
    return applyStoredAuthority(resolveStoredAuthority())
  }

  const getSnapshot = (): AuthSessionSnapshot => {
    if (!hydrated) return hydrate()
    return copySnapshot(snapshot)
  }

  const getGeneration = (): string | null => {
    if (!hydrated) hydrate()
    return familyGeneration
  }

  const replace = (next: AuthSessionSnapshot): AuthSessionSnapshot => {
    ensureStorageSync()
    const previousGeneration = readActiveHeadGeneration()
    const normalized: AuthSessionSnapshot = {
      accessToken: normalizeToken(next.accessToken),
      refreshToken: normalizeToken(next.refreshToken),
      expiresAt: normalizeExpiresAt(next.expiresAt),
      user: next.user ?? null,
    }
    if (!hasSessionMaterial(normalized)) {
      clear()
      return copySnapshot(EMPTY_SESSION)
    }

    const generation = createSessionGeneration()
    writeInvalidatedGeneration(null)
    if (storage) {
      const family: StoredAuthSessionFamily = {
        version: 1,
        generation,
        snapshot: normalized,
      }
      const head: StoredAuthSessionHead = {
        version: 1,
        state: 'active',
        generation,
      }
      try {
        // The new family is complete before any shared pointer can reference it.
        storage.setItem(
          `${AUTH_SESSION_FAMILY_RECORD_PREFIX}${generation}`,
          JSON.stringify(family),
        )
        writeLegacyMirrors(normalized, generation)
        // This fixed-key atomic write is the only family commit point.
        storage.setItem(AUTH_SESSION_STORAGE_KEYS.head, JSON.stringify(head))
      } catch (error) {
        const authority = repairLegacyMirrorsFromHead()
        bestEffortRemoveUnreferencedFamily(generation)
        applyStoredAuthority(authority)
        throw error
      }

      const committed = readStoredAuthority(false)
      if (committed.kind !== 'active' || committed.generation !== generation) {
        const winner = repairLegacyMirrorsFromHead()
        bestEffortRemoveUnreferencedFamily(generation)
        applyStoredAuthority(winner)
        throw new AuthSessionChangedError()
      }
      if (previousGeneration !== generation) {
        bestEffortRemoveUnreferencedFamily(previousGeneration)
      }
    }
    return applySnapshot(normalized, generation)
  }

  const patch = (changes: AuthSessionPatch): AuthSessionSnapshot => {
    const expectedGeneration = getGeneration()
    const authority = resolveStoredAuthority()
    if (
      !expectedGeneration
      || authority.kind !== 'active'
      || authority.generation !== expectedGeneration
    ) {
      applyStoredAuthority(authority)
      bestEffortRemoveUnreferencedFamily(expectedGeneration)
      // Patch is strictly an in-family mutation. Async 2xx responses from an
      // older family must not manufacture a new family or move the head.
      throw new AuthSessionChangedError()
    }
    const current = authority.snapshot
    const next: AuthSessionSnapshot = {
      accessToken: Object.prototype.hasOwnProperty.call(changes, 'accessToken')
        ? normalizeToken(changes.accessToken)
        : current.accessToken,
      refreshToken: Object.prototype.hasOwnProperty.call(changes, 'refreshToken')
        ? normalizeToken(changes.refreshToken)
        : current.refreshToken,
      expiresAt: Object.prototype.hasOwnProperty.call(changes, 'expiresAt')
        ? normalizeExpiresAt(changes.expiresAt)
        : current.expiresAt,
      user: Object.prototype.hasOwnProperty.call(changes, 'user')
        ? changes.user ?? null
        : current.user,
    }
    if (!hasSessionMaterial(next)) throw new AuthSessionChangedError()
    writeInvalidatedGeneration(null)
    if (storage) {
      const family: StoredAuthSessionFamily = {
        version: 1,
        generation: expectedGeneration,
        snapshot: next,
      }
      try {
        // Compatibility mirrors are written first. Until the single family
        // JSON lands, readers fail closed instead of observing a torn pair.
        writeLegacyMirrors(next, expectedGeneration)
        storage.setItem(
          `${AUTH_SESSION_FAMILY_RECORD_PREFIX}${expectedGeneration}`,
          JSON.stringify(family),
        )
      } catch (error) {
        const winner = repairLegacyMirrorsFromHead()
        bestEffortRemoveUnreferencedFamily(expectedGeneration)
        applyStoredAuthority(winner)
        throw error
      }

      // Patch never writes the head. If a peer replaced or cleared the family
      // during any mirror write, repair that winner and reject this response.
      const committed = readStoredAuthority(false)
      if (
        committed.kind !== 'active'
        || committed.generation !== expectedGeneration
      ) {
        const winner = repairLegacyMirrorsFromHead()
        bestEffortRemoveUnreferencedFamily(expectedGeneration)
        applyStoredAuthority(winner)
        throw new AuthSessionChangedError()
      }
      bestEffortWriteLegacyMirrors(committed.snapshot, committed.generation)
      return applySnapshot(committed.snapshot, committed.generation)
    }
    return applySnapshot(next, expectedGeneration)
  }

  const clear = (): void => {
    ensureStorageSync()
    const previousGeneration = readActiveHeadGeneration()
    if (storage) {
      const tombstone: StoredAuthSessionHead = {
        version: 1,
        state: 'cleared',
        tombstone: createSessionGeneration(),
      }
      const serializedTombstone = JSON.stringify(tombstone)
      try {
        // Commit emptiness first. Legacy cleanup is never the authority.
        storage.setItem(AUTH_SESSION_STORAGE_KEYS.head, serializedTombstone)
      } catch (error) {
        syncFromStorage()
        throw error
      }
      bestEffortRemoveLegacyMirrors()

      try {
        if (storage.getItem(AUTH_SESSION_STORAGE_KEYS.head) !== serializedTombstone) {
          const winner = repairLegacyMirrorsFromHead()
          bestEffortRemoveUnreferencedFamily(previousGeneration)
          writeInvalidatedGeneration(null)
          applyStoredAuthority(winner)
          return
        }
      } catch {
        // The tombstone was committed; a later read failure must not resurrect
        // the in-memory session in this tab.
      }
      bestEffortRemoveUnreferencedFamily(previousGeneration)
    }
    writeInvalidatedGeneration(null)
    applySnapshot(EMPTY_SESSION, null)
  }

  const invalidate = (
    expected: string | null | AuthSessionInvalidationExpectation,
  ): boolean => {
    ensureStorageSync()
    // Reconcile and compare at the mutation boundary. Callers that provide the
    // full identity cannot invalidate a same-family token rotation observed in
    // durable storage. This operation never deletes localStorage.
    syncFromStorage()
    const expectation = expected !== null && typeof expected === 'object'
      ? expected
      : null
    const expectedGeneration = expectation
      ? normalizeGeneration(expectation.generation)
      : normalizeGeneration(expected)
    if (familyGeneration !== expectedGeneration) return false
    if (
      expectation
      && (
        snapshot.accessToken !== normalizeToken(expectation.accessToken)
        || snapshot.refreshToken !== normalizeToken(expectation.refreshToken)
        || (snapshot.user?.id ?? null) !== expectation.userId
      )
    ) return false

    writeInvalidatedGeneration(expectedGeneration)
    applySnapshot(EMPTY_SESSION, null)
    return true
  }

  const subscribe = (subscriber: AuthSessionSubscriber): (() => void) => {
    ensureStorageSync()
    subscribers.add(subscriber)
    return () => subscribers.delete(subscriber)
  }

  const refresh = (requester: AuthSessionRefreshRequester): Promise<AuthSessionSnapshot> => {
    if (refreshFlight) return refreshFlight

    const current = getSnapshot()
    if (!current.refreshToken) {
      return Promise.reject(new Error('No refresh token available'))
    }

    const refreshToken = current.refreshToken
    const currentGeneration = familyGeneration
    const startRefreshMutationVersion = refreshMutationVersion

    const reconcileRefreshMutation = (): AuthSessionSnapshot | null => {
      // Storage events are asynchronous and may not have reached this tab yet.
      // Always inspect durable storage while holding the refresh lock and again
      // after the requester settles.
      syncFromStorage()
      const latest = getSnapshot()
      const latestGeneration = familyGeneration
      const accessTokenChanged = latest.accessToken !== current.accessToken
      const refreshTokenChanged = latest.refreshToken !== current.refreshToken
      const principalChanged = (latest.user?.id ?? null) !== (current.user?.id ?? null)
      const generationChanged = latestGeneration !== currentGeneration
      const mutationObserved = refreshMutationVersion !== startRefreshMutationVersion

      if (
        !mutationObserved
        && !accessTokenChanged
        && !refreshTokenChanged
        && !principalChanged
        && !generationChanged
      ) {
        return null
      }

      // A peer tab may have rotated this exact user's pair first. Reuse only a
      // complete, genuinely different pair; logout, login as another user, and
      // ambiguous mutations invalidate the stale refresh caller.
      if (
        latest.accessToken
        && latest.refreshToken
        && (accessTokenChanged || refreshTokenChanged)
        && isSameAuthenticatedPrincipal(current, latest)
        && latestGeneration === currentGeneration
      ) {
        return latest
      }
      throw new AuthSessionChangedError()
    }

    const runRefresh = async (): Promise<AuthSessionSnapshot> => {
      const existingRotation = reconcileRefreshMutation()
      if (existingRotation) return existingRotation

      let result: AuthSessionRefreshResult
      try {
        result = await requester(refreshToken)
      } catch (error) {
        const concurrentRotation = reconcileRefreshMutation()
        if (concurrentRotation) return concurrentRotation
        throw error
      }

      const concurrentRotation = reconcileRefreshMutation()
      if (concurrentRotation) return concurrentRotation

      const accessToken = normalizeToken(result.access_token)
      if (!accessToken) throw new Error('Token refresh response did not include an access token')

      return patch({
        accessToken,
        refreshToken: normalizeToken(result.refresh_token) ?? refreshToken,
        expiresAt: typeof result.expires_in === 'number'
          ? Date.now() + result.expires_in * 1000
          : current.expiresAt,
      })
    }

    const request = lockManager
      ? lockManager.request(AUTH_SESSION_REFRESH_LOCK_NAME, runRefresh)
      : runRefresh()

    const tracked = request.finally(() => {
      if (refreshFlight === tracked) refreshFlight = null
    })
    refreshFlight = tracked
    return tracked
  }

  const dispose = () => {
    if (listeningForStorage && eventTarget) {
      eventTarget.removeEventListener('storage', handleStorage)
      listeningForStorage = false
    }
    subscribers.clear()
  }

  return {
    hydrate,
    getSnapshot,
    getGeneration,
    replace,
    patch,
    clear,
    invalidate,
    subscribe,
    refresh,
    dispose,
  }
}

export const authSession = createAuthSession()

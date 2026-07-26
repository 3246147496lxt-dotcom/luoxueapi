import type {
  HistoryState,
  LocationQuery,
  LocationQueryValue,
  RouteLocationRaw,
  Router,
} from 'vue-router'

export const PERSONAL_SETTINGS_SECTIONS = [
  'general',
  'account',
  'security',
  'notifications',
  'billing',
] as const

export type PersonalSettingsSection = (typeof PERSONAL_SETTINGS_SECTIONS)[number]

export const PERSONAL_SETTINGS_DETAILS = [
  'profile',
  'connections',
  'password',
  'totp',
  'notification-emails',
] as const

export type PersonalSettingsDetail = (typeof PERSONAL_SETTINGS_DETAILS)[number]
export type PersonalSettingsAudience = 'user' | 'admin'
export type PersonalSettingsHostPath = '/dashboard' | '/admin/dashboard'

export interface PersonalSettingsRouteLocation {
  readonly path: string
  readonly query: LocationQuery
  readonly hash: string
}

export interface PersonalSettingsRouteState {
  readonly isOpen: boolean
  readonly section: PersonalSettingsSection
  readonly detail: PersonalSettingsDetail | null
  readonly needsCanonicalization: boolean
}

interface PersonalSettingsRouteTarget {
  readonly path: string
  readonly query: LocationQuery
  readonly hash: string
  readonly state?: HistoryState
  readonly replace?: boolean
}

export interface PersonalSettingsCanonicalizationOptions {
  /**
   * Removes settings state even when the query is otherwise valid. This is
   * used by higher-priority gates (for example mandatory admin compliance)
   * before the modal is allowed to activate.
   */
  readonly activationBlocked?: boolean
}

const SECTION_QUERY_KEY = 'account_settings'
const DETAIL_QUERY_KEY = 'account_settings_detail'
const DEFAULT_SECTION: PersonalSettingsSection = 'general'
const HISTORY_OWNER_KEY = '__sub2api_personal_settings_owner'
const HISTORY_OWNER_STORAGE_KEY = '__sub2api_personal_settings_owner_token'
const HISTORY_OWNER_VERSION = 1
const MAX_LIVE_HISTORY_CHAINS = 16

const SECTION_SET: ReadonlySet<string> = new Set(PERSONAL_SETTINGS_SECTIONS)
const DETAIL_SET: ReadonlySet<string> = new Set(PERSONAL_SETTINGS_DETAILS)
const HOST_PATH_SET: ReadonlySet<string> = new Set(['/dashboard', '/admin/dashboard'])

const DETAIL_SECTION: Readonly<Record<PersonalSettingsDetail, PersonalSettingsSection>> = {
  profile: 'account',
  connections: 'account',
  password: 'security',
  totp: 'security',
  'notification-emails': 'notifications',
}

type PersonalSettingsHistoryDepth = 0 | 1

interface PersonalSettingsHistoryOwner extends HistoryState {
  readonly version: typeof HISTORY_OWNER_VERSION
  readonly ownerToken: string
  readonly documentId: string
  readonly chainId: string
  readonly depth: PersonalSettingsHistoryDepth
  readonly originPosition: number
}

interface LivePersonalSettingsHistoryChain {
  readonly ownerToken: string
  readonly documentId: string
  readonly chainId: string
  readonly originPosition: number
  hasDepthOne: boolean
}

function createOpaqueId(prefix: string): string {
  const randomUUID = globalThis.crypto?.randomUUID?.bind(globalThis.crypto)
  if (randomUUID) return `${prefix}-${randomUUID()}`
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}`
}

function readOrCreateHistoryOwnerToken(): string {
  if (typeof window === 'undefined') {
    return createOpaqueId('personal-settings-session')
  }

  try {
    const persisted = window.sessionStorage.getItem(HISTORY_OWNER_STORAGE_KEY)
    if (persisted) return persisted

    const created = createOpaqueId('personal-settings-session')
    window.sessionStorage.setItem(HISTORY_OWNER_STORAGE_KEY, created)
    return created
  } catch {
    return createOpaqueId('personal-settings-session')
  }
}

// The session token intentionally survives a full refresh so restored entries
// remain attributable to this browser tab. The document id and live registry
// intentionally do not: a refreshed document cannot prove that the two
// adjacent entries still form the chain it created, so closing falls back to
// a query-only replace.
const HISTORY_OWNER_TOKEN = readOrCreateHistoryOwnerToken()
const HISTORY_DOCUMENT_ID = createOpaqueId('personal-settings-document')
const liveHistoryChains = new Map<string, LivePersonalSettingsHistoryChain>()

function scalarQueryValue(
  value: LocationQueryValue | LocationQueryValue[],
): LocationQueryValue {
  return Array.isArray(value) ? null : value
}

function hasQueryKey(query: LocationQuery, key: string): boolean {
  return Object.prototype.hasOwnProperty.call(query, key)
}

export function isPersonalSettingsSection(
  value: unknown,
): value is PersonalSettingsSection {
  return typeof value === 'string' && SECTION_SET.has(value)
}

export function isPersonalSettingsDetail(
  value: unknown,
): value is PersonalSettingsDetail {
  return typeof value === 'string' && DETAIL_SET.has(value)
}

export function isPersonalSettingsHostPath(
  path: string,
): path is PersonalSettingsHostPath {
  return HOST_PATH_SET.has(path)
}

export function getPersonalSettingsHostPath(
  audience: PersonalSettingsAudience,
): PersonalSettingsHostPath {
  return audience === 'admin' ? '/admin/dashboard' : '/dashboard'
}

export function getPersonalSettingsDetailSection(
  detail: PersonalSettingsDetail,
): PersonalSettingsSection {
  return DETAIL_SECTION[detail]
}

export function parsePersonalSettingsRoute(
  route: PersonalSettingsRouteLocation,
): PersonalSettingsRouteState {
  // Normalized Vue Router locations always provide query. The fallback keeps
  // guard-level test doubles and transitional callers safe as well.
  const query = route.query ?? {}
  const hasSectionQuery = hasQueryKey(query, SECTION_QUERY_KEY)
  const hasDetailQuery = hasQueryKey(query, DETAIL_QUERY_KEY)
  const sectionValue = scalarQueryValue(query[SECTION_QUERY_KEY])
  const hasValidSection = isPersonalSettingsSection(sectionValue)
  const section = hasValidSection ? sectionValue : DEFAULT_SECTION
  const isOpen = isPersonalSettingsHostPath(route.path) && hasValidSection

  const detailValue = scalarQueryValue(query[DETAIL_QUERY_KEY])
  const hasValidDetail = isPersonalSettingsDetail(detailValue)
    && getPersonalSettingsDetailSection(detailValue) === section
  const detail = isOpen && hasValidDetail ? detailValue : null

  const needsCanonicalization = (
    (hasSectionQuery || hasDetailQuery)
    && (
      !isPersonalSettingsHostPath(route.path)
      || !hasValidSection
      || (hasDetailQuery && !hasValidDetail)
    )
  )

  return {
    isOpen,
    section,
    detail,
    needsCanonicalization,
  }
}

function createSettingsQuery(
  route: Pick<PersonalSettingsRouteLocation, 'query'>,
  section: PersonalSettingsSection | null,
  detail: PersonalSettingsDetail | null = null,
): LocationQuery {
  const query: LocationQuery = { ...(route.query ?? {}) }
  delete query[SECTION_QUERY_KEY]
  delete query[DETAIL_QUERY_KEY]

  if (section) {
    query[SECTION_QUERY_KEY] = section
  }
  if (section && detail && getPersonalSettingsDetailSection(detail) === section) {
    query[DETAIL_QUERY_KEY] = detail
  }

  return query
}

function createRouteTarget(
  route: Pick<PersonalSettingsRouteLocation, 'query' | 'hash'>,
  path: string,
  query: LocationQuery,
  state?: HistoryState,
): PersonalSettingsRouteTarget {
  return {
    path,
    query,
    hash: route.hash ?? '',
    ...(state ? { state } : {}),
  }
}

function clearHistoryOwnershipState(): HistoryState {
  return {
    [HISTORY_OWNER_KEY]: undefined,
  }
}

function getHistoryPosition(): number | null {
  if (typeof window === 'undefined') return null
  const position = window.history.state?.position
  return Number.isSafeInteger(position) && position >= 0 ? position : null
}

function isHistoryOwner(value: unknown): value is PersonalSettingsHistoryOwner {
  if (!value || typeof value !== 'object') return false
  const owner = value as Partial<PersonalSettingsHistoryOwner>
  return owner.version === HISTORY_OWNER_VERSION
    && typeof owner.ownerToken === 'string'
    && typeof owner.documentId === 'string'
    && typeof owner.chainId === 'string'
    && (owner.depth === 0 || owner.depth === 1)
    && Number.isSafeInteger(owner.originPosition)
    && (owner.originPosition ?? -1) >= 0
}

function readLiveHistoryOwner(): PersonalSettingsHistoryOwner | null {
  if (typeof window === 'undefined') return null
  const owner = window.history.state?.[HISTORY_OWNER_KEY]
  if (!isHistoryOwner(owner)) return null
  if (
    owner.ownerToken !== HISTORY_OWNER_TOKEN
    || owner.documentId !== HISTORY_DOCUMENT_ID
  ) {
    return null
  }

  const chain = liveHistoryChains.get(owner.chainId)
  if (
    !chain
    || chain.ownerToken !== owner.ownerToken
    || chain.documentId !== owner.documentId
    || chain.originPosition !== owner.originPosition
  ) {
    return null
  }

  const expectedPosition = owner.originPosition + owner.depth + 1
  if (getHistoryPosition() !== expectedPosition) return null
  if (owner.depth === 1 && !chain.hasDepthOne) return null

  return owner
}

function createHistoryOwner(
  chain: LivePersonalSettingsHistoryChain,
  depth: PersonalSettingsHistoryDepth,
): PersonalSettingsHistoryOwner {
  return {
    version: HISTORY_OWNER_VERSION,
    ownerToken: chain.ownerToken,
    documentId: chain.documentId,
    chainId: chain.chainId,
    depth,
    originPosition: chain.originPosition,
  }
}

function registerHistoryChain(
  originPosition: number,
): LivePersonalSettingsHistoryChain {
  const chain: LivePersonalSettingsHistoryChain = {
    ownerToken: HISTORY_OWNER_TOKEN,
    documentId: HISTORY_DOCUMENT_ID,
    chainId: createOpaqueId('personal-settings-chain'),
    originPosition,
    hasDepthOne: false,
  }
  liveHistoryChains.set(chain.chainId, chain)

  while (liveHistoryChains.size > MAX_LIVE_HISTORY_CHAINS) {
    const oldestChainId = liveHistoryChains.keys().next().value
    if (typeof oldestChainId !== 'string') break
    liveHistoryChains.delete(oldestChainId)
  }

  return chain
}

function historyOwnerState(
  owner: PersonalSettingsHistoryOwner | null,
): HistoryState {
  return owner
    ? { [HISTORY_OWNER_KEY]: owner }
    : clearHistoryOwnershipState()
}

export function resolvePersonalSettingsCanonicalization(
  route: PersonalSettingsRouteLocation,
  options: PersonalSettingsCanonicalizationOptions = {},
): RouteLocationRaw | null {
  const state = parsePersonalSettingsRoute(route)
  const hasSettingsQuery = hasQueryKey(route.query ?? {}, SECTION_QUERY_KEY)
    || hasQueryKey(route.query ?? {}, DETAIL_QUERY_KEY)

  if (options.activationBlocked && hasSettingsQuery) {
    return {
      ...createRouteTarget(
        route,
        route.path,
        createSettingsQuery(route, null),
        clearHistoryOwnershipState(),
      ),
      replace: true,
    }
  }

  if (!state.needsCanonicalization) return null

  const keepSection = isPersonalSettingsHostPath(route.path)
    && isPersonalSettingsSection(
      scalarQueryValue((route.query ?? {})[SECTION_QUERY_KEY]),
    )

  return {
    ...createRouteTarget(
      route,
      route.path,
      createSettingsQuery(route, keepSection ? state.section : null, state.detail),
      keepSection ? undefined : clearHistoryOwnershipState(),
    ),
    replace: true,
  }
}

export function canonicalizePersonalSettingsRoute(
  router: Router,
  route: PersonalSettingsRouteLocation,
) {
  const target = resolvePersonalSettingsCanonicalization(route)
  return target ? router.replace(target) : null
}

export function openPersonalSettings(
  router: Router,
  route: PersonalSettingsRouteLocation,
  audience: PersonalSettingsAudience,
  section: PersonalSettingsSection = DEFAULT_SECTION,
) {
  const originPosition = getHistoryPosition()
  const chain = originPosition === null
    ? null
    : registerHistoryChain(originPosition)

  return router.push(
    createRouteTarget(
      route,
      getPersonalSettingsHostPath(audience),
      createSettingsQuery(route, section),
      historyOwnerState(chain ? createHistoryOwner(chain, 0) : null),
    ),
  )
}

export function replacePersonalSettingsSection(
  router: Router,
  route: PersonalSettingsRouteLocation,
  section: PersonalSettingsSection,
) {
  if (!isPersonalSettingsHostPath(route.path)) {
    return canonicalizePersonalSettingsRoute(router, route)
  }
  const owner = readLiveHistoryOwner()
  return router.replace(
    createRouteTarget(
      route,
      route.path,
      createSettingsQuery(route, section),
      historyOwnerState(owner),
    ),
  )
}

export function replacePersonalSettingsDetail(
  router: Router,
  route: PersonalSettingsRouteLocation,
  detail: PersonalSettingsDetail,
) {
  if (!isPersonalSettingsHostPath(route.path)) {
    return canonicalizePersonalSettingsRoute(router, route)
  }
  const section = getPersonalSettingsDetailSection(detail)
  const query = createSettingsQuery(route, section, detail)
  const owner = readLiveHistoryOwner()
  const routeState = parsePersonalSettingsRoute(route)

  if (owner?.depth === 0 && routeState.detail === null) {
    const chain = liveHistoryChains.get(owner.chainId)
    if (chain) chain.hasDepthOne = true
    return router.push(
      createRouteTarget(
        route,
        route.path,
        query,
        historyOwnerState(chain ? createHistoryOwner(chain, 1) : null),
      ),
    )
  }

  return router.replace(
    createRouteTarget(
      route,
      route.path,
      query,
      historyOwnerState(owner?.depth === 1 ? owner : null),
    ),
  )
}

/**
 * @deprecated Use replacePersonalSettingsDetail. The retained name now
 * performs the same depth-aware push/replace transition.
 */
export const pushPersonalSettingsDetail = replacePersonalSettingsDetail

export function closePersonalSettings(
  router: Router,
  route: PersonalSettingsRouteLocation,
) {
  const state = parsePersonalSettingsRoute(route)
  const owner = state.isOpen ? readLiveHistoryOwner() : null

  if (owner?.depth === 1) {
    return router.go(-2)
  }
  if (owner?.depth === 0) {
    return router.back()
  }

  return router.replace(
    createRouteTarget(
      route,
      route.path,
      createSettingsQuery(route, null),
      clearHistoryOwnershipState(),
    ),
  )
}

export function resolveLegacyPersonalSettingsRedirect(
  route: Pick<PersonalSettingsRouteLocation, 'query' | 'hash'>,
  audience: PersonalSettingsAudience,
  section: PersonalSettingsSection,
  detail?: PersonalSettingsDetail,
): RouteLocationRaw {
  return createRouteTarget(
    route,
    getPersonalSettingsHostPath(audience),
    createSettingsQuery(route, section, detail ?? null),
  )
}

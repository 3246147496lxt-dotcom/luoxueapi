import type {
  AccountPlatform,
  OpenAICompactMode,
  OpenAIResponsesMode,
} from '@/types'
import {
  OPENAI_WS_MODE_OFF,
  isOpenAIWSModeEnabled,
  type OpenAIWSMode,
} from '@/utils/openaiWsMode'
import type {
  BedrockAuthMode,
  CreateAccountCategory,
} from './formDraft'
import {
  normalizeOpenAIEndpointCapabilities,
  serializeOpenAIEndpointCapabilities,
  type OpenAIEndpointCapability,
} from '../openAIEndpointCapabilities'

export const DEFAULT_POOL_MODE_RETRY_COUNT = 3
export const MAX_POOL_MODE_RETRY_COUNT = 10
export const DEFAULT_POOL_MODE_RETRY_STATUS_CODES: number[] = [401, 403, 429]

export type AnthropicAPIKeyAuthScheme =
  | 'x_api_key'
  | 'authorization_bearer'

export interface TempUnschedRuleForm {
  error_code: number | null
  keywords: string
  duration_minutes: number | null
  description: string
}

export interface TempUnschedulableRule {
  error_code: number
  keywords: string[]
  duration_minutes: number
  description: string
}

export interface PoolModeInput {
  enabled: boolean
  retryCount: number
  retryStatusCodesInput: string
}

const DEFAULT_API_KEY_BASE_URL: Record<AccountPlatform, string> = {
  anthropic: 'https://api.anthropic.com',
  openai: 'https://api.openai.com',
  gemini: 'https://generativelanguage.googleapis.com',
  antigravity: 'https://api.anthropic.com',
  grok: 'https://api.x.ai/v1',
  zhipu: 'https://open.bigmodel.cn/api/paas/v4',
  deepseek: 'https://api.deepseek.com',
}

/** Official 智谱 GLM endpoints.  The account mode/protocol selectors in the
 * account form choose among these presets; custom relay URLs remain allowed. */
export const ZHIPU_BASE_URL_PRESETS = [
  { label: 'GLM API（按量）', url: 'https://open.bigmodel.cn/api/paas/v4' },
  { label: 'GLM Coding Plan', url: 'https://open.bigmodel.cn/api/coding/paas/v4' },
  { label: 'GLM Anthropic', url: 'https://open.bigmodel.cn/api/anthropic' },
] as const

export type CnAccountMode = 'payg' | 'coding'
export type CnApiProtocol = 'chat_completions' | 'anthropic' | 'responses' | 'adaptive'

/**
 * The GLM form only exposes the two protocols that the Zhipu adapter can
 * route directly.  Keep this narrower than `CnApiProtocol`: DeepSeek also
 * uses the shared type for its native Responses mode, while a Zhipu base URL
 * can only identify Chat Completions or Anthropic here.
 */
export type ZhipuApiProtocol = Extract<CnApiProtocol, 'chat_completions' | 'anthropic'>

export interface ZhipuBaseURLRouting {
  mode?: CnAccountMode
  protocol?: ZhipuApiProtocol
}

/** Normalize persisted GLM routing values using the same case/whitespace
 * tolerance as the backend. Invalid values return undefined so callers can
 * safely fall back to endpoint inference or the documented defaults. */
export function normalizeZhipuAccountMode(value: unknown): CnAccountMode | undefined {
  if (typeof value !== 'string') return undefined
  const normalized = value.trim().toLowerCase()
  return normalized === 'payg' || normalized === 'coding' ? normalized : undefined
}

export function normalizeZhipuApiProtocol(value: unknown): ZhipuApiProtocol | undefined {
  if (typeof value !== 'string') return undefined
  const normalized = value.trim().toLowerCase()
  return normalized === 'chat_completions' || normalized === 'anthropic'
    ? normalized
    : undefined
}

/**
 * Infer the routing represented by an official GLM endpoint.
 *
 * This is deliberately strict: custom relay URLs must remain opaque and must
 * never be reclassified just because their query/path happens to contain an
 * official host name.  Returning a partial result for `/api/anthropic` is
 * intentional because that endpoint is shared by pay-as-you-go and Coding
 * Plan accounts; the caller should retain an explicitly selected mode.
 */
export function inferZhipuRoutingFromBaseURL(
  value: unknown,
): ZhipuBaseURLRouting | null {
  if (typeof value !== 'string' || !value.trim()) return null

  let parsed: URL
  try {
    parsed = new URL(value.trim())
  } catch {
    return null
  }
  if (
    (parsed.protocol !== 'https:' && parsed.protocol !== 'http:') ||
    parsed.port !== ''
  ) {
    return null
  }

  const host = parsed.hostname.toLowerCase()
  if (host !== 'open.bigmodel.cn' && host !== 'api.z.ai') return null

  const path = parsed.pathname.replace(/\/+$/, '').toLowerCase()
  if (path === '/api/anthropic') {
    return { protocol: 'anthropic' }
  }
  if (path === '/api/coding/paas/v4') {
    return { mode: 'coding', protocol: 'chat_completions' }
  }
  if (path === '/api/paas/v4') {
    return { mode: 'payg', protocol: 'chat_completions' }
  }
  return null
}

/** Return true when a URL is one of the managed official GLM endpoints. */
export function isManagedZhipuBaseURL(value: unknown): boolean {
  return inferZhipuRoutingFromBaseURL(value) !== null
}

/** Resolve the official GLM endpoint for the selected mode/protocol. */
export function defaultZhipuBaseURL(
  mode: CnAccountMode = 'payg',
  protocol: CnApiProtocol = 'chat_completions',
): string {
  if (protocol === 'anthropic') return 'https://open.bigmodel.cn/api/anthropic'
  if (mode === 'coding') return 'https://open.bigmodel.cn/api/coding/paas/v4'
  return 'https://open.bigmodel.cn/api/paas/v4'
}

/**
 * Resolve a managed GLM endpoint for a new mode/protocol while retaining the
 * official host variant already selected by the user (`open.bigmodel.cn` or
 * `api.z.ai`). Custom relays intentionally fall back to the normal mainland
 * preset because their protocol namespace cannot be inferred safely.
 */
export function zhipuBaseURLForRouting(
  mode: CnAccountMode = 'payg',
  protocol: CnApiProtocol = 'chat_completions',
  current?: unknown,
): string {
  const fallback = defaultZhipuBaseURL(mode, protocol)
  if (!isManagedZhipuBaseURL(current)) return fallback
  try {
    const source = new URL(String(current).trim())
    const target = new URL(fallback)
    target.protocol = source.protocol
    target.hostname = source.hostname
    return target.toString().replace(/\/$/, '')
  } catch {
    return fallback
  }
}

/** Official DeepSeek API endpoint shown as a quick-fill option in the account
 * form. The current account form creates the default OpenAI-compatible Chat
 * Completions account; the native Anthropic-compatible endpoint is not yet a
 * selectable protocol here, so do not advertise its `/anthropic` base URL.
 * Users may still enter any compatible relay URL manually.
 */
export const DEEPSEEK_BASE_URL_PRESETS = [
  { label: 'DeepSeek API', url: 'https://api.deepseek.com' },
] as const

export function defaultAPIKeyBaseURL(platform: AccountPlatform): string {
  return DEFAULT_API_KEY_BASE_URL[platform]
}

export function parsePoolModeRetryStatusCodes(input: string): number[] {
  if (!input || !input.trim()) return []
  const seen = new Set<number>()
  const out: number[] = []
  for (const token of input.split(/[,\s]+/)) {
    const trimmed = token.trim()
    if (!trimmed) continue
    const value = Number(trimmed)
    if (!Number.isFinite(value) || !Number.isInteger(value)) continue
    if (value < 100 || value > 599 || seen.has(value)) continue
    seen.add(value)
    out.push(value)
  }
  return out.sort((left, right) => left - right)
}

export function normalizePoolModeRetryCount(value: number): number {
  if (!Number.isFinite(value)) return DEFAULT_POOL_MODE_RETRY_COUNT
  const normalized = Math.trunc(value)
  if (normalized < 0) return 0
  if (normalized > MAX_POOL_MODE_RETRY_COUNT) {
    return MAX_POOL_MODE_RETRY_COUNT
  }
  return normalized
}

export function buildPoolModeCredentialOptions(
  input: PoolModeInput,
): Record<string, unknown> {
  if (!input.enabled) return {}
  const options: Record<string, unknown> = {
    pool_mode: true,
    pool_mode_retry_count: normalizePoolModeRetryCount(input.retryCount),
  }
  const statusCodes = parsePoolModeRetryStatusCodes(
    input.retryStatusCodesInput,
  )
  if (statusCodes.length > 0) {
    options.pool_mode_retry_status_codes = statusCodes
  }
  return options
}

export function splitTempUnschedulableKeywords(value: string): string[] {
  return value
    .split(/[,;]/)
    .map((item) => item.trim())
    .filter((item) => item.length > 0)
}

export function buildTempUnschedulableRules(
  rules: TempUnschedRuleForm[],
): TempUnschedulableRule[] {
  const out: TempUnschedulableRule[] = []
  for (const rule of rules) {
    const errorCode = Number(rule.error_code)
    const duration = Number(rule.duration_minutes)
    const keywords = splitTempUnschedulableKeywords(rule.keywords)
    if (!Number.isFinite(errorCode) || errorCode < 100 || errorCode > 599) {
      continue
    }
    if (!Number.isFinite(duration) || duration <= 0 || keywords.length === 0) {
      continue
    }
    out.push({
      error_code: Math.trunc(errorCode),
      keywords,
      duration_minutes: Math.trunc(duration),
      description: rule.description.trim(),
    })
  }
  return out
}

interface BedrockCredentialInput extends PoolModeInput {
  authMode: BedrockAuthMode
  accessKeyId: string
  secretAccessKey: string
  sessionToken: string
  region: string
  forceGlobal: boolean
  apiKey: string
  modelMapping?: Record<string, string> | null
  interceptWarmupRequests: boolean
}

export function buildBedrockCredentials(
  input: BedrockCredentialInput,
): Record<string, unknown> {
  const credentials: Record<string, unknown> = {
    auth_mode: input.authMode,
    aws_region: input.region.trim() || 'us-east-1',
  }
  if (input.authMode === 'sigv4') {
    credentials.aws_access_key_id = input.accessKeyId.trim()
    credentials.aws_secret_access_key = input.secretAccessKey.trim()
    if (input.sessionToken.trim()) {
      credentials.aws_session_token = input.sessionToken.trim()
    }
  } else {
    credentials.api_key = input.apiKey.trim()
  }
  if (input.forceGlobal) credentials.aws_force_global = 'true'
  if (input.modelMapping) credentials.model_mapping = input.modelMapping
  Object.assign(credentials, buildPoolModeCredentialOptions(input))
  if (input.interceptWarmupRequests) {
    credentials.intercept_warmup_requests = true
  }
  return credentials
}

export function buildAntigravityUpstreamCredentials(input: {
  baseUrl: string
  apiKey: string
  modelMapping?: Record<string, string> | null
  interceptWarmupRequests: boolean
}): Record<string, unknown> {
  const credentials: Record<string, unknown> = {
    base_url: input.baseUrl.trim(),
    api_key: input.apiKey.trim(),
  }
  if (input.modelMapping) credentials.model_mapping = input.modelMapping
  if (input.interceptWarmupRequests) {
    credentials.intercept_warmup_requests = true
  }
  return credentials
}

export function buildVertexServiceAccountCredentials(input: {
  serviceAccountJson: string
  projectId: string
  clientEmail: string
  location: string
}): Record<string, unknown> {
  return {
    service_account_json: input.serviceAccountJson.trim(),
    project_id: input.projectId.trim(),
    client_email: input.clientEmail.trim(),
    location: input.location.trim(),
    tier_id: 'vertex',
  }
}

export interface APIKeyCredentialInput extends PoolModeInput {
  platform: AccountPlatform
  baseUrl: string
  apiKey: string
  /** Optional native CN-provider routing metadata. */
  accountMode?: 'payg' | 'coding'
  apiProtocol?: 'chat_completions' | 'anthropic' | 'responses' | 'adaptive'
  zhipuOrganization?: string
  zhipuProject?: string
  geminiTierId?: string
  modelMapping?: Record<string, string> | null
  compactModelMapping?: Record<string, string> | null
  openAIEndpointCapabilities?: OpenAIEndpointCapability[]
  customErrorCodesEnabled: boolean
  customErrorCodes: number[]
  interceptWarmupRequests: boolean
}

export function buildAPIKeyCredentials(
  input: APIKeyCredentialInput,
): Record<string, unknown> {
  const credentials: Record<string, unknown> = {
    base_url: input.baseUrl.trim() || defaultAPIKeyBaseURL(input.platform),
    api_key: input.apiKey.trim(),
  }
  if (input.platform === 'gemini') {
    credentials.tier_id = input.geminiTierId
  }
  if (input.platform === 'zhipu' || input.platform === 'deepseek') {
    if (input.accountMode) credentials.account_mode = input.accountMode
    if (input.apiProtocol) credentials.api_protocol = input.apiProtocol
  }
  if (input.platform === 'zhipu') {
    const organization = input.zhipuOrganization?.trim()
    const project = input.zhipuProject?.trim()
    if (organization) credentials.zhipu_organization = organization
    if (project) credentials.zhipu_project = project
  }
  if (input.modelMapping) credentials.model_mapping = input.modelMapping
  if (input.platform === 'openai') {
    const capabilities = serializeOpenAIEndpointCapabilities(
      normalizeOpenAIEndpointCapabilities(input.openAIEndpointCapabilities || []),
    )
    if (!capabilities.omit) {
      credentials.openai_capabilities = capabilities.values
    }
    if (input.compactModelMapping) {
      credentials.compact_model_mapping = input.compactModelMapping
    }
  }
  Object.assign(credentials, buildPoolModeCredentialOptions(input))
  if (input.customErrorCodesEnabled) {
    credentials.custom_error_codes_enabled = true
    credentials.custom_error_codes = [...input.customErrorCodes]
  }
  if (input.interceptWarmupRequests) {
    credentials.intercept_warmup_requests = true
  }
  return credentials
}

export type VertexServiceAccountParseResult =
  | {
      ok: true
      normalizedJson: string
      projectId: string
      clientEmail: string
    }
  | { ok: false; reason: 'empty' | 'invalid_json' | 'missing_fields' }

export function parseVertexServiceAccountJSON(
  value: string,
): VertexServiceAccountParseResult {
  const raw = value.trim()
  if (!raw) return { ok: false, reason: 'empty' }
  try {
    const parsed = JSON.parse(raw) as Record<string, unknown>
    const projectId =
      typeof parsed.project_id === 'string' ? parsed.project_id.trim() : ''
    const clientEmail =
      typeof parsed.client_email === 'string' ? parsed.client_email.trim() : ''
    const privateKey =
      typeof parsed.private_key === 'string' ? parsed.private_key.trim() : ''
    if (!projectId || !clientEmail || !privateKey) {
      return { ok: false, reason: 'missing_fields' }
    }
    return {
      ok: true,
      normalizedJson: JSON.stringify(parsed),
      projectId,
      clientEmail,
    }
  } catch {
    return { ok: false, reason: 'invalid_json' }
  }
}

export function buildAntigravityExtra(input: {
  mixedScheduling: boolean
  allowOverages: boolean
}): Record<string, unknown> | undefined {
  const extra: Record<string, unknown> = {}
  if (input.mixedScheduling) extra.mixed_scheduling = true
  if (input.allowOverages) extra.allow_overages = true
  return Object.keys(extra).length > 0 ? extra : undefined
}

export interface OpenAIExtraInput {
  platform: AccountPlatform
  accountCategory: CreateAccountCategory
  oauthWebSocketMode?: OpenAIWSMode
  apiKeyWebSocketMode?: OpenAIWSMode
  passthroughEnabled: boolean
  longContextBillingEnabled: boolean
  codexCLIOnlyEnabled: boolean
  codexCLIOnlyAppServerEnabled: boolean
  compactMode: OpenAICompactMode
  responsesMode: OpenAIResponsesMode
  textGenerationCapabilityEnabled: boolean
}

export function buildOpenAIExtra(
  input: OpenAIExtraInput,
  base?: Record<string, unknown>,
): Record<string, unknown> | undefined {
  if (input.platform !== 'openai') return base
  const extra: Record<string, unknown> = { ...(base || {}) }
  if (input.accountCategory === 'oauth-based') {
    const mode = input.oauthWebSocketMode || OPENAI_WS_MODE_OFF
    extra.openai_oauth_responses_websockets_v2_mode = mode
    extra.openai_oauth_responses_websockets_v2_enabled =
      isOpenAIWSModeEnabled(mode)
  } else if (input.accountCategory === 'apikey') {
    const mode = input.apiKeyWebSocketMode || OPENAI_WS_MODE_OFF
    extra.openai_apikey_responses_websockets_v2_mode = mode
    extra.openai_apikey_responses_websockets_v2_enabled =
      isOpenAIWSModeEnabled(mode)
  }
  delete extra.responses_websockets_v2_enabled
  delete extra.openai_ws_enabled
  if (input.passthroughEnabled) extra.openai_passthrough = true
  else {
    delete extra.openai_passthrough
    delete extra.openai_oauth_passthrough
  }
  extra.openai_long_context_billing_enabled =
    input.longContextBillingEnabled
  if (
    input.accountCategory === 'oauth-based' &&
    input.codexCLIOnlyEnabled
  ) {
    extra.codex_cli_only = true
  } else {
    delete extra.codex_cli_only
  }
  delete extra.codex_cli_only_allowed_clients
  if (
    input.accountCategory === 'oauth-based' &&
    input.codexCLIOnlyEnabled &&
    input.codexCLIOnlyAppServerEnabled
  ) {
    extra.codex_cli_only_allow_app_server = true
  } else {
    delete extra.codex_cli_only_allow_app_server
  }
  if (input.compactMode !== 'auto') {
    extra.openai_compact_mode = input.compactMode
  } else {
    delete extra.openai_compact_mode
  }
  if (
    input.accountCategory === 'apikey' &&
    input.textGenerationCapabilityEnabled &&
    input.responsesMode !== 'auto'
  ) {
    extra.openai_responses_mode = input.responsesMode
  } else {
    delete extra.openai_responses_mode
  }
  return Object.keys(extra).length > 0 ? extra : undefined
}

export function buildAnthropicAPIKeyExtra(
  input: {
    platform: AccountPlatform
    accountCategory: CreateAccountCategory
    passthroughEnabled: boolean
    authScheme: AnthropicAPIKeyAuthScheme
    webSearchEmulationMode: string
  },
  base?: Record<string, unknown>,
): Record<string, unknown> | undefined {
  if (input.platform !== 'anthropic' || input.accountCategory !== 'apikey') {
    return base
  }
  const extra: Record<string, unknown> = { ...(base || {}) }
  if (input.passthroughEnabled) extra.anthropic_passthrough = true
  else delete extra.anthropic_passthrough
  if (input.authScheme === 'authorization_bearer') {
    extra.anthropic_apikey_auth_scheme = 'authorization_bearer'
  } else {
    delete extra.anthropic_apikey_auth_scheme
  }
  if (input.webSearchEmulationMode === 'default') {
    delete extra.web_search_emulation
  } else {
    extra.web_search_emulation = input.webSearchEmulationMode
  }
  return Object.keys(extra).length > 0 ? extra : undefined
}

export interface AnthropicOAuthExtraInput {
  windowCostEnabled: boolean
  windowCostLimit: number | null
  windowCostStickyReserve: number | null
  sessionLimitEnabled: boolean
  maxSessions: number | null
  sessionIdleTimeout: number | null
  rpmLimitEnabled: boolean
  baseRpm: number | null
  rpmStrategy: 'tiered' | 'sticky_exempt'
  rpmStickyBuffer: number | null
  userMsgQueueMode: string
  tlsFingerprintEnabled: boolean
  tlsFingerprintProfileId: number | null
  sessionIdMaskingEnabled: boolean
  cacheTTLOverrideEnabled: boolean
  cacheTTLOverrideTarget: string
  customBaseUrlEnabled: boolean
  customBaseUrl: string
}

export function buildAnthropicOAuthExtra(
  base: Record<string, unknown> | undefined,
  input: AnthropicOAuthExtraInput,
): Record<string, unknown> {
  const extra: Record<string, unknown> = { ...(base || {}) }
  if (
    input.windowCostEnabled &&
    input.windowCostLimit != null &&
    input.windowCostLimit > 0
  ) {
    extra.window_cost_limit = input.windowCostLimit
    extra.window_cost_sticky_reserve = input.windowCostStickyReserve ?? 10
  }
  if (
    input.sessionLimitEnabled &&
    input.maxSessions != null &&
    input.maxSessions > 0
  ) {
    extra.max_sessions = input.maxSessions
    extra.session_idle_timeout_minutes = input.sessionIdleTimeout ?? 5
  }
  if (input.rpmLimitEnabled) {
    extra.base_rpm =
      input.baseRpm != null && input.baseRpm > 0 ? input.baseRpm : 15
    extra.rpm_strategy = input.rpmStrategy
    if (input.rpmStickyBuffer != null && input.rpmStickyBuffer > 0) {
      extra.rpm_sticky_buffer = input.rpmStickyBuffer
    }
  }
  if (input.userMsgQueueMode) {
    extra.user_msg_queue_mode = input.userMsgQueueMode
  }
  if (input.tlsFingerprintEnabled) {
    extra.enable_tls_fingerprint = true
    if (input.tlsFingerprintProfileId) {
      extra.tls_fingerprint_profile_id = input.tlsFingerprintProfileId
    }
  }
  if (input.sessionIdMaskingEnabled) {
    extra.session_id_masking_enabled = true
  }
  if (input.cacheTTLOverrideEnabled) {
    extra.cache_ttl_override_enabled = true
    extra.cache_ttl_override_target = input.cacheTTLOverrideTarget
  }
  if (input.customBaseUrlEnabled && input.customBaseUrl.trim()) {
    extra.custom_base_url_enabled = true
    extra.custom_base_url = input.customBaseUrl.trim()
  }
  return extra
}

export interface APIKeyQuotaExtraInput {
  quotaLimit: number | null
  quotaDailyLimit: number | null
  quotaWeeklyLimit: number | null
  dailyResetMode: 'rolling' | 'fixed' | null
  dailyResetHour: number | null
  weeklyResetMode: 'rolling' | 'fixed' | null
  weeklyResetDay: number | null
  weeklyResetHour: number | null
  resetTimezone: string | null
}

export function buildAPIKeyQuotaExtra(
  base: Record<string, unknown> | undefined,
  input: APIKeyQuotaExtraInput,
): Record<string, unknown> {
  const extra: Record<string, unknown> = { ...(base || {}) }
  if (input.quotaLimit != null && input.quotaLimit > 0) {
    extra.quota_limit = input.quotaLimit
  }
  if (input.quotaDailyLimit != null && input.quotaDailyLimit > 0) {
    extra.quota_daily_limit = input.quotaDailyLimit
  }
  if (input.quotaWeeklyLimit != null && input.quotaWeeklyLimit > 0) {
    extra.quota_weekly_limit = input.quotaWeeklyLimit
  }
  if (input.dailyResetMode === 'fixed') {
    extra.quota_daily_reset_mode = 'fixed'
    extra.quota_daily_reset_hour = input.dailyResetHour ?? 0
  }
  if (input.weeklyResetMode === 'fixed') {
    extra.quota_weekly_reset_mode = 'fixed'
    extra.quota_weekly_reset_day = input.weeklyResetDay ?? 1
    extra.quota_weekly_reset_hour = input.weeklyResetHour ?? 0
  }
  if (
    input.dailyResetMode === 'fixed' ||
    input.weeklyResetMode === 'fixed'
  ) {
    extra.quota_reset_timezone = input.resetTimezone || 'UTC'
  }
  return extra
}

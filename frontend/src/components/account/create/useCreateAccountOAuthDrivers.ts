import { computed, ref } from 'vue'
import type { AccountPlatform } from '@/types'
import {
  useAccountOAuth,
  type AddMethod,
} from '@/composables/useAccountOAuth'
import { useOpenAIOAuth } from '@/composables/useOpenAIOAuth'
import { useGeminiOAuth } from '@/composables/useGeminiOAuth'
import { useAntigravityOAuth } from '@/composables/useAntigravityOAuth'
import { useGrokOAuth } from '@/composables/useGrokOAuth'
import {
  OAuthDriverRegistry,
  type OAuthAuthorizationContext,
} from './oauthDriverRegistry'

type OAuthCodeHandlers = {
  [Platform in AccountPlatform]: (code: string) => Promise<void>
}

type OAuthRefreshTokenHandlers = Partial<{
  [Platform in AccountPlatform]: (refreshToken: string) => Promise<void>
}>

export interface CreateAccountOAuthDriverOptions {
  platform(): AccountPlatform
  anthropicAddMethod(): AddMethod
  exchangeAuthorizationCode: OAuthCodeHandlers
  validateRefreshToken?: OAuthRefreshTokenHandlers
}

/**
 * Owns the five platform OAuth clients and projects them through one registry.
 * The modal supplies only orchestration callbacks that finish account creation.
 */
export function useCreateAccountOAuthDrivers(
  options: CreateAccountOAuthDriverOptions,
) {
  const anthropicOAuth = useAccountOAuth()
  const openaiOAuth = useOpenAIOAuth()
  const geminiOAuth = useGeminiOAuth()
  const antigravityOAuth = useAntigravityOAuth()
  const grokOAuth = useGrokOAuth()

  // DeepSeek accounts are API-key based and do not have an OAuth flow.  Keep
  // a inert driver in the registry so the discriminated AccountPlatform map
  // remains total without accidentally reusing another provider's OAuth
  // state when the modal switches platforms.
  const deepseekOAuth = {
    authUrl: ref(''),
    sessionId: ref(''),
    loading: ref(false),
    error: ref(''),
    generateAuthorization: async () => undefined,
    exchangeAuthorizationCode: options.exchangeAuthorizationCode.deepseek,
    reset: () => undefined,
  }

  const kimiOAuth = {
    authUrl: ref(''), sessionId: ref(''), loading: ref(false), error: ref(''),
    generateAuthorization: async () => undefined,
    exchangeAuthorizationCode: options.exchangeAuthorizationCode.kimi,
    reset: () => undefined,
  }

  const zhipuOAuth = {
    authUrl: ref(''),
    sessionId: ref(''),
    loading: ref(false),
    error: ref(''),
    generateAuthorization: async () => undefined,
    exchangeAuthorizationCode: options.exchangeAuthorizationCode.zhipu,
    reset: () => undefined,
  }

  const registry = new OAuthDriverRegistry({
    anthropic: {
      authUrl: anthropicOAuth.authUrl,
      sessionId: anthropicOAuth.sessionId,
      loading: anthropicOAuth.loading,
      error: anthropicOAuth.error,
      generateAuthorization: async ({ proxyId }) => {
        await anthropicOAuth.generateAuthUrl(
          options.anthropicAddMethod(),
          proxyId,
        )
      },
      exchangeAuthorizationCode:
        options.exchangeAuthorizationCode.anthropic,
      validateRefreshToken: options.validateRefreshToken?.anthropic,
      reset: () => anthropicOAuth.resetState(),
    },
    openai: {
      authUrl: openaiOAuth.authUrl,
      sessionId: openaiOAuth.sessionId,
      loading: openaiOAuth.loading,
      error: openaiOAuth.error,
      generateAuthorization: async ({ proxyId }) => {
        await openaiOAuth.generateAuthUrl(proxyId)
      },
      exchangeAuthorizationCode: options.exchangeAuthorizationCode.openai,
      validateRefreshToken: options.validateRefreshToken?.openai,
      reset: () => openaiOAuth.resetState(),
    },
    gemini: {
      authUrl: geminiOAuth.authUrl,
      sessionId: geminiOAuth.sessionId,
      loading: geminiOAuth.loading,
      error: geminiOAuth.error,
      generateAuthorization: async ({ proxyId, projectId, oauthType, tier }) => {
        await geminiOAuth.generateAuthUrl(
          proxyId,
          projectId,
          oauthType,
          tier,
        )
      },
      exchangeAuthorizationCode: options.exchangeAuthorizationCode.gemini,
      validateRefreshToken: options.validateRefreshToken?.gemini,
      reset: () => geminiOAuth.resetState(),
    },
    antigravity: {
      authUrl: antigravityOAuth.authUrl,
      sessionId: antigravityOAuth.sessionId,
      loading: antigravityOAuth.loading,
      error: antigravityOAuth.error,
      generateAuthorization: async ({ proxyId }) => {
        await antigravityOAuth.generateAuthUrl(proxyId)
      },
      exchangeAuthorizationCode:
        options.exchangeAuthorizationCode.antigravity,
      validateRefreshToken: options.validateRefreshToken?.antigravity,
      reset: () => antigravityOAuth.resetState(),
    },
    grok: {
      authUrl: grokOAuth.authUrl,
      sessionId: grokOAuth.sessionId,
      loading: grokOAuth.loading,
      error: grokOAuth.error,
      generateAuthorization: async ({ proxyId }) => {
        await grokOAuth.generateAuthUrl(proxyId)
      },
      exchangeAuthorizationCode: options.exchangeAuthorizationCode.grok,
      validateRefreshToken: options.validateRefreshToken?.grok,
      reset: () => grokOAuth.resetState(),
    },
    kimi: kimiOAuth,
    deepseek: deepseekOAuth,
    zhipu: zhipuOAuth,
  })

  const currentDriver = computed(() => registry.get(options.platform()))
  const currentAuthUrl = computed(() => currentDriver.value.authUrl.value)
  const currentSessionId = computed(() => currentDriver.value.sessionId.value)
  const currentLoading = computed(() => currentDriver.value.loading.value)
  const currentError = computed(() => currentDriver.value.error.value)

  const generateAuthorization = (context: OAuthAuthorizationContext) =>
    currentDriver.value.generateAuthorization(context)

  const exchangeAuthorizationCode = (code: string) =>
    registry.exchangeAuthorizationCode(options.platform(), code)

  const validateRefreshToken = (refreshToken: string) =>
    registry.validateRefreshToken(options.platform(), refreshToken)

  return {
    anthropicOAuth,
    openaiOAuth,
    geminiOAuth,
    antigravityOAuth,
    grokOAuth,
    deepseekOAuth,
    zhipuOAuth,
    registry,
    currentAuthUrl,
    currentSessionId,
    currentLoading,
    currentError,
    generateAuthorization,
    exchangeAuthorizationCode,
    validateRefreshToken,
  }
}

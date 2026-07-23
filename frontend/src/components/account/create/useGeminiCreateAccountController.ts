import { ref, watch, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { useGeminiOAuth } from '@/composables/useGeminiOAuth'
import { useAppStore } from '@/stores/app'
import type { AccountPlatform } from '@/types'
import type {
  CreateAccountCategory,
  GeminiOAuthType,
} from './formDraft'

type GeminiOAuthClient = ReturnType<typeof useGeminiOAuth>

export interface GeminiCreateAccountControllerOptions {
  oauth: GeminiOAuthClient
  show(): boolean
  platform(): AccountPlatform
  category(): CreateAccountCategory
  proxyId(): number | null
  oauthState(): string
  selectedTier(): string
  oauthType: Ref<GeminiOAuthType>
  createAndFinish(
    credentials: Record<string, unknown>,
    extra?: Record<string, unknown>,
  ): Promise<void>
}

/** Owns Gemini capability gating and authorization-code exchange effects. */
export function useGeminiCreateAccountController(
  options: GeminiCreateAccountControllerOptions,
) {
  const appStore = useAppStore()
  const { t } = useI18n()
  const aiStudioOAuthEnabled = ref(false)

  watch(
    [options.show, options.platform, options.category],
    async ([show, platform, category]) => {
      if (!show || platform !== 'gemini' || category !== 'oauth-based') {
        aiStudioOAuthEnabled.value = false
        return
      }
      const capabilities = await options.oauth.getCapabilities()
      aiStudioOAuthEnabled.value =
        !!capabilities?.ai_studio_oauth_enabled
      if (
        !aiStudioOAuthEnabled.value &&
        options.oauthType.value === 'ai_studio'
      ) {
        options.oauthType.value = 'code_assist'
      }
    },
    { immediate: true },
  )

  const handleSelectOAuthType = (oauthType: GeminiOAuthType) => {
    if (oauthType === 'ai_studio' && !aiStudioOAuthEnabled.value) {
      appStore.showError(
        t('admin.accounts.oauth.gemini.aiStudioNotConfigured'),
      )
      return
    }
    options.oauthType.value = oauthType
  }

  const handleExchangeAuthorizationCode = async (authCode: string) => {
    if (!authCode.trim() || !options.oauth.sessionId.value) return

    options.oauth.loading.value = true
    options.oauth.error.value = ''

    try {
      const stateFromInput = options.oauthState()
      const stateToUse = stateFromInput || options.oauth.state.value
      if (!stateToUse) {
        options.oauth.error.value = t('admin.accounts.oauth.authFailed')
        appStore.showError(options.oauth.error.value)
        return
      }

      const tokenInfo = await options.oauth.exchangeAuthCode({
        code: authCode.trim(),
        sessionId: options.oauth.sessionId.value,
        state: stateToUse,
        proxyId: options.proxyId(),
        oauthType: options.oauthType.value,
        tierId: options.selectedTier(),
      })
      if (!tokenInfo) return

      const credentials = options.oauth.buildCredentials(tokenInfo)
      const extra = options.oauth.buildExtraInfo(tokenInfo)
      await options.createAndFinish(credentials, extra)
    } catch (error: any) {
      options.oauth.error.value =
        error.response?.data?.detail || t('admin.accounts.oauth.authFailed')
      appStore.showError(options.oauth.error.value)
    } finally {
      options.oauth.loading.value = false
    }
  }

  return {
    aiStudioOAuthEnabled,
    handleSelectOAuthType,
    handleExchangeAuthorizationCode,
  }
}

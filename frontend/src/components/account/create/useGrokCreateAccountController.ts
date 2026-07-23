import { ref, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { useGrokOAuth } from '@/composables/useGrokOAuth'
import { buildModelMappingObject } from '@/composables/useModelWhitelist'
import { useAppStore } from '@/stores/app'
import type { CreateAccountRequest } from '@/types'
import {
  applyHeaderOverride,
  validateHeaderOverrideRows,
  type HeaderOverrideRow,
} from '../credentialsBuilder'
import {
  buildGrokSSOImportRequest,
  type AccountBaseDraft,
} from './accountDraft'
import type {
  AccountModelMappingDraft,
  ModelRestrictionMode,
} from './formDraft'

type GrokOAuthClient = ReturnType<typeof useGrokOAuth>

export interface GrokCreateAccountControllerOptions {
  oauth: GrokOAuthClient
  proxyId(): number | null
  accountName(): string
  oauthState(): string
  headerOverrideEnabled: Ref<boolean>
  headerOverrideRows: Ref<HeaderOverrideRow[]>
  modelRestrictionMode: Ref<ModelRestrictionMode>
  allowedModels: Ref<string[]>
  modelMappings: Ref<AccountModelMappingDraft[]>
  buildBaseDraft(): AccountBaseDraft
  buildPayload(
    credentials: Record<string, unknown>,
    extra?: Record<string, unknown>,
    baseOverrides?: Partial<AccountBaseDraft>,
  ): CreateAccountRequest
  applyTempUnschedulableConfig(credentials: Record<string, unknown>): boolean
  createAndFinish(
    credentials: Record<string, unknown>,
    extra?: Record<string, unknown>,
  ): Promise<void>
  notifyCreated(): void
  close(): void
}

/**
 * Owns Grok's three OAuth creation side-effect paths. The modal keeps the
 * wizard bindings, while validation, exchange/import calls and result
 * reporting stay consistent across authorization code, RT and SSO inputs.
 */
export function useGrokCreateAccountController(
  options: GrokCreateAccountControllerOptions,
) {
  const appStore = useAppStore()
  const { t } = useI18n()
  const customBaseUrlEnabled = ref(false)
  const baseUrl = ref('')

  const resetConfiguration = () => {
    customBaseUrlEnabled.value = false
    baseUrl.value = ''
  }

  const validateUpstreamConfig = (): boolean => {
    if (customBaseUrlEnabled.value) {
      const trimmed = baseUrl.value.trim()
      if (!trimmed) {
        appStore.showError(t('admin.accounts.grokCustomBaseUrl.required'))
        return false
      }
      if (!/^https?:\/\//i.test(trimmed)) {
        appStore.showError(t('admin.accounts.grokCustomBaseUrl.invalid'))
        return false
      }
    }
    if (options.headerOverrideEnabled.value) {
      const headerError = validateHeaderOverrideRows(
        options.headerOverrideRows.value,
      )
      if (headerError) {
        appStore.showError(t(`admin.accounts.headerOverride.${headerError}`))
        return false
      }
    }
    return true
  }

  const applyUpstreamConfig = (credentials: Record<string, unknown>) => {
    if (customBaseUrlEnabled.value) {
      credentials.base_url = baseUrl.value.trim()
    }
    applyHeaderOverride(
      credentials,
      options.headerOverrideEnabled.value,
      options.headerOverrideRows.value,
      'create',
    )
  }

  const applyModelMapping = (credentials: Record<string, unknown>) => {
    const modelMapping = buildModelMappingObject(
      options.modelRestrictionMode.value,
      options.allowedModels.value,
      options.modelMappings.value,
    )
    if (modelMapping) {
      credentials.model_mapping = modelMapping
    }
  }

  const handleValidateRefreshToken = async (refreshTokenInput: string) => {
    if (!refreshTokenInput.trim()) return

    const refreshTokens = refreshTokenInput
      .split('\n')
      .map((refreshToken) => refreshToken.trim())
      .filter(Boolean)

    if (refreshTokens.length === 0) {
      options.oauth.error.value = t(
        'admin.accounts.oauth.grok.pleaseEnterRefreshToken',
      )
      return
    }
    if (!validateUpstreamConfig()) return

    options.oauth.loading.value = true
    options.oauth.error.value = ''

    let successCount = 0
    let failedCount = 0
    const errors: string[] = []

    try {
      for (let index = 0; index < refreshTokens.length; index++) {
        try {
          const tokenInfo = await options.oauth.validateRefreshToken(
            refreshTokens[index],
            options.proxyId(),
          )
          if (!tokenInfo) {
            failedCount++
            errors.push(
              `#${index + 1}: ${options.oauth.error.value || 'Validation failed'}`,
            )
            options.oauth.error.value = ''
            continue
          }

          const credentials = options.oauth.buildCredentials(tokenInfo)
          applyUpstreamConfig(credentials)
          const extra = options.oauth.buildExtraInfo(tokenInfo)
          const fallbackName = tokenInfo.email || 'Grok OAuth Account'
          const baseName = options.accountName() || fallbackName
          const accountName =
            refreshTokens.length > 1
              ? `${baseName} #${index + 1}`
              : baseName

          applyModelMapping(credentials)
          if (!options.applyTempUnschedulableConfig(credentials)) {
            return
          }

          await adminAPI.accounts.create(
            options.buildPayload(credentials, extra, { name: accountName }),
          )
          successCount++
        } catch (error: any) {
          failedCount++
          const errorMessage =
            error.response?.data?.detail || error.message || 'Unknown error'
          errors.push(`#${index + 1}: ${errorMessage}`)
        }
      }

      if (successCount > 0 && failedCount === 0) {
        appStore.showSuccess(
          refreshTokens.length > 1
            ? t('admin.accounts.oauth.batchSuccess', { count: successCount })
            : t('admin.accounts.accountCreated'),
        )
        options.notifyCreated()
        options.close()
      } else if (successCount > 0) {
        appStore.showWarning(
          t('admin.accounts.oauth.batchPartialSuccess', {
            success: successCount,
            failed: failedCount,
          }),
        )
        options.oauth.error.value = errors.join('\n')
        options.notifyCreated()
      } else {
        options.oauth.error.value = errors.join('\n')
        appStore.showError(t('admin.accounts.oauth.batchFailed'))
      }
    } finally {
      options.oauth.loading.value = false
    }
  }

  const handleImportSSO = async (ssoInput: string) => {
    const ssoTokens = ssoInput
      .split('\n')
      .map((token) => token.trim())
      .filter(Boolean)
    if (ssoTokens.length === 0) return
    if (!validateUpstreamConfig()) return

    options.oauth.loading.value = true
    options.oauth.error.value = ''

    const credentials: Record<string, unknown> = {}
    applyUpstreamConfig(credentials)
    applyModelMapping(credentials)
    if (!options.applyTempUnschedulableConfig(credentials)) {
      options.oauth.loading.value = false
      return
    }

    try {
      const result = await adminAPI.grok.createFromSSO(
        buildGrokSSOImportRequest(
          options.buildBaseDraft(),
          ssoTokens,
          credentials,
        ),
      )

      const successCount = result.created?.length || 0
      const failedCount = result.failed?.length || 0
      if (successCount > 0 && failedCount === 0) {
        appStore.showSuccess(
          ssoTokens.length > 1
            ? t('admin.accounts.oauth.batchSuccess', { count: successCount })
            : t('admin.accounts.accountCreated'),
        )
        options.notifyCreated()
        options.close()
      } else if (successCount > 0 && failedCount > 0) {
        appStore.showWarning(
          t('admin.accounts.oauth.batchPartialSuccess', {
            success: successCount,
            failed: failedCount,
          }),
        )
        options.oauth.error.value = (result.failed || [])
          .map((item) => `#${item.index}: ${item.error || 'Unknown error'}`)
          .join('\n')
        options.notifyCreated()
      } else {
        options.oauth.error.value =
          (result.failed || [])
            .map((item) => `#${item.index}: ${item.error || 'Unknown error'}`)
            .join('\n') || t('admin.accounts.oauth.grok.failedToConvertSSO')
        appStore.showError(t('admin.accounts.oauth.batchFailed'))
      }
    } catch (error: any) {
      options.oauth.error.value =
        error.response?.data?.detail ||
        error.message ||
        t('admin.accounts.oauth.grok.failedToConvertSSO')
      appStore.showError(options.oauth.error.value)
    } finally {
      options.oauth.loading.value = false
    }
  }

  const handleExchangeAuthorizationCode = async (authCode: string) => {
    if (!authCode.trim() || !options.oauth.sessionId.value) return
    if (!validateUpstreamConfig()) return

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
      })
      if (!tokenInfo) return

      const credentials = options.oauth.buildCredentials(tokenInfo)
      applyUpstreamConfig(credentials)
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
    customBaseUrlEnabled,
    baseUrl,
    resetConfiguration,
    validateUpstreamConfig,
    applyUpstreamConfig,
    handleValidateRefreshToken,
    handleImportSSO,
    handleExchangeAuthorizationCode,
  }
}

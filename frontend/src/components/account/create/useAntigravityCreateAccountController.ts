import type { Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { useAntigravityOAuth } from '@/composables/useAntigravityOAuth'
import { buildModelMappingObject } from '@/composables/useModelWhitelist'
import { useAppStore } from '@/stores/app'
import type { CreateAccountRequest } from '@/types'
import {
  applyAntigravityProjectID,
  applyInterceptWarmup,
} from '../credentialsBuilder'
import type { AccountBaseDraft } from './accountDraft'
import type { AccountModelMappingDraft } from './formDraft'

type AntigravityOAuthClient = ReturnType<typeof useAntigravityOAuth>

export interface AntigravityCreateAccountControllerOptions {
  oauth: AntigravityOAuthClient
  proxyId(): number | null
  accountName(): string
  oauthState(): string
  projectId: Ref<string>
  interceptWarmupRequests: Ref<boolean>
  modelMappings: Ref<AccountModelMappingDraft[]>
  buildExtra(): Record<string, unknown> | undefined
  buildPayload(
    credentials: Record<string, unknown>,
    extra?: Record<string, unknown>,
    baseOverrides?: Partial<AccountBaseDraft>,
  ): CreateAccountRequest
  withConfirmFlag(payload: CreateAccountRequest): CreateAccountRequest
  createAndFinish(
    credentials: Record<string, unknown>,
    extra?: Record<string, unknown>,
  ): Promise<void>
  notifyCreated(): void
  close(): void
}

/** Owns Antigravity RT-batch and authorization-code creation effects. */
export function useAntigravityCreateAccountController(
  options: AntigravityCreateAccountControllerOptions,
) {
  const appStore = useAppStore()
  const { t } = useI18n()

  const handleValidateRefreshToken = async (refreshTokenInput: string) => {
    if (!refreshTokenInput.trim()) return

    const refreshTokens = refreshTokenInput
      .split('\n')
      .map((refreshToken) => refreshToken.trim())
      .filter(Boolean)

    if (refreshTokens.length === 0) {
      options.oauth.error.value = t(
        'admin.accounts.oauth.antigravity.pleaseEnterRefreshToken',
      )
      return
    }

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

          const credentials = options.oauth.buildCredentials(
            tokenInfo,
            refreshTokens[index],
          )
          applyAntigravityProjectID(
            credentials,
            options.projectId.value,
            'create',
          )

          const accountName =
            refreshTokens.length > 1
              ? `${options.accountName()} #${index + 1}`
              : options.accountName()
          const payload = options.withConfirmFlag(
            options.buildPayload(credentials, {}, { name: accountName }),
          )
          await adminAPI.accounts.create(payload)
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
      } else if (successCount > 0 && failedCount > 0) {
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
      })
      if (!tokenInfo) return

      const credentials = options.oauth.buildCredentials(tokenInfo)
      applyAntigravityProjectID(
        credentials,
        options.projectId.value,
        'create',
      )
      applyInterceptWarmup(
        credentials,
        options.interceptWarmupRequests.value,
        'create',
      )
      const modelMapping = buildModelMappingObject(
        'mapping',
        [],
        options.modelMappings.value,
      )
      if (modelMapping) {
        credentials.model_mapping = modelMapping
      }
      await options.createAndFinish(credentials, options.buildExtra())
    } catch (error: any) {
      options.oauth.error.value =
        error.response?.data?.detail || t('admin.accounts.oauth.authFailed')
      appStore.showError(options.oauth.error.value)
    } finally {
      options.oauth.loading.value = false
    }
  }

  return {
    handleValidateRefreshToken,
    handleExchangeAuthorizationCode,
  }
}

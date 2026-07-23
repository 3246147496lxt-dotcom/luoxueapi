import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { useOpenAIOAuth } from '@/composables/useOpenAIOAuth'
import type { AuthInputMethod } from '@/composables/useAccountOAuth'
import { useAppStore } from '@/stores/app'
import type {
  AccountPlatform,
  CreateAccountRequest,
  CodexSessionImportMessage,
} from '@/types'
import {
  buildOpenAICodexPATCreateRequest,
  buildOpenAICodexSessionImportRequest,
  type AccountBaseDraft,
} from './accountDraft'

type OpenAIOAuthClient = ReturnType<typeof useOpenAIOAuth>

export interface OpenAICreateAccountControllerOptions {
  oauth: OpenAIOAuthClient
  platform(): AccountPlatform
  proxyId(): number | null
  accountName(): string
  oauthState(): string
  inputMethod(): AuthInputMethod | undefined
  buildBaseDraft(): AccountBaseDraft
  buildPayload(
    credentials: Record<string, unknown>,
    extra?: Record<string, unknown>,
    baseOverrides?: Partial<AccountBaseDraft>,
  ): CreateAccountRequest
  buildExtra(base?: Record<string, unknown>): Record<string, unknown> | undefined
  buildImportExtra(): Record<string, unknown> | undefined
  prepareCredentials(credentials: Record<string, unknown>): boolean
  notifyCreated(): void
  close(): void
}

const OPENAI_MOBILE_RT_CLIENT_ID = 'app_LlGpXReQgckcGGUo2JrYvtJK'

const formatCodexImportMessages = (
  messages?: CodexSessionImportMessage[],
) =>
  (messages || [])
    .map((item) => {
      const name = item.name ? ` ${item.name}` : ''
      return `#${item.index}${name}: ${item.message}`
    })
    .join('\n')

const isAgentIdentityImportContent = (content: string) => {
  const isAgentIdentityValue = (value: unknown): boolean => {
    if (Array.isArray(value)) {
      return value.length > 0 && value.every(isAgentIdentityValue)
    }
    if (!value || typeof value !== 'object') return false
    const record = value as Record<string, unknown>
    const authMode = record.auth_mode ?? record.authMode
    const agentIdentity = record.agent_identity ?? record.agentIdentity
    return (
      (typeof authMode === 'string' &&
        authMode.toLowerCase() === 'agentidentity') ||
      (!!agentIdentity && typeof agentIdentity === 'object')
    )
  }

  try {
    return isAgentIdentityValue(JSON.parse(content))
  } catch {
    const lines = content
      .split('\n')
      .map((line) => line.trim())
      .filter(Boolean)
    if (lines.length === 0) return false
    try {
      return lines.every((line) =>
        isAgentIdentityValue(JSON.parse(line)),
      )
    } catch {
      return false
    }
  }
}

/** Owns all OpenAI create/import effects used by the account wizard. */
export function useOpenAICreateAccountController(
  options: OpenAICreateAccountControllerOptions,
) {
  const appStore = useAppStore()
  const { t } = useI18n()

  const handleExchangeAuthorizationCode = async (authCode: string) => {
    if (!authCode.trim() || !options.oauth.sessionId.value) return

    options.oauth.loading.value = true
    options.oauth.error.value = ''
    try {
      const stateToUse = (
        options.oauthState() || options.oauth.oauthState.value || ''
      ).trim()
      if (!stateToUse) {
        options.oauth.error.value = t('admin.accounts.oauth.authFailed')
        appStore.showError(options.oauth.error.value)
        return
      }

      const tokenInfo = await options.oauth.exchangeAuthCode(
        authCode.trim(),
        options.oauth.sessionId.value,
        stateToUse,
        options.proxyId(),
      )
      if (!tokenInfo) return

      const credentials = options.oauth.buildCredentials(tokenInfo)
      if (!options.prepareCredentials(credentials)) return
      const oauthExtra = options.oauth.buildExtraInfo(tokenInfo) as
        | Record<string, unknown>
        | undefined

      if (options.platform() === 'openai') {
        await adminAPI.accounts.create(
          options.buildPayload(credentials, options.buildExtra(oauthExtra)),
        )
        appStore.showSuccess(t('admin.accounts.accountCreated'))
      }
      options.notifyCreated()
      options.close()
    } catch (error: any) {
      options.oauth.error.value =
        error.response?.data?.detail || t('admin.accounts.oauth.authFailed')
      appStore.showError(options.oauth.error.value)
    } finally {
      options.oauth.loading.value = false
    }
  }

  const buildImportCredentialExtras = () => {
    const credentials: Record<string, unknown> = {}
    return options.prepareCredentials(credentials) ? credentials : null
  }

  const handleImportCodexSession = async (content: string) => {
    const trimmed = content.trim()
    if (!trimmed) {
      options.oauth.error.value = t(
        'admin.accounts.oauth.openai.codexSessionEmpty',
      )
      return
    }
    if (
      options.inputMethod() === 'agent_identity' &&
      !isAgentIdentityImportContent(trimmed)
    ) {
      options.oauth.error.value = t(
        'admin.accounts.oauth.openai.agentIdentityInvalid',
      )
      return
    }

    const credentialExtras = buildImportCredentialExtras()
    if (!credentialExtras) return

    options.oauth.loading.value = true
    options.oauth.error.value = ''
    try {
      const result = await adminAPI.accounts.importCodexSession(
        buildOpenAICodexSessionImportRequest(
          options.buildBaseDraft(),
          trimmed,
          credentialExtras,
          options.buildImportExtra(),
        ),
      )
      const successCount = result.created + result.updated
      const params = {
        created: result.created,
        updated: result.updated,
        skipped: result.skipped,
        failed: result.failed,
      }

      if (successCount > 0 && result.failed === 0) {
        appStore.showSuccess(
          t('admin.accounts.oauth.openai.codexSessionImportSuccess', params),
        )
        options.notifyCreated()
        options.close()
        return
      }

      const errorText = formatCodexImportMessages(result.errors)
      const warningText = formatCodexImportMessages(result.warnings)
      options.oauth.error.value = [errorText, warningText]
        .filter(Boolean)
        .join('\n')

      if (result.failed === 0) {
        appStore.showWarning(
          t('admin.accounts.oauth.openai.codexSessionImportSuccess', params),
        )
      } else if (successCount > 0) {
        appStore.showWarning(
          t('admin.accounts.oauth.openai.codexSessionImportPartial', params),
        )
        options.notifyCreated()
      } else {
        appStore.showError(
          t('admin.accounts.oauth.openai.codexSessionImportFailed'),
        )
      }
    } catch (error: any) {
      options.oauth.error.value =
        error.response?.data?.detail ||
        error.response?.data?.message ||
        error.message ||
        t('admin.accounts.oauth.openai.codexSessionImportFailed')
      appStore.showError(options.oauth.error.value)
    } finally {
      options.oauth.loading.value = false
    }
  }

  const handleImportCodexPAT = async (accessToken: string) => {
    const trimmed = accessToken.trim()
    if (!trimmed) {
      options.oauth.error.value = t(
        'admin.accounts.oauth.openai.codexPatEmpty',
      )
      return
    }
    const credentialExtras = buildImportCredentialExtras()
    if (!credentialExtras) return

    options.oauth.loading.value = true
    options.oauth.error.value = ''
    try {
      await adminAPI.accounts.createOpenAICodexPAT(
        buildOpenAICodexPATCreateRequest(
          options.buildBaseDraft(),
          trimmed,
          credentialExtras,
          options.buildImportExtra(),
        ),
      )
      appStore.showSuccess(t('admin.accounts.messages.accountCreated'))
      options.notifyCreated()
      options.close()
    } catch (error: any) {
      options.oauth.error.value =
        error.response?.data?.detail ||
        error.response?.data?.message ||
        error.message ||
        t('admin.accounts.oauth.openai.codexPatImportFailed')
      appStore.showError(options.oauth.error.value)
    } finally {
      options.oauth.loading.value = false
    }
  }

  const handleBatchRefreshTokens = async (
    refreshTokenInput: string,
    clientId?: string,
  ) => {
    if (!refreshTokenInput.trim()) return
    const refreshTokens = refreshTokenInput
      .split('\n')
      .map((refreshToken) => refreshToken.trim())
      .filter(Boolean)
    if (refreshTokens.length === 0) {
      options.oauth.error.value = t(
        'admin.accounts.oauth.openai.pleaseEnterRefreshToken',
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
            clientId,
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
          if (clientId) credentials.client_id = clientId
          if (!options.prepareCredentials(credentials)) return
          const oauthExtra = options.oauth.buildExtraInfo(tokenInfo) as
            | Record<string, unknown>
            | undefined
          const baseName =
            options.accountName() || tokenInfo.email || 'OpenAI OAuth Account'
          const accountName =
            refreshTokens.length > 1
              ? `${baseName} #${index + 1}`
              : baseName

          if (options.platform() === 'openai') {
            await adminAPI.accounts.create(
              options.buildPayload(
                credentials,
                options.buildExtra(oauthExtra),
                { name: accountName },
              ),
            )
          }
          successCount++
        } catch (error: any) {
          failedCount++
          errors.push(
            `#${index + 1}: ${error.response?.data?.detail || error.message || 'Unknown error'}`,
          )
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

  return {
    handleExchangeAuthorizationCode,
    handleImportCodexSession,
    handleImportCodexPAT,
    handleValidateRefreshToken: (refreshToken: string) =>
      handleBatchRefreshTokens(refreshToken),
    handleValidateMobileRefreshToken: (refreshToken: string) =>
      handleBatchRefreshTokens(refreshToken, OPENAI_MOBILE_RT_CLIENT_ID),
  }
}

import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { useAccountOAuth } from '@/composables/useAccountOAuth'
import { useAppStore } from '@/stores/app'
import type { AccountPlatform, AccountType, CreateAccountRequest } from '@/types'
import type { AddMethod } from '@/composables/useAccountOAuth'
import type { AccountBaseDraft } from './accountDraft'

type AnthropicOAuthClient = ReturnType<typeof useAccountOAuth>

export interface AnthropicCreateAccountControllerOptions {
  oauth: AnthropicOAuthClient
  platform(): AccountPlatform
  addMethod(): AddMethod
  proxyId(): number | null
  accountName(): string
  buildExtra(base?: Record<string, unknown>): Record<string, unknown> | undefined
  buildCredentialDefaults(): Record<string, unknown> | null
  buildPayload(
    credentials: Record<string, unknown>,
    extra?: Record<string, unknown>,
    baseOverrides?: Partial<AccountBaseDraft>,
  ): CreateAccountRequest
  createAndFinish(
    platform: AccountPlatform,
    type: AccountType,
    credentials: Record<string, unknown>,
    extra?: Record<string, unknown>,
  ): Promise<void>
  notifyCreated(): void
  close(): void
}

/** Owns Anthropic authorization-code and cookie-key create effects. */
export function useAnthropicCreateAccountController(
  options: AnthropicCreateAccountControllerOptions,
) {
  const appStore = useAppStore()
  const { t } = useI18n()

  const endpointFor = (cookie: boolean) => {
    if (cookie) {
      return options.addMethod() === 'oauth'
        ? '/admin/accounts/cookie-auth'
        : '/admin/accounts/setup-token-cookie-auth'
    }
    return options.addMethod() === 'oauth'
      ? '/admin/accounts/exchange-code'
      : '/admin/accounts/exchange-setup-token-code'
  }

  const exchangeToken = (endpoint: string, code: string, sessionId: string) => {
    const proxyId = options.proxyId()
    return adminAPI.accounts.exchangeCode(endpoint, {
      session_id: sessionId,
      code,
      ...(proxyId ? { proxy_id: proxyId } : {}),
    })
  }

  const handleExchangeAuthorizationCode = async (authCode: string) => {
    if (!authCode.trim() || !options.oauth.sessionId.value) return

    options.oauth.loading.value = true
    options.oauth.error.value = ''
    try {
      const defaults = options.buildCredentialDefaults()
      if (!defaults) return
      const tokenInfo = await exchangeToken(
        endpointFor(false),
        authCode.trim(),
        options.oauth.sessionId.value,
      )
      const extra = options.buildExtra(options.oauth.buildExtraInfo(tokenInfo))
      await options.createAndFinish(
        options.platform(),
        options.addMethod() as AccountType,
        { ...tokenInfo, ...defaults },
        extra,
      )
    } catch (error: any) {
      options.oauth.error.value =
        error.response?.data?.detail || t('admin.accounts.oauth.authFailed')
      appStore.showError(options.oauth.error.value)
    } finally {
      options.oauth.loading.value = false
    }
  }

  const handleCookieAuth = async (sessionKey: string) => {
    options.oauth.loading.value = true
    options.oauth.error.value = ''
    try {
      const keys = options.oauth.parseSessionKeys(sessionKey)
      if (keys.length === 0) {
        options.oauth.error.value = t(
          'admin.accounts.oauth.pleaseEnterSessionKey',
        )
        return
      }
      const defaults = options.buildCredentialDefaults()
      if (!defaults) return

      let successCount = 0
      let failedCount = 0
      const errors: string[] = []
      for (let index = 0; index < keys.length; index++) {
        try {
          const tokenInfo = await exchangeToken(
            endpointFor(true),
            keys[index],
            '',
          )
          const extra = options.buildExtra(
            options.oauth.buildExtraInfo(tokenInfo),
          )
          const accountName =
            keys.length > 1
              ? `${options.accountName()} #${index + 1}`
              : options.accountName()
          await adminAPI.accounts.create(
            options.buildPayload(
              { ...tokenInfo, ...defaults },
              extra,
              { name: accountName },
            ),
          )
          successCount++
        } catch (error: any) {
          failedCount++
          errors.push(
            t('admin.accounts.oauth.keyAuthFailed', {
              index: index + 1,
              error:
                error.response?.data?.detail ||
                t('admin.accounts.oauth.authFailed'),
            }),
          )
        }
      }

      if (successCount > 0) {
        appStore.showSuccess(
          t('admin.accounts.oauth.successCreated', { count: successCount }),
        )
        options.notifyCreated()
        if (failedCount === 0) options.close()
      }
      if (failedCount > 0) options.oauth.error.value = errors.join('\n')
    } catch (error: any) {
      options.oauth.error.value =
        error.response?.data?.detail ||
        t('admin.accounts.oauth.cookieAuthFailed')
    } finally {
      options.oauth.loading.value = false
    }
  }

  return {
    handleExchangeAuthorizationCode,
    handleCookieAuth,
  }
}

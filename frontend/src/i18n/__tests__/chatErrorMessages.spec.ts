import { describe, expect, it } from 'vitest'
import en from '../locales/en/chat'
import zh from '../locales/zh/chat'

const errorKeys = [
  'requestFailed',
  'insufficientBalance',
  'modelUnavailable',
  'reasoningUnavailable',
  'serviceUnavailable',
  'network',
  'timeout',
  'rateLimited',
  'sessionExpired',
  'sessionChanged',
  'permissionDenied',
  'invalidRequest',
  'resourceUnavailable',
  'conflict',
  'attemptAlreadySubmitted',
] as const

describe('Chat error messages', () => {
  it('keeps the Chinese and English error contracts in sync', () => {
    expect(Object.keys(zh.chat.errors).sort()).toEqual(Object.keys(en.chat.errors).sort())
    for (const key of errorKeys) {
      expect(zh.chat.errors[key]).toBeTruthy()
      expect(en.chat.errors[key]).toBeTruthy()
    }
  })

  it('keeps retry action labels in both locales', () => {
    expect(zh.chat.actions.retryFailed).toBe('重试')
    expect(zh.chat.actions.retrying).toBe('正在重试')
    expect(en.chat.actions.retryFailed).toBe('Retry')
    expect(en.chat.actions.retrying).toBe('Retrying')
  })

  it('uses actionable Chinese copy instead of raw HTTP status text', () => {
    expect(zh.chat.errors.sessionExpired).toBe('登录状态已失效，请重新登录后继续。')
    expect(zh.chat.errors.permissionDenied).toBe('当前账户无权完成此请求。')
    expect(zh.chat.errors.serviceUnavailable).toBe('聊天服务暂不可用，请稍后重试。')
    expect(zh.chat.errors.network).toBe('网络连接不稳定，请检查网络后重试。')
    expect(zh.chat.errors.timeout).toBe('请求等待时间过长，请稍后重试。')
    expect(zh.chat.errors.rateLimited).toBe('请求过于频繁，请稍后再试。')

    const visibleCopy = errorKeys.map((key) => zh.chat.errors[key]).join(' ')
    expect(visibleCopy).not.toMatch(/Forbidden|Bad Gateway|Unauthorized/i)
  })

  it('keeps cloud history failure copy explicit and non-blocking in both locales', () => {
    expect(zh.chat.sync.errorDescription)
      .toBe('云端历史记录同步失败，当前聊天仍可正常使用')
    expect(zh.chat.sync.retry).toBe('重新同步')
    expect(zh.chat.sync.syncing).toBe('正在同步…')
    expect(zh.chat.sync.success).toBe('云端历史记录已同步')

    expect(en.chat.sync.errorDescription)
      .toBe('Cloud history failed to sync. You can keep chatting normally.')
    expect(en.chat.sync.retry).toBe('Sync again')
    expect(en.chat.sync.syncing).toBe('Syncing…')
    expect(en.chat.sync.success).toBe('Cloud history is synced')

    const visibleCopy = [
      zh.chat.sync.errorDescription,
      en.chat.sync.errorDescription,
    ].join(' ')
    expect(visibleCopy).not.toMatch(/Forbidden|Bad Gateway|Unauthorized/i)
  })
})

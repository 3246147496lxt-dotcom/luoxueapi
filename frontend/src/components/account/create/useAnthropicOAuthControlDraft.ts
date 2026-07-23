import { ref } from 'vue'
import { buildAnthropicOAuthExtra } from './credentialDraftBuilders'

export function useAnthropicOAuthControlDraft() {
  const windowCostEnabled = ref(false)
  const windowCostLimit = ref<number | null>(null)
  const windowCostStickyReserve = ref<number | null>(null)
  const sessionLimitEnabled = ref(false)
  const maxSessions = ref<number | null>(null)
  const sessionIdleTimeout = ref<number | null>(null)
  const rpmLimitEnabled = ref(false)
  const baseRpm = ref<number | null>(null)
  const rpmStrategy = ref<'tiered' | 'sticky_exempt'>('tiered')
  const rpmStickyBuffer = ref<number | null>(null)
  const userMsgQueueMode = ref('')
  const tlsFingerprintEnabled = ref(false)
  const tlsFingerprintProfileId = ref<number | null>(null)
  const sessionIdMaskingEnabled = ref(false)
  const cacheTTLOverrideEnabled = ref(false)
  const cacheTTLOverrideTarget = ref('5m')
  const customBaseUrlEnabled = ref(false)
  const customBaseUrl = ref('')

  const reset = () => {
    windowCostEnabled.value = false
    windowCostLimit.value = null
    windowCostStickyReserve.value = null
    sessionLimitEnabled.value = false
    maxSessions.value = null
    sessionIdleTimeout.value = null
    rpmLimitEnabled.value = false
    baseRpm.value = null
    rpmStrategy.value = 'tiered'
    rpmStickyBuffer.value = null
    userMsgQueueMode.value = ''
    tlsFingerprintEnabled.value = false
    tlsFingerprintProfileId.value = null
    sessionIdMaskingEnabled.value = false
    cacheTTLOverrideEnabled.value = false
    cacheTTLOverrideTarget.value = '5m'
    customBaseUrlEnabled.value = false
    customBaseUrl.value = ''
  }

  const buildExtra = (base?: Record<string, unknown>) =>
    buildAnthropicOAuthExtra(base, {
      windowCostEnabled: windowCostEnabled.value,
      windowCostLimit: windowCostLimit.value,
      windowCostStickyReserve: windowCostStickyReserve.value,
      sessionLimitEnabled: sessionLimitEnabled.value,
      maxSessions: maxSessions.value,
      sessionIdleTimeout: sessionIdleTimeout.value,
      rpmLimitEnabled: rpmLimitEnabled.value,
      baseRpm: baseRpm.value,
      rpmStrategy: rpmStrategy.value,
      rpmStickyBuffer: rpmStickyBuffer.value,
      userMsgQueueMode: userMsgQueueMode.value,
      tlsFingerprintEnabled: tlsFingerprintEnabled.value,
      tlsFingerprintProfileId: tlsFingerprintProfileId.value,
      sessionIdMaskingEnabled: sessionIdMaskingEnabled.value,
      cacheTTLOverrideEnabled: cacheTTLOverrideEnabled.value,
      cacheTTLOverrideTarget: cacheTTLOverrideTarget.value,
      customBaseUrlEnabled: customBaseUrlEnabled.value,
      customBaseUrl: customBaseUrl.value,
    })

  return {
    windowCostEnabled,
    windowCostLimit,
    windowCostStickyReserve,
    sessionLimitEnabled,
    maxSessions,
    sessionIdleTimeout,
    rpmLimitEnabled,
    baseRpm,
    rpmStrategy,
    rpmStickyBuffer,
    userMsgQueueMode,
    tlsFingerprintEnabled,
    tlsFingerprintProfileId,
    sessionIdMaskingEnabled,
    cacheTTLOverrideEnabled,
    cacheTTLOverrideTarget,
    customBaseUrlEnabled,
    customBaseUrl,
    reset,
    buildExtra,
  }
}

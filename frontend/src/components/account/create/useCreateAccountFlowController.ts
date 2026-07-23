import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import type {
  AccountPlatform,
  CheckMixedChannelResponse,
  CreateAccountRequest,
} from '@/types'

export interface CreateAccountFlowControllerOptions {
  platform(): AccountPlatform
  groupIds(): number[]
  notifyCreated(): void
  close(): void
}

/**
 * Owns the modal-level admin effects: reference-data loading, mixed-channel
 * confirmation, and the final create request. The modal only orchestrates
 * drafts and wizard steps through this controller.
 */
export function useCreateAccountFlowController(
  options: CreateAccountFlowControllerOptions,
) {
  const appStore = useAppStore()
  const { t } = useI18n()

  const submitting = ref(false)
  const showMixedChannelWarning = ref(false)
  const mixedChannelWarningDetails = ref<{
    groupName: string
    currentPlatform: string
    otherPlatform: string
  } | null>(null)
  const mixedChannelWarningRawMessage = ref('')
  const mixedChannelWarningAction = ref<(() => Promise<void>) | null>(null)
  const mixedChannelConfirmed = ref(false)

  const mixedChannelWarningMessage = computed(() => {
    if (mixedChannelWarningDetails.value) {
      return t(
        'admin.accounts.mixedChannelWarning',
        mixedChannelWarningDetails.value,
      )
    }
    return mixedChannelWarningRawMessage.value
  })

  const needsMixedChannelCheck = (platform: AccountPlatform) =>
    platform === 'antigravity' || platform === 'anthropic'

  const clearMixedChannelDialog = () => {
    showMixedChannelWarning.value = false
    mixedChannelWarningDetails.value = null
    mixedChannelWarningRawMessage.value = ''
    mixedChannelWarningAction.value = null
  }

  const resetConfirmation = () => {
    mixedChannelConfirmed.value = false
    clearMixedChannelDialog()
  }

  const openMixedChannelDialog = (input: {
    response?: CheckMixedChannelResponse
    message?: string
    onConfirm: () => Promise<void>
  }) => {
    const details = input.response?.details
    mixedChannelWarningDetails.value = details
      ? {
          groupName: details.group_name || 'Unknown',
          currentPlatform: details.current_platform || 'Unknown',
          otherPlatform: details.other_platform || 'Unknown',
        }
      : null
    mixedChannelWarningRawMessage.value =
      input.message ||
      input.response?.message ||
      t('admin.accounts.failedToCreate')
    mixedChannelWarningAction.value = input.onConfirm
    showMixedChannelWarning.value = true
  }

  const withConfirmFlag = (
    payload: CreateAccountRequest,
  ): CreateAccountRequest => {
    if (
      needsMixedChannelCheck(payload.platform) &&
      mixedChannelConfirmed.value
    ) {
      return { ...payload, confirm_mixed_channel_risk: true }
    }
    const cloned = { ...payload }
    delete cloned.confirm_mixed_channel_risk
    return cloned
  }

  const ensureMixedChannelConfirmed = async (
    onConfirm: () => Promise<void>,
  ): Promise<boolean> => {
    if (
      !needsMixedChannelCheck(options.platform()) ||
      mixedChannelConfirmed.value
    ) {
      return true
    }

    try {
      const result = await adminAPI.accounts.checkMixedChannelRisk({
        platform: options.platform(),
        group_ids: options.groupIds(),
      })
      if (!result.has_risk) return true

      openMixedChannelDialog({
        response: result,
        onConfirm: async () => {
          mixedChannelConfirmed.value = true
          await onConfirm()
        },
      })
      return false
    } catch (error: any) {
      appStore.showError(
        error.response?.data?.message ||
          error.response?.data?.detail ||
          t('admin.accounts.failedToCreate'),
      )
      return false
    }
  }

  const submitCreateAccount = async (payload: CreateAccountRequest) => {
    submitting.value = true
    try {
      await adminAPI.accounts.create(withConfirmFlag(payload))
      appStore.showSuccess(t('admin.accounts.accountCreated'))
      options.notifyCreated()
      options.close()
    } catch (error: any) {
      if (
        error.response?.status === 409 &&
        error.response?.data?.error === 'mixed_channel_warning' &&
        needsMixedChannelCheck(options.platform())
      ) {
        openMixedChannelDialog({
          message: error.response?.data?.message,
          onConfirm: async () => {
            mixedChannelConfirmed.value = true
            await submitCreateAccount(payload)
          },
        })
        return
      }
      appStore.showError(
        error.response?.data?.message ||
          error.response?.data?.detail ||
          t('admin.accounts.failedToCreate'),
      )
    } finally {
      submitting.value = false
    }
  }

  const createAccount = async (payload: CreateAccountRequest) => {
    const canContinue = await ensureMixedChannelConfirmed(async () => {
      await submitCreateAccount(payload)
    })
    if (canContinue) await submitCreateAccount(payload)
  }

  const confirmMixedChannel = async () => {
    const action = mixedChannelWarningAction.value
    if (!action) {
      clearMixedChannelDialog()
      return
    }
    clearMixedChannelDialog()
    submitting.value = true
    try {
      await action()
    } finally {
      submitting.value = false
    }
  }

  const loadTLSFingerprintProfiles = async () => {
    try {
      const profiles = await adminAPI.tlsFingerprintProfiles.list()
      return profiles.map((profile) => ({
        id: profile.id,
        name: profile.name,
      }))
    } catch {
      return []
    }
  }

  const loadWebSearchEmulationAvailability = async () => {
    try {
      const config = await adminAPI.settings.getWebSearchEmulationConfig()
      return (
        config?.enabled === true && (config?.providers?.length ?? 0) > 0
      )
    } catch {
      return false
    }
  }

  return {
    submitting,
    showMixedChannelWarning,
    mixedChannelWarningMessage,
    ensureMixedChannelConfirmed,
    withConfirmFlag,
    createAccount,
    confirmMixedChannel,
    cancelMixedChannel: clearMixedChannelDialog,
    resetConfirmation,
    loadTLSFingerprintProfiles,
    loadWebSearchEmulationAvailability,
  }
}

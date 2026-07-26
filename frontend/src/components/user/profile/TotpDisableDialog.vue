<template>
  <Teleport to="body">
    <div class="totp-modal-layer fixed inset-0 overflow-y-auto">
      <div
        class="fixed inset-0 bg-black/50 transition-opacity"
        aria-hidden="true"
        @click="requestClose"
      ></div>
      <div class="flex min-h-full items-center justify-center p-4">

        <div
          ref="dialogRef"
          class="relative w-full max-w-md transform rounded-xl bg-white p-6 shadow-xl transition-all dark:bg-dark-800"
          role="alertdialog"
          aria-modal="true"
          :aria-labelledby="dialogTitleId"
          :aria-describedby="dialogDescriptionId"
          tabindex="-1"
        >
        <!-- Header -->
        <div class="mb-6">
          <div class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-red-100 dark:bg-red-900/30">
            <svg class="h-6 w-6 text-red-600 dark:text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
            </svg>
          </div>
          <h3
            :id="dialogTitleId"
            class="mt-4 text-center text-xl font-semibold text-gray-900 dark:text-white"
          >
            {{ t('profile.totp.disableTitle') }}
          </h3>
          <p
            :id="dialogDescriptionId"
            class="mt-2 text-center text-sm text-gray-500 dark:text-gray-400"
          >
            {{ t('profile.totp.disableWarning') }}
          </p>
        </div>

        <!-- Loading verification method -->
        <div v-if="methodLoading" class="flex items-center justify-center py-8">
          <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-500"></div>
        </div>

        <form v-else @submit.prevent="handleDisable" class="space-y-4">
          <!-- Email verification -->
          <div v-if="verificationMethod === 'email'">
            <label class="input-label">{{ t('profile.totp.emailCode') }}</label>
            <div class="flex gap-2">
              <input
                v-model="form.emailCode"
                type="text"
                maxlength="6"
                inputmode="numeric"
                class="input flex-1"
                :placeholder="t('profile.totp.enterEmailCode')"
              />
              <button
                type="button"
                class="btn btn-secondary whitespace-nowrap"
                :disabled="sendingCode || codeCooldown > 0"
                @click="handleSendCode"
              >
                {{ codeCooldown > 0 ? `${codeCooldown}s` : (sendingCode ? t('common.sending') : t('profile.totp.sendCode')) }}
              </button>
            </div>
          </div>

          <!-- Password verification -->
          <div v-else>
            <label for="password" class="input-label">
              {{ t('profile.currentPassword') }}
            </label>
            <input
              id="password"
              v-model="form.password"
              type="password"
              autocomplete="current-password"
              class="input"
              :placeholder="t('profile.totp.enterPassword')"
            />
          </div>

          <!-- Actions -->
          <div class="flex justify-end gap-3 pt-4">
            <button type="button" class="btn btn-secondary" @click="requestClose">
              {{ t('common.cancel') }}
            </button>
            <button
              type="submit"
              class="btn btn-danger"
              :disabled="loading || !canSubmit"
            >
              {{ loading ? t('common.processing') : t('profile.totp.confirmDisable') }}
            </button>
          </div>
        </form>
      </div>
    </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, getCurrentInstance, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { totpAPI } from '@/api'
import { acquireBodyScrollLock, releaseBodyScrollLock } from '@/utils/bodyScrollLock'
import {
  isTopModalLayer,
  registerModalLayer,
  unregisterModalLayer,
} from '@/utils/modalStack'

const emit = defineEmits<{
  close: []
  success: []
}>()

const { t } = useI18n()
const appStore = useAppStore()
const dialogRef = ref<HTMLElement | null>(null)
const instanceId = getCurrentInstance()?.uid ?? 'dialog'
const dialogTitleId = `totp-disable-title-${instanceId}`
const dialogDescriptionId = `totp-disable-description-${instanceId}`
const modalLayerToken = Symbol('totp-disable-dialog')
const scrollLockToken = Symbol('totp-disable-dialog-scroll-lock')
let previouslyFocusedElement: HTMLElement | null = null
let underlyingModalElement: HTMLElement | null = null
let underlyingModalWasInert = false
let underlyingModalAriaHidden: string | null = null

const methodLoading = ref(true)
const verificationMethod = ref<'email' | 'password'>('password')
const loading = ref(false)
const sendingCode = ref(false)
const codeCooldown = ref(0)
const cooldownTimer = ref<ReturnType<typeof setInterval> | null>(null)
const form = ref({
  emailCode: '',
  password: ''
})

const canSubmit = computed(() => {
  if (verificationMethod.value === 'email') {
    return form.value.emailCode.length === 6
  }
  return form.value.password.length > 0
})

const loadVerificationMethod = async () => {
  methodLoading.value = true
  try {
    const method = await totpAPI.getVerificationMethod()
    verificationMethod.value = method.method
  } catch (err: any) {
    appStore.showError(err.response?.data?.message || t('common.error'))
    emit('close')
  } finally {
    methodLoading.value = false
  }
}

const handleSendCode = async () => {
  sendingCode.value = true
  try {
    await totpAPI.sendVerifyCode()
    appStore.showSuccess(t('profile.totp.codeSent'))
    // Start cooldown
    codeCooldown.value = 60
    if (cooldownTimer.value) {
      clearInterval(cooldownTimer.value)
      cooldownTimer.value = null
    }
    cooldownTimer.value = setInterval(() => {
      codeCooldown.value--
      if (codeCooldown.value <= 0) {
        if (cooldownTimer.value) {
          clearInterval(cooldownTimer.value)
          cooldownTimer.value = null
        }
      }
    }, 1000)
  } catch (err: any) {
    appStore.showError(err.response?.data?.message || t('profile.totp.sendCodeFailed'))
  } finally {
    sendingCode.value = false
  }
}

const handleDisable = async () => {
  if (!canSubmit.value) return

  loading.value = true

  try {
    const request = verificationMethod.value === 'email'
      ? { email_code: form.value.emailCode }
      : { password: form.value.password }

    await totpAPI.disable(request)
    appStore.showSuccess(t('profile.totp.disableSuccess'))
    emit('success')
  } catch (err: any) {
    appStore.showError(err.response?.data?.message || t('profile.totp.disableFailed'))
  } finally {
    loading.value = false
  }
}

function getFocusableElements(): HTMLElement[] {
  if (!dialogRef.value) return []

  return Array.from(dialogRef.value.querySelectorAll<HTMLElement>(
    'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])',
  )).filter(element => (
    !element.hasAttribute('hidden')
    && element.getAttribute('aria-hidden') !== 'true'
    && !element.closest('[hidden]')
  ))
}

function requestClose() {
  if (!isTopModalLayer(modalLayerToken)) return
  emit('close')
}

function handleDialogKeydown(event: KeyboardEvent) {
  if (!isTopModalLayer(modalLayerToken)) return

  if (event.key === 'Escape') {
    event.preventDefault()
    event.stopPropagation()
    requestClose()
    return
  }

  if (event.key !== 'Tab') return

  const focusable = getFocusableElements()
  if (focusable.length === 0) {
    event.preventDefault()
    dialogRef.value?.focus()
    return
  }

  const first = focusable[0]
  const last = focusable[focusable.length - 1]
  const activeElement = document.activeElement

  if (
    event.shiftKey
    && (activeElement === first || activeElement === dialogRef.value || !dialogRef.value?.contains(activeElement))
  ) {
    event.preventDefault()
    last.focus()
  } else if (
    !event.shiftKey
    && (activeElement === last || activeElement === dialogRef.value || !dialogRef.value?.contains(activeElement))
  ) {
    event.preventDefault()
    first.focus()
  }
}

async function focusDialog() {
  await nextTick()
  dialogRef.value?.focus()
}

function hideUnderlyingModal() {
  underlyingModalElement = previouslyFocusedElement?.closest<HTMLElement>(
    '[role="dialog"][aria-modal="true"], [role="alertdialog"][aria-modal="true"]',
  ) ?? null
  if (!underlyingModalElement) return

  underlyingModalWasInert = underlyingModalElement.inert ?? false
  underlyingModalAriaHidden = underlyingModalElement.getAttribute('aria-hidden')
  underlyingModalElement.inert = true
  underlyingModalElement.setAttribute('aria-hidden', 'true')
}

function restoreUnderlyingModal() {
  if (!underlyingModalElement) return

  underlyingModalElement.inert = underlyingModalWasInert
  if (underlyingModalAriaHidden === null) {
    underlyingModalElement.removeAttribute('aria-hidden')
  } else {
    underlyingModalElement.setAttribute('aria-hidden', underlyingModalAriaHidden)
  }
  underlyingModalElement = null
}

onMounted(() => {
  previouslyFocusedElement = document.activeElement instanceof HTMLElement
    ? document.activeElement
    : null
  hideUnderlyingModal()
  registerModalLayer(modalLayerToken)
  acquireBodyScrollLock(scrollLockToken)
  document.addEventListener('keydown', handleDialogKeydown)
  void focusDialog()
  void loadVerificationMethod()
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleDialogKeydown)
  unregisterModalLayer(modalLayerToken)
  releaseBodyScrollLock(scrollLockToken)
  restoreUnderlyingModal()
  if (cooldownTimer.value) {
    clearInterval(cooldownTimer.value)
    cooldownTimer.value = null
  }
  if (previouslyFocusedElement?.isConnected) {
    previouslyFocusedElement.focus()
  }
})
</script>

<style scoped>
.totp-modal-layer {
  z-index: 100;
}

@media (prefers-reduced-motion: reduce) {
  .totp-modal-layer,
  .totp-modal-layer * {
    transition-duration: 1ms !important;
  }
}
</style>

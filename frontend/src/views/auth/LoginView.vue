<template>
  <AuthLayout variant="snow">
    <div class="login-shell w-full space-y-8">
      <div class="login-heading text-left">
        <h1 class="login-title text-3xl font-semibold tracking-[-0.035em] text-gray-950 dark:text-white sm:text-[2rem]">
          {{ t('auth.welcomeBack') }}
        </h1>
        <p class="login-description mt-2.5 text-sm leading-6 text-gray-500 dark:text-dark-400">
          {{ t('auth.signInToAccount') }}
        </p>
      </div>

      <form @submit.prevent="handleLogin" class="login-form space-y-6">
        <div class="space-y-2">
          <label for="email" class="mb-0 block text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('auth.emailLabel') }}
          </label>
          <div class="relative">
            <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4">
              <Icon name="mail" size="md" class="text-gray-500 dark:text-dark-400" />
            </div>
            <input
              id="email"
              v-model="formData.email"
              type="email"
              required
              autofocus
              autocomplete="email"
              :disabled="authActionDisabled"
              class="input login-input h-14 rounded-[20px] bg-[#f7f8f7] pl-12 pr-4 text-base shadow-none placeholder:text-gray-500 focus:bg-white dark:bg-dark-800 dark:placeholder:text-dark-400 dark:focus:bg-dark-800"
              :class="{ 'input-error': errors.email }"
              :placeholder="t('auth.emailPlaceholder')"
            />
          </div>
        </div>

        <div class="space-y-2">
          <div class="flex items-center justify-between gap-4">
            <label for="password" class="mb-0 block text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('auth.passwordLabel') }}
            </label>
            <router-link
              v-if="passwordResetEnabled && !backendModeEnabled"
              to="/forgot-password"
              class="login-link whitespace-nowrap text-sm font-medium text-primary-700 transition-colors hover:text-primary-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/50 dark:text-primary-300 dark:hover:text-primary-200"
            >
              {{ t('auth.forgotPassword') }}
            </router-link>
          </div>
          <div class="relative">
            <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4">
              <Icon name="lock" size="md" class="text-gray-500 dark:text-dark-400" />
            </div>
            <input
              id="password"
              v-model="formData.password"
              :type="showPassword ? 'text' : 'password'"
              required
              autocomplete="current-password"
              :disabled="authActionDisabled"
              class="input login-input h-14 rounded-[20px] bg-[#f7f8f7] pl-12 pr-12 text-base shadow-none placeholder:text-gray-500 focus:bg-white dark:bg-dark-800 dark:placeholder:text-dark-400 dark:focus:bg-dark-800"
              :class="{ 'input-error': errors.password }"
              :placeholder="t('auth.passwordPlaceholder')"
            />
            <button
              type="button"
              @click="showPassword = !showPassword"
              :disabled="authActionDisabled"
              :aria-label="t('auth.passwordLabel')"
              :aria-pressed="showPassword"
              class="login-reveal absolute inset-y-0 right-0 flex items-center px-4 text-gray-500 transition-colors hover:text-gray-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary-500/50 disabled:cursor-not-allowed disabled:opacity-50 dark:text-dark-400 dark:hover:text-dark-200"
            >
              <Icon v-if="showPassword" name="eyeOff" size="md" />
              <Icon v-else name="eye" size="md" />
            </button>
          </div>
        </div>

        <div v-if="turnstileEnabled && turnstileSiteKey" class="overflow-hidden rounded-2xl">
          <TurnstileWidget
            ref="turnstileRef"
            :site-key="turnstileSiteKey"
            @verify="onTurnstileVerify"
            @expire="onTurnstileExpire"
            @error="onTurnstileError"
          />
        </div>

        <button
          type="submit"
          :disabled="authActionDisabled || (turnstileEnabled && !turnstileToken)"
          class="btn btn-primary login-submit h-14 w-full rounded-[20px] px-5 text-base"
        >
          <Icon v-if="isLoading" name="refresh" size="md" class="motion-safe:animate-spin" />
          <Icon v-else name="login" size="md" class="mr-2" />
          {{ isLoading ? t('auth.signingIn') : t('auth.signIn') }}
        </button>

        <LoginAgreementPrompt
          v-if="loginAgreementEnabled"
          :accepted="agreementAccepted"
          :documents="loginAgreementDocuments"
          :mode="loginAgreementMode"
          :updated-at="loginAgreementUpdatedAt"
          :visible="showAgreementModal"
          @accept="acceptLoginAgreement"
          @reject="rejectLoginAgreement"
          @open="showAgreementModal = true"
        />

        <div v-if="showOAuthLogin" class="space-y-4 pt-1">
          <div class="flex items-center gap-3">
            <div class="login-divider h-px flex-1 bg-gray-100 dark:bg-dark-700"></div>
            <span class="login-oauth-label text-xs text-gray-500 dark:text-dark-400">
              {{ t('auth.oauthOrContinue') }}
            </span>
            <div class="login-divider h-px flex-1 bg-gray-100 dark:bg-dark-700"></div>
          </div>

          <EmailOAuthButtons
            :disabled="authActionDisabled"
            :github-enabled="githubOAuthEnabled"
            :google-enabled="googleOAuthEnabled"
            :show-divider="false"
          />

          <LinuxDoOAuthSection
            v-if="linuxdoOAuthEnabled"
            :disabled="authActionDisabled"
            :show-divider="false"
          />
          <DingTalkOAuthSection
            v-if="dingtalkOAuthEnabled"
            :disabled="authActionDisabled"
            :show-divider="false"
          />
          <WechatOAuthSection
            v-if="wechatOAuthEnabled"
            :disabled="authActionDisabled"
            :show-divider="false"
          />
          <OidcOAuthSection
            v-if="oidcOAuthEnabled"
            :disabled="authActionDisabled"
            :provider-name="oidcOAuthProviderName"
            :show-divider="false"
          />
        </div>
      </form>
    </div>

    <!-- Footer -->
    <template v-if="!backendModeEnabled" #footer>
      <p class="login-footer text-gray-500 dark:text-dark-400">
        {{ t('auth.dontHaveAccount') }}
        <router-link
          to="/register"
          class="login-link ml-1 font-semibold text-primary-700 transition-colors hover:text-primary-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/50 dark:text-primary-300 dark:hover:text-primary-200"
        >
          {{ t('auth.signUp') }}
        </router-link>
      </p>
    </template>
  </AuthLayout>

  <!-- 2FA Modal -->
  <TotpLoginModal
    v-if="show2FAModal"
    ref="totpModalRef"
    :temp-token="totpTempToken"
    :user-email-masked="totpUserEmailMasked"
    @verify="handle2FAVerify"
    @cancel="handle2FACancel"
  />
</template>

<script setup lang="ts">
import { computed, ref, reactive, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { AuthLayout } from '@/components/layout'
import LinuxDoOAuthSection from '@/components/auth/LinuxDoOAuthSection.vue'
import DingTalkOAuthSection from '@/components/auth/DingTalkOAuthSection.vue'
import OidcOAuthSection from '@/components/auth/OidcOAuthSection.vue'
import WechatOAuthSection from '@/components/auth/WechatOAuthSection.vue'
import EmailOAuthButtons from '@/components/auth/EmailOAuthButtons.vue'
import LoginAgreementPrompt from '@/components/auth/LoginAgreementPrompt.vue'
import TotpLoginModal from '@/components/auth/TotpLoginModal.vue'
import Icon from '@/components/icons/Icon.vue'
import TurnstileWidget from '@/components/TurnstileWidget.vue'
import { useAuthStore, useAppStore } from '@/stores'
import { getPublicSettings, isTotp2FARequired, isWeChatWebOAuthEnabled } from '@/api/auth'
import type { LoginAgreementDocument, TotpLoginResponse } from '@/types'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { clearAllAffiliateReferralCodes } from '@/utils/oauthAffiliate'

const { t } = useI18n()
const LOGIN_AGREEMENT_STORAGE_KEY = 'sub2api_login_agreement_consent'

// ==================== Router & Stores ====================

const router = useRouter()
const authStore = useAuthStore()
const appStore = useAppStore()

// ==================== State ====================

const isLoading = ref<boolean>(false)
const errorMessage = ref<string>('')
const showPassword = ref<boolean>(false)
const publicSettingsLoaded = ref<boolean>(false)

// Public settings
const turnstileEnabled = ref<boolean>(false)
const turnstileSiteKey = ref<string>('')
const linuxdoOAuthEnabled = ref<boolean>(false)
const dingtalkOAuthEnabled = ref<boolean>(false)
const wechatOAuthEnabled = ref<boolean>(false)
const backendModeEnabled = ref<boolean>(false)
const oidcOAuthEnabled = ref<boolean>(false)
const oidcOAuthProviderName = ref<string>('OIDC')
const githubOAuthEnabled = ref<boolean>(false)
const googleOAuthEnabled = ref<boolean>(false)
const passwordResetEnabled = ref<boolean>(false)
const loginAgreementEnabled = ref<boolean>(false)
const loginAgreementMode = ref<'modal' | 'checkbox' | string>('modal')
const loginAgreementUpdatedAt = ref<string>('')
const loginAgreementRevision = ref<string>('')
const loginAgreementDocuments = ref<LoginAgreementDocument[]>([])
const agreementAccepted = ref<boolean>(false)
const showAgreementModal = ref<boolean>(false)

// Turnstile
const turnstileRef = ref<InstanceType<typeof TurnstileWidget> | null>(null)
const turnstileToken = ref<string>('')

// 2FA state
const show2FAModal = ref<boolean>(false)
const totpTempToken = ref<string>('')
const totpUserEmailMasked = ref<string>('')
const totpModalRef = ref<InstanceType<typeof TotpLoginModal> | null>(null)

const formData = reactive({
  email: '',
  password: ''
})

const errors = reactive({
  email: '',
  password: '',
  turnstile: ''
})

const validationToastMessage = computed(
  () => errors.email || errors.password || errors.turnstile || ''
)

const agreementGateActive = computed(
  () => loginAgreementEnabled.value && !agreementAccepted.value
)

const authActionDisabled = computed(
  () => isLoading.value || !publicSettingsLoaded.value || agreementGateActive.value
)

const showOAuthLogin = computed(
  () =>
    !backendModeEnabled.value &&
    (linuxdoOAuthEnabled.value ||
      dingtalkOAuthEnabled.value ||
      wechatOAuthEnabled.value ||
      oidcOAuthEnabled.value ||
      githubOAuthEnabled.value ||
      googleOAuthEnabled.value)
)

watch(validationToastMessage, (value, previousValue) => {
  if (value && value !== previousValue) {
    appStore.showError(value)
  }
})

// ==================== Lifecycle ====================

onMounted(async () => {
  const expiredFlag = sessionStorage.getItem('auth_expired')
  if (expiredFlag) {
    sessionStorage.removeItem('auth_expired')
    const message = t('auth.reloginRequired')
    errorMessage.value = message
    appStore.showWarning(message)
  }

  try {
    const settings = await getPublicSettings()
    turnstileEnabled.value = settings.turnstile_enabled
    turnstileSiteKey.value = settings.turnstile_site_key || ''
    linuxdoOAuthEnabled.value = settings.linuxdo_oauth_enabled
    dingtalkOAuthEnabled.value = settings.dingtalk_oauth_enabled ?? false
    wechatOAuthEnabled.value = isWeChatWebOAuthEnabled(settings)
    backendModeEnabled.value = settings.backend_mode_enabled
    oidcOAuthEnabled.value = settings.oidc_oauth_enabled
    oidcOAuthProviderName.value = settings.oidc_oauth_provider_name || 'OIDC'
    githubOAuthEnabled.value = settings.github_oauth_enabled
    googleOAuthEnabled.value = settings.google_oauth_enabled
    backendModeEnabled.value = settings.backend_mode_enabled
    passwordResetEnabled.value = settings.password_reset_enabled
    applyLoginAgreementSettings(settings)
  } catch (error) {
    console.error('Failed to load public settings:', error)
    loginAgreementEnabled.value = false
    agreementAccepted.value = true
  } finally {
    publicSettingsLoaded.value = true
  }
})

// ==================== Login Agreement ====================

function applyLoginAgreementSettings(settings: {
  login_agreement_enabled?: boolean
  login_agreement_mode?: string
  login_agreement_updated_at?: string
  login_agreement_revision?: string
  login_agreement_documents?: LoginAgreementDocument[]
}): void {
  const documents = Array.isArray(settings.login_agreement_documents)
    ? settings.login_agreement_documents.filter((doc) => doc.title?.trim())
    : []
  loginAgreementDocuments.value = documents
  loginAgreementEnabled.value = settings.login_agreement_enabled === true && documents.length > 0
  loginAgreementMode.value = settings.login_agreement_mode === 'checkbox' ? 'checkbox' : 'modal'
  loginAgreementUpdatedAt.value = settings.login_agreement_updated_at || ''
  loginAgreementRevision.value =
    settings.login_agreement_revision ||
    `${loginAgreementUpdatedAt.value}:${documents.map((doc) => `${doc.id}:${doc.title}`).join('|')}`

  agreementAccepted.value = !loginAgreementEnabled.value || hasAcceptedLoginAgreement(loginAgreementRevision.value)
  showAgreementModal.value =
    loginAgreementEnabled.value && !agreementAccepted.value && loginAgreementMode.value !== 'checkbox'
}

function hasAcceptedLoginAgreement(revision: string): boolean {
  if (!revision) {
    return false
  }
  try {
    const raw = localStorage.getItem(LOGIN_AGREEMENT_STORAGE_KEY)
    if (!raw) {
      return false
    }
    const parsed = JSON.parse(raw) as { revision?: string }
    return parsed.revision === revision
  } catch {
    return false
  }
}

function acceptLoginAgreement(): void {
  if (loginAgreementRevision.value) {
    localStorage.setItem(
      LOGIN_AGREEMENT_STORAGE_KEY,
      JSON.stringify({
        revision: loginAgreementRevision.value,
        accepted_at: new Date().toISOString()
      })
    )
  }
  agreementAccepted.value = true
  showAgreementModal.value = false
}

function rejectLoginAgreement(): void {
  localStorage.removeItem(LOGIN_AGREEMENT_STORAGE_KEY)
  agreementAccepted.value = false
  showAgreementModal.value = false
  appStore.showWarning(t('legal.loginAgreementPrompt.loginRejectedWarning'))
}

// ==================== Turnstile Handlers ====================

function onTurnstileVerify(token: string): void {
  turnstileToken.value = token
  errors.turnstile = ''
}

function onTurnstileExpire(): void {
  turnstileToken.value = ''
  errors.turnstile = t('auth.turnstileExpired')
}

function onTurnstileError(): void {
  turnstileToken.value = ''
  errors.turnstile = t('auth.turnstileFailed')
}

// ==================== Validation ====================

function validateForm(): boolean {
  // Reset errors
  errors.email = ''
  errors.password = ''
  errors.turnstile = ''

  let isValid = true

  if (agreementGateActive.value) {
    appStore.showWarning(t('legal.loginAgreementPrompt.loginRequiredWarning'))
    if (loginAgreementMode.value !== 'checkbox') {
      showAgreementModal.value = true
    }
    return false
  }

  // Email validation
  if (!formData.email.trim()) {
    errors.email = t('auth.emailRequired')
    isValid = false
  } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
    errors.email = t('auth.invalidEmail')
    isValid = false
  }

  // Password validation
  if (!formData.password) {
    errors.password = t('auth.passwordRequired')
    isValid = false
  } else if (formData.password.length < 6) {
    errors.password = t('auth.passwordMinLength')
    isValid = false
  }

  // Turnstile validation
  if (turnstileEnabled.value && !turnstileToken.value) {
    errors.turnstile = t('auth.completeVerification')
    isValid = false
  }

  return isValid
}

// ==================== Form Handlers ====================

async function handleLogin(): Promise<void> {
  // Clear previous error
  errorMessage.value = ''

  // Validate form
  if (!validateForm()) {
    return
  }

  isLoading.value = true

  try {
    // Call auth store login
    const response = await authStore.login({
      email: formData.email,
      password: formData.password,
      turnstile_token: turnstileEnabled.value ? turnstileToken.value : undefined
    })

    // Check if 2FA is required
    if (isTotp2FARequired(response)) {
      const totpResponse = response as TotpLoginResponse
      totpTempToken.value = totpResponse.temp_token || ''
      totpUserEmailMasked.value = totpResponse.user_email_masked || ''
      show2FAModal.value = true
      isLoading.value = false
      return
    }

    // Show success toast
    clearAllAffiliateReferralCodes()
    appStore.showSuccess(t('auth.loginSuccess'))

    // Redirect to dashboard or intended route
    const redirectTo = (router.currentRoute.value.query.redirect as string) || '/dashboard'
    await router.push(redirectTo)
  } catch (error: unknown) {
    // Reset Turnstile on error
    if (turnstileRef.value) {
      turnstileRef.value.reset()
      turnstileToken.value = ''
    }

    errorMessage.value = extractI18nErrorMessage(error, t, 'auth.errors', t('auth.loginFailed'))

    // Also show error toast
    appStore.showError(errorMessage.value)
  } finally {
    isLoading.value = false
  }
}

// ==================== 2FA Handlers ====================

async function handle2FAVerify(code: string): Promise<void> {
  if (totpModalRef.value) {
    totpModalRef.value.setVerifying(true)
  }

  try {
    await authStore.login2FA(totpTempToken.value, code)

    // Close modal and show success
    show2FAModal.value = false
    clearAllAffiliateReferralCodes()
    appStore.showSuccess(t('auth.loginSuccess'))

    // Redirect to dashboard or intended route
    const redirectTo = (router.currentRoute.value.query.redirect as string) || '/dashboard'
    await router.push(redirectTo)
  } catch (error: unknown) {
    const err = error as { message?: string; response?: { data?: { message?: string } } }
    const message = err.response?.data?.message || err.message || t('profile.totp.loginFailed')

    if (totpModalRef.value) {
      totpModalRef.value.setError(message)
      totpModalRef.value.setVerifying(false)
    }
  }
}

function handle2FACancel(): void {
  show2FAModal.value = false
  totpTempToken.value = ''
  totpUserEmailMasked.value = ''
}
</script>

<style scoped>
.login-shell {
  --login-text: #332f3a;
  --login-copy: #635f69;
  --login-field: #efebf5;
  --login-border: rgba(91, 80, 112, 0.16);
  --login-violet: #7c3aed;
  --login-action-start: #7c3aed;
  --login-action-end: #5b21b6;
  color: var(--login-text);
}

.login-title {
  color: var(--login-text);
  font-family: "Nunito", "DM Sans", sans-serif;
  font-size: clamp(2.25rem, 3.5vw, 2.75rem);
  font-weight: 900;
  line-height: 1.08;
  letter-spacing: -0.025em;
}

.login-description,
.login-footer,
.login-oauth-label {
  color: var(--login-copy);
}

.login-form label {
  color: var(--login-text);
  font-weight: 700;
}

.login-input {
  border: 1px solid var(--login-border);
  color: var(--login-text);
  background: var(--login-field);
  box-shadow:
    inset 8px 8px 16px rgba(91, 80, 112, 0.09),
    inset -8px -8px 16px rgba(255, 255, 255, 0.8);
  transition:
    border-color 180ms ease,
    box-shadow 180ms ease,
    background-color 180ms ease;
}

.login-input::placeholder {
  color: #6f6977;
}

.login-input:-webkit-autofill,
.login-input:-webkit-autofill:hover,
.login-input:-webkit-autofill:focus {
  -webkit-text-fill-color: var(--login-text);
  caret-color: var(--login-text);
  box-shadow: inset 0 0 0 1000px var(--login-field);
}

.login-input:focus {
  border-color: color-mix(in srgb, var(--login-violet) 55%, transparent);
  outline: none;
  background: #ffffff;
  box-shadow:
    0 0 0 4px color-mix(in srgb, var(--login-violet) 16%, transparent),
    inset 4px 4px 10px rgba(91, 80, 112, 0.05);
}

.login-input.input-error {
  border-color: #dc2626;
}

.login-link {
  color: var(--login-violet);
  text-underline-offset: 3px;
}

.login-link:hover {
  color: #5b21b6;
  text-decoration: underline;
}

.login-reveal {
  color: var(--login-copy);
}

.login-divider {
  background: var(--login-border);
}

.login-submit {
  border: 0;
  color: #ffffff;
  background: linear-gradient(135deg, var(--login-action-start), var(--login-action-end));
  box-shadow:
    12px 12px 24px rgba(139, 92, 246, 0.28),
    -8px -8px 16px rgba(255, 255, 255, 0.34),
    inset 4px 4px 8px rgba(255, 255, 255, 0.35),
    inset -4px -4px 8px rgba(0, 0, 0, 0.1);
  transition:
    transform 180ms cubic-bezier(0.22, 1, 0.36, 1),
    box-shadow 180ms cubic-bezier(0.22, 1, 0.36, 1),
    opacity 180ms ease;
}

.login-submit:hover:not(:disabled) {
  transform: translateY(-3px);
  box-shadow:
    15px 15px 28px rgba(139, 92, 246, 0.36),
    -8px -8px 16px rgba(255, 255, 255, 0.32),
    inset 4px 4px 8px rgba(255, 255, 255, 0.38);
}

.login-submit:active:not(:disabled) {
  transform: translateY(0) scale(0.98);
  box-shadow:
    inset 7px 7px 14px rgba(61, 27, 114, 0.2),
    inset -5px -5px 12px rgba(255, 255, 255, 0.2);
}

.login-submit:disabled {
  cursor: not-allowed;
  opacity: 0.58;
  box-shadow: none;
}

.login-shell :deep(.btn-secondary) {
  min-height: 52px;
  border-color: var(--login-border);
  border-radius: 18px;
  color: var(--login-text);
  background: rgba(255, 255, 255, 0.64);
  box-shadow:
    6px 6px 14px rgba(91, 80, 112, 0.08),
    -6px -6px 14px rgba(255, 255, 255, 0.7);
}

.login-shell :deep(.btn-secondary:hover:not(:disabled)) {
  color: var(--login-violet);
  border-color: color-mix(in srgb, var(--login-violet) 28%, var(--login-border));
  transform: translateY(-2px);
}

.login-shell :deep(.btn-secondary:active:not(:disabled)) {
  transform: translateY(0) scale(0.98);
}

:global(html.dark .login-shell) {
  --login-text: #f8f5fc;
  --login-copy: #c5bccf;
  --login-field: #120f18;
  --login-border: rgba(255, 255, 255, 0.12);
  --login-violet: #a78bfa;
  --login-action-start: #7c3aed;
  --login-action-end: #5b21b6;
}

:global(html.dark .login-input),
:global(html.dark .login-input:focus) {
  color: var(--login-text);
  background: var(--login-field);
  box-shadow:
    inset 8px 8px 16px rgba(0, 0, 0, 0.32),
    inset -8px -8px 16px rgba(255, 255, 255, 0.025);
}

:global(html.dark .login-input::placeholder) {
  color: #aaa0b5;
}

:global(html.dark .login-link:hover) {
  color: #ddd6fe;
}

@media (max-width: 639px) {
  .login-shell {
    padding-block: 4px;
  }

  .login-title {
    font-size: 2.1rem;
  }
}

@media (prefers-reduced-motion: reduce) {
  .login-submit,
  .login-input {
    transition-duration: 0.01ms;
  }
}

.fade-enter-active,
.fade-leave-active {
  transition: all 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>

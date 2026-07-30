<template>
  <AuthLayout variant="snow">
    <div class="quota-viewer-auth-flow w-full">
      <div class="mb-8 flex items-start gap-4">
        <span class="quota-viewer-auth-icon" aria-hidden="true">
          <Icon :name="approved ? 'checkCircle' : 'shield'" size="lg" :stroke-width="1.8" />
        </span>
        <div class="min-w-0">
          <p class="text-sm font-semibold text-[var(--lx-clay-accent)]">
            {{ t('quotaViewerAuthorization.eyebrow') }}
          </p>
          <h1 class="mt-1 text-3xl font-bold text-[var(--lx-clay-text)]">
            {{ approved ? t('quotaViewerAuthorization.successTitle') : t('quotaViewerAuthorization.title') }}
          </h1>
          <p class="mt-2 text-sm leading-6 text-[var(--lx-clay-text-secondary)]">
            {{ approved ? t('quotaViewerAuthorization.successDescription') : t('quotaViewerAuthorization.description') }}
          </p>
        </div>
      </div>

      <div
        v-if="errorMessage"
        class="mb-5 flex items-start gap-3 rounded-2xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300"
        role="alert"
      >
        <Icon name="exclamationTriangle" size="sm" class="mt-0.5 shrink-0" />
        <span class="min-w-0 flex-1">
          {{ errorMessage }}
          <RouterLink
            v-if="errorCode === 'QUOTA_AUTH_DEVICE_LIMIT'"
            to="/quota-viewer/devices"
            class="quota-viewer-inline-link"
          >
            {{ t('quotaViewerAuthorization.manageDevicesAction') }}
          </RouterLink>
        </span>
      </div>

      <div
        v-if="loading"
        class="flex min-h-48 items-center justify-center gap-3 text-sm text-[var(--lx-clay-text-secondary)]"
        role="status"
      >
        <Icon name="refresh" size="md" class="motion-safe:animate-spin" />
        <span>{{ t('quotaViewerAuthorization.loading') }}</span>
      </div>

      <div v-else-if="approved" class="space-y-4">
        <div class="quota-viewer-readonly-note">
          <Icon name="checkCircle" size="sm" />
          <span>
            {{ t('quotaViewerAuthorization.successSafety') }}
            <RouterLink to="/quota-viewer/devices" class="quota-viewer-inline-link">
              {{ t('quotaViewerAuthorization.manageDevicesAction') }}
            </RouterLink>
          </span>
        </div>
        <button type="button" class="btn btn-primary h-12 w-full" @click="goToDashboard">
          {{ t('quotaViewerAuthorization.backToDashboard') }}
          <Icon name="arrowRight" size="sm" />
        </button>
      </div>

      <form v-else-if="!preview" class="space-y-5" @submit.prevent="loadPreview">
        <label class="block">
          <span class="mb-2 block text-sm font-semibold text-[var(--lx-clay-text)]">
            {{ t('quotaViewerAuthorization.codeLabel') }}
          </span>
          <input
            v-model="userCode"
            type="text"
            required
            autocomplete="one-time-code"
            maxlength="9"
            class="input h-14 w-full font-mono text-lg uppercase"
            :placeholder="t('quotaViewerAuthorization.codePlaceholder')"
          />
        </label>
        <button type="submit" class="btn btn-primary h-14 w-full" :disabled="busy">
          <Icon v-if="busy" name="refresh" size="md" class="motion-safe:animate-spin" />
          {{ t('quotaViewerAuthorization.continue') }}
        </button>
      </form>

      <div v-else class="space-y-6">
        <dl class="quota-viewer-auth-details">
          <div>
            <dt>{{ t('quotaViewerAuthorization.device') }}</dt>
            <dd>{{ preview.device_name }}</dd>
          </div>
          <div>
            <dt>{{ t('quotaViewerAuthorization.system') }}</dt>
            <dd>{{ systemLabel }}</dd>
          </div>
          <div v-if="preview.app_version">
            <dt>{{ t('quotaViewerAuthorization.appVersion') }}</dt>
            <dd>{{ preview.app_version }}</dd>
          </div>
        </dl>

        <section aria-labelledby="quota-viewer-permissions-title">
          <h2
            id="quota-viewer-permissions-title"
            class="text-sm font-semibold text-[var(--lx-clay-text)]"
          >
            {{ t('quotaViewerAuthorization.permissions') }}
          </h2>
          <ul class="mt-3 space-y-3">
            <li class="flex items-start gap-3 text-sm leading-6 text-[var(--lx-clay-text-secondary)]">
              <Icon name="checkCircle" size="sm" class="mt-1 shrink-0 text-emerald-600 dark:text-emerald-400" />
              <span>{{ t('quotaViewerAuthorization.scopeRead') }}</span>
            </li>
            <li class="flex items-start gap-3 text-sm leading-6 text-[var(--lx-clay-text-secondary)]">
              <Icon name="shield" size="sm" class="mt-1 shrink-0 text-[var(--lx-clay-accent)]" />
              <span>{{ t('quotaViewerAuthorization.scopeSafety') }}</span>
            </li>
          </ul>
        </section>

        <p class="text-xs text-[var(--lx-clay-text-secondary)]">
          {{ t('quotaViewerAuthorization.expiresAt', { time: expiresAtLabel }) }}
        </p>

        <div class="grid grid-cols-2 gap-3">
          <button type="button" class="btn btn-secondary h-12" :disabled="busy" @click="goToDashboard">
            {{ t('quotaViewerAuthorization.cancel') }}
          </button>
          <button type="button" class="btn btn-primary h-12" :disabled="busy" @click="approve">
            <Icon v-if="busy" name="refresh" size="sm" class="motion-safe:animate-spin" />
            {{ busy ? t('quotaViewerAuthorization.approving') : t('quotaViewerAuthorization.approve') }}
          </button>
        </div>
      </div>
    </div>
  </AuthLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import {
  quotaViewerAPI,
  type QuotaViewerPairingPreview,
} from '@/api/quotaViewer'
import { AuthLayout } from '@/components/layout'
import Icon from '@/components/icons/Icon.vue'
import { useAuthStore } from '@/stores/auth'
import {
  extractApiErrorCode,
  extractI18nErrorMessage,
} from '@/utils/apiError'

const QUOTA_VIEWER_CLIENT_ID = 'luoxue-quota-viewer'
const QUOTA_VIEWER_SCOPE = 'quota:read'
const QUOTA_VIEWER_APPROVED_STATUS = 'pending'
const QUOTA_VIEWER_RESPONSE_INVALID = 'QUOTA_AUTH_RESPONSE_INVALID'
const QUOTA_VIEWER_CODE_PATTERN = /^[23456789ABCDEFGHJKLMNPQRSTUVWXYZ]{8}$/

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const userCode = ref('')
const preview = ref<QuotaViewerPairingPreview | null>(null)
const loading = ref(false)
const busy = ref(false)
const approved = ref(false)
const errorMessage = ref('')
const errorCode = ref('')

const systemLabel = computed(() => {
  if (!preview.value) return ''
  return [preview.value.platform, preview.value.os_version, preview.value.architecture]
    .filter(Boolean)
    .join(' · ')
})

const expiresAtLabel = computed(() => {
  if (!preview.value) return ''
  return new Intl.DateTimeFormat(locale.value, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(preview.value.expires_at))
})

function normalizeUserCode(value: string): string {
  const compact = value.trim().toUpperCase().replace(/[\s-]+/g, '')
  return QUOTA_VIEWER_CODE_PATTERN.test(compact)
    ? `${compact.slice(0, 4)}-${compact.slice(4)}`
    : ''
}

function invalidAuthorizationResponse() {
  return { code: QUOTA_VIEWER_RESPONSE_INVALID }
}

function setApiError(error: unknown) {
  errorCode.value = extractApiErrorCode(error) ?? ''
  errorMessage.value = extractI18nErrorMessage(
    error,
    t,
    'quotaViewerAuthorization.errors',
    t('quotaViewerAuthorization.errors.fallback'),
  )
}

async function loadPreview() {
  if (busy.value) return
  const normalized = normalizeUserCode(userCode.value)
  if (!normalized) {
    errorCode.value = ''
    errorMessage.value = t('quotaViewerAuthorization.invalidCode')
    return
  }
  userCode.value = normalized
  errorCode.value = ''
  errorMessage.value = ''
  busy.value = true
  try {
    const result = await quotaViewerAPI.getPairingPreview(normalized)
    if (
      result.client_id !== QUOTA_VIEWER_CLIENT_ID
      || result.scope !== QUOTA_VIEWER_SCOPE
      || result.read_only !== true
    ) {
      throw invalidAuthorizationResponse()
    }
    preview.value = result
  } catch (error) {
    preview.value = null
    setApiError(error)
  } finally {
    busy.value = false
  }
}

async function approve() {
  if (!preview.value || busy.value) return
  errorCode.value = ''
  errorMessage.value = ''
  busy.value = true
  try {
    const device = await quotaViewerAPI.approvePairing(preview.value.user_code)
    if (
      device.client_id !== QUOTA_VIEWER_CLIENT_ID
      || device.scope !== QUOTA_VIEWER_SCOPE
      || device.status !== QUOTA_VIEWER_APPROVED_STATUS
    ) {
      throw invalidAuthorizationResponse()
    }
    approved.value = true
    preview.value = null
  } catch (error) {
    setApiError(error)
  } finally {
    busy.value = false
  }
}

function goToDashboard() {
  router.push(authStore.isAdmin ? '/admin/dashboard' : '/dashboard')
}

onMounted(async () => {
  const queryCode = Array.isArray(route.query.user_code)
    ? route.query.user_code[0]
    : route.query.user_code
  if (!queryCode) return
  userCode.value = queryCode
  loading.value = true
  await loadPreview()
  loading.value = false
})
</script>

<style scoped>
.quota-viewer-auth-icon {
  display: inline-flex;
  width: 52px;
  height: 52px;
  flex: 0 0 52px;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--lx-clay-border);
  border-radius: 18px;
  color: var(--lx-clay-accent);
  background: var(--lx-clay-recessed);
}

.quota-viewer-auth-details {
  border-block: 1px solid var(--lx-clay-border);
}

.quota-viewer-auth-details > div {
  display: grid;
  grid-template-columns: minmax(7rem, 0.8fr) minmax(0, 1.7fr);
  gap: 16px;
  padding: 14px 0;
  border-bottom: 1px solid var(--lx-clay-border);
}

.quota-viewer-auth-details > div:last-child {
  border-bottom: 0;
}

.quota-viewer-auth-details dt {
  color: var(--lx-clay-text-secondary);
  font-size: 13px;
  font-weight: 600;
}

.quota-viewer-auth-details dd {
  min-width: 0;
  overflow-wrap: anywhere;
  color: var(--lx-clay-text);
  font-size: 14px;
  font-weight: 700;
  text-align: right;
}

.quota-viewer-readonly-note {
  display: flex;
  padding: 14px 16px;
  align-items: flex-start;
  gap: 10px;
  border: 1px solid rgb(167 243 208);
  border-radius: 16px;
  background: rgb(236 253 245);
  color: rgb(4 120 87);
  font-size: 13px;
  line-height: 1.6;
}

.quota-viewer-inline-link {
  margin-left: 4px;
  color: currentColor;
  font-weight: 800;
  text-decoration: underline;
  text-underline-offset: 3px;
}

.quota-viewer-inline-link:focus-visible {
  border-radius: 4px;
  outline: 2px solid currentColor;
  outline-offset: 2px;
}

@media (max-width: 420px) {
  .quota-viewer-auth-details > div {
    grid-template-columns: 1fr;
    gap: 4px;
  }

  .quota-viewer-auth-details dd {
    text-align: left;
  }
}
</style>

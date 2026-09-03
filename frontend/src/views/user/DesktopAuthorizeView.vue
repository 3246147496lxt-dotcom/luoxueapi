<template>
  <AuthLayout variant="snow">
    <div class="desktop-auth-flow w-full">
      <div class="mb-8 flex items-start gap-4">
        <span class="desktop-auth-icon" aria-hidden="true">
          <Icon :name="approved ? 'checkCircle' : 'shield'" size="lg" :stroke-width="1.8" />
        </span>
        <div class="min-w-0">
          <p class="text-sm font-semibold text-[var(--lx-clay-accent)]">
            {{ t('desktopAuthorization.eyebrow') }}
          </p>
          <h1 class="mt-1 text-3xl font-bold text-[var(--lx-clay-text)]">
            {{ approved ? t('desktopAuthorization.successTitle') : t('desktopAuthorization.title') }}
          </h1>
          <p class="mt-2 text-sm leading-6 text-[var(--lx-clay-text-secondary)]">
            {{ approved ? t('desktopAuthorization.successDescription') : t('desktopAuthorization.description') }}
          </p>
        </div>
      </div>

      <div v-if="errorMessage" class="mb-5 flex items-start gap-3 rounded-2xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300" role="alert">
        <Icon name="exclamationTriangle" size="sm" class="mt-0.5 shrink-0" />
        <span>{{ errorMessage }}</span>
      </div>

      <div v-if="loading" class="flex min-h-48 items-center justify-center gap-3 text-sm text-[var(--lx-clay-text-secondary)]" role="status">
        <Icon name="refresh" size="md" class="motion-safe:animate-spin" />
        <span>{{ t('desktopAuthorization.loading') }}</span>
      </div>

      <div v-else-if="approved" class="space-y-4">
        <button type="button" class="btn btn-primary h-12 w-full" @click="goToDashboard">
          {{ t('desktopAuthorization.backToDashboard') }}
          <Icon name="arrowRight" size="sm" />
        </button>
      </div>

      <form v-else-if="!preview" class="space-y-5" @submit.prevent="loadPreview">
        <label class="block">
          <span class="mb-2 block text-sm font-semibold text-[var(--lx-clay-text)]">
            {{ t('desktopAuthorization.codeLabel') }}
          </span>
          <input
            v-model="userCode"
            type="text"
            required
            autocomplete="one-time-code"
            maxlength="9"
            class="input h-14 w-full font-mono text-lg uppercase"
            :placeholder="t('desktopAuthorization.codePlaceholder')"
          />
        </label>
        <button type="submit" class="btn btn-primary h-14 w-full" :disabled="busy">
          <Icon v-if="busy" name="refresh" size="md" class="motion-safe:animate-spin" />
          {{ t('desktopAuthorization.continue') }}
        </button>
      </form>

      <div v-else class="space-y-6">
        <dl class="desktop-auth-details">
          <div>
            <dt>{{ t('desktopAuthorization.device') }}</dt>
            <dd>{{ preview.device_name }}</dd>
          </div>
          <div>
            <dt>{{ t('desktopAuthorization.system') }}</dt>
            <dd>{{ systemLabel }}</dd>
          </div>
          <div v-if="preview.app_version">
            <dt>{{ t('desktopAuthorization.appVersion') }}</dt>
            <dd>{{ preview.app_version }}</dd>
          </div>
        </dl>

        <section aria-labelledby="desktop-permissions-title">
          <h2 id="desktop-permissions-title" class="text-sm font-semibold text-[var(--lx-clay-text)]">
            {{ t('desktopAuthorization.permissions') }}
          </h2>
          <ul class="mt-3 space-y-3">
            <li v-for="scope in preview.requested_scopes" :key="scope" class="flex items-start gap-3 text-sm leading-6 text-[var(--lx-clay-text-secondary)]">
              <Icon name="checkCircle" size="sm" class="mt-1 shrink-0 text-emerald-600 dark:text-emerald-400" />
              <span>{{ scopeLabel(scope) }}</span>
            </li>
          </ul>
        </section>

        <p class="text-xs text-[var(--lx-clay-text-secondary)]">
          {{ t('desktopAuthorization.expiresAt', { time: expiresAtLabel }) }}
        </p>

        <div class="grid grid-cols-2 gap-3">
          <button type="button" class="btn btn-secondary h-12" :disabled="busy" @click="goToDashboard">
            {{ t('desktopAuthorization.cancel') }}
          </button>
          <button type="button" class="btn btn-primary h-12" :disabled="busy" @click="approve">
            <Icon v-if="busy" name="refresh" size="sm" class="motion-safe:animate-spin" />
            {{ busy ? t('desktopAuthorization.approving') : t('desktopAuthorization.approve') }}
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
import { desktopAPI, type DesktopPairingPreview } from '@/api/desktop'
import { AuthLayout } from '@/components/layout'
import Icon from '@/components/icons/Icon.vue'
import { useAuthStore } from '@/stores/auth'
import { extractI18nErrorMessage } from '@/utils/apiError'

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const userCode = ref('')
const preview = ref<DesktopPairingPreview | null>(null)
const loading = ref(false)
const busy = ref(false)
const approved = ref(false)
const errorMessage = ref('')

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
  return compact.length === 8 ? `${compact.slice(0, 4)}-${compact.slice(4)}` : ''
}

function setApiError(error: unknown) {
  errorMessage.value = extractI18nErrorMessage(
    error,
    t,
    'desktopAuthorization.errors',
    t('desktopAuthorization.errors.fallback'),
  )
}

async function loadPreview() {
  const normalized = normalizeUserCode(userCode.value)
  if (!normalized) {
    errorMessage.value = t('desktopAuthorization.invalidCode')
    return
  }
  userCode.value = normalized
  errorMessage.value = ''
  busy.value = true
  try {
    preview.value = await desktopAPI.getPairingPreview(normalized)
  } catch (error) {
    preview.value = null
    setApiError(error)
  } finally {
    busy.value = false
  }
}

async function approve() {
  if (!preview.value) return
  errorMessage.value = ''
  busy.value = true
  try {
    await desktopAPI.approvePairing(preview.value.user_code)
    approved.value = true
    preview.value = null
  } catch (error) {
    setApiError(error)
  } finally {
    busy.value = false
  }
}

function scopeLabel(scope: string): string {
  const key = `desktopAuthorization.scopes.${scope}`
  const translated = t(key)
  return translated === key ? scope : translated
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
.desktop-auth-icon {
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

.desktop-auth-details {
  border-block: 1px solid var(--lx-clay-border);
}

.desktop-auth-details > div {
  display: grid;
  grid-template-columns: minmax(7rem, 0.8fr) minmax(0, 1.7fr);
  gap: 16px;
  padding: 14px 0;
  border-bottom: 1px solid var(--lx-clay-border);
}

.desktop-auth-details > div:last-child {
  border-bottom: 0;
}

.desktop-auth-details dt {
  color: var(--lx-clay-text-secondary);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
}

.desktop-auth-details dd {
  min-width: 0;
  overflow-wrap: anywhere;
  color: var(--lx-clay-text);
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
  text-align: right;
}

@media (max-width: 420px) {
  .desktop-auth-details > div {
    grid-template-columns: 1fr;
    gap: 4px;
  }

  .desktop-auth-details dd {
    text-align: left;
  }
}
</style>

<template>
  <div
    class="personal-settings-panel"
    data-testid="personal-settings-security"
  >
    <template v-if="!detail">
      <div
        v-if="statusError"
        class="personal-settings-error"
        data-testid="personal-settings-security-error"
        role="alert"
      >
        <span>{{ t('personalSettings.security.loadError') }}</span>
        <button type="button" @click="loadStatus(true)">
          {{ t('personalSettings.common.retry') }}
        </button>
      </div>

      <div class="personal-settings-group">
        <button
          type="button"
          class="personal-settings-row"
          data-testid="personal-settings-open-password"
          @click="emit('open-detail', 'password')"
        >
          <Icon name="lock" size="sm" class="text-[var(--lx-clay-text-muted)]" aria-hidden="true" />
          <span class="personal-settings-row-copy">
            <span class="personal-settings-row-title">
              {{ t('personalSettings.security.password') }}
            </span>
            <span class="personal-settings-row-description">
              {{ t('personalSettings.security.passwordHint') }}
            </span>
          </span>
          <span class="personal-settings-row-value">
            <Icon name="chevronRight" size="xs" aria-hidden="true" />
          </span>
        </button>

        <button
          type="button"
          class="personal-settings-row"
          data-testid="personal-settings-open-totp"
          @click="emit('open-detail', 'totp')"
        >
          <Icon name="shield" size="sm" class="text-[var(--lx-clay-text-muted)]" aria-hidden="true" />
          <span class="personal-settings-row-copy">
            <span class="personal-settings-row-title">
              {{ t('personalSettings.security.totp') }}
            </span>
            <span class="personal-settings-row-description">
              {{ t('personalSettings.security.totpHint') }}
            </span>
          </span>
          <span class="personal-settings-row-value">
            <span v-if="statusLoading" class="personal-settings-skeleton w-16"></span>
            <span v-else>{{ statusLabel }}</span>
            <Icon name="chevronRight" size="xs" aria-hidden="true" />
          </span>
        </button>
      </div>
    </template>

    <ProfilePasswordForm
      v-else-if="detail === 'password'"
      embedded
      data-testid="personal-settings-password-detail"
    />

    <ProfileTotpCard
      v-else
      embedded
      data-testid="personal-settings-totp-detail"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { totpAPI } from '@/api'
import Icon from '@/components/icons/Icon.vue'
import ProfilePasswordForm from '@/components/user/profile/ProfilePasswordForm.vue'
import ProfileTotpCard from '@/components/user/profile/ProfileTotpCard.vue'
import type { PersonalSettingsDetail } from '@/navigation/personalSettingsRoute'
import type { TotpStatus } from '@/types'

const props = defineProps<{
  detail: PersonalSettingsDetail | null
}>()

const emit = defineEmits<{
  'open-detail': [detail: 'password' | 'totp']
}>()

const { t } = useI18n()
const status = ref<TotpStatus | null>(null)
const statusLoading = ref(false)
const statusError = ref(false)

const statusLabel = computed(() => {
  if (!status.value || !status.value.feature_enabled) {
    return t('personalSettings.security.totpUnavailable')
  }
  return status.value.enabled
    ? t('personalSettings.security.totpEnabled')
    : t('personalSettings.security.totpDisabled')
})

async function loadStatus(force = false) {
  if (!force && (status.value || statusLoading.value)) return
  statusLoading.value = true
  statusError.value = false
  try {
    status.value = await totpAPI.getStatus()
  } catch {
    statusError.value = true
  } finally {
    statusLoading.value = false
  }
}

watch(
  () => props.detail,
  (currentDetail) => {
    // Secondary editors own their own requests. Only fetch the summary status
    // when the first-level Security list is actually visible.
    if (currentDetail === null) {
      void loadStatus()
    }
  },
  { immediate: true },
)
</script>

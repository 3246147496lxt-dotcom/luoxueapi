<template>
  <div
    class="personal-settings-panel"
    data-testid="personal-settings-notifications"
  >
    <template v-if="!detail">
      <div
        v-if="error"
        class="personal-settings-error"
        data-testid="personal-settings-notifications-error"
        role="alert"
      >
        <span>{{ t('personalSettings.notifications.loadError') }}</span>
        <button type="button" @click="emit('retry')">
          {{ t('personalSettings.common.retry') }}
        </button>
      </div>

      <template v-if="!(error && !publicSettings)">
        <div
          v-if="loading && !publicSettings"
          class="space-y-5 py-3"
          aria-busy="true"
        >
          <div class="personal-settings-skeleton w-full"></div>
          <div class="personal-settings-skeleton w-5/6"></div>
        </div>

        <p
          v-else-if="!featureEnabled"
          class="personal-settings-callout"
          data-testid="personal-settings-notifications-disabled"
        >
          {{ t('personalSettings.notifications.systemDisabled') }}
        </p>

        <div v-else class="personal-settings-group">
          <label class="personal-settings-row">
            <Icon name="bell" size="sm" class="text-[var(--lx-clay-text-muted)]" aria-hidden="true" />
            <span class="personal-settings-row-copy">
              <span class="personal-settings-row-title">
                {{ t('personalSettings.notifications.balanceAlert') }}
              </span>
              <span class="personal-settings-row-description">
                {{ t('personalSettings.notifications.balanceAlertHint') }}
              </span>
            </span>
            <input
              class="sr-only"
              type="checkbox"
              data-testid="personal-settings-notification-toggle"
              :checked="notifyEnabled"
              :disabled="saving || !user"
              @change="handleToggle"
            >
            <span class="personal-settings-switch" aria-hidden="true"></span>
          </label>

          <button
            type="button"
            class="personal-settings-row"
            data-testid="personal-settings-open-notification-emails"
            @click="emit('open-detail', 'notification-emails')"
          >
            <Icon name="mail" size="sm" class="text-[var(--lx-clay-text-muted)]" aria-hidden="true" />
            <span class="personal-settings-row-copy">
              <span class="personal-settings-row-title">
                {{ t('personalSettings.notifications.recipients') }}
              </span>
              <span class="personal-settings-row-description">
                {{ t('personalSettings.notifications.recipientsHint') }}
              </span>
            </span>
            <span class="personal-settings-row-value">
              <span>{{ recipientsLabel }}</span>
              <Icon name="chevronRight" size="xs" aria-hidden="true" />
            </span>
          </button>

          <button
            type="button"
            class="personal-settings-row"
            @click="emit('open-detail', 'notification-emails')"
          >
            <Icon name="gauge" size="sm" class="text-[var(--lx-clay-text-muted)]" aria-hidden="true" />
            <span class="personal-settings-row-copy">
              <span class="personal-settings-row-title">
                {{ t('personalSettings.notifications.threshold') }}
              </span>
            </span>
            <span class="personal-settings-row-value">
              <CreditAmount :value="thresholdLabel" icon-size="xs" />
              <Icon name="chevronRight" size="xs" aria-hidden="true" />
            </span>
          </button>
        </div>
      </template>
    </template>

    <div
      v-else-if="error && !publicSettings"
      class="personal-settings-error"
      data-testid="personal-settings-notification-detail-error"
      role="alert"
    >
      <span>{{ t('personalSettings.notifications.loadError') }}</span>
      <button type="button" @click="emit('retry')">
        {{ t('personalSettings.common.retry') }}
      </button>
    </div>

    <div
      v-else-if="loading && !publicSettings"
      class="space-y-5 py-3"
      aria-busy="true"
    >
      <div class="personal-settings-skeleton w-full"></div>
      <div class="personal-settings-skeleton w-5/6"></div>
    </div>

    <ProfileBalanceNotifyCard
      v-else-if="featureEnabled && user"
      :enabled="user.balance_notify_enabled ?? true"
      :threshold="user.balance_notify_threshold"
      :extra-emails="user.balance_notify_extra_emails ?? []"
      :system-default-threshold="systemDefaultThreshold"
      :user-email="user.email"
      embedded
      data-testid="personal-settings-notification-emails-detail"
    />

    <p v-else class="personal-settings-callout">
      {{ t('personalSettings.notifications.systemDisabled') }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { userAPI } from '@/api'
import CreditAmount from '@/components/common/CreditAmount.vue'
import Icon from '@/components/icons/Icon.vue'
import ProfileBalanceNotifyCard from '@/components/user/profile/ProfileBalanceNotifyCard.vue'
import type { PersonalSettingsDetail } from '@/navigation/personalSettingsRoute'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import type { PublicSettings, User } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'

const props = defineProps<{
  user: User | null
  publicSettings: PublicSettings | null
  detail: PersonalSettingsDetail | null
  loading: boolean
  error: boolean
}>()

const emit = defineEmits<{
  retry: []
  'open-detail': [detail: 'notification-emails']
}>()

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const saving = ref(false)

const featureEnabled = computed(
  () => props.publicSettings?.balance_low_notify_enabled ?? false,
)
const notifyEnabled = computed(
  () => props.user?.balance_notify_enabled ?? true,
)
const systemDefaultThreshold = computed(
  () => Number(props.publicSettings?.balance_low_notify_threshold ?? 0),
)
const thresholdLabel = computed(() => {
  const custom = Number(props.user?.balance_notify_threshold ?? 0)
  if (custom > 0) {
    return t('personalSettings.notifications.customThreshold', {
      value: custom.toFixed(2),
    })
  }
  return t('personalSettings.notifications.systemDefault', {
    value: systemDefaultThreshold.value.toFixed(2),
  })
})
const recipientsLabel = computed(() => {
  const count = props.user?.balance_notify_extra_emails?.length ?? 0
  return count > 0
    ? t('personalSettings.notifications.recipientCount', { count })
    : t('personalSettings.notifications.accountEmailOnly')
})

async function handleToggle(event: Event) {
  if (!props.user || saving.value) return
  const target = event.target as HTMLInputElement
  saving.value = true
  try {
    const updated = await userAPI.updateProfile({
      balance_notify_enabled: target.checked,
    })
    authStore.user = updated
  } catch (error: unknown) {
    target.checked = notifyEnabled.value
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    saving.value = false
  }
}
</script>

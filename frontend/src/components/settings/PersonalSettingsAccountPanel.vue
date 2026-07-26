<template>
  <div
    class="personal-settings-panel"
    data-testid="personal-settings-account"
  >
    <template v-if="!detail">
      <div
        v-if="error"
        class="personal-settings-error"
        data-testid="personal-settings-account-error"
        role="alert"
      >
        <span>{{ t('personalSettings.account.loadError') }}</span>
        <button type="button" @click="emit('retry')">
          {{ t('personalSettings.common.retry') }}
        </button>
      </div>

      <div v-if="loading && !user" class="space-y-5 py-3" aria-busy="true">
        <div class="personal-settings-skeleton w-2/3"></div>
        <div class="personal-settings-skeleton w-full"></div>
        <div class="personal-settings-skeleton w-5/6"></div>
      </div>

      <div v-else class="personal-settings-group">
        <button
          type="button"
          class="personal-settings-row"
          data-testid="personal-settings-open-profile"
          @click="emit('open-detail', 'profile')"
        >
          <span class="personal-settings-avatar" aria-hidden="true">
            <img v-if="avatarUrl" :src="avatarUrl" alt="">
            <span v-else>{{ initials }}</span>
          </span>
          <span class="personal-settings-row-copy">
            <span class="personal-settings-row-title">
              {{ t('personalSettings.account.profile') }}
            </span>
            <span class="personal-settings-row-description">
              {{ t('personalSettings.account.profileHint') }}
            </span>
          </span>
          <span class="personal-settings-row-value">
            <span class="max-w-44 truncate">{{ displayName }}</span>
            <Icon name="chevronRight" size="xs" aria-hidden="true" />
          </span>
        </button>

        <button
          type="button"
          class="personal-settings-row"
          data-testid="personal-settings-open-connections"
          @click="emit('open-detail', 'connections')"
        >
          <Icon name="link" size="sm" class="text-[var(--lx-clay-text-muted)]" aria-hidden="true" />
          <span class="personal-settings-row-copy">
            <span class="personal-settings-row-title">
              {{ t('personalSettings.account.connections') }}
            </span>
            <span class="personal-settings-row-description">
              {{ t('personalSettings.account.connectionsHint') }}
            </span>
          </span>
          <span class="personal-settings-row-value">
            <span class="max-w-48 truncate">
              {{ user?.email || t('personalSettings.account.noEmail') }}
            </span>
            <Icon name="chevronRight" size="xs" aria-hidden="true" />
          </span>
        </button>
      </div>
    </template>

    <template v-else-if="detail === 'profile'">
      <div v-if="user" class="space-y-8" data-testid="personal-settings-profile-detail">
        <ProfileAvatarCard :user="user" embedded />
        <div class="border-t border-[var(--lx-clay-border)] pt-7">
          <ProfileEditForm :initial-username="user.username" embedded />
        </div>
      </div>
      <div v-else class="space-y-5 py-3" aria-busy="true">
        <div class="personal-settings-skeleton w-1/2"></div>
        <div class="personal-settings-skeleton h-16 w-full"></div>
      </div>
    </template>

    <template v-else>
      <div
        v-if="error"
        class="personal-settings-error mb-4"
        data-testid="personal-settings-connections-error"
        role="alert"
      >
        <span>{{ t('personalSettings.account.loadError') }}</span>
        <button type="button" @click="emit('retry')">
          {{ t('personalSettings.common.retry') }}
        </button>
      </div>
      <div
        v-if="loading && !publicSettings"
        class="space-y-5 py-3"
        aria-busy="true"
      >
        <div class="personal-settings-skeleton w-2/3"></div>
        <div class="personal-settings-skeleton w-full"></div>
      </div>
      <ProfileIdentityBindingsSection
        v-else-if="!error || !!publicSettings"
        :user="user"
        :linuxdo-enabled="publicSettings?.linuxdo_oauth_enabled"
        :dingtalk-enabled="publicSettings?.dingtalk_oauth_enabled"
        :oidc-enabled="publicSettings?.oidc_oauth_enabled"
        :oidc-provider-name="publicSettings?.oidc_oauth_provider_name"
        :wechat-enabled="wechatEnabled"
        :wechat-open-enabled="publicSettings?.wechat_oauth_open_enabled"
        :wechat-mp-enabled="publicSettings?.wechat_oauth_mp_enabled"
        embedded
        compact
        data-testid="personal-settings-connections-detail"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { isWeChatWebOAuthEnabled } from '@/api/auth'
import Icon from '@/components/icons/Icon.vue'
import ProfileAvatarCard from '@/components/user/profile/ProfileAvatarCard.vue'
import ProfileEditForm from '@/components/user/profile/ProfileEditForm.vue'
import ProfileIdentityBindingsSection from '@/components/user/profile/ProfileIdentityBindingsSection.vue'
import type { PersonalSettingsDetail } from '@/navigation/personalSettingsRoute'
import type { PublicSettings, User } from '@/types'

const props = defineProps<{
  user: User | null
  publicSettings: PublicSettings | null
  detail: PersonalSettingsDetail | null
  loading: boolean
  error: boolean
}>()

const emit = defineEmits<{
  retry: []
  'open-detail': [detail: 'profile' | 'connections']
}>()

const { t } = useI18n()
const displayName = computed(
  () => props.user?.username?.trim()
    || props.user?.email?.split('@')[0]?.trim()
    || t('profile.user'),
)
const initials = computed(
  () => Array.from(displayName.value)[0]?.toLocaleUpperCase() || 'U',
)
const avatarUrl = computed(() => props.user?.avatar_url?.trim() || '')
const wechatEnabled = computed(
  () => props.publicSettings
    ? isWeChatWebOAuthEnabled(props.publicSettings)
    : false,
)
</script>

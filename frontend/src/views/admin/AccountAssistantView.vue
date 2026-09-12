<template>
  <AppLayout variant="home-clay" content-mode="workbench">
    <div class="account-assistant-page" data-admin-page-kind="ops">
      <header class="account-assistant-page__header">
        <AdminPageHeader
          :title="t('admin.accounts.copilot.pageTitle')"
          :description="t('admin.accounts.copilot.pageDescription')"
        >
          <template #secondary-actions>
            <router-link
              to="/admin/accounts"
              class="btn btn-secondary min-h-11 px-4"
              data-test="account-assistant-back"
            >
              {{ t('admin.accounts.copilot.backToAccounts') }}
            </router-link>
          </template>
        </AdminPageHeader>
      </header>
      <AccountAssistantWorkspace />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import AdminPageHeader from '@/components/layout/AdminPageHeader.vue'
import AccountAssistantWorkspace from '@/components/admin/account/AccountAssistantWorkspace.vue'

const { t } = useI18n()
</script>

<style scoped>
.account-assistant-page {
  display: flex;
  min-height: calc(100dvh - var(--app-shell-top-offset, 0px));
  flex-direction: column;
  gap: 16px;
  padding: 20px 24px 24px;
}

.account-assistant-page__header {
  flex: none;
}

.account-assistant-page :deep(.account-assistant) {
  flex: 1;
  min-height: 0;
}
</style>

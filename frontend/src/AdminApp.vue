<script setup lang="ts">
import AppRoot from '@/app/AppRoot.vue'
import AdminComplianceRuntime from '@/app/AdminComplianceRuntime.vue'
import UserSubscriptionRuntime from '@/app/UserSubscriptionRuntime.vue'
import { useAdminSettingsStore } from '@/stores/adminSettings'

const adminSettingsStore = useAdminSettingsStore()
</script>

<template>
  <AppRoot :custom-menu-items="adminSettingsStore.customMenuItems" />
  <AdminComplianceRuntime />
  <UserSubscriptionRuntime />
</template>

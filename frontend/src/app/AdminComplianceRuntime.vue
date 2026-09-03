<script setup lang="ts">
import { onBeforeUnmount, onMounted, watch } from 'vue'
import AdminComplianceDialog from '@/components/admin/AdminComplianceDialog.vue'
import { useAdminComplianceStore } from '@/stores/adminCompliance'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
const adminComplianceStore = useAdminComplianceStore()

function onAdminComplianceRequired(event: Event) {
  const detail = (event as CustomEvent<Record<string, string>>).detail || {}
  adminComplianceStore.requireAcknowledgement(detail)
}

watch(
  [() => authStore.isAuthenticated, () => authStore.isAdmin],
  ([isAuthenticated, isAdmin]) => {
    if (isAuthenticated && isAdmin) {
      if (!adminComplianceStore.initialized) {
        adminComplianceStore.fetchStatus().catch((error) => {
          console.error('Failed to fetch admin compliance status:', error)
        })
      }
      return
    }
    adminComplianceStore.reset()
  },
  { immediate: true },
)

onMounted(() => {
  window.addEventListener('admin-compliance-required', onAdminComplianceRequired)
})

onBeforeUnmount(() => {
  window.removeEventListener('admin-compliance-required', onAdminComplianceRequired)
})
</script>

<template>
  <AdminComplianceDialog />
</template>

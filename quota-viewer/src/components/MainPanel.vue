<script setup lang="ts">
import FloatingQuotaWidget from '@/components/FloatingQuotaWidget.vue'
import type { ViewerUiStatus } from '@/composables/useQuotaViewer'
import type { QuotaOverview, ViewerDataStatus } from '@/types'

defineProps<{
  overview: QuotaOverview | null
  status?: ViewerUiStatus
  dataStatus?: ViewerDataStatus
  refreshing?: boolean
  lastRefreshAt?: number | null
  errorCode?: string | null
  errorMessage?: string | null
  pairingCode?: string | null
  collapsed?: boolean
}>()

const emit = defineEmits<{
  refresh: []
  connect: []
  reopen: []
  toggle: []
}>()
</script>

<template>
  <section
    class="floating-quota-viewport"
    :class="{ 'floating-quota-viewport--collapsed': collapsed }"
    aria-label="Codex 会员周积分"
  >
    <FloatingQuotaWidget
      :overview="overview"
      :status="status"
      :data-status="dataStatus"
      :refreshing="refreshing"
      :last-refresh-at="lastRefreshAt"
      :error-code="errorCode"
      :error-message="errorMessage"
      :pairing-code="pairingCode"
      :collapsed="collapsed"
      @refresh="emit('refresh')"
      @connect="emit('connect')"
      @reopen="emit('reopen')"
      @toggle="emit('toggle')"
    />
  </section>
</template>

<style scoped>
.floating-quota-viewport {
  position: relative;
  display: grid;
  width: 314px;
  height: 314px;
  overflow: hidden;
  place-items: center;
}

.floating-quota-viewport--collapsed {
  width: 80px;
  height: 80px;
}

</style>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import ConnectionStatePanel from '@/components/ConnectionStatePanel.vue'
import MainPanel from '@/components/MainPanel.vue'
import QuotaDetailPanel from '@/components/QuotaDetailPanel.vue'
import { useQuotaViewer } from '@/composables/useQuotaViewer'
import type { DemoPreviewState } from '@/data/demo'
import {
  hasTauriRuntime,
  hideQuotaViewer,
  onQuotaViewerShown,
  setDetailPanelOpen
} from '@/lib/desktop'
import type { QuotaOverview } from '@/types'

const previewOptions: Array<{
  value: DemoPreviewState
  label: string
}> = [
  { value: 'available', label: '可用' },
  { value: 'warning', label: '警告' },
  { value: 'exhausted', label: '耗尽' },
  { value: 'expired', label: '失效' },
  { value: 'no-membership', label: '无会员' }
]

const showDevStateSwitcher =
  import.meta.env.DEV &&
  document.documentElement.dataset.runtime === 'browser'
const isTauri = hasTauriRuntime()
const viewer = useQuotaViewer()

const previewState = ref<DemoPreviewState>('available')
const demoOverview = ref<QuotaOverview | null>(null)
let createDemoOverview:
  | ((state: DemoPreviewState) => QuotaOverview)
  | undefined
const overview = computed(() =>
  isTauri ? viewer.overview.value : demoOverview.value
)
const selectedQuotaId = ref<string | null>(null)
let unlistenViewerShown: (() => void) | undefined

const selectedQuota = computed(
  () =>
    overview.value?.quotas.find((quota) => quota.id === selectedQuotaId.value) ??
    null
)

const openQuota = (quotaId: string) => {
  selectedQuotaId.value = selectedQuotaId.value === quotaId ? null : quotaId
}

const closeDetail = () => {
  selectedQuotaId.value = null
}

const setPreviewState = (state: DemoPreviewState) => {
  if (!createDemoOverview) return
  previewState.value = state
  demoOverview.value = createDemoOverview(state)

  if (state === 'no-membership') {
    closeDetail()
  }
}

const hideViewer = async () => {
  selectedQuotaId.value = null
  await hideQuotaViewer()
}

const refreshViewer = () => {
  if (isTauri) void viewer.refresh()
}

const disconnectViewer = async () => {
  closeDetail()
  await viewer.disconnect()
}

const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape' && selectedQuotaId.value) {
    closeDetail()
  }
}

watch(
  () => Boolean(selectedQuota.value),
  (open) => {
    void setDetailPanelOpen(open)
  }
)

watch(
  () => overview.value?.quotas.map((quota) => quota.id).join(',') ?? '',
  () => {
    if (selectedQuotaId.value && !selectedQuota.value) closeDetail()
  }
)

onMounted(async () => {
  window.addEventListener('keydown', handleKeydown)
  unlistenViewerShown = await onQuotaViewerShown(() => {
    closeDetail()
    if (
      viewer.status.value === 'ready' ||
      viewer.status.value === 'stale'
    ) {
      void viewer.refresh()
    }
  })
  if (isTauri) {
    await viewer.boot()
    return
  }
  if (import.meta.env.DEV) {
    const demo = await import('@/data/demo')
    createDemoOverview = demo.createDemoOverview
    demoOverview.value = createDemoOverview(previewState.value)
    return
  }
  viewer.status.value = 'unavailable'
  viewer.errorCode.value = 'TAURI_RUNTIME_REQUIRED'
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleKeydown)
  unlistenViewerShown?.()
})
</script>

<template>
  <main class="monitor-stage" :class="{ 'monitor-stage--detail': selectedQuota }">
    <MainPanel
      v-if="overview"
      :overview="overview"
      :selected-quota-id="selectedQuotaId"
      :refreshing="isTauri && viewer.refreshing.value"
      :data-status="isTauri ? viewer.dataStatus.value : 'ready'"
      :connected="isTauri"
      @select-quota="openQuota"
      @refresh="refreshViewer"
      @disconnect="disconnectViewer"
      @close="hideViewer"
    />
    <ConnectionStatePanel
      v-else
      :status="viewer.status.value"
      :pairing="viewer.pairing.value"
      :error-code="viewer.errorCode.value"
      @connect="viewer.connect"
      @retry="viewer.refresh"
      @reopen="viewer.reopenPairing"
      @cancel="viewer.cancelConnection"
      @close="hideViewer"
    />

    <Transition name="detail-panel">
      <QuotaDetailPanel
        v-if="selectedQuota"
        :quota="selectedQuota"
        @close="closeDetail"
      />
    </Transition>
  </main>

  <aside
    v-if="showDevStateSwitcher"
    class="dev-state-switcher"
    aria-label="开发状态预览"
  >
    <header>
      <strong>状态预览</strong>
      <small>仅开发</small>
    </header>
    <div class="dev-state-switcher__options" role="group" aria-label="会员状态">
      <button
        v-for="option in previewOptions"
        :key="option.value"
        type="button"
        class="dev-state-option"
        :class="[
          `dev-state-option--${option.value}`,
          { 'dev-state-option--active': previewState === option.value }
        ]"
        :aria-pressed="previewState === option.value"
        :data-preview-state="option.value"
        @click="setPreviewState(option.value)"
      >
        <i aria-hidden="true" />
        {{ option.label }}
      </button>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import MainPanel from '@/components/MainPanel.vue'
import QuotaDetailPanel from '@/components/QuotaDetailPanel.vue'
import TrayPopover from '@/components/TrayPopover.vue'
import { useQuotaViewer } from '@/composables/useQuotaViewer'
import type { DemoPreviewState } from '@/data/demo'
import {
  hasTauriRuntime,
  hideTrayPopover,
  onMainPanelLayoutChanged,
  onTrayPopoverShown,
  onTrayPanelLayoutChanged,
  onQuotaViewerShown,
  openMainPanel,
  setMainPanelExpanded,
  setTrayDetailPanelOpen
} from '@/lib/desktop'
import type { QuotaOverview } from '@/types'

type PreviewState = DemoPreviewState | 'stale'

const previewOptions: Array<{
  value: PreviewState
  label: string
}> = [
  { value: 'available', label: '可用' },
  { value: 'warning', label: '警告' },
  { value: 'exhausted', label: '耗尽' },
  { value: 'expired', label: '失效' },
  { value: 'unknown', label: '暂不可用' },
  { value: 'stale', label: '旧数据' },
  { value: 'no-membership', label: '无会员' }
]

const isTraySurface =
  new URLSearchParams(window.location.search).get('surface') === 'tray'

const showDevStateSwitcher =
  !isTraySurface &&
  import.meta.env.DEV &&
  document.documentElement.dataset.runtime === 'browser'
const isTauri = hasTauriRuntime()
const viewer = useQuotaViewer({
  autoRefresh: !isTraySurface,
  refreshOnBoot: !isTraySurface
})

const previewState = ref<PreviewState>('available')
const previewDataStatus = ref<'ready' | 'stale'>('ready')
const demoOverview = ref<QuotaOverview | null>(null)
let createDemoOverview:
  | ((state: DemoPreviewState) => QuotaOverview)
  | undefined
let trayBooted = false
let trayLoading = false
const overview = computed(() =>
  isTauri ? viewer.overview.value : demoOverview.value
)
const selectedQuotaId = ref<string | null>(null)
const isMainExpanded = ref(!isTauri)
let unlistenSurfaceShown: (() => void) | undefined
let unlistenLayoutChanged: (() => void) | undefined

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

const setPreviewState = (state: PreviewState) => {
  if (!createDemoOverview) return
  previewState.value = state
  previewDataStatus.value = state === 'stale' ? 'stale' : 'ready'
  const nextOverview = createDemoOverview(
    state === 'stale' ? 'available' : state
  )
  // Production floating cards select only an authoritative current membership.
  // The browser-only state switcher deliberately promotes its single expired
  // fixture so designers can still inspect that otherwise unreachable surface.
  demoOverview.value =
    state === 'expired'
      ? {
          ...nextOverview,
          quotas: nextOverview.quotas.map((quota) => ({
            ...quota,
            isCurrentMembership: true
          }))
        }
      : nextOverview
  if (state === 'no-membership') closeDetail()
}

const hideTray = async () => {
  await hideTrayPopover()
}

const openFullViewer = async () => {
  closeDetail()
  await openMainPanel()
}

const refreshViewer = () => {
  if (isTauri) void viewer.refresh()
}

const toggleMainPanel = async () => {
  if (isTraySurface) return
  const expanded = !isMainExpanded.value
  try {
    isMainExpanded.value = await setMainPanelExpanded(expanded)
  } catch {
    isMainExpanded.value = !expanded
  }
}

const handleKeydown = (event: KeyboardEvent) => {
  if (event.key !== 'Escape') return
  if (isTraySurface) {
    if (selectedQuotaId.value) {
      closeDetail()
    } else {
      void hideTray()
    }
  } else if (isMainExpanded.value) {
    void toggleMainPanel()
  }
}

watch(
  () => Boolean(selectedQuota.value),
  (open) => {
    if (isTraySurface) void setTrayDetailPanelOpen(open)
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
  if (isTauri) {
    const refreshWhenShown = () => {
      if (
        viewer.status.value === 'loading' ||
        viewer.status.value === 'ready' ||
        viewer.status.value === 'stale' ||
        viewer.status.value === 'unavailable'
      ) {
        void viewer.refresh()
      }
    }

    if (isTraySurface) {
      unlistenLayoutChanged = await onTrayPanelLayoutChanged((open) => {
        if (!open) closeDetail()
      })
      unlistenSurfaceShown = await onTrayPopoverShown(() => {
        closeDetail()
        if (trayLoading) return
        trayLoading = true
        void (async () => {
          if (!trayBooted) {
            trayBooted = true
            await viewer.boot()
          }
          refreshWhenShown()
        })().finally(() => {
          trayLoading = false
        })
      })
      return
    }

    unlistenLayoutChanged = await onMainPanelLayoutChanged((expanded) => {
      isMainExpanded.value = expanded
    })
    unlistenSurfaceShown = await onQuotaViewerShown(refreshWhenShown)
    await viewer.boot()
    return
  }
  if (import.meta.env.DEV) {
    const demo = await import('@/data/demo')
    createDemoOverview = demo.createDemoOverview
    const initialPreviewState: DemoPreviewState =
      previewState.value === 'stale' ? 'available' : previewState.value
    demoOverview.value = createDemoOverview(initialPreviewState)
    return
  }
  viewer.status.value = 'unavailable'
  viewer.errorCode.value = 'TAURI_RUNTIME_REQUIRED'
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleKeydown)
  unlistenSurfaceShown?.()
  unlistenLayoutChanged?.()
})
</script>

<template>
  <main
    v-if="isTraySurface"
    class="tray-popover-shell"
    :class="{ 'tray-popover-shell--detail': selectedQuota }"
  >
    <TrayPopover
      :overview="overview"
      :status="viewer.status.value"
      :selected-quota-id="selectedQuotaId"
      :refreshing="isTauri && viewer.refreshing.value"
      :data-status="isTauri ? viewer.dataStatus.value : 'ready'"
      @select-quota="openQuota"
      @refresh="refreshViewer"
      @open-main="openFullViewer"
    />

    <Transition name="detail-panel">
      <QuotaDetailPanel
        v-if="selectedQuota"
        class="tray-detail-panel"
        :quota="selectedQuota"
        :draggable="false"
        @close="closeDetail"
      />
    </Transition>
  </main>

  <template v-else>
    <main
      class="monitor-stage floating-monitor-stage"
      :class="{ 'floating-monitor-stage--expanded': isMainExpanded }"
    >
      <MainPanel
        :overview="overview"
        :status="viewer.status.value"
        :data-status="isTauri ? viewer.dataStatus.value : previewDataStatus"
        :refreshing="isTauri && viewer.refreshing.value"
        :error-code="viewer.errorCode.value"
        :error-message="viewer.errorMessage.value"
        :pairing-code="viewer.pairing.value?.user_code"
        :collapsed="!isMainExpanded"
        @toggle="toggleMainPanel"
        @connect="viewer.connect"
        @refresh="viewer.refresh"
        @reopen="viewer.reopenPairing"
      />
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
</template>

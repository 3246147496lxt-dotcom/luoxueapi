import { computed, onMounted, onUnmounted, reactive, readonly, ref, shallowReadonly } from 'vue'
import { desktopApi } from '@/api'
import type { DesktopSettings, DesktopSnapshot, PairingState, RequestMetadata } from '@/types'

const snapshot = ref<DesktopSnapshot | null>(null)
const requests = ref<RequestMetadata[]>([])
const pairing = ref<PairingState>({ status: 'idle' })
const state = reactive({ loading: true, busy: '', error: '' })
let pairingTimer: ReturnType<typeof setTimeout> | undefined
let unlistenStateChanged: (() => void) | undefined

function stopPairingPolling() {
  if (pairingTimer) clearTimeout(pairingTimer)
  pairingTimer = undefined
}

function setDocumentTheme(theme: DesktopSettings['theme']) {
  const dark = theme === 'dark' || (theme === 'system' && matchMedia('(prefers-color-scheme: dark)').matches)
  document.documentElement.classList.toggle('dark', dark)
}

async function run<T>(name: string, operation: () => Promise<T>): Promise<T | undefined> {
  state.busy = name
  state.error = ''
  try {
    return await operation()
  } catch (error) {
    state.error = error instanceof Error ? error.message : String(error)
  } finally {
    state.busy = ''
  }
}

export function useDesktop() {
  async function refresh() {
    state.loading = true
    const value = await run('refresh', () => desktopApi.snapshot())
    if (value) {
      snapshot.value = value
      setDocumentTheme(value.settings.theme)
    }
    state.loading = false
  }

  async function loadRequests() {
    const value = await run('requests', () => desktopApi.requests())
    if (value) requests.value = value
  }

  async function startPairing() {
    stopPairingPolling()
    const value = await run('pairing', () => desktopApi.startPairing())
    if (value) {
      pairing.value = value
      if (value.status === 'waiting') schedulePairingPoll()
    }
  }

  async function pollPairing() {
    stopPairingPolling()
    const value = await run('pairing', () => desktopApi.pollPairing())
    if (!value) {
      schedulePairingPoll()
      return
    }
    pairing.value = value
    if (value.status === 'approved') {
      await refresh()
    } else if (value.status === 'waiting') {
      schedulePairingPoll()
    }
  }

  function schedulePairingPoll() {
    stopPairingPolling()
    pairingTimer = setTimeout(pollPairing, 5_000)
  }

  async function restartGateway() {
    const value = await run('gateway', () => desktopApi.restartGateway())
    if (value) snapshot.value = value
  }

  async function setTakeover(enabled: boolean) {
    const value = await run('takeover', () => desktopApi.setTakeover(enabled))
    if (value) snapshot.value = value
  }

  async function selectRoute(groupId: number, model: string) {
    const shouldCompleteFirstSetup = !snapshot.value?.selectedGroupId && (snapshot.value?.today.balance ?? 0) > 0
    const value = await run('route', () => desktopApi.selectRoute(groupId, model))
    if (!value) return
    snapshot.value = value
    if (shouldCompleteFirstSetup) {
      const configured = await run('takeover', () => desktopApi.setTakeover(true))
      if (configured) snapshot.value = configured
    }
  }

  async function updateSettings(settings: DesktopSettings) {
    const value = await run('settings', () => desktopApi.updateSettings(settings))
    if (value) {
      snapshot.value = value
      setDocumentTheme(value.settings.theme)
    }
  }

  async function checkForUpdates() {
    const value = await run('update', () => desktopApi.checkForUpdates())
    if (value) snapshot.value = value
  }

  async function installUpdate() {
    await run('update', () => desktopApi.installUpdate())
  }

  onMounted(async () => {
    await refresh()
    unlistenStateChanged = await desktopApi.onStateChanged(refresh)
  })
  onUnmounted(() => {
    stopPairingPolling()
    unlistenStateChanged?.()
    unlistenStateChanged = undefined
  })

  return {
    snapshot: shallowReadonly(snapshot), requests: shallowReadonly(requests), pairing: shallowReadonly(pairing), state: readonly(state),
    isRunning: computed(() => snapshot.value?.gatewayStatus === 'running'), refresh, loadRequests,
    startPairing, pollPairing, restartGateway, setTakeover, selectRoute, updateSettings,
    checkForUpdates, installUpdate
  }
}

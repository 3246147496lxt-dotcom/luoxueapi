import { computed, onBeforeUnmount, ref } from 'vue'
import { adaptQuotaOverview } from '@/api/quotaOverview'
import {
  cancelQuotaPairing,
  disconnectQuotaAccount,
  getViewerSnapshot,
  openQuotaPairingPage,
  pollQuotaPairing,
  refreshQuotaOverview,
  startQuotaPairing,
  type DesktopCommandError,
  type PairingState,
  type ViewerSnapshot
} from '@/lib/desktop'
import type { QuotaOverview, ViewerDataStatus } from '@/types'

export type ViewerUiStatus =
  | 'disconnected'
  | 'connecting'
  | 'loading'
  | 'ready'
  | 'stale'
  | 'unavailable'
  | 'auth-invalid'
  | 'pairing-expired'

const AUTO_REFRESH_MS = 5 * 60 * 1000

interface UseQuotaViewerOptions {
  autoRefresh?: boolean
  refreshOnBoot?: boolean
}

const displayTimezone = () =>
  Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'

const normalizeError = (error: unknown) => {
  const commandError = error as DesktopCommandError
  return {
    code: commandError?.code ?? 'QUOTA_VIEWER_ERROR',
    message: commandError?.message ?? '暂时无法获取额度，请稍后重试。'
  }
}

export const useQuotaViewer = (options: UseQuotaViewerOptions = {}) => {
  const status = ref<ViewerUiStatus>('loading')
  const overview = ref<QuotaOverview | null>(null)
  const pairing = ref<PairingState | null>(null)
  const refreshing = ref(false)
  const lastRefreshAt = ref<number | null>(null)
  const errorCode = ref<string | null>(null)
  const errorMessage = ref<string | null>(null)
  let requestSequence = 0
  let pollTimer: number | undefined
  let freshnessTimer: number | undefined
  let autoRefreshTimer: number | undefined

  const dataStatus = computed<ViewerDataStatus>(() =>
    status.value === 'stale' ||
    (status.value === 'unavailable' && overview.value !== null)
      ? 'stale'
      : 'ready'
  )

  const clearPollTimer = () => {
    if (pollTimer) window.clearTimeout(pollTimer)
    pollTimer = undefined
  }

  const clearFreshnessTimer = () => {
    if (freshnessTimer) window.clearTimeout(freshnessTimer)
    freshnessTimer = undefined
  }

  const scheduleFreshness = () => {
    clearFreshnessTimer()
    if (!overview.value || status.value !== 'ready') return
    const freshUntil = new Date(overview.value.freshUntil).getTime()
    if (!Number.isFinite(freshUntil)) return
    const delay = freshUntil - Date.now()
    if (delay <= 0) {
      status.value = 'stale'
      return
    }
    freshnessTimer = window.setTimeout(() => {
      if (status.value === 'ready') status.value = 'stale'
    }, Math.min(delay + 50, 2_147_483_647))
  }

  const applySnapshot = (snapshot: ViewerSnapshot, sequence: number) => {
    if (sequence !== requestSequence) return
    errorCode.value = snapshot.error_code
    errorMessage.value = snapshot.message

    if (snapshot.status === 'disconnected' || snapshot.status === 'auth-invalid') {
      overview.value = null
      status.value = snapshot.status
      clearFreshnessTimer()
      return
    }

    if (snapshot.overview) {
      try {
        overview.value = adaptQuotaOverview(snapshot.overview)
      } catch (error) {
        const normalized = normalizeError(error)
        errorCode.value = normalized.code
        errorMessage.value = normalized.message
        status.value = overview.value ? 'stale' : 'unavailable'
        return
      }
    } else if (snapshot.status === 'unavailable' && overview.value) {
      status.value = 'stale'
      clearFreshnessTimer()
      return
    } else if (snapshot.status === 'unavailable') {
      overview.value = null
    }

    status.value = snapshot.status
    scheduleFreshness()
  }

  const refresh = async () => {
    if (refreshing.value || status.value === 'connecting') return
    const sequence = ++requestSequence
    refreshing.value = true
    if (!overview.value) status.value = 'loading'
    try {
      const snapshot = await refreshQuotaOverview(displayTimezone())
      applySnapshot(snapshot, sequence)
    } catch (error) {
      if (sequence !== requestSequence) return
      const normalized = normalizeError(error)
      errorCode.value = normalized.code
      errorMessage.value = normalized.message
      status.value = overview.value ? 'stale' : 'unavailable'
    } finally {
      if (sequence === requestSequence) {
        refreshing.value = false
        lastRefreshAt.value = Date.now()
      }
    }
  }

  const poll = async () => {
    clearPollTimer()
    if (status.value !== 'connecting') return
    try {
      const next = await pollQuotaPairing()
      if (status.value !== 'connecting') return
      pairing.value = next
      if (next.status === 'approved') {
        status.value = 'loading'
        pairing.value = null
        await refresh()
        return
      }
      if (next.status === 'expired') {
        status.value = 'pairing-expired'
        return
      }
      pollTimer = window.setTimeout(poll, Math.max(5, next.interval) * 1000)
    } catch (error) {
      const normalized = normalizeError(error)
      errorCode.value = normalized.code
      errorMessage.value = normalized.message
      pollTimer = window.setTimeout(poll, 5000)
    }
  }

  const connect = async () => {
    clearPollTimer()
    errorCode.value = null
    errorMessage.value = null
    status.value = 'connecting'
    lastRefreshAt.value = null
    try {
      pairing.value = await startQuotaPairing()
      pollTimer = window.setTimeout(
        poll,
        Math.max(5, pairing.value.interval) * 1000
      )
    } catch (error) {
      const normalized = normalizeError(error)
      errorCode.value = normalized.code
      errorMessage.value = normalized.message
      status.value = 'unavailable'
    }
  }

  const reopenPairing = async () => {
    try {
      await openQuotaPairingPage()
    } catch (error) {
      const normalized = normalizeError(error)
      errorCode.value = normalized.code
      errorMessage.value = normalized.message
    }
  }

  const cancelConnection = async () => {
    clearPollTimer()
    pairing.value = null
    status.value = 'disconnected'
    await cancelQuotaPairing().catch(() => undefined)
  }

  const disconnect = async () => {
    ++requestSequence
    clearPollTimer()
    clearFreshnessTimer()
    pairing.value = null
    overview.value = null
    status.value = 'disconnected'
    refreshing.value = false
    lastRefreshAt.value = null
    errorCode.value = null
    errorMessage.value = null
    try {
      await disconnectQuotaAccount()
    } catch (error) {
      const normalized = normalizeError(error)
      errorCode.value = normalized.code
      errorMessage.value = normalized.message
    }
  }

  const boot = async () => {
    const sequence = ++requestSequence
    try {
      const snapshot = await getViewerSnapshot(displayTimezone())
      applySnapshot(snapshot, sequence)
      if (
        options.refreshOnBoot !== false &&
        (snapshot.status === 'loading' ||
          snapshot.status === 'ready' ||
          snapshot.status === 'stale')
      ) {
        void refresh()
      }
    } catch (error) {
      const normalized = normalizeError(error)
      errorCode.value = normalized.code
      errorMessage.value = normalized.message
      status.value = 'unavailable'
    }
    if (options.autoRefresh !== false) {
      autoRefreshTimer = window.setInterval(() => {
        if (
          status.value === 'ready' ||
          status.value === 'stale' ||
          status.value === 'unavailable'
        ) {
          void refresh()
        }
      }, AUTO_REFRESH_MS)
    }
  }

  onBeforeUnmount(() => {
    ++requestSequence
    clearPollTimer()
    clearFreshnessTimer()
    if (autoRefreshTimer) window.clearInterval(autoRefreshTimer)
  })

  return {
    status,
    overview,
    pairing,
    refreshing,
    lastRefreshAt,
    errorCode,
    errorMessage,
    dataStatus,
    boot,
    refresh,
    connect,
    reopenPairing,
    cancelConnection,
    disconnect
  }
}

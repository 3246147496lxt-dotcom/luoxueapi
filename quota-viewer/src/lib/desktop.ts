import { getCurrentWindow } from '@tauri-apps/api/window'

export type ViewerSnapshotStatus =
  | 'disconnected'
  | 'loading'
  | 'ready'
  | 'stale'
  | 'unavailable'
  | 'auth-invalid'

export interface ViewerSnapshot {
  status: ViewerSnapshotStatus
  overview: unknown | null
  source: 'cache' | 'network' | null
  fetched_at: string | null
  error_code: string | null
  message: string | null
}

export interface PairingState {
  status: 'waiting' | 'approved' | 'expired'
  user_code: string | null
  verification_uri: string | null
  expires_at: string | null
  interval: number
}

export interface DesktopCommandError {
  code?: string
  message?: string
  retryable?: boolean
}

export const hasTauriRuntime = () =>
  typeof window !== 'undefined' && Boolean(window.__TAURI_INTERNALS__)

const interactiveDragTargets =
  'button, a, input, select, textarea, [data-no-window-drag]'

export const startQuotaViewerDrag = (event: MouseEvent | PointerEvent) => {
  const primaryButtonHeld =
    event.type === 'pointermove'
      ? (event.buttons & 1) === 1
      : event.button === 0
  if (!hasTauriRuntime() || !primaryButtonHeld) return

  const target = event.target
  if (
    target instanceof Element &&
    target.closest(interactiveDragTargets)
  ) {
    return
  }

  void getCurrentWindow().startDragging().catch(() => undefined)
}

const call = async <T>(command: string, args?: Record<string, unknown>) => {
  const { invoke } = await import('@tauri-apps/api/core')
  return invoke<T>(command, args)
}

export const markRuntime = () => {
  document.documentElement.dataset.runtime = hasTauriRuntime() ? 'tauri' : 'browser'
  document.documentElement.dataset.surface =
    new URLSearchParams(window.location.search).get('surface') === 'tray'
      ? 'tray'
      : 'main'
}

export const setMainPanelExpanded = async (expanded: boolean) => {
  if (!hasTauriRuntime()) return expanded

  return call<boolean>('set_main_panel_expanded', { expanded })
}

export const setTrayDetailPanelOpen = async (open: boolean) => {
  if (!hasTauriRuntime()) return open

  return call<boolean>('set_tray_detail_open', { open })
}

export const hideQuotaViewer = async () => {
  if (!hasTauriRuntime()) return

  await call('hide_panel')
}

export const hideTrayPopover = async () => {
  if (!hasTauriRuntime()) return

  await call('hide_tray_panel')
}

export const openMainPanel = async () => {
  if (!hasTauriRuntime()) return

  await call('open_main_panel')
}

export const onQuotaViewerShown = async (callback: () => void) => {
  if (!hasTauriRuntime()) return () => {}

  const { listen } = await import('@tauri-apps/api/event')
  return listen('quota-panel-shown', callback)
}

export const onTrayPopoverShown = async (callback: () => void) => {
  if (!hasTauriRuntime()) return () => {}

  const { listen } = await import('@tauri-apps/api/event')
  return listen('quota-tray-shown', callback)
}

export const onMainPanelLayoutChanged = async (
  callback: (expanded: boolean) => void
) => {
  if (!hasTauriRuntime()) return () => {}

  const { listen } = await import('@tauri-apps/api/event')
  return listen<boolean>('quota-main-layout-changed', (event) => {
    callback(event.payload)
  })
}

export const onTrayPanelLayoutChanged = async (
  callback: (open: boolean) => void
) => {
  if (!hasTauriRuntime()) return () => {}

  const { listen } = await import('@tauri-apps/api/event')
  return listen<boolean>('quota-tray-layout-changed', (event) => {
    callback(event.payload)
  })
}

export const getViewerSnapshot = async (
  timezone: string
): Promise<ViewerSnapshot> =>
  call<ViewerSnapshot>('get_viewer_snapshot', { timezone })

export const refreshQuotaOverview = async (
  timezone: string
): Promise<ViewerSnapshot> =>
  call<ViewerSnapshot>('refresh_quota_overview', { timezone })

export const startQuotaPairing = async (): Promise<PairingState> =>
  call<PairingState>('start_quota_pairing')

export const pollQuotaPairing = async (): Promise<PairingState> =>
  call<PairingState>('poll_quota_pairing')

export const openQuotaPairingPage = async () => {
  await call('open_quota_pairing_page')
}

export const cancelQuotaPairing = async () => {
  await call('cancel_quota_pairing')
}

export const disconnectQuotaAccount = async () => {
  await call('disconnect_quota_account')
}

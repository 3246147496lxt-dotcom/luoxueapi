import { invoke } from '@tauri-apps/api/core'
import { listen, type UnlistenFn } from '@tauri-apps/api/event'
import type {
  DesktopSettings,
  DesktopSnapshot,
  DiagnosticReceipt,
  DiagnosticSummary,
  PairingState,
  RequestMetadata
} from './types'

const isTauri = typeof window !== 'undefined' && '__TAURI_INTERNALS__' in window

const mockSnapshot: DesktopSnapshot = {
  paired: true,
  accountEmail: 'pu@luoxueapi.cc',
  deviceName: 'Pu 的 MacBook Pro',
  gatewayStatus: 'running',
  gatewayPort: 11430,
  takeoverEnabled: true,
  configStatus: 'managed',
  selectedGroupId: 12,
  selectedModel: 'gpt-5.3-codex',
  routes: [
    { groupId: 12, name: 'Codex 高速线路', rateMultiplier: 0.36, models: ['gpt-5.3-codex', 'gpt-5.2-codex'] },
    { groupId: 18, name: 'OpenAI 标准线路', rateMultiplier: 0.58, models: ['gpt-5.3-codex', 'gpt-5.4'] }
  ],
  installations: [
    { kind: 'cli', installed: true, version: '0.141.0', path: '/opt/homebrew/bin/codex' },
    { kind: 'desktop', installed: true, version: '2.1.0', path: '/Applications/Codex.app' }
  ],
  today: { requests: 65, tokens: 56_130_000, cost: 3.28, balance: 42.86, averageFirstTokenMs: 3039 },
  settings: { launchAtLogin: true, notifications: true, retentionDays: 7, theme: 'system', locale: 'zh-CN' },
  update: { state: 'current', currentVersion: '0.1.0' }
}

const mockRequests: RequestMetadata[] = [
  { id: '1', occurredAt: new Date(Date.now() - 4 * 60_000).toISOString(), model: 'gpt-5.3-codex', statusCode: 200, inputTokens: 18420, outputTokens: 2864, durationMs: 18420, firstTokenMs: 2610, requestId: 'req_01K2F9XS8M4G' },
  { id: '2', occurredAt: new Date(Date.now() - 31 * 60_000).toISOString(), model: 'gpt-5.3-codex', statusCode: 200, inputTokens: 9628, outputTokens: 1190, durationMs: 9670, firstTokenMs: 2840, requestId: 'req_01K2F8VJQRA5' },
  { id: '3', occurredAt: new Date(Date.now() - 74 * 60_000).toISOString(), model: 'gpt-5.2-codex', statusCode: 429, inputTokens: 0, outputTokens: 0, durationMs: 410, requestId: 'req_01K2F6Y4J5F0' }
]

async function call<T>(command: string, args?: Record<string, unknown>): Promise<T> {
  return invoke<T>(command, args)
}

export const desktopApi = {
  async snapshot(): Promise<DesktopSnapshot> {
    return isTauri ? call('get_snapshot') : structuredClone(mockSnapshot)
  },
  async requests(): Promise<RequestMetadata[]> {
    return isTauri ? call('list_requests') : structuredClone(mockRequests)
  },
  async startPairing(): Promise<PairingState> {
    if (isTauri) return call('start_pairing')
    return { status: 'waiting', userCode: 'LX7P-K9Q2', verificationUri: 'https://luoxueapi.cc/desktop/authorize', expiresAt: new Date(Date.now() + 600_000).toISOString() }
  },
  async pollPairing(): Promise<PairingState> {
    return isTauri ? call('poll_pairing') : { status: 'approved' }
  },
  async openPairingPage(): Promise<void> {
    if (isTauri) await call('open_pairing_page')
  },
  async restartGateway(): Promise<DesktopSnapshot> {
    return isTauri ? call('restart_gateway') : structuredClone(mockSnapshot)
  },
  async setTakeover(enabled: boolean): Promise<DesktopSnapshot> {
    if (isTauri) return call('set_takeover', { enabled })
    mockSnapshot.takeoverEnabled = enabled
    mockSnapshot.configStatus = enabled ? 'managed' : 'clean'
    return structuredClone(mockSnapshot)
  },
  async selectRoute(groupId: number, model: string): Promise<DesktopSnapshot> {
    if (isTauri) return call('select_route', { groupId, model })
    mockSnapshot.selectedGroupId = groupId
    mockSnapshot.selectedModel = model
    return structuredClone(mockSnapshot)
  },
  async updateSettings(settings: DesktopSettings): Promise<DesktopSnapshot> {
    if (isTauri) return call('update_settings', { settings })
    mockSnapshot.settings = settings
    return structuredClone(mockSnapshot)
  },
  async openCodex(): Promise<void> {
    if (isTauri) await call('open_codex')
  },
  async openAccountPage(destination: 'routes' | 'balance'): Promise<void> {
    if (isTauri) await call('open_account_page', { destination })
  },
  async copyCliCommand(): Promise<string> {
    const command = isTauri ? await call<string>('cli_launch_command') : 'codex'
    await navigator.clipboard.writeText(command)
    return command
  },
  async checkForUpdates(): Promise<DesktopSnapshot> {
    return isTauri ? call('check_for_updates') : structuredClone(mockSnapshot)
  },
  async installUpdate(): Promise<void> {
    if (isTauri) await call('install_update')
  },
  async diagnosticSummary(): Promise<DiagnosticSummary> {
    if (isTauri) return call('diagnostic_summary')
    return {
      appVersion: '0.1.0', platform: 'macos', architecture: 'arm64', osVersion: '15.0',
      gateway: { status: mockSnapshot.gatewayStatus, port: mockSnapshot.gatewayPort, takeoverEnabled: mockSnapshot.takeoverEnabled },
      codex: {
        configStatus: mockSnapshot.configStatus,
        installations: mockSnapshot.installations.map(({ kind, installed, version }) => ({ kind, installed, version }))
      },
      route: { groupId: mockSnapshot.selectedGroupId, model: mockSnapshot.selectedModel, availableRouteCount: mockSnapshot.routes.length },
      requests: {
        sampleCount: mockRequests.length,
        successCount: mockRequests.filter((request) => request.statusCode >= 200 && request.statusCode < 300).length,
        errorCount: mockRequests.filter((request) => request.statusCode < 200 || request.statusCode >= 300).length,
        averageDurationMs: Math.round(mockRequests.reduce((sum, request) => sum + request.durationMs, 0) / mockRequests.length),
        averageFirstTokenMs: Math.round(mockRequests.reduce((sum, request) => sum + (request.firstTokenMs ?? 0), 0) / mockRequests.length),
        recent: mockRequests.map(({ occurredAt, model, statusCode, durationMs, firstTokenMs, requestId }) => ({ occurredAt, model, statusCode, durationMs, firstTokenMs, requestId }))
      }
    }
  },
  async uploadDiagnostic(): Promise<DiagnosticReceipt> {
    if (isTauri) return call('upload_diagnostic')
    return { id: 'diag_mock', createdAt: new Date().toISOString(), expiresAt: new Date(Date.now() + 7 * 86_400_000).toISOString() }
  },
  async prepareUninstall(): Promise<void> {
    if (isTauri) await call('prepare_uninstall')
  },
  async logout(): Promise<void> {
    if (isTauri) await call('logout')
  },
  async onStateChanged(handler: () => void): Promise<UnlistenFn> {
    if (!isTauri) return () => undefined
    return listen('desktop-state-changed', handler)
  }
}

export type ViewName = 'overview' | 'codex' | 'requests' | 'settings'
export type ThemeMode = 'system' | 'light' | 'dark'
export type GatewayStatus = 'running' | 'stopped' | 'error'
export type ConfigStatus = 'clean' | 'managed' | 'conflict' | 'invalid' | 'permission_denied'

export interface RouteOption {
  groupId: number
  name: string
  rateMultiplier: number
  models: string[]
}

export interface CodexInstallation {
  kind: 'cli' | 'desktop'
  installed: boolean
  version?: string
  path?: string
}

export interface RequestMetadata {
  id: string
  occurredAt: string
  model: string
  statusCode: number
  inputTokens: number
  outputTokens: number
  durationMs: number
  firstTokenMs?: number
  requestId: string
}

export interface DesktopSettings {
  launchAtLogin: boolean
  notifications: boolean
  retentionDays: number
  theme: ThemeMode
  locale: 'zh-CN' | 'en'
}

export interface PairingState {
  status: 'idle' | 'waiting' | 'approved' | 'expired' | 'error'
  userCode?: string
  verificationUri?: string
  expiresAt?: string
}

export interface DiagnosticSummary {
  appVersion: string
  platform: string
  architecture: string
  osVersion: string
  gateway: {
    status: string
    port?: number
    takeoverEnabled: boolean
  }
  codex: {
    configStatus: ConfigStatus
    installations: Array<Pick<CodexInstallation, 'kind' | 'installed' | 'version'>>
  }
  route: {
    groupId?: number
    model?: string
    availableRouteCount: number
  }
  requests: {
    sampleCount: number
    successCount: number
    errorCount: number
    averageDurationMs?: number
    averageFirstTokenMs?: number
    recent: Array<Pick<RequestMetadata, 'occurredAt' | 'model' | 'statusCode' | 'durationMs' | 'firstTokenMs' | 'requestId'>>
  }
}

export interface DiagnosticReceipt {
  id: string
  createdAt: string
  expiresAt: string
}

export interface DesktopSnapshot {
  paired: boolean
  accountEmail?: string
  deviceName: string
  gatewayStatus: GatewayStatus
  gatewayPort?: number
  takeoverEnabled: boolean
  configStatus: ConfigStatus
  configMessage?: string
  selectedGroupId?: number
  selectedModel?: string
  routes: RouteOption[]
  installations: CodexInstallation[]
  today: {
    requests: number
    tokens: number
    cost: number
    balance: number
    averageFirstTokenMs?: number
  }
  settings: DesktopSettings
  update?: {
    state: 'idle' | 'checking' | 'available' | 'downloading' | 'restart_required' | 'current' | 'error'
    currentVersion: string
    version?: string
  }
}

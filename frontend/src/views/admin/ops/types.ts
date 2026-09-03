// Ops 前端视图层的共享类型（与后端 DTO 解耦）。

export type ChartState = 'loading' | 'empty' | 'ready'

/**
 * Real alert context that can be handed to the log investigation workspace.
 * Group and region remain contextual because the system-log API cannot filter
 * by either field today.
 */
export interface OpsAlertLogContext {
  alertId: number
  firedAt: string
  resolvedAt?: string | null
  title?: string
  severity?: string
  status?: string
  platform?: string
  groupId?: number
  region?: string
  requestId?: string
}

/** A materialized log-query window created from an alert investigation. */
export interface OpsLogInvestigationPreset extends OpsAlertLogContext {
  key: number
  startTime: string
  endTime: string
}

// Re-export ops alert/settings types so view components can import from a single place
// while keeping the API contract centralized in `@/api/admin/ops`.
export type {
  AlertRule,
  AlertEvent,
  AlertSeverity,
  ThresholdMode,
  MetricType,
  Operator,
  EmailNotificationConfig,
  OpsDistributedLockSettings,
  OpsAlertRuntimeSettings,
  OpsMetricThresholds,
  OpsAdvancedSettings,
  OpsDataRetentionSettings,
  OpsAggregationSettings,
  OpsRuntimeLogConfig,
  OpsSystemLog,
  OpsSystemLogSinkHealth
} from '@/api/admin/ops'

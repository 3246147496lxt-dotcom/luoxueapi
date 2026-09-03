import type { SkillImportItemStatus, SkillImportRunStatus } from '@/api/admin/skillImport'

export type CanonicalSkillImportRunStatus =
  | 'queued'
  | 'discovering'
  | 'preparing'
  | 'waiting_retry'
  | 'ready'
  | 'awaiting_review'
  | 'publishing'
  | 'succeeded'
  | 'partial_succeeded'
  | 'failed'
  | 'cancelled'

export type CanonicalSkillImportItemStatus =
  | 'queued'
  | 'processing'
  | 'ready'
  | 'unchanged'
  | 'blocked'
  | 'failed'
  | 'published'
  | 'skipped'
  | 'cancelled'

const runAliases: Partial<Record<SkillImportRunStatus, CanonicalSkillImportRunStatus>> = {
  running: 'preparing',
  partial: 'partial_succeeded',
  canceled: 'cancelled',
}

const itemAliases: Partial<Record<SkillImportItemStatus, CanonicalSkillImportItemStatus>> = {
  pending: 'queued',
  fetching: 'processing',
  prepared: 'ready',
  canceled: 'cancelled',
}

export const activeSkillImportRunStatuses = new Set<CanonicalSkillImportRunStatus>([
  'queued',
  'discovering',
  'preparing',
  'waiting_retry',
  'publishing',
])

export function canonicalRunStatus(status: SkillImportRunStatus): CanonicalSkillImportRunStatus {
  return runAliases[status] ?? status as CanonicalSkillImportRunStatus
}

export function canonicalItemStatus(status: SkillImportItemStatus): CanonicalSkillImportItemStatus {
  return itemAliases[status] ?? status as CanonicalSkillImportItemStatus
}

export function isSkillImportRunActive(status: SkillImportRunStatus): boolean {
  return activeSkillImportRunStatuses.has(canonicalRunStatus(status))
}

export function skillImportPollDelay(failureCount: number): number {
  if (failureCount <= 0) return 2_000
  if (failureCount === 1) return 5_000
  if (failureCount === 2) return 10_000
  return 30_000
}

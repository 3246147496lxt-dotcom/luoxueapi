import type { OpsErrorLog } from '@/api/admin/ops'

export interface ErrorPhasePresentation {
  labelKey?: string
  fallback: string
  className: string
}

export function isUpstreamError(log: Pick<OpsErrorLog, 'phase' | 'error_owner'> | null): boolean {
  if (!log) return false
  return String(log.phase || '').toLowerCase() === 'upstream'
    && String(log.error_owner || '').toLowerCase() === 'provider'
}

export function errorPhasePresentation(
  log: Pick<OpsErrorLog, 'phase' | 'error_owner'>
): ErrorPhasePresentation {
  const phase = String(log.phase || '').trim().toLowerCase()
  const owner = String(log.error_owner || '').trim().toLowerCase()

  if (isUpstreamError(log)) {
    return {
      labelKey: 'admin.ops.errorLog.typeUpstream',
      fallback: phase,
      className: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
    }
  }
  if (phase === 'request' && owner === 'client') {
    return {
      labelKey: 'admin.ops.errorLog.typeRequest',
      fallback: phase,
      className: 'bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-200'
    }
  }
  if (phase === 'auth' && owner === 'client') {
    return {
      labelKey: 'admin.ops.errorLog.typeAuth',
      fallback: phase,
      className: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
    }
  }
  if (phase === 'account_auth') {
    return {
      labelKey: 'admin.ops.errorLog.typeAccountAuth',
      fallback: phase,
      className: 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200'
    }
  }
  if (phase === 'routing' && owner === 'platform') {
    return {
      labelKey: 'admin.ops.errorLog.typeRouting',
      fallback: phase,
      className: 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200'
    }
  }
  if (phase === 'internal' && owner === 'platform') {
    return {
      labelKey: 'admin.ops.errorLog.typeInternal',
      fallback: phase,
      className: 'bg-gray-100 text-gray-800 dark:bg-dark-700 dark:text-gray-200'
    }
  }

  const knownPhaseClasses: Record<string, string> = {
    request: 'bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-200',
    auth: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
    account_auth: 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200',
    routing: 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200',
    upstream: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
    network: 'bg-rose-100 text-rose-800 dark:bg-rose-900 dark:text-rose-200',
    internal: 'bg-gray-100 text-gray-800 dark:bg-dark-700 dark:text-gray-200'
  }
  if (knownPhaseClasses[phase]) {
    return {
      labelKey: `admin.ops.errorDetails.phase.${phase}`,
      fallback: phase,
      className: knownPhaseClasses[phase]
    }
  }

  return {
    fallback: phase || owner,
    className: 'bg-gray-100 text-gray-800 dark:bg-dark-700 dark:text-gray-200'
  }
}

export function errorCategoryBadgeClass(category: string): string {
  switch (category) {
    case 'auth':
      return 'bg-blue-100 text-blue-800 dark:bg-blue-900/40 dark:text-blue-200'
    case 'rate_limit':
    case 'quota':
      return 'bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-200'
    case 'upstream':
    case 'service_unavailable':
    case 'internal':
    case 'cyber':
      return 'bg-rose-100 text-rose-800 dark:bg-rose-900/40 dark:text-rose-200'
    case 'invalid_request':
      return 'bg-violet-100 text-violet-800 dark:bg-violet-900/40 dark:text-violet-200'
    default:
      return 'bg-gray-100 text-gray-800 dark:bg-dark-700 dark:text-gray-200'
  }
}

export function normalizedErrorPriority(severity?: string | null): string {
  const value = String(severity || '').trim().toUpperCase()
  return /^P[0-3]$/.test(value) ? value : ''
}

export function errorOwnerLabelKey(owner?: string | null): string | null {
  const value = String(owner || '').trim().toLowerCase()
  return ['provider', 'client', 'platform'].includes(value)
    ? `admin.ops.errorDetails.owner.${value}`
    : null
}

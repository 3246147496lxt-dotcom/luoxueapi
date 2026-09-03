const LEGACY_SETTLEMENT_FAILURE_CODE = 'CHAT_SETTLEMENT_FAILED'

const COMPLETED_CHAT_FINISH_REASONS = new Set([
  'stop',
  'length',
  'tool_calls',
  'content_filter',
])

interface SettlementDeliveryCandidate {
  role: unknown
  status: unknown
  content: unknown
  finishReason?: unknown
  errorCode?: unknown
}

export function isSettlementFailureWithCompletedDelivery(
  candidate: SettlementDeliveryCandidate,
): boolean {
  if (
    candidate.role !== 'assistant'
    || candidate.errorCode !== LEGACY_SETTLEMENT_FAILURE_CODE
  ) {
    return false
  }
  if (candidate.status === 'complete') return true
  return candidate.status === 'error'
    && typeof candidate.content === 'string'
    && candidate.content.trim().length > 0
    && typeof candidate.finishReason === 'string'
    && COMPLETED_CHAT_FINISH_REASONS.has(candidate.finishReason)
}

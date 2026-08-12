import { describe, expect, it } from 'vitest'
import {
  canonicalItemStatus,
  canonicalRunStatus,
  isSkillImportRunActive,
  skillImportPollDelay,
} from '../status'

describe('Skill import status compatibility', () => {
  it('keeps canonical statuses and maps rollout aliases', () => {
    expect(canonicalRunStatus('partial_succeeded')).toBe('partial_succeeded')
    expect(canonicalRunStatus('partial')).toBe('partial_succeeded')
    expect(canonicalRunStatus('canceled')).toBe('cancelled')
    expect(canonicalItemStatus('fetching')).toBe('processing')
    expect(canonicalItemStatus('prepared')).toBe('ready')
  })

  it('polls only while work can still move', () => {
    expect(isSkillImportRunActive('discovering')).toBe(true)
    expect(isSkillImportRunActive('waiting_retry')).toBe(true)
    expect(isSkillImportRunActive('awaiting_review')).toBe(false)
    expect(isSkillImportRunActive('failed')).toBe(false)
  })

  it('uses bounded polling backoff', () => {
    expect([0, 1, 2, 3, 12].map(skillImportPollDelay)).toEqual([
      2_000,
      5_000,
      10_000,
      30_000,
      30_000,
    ])
  })
})

import { describe, expect, it } from 'vitest'

import {
  DEFAULT_GROUP_EDITOR_SECTION,
  GROUP_EDITOR_SECTION_IDS,
  isCanonicalGroupEditorSection,
  normalizeGroupEditorSection,
  parsePositiveGroupId
} from '../groupEditorRoute'

describe('group editor route state', () => {
  it('keeps the public section vocabulary stable', () => {
    expect(GROUP_EDITOR_SECTION_IDS).toEqual([
      'general',
      'access',
      'models',
      'pricing',
      'routing',
      'advanced'
    ])
    expect(DEFAULT_GROUP_EDITOR_SECTION).toBe('general')
  })

  it('normalizes missing, array and invalid sections to general', () => {
    expect(normalizeGroupEditorSection(undefined)).toBe('general')
    expect(normalizeGroupEditorSection('unknown')).toBe('general')
    expect(normalizeGroupEditorSection(['pricing'])).toBe('general')
    expect(normalizeGroupEditorSection('pricing')).toBe('pricing')
    expect(isCanonicalGroupEditorSection('routing')).toBe(true)
    expect(isCanonicalGroupEditorSection('')).toBe(false)
  })

  it('accepts only positive safe integer route IDs', () => {
    expect(parsePositiveGroupId('42')).toBe(42)
    expect(parsePositiveGroupId('0')).toBeNull()
    expect(parsePositiveGroupId('-1')).toBeNull()
    expect(parsePositiveGroupId('1.5')).toBeNull()
    expect(parsePositiveGroupId(['42'])).toBeNull()
    expect(parsePositiveGroupId('9007199254740992')).toBeNull()
  })
})

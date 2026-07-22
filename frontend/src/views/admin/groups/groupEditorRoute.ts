export const GROUP_EDITOR_SECTION_IDS = [
  'general',
  'access',
  'models',
  'pricing',
  'routing',
  'advanced'
] as const

export type GroupEditorSectionId = (typeof GROUP_EDITOR_SECTION_IDS)[number]

export const DEFAULT_GROUP_EDITOR_SECTION: GroupEditorSectionId = 'general'

const groupEditorSectionSet = new Set<string>(GROUP_EDITOR_SECTION_IDS)

export function normalizeGroupEditorSection(value: unknown): GroupEditorSectionId {
  if (typeof value !== 'string' || !groupEditorSectionSet.has(value)) {
    return DEFAULT_GROUP_EDITOR_SECTION
  }
  return value as GroupEditorSectionId
}

export function isCanonicalGroupEditorSection(value: unknown): value is GroupEditorSectionId {
  return typeof value === 'string' && groupEditorSectionSet.has(value)
}

export function parsePositiveGroupId(value: unknown): number | null {
  if (typeof value !== 'string' || !/^[1-9]\d*$/.test(value)) {
    return null
  }
  const id = Number(value)
  return Number.isSafeInteger(id) ? id : null
}

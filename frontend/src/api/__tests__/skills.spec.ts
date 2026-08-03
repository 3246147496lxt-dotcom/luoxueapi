import { describe, expect, it } from 'vitest'
import {
  getSkillVersionDownloadURL,
  normalizePublicSkill,
  normalizeSkillCatalogResponse,
} from '../skills'

describe('public Skill API contract', () => {
  it('normalizes a current version object and released_at timestamps', () => {
    const skill = normalizePublicSkill({
      slug: 'api-docs',
      display_name: 'API Docs',
      tags: ['docs', null, 'api'],
      current_version: {
        version: '1.2.0',
        released_at: '2026-08-03T10:00:00Z',
        skill_md: '# API Docs',
        file_manifest: [{ path: 'SKILL.md', byte_size: 48, sha256: 'abc' }],
        validation_report: {
          valid: true,
          warnings: [{ code: 'NETWORK_REFERENCE', message: 'Uses the network', path: 'api-docs/SKILL.md' }],
        },
      },
    })

    expect(skill.current_version).toMatchObject({
      version: '1.2.0',
      created_at: '2026-08-03T10:00:00Z',
      skill_md: '# API Docs',
    })
    expect(skill.current_version?.file_manifest).toEqual([
      { path: 'SKILL.md', byte_size: 48, sha256: 'abc' },
    ])
    expect(skill.current_version?.validation_report.warnings).toEqual([
      { code: 'NETWORK_REFERENCE', message: 'Uses the network', path: 'api-docs/SKILL.md' },
    ])
    expect(skill.tags).toEqual(['docs', 'api'])
  })

  it('normalizes string categories for task filters', () => {
    const catalog = normalizeSkillCatalogResponse({
      items: [{ slug: 'testing', display_name: 'Testing' }],
      total: 8,
      page: 2,
      page_size: 1,
      categories: ['testing', 'documentation'],
    })

    expect(catalog).toMatchObject({ total: 8, page: 2, page_size: 1 })
    expect(catalog.categories).toEqual([
      { slug: 'testing', label: 'testing', count: null },
      { slug: 'documentation', label: 'documentation', count: null },
    ])
  })

  it('builds only an exact-version download URL and encodes path segments', () => {
    expect(getSkillVersionDownloadURL('api/docs', '1.0.0+build')).toBe(
      '/api/v1/catalog/skills/api%2Fdocs/versions/1.0.0%2Bbuild/download',
    )
  })
})

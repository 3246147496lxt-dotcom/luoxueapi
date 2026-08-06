import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SkillDetailView from '@/views/public/SkillDetailView.vue'
import type { PublicSkill, PublicSkillVersion } from '@/api/skills'

const api = vi.hoisted(() => ({
  getSkill: vi.fn(),
  getVersions: vi.fn(),
}))

const clipboardState = vi.hoisted(() => ({
  copy: vi.fn().mockResolvedValue(true),
}))

vi.mock('@/api/skills', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/skills')>()
  return {
    ...actual,
    getPublicSkill: api.getSkill,
    getPublicSkillVersions: api.getVersions,
  }
})

vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-router')>()
  return {
    ...actual,
    useRoute: () => ({ params: { slug: 'frontend-design' } }),
  }
})

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      locale: { value: 'zh-CN' },
      t: (key: string) => key,
      te: () => false,
    }),
  }
})

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copyToClipboard: clipboardState.copy }),
}))

function version(): PublicSkillVersion {
  return {
    version: '1.0.0',
    changelog: 'Initial release',
    manifest_name: 'frontend-design',
    manifest_description: 'Design guidance',
    skill_md: '---\nname: frontend-design\n---\n# Frontend Design\n\nOriginal guidance.',
    sha256: 'a'.repeat(64),
    byte_size: 7606,
    unpacked_size: 18395,
    file_count: 2,
    file_manifest: [{ path: 'frontend-design/SKILL.md', byte_size: 8221, sha256: 'b'.repeat(64) }],
    validation_report: {
      valid: true,
      errors: [],
      warnings: [{ code: 'NETWORK_REFERENCE', message: 'risk-warning-copy', path: 'LICENSE.txt' }],
    },
    created_at: '2026-08-04T00:00:00Z',
    download_count: 1,
  }
}

function skill(overrides: Partial<PublicSkill> = {}): PublicSkill {
  const current = version()
  return {
    slug: 'frontend-design',
    display_name: 'frontend-design',
    summary: '创建具有鲜明辨识度与高设计品质的生产级前端界面。',
    description: '这是 Anthropic frontend-design 指引 Skill。',
    category: 'design-ui',
    tags: ['ui/ux', '前端设计'],
    icon: '',
    example_prompts: ['Design a landing page.'],
    risk_notes: 'risk-note-copy',
    featured: true,
    current_version: current,
    download_count: 1,
    published_at: '2026-08-04T00:00:00Z',
    updated_at: '2026-08-04T00:00:00Z',
    versions: [current],
    ...overrides,
  }
}

function mountView() {
  return mount(SkillDetailView, {
    global: {
      stubs: {
        PublicSiteLayout: { template: '<div><slot /></div>' },
        SafeMarkdown: { props: ['content'], template: '<div class="safe-markdown-stub">{{ content }}</div>' },
        SkillInstallPanel: {
          props: ['version', 'sha256'],
          template: '<section class="skill-install-stub" :data-version="version" :data-sha="sha256">install</section>',
        },
        GitHubMark: true,
        Icon: true,
        RouterLink: { props: ['to'], template: '<a><slot /></a>' },
      },
    },
  })
}

describe('SkillDetailView', () => {
  beforeEach(() => {
    api.getSkill.mockReset().mockResolvedValue(skill())
    api.getVersions.mockReset().mockResolvedValue([])
    clipboardState.copy.mockClear()
  })

  it('keeps the reading flow focused on install, overview, supporting info, and SKILL.md', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.get('h1').text()).toBe('frontend-design')
    expect(wrapper.text()).toContain('创建具有鲜明辨识度与高设计品质的生产级前端界面。')
    expect(wrapper.find('.skill-detail-hero__mark').exists()).toBe(false)
    expect(wrapper.find('.skill-detail-meta').exists()).toBe(false)

    const orderedBlocks = wrapper.find('.skill-detail-layout').element.children
    expect(Array.from(orderedBlocks).map((element) => element.className)).toEqual([
      expect.stringContaining('skill-detail-install'),
      'skill-detail-overview',
      'skill-detail-info',
      'skill-detail-source',
    ])

    expect(wrapper.get('.skill-detail-overview__card').text()).toContain('这是 Anthropic frontend-design 指引 Skill。')
    expect(wrapper.find('.skill-detail-review').exists()).toBe(false)
    expect(wrapper.find('.skill-detail-closing').exists()).toBe(false)
    expect(wrapper.find('#skill-risk-title').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('risk-note-copy')
    expect(wrapper.text()).not.toContain('risk-warning-copy')
    expect(wrapper.text()).not.toContain('LICENSE.txt')
    expect(wrapper.text()).not.toContain('a'.repeat(64))
    expect(wrapper.text()).not.toContain('frontend-design/SKILL.md')
    expect(wrapper.get('.skill-install-stub').attributes('data-version')).toBe('1.0.0')
    expect(wrapper.get('.skill-install-stub').attributes('data-sha')).toBe('a'.repeat(64))
  })

  it('shows source metadata only when the API supplies it', async () => {
    api.getSkill.mockResolvedValue(skill({
      source_url: 'https://github.com/anthropics/skills/tree/main/skills/frontend-design',
      source_repository: 'anthropics/skills',
      repository_stars: 166_100,
    }))
    const wrapper = mountView()
    await flushPromises()

    const sourceLink = wrapper.get('.skill-detail-info__source a')
    expect(sourceLink.attributes('href')).toBe('https://github.com/anthropics/skills/tree/main/skills/frontend-design')
    expect(sourceLink.attributes('target')).toBe('_blank')
    expect(sourceLink.attributes('rel')).toContain('noopener')
    expect(sourceLink.text()).toContain('anthropics/skills')
    expect(sourceLink.text()).toContain('★ 166.1k')
    expect(sourceLink.text()).not.toContain('1 次下载')
    expect(wrapper.findAll('.skill-detail-info')).toHaveLength(1)
  })

  it('uses the current published version for installation and SKILL.md', async () => {
    const current = version()
    const previous: PublicSkillVersion = {
      ...version(),
      version: '0.9.0',
      sha256: 'c'.repeat(64),
      skill_md: '# Previous guidance',
      created_at: '2026-07-01T00:00:00Z',
    }
    api.getSkill.mockResolvedValue(skill({
      current_version: { ...current, skill_md: '' },
      versions: [current, previous],
    }))
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('select').exists()).toBe(false)
    expect(wrapper.get('.skill-install-stub').attributes('data-version')).toBe('1.0.0')
    expect(wrapper.get('.skill-install-stub').attributes('data-sha')).toBe('a'.repeat(64))
    expect(wrapper.get('.skill-detail-source__body').text()).toContain('# Frontend Design')
    expect(wrapper.get('.skill-detail-source__body').text()).not.toContain('Previous guidance')
  })

  it.each([
    { stars: null, visible: false, label: 'null' },
    { stars: 0, visible: true, label: 'zero' },
  ])('renders a repository star count correctly when it is $label', async ({ stars, visible }) => {
    api.getSkill.mockResolvedValue(skill({
      source_url: 'https://github.com/anthropics/skills/tree/main/skills/frontend-design',
      source_repository: 'anthropics/skills',
      repository_stars: stars,
    }))
    const wrapper = mountView()
    await flushPromises()

    const starCount = wrapper.find('.skill-detail-info__source small')
    expect(starCount.exists()).toBe(visible)
    if (visible) expect(starCount.text()).toBe('★ 0')
  })

  it('copies the complete SKILL.md while keeping the reading view concise', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.get('.skill-detail-source__body').text()).toContain('# Frontend Design')
    expect(wrapper.get('.skill-detail-source__body').text()).not.toContain('name: frontend-design')
    await wrapper.get('.skill-detail-source__heading button').trigger('click')
    expect(clipboardState.copy).toHaveBeenCalledWith(
      '---\nname: frontend-design\n---\n# Frontend Design\n\nOriginal guidance.',
      'skills.detail.fullCopySuccess',
    )
  })

  it('does not infer source provenance from a well-known slug', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('.skill-detail-info__source').exists()).toBe(false)
    expect(wrapper.get('.skill-detail-info__category').text()).toContain('design-ui')
  })

  it('does not render a non-GitHub HTTPS URL as GitHub provenance', async () => {
    api.getSkill.mockResolvedValue(skill({
      source_url: 'https://example.com/anthropics/skills',
      source_repository: 'anthropics/skills',
      repository_stars: 12,
    }))
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('.skill-detail-info__source').exists()).toBe(false)
  })
})

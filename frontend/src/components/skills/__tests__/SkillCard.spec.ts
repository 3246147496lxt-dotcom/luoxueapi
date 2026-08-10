import { defineComponent } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import SkillCard from '../SkillCard.vue'
import type { PublicSkill, PublicSkillVersion } from '@/api/skills'

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      locale: { value: 'zh-CN' },
      t: (key: string, params?: Record<string, string | number>) => {
        const messages: Record<string, string> = {
          'skills.labels.featured': '官方精选',
          'skills.labels.uncategorized': '其他',
          'skills.card.noSummary': '管理员暂未提供简介。',
          'skills.card.tags': 'Skill 标签',
          'skills.card.downloads': '{count} 次下载',
        }
        return Object.entries(params ?? {}).reduce(
          (message, [name, value]) => message.replace(`{${name}}`, String(value)),
          messages[key] ?? key,
        )
      },
    }),
  }
})

const RouterLinkStub = defineComponent({
  inheritAttrs: false,
  props: {
    to: {
      type: [String, Object],
      required: true,
    },
  },
  template: '<a v-bind="$attrs" :data-to="typeof to === \'string\' ? to : JSON.stringify(to)"><slot /></a>',
})

const IconStub = defineComponent({
  props: {
    name: {
      type: String,
      required: true,
    },
  },
  template: '<span class="icon-stub" :data-icon-name="name"></span>',
})

function version(): PublicSkillVersion {
  return {
    version: '1.0.0',
    changelog: '',
    manifest_name: 'frontend-design',
    manifest_description: '',
    skill_md: '',
    sha256: 'a'.repeat(64),
    byte_size: 128,
    unpacked_size: 256,
    file_count: 1,
    file_manifest: [],
    validation_report: { valid: true, errors: [], warnings: [] },
    created_at: '2026-08-05T00:00:00Z',
    download_count: 12_345,
  }
}

function skill(): PublicSkill {
  const currentVersion = version()
  return {
    slug: 'frontend-design',
    display_name: 'Frontend Design 前端设计',
    summary: '这段摘要不应出现在精简卡片中。',
    description: '',
    category: 'design-ui',
    tags: ['anthropic', 'codex', 'html/css', 'accessibility', 'ui/ux'],
    icon: '/skill-icon.png',
    example_prompts: [],
    risk_notes: '',
    featured: true,
    current_version: currentVersion,
    download_count: 12_345,
    published_at: '2026-08-05T00:00:00Z',
    updated_at: '2026-08-05T00:00:00Z',
    versions: [currentVersion],
  }
}

describe('SkillCard', () => {
  it('keeps the whole compact card linked while showing only two useful tags', () => {
    const wrapper = mount(SkillCard, {
      props: {
        skill: skill(),
        categoryName: '设计与 UI',
      },
      global: {
        stubs: {
          RouterLink: RouterLinkStub,
          Icon: IconStub,
        },
      },
    })

    const card = wrapper.get('article.skill-card')
    const link = card.get('a.skill-card__link')
    expect(card.findAll('a')).toHaveLength(1)
    expect(link.attributes('data-to')).toBe('/skills/frontend-design')
    expect(link.element.parentElement).toBe(card.element)

    const category = link.get('.skill-card__category')
    const title = link.get('h3')
    expect(category.text()).toBe('设计与 UI')
    expect(title.text()).toBe('Frontend Design 前端设计')
    expect(
      category.element.compareDocumentPosition(title.element) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()

    expect(link.get('.skill-card__tags').findAll('li').map((tag) => tag.text())).toEqual([
      'html/css',
      'ui/ux',
    ])
    expect(link.text()).not.toContain('anthropic')
    expect(link.text()).not.toContain('codex')
    expect(link.text()).not.toContain('accessibility')
    expect(link.text()).not.toContain('+3')

    expect(link.find('img').exists()).toBe(false)
    expect(link.find('.skill-card__mark').exists()).toBe(false)
    expect(link.text()).not.toContain('这段摘要不应出现在精简卡片中。')
    expect(link.text()).not.toContain('v1.0.0')
    expect(link.text()).not.toContain('8月5日')
    expect(link.text()).not.toContain('1.2万 次下载')
    expect(link.text()).not.toContain('查看详情')
    expect(link.find('[data-icon-name="arrowRight"]').exists()).toBe(false)
    expect(link.find('[data-icon-name="chevronRight"]').exists()).toBe(false)
  })
})

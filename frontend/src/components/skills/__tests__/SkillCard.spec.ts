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
          'skills.card.view': '查看',
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
    display_name: '前端界面设计',
    summary: '创建具有鲜明辨识度与高设计品质的生产级前端界面。',
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
  it('keeps one whole-card link with the summary, category at bottom-left, and View at bottom-right', () => {
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

    const title = link.get('h3')
    const summary = link.get('.skill-card__body p')
    const footer = link.get('.skill-card__footer')
    const category = footer.get('.skill-card__category')
    const action = footer.get('.skill-card__action')
    expect(title.text()).toBe('前端界面设计')
    expect(summary.text()).toBe('创建具有鲜明辨识度与高设计品质的生产级前端界面。')
    expect(category.text()).toBe('设计与 UI')
    expect(action.text()).toBe('查看')
    expect(action.attributes('aria-hidden')).toBe('true')
    expect(
      title.element.compareDocumentPosition(summary.element) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
    expect(
      category.element.compareDocumentPosition(action.element) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()

    expect(link.find('.skill-card__tags').exists()).toBe(false)
    expect(link.text()).not.toContain('anthropic')
    expect(link.text()).not.toContain('codex')
    expect(link.text()).not.toContain('accessibility')
    expect(link.text()).not.toContain('+3')

    expect(link.find('img').exists()).toBe(false)
    expect(link.find('.skill-card__mark').exists()).toBe(false)
    expect(link.text()).not.toContain('v1.0.0')
    expect(link.text()).not.toContain('8月5日')
    expect(link.text()).not.toContain('1.2万 次下载')
    expect(link.find('[data-icon-name="arrowRight"]').exists()).toBe(false)
    expect(link.find('[data-icon-name="chevronRight"]').exists()).toBe(false)
  })

  it('uses the fallback summary when the catalog item has no summary', () => {
    const item = skill()
    item.summary = ''
    const wrapper = mount(SkillCard, {
      props: { skill: item },
      global: { stubs: { RouterLink: RouterLinkStub } },
    })

    expect(wrapper.get('.skill-card__body p').text()).toBe('管理员暂未提供简介。')
    expect(wrapper.get('.skill-card__category').text()).toBe('design-ui')
  })
})

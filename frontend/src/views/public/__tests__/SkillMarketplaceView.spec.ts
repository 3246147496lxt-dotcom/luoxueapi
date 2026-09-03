import { defineComponent, nextTick, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import SkillMarketplaceView from '../SkillMarketplaceView.vue'
import type { PublicSkill } from '@/api/skills'

const testState = vi.hoisted(() => ({
  listPublicSkills: vi.fn(),
}))

const messages: Record<string, string> = {
  'skills.market.title': 'Skill 市场',
  'skills.market.description': 'Skill 是 Agent 的可复用能力',
  'skills.market.countBadge': '{count} 个 Skill',
  'skills.market.searchLabel': '搜索 Skill',
  'skills.market.searchPlaceholder': '搜索 skill',
  'skills.market.trustLabel': '市场说明',
  'skills.market.officialOnly': '仅管理员发布',
  'skills.market.localCodex': '安装到本地 Codex',
  'skills.market.versioned': '指定版本可核验',
  'skills.market.preview.ariaLabel': 'Skill 使用流程预览',
  'skills.market.preview.find': '找到适合任务的 Skill',
  'skills.market.preview.copy': '复制指令给 Codex',
  'skills.market.preview.verify': '验证本地安装结果',
  'skills.market.categoriesTitle': '按任务缩小范围',
  'skills.market.resultCount': '共 {count} 个 Skill',
  'skills.market.resultsTitle': '搜索结果',
  'skills.market.resultsDescription': '以下结果来自当前关键词与任务分类。',
  'skills.market.allTitle': '全部 Skill',
  'skills.market.allDescription': '选择一个 Skill，查看能力与安装方式。',
  'skills.market.latestTitle': '最新上架',
  'skills.market.loading': '正在加载 Skill',
  'skills.market.unavailableTitle': 'Skill 市场暂未开放',
  'skills.market.unavailableDescription': '当前站点还没有启用官方 Skill 市场。',
  'skills.market.errorTitle': 'Skill 市场暂时无法加载',
  'skills.market.errorDescription': '请稍后重试。',
  'skills.market.retry': '重新加载',
  'skills.market.noResultsTitle': '没有匹配的 Skill',
  'skills.market.noResultsDescription': '换一种任务说法。',
  'skills.market.emptyTitle': '官方 Skill 正在准备中',
  'skills.market.emptyDescription': '发布后会显示在这里。',
  'skills.market.clearFilters': '清除筛选',
  'skills.market.paginationLabel': 'Skill 目录分页',
  'skills.market.previous': '上一页',
  'skills.market.next': '下一页',
  'skills.market.pageStatus': '第 {page} / {pages} 页',
  'skills.market.featuredTitle': '值得先看的官方精选',
  'skills.categories.all': '全部任务',
  'skills.categories.design-ui': '设计与 UI',
}
const testLocale = ref('zh-CN')

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      locale: testLocale,
      te: (key: string) => key in messages,
      t: (key: string, params?: Record<string, string | number>) => Object.entries(params ?? {}).reduce(
        (message, [name, value]) => message.replace(`{${name}}`, String(value)),
        messages[key] ?? key,
      ),
    }),
  }
})

vi.mock('@/api/skills', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/skills')>()
  return {
    ...actual,
    listPublicSkills: (...args: unknown[]) => testState.listPublicSkills(...args),
  }
})

const AppLayoutStub = defineComponent({
  template: '<div class="app-layout-stub"><slot /></div>',
})

const SkillCardStub = defineComponent({
  props: {
    skill: {
      type: Object,
      required: true,
    },
    categoryName: {
      type: String,
      default: '',
    },
  },
  template: '<article class="skill-card-stub" :data-category="categoryName"><h3>{{ skill.display_name }}</h3><p>{{ skill.summary }}</p></article>',
})

function skill(): PublicSkill {
  return {
    slug: 'frontend-design',
    display_name: '前端界面设计',
    summary: '创建具有鲜明辨识度与高设计品质的生产级前端界面。',
    description: '',
    category: 'design-ui',
    tags: ['html/css', 'ui/ux'],
    icon: '',
    example_prompts: [],
    risk_notes: '',
    featured: true,
    current_version: null,
    download_count: 0,
    published_at: '',
    updated_at: '',
    versions: [],
  }
}

beforeEach(() => {
  testLocale.value = 'zh-CN'
  testState.listPublicSkills.mockReset().mockResolvedValue({
    items: [skill()],
    total: 1,
    page: 1,
    page_size: 12,
    categories: [{ slug: 'design-ui', label: '设计与 UI', count: 1 }],
  })
})

describe('SkillMarketplaceView', () => {
  it('renders the approved embedded marketplace with a compact header and card grid', async () => {
    const wrapper = mount(SkillMarketplaceView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          SkillCard: SkillCardStub,
          Icon: true,
        },
      },
    })
    await flushPromises()

    expect(wrapper.find('.app-layout-stub').exists()).toBe(true)
    expect(wrapper.get('#skill-market-title').text()).toBe('Skill 市场')
    expect(wrapper.get('.skill-market-hero__copy > p').text()).toBe('Skill 是 Agent 的可复用能力')
    expect(wrapper.get('.skill-market-count-badge').text()).toBe('1 个 Skill')
    expect(wrapper.get('#skill-market-search-input').attributes('placeholder')).toBe('搜索 skill')
    expect(wrapper.find('.skill-market-search button').exists()).toBe(false)
    expect(wrapper.find('.skill-market-trust').exists()).toBe(false)
    expect(wrapper.find('.skill-market-proof').exists()).toBe(false)
    expect(wrapper.find('.skill-market-proof__route').exists()).toBe(false)
    expect(wrapper.find('.skill-market-featured').exists()).toBe(false)
    expect(wrapper.get('#skill-results-title').text()).toBe('全部 Skill')
    expect(wrapper.get('.skill-market-results-heading > p').text()).toBe('选择一个 Skill，查看能力与安装方式。')

    expect(wrapper.text()).not.toContain('找到适合任务的 Skill')
    expect(wrapper.text()).not.toContain('复制指令给 Codex')
    expect(wrapper.text()).not.toContain('验证本地安装结果')
    expect(wrapper.text()).not.toContain('值得先看的官方精选')

    expect(wrapper.get('.skill-card-stub h3').text()).toBe('前端界面设计')
    expect(wrapper.get('.skill-card-stub p').text()).toBe('创建具有鲜明辨识度与高设计品质的生产级前端界面。')
    expect(wrapper.get('.skill-card-stub').attributes('data-category')).toBe('设计与 UI')
    expect(testState.listPublicSkills).toHaveBeenCalledOnce()
    expect(testState.listPublicSkills).toHaveBeenCalledWith(
      {
        search: '',
        category: '',
        page: 1,
        page_size: 12,
      },
      { signal: expect.any(AbortSignal) },
    )

    wrapper.unmount()
  })

  it('reloads localized catalog metadata when the interface language changes', async () => {
    const wrapper = mount(SkillMarketplaceView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          SkillCard: SkillCardStub,
          Icon: true,
        },
      },
    })
    await flushPromises()
    expect(testState.listPublicSkills).toHaveBeenCalledOnce()

    testLocale.value = 'en'
    await nextTick()
    await flushPromises()

    expect(testState.listPublicSkills).toHaveBeenCalledTimes(2)
    wrapper.unmount()
  })
})

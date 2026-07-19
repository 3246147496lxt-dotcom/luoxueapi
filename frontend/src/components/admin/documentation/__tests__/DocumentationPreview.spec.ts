import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import DocumentationPreview from '../DocumentationPreview.vue'
import DocumentationCategoryIcon from '../DocumentationCategoryIcon.vue'
import type { DocumentationContent } from '@/api/admin/documentation'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

const content: DocumentationContent = {
  schema_version: 1,
  tutorials: [
    {
      id: 'quick-start',
      tab_label: '新手教程',
      icon: 'key',
      description: '快速开始',
      steps: [
        {
          title: '创建密钥',
          description: '打开密钥页面。',
          image: { src: '', alt: '空图片不应渲染' },
        },
      ],
    },
    {
      id: 'clients',
      tab_label: '客户端配置',
      icon: 'terminal',
      icon_svg: '<svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="8"/></svg>',
      description: '配置客户端',
      steps: [
        {
          title: '填写配置',
          description: '使用站内生成的配置。',
          image: { src: 'https://example.com/tutorial.png', alt: '客户端配置截图' },
        },
      ],
    },
  ],
}

describe('DocumentationPreview', () => {
  it('maps icon keys, omits empty image URLs, and supports arrow-key tab navigation', async () => {
    const wrapper = mount(DocumentationPreview, { props: { content } })
    const tabs = wrapper.findAll('[role="tab"]')

    expect(tabs[0].attributes('aria-selected')).toBe('true')
    expect(tabs[0].findComponent(DocumentationCategoryIcon).props('icon')).toBe('key')
    expect(tabs[0].findComponent(DocumentationCategoryIcon).props('iconSvg')).toBe('')
    expect(wrapper.find('img').exists()).toBe(false)

    await tabs[0].trigger('keydown', { key: 'ArrowRight' })

    expect(wrapper.findAll('[role="tab"]')[1].attributes('aria-selected')).toBe('true')
    expect(wrapper.findAll('[role="tab"]')[1].findComponent(DocumentationCategoryIcon).props('icon')).toBe('terminal')
    expect(wrapper.findAll('[role="tab"]')[1].findComponent(DocumentationCategoryIcon).props('iconSvg')).toContain('<svg')
    expect(wrapper.get('img').attributes('src')).toBe('https://example.com/tutorial.png')
  })
})

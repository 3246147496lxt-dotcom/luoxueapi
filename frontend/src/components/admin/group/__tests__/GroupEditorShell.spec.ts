import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'

import GroupEditorShell from '../GroupEditorShell.vue'

const baseProps = {
  title: '编辑分组',
  description: '按业务区域完成配置',
  contextLabel: 'Anthropic',
  backLabel: '返回分组',
  navigationLabel: '分组配置区域',
  sections: [
    { id: 'basic', label: '基础归属', description: '名称、平台与账号来源' },
    { id: 'quota', label: '访问配额', description: '订阅与限额' }
  ],
  activeSection: 'basic',
  loadingLabel: '正在加载分组配置',
  dirtyLabel: '有未保存更改',
  savedLabel: '所有更改已保存'
}

describe('GroupEditorShell', () => {
  it('exposes task navigation, context and a sticky save region', async () => {
    const wrapper = mount(GroupEditorShell, {
      props: baseProps,
      slots: {
        default: '<div data-test="editor-content">真实表单</div>',
        footer: '<button type="submit">保存</button>'
      },
      global: {
        stubs: { Icon: true }
      }
    })

    expect(wrapper.get('h1').text()).toBe('编辑分组')
    expect(wrapper.get('nav').attributes('aria-label')).toBe('分组配置区域')
    expect(wrapper.get('[aria-current="step"]').text()).toContain('基础归属')
    expect(wrapper.get('#group-editor-active-title').text()).toBe('基础归属')
    expect(wrapper.get('[data-test="editor-content"]').text()).toBe('真实表单')
    expect(wrapper.find('.group-editor-footer').exists()).toBe(true)
    expect(wrapper.text()).toContain('所有更改已保存')

    await wrapper.findAll('.group-editor-nav-item')[1].trigger('click')
    expect(wrapper.emitted('select-section')).toEqual([['quota']])

    await wrapper.get('.group-editor-back').trigger('click')
    expect(wrapper.emitted('back')).toHaveLength(1)
  })

  it('announces dirty and loading states without rendering stale form content', () => {
    const wrapper = mount(GroupEditorShell, {
      props: {
        ...baseProps,
        dirty: true,
        loading: true
      },
      slots: { default: '<div data-test="editor-content">stale</div>' },
      global: {
        stubs: { Icon: true }
      }
    })

    expect(wrapper.text()).toContain('有未保存更改')
    expect(wrapper.find('.group-editor-save-dot--dirty').exists()).toBe(true)
    expect(wrapper.get('.group-editor-content').attributes('aria-busy')).toBe('true')
    expect(wrapper.find('[data-test="editor-content"]').exists()).toBe(false)
    expect(wrapper.find('.group-editor-loading').exists()).toBe(true)
    expect(wrapper.get('[role="status"]').text()).toContain('正在加载分组配置')
  })

  it('uses real links when section routes are provided', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/admin/groups/2/edit', component: { template: '<div />' } }]
    })
    await router.push('/admin/groups/2/edit?section=basic&source=list')
    await router.isReady()

    const wrapper = mount(GroupEditorShell, {
      props: {
        ...baseProps,
        sections: baseProps.sections.map(section => ({
          ...section,
          to: {
            path: '/admin/groups/2/edit',
            query: { section: section.id, source: 'list' }
          }
        }))
      },
      global: {
        plugins: [router],
        stubs: { Icon: true }
      }
    })

    const links = wrapper.findAll('a.group-editor-nav-item')
    expect(links).toHaveLength(2)
    expect(links[0].attributes('aria-current')).toBe('page')
    expect(links[1].attributes('href')).toContain('section=quota')
    expect(links[1].attributes('href')).toContain('source=list')
    expect(wrapper.emitted('select-section')).toBeUndefined()
  })
})

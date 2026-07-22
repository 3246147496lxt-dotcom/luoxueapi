import { defineComponent, nextTick, ref } from 'vue'
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import OpsWorkspaceNav from '../OpsWorkspaceNav.vue'

const items = [
  { id: 'live', label: '实时流量' },
  { id: 'quality', label: '质量与异常' },
  { id: 'alerts', label: '告警事件' },
  { id: 'logs', label: '系统日志' }
]

function mountControlled(initialValue = 'live') {
  const Harness = defineComponent({
    components: { OpsWorkspaceNav },
    setup() {
      const selected = ref(initialValue)
      return { items, selected }
    },
    template: `
      <OpsWorkspaceNav
        v-model="selected"
        :items="items"
        label="监控工作区"
      />
    `
  })

  return mount(Harness, { attachTo: document.body })
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('OpsWorkspaceNav', () => {
  it('提供命名清晰的页内标签导航与 roving tabindex', () => {
    const wrapper = mountControlled('quality')
    const nav = wrapper.get('nav')
    const tablist = wrapper.get('[role="tablist"]')
    const tabs = wrapper.findAll<HTMLButtonElement>('[role="tab"]')

    expect(nav.attributes('aria-label')).toBe('监控工作区')
    expect(tablist.attributes()).toMatchObject({
      'aria-label': '监控工作区',
      'aria-orientation': 'horizontal'
    })
    expect(tabs).toHaveLength(4)
    expect(tabs[1].attributes()).toMatchObject({
      id: 'ops-workspace-tab-quality',
      'aria-controls': 'ops-workspace-panel-quality',
      'aria-selected': 'true',
      tabindex: '0'
    })
    expect(tabs[0].attributes('aria-selected')).toBe('false')
    expect(tabs[0].attributes('tabindex')).toBe('-1')
    expect(tabs[2].attributes('tabindex')).toBe('-1')
    expect(tabs[3].attributes('tabindex')).toBe('-1')

    wrapper.unmount()
  })

  it('鼠标点击会切换选中标签', async () => {
    const wrapper = mountControlled()
    const tabs = wrapper.findAll('[role="tab"]')

    await tabs[3].trigger('click')

    expect(wrapper.findAll('[role="tab"]')[3].attributes('aria-selected')).toBe('true')
    expect(wrapper.findAll('[role="tab"]')[3].attributes('tabindex')).toBe('0')

    wrapper.unmount()
  })

  it('方向键循环切换并聚焦，Home 与 End 跳转到边界标签', async () => {
    const wrapper = mountControlled()
    let tabs = wrapper.findAll<HTMLButtonElement>('[role="tab"]')

    tabs[0].element.focus()
    await tabs[0].trigger('keydown', { key: 'ArrowRight' })
    await nextTick()
    tabs = wrapper.findAll<HTMLButtonElement>('[role="tab"]')
    expect(tabs[1].attributes('aria-selected')).toBe('true')
    expect(document.activeElement).toBe(tabs[1].element)

    await tabs[1].trigger('keydown', { key: 'End' })
    await nextTick()
    tabs = wrapper.findAll<HTMLButtonElement>('[role="tab"]')
    expect(tabs[3].attributes('aria-selected')).toBe('true')
    expect(document.activeElement).toBe(tabs[3].element)

    await tabs[3].trigger('keydown', { key: 'ArrowRight' })
    await nextTick()
    tabs = wrapper.findAll<HTMLButtonElement>('[role="tab"]')
    expect(tabs[0].attributes('aria-selected')).toBe('true')
    expect(document.activeElement).toBe(tabs[0].element)

    await tabs[0].trigger('keydown', { key: 'ArrowLeft' })
    await nextTick()
    tabs = wrapper.findAll<HTMLButtonElement>('[role="tab"]')
    expect(tabs[3].attributes('aria-selected')).toBe('true')
    expect(document.activeElement).toBe(tabs[3].element)

    await tabs[3].trigger('keydown', { key: 'Home' })
    await nextTick()
    tabs = wrapper.findAll<HTMLButtonElement>('[role="tab"]')
    expect(tabs[0].attributes('aria-selected')).toBe('true')
    expect(document.activeElement).toBe(tabs[0].element)

    wrapper.unmount()
  })

  it('modelValue 无匹配项时仍为第一项保留唯一的活动焦点入口', () => {
    const wrapper = mountControlled('missing')
    const tabs = wrapper.findAll('[role="tab"]')

    expect(tabs[0].attributes('aria-selected')).toBe('true')
    expect(tabs.map((tab) => tab.attributes('tabindex'))).toEqual(['0', '-1', '-1', '-1'])

    wrapper.unmount()
  })
})

import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Icon from '../Icon.vue'

describe('ChatGPT-like sidebar icons', () => {
  it.each([
    'chatSidebarCompose',
    'chatSidebarLibrary',
    'chatSidebarProjects',
    'chatSidebarScheduled',
    'chatSidebarPlugins',
  ] as const)('renders %s on the measured 20px coordinate system', (name) => {
    const wrapper = mount(Icon, { props: { name, size: 'md' } })
    const svg = wrapper.get('svg')

    expect(svg.attributes('viewBox')).toBe('0 0 20 20')
    expect(svg.classes()).toEqual(expect.arrayContaining(['h-5', 'w-5']))
  })

  it('keeps the compose and projects glyphs filled', () => {
    const compose = mount(Icon, { props: { name: 'chatSidebarCompose' } })
    const projects = mount(Icon, { props: { name: 'chatSidebarProjects' } })

    expect(compose.get('g').attributes()).toMatchObject({
      fill: 'currentColor',
      stroke: 'none',
    })
    expect(compose.findAll('path')).toHaveLength(2)
    expect(compose.findAll('path')[0]!.attributes('d')).toContain('M8.167 2.501')
    expect(projects.get('g').attributes()).toMatchObject({
      fill: 'currentColor',
      stroke: 'none',
    })
    expect(projects.get('path').attributes('d')).toContain('M6.95 2.668')
  })

  it('keeps the library, scheduled, and plugins glyphs on the official 1.33 stroke', () => {
    const library = mount(Icon, { props: { name: 'chatSidebarLibrary' } })
    const scheduled = mount(Icon, { props: { name: 'chatSidebarScheduled' } })
    const plugins = mount(Icon, { props: { name: 'chatSidebarPlugins' } })

    expect(library.get('path').attributes('stroke-width')).toBe('1.33')
    expect(library.get('path').attributes('d')).toContain('M6.804 15.133')
    expect(scheduled.get('g').attributes('stroke-width')).toBe('1.33')
    expect(scheduled.get('circle').attributes()).toMatchObject({
      cx: '10',
      cy: '10',
      r: '7.5',
    })
    expect(plugins.get('g').attributes()).toMatchObject({
      'stroke-linecap': 'round',
      'stroke-width': '1.33',
    })
    expect(plugins.findAll('path')).toHaveLength(2)
    expect(plugins.findAll('path')[0]!.attributes('d')).toContain('M12.524 12.192')
  })
})

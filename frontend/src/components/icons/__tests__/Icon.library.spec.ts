import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Icon from '../Icon.vue'

const officialLibraryGlyphs = [
  ['librarySearch', 'M9.162 2.38'],
  ['libraryFilter', 'M12.5 14.335'],
  ['libraryGrid', 'M7.5 11.002'],
  ['libraryList', 'M4.167 13.729'],
] as const

describe('official library icons', () => {
  it.each(officialLibraryGlyphs)('renders %s as a filled 20px glyph', (name, pathStart) => {
    const wrapper = mount(Icon, { props: { name, size: 'md' } })
    const svg = wrapper.get('svg')
    const path = wrapper.get('path')

    expect(svg.attributes('viewBox')).toBe('0 0 20 20')
    expect(svg.classes()).toEqual(expect.arrayContaining(['h-5', 'w-5']))
    expect(path.attributes()).toMatchObject({
      fill: 'currentColor',
      stroke: 'none',
    })
    expect(path.attributes('d')).toContain(pathStart)
  })
})

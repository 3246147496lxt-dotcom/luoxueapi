import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import snowflakeCreditUrl from '@/assets/icons/snowflake-credit.svg'
import SnowflakeCreditIcon from '../SnowflakeCreditIcon.vue'

describe('SnowflakeCreditIcon', () => {
  it('renders the selected snowflake asset as a decorative 20px icon by default', () => {
    const wrapper = mount(SnowflakeCreditIcon)
    const image = wrapper.get('img')

    expect(image.attributes('src')).toBe(snowflakeCreditUrl)
    expect(image.attributes('alt')).toBe('')
    expect(image.attributes('aria-hidden')).toBe('true')
    expect(image.attributes('draggable')).toBe('false')
    expect(image.classes()).toEqual(expect.arrayContaining(['h-5', 'w-5']))
  })

  it('supports compact sizes and an explicit accessible label', () => {
    const wrapper = mount(SnowflakeCreditIcon, {
      props: { size: 'sm', label: 'Snow credits' },
    })
    const image = wrapper.get('img')

    expect(image.classes()).toEqual(expect.arrayContaining(['h-4', 'w-4']))
    expect(image.attributes('alt')).toBe('Snow credits')
    expect(image.attributes('aria-hidden')).toBeUndefined()
  })

  it('keeps the bundled SVG self-contained and free from external document metadata', () => {
    const assetPath = resolve(process.cwd(), 'src/assets/icons/snowflake-credit.svg')
    const svg = readFileSync(assetPath, 'utf8')

    expect(svg).toContain('viewBox="0 0 1024 1024"')
    expect(svg).toContain('fill="#09AFFF"')
    expect(svg).not.toMatch(/<!DOCTYPE|<script|<foreignObject|xlink:href|\shref=/i)
    expect(svg.match(/<path\b/g)).toHaveLength(1)
  })
})

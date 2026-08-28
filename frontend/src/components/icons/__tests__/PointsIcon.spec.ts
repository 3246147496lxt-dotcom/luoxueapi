import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import pointsUrl from '@/assets/icons/points.svg'
import PointsIcon from '../PointsIcon.vue'

describe('PointsIcon', () => {
  it('renders the points artwork with a decorative accessible default', () => {
    const wrapper = mount(PointsIcon)
    const image = wrapper.get('img')

    expect(image.attributes('src')).toBe(pointsUrl)
    expect(image.attributes('alt')).toBe('')
    expect(image.attributes('aria-hidden')).toBe('true')
    expect(image.attributes('data-testid')).toBe('points-icon')
    expect(image.attributes('data-icon')).toBe('points')
    expect(image.classes()).toEqual(expect.arrayContaining(['h-5', 'w-5']))
  })

  it('supports compact sizes and explicit labels', () => {
    const wrapper = mount(PointsIcon, {
      props: { size: 'sm', label: 'Points' },
    })
    const image = wrapper.get('img')

    expect(image.classes()).toEqual(expect.arrayContaining(['h-4', 'w-4']))
    expect(image.attributes('alt')).toBe('Points')
    expect(image.attributes('aria-hidden')).toBeUndefined()
  })
})

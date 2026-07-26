import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h, ref, toRef } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useAnchoredOverlay } from '../useAnchoredOverlay'

const originalInnerWidth = window.innerWidth
const originalInnerHeight = window.innerHeight

function rect(
  left: number,
  top: number,
  width: number,
  height: number,
): DOMRect {
  return {
    x: left,
    y: top,
    left,
    top,
    width,
    height,
    right: left + width,
    bottom: top + height,
    toJSON: () => ({}),
  }
}

const Harness = defineComponent({
  props: {
    open: {
      type: Boolean,
      required: true,
    },
  },
  setup(props) {
    const anchorElement = ref<HTMLElement | null>(null)
    const panelElement = ref<HTMLElement | null>(null)
    const { desktopStyle } = useAnchoredOverlay(
      toRef(props, 'open'),
      anchorElement,
      panelElement,
    )

    return () => h('div', [
      h('button', {
        ref: anchorElement,
        'data-testid': 'anchor',
      }),
      h('section', {
        ref: panelElement,
        'data-testid': 'panel',
        style: desktopStyle.value,
      }),
    ])
  },
})

afterEach(() => {
  Object.defineProperty(window, 'innerWidth', {
    configurable: true,
    value: originalInnerWidth,
  })
  Object.defineProperty(window, 'innerHeight', {
    configurable: true,
    value: originalInnerHeight,
  })
  vi.restoreAllMocks()
})

describe('useAnchoredOverlay', () => {
  it('pins the panel immediately above and left-aligned with its dock anchor', async () => {
    Object.defineProperty(window, 'innerWidth', {
      configurable: true,
      value: 1440,
    })
    Object.defineProperty(window, 'innerHeight', {
      configurable: true,
      value: 964,
    })

    const wrapper = mount(Harness, { props: { open: false } })
    const anchor = wrapper.get<HTMLElement>('[data-testid="anchor"]')
    const panel = wrapper.get<HTMLElement>('[data-testid="panel"]')
    vi.spyOn(anchor.element, 'getBoundingClientRect').mockReturnValue(
      rect(8, 900, 246, 52),
    )
    vi.spyOn(panel.element, 'getBoundingClientRect').mockReturnValue(
      rect(0, 0, 248, 360),
    )

    await wrapper.setProps({ open: true })
    await flushPromises()

    expect(panel.element.style.left).toBe('8px')
    expect(panel.element.style.bottom).toBe('72px')
    expect(panel.element.style.maxHeight).toBe('884px')

    wrapper.unmount()
  })

  it('clamps horizontally, recomputes on viewport changes, and removes listeners', async () => {
    Object.defineProperty(window, 'innerWidth', {
      configurable: true,
      value: 1440,
    })
    Object.defineProperty(window, 'innerHeight', {
      configurable: true,
      value: 964,
    })
    const removeEventListener = vi.spyOn(window, 'removeEventListener')
    const wrapper = mount(Harness, { props: { open: false } })
    const anchor = wrapper.get<HTMLElement>('[data-testid="anchor"]')
    const panel = wrapper.get<HTMLElement>('[data-testid="panel"]')
    const anchorRect = vi.spyOn(anchor.element, 'getBoundingClientRect')
      .mockReturnValue(rect(8, 900, 246, 52))
    vi.spyOn(panel.element, 'getBoundingClientRect').mockReturnValue(
      rect(0, 0, 248, 360),
    )

    await wrapper.setProps({ open: true })
    await flushPromises()

    anchorRect.mockReturnValue(rect(1400, 700, 246, 52))
    window.dispatchEvent(new Event('resize'))
    await flushPromises()

    expect(panel.element.style.left).toBe('1184px')
    expect(panel.element.style.bottom).toBe('272px')
    expect(panel.element.style.maxHeight).toBe('684px')

    wrapper.unmount()
    expect(removeEventListener).toHaveBeenCalledWith('resize', expect.any(Function))
    expect(removeEventListener).toHaveBeenCalledWith('scroll', expect.any(Function), true)
  })
})

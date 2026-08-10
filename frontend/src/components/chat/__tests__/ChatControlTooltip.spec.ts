import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { h, nextTick } from 'vue'
import { mount, type VueWrapper } from '@vue/test-utils'
import ChatControlTooltip from '../ChatControlTooltip.vue'

const COMPONENT_SOURCE = readFileSync(
  resolve(process.cwd(), 'src/components/chat/ChatControlTooltip.vue'),
  'utf8',
)
const wrappers: VueWrapper[] = []

function portalTooltip(): HTMLElement {
  const tooltip = document.body.querySelector<HTMLElement>(
    '[data-ui-portal="chat-control-tooltip"][role="tooltip"]',
  )
  if (!tooltip) throw new Error('Expected the tooltip to be portaled to document.body')
  return tooltip
}

beforeEach(() => {
  vi.spyOn(window, 'matchMedia').mockImplementation((query: string) => ({
    matches: query === '(hover: hover) and (pointer: fine)',
    media: query,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
})

afterEach(() => {
  wrappers.splice(0).forEach((wrapper) => wrapper.unmount())
  document.body.innerHTML = ''
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

function mountTooltip(props: Record<string, unknown> = {}) {
  const wrapper = mount(ChatControlTooltip, {
    attachTo: document.body,
    props: {
      label: '思考强度',
      shortcut: ['⌃', '⇧', 'M'],
      ...props,
    },
    slots: {
      default: ({ tooltipId }: { tooltipId: string }) => h('button', {
        'aria-describedby': tooltipId,
        'data-chat-control-anchor': '',
      }, 'Trigger'),
    },
  })
  wrappers.push(wrapper)
  return wrapper
}

describe('ChatControlTooltip', () => {
  it('uses the shared ChatGPT tooltip geometry and typography', () => {
    const wrapper = mountTooltip()
    const trigger = wrapper.get('button')
    const tooltip = portalTooltip()

    expect(wrapper.find('[role="tooltip"]').exists()).toBe(false)
    expect(trigger.attributes('aria-describedby')).toBe(tooltip.id)
    expect(tooltip.textContent).toContain('思考强度')
    expect(tooltip.textContent).toContain('M')
    expect(COMPONENT_SOURCE).toContain('<Teleport to="body">')
    expect(COMPONENT_SOURCE).toContain('position: fixed;')
    expect(COMPONENT_SOURCE).toContain('display: flex;')
    expect(COMPONENT_SOURCE).toContain('padding: 5px 12px;')
    expect(COMPONENT_SOURCE).toContain('border-radius: 999px;')
    expect(COMPONENT_SOURCE).toContain('background: #1b1b1b;')
    expect(COMPONENT_SOURCE).toContain('font-size: 14px;')
    expect(COMPONENT_SOURCE).toContain('font-weight: 600;')
    expect(COMPONENT_SOURCE).toContain('line-height: 18px;')
    expect(COMPONENT_SOURCE).toContain('letter-spacing: -0.15px;')
    expect(COMPONENT_SOURCE).toContain('transition-delay: 250ms, 250ms;')
  })

  it('flips above near the viewport edge and keeps Escape dismissal while focus remains', async () => {
    const wrapper = mountTooltip()
    const host = wrapper.get('.chat-control-tooltip-host')
    const trigger = wrapper.get('button')
    const tooltip = portalTooltip()
    Object.defineProperty(trigger.element, 'getBoundingClientRect', {
      configurable: true,
      value: () => ({
        left: 960,
        right: 996,
        top: 760,
        bottom: 796,
        width: 36,
        height: 36,
        x: 960,
        y: 760,
        toJSON: () => ({}),
      }),
    })
    Object.defineProperty(tooltip, 'getBoundingClientRect', {
      configurable: true,
      value: () => ({
        left: 0,
        right: 140,
        top: 0,
        bottom: 30,
        width: 140,
        height: 30,
        x: 0,
        y: 0,
        toJSON: () => ({}),
      }),
    })
    vi.stubGlobal('innerWidth', 1015)
    vi.stubGlobal('innerHeight', 820)

    await host.trigger('pointerenter')
    await nextTick()
    await nextTick()
    expect(tooltip.classList).toContain('chat-control-tooltip--above')
    expect(tooltip.classList).toContain('chat-control-tooltip--visible')
    expect(tooltip.style.left).toBe('863px')
    expect(tooltip.style.top).toBe('730px')

    ;(trigger.element as HTMLButtonElement).focus()
    await trigger.trigger('keydown', { key: 'Escape' })
    expect(host.classes()).toContain('chat-control-tooltip-host--suppressed')
    expect(tooltip.classList).not.toContain('chat-control-tooltip--visible')
    await host.trigger('pointerleave')
    expect(host.classes()).toContain('chat-control-tooltip-host--suppressed')
  })

  it('keeps the portaled surface hoverable while crossing from the trigger', async () => {
    const wrapper = mountTooltip()
    const host = wrapper.get('.chat-control-tooltip-host')
    const tooltip = portalTooltip()

    await host.trigger('pointerenter')
    await nextTick()
    await host.trigger('pointerleave', { relatedTarget: tooltip })
    tooltip.dispatchEvent(new MouseEvent('pointerenter'))
    await nextTick()

    expect(tooltip.classList).toContain('chat-control-tooltip--visible')
  })

  it('can remain visual-only when the button name already supplies the same text', () => {
    mountTooltip({ accessible: false })
    expect(portalTooltip().getAttribute('aria-hidden')).toBe('true')
  })
})

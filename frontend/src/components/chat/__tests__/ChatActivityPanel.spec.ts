import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import ChatActivityPanel from '../ChatActivityPanel.vue'
import type { ChatActivity, ChatActivityStatus } from '@/types/chat'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

const IconStub = {
  props: ['name'],
  template: '<span :data-icon="name" />',
}

const mountedPanels: Array<{ unmount: () => void }> = []

function activity(
  status: ChatActivityStatus = 'completed',
  overrides: Partial<ChatActivity> = {},
): ChatActivity {
  return {
    key: 'resp-1',
    responseId: 'resp-1',
    status,
    reasoningMode: 'standard',
    reasoningEffort: 'medium',
    startedAt: 1_000,
    updatedAt: 126_000,
    completedAt: status === 'pending' || status === 'streaming' ? undefined : 126_000,
    items: [{
      key: 'item-1',
      itemId: 'rs-1',
      outputIndex: 0,
      status,
      startedAt: 1_000,
      updatedAt: 126_000,
      parts: [
        {
          key: 'part-1',
          itemId: 'rs-1',
          outputIndex: 0,
          summaryIndex: 1,
          text: 'Second **summary**',
          status,
          startedAt: 2_000,
          updatedAt: 126_000,
        },
        {
          key: 'part-0',
          itemId: 'rs-1',
          outputIndex: 0,
          summaryIndex: 0,
          text: 'First summary',
          status,
          startedAt: 1_000,
          updatedAt: 125_000,
        },
      ],
    }],
    ...overrides,
  }
}

function mountPanel(activities: ChatActivity[], mobile = false, attachToDocument = false) {
  const wrapper = mount(ChatActivityPanel, {
    props: { activities, mobile },
    ...(attachToDocument ? { attachTo: document.body } : {}),
    global: { stubs: { Icon: IconStub } },
  })
  mountedPanels.push(wrapper)
  return wrapper
}

afterEach(() => {
  for (const panel of mountedPanels.splice(0).reverse()) panel.unmount()
  document.body.innerHTML = ''
  vi.useRealTimers()
  vi.restoreAllMocks()
})

describe('ChatActivityPanel', () => {
  it('does not render an empty panel without an official non-empty summary', () => {
    const empty = activity('completed')
    empty.items[0]!.parts[0]!.text = ''
    empty.items[0]!.parts[1]!.text = '   '
    const wrapper = mountPanel([empty])

    expect(wrapper.find('[data-test="chat-activity-panel"]').exists()).toBe(false)
  })

  it('renders sanitized summary parts in deterministic summary-index order', () => {
    const unsafe = activity('completed')
    unsafe.items[0]!.parts[0]!.text = 'Second **summary** <script>alert(1)</script>'
    const wrapper = mountPanel([unsafe])
    const parts = wrapper.findAll('[data-test="chat-activity-part"]')

    expect(parts).toHaveLength(2)
    expect(parts[0]!.text()).toContain('First summary')
    expect(parts[1]!.text()).toContain('Second summary')
    expect(parts[1]!.find('strong').text()).toBe('summary')
    expect(wrapper.find('script').exists()).toBe(false)
  })

  it('reuses terminal sanitized Markdown and only rerenders the changed part', async () => {
    const parseSpy = vi.spyOn(marked, 'parse')
    try {
      const initial = activity('completed')
      const wrapper = mountPanel([initial])
      expect(parseSpy).toHaveBeenCalledTimes(2)

      await wrapper.setProps({
        activities: [activity('completed', {
          updatedAt: initial.updatedAt + 1,
          items: initial.items.map((item) => ({ ...item, updatedAt: item.updatedAt + 1 })),
        })],
      })
      expect(parseSpy).toHaveBeenCalledTimes(2)

      const changed = activity('completed')
      changed.items[0]!.parts[1] = {
        ...changed.items[0]!.parts[1]!,
        text: 'First summary updated',
        updatedAt: changed.items[0]!.parts[1]!.updatedAt + 2,
      }
      await wrapper.setProps({ activities: [changed] })
      expect(parseSpy).toHaveBeenCalledTimes(3)
    } finally {
      parseSpy.mockRestore()
    }
  })

  it('keeps cross-batch streaming text unparsed, then sanitizes the terminal summary once', async () => {
    const parseSpy = vi.spyOn(marked, 'parse')
    const sanitizeSpy = vi.spyOn(DOMPurify, 'sanitize')
    try {
      const first = activity('streaming')
      first.items[0]!.parts = [{
        ...first.items[0]!.parts[1]!,
        text: 'Plan **one**\n',
        streamingTextChunks: ['Check '],
        status: 'streaming',
        lastSequenceNumber: 10,
      }]
      const wrapper = mountPanel([first])
      const streaming = wrapper.get('.chat-activity-panel__markdown--streaming')

      expect(streaming.element.textContent).toBe('Plan **one**\nCheck ')
      expect(streaming.find('strong').exists()).toBe(false)
      expect(parseSpy).not.toHaveBeenCalled()
      expect(sanitizeSpy).not.toHaveBeenCalled()

      const second = activity('streaming')
      second.items[0]!.parts = [{
        ...second.items[0]!.parts[1]!,
        text: 'Plan **one**\n',
        streamingTextChunks: [
          'Check ',
          '<img src=x onerror=alert(1)>',
          '\nFinish',
        ],
        status: 'streaming',
        updatedAt: 127_000,
        lastSequenceNumber: 11,
      }]
      await wrapper.setProps({ activities: [second] })

      const updatedStreaming = wrapper.get('.chat-activity-panel__markdown--streaming')
      expect(updatedStreaming.element.textContent).toBe(
        'Plan **one**\nCheck <img src=x onerror=alert(1)>\nFinish',
      )
      expect(updatedStreaming.find('img').exists()).toBe(false)
      expect(parseSpy).not.toHaveBeenCalled()
      expect(sanitizeSpy).not.toHaveBeenCalled()

      const terminal = activity('completed')
      terminal.items[0]!.parts = [{
        ...terminal.items[0]!.parts[1]!,
        text: 'Plan **one**\nCheck <img src=x onerror=alert(1)>\nFinish',
        status: 'completed',
        updatedAt: 128_000,
        lastSequenceNumber: 12,
      }]
      await wrapper.setProps({ activities: [terminal] })

      expect(wrapper.find('.chat-activity-panel__markdown--streaming').exists()).toBe(false)
      expect(wrapper.get('.chat-activity-panel__markdown').find('strong').text()).toBe('one')
      expect(wrapper.find('img').exists()).toBe(false)
      expect(parseSpy).toHaveBeenCalledTimes(1)
      expect(sanitizeSpy).toHaveBeenCalledTimes(1)

      await wrapper.setProps({
        activities: [{ ...terminal, updatedAt: terminal.updatedAt + 1 }],
      })
      expect(parseSpy).toHaveBeenCalledTimes(1)
      expect(sanitizeSpy).toHaveBeenCalledTimes(1)
    } finally {
      parseSpy.mockRestore()
      sanitizeSpy.mockRestore()
    }
  })

  it('shows standard, Pro, elapsed, and terminal status labels from Activity state', async () => {
    const wrapper = mountPanel([activity('completed')])
    expect(wrapper.text()).toContain('chat.activity.completed')
    expect(wrapper.text()).toContain('2m 5s')
    expect(wrapper.find('[data-icon="activity"]').exists()).toBe(true)

    await wrapper.setProps({
      activities: [activity('streaming', {
        reasoningMode: 'pro',
        startedAt: Date.now(),
        updatedAt: Date.now(),
        completedAt: undefined,
      })],
    })
    expect(wrapper.text()).toContain('chat.activity.proThinking')
    expect(wrapper.find('[data-icon="brain"]').exists()).toBe(true)
    expect(wrapper.find('[role="progressbar"]').exists()).toBe(true)

    for (const status of ['failed', 'stopped', 'incomplete', 'disconnected'] as const) {
      await wrapper.setProps({ activities: [activity(status)] })
      expect(wrapper.text()).toContain(`chat.activity.${status}`)
    }
  })

  it('uses an accessible mobile dialog and closes on Escape', async () => {
    const wrapper = mountPanel([activity()], true, true)
    await nextTick()
    const panel = wrapper.get('[data-test="chat-activity-panel"]')

    expect(panel.attributes('role')).toBe('dialog')
    expect(panel.attributes('aria-modal')).toBe('true')
    expect(panel.attributes('aria-labelledby')).toBeTruthy()
    expect(wrapper.get('.chat-activity-panel__close').attributes('aria-label')).toBe(
      'chat.activity.close',
    )

    await panel.trigger('keydown', { key: 'Escape' })
    expect(wrapper.emitted('close')).toHaveLength(1)

    await wrapper.get('.chat-activity-panel__close').trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(2)
  })

  it('traps forward and reverse Tab navigation inside the mobile dialog', async () => {
    vi.spyOn(HTMLElement.prototype, 'getClientRects').mockReturnValue([
      { width: 1, height: 1 },
    ] as unknown as DOMRectList)
    const wrapper = mountPanel([activity()], true, true)
    await nextTick()
    const panel = wrapper.get('[data-test="chat-activity-panel"]')
    const close = wrapper.get<HTMLButtonElement>('.chat-activity-panel__close').element
    const trailingButton = document.createElement('button')
    trailingButton.type = 'button'
    panel.element.append(trailingButton)

    close.focus()
    expect(document.activeElement).toBe(close)
    await panel.trigger('keydown', { key: 'Tab', shiftKey: true })
    expect(document.activeElement).toBe(trailingButton)

    await panel.trigger('keydown', { key: 'Tab' })
    expect(document.activeElement).toBe(close)

    const outsideButton = document.createElement('button')
    document.body.append(outsideButton)
    outsideButton.focus()
    await panel.trigger('keydown', { key: 'Tab' })
    expect(document.activeElement).toBe(close)
    outsideButton.remove()
  })

  it('locks body scrolling and moves focus inside when a desktop panel becomes mobile', async () => {
    const initialOverflow = document.body.style.overflow
    const outsideButton = document.createElement('button')
    document.body.append(outsideButton)
    const wrapper = mountPanel([activity()], false, true)
    outsideButton.focus()
    expect(document.activeElement).toBe(outsideButton)

    await wrapper.setProps({ mobile: true })
    await nextTick()

    const close = wrapper.get<HTMLButtonElement>('.chat-activity-panel__close').element
    expect(wrapper.get('[data-test="chat-activity-panel"]').attributes('aria-modal')).toBe('true')
    expect(document.activeElement).toBe(close)
    expect(document.body.style.overflow).toBe('hidden')

    await wrapper.setProps({ mobile: false })
    await nextTick()
    expect(document.body.style.overflow).toBe(initialOverflow)
    outsideButton.remove()
  })

  it('follows streaming output, pauses after user scroll-up, and resumes at the bottom', async () => {
    const first = activity('streaming')
    const wrapper = mountPanel([first])
    const scroller = wrapper.get<HTMLElement>('[data-test="chat-activity-scroll-region"]')
      .element
    let scrollHeight = 500
    Object.defineProperty(scroller, 'scrollHeight', {
      configurable: true,
      get: () => scrollHeight,
    })
    Object.defineProperty(scroller, 'clientHeight', {
      configurable: true,
      get: () => 100,
    })

    const followed = activity('streaming')
    followed.items[0]!.parts[0] = {
      ...followed.items[0]!.parts[0]!,
      text: 'Second summary grows',
      updatedAt: followed.items[0]!.parts[0]!.updatedAt + 1,
      lastSequenceNumber: 10,
    }
    scrollHeight = 600
    await wrapper.setProps({ activities: [followed] })
    await nextTick()
    expect(scroller.scrollTop).toBe(600)

    scroller.scrollTop = 100
    await wrapper.get('[data-test="chat-activity-scroll-region"]').trigger('scroll')
    const paused = activity('streaming')
    paused.items[0]!.parts[0] = {
      ...paused.items[0]!.parts[0]!,
      text: 'Second summary grows while paused',
      updatedAt: paused.items[0]!.parts[0]!.updatedAt + 2,
      lastSequenceNumber: 11,
    }
    scrollHeight = 700
    await wrapper.setProps({ activities: [paused] })
    await nextTick()
    expect(scroller.scrollTop).toBe(100)

    scroller.scrollTop = 590
    await wrapper.get('[data-test="chat-activity-scroll-region"]').trigger('scroll')
    const resumed = activity('streaming')
    resumed.items[0]!.parts[0] = {
      ...resumed.items[0]!.parts[0]!,
      text: 'Second summary grows after resume',
      updatedAt: resumed.items[0]!.parts[0]!.updatedAt + 3,
      lastSequenceNumber: 12,
    }
    scrollHeight = 800
    await wrapper.setProps({ activities: [resumed] })
    await nextTick()
    expect(scroller.scrollTop).toBe(800)
  })

  it('throttles aria-live updates and announces only a bounded new fragment', async () => {
    vi.useFakeTimers()
    const initial = activity('streaming')
    const wrapper = mountPanel([initial])
    const liveRegion = wrapper.get('[data-test="chat-activity-live-region"]')

    expect(liveRegion.text()).toBe('')
    vi.advanceTimersByTime(799)
    await nextTick()
    expect(liveRegion.text()).toBe('')
    vi.advanceTimersByTime(1)
    await nextTick()
    expect(liveRegion.text()).toContain('chat.activity.thinking')
    expect(liveRegion.text()).toContain('Second summary')

    const appended = activity('streaming')
    appended.items[0]!.parts[0] = {
      ...appended.items[0]!.parts[0]!,
      streamingTextChunks: ['middle fragment', 'x'.repeat(500)],
      updatedAt: appended.items[0]!.parts[0]!.updatedAt + 1,
      lastSequenceNumber: 10,
    }
    await wrapper.setProps({ activities: [appended] })
    vi.advanceTimersByTime(799)
    await nextTick()
    expect(liveRegion.text()).toContain('Second summary')
    vi.advanceTimersByTime(1)
    await nextTick()
    expect(liveRegion.text()).not.toContain('Second summary')
    expect(liveRegion.text().endsWith('x'.repeat(320))).toBe(true)
  })

  it('keeps desktop width, mobile safe-area, dark-token, and reduced-motion contracts', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src/components/chat/ChatActivityPanel.vue'),
      'utf8',
    )

    expect(source).toContain('width: 392px')
    expect(source).toContain('min-width: 360px')
    expect(source).toContain('max-width: 420px')
    expect(source).toContain('env(safe-area-inset-bottom)')
    expect(source).toContain('var(--workspace-surface)')
    expect(source).toMatch(
      /\.chat-activity-panel__elapsed\s*\{[^}]*color: var\(--workspace-text-secondary\)/,
    )
    expect(source).toMatch(
      /\.chat-activity-panel--mobile \.chat-activity-panel__close\s*\{[^}]*width: 44px;[^}]*height: 44px/,
    )
    expect(source).toMatch(
      /\.chat-activity-panel__markdown--streaming\s*\{[^}]*white-space: pre-wrap/,
    )
    expect(source).toContain('@media (prefers-reduced-motion: reduce)')
  })
})

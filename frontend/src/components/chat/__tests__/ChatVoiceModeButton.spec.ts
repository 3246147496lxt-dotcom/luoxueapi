import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import ChatVoiceModeButton from '../ChatVoiceModeButton.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

const IconStub = {
  props: ['name', 'size'],
  template: '<span :data-icon="name" :data-size="size" />',
}

const wrappers: VueWrapper[] = []

afterEach(() => {
  wrappers.splice(0).forEach((wrapper) => wrapper.unmount())
  document.body.innerHTML = ''
})

function latestTooltip(): HTMLElement {
  const tooltips = document.body.querySelectorAll<HTMLElement>(
    '[data-ui-portal="chat-control-tooltip"][role="tooltip"]',
  )
  const tooltip = tooltips.item(tooltips.length - 1)
  if (!tooltip) throw new Error('Expected a portaled chat control tooltip')
  return tooltip
}

function mountVoiceMode(props: Record<string, unknown> = {}) {
  const wrapper = mount(ChatVoiceModeButton, {
    props,
    global: { stubs: { Icon: IconStub } },
  })
  wrappers.push(wrapper)
  return wrapper
}

describe('ChatVoiceModeButton', () => {
  it('uses the waveform control and copies the voice-mode hover hint', async () => {
    const wrapper = mountVoiceMode()
    const trigger = wrapper.get('[data-test="chat-voice-mode-trigger"]')
    const tooltip = latestTooltip()

    expect(trigger.get('[data-icon="chatVoiceMode"]').attributes('data-icon')).toBe('chatVoiceMode')
    expect(trigger.get('[data-icon="chatVoiceMode"]').attributes('data-size')).toBe('lg')
    expect(trigger.attributes('aria-label')).toBe('chat.voice.modeStart')
    expect(trigger.attributes('aria-describedby')).toBeUndefined()
    expect(tooltip.getAttribute('aria-hidden')).toBe('true')
    expect(trigger.attributes('aria-keyshortcuts')).toBe('Control+Shift+V')
    expect(tooltip.textContent).toContain('chat.voice.modeStart')
    expect(tooltip.textContent).toContain('V')

    await trigger.trigger('click')
    expect(wrapper.emitted('activate')).toHaveLength(1)

    const shortcut = new KeyboardEvent('keydown', {
      code: 'KeyV',
      ctrlKey: true,
      shiftKey: true,
      bubbles: true,
      cancelable: true,
    })
    window.dispatchEvent(shortcut)
    expect(shortcut.defaultPrevented).toBe(true)
    expect(wrapper.emitted('activate')).toHaveLength(2)
  })

  it('stays visible and focusable but truthfully unavailable without realtime voice', async () => {
    const wrapper = mountVoiceMode({ available: false })
    const trigger = wrapper.get('[data-test="chat-voice-mode-trigger"]')

    expect(trigger.attributes('disabled')).toBeUndefined()
    expect(trigger.attributes('aria-disabled')).toBe('true')
    expect(trigger.attributes('aria-label')).toBe('chat.voice.modeUnavailable')
    expect(trigger.attributes('aria-keyshortcuts')).toBeUndefined()
    expect(latestTooltip().textContent).toBe('chat.voice.modeUnavailable')

    await trigger.trigger('click')
    expect(wrapper.emitted('activate')).toBeUndefined()
  })

  it('does not advertise or invoke the shortcut while disabled or behind a modal', () => {
    const disabled = mountVoiceMode({ disabled: true })
    expect(disabled.get('[data-test="chat-voice-mode-trigger"]')
      .attributes('aria-keyshortcuts')).toBeUndefined()
    expect(latestTooltip().getAttribute('aria-hidden')).toBe('true')

    const enabled = mountVoiceMode()
    const dialog = document.createElement('div')
    dialog.setAttribute('role', 'dialog')
    dialog.setAttribute('aria-modal', 'true')
    document.body.append(dialog)
    window.dispatchEvent(new KeyboardEvent('keydown', {
      code: 'KeyV',
      ctrlKey: true,
      shiftKey: true,
      bubbles: true,
      cancelable: true,
    }))
    expect(enabled.emitted('activate')).toBeUndefined()
    dialog.remove()
  })

  it('keeps the shortcut active when a closed inert overlay retains its dialog in the DOM', () => {
    const enabled = mountVoiceMode()
    const hiddenOverlay = document.createElement('div')
    hiddenOverlay.setAttribute('aria-hidden', 'true')
    hiddenOverlay.setAttribute('inert', '')

    const dialog = document.createElement('div')
    dialog.setAttribute('role', 'dialog')
    dialog.setAttribute('aria-modal', 'true')
    hiddenOverlay.append(dialog)
    document.body.append(hiddenOverlay)

    const shortcut = new KeyboardEvent('keydown', {
      code: 'KeyV',
      ctrlKey: true,
      shiftKey: true,
      bubbles: true,
      cancelable: true,
    })
    window.dispatchEvent(shortcut)

    expect(shortcut.defaultPrevented).toBe(true)
    expect(enabled.emitted('activate')).toHaveLength(1)
  })
})

import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'

import ApiKeyDetailSheet from '../ApiKeyDetailSheet.vue'
import detailSheetSource from '../ApiKeyDetailSheet.vue?raw'
import inspectorSource from '../ApiKeyInspector.vue?raw'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key === 'common.close' ? 'Close' : key }),
  }
})

const mountedWrappers: VueWrapper[] = []

function mountSheet(show = true) {
  const wrapper = mount(ApiKeyDetailSheet, {
    attachTo: document.body,
    props: {
      show,
      title: 'Key details',
      subtitle: 'Production key',
    },
    slots: {
      content: '<button data-test="sheet-action">Action</button>',
    },
    global: {
      stubs: {
        Icon: {
          props: ['name'],
          template: '<span :data-icon="name">{{ name }}</span>',
        },
      },
    },
  })
  mountedWrappers.push(wrapper)
  return wrapper
}

afterEach(() => {
  while (mountedWrappers.length) mountedWrappers.pop()?.unmount()
  document.body.classList.remove('api-key-detail-sheet-open')
  document.body.replaceChildren()
})

describe('ApiKeyDetailSheet', () => {
  it('keeps the Superdesign bottom-sheet sizing without a fixed-height desktop drawer', () => {
    const sheetRule = detailSheetSource.match(/\.api-key-detail-sheet\s*\{([^}]*)\}/)?.[1]

    expect(sheetRule).toBeDefined()
    expect(sheetRule).toContain('max-height: 94vh;')
    expect(sheetRule).not.toMatch(/(?:^|\n)\s*height\s*:/)
    expect(detailSheetSource).not.toMatch(/(?:94|100)dvh/)
    expect(detailSheetSource).not.toContain('@media (min-width: 768px)')
  })

  it('keeps the sheet content flexible while allowing the inspector to shrink', () => {
    const contentRule = detailSheetSource.match(/\.api-key-detail-sheet__content\s*\{([^}]*)\}/)?.[1]
    const sheetInspectorRule = inspectorSource.match(/\.api-key-inspector--sheet\s*\{([^}]*)\}/)?.[1]

    expect(contentRule).toContain('flex: 1 1 auto;')
    expect(sheetInspectorRule).toContain('flex: 0 1 auto;')
  })

  it('teleports an accessible overlay, locks body scrolling, and closes from its backdrop', async () => {
    const trigger = document.createElement('button')
    trigger.textContent = 'Open details'
    document.body.appendChild(trigger)
    trigger.focus()

    const wrapper = mountSheet()
    await flushPromises()

    const overlay = document.body.querySelector<HTMLElement>('[data-test="api-key-detail-sheet-overlay"]')
    const sheet = document.body.querySelector<HTMLElement>('[data-test="api-key-detail-sheet"]')
    const closeButton = document.body.querySelector<HTMLButtonElement>('[data-test="api-key-detail-sheet-close"]')

    expect(overlay).not.toBeNull()
    expect(sheet?.getAttribute('role')).toBe('dialog')
    expect(sheet?.getAttribute('aria-modal')).toBe('true')
    expect(sheet?.getAttribute('aria-describedby')).toMatch(/^api-key-detail-sheet-subtitle-/)
    expect(document.body.classList.contains('api-key-detail-sheet-open')).toBe(true)
    expect(document.activeElement).toBe(closeButton)

    overlay?.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await flushPromises()
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('closes on Escape, unlocks the body, and restores focus after the parent hides it', async () => {
    const trigger = document.createElement('button')
    trigger.textContent = 'Selected key'
    document.body.appendChild(trigger)
    trigger.focus()

    const wrapper = mountSheet()
    await flushPromises()

    document.dispatchEvent(new KeyboardEvent('keydown', {
      key: 'Escape',
      bubbles: true,
      cancelable: true,
    }))
    await flushPromises()
    expect(wrapper.emitted('close')).toHaveLength(1)

    await wrapper.setProps({ show: false })
    await flushPromises()

    expect(document.body.classList.contains('api-key-detail-sheet-open')).toBe(false)
    expect(document.activeElement).toBe(trigger)
  })

  it('removes the body lock and restores focus when an open sheet unmounts', async () => {
    const trigger = document.createElement('button')
    document.body.appendChild(trigger)
    trigger.focus()

    const wrapper = mountSheet()
    await flushPromises()
    expect(document.body.classList.contains('api-key-detail-sheet-open')).toBe(true)

    wrapper.unmount()

    expect(document.body.classList.contains('api-key-detail-sheet-open')).toBe(false)
    expect(document.activeElement).toBe(trigger)
  })
})

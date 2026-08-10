import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import BaseDialog from '../BaseDialog.vue'

let wrapper: VueWrapper | undefined

beforeEach(() => {
  vi.spyOn(HTMLElement.prototype, 'getClientRects').mockImplementation(function (this: HTMLElement) {
    return this.isConnected
      ? [new DOMRect(0, 0, 100, 36)] as unknown as DOMRectList
      : [] as unknown as DOMRectList
  })
})

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  document.body.innerHTML = ''
  document.body.classList.remove('modal-open')
  document.body.style.overflow = ''
  vi.restoreAllMocks()
})

describe('BaseDialog', () => {
  it('keeps the default dialog contract unchanged', async () => {
    wrapper = mount(BaseDialog, {
      attachTo: document.body,
      props: {
        show: true,
        title: 'Default dialog',
      },
      global: {
        stubs: {
          Icon: true,
          Transition: false,
        },
      },
    })
    await flushPromises()

    const dialog = document.body.querySelector<HTMLElement>('[role="dialog"]')
    expect(dialog).not.toBeNull()
    expect(dialog?.classList.contains('modal-overlay--workspace-confirm')).toBe(false)
    expect(dialog?.getAttribute('aria-describedby')).toBeNull()
    expect(dialog?.querySelector('.modal-content--workspace-confirm')).toBeNull()
    expect(dialog?.querySelector('button[aria-label="Close modal"]')).not.toBeNull()
  })

  it('applies the opt-in Workspace confirmation variant and focuses Cancel first', async () => {
    wrapper = mount(BaseDialog, {
      attachTo: document.body,
      props: {
        show: true,
        title: 'Delete chat?',
        width: 'narrow',
        variant: 'workspace-confirm',
        descriptionId: 'delete-description',
        showCloseButton: false,
      },
      slots: {
        default: '<p id="delete-description">Delete this chat.</p>',
        footer: `
          <button type="button" data-test="cancel">Cancel</button>
          <button type="button" data-test="delete">Delete</button>
        `,
      },
      global: {
        stubs: {
          Icon: true,
          Transition: false,
        },
      },
    })
    await flushPromises()

    const dialog = document.body.querySelector<HTMLElement>('[role="dialog"]')
    expect(dialog?.classList.contains('modal-overlay--workspace-confirm')).toBe(true)
    expect(dialog?.getAttribute('aria-describedby')).toBe('delete-description')
    expect(dialog?.querySelector('.modal-content--workspace-confirm')).not.toBeNull()
    expect(dialog?.querySelector('button[aria-label="Close modal"]')).toBeNull()
    expect(document.activeElement?.getAttribute('data-test')).toBe('cancel')

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('keeps sequential focus inside the dialog and restores the opener', async () => {
    const opener = document.createElement('button')
    document.body.append(opener)
    opener.focus()

    wrapper = mount(BaseDialog, {
      attachTo: document.body,
      props: {
        show: true,
        title: 'Delete chat?',
        showCloseButton: false,
      },
      slots: {
        footer: `
          <button type="button" data-test="cancel">Cancel</button>
          <button type="button" data-test="delete">Delete</button>
        `,
      },
      global: {
        stubs: {
          Icon: true,
          Transition: false,
        },
      },
    })
    await flushPromises()

    const cancel = document.body.querySelector<HTMLButtonElement>('[data-test="cancel"]')
    const deleteButton = document.body.querySelector<HTMLButtonElement>('[data-test="delete"]')
    expect(document.activeElement).toBe(cancel)
    expect(document.body.style.overflow).toBe('hidden')

    deleteButton?.focus()
    const forwardTab = new KeyboardEvent('keydown', {
      key: 'Tab',
      bubbles: true,
      cancelable: true,
    })
    document.dispatchEvent(forwardTab)
    expect(forwardTab.defaultPrevented).toBe(true)
    expect(document.activeElement).toBe(cancel)

    cancel?.focus()
    const backwardTab = new KeyboardEvent('keydown', {
      key: 'Tab',
      shiftKey: true,
      bubbles: true,
      cancelable: true,
    })
    document.dispatchEvent(backwardTab)
    expect(backwardTab.defaultPrevented).toBe(true)
    expect(document.activeElement).toBe(deleteButton)

    await wrapper.setProps({ show: false })
    await flushPromises()
    expect(document.body.style.overflow).toBe('')
    expect(document.activeElement).toBe(opener)
  })
})

import { afterEach, describe, expect, it, vi } from 'vitest'
import { startQuotaViewerDrag } from './desktop'

const startDragging = vi.fn<() => Promise<void>>().mockResolvedValue()

vi.mock('@tauri-apps/api/window', () => ({
  getCurrentWindow: () => ({ startDragging })
}))

const installTauriRuntime = () => {
  Object.defineProperty(window, '__TAURI_INTERNALS__', {
    configurable: true,
    value: {}
  })
}

const removeTauriRuntime = () => {
  Reflect.deleteProperty(window, '__TAURI_INTERNALS__')
}

describe('quota viewer window dragging', () => {
  afterEach(() => {
    removeTauriRuntime()
    startDragging.mockClear()
    document.body.replaceChildren()
  })

  it('starts dragging from a non-interactive region in Tauri', async () => {
    installTauriRuntime()
    const region = document.createElement('section')
    region.addEventListener('mousedown', startQuotaViewerDrag)
    document.body.append(region)

    region.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, button: 0 }))

    await vi.waitFor(() => expect(startDragging).toHaveBeenCalledOnce())
  })

  it('does not drag from controls, secondary buttons, or a browser preview', async () => {
    installTauriRuntime()
    const region = document.createElement('section')
    const button = document.createElement('button')
    region.append(button)
    region.addEventListener('mousedown', startQuotaViewerDrag)
    document.body.append(region)

    button.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, button: 0 }))
    region.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, button: 2 }))
    removeTauriRuntime()
    region.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, button: 0 }))

    await Promise.resolve()
    expect(startDragging).not.toHaveBeenCalled()
  })
})

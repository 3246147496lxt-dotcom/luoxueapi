import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  hideTrayPopover,
  onMainPanelLayoutChanged,
  onTrayPopoverShown,
  openMainPanel,
  setMainPanelExpanded,
  setTrayDetailPanelOpen,
  startQuotaViewerDrag
} from './desktop'

const startDragging = vi.fn<() => Promise<void>>().mockResolvedValue()
const invoke = vi
  .fn<(command: string, args?: Record<string, unknown>) => Promise<void>>()
  .mockResolvedValue()
const unlisten = vi.fn()
const listen = vi
  .fn<(event: string, callback: () => void) => Promise<() => void>>()
  .mockResolvedValue(unlisten)

vi.mock('@tauri-apps/api/window', () => ({
  getCurrentWindow: () => ({ startDragging })
}))

vi.mock('@tauri-apps/api/core', () => ({ invoke }))
vi.mock('@tauri-apps/api/event', () => ({ listen }))

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
    invoke.mockClear()
    listen.mockClear()
    unlisten.mockClear()
    document.body.replaceChildren()
  })

  it('starts dragging from a non-interactive region in Tauri', async () => {
    installTauriRuntime()
    const region = document.createElement('section')
    region.addEventListener('mousedown', startQuotaViewerDrag)
    document.body.append(region)

    region.dispatchEvent(
      new MouseEvent('mousedown', { bubbles: true, button: 0 })
    )

    await vi.waitFor(() => expect(startDragging).toHaveBeenCalledOnce())
  })

  it('does not drag from controls, secondary buttons, or a browser preview', async () => {
    installTauriRuntime()
    const region = document.createElement('section')
    const button = document.createElement('button')
    region.append(button)
    region.addEventListener('mousedown', startQuotaViewerDrag)
    document.body.append(region)

    button.dispatchEvent(
      new MouseEvent('mousedown', { bubbles: true, button: 0 })
    )
    region.dispatchEvent(
      new MouseEvent('mousedown', { bubbles: true, button: 2 })
    )
    removeTauriRuntime()
    region.dispatchEvent(
      new MouseEvent('mousedown', { bubbles: true, button: 0 })
    )

    await Promise.resolve()
    expect(startDragging).not.toHaveBeenCalled()
  })

  it('opens and hides the tray popover through native commands', async () => {
    installTauriRuntime()

    await openMainPanel()
    await hideTrayPopover()

    expect(invoke).toHaveBeenNthCalledWith(1, 'open_main_panel', undefined)
    expect(invoke).toHaveBeenNthCalledWith(2, 'hide_tray_panel', undefined)
  })

  it('resizes the main and tray surfaces through independent commands', async () => {
    installTauriRuntime()

    await setMainPanelExpanded(true)
    await setTrayDetailPanelOpen(true)

    expect(invoke).toHaveBeenNthCalledWith(1, 'set_main_panel_expanded', {
      expanded: true
    })
    expect(invoke).toHaveBeenNthCalledWith(2, 'set_tray_detail_open', {
      open: true
    })
  })

  it('listens for each native tray reveal', async () => {
    installTauriRuntime()
    const callback = vi.fn()

    const stop = await onTrayPopoverShown(callback)

    expect(listen).toHaveBeenCalledWith('quota-tray-shown', callback)
    expect(stop).toBe(unlisten)
  })

  it('maps the native main layout payload to a boolean callback', async () => {
    installTauriRuntime()
    const callback = vi.fn()

    await onMainPanelLayoutChanged(callback)

    const listener = listen.mock.calls[0]?.[1]
    ;(listener as unknown as (event: { payload: boolean }) => void)({
      payload: true
    })

    expect(listen).toHaveBeenCalledWith(
      'quota-main-layout-changed',
      expect.any(Function)
    )
    expect(callback).toHaveBeenCalledWith(true)
  })
})

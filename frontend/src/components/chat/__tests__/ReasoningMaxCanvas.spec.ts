import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ReasoningMaxCanvas from '../ReasoningMaxCanvas.vue'

describe('ReasoningMaxCanvas', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('draws the moving purple track and releases its animation lifecycle', () => {
    const addColorStop = vi.fn()
    const gradient = { addColorStop }
    const frameCallbacks: FrameRequestCallback[] = []
    const createRadialGradient = vi.fn((
      _x0: number,
      _y0: number,
      _r0: number,
      _x1: number,
      _y1: number,
      _r1: number,
    ) => gradient)
    const context = {
      clearRect: vi.fn(),
      createRadialGradient,
      fillRect: vi.fn(),
      setTransform: vi.fn(),
      fillStyle: '',
      globalCompositeOperation: 'source-over',
    }
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(
      context as unknown as CanvasRenderingContext2D,
    )
    const requestAnimationFrame = vi.fn((callback: FrameRequestCallback) => {
      frameCallbacks.push(callback)
      return 41
    })
    const cancelAnimationFrame = vi.fn()
    const addMediaListener = vi.fn()
    const removeMediaListener = vi.fn()
    vi.stubGlobal('requestAnimationFrame', requestAnimationFrame)
    vi.stubGlobal('cancelAnimationFrame', cancelAnimationFrame)
    vi.stubGlobal('matchMedia', vi.fn(() => ({
      matches: false,
      addEventListener: addMediaListener,
      removeEventListener: removeMediaListener,
    })))
    vi.stubGlobal('ResizeObserver', class {
      observe() {}
      disconnect() {}
    })

    const view = mount(ReasoningMaxCanvas)

    expect(addColorStop).toHaveBeenCalledWith(0, 'rgba(27, 5, 82, 0.46)')
    expect(addColorStop).toHaveBeenCalledWith(0, 'rgba(197, 91, 232, 0.34)')
    expect(addColorStop).toHaveBeenCalledWith(0, 'rgba(111, 44, 205, 0.28)')
    expect(addColorStop.mock.calls.some(([, color]) => String(color).includes('255'))).toBe(false)
    expect(context.createRadialGradient).toHaveBeenCalledTimes(3)
    expect(context.globalCompositeOperation).toBe('source-over')
    expect(context.fillRect).toHaveBeenCalledTimes(3)
    expect(requestAnimationFrame).toHaveBeenCalledOnce()
    expect(addMediaListener).toHaveBeenCalledWith('change', expect.any(Function))

    const initialDeepCenter = context.createRadialGradient.mock.calls[0]?.[0]
    frameCallbacks.shift()?.(1_000)
    frameCallbacks.shift()?.(2_950)
    const movingDeepCenter = context.createRadialGradient.mock.calls[6]?.[0]
    expect(movingDeepCenter).not.toBe(initialDeepCenter)
    expect(context.createRadialGradient).toHaveBeenCalledTimes(9)

    view.unmount()

    expect(cancelAnimationFrame).toHaveBeenCalledWith(41)
    expect(removeMediaListener).toHaveBeenCalledWith('change', expect.any(Function))
  })

  it('keeps a static purple texture without scheduling frames for reduced motion', () => {
    const gradient = { addColorStop: vi.fn() }
    const context = {
      clearRect: vi.fn(),
      createRadialGradient: vi.fn(() => gradient),
      fillRect: vi.fn(),
      setTransform: vi.fn(),
      fillStyle: '',
      globalCompositeOperation: 'source-over',
    }
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(
      context as unknown as CanvasRenderingContext2D,
    )
    const requestAnimationFrame = vi.fn(() => 42)
    vi.stubGlobal('requestAnimationFrame', requestAnimationFrame)
    vi.stubGlobal('cancelAnimationFrame', vi.fn())
    vi.stubGlobal('matchMedia', vi.fn(() => ({
      matches: true,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })))
    vi.stubGlobal('ResizeObserver', class {
      observe() {}
      disconnect() {}
    })

    const view = mount(ReasoningMaxCanvas)

    expect(context.createRadialGradient).toHaveBeenCalledTimes(3)
    expect(context.fillRect).toHaveBeenCalledTimes(3)
    expect(requestAnimationFrame).not.toHaveBeenCalled()

    view.unmount()
  })
})

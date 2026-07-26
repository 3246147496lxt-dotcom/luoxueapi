import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  getVisibleFocusableElements,
  isVisibleFocusableElement,
} from '../modalStack'

function rectList(length = 1): DOMRectList {
  if (length === 0) return [] as unknown as DOMRectList
  const rect = {
    bottom: 20,
    height: 20,
    left: 0,
    right: 20,
    top: 0,
    width: 20,
    x: 0,
    y: 0,
    toJSON: () => ({}),
  } as DOMRect
  return [rect] as unknown as DOMRectList
}

beforeEach(() => {
  vi.spyOn(Element.prototype, 'getClientRects').mockImplementation(function (this: Element) {
    return (this as HTMLElement).dataset.noLayout === 'true'
      ? rectList(0)
      : rectList()
  })
})

afterEach(() => {
  document.body.replaceChildren()
  vi.restoreAllMocks()
})

describe('modal focus visibility', () => {
  it('excludes disabled, hidden, inert, and layoutless controls', () => {
    const container = document.createElement('div')
    const visible = document.createElement('button')
    const disabled = document.createElement('button')
    const displayNone = document.createElement('button')
    const visibilityHidden = document.createElement('button')
    const layoutless = document.createElement('button')
    const inertParent = document.createElement('div')
    const inertChild = document.createElement('button')

    disabled.disabled = true
    displayNone.style.display = 'none'
    visibilityHidden.style.visibility = 'hidden'
    layoutless.dataset.noLayout = 'true'
    inertParent.inert = true
    inertParent.appendChild(inertChild)
    container.append(
      visible,
      disabled,
      displayNone,
      visibilityHidden,
      layoutless,
      inertParent,
    )
    document.body.appendChild(container)

    expect(getVisibleFocusableElements(container)).toEqual([visible])
    expect(isVisibleFocusableElement(document.body)).toBe(false)
    expect(isVisibleFocusableElement(inertChild)).toBe(false)
  })

  it('excludes controls under hidden layout ancestors', () => {
    const container = document.createElement('div')
    const hiddenParent = document.createElement('div')
    const child = document.createElement('button')
    hiddenParent.style.display = 'none'
    hiddenParent.appendChild(child)
    container.appendChild(hiddenParent)
    document.body.appendChild(container)

    child.dataset.noLayout = 'true'

    expect(getVisibleFocusableElements(container)).toEqual([])
  })
})

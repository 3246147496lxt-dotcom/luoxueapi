import { afterEach, describe, expect, it } from 'vitest'

import {
  WORKSPACE_MOBILE_DRAWER_MEDIA_QUERY,
  WORKSPACE_NARROW_SIDEBAR_MEDIA_QUERY,
  workspaceMobileDrawerFallback,
  workspaceNarrowSidebarFallback,
} from '../workspaceResponsive'

const originalInnerWidth = window.innerWidth
const originalMaxTouchPoints = Object.getOwnPropertyDescriptor(
  window.navigator,
  'maxTouchPoints',
)

function setViewportEnvironment(width: number, maxTouchPoints: number) {
  Object.defineProperty(window, 'innerWidth', {
    configurable: true,
    value: width,
  })
  Object.defineProperty(window.navigator, 'maxTouchPoints', {
    configurable: true,
    value: maxTouchPoints,
  })
}

afterEach(() => {
  Object.defineProperty(window, 'innerWidth', {
    configurable: true,
    value: originalInnerWidth,
  })
  if (originalMaxTouchPoints) {
    Object.defineProperty(window.navigator, 'maxTouchPoints', originalMaxTouchPoints)
  } else {
    Reflect.deleteProperty(window.navigator, 'maxTouchPoints')
  }
})

describe('workspace responsive fallback', () => {
  it('copies the ChatGPT 768px boundary into mutually exclusive shell queries', () => {
    expect(WORKSPACE_NARROW_SIDEBAR_MEDIA_QUERY).toBe(
      '(max-width: 767px) and (hover: hover) and (pointer: fine)',
    )
    expect(WORKSPACE_MOBILE_DRAWER_MEDIA_QUERY).toBe(
      '(max-width: 767px) and (hover: none) and (pointer: coarse)',
    )
  })

  it('uses the hidden overlay presentation for a 614px fine-pointer desktop window', () => {
    setViewportEnvironment(614, 0)

    expect(workspaceMobileDrawerFallback()).toBe(false)
    expect(workspaceNarrowSidebarFallback()).toBe(true)
  })

  it('uses the drawer for a 614px touch-first environment', () => {
    setViewportEnvironment(614, 5)

    expect(workspaceMobileDrawerFallback()).toBe(true)
    expect(workspaceNarrowSidebarFallback()).toBe(false)
  })

  it('keeps the 768px tablet boundary on the desktop shell', () => {
    setViewportEnvironment(768, 5)

    expect(workspaceMobileDrawerFallback()).toBe(false)
    expect(workspaceNarrowSidebarFallback()).toBe(false)
  })
})

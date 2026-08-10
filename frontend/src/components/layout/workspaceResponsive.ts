import {
  computed,
  getCurrentScope,
  onScopeDispose,
} from 'vue'
import { useAppStore } from '@/stores/app'

export const WORKSPACE_MOBILE_DRAWER_MEDIA_QUERY =
  '(max-width: 767px) and (hover: none) and (pointer: coarse)'

export const WORKSPACE_NARROW_SIDEBAR_MEDIA_QUERY =
  '(max-width: 767px) and (hover: hover) and (pointer: fine)'

export function workspaceMobileDrawerFallback(): boolean {
  if (typeof window === 'undefined') return false

  return window.innerWidth < 768 && window.navigator.maxTouchPoints > 0
}

export function workspaceNarrowSidebarFallback(): boolean {
  if (typeof window === 'undefined') return false

  return window.innerWidth < 768 && window.navigator.maxTouchPoints === 0
}

/**
 * Shared Workspace viewport classifier.
 *
 * Chat and Work may mount independently, so every host observes the same two
 * media queries and writes the result into the shared UI store. The queries
 * are mutually exclusive: touch-first narrow viewports use a Drawer, while
 * fine-pointer narrow windows hide the docked Sidebar and use an overlay.
 */
export function useWorkspaceResponsiveState() {
  const appStore = useAppStore()
  let mobileQuery: MediaQueryList | null = null
  let narrowQuery: MediaQueryList | null = null

  function syncViewportState() {
    const mobileDrawer = mobileQuery?.matches ?? workspaceMobileDrawerFallback()
    const narrowSidebar = !mobileDrawer
      && (narrowQuery?.matches ?? workspaceNarrowSidebarFallback())

    appStore.setWorkspaceMobileDrawer(mobileDrawer)
    appStore.setWorkspaceNarrowSidebar(narrowSidebar)
    if (!mobileDrawer && appStore.mobileOpen) appStore.setMobileOpen(false)
  }

  if (typeof window !== 'undefined') {
    if (typeof window.matchMedia === 'function') {
      mobileQuery = window.matchMedia(WORKSPACE_MOBILE_DRAWER_MEDIA_QUERY)
      narrowQuery = window.matchMedia(WORKSPACE_NARROW_SIDEBAR_MEDIA_QUERY)
      syncViewportState()

      if (getCurrentScope()) {
        mobileQuery.addEventListener?.('change', syncViewportState)
        narrowQuery.addEventListener?.('change', syncViewportState)
        onScopeDispose(() => {
          mobileQuery?.removeEventListener?.('change', syncViewportState)
          narrowQuery?.removeEventListener?.('change', syncViewportState)
        })
      }
    } else {
      syncViewportState()
    }
  }

  return {
    mobileDrawer: computed(() => appStore.workspaceMobileDrawer),
    narrowSidebar: computed(() => appStore.workspaceNarrowSidebar),
  }
}

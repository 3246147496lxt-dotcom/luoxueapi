import {
  computed,
  nextTick,
  ref,
  toValue,
  type MaybeRefOrGetter,
} from 'vue'
import { useAppStore } from '@/stores/app'

export interface WorkspaceSidebarHeaderHandle {
  focusSearch: () => void
  focusToggle: () => void
  focusClose: () => void
}

interface WorkspaceSidebarCollapseOptions {
  enabled?: MaybeRefOrGetter<boolean>
  mobile?: MaybeRefOrGetter<boolean>
  overlay?: MaybeRefOrGetter<boolean>
  beforeCollapse?: () => void
}

/**
 * Canonical Workspace Sidebar state adapter.
 * Pinia owns the requested desktop state; every host receives the same
 * mobile-aware rendered state and the same transition/focus semantics.
 */
export function useWorkspaceSidebarCollapse(options: WorkspaceSidebarCollapseOptions = {}) {
  const appStore = useAppStore()
  const headerRef = ref<WorkspaceSidebarHeaderHandle | null>(null)
  const enabled = computed(() => toValue(options.enabled ?? true))
  const mobile = computed(() => toValue(options.mobile ?? false))
  const overlay = computed(() => toValue(options.overlay ?? false))
  const requestedCollapsed = computed(() => appStore.sidebarCollapsed)
  const collapsed = computed(() => (
    enabled.value
    && !mobile.value
    && !overlay.value
    && requestedCollapsed.value
  ))

  function expand() {
    if (!collapsed.value) return

    appStore.setSidebarCollapsed(false)
  }

  async function toggle() {
    if (!enabled.value || mobile.value || overlay.value) return

    if (collapsed.value) {
      expand()
    } else {
      options.beforeCollapse?.()
      appStore.setSidebarCollapsed(true)
    }
    await nextTick()
    headerRef.value?.focusToggle()
  }

  return {
    collapsed,
    requestedCollapsed,
    headerRef,
    expand,
    toggle,
  }
}

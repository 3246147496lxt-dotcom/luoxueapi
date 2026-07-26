import {
  nextTick,
  onBeforeUnmount,
  ref,
  type CSSProperties,
  type Ref,
  watch,
} from 'vue'

const VIEWPORT_GUTTER = 8
const ANCHOR_GAP = 8

export function useAnchoredOverlay(
  open: Ref<boolean>,
  anchorElement: Ref<HTMLElement | null>,
  panelElement: Ref<HTMLElement | null>,
) {
  const desktopStyle = ref<CSSProperties>({})
  let resizeObserver: ResizeObserver | null = null

  function updatePosition() {
    if (
      !open.value
      || !anchorElement.value
      || !panelElement.value
      || typeof window === 'undefined'
    ) {
      return
    }

    const anchorRect = anchorElement.value.getBoundingClientRect()
    const panelRect = panelElement.value.getBoundingClientRect()
    const maxLeft = Math.max(VIEWPORT_GUTTER, window.innerWidth - panelRect.width - VIEWPORT_GUTTER)
    const left = Math.min(Math.max(VIEWPORT_GUTTER, anchorRect.left), maxLeft)
    const bottom = Math.max(
      VIEWPORT_GUTTER,
      window.innerHeight - anchorRect.top + ANCHOR_GAP,
    )
    const maxHeight = Math.max(
      0,
      anchorRect.top - ANCHOR_GAP - VIEWPORT_GUTTER,
    )

    desktopStyle.value = {
      left: `${Math.round(left)}px`,
      bottom: `${Math.round(bottom)}px`,
      maxHeight: `${Math.floor(maxHeight)}px`,
    }
  }

  function handleViewportChange() {
    updatePosition()
  }

  async function startTracking() {
    await nextTick()
    updatePosition()
    window.addEventListener('resize', handleViewportChange)
    window.addEventListener('scroll', handleViewportChange, true)

    if (typeof ResizeObserver !== 'undefined' && panelElement.value) {
      resizeObserver = new ResizeObserver(updatePosition)
      resizeObserver.observe(panelElement.value)
    }
  }

  function stopTracking() {
    window.removeEventListener('resize', handleViewportChange)
    window.removeEventListener('scroll', handleViewportChange, true)
    resizeObserver?.disconnect()
    resizeObserver = null
  }

  watch(open, (isOpen) => {
    stopTracking()
    if (isOpen) {
      void startTracking()
    }
  }, { immediate: true })

  watch([anchorElement, panelElement], () => {
    if (open.value) {
      stopTracking()
      void startTracking()
    }
  })

  onBeforeUnmount(stopTracking)

  return {
    desktopStyle,
    updatePosition,
  }
}

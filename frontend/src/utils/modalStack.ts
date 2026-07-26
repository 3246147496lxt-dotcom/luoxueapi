const modalStack: symbol[] = []

const FOCUSABLE_SELECTOR = [
  'a[href]',
  'button',
  'input',
  'select',
  'textarea',
  'iframe',
  'audio[controls]',
  'video[controls]',
  '[contenteditable]:not([contenteditable="false"])',
  '[tabindex]',
].join(',')

export function registerModalLayer(token: symbol): void {
  const existingIndex = modalStack.indexOf(token)
  if (existingIndex >= 0) {
    modalStack.splice(existingIndex, 1)
  }
  modalStack.push(token)
}

export function unregisterModalLayer(token: symbol): void {
  const index = modalStack.lastIndexOf(token)
  if (index >= 0) {
    modalStack.splice(index, 1)
  }
}

export function isTopModalLayer(token: symbol): boolean {
  return modalStack.at(-1) === token
}

function isDisabled(element: HTMLElement): boolean {
  return element.matches(':disabled')
    || (
      'disabled' in element
      && (element as HTMLElement & { disabled?: boolean }).disabled === true
    )
}

function hasHiddenOrInertAncestor(element: HTMLElement): boolean {
  let current: HTMLElement | null = element
  while (current) {
    if (
      current.hidden
      || current.inert
      || current.hasAttribute('inert')
      || current.getAttribute('aria-hidden') === 'true'
    ) {
      return true
    }
    current = current.parentElement
  }
  return false
}

/**
 * Whether an element can safely participate in a modal's sequential focus order.
 *
 * `getClientRects()` is intentionally part of this check: computed styles on the
 * element alone do not reveal a `display: none` ancestor or an element that has
 * otherwise been removed from layout.
 */
export function isVisibleFocusableElement(element: HTMLElement): boolean {
  if (typeof document === 'undefined' || typeof window === 'undefined') {
    return false
  }
  if (
    element === document.body
    || !element.isConnected
    || element.tabIndex < 0
    || isDisabled(element)
    || hasHiddenOrInertAncestor(element)
  ) {
    return false
  }

  const style = window.getComputedStyle(element)
  return (
    style.display !== 'none'
    && style.visibility !== 'hidden'
    && style.visibility !== 'collapse'
    && element.getClientRects().length > 0
  )
}

export function getVisibleFocusableElements(
  container: HTMLElement,
): HTMLElement[] {
  return Array
    .from(container.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR))
    .filter(isVisibleFocusableElement)
}

export function focusFirstVisibleTarget(
  targets: Iterable<HTMLElement | null | undefined>,
): boolean {
  for (const target of targets) {
    if (!target || !isVisibleFocusableElement(target)) continue
    target.focus()
    if (document.activeElement === target) return true
  }
  return false
}

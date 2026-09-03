export function chatControlShortcutIsInScope(
  event: KeyboardEvent,
  trigger: HTMLElement | null,
  additionalRoot: HTMLElement | null = null,
): boolean {
  if (event.defaultPrevented || event.isComposing || !trigger) return false
  if (trigger.closest('[inert], [aria-hidden="true"]')) return false
  const hasActiveModal = Array.from(
    document.querySelectorAll<HTMLElement>('[role="dialog"][aria-modal="true"]'),
  ).some(dialog => !dialog.closest('[inert], [aria-hidden="true"]'))
  if (hasActiveModal) return false

  const composer = trigger.closest('.chat-composer')
  const activeElement = document.activeElement
  if (activeElement && additionalRoot?.contains(activeElement)) return true
  if (!composer) {
    return !activeElement || activeElement === document.body || trigger.contains(activeElement)
  }
  return (
    !activeElement
    || activeElement === document.body
    || composer.contains(activeElement)
  )
}

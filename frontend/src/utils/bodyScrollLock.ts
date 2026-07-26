const activeLocks = new Set<symbol>()
let previousOverflow: string | null = null

export function acquireBodyScrollLock(token: symbol) {
  if (typeof document === 'undefined' || activeLocks.has(token)) return

  if (activeLocks.size === 0) {
    previousOverflow = document.body.style.overflow
  }

  activeLocks.add(token)
  document.body.style.overflow = 'hidden'
}

export function releaseBodyScrollLock(token: symbol) {
  if (typeof document === 'undefined' || !activeLocks.delete(token)) return

  if (activeLocks.size === 0) {
    document.body.style.overflow = previousOverflow ?? ''
    previousOverflow = null
  }
}

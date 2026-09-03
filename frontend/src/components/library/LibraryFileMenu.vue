<template>
  <Teleport to="body">
    <Transition name="library-file-menu">
      <div
        v-if="show && file"
        ref="menuRef"
        class="library-file-menu"
        :style="positionStyle"
        role="menu"
        :aria-label="t('library.table.actions', { name: file.name })"
        @keydown="onKeydown"
      >
        <button type="button" role="menuitem" tabindex="-1" @click="choose('preview')">
          <Icon name="eye" size="sm" aria-hidden="true" />
          <span>{{ t('library.actions.preview') }}</span>
        </button>
        <button type="button" role="menuitem" tabindex="-1" @click="choose('download')">
          <Icon name="download" size="sm" aria-hidden="true" />
          <span>{{ t('library.actions.download') }}</span>
        </button>
        <div class="library-file-menu__divider"></div>
        <button
          type="button"
          class="library-file-menu__danger"
          role="menuitem"
          tabindex="-1"
          @click="choose('delete')"
        >
          <Icon name="trash" size="sm" aria-hidden="true" />
          <span>{{ t('library.actions.delete') }}</span>
        </button>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch, type CSSProperties } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { LibraryFile } from '@/types/library'

const props = defineProps<{
  show: boolean
  file: LibraryFile | null
  anchor?: HTMLElement | null
}>()

const emit = defineEmits<{
  close: []
  preview: [file: LibraryFile]
  download: [file: LibraryFile]
  delete: [file: LibraryFile]
}>()

const { t } = useI18n()
const menuRef = ref<HTMLElement | null>(null)
const rect = ref<DOMRect | null>(null)

const positionStyle = computed<CSSProperties>(() => {
  const anchor = rect.value
  const width = 156
  if (!anchor) return { top: '12px', left: '12px' }
  const left = Math.max(10, Math.min(anchor.right - width, window.innerWidth - width - 10))
  const opensAbove = window.innerHeight - anchor.bottom < 152 && anchor.top > 152
  return {
    left: `${left}px`,
    top: opensAbove ? `${anchor.top - 140}px` : `${anchor.bottom + 6}px`,
    width: `${width}px`,
  }
})

function controls(): HTMLButtonElement[] {
  return Array.from(menuRef.value?.querySelectorAll<HTMLButtonElement>('[role="menuitem"]') ?? [])
}

function onDocumentPointerDown(event: PointerEvent): void {
  const target = event.target as Node | null
  if (target && (menuRef.value?.contains(target) || props.anchor?.contains(target))) return
  emit('close')
}

function onDocumentFocusIn(event: FocusEvent): void {
  const target = event.target as Node | null
  if (target && (menuRef.value?.contains(target) || props.anchor?.contains(target))) return
  emit('close')
}

watch(() => props.show, async (visible) => {
  if (!visible) {
    document.removeEventListener('pointerdown', onDocumentPointerDown, true)
    document.removeEventListener('focusin', onDocumentFocusIn, true)
    return
  }
  rect.value = props.anchor?.getBoundingClientRect() ?? null
  document.addEventListener('pointerdown', onDocumentPointerDown, true)
  document.addEventListener('focusin', onDocumentFocusIn, true)
  await nextTick()
  controls()[0]?.focus({ preventScroll: true })
})

function onKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape') {
    event.preventDefault()
    emit('close')
    props.anchor?.focus({ preventScroll: true })
    return
  }
  if (!['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) return
  const items = controls()
  if (items.length === 0) return
  event.preventDefault()
  const current = items.indexOf(event.target as HTMLButtonElement)
  const next = event.key === 'Home'
    ? 0
    : event.key === 'End'
      ? items.length - 1
      : (current + (event.key === 'ArrowDown' ? 1 : -1) + items.length) % items.length
  items[next]?.focus()
}

function choose(action: 'preview' | 'download' | 'delete'): void {
  if (!props.file) return
  if (action === 'preview') emit('preview', props.file)
  else if (action === 'download') emit('download', props.file)
  else emit('delete', props.file)
  emit('close')
}

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onDocumentPointerDown, true)
  document.removeEventListener('focusin', onDocumentFocusIn, true)
})
</script>

<style scoped>
.library-file-menu {
  position: fixed;
  z-index: 85;
  border: 1px solid var(--workspace-popover-border);
  border-radius: 12px;
  padding: 5px;
  color: var(--workspace-popover-text);
  background: var(--workspace-popover-surface);
  box-shadow: var(--workspace-popover-shadow);
}

.library-file-menu button {
  display: flex;
  width: 100%;
  min-height: 36px;
  align-items: center;
  gap: 10px;
  border: 0;
  border-radius: 8px;
  padding: 0 9px;
  color: inherit;
  background: transparent;
  font: inherit;
  font-size: 14px;
  text-align: left;
  cursor: pointer;
}

.library-file-menu button:hover,
.library-file-menu button:focus {
  background: var(--workspace-popover-hover);
  outline: none;
}

.library-file-menu__divider {
  height: 1px;
  margin: 4px;
  background: var(--workspace-popover-divider);
}

.library-file-menu button.library-file-menu__danger {
  color: var(--workspace-menu-danger);
}

.library-file-menu-enter-active,
.library-file-menu-leave-active {
  transition: opacity 100ms ease, transform 120ms ease;
  transform-origin: top right;
}

.library-file-menu-enter-from,
.library-file-menu-leave-to {
  opacity: 0;
  transform: scale(0.98) translateY(-3px);
}

@media (prefers-reduced-motion: reduce) {
  .library-file-menu-enter-active,
  .library-file-menu-leave-active { transition: none; }
}
</style>

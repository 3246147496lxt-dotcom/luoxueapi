<template>
  <nav class="ops-workspace-nav" :aria-label="label">
    <div
      class="ops-workspace-nav__tabs"
      :role="tabSemantics ? 'tablist' : undefined"
      :aria-orientation="tabSemantics ? 'horizontal' : undefined"
      :aria-label="label"
    >
      <button
        v-for="(item, index) in items"
        :id="tabId(item.id)"
        :key="item.id"
        type="button"
        :role="tabSemantics ? 'tab' : undefined"
        class="ops-workspace-nav__tab"
        :class="{ 'ops-workspace-nav__tab--active': index === activeIndex }"
        :aria-selected="tabSemantics ? index === activeIndex : undefined"
        :aria-controls="tabSemantics ? panelId(item.id) : undefined"
        :aria-current="!tabSemantics && index === activeIndex ? 'page' : undefined"
        :tabindex="index === activeIndex ? 0 : -1"
        @click="selectItem(item.id)"
        @keydown="handleKeydown($event, index)"
      >
        {{ item.label }}
      </button>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { computed, nextTick } from 'vue'

interface WorkspaceNavItem {
  id: string
  label: string
}

interface Props {
  modelValue: string
  items: WorkspaceNavItem[]
  label: string
  tabSemantics?: boolean
}

interface Emits {
  (event: 'update:modelValue', value: string): void
}

const props = withDefaults(defineProps<Props>(), {
  tabSemantics: true
})
const emit = defineEmits<Emits>()

const activeIndex = computed(() => {
  const selectedIndex = props.items.findIndex((item) => item.id === props.modelValue)
  return selectedIndex >= 0 ? selectedIndex : props.items.length ? 0 : -1
})

function tabId(itemId: string): string {
  return `ops-workspace-tab-${itemId}`
}

function panelId(itemId: string): string {
  return `ops-workspace-panel-${itemId}`
}

function selectItem(itemId: string): void {
  emit('update:modelValue', itemId)
}

function handleKeydown(event: KeyboardEvent, index: number): void {
  const itemCount = props.items.length
  if (!itemCount) return

  let nextIndex: number
  switch (event.key) {
    case 'ArrowLeft':
      nextIndex = (index - 1 + itemCount) % itemCount
      break
    case 'ArrowRight':
      nextIndex = (index + 1) % itemCount
      break
    case 'Home':
      nextIndex = 0
      break
    case 'End':
      nextIndex = itemCount - 1
      break
    default:
      return
  }

  event.preventDefault()
  const tabButtons = (event.currentTarget as HTMLButtonElement)
    .closest('.ops-workspace-nav__tabs')
    ?.querySelectorAll<HTMLButtonElement>('.ops-workspace-nav__tab')

  selectItem(props.items[nextIndex].id)
  nextTick(() => tabButtons?.[nextIndex]?.focus())
}
</script>

<style scoped>
.ops-workspace-nav {
  min-width: 0;
  max-width: 100%;
  overflow-x: auto;
  overscroll-behavior-inline: contain;
  border-bottom: 1px solid var(--lx-clay-border);
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-ui);
  scrollbar-width: thin;
  -webkit-overflow-scrolling: touch;
}

.ops-workspace-nav__tabs {
  display: flex;
  width: max-content;
  min-width: 100%;
  align-items: center;
  gap: 4px;
}

.ops-workspace-nav__tab {
  min-width: 44px;
  min-height: 48px;
  flex: 0 0 auto;
  margin-bottom: -1px;
  padding: 0 16px;
  border: 0;
  border-bottom: 2px solid transparent;
  color: var(--lx-clay-text-secondary);
  background: transparent;
  font: inherit;
  font-size: 12px;
  font-weight: 700;
  line-height: 1.2;
  white-space: nowrap;
  cursor: pointer;
  transition:
    color 180ms ease-out,
    border-color 180ms ease-out;
}

.ops-workspace-nav__tab:hover:not(.ops-workspace-nav__tab--active) {
  color: var(--lx-clay-text);
  border-bottom-color: color-mix(in srgb, var(--lx-clay-text-secondary) 34%, transparent);
}

.ops-workspace-nav__tab--active {
  border-bottom-color: var(--lx-clay-accent);
  color: var(--lx-clay-accent);
}

.ops-workspace-nav__tab:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 34%, transparent);
  outline-offset: -3px;
}

@media (max-width: 640px) {
  .ops-workspace-nav__tabs {
    gap: 14px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .ops-workspace-nav__tab {
    transition: none;
  }
}
</style>

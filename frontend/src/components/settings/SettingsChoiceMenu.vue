<template>
  <div
    ref="rootRef"
    class="settings-choice-menu"
    data-ui-component="settings-choice-menu"
  >
    <button
      :id="triggerId"
      ref="triggerRef"
      type="button"
      class="settings-choice-menu__trigger"
      :class="{ 'settings-choice-menu__trigger--open': isOpen }"
      :aria-label="ariaLabel"
      aria-haspopup="menu"
      :aria-expanded="isOpen"
      :aria-controls="menuId"
      :disabled="disabled"
      :data-testid="testId"
      @click="toggleMenu"
      @keydown="handleTriggerKeydown"
    >
      <span class="settings-choice-menu__value">{{ selectedLabel }}</span>
      <Icon
        name="chevronDown"
        size="xs"
        class="settings-choice-menu__chevron"
        :class="{ 'settings-choice-menu__chevron--open': isOpen }"
        aria-hidden="true"
      />
    </button>

    <Teleport to="body">
      <Transition name="settings-choice-menu">
        <div
          v-if="isOpen"
          :id="menuId"
          ref="menuRef"
          class="settings-choice-menu__popover"
          data-ui-portal="settings-choice-menu"
          role="menu"
          aria-orientation="vertical"
          :aria-labelledby="triggerId"
          :style="popoverStyle"
          @keydown="handleMenuKeydown"
          @pointerdown.stop
        >
          <button
            v-for="(option, index) in options"
            :key="option.value"
            ref="optionRefs"
            type="button"
            class="settings-choice-menu__option"
            :class="{
              'settings-choice-menu__option--selected': option.value === modelValue,
              'settings-choice-menu__option--focused': index === focusedIndex,
            }"
            role="menuitemradio"
            :aria-checked="option.value === modelValue"
            :aria-disabled="option.disabled ? 'true' : undefined"
            :disabled="option.disabled"
            tabindex="-1"
            :data-choice-value="option.value"
            @click="selectOption(option)"
            @focus="focusedIndex = index"
            @mouseenter="focusOption(index, false)"
          >
            <span class="settings-choice-menu__option-label">{{ option.label }}</span>
            <span class="settings-choice-menu__check-slot" aria-hidden="true">
              <Icon
                v-if="option.value === modelValue"
                name="check"
                size="sm"
                :stroke-width="2"
              />
            </span>
          </button>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import {
  computed,
  getCurrentInstance,
  nextTick,
  onBeforeUnmount,
  ref,
  watch,
  type CSSProperties,
} from 'vue'
import Icon from '@/components/icons/Icon.vue'
import {
  isTopModalLayer,
  registerModalLayer,
  unregisterModalLayer,
} from '@/utils/modalStack'

export interface SettingsChoiceOption {
  value: string
  label: string
  disabled?: boolean
}

const props = withDefaults(defineProps<{
  modelValue: string
  options: readonly SettingsChoiceOption[]
  ariaLabel: string
  disabled?: boolean
  testId?: string
}>(), {
  disabled: false,
  testId: undefined,
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
  change: [value: string, option: SettingsChoiceOption]
}>()

const VIEWPORT_GUTTER = 12
const POPOVER_GAP = 6
const POPOVER_WIDTH = 220
const POPOVER_MAX_HEIGHT = 320
const OPTION_HEIGHT = 36
const POPOVER_VERTICAL_PADDING = 12

const instanceId = `settings-choice-${getCurrentInstance()?.uid ?? 'unknown'}`
const triggerId = `${instanceId}-trigger`
const menuId = `${instanceId}-menu`
const layerToken = Symbol(instanceId)

const rootRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLButtonElement | null>(null)
const menuRef = ref<HTMLElement | null>(null)
const optionRefs = ref<HTMLButtonElement[]>([])
const isOpen = ref(false)
const focusedIndex = ref(-1)
const popoverStyle = ref<CSSProperties>({
  position: 'fixed',
  left: `${VIEWPORT_GUTTER}px`,
  top: `${VIEWPORT_GUTTER}px`,
  width: `${POPOVER_WIDTH}px`,
  maxHeight: `${POPOVER_MAX_HEIGHT}px`,
  visibility: 'hidden',
})

let resizeObserver: ResizeObserver | null = null

const selectedOption = computed(
  () => props.options.find((option) => option.value === props.modelValue) ?? null,
)
const selectedLabel = computed(
  () => selectedOption.value?.label ?? props.modelValue,
)

function firstEnabledIndex(): number {
  return props.options.findIndex((option) => !option.disabled)
}

function lastEnabledIndex(): number {
  for (let index = props.options.length - 1; index >= 0; index -= 1) {
    if (!props.options[index]?.disabled) return index
  }
  return -1
}

function nextEnabledIndex(startIndex: number, direction: 1 | -1): number {
  if (props.options.length === 0) return -1

  for (let offset = 0; offset < props.options.length; offset += 1) {
    const index = (
      startIndex
      + direction * offset
      + props.options.length * 2
    ) % props.options.length
    if (!props.options[index]?.disabled) return index
  }

  return -1
}

function scrollFocusedOptionIntoView(index: number) {
  void nextTick(() => {
    optionRefs.value[index]?.scrollIntoView?.({
      block: 'nearest',
      inline: 'nearest',
    })
  })
}

function focusOption(index: number, scroll = true) {
  const option = props.options[index]
  if (!option || option.disabled) return

  focusedIndex.value = index
  optionRefs.value[index]?.focus()
  if (scroll) scrollFocusedOptionIntoView(index)
}

function initialFocusIndex(mode: 'current' | 'first' | 'last'): number {
  if (mode === 'first') return firstEnabledIndex()
  if (mode === 'last') return lastEnabledIndex()

  const selectedIndex = props.options.findIndex(
    (option) => option.value === props.modelValue && !option.disabled,
  )
  return selectedIndex >= 0 ? selectedIndex : firstEnabledIndex()
}

function updatePosition() {
  if (
    !isOpen.value
    || !triggerRef.value
    || !menuRef.value
    || typeof window === 'undefined'
  ) {
    return
  }

  const triggerRect = triggerRef.value.getBoundingClientRect()
  const viewportWidth = window.innerWidth
  const viewportHeight = window.innerHeight
  const width = Math.max(
    0,
    Math.min(POPOVER_WIDTH, viewportWidth - VIEWPORT_GUTTER * 2),
  )
  const naturalHeight = Math.min(
    POPOVER_MAX_HEIGHT,
    menuRef.value.scrollHeight
      || props.options.length * OPTION_HEIGHT + POPOVER_VERTICAL_PADDING,
  )
  const availableBelow = Math.max(
    0,
    viewportHeight - triggerRect.bottom - POPOVER_GAP - VIEWPORT_GUTTER,
  )
  const availableAbove = Math.max(
    0,
    triggerRect.top - POPOVER_GAP - VIEWPORT_GUTTER,
  )
  const placeAbove = availableBelow < naturalHeight && availableAbove > availableBelow
  const availableHeight = placeAbove ? availableAbove : availableBelow
  const maxHeight = Math.max(
    OPTION_HEIGHT,
    Math.min(POPOVER_MAX_HEIGHT, availableHeight),
  )
  const renderedHeight = Math.min(naturalHeight, maxHeight)
  const maxLeft = Math.max(
    VIEWPORT_GUTTER,
    viewportWidth - VIEWPORT_GUTTER - width,
  )
  const left = Math.min(
    Math.max(VIEWPORT_GUTTER, triggerRect.right - width),
    maxLeft,
  )
  const preferredTop = placeAbove
    ? triggerRect.top - POPOVER_GAP - renderedHeight
    : triggerRect.bottom + POPOVER_GAP
  const maxTop = Math.max(
    VIEWPORT_GUTTER,
    viewportHeight - VIEWPORT_GUTTER - renderedHeight,
  )
  const top = Math.min(
    Math.max(VIEWPORT_GUTTER, preferredTop),
    maxTop,
  )

  popoverStyle.value = {
    position: 'fixed',
    left: `${Math.round(left)}px`,
    top: `${Math.round(top)}px`,
    width: `${Math.round(width)}px`,
    maxHeight: `${Math.round(maxHeight)}px`,
    visibility: 'visible',
  }
}

function handleViewportChange() {
  updatePosition()
}

function startTracking() {
  if (typeof window === 'undefined') return
  window.addEventListener('resize', handleViewportChange)
  window.addEventListener('scroll', handleViewportChange, true)
  document.addEventListener('pointerdown', handleDocumentPointerDown, true)
  document.addEventListener('keydown', handleDocumentKeydown)

  if (typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver(updatePosition)
    if (triggerRef.value) resizeObserver.observe(triggerRef.value)
    if (menuRef.value) resizeObserver.observe(menuRef.value)
  }
}

function stopTracking() {
  if (typeof window !== 'undefined') {
    window.removeEventListener('resize', handleViewportChange)
    window.removeEventListener('scroll', handleViewportChange, true)
  }
  if (typeof document !== 'undefined') {
    document.removeEventListener('pointerdown', handleDocumentPointerDown, true)
    document.removeEventListener('keydown', handleDocumentKeydown)
  }
  resizeObserver?.disconnect()
  resizeObserver = null
}

async function openMenu(initialFocus: 'current' | 'first' | 'last' = 'current') {
  if (props.disabled || isOpen.value) return

  isOpen.value = true
  popoverStyle.value = {
    ...popoverStyle.value,
    visibility: 'hidden',
  }
  registerModalLayer(layerToken)
  await nextTick()
  startTracking()
  updatePosition()
  focusOption(initialFocusIndex(initialFocus))
}

function closeMenu(restoreFocus = false) {
  if (!isOpen.value) return

  isOpen.value = false
  focusedIndex.value = -1
  stopTracking()
  unregisterModalLayer(layerToken)

  if (restoreFocus) {
    void nextTick(() => triggerRef.value?.focus())
  }
}

function toggleMenu() {
  if (isOpen.value) {
    closeMenu()
    return
  }
  void openMenu('current')
}

function selectOption(option: SettingsChoiceOption) {
  if (option.disabled) return
  emit('update:modelValue', option.value)
  emit('change', option.value, option)
  closeMenu(true)
}

function handleTriggerKeydown(event: KeyboardEvent) {
  switch (event.key) {
    case 'ArrowDown':
      event.preventDefault()
      void openMenu('first')
      break
    case 'ArrowUp':
      event.preventDefault()
      void openMenu('last')
      break
    case 'Enter':
    case ' ':
      if (isOpen.value) return
      event.preventDefault()
      void openMenu('current')
      break
    case 'Escape':
      if (!isOpen.value || !isTopModalLayer(layerToken)) return
      event.preventDefault()
      event.stopPropagation()
      closeMenu(true)
      break
  }
}

function handleMenuKeydown(event: KeyboardEvent) {
  if (!isTopModalLayer(layerToken)) return

  switch (event.key) {
    case 'ArrowDown': {
      event.preventDefault()
      const index = nextEnabledIndex(focusedIndex.value + 1, 1)
      if (index >= 0) focusOption(index)
      break
    }
    case 'ArrowUp': {
      event.preventDefault()
      const index = nextEnabledIndex(focusedIndex.value - 1, -1)
      if (index >= 0) focusOption(index)
      break
    }
    case 'Home': {
      event.preventDefault()
      const index = firstEnabledIndex()
      if (index >= 0) focusOption(index)
      break
    }
    case 'End': {
      event.preventDefault()
      const index = lastEnabledIndex()
      if (index >= 0) focusOption(index)
      break
    }
    case 'Enter':
    case ' ': {
      event.preventDefault()
      const option = props.options[focusedIndex.value]
      if (option) selectOption(option)
      break
    }
    case 'Escape':
      event.preventDefault()
      event.stopPropagation()
      closeMenu(true)
      break
    case 'Tab':
      event.preventDefault()
      event.stopPropagation()
      closeMenu(true)
      break
  }
}

function handleDocumentPointerDown(event: PointerEvent) {
  const target = event.target
  if (!(target instanceof Node)) return
  if (
    rootRef.value?.contains(target)
    || menuRef.value?.contains(target)
  ) {
    return
  }
  closeMenu()
}

function handleDocumentKeydown(event: KeyboardEvent) {
  if (
    event.key !== 'Escape'
    || !isOpen.value
    || !isTopModalLayer(layerToken)
  ) {
    return
  }

  event.preventDefault()
  event.stopPropagation()
  closeMenu(true)
}

watch(
  () => props.options,
  () => {
    if (!isOpen.value) return
    void nextTick(updatePosition)
  },
)

onBeforeUnmount(() => {
  stopTracking()
  unregisterModalLayer(layerToken)
})
</script>

<style scoped>
.settings-choice-menu {
  min-width: 0;
  flex: 0 1 auto;
}

.settings-choice-menu__trigger {
  display: inline-flex;
  min-width: 118px;
  max-width: min(220px, 46vw);
  height: 36px;
  align-items: center;
  justify-content: flex-end;
  gap: 7px;
  padding: 0 9px 0 11px;
  border: 1px solid transparent;
  border-radius: 8px;
  color: var(--lx-clay-text-secondary);
  background: transparent;
  font-size: 0.875rem;
  font-weight: 560;
  line-height: 1;
  transition:
    color 140ms ease,
    background-color 140ms ease,
    border-color 140ms ease;
}

.settings-choice-menu__trigger:hover,
.settings-choice-menu__trigger--open {
  color: var(--lx-clay-text);
  background: color-mix(in srgb, var(--lx-clay-text) 5%, transparent);
}

.settings-choice-menu__trigger:disabled {
  cursor: not-allowed;
  opacity: 0.52;
}

.settings-choice-menu__trigger:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--lx-clay-accent) 70%, transparent);
  outline-offset: 2px;
}

.settings-choice-menu__value {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.settings-choice-menu__chevron {
  flex: 0 0 auto;
  transition: transform 140ms ease;
}

.settings-choice-menu__chevron--open {
  transform: rotate(180deg);
}

.settings-choice-menu__popover {
  z-index: 95;
  overflow-x: hidden;
  overflow-y: auto;
  padding: 6px 0;
  border: 0;
  border-radius: 16px;
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface);
  box-shadow:
    0 8px 12px rgb(0 0 0 / 0.08),
    0 0 1px rgb(0 0 0 / 0.62);
  overscroll-behavior: contain;
  scrollbar-width: thin;
}

.settings-choice-menu__option {
  display: grid;
  width: calc(100% - 12px);
  height: 36px;
  grid-template-columns: minmax(0, 1fr) 18px;
  align-items: center;
  gap: 9px;
  margin: 0 6px;
  padding: 0 9px 0 11px;
  border-radius: 10px;
  color: var(--lx-clay-text-secondary);
  background: transparent;
  font-size: 0.875rem;
  font-weight: 540;
  line-height: 1.25;
  text-align: left;
  transition:
    color 120ms ease,
    background-color 120ms ease;
}

.settings-choice-menu__option:hover,
.settings-choice-menu__option--focused {
  color: var(--lx-clay-text);
  background: var(--lx-clay-hover);
}

.settings-choice-menu__option--selected {
  color: var(--lx-clay-text);
  background: var(--lx-clay-selected);
}

.settings-choice-menu__option:disabled {
  cursor: not-allowed;
  opacity: 0.42;
}

.settings-choice-menu__option:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--lx-clay-accent) 68%, transparent);
  outline-offset: -2px;
}

.settings-choice-menu__option-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.settings-choice-menu__check-slot {
  display: inline-flex;
  width: 18px;
  height: 18px;
  align-items: center;
  justify-content: center;
}

:global(html.dark) .settings-choice-menu__popover {
  box-shadow:
    0 0 0 1px color-mix(in srgb, var(--lx-clay-text) 14%, transparent),
    0 18px 46px rgb(0 0 0 / 0.42),
    0 3px 12px rgb(0 0 0 / 0.28);
}

.settings-choice-menu-enter-active,
.settings-choice-menu-leave-active {
  transition:
    opacity 120ms ease,
    transform 140ms cubic-bezier(0.16, 1, 0.3, 1);
  transform-origin: top right;
}

.settings-choice-menu-enter-from,
.settings-choice-menu-leave-to {
  opacity: 0;
  transform: translateY(-3px) scale(0.98);
}

@media (max-width: 480px) {
  .settings-choice-menu__trigger {
    min-width: 104px;
    max-width: 44vw;
  }
}

@media (prefers-reduced-motion: reduce) {
  .settings-choice-menu__trigger,
  .settings-choice-menu__chevron,
  .settings-choice-menu__option,
  .settings-choice-menu-enter-active,
  .settings-choice-menu-leave-active {
    transition-duration: 0.01ms !important;
  }
}
</style>

<template>
  <span
    ref="hostRef"
    class="chat-control-tooltip-host"
    :class="{
      'chat-control-tooltip-host--contents': contents,
      'chat-control-tooltip-host--suppressed': !enabled || dismissed,
    }"
    @pointerenter="onHostPointerEnter"
    @pointerleave="onHostPointerLeave"
    @focusin="onFocusIn"
    @focusout="onFocusOut"
    @keydown.esc="dismissTooltip"
  >
    <slot :tooltip-id="tooltipId"></slot>

    <Teleport to="body">
      <span
        :id="tooltipId"
        ref="tooltipRef"
        class="chat-control-tooltip"
        :class="[
          `chat-control-tooltip--${placement}`,
          {
            'chat-control-tooltip--wide-gap': wideGap,
            'chat-control-tooltip--visible': tooltipVisible,
          },
        ]"
        :style="tooltipStyle"
        :aria-hidden="enabled && accessible ? undefined : 'true'"
        role="tooltip"
        data-test="chat-control-tooltip"
        data-ui-portal="chat-control-tooltip"
        @pointerenter="onTooltipPointerEnter"
        @pointerleave="onTooltipPointerLeave"
      >
        <span class="chat-control-tooltip__surface">
          <span class="chat-control-tooltip__content">
            <span class="chat-control-tooltip__label">{{ label }}</span>
            <span
              v-if="shortcut.length"
              class="chat-control-tooltip__shortcut"
              aria-hidden="true"
            >
              <kbd v-for="key in shortcut" :key="key"><span>{{ key }}</span></kbd>
            </span>
          </span>
        </span>
      </span>
    </Teleport>
  </span>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, useId, watch } from 'vue'

const props = withDefaults(defineProps<{
  label: string
  shortcut?: string[]
  enabled?: boolean
  accessible?: boolean
  contents?: boolean
  wideGap?: boolean
}>(), {
  shortcut: () => [],
  enabled: true,
  accessible: true,
  contents: false,
  wideGap: false,
})

const tooltipId = `chat-control-tooltip-${useId()}`
const hostRef = ref<HTMLElement | null>(null)
const tooltipRef = ref<HTMLElement | null>(null)
const placement = ref<'above' | 'below'>('below')
const tooltipLeft = ref(0)
const tooltipTop = ref(0)
const hostHovered = ref(false)
const tooltipHovered = ref(false)
const focusVisibleWithin = ref(false)
const dismissed = ref(false)

const tooltipVisible = computed(() => (
  props.enabled
  && !dismissed.value
  && (hostHovered.value || tooltipHovered.value || focusVisibleWithin.value)
))

const tooltipStyle = computed(() => ({
  left: `${tooltipLeft.value}px`,
  top: `${tooltipTop.value}px`,
}))

function anchorElement(): HTMLElement | null {
  return hostRef.value?.querySelector<HTMLElement>('[data-chat-control-anchor], button') ?? null
}

async function updatePlacement(): Promise<void> {
  if (!props.enabled) return
  await nextTick()
  const anchor = anchorElement()
  const tooltip = tooltipRef.value
  if (!anchor || !tooltip) return

  const anchorRect = anchor.getBoundingClientRect()
  const tooltipRect = tooltip.getBoundingClientRect()
  const tooltipWidth = tooltipRect.width
  const tooltipHeight = tooltipRect.height || 36
  const viewportPadding = 12

  placement.value = anchorRect.bottom + tooltipHeight <= window.innerHeight - viewportPadding
    ? 'below'
    : 'above'

  if (tooltipWidth <= 0) return

  const desiredLeft = anchorRect.left + (anchorRect.width - tooltipWidth) / 2
  const clampedLeft = Math.min(
    Math.max(desiredLeft, viewportPadding),
    Math.max(viewportPadding, window.innerWidth - viewportPadding - tooltipWidth),
  )
  const desiredTop = placement.value === 'below'
    ? anchorRect.bottom
    : anchorRect.top - tooltipHeight
  const clampedTop = Math.min(
    Math.max(desiredTop, viewportPadding),
    Math.max(viewportPadding, window.innerHeight - viewportPadding - tooltipHeight),
  )

  tooltipLeft.value = clampedLeft
  tooltipTop.value = clampedTop
}

function prepareTooltip(): void {
  dismissed.value = false
  void updatePlacement()
}

function finePointerCanHover(): boolean {
  return typeof window.matchMedia !== 'function'
    || window.matchMedia('(hover: hover) and (pointer: fine)').matches
}

function onHostPointerEnter(): void {
  if (!finePointerCanHover()) return
  hostHovered.value = true
  prepareTooltip()
}

function onHostPointerLeave(event: PointerEvent): void {
  hostHovered.value = false
  const nextTarget = event.relatedTarget as Node | null
  if (nextTarget && tooltipRef.value?.contains(nextTarget)) tooltipHovered.value = true

  const activeElement = document.activeElement
  if (!activeElement || !hostRef.value?.contains(activeElement)) dismissed.value = false
}

function onTooltipPointerEnter(): void {
  if (!finePointerCanHover()) return
  tooltipHovered.value = true
}

function onTooltipPointerLeave(event: PointerEvent): void {
  tooltipHovered.value = false
  const nextTarget = event.relatedTarget as Node | null
  if (nextTarget && hostRef.value?.contains(nextTarget)) hostHovered.value = true

  const activeElement = document.activeElement
  if (!activeElement || !hostRef.value?.contains(activeElement)) dismissed.value = false
}

function dismissTooltip(): void {
  dismissed.value = true
}

function onFocusOut(event: FocusEvent): void {
  const nextTarget = event.relatedTarget as Node | null
  if (nextTarget && hostRef.value?.contains(nextTarget)) return

  focusVisibleWithin.value = false
  if (!hostHovered.value && !tooltipHovered.value) dismissed.value = false
}

function updateActiveTooltip(): void {
  const host = hostRef.value
  if (!host) return

  hostHovered.value = finePointerCanHover() && host.matches(':hover')
  focusVisibleWithin.value = !!host.querySelector(':focus-visible')
  if (!tooltipVisible.value) return
  void updatePlacement()
}

async function onFocusIn(): Promise<void> {
  dismissed.value = false
  await nextTick()
  focusVisibleWithin.value = !!hostRef.value?.querySelector(':focus-visible')
  if (focusVisibleWithin.value) void updatePlacement()
}

watch(
  () => [props.label, props.shortcut.join('\u0000'), props.enabled],
  () => {
    if (!props.enabled) tooltipHovered.value = false
    void nextTick(updateActiveTooltip)
  },
)

onMounted(() => {
  window.addEventListener('resize', updateActiveTooltip)
  window.addEventListener('scroll', updateActiveTooltip, true)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', updateActiveTooltip)
  window.removeEventListener('scroll', updateActiveTooltip, true)
})
</script>

<style scoped>
.chat-control-tooltip-host {
  position: relative;
  display: inline-grid;
  min-width: 0;
  flex: none;
}

.chat-control-tooltip-host--contents {
  display: contents;
}

.chat-control-tooltip {
  position: fixed;
  z-index: 50;
  display: flex;
  width: max-content;
  max-width: calc(100vw - 24px);
  justify-content: center;
  color: #fff;
  opacity: 0;
  pointer-events: none;
  visibility: hidden;
  transition:
    opacity 120ms ease,
    visibility 0s linear 120ms;
  user-select: none;
}

.chat-control-tooltip--below {
  padding-top: 6px;
}

.chat-control-tooltip--above {
  padding-bottom: 6px;
}

.chat-control-tooltip__surface {
  display: flex;
  max-width: 100%;
  border: 1px solid rgb(255 255 255 / 5%);
  border-radius: 999px;
  padding: 5px 12px;
  overflow: hidden;
  background: #1b1b1b;
  box-shadow: 0 8px 18px rgb(15 23 42 / 20%);
}

.chat-control-tooltip__content {
  display: inline-flex;
  max-width: 100%;
  align-items: center;
  gap: 6px;
  color: #fff;
  font-size: 14px;
  font-weight: 600;
  letter-spacing: -0.15px;
  line-height: 18px;
  text-align: center;
  white-space: pre-wrap;
}

.chat-control-tooltip--wide-gap .chat-control-tooltip__content {
  gap: 8px;
}

.chat-control-tooltip--visible {
  opacity: 1;
  pointer-events: auto;
  visibility: visible;
  transition-delay: 250ms, 250ms;
}

.chat-control-tooltip__label {
  min-width: 0;
  overflow-wrap: anywhere;
}

.chat-control-tooltip__shortcut {
  display: inline-flex;
  height: 18px;
  align-items: center;
  flex: none;
  border-radius: 999px;
  padding-inline: 6px;
  color: #cdcdcd;
  background: rgb(255 255 255 / 25%);
  font: inherit;
}

.chat-control-tooltip__shortcut kbd {
  display: flex;
  min-width: 1em;
  justify-content: center;
  color: inherit;
  font: inherit;
}

@media (prefers-reduced-motion: reduce) {
  .chat-control-tooltip {
    transition: none;
  }
}
</style>

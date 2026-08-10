<template>
  <div class="chat-model-settings">
    <ChatControlTooltip
      v-slot="{ tooltipId }"
      :label="t('chat.settings.tooltip')"
      :shortcut="['⌃', '⇧', 'M']"
      :enabled="!open && !disabled"
      contents
    >
      <span class="chat-model-settings__sizer" aria-hidden="true">
        <span class="chat-model-settings__sizer-selection">
          <strong
            v-if="selectedTriggerModelLabel"
            class="chat-model-settings__trigger-model"
          >{{ selectedTriggerModelLabel }}</strong>
          <strong
            class="chat-model-settings__trigger-effort"
            :class="{ 'chat-model-settings__trigger-effort--solo': !selectedTriggerModelLabel }"
          >{{ selectedTriggerValueLabel }}</strong>
        </span>
      </span>
      <button
        ref="triggerRef"
        type="button"
        class="chat-model-settings__trigger"
        :class="{ 'chat-model-settings__trigger--open': open }"
        :disabled="disabled"
        :aria-expanded="open"
        :aria-describedby="open || disabled ? undefined : tooltipId"
        :data-state="open ? 'open' : 'closed'"
        aria-haspopup="menu"
        :aria-keyshortcuts="disabled ? undefined : 'Control+Shift+M'"
        :aria-controls="panelId"
        :aria-label="t('chat.settings.label', {
          model: selectedModelLabel,
          effort: selectedEffortLabel,
        })"
        data-chat-control-anchor
        data-test="chat-model-settings-trigger"
        @pointerdown="keyboardFocusVisible = false"
        @keydown="keyboardFocusVisible = true"
        @selectstart.prevent
        @dragstart.prevent
        @click="toggleMenu"
        @transitionend="onTriggerTransitionEnd"
      >
        <span class="chat-model-settings__selection" aria-hidden="true">
          <strong
            v-if="selectedTriggerModelLabel"
            class="chat-model-settings__trigger-model"
            data-test="chat-model-settings-trigger-model"
          >{{ selectedTriggerModelLabel }}</strong>
          <strong
            class="chat-model-settings__trigger-effort"
            :class="{ 'chat-model-settings__trigger-effort--solo': !selectedTriggerModelLabel }"
            data-test="chat-model-settings-trigger-value"
          >{{ selectedTriggerValueLabel }}</strong>
        </span>
        <Icon
          name="chevronDown"
          size="xs"
          class="chat-model-settings__chevron"
          :class="{ 'chat-model-settings__chevron--open': open }"
        />
      </button>
    </ChatControlTooltip>

    <Teleport to="body">
      <Transition name="chat-settings-popover">
        <div
          v-if="open"
          :id="panelId"
          ref="panelRef"
          class="chat-model-settings-popover"
          :class="{ 'chat-model-settings-popover--compact': isCompact }"
          :style="panelStyle"
          data-ui-portal="chat-model-settings"
          data-test="chat-model-settings-popover"
          @pointerdown.capture="keyboardFocusVisible = false"
          @selectstart.prevent
          @dragstart.prevent
          @keydown="onPanelKeydown"
        >
          <div
            ref="rootSurfaceRef"
            class="chat-model-settings-popover__surface chat-model-settings-popover__surface--root"
            :class="{
              'chat-model-settings-popover__surface--advanced': showsReasoningSlider && advancedExpanded,
              'chat-model-settings-popover__surface--plain': !showsReasoningSlider,
            }"
            role="menu"
            :aria-label="t('chat.settings.menuLabel')"
            data-test="chat-settings-root-panel"
          >
            <div class="chat-model-settings-popover__root-content" role="presentation">
              <Transition name="chat-settings-advanced-shift">
                <button
                  v-if="showsReasoningSlider && !advancedExpanded"
                  type="button"
                  class="chat-model-settings-popover__advanced"
                  :class="{
                    'chat-model-settings-popover__advanced--keyboard-focus': keyboardFocusVisible,
                  }"
                  role="menuitem"
                  :aria-expanded="false"
                  :aria-controls="advancedContentId"
                  data-test="chat-settings-advanced-toggle"
                  @click="toggleAdvanced"
                >
                  <span>{{ t('chat.settings.advanced') }}</span>
                  <Icon name="chevronRight" size="xs" aria-hidden="true" />
                </button>
              </Transition>

              <Transition name="chat-settings-disclosure">
                <div
                  v-if="showsReasoningSlider && !advancedExpanded"
                  class="chat-model-settings-popover__capability"
                >
                  <div
                    class="chat-model-settings-popover__range-wrap"
                    :class="{
                      'chat-model-settings-popover__range-wrap--dragging': rangeDragging,
                      'chat-model-settings-popover__range-wrap--settling': rangeSettling,
                      'chat-model-settings-popover__range-wrap--keyboard-focus': rangeFocused && keyboardFocusVisible,
                      'chat-model-settings-popover__range-wrap--thumb-nearby': nearbyReasoningIndex === visualReasoningIndex,
                      'chat-model-settings-popover__range-wrap--maximum': isMaximumReasoning,
                    }"
                    :style="capabilityRangeStyle"
                    :data-max="isMaximumReasoning ? 'true' : undefined"
                    data-test="chat-settings-capability-visual"
                  >
                    <span class="chat-model-settings-popover__range-rail" aria-hidden="true">
                      <span class="chat-model-settings-popover__range-fill" />
                      <span class="chat-model-settings-popover__max-effects-viewport">
                        <Transition name="chat-reasoning-max-effects">
                          <span
                            v-if="isMaximumReasoning"
                            :key="maximumEntryGeneration ?? 'stable'"
                            class="chat-model-settings-popover__max-effects"
                            :class="{
                              'chat-model-settings-popover__max-effects--entering': maximumEntranceActive,
                            }"
                            :data-entry-token="maximumEntryGeneration ?? undefined"
                            data-test="chat-settings-maximum-effects"
                          >
                            <span
                              class="chat-model-settings-popover__max-fill"
                              :data-entry-token="maximumEntryGeneration ?? undefined"
                              @animationend="onMaximumRevealEnd"
                              @animationcancel="onMaximumRevealEnd"
                            >
                              <ReasoningMaxCanvas data-test="chat-settings-maximum-canvas" />
                            </span>
                            <span class="chat-model-settings-popover__max-track-particles">
                              <span
                                v-for="particleIndex in MAX_TRACK_PARTICLE_COUNT"
                                :key="particleIndex"
                              />
                            </span>
                          </span>
                        </Transition>
                      </span>
                      <span class="chat-model-settings-popover__range-ticks">
                        <span
                          v-for="(stop, index) in reasoningStops"
                          :key="stop.value"
                          :class="{
                            'chat-model-settings-popover__range-tick--active': index <= visualReasoningIndex,
                            'chat-model-settings-popover__range-tick--nearby': nearbyReasoningIndex === index
                              && index !== visualReasoningIndex,
                          }"
                          :style="{ left: stop.position }"
                          :data-value="stop.value"
                        />
                      </span>
                      <span class="chat-model-settings-popover__range-thumb-track">
                        <span
                          v-if="isMaximumReasoning && maxBurstToken !== null"
                          :key="maxBurstToken"
                          class="chat-model-settings-popover__max-burst"
                          :data-entry-token="maxBurstToken"
                          aria-hidden="true"
                          @animationend.self="onMaximumBurstLifecycleEnd"
                          @animationcancel.self="onMaximumBurstLifecycleEnd"
                        >
                          <span
                            v-for="particleIndex in MAX_BURST_PARTICLE_COUNT"
                            :key="particleIndex"
                          />
                        </span>
                        <span
                          class="chat-model-settings-popover__range-thumb"
                          @animationend="onThumbSettleAnimationEnd"
                          @animationcancel="onThumbSettleAnimationEnd"
                        />
                      </span>
                    </span>
                    <input
                      class="chat-model-settings-popover__range"
                      type="range"
                      min="0"
                      :max="reasoningValues.length - 1"
                      step="1"
                      :value="visualReasoningIndex"
                      :aria-label="t('chat.settings.reasoning')"
                      :aria-valuetext="visualEffortLabel"
                      dir="ltr"
                      data-test="chat-settings-capability-slider"
                      @input="onCapabilityInput"
                      @pointerdown="onRangePointerDown"
                      @pointermove="onRangePointerMove"
                      @pointerup="finishRangeInteraction"
                      @pointercancel="onRangePointerCancel"
                      @lostpointercapture="onRangeLostPointerCapture"
                      @pointerleave="clearRangeProximity"
                      @focus="rangeFocused = true"
                      @blur="onRangeBlur"
                    >
                  </div>
                </div>
              </Transition>

              <Transition name="chat-settings-disclosure">
                <div
                  v-if="!showsReasoningSlider || advancedExpanded"
                  :id="advancedContentId"
                  class="chat-model-settings-popover__rows"
                >
                  <button
                    type="button"
                    class="chat-model-settings-popover__row"
                    role="menuitem"
                    aria-haspopup="menu"
                    :aria-label="`${t('chat.settings.model')} ${selectedModelLabel}`"
                    :aria-expanded="pane === 'models'"
                    :data-state="pane === 'models' ? 'open' : 'closed'"
                    :aria-controls="submenuId"
                    data-pane-target="models"
                    data-test="chat-settings-model-menu"
                    @pointerenter="onPanePointerEnter($event, 'models')"
                    @pointerleave="onPanePointerLeave($event, 'models')"
                    @click="togglePane('models')"
                  >
                    <span class="chat-model-settings-popover__row-copy">
                      <small>{{ t('chat.settings.model') }}</small>
                    </span>
                    <span class="chat-model-settings-popover__row-trailing">
                      <strong>{{ selectedMenuModelLabel }}</strong>
                      <Icon name="chevronRight" size="sm" aria-hidden="true" />
                    </span>
                  </button>

                  <button
                    type="button"
                    class="chat-model-settings-popover__row"
                    role="menuitem"
                    aria-haspopup="menu"
                    :aria-label="`${t('chat.settings.reasoning')} ${selectedEffortLabel}`"
                    :aria-expanded="pane === 'reasoning'"
                    :data-state="pane === 'reasoning' ? 'open' : 'closed'"
                    :aria-controls="submenuId"
                    data-pane-target="reasoning"
                    data-test="chat-settings-reasoning-menu"
                    @pointerenter="onPanePointerEnter($event, 'reasoning')"
                    @pointerleave="onPanePointerLeave($event, 'reasoning')"
                    @click="togglePane('reasoning')"
                  >
                    <span class="chat-model-settings-popover__row-copy">
                      <small>{{ t('chat.settings.reasoning') }}</small>
                    </span>
                    <span class="chat-model-settings-popover__row-trailing">
                      <strong>{{ selectedEffortLabel }}</strong>
                      <Icon name="chevronRight" size="sm" aria-hidden="true" />
                    </span>
                  </button>
                </div>
              </Transition>

              <span
                v-if="showsReasoningSlider && advancedExpanded"
                class="chat-model-settings-popover__divider"
                aria-hidden="true"
              />

              <Transition name="chat-settings-advanced-shift">
                <button
                  v-if="showsReasoningSlider && advancedExpanded"
                  type="button"
                  class="chat-model-settings-popover__advanced chat-model-settings-popover__advanced--open"
                  :class="{
                    'chat-model-settings-popover__advanced--keyboard-focus': keyboardFocusVisible,
                  }"
                  role="menuitem"
                  :aria-expanded="true"
                  :aria-controls="advancedContentId"
                  data-test="chat-settings-advanced-toggle"
                  @click="toggleAdvanced"
                >
                  <span>{{ t('chat.settings.advanced') }}</span>
                  <Icon name="chevronRight" size="xs" aria-hidden="true" />
                </button>
              </Transition>
            </div>
          </div>

          <Transition name="chat-settings-submenu">
            <div
              v-if="pane !== 'root'"
              :id="submenuId"
              ref="submenuSurfaceRef"
              class="chat-model-settings-popover__surface chat-model-settings-popover__submenu"
              :class="[
                `chat-model-settings-popover__submenu--${submenuPlacement}`,
                `chat-model-settings-popover__submenu--${pane}`,
                `chat-model-settings-popover__submenu--${panelPlacement}`,
                { 'chat-model-settings-popover__submenu--compact': isCompact },
              ]"
              role="menu"
              :aria-label="pane === 'models' ? t('chat.settings.model') : t('chat.settings.reasoning')"
              data-test="chat-settings-submenu"
              @pointerenter="onSubmenuPointerEnter"
              @pointerleave="onSubmenuPointerLeave"
              @focusin="onSubmenuFocusIn"
            >
              <div
                v-if="pane === 'models'"
                class="chat-model-settings-popover__options"
                role="group"
                :aria-label="t('chat.settings.model')"
                data-test="chat-settings-model-options"
              >
                <button
                  v-for="option in modelOptions"
                  :key="option.value"
                  type="button"
                  class="chat-model-settings-popover__option"
                  :class="{ 'chat-model-settings-popover__option--selected': option.value === modelValue }"
                  role="menuitemradio"
                  :aria-checked="option.value === modelValue"
                  :data-value="option.value"
                  @click="selectModel(option.value)"
                >
                  <strong>{{ option.label }}</strong>
                  <span class="chat-model-settings-popover__check-slot" aria-hidden="true">
                    <Icon
                      v-if="option.value === modelValue"
                      name="chatCheck"
                      size="sm"
                      class="chat-model-settings-popover__check"
                    />
                  </span>
                </button>
              </div>

              <div
                v-else
                class="chat-model-settings-popover__options"
                role="group"
                :aria-label="t('chat.settings.reasoning')"
              >
                <button
                  v-for="option in reasoningOptions"
                  :key="option.value"
                  type="button"
                  class="chat-model-settings-popover__option"
                  :class="{ 'chat-model-settings-popover__option--selected': option.value === reasoningEffort }"
                  role="menuitemradio"
                  :aria-checked="option.value === reasoningEffort"
                  :data-value="option.value"
                  @click="selectReasoning(option.value)"
                >
                  <strong>{{ option.label }}</strong>
                  <span class="chat-model-settings-popover__check-slot" aria-hidden="true">
                    <Icon
                      v-if="option.value === reasoningEffort"
                      name="chatCheck"
                      size="sm"
                      class="chat-model-settings-popover__check"
                    />
                  </span>
                </button>
              </div>
            </div>
          </Transition>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
  type CSSProperties,
} from 'vue'
import { useI18n } from 'vue-i18n'
import ChatControlTooltip from '@/components/chat/ChatControlTooltip.vue'
import { chatControlShortcutIsInScope } from '@/components/chat/chatControlShortcut'
import Icon from '@/components/icons/Icon.vue'
import ReasoningMaxCanvas from '@/components/chat/ReasoningMaxCanvas.vue'
import type { ChatReasoningEffort } from '@/types/chat'

export interface ChatModelSettingsOption {
  value: string
  label: string
  description?: string
  recommended?: boolean
  supportsReasoningSlider?: boolean
}

type SettingsPane = 'root' | 'models' | 'reasoning'
type SubmenuPane = Exclude<SettingsPane, 'root'>
type PaneActivation = 'none' | 'hover' | 'persistent'
type SupportedReasoningEffort = Extract<ChatReasoningEffort, 'low' | 'medium' | 'high' | 'xhigh'>
type PanelPlacement = 'above' | 'below'
type SubmenuPlacement = 'left' | 'right'

const REASONING_VALUES: readonly SupportedReasoningEffort[] = [
  'low',
  'medium',
  'high',
  'xhigh',
]
const REASONING_SLIDER_RAIL_WIDTH = 196
const REASONING_SLIDER_THUMB_SIZE = 28
const REASONING_SLIDER_STOP_INSET = 13
const REASONING_SLIDER_TICK_SIZE = 4
const REASONING_SLIDER_TICK_HIT_RADIUS = 8
const REASONING_SLIDER_THUMB_HIT_RADIUS = 17
const MAX_TRACK_PARTICLE_COUNT = 14
const MAX_BURST_PARTICLE_COUNT = 16
const ROOT_PANEL_WIDTH = 224
const OPEN_TRIGGER_WIDTH = 164
const DESKTOP_SUBMENU_RESERVE_WIDTH = 170
const SUBMENU_OVERLAP = 8
const VIEWPORT_PADDING = 12
const POPOVER_TRIGGER_GAP = 4
const SUBMENU_HOVER_OPEN_DELAY_MS = 100
const SUBMENU_HOVER_CLOSE_DELAY_MS = 150
const EFFORT_ONLY_TRIGGER_MODEL_ID = 'gpt-5.6-sol'

const props = withDefaults(defineProps<{
  modelValue: string
  reasoningEffort: SupportedReasoningEffort
  modelOptions: ChatModelSettingsOption[]
  disabled?: boolean
  loading?: boolean
}>(), {
  disabled: false,
  loading: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
  'update:reasoningEffort': [value: SupportedReasoningEffort]
}>()

const { t } = useI18n()
const panelId = `chat-model-settings-${Math.random().toString(36).slice(2, 9)}`
const submenuId = `${panelId}-submenu`
const advancedContentId = `${panelId}-advanced`
const reasoningValues = REASONING_VALUES
const open = ref(false)
const pane = ref<SettingsPane>('root')
const advancedExpanded = ref(false)
const lastSubmenu = ref<SubmenuPane>('models')
const panelStyle = ref<CSSProperties>({})
const panelPlacement = ref<PanelPlacement>('above')
const submenuPlacement = ref<SubmenuPlacement>('right')
const isCompact = ref(false)
const visualReasoningIndex = ref(Math.max(0, REASONING_VALUES.indexOf(props.reasoningEffort)))
const rangeDragging = ref(false)
const rangeSettling = ref(false)
const rangeFocused = ref(false)
const nearbyReasoningIndex = ref<number | null>(null)
const maximumEntranceActive = ref(false)
const maximumEntryGeneration = ref<number | null>(null)
const maxBurstToken = ref<number | null>(null)
const keyboardFocusVisible = ref(false)
let rangePointerId: number | null = null
let nextMaxBurstToken = 0
let paneActivation: PaneActivation = 'none'
let paneHoverOpenTimer: ReturnType<typeof setTimeout> | null = null
let paneHoverCloseTimer: ReturnType<typeof setTimeout> | null = null
let paneRequestId = 0
const triggerRef = ref<HTMLButtonElement | null>(null)
const panelRef = ref<HTMLElement | null>(null)
const rootSurfaceRef = ref<HTMLElement | null>(null)
const submenuSurfaceRef = ref<HTMLElement | null>(null)

const selectedModel = computed(() => (
  props.modelOptions.find((option) => option.value === props.modelValue) ?? null
))
const showsReasoningSlider = computed(() => (
  selectedModel.value !== null
))
const selectedModelLabel = computed(() => {
  if (props.loading) return t('chat.models.loading')
  return selectedModel.value?.label || t('chat.models.select')
})
const usesEffortOnlyTrigger = computed(() => (
  selectedModel.value?.value.trim().toLowerCase() === EFFORT_ONLY_TRIGGER_MODEL_ID
))
const selectedMenuModelLabel = computed(() => (
  usesEffortOnlyTrigger.value
    ? selectedModelLabel.value
    : compactModelLabel(selectedModelLabel.value)
))
const selectedTriggerModelLabel = computed(() => {
  if (!selectedModel.value) return ''
  if (usesEffortOnlyTrigger.value) return ''
  return selectedMenuModelLabel.value
})
const selectedTriggerValueLabel = computed(() => (
  selectedModel.value ? selectedEffortLabel.value : selectedModelLabel.value
))
const reasoningOptions = computed<Array<{
  value: SupportedReasoningEffort
  label: string
}>>(() => [
  { value: 'low', label: t('chat.settings.reasoningLevels.low') },
  { value: 'medium', label: t('chat.settings.reasoningLevels.medium') },
  { value: 'high', label: t('chat.settings.reasoningLevels.high') },
  { value: 'xhigh', label: t('chat.settings.reasoningLevels.xhigh') },
])
const selectedEffortLabel = computed(() => (
  reasoningOptions.value.find((option) => option.value === props.reasoningEffort)?.label
  || t('chat.settings.reasoningLevels.low')
))
const visualEffortLabel = computed(() => (
  reasoningOptions.value[visualReasoningIndex.value]?.label
  || t('chat.settings.reasoningLevels.low')
))
const isMaximumReasoning = computed(() => (
  visualReasoningIndex.value === REASONING_VALUES.length - 1
))
const selectedReasoningIndex = computed(() => {
  const index = REASONING_VALUES.indexOf(props.reasoningEffort)
  return index >= 0 ? index : 0
})
const reasoningStops = REASONING_VALUES.map((value, index) => {
  const progress = index / Math.max(1, REASONING_VALUES.length - 1)
  const position = REASONING_SLIDER_STOP_INSET
    + progress * (REASONING_SLIDER_RAIL_WIDTH - (REASONING_SLIDER_STOP_INSET * 2))
  return {
    value,
    offset: position,
    position: `${position}px`,
  }
})
const capabilityRangeStyle = computed<CSSProperties>(() => {
  const index = Math.min(
    REASONING_VALUES.length - 1,
    Math.max(0, visualReasoningIndex.value),
  )
  const progress = index / Math.max(1, REASONING_VALUES.length - 1)
  const position = REASONING_SLIDER_STOP_INSET
    + progress * (REASONING_SLIDER_RAIL_WIDTH - (REASONING_SLIDER_STOP_INSET * 2))
  const fillWidth = index === 0
    ? 0
    : index === REASONING_VALUES.length - 1
      ? REASONING_SLIDER_RAIL_WIDTH
      : position
  return {
    '--reasoning-progress': String(progress),
    '--reasoning-position': `${position}px`,
    '--reasoning-fill-width': `${fillWidth}px`,
    '--reasoning-fill-inset': `${REASONING_SLIDER_RAIL_WIDTH - fillWidth}px`,
    '--reasoning-thumb-size': `${REASONING_SLIDER_THUMB_SIZE}px`,
    '--reasoning-tick-size': `${REASONING_SLIDER_TICK_SIZE}px`,
  } as CSSProperties
})

watch(() => props.disabled, (disabled) => {
  if (disabled) closeMenu(false)
})

watch(() => props.reasoningEffort, () => {
  const nextIndex = selectedReasoningIndex.value
  if (nextIndex !== visualReasoningIndex.value) resetMaximumEntryMotion()
  visualReasoningIndex.value = nextIndex
})

watch(() => props.modelValue, () => {
  resetMaximumEntryMotion()
})

function compactModelLabel(label: string) {
  const trimmed = label.trim()
  const compact = trimmed.replace(/^GPT[-\s]+(?=\d)/i, '')
  return compact || trimmed
}

function toggleMenu() {
  if (props.disabled) return
  if (open.value) {
    closeMenu(false)
    return
  }
  resetPaneInteraction()
  resetMaximumEntryMotion()
  clearRangeProximity()
  open.value = true
  pane.value = 'root'
  advancedExpanded.value = false
  updateCompactMode()
  const requestId = paneRequestId
  void nextTick(() => {
    if (!open.value || pane.value !== 'root' || requestId !== paneRequestId) return
    updatePanelPosition()
    focusFirstActiveControl()
  })
}

function onTriggerTransitionEnd(event: TransitionEvent) {
  if (!open.value || event.propertyName !== 'width') return
  updatePanelPosition()
}

function closeMenu(restoreFocus = true) {
  resetPaneInteraction()
  resetMaximumEntryMotion()
  if (!open.value) return
  resetRangeInteraction()
  open.value = false
  clearRangeProximity()
  pane.value = 'root'
  advancedExpanded.value = false
  const requestId = paneRequestId
  if (restoreFocus) {
    void nextTick(() => {
      if (open.value || requestId !== paneRequestId) return
      triggerRef.value?.focus()
    })
  }
}

function openPane(
  nextPane: SubmenuPane,
  {
    activation = 'persistent',
    focusOption = true,
  }: {
    activation?: Exclude<PaneActivation, 'none'>
    focusOption?: boolean
  } = {},
) {
  clearPaneHoverOpen()
  clearPaneHoverClose()
  paneActivation = activation
  pane.value = nextPane
  lastSubmenu.value = nextPane
  const requestId = ++paneRequestId
  void nextTick(() => {
    if (!open.value || pane.value !== nextPane || requestId !== paneRequestId) return
    updatePanelPosition()
    if (focusOption) focusFirstOption()
  })
}

function togglePane(nextPane: SubmenuPane) {
  clearPaneHoverOpen()
  clearPaneHoverClose()
  if (pane.value === nextPane) {
    if (paneActivation === 'hover') {
      openPane(nextPane, { activation: 'persistent', focusOption: true })
      return
    }
    returnToRoot()
    return
  }
  openPane(nextPane)
}

function toggleAdvanced() {
  if (!showsReasoningSlider.value) return
  resetPaneInteraction()
  resetMaximumEntryMotion()
  advancedExpanded.value = !advancedExpanded.value
  const expandedState = advancedExpanded.value
  if (advancedExpanded.value) {
    resetRangeInteraction()
    clearRangeProximity()
  }
  if (pane.value !== 'root') pane.value = 'root'
  const requestId = paneRequestId
  void nextTick(() => {
    if (!open.value || requestId !== paneRequestId) return
    updatePanelPosition()
    rootSurfaceRef.value
      ?.querySelector<HTMLElement>(
        `[data-test="chat-settings-advanced-toggle"][aria-expanded="${expandedState}"]`,
      )
      ?.focus()
  })
}

function returnToRoot({ restorePaneFocus = true }: { restorePaneFocus?: boolean } = {}) {
  clearPaneHoverOpen()
  clearPaneHoverClose()
  paneActivation = 'none'
  pane.value = 'root'
  const requestId = ++paneRequestId
  const paneToRestore = lastSubmenu.value
  void nextTick(() => {
    if (!open.value || pane.value !== 'root' || requestId !== paneRequestId) return
    updatePanelPosition()
    if (!restorePaneFocus) return
    rootSurfaceRef.value
      ?.querySelector<HTMLElement>(`[data-pane-target="${paneToRestore}"]`)
      ?.focus()
  })
}

function clearPaneHoverClose() {
  if (paneHoverCloseTimer === null) return
  clearTimeout(paneHoverCloseTimer)
  paneHoverCloseTimer = null
}

function clearPaneHoverOpen() {
  if (paneHoverOpenTimer === null) return
  clearTimeout(paneHoverOpenTimer)
  paneHoverOpenTimer = null
}

function resetPaneInteraction() {
  clearPaneHoverOpen()
  clearPaneHoverClose()
  paneActivation = 'none'
  paneRequestId += 1
}

function supportsPaneHover(event: PointerEvent) {
  return !isCompact.value
    && event.pointerType === 'mouse'
    && event.buttons === 0
    && typeof window.matchMedia === 'function'
    && window.matchMedia('(hover: hover) and (pointer: fine)').matches
}

function paneHoverPairContains(target: EventTarget | null, expectedPane: SubmenuPane) {
  if (!(target instanceof Node)) return false
  const paneTarget = rootSurfaceRef.value
    ?.querySelector<HTMLElement>(`[data-pane-target="${expectedPane}"]`)
  return Boolean(paneTarget?.contains(target) || submenuSurfaceRef.value?.contains(target))
}

function onPanePointerEnter(event: PointerEvent, nextPane: SubmenuPane) {
  if (!supportsPaneHover(event)) return
  clearPaneHoverOpen()
  clearPaneHoverClose()
  if (
    pane.value !== nextPane
    && submenuSurfaceRef.value?.contains(document.activeElement)
  ) return
  if (pane.value === nextPane) return
  paneHoverOpenTimer = setTimeout(() => {
    paneHoverOpenTimer = null
    if (!open.value || isCompact.value) return
    openPane(nextPane, { activation: 'hover', focusOption: false })
  }, SUBMENU_HOVER_OPEN_DELAY_MS)
}

function schedulePaneHoverClose(event: PointerEvent, expectedPane: SubmenuPane) {
  if (
    !supportsPaneHover(event)
    || paneActivation !== 'hover'
    || pane.value !== expectedPane
    || paneHoverPairContains(event.relatedTarget, expectedPane)
  ) return
  clearPaneHoverClose()
  paneHoverCloseTimer = setTimeout(() => {
    paneHoverCloseTimer = null
    if (
      !open.value
      || pane.value !== expectedPane
      || paneActivation !== 'hover'
    ) return
    returnToRoot({ restorePaneFocus: false })
  }, SUBMENU_HOVER_CLOSE_DELAY_MS)
}

function onPanePointerLeave(event: PointerEvent, currentPane: SubmenuPane) {
  clearPaneHoverOpen()
  const paneToClose = paneActivation === 'hover' && pane.value !== 'root'
    ? pane.value
    : currentPane
  schedulePaneHoverClose(event, paneToClose)
}

function onSubmenuPointerEnter(event: PointerEvent) {
  if (!supportsPaneHover(event)) return
  clearPaneHoverOpen()
  clearPaneHoverClose()
}

function onSubmenuPointerLeave(event: PointerEvent) {
  if (pane.value === 'root') return
  schedulePaneHoverClose(event, pane.value)
}

function onSubmenuFocusIn() {
  clearPaneHoverClose()
  if (paneActivation === 'hover') paneActivation = 'persistent'
}

function selectModel(value: string) {
  emit('update:modelValue', value)
  closeMenu()
}

function selectReasoning(value: SupportedReasoningEffort) {
  emit('update:reasoningEffort', value)
  closeMenu()
}

function onCapabilityInput(event: Event) {
  const index = Number((event.currentTarget as HTMLInputElement).value)
  const value = REASONING_VALUES[index]
  if (!value) return
  const previousIndex = visualReasoningIndex.value
  if (index !== REASONING_VALUES.length - 1) resetMaximumEntryMotion()
  visualReasoningIndex.value = index
  if (index === REASONING_VALUES.length - 1 && previousIndex !== index) {
    startMaximumEntryMotion()
  }
  if (value && value !== props.reasoningEffort) emit('update:reasoningEffort', value)
}

function resetMaximumEntryMotion() {
  maximumEntranceActive.value = false
  maximumEntryGeneration.value = null
  maxBurstToken.value = null
}

function startMaximumEntryMotion() {
  resetMaximumEntryMotion()
  if (!open.value || !canAnimateRangeSettle()) return
  maximumEntranceActive.value = true
  nextMaxBurstToken += 1
  maximumEntryGeneration.value = nextMaxBurstToken
  maxBurstToken.value = nextMaxBurstToken
}

function onMaximumRevealEnd(event: AnimationEvent) {
  if (event.animationName !== 'chat-reasoning-max-reveal') return
  const eventToken = Number((event.currentTarget as HTMLElement).dataset.entryToken)
  if (!Number.isFinite(eventToken) || eventToken !== maximumEntryGeneration.value) return
  maximumEntranceActive.value = false
}

function onMaximumBurstLifecycleEnd(event: AnimationEvent) {
  if (event.animationName !== 'chat-reasoning-max-burst-lifecycle') return
  const eventToken = Number((event.currentTarget as HTMLElement).dataset.entryToken)
  if (!Number.isFinite(eventToken) || eventToken !== maxBurstToken.value) return
  maxBurstToken.value = null
}

function onRangePointerDown(event: PointerEvent) {
  if (event.isPrimary === false) return
  if (rangePointerId !== null && rangePointerId !== event.pointerId) return
  if (event.shiftKey) window.getSelection()?.removeAllRanges()
  updateRangeProximity(event)
  rangePointerId = event.pointerId
  rangeDragging.value = true
  rangeSettling.value = false
  const range = event.currentTarget as HTMLInputElement
  if (typeof range.setPointerCapture === 'function') {
    try {
      range.setPointerCapture(event.pointerId)
    } catch {
      // Older engines may already own or have released this pointer.
    }
  }
}

function onRangePointerMove(event: PointerEvent) {
  if (rangeDragging.value) {
    clearRangeProximity()
    return
  }
  updateRangeProximity(event)
}

function updateRangeProximity(event: PointerEvent) {
  if (event.pointerType && event.pointerType !== 'mouse') {
    clearRangeProximity()
    return
  }

  const range = event.currentTarget as HTMLInputElement
  const rect = range.parentElement?.getBoundingClientRect()
  if (!rect || rect.width <= 0 || rect.height <= 0) {
    clearRangeProximity()
    return
  }

  const pointerX = event.clientX - rect.left
  const pointerY = event.clientY - rect.top
  const trackCenterY = rect.height / 2
  let nearestIndex: number | null = null
  let nearestDistance = Number.POSITIVE_INFINITY

  reasoningStops.forEach((stop, index) => {
    const distance = Math.max(
      Math.abs(pointerX - stop.offset),
      Math.abs(pointerY - trackCenterY),
    )
    const hitRadius = index === visualReasoningIndex.value
      ? REASONING_SLIDER_THUMB_HIT_RADIUS
      : REASONING_SLIDER_TICK_HIT_RADIUS
    if (distance > hitRadius || distance >= nearestDistance) return
    nearestDistance = distance
    nearestIndex = index
  })

  if (nearbyReasoningIndex.value !== nearestIndex) nearbyReasoningIndex.value = nearestIndex
}

function clearRangeProximity() {
  nearbyReasoningIndex.value = null
}

function resetRangeInteraction() {
  rangePointerId = null
  rangeDragging.value = false
  rangeSettling.value = false
}

function canAnimateRangeSettle() {
  return typeof window.matchMedia !== 'function'
    || !window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

function finishRangeInteraction(event?: PointerEvent) {
  if (
    event
    && rangePointerId !== null
    && event.pointerId !== rangePointerId
  ) return
  const shouldSettle = rangeDragging.value
  const range = event?.currentTarget as HTMLInputElement | undefined
  if (
    range
    && rangePointerId !== null
    && typeof range.hasPointerCapture === 'function'
    && typeof range.releasePointerCapture === 'function'
    && range.hasPointerCapture(rangePointerId)
  ) {
    try {
      range.releasePointerCapture(rangePointerId)
    } catch {
      // Capture can be released automatically before pointerup reaches us.
    }
  }
  rangePointerId = null
  rangeDragging.value = false
  rangeSettling.value = shouldSettle && canAnimateRangeSettle()
}

function onRangePointerCancel(event: PointerEvent) {
  finishRangeInteraction(event)
  clearRangeProximity()
}

function onRangeLostPointerCapture(event: PointerEvent) {
  if (rangePointerId === null || event.pointerId !== rangePointerId) return
  const shouldSettle = rangeDragging.value
  rangePointerId = null
  rangeDragging.value = false
  rangeSettling.value = shouldSettle && canAnimateRangeSettle()
  clearRangeProximity()
}

function onThumbSettleAnimationEnd(event: AnimationEvent) {
  if (event.animationName !== 'chat-reasoning-thumb-settle') return
  rangeSettling.value = false
}

function onRangeBlur() {
  finishRangeInteraction()
  clearRangeProximity()
  rangeFocused.value = false
}

function activeSurface() {
  return pane.value === 'root' ? rootSurfaceRef.value : submenuSurfaceRef.value
}

function focusableControls() {
  return Array.from(activeSurface()?.querySelectorAll<HTMLElement>(
    'button:not([disabled]), input:not([disabled])',
  ) ?? [])
}

function focusFirstActiveControl() {
  focusableControls()[0]?.focus()
}

function focusFirstOption() {
  const surface = submenuSurfaceRef.value
  const selectedOption = surface?.querySelector<HTMLElement>('[role="menuitemradio"][aria-checked="true"]')
  const firstOption = surface?.querySelector<HTMLElement>('[role="menuitemradio"]')
  const targetOption = selectedOption ?? firstOption
  targetOption?.focus()
}

function updateCompactMode() {
  const nextCompact = window.innerWidth <= 640
  if (nextCompact === isCompact.value) return
  isCompact.value = nextCompact
  if (nextCompact && paneActivation === 'hover' && pane.value !== 'root') {
    returnToRoot({ restorePaneFocus: false })
  }
}

function updatePanelPosition() {
  const trigger = triggerRef.value
  const panel = panelRef.value
  if (!trigger || !panel) return

  const triggerRect = trigger.getBoundingClientRect()
  const panelWidth = Math.min(
    ROOT_PANEL_WIDTH,
    window.innerWidth - VIEWPORT_PADDING * 2,
  )
  const rootHeight = rootSurfaceRef.value?.offsetHeight ?? 0
  const submenuHeight = submenuSurfaceRef.value?.offsetHeight ?? 0
  const panelHeight = pane.value === 'root'
    ? rootHeight
    : isCompact.value
      ? rootHeight + submenuHeight + 8
      : Math.max(rootHeight, submenuHeight)
  const openTriggerWidth = Math.min(
    OPEN_TRIGGER_WIDTH,
    window.innerWidth - VIEWPORT_PADDING * 2,
  )
  const openTriggerCenter = triggerRect.right - (openTriggerWidth / 2)
  const left = Math.min(
    Math.max(
      VIEWPORT_PADDING,
      openTriggerCenter - (panelWidth / 2),
    ),
    window.innerWidth - panelWidth - VIEWPORT_PADDING,
  )

  if (!isCompact.value && pane.value !== 'root') {
    const measuredSubmenuWidth = submenuSurfaceRef.value?.offsetWidth ?? 0
    const submenuWidth = measuredSubmenuWidth || DESKTOP_SUBMENU_RESERVE_WIDTH
    const rightSpace = window.innerWidth
      - VIEWPORT_PADDING
      - (left + panelWidth - SUBMENU_OVERLAP)
    const leftSpace = left + SUBMENU_OVERLAP - VIEWPORT_PADDING
    submenuPlacement.value = rightSpace >= submenuWidth || rightSpace >= leftSpace
      ? 'right'
      : 'left'
  } else {
    submenuPlacement.value = 'right'
  }

  if (isCompact.value) {
    const availableAbove = Math.max(
      0,
      triggerRect.top - POPOVER_TRIGGER_GAP - VIEWPORT_PADDING,
    )
    const availableBelow = Math.max(
      0,
      window.innerHeight - triggerRect.bottom - POPOVER_TRIGGER_GAP - VIEWPORT_PADDING,
    )
    const opensAbove = availableAbove >= availableBelow
    const availableHeight = Math.max(
      1,
      opensAbove ? availableAbove : availableBelow,
    )
    const visiblePanelHeight = Math.min(
      Math.max(1, panelHeight),
      availableHeight,
    )
    const top = opensAbove
      ? Math.max(
          VIEWPORT_PADDING,
          triggerRect.top - POPOVER_TRIGGER_GAP - visiblePanelHeight,
        )
      : Math.min(
          triggerRect.bottom + POPOVER_TRIGGER_GAP,
          window.innerHeight - VIEWPORT_PADDING - visiblePanelHeight,
        )

    panelPlacement.value = opensAbove ? 'above' : 'below'
    panelStyle.value = {
      position: 'fixed',
      top: `${top}px`,
      left: `${Math.max(VIEWPORT_PADDING, left)}px`,
      width: `${panelWidth}px`,
      maxHeight: `${availableHeight}px`,
    }
    return
  }

  const hasRoomAbove = triggerRect.top >= panelHeight + VIEWPORT_PADDING + POPOVER_TRIGGER_GAP
  panelPlacement.value = hasRoomAbove ? 'above' : 'below'
  panelStyle.value = {
    position: 'fixed',
    left: `${Math.max(VIEWPORT_PADDING, left)}px`,
    width: `${panelWidth}px`,
    maxHeight: `calc(100vh - ${VIEWPORT_PADDING * 2}px)`,
    ...(hasRoomAbove
      ? { bottom: `${window.innerHeight - triggerRect.top + POPOVER_TRIGGER_GAP}px` }
      : { top: `${triggerRect.bottom + POPOVER_TRIGGER_GAP}px` }),
  }
}

function onPanelKeydown(event: KeyboardEvent) {
  keyboardFocusVisible.value = true
  const target = event.target as HTMLElement | null
  if (event.key === 'Escape') {
    event.preventDefault()
    if (isCompact.value && pane.value !== 'root') {
      returnToRoot()
      return
    }
    closeMenu()
    return
  }

  if (
    paneActivation === 'hover'
    && target
    && rootSurfaceRef.value?.contains(target)
  ) {
    returnToRoot({ restorePaneFocus: false })
  }

  const isRangeInput = target instanceof HTMLInputElement && target.type === 'range'

  if (event.key === 'ArrowLeft' && pane.value !== 'root') {
    event.preventDefault()
    returnToRoot()
    return
  }

  if (
    event.key === 'ArrowLeft'
    && pane.value === 'root'
    && advancedExpanded.value
    && target?.matches('[data-test="chat-settings-advanced-toggle"]')
  ) {
    event.preventDefault()
    toggleAdvanced()
    return
  }

  if (event.key === 'ArrowRight' && pane.value === 'root' && !isRangeInput) {
    if (
      target?.matches('[data-test="chat-settings-advanced-toggle"]')
      && !advancedExpanded.value
    ) {
      event.preventDefault()
      toggleAdvanced()
      return
    }
    const nextPane = target?.closest<HTMLElement>('[data-pane-target]')?.dataset.paneTarget
    if (nextPane === 'models' || nextPane === 'reasoning') {
      event.preventDefault()
      openPane(nextPane)
      return
    }
  }

  if ((event.key === 'Home' || event.key === 'End') && !isRangeInput) {
    const controls = focusableControls()
    if (!controls.length) return
    event.preventDefault()
    controls[event.key === 'Home' ? 0 : controls.length - 1]?.focus()
    return
  }

  if (event.key !== 'ArrowDown' && event.key !== 'ArrowUp') return
  if (isRangeInput) return
  const controls = focusableControls()
  if (!controls.length) return
  event.preventDefault()
  const currentIndex = target ? controls.indexOf(target) : -1
  const direction = event.key === 'ArrowDown' ? 1 : -1
  const fallbackIndex = direction > 0 ? 0 : controls.length - 1
  const nextIndex = currentIndex < 0
    ? fallbackIndex
    : (currentIndex + direction + controls.length) % controls.length
  controls[nextIndex]?.focus()
}

function onDocumentPointerDown(event: PointerEvent) {
  const target = event.target as Node | null
  if (
    !open.value
    || (target && triggerRef.value?.contains(target))
    || (target && panelRef.value?.contains(target))
  ) {
    return
  }
  closeMenu(false)
}

function onDocumentFocusIn(event: FocusEvent) {
  const target = event.target as Node | null
  if (
    !open.value
    || (target && triggerRef.value?.contains(target))
    || (target && panelRef.value?.contains(target))
  ) {
    return
  }
  closeMenu(false)
}

function onViewportChange() {
  updateCompactMode()
  if (open.value) void nextTick(updatePanelPosition)
}

function onModelSettingsShortcut(event: KeyboardEvent): void {
  if (
    event.repeat
    || event.code !== 'KeyM'
    || !event.ctrlKey
    || !event.shiftKey
    || event.altKey
    || event.metaKey
    || props.disabled
    || !chatControlShortcutIsInScope(event, triggerRef.value, panelRef.value)
  ) return
  event.preventDefault()
  toggleMenu()
}

onMounted(() => {
  updateCompactMode()
  document.addEventListener('pointerdown', onDocumentPointerDown)
  document.addEventListener('focusin', onDocumentFocusIn)
  window.addEventListener('resize', onViewportChange)
  window.addEventListener('scroll', updatePanelPosition, true)
  window.addEventListener('keydown', onModelSettingsShortcut)
})

onBeforeUnmount(() => {
  resetPaneInteraction()
  resetMaximumEntryMotion()
  resetRangeInteraction()
  document.removeEventListener('pointerdown', onDocumentPointerDown)
  document.removeEventListener('focusin', onDocumentFocusIn)
  window.removeEventListener('resize', onViewportChange)
  window.removeEventListener('scroll', updatePanelPosition, true)
  window.removeEventListener('keydown', onModelSettingsShortcut)
})
</script>

<style scoped>
.chat-model-settings {
  --chat-model-trigger-font-size: 16px;
  --chat-model-trigger-font-weight: 400;
  --chat-model-trigger-line-height: 26px;

  position: relative;
  z-index: 2;
  display: flex;
  width: max-content;
  max-width: min(164px, 42vw);
  height: 36px;
  flex: 0 0 auto;
  justify-content: flex-end;
  margin-inline-start: 4px;
  min-width: 0;
  overflow: visible;
  -webkit-user-select: none;
  user-select: none;
}

.chat-model-settings__sizer {
  display: flex;
  max-width: 100%;
  height: 36px;
  align-items: center;
  gap: 6px;
  padding: 0 12px 0 14px;
  visibility: hidden;
  pointer-events: none;
}

.chat-model-settings__sizer-selection {
  display: flex;
  min-width: 0;
  align-items: baseline;
  gap: 4px;
  overflow: hidden;
}

.chat-model-settings__sizer-selection strong {
  overflow: hidden;
  font-size: var(--chat-model-trigger-font-size);
  font-weight: var(--chat-model-trigger-font-weight);
  line-height: var(--chat-model-trigger-line-height);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-model-settings__sizer::after {
  width: 14px;
  height: 14px;
  flex: 0 0 14px;
  margin-inline-end: -2px;
  content: '';
}

.chat-model-settings__trigger {
  position: absolute;
  inset-block-start: 0;
  inset-inline-end: 0;
  display: flex;
  width: 100%;
  max-width: none;
  height: 36px;
  align-items: center;
  gap: 6px;
  border: 0;
  border-radius: 999px;
  padding: 0 12px 0 14px;
  color: var(--chat-composer-muted-fg, var(--lx-clay-text-secondary));
  background: transparent;
  box-shadow: none;
  transition:
    width 180ms cubic-bezier(0.23, 1, 0.32, 1),
    color 140ms ease,
    background-color 140ms ease;
}

.chat-model-settings__trigger:hover:not(:disabled),
.chat-model-settings__trigger--open {
  color: var(--chat-composer-primary-fg, var(--lx-clay-text));
  background: var(--chat-composer-secondary-hover, var(--lx-clay-recessed));
}

.chat-model-settings__trigger--open {
  width: 164px;
  max-width: min(164px, calc(100vw - 24px));
}

.chat-model-settings__trigger:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}

.chat-model-settings__trigger:focus-visible {
  outline: 2px solid var(--lx-clay-accent);
  outline-offset: 2px;
}

.chat-model-settings__selection {
  display: flex;
  min-width: 0;
  flex: 1;
  align-items: baseline;
  gap: 4px;
  justify-content: center;
  overflow: hidden;
}

.chat-model-settings__selection strong {
  overflow: hidden;
  font-size: var(--chat-model-trigger-font-size);
  font-weight: var(--chat-model-trigger-font-weight);
  line-height: var(--chat-model-trigger-line-height);
  letter-spacing: normal;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-model-settings__trigger-model {
  flex: 0 1 auto;
  color: var(--chat-composer-primary-fg, var(--lx-clay-text));
}

.chat-model-settings__trigger-effort {
  flex: 0 1 auto;
  color: var(--chat-composer-tertiary-fg, var(--lx-clay-text-muted));
}

.chat-model-settings__trigger-effort--solo {
  color: var(--chat-composer-tertiary-fg, var(--lx-clay-text-muted));
}

.chat-model-settings__chevron {
  width: 14px;
  height: 14px;
  flex: 0 0 auto;
  margin-inline-end: -2px;
  color: var(--chat-composer-tertiary-fg, var(--lx-clay-text-muted));
  transition: transform 180ms cubic-bezier(0.23, 1, 0.32, 1);
}

.chat-model-settings__chevron--open {
  transform: rotate(180deg);
}

@media (prefers-reduced-motion: reduce) {
  .chat-model-settings__trigger,
  .chat-model-settings__chevron {
    transition: none;
  }
}
</style>

<style>
@property --chat-reasoning-max-fill-mask-position {
  syntax: "<percentage>";
  inherits: false;
  initial-value: 0%;
}

.chat-model-settings-popover {
  --chat-settings-surface: var(--workspace-popup-surface);
  --chat-settings-slider-track: var(--workspace-hover);
  --chat-settings-slider-tick: color-mix(in srgb, var(--workspace-text-muted) 50%, transparent);
  --chat-settings-slider-thumb: var(--workspace-popup-surface);
  --chat-settings-text-primary: var(--workspace-text);
  --chat-settings-text-secondary: var(--workspace-text-secondary);
  --chat-settings-text-tertiary: var(--workspace-text-muted);
  --chat-settings-divider: var(--workspace-divider);
  --chat-settings-hover: var(--workspace-hover);
  --chat-settings-shadow:
    0 0 0 1px var(--workspace-border),
    var(--workspace-popover-shadow);
  z-index: 60;
  overflow: visible;
  color: var(--chat-settings-text-primary);
  -webkit-user-select: none;
  user-select: none;
}

html.dark .chat-model-settings-popover {
  --chat-settings-surface: #353535;
  --chat-settings-slider-track: #4a4a4a;
  --chat-settings-slider-tick: #7d7d7d;
  --chat-settings-slider-thumb: #fff;
}

.chat-model-settings-popover--compact {
  overflow-x: hidden;
  overflow-y: auto;
  overscroll-behavior: contain;
  scrollbar-width: thin;
}

.chat-model-settings-popover--compact .chat-model-settings-popover__surface {
  max-height: none;
}

.chat-model-settings-popover--compact .chat-model-settings-popover__options {
  max-height: none;
  overflow-y: visible;
}

.chat-model-settings-popover__surface {
  width: 100%;
  max-height: calc(100vh - 24px);
  overflow: hidden;
  border-radius: 20px;
  color: var(--chat-settings-text-primary);
  background: var(--chat-settings-surface);
  box-shadow: var(--chat-settings-shadow);
}

.chat-model-settings-popover__surface--root {
  position: relative;
  display: grid;
  grid-template-rows: 96px;
  transition: grid-template-rows 400ms cubic-bezier(0.19, 1, 0.22, 1);
}

.chat-model-settings-popover__surface--advanced {
  grid-template-rows: 132px;
}

.chat-model-settings-popover__surface--plain {
  grid-template-rows: 92px;
}

.chat-model-settings-popover__root-content {
  position: relative;
  min-height: 0;
  overflow: hidden;
}

.chat-model-settings-popover__capability {
  --chat-reasoning-slider-accent: #3a83f7;
  --chat-reasoning-slider-track: var(--chat-settings-slider-track);
  --chat-reasoning-slider-tick: var(--chat-settings-slider-tick);
  --chat-reasoning-max-gradient-start: #250e7a;
  --chat-reasoning-max-gradient-middle: #c775e9;
  --chat-reasoning-max-gradient-end: #7849d1;
  --chat-reasoning-max-particle: rgb(255 255 255 / 72%);
  --chat-reasoning-max-particle-glow: rgb(255 255 255 / 34%);
  position: absolute;
  top: 54px;
  right: 14px;
  left: 14px;
  height: 32px;
}

.chat-model-settings-popover__range-wrap {
  --reasoning-motion-duration: 300ms;
  --reasoning-motion-easing: cubic-bezier(0.23, 1, 0.32, 1);
  position: relative;
  height: 32px;
  touch-action: none;
  -webkit-user-select: none;
  user-select: none;
}

.chat-model-settings-popover__range-wrap--dragging {
  --reasoning-motion-duration: 150ms;
}

.chat-model-settings-popover__range-rail {
  position: absolute;
  top: 4px;
  right: 0;
  left: 0;
  height: 24px;
  overflow: visible;
  border-radius: 999px;
  background: var(--chat-reasoning-slider-track);
  box-shadow: inset 0 0 2px color-mix(in srgb, var(--workspace-text) 18%, transparent);
}

.chat-model-settings-popover__range-fill {
  position: absolute;
  z-index: 0;
  inset: 0;
  border-radius: inherit;
  background: var(--chat-reasoning-slider-accent);
  clip-path: inset(0 var(--reasoning-fill-inset) 0 0 round 999px);
  transition: clip-path var(--reasoning-motion-duration) var(--reasoning-motion-easing);
  will-change: clip-path;
}

.chat-model-settings-popover__max-effects-viewport {
  position: absolute;
  z-index: 1;
  inset: 0;
  overflow: hidden;
  border-radius: inherit;
  clip-path: inset(0 var(--reasoning-fill-inset) 0 0 round 999px);
  pointer-events: none;
  transition: clip-path var(--reasoning-motion-duration) var(--reasoning-motion-easing);
  will-change: clip-path;
}

.chat-model-settings-popover__max-effects {
  position: absolute;
  inset: 0;
  border-radius: inherit;
  pointer-events: none;
  will-change: opacity, transform;
}

.chat-reasoning-max-effects-leave-active {
  transition:
    opacity 220ms cubic-bezier(0.23, 1, 0.32, 1),
    transform 220ms cubic-bezier(0.23, 1, 0.32, 1);
}

.chat-reasoning-max-effects-leave-to {
  opacity: 0;
  transform: translateX(50px);
}

.chat-model-settings-popover__max-fill {
  --chat-reasoning-max-fill-mask-position: 0%;
  position: absolute;
  inset: -1px;
  overflow: hidden;
  contain: paint;
  border-radius: 13px;
  background: linear-gradient(
    90deg,
    var(--chat-reasoning-max-gradient-start),
    var(--chat-reasoning-max-gradient-middle) 55%,
    var(--chat-reasoning-max-gradient-end)
  );
  isolation: isolate;
  mask-image: linear-gradient(
    90deg,
    transparent calc(var(--chat-reasoning-max-fill-mask-position) - 120px),
    #000 var(--chat-reasoning-max-fill-mask-position),
    #000 100%
  );
  mask-repeat: no-repeat;
  mask-size: 100% 100%;
}

.chat-model-settings-popover__max-effects--entering
  .chat-model-settings-popover__max-fill {
  animation: chat-reasoning-max-reveal 2s ease both;
}

.chat-model-settings-popover__max-track-particles {
  position: absolute;
  inset: 0;
  overflow: hidden;
  pointer-events: none;
}

.chat-model-settings-popover__max-effects--entering
  .chat-model-settings-popover__max-track-particles {
  animation: chat-reasoning-max-particles-enter 150ms cubic-bezier(0.23, 1, 0.32, 1) both;
}

.chat-model-settings-popover__max-track-particles > span {
  position: absolute;
  width: 3px;
  height: 3px;
  border-radius: 999px;
  background: var(--chat-reasoning-max-particle);
  box-shadow: 0 0 5px var(--chat-reasoning-max-particle-glow);
  opacity: var(--particle-opacity, 0.82);
  transform: translate3d(-50%, -50%, 0) scale(var(--particle-scale, 0.82));
  animation: chat-reasoning-max-particle-drift var(--particle-duration, 2s)
    cubic-bezier(0.45, 0, 0.55, 1) var(--particle-delay, 0s) infinite alternate;
  will-change: transform, opacity;
}

.chat-model-settings-popover__max-track-particles > span:nth-child(1) {
  top: 32%; left: 8%; --particle-dx: 9px; --particle-dy: 3px; --particle-duration: 1.72s; --particle-delay: -0.4s; --particle-opacity: 0.99; --particle-scale: 0.95;
}

.chat-model-settings-popover__max-track-particles > span:nth-child(2) {
  top: 72%; left: 18%; --particle-dx: -7px; --particle-dy: -4px; --particle-duration: 2.36s; --particle-delay: -1.1s; --particle-scale: 0.58;
}

.chat-model-settings-popover__max-track-particles > span:nth-child(3) {
  top: 26%; left: 31%; --particle-dx: 12px; --particle-dy: 5px; --particle-duration: 1.54s; --particle-delay: -0.8s; --particle-opacity: 0.9; --particle-scale: 0.86;
}

.chat-model-settings-popover__max-track-particles > span:nth-child(4) {
  top: 68%; left: 43%; --particle-dx: -10px; --particle-dy: -3px; --particle-duration: 2.88s; --particle-delay: -2.1s; --particle-scale: 0.6;
}

.chat-model-settings-popover__max-track-particles > span:nth-child(5) {
  top: 44%; left: 55%; --particle-dx: 8px; --particle-dy: -5px; --particle-duration: 2.18s; --particle-delay: -1.4s;
}

.chat-model-settings-popover__max-track-particles > span:nth-child(6) {
  top: 76%; left: 67%; --particle-dx: -12px; --particle-dy: 2px; --particle-duration: 1.46s; --particle-delay: -0.3s; --particle-opacity: 0.72; --particle-scale: 0.75;
}

.chat-model-settings-popover__max-track-particles > span:nth-child(7) {
  top: 28%; left: 79%; --particle-dx: 7px; --particle-dy: 5px; --particle-duration: 2.64s; --particle-delay: -1.8s; --particle-scale: 0.55;
}

.chat-model-settings-popover__max-track-particles > span:nth-child(8) {
  top: 62%; left: 92%; --particle-dx: -8px; --particle-dy: -5px; --particle-duration: 2.22s; --particle-delay: -0.9s; --particle-opacity: 0.95; --particle-scale: 0.9;
}

.chat-model-settings-popover__max-track-particles > span:nth-child(9) {
  top: 52%; left: 13%; --particle-dx: 11px; --particle-dy: -2px; --particle-duration: 1.82s; --particle-delay: -1.3s; --particle-scale: 0.52;
}

.chat-model-settings-popover__max-track-particles > span:nth-child(10) {
  top: 84%; left: 37%; --particle-dx: -6px; --particle-dy: -5px; --particle-duration: 2.96s; --particle-delay: -2.5s;
}

.chat-model-settings-popover__max-track-particles > span:nth-child(11) {
  top: 18%; left: 61%; --particle-dx: 10px; --particle-dy: 4px; --particle-duration: 1.58s; --particle-delay: -0.7s; --particle-opacity: 0.88; --particle-scale: 0.88;
}

.chat-model-settings-popover__max-track-particles > span:nth-child(12) {
  top: 48%; left: 86%; --particle-dx: -12px; --particle-dy: 3px; --particle-duration: 2.26s; --particle-delay: -1.6s; --particle-scale: 0.64;
}

.chat-model-settings-popover__max-track-particles > span:nth-child(13) {
  top: 54%; left: 49%; --particle-dx: 6px; --particle-dy: -4px; --particle-duration: 1.69s; --particle-delay: -1s; --particle-opacity: 0.8;
}

.chat-model-settings-popover__max-track-particles > span:nth-child(14) {
  top: 60%; left: 73%; --particle-dx: -9px; --particle-dy: 5px; --particle-duration: 2.67s; --particle-delay: -2.2s; --particle-scale: 0.57;
}

.chat-model-settings-popover__range-ticks {
  position: absolute;
  z-index: 2;
  inset: 0;
  pointer-events: none;
}

.chat-model-settings-popover__range-ticks > span {
  position: absolute;
  top: 50%;
  width: var(--reasoning-tick-size);
  height: var(--reasoning-tick-size);
  border-radius: 50%;
  background: var(--chat-reasoning-slider-tick);
  transform: translate(-50%, -50%);
  filter: brightness(1);
  transition:
    transform 120ms ease,
    filter 120ms ease,
    background-color var(--reasoning-motion-duration) var(--reasoning-motion-easing),
    opacity 200ms ease-in-out;
}

.chat-model-settings-popover__range-ticks > .chat-model-settings-popover__range-tick--active {
  background: rgb(255 255 255 / 30%);
}

.chat-model-settings-popover__range-ticks > .chat-model-settings-popover__range-tick--nearby {
  transform: translate(-50%, -50%) scale(2);
  filter: brightness(0.85);
}

.chat-model-settings-popover__range-wrap--maximum
  .chat-model-settings-popover__range-ticks > span {
  opacity: 0;
  transform: translate(-50%, -50%) scale(0.75);
  transition:
    background-color var(--reasoning-motion-duration) var(--reasoning-motion-easing),
    opacity 200ms ease-in-out,
    filter 120ms ease,
    transform 120ms ease;
}

.chat-model-settings-popover__range-thumb-track {
  position: absolute;
  z-index: 3;
  top: 0;
  bottom: 0;
  left: 0;
  width: 0;
  overflow: visible;
  pointer-events: none;
  transform: translate3d(var(--reasoning-position), 0, 0);
  transition: transform var(--reasoning-motion-duration) var(--reasoning-motion-easing);
  will-change: transform;
}

.chat-model-settings-popover__range-thumb {
  position: absolute;
  top: 50%;
  left: 0;
  width: var(--reasoning-thumb-size);
  height: var(--reasoning-thumb-size);
  border: 0.5px solid color-mix(in srgb, var(--workspace-text) 20%, transparent);
  border-radius: 50%;
  background: var(--chat-settings-slider-thumb);
  box-shadow: 0 0 2px color-mix(in srgb, var(--workspace-text) 10%, transparent);
  transform: translate(-50%, -50%);
  transition: transform 120ms var(--reasoning-motion-easing);
  will-change: transform;
}

.chat-model-settings-popover__max-burst {
  position: absolute;
  top: 50%;
  left: 0;
  width: 76px;
  height: 76px;
  pointer-events: none;
  transform: translate(-50%, -50%);
  animation: chat-reasoning-max-burst-lifecycle 640ms linear both;
}

.chat-model-settings-popover__max-burst > span {
  --particle-x: 0px;
  --particle-y: -36px;
  position: absolute;
  top: 50%;
  left: 50%;
  width: 5px;
  height: 5px;
  border-radius: 999px;
  background: #752aff;
  transform: translate(-50%, -50%) scale(0.2);
  animation: chat-reasoning-max-particle-burst 620ms cubic-bezier(0.25, 1, 0.5, 1) both;
}

.chat-model-settings-popover__max-burst > span:nth-child(1) { --particle-x: -3px; --particle-y: -34px; }
.chat-model-settings-popover__max-burst > span:nth-child(2) { --particle-x: 15px; --particle-y: -29px; animation-delay: 8ms; }
.chat-model-settings-popover__max-burst > span:nth-child(3) { --particle-x: 30px; --particle-y: -19px; background: #8f5cff; }
.chat-model-settings-popover__max-burst > span:nth-child(4) { --particle-x: 34px; --particle-y: -2px; animation-delay: 12ms; }
.chat-model-settings-popover__max-burst > span:nth-child(5) { --particle-x: 26px; --particle-y: 20px; background: #9d6bff; }
.chat-model-settings-popover__max-burst > span:nth-child(6) { --particle-x: 12px; --particle-y: 31px; animation-delay: 10ms; }
.chat-model-settings-popover__max-burst > span:nth-child(7) { --particle-x: -6px; --particle-y: 34px; background: #8f5cff; }
.chat-model-settings-popover__max-burst > span:nth-child(8) { --particle-x: -22px; --particle-y: 26px; animation-delay: 14ms; }
.chat-model-settings-popover__max-burst > span:nth-child(9) { --particle-x: -32px; --particle-y: 9px; background: #9d6bff; }
.chat-model-settings-popover__max-burst > span:nth-child(10) { --particle-x: -32px; --particle-y: -10px; animation-delay: 8ms; }
.chat-model-settings-popover__max-burst > span:nth-child(11) { --particle-x: -21px; --particle-y: -26px; background: #8f5cff; }
.chat-model-settings-popover__max-burst > span:nth-child(12) { --particle-x: 7px; --particle-y: -24px; animation-delay: 6ms; }
.chat-model-settings-popover__max-burst > span:nth-child(13) { --particle-x: 24px; --particle-y: -9px; background: #9d6bff; }
.chat-model-settings-popover__max-burst > span:nth-child(14) { --particle-x: 20px; --particle-y: 10px; animation-delay: 10ms; }
.chat-model-settings-popover__max-burst > span:nth-child(15) { --particle-x: -9px; --particle-y: 21px; background: #8f5cff; }
.chat-model-settings-popover__max-burst > span:nth-child(16) { --particle-x: -25px; --particle-y: -5px; animation-delay: 12ms; }

@keyframes chat-reasoning-max-burst-lifecycle {
  from,
  to {
    opacity: 1;
  }
}

.chat-model-settings-popover__range-wrap--thumb-nearby
  .chat-model-settings-popover__range-thumb,
.chat-model-settings-popover__range-wrap--dragging
  .chat-model-settings-popover__range-thumb {
  transform: translate(-50%, -50%) scale(1.142857);
}

.chat-model-settings-popover__range-wrap--settling
  .chat-model-settings-popover__range-thumb {
  animation: chat-reasoning-thumb-settle 350ms linear both;
}

.chat-model-settings-popover__range {
  position: absolute;
  top: 0;
  bottom: 0;
  left: -1px;
  z-index: 2;
  width: calc(100% + 2px);
  height: 32px;
  margin: 0;
  appearance: none;
  background: transparent;
  cursor: pointer;
  opacity: 0;
}

.chat-model-settings-popover__range::-webkit-slider-runnable-track {
  height: 24px;
  background: transparent;
}

.chat-model-settings-popover__range::-webkit-slider-thumb {
  width: var(--reasoning-thumb-size);
  height: var(--reasoning-thumb-size);
  border: 0;
  -webkit-appearance: none;
  appearance: none;
  background: transparent;
}

.chat-model-settings-popover__range::-moz-range-track {
  height: 24px;
  background: transparent;
}

.chat-model-settings-popover__range::-moz-range-thumb {
  width: var(--reasoning-thumb-size);
  height: var(--reasoning-thumb-size);
  border: 0;
  background: transparent;
}

.chat-model-settings-popover__range-wrap--keyboard-focus
  .chat-model-settings-popover__range-rail {
  outline: 2px solid var(--lx-clay-accent);
  outline-offset: 2px;
}

@keyframes chat-reasoning-max-reveal {
  from { --chat-reasoning-max-fill-mask-position: 100%; }
  to { --chat-reasoning-max-fill-mask-position: 0%; }
}

@keyframes chat-reasoning-max-particles-enter {
  from { opacity: 0.06; }
  to { opacity: 1; }
}

@keyframes chat-reasoning-max-particle-drift {
  from {
    opacity: 0.4;
    transform: translate3d(-50%, -50%, 0) scale(0.52);
  }
  to {
    opacity: var(--particle-opacity, 0.72);
    transform: translate3d(
      calc(-50% + var(--particle-dx, 0px)),
      calc(-50% + var(--particle-dy, 0px)),
      0
    ) scale(var(--particle-scale, 0.7));
  }
}

@keyframes chat-reasoning-thumb-settle {
  0% { transform: translate(-50%, -50%) scale(1.142857); }
  4.6% { transform: translate(-50%, -50%) scale(1.1181); }
  9.1% { transform: translate(-50%, -50%) scale(1.104); }
  14.3% { transform: translate(-50%, -50%) scale(1.0766); }
  28.6% { transform: translate(-50%, -50%) scale(1.0422); }
  42.9% { transform: translate(-50%, -50%) scale(1.0206); }
  62.9% { transform: translate(-50%, -50%) scale(1.0043); }
  100% { transform: translate(-50%, -50%) scale(1); }
}

@keyframes chat-reasoning-max-particle-burst {
  0% {
    opacity: 0;
    transform: translate(-50%, -50%) scale(0.25);
  }
  22% {
    opacity: 1;
    transform: translate(-50%, -50%) scale(1.28);
  }
  100% {
    opacity: 0;
    transform: translate(
      calc(-50% + var(--particle-x)),
      calc(-50% + var(--particle-y))
    ) scale(0.55);
  }
}

.chat-model-settings-popover__advanced {
  position: absolute;
  top: 10px;
  left: 6px;
  display: flex;
  height: 32px;
  align-items: center;
  gap: 4px;
  border: 0;
  border-radius: 8px;
  padding: 4px 8px;
  color: var(--chat-settings-text-secondary);
  background: transparent;
  font: inherit;
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
  letter-spacing: normal;
  line-height: 20px;
  transition:
    color 120ms ease,
    background-color 120ms ease;
}

.chat-model-settings-popover__surface--advanced
  .chat-model-settings-popover__advanced {
  top: 90px;
}

.chat-model-settings-popover__advanced:hover,
.chat-model-settings-popover__advanced:focus-visible {
  color: var(--chat-settings-text-secondary);
  background: transparent;
  outline: none;
}

.chat-model-settings-popover__advanced--keyboard-focus:focus {
  box-shadow: inset 0 0 0 2px var(--chat-settings-text-primary);
}

.chat-model-settings-popover__advanced svg {
  transition: transform 240ms cubic-bezier(0.23, 1, 0.32, 1);
}

.chat-model-settings-popover__advanced--open svg {
  transform: rotate(-90deg);
}

.chat-settings-advanced-shift-enter-active,
.chat-settings-advanced-shift-leave-active {
  transition:
    opacity 200ms cubic-bezier(0.23, 1, 0.32, 1),
    transform 400ms cubic-bezier(0.19, 1, 0.22, 1);
}

.chat-model-settings-popover__advanced:not(
  .chat-model-settings-popover__advanced--open
).chat-settings-advanced-shift-enter-from,
.chat-model-settings-popover__advanced:not(
  .chat-model-settings-popover__advanced--open
).chat-settings-advanced-shift-leave-to {
  opacity: 0;
  transform: translateY(80px);
}

.chat-model-settings-popover__advanced--open.chat-settings-advanced-shift-enter-from,
.chat-model-settings-popover__advanced--open.chat-settings-advanced-shift-leave-to {
  opacity: 0;
  transform: translateY(-80px);
}

.chat-model-settings-popover__divider {
  position: absolute;
  top: 85px;
  right: 6px;
  left: 6px;
  height: 1px;
  background: var(--chat-settings-divider);
}

.chat-model-settings-popover__rows {
  position: absolute;
  top: 10px;
  right: 0;
  left: 0;
}

.chat-model-settings-popover__surface--plain .chat-model-settings-popover__rows {
  top: 10px;
}

.chat-model-settings-popover__row {
  display: flex;
  width: calc(100% - 20px);
  height: 36px;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  margin: 0 10px;
  border: 0;
  border-radius: 12px;
  padding: 6px 10px;
  color: var(--chat-settings-text-primary);
  background: transparent;
  text-align: left;
  transition: background-color 120ms ease, color 120ms ease;
}

.chat-model-settings-popover__row:hover,
.chat-model-settings-popover__row:focus-visible {
  color: var(--chat-settings-text-primary);
  background: var(--chat-settings-hover);
  outline: none;
}

.chat-model-settings-popover__row-copy {
  display: flex;
  min-width: 0;
  flex: 1;
  align-items: center;
}

.chat-model-settings-popover__row-trailing {
  display: flex;
  min-width: 0;
  align-self: stretch;
  align-items: center;
  gap: 4px;
}

.chat-model-settings-popover__row-copy small,
.chat-model-settings-popover__row-trailing strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-model-settings-popover__row-copy small {
  color: var(--chat-settings-text-primary);
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
  letter-spacing: normal;
  line-height: 20px;
}

.chat-model-settings-popover__row-trailing strong {
  color: var(--chat-settings-text-secondary);
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
  letter-spacing: normal;
  line-height: 20px;
}

.chat-model-settings-popover__row-trailing svg {
  width: 16px;
  height: 16px;
  flex: 0 0 16px;
  margin-inline-end: -1px;
  color: var(--chat-settings-text-secondary);
}

.chat-model-settings-popover__submenu {
  position: absolute;
  width: max-content;
  min-width: 100px;
  max-width: min(320px, calc(100vw - 24px));
}

.chat-model-settings-popover__submenu--right {
  left: calc(100% - 8px);
}

.chat-model-settings-popover__submenu--left {
  right: calc(100% - 8px);
}

.chat-model-settings-popover__submenu--above {
  bottom: 0;
}

.chat-model-settings-popover__submenu--below {
  top: 0;
}

.chat-model-settings-popover__submenu--compact {
  position: static;
  width: 100%;
  min-width: 0;
  max-width: none;
  margin-top: 8px;
}

.chat-model-settings-popover__options {
  max-height: min(340px, calc(100vh - 82px));
  overflow-y: auto;
  padding: 10px 0;
  overscroll-behavior: contain;
}

.chat-model-settings-popover__option {
  display: flex;
  width: calc(100% - 20px);
  height: 36px;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  margin: 0 10px;
  border: 0;
  border-radius: 12px;
  padding: 6px 10px;
  color: var(--chat-settings-text-primary);
  background: transparent;
  text-align: left;
  transition: background-color 120ms ease, color 120ms ease;
}

.chat-model-settings-popover__option:hover,
.chat-model-settings-popover__option:focus-visible {
  color: var(--chat-settings-text-primary);
  background: var(--chat-settings-hover);
  outline: none;
}

.chat-model-settings-popover__option > strong {
  overflow: hidden;
  min-width: 0;
  flex: 1 1 auto;
  color: inherit;
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
  letter-spacing: normal;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-model-settings-popover__check-slot {
  display: flex;
  width: 16px;
  flex: 0 0 16px;
  align-self: stretch;
  align-items: center;
  justify-content: center;
}

.chat-model-settings-popover__check {
  width: 16px;
  height: 16px;
  flex: 0 0 16px;
  color: var(--chat-settings-text-primary);
}

.chat-settings-popover-enter-active,
.chat-settings-popover-leave-active {
  transition: opacity 140ms ease, transform 140ms cubic-bezier(0.22, 1, 0.36, 1);
}

.chat-settings-popover-enter-from,
.chat-settings-popover-leave-to {
  opacity: 0;
  transform: translateY(5px);
}

.chat-settings-submenu-enter-active,
.chat-settings-submenu-leave-active {
  transition:
    opacity 150ms ease,
    transform 190ms cubic-bezier(0.23, 1, 0.32, 1);
}

.chat-settings-submenu-enter-from,
.chat-settings-submenu-leave-to {
  opacity: 0;
  transform: translateX(-6px);
}

.chat-model-settings-popover__submenu--left.chat-settings-submenu-enter-from,
.chat-model-settings-popover__submenu--left.chat-settings-submenu-leave-to {
  transform: translateX(6px);
}

.chat-settings-disclosure-enter-active,
.chat-settings-disclosure-leave-active {
  transition:
    opacity 180ms cubic-bezier(0.23, 1, 0.32, 1),
    transform 240ms cubic-bezier(0.19, 1, 0.22, 1);
}

.chat-settings-disclosure-enter-from,
.chat-settings-disclosure-leave-to {
  opacity: 0;
  transform: translateY(4px);
}

@media (prefers-reduced-motion: reduce) {
  .chat-model-settings-popover__surface--root,
  .chat-model-settings-popover__row,
  .chat-model-settings-popover__option,
  .chat-model-settings-popover__advanced,
  .chat-model-settings-popover__advanced svg,
  .chat-model-settings-popover__range-fill,
  .chat-model-settings-popover__range-thumb-track,
  .chat-model-settings-popover__range-thumb,
  .chat-model-settings-popover__range-ticks > span,
  .chat-model-settings-popover__max-fill,
  .chat-model-settings-popover__max-track-particles,
  .chat-model-settings-popover__max-burst,
  .chat-model-settings-popover__max-track-particles > span,
  .chat-model-settings-popover__max-burst > span,
  .chat-reasoning-max-effects-leave-active,
  .chat-settings-popover-enter-active,
  .chat-settings-popover-leave-active,
  .chat-settings-submenu-enter-active,
  .chat-settings-submenu-leave-active,
  .chat-settings-advanced-shift-enter-active,
  .chat-settings-advanced-shift-leave-active,
  .chat-settings-disclosure-enter-active,
  .chat-settings-disclosure-leave-active {
    transition: none;
    animation: none;
  }

  .chat-model-settings-popover__max-track-particles,
  .chat-model-settings-popover__max-burst {
    display: none;
  }
}
</style>

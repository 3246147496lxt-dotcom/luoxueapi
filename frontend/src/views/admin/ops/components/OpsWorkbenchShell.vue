<template>
  <section class="ops-workbench-shell">
    <section
      class="ops-workbench-shell__evidence"
      role="region"
      :aria-label="evidenceLabel"
    >
      <slot name="evidence">
        <slot />
      </slot>
    </section>

    <aside
      class="ops-workbench-shell__rail"
      :aria-label="railLabel"
    >
      <div class="ops-workbench-shell__mobile-rail-header">
        <span>{{ railLabel }}</span>
        <button
          type="button"
          class="ops-workbench-shell__rail-toggle"
          :aria-controls="resolvedRailId"
          :aria-expanded="isMobileRailOpen"
          :aria-label="mobileToggleLabel || railLabel"
          @click="toggleMobileRail"
        >
          <Icon name="chevronDown" size="sm" class="ops-workbench-shell__rail-chevron" aria-hidden="true" />
        </button>
      </div>

      <div
        :id="resolvedRailId"
        class="ops-workbench-shell__rail-body"
        :class="{ 'ops-workbench-shell__rail-body--mobile-collapsed': !isMobileRailOpen }"
      >
        <slot name="rail" />
      </div>
    </aside>
  </section>
</template>

<script setup lang="ts">
import { computed, getCurrentInstance, ref } from 'vue'
import Icon from '@/components/icons/Icon.vue'

interface Props {
  railLabel: string
  evidenceLabel: string
  railId?: string
  mobileToggleLabel?: string
  mobileRailOpen?: boolean
  defaultMobileRailOpen?: boolean
}

interface Emits {
  (event: 'update:mobileRailOpen', value: boolean): void
}

const props = withDefaults(defineProps<Props>(), {
  railId: undefined,
  mobileToggleLabel: undefined,
  mobileRailOpen: undefined,
  defaultMobileRailOpen: true
})
const emit = defineEmits<Emits>()

const instance = getCurrentInstance()
const internalMobileRailOpen = ref(props.defaultMobileRailOpen)
const resolvedRailId = computed(
  () => props.railId || `ops-workbench-investigation-rail-${instance?.uid ?? 'default'}`
)
const isMobileRailOpen = computed(
  () => props.mobileRailOpen ?? internalMobileRailOpen.value
)

function toggleMobileRail(): void {
  const nextValue = !isMobileRailOpen.value
  internalMobileRailOpen.value = nextValue
  emit('update:mobileRailOpen', nextValue)
}
</script>

<style scoped>
.ops-workbench-shell {
  display: grid;
  grid-template-columns: 340px minmax(0, 1fr);
  grid-template-areas: 'rail evidence';
  min-width: 0;
  height: calc(100dvh - 104px);
  min-height: 0;
  overflow: hidden;
  color: var(--lx-clay-text);
  background: #ffffff;
  font-family: var(--lx-clay-font-ui);
}

.ops-workbench-shell__rail {
  display: flex;
  grid-area: rail;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
  border-right: 1px solid rgba(91, 80, 112, 0.14);
  background: #ffffff;
}

.ops-workbench-shell__rail-body {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex: 1 1 auto;
  overflow: hidden;
}

.ops-workbench-shell__evidence {
  grid-area: evidence;
  min-width: 0;
  min-height: 0;
  overflow-y: auto;
  padding: 32px;
  background: #fcfcfc;
}

.ops-workbench-shell__mobile-rail-header {
  display: none;
}

.ops-workbench-shell__rail-toggle {
  display: inline-grid;
  width: 44px;
  height: 44px;
  place-items: center;
  border: 0;
  border-radius: var(--lx-clay-radius-control);
  color: var(--lx-clay-text-secondary);
  background: transparent;
  cursor: pointer;
}

.ops-workbench-shell__rail-chevron {
  width: 20px;
  height: 20px;
  transition: transform 180ms ease-out;
}

.ops-workbench-shell__rail-toggle[aria-expanded='false'] .ops-workbench-shell__rail-chevron {
  transform: rotate(-90deg);
}

.ops-workbench-shell__rail-toggle:hover {
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface-soft);
}

.ops-workbench-shell__rail-toggle:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 34%, transparent);
  outline-offset: -3px;
}

@media (max-width: 1180px) {
  .ops-workbench-shell__evidence { padding: 32px; }
}

@media (max-width: 860px) {
  .ops-workbench-shell {
    height: auto;
    min-height: calc(100dvh - 104px);
    overflow: visible;
    grid-template-columns: minmax(0, 1fr);
    grid-template-areas:
      'rail'
      'evidence';
  }

  .ops-workbench-shell__rail {
    overflow: visible;
    border-right: 0;
    border-bottom: 1px solid var(--lx-clay-border);
  }

  .ops-workbench-shell__mobile-rail-header {
    display: flex;
    min-height: 52px;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    padding: 4px 12px 4px 16px;
    color: var(--lx-clay-text);
    font-size: 0.8125rem;
    font-weight: 750;
  }

  .ops-workbench-shell__rail-body--mobile-collapsed {
    display: none;
  }

  .ops-workbench-shell__evidence {
    overflow: visible;
    padding: 16px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .ops-workbench-shell__rail-chevron {
    transition: none;
  }
}
</style>

<template>
  <RouterLink
    v-if="!collapsed"
    to="/chat"
    class="chat-sidebar-brand"
    :aria-label="brandLabel"
  >
    <span class="chat-sidebar-brand__copy">
      <strong>{{ siteName }}</strong>
      <span v-if="membership.productPlanLabel.value">
        {{ membership.productPlanLabel.value }}
      </span>
    </span>
  </RouterLink>

  <span v-else class="chat-sidebar-brand__collapsed-entry">
    <button
      ref="collapsedTriggerRef"
      type="button"
      class="chat-sidebar-brand chat-sidebar-brand--collapsed"
      :aria-label="openSidebarLabel"
      :aria-controls="controls || undefined"
      aria-expanded="false"
      :aria-describedby="tooltipId"
      @click="$emit('expand')"
    >
      <span class="chat-sidebar-brand__monogram" aria-hidden="true">
        {{ brandMonogram }}
      </span>
      <SidebarCollapseIcon class="chat-sidebar-brand__expand-icon" :collapsed="true" />
    </button>
    <span :id="tooltipId" class="chat-sidebar-brand__tooltip" role="tooltip">
      {{ openSidebarLabel }}
    </span>
  </span>
</template>

<script setup lang="ts">
import { computed, ref, useId } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import SidebarCollapseIcon from '@/components/icons/SidebarCollapseIcon.vue'
import { useUserMembership } from '@/composables/useUserMembership'
import { useAppStore } from '@/stores/app'
import { useUserProfileStore } from '@/stores/userProfile'

withDefaults(defineProps<{
  collapsed?: boolean
  controls?: string
}>(), {
  collapsed: false,
  controls: '',
})

const { t } = useI18n()
const appStore = useAppStore()
const userProfileStore = useUserProfileStore()
const {
  subscriptionsLoaded,
  activeSubscriptionCount,
  primarySubscription,
} = storeToRefs(userProfileStore)
const membership = useUserMembership({
  subscriptionsLoaded,
  activeSubscriptionCount,
  primarySubscription,
})
const collapsedTriggerRef = ref<HTMLButtonElement | null>(null)
const tooltipId = `${useId()}-open-sidebar-tooltip`

const siteName = computed(() => appStore.siteName.trim() || '落雪AI')
const brandMonogram = computed(() => (
  Array.from(siteName.value.replace(/\s+/g, '')).slice(0, 2).join('').toLocaleUpperCase()
))
const brandLabel = computed(() => (
  membership.productPlanLabel.value
    ? `${siteName.value} · ${membership.productPlanLabel.value}`
    : siteName.value
))
const openSidebarLabel = computed(() => t('chat.actions.openSidebar'))

function focus() {
  collapsedTriggerRef.value?.focus()
}

defineExpose({ focus })
</script>

<style scoped>
.chat-sidebar-brand {
  display: flex;
  width: 100%;
  min-width: 0;
  min-height: 36px;
  align-items: center;
  border-radius: 8px;
  color: var(--workspace-text);
  background: transparent;
  text-decoration: none;
  transition: color 140ms ease, background-color 140ms ease;
}

.chat-sidebar-brand__copy {
  display: flex;
  width: 100%;
  min-width: 0;
  align-items: baseline;
  gap: 6px;
}

.chat-sidebar-brand__copy strong,
.chat-sidebar-brand__copy span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-sidebar-brand__copy strong {
  flex: 1 1 auto;
  font-size: var(--workspace-type-brand-size);
  font-weight: var(--workspace-type-brand-weight);
  letter-spacing: -0.02em;
  line-height: 1.2;
}

.chat-sidebar-brand__copy span {
  max-width: 72px;
  flex: 0 1 auto;
  color: var(--workspace-text-secondary);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  line-height: 1.2;
}

.chat-sidebar-brand__collapsed-entry {
  position: relative;
  display: inline-grid;
  width: 44px;
  height: 44px;
  place-items: center;
}

.chat-sidebar-brand--collapsed {
  width: 44px;
  height: 44px;
  min-height: 44px;
  justify-content: center;
  border: 0;
  padding: 0;
  cursor: pointer;
}

.chat-sidebar-brand__monogram,
.chat-sidebar-brand__expand-icon {
  grid-area: 1 / 1;
  transition: opacity 120ms ease, transform 120ms ease;
}

.chat-sidebar-brand__monogram {
  display: grid;
  width: 32px;
  height: 32px;
  place-items: center;
  border: 1px solid var(--workspace-border);
  border-radius: 9px;
  color: var(--workspace-text);
  background: var(--workspace-surface);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-brand-weight);
  letter-spacing: -0.02em;
  line-height: 1;
}

.chat-sidebar-brand__expand-icon {
  width: 18px;
  height: 18px;
  opacity: 0;
  transform: scale(0.92);
}

.chat-sidebar-brand:hover {
  background: var(--workspace-hover);
}

.chat-sidebar-brand--collapsed:hover .chat-sidebar-brand__monogram,
.chat-sidebar-brand--collapsed:focus-visible .chat-sidebar-brand__monogram {
  opacity: 0;
  transform: scale(0.92);
}

.chat-sidebar-brand--collapsed:hover .chat-sidebar-brand__expand-icon,
.chat-sidebar-brand--collapsed:focus-visible .chat-sidebar-brand__expand-icon {
  opacity: 1;
  transform: scale(1);
}

.chat-sidebar-brand--collapsed:focus-visible {
  outline: 2px solid var(--workspace-text-secondary);
  outline-offset: 2px;
}

.chat-sidebar-brand__tooltip {
  position: absolute;
  top: 50%;
  left: calc(100% + 8px);
  z-index: 30;
  width: max-content;
  max-width: min(220px, calc(100vw - 96px));
  border-radius: 7px;
  padding: 6px 8px;
  color: var(--workspace-text);
  background: var(--workspace-popup-surface);
  box-shadow: 0 2px 8px rgb(17 24 39 / 0.14);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  line-height: 1.25;
  opacity: 0;
  pointer-events: none;
  transform: translate(3px, -50%);
  visibility: hidden;
  white-space: nowrap;
  transition: opacity 120ms ease, transform 120ms ease, visibility 120ms ease;
}

.chat-sidebar-brand__collapsed-entry:hover .chat-sidebar-brand__tooltip,
.chat-sidebar-brand__collapsed-entry:focus-within .chat-sidebar-brand__tooltip {
  opacity: 1;
  transform: translate(0, -50%);
  visibility: visible;
}

@media (prefers-reduced-motion: reduce) {
  .chat-sidebar-brand,
  .chat-sidebar-brand__monogram,
  .chat-sidebar-brand__expand-icon,
  .chat-sidebar-brand__tooltip {
    transition: none;
  }
}
</style>

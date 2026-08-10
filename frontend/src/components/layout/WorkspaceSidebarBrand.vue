<template>
  <RouterLink
    :to="homePath"
    class="workspace-sidebar-brand"
    :aria-label="brandLabel"
  >
    <span class="workspace-sidebar-brand__copy">
      <strong data-testid="workspace-sidebar-brand-name">{{ brandName }}</strong>
      <span v-if="topPlanLabel" data-testid="workspace-sidebar-plan">
        {{ topPlanLabel }}
      </span>
    </span>
  </RouterLink>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useUserMembership } from '@/composables/useUserMembership'
import { useUserProfileStore } from '@/stores/userProfile'

defineProps<{
  homePath: string
}>()

const brandName = '落雪AI'
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
const topPlanLabel = computed(() => (
  membership.state.value === 'pending'
    ? null
    : membership.accountPlanLabel.value
))
const brandLabel = computed(() => (
  topPlanLabel.value ? `${brandName} · ${topPlanLabel.value}` : brandName
))
</script>

<style scoped>
.workspace-sidebar-brand {
  display: flex;
  width: 100%;
  min-width: 0;
  min-height: var(--workspace-sidebar-action-size);
  align-items: center;
  border-radius: var(--workspace-radius-compact);
  padding-inline-start: 6px;
  color: var(--workspace-identity-text);
  background: transparent;
  text-decoration: none;
  transition: color 150ms ease;
}

.workspace-sidebar-brand:focus-visible {
  outline: 2px solid var(--workspace-text-secondary);
  outline-offset: 2px;
}

.workspace-sidebar-brand__copy {
  display: flex;
  width: 100%;
  min-width: 0;
  align-items: baseline;
  gap: var(--workspace-space-1);
}

.workspace-sidebar-brand__copy strong,
.workspace-sidebar-brand__copy span {
  min-width: 0;
  overflow: hidden;
  font-size: 18px;
  font-weight: 600;
  letter-spacing: -0.27px;
  line-height: 26px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workspace-sidebar-brand__copy strong {
  flex: 0 0 auto;
  color: var(--workspace-identity-text);
}

.workspace-sidebar-brand__copy span {
  flex: 1 1 auto;
  color: var(--workspace-identity-text-tertiary);
}

@media (prefers-reduced-motion: reduce) {
  .workspace-sidebar-brand {
    transition: none;
  }
}
</style>

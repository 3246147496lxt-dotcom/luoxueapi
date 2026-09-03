<template>
  <span class="skill-status-badge" :class="`skill-status-badge--${status}`">
    <span class="skill-status-badge__dot" aria-hidden="true"></span>
    {{ label }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { SkillStatus, SkillVersionStatus } from '@/api/admin/skills'

const props = defineProps<{
  status: SkillStatus | SkillVersionStatus
}>()

const { t } = useI18n()

const versionStatuses: SkillVersionStatus[] = ['available', 'active', 'yanked']
const label = computed(() => t(
  versionStatuses.includes(props.status as SkillVersionStatus)
    ? `admin.skills.versionStatus.${props.status}`
    : `admin.skills.status.${props.status}`,
))
</script>

<style scoped>
.skill-status-badge {
  display: inline-flex;
  min-height: 26px;
  align-items: center;
  gap: 7px;
  padding: 4px 9px;
  border-radius: 999px;
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-recessed);
  font-size: 0.75rem;
  font-weight: 750;
  line-height: 1;
  white-space: nowrap;
}

.skill-status-badge__dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
}

.skill-status-badge--published,
.skill-status-badge--active {
  color: var(--lx-clay-success-text);
  background: var(--lx-clay-success-soft);
}

.skill-status-badge--draft,
.skill-status-badge--available {
  color: var(--lx-clay-warning);
  background: var(--lx-clay-warning-soft);
}

.skill-status-badge--archived,
.skill-status-badge--yanked {
  color: var(--lx-clay-danger);
  background: var(--lx-clay-danger-soft);
}
</style>

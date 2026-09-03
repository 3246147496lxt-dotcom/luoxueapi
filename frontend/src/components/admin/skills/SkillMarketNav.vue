<template>
  <nav class="skill-market-nav" :aria-label="t('admin.skills.navigation.label')">
    <RouterLink
      v-for="item in items"
      :key="item.key"
      :to="item.to"
      class="skill-market-nav__link"
      :class="{ 'skill-market-nav__link--active': active === item.key }"
      :aria-current="active === item.key ? 'page' : undefined"
    >
      <Icon :name="item.icon" size="sm" aria-hidden="true" />
      <span>{{ t(item.labelKey) }}</span>
    </RouterLink>
  </nav>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'

export type SkillMarketSection = 'catalog' | 'imports' | 'schedules'

defineProps<{
  active: SkillMarketSection
}>()

const { t } = useI18n()

const items = [
  {
    key: 'catalog' as const,
    to: '/admin/skills',
    icon: 'cube',
    labelKey: 'admin.skills.navigation.catalog',
  },
  {
    key: 'imports' as const,
    to: '/admin/skills/imports',
    icon: 'sync',
    labelKey: 'admin.skills.navigation.imports',
  },
  {
    key: 'schedules' as const,
    to: { path: '/admin/skills/imports', query: { tab: 'schedules' } },
    icon: 'calendar',
    labelKey: 'admin.skills.navigation.schedules',
  },
] as const
</script>

<style scoped>
.skill-market-nav {
  display: inline-flex;
  max-width: 100%;
  gap: 3px;
  overflow-x: auto;
  padding: 3px;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-control);
  background: var(--lx-clay-recessed);
  box-shadow: var(--lx-clay-shadow-inset);
  scrollbar-width: none;
}

.skill-market-nav::-webkit-scrollbar {
  display: none;
}

.skill-market-nav__link {
  display: inline-flex;
  min-height: 36px;
  flex: 0 0 auto;
  align-items: center;
  gap: 7px;
  padding: 7px 12px;
  border-radius: 10px;
  color: var(--lx-clay-text-secondary);
  font-size: 0.8125rem;
  font-weight: 750;
  text-decoration: none;
  transition: color 150ms ease, background-color 150ms ease, box-shadow 150ms ease;
}

.skill-market-nav__link:hover {
  color: var(--lx-clay-text);
  background: var(--lx-clay-hover);
}

.skill-market-nav__link--active {
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-flat);
}

html.dark .skill-market-nav__link--active {
  color: var(--lx-clay-accent);
}

.skill-market-nav__link:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 30%, transparent);
  outline-offset: 2px;
}

@media (max-width: 767px) {
  .skill-market-nav__link {
    min-height: 44px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .skill-market-nav__link {
    transition: none;
  }
}
</style>

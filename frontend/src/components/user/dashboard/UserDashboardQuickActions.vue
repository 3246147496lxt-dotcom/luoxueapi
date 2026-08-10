<template>
  <section class="dashboard-panel" :aria-label="t('dashboard.workspace.quickActions')">
    <header class="dashboard-panel-header">
      <div>
        <h2>{{ t('dashboard.workspace.quickActions') }}</h2>
        <p>{{ t('dashboard.workspace.quickActionsDescription') }}</p>
      </div>
    </header>

    <nav class="dashboard-action-list" :aria-label="t('dashboard.workspace.quickActions')">
      <RouterLink
        v-for="action in actions"
        :key="action.to"
        :to="action.to"
        class="dashboard-action"
      >
        <span class="dashboard-action__icon" aria-hidden="true">
          <Icon :name="action.icon" size="sm" :stroke-width="1.7" />
        </span>
        <span class="dashboard-action__copy">
          <strong>{{ t(action.labelKey) }}</strong>
          <span>{{ t(action.descriptionKey) }}</span>
        </span>
        <Icon name="chevronRight" size="xs" class="dashboard-action__arrow" aria-hidden="true" />
      </RouterLink>
    </nav>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()

const actions = [
  {
    to: '/chat',
    icon: 'chat' as const,
    labelKey: 'dashboard.workspace.actions.newChat',
    descriptionKey: 'dashboard.workspace.actions.newChatDescription',
  },
  {
    to: '/purchase',
    icon: 'wallet' as const,
    labelKey: 'dashboard.workspace.actions.balance',
    descriptionKey: 'dashboard.workspace.actions.balanceDescription',
  },
  {
    to: '/keys',
    icon: 'key' as const,
    labelKey: 'dashboard.workspace.actions.apiKeys',
    descriptionKey: 'dashboard.workspace.actions.apiKeysDescription',
  },
  {
    to: '/usage',
    icon: 'chart' as const,
    labelKey: 'dashboard.workspace.actions.usage',
    descriptionKey: 'dashboard.workspace.actions.usageDescription',
  },
]
</script>

<style scoped>
.dashboard-panel {
  min-width: 0;
  overflow: hidden;
  border: 1px solid var(--workspace-border);
  border-radius: var(--workspace-radius-work-card);
  background: var(--workspace-card-surface);
  box-shadow: var(--workspace-work-shadow-card);
}

.dashboard-panel-header {
  display: flex;
  min-height: 78px;
  align-items: center;
  padding: var(--workspace-space-4-5) var(--workspace-space-5);
  border-bottom: 1px solid var(--workspace-border);
}

.dashboard-panel-header h2 {
  color: var(--workspace-work-text);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
  line-height: 1.35rem;
}

.dashboard-panel-header p {
  margin-top: 3px;
  color: var(--workspace-work-text-muted);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  line-height: 1rem;
}

.dashboard-action-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 4px;
  padding: 8px;
}

.dashboard-action {
  display: grid;
  min-width: 0;
  min-height: 72px;
  grid-template-columns: 34px minmax(0, 1fr) 16px;
  align-items: center;
  gap: 11px;
  padding: 12px;
  border-radius: var(--workspace-radius-button);
  border: 1px solid transparent;
  color: var(--workspace-work-text-secondary);
  transition: color 140ms ease, background-color 140ms ease, border-color 140ms ease;
}

.dashboard-action:first-child {
  border-color: var(--workspace-work-accent-border);
  background: color-mix(in srgb, var(--workspace-work-accent-soft) 58%, transparent);
}

.dashboard-action:hover {
  border-color: var(--workspace-work-accent-border);
  color: var(--workspace-work-accent-hover);
  background: var(--workspace-work-accent-soft);
}

.dashboard-action:focus-visible {
  outline: 2px solid var(--workspace-work-accent);
  outline-offset: 2px;
}

.dashboard-action__icon {
  display: inline-flex;
  width: 34px;
  height: 34px;
  align-items: center;
  justify-content: center;
  border-radius: var(--workspace-radius-button);
  color: var(--workspace-work-accent);
  background: var(--workspace-work-accent-soft);
}

.dashboard-action:nth-child(even) .dashboard-action__icon {
  color: var(--workspace-work-info);
  background: var(--workspace-work-info-soft);
}

.dashboard-action__copy {
  min-width: 0;
}

.dashboard-action__copy strong,
.dashboard-action__copy span {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-action__copy strong {
  color: inherit;
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
  line-height: 1.15rem;
}

.dashboard-action__copy span {
  margin-top: 3px;
  color: var(--workspace-work-text-muted);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  line-height: 1rem;
}

.dashboard-action__arrow {
  color: var(--workspace-border-strong);
  transition: color 140ms ease, transform 140ms ease;
}

.dashboard-action:hover .dashboard-action__arrow {
  color: var(--workspace-work-accent);
  transform: translateX(2px);
}

@media (max-width: 479px) {
  .dashboard-action-list {
    grid-template-columns: minmax(0, 1fr);
  }

  .dashboard-panel-header {
    padding: 16px 18px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .dashboard-action,
  .dashboard-action__arrow {
    transition-duration: 0.01ms;
  }
}

:global(html.dark) .dashboard-panel {
  border-color: var(--workspace-border);
  background: var(--workspace-card-surface);
  box-shadow: none;
}

:global(html.dark) .dashboard-panel-header {
  border-color: var(--workspace-border);
}

:global(html.dark) .dashboard-panel-header h2,
:global(html.dark) .dashboard-action {
  color: var(--workspace-dark-text);
}

:global(html.dark) .dashboard-panel-header p,
:global(html.dark) .dashboard-action__copy span {
  color: var(--workspace-dark-text-muted);
}

:global(html.dark) .dashboard-action:hover {
  background: var(--workspace-surface-subtle);
}
</style>

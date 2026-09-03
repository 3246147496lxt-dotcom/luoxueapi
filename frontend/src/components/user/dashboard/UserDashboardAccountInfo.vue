<template>
  <section class="dashboard-panel" :aria-label="t('dashboard.workspace.accountInfo')">
    <header class="dashboard-panel-header">
      <div>
        <h2>{{ t('dashboard.workspace.accountInfo') }}</h2>
        <p>{{ t('dashboard.workspace.accountInfoDescription') }}</p>
      </div>
      <RouterLink to="/profile" class="dashboard-panel-link">
        {{ t('dashboard.workspace.manageAccount') }}
        <Icon name="arrowRight" size="xs" aria-hidden="true" />
      </RouterLink>
    </header>

    <div class="dashboard-account-profile">
      <span class="dashboard-account-avatar" aria-hidden="true">{{ initials }}</span>
      <span class="min-w-0">
        <strong :title="displayName">{{ displayName }}</strong>
        <span :title="email">{{ email }}</span>
      </span>
    </div>

    <dl class="dashboard-account-details">
      <div>
        <dt>
          <Icon name="checkCircle" size="sm" aria-hidden="true" />
          {{ t('dashboard.workspace.accountStatus') }}
        </dt>
        <dd>
          <span class="dashboard-account-status-dot" :class="status === 'active' ? 'is-active' : ''" aria-hidden="true" />
          {{ t(`dashboard.workspace.accountStatuses.${status}`) }}
        </dd>
      </div>
      <div>
        <dt>
          <Icon name="key" size="sm" aria-hidden="true" />
          {{ t('dashboard.workspace.apiKeys') }}
        </dt>
        <dd>{{ t('dashboard.workspace.activeApiKeys', { active: activeApiKeys, total: totalApiKeys }) }}</dd>
      </div>
      <div>
        <dt>
          <Icon name="calendar" size="sm" aria-hidden="true" />
          {{ t('dashboard.workspace.memberSince') }}
        </dt>
        <dd>{{ formatMemberSince(createdAt) }}</dd>
      </div>
    </dl>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{
  displayName: string
  email: string
  status: 'active' | 'disabled'
  activeApiKeys: number
  totalApiKeys: number
  createdAt: string
}>()

const { t, locale } = useI18n()
const initials = computed(() => Array.from(props.displayName.trim())[0]?.toLocaleUpperCase() || 'U')

function formatMemberSince(value: string): string {
  const date = new Date(value)
  if (!Number.isFinite(date.getTime())) return t('dashboard.workspace.notAvailable')
  return new Intl.DateTimeFormat(locale.value.startsWith('zh') ? 'zh-CN' : 'en-US', {
    year: 'numeric',
    month: 'short',
  }).format(date)
}
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
  justify-content: space-between;
  gap: 18px;
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

.dashboard-panel-link {
  display: inline-flex;
  min-height: 34px;
  flex: 0 0 auto;
  align-items: center;
  gap: 5px;
  border-radius: var(--workspace-radius-compact);
  color: var(--workspace-work-text-secondary);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
}

.dashboard-panel-link:hover {
  color: var(--workspace-work-accent-hover);
  text-decoration: underline;
  text-underline-offset: 3px;
}

.dashboard-panel-link:focus-visible {
  outline: 2px solid var(--workspace-work-accent);
  outline-offset: 3px;
}

.dashboard-account-profile {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: var(--workspace-space-5);
  border-bottom: 1px solid var(--workspace-border);
}

.dashboard-account-avatar {
  display: flex;
  width: 38px;
  height: 38px;
  flex: 0 0 38px;
  align-items: center;
  justify-content: center;
  border-radius: var(--workspace-radius-button);
  color: var(--workspace-light-surface);
  background: var(--workspace-work-accent-deep);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
}

.dashboard-account-profile strong,
.dashboard-account-profile span > span {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-account-profile strong {
  color: var(--workspace-work-text);
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
  line-height: 1.25rem;
}

.dashboard-account-profile span > span {
  margin-top: 2px;
  color: var(--workspace-work-text-muted);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  line-height: 1rem;
}

.dashboard-account-details {
  padding: 6px 20px;
}

.dashboard-account-details > div {
  display: flex;
  min-height: 54px;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  border-bottom: 1px solid var(--workspace-divider);
}

.dashboard-account-details > div:last-child {
  border-bottom: 0;
}

.dashboard-account-details dt {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  gap: 8px;
  color: var(--workspace-work-text-muted);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
}

.dashboard-account-details dt :deep(svg) {
  color: var(--workspace-work-text-muted);
}

.dashboard-account-details > div:nth-child(2) dt :deep(svg) {
  color: var(--workspace-work-accent);
}

.dashboard-account-details dd {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  gap: 7px;
  color: var(--workspace-work-text);
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
  text-align: right;
}

.dashboard-account-status-dot {
  width: 6px;
  height: 6px;
  flex: 0 0 6px;
  border-radius: 999px;
  background: var(--workspace-dark-text-muted);
}

.dashboard-account-status-dot.is-active {
  background: var(--workspace-work-success);
}

@media (max-width: 479px) {
  .dashboard-panel-link {
    min-height: 44px;
  }

  .dashboard-panel-header {
    align-items: flex-start;
    padding: 16px 18px;
  }

  .dashboard-account-profile,
  .dashboard-account-details {
    padding-right: 18px;
    padding-left: 18px;
  }
}

:global(html.dark) .dashboard-panel {
  border-color: var(--workspace-border);
  background: var(--workspace-card-surface);
  box-shadow: none;
}

:global(html.dark) .dashboard-panel-header,
:global(html.dark) .dashboard-account-profile {
  border-color: var(--workspace-border);
}

:global(html.dark) .dashboard-panel-header h2,
:global(html.dark) .dashboard-account-profile strong,
:global(html.dark) .dashboard-account-details dd {
  color: var(--workspace-dark-text);
}

:global(html.dark) .dashboard-panel-header p,
:global(html.dark) .dashboard-account-profile span > span,
:global(html.dark) .dashboard-account-details dt {
  color: var(--workspace-dark-text-muted);
}

:global(html.dark) .dashboard-account-details > div {
  border-color: var(--workspace-border);
}
</style>

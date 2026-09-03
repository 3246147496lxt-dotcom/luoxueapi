<template>
  <section class="dashboard-panel" :aria-label="t('dashboard.apiInfo.title')">
    <header class="dashboard-panel-header">
      <div>
        <h2>{{ t('dashboard.apiInfo.title') }}</h2>
        <p>{{ t('dashboard.workspace.apiInfoDescription') }}</p>
      </div>
      <RouterLink to="/keys" class="dashboard-panel-link">
        {{ t('dashboard.workspace.manageApiKeys') }}
        <Icon name="arrowRight" size="xs" aria-hidden="true" />
      </RouterLink>
    </header>

    <div v-if="endpoints.length > 0" class="dashboard-endpoint-list">
      <article v-for="endpoint in endpoints" :key="endpoint.endpoint" class="dashboard-endpoint">
        <div class="dashboard-endpoint__heading">
          <strong>{{ endpoint.name }}</strong>
          <span :class="{ 'is-primary': endpoint.kind === 'primary' }">{{ endpoint.kind === 'primary'
            ? t('dashboard.workspace.primaryEndpoint')
            : t('dashboard.workspace.customEndpoint') }}</span>
        </div>
        <code :title="endpoint.endpoint">{{ endpoint.endpoint }}</code>
        <p v-if="endpoint.description">{{ endpoint.description }}</p>
        <div class="dashboard-endpoint__actions">
          <button
            type="button"
            class="dashboard-endpoint-action"
            :aria-label="t('dashboard.workspace.copyEndpoint', { name: endpoint.name })"
            @click="copyEndpoint(endpoint.endpoint)"
          >
            <Icon :name="copiedEndpoint === endpoint.endpoint ? 'check' : 'copy'" size="xs" aria-hidden="true" />
            {{ copiedEndpoint === endpoint.endpoint
              ? t('dashboard.workspace.copied')
              : t('dashboard.workspace.copy') }}
          </button>
          <a
            :href="endpoint.endpoint"
            target="_blank"
            rel="noopener noreferrer"
            class="dashboard-endpoint-action"
          >
            <Icon name="externalLink" size="xs" aria-hidden="true" />
            {{ t('dashboard.apiInfo.open') }}
          </a>
        </div>
      </article>
    </div>

    <div v-else class="dashboard-endpoint-empty">
      <Icon name="link" size="lg" aria-hidden="true" />
      <p>{{ t('dashboard.apiInfo.empty') }}</p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { CustomEndpoint } from '@/types'
import { sanitizeUrl } from '@/utils/url'

const props = defineProps<{
  apiBaseUrl: string
  customEndpoints: CustomEndpoint[]
}>()

const { t } = useI18n()
const copiedEndpoint = ref<string | null>(null)
let copiedTimer: ReturnType<typeof setTimeout> | null = null

const endpoints = computed(() => {
  const items: Array<{
    name: string
    endpoint: string
    description: string
    kind: 'primary' | 'custom'
  }> = []
  const seen = new Set<string>()
  const primary = sanitizeUrl(props.apiBaseUrl)
  if (primary) {
    seen.add(primary)
    items.push({
      name: t('dashboard.apiInfo.primary'),
      endpoint: primary,
      description: t('dashboard.apiInfo.primaryDescription'),
      kind: 'primary',
    })
  }

  for (const endpoint of props.customEndpoints || []) {
    const safeEndpoint = sanitizeUrl(endpoint.endpoint)
    if (!safeEndpoint || seen.has(safeEndpoint)) continue
    seen.add(safeEndpoint)
    items.push({
      name: endpoint.name || t('dashboard.apiInfo.custom'),
      endpoint: safeEndpoint,
      description: endpoint.description || '',
      kind: 'custom',
    })
  }

  return items
})

async function copyEndpoint(endpoint: string): Promise<void> {
  if (typeof navigator === 'undefined' || !navigator.clipboard) return
  try {
    await navigator.clipboard.writeText(endpoint)
    copiedEndpoint.value = endpoint
    if (copiedTimer) clearTimeout(copiedTimer)
    copiedTimer = setTimeout(() => {
      copiedEndpoint.value = null
      copiedTimer = null
    }, 1800)
  } catch {
    copiedEndpoint.value = null
  }
}

onBeforeUnmount(() => {
  if (copiedTimer) clearTimeout(copiedTimer)
})
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

.dashboard-panel-link:focus-visible,
.dashboard-endpoint-action:focus-visible {
  outline: 2px solid var(--workspace-work-accent);
  outline-offset: 2px;
}

.dashboard-endpoint-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0;
}

.dashboard-endpoint {
  min-width: 0;
  padding: var(--workspace-space-5);
}

.dashboard-endpoint:nth-child(even) {
  border-left: 1px solid var(--workspace-border);
}

.dashboard-endpoint:nth-child(n + 3) {
  border-top: 1px solid var(--workspace-border);
}

.dashboard-endpoint__heading {
  display: flex;
  align-items: center;
  gap: 9px;
}

.dashboard-endpoint__heading strong {
  overflow: hidden;
  color: var(--workspace-work-text);
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
  line-height: 1.15rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-endpoint__heading span {
  flex: 0 0 auto;
  padding: 2px 6px;
  border: 1px solid var(--workspace-border);
  border-radius: 999px;
  color: var(--workspace-work-text-muted);
  background: var(--workspace-surface-subtle);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  line-height: 0.875rem;
}

.dashboard-endpoint__heading span.is-primary {
  border-color: var(--workspace-work-accent-border);
  color: var(--workspace-work-accent-hover);
  background: var(--workspace-work-accent-soft);
}

.dashboard-endpoint code {
  display: block;
  overflow: hidden;
  margin-top: 13px;
  padding: 9px 10px;
  border: 1px solid var(--workspace-border);
  border-radius: var(--workspace-radius-work-button);
  color: var(--workspace-work-text-secondary);
  background: var(--workspace-surface-subtle);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  line-height: 1.1rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-endpoint > p {
  min-height: 2rem;
  margin-top: 9px;
  color: var(--workspace-work-text-muted);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  line-height: 1rem;
}

.dashboard-endpoint__actions {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 12px;
}

.dashboard-endpoint-action {
  display: inline-flex;
  min-height: 32px;
  align-items: center;
  gap: 6px;
  padding: 0 9px;
  border: 1px solid var(--workspace-border);
  border-radius: var(--workspace-radius-compact);
  color: var(--workspace-work-text-muted);
  background: var(--workspace-card-surface);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
  transition: color 140ms ease, background-color 140ms ease, border-color 140ms ease;
}

.dashboard-endpoint-action:hover {
  border-color: var(--workspace-work-accent-border);
  color: var(--workspace-work-accent-hover);
  background: var(--workspace-work-accent-soft);
}

.dashboard-endpoint-empty {
  display: flex;
  min-height: 248px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 28px;
  color: var(--workspace-work-text-muted);
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
  text-align: center;
}

.dashboard-endpoint-empty :deep(svg) {
  color: var(--workspace-work-accent);
}

@media (max-width: 767px) {
  .dashboard-panel-link,
  .dashboard-endpoint-action {
    min-height: 44px;
  }

  .dashboard-endpoint-list {
    grid-template-columns: minmax(0, 1fr);
  }

  .dashboard-endpoint:nth-child(even) {
    border-left: 0;
  }

  .dashboard-endpoint + .dashboard-endpoint {
    border-top: 1px solid var(--workspace-border);
  }
}

@media (max-width: 479px) {
  .dashboard-panel-header {
    align-items: flex-start;
    padding: 16px 18px;
  }

  .dashboard-endpoint {
    padding: 18px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .dashboard-endpoint-action {
    transition-duration: 0.01ms;
  }
}

:global(html.dark) .dashboard-panel {
  border-color: var(--workspace-border);
  background: var(--workspace-card-surface);
  box-shadow: none;
}

:global(html.dark) .dashboard-panel-header,
:global(html.dark) .dashboard-endpoint:nth-child(even),
:global(html.dark) .dashboard-endpoint:nth-child(n + 3) {
  border-color: var(--workspace-border);
}

:global(html.dark) .dashboard-panel-header h2,
:global(html.dark) .dashboard-endpoint__heading strong {
  color: var(--workspace-dark-text);
}

:global(html.dark) .dashboard-panel-header p,
:global(html.dark) .dashboard-endpoint > p {
  color: var(--workspace-dark-text-muted);
}

:global(html.dark) .dashboard-endpoint__heading span,
:global(html.dark) .dashboard-endpoint code,
:global(html.dark) .dashboard-endpoint-action {
  border-color: var(--workspace-border);
  color: var(--workspace-dark-text-secondary);
  background: var(--workspace-surface-subtle);
}
</style>

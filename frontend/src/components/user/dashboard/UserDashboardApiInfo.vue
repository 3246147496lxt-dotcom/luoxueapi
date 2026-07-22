<template>
  <aside class="dashboard-panel overflow-hidden" :aria-label="t('dashboard.apiInfo.title')">
    <div class="flex h-[61px] items-center gap-2 border-b border-gray-100 px-5 text-sm font-medium text-gray-800 dark:border-dark-700 dark:text-gray-100">
      <Icon name="server" size="sm" :stroke-width="1.8" />
      <span>{{ t('dashboard.apiInfo.title') }}</span>
    </div>

    <div v-if="endpoints.length > 0" class="divide-y divide-gray-100 dark:divide-dark-700">
      <div v-for="endpoint in endpoints" :key="endpoint.endpoint" class="px-5 py-4">
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <p class="truncate text-sm font-semibold text-gray-800 dark:text-gray-100">
              {{ endpoint.name }}
            </p>
            <a
              :href="endpoint.endpoint"
              target="_blank"
              rel="noopener noreferrer"
              class="mt-1 block truncate text-xs text-indigo-500 hover:text-indigo-600 hover:underline dark:text-indigo-300 dark:hover:text-indigo-200"
              :title="endpoint.endpoint"
            >
              {{ endpoint.endpoint }}
            </a>
            <p v-if="endpoint.description" class="mt-1 truncate text-[11px] text-gray-400 dark:text-dark-500">
              {{ endpoint.description }}
            </p>
          </div>

          <div class="flex shrink-0 items-center gap-1">
            <a
              :href="speedTestUrl(endpoint.endpoint)"
              target="_blank"
              rel="noopener noreferrer"
              class="endpoint-action"
              :title="t('dashboard.apiInfo.speedTest')"
              :aria-label="t('dashboard.apiInfo.speedTestEndpoint', { name: endpoint.name })"
            >
              <Icon name="bolt" size="xs" :stroke-width="1.8" />
            </a>
            <a
              :href="endpoint.endpoint"
              target="_blank"
              rel="noopener noreferrer"
              class="endpoint-action"
              :title="t('dashboard.apiInfo.open')"
              :aria-label="t('dashboard.apiInfo.openEndpoint', { name: endpoint.name })"
            >
              <Icon name="externalLink" size="xs" :stroke-width="1.8" />
            </a>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="flex min-h-[220px] flex-col items-center justify-center px-6 text-center">
      <Icon name="server" size="lg" class="text-gray-300 dark:text-dark-600" />
      <p class="mt-3 text-sm text-gray-400 dark:text-dark-500">{{ t('dashboard.apiInfo.empty') }}</p>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { CustomEndpoint } from '@/types'
import { sanitizeUrl } from '@/utils/url'

const props = defineProps<{
  apiBaseUrl: string
  customEndpoints: CustomEndpoint[]
}>()

const { t } = useI18n()

const endpoints = computed(() => {
  const items: Array<{ name: string; endpoint: string; description: string }> = []
  const seen = new Set<string>()
  const primary = sanitizeUrl(props.apiBaseUrl)
  if (primary) {
    seen.add(primary)
    items.push({
      name: t('dashboard.apiInfo.primary'),
      endpoint: primary,
      description: t('dashboard.apiInfo.primaryDescription'),
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
    })
  }

  return items
})

function speedTestUrl(endpoint: string): string {
  return `https://www.tcptest.cn/http/${encodeURIComponent(endpoint)}`
}
</script>

<style scoped>
.dashboard-panel {
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-surface);
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-form);
}

.endpoint-action {
  display: inline-flex;
  height: 1.75rem;
  width: 1.75rem;
  align-items: center;
  justify-content: center;
  border-radius: 9999px;
  color: rgb(107 114 128);
  transition: color 150ms ease, background-color 150ms ease;
}

.endpoint-action:hover {
  color: rgb(79 105 224);
  background: rgb(239 243 255);
}

:global(.dark) .endpoint-action {
  color: rgb(148 163 184);
}

:global(.dark) .endpoint-action:hover {
  color: rgb(165 180 252);
  background: rgb(49 46 129 / 0.34);
}
</style>

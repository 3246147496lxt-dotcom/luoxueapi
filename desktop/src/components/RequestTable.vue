<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Copy, ExternalLink } from 'lucide-vue-next'
import type { RequestMetadata } from '@/types'

defineProps<{ requests: readonly RequestMetadata[]; compact?: boolean }>()
const { t, locale } = useI18n()

function formatTime(value: string): string {
  return new Intl.DateTimeFormat(locale.value, { hour: '2-digit', minute: '2-digit', second: '2-digit' }).format(new Date(value))
}

function formatTokens(item: RequestMetadata): string {
  return Intl.NumberFormat(locale.value, { notation: 'compact', maximumFractionDigits: 1 }).format(item.inputTokens + item.outputTokens)
}

async function copy(value: string) {
  await navigator.clipboard.writeText(value)
}
</script>

<template>
  <div v-if="requests.length" class="table-wrap">
    <table class="request-table">
      <thead>
        <tr>
          <th>{{ t('requests.time') }}</th>
          <th>{{ t('requests.model') }}</th>
          <th>{{ t('requests.status') }}</th>
          <th>{{ t('requests.tokens') }}</th>
          <th>{{ t('requests.duration') }}</th>
          <th v-if="!compact">{{ t('requests.requestId') }}</th>
          <th><span class="sr-only">{{ t('common.actions') }}</span></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="item in requests" :key="item.id">
          <td class="tabular">{{ formatTime(item.occurredAt) }}</td>
          <td><span class="model-name">{{ item.model }}</span></td>
          <td><span class="http-status" :class="{ 'http-status--error': item.statusCode >= 400 }">{{ item.statusCode }}</span></td>
          <td class="tabular">{{ formatTokens(item) }}</td>
          <td class="tabular">{{ (item.durationMs / 1000).toFixed(1) }}s</td>
          <td v-if="!compact"><code class="request-id">{{ item.requestId }}</code></td>
          <td>
            <button class="icon-button icon-button--small" type="button" :title="t('requests.copyRequestId')" @click="copy(item.requestId)">
              <Copy :size="15" />
            </button>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
  <div v-else class="empty-state">
    <ExternalLink :size="24" />
    <p>{{ t('requests.empty') }}</p>
  </div>
</template>

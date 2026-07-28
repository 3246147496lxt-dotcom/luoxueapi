<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Search, ShieldCheck } from 'lucide-vue-next'
import RequestTable from '@/components/RequestTable.vue'
import type { RequestMetadata } from '@/types'

const props = defineProps<{ requests: readonly RequestMetadata[]; loading: boolean }>()
const emit = defineEmits<{ load: [] }>()
const { t } = useI18n()
const query = ref('')
const status = ref('all')

const filtered = computed(() => props.requests.filter((item) => {
  const matchesQuery = !query.value || item.model.toLowerCase().includes(query.value.toLowerCase()) || item.requestId.includes(query.value)
  const matchesStatus = status.value === 'all' || (status.value === 'success' ? item.statusCode < 400 : item.statusCode >= 400)
  return matchesQuery && matchesStatus
}))

onMounted(() => emit('load'))
</script>

<template>
  <div class="view requests-view">
    <section class="request-toolbar">
      <label class="search-field">
        <Search :size="17" />
        <input v-model="query" type="search" :placeholder="t('requests.searchPlaceholder')" />
      </label>
      <div class="segmented-control" :aria-label="t('requests.statusFilter')">
        <button type="button" :class="{ active: status === 'all' }" @click="status = 'all'">{{ t('common.all') }}</button>
        <button type="button" :class="{ active: status === 'success' }" @click="status = 'success'">2xx</button>
        <button type="button" :class="{ active: status === 'error' }" @click="status = 'error'">4xx / 5xx</button>
      </div>
      <span class="request-count">{{ filtered.length }}</span>
    </section>

    <div class="privacy-banner">
      <ShieldCheck :size="17" />
      <span>{{ t('requests.privacy') }}</span>
    </div>

    <div v-if="loading" class="table-skeleton" :aria-label="t('requests.loading')">
      <span v-for="index in 6" :key="index" />
    </div>
    <RequestTable v-else :requests="filtered" />
  </div>
</template>

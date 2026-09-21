<template>
  <component :is="layoutComponent" v-bind="layoutBindings" class="model-catalog-page">
    <component :is="embedded ? 'div' : 'main'" class="catalog-main">
      <div class="catalog-shell">
        <header class="page-heading"><h1>{{ t('modelCatalog.workspaceTitle') }}</h1></header>
        <section class="filter-panel" :aria-label="t('modelCatalog.filters.ariaLabel')">
          <div class="filter-row" data-filter="platform">
            <h2 class="filter-label">{{ t('modelCatalog.table.platform') }}</h2>
            <div class="filter-options">
              <button type="button" class="filter-chip" data-filter="platform" data-value="all" :class="{ 'is-active': !providerFilter }" :aria-pressed="!providerFilter" @click="selectPlatform('')">{{ t('modelCatalog.filters.all') }}</button>
              <button v-for="provider in providerOptions" :key="provider" type="button" class="filter-chip" data-filter="platform" :data-tone="tone(provider)" :class="{ 'is-active': providerFilter === provider }" :aria-pressed="providerFilter === provider" @click="selectPlatform(provider)">
                <ModelIcon :model="providerIcon(provider)" size="17px" class="platform-icon" />{{ providerLabel(provider) }}
              </button>
            </div>
          </div>
          <div class="filter-row" data-filter="group">
            <h2 class="filter-label">{{ t('modelCatalog.table.group') }}</h2>
            <div class="filter-options">
              <button type="button" class="filter-chip" data-filter="group" data-value="all" :class="{ 'is-active': !groupFilter }" :aria-pressed="!groupFilter" @click="selectGroup('')">{{ t('modelCatalog.filters.all') }}</button>
              <button v-for="group in allGroups" :key="group.id" type="button" class="filter-chip" data-filter="group" :data-tone="tone(group.platform)" :disabled="!groupAvailable(group.id)" :class="{ 'is-active': groupFilter === group.id }" :aria-pressed="groupFilter === group.id" @click="selectGroup(group.id)">{{ group.name }}</button>
            </div>
          </div>
          <div class="filter-row" data-filter="rate">
            <h2 class="filter-label">{{ t('modelCatalog.table.multiplier') }}</h2>
            <div class="filter-options">
              <button type="button" class="filter-chip" data-filter="rate" data-value="all" :class="{ 'is-active': !rateFilter }" :aria-pressed="!rateFilter" @click="rateFilter = ''">{{ t('modelCatalog.filters.all') }}</button>
              <button v-for="rate in rateOptions" :key="rate" type="button" class="filter-chip" data-filter="rate" :class="{ 'is-active': rateFilter === String(rate) }" :aria-pressed="rateFilter === String(rate)" :disabled="!rateAvailable(String(rate))" @click="rateFilter = String(rate)">{{ catalogRateLabel(rate) }}</button>
            </div>
          </div>
          <div class="filter-row filter-search-row">
            <label class="filter-label" for="catalog-search">{{ t('modelCatalog.table.model') }}</label>
            <div class="search-control">
              <Icon name="search" size="sm" class="search-icon" />
              <input id="catalog-search" v-model="searchQuery" type="search" :placeholder="t('modelCatalog.table.searchPlaceholder')" autocomplete="off">
            </div>
            <button type="button" class="reset-button" @click="clearFilters"><Icon name="refresh" size="sm" />{{ t('modelCatalog.table.clear') }}</button>
          </div>
        </section>

        <div v-if="loading" class="empty-state" role="status">{{ t('modelCatalog.loading') }}</div>
        <div v-else-if="errorState" class="empty-state" role="alert">
          <h2>{{ t(`modelCatalog.${errorState}.title`) }}</h2>
          <p>{{ t(`modelCatalog.${errorState}.description`) }}</p>
          <button v-if="errorState === 'error'" type="button" @click="loadCatalog">{{ t('modelCatalog.error.retry') }}</button>
        </div>
        <div v-else-if="!items.length" class="empty-state" role="status">
          <h2>{{ t('modelCatalog.empty.title') }}</h2><p>{{ t('modelCatalog.empty.description') }}</p>
        </div>
        <div v-else-if="!visibleGroups.length" class="empty-state" role="status">
          <h2>{{ t('modelCatalog.noResults.title') }}</h2><p>{{ t('modelCatalog.table.noResults') }}</p>
          <button type="button" @click="clearFilters">{{ t('modelCatalog.table.clear') }}</button>
        </div>
        <div v-else class="group-stack">
          <section v-for="group in visibleGroups" :key="group.id" class="group-card" :data-group-id="group.id" :data-tone="tone(group.platform)" :aria-labelledby="`catalog-group-${group.id}`">
            <header class="group-header">
              <div class="group-chip">
                <ModelIcon :model="providerIcon(group.platform)" size="22px" class="provider-logo" />
                <h2 :id="`catalog-group-${group.id}`" class="group-title">{{ group.name }}</h2>
                <span v-if="group.rate !== null" class="multiplier-badge">{{ catalogRateLabel(group.rate) }}</span>
              </div>
            </header>
            <ModelPriceTable v-for="table in group.tables" :key="table.key" :items="table.items" :timezone="serverTimezone" />
          </section>
        </div>
      </div>
    </component>
  </component>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import PublicSiteLayout from '@/components/public/PublicSiteLayout.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import ModelPriceTable from '@/components/catalog/ModelPriceTable.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { getPublicModelCatalog } from '@/api/catalog'
import type { PublicModelCatalogItem } from '@/api/catalog'
import { catalogComparisonRows, catalogGroupKey, catalogPlatform, catalogRate, catalogRateLabel } from '@/utils/modelCatalogTable'

const props = withDefaults(defineProps<{ embedded?: boolean }>(), { embedded: false })
const layoutComponent = computed(() => props.embedded ? AppLayout : PublicSiteLayout)
const layoutBindings = computed(() => props.embedded ? { contentMode: 'workbench' as const } : { page: 'models' as const })
const { t, te, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
function queryValue(value: unknown): string {
  if (Array.isArray(value)) return typeof value[0] === 'string' ? value[0] : ''
  return typeof value === 'string' ? value : ''
}
const searchQuery = ref(queryValue(route.query.q))
const providerFilter = ref(queryValue(route.query.provider).toLowerCase())
const groupFilter = ref(queryValue(route.query.group))
const rateFilter = ref(queryValue(route.query.rate))
const items = ref<PublicModelCatalogItem[]>([])
const serverTimezone = ref('')
const loading = ref(true)
const errorState = ref<'unavailable' | 'error' | null>(null)
let requestController: AbortController | null = null
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || '落雪API')
const providerOrder: Record<string, number> = { openai: 0, anthropic: 1, grok: 2, antigravity: 3 }
const providerOptions = computed(() => Array.from(new Set(items.value.map(catalogPlatform)))
  .sort((a, b) => (providerOrder[a] ?? 4) - (providerOrder[b] ?? 4)))
const rateOptions = computed(() => Array.from(new Set(items.value.map(catalogRate).filter((r): r is number => r !== null))).sort((a, b) => a - b))
function providerLabel(provider: string) {
  const key = `modelCatalog.providers.${provider.replace(/[^a-z0-9]+/g, '_')}`
  return te(key) ? t(key) : provider
}
function providerIcon(provider: string) {
  if (provider === 'antigravity') return 'gemini'
  return provider === 'domestic' ? (items.value.find(item => catalogPlatform(item) === provider)?.logo_key || 'deepseek') : provider
}
function tone(provider: string) {
  if (['openai', 'gpt'].includes(provider)) return 'openai'
  if (['anthropic', 'claude'].includes(provider)) return 'anthropic'
  if (['grok', 'xai'].includes(provider)) return 'grok'
  return 'domestic'
}
function matchesPlatform(item: PublicModelCatalogItem) {
  return !providerFilter.value || catalogPlatform(item) === providerFilter.value
}
function groupAvailable(id: string) {
  return items.value.some(item => catalogGroupKey(item) === id && matchesPlatform(item))
}
function rateAvailable(rate: string) {
  return items.value.some(item => matchesPlatform(item) && (!groupFilter.value || catalogGroupKey(item) === groupFilter.value) && catalogRate(item) !== null && String(catalogRate(item)) === rate)
}
function normalizeFilters() {
  if (providerFilter.value && !providerOptions.value.includes(providerFilter.value)) providerFilter.value = ''
  if (groupFilter.value && !groupAvailable(groupFilter.value)) groupFilter.value = ''
  if (rateFilter.value && !rateAvailable(rateFilter.value)) rateFilter.value = ''
}
function selectPlatform(provider: string) { providerFilter.value = provider; normalizeFilters() }
function selectGroup(group: string) { groupFilter.value = group; normalizeFilters() }
function clearFilters() { searchQuery.value = ''; providerFilter.value = ''; groupFilter.value = ''; rateFilter.value = '' }

type Group = { id: string; name: string; platform: string; rate: number | null; items: PublicModelCatalogItem[]; tables: { key: string; items: PublicModelCatalogItem[] }[] }
function comparePaidPricesDescending(a: PublicModelCatalogItem, b: PublicModelCatalogItem): number {
  // Match the first displayed context tier and keep each model's tiers together.
  const left = catalogComparisonRows(a)[0]?.paid
  const right = catalogComparisonRows(b)[0]?.paid
  if (a.pricing.billing_mode !== 'token') return (right?.per_request_price ?? -1) - (left?.per_request_price ?? -1)
  return (right?.input_price ?? -1) - (left?.input_price ?? -1)
    || (right?.output_price ?? -1) - (left?.output_price ?? -1)
}
function groupItems(source: PublicModelCatalogItem[]): Group[] {
  const groups = new Map<string, Group>()
  for (const item of source) {
    const id = catalogGroupKey(item)
    let group = groups.get(id)
    if (!group) {
      group = { id, name: item.public_group?.name || providerLabel(catalogPlatform(item)), platform: catalogPlatform(item), rate: catalogRate(item), items: [], tables: [] }
      groups.set(id, group)
    }
    group.items.push(item)
    if (catalogRate(item) !== group.rate) group.rate = null
    // Each header describes one billing unit/currency, including image quotes.
    const key = `${item.pricing.billing_mode}:${item.pricing.currency}:${item.official_pricing?.currency || item.pricing.currency}`
    let table = group.tables.find(table => table.key === key)
    if (!table) { table = { key, items: [] }; group.tables.push(table) }
    table.items.push(item)
  }
  for (const group of groups.values()) {
    if (['anthropic', 'claude'].includes(group.platform)) {
      for (const table of group.tables) table.items.sort(comparePaidPricesDescending)
    }
  }
  return [...groups.values()].sort((a, b) => Math.min(...a.items.map(i => catalogRate(i) ?? Infinity)) - Math.min(...b.items.map(i => catalogRate(i) ?? Infinity)))
}
const allGroups = computed(() => groupItems(items.value))
const visibleGroups = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()
  return groupItems(items.value.filter(item => matchesPlatform(item)
    && (!groupFilter.value || catalogGroupKey(item) === groupFilter.value)
    && (!rateFilter.value || (catalogRate(item) !== null && String(catalogRate(item)) === rateFilter.value))
    && (!query || [item.model, item.display_name, item.summary, item.provider, ...item.tags, ...item.capabilities].join(' ').toLowerCase().includes(query))))
})
watch(() => route.query, query => {
  searchQuery.value = queryValue(query.q)
  providerFilter.value = queryValue(query.provider).toLowerCase()
  groupFilter.value = queryValue(query.group)
  rateFilter.value = queryValue(query.rate)
  if (!loading.value) normalizeFilters()
})
watch([searchQuery, providerFilter, groupFilter, rateFilter], () => {
  const nextQuery = { ...route.query }
  const fields = { q: searchQuery.value, provider: providerFilter.value, group: groupFilter.value, rate: rateFilter.value }
  for (const [key, value] of Object.entries(fields)) {
    if (value) nextQuery[key] = value
    else delete nextQuery[key]
  }
  // Retire filters from the previous card view when writing the new URL.
  for (const key of ['type', 'billing', 'tag', 'priced', 'sort']) delete nextQuery[key]
  if (JSON.stringify(nextQuery) !== JSON.stringify(route.query)) void router.replace({ path: route.path, query: nextQuery }).catch(() => undefined)
})
async function loadCatalog() {
  requestController?.abort()
  const controller = new AbortController()
  requestController = controller
  loading.value = true
  errorState.value = null
  try {
    const response = await getPublicModelCatalog({ signal: controller.signal })
    if (controller.signal.aborted || requestController !== controller) return
    items.value = response.items || []
    serverTimezone.value = response.server_timezone || ''
    normalizeFilters()
  } catch (error) {
    if (controller.signal.aborted || requestController !== controller) return
    const status = (error as { status?: number; response?: { status?: number } }).status ?? (error as { response?: { status?: number } }).response?.status
    errorState.value = status === 404 ? 'unavailable' : 'error'
  } finally {
    if (!controller.signal.aborted && requestController === controller) loading.value = false
  }
}
type ManagedHeadElement = {
  element: HTMLElement
  attribute: 'content' | 'href'
  previousValue: string | null
  created: boolean
}
const managedHead = new Map<string, ManagedHeadElement>()

function updateHeadElement(
  key: string,
  selector: string,
  create: () => HTMLElement,
  attribute: 'content' | 'href',
  value: string
) {
  let managed = managedHead.get(key)
  if (!managed) {
    let element = document.head.querySelector<HTMLElement>(selector)
    const created = !element
    if (!element) {
      element = create()
      document.head.appendChild(element)
    }
    managed = { element, attribute, previousValue: element.getAttribute(attribute), created }
    managedHead.set(key, managed)
  }
  managed.element.setAttribute(attribute, value)
}

function updateDocumentMeta() {
  const description = t('modelCatalog.meta.description')
  const title = `${t('modelCatalog.meta.title')} - ${siteName.value}`
  const canonicalUrl = new URL('/models.html', window.location.origin).toString()

  updateHeadElement('description', 'meta[name="description"]', () => {
    const element = document.createElement('meta')
    element.setAttribute('name', 'description')
    return element
  }, 'content', description)
  updateHeadElement('og:title', 'meta[property="og:title"]', () => {
    const element = document.createElement('meta')
    element.setAttribute('property', 'og:title')
    return element
  }, 'content', title)
  updateHeadElement('og:description', 'meta[property="og:description"]', () => {
    const element = document.createElement('meta')
    element.setAttribute('property', 'og:description')
    return element
  }, 'content', description)
  updateHeadElement('og:type', 'meta[property="og:type"]', () => {
    const element = document.createElement('meta')
    element.setAttribute('property', 'og:type')
    return element
  }, 'content', 'website')
  updateHeadElement('og:url', 'meta[property="og:url"]', () => {
    const element = document.createElement('meta')
    element.setAttribute('property', 'og:url')
    return element
  }, 'content', canonicalUrl)
  updateHeadElement('canonical', 'link[rel="canonical"]', () => {
    const element = document.createElement('link')
    element.setAttribute('rel', 'canonical')
    return element
  }, 'href', canonicalUrl)
}

function restoreDocumentMeta() {
  managedHead.forEach(({ element, attribute, previousValue, created }) => {
    if (created) element.remove()
    else if (previousValue == null) element.removeAttribute(attribute)
    else element.setAttribute(attribute, previousValue)
  })
  managedHead.clear()
}


watch([locale, siteName], () => { if (!props.embedded && managedHead.size) updateDocumentMeta() })
onMounted(() => { if (!props.embedded) updateDocumentMeta(); void loadCatalog() })
onBeforeUnmount(() => { requestController?.abort(); restoreDocumentMeta() })
</script>

<style scoped>
.model-catalog-page { min-width: 0; color: var(--workspace-text); font-family: var(--workspace-font-ui); }
.catalog-main { min-width: 0; min-height: 100%; padding: 28px 28px 48px; background: var(--workspace-canvas, #f8fafc); }
.catalog-shell { width: 100%; max-width: 1280px; margin: 0 auto; min-width: 0; }
.page-heading { margin-bottom: 18px; }
.model-catalog-page:has(.workspace-sidebar-overlay-trigger) .page-heading { padding-left: 36px; }
.page-heading h1 { margin: 0; font-size: 28px; line-height: 36px; font-weight: 600; letter-spacing: -.02em; }
    [data-tone="openai"] {
      --group-accent: #15803d;
      --group-border: #a5d8b8;
      --group-soft: #f2faf5;
      --group-badge: #e5f4eb;
      --group-selected: #b8e6c9;
      --group-selected-ink: #14532d;
      --group-selected-hover: #a6dcb9;
    }
    [data-tone="anthropic"] {
      --group-accent: #b65a25;
      --group-border: #e6c0a9;
      --group-soft: #fdf7f2;
      --group-badge: #faecdf;
      --group-selected: #f4d0b4;
      --group-selected-ink: #7c2d12;
      --group-selected-hover: #edbf9b;
    }
    [data-tone="grok"] {
      --group-accent: #262626;
      --group-border: #d4d4d4;
      --group-soft: #fafafa;
      --group-badge: #f0f0f0;
      --group-selected: #d4d4d4;
      --group-selected-ink: #171717;
      --group-selected-hover: #c3c3c3;
    }
    [data-tone="domestic"] {
      --group-accent: #2563b8;
      --group-border: #b2ccef;
      --group-soft: #f3f7fd;
      --group-badge: #e7effb;
      --group-selected: #bfd5f9;
      --group-selected-ink: #1e3a8a;
      --group-selected-hover: #aac8f4;
    }

    .filter-panel {
      margin-bottom: 16px;
      border: 1px solid var(--workspace-border);
      border-radius: 12px;
      background: var(--workspace-card-surface);
      padding: 16px;
    }
    .filter-row {
      display: flex;
      min-width: 0;
      align-items: flex-start;
      gap: 16px;
    }
    .filter-row + .filter-row { margin-top: 12px; }
    .filter-label {
      flex: 0 0 30px;
      margin: 0;
      padding-top: 8px;
      color: var(--workspace-text-secondary);
      font-size: 12px;
      font-weight: 500;
      line-height: 18px;
    }
    .filter-options {
      display: flex;
      flex: 1 1 auto;
      min-width: 0;
      flex-wrap: wrap;
      align-items: center;
      gap: 8px;
    }
    .filter-chip {
      display: inline-flex;
      min-height: 34px;
      align-items: center;
      justify-content: center;
      gap: 7px;
      max-width: 100%;
      border: 1px solid var(--group-border, #e5e5e5);
      border-radius: 7px;
      background: var(--group-soft, #ffffff);
      padding: 6px 11px;
      color: var(--group-accent, #525252);
      font-size: 12px;
      font-weight: 500;
      line-height: 20px;
      cursor: pointer;
      transition: background-color 200ms cubic-bezier(.2,.7,.2,1), border-color 200ms ease, color 200ms ease;
    }
    .filter-chip:hover:not(:disabled) {
      border-color: var(--group-accent, #737373);
      background: var(--group-badge, #f5f5f5);
    }
    .filter-chip.is-active {
      border-color: var(--group-accent, #70b8a7);
      background: var(--group-badge, #e7f5ef);
      color: var(--group-accent, #0f766e);
      font-weight: 600;
    }
    .filter-chip.is-active:hover:not(:disabled) {
      background: var(--group-badge, #def0e8);
    }
    .filter-chip[data-tone].is-active {
      background: var(--group-selected);
      border-color: var(--group-accent);
      color: var(--group-selected-ink);
    }
    .filter-chip[data-tone].is-active:hover:not(:disabled),
    .filter-chip[data-tone]:active:not(:disabled) {
      background: var(--group-selected-hover);
      border-color: var(--group-accent);
    }
    .filter-chip[data-value="all"].is-active {
      border-color: #f4b4cd;
      background: #fce7f3;
      color: #9d174d;
    }
    .filter-chip[data-value="all"].is-active:hover:not(:disabled) {
      background: #fbcfe8;
    }
    .filter-chip:disabled {
      opacity: .35;
      cursor: not-allowed;
    }
    .platform-icon {
      width: 17px;
      height: 17px;
      flex: 0 0 17px;
      object-fit: contain;
      mix-blend-mode: multiply;
    }
    .filter-chip[data-filter="platform"] { font-size: 13px; padding-inline: 13px; }
    .filter-chip[data-filter="rate"] {
      min-width: 57px;
      font-variant-numeric: tabular-nums;
    }
    .filter-chip[data-value="all"] { min-width: 52px; }
    .filter-search-row {
      padding-top: 12px;
      border-top: 1px solid var(--workspace-divider);
    }
    .search-control {
      position: relative;
      min-width: 150px;
      max-width: 390px;
      flex: 1 1 300px;
    }
    .search-control .search-icon {
      position: absolute;
      top: 50%;
      left: 11px;
      color: var(--workspace-text-muted);
      font-size: 16px;
      transform: translateY(-50%);
      pointer-events: none;
    }
    .search-control input {
      height: 36px;
      width: 100%;
      border: 1px solid var(--workspace-border);
      border-radius: 7px;
      outline: none;
      background: var(--workspace-card-surface);
      padding: 0 12px 0 34px;
      color: var(--workspace-text);
      font-size: 12px;
    }
    .search-control input::placeholder { color: var(--workspace-text-muted); }
    .search-control input:focus { border-color: #93c5fd; box-shadow: 0 0 0 3px rgba(37,99,235,.08); }
    .filter-search-row .reset-button { margin-left: auto; height: 36px; }

    .reset-button {
      display: inline-flex;
      flex: 0 0 auto;
      height: 38px;
      align-items: center;
      justify-content: center;
      gap: 6px;
      border: 0;
      border-radius: 9px;
      background: transparent;
      padding: 0 10px;
      color: var(--workspace-text-secondary);
      font-size: 13px;
      cursor: pointer;
    }

    .reset-button:hover { background: var(--workspace-hover); color: var(--workspace-text); }



    .group-stack { display: grid; gap: 18px; }
    .group-card {
      min-width: 0;
      overflow: hidden;
      border: 1px solid var(--group-border);
      border-radius: 12px;
      background: var(--workspace-card-surface);
    }
    .group-header {
      display: flex;
      min-height: 64px;
      align-items: center;
      padding: 12px 16px;
      border-bottom: 1px solid var(--workspace-border);
      background: var(--workspace-card-surface);
    }
    .group-chip {
      display: inline-flex;
      min-width: 0;
      max-width: 100%;
      align-items: center;
      gap: 8px;
      padding: 6px 9px;
      border-radius: 7px;
      background: var(--group-badge);
      color: var(--group-accent);
    }
    .provider-logo { width: 22px; height: 22px; flex: 0 0 22px; object-fit: contain; mix-blend-mode: multiply; }
    .provider-symbol { display: inline-flex; align-items: center; justify-content: center; width: 22px; height: 22px; flex: 0 0 22px; color: var(--group-accent); font-size: 20px; }
    .group-title { min-width: 0; margin: 0; font-size: 13px; font-weight: 600; line-height: 22px; color: var(--group-accent); }
    .group-detail { font-size: 12px; font-weight: 400; opacity: .85; }
    .multiplier-badge {
      display: inline-flex;
      height: 24px;
      flex: 0 0 auto;
      align-items: center;
      border-radius: 5px;
      padding: 0 8px;
      background: var(--group-selected);
      color: var(--group-selected-ink);
      font-size: 12px;
      font-weight: 600;
      font-variant-numeric: tabular-nums;
      white-space: nowrap;
    }

.empty-state { display: flex; flex-direction: column; min-height: 250px; align-items: center; justify-content: center; border: 1px solid var(--workspace-border); border-radius: 12px; background: var(--workspace-card-surface); padding: 36px 20px; text-align: center; font-size: 13px; }
.empty-state h2 { margin: 10px 0 4px; font-size: 15px; font-weight: 600; }
.empty-state p { margin: 0; color: var(--workspace-text-secondary); }
.empty-state button { min-height: 36px; margin-top: 14px; border: 1px solid var(--workspace-border); border-radius: 8px; background: var(--workspace-card-surface); padding: 0 12px; color: var(--workspace-text); }
.filter-chip { overflow-wrap: anywhere; }
.group-chip { flex-wrap: wrap; }
.provider-logo { flex-shrink: 0; }
:where(button, input):focus-visible { outline: 2px solid var(--workspace-work-accent); outline-offset: 2px; }
.filter-chip[data-tone="grok"]:focus-visible { outline-color: var(--group-accent); }
[data-tone="grok"] :deep(.model-icon path) { fill: currentColor; }
.dark .model-catalog-page [data-tone="openai"] { --group-accent:#75d79c; --group-border:#315842; --group-soft:#182c22; --group-badge:#203c2c; --group-selected:#315e43; --group-selected-ink:#d5f5df; --group-selected-hover:#3b7150; }
.dark .model-catalog-page [data-tone="anthropic"] { --group-accent:#f0b58b; --group-border:#654938; --group-soft:#30251e; --group-badge:#443124; --group-selected:#765137; --group-selected-ink:#ffdfc7; --group-selected-hover:#895e40; }
.dark .model-catalog-page [data-tone="grok"] { --group-accent:#e5e5e5; --group-border:#525252; --group-soft:#242424; --group-badge:#303030; --group-selected:#494949; --group-selected-ink:#fafafa; --group-selected-hover:#595959; }
.dark .model-catalog-page [data-tone="domestic"] { --group-accent:#91b8f6; --group-border:#3e5376; --group-soft:#1d293c; --group-badge:#283b59; --group-selected:#3c5986; --group-selected-ink:#d9e8ff; --group-selected-hover:#486ca2; }
.dark .filter-chip[data-value="all"].is-active { background:#502638; color:#f7c5da; border-color:#96536f; }
.dark .filter-chip[data-value="all"].is-active:hover { background:#663349; }
.dark .filter-chip:not([data-tone]):not(.is-active) { background:var(--workspace-card-surface); color:var(--workspace-text-secondary); border-color:var(--workspace-border); }
.dark .platform-icon, .dark .provider-logo { mix-blend-mode:screen; }
.dark [data-tone="openai"] :deep(.model-icon-image) { filter:invert(1) grayscale(1); }
@media (max-width:1440px) { .catalog-main { padding-inline:20px; } }
@media (max-width:767px) {
  .catalog-main { padding:20px 16px 40px; }
  .filter-panel { padding:12px; }
  .filter-row { gap:10px; }
  .filter-label { flex-basis:25px; font-size:11px; }
  .filter-options { gap:6px; }
  .filter-chip { min-height:34px; padding:6px 8px; font-size:11px; }
  .filter-chip[data-filter="platform"] { font-size:11px; padding-inline:9px; }
  .filter-chip[data-filter="rate"] { min-width:50px; }
  .filter-chip[data-value="all"] { min-width:44px; }
  .filter-search-row { flex-wrap:wrap; }
  .filter-search-row .search-control { max-width:none; }
  .filter-search-row .reset-button { margin-left:35px; height:30px; }
  .group-header { align-items:flex-start; }
}
@media (prefers-reduced-motion:reduce) { *, *::before, *::after { transition:none !important; } }
</style>

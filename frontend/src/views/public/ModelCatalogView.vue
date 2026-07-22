<template>
  <PublicSiteLayout class="model-catalog-page" page="models">
    <main id="top" class="catalog-main">
      <section class="catalog-hero" aria-labelledby="catalog-title">
        <div class="catalog-shell catalog-hero-inner">
          <div class="catalog-heading">
            <div class="catalog-title-line">
              <h1 id="catalog-title">{{ t('modelCatalog.title') }}</h1>
              <span v-if="!loading && !errorState" class="catalog-count-badge">
                {{ t('modelCatalog.modelCount', { count: items.length }) }}
              </span>
            </div>
            <p>{{ t('modelCatalog.description') }}</p>
          </div>
          <div class="catalog-price-note">
            <Icon name="infoCircle" size="sm" aria-hidden="true" />
            <span>{{ t('modelCatalog.publicPriceNote') }}</span>
          </div>
          <p v-if="formattedPricingUpdatedAt" class="catalog-updated-at">
            {{ t('modelCatalog.pricingUpdatedAt', { time: formattedPricingUpdatedAt }) }}
          </p>
        </div>
      </section>

      <section class="catalog-shell catalog-browser" :aria-label="t('modelCatalog.filters.ariaLabel')">
        <div class="catalog-search-wrap">
          <Icon name="search" size="md" aria-hidden="true" />
          <label class="catalog-sr-only" for="catalog-search">{{ t('modelCatalog.searchLabel') }}</label>
          <input
            id="catalog-search"
            v-model="searchQuery"
            type="search"
            autocomplete="off"
            :placeholder="t('modelCatalog.searchPlaceholder')"
          />
        </div>

        <div v-if="providerOptions.length" class="catalog-filter-row">
          <span class="catalog-filter-label">{{ t('modelCatalog.filters.provider') }}</span>
          <div class="catalog-filter-options" role="group" :aria-label="t('modelCatalog.filters.provider')">
            <button
              type="button"
              :class="{ 'is-active': providerFilter === '' }"
              :aria-pressed="providerFilter === ''"
              @click="providerFilter = ''"
            >
              {{ t('modelCatalog.filters.all') }}
            </button>
            <button
              v-for="provider in providerOptions"
              :key="provider"
              type="button"
              :class="{ 'is-active': providerFilter === provider }"
              :aria-pressed="providerFilter === provider"
              @click="providerFilter = provider"
            >
              {{ providerLabel(provider) }}
            </button>
          </div>
        </div>

        <div v-if="categoryOptions.length" class="catalog-filter-row">
          <span class="catalog-filter-label">{{ t('modelCatalog.filters.category') }}</span>
          <div class="catalog-filter-options" role="group" :aria-label="t('modelCatalog.filters.category')">
            <button
              type="button"
              :class="{ 'is-active': categoryFilter === '' }"
              :aria-pressed="categoryFilter === ''"
              @click="categoryFilter = ''"
            >
              {{ t('modelCatalog.filters.all') }}
            </button>
            <button
              v-for="category in categoryOptions"
              :key="category"
              type="button"
              :class="{ 'is-active': categoryFilter === category }"
              :aria-pressed="categoryFilter === category"
              @click="categoryFilter = category"
            >
              {{ categoryLabel(category) }}
            </button>
          </div>
        </div>
      </section>

      <section class="catalog-shell catalog-results" aria-labelledby="catalog-results-title">
        <div class="catalog-results-heading">
          <h2 id="catalog-results-title">{{ t('modelCatalog.resultsTitle') }}</h2>
          <p v-if="!loading && !errorState" aria-live="polite">
            {{ t('modelCatalog.resultCount', { count: filteredItems.length }) }}
          </p>
        </div>

        <div v-if="loading" class="catalog-grid" aria-busy="true" :aria-label="t('modelCatalog.loading')">
          <article v-for="index in 6" :key="index" class="catalog-card catalog-skeleton" aria-hidden="true">
            <span class="skeleton-block skeleton-icon"></span>
            <span class="skeleton-block skeleton-title"></span>
            <span class="skeleton-block skeleton-model"></span>
            <span class="skeleton-block skeleton-copy"></span>
            <span class="skeleton-block skeleton-price"></span>
          </article>
        </div>

        <div v-else-if="errorState" class="catalog-state" role="status">
          <span class="catalog-state-icon" aria-hidden="true">
            <Icon :name="errorState === 'unavailable' ? 'inbox' : 'exclamationCircle'" size="lg" />
          </span>
          <h2>{{ t(errorState === 'unavailable' ? 'modelCatalog.unavailable.title' : 'modelCatalog.error.title') }}</h2>
          <p>{{ t(errorState === 'unavailable' ? 'modelCatalog.unavailable.description' : 'modelCatalog.error.description') }}</p>
          <button v-if="errorState === 'error'" type="button" class="catalog-secondary-button" @click="loadCatalog">
            <Icon name="refresh" size="sm" aria-hidden="true" />
            {{ t('modelCatalog.error.retry') }}
          </button>
        </div>

        <div v-else-if="filteredItems.length === 0" class="catalog-state" role="status">
          <span class="catalog-state-icon" aria-hidden="true"><Icon name="search" size="lg" /></span>
          <h2>{{ t(items.length === 0 ? 'modelCatalog.empty.title' : 'modelCatalog.noResults.title') }}</h2>
          <p>{{ t(items.length === 0 ? 'modelCatalog.empty.description' : 'modelCatalog.noResults.description') }}</p>
          <button
            v-if="items.length > 0"
            type="button"
            class="catalog-secondary-button"
            @click="clearFilters"
          >
            {{ t('modelCatalog.noResults.clear') }}
          </button>
        </div>

        <div v-else class="catalog-grid">
          <article v-for="model in filteredItems" :key="model.slug" class="catalog-card">
            <div class="catalog-card-header">
              <span class="catalog-model-icon" aria-hidden="true">
                <ModelIcon :model="model.logo_key || model.model" size="28px" />
              </span>
              <div class="catalog-card-identity">
                <div class="catalog-provider-line">
                  <span>{{ providerLabel(model.provider) }}</span>
                  <span aria-hidden="true">·</span>
                  <span>{{ categoryLabel(model.category) }}</span>
                  <span v-if="model.featured" class="catalog-featured">
                    {{ t('modelCatalog.featured') }}
                  </span>
                </div>
                <h3>{{ model.display_name }}</h3>
              </div>
            </div>

            <div class="catalog-model-id-row">
              <code :title="model.model">{{ model.model }}</code>
              <button
                type="button"
                class="catalog-copy-button"
                :aria-label="t('modelCatalog.copyModelAria', { model: model.model })"
                @click="copyModelId(model.model)"
              >
                <Icon :name="copiedModel === model.model ? 'check' : 'copy'" size="sm" aria-hidden="true" />
                <span>{{ t(copiedModel === model.model ? 'modelCatalog.copied' : 'modelCatalog.copy') }}</span>
              </button>
            </div>

            <p class="catalog-summary">{{ model.summary }}</p>

            <dl class="catalog-facts">
              <div v-if="model.context_window != null">
                <dt>{{ t('modelCatalog.contextWindow') }}</dt>
                <dd>{{ formatTokenCount(model.context_window) }}</dd>
              </div>
              <div v-if="model.max_output_tokens != null">
                <dt>{{ t('modelCatalog.maxOutput') }}</dt>
                <dd>{{ formatTokenCount(model.max_output_tokens) }}</dd>
              </div>
            </dl>

            <ul v-if="visibleCapabilities(model).length" class="catalog-capabilities" :aria-label="t('modelCatalog.capabilities')">
              <li v-for="capability in visibleCapabilities(model)" :key="capability">
                {{ capabilityLabel(capability) }}
              </li>
              <li v-if="allCapabilities(model).length > visibleCapabilities(model).length">
                +{{ allCapabilities(model).length - visibleCapabilities(model).length }}
              </li>
            </ul>

            <div class="catalog-pricing">
              <div class="catalog-pricing-heading">
                <span>{{ model.pricing.label || t('modelCatalog.pricing.publicLabel') }}</span>
                <button type="button" @click="openPricingDetails(model)">
                  {{ t('modelCatalog.pricing.details') }}
                  <Icon name="chevronRight" size="xs" aria-hidden="true" />
                </button>
              </div>
              <div v-if="priceSummaryRows(model.pricing).length" class="catalog-price-grid">
                <div v-for="row in priceSummaryRows(model.pricing)" :key="row.label">
                  <span>{{ row.label }}</span>
                  <strong>{{ row.value }}</strong>
                  <small>{{ row.unit }}</small>
                </div>
              </div>
              <p v-else class="catalog-price-unknown">{{ t('modelCatalog.pricing.unknown') }}</p>
            </div>
          </article>
        </div>
      </section>

      <section v-if="!errorState" class="catalog-shell catalog-cta" aria-labelledby="catalog-cta-title">
        <div>
          <h2 id="catalog-cta-title">{{ t('modelCatalog.cta.title') }}</h2>
          <p>{{ t('modelCatalog.cta.description') }}</p>
        </div>
        <div class="catalog-cta-actions">
          <router-link :to="primaryCta.to" class="catalog-primary-button">
            {{ primaryCta.label }}
            <Icon name="arrowRight" size="sm" aria-hidden="true" />
          </router-link>
          <a v-if="tutorialUrl" :href="tutorialUrl" class="catalog-secondary-link">
            {{ t('modelCatalog.cta.tutorial') }}
          </a>
        </div>
      </section>
    </main>

    <dialog
      v-if="selectedModel"
      ref="pricingDialogRef"
      class="catalog-price-dialog"
      :aria-labelledby="`pricing-dialog-${selectedModel.slug}`"
      @close="selectedModel = null"
    >
      <div class="catalog-dialog-header">
        <div>
          <p>{{ selectedModel.display_name }}</p>
          <h2 :id="`pricing-dialog-${selectedModel.slug}`">{{ t('modelCatalog.pricing.dialogTitle') }}</h2>
        </div>
        <button type="button" :aria-label="t('common.close')" @click="closePricingDetails">
          <Icon name="x" size="md" aria-hidden="true" />
        </button>
      </div>

      <div class="catalog-dialog-body">
        <p class="catalog-dialog-note">{{ t('modelCatalog.pricing.dialogDescription') }}</p>

        <dl class="catalog-price-details">
          <div>
            <dt>{{ t('modelCatalog.pricing.billingMode') }}</dt>
            <dd>{{ billingModeLabel(selectedModel.pricing.billing_mode) }}</dd>
          </div>
          <div v-for="row in detailPriceRows(selectedModel.pricing)" :key="row.label">
            <dt>{{ row.label }}</dt>
            <dd>{{ row.value }} <small>{{ row.unit }}</small></dd>
          </div>
        </dl>

        <div v-if="selectedModel.pricing.intervals.length" class="catalog-intervals">
          <h3>{{ t('modelCatalog.pricing.intervalTitle') }}</h3>
          <div
            v-for="(interval, intervalIndex) in selectedModel.pricing.intervals"
            :key="`${interval.tier_label || 'range'}-${interval.min_tokens}-${interval.max_tokens}-${intervalIndex}`"
            class="catalog-interval-row"
          >
            <div>
              <strong>{{ intervalLabel(interval) }}</strong>
              <span v-if="interval.tier_label && selectedModel.pricing.billing_mode === BILLING_MODE_TOKEN">
                {{ formatIntervalRange(interval) }}
              </span>
            </div>
            <span>{{ formatIntervalPrice(interval, selectedModel.pricing.billing_mode) }}</span>
          </div>
        </div>

        <div v-if="selectedModel.pricing.peak_rate.enabled" class="catalog-peak-note">
          <Icon name="clock" size="sm" aria-hidden="true" />
          <span>
            {{ t('modelCatalog.pricing.peakRate', {
              start: selectedModel.pricing.peak_rate.start,
              end: selectedModel.pricing.peak_rate.end,
              multiplier: selectedModel.pricing.peak_rate.multiplier,
              timezone: serverTimezone || 'UTC'
            }) }}
          </span>
        </div>
      </div>
    </dialog>
  </PublicSiteLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import PublicSiteLayout from '@/components/public/PublicSiteLayout.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore, useAuthStore } from '@/stores'
import { useClipboard } from '@/composables/useClipboard'
import { getPublicModelCatalog } from '@/api/catalog'
import type {
  PublicModelCatalogItem,
  PublicModelCatalogPricing,
  PublicModelCatalogPricingInterval
} from '@/api/catalog'
import type { BillingMode } from '@/constants/channel'
import { BILLING_MODE_IMAGE, BILLING_MODE_PER_REQUEST, BILLING_MODE_TOKEN } from '@/constants/channel'
import { resolveTutorialUrl } from '@/utils/documentationUrl'
import { formatScaled } from '@/utils/pricing'
import { sanitizeUrl } from '@/utils/url'

type ErrorState = 'unavailable' | 'error' | null

const { t, te, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()
const { copyToClipboard } = useClipboard()

function queryValue(value: unknown): string {
  if (Array.isArray(value)) return typeof value[0] === 'string' ? value[0] : ''
  return typeof value === 'string' ? value : ''
}

const searchQuery = ref(queryValue(route.query.q))
const providerFilter = ref(queryValue(route.query.provider).toLowerCase())
const categoryFilter = ref(queryValue(route.query.type).toLowerCase())
const items = ref<PublicModelCatalogItem[]>([])
const serverTimezone = ref('')
const pricingUpdatedAt = ref('')
const loading = ref(true)
const errorState = ref<ErrorState>(null)
const copiedModel = ref('')
const selectedModel = ref<PublicModelCatalogItem | null>(null)
const pricingDialogRef = ref<HTMLDialogElement | null>(null)
let requestController: AbortController | null = null
let copiedResetTimer: ReturnType<typeof setTimeout> | null = null

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || '落雪API')
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const tutorialUrl = computed(() => resolveTutorialUrl(docUrl.value))
const providerOptions = computed(() => Array.from(new Set(
  items.value.map((item) => item.provider.trim().toLowerCase()).filter(Boolean)
)).sort((a, b) => providerLabel(a).localeCompare(providerLabel(b), locale.value)))
const categoryOptions = computed(() => Array.from(new Set(
  items.value.map((item) => item.category.trim().toLowerCase()).filter(Boolean)
)).sort((a, b) => categoryLabel(a).localeCompare(categoryLabel(b), locale.value)))

const filteredItems = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()
  return items.value.filter((item) => {
    if (providerFilter.value && item.provider.trim().toLowerCase() !== providerFilter.value) return false
    if (categoryFilter.value && item.category.trim().toLowerCase() !== categoryFilter.value) return false
    if (!query) return true
    const searchable = [
      item.model,
      item.display_name,
      item.summary,
      item.provider,
      item.category,
      ...item.tags,
      ...item.capabilities
    ].join(' ').toLowerCase()
    return searchable.includes(query)
  })
})

const formattedPricingUpdatedAt = computed(() => {
  if (!pricingUpdatedAt.value) return ''
  const date = new Date(pricingUpdatedAt.value)
  if (Number.isNaN(date.getTime())) return ''
  return new Intl.DateTimeFormat(locale.value === 'zh' ? 'zh-CN' : 'en-US', {
    dateStyle: 'medium',
    timeStyle: 'short'
  }).format(date)
})

const primaryCta = computed(() => {
  if (authStore.isAuthenticated) {
    if (!authStore.isAdmin && appStore.cachedPublicSettings?.available_channels_enabled === true) {
      return { to: '/available-channels', label: t('modelCatalog.cta.availableChannels') }
    }
    return {
      to: authStore.isAdmin ? '/admin/dashboard' : '/dashboard',
      label: t('modelCatalog.cta.dashboard')
    }
  }
  if (appStore.cachedPublicSettings?.registration_enabled === true) {
    return { to: '/register', label: t('modelCatalog.cta.register') }
  }
  return { to: '/login', label: t('modelCatalog.cta.login') }
})

watch(
  () => route.query,
  (query) => {
    const nextSearch = queryValue(query.q)
    const nextProvider = queryValue(query.provider).toLowerCase()
    const nextCategory = queryValue(query.type).toLowerCase()
    if (searchQuery.value !== nextSearch) searchQuery.value = nextSearch
    if (providerFilter.value !== nextProvider) providerFilter.value = nextProvider
    if (categoryFilter.value !== nextCategory) categoryFilter.value = nextCategory
  },
  { deep: true }
)

watch([searchQuery, providerFilter, categoryFilter], () => {
  const nextQuery = { ...route.query }
  if (searchQuery.value) nextQuery.q = searchQuery.value
  else delete nextQuery.q
  if (providerFilter.value) nextQuery.provider = providerFilter.value
  else delete nextQuery.provider
  if (categoryFilter.value) nextQuery.type = categoryFilter.value
  else delete nextQuery.type

  if (
    queryValue(route.query.q) === queryValue(nextQuery.q)
    && queryValue(route.query.provider) === queryValue(nextQuery.provider)
    && queryValue(route.query.type) === queryValue(nextQuery.type)
  ) return

  void router.replace({ path: route.path, query: nextQuery }).catch(() => undefined)
})

async function loadCatalog() {
  requestController?.abort()
  requestController = new AbortController()
  loading.value = true
  errorState.value = null

  try {
    const response = await getPublicModelCatalog({ signal: requestController.signal })
    items.value = Array.isArray(response.items) ? response.items : []
    serverTimezone.value = response.server_timezone || ''
    pricingUpdatedAt.value = response.pricing_updated_at || ''
  } catch (error) {
    if (requestController.signal.aborted) return
    const status = (error as { status?: number; response?: { status?: number } }).status
      ?? (error as { response?: { status?: number } }).response?.status
    errorState.value = status === 404 ? 'unavailable' : 'error'
  } finally {
    if (!requestController.signal.aborted) loading.value = false
  }
}

function clearFilters() {
  searchQuery.value = ''
  providerFilter.value = ''
  categoryFilter.value = ''
}

async function copyModelId(model: string) {
  if (copiedResetTimer) clearTimeout(copiedResetTimer)
  const copied = await copyToClipboard(model, t('modelCatalog.copySuccess'))
  copiedModel.value = copied ? model : ''
  if (copied) {
    copiedResetTimer = setTimeout(() => {
      copiedModel.value = ''
    }, 2000)
  }
}

function providerLabel(provider: string): string {
  const key = provider.trim().toLowerCase().replace(/[^a-z0-9]+/g, '_')
  const translationKey = `modelCatalog.providers.${key}`
  return te(translationKey) ? t(translationKey) : provider
}

function categoryLabel(category: string): string {
  const normalized = category.trim().toLowerCase().replace(/[^a-z0-9]+/g, '_') || 'other'
  const translationKey = `modelCatalog.categories.${normalized}`
  return te(translationKey) ? t(translationKey) : category
}

function capabilityLabel(capability: string): string {
  const normalized = capability.trim().toLowerCase().replace(/[^a-z0-9]+/g, '_')
  const translationKey = `modelCatalog.capabilityLabels.${normalized}`
  return te(translationKey) ? t(translationKey) : capability.split('_').join(' ')
}

function allCapabilities(model: PublicModelCatalogItem): string[] {
  return Array.from(new Set([...model.tags, ...model.capabilities].filter(Boolean)))
}

function visibleCapabilities(model: PublicModelCatalogItem): string[] {
  return allCapabilities(model).slice(0, 5)
}

function formatTokenCount(value: number): string {
  if (value >= 1_000_000 && value % 1_000_000 === 0) return `${value / 1_000_000}M`
  if (value >= 1_000 && value % 1_000 === 0) return `${value / 1_000}K`
  return new Intl.NumberFormat(locale.value === 'zh' ? 'zh-CN' : 'en-US').format(value)
}

function priceSummaryRows(pricing: PublicModelCatalogPricing) {
  if (pricing.billing_mode === BILLING_MODE_TOKEN) {
    return [
      {
        label: t('modelCatalog.pricing.input'),
        value: formatScaled(pricing.input_price, 1_000_000),
        unit: t('modelCatalog.pricing.perMillionTokens')
      },
      {
        label: t('modelCatalog.pricing.output'),
        value: formatScaled(pricing.output_price, 1_000_000),
        unit: t('modelCatalog.pricing.perMillionTokens')
      }
    ].filter((row) => row.value !== '-')
  }

  if (pricing.per_request_price != null) {
    return [{
      label: t(pricing.billing_mode === BILLING_MODE_IMAGE
        ? 'modelCatalog.pricing.image'
        : 'modelCatalog.pricing.request'),
      value: formatScaled(pricing.per_request_price, 1),
      unit: t(pricing.billing_mode === BILLING_MODE_IMAGE
        ? 'modelCatalog.pricing.perImage'
        : 'modelCatalog.pricing.perRequest')
    }]
  }

  return []
}

function detailPriceRows(pricing: PublicModelCatalogPricing) {
  const tokenUnit = t('modelCatalog.pricing.perMillionTokens')
  const rows = [
    { key: 'input_price', label: t('modelCatalog.pricing.input'), value: pricing.input_price, scale: 1_000_000, unit: tokenUnit },
    { key: 'output_price', label: t('modelCatalog.pricing.output'), value: pricing.output_price, scale: 1_000_000, unit: tokenUnit },
    { key: 'cache_write_price', label: t('modelCatalog.pricing.cacheWrite'), value: pricing.cache_write_price, scale: 1_000_000, unit: tokenUnit },
    { key: 'cache_write_1h_price', label: t('modelCatalog.pricing.cacheWrite1h'), value: pricing.cache_write_1h_price, scale: 1_000_000, unit: tokenUnit },
    { key: 'cache_read_price', label: t('modelCatalog.pricing.cacheRead'), value: pricing.cache_read_price, scale: 1_000_000, unit: tokenUnit },
    { key: 'priority_input_price', label: t('modelCatalog.pricing.priorityInput'), value: pricing.priority_input_price, scale: 1_000_000, unit: tokenUnit },
    { key: 'priority_output_price', label: t('modelCatalog.pricing.priorityOutput'), value: pricing.priority_output_price, scale: 1_000_000, unit: tokenUnit },
    { key: 'priority_cache_write_price', label: t('modelCatalog.pricing.priorityCacheWrite'), value: pricing.priority_cache_write_price, scale: 1_000_000, unit: tokenUnit },
    { key: 'priority_cache_read_price', label: t('modelCatalog.pricing.priorityCacheRead'), value: pricing.priority_cache_read_price, scale: 1_000_000, unit: tokenUnit },
    { key: 'image_input_price', label: t('modelCatalog.pricing.imageInput'), value: pricing.image_input_price, scale: 1_000_000, unit: tokenUnit },
    { key: 'image_output_price', label: t('modelCatalog.pricing.imageOutput'), value: pricing.image_output_price, scale: 1_000_000, unit: tokenUnit },
    {
      key: 'per_request_price',
      label: t(pricing.billing_mode === BILLING_MODE_IMAGE ? 'modelCatalog.pricing.image' : 'modelCatalog.pricing.request'),
      value: pricing.per_request_price,
      scale: 1,
      unit: t(pricing.billing_mode === BILLING_MODE_IMAGE ? 'modelCatalog.pricing.perImage' : 'modelCatalog.pricing.perRequest')
    }
  ]
  return rows
    .filter((row) => row.value != null)
    .map((row) => ({ ...row, value: formatScaled(row.value, row.scale) }))
}

function billingModeLabel(mode: BillingMode): string {
  if (mode === BILLING_MODE_TOKEN) return t('modelCatalog.pricing.billingModes.token')
  if (mode === BILLING_MODE_IMAGE) return t('modelCatalog.pricing.billingModes.image')
  if (mode === BILLING_MODE_PER_REQUEST) return t('modelCatalog.pricing.billingModes.perRequest')
  return mode
}

function formatIntervalRange(interval: PublicModelCatalogPricingInterval): string {
  const maximum = interval.max_tokens == null ? '∞' : formatTokenCount(interval.max_tokens)
  return `(${formatTokenCount(interval.min_tokens)}, ${maximum}] Token`
}

function intervalLabel(interval: PublicModelCatalogPricingInterval): string {
  if (interval.tier_label === 'long_context') return t('modelCatalog.pricing.longContext')
  return interval.tier_label || formatIntervalRange(interval)
}

function formatIntervalPrice(interval: PublicModelCatalogPricingInterval, mode: BillingMode): string {
  if (mode === BILLING_MODE_PER_REQUEST || mode === BILLING_MODE_IMAGE) {
    return `${formatScaled(interval.per_request_price, 1)} ${t(mode === BILLING_MODE_IMAGE ? 'modelCatalog.pricing.perImage' : 'modelCatalog.pricing.perRequest')}`
  }
  const unit = t('modelCatalog.pricing.perMillionTokens')
  const parts: string[] = []
  if (interval.input_price != null || interval.output_price != null) {
    parts.push(`${t('modelCatalog.pricing.input')}/${t('modelCatalog.pricing.output')} ${formatScaled(interval.input_price, 1_000_000)} / ${formatScaled(interval.output_price, 1_000_000)} ${unit}`)
  }
  if (interval.cache_write_price != null) {
    parts.push(`${t('modelCatalog.pricing.cacheWrite')} ${formatScaled(interval.cache_write_price, 1_000_000)} ${unit}`)
  }
  if (interval.cache_write_1h_price != null) {
    parts.push(`${t('modelCatalog.pricing.cacheWrite1h')} ${formatScaled(interval.cache_write_1h_price, 1_000_000)} ${unit}`)
  }
  if (interval.cache_read_price != null) {
    parts.push(`${t('modelCatalog.pricing.cacheRead')} ${formatScaled(interval.cache_read_price, 1_000_000)} ${unit}`)
  }
  return parts.join(' · ') || '—'
}

async function openPricingDetails(model: PublicModelCatalogItem) {
  selectedModel.value = model
  await nextTick()
  const dialog = pricingDialogRef.value
  if (!dialog) return
  if (typeof dialog.showModal === 'function') dialog.showModal()
  else dialog.setAttribute('open', '')
}

function closePricingDetails() {
  const dialog = pricingDialogRef.value
  if (!dialog) return
  if (typeof dialog.close === 'function') dialog.close()
  else {
    dialog.removeAttribute('open')
    selectedModel.value = null
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

watch([locale, siteName], () => {
  if (managedHead.size) updateDocumentMeta()
})

onMounted(() => {
  updateDocumentMeta()
  void loadCatalog()
})

onBeforeUnmount(() => {
  requestController?.abort()
  if (copiedResetTimer) clearTimeout(copiedResetTimer)
  restoreDocumentMeta()
})
</script>

<style scoped>
.model-catalog-page {
  min-height: 100vh;
}

.model-catalog-page :where(a, button, input):focus-visible {
  outline: 3px solid color-mix(in srgb, var(--accent) 72%, white);
  outline-offset: 3px;
}

.catalog-sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

.catalog-shell {
  width: min(1120px, calc(100% - 48px));
  margin-inline: auto;
}

.catalog-main {
  min-height: calc(100vh - 145px);
}

.catalog-hero {
  border-bottom: 1px solid var(--border);
  padding-block: clamp(62px, 8vw, 92px) clamp(46px, 6vw, 68px);
}

.catalog-hero-inner {
  display: grid;
  justify-items: center;
  text-align: center;
}

.catalog-heading {
  max-width: 720px;
}

.catalog-title-line {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-wrap: wrap;
  gap: 14px;
}

.catalog-title-line h1 {
  margin: 0;
  color: var(--ink);
  font-size: clamp(2.3rem, 5vw, 4rem);
  font-weight: 760;
  letter-spacing: -0.038em;
  line-height: 1.08;
  text-wrap: balance;
}

.catalog-count-badge {
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--surface-soft);
  color: var(--copy);
  padding: 5px 10px;
  font-size: 12px;
  font-weight: 680;
}

.catalog-heading > p {
  max-width: 65ch;
  margin: 22px auto 0;
  color: var(--copy);
  font-size: 16px;
  line-height: 1.75;
  text-wrap: pretty;
}

.catalog-price-note {
  max-width: 680px;
  display: flex;
  align-items: flex-start;
  gap: 9px;
  margin-top: 28px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-accent);
  color: var(--accent-ink);
  padding: 11px 14px;
  font-size: 13px;
  line-height: 1.65;
  text-align: left;
}

.catalog-price-note svg {
  flex: 0 0 auto;
  margin-top: 2px;
}

.catalog-updated-at {
  margin: 12px 0 0;
  color: var(--muted);
  font-size: 12px;
}

.catalog-browser {
  display: grid;
  gap: 20px;
  padding-block: 40px 32px;
}

.catalog-search-wrap {
  position: relative;
  max-width: 760px;
  width: 100%;
  margin-inline: auto;
}

.catalog-search-wrap > svg {
  position: absolute;
  top: 50%;
  left: 16px;
  color: var(--muted);
  transform: translateY(-50%);
  pointer-events: none;
}

.catalog-search-wrap input {
  width: 100%;
  min-height: 50px;
  box-sizing: border-box;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface);
  color: var(--ink);
  padding: 0 46px;
  font: inherit;
  font-size: 14px;
  transition: border-color 150ms ease, background-color 150ms ease;
}

.catalog-search-wrap input::placeholder {
  color: var(--copy);
  opacity: 1;
}

.catalog-search-wrap input:hover,
.catalog-search-wrap input:focus {
  border-color: var(--accent);
}

.catalog-filter-row {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr);
  align-items: start;
  gap: 14px;
}

.catalog-filter-label {
  padding-top: 8px;
  color: var(--muted);
  font-size: 12px;
  font-weight: 680;
}

.catalog-filter-options {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.catalog-filter-options button {
  min-height: 36px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: var(--copy);
  padding: 0 13px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 620;
  transition: background-color 150ms ease, border-color 150ms ease, color 150ms ease;
}

.catalog-filter-options button:hover {
  border-color: var(--border);
  color: var(--ink);
}

.catalog-filter-options button.is-active {
  border-color: var(--border);
  background: var(--surface-soft);
  color: var(--ink);
}

.catalog-results {
  padding-block: 18px clamp(76px, 9vw, 108px);
}

.catalog-results-heading {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 20px;
  border-bottom: 1px solid var(--border);
  padding-bottom: 14px;
}

.catalog-results-heading h2 {
  margin: 0;
  color: var(--ink);
  font-size: 16px;
  font-weight: 700;
}

.catalog-results-heading p {
  margin: 0;
  color: var(--muted);
  font-size: 13px;
}

.catalog-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}

.catalog-card {
  min-width: 0;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface);
  padding: 20px;
  transition: border-color 160ms ease, transform 160ms cubic-bezier(0.22, 1, 0.36, 1);
}

.catalog-card:hover {
  border-color: color-mix(in srgb, var(--accent) 48%, var(--border));
  transform: translateY(-2px);
}

.catalog-card-header {
  display: flex;
  align-items: flex-start;
  gap: 13px;
}

.catalog-model-icon {
  width: 44px;
  height: 44px;
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: #ffffff;
}

.catalog-card-identity {
  min-width: 0;
}

.catalog-provider-line {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 5px;
  color: var(--muted);
  font-size: 11px;
  font-weight: 620;
}

.catalog-featured {
  border-radius: 999px;
  background: var(--surface-accent);
  color: var(--accent-ink);
  padding: 2px 6px;
}

.catalog-card h3 {
  overflow-wrap: anywhere;
  margin: 5px 0 0;
  color: var(--ink);
  font-size: 17px;
  font-weight: 700;
  letter-spacing: -0.018em;
  line-height: 1.4;
  text-wrap: balance;
}

.catalog-model-id-row {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 17px;
}

.catalog-model-id-row code {
  min-width: 0;
  flex: 1 1 auto;
  overflow: hidden;
  border-radius: 6px;
  background: var(--surface-soft);
  color: var(--copy);
  padding: 7px 9px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.catalog-copy-button {
  min-height: 32px;
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  border: 1px solid var(--border);
  border-radius: 7px;
  background: var(--surface);
  color: var(--copy);
  padding: 0 8px;
  cursor: pointer;
  font-size: 11px;
  font-weight: 680;
}

.catalog-summary {
  min-height: 66px;
  margin: 16px 0 0;
  color: var(--copy);
  font-size: 13px;
  line-height: 1.7;
  text-wrap: pretty;
}

.catalog-facts {
  display: flex;
  flex-wrap: wrap;
  gap: 10px 20px;
  margin: 16px 0 0;
}

.catalog-facts > div {
  display: grid;
  gap: 2px;
}

.catalog-facts dt {
  color: var(--muted);
  font-size: 10px;
}

.catalog-facts dd {
  margin: 0;
  color: var(--ink);
  font-size: 13px;
  font-weight: 680;
}

.catalog-capabilities {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin: 16px 0 0;
  padding: 0;
  list-style: none;
}

.catalog-capabilities li {
  border: 1px solid var(--border);
  border-radius: 999px;
  color: var(--copy);
  padding: 3px 8px;
  font-size: 10px;
  line-height: 1.4;
}

.catalog-pricing {
  margin-top: auto;
  padding-top: 20px;
}

.catalog-pricing-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  border-top: 1px solid var(--border);
  padding-top: 15px;
}

.catalog-pricing-heading > span {
  color: var(--ink);
  font-size: 12px;
  font-weight: 680;
}

.catalog-pricing-heading button {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  border: 0;
  background: transparent;
  color: var(--accent-ink);
  padding: 5px 0;
  cursor: pointer;
  font-size: 11px;
  font-weight: 680;
}

.catalog-price-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  margin-top: 13px;
}

.catalog-price-grid > div {
  min-width: 0;
  display: grid;
  gap: 2px;
}

.catalog-price-grid span,
.catalog-price-grid small {
  color: var(--muted);
  font-size: 10px;
}

.catalog-price-grid strong {
  color: var(--ink);
  font-size: 18px;
  font-weight: 720;
  letter-spacing: -0.02em;
}

.catalog-price-unknown {
  margin: 13px 0 0;
  color: var(--muted);
  font-size: 13px;
}

.catalog-state {
  min-height: 330px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 44px 24px;
  text-align: center;
}

.catalog-state-icon {
  width: 48px;
  height: 48px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: var(--surface-soft);
  color: var(--copy);
}

.catalog-state h2 {
  margin: 18px 0 0;
  color: var(--ink);
  font-size: 18px;
  font-weight: 700;
}

.catalog-state p {
  max-width: 50ch;
  margin: 9px 0 0;
  color: var(--copy);
  font-size: 14px;
  line-height: 1.7;
}

.catalog-secondary-button,
.catalog-primary-button {
  min-height: 44px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border-radius: 10px;
  padding: 0 16px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 700;
  text-decoration: none;
}

.catalog-secondary-button {
  margin-top: 20px;
  border: 1px solid var(--border);
  background: var(--surface);
  color: var(--ink);
}

.catalog-primary-button {
  border: 1px solid var(--ink);
  background: var(--ink);
  color: var(--page);
}

.catalog-cta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 32px;
  margin-bottom: clamp(70px, 9vw, 104px);
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface-soft);
  padding: clamp(28px, 5vw, 46px);
}

.catalog-cta h2 {
  margin: 0;
  color: var(--ink);
  font-size: clamp(1.5rem, 3vw, 2rem);
  font-weight: 720;
  letter-spacing: -0.025em;
  text-wrap: balance;
}

.catalog-cta p {
  max-width: 58ch;
  margin: 10px 0 0;
  color: var(--copy);
  font-size: 14px;
  line-height: 1.7;
}

.catalog-cta-actions {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  gap: 16px;
}

.catalog-secondary-link {
  min-height: 44px;
  display: inline-flex;
  align-items: center;
  color: var(--accent-ink);
  font-size: 13px;
  font-weight: 680;
  text-decoration: none;
}

.catalog-price-dialog {
  width: min(620px, calc(100% - 32px));
  max-height: min(760px, calc(100vh - 48px));
  overflow: auto;
  box-sizing: border-box;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface);
  color: var(--ink);
  padding: 0;
  box-shadow: 0 24px 68px rgba(15, 23, 42, 0.2);
}

.catalog-price-dialog::backdrop {
  background: rgba(15, 23, 42, 0.56);
}

.public-site-page--dark .catalog-price-dialog {
  box-shadow: 0 24px 68px rgba(0, 0, 0, 0.48);
}

.catalog-dialog-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  border-bottom: 1px solid var(--border);
  padding: 20px 22px;
}

.catalog-dialog-header p {
  margin: 0 0 3px;
  color: var(--muted);
  font-size: 12px;
}

.catalog-dialog-header h2 {
  margin: 0;
  color: var(--ink);
  font-size: 19px;
  font-weight: 700;
}

.catalog-dialog-header button {
  width: 40px;
  height: 40px;
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: var(--copy);
  cursor: pointer;
}

.catalog-dialog-header button:hover {
  border-color: var(--border);
  background: var(--surface-soft);
}

.catalog-dialog-body {
  padding: 22px;
}

.catalog-dialog-note {
  margin: 0 0 20px;
  color: var(--copy);
  font-size: 13px;
  line-height: 1.7;
}

.catalog-price-details {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1px;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--border);
}

.catalog-price-details > div {
  display: grid;
  gap: 4px;
  background: var(--surface);
  padding: 13px;
}

.catalog-price-details dt {
  color: var(--muted);
  font-size: 11px;
}

.catalog-price-details dd {
  margin: 0;
  color: var(--ink);
  font-size: 14px;
  font-weight: 680;
}

.catalog-price-details small {
  color: var(--muted);
  font-size: 10px;
  font-weight: 400;
}

.catalog-intervals {
  margin-top: 24px;
}

.catalog-intervals h3 {
  margin: 0 0 10px;
  color: var(--ink);
  font-size: 14px;
  font-weight: 700;
}

.catalog-interval-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  border-top: 1px solid var(--border);
  padding-block: 11px;
  color: var(--copy);
  font-size: 12px;
}

.catalog-interval-row > div {
  display: grid;
  gap: 2px;
}

.catalog-interval-row strong {
  color: var(--ink);
}

.catalog-interval-row > span {
  text-align: right;
}

.catalog-peak-note {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-top: 22px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-accent);
  color: var(--accent-ink);
  padding: 12px;
  font-size: 12px;
  line-height: 1.6;
}

.catalog-peak-note svg {
  flex: 0 0 auto;
  margin-top: 2px;
}

.catalog-skeleton {
  min-height: 360px;
  pointer-events: none;
}

.skeleton-block {
  display: block;
  border-radius: 6px;
  background: var(--surface-soft);
  animation: catalog-pulse 1.35s ease-in-out infinite;
}

.skeleton-icon {
  width: 44px;
  height: 44px;
}

.skeleton-title {
  width: 58%;
  height: 18px;
  margin-top: 18px;
}

.skeleton-model {
  width: 84%;
  height: 32px;
  margin-top: 18px;
}

.skeleton-copy {
  width: 100%;
  height: 64px;
  margin-top: 16px;
}

.skeleton-price {
  width: 100%;
  height: 74px;
  margin-top: auto;
}

@keyframes catalog-pulse {
  0%, 100% { opacity: 0.62; }
  50% { opacity: 1; }
}

@media (max-width: 960px) {
  .catalog-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .catalog-cta {
    align-items: flex-start;
    flex-direction: column;
  }
}

@media (max-width: 767px) {
  .catalog-shell {
    width: min(100% - 32px, 1120px);
  }

  .catalog-hero {
    padding-block: 52px 42px;
  }

  .catalog-title-line {
    align-items: center;
    flex-direction: column;
    gap: 11px;
  }

  .catalog-title-line h1 {
    font-size: clamp(2.1rem, 10vw, 2.7rem);
  }

  .catalog-heading > p {
    font-size: 15px;
  }

  .catalog-browser {
    padding-block: 28px 24px;
  }

  .catalog-filter-row {
    grid-template-columns: 1fr;
    gap: 8px;
  }

  .catalog-filter-label {
    padding-top: 0;
  }

  .catalog-filter-options {
    flex-wrap: nowrap;
    margin-inline: -16px;
    padding: 0 16px 4px;
    overflow-x: auto;
    overscroll-behavior-inline: contain;
    scrollbar-width: none;
  }

  .catalog-filter-options::-webkit-scrollbar {
    display: none;
  }

  .catalog-filter-options button {
    flex: 0 0 auto;
    min-height: 40px;
  }

  .catalog-results-heading {
    align-items: flex-start;
    flex-direction: column;
    gap: 6px;
  }

  .catalog-grid {
    grid-template-columns: 1fr;
  }

  .catalog-card {
    padding: 18px;
  }

  .catalog-summary {
    min-height: auto;
  }

  .catalog-copy-button {
    min-height: 40px;
  }

  .catalog-cta-actions {
    width: 100%;
    align-items: stretch;
    flex-direction: column;
  }

  .catalog-primary-button,
  .catalog-secondary-link {
    width: 100%;
    box-sizing: border-box;
    justify-content: center;
  }

  .catalog-price-dialog {
    width: calc(100% - 24px);
    max-height: calc(100vh - 24px);
  }

  .catalog-dialog-header,
  .catalog-dialog-body {
    padding: 18px;
  }

  .catalog-price-details {
    grid-template-columns: 1fr;
  }

  .catalog-interval-row {
    align-items: flex-start;
    flex-direction: column;
    gap: 5px;
  }

  .catalog-interval-row > span {
    text-align: left;
  }
}

@media (prefers-reduced-motion: reduce) {
  .catalog-card,
  .catalog-filter-options button,
  .catalog-search-wrap input {
    transition-duration: 0.01ms !important;
  }

  .catalog-card:hover {
    transform: none;
  }

  .skeleton-block {
    animation: none;
  }
}
</style>

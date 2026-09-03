<template>
  <component
    :is="layoutComponent"
    v-bind="layoutBindings"
    class="model-catalog-page"
    :class="{ 'model-catalog-page--embedded': props.embedded }"
  >
    <component :is="props.embedded ? 'div' : 'main'" id="top" class="catalog-main">
      <section class="catalog-hero" aria-labelledby="catalog-title">
        <div class="catalog-shell catalog-hero-inner">
          <div class="catalog-heading">
            <div class="catalog-title-line">
              <h1 id="catalog-title">
                {{ t(props.embedded ? 'modelCatalog.workspaceTitle' : 'modelCatalog.title') }}
              </h1>
              <span v-if="!loading && !errorState" class="catalog-count-badge">
                {{ t('modelCatalog.modelCount', { count: items.length }) }}
              </span>
            </div>
            <p>
              {{ t(props.embedded ? 'modelCatalog.workspaceDescription' : 'modelCatalog.description') }}
            </p>
          </div>
          <div class="catalog-hero-meta">
            <div id="catalog-pricing-note" class="catalog-price-note">
              <Icon name="infoCircle" size="sm" aria-hidden="true" />
              <span>{{ t('modelCatalog.publicPriceNote') }}</span>
            </div>
            <p v-if="formattedPricingUpdatedAt" class="catalog-updated-at">
              {{ t('modelCatalog.pricingUpdatedAt', { time: formattedPricingUpdatedAt }) }}
            </p>
          </div>
        </div>
      </section>

      <section class="catalog-shell catalog-browser" :aria-label="t('modelCatalog.searchLabel')">
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
      </section>

      <div class="catalog-shell catalog-layout">
        <button
          v-if="filtersOpen"
          type="button"
          class="catalog-filter-backdrop"
          :aria-label="t('modelCatalog.filters.close')"
          @click="closeFilters(true)"
        ></button>

        <aside
          id="catalog-filters"
          class="catalog-sidebar"
          :class="{ 'is-open': filtersOpen }"
          :aria-label="t('modelCatalog.filters.ariaLabel')"
          @keydown.esc="closeFilters(true)"
        >
          <div class="catalog-sidebar-head">
            <div class="catalog-sidebar-title">
              <Icon name="slidersHorizontal" size="sm" aria-hidden="true" />
              <h2>{{ t('modelCatalog.filters.sidebarTitle') }}</h2>
            </div>
            <div class="catalog-sidebar-actions">
              <button
                type="button"
                class="catalog-reset-button"
                :disabled="activeFilterCount === 0"
                @click="clearFilters"
              >
                {{ t('modelCatalog.filters.reset') }}
              </button>
              <button
                type="button"
                class="catalog-sidebar-close"
                ref="filterCloseButton"
                :aria-label="t('modelCatalog.filters.close')"
                @click="closeFilters(true)"
              >
                <Icon name="x" size="sm" aria-hidden="true" />
              </button>
            </div>
          </div>

          <div class="catalog-sidebar-body">
            <details class="catalog-filter-section" open>
              <summary>
                <span>{{ t('modelCatalog.filters.group') }}</span>
                <Icon name="chevronDown" size="xs" aria-hidden="true" />
              </summary>
              <div class="catalog-filter-options" role="group" :aria-label="t('modelCatalog.filters.group')">
                <button
                  type="button"
                  class="catalog-filter-chip"
                  :class="{ 'is-active': !pricedOnly }"
                  :aria-pressed="!pricedOnly"
                  @click="pricedOnly = false"
                >
                  <span>{{ t('modelCatalog.filters.allModels') }}</span>
                  <span class="catalog-filter-count" :data-count="items.length" aria-hidden="true"></span>
                </button>
                <button
                  type="button"
                  class="catalog-filter-chip"
                  :class="{ 'is-active': pricedOnly }"
                  :aria-pressed="pricedOnly"
                  @click="pricedOnly = true"
                >
                  <span>{{ t('modelCatalog.filters.pricedModels') }}</span>
                  <span class="catalog-filter-count" :data-count="pricedModelCount" aria-hidden="true"></span>
                </button>
              </div>
            </details>

            <details v-if="providerOptions.length" class="catalog-filter-section" open>
              <summary>
                <span>{{ t('modelCatalog.filters.provider') }}</span>
                <Icon name="chevronDown" size="xs" aria-hidden="true" />
              </summary>
              <div class="catalog-filter-options" role="group" :aria-label="t('modelCatalog.filters.provider')">
                <button
                  type="button"
                  class="catalog-filter-chip"
                  :class="{ 'is-active': providerFilter === '' }"
                  :aria-pressed="providerFilter === ''"
                  @click="providerFilter = ''"
                >
                  <span>{{ t('modelCatalog.filters.all') }}</span>
                  <span class="catalog-filter-count" :data-count="items.length" aria-hidden="true"></span>
                </button>
                <button
                  v-for="provider in providerOptions"
                  :key="provider"
                  type="button"
                  class="catalog-filter-chip"
                  :class="{ 'is-active': providerFilter === provider }"
                  :aria-pressed="providerFilter === provider"
                  @click="providerFilter = provider"
                >
                  <span>{{ providerLabel(provider) }}</span>
                  <span class="catalog-filter-count" :data-count="countForProvider(provider)" aria-hidden="true"></span>
                </button>
              </div>
            </details>

            <details v-if="categoryOptions.length" class="catalog-filter-section" open>
              <summary>
                <span>{{ t('modelCatalog.filters.category') }}</span>
                <Icon name="chevronDown" size="xs" aria-hidden="true" />
              </summary>
              <div class="catalog-filter-options" role="group" :aria-label="t('modelCatalog.filters.category')">
                <button
                  type="button"
                  class="catalog-filter-chip"
                  :class="{ 'is-active': categoryFilter === '' }"
                  :aria-pressed="categoryFilter === ''"
                  @click="categoryFilter = ''"
                >
                  <span>{{ t('modelCatalog.filters.all') }}</span>
                  <span class="catalog-filter-count" :data-count="items.length" aria-hidden="true"></span>
                </button>
                <button
                  v-for="category in categoryOptions"
                  :key="category"
                  type="button"
                  class="catalog-filter-chip"
                  :class="{ 'is-active': categoryFilter === category }"
                  :aria-pressed="categoryFilter === category"
                  @click="categoryFilter = category"
                >
                  <span>{{ categoryLabel(category) }}</span>
                  <span class="catalog-filter-count" :data-count="countForCategory(category)" aria-hidden="true"></span>
                </button>
              </div>
            </details>

            <details v-if="billingOptions.length" class="catalog-filter-section" open>
              <summary>
                <span>{{ t('modelCatalog.filters.billing') }}</span>
                <Icon name="chevronDown" size="xs" aria-hidden="true" />
              </summary>
              <div class="catalog-filter-options" role="group" :aria-label="t('modelCatalog.filters.billing')">
                <button
                  type="button"
                  class="catalog-filter-chip"
                  :class="{ 'is-active': billingFilter === '' }"
                  :aria-pressed="billingFilter === ''"
                  @click="billingFilter = ''"
                >
                  <span>{{ t('modelCatalog.filters.all') }}</span>
                  <span class="catalog-filter-count" :data-count="items.length" aria-hidden="true"></span>
                </button>
                <button
                  v-for="mode in billingOptions"
                  :key="mode"
                  type="button"
                  class="catalog-filter-chip"
                  :class="{ 'is-active': billingFilter === mode }"
                  :aria-pressed="billingFilter === mode"
                  @click="billingFilter = mode"
                >
                  <span>{{ billingModeLabel(mode) }}</span>
                  <span class="catalog-filter-count" :data-count="countForBilling(mode)" aria-hidden="true"></span>
                </button>
              </div>
            </details>

            <details v-if="tagOptions.length" class="catalog-filter-section" open>
              <summary>
                <span>{{ t('modelCatalog.filters.tags') }}</span>
                <Icon name="chevronDown" size="xs" aria-hidden="true" />
              </summary>
              <div class="catalog-filter-options" role="group" :aria-label="t('modelCatalog.filters.tags')">
                <button
                  type="button"
                  class="catalog-filter-chip"
                  :class="{ 'is-active': tagFilter === '' }"
                  :aria-pressed="tagFilter === ''"
                  @click="tagFilter = ''"
                >
                  <span>{{ t('modelCatalog.filters.all') }}</span>
                  <span class="catalog-filter-count" :data-count="items.length" aria-hidden="true"></span>
                </button>
                <button
                  v-for="tag in tagOptions"
                  :key="tag"
                  type="button"
                  class="catalog-filter-chip"
                  :class="{ 'is-active': tagFilter === tag }"
                  :aria-pressed="tagFilter === tag"
                  @click="tagFilter = tag"
                >
                  <span>{{ tagLabel(tag) }}</span>
                  <span class="catalog-filter-count" :data-count="countForTag(tag)" aria-hidden="true"></span>
                </button>
              </div>
            </details>
          </div>
        </aside>

        <section class="catalog-results catalog-results-main" aria-labelledby="catalog-results-title">
          <div class="catalog-results-toolbar">
            <div class="catalog-results-heading">
              <div class="catalog-results-title-line">
                <h2 id="catalog-results-title">{{ t('modelCatalog.resultsTitle') }}</h2>
                <span v-if="!loading && !errorState" class="catalog-results-count" aria-live="polite">
                  {{ filteredItems.length }}
                </span>
              </div>
              <p v-if="!loading && !errorState" aria-live="polite">
                {{ t('modelCatalog.resultCount', { count: filteredItems.length }) }}
              </p>
            </div>
            <div class="catalog-toolbar-actions">
              <button
                type="button"
                class="catalog-mobile-filter-button"
                ref="filterTriggerButton"
                :aria-expanded="filtersOpen"
                aria-controls="catalog-filters"
                @click="toggleFilters"
              >
                <Icon name="slidersHorizontal" size="sm" aria-hidden="true" />
                <span>{{ t('modelCatalog.filters.filterButton') }}</span>
                <span v-if="activeFilterCount" class="catalog-toolbar-badge">{{ activeFilterCount }}</span>
              </button>
              <label class="catalog-sort-control">
                <span class="catalog-sr-only">{{ t('modelCatalog.toolbar.sort') }}</span>
                <select v-model="sortMode" :aria-label="t('modelCatalog.toolbar.sort')">
                  <option value="default">{{ t('modelCatalog.toolbar.defaultSort') }}</option>
                  <option value="name">{{ t('modelCatalog.toolbar.nameAsc') }}</option>
                  <option value="priceAsc">{{ t('modelCatalog.toolbar.priceAsc') }}</option>
                  <option value="priceDesc">{{ t('modelCatalog.toolbar.priceDesc') }}</option>
                </select>
                <Icon name="chevronDown" size="xs" aria-hidden="true" />
              </label>
            </div>
          </div>

          <div v-if="loading" class="catalog-grid" aria-busy="true" :aria-label="t('modelCatalog.loading')">
            <article v-for="index in 6" :key="index" class="catalog-card catalog-skeleton" aria-hidden="true">
              <div class="catalog-skeleton-header">
                <div class="catalog-skeleton-identity">
                  <span class="skeleton-block skeleton-icon"></span>
                  <span class="skeleton-block skeleton-title"></span>
                </div>
                <span class="skeleton-block skeleton-copy"></span>
              </div>
              <div class="catalog-skeleton-price-grid">
                <div class="catalog-skeleton-price-item">
                  <span class="skeleton-block skeleton-price-label"></span>
                  <span class="skeleton-block skeleton-price-value"></span>
                </div>
                <div class="catalog-skeleton-price-item">
                  <span class="skeleton-block skeleton-price-label"></span>
                  <span class="skeleton-block skeleton-price-value skeleton-price-value--wide"></span>
                </div>
                <div class="catalog-skeleton-price-item catalog-skeleton-price-item--cache">
                  <span class="skeleton-block skeleton-price-label"></span>
                  <span class="skeleton-block skeleton-price-value"></span>
                </div>
              </div>
              <div class="catalog-skeleton-footer">
                <span class="skeleton-block skeleton-billing"></span>
                <span class="skeleton-block skeleton-details"></span>
              </div>
            </article>
          </div>

          <div v-else-if="errorState" class="catalog-state" role="alert" aria-live="assertive">
            <span class="catalog-state-icon" aria-hidden="true">
              <Icon :name="errorState === 'unavailable' ? 'inbox' : 'exclamationCircle'" size="lg" />
            </span>
            <h3>{{ t(errorState === 'unavailable' ? 'modelCatalog.unavailable.title' : 'modelCatalog.error.title') }}</h3>
            <p>{{ t(errorState === 'unavailable' ? 'modelCatalog.unavailable.description' : 'modelCatalog.error.description') }}</p>
            <button v-if="errorState === 'error'" type="button" class="catalog-secondary-button" @click="loadCatalog">
              <Icon name="refresh" size="sm" aria-hidden="true" />
              {{ t('modelCatalog.error.retry') }}
            </button>
          </div>

          <div v-else-if="filteredItems.length === 0" class="catalog-state" role="status">
            <span class="catalog-state-icon" aria-hidden="true"><Icon name="search" size="lg" /></span>
            <h3>{{ t(items.length === 0 ? 'modelCatalog.empty.title' : 'modelCatalog.noResults.title') }}</h3>
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
                <div class="catalog-card-identity">
                  <span class="catalog-model-icon" aria-hidden="true">
                    <ModelIcon :model="model.logo_key || model.model" size="32px" />
                  </span>
                  <h3 :title="model.display_name">{{ model.display_name }}</h3>
                </div>
                <button
                  type="button"
                  class="catalog-copy-button"
                  :aria-label="t('modelCatalog.copyModelAria', { model: model.model })"
                  :title="t(copiedModel === model.model ? 'modelCatalog.copied' : 'modelCatalog.copy')"
                  @click="copyModelId(model.model)"
                >
                  <Icon :name="copiedModel === model.model ? 'check' : 'copy'" size="sm" aria-hidden="true" />
                  <span class="catalog-sr-only">{{ t(copiedModel === model.model ? 'modelCatalog.copied' : 'modelCatalog.copy') }}</span>
                </button>
              </div>

              <div v-if="priceSummaryRows(model.pricing).length" class="catalog-price-grid">
                <div
                  v-for="row in priceSummaryRows(model.pricing)"
                  :key="row.key"
                  class="catalog-price-item"
                  :class="{ 'catalog-price-item--cache': row.key === 'cache' }"
                >
                  <span>{{ row.label }}</span>
                  <strong>
                    <CatalogPriceAmount
                      :value="row.value"
                      :scale="row.scale"
                      :currency="model.pricing.currency"
                      credit-display="cny"
                    />
                  </strong>
                </div>
              </div>
              <p v-else class="catalog-price-unknown">{{ t('modelCatalog.pricing.unknown') }}</p>

              <div class="catalog-card-footer">
                <span class="catalog-billing-tag">{{ cardBillingLabel(model.pricing.billing_mode) }}</span>
                <button
                  type="button"
                  class="catalog-details-button"
                  :aria-label="t('modelCatalog.pricing.details')"
                  @click="openPricingDetails(model)"
                >
                  <span>{{ t('modelCatalog.pricing.shortDetails') }}</span>
                </button>
              </div>
            </article>
          </div>
        </section>
      </div>

      <section
        v-if="!loading && !errorState && items.length > 0 && !props.embedded"
        class="catalog-shell catalog-cta"
        aria-labelledby="catalog-cta-title"
      >
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
    </component>

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
            <dd>
              <CatalogPriceAmount
                :value="row.value"
                :scale="row.scale"
                :currency="selectedModel.pricing.currency"
                credit-display="cny"
              />
              <small>{{ row.unit }}</small>
            </dd>
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
            <div class="catalog-interval-values">
              <span
                v-for="part in intervalPriceParts(interval, selectedModel.pricing.billing_mode)"
                :key="part.key"
                class="catalog-interval-value"
              >
                <span v-if="part.label">{{ part.label }}</span>
                <template v-for="(value, valueIndex) in part.values" :key="valueIndex">
                  <span v-if="valueIndex > 0" aria-hidden="true">/</span>
                  <CatalogPriceAmount
                    :value="value"
                    :scale="part.scale"
                    :currency="selectedModel.pricing.currency"
                    credit-display="cny"
                    icon-size="xs"
                  />
                </template>
                <span>{{ part.unit }}</span>
              </span>
            </div>
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
  </component>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import PublicSiteLayout from '@/components/public/PublicSiteLayout.vue'
import CatalogPriceAmount from '@/components/common/CatalogPriceAmount.vue'
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
import { sanitizeUrl } from '@/utils/url'

type ErrorState = 'unavailable' | 'error' | null
type SortMode = 'default' | 'name' | 'priceAsc' | 'priceDesc'

const TAG_ALIASES: Record<string, string> = {
  parallel_function_calling: 'function_calling',
  pdf_input: 'pdf',
  response_schema: 'structured_output',
  tool_choice: 'tool_calling'
}

const props = withDefaults(defineProps<{
  embedded?: boolean
}>(), {
  embedded: false
})

const layoutComponent = computed(() => (
  props.embedded ? AppLayout : PublicSiteLayout
))
const layoutBindings = computed(() => (
  props.embedded ? {} : { page: 'models' as const }
))

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
const billingFilter = ref(queryValue(route.query.billing).toLowerCase())
const tagFilter = ref(normalizeTagValue(queryValue(route.query.tag)))
const pricedOnly = ref(queryValue(route.query.priced) === '1')
const sortMode = ref<SortMode>((queryValue(route.query.sort) as SortMode) || 'default')
const filtersOpen = ref(false)
const filterTriggerButton = ref<HTMLButtonElement | null>(null)
const filterCloseButton = ref<HTMLButtonElement | null>(null)
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
const billingOptions = computed(() => Array.from(new Set(
  items.value.map((item) => item.pricing.billing_mode.trim().toLowerCase()).filter(Boolean)
)) as BillingMode[])
const tagOptions = computed(() => {
  const tags = items.value.flatMap((item) => [...item.tags, ...item.capabilities])
  return Array.from(new Set(tags.map(normalizeTagValue).filter(Boolean)))
    .sort((a, b) => tagLabel(a).localeCompare(tagLabel(b), locale.value))
})
const pricedModelCount = computed(() => items.value.filter((item) => hasCatalogPrice(item.pricing)).length)
const activeFilterCount = computed(() => (
  Number(Boolean(searchQuery.value.trim()))
  + Number(Boolean(providerFilter.value))
  + Number(Boolean(categoryFilter.value))
  + Number(Boolean(billingFilter.value))
  + Number(Boolean(tagFilter.value))
  + Number(pricedOnly.value)
))

const filteredItems = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()
  const filtered = items.value
    .map((item, index) => ({ item, index }))
    .filter(({ item }) => {
      if (providerFilter.value && item.provider.trim().toLowerCase() !== providerFilter.value) return false
      if (categoryFilter.value && item.category.trim().toLowerCase() !== categoryFilter.value) return false
      if (billingFilter.value && item.pricing.billing_mode.trim().toLowerCase() !== billingFilter.value) return false
      if (tagFilter.value) {
        const modelTags = [...item.tags, ...item.capabilities].map(normalizeTagValue)
        if (!modelTags.includes(tagFilter.value)) return false
      }
      if (pricedOnly.value && !hasCatalogPrice(item.pricing)) return false
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
  return filtered
    .sort((a, b) => {
      if (sortMode.value === 'name') {
        return a.item.display_name.localeCompare(b.item.display_name, locale.value) || a.index - b.index
      }
      if (sortMode.value === 'priceAsc' || sortMode.value === 'priceDesc') {
        const aPrice = modelSortPrice(a.item)
        const bPrice = modelSortPrice(b.item)
        const difference = sortMode.value === 'priceAsc' ? aPrice - bPrice : bPrice - aPrice
        return difference || a.index - b.index
      }
      // Featured models are pinned silently; the UI never labels them as “recommended”.
      return Number(b.item.featured === true) - Number(a.item.featured === true) || a.index - b.index
    })
    .map(({ item }) => item)
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
    const nextBilling = queryValue(query.billing).toLowerCase()
    const nextTag = normalizeTagValue(queryValue(query.tag))
    const nextPriced = queryValue(query.priced) === '1'
    const nextSort = queryValue(query.sort) as SortMode
    if (searchQuery.value !== nextSearch) searchQuery.value = nextSearch
    if (providerFilter.value !== nextProvider) providerFilter.value = nextProvider
    if (categoryFilter.value !== nextCategory) categoryFilter.value = nextCategory
    if (billingFilter.value !== nextBilling) billingFilter.value = nextBilling
    if (tagFilter.value !== nextTag) tagFilter.value = nextTag
    if (pricedOnly.value !== nextPriced) pricedOnly.value = nextPriced
    if (['default', 'name', 'priceAsc', 'priceDesc'].includes(nextSort) && sortMode.value !== nextSort) {
      sortMode.value = nextSort
    }
  },
  { deep: true }
)

watch([searchQuery, providerFilter, categoryFilter, billingFilter, tagFilter, pricedOnly, sortMode], () => {
  const nextQuery = { ...route.query }
  if (searchQuery.value) nextQuery.q = searchQuery.value
  else delete nextQuery.q
  if (providerFilter.value) nextQuery.provider = providerFilter.value
  else delete nextQuery.provider
  if (categoryFilter.value) nextQuery.type = categoryFilter.value
  else delete nextQuery.type
  if (billingFilter.value) nextQuery.billing = billingFilter.value
  else delete nextQuery.billing
  if (tagFilter.value) nextQuery.tag = tagFilter.value
  else delete nextQuery.tag
  if (pricedOnly.value) nextQuery.priced = '1'
  else delete nextQuery.priced
  if (sortMode.value !== 'default') nextQuery.sort = sortMode.value
  else delete nextQuery.sort

  if (
    queryValue(route.query.q) === queryValue(nextQuery.q)
    && queryValue(route.query.provider) === queryValue(nextQuery.provider)
    && queryValue(route.query.type) === queryValue(nextQuery.type)
    && queryValue(route.query.billing) === queryValue(nextQuery.billing)
    && queryValue(route.query.tag) === queryValue(nextQuery.tag)
    && queryValue(route.query.priced) === queryValue(nextQuery.priced)
    && queryValue(route.query.sort) === queryValue(nextQuery.sort)
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
  billingFilter.value = ''
  tagFilter.value = ''
  pricedOnly.value = false
  sortMode.value = 'default'
  closeFilters()
  void nextTick(() => document.getElementById('catalog-search')?.focus())
}

function toggleFilters() {
  filtersOpen.value = !filtersOpen.value
  if (filtersOpen.value) {
    void nextTick(() => filterCloseButton.value?.focus())
  } else {
    void nextTick(() => filterTriggerButton.value?.focus())
  }
}

function closeFilters(restoreFocus = false) {
  filtersOpen.value = false
  if (restoreFocus) void nextTick(() => filterTriggerButton.value?.focus())
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

function tagLabel(tag: string): string {
  const normalized = tag.trim().toLowerCase().replace(/[^a-z0-9]+/g, '_')
  const translationKey = `modelCatalog.capabilityLabels.${normalized}`
  return te(translationKey) ? t(translationKey) : tag
}

function normalizeTagValue(tag: string): string {
  const raw = tag.trim().toLowerCase()
  if (!raw) return ''
  const normalized = raw.replace(/[^a-z0-9_]+/g, '_')
  if (/^[a-z0-9_]+$/.test(raw)) return TAG_ALIASES[normalized] || normalized
  return raw
}

function countForProvider(provider: string): number {
  return items.value.filter((item) => item.provider.trim().toLowerCase() === provider).length
}

function countForCategory(category: string): number {
  return items.value.filter((item) => item.category.trim().toLowerCase() === category).length
}

function countForBilling(mode: BillingMode): number {
  return items.value.filter((item) => item.pricing.billing_mode.trim().toLowerCase() === mode).length
}

function countForTag(tag: string): number {
  return items.value.filter((item) => (
    [...item.tags, ...item.capabilities].some((itemTag) => normalizeTagValue(itemTag) === tag)
  )).length
}

function modelSortPrice(item: PublicModelCatalogItem): number {
  const pricing = item.pricing
  if (pricing.billing_mode === BILLING_MODE_TOKEN) {
    return pricing.input_price ?? pricing.output_price ?? pricing.cache_read_price ?? pricing.cache_write_price ?? Number.POSITIVE_INFINITY
  }
  return pricing.per_request_price ?? Number.POSITIVE_INFINITY
}

function cardBillingLabel(mode: BillingMode): string {
  return mode === BILLING_MODE_TOKEN
    ? t('modelCatalog.pricing.usageBased')
    : billingModeLabel(mode)
}

function hasCatalogPrice(pricing: PublicModelCatalogPricing): boolean {
  return pricing.billing_mode === BILLING_MODE_TOKEN
    ? [pricing.input_price, pricing.output_price, pricing.cache_read_price, pricing.cache_write_price].some((value) => value != null)
    : pricing.per_request_price != null
}

function formatTokenCount(value: number): string {
  if (value >= 1_000_000 && value % 1_000_000 === 0) return `${value / 1_000_000}M`
  if (value >= 1_000 && value % 1_000 === 0) return `${value / 1_000}K`
  return new Intl.NumberFormat(locale.value === 'zh' ? 'zh-CN' : 'en-US').format(value)
}

function priceSummaryRows(pricing: PublicModelCatalogPricing) {
  if (pricing.billing_mode === BILLING_MODE_TOKEN) {
    const cachePrice = pricing.cache_read_price ?? pricing.cache_write_price
    const rows = [
      {
        key: 'input',
        label: t('modelCatalog.pricing.input'),
        value: pricing.input_price,
        scale: 1_000_000
      },
      {
        key: 'output',
        label: t('modelCatalog.pricing.output'),
        value: pricing.output_price,
        scale: 1_000_000
      },
      {
        key: 'cache',
        label: t('modelCatalog.pricing.cache'),
        value: cachePrice,
        scale: 1_000_000
      }
    ]
    return hasCatalogPrice(pricing) ? rows : []
  }

  if (pricing.per_request_price != null) {
    return [{
      key: pricing.billing_mode === BILLING_MODE_IMAGE ? 'image' : 'request',
      label: t(pricing.billing_mode === BILLING_MODE_IMAGE
        ? 'modelCatalog.pricing.image'
        : 'modelCatalog.pricing.request'),
      value: pricing.per_request_price,
      scale: 1
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

function intervalPriceParts(interval: PublicModelCatalogPricingInterval, mode: BillingMode) {
  if (mode === BILLING_MODE_PER_REQUEST || mode === BILLING_MODE_IMAGE) {
    return [{
      key: 'request',
      label: '',
      values: [interval.per_request_price],
      scale: 1,
      unit: t(mode === BILLING_MODE_IMAGE ? 'modelCatalog.pricing.perImage' : 'modelCatalog.pricing.perRequest')
    }]
  }
  const unit = t('modelCatalog.pricing.perMillionTokens')
  const parts: Array<{
    key: string
    label: string
    values: Array<number | null>
    scale: number
    unit: string
  }> = []
  if (interval.input_price != null || interval.output_price != null) {
    parts.push({
      key: 'input-output',
      label: `${t('modelCatalog.pricing.input')}/${t('modelCatalog.pricing.output')}`,
      values: [interval.input_price, interval.output_price],
      scale: 1_000_000,
      unit
    })
  }
  if (interval.cache_write_price != null) {
    parts.push({
      key: 'cache-write',
      label: t('modelCatalog.pricing.cacheWrite'),
      values: [interval.cache_write_price],
      scale: 1_000_000,
      unit
    })
  }
  if (interval.cache_write_1h_price != null) {
    parts.push({
      key: 'cache-write-1h',
      label: t('modelCatalog.pricing.cacheWrite1h'),
      values: [interval.cache_write_1h_price],
      scale: 1_000_000,
      unit
    })
  }
  if (interval.cache_read_price != null) {
    parts.push({
      key: 'cache-read',
      label: t('modelCatalog.pricing.cacheRead'),
      values: [interval.cache_read_price],
      scale: 1_000_000,
      unit
    })
  }
  return parts.length > 0
    ? parts
    : [{ key: 'unavailable', label: '', values: [null], scale: 1, unit: '' }]
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
  if (!props.embedded && managedHead.size) updateDocumentMeta()
})

onMounted(() => {
  if (!props.embedded) updateDocumentMeta()
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
  --catalog-page: #f8fafc;
  --catalog-surface: #ffffff;
  --catalog-surface-soft: #f7f7f8;
  --catalog-ink: #111827;
  --catalog-copy: #4b5563;
  --catalog-muted: #6b7280;
  --catalog-border: #e5e7eb;
  --catalog-accent: #374151;
  --catalog-accent-strong: #1f2937;
  --catalog-accent-ink: #374151;
  min-height: 100vh;
  background: var(--catalog-page);
  color: var(--catalog-ink);
}

.model-catalog-page :where(a, button, input):focus-visible {
  outline: 3px solid color-mix(in srgb, var(--catalog-ink) 32%, transparent);
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
  width: min(1240px, calc(100% - 40px));
  margin-inline: auto;
}

.catalog-main {
  --page: var(--catalog-page);
  --surface: var(--catalog-surface);
  --surface-soft: var(--catalog-surface-soft);
  --surface-accent: var(--catalog-surface-soft);
  --ink: var(--catalog-ink);
  --copy: var(--catalog-copy);
  --muted: var(--catalog-muted);
  --border: var(--catalog-border);
  --accent: var(--catalog-accent);
  --accent-strong: var(--catalog-accent-strong);
  --accent-ink: var(--catalog-accent-ink);
  min-height: calc(100vh - 132px);
}

.catalog-hero {
  border-bottom: 0;
  padding-block: 42px 22px;
}

.catalog-hero-inner {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 32px;
}

.catalog-heading {
  max-width: 680px;
}

.catalog-title-line {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  flex-wrap: wrap;
  gap: 12px;
}

.catalog-title-line h1 {
  margin: 0;
  color: var(--ink);
  font-size: clamp(2rem, 3.8vw, 2.75rem);
  font-weight: 720;
  letter-spacing: -0.035em;
  line-height: 1.12;
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
  margin: 12px 0 0;
  color: var(--copy);
  font-size: 14px;
  line-height: 1.6;
  text-wrap: pretty;
}

.catalog-hero-meta {
  flex: 0 1 430px;
  display: grid;
  justify-items: end;
  gap: 8px;
  padding-bottom: 2px;
}

.catalog-price-note {
  max-width: 430px;
  display: flex;
  align-items: center;
  gap: 7px;
  margin-top: 0;
  border: 0;
  border-radius: 0;
  background: transparent;
  color: var(--muted);
  padding: 0;
  font-size: 12px;
  line-height: 1.5;
  text-align: right;
}

.catalog-price-note svg {
  flex: 0 0 auto;
}

.catalog-updated-at {
  margin: 0;
  color: var(--muted);
  font-size: 12px;
}

.catalog-browser {
  padding-block: 0 22px;
}

.catalog-search-wrap {
  position: relative;
  max-width: none;
  width: 100%;
  margin: 0;
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
  min-height: 46px;
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

.catalog-search-wrap input:hover {
  border-color: color-mix(in srgb, var(--border) 60%, var(--ink));
}

.catalog-search-wrap input:focus {
  border-color: color-mix(in srgb, var(--border) 42%, var(--ink));
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--ink) 9%, transparent);
}

.catalog-layout {
  position: relative;
  display: grid;
  grid-template-columns: 300px minmax(0, 1fr);
  align-items: start;
  gap: 32px;
}

.catalog-sidebar {
  min-width: 0;
  position: sticky;
  top: 84px;
  max-height: calc(100vh - 104px);
  overflow: auto;
  border: 1px solid var(--border);
  border-radius: 20px;
  background: var(--surface);
  scrollbar-width: thin;
  scrollbar-color: var(--border) transparent;
}

.catalog-sidebar::-webkit-scrollbar {
  width: 6px;
}

.catalog-sidebar::-webkit-scrollbar-thumb {
  border-radius: 999px;
  background: var(--border);
}

.catalog-sidebar-head {
  min-height: 58px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  border-bottom: 1px solid var(--border);
  padding: 0 16px;
}

.catalog-sidebar-title,
.catalog-sidebar-actions,
.catalog-toolbar-actions,
.catalog-results-title-line {
  display: flex;
  align-items: center;
}

.catalog-sidebar-title {
  min-width: 0;
  gap: 8px;
  color: var(--ink);
}

.catalog-sidebar-title svg {
  color: var(--muted);
}

.catalog-sidebar-title h2 {
  margin: 0;
  font-size: 14px;
  font-weight: 720;
}

.catalog-sidebar-actions {
  flex: 0 0 auto;
  gap: 4px;
}

.catalog-reset-button,
.catalog-sidebar-close {
  border: 0;
  background: transparent;
  color: var(--muted);
  cursor: pointer;
}

.catalog-reset-button {
  min-height: 32px;
  border-radius: 8px;
  padding: 0 7px;
  font-size: 12px;
  font-weight: 650;
}

.catalog-reset-button:hover:not(:disabled) {
  background: var(--surface-soft);
  color: var(--ink);
}

.catalog-reset-button:disabled {
  cursor: default;
  opacity: 0.42;
}

.catalog-sidebar-close {
  display: none;
  width: 36px;
  height: 36px;
  align-items: center;
  justify-content: center;
  border-radius: 9px;
}

.catalog-sidebar-close:hover {
  background: var(--surface-soft);
  color: var(--ink);
}

.catalog-sidebar-body {
  display: grid;
  gap: 4px;
  padding: 10px 10px 14px;
}

.catalog-filter-section {
  min-width: 0;
  border-bottom: 1px solid color-mix(in srgb, var(--border) 72%, transparent);
  padding: 0 0 10px;
}

.catalog-filter-section:last-child {
  border-bottom: 0;
  padding-bottom: 0;
}

.catalog-filter-section summary {
  min-height: 34px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  list-style: none;
  border-radius: 8px;
  color: var(--muted);
  padding: 0 8px;
  cursor: pointer;
  font-size: 11px;
  font-weight: 760;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.catalog-filter-section summary::-webkit-details-marker {
  display: none;
}

.catalog-filter-section summary:hover {
  background: var(--surface-soft);
  color: var(--ink);
}

.catalog-filter-section[open] summary svg {
  transform: rotate(180deg);
}

.catalog-filter-section summary svg {
  transition: transform 150ms ease;
}

.catalog-filter-options {
  min-width: 0;
  display: grid;
  gap: 2px;
  padding-top: 2px;
}

.catalog-filter-chip {
  min-width: 0;
  min-height: 36px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  border: 1px solid transparent;
  border-radius: 9px;
  background: transparent;
  color: var(--copy);
  padding: 0 10px;
  cursor: pointer;
  font: inherit;
  font-size: 13px;
  font-weight: 540;
  text-align: left;
  transition: background-color 150ms ease, border-color 150ms ease, color 150ms ease, box-shadow 150ms ease;
}

.catalog-filter-chip > span:first-child {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.catalog-filter-chip:hover {
  background: var(--surface-soft);
  color: var(--ink);
}

.catalog-filter-chip.is-active {
  border-color: color-mix(in srgb, var(--ink) 20%, var(--border));
  background: color-mix(in srgb, var(--ink) 5%, transparent);
  box-shadow: inset 2px 0 var(--ink);
  color: var(--ink);
  font-weight: 700;
}

.catalog-filter-count {
  flex: 0 0 auto;
  min-width: 1.2em;
  color: var(--muted);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  text-align: right;
}

.catalog-filter-count::before {
  content: attr(data-count);
}

.catalog-filter-backdrop {
  display: none;
}

.catalog-results {
  min-width: 0;
  padding-block: 0 72px;
}

.catalog-results-heading {
  min-width: 0;
  display: grid;
  gap: 4px;
}

.catalog-results-toolbar {
  min-height: 42px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
}

.catalog-results-title-line {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.catalog-results-heading h2 {
  margin: 0;
  color: var(--ink);
  font-size: 17px;
  font-weight: 700;
}

.catalog-results-count {
  min-width: 22px;
  height: 22px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  background: var(--surface-soft);
  color: var(--muted);
  padding: 0 7px;
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

.catalog-results-heading p {
  margin: 0;
  color: var(--muted);
  font-size: 12px;
}

.catalog-toolbar-actions {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}

.catalog-mobile-filter-button {
  display: none;
  min-height: 36px;
  align-items: center;
  gap: 6px;
  border: 1px solid var(--border);
  border-radius: 9px;
  background: var(--surface);
  color: var(--copy);
  padding: 0 10px;
  cursor: pointer;
  font: inherit;
  font-size: 12px;
  font-weight: 650;
}

.catalog-mobile-filter-button:hover {
  border-color: color-mix(in srgb, var(--border) 60%, var(--ink));
  color: var(--ink);
}

.catalog-toolbar-badge {
  min-width: 18px;
  height: 18px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  background: var(--surface-soft);
  color: var(--ink);
  padding: 0 4px;
  font-size: 10px;
  font-variant-numeric: tabular-nums;
}

.catalog-sort-control {
  position: relative;
  min-width: 128px;
  display: inline-flex;
  align-items: center;
}

.catalog-sort-control select {
  width: 100%;
  min-height: 36px;
  appearance: none;
  border: 1px solid var(--border);
  border-radius: 9px;
  background: var(--surface);
  color: var(--copy);
  padding: 0 30px 0 11px;
  cursor: pointer;
  font: inherit;
  font-size: 12px;
  font-weight: 620;
}

.catalog-sort-control select:hover,
.catalog-sort-control select:focus {
  border-color: color-mix(in srgb, var(--border) 60%, var(--ink));
  color: var(--ink);
}

.catalog-sort-control > svg {
  position: absolute;
  right: 10px;
  color: var(--muted);
  pointer-events: none;
}

.catalog-grid {
  min-width: 0;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}

.catalog-card {
  min-width: 0;
  height: 192px;
  position: relative;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border);
  border-radius: 22px;
  background: var(--surface);
  padding: 14px 16px;
  box-sizing: border-box;
  transition: border-color 160ms ease;
}

.catalog-card:hover {
  border-color: color-mix(in srgb, var(--border) 68%, var(--ink));
}

.catalog-card-header {
  min-width: 0;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
}

.catalog-model-icon {
  width: 32px;
  height: 32px;
  flex: 0 0 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 0;
  background: transparent;
}

.catalog-card-identity {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
}

.catalog-model-icon :deep(.model-icon-fallback) {
  border: 0;
  border-radius: 0;
  background: transparent;
  color: var(--ink);
}

.catalog-model-icon :deep(.model-icon-image) {
  border-radius: 0;
}

:global(html.dark .model-catalog-page .catalog-model-icon .model-icon-image) {
  /* The supplied provider marks are opaque raster files. In dark mode an
     inverted monochrome treatment lets their background merge into the card
     while keeping the mark legible without adding a box. */
  filter: invert(1) grayscale(1);
  mix-blend-mode: screen;
}

.catalog-card-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.catalog-card h3 {
  min-width: 0;
  overflow: hidden;
  margin: 0;
  color: var(--ink);
  font-size: 16px;
  font-weight: 720;
  letter-spacing: -0.018em;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.catalog-details-button,
.catalog-copy-button {
  position: relative;
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  background: transparent;
  color: var(--copy);
  padding: 0;
  cursor: pointer;
}

.catalog-details-button {
  width: 60px;
  height: 38px;
}

.catalog-details-button::before {
  position: absolute;
  inset: 4px 0;
  border: 1px solid var(--border);
  border-radius: 999px;
  content: '';
  transition: background-color 150ms ease, border-color 150ms ease;
}

.catalog-details-button > span {
  position: relative;
  z-index: 1;
  color: inherit;
  font-size: 13px;
  font-weight: 600;
  line-height: 1;
  white-space: nowrap;
}

.catalog-copy-button {
  width: 44px;
  height: 44px;
}

.catalog-copy-button::before {
  position: absolute;
  width: 28px;
  height: 28px;
  border: 1px solid var(--border);
  border-radius: 50%;
  content: '';
  transition: background-color 150ms ease, border-color 150ms ease;
}

.catalog-copy-button > svg {
  position: relative;
  z-index: 1;
}

.catalog-details-button:hover::before,
.catalog-copy-button:hover::before {
  border-color: color-mix(in srgb, var(--border) 70%, var(--ink));
  background: var(--surface-soft);
}

.model-catalog-page .catalog-details-button:focus-visible,
.model-catalog-page .catalog-copy-button:focus-visible {
  outline: 2px solid var(--muted);
  outline-offset: 2px;
}

.catalog-price-grid {
  min-width: 0;
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  row-gap: 5px;
  margin-top: 12px;
}

.catalog-price-item {
  min-width: 0;
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
  white-space: nowrap;
}

.catalog-price-item--cache {
  grid-column: auto;
}

.catalog-price-item > span {
  color: var(--muted);
  font-size: 13px;
  font-weight: 400;
  line-height: 1.35;
}

.catalog-price-item strong {
  min-width: 0;
  color: var(--ink);
  font-size: 15px;
  font-weight: 650;
  letter-spacing: -0.01em;
  line-height: 1.35;
  font-variant-numeric: tabular-nums;
}

.catalog-price-item strong :deep(span) {
  color: inherit;
  font: inherit;
  font-variant-numeric: inherit;
}

.catalog-card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: auto;
  padding-top: 10px;
}

.catalog-billing-tag {
  color: var(--muted);
  font-size: 12px;
  font-weight: 400;
  line-height: 1.2;
}

.catalog-price-unknown {
  margin: 10px 0 0;
  color: var(--muted);
  font-size: 13px;
  line-height: 1.35;
}

.catalog-state {
  min-height: 210px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  border: 1px solid var(--border);
  border-radius: 16px;
  background: var(--surface);
  padding: 32px 24px;
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

.catalog-state h3 {
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
  min-height: 40px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border-radius: 999px;
  padding: 0 15px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 700;
  text-decoration: none;
}

.catalog-secondary-button {
  margin-top: 16px;
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
  gap: 24px;
  margin-bottom: 48px;
  border-top: 1px solid var(--border);
  border-bottom: 1px solid var(--border);
  background: transparent;
  padding: 16px 0;
}

.catalog-cta h2 {
  margin: 0;
  color: var(--ink);
  font-size: 15px;
  font-weight: 720;
  letter-spacing: -0.01em;
  text-wrap: balance;
}

.catalog-cta p {
  max-width: 70ch;
  margin: 4px 0 0;
  color: var(--copy);
  font-size: 12px;
  line-height: 1.5;
}

.catalog-cta-actions {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  gap: 12px;
}

.catalog-secondary-link {
  min-height: 40px;
  display: inline-flex;
  align-items: center;
  color: var(--copy);
  font-size: 13px;
  font-weight: 600;
  text-decoration: none;
}

.catalog-secondary-link:hover {
  color: var(--ink);
}

.catalog-price-dialog {
  --page: var(--catalog-page);
  --surface: var(--catalog-surface);
  --surface-soft: var(--catalog-surface-soft);
  --surface-accent: var(--catalog-surface-soft);
  --ink: var(--catalog-ink);
  --copy: var(--catalog-copy);
  --muted: var(--catalog-muted);
  --border: var(--catalog-border);
  --accent: var(--catalog-accent);
  --accent-strong: var(--catalog-accent-strong);
  --accent-ink: var(--catalog-accent-ink);
  width: min(620px, calc(100% - 32px));
  max-height: min(760px, calc(100vh - 48px));
  overflow: auto;
  box-sizing: border-box;
  border: 1px solid var(--border);
  border-radius: 16px;
  background: var(--surface);
  color: var(--ink);
  padding: 0;
  box-shadow: 0 18px 48px rgb(15 23 42 / 0.16);
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

.catalog-interval-values {
  display: grid;
  justify-items: end;
  gap: 4px;
  text-align: right;
}

.catalog-interval-value {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 4px;
}

.catalog-peak-note {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-top: 22px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-soft);
  color: var(--copy);
  padding: 12px;
  font-size: 12px;
  line-height: 1.6;
}

.catalog-peak-note svg {
  flex: 0 0 auto;
  margin-top: 2px;
}

.catalog-skeleton {
  height: 192px;
  pointer-events: none;
}

.catalog-skeleton-header {
  min-width: 0;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
}

.catalog-skeleton-identity,
.catalog-skeleton-actions,
.catalog-skeleton-price-item {
  display: flex;
  align-items: center;
}

.catalog-skeleton-identity {
  min-width: 0;
  gap: 10px;
}

.skeleton-block {
  flex: 0 0 auto;
  display: inline-block;
  border-radius: 6px;
  background: var(--surface-soft);
  animation: catalog-pulse 1.35s ease-in-out infinite;
}

.skeleton-icon {
  width: 32px;
  height: 32px;
  border-radius: 0;
}

.skeleton-title {
  flex: 1 1 112px;
  width: auto;
  max-width: 112px;
  height: 16px;
}

.skeleton-copy {
  width: 28px;
  height: 28px;
  border-radius: 50%;
}

.skeleton-details {
  width: 60px;
  height: 30px;
  border-radius: 999px;
}

.catalog-skeleton-price-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  row-gap: 5px;
  margin-top: 12px;
}

.catalog-skeleton-price-item {
  justify-content: space-between;
  gap: 8px;
}

.catalog-skeleton-price-item--cache {
  grid-column: auto;
}

.skeleton-price-label {
  width: 28px;
  height: 13px;
}

.skeleton-price-value {
  width: 42px;
  height: 15px;
}

.skeleton-price-value--wide {
  width: 52px;
}

.catalog-skeleton-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: auto;
  padding-top: 10px;
}

.skeleton-billing {
  width: 56px;
  height: 12px;
}

@keyframes catalog-pulse {
  0%, 100% { opacity: 0.62; }
  50% { opacity: 1; }
}

@media (max-width: 1199px) {
  .catalog-layout {
    grid-template-columns: 260px minmax(0, 1fr);
    gap: 22px;
  }

  .catalog-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .catalog-cta {
    gap: 18px;
  }
}

@media (max-width: 900px) {
  .catalog-hero-inner {
    align-items: flex-start;
    flex-direction: column;
    gap: 14px;
  }

  .catalog-hero-meta {
    flex: 0 0 auto;
    width: 100%;
    height: auto;
    justify-items: start;
    align-self: stretch;
    padding-bottom: 0;
  }

  .catalog-price-note {
    max-width: 100%;
    text-align: left;
  }

  .catalog-search-wrap {
    max-width: none;
  }
}

@media (max-width: 1023px) {
  .catalog-layout {
    display: block;
  }

  .catalog-sidebar {
    position: fixed;
    z-index: 70;
    top: 0;
    bottom: 0;
    left: 0;
    width: min(320px, calc(100vw - 42px));
    max-height: none;
    overflow-y: auto;
    border-radius: 0 20px 20px 0;
    visibility: hidden;
    pointer-events: none;
    transform: translateX(-105%);
    transition: transform 180ms ease;
  }

  .catalog-sidebar.is-open {
    visibility: visible;
    pointer-events: auto;
    transform: translateX(0);
  }

  .catalog-sidebar-close {
    display: inline-flex;
  }

  .catalog-filter-backdrop {
    position: fixed;
    z-index: 60;
    inset: 0;
    display: block;
    border: 0;
    background: rgb(15 23 42 / 0.28);
    cursor: pointer;
  }

  .catalog-mobile-filter-button {
    display: inline-flex;
  }

  .catalog-results {
    padding-block: 0 64px;
  }
}

@media (max-width: 640px) {
  .catalog-grid {
    grid-template-columns: 1fr;
  }

  .catalog-shell {
    width: min(100% - 32px, 1200px);
  }

  .catalog-hero {
    padding-block: 28px 16px;
  }

  .catalog-title-line {
    align-items: center;
    justify-content: flex-start;
    gap: 9px;
  }

  .catalog-title-line h1 {
    font-size: clamp(1.8rem, 8vw, 2.25rem);
  }

  .catalog-heading > p {
    margin-top: 9px;
    font-size: 13px;
  }

  .catalog-browser {
    padding-block: 0 16px;
  }

  .catalog-results {
    padding-block: 0 52px;
  }

  .catalog-results-toolbar {
    align-items: flex-start;
    flex-direction: column;
    gap: 10px;
    margin-bottom: 12px;
  }

  .catalog-results-heading {
    width: 100%;
  }

  .catalog-toolbar-actions {
    width: 100%;
    justify-content: space-between;
  }

  .catalog-sort-control {
    min-width: 132px;
  }

  .catalog-card {
    height: 192px;
    padding: 12px 14px;
    border-radius: 20px;
  }

  .catalog-copy-button {
    min-height: 44px;
  }

  .catalog-price-grid {
    column-gap: 12px;
  }

  .catalog-cta-actions {
    gap: 8px;
  }

  .catalog-primary-button,
  .catalog-secondary-link {
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

  .catalog-interval-values {
    justify-items: start;
    text-align: left;
  }
}

@media (max-width: 560px) {
  .catalog-cta {
    align-items: flex-start;
    flex-direction: column;
    gap: 12px;
    margin-bottom: 32px;
  }

  .catalog-cta-actions {
    width: 100%;
    align-items: stretch;
    flex-direction: column;
  }

  .catalog-primary-button,
  .catalog-secondary-link {
    width: 100%;
  }
}

.model-catalog-page--embedded {
  --catalog-page: var(--workspace-canvas);
  --catalog-surface: var(--workspace-surface);
  --catalog-surface-soft: var(--workspace-surface-subtle);
  --catalog-ink: var(--workspace-text);
  --catalog-copy: var(--workspace-text-secondary);
  --catalog-muted: var(--workspace-text-secondary);
  --catalog-border: var(--workspace-border);
  --catalog-accent: #374151;
  --catalog-accent-strong: #1f2937;
  --catalog-accent-ink: #374151;
  min-height: 0;
  background: var(--catalog-page);
  color: var(--catalog-ink);
}

.model-catalog-page--embedded .catalog-main {
  min-height: 0;
}

.model-catalog-page--embedded .catalog-shell {
  width: 100%;
  max-width: 1200px;
}

.model-catalog-page--embedded .catalog-hero {
  padding-block: 0 20px;
}

.model-catalog-page--embedded .catalog-hero-inner {
  align-items: flex-start;
  justify-content: space-between;
  text-align: left;
}

.model-catalog-page--embedded .catalog-heading {
  max-width: 820px;
}

.model-catalog-page--embedded .catalog-title-line {
  align-items: flex-start;
  justify-content: flex-start;
  gap: 10px;
}

.model-catalog-page--embedded .catalog-title-line h1 {
  font-size: clamp(1.75rem, 3vw, 2.25rem);
  letter-spacing: -0.03em;
  line-height: 1.2;
}

.model-catalog-page--embedded .catalog-heading > p {
  margin: 8px 0 0;
  font-size: 14px;
  line-height: 1.55;
}

.model-catalog-page--embedded .catalog-price-note {
  max-width: 420px;
  margin-top: 0;
  border-radius: 0;
}

.model-catalog-page--embedded .catalog-browser {
  padding-block: 10px;
}

.model-catalog-page--embedded .catalog-search-wrap {
  max-width: 680px;
  margin-inline: 0;
}

.model-catalog-page--embedded .catalog-search-wrap input {
  border-radius: var(--workspace-radius-input);
}

.model-catalog-page--embedded .catalog-results {
  padding-block: 24px 40px;
}

.model-catalog-page--embedded .catalog-card,
.model-catalog-page--embedded .catalog-cta {
  border-radius: 20px;
}

.model-catalog-page--embedded .catalog-cta {
  margin-bottom: 0;
  padding: 24px;
}

:global(html.dark .model-catalog-page) {
  --catalog-page: #111111;
  --catalog-surface: #171717;
  --catalog-surface-soft: #212121;
  --catalog-ink: #ececec;
  --catalog-copy: #b4b4b4;
  --catalog-muted: #8a8a8a;
  --catalog-border: rgb(255 255 255 / 0.1);
  --catalog-accent: #ececec;
  --catalog-accent-strong: #ffffff;
  --catalog-accent-ink: #ececec;
  background: var(--catalog-page);
  color: var(--catalog-ink);
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

<template>
  <div class="table-scroll" tabindex="0" :aria-label="t('modelCatalog.table.scrollLabel')">
    <table class="price-table" :class="{ 'per-request-table': !perToken }">
      <colgroup>
        <col class="model-col"><col class="input-col"><col class="output-col"><col class="cache-col">
        <col class="input-col"><col class="output-col"><col class="cache-col"><col class="rate-col">
      </colgroup>
      <thead>
        <tr>
          <th :rowspan="perToken ? 2 : 1" scope="col" class="model-heading">{{ t('modelCatalog.table.model') }}</th>
          <th colspan="3" scope="colgroup" class="price-tier-heading discount-heading discount-start">
            <span class="price-label">{{ t('modelCatalog.table.paid') }}</span> <span class="price-unit">{{ unit(items[0]?.pricing) }}</span>
          </th>
          <th colspan="3" scope="colgroup" class="price-tier-heading official-start">
            <span class="price-label">{{ t('modelCatalog.table.official') }}</span> <span class="price-unit">{{ unit(items[0]?.official_pricing || items[0]?.pricing) }}</span>
          </th>
          <th :rowspan="perToken ? 2 : 1" scope="col" class="rate-heading rate-start">{{ t('modelCatalog.table.rate') }}</th>
        </tr>
        <tr v-if="perToken">
          <th scope="col" class="input-heading discount-subhead discount-start">{{ t('modelCatalog.pricing.input') }}</th>
          <th scope="col" class="output-heading discount-subhead">{{ t('modelCatalog.pricing.output') }}</th>
          <th scope="col" class="cache-heading discount-subhead">{{ t('modelCatalog.pricing.cache') }}</th>
          <th scope="col" class="input-heading official-start">{{ t('modelCatalog.pricing.input') }}</th>
          <th scope="col" class="output-heading">{{ t('modelCatalog.pricing.output') }}</th>
          <th scope="col" class="cache-heading">{{ t('modelCatalog.pricing.cache') }}</th>
        </tr>
      </thead>
      <tbody v-for="entry in entries" :key="entry.item.slug" :data-model="entry.item.model">
        <tr v-for="(row, index) in entry.rows" :key="row.key">
          <th v-if="index === 0" :rowspan="entry.rows.length" scope="rowgroup" class="model-cell">
            <div class="model-id" :title="entry.item.model">{{ entry.item.model }}</div>
            <p v-if="entry.item.pricing.peak_rate.enabled" class="peak-note">{{ t('modelCatalog.pricing.peakRate', { start: entry.item.pricing.peak_rate.start, end: entry.item.pricing.peak_rate.end, timezone, multiplier: entry.item.pricing.peak_rate.multiplier }) }}</p>
          </th>
          <template v-for="side in sides" :key="side">
            <template v-if="perToken">
              <td class="input-cell" :class="[`${side}-cell`, `${side}-start`, { 'discount-cell': side === 'paid' }]">
                <div class="input-price" :class="{ 'without-context': !row.context }">
                  <span v-if="row.context" class="tier-context" :title="row.context">{{ row.context }}</span>
                  <span>{{ amount(row[side]?.input_price, entry.item, side) }}</span>
                </div>
              </td>
              <td class="numeric-cell" :class="[`${side}-cell`, { 'discount-cell': side === 'paid' }]">{{ amount(row[side]?.output_price, entry.item, side) }}</td>
              <td class="cache-cell" :class="[`${side}-cell`, { 'discount-cell': side === 'paid' }]">
                <div class="cache-content">
                  <span class="cache-part"><span class="cache-key">{{ t('modelCatalog.table.write') }}</span>{{ amount(row[side]?.cache_write_price, entry.item, side) }}</span>
                  <span class="cache-part"><span class="cache-key">{{ t('modelCatalog.table.read') }}</span>{{ amount(row[side]?.cache_read_price, entry.item, side) }}</span>
                </div>
              </td>
            </template>
            <td v-else colspan="3" class="request-amount" :class="[`${side}-cell`, `${side}-start`, { 'discount-cell': side === 'paid' }]">
              <span v-if="requestContext(row, entry.item, side)" class="tier-context request-context">{{ requestContext(row, entry.item, side) }}</span>
              {{ amount(row[side]?.per_request_price, entry.item, side) }}
            </td>
          </template>
          <td v-if="index === 0" :rowspan="entry.rows.length" class="rate-cell rate-start" :title="t('modelCatalog.table.appliedRate')">{{ catalogRateLabel(catalogRate(entry.item)) }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PublicModelCatalogItem, PublicModelCatalogPricing } from '@/api/catalog'
import { catalogAmount, catalogComparisonRows, catalogCurrency, catalogRate, catalogRateLabel } from '@/utils/modelCatalogTable'
import type { CatalogComparisonRow } from '@/utils/modelCatalogTable'

const props = defineProps<{ items: PublicModelCatalogItem[]; timezone: string }>()
const { t } = useI18n()
const sides = ['paid', 'official'] as const
const perToken = computed(() => props.items[0]?.pricing.billing_mode === 'token')
const entries = computed(() => props.items.map(item => ({ item, rows: catalogComparisonRows(item) })))
function unit(pricing?: PublicModelCatalogPricing | null) {
  if (!pricing) return ''
  return `${catalogCurrency(pricing.currency)} ${pricing.billing_mode === 'token' ? '/ 1M token' : t(pricing.billing_mode === 'image' ? 'modelCatalog.pricing.perImage' : 'modelCatalog.pricing.perRequest')}`
}
function amount(value: number | null | undefined, item: PublicModelCatalogItem, side: 'paid' | 'official') {
  return catalogAmount(value, (side === 'official' ? item.official_pricing?.currency : item.pricing.currency) || item.pricing.currency, perToken.value)
}
function requestContext(row: CatalogComparisonRow, item: PublicModelCatalogItem, side: 'paid' | 'official') {
  if (row.context) return row.context
  if (item.pricing.billing_mode !== 'image' || row.key !== 'base') return ''
  if (side === 'official') return t('modelCatalog.table.officialImageBase')
  const defaultTier = item.pricing.intervals.find(tier => tier.tier_label === '2K' && tier.per_request_price === item.pricing.per_request_price)
  return t(defaultTier ? 'modelCatalog.table.defaultImage2K' : 'modelCatalog.table.basePrice')
}
</script>

<style scoped>
    .table-scroll { width: 100%; overflow-x: auto; overscroll-behavior-inline: contain; }
    .price-table {
      width: 100%;
      min-width: 960px;
      table-layout: fixed;
      border-collapse: separate;
      border-spacing: 0;
      font-size: 12px;
    }
    .price-table col.model-col { width: 180px; }
    .price-table col.input-col { width: 124px; }
    .price-table col.output-col { width: 76px; }
    .price-table col.cache-col { width: calc((100% - 654px) / 2); }
    .price-table col.rate-col { width: 74px; }
    .price-table th, .price-table td { padding: 0 8px; vertical-align: middle; }
    .price-table thead th {
      height: 32px;
      border-bottom: 1px solid var(--workspace-divider);
      background: var(--workspace-surface-subtle);
      color: var(--workspace-text-secondary);
      font-weight: 500;
    }
    .price-table thead tr:first-child th { height: 36px; }
    .price-table .model-heading { text-align: left; padding-left: 14px; }
    .price-tier-heading { text-align: center; font-size: 12px; white-space: nowrap; }
    .price-label { font-weight: 600; }
    .price-unit { font-family: var(--workspace-font-mono); font-size: 11px; font-weight: 400; opacity: .75; white-space: nowrap; }
    .price-table .input-heading { text-align: left; }
    .price-table .output-heading, .price-table .cache-heading, .price-table .rate-heading { text-align: center; }
    .price-table .rate-heading { white-space: nowrap; font-size: 12px; }
    .price-table thead .discount-heading,
    .price-table thead .discount-subhead,
    .price-table .paid-cell { background: var(--group-soft); }
    .price-table thead .discount-heading { color: var(--group-accent); box-shadow: inset 0 -1px 0 var(--group-border); }
    .price-table thead .discount-subhead { color: var(--workspace-text-muted); border-bottom-color: var(--group-border); }
    .price-table .paid-cell { color: var(--workspace-text); }
    .price-table .paid-cell.cache-cell { color: var(--workspace-text-secondary); font-weight: 400; }
    .paid-start { border-left: 1px solid var(--group-border); }
    .official-start, .rate-start { border-left: 1px solid var(--workspace-border); }
    .price-table tbody td { height: 36px; padding-block: 2px; border-bottom: 0; }
    .price-table tbody tr:first-child td { padding-top: 8px; }
    .price-table tbody tr:last-child td { padding-bottom: 8px; }
    .model-cell { background: var(--workspace-card-surface); text-align: left; font-weight: 500; }
    .price-table .model-cell { padding: 10px 12px 10px 14px; }
    .model-cell-content { display: flex; min-width: 0; align-items: center; gap: 6px; }
    .model-id { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--workspace-text); font-size: 13px; font-weight: 500; line-height: 20px; }
    .numeric-cell, .input-cell, .cache-cell { color: var(--workspace-text-secondary); font-family: var(--workspace-font-mono); font-variant-numeric: tabular-nums; white-space: nowrap; }
    .numeric-cell { text-align: center; }
    .paid-cell.input-cell, .paid-cell.numeric-cell, .paid-cell.request-amount { font-weight: 700; }
    .input-price { display: flex; align-items: center; justify-content: space-between; gap: 6px; min-height: 24px; }
    .tier-context { white-space: normal; line-height: 1.35; color: var(--workspace-text-muted); font-weight: 400; font-size: 11px; }
    .input-price.without-context { justify-content: flex-end; }
    .cache-content { display: flex; align-items: center; justify-content: center; gap: 9px; min-height: 24px; }
    .cache-part { display: inline-flex; align-items: baseline; gap: 5px; }
    .cache-key { color: var(--workspace-text-muted); font-family: var(--workspace-font-ui); font-size: 11px; font-weight: 400; }
    .rate-cell { background: var(--workspace-card-surface); color: var(--workspace-text); text-align: center; font-family: var(--workspace-font-mono); font-weight: 600; font-variant-numeric: tabular-nums; white-space: nowrap; }
    .per-request-table thead tr:first-child th { height: 56px; }
    .per-request-table tbody td { height: 56px; }
    .request-amount { text-align: center; font-family: var(--workspace-font-mono); font-size: 13px; font-variant-numeric: tabular-nums; }


.discount-start { border-left: 1px solid var(--group-border); }

.price-table tbody + tbody tr:first-child > * { border-top: 1px solid var(--workspace-divider); }
.price-table .model-cell { border-bottom: 0; }
.official-cell { background: var(--workspace-card-surface); }
.price-unit { margin-left: 5px; }
.peak-note { margin: 5px 0 0; color: var(--workspace-text-muted); font-size: 10px; font-weight: 400; line-height: 1.5; }
.request-context { margin-right: 14px; }
.table-scroll:focus-visible { outline: 2px solid var(--workspace-work-accent); outline-offset: -2px; }
@media (max-width: 767px) {
  .price-table .model-heading, .price-table .model-cell { position: sticky; left: 0; z-index: 3; box-shadow: 1px 0 var(--workspace-border); }
  .price-table thead .model-heading { z-index: 4; background: var(--workspace-surface-subtle); }
}
</style>

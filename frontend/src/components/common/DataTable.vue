<template>
  <div
    v-if="!isDesktopViewport"
    class="data-table data-table--mobile space-y-3"
    data-ui-component="data-table"
    data-ui-mode="mobile"
  >
    <template v-if="loading">
      <div v-for="i in 5" :key="i" class="data-table-mobile-card data-table-loading-card rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-900">
        <div class="data-table-mobile-fields space-y-3">
          <div
            v-if="mobileCardConfigured && mobilePrimaryColumn"
            class="data-table-mobile-primary data-table-mobile-primary--loading"
          >
            <div class="data-table-loading-value h-5 w-40 animate-pulse rounded bg-gray-200 dark:bg-dark-700"></div>
          </div>
          <div v-for="column in dataColumns" :key="column.key" class="data-table-mobile-field flex justify-between">
            <div class="data-table-loading-label h-4 w-20 animate-pulse rounded bg-gray-200 dark:bg-dark-700"></div>
            <div class="data-table-loading-value h-4 w-32 animate-pulse rounded bg-gray-200 dark:bg-dark-700"></div>
          </div>
          <div v-if="hasActionsColumn" class="data-table-mobile-actions border-t border-gray-200 pt-3 dark:border-dark-700">
            <div class="data-table-loading-actions h-8 w-full animate-pulse rounded bg-gray-200 dark:bg-dark-700"></div>
          </div>
        </div>
      </div>
    </template>

    <template v-else-if="error">
      <div
        class="data-table-mobile-card data-table-error-card rounded-lg border border-red-200 bg-red-50 p-8 text-center dark:border-red-900/60 dark:bg-red-950/20"
        role="alert"
      >
        <slot name="error" :message="error">
          <div class="data-table-error-state flex flex-col items-center">
            <Icon name="exclamationTriangle" size="xl" class="mb-3 h-10 w-10 text-red-500 dark:text-red-400" />
            <p class="text-sm font-medium text-red-700 dark:text-red-300">{{ error }}</p>
          </div>
        </slot>
      </div>
    </template>

    <template v-else-if="!data || data.length === 0">
      <div class="data-table-mobile-card data-table-empty-card rounded-lg border border-gray-200 bg-white p-12 text-center dark:border-dark-700 dark:bg-dark-900">
        <slot name="empty">
          <div class="data-table-empty-state flex flex-col items-center">
            <Icon
              name="inbox"
              size="xl"
              class="mb-4 h-12 w-12 text-gray-400 dark:text-dark-500"
            />
            <p class="text-lg font-medium text-gray-900 dark:text-gray-100">
              {{ t('empty.noData') }}
            </p>
          </div>
        </slot>
      </div>
    </template>

    <template v-else>
      <div
        v-for="(row, index) in sortedData"
        :key="resolveRowKey(row, index)"
        class="data-table-mobile-card data-table-mobile-row rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-900"
        :class="{ 'cursor-pointer': clickableRows }"
        :role="clickableRows ? 'button' : undefined"
        :tabindex="clickableRows ? 0 : undefined"
        :aria-label="clickableRows ? getRowAriaLabel(row) : undefined"
        @click="clickableRows && emit('rowClick', row)"
        @keydown="handleRowKeydown($event, row)"
      >
        <div class="data-table-mobile-fields space-y-3">
          <div
            v-if="mobileCardConfigured && (mobileSelectColumn || mobilePrimaryColumn)"
            class="data-table-mobile-primary"
            data-mobile-primary
          >
            <div v-if="mobileSelectColumn" class="data-table-mobile-select" @click.stop>
              <slot
                :name="`cell-${mobileSelectColumn.key}`"
                :row="row"
                :value="row[mobileSelectColumn.key]"
                :expanded="actionsExpanded"
              >
                {{ mobileSelectColumn.formatter
                  ? mobileSelectColumn.formatter(row[mobileSelectColumn.key], row)
                  : row[mobileSelectColumn.key] }}
              </slot>
            </div>

            <div v-if="mobilePrimaryColumn" class="data-table-mobile-primary-content">
              <span class="data-table-mobile-primary-label">
                {{ mobilePrimaryColumn.label }}
              </span>
              <div class="data-table-mobile-primary-value">
                <slot
                  :name="`cell-${mobilePrimaryColumn.key}`"
                  :row="row"
                  :value="row[mobilePrimaryColumn.key]"
                  :expanded="actionsExpanded"
                >
                  {{ mobilePrimaryColumn.formatter
                    ? mobilePrimaryColumn.formatter(row[mobilePrimaryColumn.key], row)
                    : row[mobilePrimaryColumn.key] }}
                </slot>
              </div>
            </div>
          </div>

          <div
            v-for="column in dataColumns"
            :key="column.key"
            class="data-table-mobile-field flex items-start justify-between gap-4"
            data-mobile-field
            :data-mobile-field-key="column.key"
          >
            <span class="data-table-mobile-label text-xs font-medium text-gray-500 dark:text-dark-400">
              {{ column.label }}
            </span>
            <div class="data-table-mobile-value text-right text-sm text-gray-900 dark:text-gray-100">
              <slot :name="`cell-${column.key}`" :row="row" :value="row[column.key]" :expanded="actionsExpanded">
                {{ column.formatter ? column.formatter(row[column.key], row) : row[column.key] }}
              </slot>
            </div>
          </div>
          <div v-if="hasActionsColumn" class="data-table-mobile-actions border-t border-gray-200 pt-3 dark:border-dark-700">
            <slot name="cell-actions" :row="row" :value="row['actions']" :expanded="actionsExpanded"></slot>
          </div>
        </div>
      </div>
    </template>
  </div>

  <div
    v-else
    ref="tableWrapperRef"
    class="data-table data-table--desktop table-wrapper"
    data-ui-component="data-table"
    data-ui-mode="desktop"
    :class="{
      'actions-expanded': actionsExpanded,
      'is-scrollable': isScrollable
    }"
  >
    <table class="data-table-table w-full min-w-max divide-y divide-gray-200 dark:divide-dark-700">
      <thead class="data-table-header table-header bg-gray-50 dark:bg-dark-800">
        <tr class="data-table-header-row">
          <th
            v-for="(column, index) in columns"
            :key="column.key"
            scope="col"
            :aria-sort="column.sortable ? getColumnAriaSort(column.key) : undefined"
            :tabindex="column.sortable ? 0 : undefined"
            :class="[
              'data-table-header-cell sticky-header-cell py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400',
              getAdaptivePaddingClass(),
              { 'cursor-pointer hover:bg-gray-100 dark:hover:bg-dark-700': column.sortable },
              getStickyColumnClass(column, index),
              column.class
            ]"
            @click="column.sortable && handleSort(column.key)"
            @keydown.enter.prevent="column.sortable && handleSort(column.key)"
            @keydown.space.prevent="column.sortable && handleSort(column.key)"
          >
            <slot
              :name="`header-${column.key}`"
              :column="column"
              :sort-key="sortKey"
              :sort-order="sortOrder"
            >
              <div :class="['data-table-header-content flex items-center space-x-1', getHeaderContentAlignmentClass(column)]">
                <span>{{ column.label }}</span>
                <span
                  v-if="column.sortable"
                  class="data-table-sort-indicator inline-flex h-5 w-4 flex-col items-center justify-center"
                  aria-hidden="true"
                >
                  <svg
                    class="h-2.5 w-2.5"
                    :class="getSortIndicatorClass(column.key, 'asc')"
                    fill="currentColor"
                    viewBox="0 0 10 10"
                  >
                    <path d="M5 2L1.5 6.5h7L5 2z" />
                  </svg>
                  <svg
                    class="-mt-0.5 h-2.5 w-2.5"
                    :class="getSortIndicatorClass(column.key, 'desc')"
                    fill="currentColor"
                    viewBox="0 0 10 10"
                  >
                    <path d="M5 8L1.5 3.5h7L5 8z" />
                  </svg>
                </span>
              </div>
            </slot>
          </th>
        </tr>
      </thead>
      <tbody class="data-table-body table-body divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
        <!-- Loading skeleton -->
        <tr v-if="loading" v-for="i in 5" :key="i" class="data-table-loading-row">
          <td v-for="column in columns" :key="column.key" :class="['data-table-loading-cell whitespace-nowrap py-4', getAdaptivePaddingClass()]">
            <div class="data-table-loading-pulse animate-pulse">
              <div class="data-table-loading-line h-4 w-3/4 rounded bg-gray-200 dark:bg-dark-700"></div>
            </div>
          </td>
        </tr>

        <!-- Error state -->
        <tr v-else-if="error" class="data-table-error-row">
          <td
            :colspan="columns.length"
            :class="['data-table-error-cell py-12 text-center', getAdaptivePaddingClass()]"
            role="alert"
          >
            <slot name="error" :message="error">
              <div class="data-table-error-state flex flex-col items-center">
                <Icon name="exclamationTriangle" size="xl" class="mb-3 h-10 w-10 text-red-500 dark:text-red-400" />
                <p class="text-sm font-medium text-red-700 dark:text-red-300">{{ error }}</p>
              </div>
            </slot>
          </td>
        </tr>

        <!-- Empty state -->
        <tr v-else-if="!data || data.length === 0" class="data-table-empty-row">
          <td
            :colspan="columns.length"
            :class="['data-table-empty-cell py-12 text-center text-gray-500 dark:text-dark-400', getAdaptivePaddingClass()]"
          >
            <slot name="empty">
              <div class="data-table-empty-state flex flex-col items-center">
                <Icon
                  name="inbox"
                  size="xl"
                  class="mb-4 h-12 w-12 text-gray-400 dark:text-dark-500"
                />
                <p class="text-lg font-medium text-gray-900 dark:text-gray-100">
                  {{ t('empty.noData') }}
                </p>
              </div>
            </slot>
          </td>
        </tr>

        <!-- Data rows: windowed when large, fully rendered when small (shared row/cell template) -->
        <template v-else>
          <tr v-if="virtualPaddingTop > 0" class="data-table-virtual-spacer data-table-virtual-spacer--top" aria-hidden="true">
            <td :colspan="columns.length"
                :style="{ height: virtualPaddingTop + 'px', padding: 0, border: 'none' }">
            </td>
          </tr>
          <tr
            v-for="item in renderRows"
            :key="resolveRowKey(item.row, item.index)"
            :data-row-id="resolveRowKey(item.row, item.index)"
            :data-index="item.index"
            :ref="item.measure ? measureElement : undefined"
            class="data-table-row hover:bg-gray-50 dark:hover:bg-dark-800"
            :class="{ 'cursor-pointer': clickableRows }"
            :tabindex="clickableRows ? 0 : undefined"
            :aria-label="clickableRows ? getRowAriaLabel(item.row) : undefined"
            @click="clickableRows && emit('rowClick', item.row)"
            @keydown="handleRowKeydown($event, item.row)"
          >
            <td
              v-for="(column, colIndex) in columns"
              :key="column.key"
              :class="[
                'data-table-cell whitespace-nowrap py-4 text-sm text-gray-900 dark:text-gray-100',
                getAdaptivePaddingClass(),
                getStickyColumnClass(column, colIndex),
                column.class
              ]"
            >
              <slot :name="`cell-${column.key}`"
                    :row="item.row"
                    :value="item.row[column.key]"
                    :expanded="actionsExpanded">
                {{ column.formatter
                   ? column.formatter(item.row[column.key], item.row)
                   : item.row[column.key] }}
              </slot>
            </td>
          </tr>
          <tr v-if="virtualPaddingBottom > 0" class="data-table-virtual-spacer data-table-virtual-spacer--bottom" aria-hidden="true">
            <td :colspan="columns.length"
                :style="{ height: virtualPaddingBottom + 'px', padding: 0, border: 'none' }">
            </td>
          </tr>
        </template>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useVirtualizer, observeElementRect as observeElementRectDefault } from '@tanstack/vue-virtual'
import { useI18n } from 'vue-i18n'
import type { Column } from './types'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()

// Match TablePageLayout's structural breakpoint so tablet widths never mix
// the mobile page shell with the desktop table implementation.
const desktopViewportQuery = '(min-width: 1024px)'
const isDesktopViewport = ref(
  typeof window === 'undefined' ? true : window.matchMedia(desktopViewportQuery).matches
)

const emit = defineEmits<{
  sort: [key: string, order: 'asc' | 'desc']
  rowClick: [row: any]
}>()

// 表格容器引用
const tableWrapperRef = ref<HTMLElement | null>(null)
const isScrollable = ref(false)
const actionsColumnNeedsExpanding = ref(false)

// --- 虚拟滚动「整表空白」根治 ---
// 根因:本组件根 .table-wrapper 为 flex:1 / min-h-0,高度由父级 flex 链决定。@tanstack 虚拟化器
// 仅在 observeElementRect 回调里写 scrollRect;一旦该回调读到 0 高度(加载瞬间 flex 未结算,或
// 滚动中动态行高校正触发的 reflow),scrollRect 被钉死为 0 → calculateRange 返回 null → 整表空白。
// 对策(见下方 virtualizer 选项):
//   1) 覆写 observeElementRect,直接丢弃 height<=0 的读数,scrollRect 永不被钉成 0;
//   2) initialRect 给一屏兜底高度,首个有效读数到来前也有行可渲染,绝不空白。
// 兜底高度:表格区域大致 = 视口高度 - 顶栏/外边距/筛选/分页 ≈ 320px
const estimatedViewportHeight = () => {
  if (typeof window === 'undefined') return 600
  return Math.max(window.innerHeight - 320, 400)
}

// 覆写默认 observeElementRect:过滤掉 0 高度读数(根治整表空白的关键)
const observeElementRectNonZero = (
  instance: any,
  cb: (rect: { width: number; height: number }) => void
) => observeElementRectDefault(instance, (rect) => {
  if (rect.height > 0) cb(rect)
})

// 检查是否可滚动
const checkScrollable = () => {
  if (tableWrapperRef.value) {
    isScrollable.value = tableWrapperRef.value.scrollWidth > tableWrapperRef.value.clientWidth
  }
}

// 检查操作列是否需要展开
const checkActionsColumnWidth = () => {
  if (!props.expandableActions) {
    actionsColumnNeedsExpanding.value = false
    actionsExpanded.value = false
    return
  }
  if (!tableWrapperRef.value) return

  // 查找第一行的操作列单元格
  const firstActionCell = tableWrapperRef.value.querySelector('tbody tr:first-child td:last-child')
  if (!firstActionCell) return

  // 查找操作列内容的容器div
  const actionsContainer = firstActionCell.querySelector('div')
  if (!actionsContainer) return

  // 临时展开以测量完整宽度
  const wasExpanded = actionsExpanded.value
  actionsExpanded.value = true

  // 等待DOM更新
  nextTick(() => {
    // 测量所有按钮的总宽度
    const actionItems = actionsContainer.querySelectorAll('button, a, [role="button"]')
    if (actionItems.length <= 2) {
      actionsColumnNeedsExpanding.value = false
      actionsExpanded.value = wasExpanded
      return
    }

    // 计算所有按钮的总宽度（包括gap）
    let totalWidth = 0
    actionItems.forEach((item, index) => {
      totalWidth += (item as HTMLElement).offsetWidth
      if (index < actionItems.length - 1) {
        totalWidth += 4 // gap-1 = 4px
      }
    })

    // 获取单元格可用宽度（减去padding）
    const cellWidth = (firstActionCell as HTMLElement).clientWidth - 32 // 减去左右padding

    // 如果总宽度超过可用宽度，需要展开功能
    actionsColumnNeedsExpanding.value = totalWidth > cellWidth

    // 恢复原来的展开状态
    actionsExpanded.value = wasExpanded
  })
}

// 监听尺寸变化
let resizeObserver: ResizeObserver | null = null
let resizeHandler: (() => void) | null = null
let desktopViewportMediaQuery: MediaQueryList | null = null
let desktopViewportListener: ((event: MediaQueryListEvent) => void) | null = null

const detachDesktopTableTracking = () => {
  resizeObserver?.disconnect()
  resizeObserver = null
  if (resizeHandler) {
    window.removeEventListener('resize', resizeHandler)
    resizeHandler = null
  }
}

const attachDesktopTableTracking = () => {
  checkScrollable()
  checkActionsColumnWidth()
  if (tableWrapperRef.value && typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver(() => {
      checkScrollable()
      checkActionsColumnWidth()
    })
    resizeObserver.observe(tableWrapperRef.value)
  } else {
    // 降级方案：不支持 ResizeObserver 时使用 window resize
    resizeHandler = () => {
      checkScrollable()
      checkActionsColumnWidth()
    }
    window.addEventListener('resize', resizeHandler)
  }
}

onMounted(() => {
  if (typeof window !== 'undefined') {
    desktopViewportMediaQuery = window.matchMedia(desktopViewportQuery)
    isDesktopViewport.value = desktopViewportMediaQuery.matches
    desktopViewportListener = (event: MediaQueryListEvent) => {
      isDesktopViewport.value = event.matches
    }
    if (typeof desktopViewportMediaQuery.addEventListener === 'function') {
      desktopViewportMediaQuery.addEventListener('change', desktopViewportListener)
    } else {
      desktopViewportMediaQuery.addListener(desktopViewportListener)
    }
  }
})

onUnmounted(() => {
  detachDesktopTableTracking()
  if (desktopViewportMediaQuery && desktopViewportListener) {
    if (typeof desktopViewportMediaQuery.removeEventListener === 'function') {
      desktopViewportMediaQuery.removeEventListener('change', desktopViewportListener)
    } else {
      desktopViewportMediaQuery.removeListener(desktopViewportListener)
    }
    desktopViewportListener = null
  }
  desktopViewportMediaQuery = null
})

interface Props {
  columns: Column[]
  data: any[]
  loading?: boolean
  /** Human-readable load error. Rendered before the empty state when present. */
  error?: string | null
  stickyFirstColumn?: boolean
  stickyActionsColumn?: boolean
  expandableActions?: boolean
  actionsCount?: number // 操作按钮总数，用于判断是否需要展开功能
  rowKey?: string | ((row: any) => string | number)
  /**
   * Default sort configuration (only applied when there is no persisted sort state)
   */
  defaultSortKey?: string
  defaultSortOrder?: 'asc' | 'desc'
  /**
   * Persist sort state (key + order) to localStorage using this key.
   * If provided, DataTable will load the stored sort state on mount.
   */
  sortStorageKey?: string
  /**
   * Enable server-side sorting mode. When true, clicking sort headers
   * will emit 'sort' events instead of performing client-side sorting.
   */
  serverSideSort?: boolean
  /** Emit 'rowClick' on row/card click and show pointer cursor (interactive cells should @click.stop) */
  clickableRows?: boolean
  /** Accessible label for clickable rows/cards. Falls back to the first data column. */
  rowAriaLabel?: (row: any) => string
  /**
   * Promote one column into the mobile card header. Desktop rendering is unchanged.
   * When omitted, the legacy mobile field list is preserved.
   */
  mobilePrimaryKey?: string
  /**
   * Ordered mobile field allow-list. Selection, primary and actions columns are
   * handled separately, so callers only list supporting information here.
   */
  mobileVisibleKeys?: string[]
  /** Estimated row height in px for the virtualizer (default 56) */
  estimateRowHeight?: number
  /** Number of rows to render beyond the visible area (default 5) */
  overscan?: number
  /**
   * Only virtualize when the row count exceeds this threshold (default 100).
   * Smaller lists render in full, avoiding the scroll-compensation jank caused by
   * estimated-vs-actual row heights when rows have variable height.
   */
  virtualizeThreshold?: number
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  error: null,
  stickyFirstColumn: true,
  stickyActionsColumn: true,
  expandableActions: true,
  defaultSortOrder: 'asc',
  serverSideSort: false
})

const sortKey = ref<string>('')
const sortOrder = ref<'asc' | 'desc'>('asc')
const actionsExpanded = ref(false)

type PersistedSortState = {
  key: string
  order: 'asc' | 'desc'
}

const collator = new Intl.Collator(undefined, {
  numeric: true,
  sensitivity: 'base'
})

const getSortableKeys = () => {
  const keys = new Set<string>()
  for (const col of props.columns) {
    if (col.sortable) keys.add(col.key)
  }
  return keys
}

const normalizeSortKey = (candidate: string) => {
  if (!candidate) return ''
  const sortableKeys = getSortableKeys()
  return sortableKeys.has(candidate) ? candidate : ''
}

const normalizeSortOrder = (candidate: any): 'asc' | 'desc' => {
  return candidate === 'desc' ? 'desc' : 'asc'
}

const readPersistedSortState = (): PersistedSortState | null => {
  if (!props.sortStorageKey) return null
  try {
    const raw = localStorage.getItem(props.sortStorageKey)
    if (!raw) return null
    const parsed = JSON.parse(raw) as Partial<PersistedSortState>
    const key = normalizeSortKey(typeof parsed.key === 'string' ? parsed.key : '')
    if (!key) return null
    return { key, order: normalizeSortOrder(parsed.order) }
  } catch (e) {
    console.error('[DataTable] Failed to read persisted sort state:', e)
    return null
  }
}

const writePersistedSortState = (state: PersistedSortState) => {
  if (!props.sortStorageKey) return
  try {
    localStorage.setItem(props.sortStorageKey, JSON.stringify(state))
  } catch (e) {
    console.error('[DataTable] Failed to persist sort state:', e)
  }
}

const resolveInitialSortState = (): PersistedSortState | null => {
  const persisted = readPersistedSortState()
  if (persisted) return persisted

  const key = normalizeSortKey(props.defaultSortKey || '')
  if (!key) return null
  return { key, order: normalizeSortOrder(props.defaultSortOrder) }
}

const applySortState = (state: PersistedSortState | null) => {
  if (!state) return
  sortKey.value = state.key
  sortOrder.value = state.order
}

const getSortIndicatorClass = (key: string, order: 'asc' | 'desc') => {
  return sortKey.value === key && sortOrder.value === order
    ? 'data-table-sort-active'
    : 'data-table-sort-inactive transition-colors'
}

const getColumnAriaSort = (key: string) => {
  if (sortKey.value !== key) return 'none'
  return sortOrder.value === 'asc' ? 'ascending' : 'descending'
}

const getHeaderContentAlignmentClass = (column: Column) => {
  const className = column.class || ''
  if (className.includes('text-center')) return 'justify-center'
  if (className.includes('text-right')) return 'justify-end'
  return 'justify-start'
}

const isNullishOrEmpty = (value: any) => value === null || value === undefined || value === ''

const toFiniteNumberOrNull = (value: any): number | null => {
  if (typeof value === 'number') return Number.isFinite(value) ? value : null
  if (typeof value === 'boolean') return value ? 1 : 0
  if (typeof value === 'string') {
    const trimmed = value.trim()
    if (!trimmed) return null
    const n = Number(trimmed)
    return Number.isFinite(n) ? n : null
  }
  return null
}

const toSortableString = (value: any): string => {
  if (value === null || value === undefined) return ''
  if (typeof value === 'string') return value
  if (typeof value === 'number' || typeof value === 'boolean') return String(value)
  if (value instanceof Date) return value.toISOString()
  try {
    return JSON.stringify(value)
  } catch {
    return String(value)
  }
}

const compareSortValues = (a: any, b: any): number => {
  const aEmpty = isNullishOrEmpty(a)
  const bEmpty = isNullishOrEmpty(b)
  if (aEmpty && bEmpty) return 0
  if (aEmpty) return 1
  if (bEmpty) return -1

  const aNum = toFiniteNumberOrNull(a)
  const bNum = toFiniteNumberOrNull(b)
  if (aNum !== null && bNum !== null) {
    if (aNum === bNum) return 0
    return aNum < bNum ? -1 : 1
  }

  const aStr = toSortableString(a)
  const bStr = toSortableString(b)
  const res = collator.compare(aStr, bStr)
  if (res === 0) return 0
  return res < 0 ? -1 : 1
}
const resolveStableRowKey = (row: any): string | number | undefined => {
  if (typeof props.rowKey === 'function') {
    const key = props.rowKey(row)
    return key ?? undefined
  }
  if (typeof props.rowKey === 'string' && props.rowKey) {
    const key = row?.[props.rowKey]
    return key ?? undefined
  }
  const key = row?.id
  return key ?? undefined
}

const resolveRowKey = (row: any, index: number) => resolveStableRowKey(row) ?? index

const getRowAriaLabel = (row: any) => {
  const explicitLabel = props.rowAriaLabel?.(row)?.trim()
  if (explicitLabel) return explicitLabel

  const firstColumn = props.columns.find((column) => column.key !== 'actions' && column.key !== 'select')
  if (!firstColumn) return undefined
  const rawValue = row?.[firstColumn.key]
  const displayValue = firstColumn.formatter ? firstColumn.formatter(rawValue, row) : rawValue
  return displayValue == null ? undefined : String(displayValue)
}

const handleRowKeydown = (event: KeyboardEvent, row: any) => {
  if (!props.clickableRows || event.target !== event.currentTarget) return
  if (event.key !== 'Enter' && event.key !== ' ') return
  event.preventDefault()
  emit('rowClick', row)
}

const mobileCardConfigured = computed(() =>
  Boolean(props.mobilePrimaryKey || props.mobileVisibleKeys)
)

const mobilePrimaryColumn = computed(() => {
  if (!mobileCardConfigured.value || !props.mobilePrimaryKey) return null
  return props.columns.find(column => column.key === props.mobilePrimaryKey) ?? null
})

const mobileSelectColumn = computed(() => {
  if (!mobileCardConfigured.value) return null
  return props.columns.find(column => column.key === 'select') ?? null
})

const dataColumns = computed(() => {
  if (!mobileCardConfigured.value) {
    return props.columns.filter(column => column.key !== 'actions')
  }

  const excludedKeys = new Set([
    'actions',
    'select',
    ...(props.mobilePrimaryKey ? [props.mobilePrimaryKey] : [])
  ])

  if (!props.mobileVisibleKeys) {
    return props.columns.filter(column => !excludedKeys.has(column.key))
  }

  const columnsByKey = new Map(props.columns.map(column => [column.key, column]))
  return props.mobileVisibleKeys
    .filter(key => !excludedKeys.has(key))
    .map(key => columnsByKey.get(key))
    .filter((column): column is Column => Boolean(column))
})
const columnsSignature = computed(() =>
  props.columns.map((column) => `${column.key}:${column.sortable ? '1' : '0'}`).join('|')
)

watch(
  isDesktopViewport,
  async (isDesktop) => {
    detachDesktopTableTracking()
    if (!isDesktop) return
    await nextTick()
    attachDesktopTableTracking()
  },
  { immediate: true, flush: 'post' }
)

// 数据/列变化时重新检查滚动状态
// 注意：不能监听 actionsExpanded，因为 checkActionsColumnWidth 会临时修改它，会导致无限循环
watch(
  [() => props.data.length, columnsSignature],
  async () => {
    await nextTick()
    checkScrollable()
    checkActionsColumnWidth()
  },
  { flush: 'post' }
)

// 单独监听展开状态变化，只更新滚动状态
watch(actionsExpanded, async () => {
  await nextTick()
  checkScrollable()
})

const handleSort = (key: string) => {
  let newOrder: 'asc' | 'desc' = 'asc'
  if (sortKey.value === key) {
    newOrder = sortOrder.value === 'asc' ? 'desc' : 'asc'
  }

  if (props.serverSideSort) {
    // Server-side sort mode: emit event and update internal state for UI feedback
    sortKey.value = key
    sortOrder.value = newOrder
    emit('sort', key, newOrder)
  } else {
    // Client-side sort mode: just update internal state
    sortKey.value = key
    sortOrder.value = newOrder
  }
}

const sortedData = computed(() => {
  // Server-side sort mode: return data as-is (server handles sorting)
  if (props.serverSideSort || !sortKey.value || !props.data) return props.data

  const key = sortKey.value
  const order = sortOrder.value

  // Stable sort (tie-break with original index) to avoid jitter when values are equal.
  return props.data
    .map((row, index) => ({ row, index }))
    .sort((a, b) => {
      const cmp = compareSortValues(a.row?.[key], b.row?.[key])
      if (cmp !== 0) return order === 'asc' ? cmp : -cmp
      return a.index - b.index
    })
    .map(item => item.row)
})

// --- Virtual scrolling ---
// 是否启用虚拟化:仅桌面端且行数超过阈值时开启。小列表全量渲染,彻底绕开虚拟器的
// 估算/测量/滚动补偿链路,消除可变行高导致的滚动抖动。
const shouldVirtualize = computed(() =>
  isDesktopViewport.value && (sortedData.value?.length ?? 0) > (props.virtualizeThreshold ?? 100)
)

const rowVirtualizer = useVirtualizer(computed(() => ({
  count: shouldVirtualize.value ? (sortedData.value?.length ?? 0) : 0,
  getScrollElement: () => tableWrapperRef.value,
  // 用行主键(与模板 :key 一致)而非默认的 index 作为 itemSizeCache 键,
  // 这样排序/筛选/跨阈值来回都能复用正确的已测行高,而不是残留的按 index 缓存 → 消除高度校正抖动。
  getItemKey: (index: number) => {
    const row = sortedData.value?.[index]
    return row != null ? resolveRowKey(row, index) : index
  },
  estimateSize: () => props.estimateRowHeight ?? 56,
  overscan: props.overscan ?? 5,
  // 兜底高度:首个有效高度读数到来前,先按一屏渲染,避免空白帧
  initialRect: { width: 0, height: estimatedViewportHeight() },
  // 关键:过滤 0 高度读数,杜绝 scrollRect 被钉成 0 → calculateRange 返回 null → 整表空白
  observeElementRect: observeElementRectNonZero,
  // 把测量类 ResizeObserver 回调批到 rAF,避免滚动中同步 reflow 风暴导致的校正抖动/空白
  useAnimationFrameWithResizeObserver: true,
})))

const virtualItems = computed(() => rowVirtualizer.value.getVirtualItems())

const virtualPaddingTop = computed(() => {
  const items = virtualItems.value
  return items.length > 0 ? items[0].start : 0
})

const virtualPaddingBottom = computed(() => {
  const items = virtualItems.value
  if (items.length === 0) return 0
  return rowVirtualizer.value.getTotalSize() - items[items.length - 1].end
})

const measureElement = (el: any) => {
  if (el) {
    rowVirtualizer.value.measureElement(el as Element)
  }
}

type RowIdentityToken = string | number | object | symbol

const rowIdentityKeys = computed<RowIdentityToken[]>(() =>
  (sortedData.value ?? []).map((row) => {
    const stableKey = resolveStableRowKey(row)
    if (stableKey !== undefined) return stableKey

    // Object references survive pure reordering but change across page/filter results.
    // Primitive rows have no stable identity, so force conservative invalidation.
    return row !== null && typeof row === 'object' ? row : Symbol('unstable-row')
  })
)

const hasSameRowIdentitySet = (
  current: RowIdentityToken[],
  previous: RowIdentityToken[]
) => {
  if (current.length !== previous.length) return false
  const currentKeys = new Set(current)
  const previousKeys = new Set(previous)
  // Duplicate keys make row-to-cache ownership ambiguous, even when the unique
  // key set looks unchanged (for example [1, 1, 2] -> [1, 2, 2]).
  if (currentKeys.size !== current.length || previousKeys.size !== previous.length) return false
  return [...currentKeys].every(key => previousKeys.has(key))
}

watch(
  rowIdentityKeys,
  (current, previous) => {
    if (hasSameRowIdentitySet(current, previous)) return

    // The virtualizer owns caches across option updates. A new page/filter result
    // must release detached rows and sizes, while pure reordering keeps them.
    rowVirtualizer.value.measureElement(null)
    rowVirtualizer.value.measure()
  },
  { flush: 'post' }
)

// 统一的渲染行列表:虚拟化开启时只取窗口内的行(需 measure 交给虚拟器测量),
// 关闭时取全部行(无需测量)。模板据此渲染,两种模式共用同一套单元格结构。
const renderRows = computed<Array<{ index: number; row: any; measure: boolean }>>(() => {
  const data = sortedData.value ?? []
  if (shouldVirtualize.value) {
    return virtualItems.value.map(vr => ({ index: vr.index, row: data[vr.index], measure: true }))
  }
  return data.map((row, index) => ({ index, row, measure: false }))
})

const hasActionsColumn = computed(() => {
  return props.columns.some(column => column.key === 'actions')
})

const hasSelectColumn = computed(() => {
  return props.columns.length > 0 && props.columns[0].key === 'select'
})

// 生成固定列的 CSS 类
const getStickyColumnClass = (column: Column, index: number) => {
  const classes: string[] = []

  if (props.stickyFirstColumn) {
    // 如果第一列是勾选列，固定前两列（勾选+名称）
    if (hasSelectColumn.value) {
      if (index === 0) {
        classes.push('sticky-col sticky-col-left-first')
      } else if (index === 1) {
        classes.push('sticky-col sticky-col-left-second')
      }
    } else {
      // 否则只固定第一列
      if (index === 0) {
        classes.push('sticky-col sticky-col-left')
      }
    }
  }

  // 操作列固定（最后一列）
  if (props.stickyActionsColumn && column.key === 'actions') {
    classes.push('sticky-col sticky-col-right')
  }

  return classes.join(' ')
}

// 根据列数自适应调整内边距
const getAdaptivePaddingClass = () => {
  const columnCount = props.columns.length

  // 列数越多，内边距越小
  if (columnCount >= 10) {
    return 'px-2' // 8px
  } else if (columnCount >= 7) {
    return 'px-3' // 12px
  } else if (columnCount >= 5) {
    return 'px-4' // 16px
  } else {
    return 'px-6' // 24px (原始值)
  }
}

// Init + keep persisted sort state consistent with current columns
const didInitSort = ref(false)

onMounted(() => {
  const initial = resolveInitialSortState()
  applySortState(initial)
  didInitSort.value = true
})

watch(
  columnsSignature,
  () => {
    // If current sort key is no longer sortable/visible, fall back to default/persisted.
    const normalized = normalizeSortKey(sortKey.value)
    if (!sortKey.value) {
      const initial = resolveInitialSortState()
      applySortState(initial)
      return
    }

    if (!normalized) {
      const fallback = resolveInitialSortState()
      if (fallback) {
        applySortState(fallback)
      } else {
        sortKey.value = ''
        sortOrder.value = 'asc'
      }
    }
  },
  { flush: 'post' }
)

watch(
  [sortKey, sortOrder],
  ([nextKey, nextOrder]) => {
    if (!didInitSort.value) return
    if (!props.sortStorageKey) return
    const key = normalizeSortKey(nextKey)
    if (!key) return
    writePersistedSortState({ key, order: normalizeSortOrder(nextOrder) })
  },
  { flush: 'post' }
)

defineExpose({
  virtualizer: rowVirtualizer,
  shouldVirtualize,
  sortedData,
  resolveRowKey,
  tableWrapperEl: tableWrapperRef,
})
</script>

<style scoped>
/* Mobile cards use one clear identity row followed by a deliberately short
   list of supporting fields. Pages opt into this structure through the mobile
   props; legacy callers keep their previous field list unchanged. */
.data-table--mobile {
  min-width: 0;
}

.data-table-mobile-card {
  min-width: 0;
  border-color: var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-card);
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-flat);
}

.data-table-mobile-primary {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  gap: 0.75rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid var(--lx-clay-border);
}

.data-table-mobile-primary--loading {
  min-height: 2.5rem;
  align-items: center;
}

.data-table-mobile-select {
  display: flex;
  min-width: 2.75rem;
  min-height: 2.75rem;
  align-items: center;
  justify-content: center;
}

.data-table-mobile-primary-content,
.data-table-mobile-value {
  min-width: 0;
  overflow-wrap: anywhere;
}

.data-table-mobile-primary-content {
  flex: 1;
}

.data-table-mobile-primary-label,
.data-table-mobile-label {
  color: var(--lx-clay-text-muted);
}

.data-table-mobile-primary-label {
  display: block;
  margin-bottom: 0.25rem;
  font-size: 0.75rem;
  font-weight: 650;
  line-height: 1rem;
}

.data-table-mobile-primary-value {
  min-width: 0;
  color: var(--lx-clay-text);
  font-size: 1rem;
  font-weight: 750;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

.data-table-mobile-field {
  min-width: 0;
}

.data-table-mobile-label {
  flex: 0 0 auto;
  line-height: 1.35;
}

.data-table-mobile-actions {
  border-color: var(--lx-clay-border);
}

@media (pointer: coarse), (max-width: 1023px) {
  .data-table--mobile :deep(button),
  .data-table--mobile :deep(a[role='button']) {
    min-height: 44px;
  }
}

@media (min-width: 768px) and (max-width: 1023px) {
  .data-table--mobile {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.75rem;
  }

  .data-table--mobile > :not([hidden]) ~ :not([hidden]) {
    margin-top: 0 !important;
  }

  .data-table-error-card,
  .data-table-empty-card {
    grid-column: 1 / -1;
  }
}

/* 表格横向滚动 */
.table-wrapper {
  --select-col-width: 52px; /* 勾选列宽度：px-6 (24px*2) + checkbox (16px) */
  position: relative;
  overflow-x: auto;
  overflow-y: auto;
  flex: 1;
  min-height: 0;
  isolation: isolate;
}

/* 表头容器，确保在滚动时覆盖表体内容 */
.table-wrapper .table-header {
  position: sticky;
  top: 0;
  z-index: 200;
  background-color: var(--lx-clay-recessed);
}

.dark .table-wrapper .table-header {
  background-color: var(--lx-clay-recessed);
}

/* 表体保持在表头下方 */
.table-body {
  position: relative;
  z-index: 0;
}

/* 所有表头单元格固定在顶部 */
.sticky-header-cell {
  position: sticky;
  top: 0;
  z-index: 210; /* 必须高于所有表体内容 */
  background-color: var(--lx-clay-recessed);
}

.dark .sticky-header-cell {
  background-color: var(--lx-clay-recessed);
}

/* Sticky 列基础样式 */
.sticky-col {
  position: sticky;
  z-index: 20; /* 表体固定列 */
}

/* 单列固定（无勾选列时） */
.sticky-col-left {
  left: 0;
}

/* 双列固定（有勾选列时）：第一列（勾选） */
.sticky-col-left-first {
  left: 0;
}

/* 双列固定（有勾选列时）：第二列（名称） */
.sticky-col-left-second {
  left: var(--select-col-width);
}

/* 操作列固定 */
.sticky-col-right {
  right: 0;
}

/* 表头 sticky 列 - 需要比普通表头单元格更高的 z-index */
.sticky-header-cell.sticky-col {
  z-index: 220; /* 高于普通表头单元格和表体固定列 */
}

/* 表体 sticky 列背景 */
tbody .sticky-col {
  background-color: var(--lx-clay-surface);
}

.dark tbody .sticky-col {
  background-color: var(--lx-clay-surface);
}

/* hover 状态保持 */
tbody tr:hover .sticky-col {
  background-color: color-mix(in srgb, var(--lx-clay-surface) 90%, var(--lx-clay-accent-soft));
}

.dark tbody tr:hover .sticky-col {
  background-color: color-mix(in srgb, var(--lx-clay-surface) 90%, var(--lx-clay-accent-soft));
}

/* 阴影只在可滚动时显示 */
/* 单列固定右侧阴影 */
.is-scrollable .sticky-col-left::after {
  content: '';
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  width: 10px;
  transform: translateX(100%);
  background: linear-gradient(to right, color-mix(in srgb, var(--lx-clay-text) 8%, transparent), transparent);
  pointer-events: none;
}

/* 双列固定：只在第二列显示阴影 */
.is-scrollable .sticky-col-left-second::after {
  content: '';
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  width: 10px;
  transform: translateX(100%);
  background: linear-gradient(to right, color-mix(in srgb, var(--lx-clay-text) 8%, transparent), transparent);
  pointer-events: none;
}

/* 操作列左侧阴影 */
.is-scrollable .sticky-col-right::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  bottom: 0;
  width: 10px;
  transform: translateX(-100%);
  background: linear-gradient(to left, color-mix(in srgb, var(--lx-clay-text) 8%, transparent), transparent);
  pointer-events: none;
}

/* 暗色模式阴影 */
.dark .is-scrollable .sticky-col-left::after,
.dark .is-scrollable .sticky-col-left-second::after {
  background: linear-gradient(to right, color-mix(in srgb, var(--lx-clay-text) 12%, transparent), transparent);
}

.dark .is-scrollable .sticky-col-right::before {
  background: linear-gradient(to left, color-mix(in srgb, var(--lx-clay-text) 12%, transparent), transparent);
}
</style>

<style>
/* ==========================================================================
   终极悬浮滚动条防丢器 (Sledgehammer Override)
   绕过 style.css 中 `* { scrollbar-color: transparent }` 的全局悬停隐身诅咒！
   ========================================================================== */

/* 1. 废除全局针对所有元素的 scrollbar-width 设定，拿回 Chrome/Safari 下 Webkit 滚动条规则的控制权！ */
.table-wrapper {
  scrollbar-width: auto !important; /* 阻止 Chrome 121 退化到原生 Mac 闪隐滚动条 */
}

/* 2. 重写 Webkit 滚动层，全部加上 !important 强制覆盖透明悬停陷阱 */
.table-wrapper::-webkit-scrollbar {
  height: 12px !important;
  width: 12px !important;
  display: block !important;
  background-color: transparent !important;
}

.table-wrapper::-webkit-scrollbar-track {
  background-color: var(--lx-clay-recessed) !important;
  border-radius: 6px !important;
  margin: 0 4px !important;
}
.dark .table-wrapper::-webkit-scrollbar-track {
  background-color: var(--lx-clay-recessed) !important;
}

/* 常驻、不透明的滑块，无视鼠标是否 hover 都在那！ */
.table-wrapper::-webkit-scrollbar-thumb {
  background-color: color-mix(in srgb, var(--lx-clay-text-muted) 72%, transparent) !important;
  border-radius: 6px !important;
  border: 2px solid transparent !important;
  background-clip: padding-box !important;
  -webkit-appearance: none !important;
}
.table-wrapper::-webkit-scrollbar-thumb:hover {
  background-color: var(--lx-clay-text-secondary) !important;
}

.dark .table-wrapper::-webkit-scrollbar-thumb {
  background-color: color-mix(in srgb, var(--lx-clay-text-muted) 72%, transparent) !important;
}
.dark .table-wrapper::-webkit-scrollbar-thumb:hover {
  background-color: var(--lx-clay-text-secondary) !important;
}

/* 3. 仅给真正的 Firefox 留的后路 */
@supports (-moz-appearance:none) {
  .table-wrapper {
    scrollbar-width: thin !important;
    scrollbar-color: var(--lx-clay-text-muted) var(--lx-clay-recessed) !important;
  }
  .dark .table-wrapper {
    scrollbar-color: var(--lx-clay-text-muted) var(--lx-clay-recessed) !important;
  }
}
</style>

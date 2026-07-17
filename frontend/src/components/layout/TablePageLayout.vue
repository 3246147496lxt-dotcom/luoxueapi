<template>
  <div class="table-page-layout" :class="{ 'mobile-mode': isMobile }">
    <!-- 固定区域：操作按钮 -->
    <div v-if="$slots.actions" class="layout-section-fixed">
      <slot name="actions" />
    </div>

    <!-- 固定区域：搜索和过滤器 -->
    <div v-if="$slots.filters" class="layout-section-fixed">
      <slot name="filters" />
    </div>

    <!-- 滚动区域：表格 -->
    <div class="layout-section-scrollable">
      <div class="card table-scroll-container">
        <slot name="table" />
      </div>
    </div>

    <!-- 固定区域：分页器 -->
    <div v-if="$slots.pagination" class="layout-section-fixed">
      <slot name="pagination" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

const isMobile = ref(false)

const checkMobile = () => {
  isMobile.value = window.innerWidth < 1024
}

onMounted(() => {
  checkMobile()
  window.addEventListener('resize', checkMobile)
})

onUnmounted(() => {
  window.removeEventListener('resize', checkMobile)
})
</script>

<style scoped>
/* 桌面端：Flexbox 布局 */
.table-page-layout {
  @apply flex flex-col gap-5;
  height: calc(100vh - 64px - 4rem); /* 减去 header + lg:p-8 的上下padding */
}

.layout-section-fixed {
  @apply min-w-0 flex-shrink-0;
}

.layout-section-scrollable {
  @apply flex min-h-0 min-w-0 flex-1 flex-col;
}

/* 表格滚动容器 - 增强版表体滚动方案 */
.table-scroll-container {
  @apply flex h-full flex-col overflow-hidden rounded-3xl border border-gray-100 bg-white dark:border-dark-700 dark:bg-dark-800;
  box-shadow: 0 1px 2px rgb(15 23 42 / 0.04);
  transition:
    border-color 0.18s ease-out,
    box-shadow 0.18s ease-out;
}

.table-scroll-container :deep(.table-wrapper) {
  @apply flex-1 overflow-x-auto overflow-y-auto;
  /* 确保横向滚动条显示在最底部 */
  scrollbar-gutter: stable;
}

.table-scroll-container :deep(table) {
  @apply w-full;
  min-width: max-content; /* 关键：确保表格宽度根据内容撑开，从而触发横向滚动 */
  display: table; /* 使用标准 table 布局以支持 sticky 列 */
}

.table-scroll-container :deep(thead) {
  @apply bg-gray-50/90 dark:bg-dark-900/60;
}

.table-scroll-container :deep(tbody) {
  /* 保持默认 table-row-group 显示，不使用 block */
}

.table-scroll-container :deep(th) {
  @apply border-b border-gray-200 px-5 py-3.5 text-left text-xs font-semibold text-gray-600 dark:border-dark-700 dark:text-dark-300;
}

.table-scroll-container :deep(td) {
  @apply border-b border-gray-100 px-5 py-3.5 text-sm text-gray-700 dark:border-dark-700/60 dark:text-gray-300;
}

.table-scroll-container :deep(tbody tr:last-child td) {
  @apply border-b-0;
}

/* 移动端：恢复正常滚动 */
.table-page-layout.mobile-mode {
  @apply gap-4;
  height: auto;
}

.table-page-layout.mobile-mode .table-scroll-container {
  @apply h-auto;
}

.table-page-layout.mobile-mode .layout-section-scrollable {
  @apply flex-none min-h-fit;
}

.table-page-layout.mobile-mode .table-scroll-container :deep(table) {
  @apply flex-none;
  display: table;
  min-width: 100%;
}

@media (prefers-reduced-motion: reduce) {
  .table-scroll-container {
    transition-duration: 0.01ms;
  }
}
</style>

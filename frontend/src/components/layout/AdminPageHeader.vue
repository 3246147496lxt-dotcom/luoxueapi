<template>
  <header class="admin-page-header">
    <div class="admin-page-header__copy">
      <div class="admin-page-header__title-row">
        <h1 class="admin-page-header__title">
          <slot name="title">{{ title }}</slot>
        </h1>

        <div v-if="$slots.meta" class="admin-page-header__meta">
          <slot name="meta" />
        </div>
      </div>

      <p
        v-if="description || $slots.description"
        class="admin-page-header__description"
      >
        <slot name="description">{{ description }}</slot>
      </p>
    </div>

    <div
      v-if="$slots['secondary-actions'] || $slots['primary-actions']"
      class="admin-page-header__actions"
    >
      <div v-if="$slots['secondary-actions']" class="admin-page-header__action-group">
        <slot name="secondary-actions" />
      </div>
      <div v-if="$slots['primary-actions']" class="admin-page-header__action-group">
        <slot name="primary-actions" />
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
defineProps<{
  title: string
  description?: string
}>()

defineSlots<{
  title?: () => unknown
  description?: () => unknown
  meta?: () => unknown
  'secondary-actions'?: () => unknown
  'primary-actions'?: () => unknown
}>()
</script>

<style scoped>
.admin-page-header {
  @apply flex min-w-0 flex-col gap-4 py-0.5 sm:flex-row sm:items-start sm:justify-between;
}

.admin-page-header__copy {
  @apply min-w-0 flex-1;
}

.admin-page-header__title-row {
  @apply flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1.5;
}

.admin-page-header__title {
  @apply min-w-0 text-xl font-semibold leading-7 tracking-tight text-gray-950 sm:text-2xl sm:leading-8 dark:text-white;
  text-wrap: balance;
}

.admin-page-header__meta {
  @apply flex flex-wrap items-center gap-2 text-xs font-medium text-gray-500 dark:text-dark-300;
}

.admin-page-header__description {
  @apply mt-1 max-w-3xl text-sm leading-5 text-gray-600 dark:text-dark-300;
  text-wrap: pretty;
}

.admin-page-header__actions,
.admin-page-header__action-group {
  @apply flex flex-wrap items-center gap-2;
}

.admin-page-header__actions {
  @apply w-full sm:w-auto sm:flex-shrink-0 sm:justify-end;
}
</style>

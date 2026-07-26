<template>
  <div class="account-table-actions">
    <slot name="before"></slot>

    <button
      type="button"
      class="account-table-action-button"
      :aria-label="t('common.refresh')"
      :title="t('common.refresh')"
      :disabled="loading"
      @click="$emit('refresh')"
    >
      <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
    </button>

    <slot name="after"></slot>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

defineProps<{ loading?: boolean }>()
defineEmits<{
  (event: 'refresh'): void
}>()

const { t } = useI18n()
</script>

<style scoped>
.account-table-actions {
  display: flex;
  width: auto;
  flex: none;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}

.account-table-action-button {
  display: inline-flex;
  width: 44px;
  height: 44px;
  flex: none;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--lx-clay-border);
  border-radius: 12px;
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-surface);
  box-shadow: 0 5px 13px rgb(70 55 96 / 0.06);
  transition:
    color 150ms ease-out,
    background-color 150ms ease-out,
    border-color 150ms ease-out;
}

.account-table-action-button:hover:not(:disabled) {
  color: var(--lx-clay-text);
  border-color: var(--lx-clay-border-strong);
  background: var(--lx-clay-surface-soft);
}

.account-table-action-button:disabled {
  cursor: wait;
  opacity: 0.6;
}

@media (max-width: 639px) {
  .account-table-actions {
    width: 100%;
  }
}
</style>

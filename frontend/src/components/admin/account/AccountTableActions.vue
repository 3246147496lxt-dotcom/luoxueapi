<template>
  <div
    class="grid w-full grid-cols-[minmax(0,1fr)_auto_auto] items-center gap-3 lg:flex lg:w-auto lg:flex-wrap"
  >
    <slot name="before"></slot>

    <button
      type="button"
      class="btn btn-secondary order-2 min-h-11 min-w-11 justify-center px-3 lg:order-none"
      :aria-label="t('common.refresh')"
      :title="t('common.refresh')"
      :disabled="loading"
      @click="$emit('refresh')"
    >
      <Icon name="refresh" size="md" :class="[loading ? 'animate-spin' : '']" />
    </button>

    <div
      id="account-secondary-actions"
      data-testid="account-secondary-actions"
      :class="[
        'order-4 col-span-3 min-w-0 flex-wrap items-center gap-3',
        secondaryActionsOpen ? 'flex' : 'hidden',
        'lg:contents'
      ]"
    >
      <slot name="after"></slot>
    </div>

    <slot name="beforeCreate"></slot>

    <button
      type="button"
      class="btn btn-primary order-1 min-h-11 w-full justify-center lg:order-none lg:w-auto"
      @click="$emit('create')"
    >
      {{ t('admin.accounts.createAccount') }}
    </button>

    <slot name="afterCreate"></slot>

    <button
      type="button"
      class="btn btn-secondary order-3 min-h-11 min-w-11 justify-center px-3 lg:hidden"
      data-testid="account-actions-toggle"
      :aria-expanded="secondaryActionsOpen"
      aria-controls="account-secondary-actions"
      @click="secondaryActionsOpen = !secondaryActionsOpen"
    >
      <Icon name="more" size="sm" />
      <span>{{ t('common.more') }}</span>
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

defineProps<{ loading?: boolean }>()
defineEmits<{
  (event: 'refresh'): void
  (event: 'create'): void
}>()

const { t } = useI18n()
const secondaryActionsOpen = ref(false)
</script>

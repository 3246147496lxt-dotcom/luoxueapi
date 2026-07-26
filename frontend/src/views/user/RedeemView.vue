<template>
  <AppLayout>
    <div class="mx-auto max-w-4xl">
      <h1 class="sr-only">{{ t('redeem.title') }}</h1>
      <RedeemCodePanel />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import RedeemCodePanel from '@/components/payment/RedeemCodePanel.vue'

const { t } = useI18n()
</script>

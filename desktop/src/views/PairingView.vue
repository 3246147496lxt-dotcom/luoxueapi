<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { AlertTriangle, ArrowRight, Coins, ExternalLink, LoaderCircle, ShieldCheck } from 'lucide-vue-next'
import type { PairingState } from '@/types'

defineProps<{ pairing: PairingState; busy: boolean; error?: string }>()
defineEmits<{ start: []; poll: []; open: [] }>()
const { t } = useI18n()
</script>

<template>
  <main class="pairing-shell">
    <section class="pairing-panel">
      <div class="pairing-brand">
        <span class="brand-mark brand-mark--large"><img class="brand-mark__image" src="/logo.png" alt="" /></span>
        <strong>{{ t('pairing.brand') }}</strong>
      </div>

      <div class="pairing-copy">
        <h1>{{ t('pairing.title') }}</h1>
        <p>{{ t('pairing.body') }}</p>
      </div>

      <div v-if="error" class="inline-alert" role="alert">
        <AlertTriangle :size="18" />
        <span>{{ error }}</span>
      </div>

      <template v-if="pairing.status === 'idle' || pairing.status === 'error' || pairing.status === 'expired'">
        <button class="primary-button primary-button--wide" type="button" :disabled="busy" @click="$emit('start')">
          <LoaderCircle v-if="busy" class="spin" :size="18" />
          <Coins v-else :size="18" />
          <span>{{ t('pairing.start') }}</span>
          <ArrowRight :size="18" />
        </button>
      </template>

      <template v-else>
        <div class="pairing-code-block">
          <span>{{ t('pairing.code') }}</span>
          <strong>{{ pairing.userCode }}</strong>
        </div>
        <div class="pairing-waiting" role="status">
          <LoaderCircle class="spin" :size="17" />
          <span>{{ t('pairing.waiting') }}</span>
        </div>
        <div class="button-row">
          <button class="secondary-button" type="button" @click="$emit('open')">
            <ExternalLink :size="17" />{{ t('pairing.openAgain') }}
          </button>
          <button class="primary-button" type="button" :disabled="busy" @click="$emit('poll')">
            {{ t('common.retry') }}
          </button>
        </div>
      </template>

      <footer class="pairing-footer">
        <ShieldCheck :size="16" />
        <span>{{ t('pairing.limit') }}</span>
      </footer>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { AlertTriangle, Check, ChevronDown, Clipboard, ExternalLink, Power, RefreshCw, RotateCcw, TerminalSquare } from 'lucide-vue-next'
import { desktopApi } from '@/api'
import MetricStrip from '@/components/MetricStrip.vue'
import RequestTable from '@/components/RequestTable.vue'
import type { DesktopSnapshot, RequestMetadata } from '@/types'

const props = defineProps<{ snapshot: DesktopSnapshot; requests: readonly RequestMetadata[]; busy: string }>()
const emit = defineEmits<{
  restart: []
  selectRoute: [groupId: number, model: string]
  loadRequests: []
  openCodex: []
  copyCommand: []
}>()
const { t } = useI18n()
const routeOpen = ref(false)

const selectedRoute = computed(() => props.snapshot.routes.find((route) => route.groupId === props.snapshot.selectedGroupId))

async function choose(groupId: number, model: string) {
  routeOpen.value = false
  emit('selectRoute', groupId, model)
}

onMounted(() => emit('loadRequests'))
</script>

<template>
  <div class="view overview-view">
    <MetricStrip :today="snapshot.today" />

    <div v-if="snapshot.routes.length === 0" class="availability-banner" role="status">
      <AlertTriangle :size="18" />
      <div><strong>{{ t('overview.noRoutes') }}</strong><span>{{ t('overview.noRoutesHint') }}</span></div>
      <button class="secondary-button" type="button" @click="desktopApi.openAccountPage('routes')">
        {{ t('overview.manageRoutes') }}<ExternalLink :size="15" />
      </button>
    </div>
    <div v-else-if="snapshot.today.balance <= 0" class="availability-banner" role="status">
      <AlertTriangle :size="18" />
      <div><strong>{{ t('overview.noBalance') }}</strong><span>{{ t('overview.noBalanceHint') }}</span></div>
      <button class="secondary-button" type="button" @click="desktopApi.openAccountPage('balance')">
        {{ t('overview.recharge') }}<ExternalLink :size="15" />
      </button>
    </div>

    <div class="overview-control-grid">
      <section class="route-section panel-section">
        <div class="section-heading">
          <div>
            <h2>{{ t('overview.route') }}</h2>
            <p>{{ t('overview.restartHint') }}</p>
          </div>
          <button class="secondary-button" type="button" @click="$emit('copyCommand')">
            <Clipboard :size="16" />{{ t('common.copyCommand') }}
          </button>
        </div>

        <div class="route-controls">
          <div class="route-select-wrap">
            <button class="route-select" type="button" :aria-expanded="routeOpen" @click="routeOpen = !routeOpen">
              <span>
                <small>{{ t('overview.route') }}</small>
                <strong>{{ selectedRoute?.name || '—' }}</strong>
              </span>
              <span class="route-rate">{{ selectedRoute ? `${selectedRoute.rateMultiplier}x` : '—' }}</span>
              <ChevronDown :size="17" />
            </button>
            <div v-if="routeOpen" class="route-menu">
              <button
                v-for="route in snapshot.routes"
                :key="route.groupId"
                type="button"
                class="route-menu__item"
                @click="choose(route.groupId, route.models[0] || '')"
              >
                <span>
                  <strong>{{ route.name }}</strong>
                  <small>{{ route.models.length }} models · {{ route.rateMultiplier }}x</small>
                </span>
                <Check v-if="route.groupId === snapshot.selectedGroupId" :size="17" />
              </button>
            </div>
          </div>

          <label class="field-group">
            <span>{{ t('overview.model') }}</span>
            <select
              :value="snapshot.selectedModel"
              :disabled="busy === 'route' || !selectedRoute"
              @change="choose(snapshot.selectedGroupId || 0, ($event.target as HTMLSelectElement).value)"
            >
              <option v-for="model in selectedRoute?.models" :key="model" :value="model">{{ model }}</option>
            </select>
          </label>

          <div class="session-note">
            <RotateCcw :size="17" />
            <span>{{ t('overview.sessionPinned') }}</span>
          </div>
        </div>
      </section>

      <section class="gateway-hero gateway-hero--compact" :class="{ 'gateway-hero--offline': snapshot.gatewayStatus !== 'running' }">
        <div class="gateway-hero__status">
          <div class="gateway-hero__icon"><Power :size="22" /></div>
          <div>
            <div class="section-title-row">
              <h2>{{ t('overview.title') }}</h2>
              <span class="status-label" :class="snapshot.gatewayStatus === 'running' ? 'status-label--success' : 'status-label--muted'">
                {{ snapshot.gatewayStatus === 'running' ? t('common.running') : t('common.stopped') }}
              </span>
            </div>
            <p>
              {{ snapshot.takeoverEnabled ? t('overview.takeoverOn') : t('overview.takeoverOff') }}
              <code v-if="snapshot.gatewayPort">127.0.0.1:{{ snapshot.gatewayPort }}</code>
            </p>
          </div>
        </div>
        <div class="hero-actions">
          <button class="icon-button" type="button" :disabled="busy === 'gateway'" :title="t('common.restart')" @click="$emit('restart')">
            <RefreshCw :class="{ spin: busy === 'gateway' }" :size="17" />
          </button>
          <button class="primary-button" type="button" @click="$emit('openCodex')">
            <TerminalSquare :size="17" />{{ t('common.openCodex') }}
          </button>
        </div>
      </section>
    </div>

    <section class="recent-section">
      <div class="section-heading">
        <h2>{{ t('overview.recent') }}</h2>
        <span class="privacy-note">{{ t('requests.privacy') }}</span>
      </div>
      <RequestTable :requests="requests.slice(0, 5)" compact />
    </section>
  </div>
</template>

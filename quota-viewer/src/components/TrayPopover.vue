<script setup lang="ts">
import { computed } from 'vue'
import {
  ArrowUpRight,
  CalendarDays,
  CircleAlert,
  Crown,
  LoaderCircle,
  RefreshCw,
  SunMedium,
  WalletCards
} from 'lucide-vue-next'
import MembershipCard from '@/components/MembershipCard.vue'
import type { ViewerUiStatus } from '@/composables/useQuotaViewer'
import type { QuotaOverview, ViewerDataStatus } from '@/types'

const props = defineProps<{
  overview: QuotaOverview | null
  status: ViewerUiStatus
  selectedQuotaId?: string | null
  refreshing?: boolean
  dataStatus?: ViewerDataStatus
}>()

const emit = defineEmits<{
  refresh: []
  openMain: []
  selectQuota: [quotaId: string]
}>()

const primaryQuota = computed(() => props.overview?.quotas[0] ?? null)

const emptyState = computed(() => {
  switch (props.status) {
    case 'loading':
    case 'connecting':
      return {
        title: '正在读取积分',
        copy: '连接账户并同步最新数据，请稍候。',
        loading: true
      }
    case 'auth-invalid':
      return {
        title: '账户连接已失效',
        copy: '请在完整面板中重新连接账户。',
        loading: false
      }
    case 'pairing-expired':
      return {
        title: '本次连接已过期',
        copy: '请打开完整面板重新发起连接。',
        loading: false
      }
    case 'unavailable':
      return {
        title: '暂时无法获取积分',
        copy: '可以重试，或打开完整面板查看原因。',
        loading: false
      }
    default:
      return {
        title: '尚未连接账户',
        copy: '打开完整面板，完成一次安全连接。',
        loading: false
      }
  }
})

const formatPoints = (amount: string | null | undefined) => {
  if (amount == null) return '—'
  const parsed = Number(amount)
  return Number.isFinite(parsed) ? `${parsed.toFixed(2)} 积分` : '—'
}
</script>

<template>
  <section
    class="tray-popover-stage"
    :class="{ 'tray-popover-stage--empty': !overview }"
    aria-label="落雪积分菜单栏速览"
  >
    <template v-if="overview">
      <section class="tray-balance" aria-labelledby="tray-balance-title">
        <div class="tray-section-heading">
          <span id="tray-balance-title">
            <WalletCards :size="14" aria-hidden="true" />
            账户余额
          </span>
          <div class="tray-heading-actions">
            <small
              class="tray-wallet-state"
              :class="[
                `tray-wallet-state--${overview.walletState}`,
                { 'tray-wallet-state--stale': dataStatus === 'stale' }
              ]"
            >
              <i aria-hidden="true" />
              {{
                dataStatus === 'stale'
                  ? '上次数据'
                  : overview.walletState === 'available'
                    ? '可用'
                    : overview.walletState === 'exhausted'
                      ? '余额不足'
                      : '待确认'
              }}
            </small>
            <span class="tray-updated-at">
              {{ refreshing ? '更新中' : overview.updatedAt }}
            </span>
            <button
              type="button"
              class="tray-icon-button"
              :class="{ 'tray-icon-button--spinning': refreshing }"
              :disabled="refreshing || status === 'connecting'"
              aria-label="刷新积分"
              title="刷新积分"
              @click="emit('refresh')"
            >
              <RefreshCw :size="14" :stroke-width="2.1" />
            </button>
          </div>
        </div>

        <strong class="tray-balance-value">
          {{ formatPoints(overview.balance) }}
        </strong>

        <div class="tray-spend-row">
          <span>
            <small><SunMedium :size="13" aria-hidden="true" />今日消费</small>
            <strong>{{ formatPoints(overview.todaySpend) }}</strong>
          </span>
          <span>
            <small><CalendarDays :size="13" aria-hidden="true" />本月消费</small>
            <strong>{{ formatPoints(overview.monthSpend) }}</strong>
          </span>
        </div>
      </section>

      <MembershipCard
        v-if="primaryQuota"
        :quota="primaryQuota"
        :selected="selectedQuotaId === primaryQuota.id"
        :stale="dataStatus === 'stale'"
        @select="emit('selectQuota', $event)"
      />

      <section
        v-else
        class="panel-card membership-empty tray-membership-empty"
        aria-label="暂无会员订阅"
      >
        <span class="membership-icon" aria-hidden="true">
          <Crown :size="17" :stroke-width="2.1" />
        </span>
        <strong>暂无会员订阅</strong>
        <p>账户余额仍可用于余额计费。</p>
      </section>
    </template>

    <section v-else class="tray-empty-state" aria-live="polite">
      <button
        type="button"
        class="tray-icon-button tray-icon-button--empty"
        :class="{ 'tray-icon-button--spinning': refreshing }"
        :disabled="refreshing || status === 'connecting'"
        aria-label="刷新积分"
        title="刷新积分"
        @click="emit('refresh')"
      >
        <RefreshCw :size="14" :stroke-width="2.1" />
      </button>
      <span class="tray-empty-state__icon" aria-hidden="true">
        <LoaderCircle
          v-if="emptyState.loading"
          class="tray-empty-state__loader"
          :size="24"
        />
        <CircleAlert v-else :size="24" />
      </span>
      <strong>{{ emptyState.title }}</strong>
      <p>{{ emptyState.copy }}</p>
    </section>

    <button
      type="button"
      class="tray-open-main"
      @click="emit('openMain')"
    >
      <span>打开完整面板</span>
      <ArrowUpRight :size="16" :stroke-width="2.1" aria-hidden="true" />
    </button>
  </section>
</template>

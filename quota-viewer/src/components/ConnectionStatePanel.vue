<script setup lang="ts">
import {
  ExternalLink,
  Link2,
  LoaderCircle,
  RefreshCw,
  ShieldCheck,
  Unplug,
  X
} from 'lucide-vue-next'
import { computed } from 'vue'
import type { ViewerUiStatus } from '@/composables/useQuotaViewer'
import {
  startQuotaViewerDrag,
  type PairingState
} from '@/lib/desktop'

const props = defineProps<{
  status: ViewerUiStatus
  pairing: PairingState | null
  errorCode: string | null
}>()

const emit = defineEmits<{
  connect: []
  retry: []
  reopen: []
  cancel: []
  close: []
}>()

const stateCopy: Record<
  Exclude<ViewerUiStatus, 'ready' | 'stale'>,
  { title: string; description: string }
> = {
  disconnected: {
    title: '连接落雪账户',
    description: '通过系统浏览器完成只读授权。查看器不会读取密码，也不能消费额度。'
  },
  connecting: {
    title: '等待浏览器确认',
    description: '完成只读授权后，这里会自动加载会员周额度。'
  },
  loading: {
    title: '正在获取额度',
    description: '正在读取会员套餐与本周剩余额度。'
  },
  unavailable: {
    title: '暂时无法获取额度',
    description: '未取得可安全展示的数据，请稍后重新获取。'
  },
  'auth-invalid': {
    title: '需要重新连接',
    description: '只读授权已失效。重新连接后才能获取最新额度。'
  },
  'pairing-expired': {
    title: '授权已超时',
    description: '本次只读授权没有在有效期内完成，请重新发起连接。'
  }
}

const visibleStatus = computed(() =>
  props.status === 'ready' || props.status === 'stale'
    ? 'unavailable'
    : props.status
)
</script>

<template>
  <section
    class="monitor-panel monitor-panel--main connection-panel window-drag-region"
    aria-label="落雪额度"
    @mousedown="startQuotaViewerDrag"
  >
    <header class="panel-header">
      <nav class="header-actions" aria-label="面板操作">
        <button
          type="button"
          class="icon-button icon-button--hide-panel"
          aria-label="隐藏额度面板"
          title="隐藏"
          @click="emit('close')"
        >
          <X :size="17" :stroke-width="2" />
        </button>
      </nav>
    </header>

    <section class="panel-card connection-card" aria-live="polite">
      <span
        class="connection-card__icon"
        :class="{ 'connection-card__icon--loading': status === 'connecting' || status === 'loading' }"
        aria-hidden="true"
      >
        <LoaderCircle v-if="status === 'connecting' || status === 'loading'" :size="28" />
        <Unplug v-else-if="status === 'auth-invalid'" :size="27" />
        <RefreshCw v-else-if="status === 'unavailable' || status === 'pairing-expired'" :size="27" />
        <ShieldCheck v-else :size="28" />
      </span>

      <h2>{{ stateCopy[visibleStatus].title }}</h2>
      <p>{{ stateCopy[visibleStatus].description }}</p>

      <div v-if="status === 'connecting' && pairing?.user_code" class="pairing-code">
        <small>浏览器确认码</small>
        <strong>{{ pairing.user_code }}</strong>
      </div>

      <small v-if="errorCode && status === 'unavailable'" class="connection-error">
        {{ errorCode }}
      </small>

      <div class="connection-actions">
        <button
          v-if="status === 'disconnected' || status === 'auth-invalid' || status === 'pairing-expired'"
          type="button"
          class="primary-action"
          @click="emit('connect')"
        >
          <Link2 :size="15" />
          连接账户
        </button>
        <button
          v-else-if="status === 'unavailable'"
          type="button"
          class="primary-action"
          @click="emit('retry')"
        >
          <RefreshCw :size="15" />
          重新获取
        </button>
        <template v-else-if="status === 'connecting'">
          <button type="button" class="primary-action" @click="emit('reopen')">
            <ExternalLink :size="15" />
            重新打开授权页
          </button>
          <button type="button" class="secondary-action" @click="emit('cancel')">
            取消
          </button>
        </template>
      </div>
    </section>

    <footer class="connection-assurance">
      <ShieldCheck :size="13" aria-hidden="true" />
      <span>仅授予额度读取权限，无法发起 API 请求或修改账户。</span>
    </footer>
  </section>
</template>

<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="hasHomeContent" class="min-h-screen">
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      :title="`${siteName} ${t('home.nav.ariaLabel')}`"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <!-- SECURITY: homeContent is an administrator-only setting. -->
    <div v-else v-html="homeContent"></div>
  </div>

  <PublicSiteLayout v-else class="home-page" page="home" :show-model-catalog="true">
    <main id="top" class="hero-only-main">
      <section
        id="steps"
        class="content-shell hero-section reference-hero-shell glass-gradient"
        aria-labelledby="home-hero-title"
      >
        <div class="hero-top-region">
          <div class="status-pill">
            <span class="status-dot" aria-hidden="true"></span>
            {{ t('home.hero.status') }}
          </div>

          <h1 id="home-hero-title" class="hero-title">
            <span class="hero-title-lead">{{ heroTitleParts.lead }}</span><br v-if="heroTitleParts.accent">
            <span v-if="heroTitleParts.accent" class="hero-title-accent">{{ heroTitleParts.accent }}</span>
          </h1>

          <div class="hero-actions">
            <router-link
              data-testid="hero-primary-cta"
              :to="primaryCta.to"
              class="primary-action hero-primary-action"
            >
              {{ primaryCta.label }}
              <Icon name="key" size="sm" :stroke-width="2" aria-hidden="true" />
            </router-link>
          </div>

          <div class="api-control-bar" aria-label="API 接入地址">
            <div class="api-protocol">
              <span>OpenAI 协议</span>
              <Icon name="chevronDown" size="xs" :stroke-width="2" aria-hidden="true" />
            </div>
            <span class="api-url">{{ apiV1Base }}</span>
            <button
              type="button"
              class="api-copy-btn"
              data-testid="copy-api-url-button"
              :aria-label="apiCopyAriaLabel"
              @click="copyApiBase"
            >
              <Icon
                :name="apiCopyStatus === 'copied' ? 'check' : 'lucideCopy'"
                size="sm"
                :stroke-width="2"
                aria-hidden="true"
              />
            </button>
          </div>
          <p class="sr-only" aria-live="polite">{{ apiCopyLiveMessage }}</p>
        </div>

        <div class="hero-stage-region" aria-label="AI 模型连接示意">
          <svg
            class="bezier-svg"
            viewBox="0 0 1320 340"
            preserveAspectRatio="none"
            fill="none"
            aria-hidden="true"
            focusable="false"
          >
            <defs>
              <filter id="route-soft-blur" x="-20%" y="-50%" width="140%" height="200%">
                <feGaussianBlur stdDeviation="3.2" />
              </filter>
            </defs>

            <g class="route-base-layer">
              <path class="route-base route-claude" pathLength="1" d="M137 80 C430 84 390 170 617 170" />
              <path class="route-base route-gpt" pathLength="1" d="M127 170 L617 170" />
              <path class="route-base route-gemini" pathLength="1" d="M137 260 C450 260 410 170 617 170" />
              <path class="route-base route-claude right" pathLength="1" d="M1161 63 C870 63 910 170 703 170" />
              <path class="route-base route-codex right" pathLength="1" d="M1185 134 C850 134 890 170 703 170" />
              <path class="route-base route-ccswitch right" pathLength="1" d="M1170 206 C850 206 890 170 703 170" />
              <path class="route-base route-openclaw right" pathLength="1" d="M1170 277 C870 277 910 170 703 170" />
            </g>

            <g class="route-bloom-layer" aria-hidden="true">
              <path class="route-bloom route-claude" pathLength="1" d="M137 80 C430 84 390 170 617 170" />
              <path class="route-bloom route-gpt" pathLength="1" d="M127 170 L617 170" />
              <path class="route-bloom route-gemini" pathLength="1" d="M137 260 C450 260 410 170 617 170" />
              <path class="route-bloom route-claude right" pathLength="1" d="M1161 63 C870 63 910 170 703 170" />
              <path class="route-bloom route-codex right" pathLength="1" d="M1185 134 C850 134 890 170 703 170" />
              <path class="route-bloom route-ccswitch right" pathLength="1" d="M1170 206 C850 206 890 170 703 170" />
              <path class="route-bloom route-openclaw right" pathLength="1" d="M1170 277 C870 277 910 170 703 170" />
            </g>

            <g class="route-flow-layer" aria-hidden="true">
              <path class="route-flow route-claude" pathLength="1" d="M137 80 C430 84 390 170 617 170" />
              <path class="route-flow route-gpt" pathLength="1" d="M127 170 L617 170" />
              <path class="route-flow route-gemini" pathLength="1" d="M137 260 C450 260 410 170 617 170" />
              <path class="route-flow route-claude right" pathLength="1" d="M1161 63 C870 63 910 170 703 170" />
              <path class="route-flow route-codex right" pathLength="1" d="M1185 134 C850 134 890 170 703 170" />
              <path class="route-flow route-ccswitch right" pathLength="1" d="M1170 206 C850 206 890 170 703 170" />
              <path class="route-flow route-openclaw right" pathLength="1" d="M1170 277 C870 277 910 170 703 170" />
            </g>

            <g class="route-particles" aria-hidden="true">
              <circle class="route-particle route-claude" r="3">
                <animateMotion dur="4.8s" begin="1.95s" repeatCount="indefinite" path="M137 80 C430 84 390 170 617 170" />
              </circle>
              <circle class="route-particle route-gpt" r="2.6">
                <animateMotion dur="4.8s" begin="2.05s" repeatCount="indefinite" path="M127 170 L617 170" />
              </circle>
              <circle class="route-particle route-gemini" r="3">
                <animateMotion dur="4.8s" begin="2.15s" repeatCount="indefinite" path="M137 260 C450 260 410 170 617 170" />
              </circle>
              <circle class="route-particle route-claude right" r="3">
                <animateMotion dur="4.8s" begin="2.25s" repeatCount="indefinite" path="M703 170 C910 170 870 63 1161 63" />
              </circle>
              <circle class="route-particle route-codex right" r="2.8">
                <animateMotion dur="4.8s" begin="2.35s" repeatCount="indefinite" path="M703 170 C890 170 850 134 1185 134" />
              </circle>
              <circle class="route-particle route-ccswitch right" r="2.8">
                <animateMotion dur="4.8s" begin="2.45s" repeatCount="indefinite" path="M703 170 C890 170 850 206 1170 206" />
              </circle>
              <circle class="route-particle route-openclaw right" r="3">
                <animateMotion dur="4.8s" begin="2.55s" repeatCount="indefinite" path="M703 170 C910 170 870 277 1170 277" />
              </circle>
            </g>
          </svg>

          <div class="stage-route-label stage-route-label-left">全球模型直连</div>
          <div class="stage-route-label stage-route-label-right">AGENT 客户端</div>

          <div class="stage-center-node" aria-hidden="true">
            <span class="stage-center-logo"></span>
          </div>
          <div class="stage-center-caption">
            <span class="caption-dot" aria-hidden="true"></span>
            智能调度
          </div>

          <div
            v-for="model in modelNodes"
            :key="model.id"
            class="node-card node-left"
            :class="`node-${model.id}`"
            :style="{ top: model.top }"
          >
            <PlatformIcon :platform="model.platform" size="lg" aria-hidden="true" />
            <span>{{ model.label }}</span>
          </div>

          <div
            v-for="client in clientNodes"
            :key="client.id"
            class="node-card node-right"
            :class="[`node-${client.id}`, client.tone]"
            :style="{ top: client.top }"
          >
            <Icon :name="client.icon" size="sm" :stroke-width="2" aria-hidden="true" />
            <span>{{ client.label }}</span>
          </div>
        </div>

        <!-- Background layers copied from the supplied reference code. Keep values and stacking intact. -->
        <div class="reference-background-wash" aria-hidden="true"></div>
        <div class="blue-mesh" aria-hidden="true"></div>
        <div class="pixel-grid" aria-hidden="true"></div>
      </section>
    </main>
  </PublicSiteLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import PublicSiteLayout from '@/components/public/PublicSiteLayout.vue'
import { useClipboard } from '@/composables/useClipboard'
import type { GroupPlatform } from '@/types'

type CopyStatus = 'idle' | 'copied' | 'failed'
type ClientIcon = 'terminal' | 'code2' | 'zap' | 'box'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || '落雪API')
const apiBaseUrl = computed(() => appStore.cachedPublicSettings?.api_base_url || appStore.apiBaseUrl || '')
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const hasHomeContent = computed(() => homeContent.value.trim().length > 0)
const isHomeContentUrl = computed(() => /^https?:\/\//i.test(homeContent.value.trim()))
const heroTitleParts = computed(() => {
  const title = t('home.hero.title')
  const [, lead = title, accent = ''] = title.match(/^(.*?)[，,]\s*(.+)$/) || []
  return { lead, accent }
})

function normalizeApiV1Base(value: string): string {
  const fallback = typeof window === 'undefined' ? 'https://luoxueapi.cc' : window.location.origin
  const base = value.trim().replace(/\/+$/, '') || fallback
  return /\/v1$/i.test(base) ? base : `${base}/v1`
}

const apiV1Base = computed(() => normalizeApiV1Base(apiBaseUrl.value))

const modelNodes: Array<{
  id: string
  label: string
  platform: GroupPlatform
  top: string
}> = [
  { id: 'claude', label: 'Claude', platform: 'anthropic', top: '23.5%' },
  { id: 'gpt', label: 'GPT', platform: 'openai', top: '50%' },
  { id: 'gemini', label: 'Gemini', platform: 'gemini', top: '76.5%' }
]

const clientNodes: Array<{
  id: string
  label: string
  icon: ClientIcon
  tone: string
  top: string
}> = [
  { id: 'claude-code', label: 'Claude Code', icon: 'terminal', tone: 'node-tone-orange', top: '18.5%' },
  { id: 'codex', label: 'Codex', icon: 'code2', tone: 'node-tone-indigo', top: '39.5%' },
  { id: 'ccswitch', label: 'CC Switch', icon: 'zap', tone: 'node-tone-blue', top: '60.5%' },
  { id: 'openclaw', label: 'OpenClaw', icon: 'box', tone: 'node-tone-red', top: '81.5%' }
]

const isAuthenticated = computed(() => authStore.isAuthenticated)
const registrationEnabled = computed(
  () => appStore.cachedPublicSettings?.registration_enabled === true
)
const primaryCta = computed(() => {
  if (isAuthenticated.value) {
    return { to: '/keys', label: t('home.hero.createKey') }
  }
  if (registrationEnabled.value) {
    return { to: '/register', label: t('home.hero.createKey') }
  }
  return { to: '/login', label: t('home.hero.createKey') }
})
const apiCopyStatus = ref<CopyStatus>('idle')
const apiCopyAriaLabel = computed(() => (
  apiCopyStatus.value === 'copied'
    ? 'API 地址已复制'
    : apiCopyStatus.value === 'failed'
      ? 'API 地址复制失败'
      : '复制 API 地址'
))
const apiCopyLiveMessage = computed(() => (
  apiCopyStatus.value === 'idle' ? '' : apiCopyAriaLabel.value
))
let apiCopyRequestId = 0
let apiCopyResetTimer: ReturnType<typeof setTimeout> | undefined

async function copyApiBase() {
  if (apiCopyResetTimer) clearTimeout(apiCopyResetTimer)
  const requestId = ++apiCopyRequestId
  const didCopy = await copyToClipboard(apiV1Base.value, 'API 地址已复制')
  if (requestId !== apiCopyRequestId) return

  apiCopyStatus.value = didCopy ? 'copied' : 'failed'
  apiCopyResetTimer = setTimeout(() => {
    apiCopyStatus.value = 'idle'
  }, 2000)
}

onBeforeUnmount(() => {
  apiCopyRequestId += 1
  if (apiCopyResetTimer) clearTimeout(apiCopyResetTimer)
})
</script>

<style scoped src="./HomeView.clay.css"></style>

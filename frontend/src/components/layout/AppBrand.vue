<template>
  <router-link
    :to="homePath"
    :data-testid="`${placement}-brand`"
    class="app-brand flex h-10 w-full min-w-0 items-center gap-2 rounded-xl focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 md:h-16"
    :class="[
      `app-brand--${placement}`,
      {
        'app-brand--collapsed': collapsed,
        'app-brand--wordmark': wordmark && !collapsed,
        'app-brand--wordmark-only': wordmark && wordmarkOnly && !collapsed,
      }
    ]"
    :aria-label="siteName"
  >
    <span
      v-if="logoVisible"
      :data-testid="`${placement}-brand-logo`"
      class="app-brand-logo-frame flex h-10 w-10 flex-shrink-0 items-center justify-center md:h-12 md:w-12"
    >
      <img
        :src="brandLogoSrc"
        alt=""
        class="app-brand-logo-image block h-full w-full max-w-none object-contain"
        :class="{ 'app-brand-logo-image-default': isDefaultLogo }"
      >
    </span>
    <span
      v-if="wordmark && !collapsed"
      :data-testid="`${placement}-brand-wordmark`"
      class="app-brand-wordmark"
      :title="siteName"
    >
      {{ siteName }}
    </span>
  </router-link>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useAppStore } from '@/stores/app'
import { sanitizeUrl } from '@/utils/url'

const props = withDefaults(defineProps<{
  placement: 'header' | 'sidebar'
  collapsed?: boolean
  homePath?: string
  wordmark?: boolean
  wordmarkOnly?: boolean
}>(), {
  collapsed: false,
  homePath: '',
  wordmark: false,
  wordmarkOnly: false,
})

const appStore = useAppStore()
const placement = computed(() => props.placement)
const collapsed = computed(() => props.collapsed)
const wordmark = computed(() => props.wordmark)
const wordmarkOnly = computed(() => props.wordmarkOnly)
const logoVisible = computed(() => (
  collapsed.value || !wordmark.value || !wordmarkOnly.value
))

const siteName = computed(() => appStore.siteName || '落雪API')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', {
  allowRelative: true,
  allowDataUrl: true,
}))
const brandLogoSrc = computed(() => siteLogo.value || '/logo.png')
const isDefaultLogo = computed(() => brandLogoSrc.value === '/logo.png')
const frontendEntry = document.querySelector<HTMLMetaElement>('meta[name="app-entry"]')?.content
const homePath = computed(() => (
  props.homePath || (frontendEntry === 'admin' ? '/admin/dashboard' : '/dashboard')
))
</script>

<style scoped>
.app-brand--header,
.app-brand--collapsed {
  justify-content: center;
}

.app-brand--sidebar:not(.app-brand--collapsed) {
  justify-content: flex-start;
}

.app-brand-logo-frame {
  position: relative;
}

.app-brand--wordmark {
  min-width: 0;
  gap: 8px;
  color: var(--workspace-text);
  text-decoration: none;
}

.app-brand--wordmark-only {
  gap: 0;
}

.app-brand-wordmark {
  min-width: 0;
  overflow: hidden;
  font-size: var(--workspace-type-brand-size);
  font-weight: var(--workspace-type-brand-weight);
  letter-spacing: -0.01em;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.app-brand-logo-image-default {
  transform: scale(1.1);
  transform-origin: center;
}
</style>

<template>
  <router-link
    :to="homePath"
    :data-testid="`${placement}-brand`"
    class="app-brand flex h-10 w-full min-w-0 items-center gap-2 rounded-xl focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 md:h-16"
    :class="[
      `app-brand--${placement}`,
      { 'app-brand--collapsed': collapsed }
    ]"
    :aria-label="siteName"
  >
    <span
      :data-testid="`${placement}-brand-logo`"
      class="app-brand-logo-frame flex h-10 w-10 flex-shrink-0 items-center justify-center md:h-12 md:w-12"
    >
      <img
        :src="brandLogoSrc"
        alt=""
        class="app-brand-logo-image block h-full w-full max-w-none object-contain"
        :class="{ 'app-brand-logo-image-luoxue': isLuoxueLogo }"
      >
    </span>
  </router-link>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useAppStore, useAuthStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'

const props = withDefaults(defineProps<{
  placement: 'header' | 'sidebar'
  collapsed?: boolean
}>(), {
  collapsed: false
})

const appStore = useAppStore()
const authStore = useAuthStore()
const placement = computed(() => props.placement)
const collapsed = computed(() => props.collapsed)

const siteName = computed(() => appStore.siteName || '落雪API')
const isLuoxueBrand = computed(() => /^落雪\s*API$/i.test(siteName.value.trim()))
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', {
  allowRelative: true,
  allowDataUrl: true,
}))
const brandLogoSrc = computed(() => {
  if (siteLogo.value) return siteLogo.value

  return isLuoxueBrand.value
    ? '/brand/luoxue-snowflake-cloud-palette.png'
    : '/logo.png'
})
const isLuoxueLogo = computed(() => {
  if (!isLuoxueBrand.value) return false

  return /luoxue-snowflake-cloud-palette\.(?:png|svg)(?:[?#].*)?$/i.test(brandLogoSrc.value)
})
const homePath = computed(() => (authStore.isAdmin ? '/admin/dashboard' : '/dashboard'))
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

.app-brand-logo-image-luoxue {
  transform: scale(1.13);
  transform-origin: center;
}

:global(html.dark .app-brand-logo-image-luoxue) {
  filter:
    drop-shadow(0 0 0.75px rgb(255 255 255 / 0.95))
    drop-shadow(0 0 5px rgb(var(--luoxue-blue-rgb) / 0.22));
}
</style>

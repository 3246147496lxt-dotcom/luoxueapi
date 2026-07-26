<template>
  <header
    data-testid="app-mobile-header"
    class="app-mobile-header lg:hidden"
  >
    <div class="app-mobile-header__content">
      <button
        type="button"
        data-testid="mobile-header-menu"
        class="app-mobile-header__menu"
        :aria-label="mobileOpen ? t('nav.closeNavigation') : t('nav.openNavigation')"
        aria-controls="app-sidebar"
        :aria-expanded="mobileOpen"
        @click="toggleMobileSidebar"
      >
        <Icon name="menu" size="md" aria-hidden="true" />
      </button>

      <AppBrand placement="header" :collapsed="false" />

      <span
        data-testid="mobile-header-page-title"
        class="app-mobile-header__title"
      >
        {{ pageTitle }}
      </span>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { usePageContext } from '@/composables/usePageContext'
import Icon from '@/components/icons/Icon.vue'
import AppBrand from './AppBrand.vue'

const { t } = useI18n()
const appStore = useAppStore()
const { pageTitle } = usePageContext()

const mobileOpen = computed(() => appStore.mobileOpen)

function toggleMobileSidebar() {
  if (!mobileOpen.value) {
    appStore.setSidebarCollapsed(false)
  }
  appStore.toggleMobileSidebar()
}
</script>

<style scoped>
.app-mobile-header {
  position: fixed;
  inset: 0 0 auto;
  z-index: 50;
  height: var(--app-shell-top-offset);
  padding-top: env(safe-area-inset-top);
  border-bottom: 1px solid var(--app-shell-sidebar-border, rgb(229 231 235));
  color: var(--lx-clay-text);
  background: color-mix(in srgb, var(--app-shell-canvas, #fff) 94%, transparent);
  box-shadow: 0 1px 0 rgb(15 23 42 / 0.025);
  backdrop-filter: blur(18px) saturate(1.25);
  -webkit-backdrop-filter: blur(18px) saturate(1.25);
}

.app-mobile-header__content {
  display: grid;
  height: 56px;
  grid-template-columns: 44px 36px minmax(0, 1fr);
  align-items: center;
  gap: 6px;
  padding: 0 12px 0 8px;
}

.app-mobile-header__menu {
  display: inline-flex;
  width: 44px;
  height: 44px;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  color: rgb(71 85 105);
  transition:
    color 160ms ease,
    background-color 160ms ease;
}

.app-mobile-header__menu:hover {
  color: rgb(15 23 42);
  background: rgb(15 23 42 / 0.055);
}

.app-mobile-header__menu:focus-visible {
  outline: 2px solid var(--app-shell-sidebar-focus, rgb(0 132 255 / 0.5));
  outline-offset: 1px;
}

.app-mobile-header :deep(.app-brand) {
  width: 36px;
  height: 36px;
}

.app-mobile-header :deep(.app-brand-logo-frame) {
  width: 36px;
  height: 36px;
}

.app-mobile-header__title {
  min-width: 0;
  overflow: hidden;
  color: rgb(17 24 39);
  font-size: 0.9375rem;
  font-weight: 650;
  line-height: 1.25rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

:global(html.dark .app-mobile-header) {
  background: rgb(11 15 26 / 0.92);
}

:global(html.dark .app-mobile-header__menu) {
  color: rgb(203 213 225);
}

:global(html.dark .app-mobile-header__menu:hover) {
  color: #fff;
  background: rgb(255 255 255 / 0.08);
}

:global(html.dark .app-mobile-header__title) {
  color: rgb(248 250 252);
}

@media (prefers-reduced-motion: reduce) {
  .app-mobile-header__menu {
    transition-duration: 0.01ms;
  }
}
</style>

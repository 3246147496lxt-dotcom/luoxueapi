<template>
  <header
    data-testid="app-mobile-header"
    class="app-mobile-header"
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
  appStore.toggleMobileSidebar()
}
</script>

<style scoped>
.app-mobile-header {
  display: none;
  position: fixed;
  inset: 0 0 auto;
  z-index: 50;
  height: var(--app-shell-top-offset);
  padding-top: env(safe-area-inset-top);
  border-bottom: 1px solid var(--app-shell-sidebar-border, var(--workspace-divider));
  color: var(--workspace-text);
  background: var(--app-shell-sidebar-bg, var(--workspace-sidebar-surface));
  box-shadow: 0 1px 0 var(--workspace-divider);
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
  color: var(--workspace-text-secondary);
  transition:
    color 160ms ease,
    background-color 160ms ease;
}

.app-mobile-header__menu:hover {
  color: var(--workspace-text);
  background: var(--workspace-hover);
}

.app-mobile-header__menu:focus-visible {
  outline: 2px solid var(--app-shell-sidebar-focus, var(--lx-clay-accent));
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
  color: var(--workspace-text);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
  line-height: 1.25rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 767px) and (hover: none) and (pointer: coarse) {
  .app-mobile-header {
    display: block;
  }
}

@media (prefers-reduced-motion: reduce) {
  .app-mobile-header__menu {
    transition-duration: 0.01ms;
  }
}
</style>

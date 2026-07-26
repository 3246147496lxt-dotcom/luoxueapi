<template>
  <div
    class="public-site-page public-site-page--snow"
    :class="{
      'public-site-page--dark': isDark,
      'public-site-page--home': page === 'home',
      'public-site-page--models': page === 'models'
    }"
  >
    <header class="public-site-header" :class="{ 'public-site-header--elevated': isHeaderElevated }">
      <nav class="public-site-nav" :aria-label="t('home.nav.ariaLabel')">
        <a
          href="/home#top"
          class="public-site-brand"
          :aria-label="siteName"
          @click="closeMobileMenu()"
        >
          <span class="public-site-brand-mark">
            <img
              data-testid="public-site-brand-logo"
              :src="displayLogo"
              alt=""
              width="40"
              height="40"
              aria-hidden="true"
            />
          </span>
          <span class="public-site-brand-name" aria-hidden="true">
            <span v-if="brandNameParts.base">{{ brandNameParts.base }}</span>
            <span v-if="brandNameParts.apiSuffix" class="public-site-brand-api">
              {{ brandNameParts.apiSuffix }}
            </span>
          </span>
        </a>

        <div class="public-site-desktop-nav">
          <a
            v-for="item in navItems"
            :key="item.href"
            :href="item.href"
          >
            {{ t(item.labelKey) }}
          </a>
          <router-link
            v-if="catalogEntryVisible"
            to="/models.html"
            :aria-current="page === 'models' ? 'page' : undefined"
          >
            {{ t('modelCatalog.navLabel') }}
          </router-link>
          <a v-if="tutorialUrl" :href="tutorialUrl">{{ t('home.nav.tutorial') }}</a>
        </div>

        <div class="public-site-actions">
          <div class="public-site-desktop-action">
            <LocaleSwitcher icon-variant="lucide" />
          </div>

          <button
            type="button"
            class="public-site-icon-button"
            data-testid="theme-toggle"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            :aria-label="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            :aria-pressed="isDark"
            @click="toggleTheme"
          >
            <Icon
              :name="isDark ? 'lucideSun' : 'lucideMoon'"
              size="md"
              :stroke-width="2"
              aria-hidden="true"
            />
          </button>

          <router-link class="public-site-account-link public-site-desktop-action" :to="headerAccountPath">
            {{ headerAccountLabel }}
          </router-link>

          <button
            ref="mobileMenuButtonRef"
            type="button"
            class="public-site-icon-button public-site-mobile-menu-button"
            data-testid="mobile-menu-toggle"
            aria-controls="public-site-mobile-menu"
            :aria-expanded="mobileMenuOpen"
            :aria-label="mobileMenuOpen ? t('home.nav.closeMenu') : t('home.nav.openMenu')"
            @click="mobileMenuOpen = !mobileMenuOpen"
          >
            <Icon :name="mobileMenuOpen ? 'x' : 'menu'" size="md" aria-hidden="true" />
          </button>
        </div>
      </nav>

      <transition name="public-site-mobile-menu">
        <div v-if="mobileMenuOpen" id="public-site-mobile-menu" class="public-site-mobile-panel">
          <div class="public-site-mobile-inner">
            <a
              v-for="item in navItems"
              :key="item.href"
              :href="item.href"
              @click="closeMobileMenu()"
            >
              {{ t(item.labelKey) }}
            </a>
            <router-link
              v-if="catalogEntryVisible"
              to="/models.html"
              :aria-current="page === 'models' ? 'page' : undefined"
              @click="closeMobileMenu()"
            >
              {{ t('modelCatalog.navLabel') }}
            </router-link>
            <a v-if="tutorialUrl" :href="tutorialUrl" @click="closeMobileMenu()">
              {{ t('home.nav.tutorial') }}
            </a>
            <div class="public-site-mobile-footer">
              <LocaleSwitcher
                data-testid="mobile-locale-switcher"
                icon-variant="lucide"
              />
              <router-link :to="headerAccountPath" @click="closeMobileMenu()">
                {{ headerAccountLabel }}
                <Icon name="arrowRight" size="sm" aria-hidden="true" />
              </router-link>
            </div>
          </div>
        </div>
      </transition>
    </header>

    <slot />

    <footer class="public-site-footer">
      <div class="public-site-shell public-site-footer-layout">
        <div class="public-site-footer-brand">
          <span class="public-site-footer-mark" aria-hidden="true">
            <img :src="displayLogo" alt="" width="36" height="36" />
          </span>
          <p>&copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}</p>
        </div>
        <nav :aria-label="t('home.footer.ariaLabel')">
          <router-link v-if="catalogEntryVisible" to="/models.html">
            {{ t('modelCatalog.navLabel') }}
          </router-link>
          <a v-if="tutorialUrl" :href="tutorialUrl">{{ t('home.footer.tutorial') }}</a>
          <a
            v-if="docUrl"
            :href="docUrl"
            data-testid="public-api-docs-link"
          >{{ t('home.footer.apiDocs') }}</a>
          <router-link to="/monitor">{{ t('home.footer.channelStatus') }}</router-link>
        </nav>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore, useAuthStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { useThemePreference } from '@/composables/useThemePreference'
import { splitBrandApiSuffix } from '@/utils/brand'
import { resolveDocumentationUrl, resolveTutorialUrl } from '@/utils/documentationUrl'
import { sanitizeUrl } from '@/utils/url'

const props = withDefaults(defineProps<{
  page?: 'home' | 'models'
  showModelCatalog?: boolean
}>(), {
  page: 'home',
  showModelCatalog: undefined
})

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const { isDark, toggleTheme } = useThemePreference()

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || '落雪API')
const brandNameParts = computed(() => splitBrandApiSuffix(siteName.value))
const siteLogo = computed(() => sanitizeUrl(
  appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '',
  { allowRelative: true, allowDataUrl: true }
))
const displayLogo = computed(() => (
  siteLogo.value || '/brand/luoxue-snowpuff-extracted.svg'
))
const configuredDocUrl = computed(() => sanitizeUrl(
  appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '',
  { allowRelative: true },
))
const docUrl = computed(() => (
  configuredDocUrl.value ? resolveDocumentationUrl(configuredDocUrl.value) : ''
))
const tutorialUrl = computed(() => resolveTutorialUrl(configuredDocUrl.value))
const catalogEntryVisible = computed(() => (
  !appStore.backendModeEnabled
  && (
    props.page === 'models'
    || props.showModelCatalog
    || appStore.cachedPublicSettings?.public_model_catalog_enabled === true
  )
))

const navItems = [
  { href: '/home#steps', labelKey: 'home.nav.steps' },
  { href: '/home#capabilities', labelKey: 'home.nav.capabilities' },
  { href: '/home#providers', labelKey: 'home.nav.providers' },
  { href: '/home#faq', labelKey: 'home.nav.faq' }
] as const

const dashboardPath = computed(() => (authStore.isAdmin ? '/admin/dashboard' : '/dashboard'))
const headerAccountPath = computed(() => (authStore.isAuthenticated ? dashboardPath.value : '/login'))
const headerAccountLabel = computed(() => (
  authStore.isAuthenticated ? t('home.dashboard') : t('home.login')
))
const currentYear = computed(() => new Date().getFullYear())

const isHeaderElevated = ref(false)
const mobileMenuOpen = ref(false)
const mobileMenuButtonRef = ref<HTMLButtonElement | null>(null)

function closeMobileMenu(restoreFocus = false) {
  mobileMenuOpen.value = false
  if (restoreFocus) requestAnimationFrame(() => mobileMenuButtonRef.value?.focus())
}

function handleGlobalKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && mobileMenuOpen.value) closeMobileMenu(true)
}

function handleScroll() {
  isHeaderElevated.value = window.scrollY > 12
}

function handleViewportChange() {
  if (window.innerWidth >= 1024) closeMobileMenu()
}

onMounted(() => {
  handleScroll()
  window.addEventListener('scroll', handleScroll, { passive: true })
  window.addEventListener('keydown', handleGlobalKeydown)
  window.addEventListener('resize', handleViewportChange)
})

onBeforeUnmount(() => {
  window.removeEventListener('scroll', handleScroll)
  window.removeEventListener('keydown', handleGlobalKeydown)
  window.removeEventListener('resize', handleViewportChange)
})
</script>

<style scoped>
.public-site-page {
  --page: var(--lx-clay-canvas);
  --surface: var(--lx-clay-surface);
  --surface-soft: var(--lx-clay-recessed);
  --surface-accent: var(--lx-clay-surface);
  --ink: var(--lx-clay-text);
  --copy: var(--lx-clay-text-secondary);
  --muted: var(--lx-clay-text-secondary);
  --border: var(--lx-clay-border);
  --accent: var(--lx-clay-accent);
  --accent-strong: var(--lx-clay-accent);
  --accent-ink: var(--lx-clay-accent);
  min-height: 100vh;
  overflow-x: clip;
  background: var(--page);
  color: var(--ink);
  font-family: var(--lx-clay-font-ui);
}

.public-site-page :where(a, button):focus-visible {
  outline: 3px solid color-mix(in srgb, var(--accent) 72%, white);
  outline-offset: 3px;
}

.public-site-shell,
.public-site-nav,
.public-site-mobile-inner {
  width: min(1120px, calc(100% - 48px));
  margin-inline: auto;
}

.public-site-header {
  position: sticky;
  top: 0;
  z-index: 40;
  border-bottom: 1px solid transparent;
  background: var(--page);
  transition: border-color 170ms ease, box-shadow 170ms ease;
}

.public-site-header--elevated {
  border-bottom-color: var(--border);
  box-shadow: 0 12px 32px rgba(15, 23, 42, 0.08);
}

.public-site-page--dark .public-site-header--elevated {
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.34);
}

.public-site-nav {
  min-height: 72px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
}

.public-site-brand,
.public-site-actions,
.public-site-desktop-nav,
.public-site-mobile-footer,
.public-site-account-link {
  display: flex;
  align-items: center;
}

.public-site-brand {
  min-height: 44px;
  min-width: 0;
  gap: 12px;
  color: var(--ink);
  text-decoration: none;
}

.public-site-brand-mark {
  width: 40px;
  height: 40px;
  flex: 0 0 auto;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface);
}

.public-site-brand-mark img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.public-site-brand-name {
  overflow: hidden;
  color: var(--ink);
  font-size: 17px;
  font-weight: 700;
  letter-spacing: -0.02em;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.public-site-desktop-nav {
  gap: clamp(16px, 1.8vw, 26px);
}

.public-site-desktop-nav a,
.public-site-mobile-panel a,
.public-site-footer a {
  color: var(--copy);
  text-decoration: none;
  transition: color 150ms ease, background-color 150ms ease;
}

.public-site-desktop-nav a {
  padding-block: 12px;
  font-size: 14px;
  font-weight: 600;
}

.public-site-desktop-nav a[aria-current="page"],
.public-site-desktop-nav a:hover,
.public-site-mobile-panel a:hover,
.public-site-footer a:hover {
  color: var(--accent-ink);
}

.public-site-actions {
  flex: 0 0 auto;
  gap: 8px;
}

.public-site-icon-button {
  width: 44px;
  height: 44px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: 10px;
  background: transparent;
  color: var(--copy);
  cursor: pointer;
  transition: background-color 150ms ease, border-color 150ms ease, color 150ms ease;
}

.public-site-icon-button:hover {
  border-color: var(--border);
  background: var(--surface-soft);
  color: var(--ink);
}

.public-site-account-link {
  min-height: 44px;
  gap: 7px;
  border-radius: 10px;
  background: var(--ink);
  color: var(--page);
  padding: 0 16px;
  font-size: 13px;
  font-weight: 700;
  text-decoration: none;
  transition: transform 150ms ease;
}

.public-site-account-link:hover {
  transform: translateY(-1px);
}

.public-site-mobile-menu-button,
.public-site-mobile-panel {
  display: none;
}

.public-site-footer {
  border-top: 1px solid var(--border);
  padding-block: 30px;
}

.public-site-footer-layout {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
}

.public-site-footer-brand {
  display: flex;
  align-items: center;
  gap: 12px;
}

.public-site-footer-mark {
  width: 36px;
  height: 36px;
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
}

.public-site-footer-mark img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.public-site-footer p {
  margin: 0;
  color: var(--muted);
  font-size: 13px;
}

.public-site-footer nav {
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
}

.public-site-footer a {
  min-height: 44px;
  display: inline-flex;
  align-items: center;
  font-size: 13px;
  font-weight: 620;
}

.public-site-mobile-menu-enter-active,
.public-site-mobile-menu-leave-active {
  transition: opacity 170ms ease, transform 170ms cubic-bezier(0.22, 1, 0.36, 1);
}

.public-site-mobile-menu-enter-from,
.public-site-mobile-menu-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

@media (max-width: 1023px) {
  .public-site-desktop-nav,
  .public-site-desktop-action {
    display: none;
  }

  .public-site-mobile-menu-button {
    display: inline-flex;
  }

  .public-site-mobile-panel {
    position: absolute;
    top: 100%;
    right: 0;
    left: 0;
    display: block;
    max-height: calc(100dvh - 100px);
    overflow-y: auto;
    overscroll-behavior: contain;
    border-bottom: 1px solid var(--border);
    background: var(--page);
    box-shadow: 0 20px 34px rgba(15, 23, 42, 0.1);
  }

  .public-site-page--dark .public-site-mobile-panel {
    box-shadow: 0 20px 34px rgba(0, 0, 0, 0.34);
  }

  .public-site-mobile-inner {
    display: grid;
    padding-block: 12px 20px;
  }

  .public-site-mobile-inner > a {
    min-height: 52px;
    display: flex;
    align-items: center;
    border-bottom: 1px solid var(--border);
    font-size: 15px;
    width: 100%;
  }

  .public-site-mobile-footer {
    min-height: 62px;
    justify-content: space-between;
    gap: 16px;
    padding-top: 10px;
  }

  .public-site-mobile-footer > a {
    min-height: 44px;
    display: inline-flex;
    align-items: center;
    gap: 7px;
    border: 1px solid var(--border);
    border-radius: 10px;
    padding-inline: 14px;
    color: var(--ink);
    font-size: 13px;
    font-weight: 700;
  }
}

@media (max-width: 767px) {
  .public-site-shell,
  .public-site-nav,
  .public-site-mobile-inner {
    width: min(100% - 32px, 1120px);
  }

  .public-site-nav {
    min-height: 64px;
  }

  .public-site-brand {
    gap: 9px;
  }

  .public-site-brand-mark {
    width: 38px;
    height: 38px;
  }

  .public-site-brand-name {
    max-width: 128px;
    font-size: 16px;
  }

  .public-site-footer-layout {
    align-items: flex-start;
    flex-direction: column;
  }
}

@media (max-width: 359px) {
  .public-site-brand-name {
    max-width: 92px;
  }

  .public-site-actions {
    gap: 2px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .public-site-page *,
  .public-site-page *::before,
  .public-site-page *::after {
    scroll-behavior: auto !important;
    transition-duration: 0.01ms !important;
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
  }
}

/* Exact public shell from “Candy Clay Snowflake Stage - Refined Fidelity”. */
.public-site-page--snow {
  --surface-elevated: var(--lx-clay-surface-elevated);
  --primary-violet: var(--lx-clay-accent);
  --violet-highlight: var(--lx-clay-accent-highlight);
  --action-violet-start: var(--lx-clay-light-accent);
  --action-violet-end: var(--lx-clay-light-accent-deep);
  min-height: 100vh;
  color: var(--ink);
  background-color: var(--page);
  background-image:
    radial-gradient(circle at 15% 15%, color-mix(in srgb, var(--lx-clay-light-accent) 8%, transparent) 0%, transparent 40%),
    radial-gradient(circle at 85% 20%, color-mix(in srgb, var(--lx-clay-light-info) 8%, transparent) 0%, transparent 40%);
  font-family: var(--lx-clay-font-ui);
}

.public-site-page--snow .public-site-header,
.public-site-page--snow .public-site-header--elevated {
  top: 24px;
  padding: 0;
  background: transparent;
  border: 0;
  box-shadow: none;
}

.public-site-page--snow .public-site-nav,
.public-site-page--snow .public-site-header--elevated .public-site-nav,
.public-site-page--snow.public-site-page--dark .public-site-header--elevated .public-site-nav {
  width: min(1192px, calc(100% - 48px));
  min-height: 80px;
  height: 80px;
  padding: 0 32px;
  background: var(--surface-elevated);
  border: 1px solid var(--border);
  border-radius: 20px;
  box-shadow:
    16px 16px 32px rgba(160, 150, 180, 0.2),
    -10px -10px 24px rgba(255, 255, 255, 0.9),
    inset 6px 6px 12px rgba(139, 92, 246, 0.03),
    inset -6px -6px 12px rgba(255, 255, 255, 1);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  transform: none;
}

.public-site-page--snow .public-site-brand {
  gap: 12px;
}

.public-site-page--snow .public-site-brand-mark,
.public-site-page--snow.public-site-page--dark .public-site-brand-mark {
  display: flex;
  width: 48px;
  height: 48px;
  align-items: center;
  justify-content: center;
  padding: 0;
  overflow: visible;
  background: transparent;
  border: 0;
  border-radius: 0;
  box-shadow: none;
}

.public-site-page--snow .public-site-brand-name {
  display: inline-flex;
  align-items: baseline;
  gap: 0.24em;
  max-width: none;
  color: var(--ink);
  font-family: "Nunito", sans-serif;
  font-size: 24px;
  font-weight: 900;
  letter-spacing: 0;
}

.public-site-page--snow .public-site-brand-api {
  color: var(--lx-clay-light-info);
}

.public-site-page--snow.public-site-page--dark .public-site-brand-api {
  color: var(--lx-clay-dark-info-deep);
}

.public-site-page--snow .public-site-desktop-nav {
  gap: 32px;
}

.public-site-page--snow .public-site-desktop-nav a {
  padding: 12px 0;
  color: var(--copy);
  background: transparent;
  border-radius: 0;
  box-shadow: none;
  font-size: 14px;
  font-weight: 700;
}

.public-site-page--snow .public-site-desktop-nav a:hover,
.public-site-page--snow .public-site-desktop-nav a[aria-current="page"],
.public-site-page--snow.public-site-page--dark .public-site-desktop-nav a:hover,
.public-site-page--snow.public-site-page--dark .public-site-desktop-nav a[aria-current="page"] {
  color: var(--primary-violet);
  background: transparent;
  box-shadow: none;
}

.public-site-page--snow .public-site-actions {
  gap: 8px;
}

.public-site-page--snow .public-site-icon-button,
.public-site-page--snow .public-site-desktop-action :deep(button[aria-haspopup="menu"]),
.public-site-page--snow.public-site-page--dark .public-site-icon-button,
.public-site-page--snow.public-site-page--dark .public-site-desktop-action :deep(button[aria-haspopup="menu"]) {
  width: 44px;
  min-width: 44px;
  height: 44px;
  min-height: 44px;
  padding: 0;
  color: var(--copy) !important;
  background: var(--surface-soft);
  border: 0;
  border-radius: 20px;
  box-shadow:
    inset 10px 10px 20px rgba(91, 80, 112, 0.1),
    inset -10px -10px 20px rgba(255, 255, 255, 0.8);
}

.public-site-page--snow .public-site-icon-button:hover,
.public-site-page--snow .public-site-desktop-action :deep(button[aria-haspopup="menu"]):hover {
  color: var(--primary-violet) !important;
  background: var(--surface-soft);
}

.public-site-page--snow .public-site-account-link {
  min-height: 44px;
  height: 44px;
  gap: 0;
  padding: 0 24px;
  color: #ffffff;
  background: linear-gradient(135deg, var(--action-violet-start), var(--action-violet-end));
  border-radius: 20px;
  box-shadow:
    12px 12px 24px rgba(139, 92, 246, 0.3),
    -8px -8px 16px rgba(255, 255, 255, 0.4),
    inset 4px 4px 8px rgba(255, 255, 255, 0.4),
    inset -4px -4px 8px rgba(0, 0, 0, 0.1);
  font-size: 14px;
  font-weight: 700;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.public-site-page--snow .public-site-account-link:hover {
  transform: translateY(-4px);
  box-shadow:
    16px 16px 32px rgba(139, 92, 246, 0.4),
    -8px -8px 16px rgba(255, 255, 255, 0.4);
}

.public-site-page--snow .public-site-account-link:active {
  transform: scale(0.92);
}

.public-site-page--snow .public-site-footer {
  padding: 64px 0;
  background: transparent;
  border-top: 1px solid var(--border);
}

.public-site-page--snow .public-site-shell {
  width: min(1192px, calc(100% - 48px));
}

.public-site-page--snow .public-site-footer-layout {
  gap: 32px;
}

.public-site-page--snow .public-site-footer-brand {
  gap: 12px;
}

.public-site-page--snow .public-site-footer-mark,
.public-site-page--snow.public-site-page--dark .public-site-footer-mark {
  display: flex;
  width: 40px;
  height: 40px;
  align-items: center;
  justify-content: center;
  padding: 0;
  background: transparent;
  border: 0;
  border-radius: 0;
  box-shadow: none;
}

.public-site-page--snow .public-site-footer p {
  color: var(--copy);
  font-size: 14px;
  font-style: normal;
  font-weight: 700;
}

.public-site-page--snow .public-site-footer nav {
  gap: 32px;
}

.public-site-page--snow .public-site-footer a {
  padding: 0;
  color: var(--copy);
  background: transparent;
  border-radius: 0;
  font-size: 14px;
  font-weight: 700;
}

.public-site-page--snow .public-site-footer a:hover {
  color: var(--primary-violet);
  background: transparent;
}

.public-site-page--snow.public-site-page--dark .public-site-nav,
.public-site-page--snow.public-site-page--dark .public-site-header--elevated .public-site-nav {
  box-shadow:
    16px 16px 32px rgba(0, 0, 0, 0.34),
    -10px -10px 24px rgba(255, 255, 255, 0.025),
    inset 6px 6px 12px rgba(255, 255, 255, 0.02);
}

.public-site-page--snow.public-site-page--dark .public-site-icon-button,
.public-site-page--snow.public-site-page--dark .public-site-desktop-action :deep(button[aria-haspopup="menu"]) {
  box-shadow:
    inset 10px 10px 20px rgba(0, 0, 0, 0.34),
    inset -10px -10px 20px rgba(255, 255, 255, 0.025);
}

.public-site-page--snow.public-site-page--dark .public-site-account-link {
  box-shadow:
    12px 12px 24px rgba(0, 0, 0, 0.32),
    -8px -8px 16px rgba(255, 255, 255, 0.025),
    inset 4px 4px 8px rgba(255, 255, 255, 0.08),
    inset -4px -4px 8px rgba(0, 0, 0, 0.22);
}

@media (max-width: 1023px) {
  .public-site-page--snow .public-site-header,
  .public-site-page--snow .public-site-header--elevated {
    top: 16px;
  }

  .public-site-page--snow .public-site-nav,
  .public-site-page--snow .public-site-header--elevated .public-site-nav,
  .public-site-page--snow.public-site-page--dark .public-site-header--elevated .public-site-nav {
    width: calc(100% - 32px);
    min-height: 72px;
    height: 72px;
    padding: 0 20px;
  }

  .public-site-page--snow .public-site-desktop-action {
    display: flex;
  }

  .public-site-page--snow .public-site-mobile-panel {
    top: calc(100% + 12px);
    right: 16px;
    left: 16px;
    max-height: calc(100dvh - 104px);
    overflow-x: hidden;
    overflow-y: auto;
    background: var(--surface-elevated);
    border: 1px solid var(--border);
    border-radius: 20px;
    box-shadow:
      16px 16px 32px rgba(160, 150, 180, 0.2),
      -10px -10px 24px rgba(255, 255, 255, 0.9),
      inset 6px 6px 12px rgba(139, 92, 246, 0.03),
      inset -6px -6px 12px rgba(255, 255, 255, 1);
    backdrop-filter: blur(20px);
  }

  .public-site-page--snow.public-site-page--dark .public-site-mobile-panel {
    box-shadow:
      16px 16px 32px rgba(0, 0, 0, 0.34),
      -10px -10px 24px rgba(255, 255, 255, 0.025),
      inset 6px 6px 12px rgba(255, 255, 255, 0.02);
  }

  .public-site-page--snow .public-site-mobile-inner {
    width: auto;
    margin: 0;
    padding: 14px;
  }

  .public-site-page--snow .public-site-mobile-inner > a {
    min-height: 52px;
    padding: 0 12px;
    color: var(--copy);
    border-color: var(--border);
    border-radius: 12px;
    font-weight: 700;
  }

  .public-site-page--snow .public-site-mobile-inner > a:hover {
    color: var(--primary-violet);
    background: var(--surface-soft);
  }

  .public-site-page--snow .public-site-mobile-footer {
    display: none;
  }
}

@media (max-width: 767px) {
  .public-site-page--snow .public-site-header,
  .public-site-page--snow .public-site-header--elevated {
    top: 12px;
  }

  .public-site-page--snow .public-site-nav,
  .public-site-page--snow .public-site-header--elevated .public-site-nav,
  .public-site-page--snow.public-site-page--dark .public-site-header--elevated .public-site-nav {
    width: calc(100% - 24px);
    min-height: 64px;
    height: 64px;
    padding: 0 10px;
  }

  .public-site-page--snow .public-site-brand {
    gap: 0;
  }

  .public-site-page--snow .public-site-brand-mark,
  .public-site-page--snow.public-site-page--dark .public-site-brand-mark {
    width: 42px;
    height: 42px;
    padding: 0;
  }

  .public-site-page--snow .public-site-brand-name {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip: rect(0 0 0 0);
    white-space: nowrap;
  }

  .public-site-page--snow .public-site-actions {
    gap: 5px;
  }

  .public-site-page--snow .public-site-icon-button,
  .public-site-page--snow .public-site-desktop-action :deep(button[aria-haspopup="menu"]),
  .public-site-page--snow.public-site-page--dark .public-site-icon-button,
  .public-site-page--snow.public-site-page--dark .public-site-desktop-action :deep(button[aria-haspopup="menu"]) {
    width: 44px;
    min-width: 44px;
    height: 44px;
    min-height: 44px;
  }

  .public-site-page--snow .public-site-account-link {
    min-height: 44px;
    height: 44px;
    padding: 0 12px;
    font-size: 12px;
  }

  .public-site-page--snow .public-site-mobile-panel {
    right: 12px;
    left: 12px;
  }

  .public-site-page--snow .public-site-footer {
    padding: 64px 0;
  }

  .public-site-page--snow .public-site-shell {
    width: calc(100% - 32px);
  }

  .public-site-page--snow .public-site-footer-layout {
    align-items: flex-start;
    gap: 24px;
  }

  .public-site-page--snow .public-site-footer nav {
    gap: 12px 24px;
  }
}

@media (max-width: 420px) {
  .public-site-page--snow .public-site-desktop-action:first-child {
    display: none;
  }

  .public-site-page--snow .public-site-mobile-footer {
    display: flex;
    justify-content: flex-end;
  }

  .public-site-page--snow .public-site-mobile-footer > a {
    display: none;
  }
}
</style>

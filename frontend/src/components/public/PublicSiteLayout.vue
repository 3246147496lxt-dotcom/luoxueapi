<template>
  <div class="public-site-page" :class="{ 'public-site-page--dark': isDark }">
    <header class="public-site-header" :class="{ 'public-site-header--elevated': isHeaderElevated }">
      <nav class="public-site-nav" :aria-label="t('home.nav.ariaLabel')">
        <a href="/home#top" class="public-site-brand" @click="closeMobileMenu()">
          <span class="public-site-brand-mark">
            <img :src="siteLogo || '/logo.png'" :alt="siteName" width="40" height="40" />
          </span>
          <span class="public-site-brand-name">{{ siteName }}</span>
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
          <div class="public-site-desktop-action"><LocaleSwitcher /></div>

          <button
            type="button"
            class="public-site-icon-button"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            :aria-label="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            :aria-pressed="isDark"
            @click="toggleTheme"
          >
            <Icon v-if="isDark" name="sun" size="md" aria-hidden="true" />
            <Icon v-else name="moon" size="md" aria-hidden="true" />
          </button>

          <router-link class="public-site-account-link public-site-desktop-action" :to="headerAccountPath">
            {{ headerAccountLabel }}
            <Icon name="arrowRight" size="sm" aria-hidden="true" />
          </router-link>

          <button
            ref="mobileMenuButtonRef"
            type="button"
            class="public-site-icon-button public-site-mobile-menu-button"
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
              <LocaleSwitcher />
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
        <p>&copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}</p>
        <nav :aria-label="t('home.nav.ariaLabel')">
          <router-link v-if="catalogEntryVisible" to="/models.html">
            {{ t('modelCatalog.navLabel') }}
          </router-link>
          <a v-if="tutorialUrl" :href="tutorialUrl">{{ t('home.footer.tutorial') }}</a>
          <a v-if="docUrl" :href="docUrl">{{ t('home.footer.apiDocs') }}</a>
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

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || '落雪API')
const siteLogo = computed(() => sanitizeUrl(
  appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '',
  { allowRelative: true, allowDataUrl: true }
))
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const tutorialUrl = computed(() => {
  const base = (docUrl.value || '/tutorial-docs/').replace(/#.*$/, '')
  return `${base}#quick-start`
})
const catalogEntryVisible = computed(() => (
  !appStore.backendModeEnabled
  && (
    props.page === 'models'
    || props.showModelCatalog
    || appStore.cachedPublicSettings?.public_model_catalog_enabled === true
  )
))

const navItems = [
  { href: '/home#capabilities', labelKey: 'home.nav.capabilities' },
  { href: '/home#steps', labelKey: 'home.nav.steps' },
  { href: '/home#providers', labelKey: 'home.nav.providers' },
  { href: '/home#faq', labelKey: 'home.nav.faq' }
] as const

const dashboardPath = computed(() => (authStore.isAdmin ? '/admin/dashboard' : '/dashboard'))
const headerAccountPath = computed(() => (authStore.isAuthenticated ? dashboardPath.value : '/login'))
const headerAccountLabel = computed(() => (
  authStore.isAuthenticated ? t('home.dashboard') : t('home.login')
))
const currentYear = computed(() => new Date().getFullYear())

const isDark = ref(document.documentElement.classList.contains('dark'))
const isHeaderElevated = ref(false)
const mobileMenuOpen = ref(false)
const mobileMenuButtonRef = ref<HTMLButtonElement | null>(null)

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

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

onMounted(() => {
  handleScroll()
  window.addEventListener('scroll', handleScroll, { passive: true })
  window.addEventListener('keydown', handleGlobalKeydown)
})

onBeforeUnmount(() => {
  window.removeEventListener('scroll', handleScroll)
  window.removeEventListener('keydown', handleGlobalKeydown)
})
</script>

<style scoped>
.public-site-page {
  --page: #ffffff;
  --surface: #ffffff;
  --surface-soft: #f4f7f5;
  --surface-accent: #f0fdfa;
  --ink: #0f172a;
  --copy: #475569;
  --muted: #64748b;
  --border: #dde4e0;
  --accent: #0f766e;
  --accent-strong: #0b5f59;
  --accent-ink: #115e59;
  min-height: 100vh;
  overflow-x: clip;
  background: var(--page);
  color: var(--ink);
  font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC",
    "Microsoft YaHei", sans-serif;
}

.public-site-page--dark {
  --page: #0e1211;
  --surface: #111816;
  --surface-soft: #1a211e;
  --surface-accent: #17312e;
  --ink: #f4f7f5;
  --copy: #c1cbc6;
  --muted: #9aa6a0;
  --border: #28312c;
  --accent: #5eead4;
  --accent-strong: #99f6e4;
  --accent-ink: #99f6e4;
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
</style>

<template>
  <header
    data-testid="app-header"
    class="app-header fixed left-0 right-0 top-0 z-50 h-[81px] bg-transparent p-2 transition-[left] duration-300 ease-out motion-reduce:transition-none"
    :class="
      sidebarCollapsed
        ? 'lg:left-[68px]'
        : 'lg:left-[184px] min-[1025px]:left-[196px] min-[1281px]:left-[208px]'
    "
  >
    <div
      data-testid="header-surface"
      class="topup-header-surface relative isolate flex h-[65px] w-full items-center justify-between overflow-visible rounded-2xl border pr-2"
    >
      <!-- Left: brand, sidebar controls, and page context -->
      <div class="relative z-[1] flex min-w-0 flex-1 items-center gap-0 lg:gap-2">
        <button
          type="button"
          @click="toggleMobileSidebar"
          data-testid="header-mobile-menu"
          class="relative flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-[10px] text-gray-600 transition-colors after:absolute after:-inset-1.5 after:content-[''] hover:bg-gray-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 lg:hidden dark:text-dark-300 dark:hover:bg-dark-800 dark:focus-visible:ring-offset-dark-900"
          :aria-label="mobileOpen ? t('nav.collapse') : t('nav.expand')"
          aria-controls="app-sidebar"
          :aria-expanded="mobileOpen"
        >
          <Icon name="menu" size="md" />
        </button>

        <div
          data-testid="header-brand-slot"
          class="flex w-10 min-w-0 flex-shrink-0 items-center justify-center md:w-12 lg:hidden"
        >
          <AppBrand placement="header" :collapsed="sidebarCollapsed" />
        </div>

        <div
          data-testid="header-page-context"
          class="hidden min-w-0 items-center gap-2 text-sm lg:flex"
        >
          <span class="flex-none font-medium text-gray-400 dark:text-dark-400">
            {{ t('nav.management') }}
          </span>
          <span aria-hidden="true" class="text-gray-300 dark:text-dark-600">/</span>
          <h1 class="truncate font-semibold text-gray-950 dark:text-white">
            {{ pageTitle }}
          </h1>
        </div>
      </div>

      <!-- Right: utility icons + supporting status + account pill -->
      <div class="relative z-[1] flex flex-shrink-0 items-center gap-2 md:gap-3">
        <div
          data-testid="header-utility-actions"
          class="hidden items-center gap-3 md:flex"
        >
          <!-- Announcement Bell -->
          <AnnouncementBell v-if="user" compact />

          <!-- Theme Toggle -->
          <button
            type="button"
            data-testid="header-theme-toggle"
            class="brand-utility-icon relative flex h-8 w-8 items-center justify-center rounded-full transition-colors duration-200 after:absolute after:-inset-1.5 after:content-[''] hover:bg-[rgba(46,50,56,0.05)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 dark:hover:bg-white/[0.08] dark:focus-visible:ring-offset-dark-900"
            :title="isDark ? t('nav.lightMode') : t('nav.darkMode')"
            :aria-label="isDark ? t('nav.lightMode') : t('nav.darkMode')"
            @click="toggleTheme"
          >
            <Icon
              :name="isDark ? 'lucideSun' : 'lucideMoon'"
              size="md"
              :stroke-width="2"
              aria-hidden="true"
            />
          </button>

          <!-- Language Switcher -->
          <LocaleSwitcher compact icon-variant="lucide" />
        </div>

        <!-- Subscription Progress (for users with active subscriptions) -->
        <div v-if="user" class="hidden 2xl:block">
          <SubscriptionProgressMini />
        </div>

        <!-- Balance Display -->
        <div
          v-if="user"
          data-testid="header-balance"
          class="group relative hidden h-9 items-center gap-2 rounded-lg border border-primary-100 bg-primary-50/80 px-3 dark:border-primary-900/50 dark:bg-primary-900/20 min-[1400px]:flex"
        >
          <CreditAmount
            class="text-sm font-semibold text-primary-700 dark:text-primary-300"
            :value="formatHeaderCredit(availableBalance)"
            icon-size="sm"
            :label="`${balanceAvailableText} ${formatHeaderCredit(availableBalance)}`"
          />
          <span
            v-if="frozenBalance > 0"
            class="inline-flex items-center gap-1 rounded-full bg-amber-100 px-1.5 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-900/40 dark:text-amber-200"
          >
            <span>{{ balanceFrozenText }}</span>
            <CreditAmount
              :value="formatHeaderCredit(frozenBalance)"
              icon-size="xs"
              :label="`${balanceFrozenText} ${formatHeaderCredit(frozenBalance)}`"
            />
          </span>
          <div
            class="pointer-events-none absolute right-0 top-full mt-2 hidden w-56 rounded-xl border border-gray-200 bg-white p-3 text-xs shadow-[0_12px_32px_rgba(15,23,42,0.10)] group-hover:block dark:border-dark-700 dark:bg-dark-800 dark:shadow-black/30"
          >
            <div class="flex items-center justify-between">
              <span class="text-gray-500 dark:text-dark-400">{{ balanceAvailableText }}</span>
              <CreditAmount
                class="font-medium text-gray-900 dark:text-white"
                :value="formatHeaderCredit(availableBalance)"
                icon-size="xs"
                :label="`${balanceAvailableText} ${formatHeaderCredit(availableBalance)}`"
              />
            </div>
            <div class="mt-2 flex items-center justify-between">
              <span class="text-gray-500 dark:text-dark-400">{{ balanceFrozenText }}</span>
              <CreditAmount
                class="font-medium text-amber-700 dark:text-amber-200"
                :value="formatHeaderCredit(frozenBalance)"
                icon-size="xs"
                :label="`${balanceFrozenText} ${formatHeaderCredit(frozenBalance)}`"
              />
            </div>
            <div class="mt-2 border-t border-gray-100 pt-2 dark:border-dark-700">
              <div class="flex items-center justify-between">
                <span class="text-gray-500 dark:text-dark-400">{{ balanceTotalText }}</span>
                <CreditAmount
                  class="font-semibold text-gray-900 dark:text-white"
                  :value="formatHeaderCredit(totalBalance)"
                  icon-size="xs"
                  :label="`${balanceTotalText} ${formatHeaderCredit(totalBalance)}`"
                />
              </div>
            </div>
          </div>
        </div>

        <!-- User Dropdown -->
        <div
          v-if="user"
          ref="dropdownRef"
          class="relative"
        >
          <button
            type="button"
            @click="toggleDropdown"
            @keydown.esc.stop.prevent="closeDropdown"
            data-testid="header-account-trigger"
            class="relative flex h-8 items-center gap-0 rounded-full bg-slate-900/[0.04] p-1 transition-colors after:absolute after:-inset-y-1.5 after:-inset-x-1 after:content-[''] hover:bg-slate-900/[0.08] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 dark:bg-white/[0.06] dark:hover:bg-white/[0.10] dark:focus-visible:ring-offset-dark-900"
            :class="{
              'bg-primary-50 dark:bg-primary-900/25': dropdownOpen
            }"
            :aria-label="`${displayName} · ${t('nav.profile')}`"
            aria-haspopup="menu"
            :aria-expanded="dropdownOpen"
            aria-controls="header-account-menu"
          >
            <div
              data-testid="header-account-avatar"
              class="header-account-avatar mr-1 flex h-6 w-6 flex-shrink-0 items-center justify-center overflow-hidden rounded-full text-[10px] font-semibold"
            >
              <img
                v-if="avatarUrl"
                :src="avatarUrl"
                :alt="displayName"
                class="h-full w-full object-cover"
              >
              <span v-else>{{ userInitials }}</span>
            </div>
            <div class="hidden min-w-0 max-w-24 text-left md:block">
              <div class="truncate text-xs font-medium text-gray-900 dark:text-white">
                {{ displayName }}
              </div>
            </div>
            <Icon
              name="chevronDown"
              size="sm"
              class="!h-3.5 !w-3.5 text-gray-400 transition-transform duration-200 motion-reduce:transition-none dark:text-dark-400"
              :class="{ 'rotate-180': dropdownOpen }"
              aria-hidden="true"
            />
          </button>

          <!-- Dropdown Menu -->
          <transition name="dropdown">
            <div
              v-if="dropdownOpen"
              id="header-account-menu"
              class="header-dropdown dropdown right-0 mt-2 w-56 max-w-[calc(100vw-2rem)]"
              role="menu"
            >
              <!-- User Info -->
              <div class="border-b border-gray-100 px-3 py-2 dark:border-dark-700">
                <div class="text-sm font-medium text-gray-900 dark:text-white">
                  {{ displayName }}
                </div>
                <div class="truncate text-xs text-gray-500 dark:text-dark-400">{{ user.email }}</div>
              </div>

              <!-- Balance (mobile only) -->
              <div class="border-b border-gray-100 px-3 py-2 dark:border-dark-700 sm:hidden">
                <div class="text-xs text-gray-500 dark:text-dark-400">
                  {{ t('common.balance') }}
                </div>
                <div class="text-sm font-semibold text-primary-600 dark:text-primary-400">
                  <CreditAmount
                    :value="formatHeaderCredit(availableBalance)"
                    icon-size="sm"
                    :label="`${balanceAvailableText} ${formatHeaderCredit(availableBalance)}`"
                  />
                </div>
                <div v-if="frozenBalance > 0" class="mt-1 flex items-center gap-1 text-xs text-amber-600 dark:text-amber-300">
                  <span>{{ balanceFrozenText }}</span>
                  <CreditAmount
                    :value="formatHeaderCredit(frozenBalance)"
                    icon-size="xs"
                    :label="`${balanceFrozenText} ${formatHeaderCredit(frozenBalance)}`"
                  />
                </div>
              </div>

              <div class="py-1">
                <router-link to="/profile" @click="closeDropdown" class="dropdown-item">
                  <Icon name="user" size="sm" />
                  {{ t('nav.profile') }}
                </router-link>

                <router-link to="/keys" @click="closeDropdown" class="dropdown-item">
                  <Icon name="key" size="sm" />
                  {{ t('nav.apiKeys') }}
                </router-link>
              </div>

              <!-- Contact Support (only show if configured) -->
              <div
                v-if="contactInfo"
                class="border-t border-gray-100 px-3 py-2 dark:border-dark-700"
              >
                <div class="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                  <svg
                    class="h-3.5 w-3.5 flex-shrink-0"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    stroke-width="1.5"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      d="M20.25 8.511c.884.284 1.5 1.128 1.5 2.097v4.286c0 1.136-.847 2.1-1.98 2.193-.34.027-.68.052-1.02.072v3.091l-3-3c-1.354 0-2.694-.055-4.02-.163a2.115 2.115 0 01-.825-.242m9.345-8.334a2.126 2.126 0 00-.476-.095 48.64 48.64 0 00-8.048 0c-1.131.094-1.976 1.057-1.976 2.192v4.286c0 .837.46 1.58 1.155 1.951m9.345-8.334V6.637c0-1.621-1.152-3.026-2.76-3.235A48.455 48.455 0 0011.25 3c-2.115 0-4.198.137-6.24.402-1.608.209-2.76 1.614-2.76 3.235v6.226c0 1.621 1.152 3.026 2.76 3.235.577.075 1.157.14 1.74.194V21l4.155-4.155"
                    />
                  </svg>
                  <span>{{ t('common.contactSupport') }}:</span>
                  <span class="font-medium text-gray-700 dark:text-gray-300">{{
                    contactInfo
                  }}</span>
                </div>
              </div>

              <div v-if="showOnboardingButton" class="border-t border-gray-100 py-1 dark:border-dark-700">
                <button @click="handleReplayGuide" class="dropdown-item w-full">
                  <svg class="h-4 w-4" fill="currentColor" viewBox="0 0 24 24">
                    <path
                      d="M12 2a10 10 0 100 20 10 10 0 000-20zm0 14a1 1 0 110 2 1 1 0 010-2zm1.07-7.75c0-.6-.49-1.25-1.32-1.25-.7 0-1.22.4-1.43 1.02a1 1 0 11-1.9-.62A3.41 3.41 0 0111.8 5c2.02 0 3.25 1.4 3.25 2.9 0 2-1.83 2.55-2.43 3.12-.43.4-.47.75-.47 1.23a1 1 0 01-2 0c0-1 .16-1.82 1.1-2.7.69-.64 1.82-1.05 1.82-2.06z"
                    />
                  </svg>
                  {{ $t('onboarding.restartTour') }}
                </button>
              </div>

              <div class="border-t border-gray-100 py-1 dark:border-dark-700">
                <button
                  @click="handleLogout"
                  class="dropdown-item w-full text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20"
                >
                  <svg
                    class="h-4 w-4"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    stroke-width="1.5"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      d="M15.75 9V5.25A2.25 2.25 0 0013.5 3h-6a2.25 2.25 0 00-2.25 2.25v13.5A2.25 2.25 0 007.5 21h6a2.25 2.25 0 002.25-2.25V15M12 9l-3 3m0 0l3 3m-3-3h12.75"
                    />
                  </svg>
                  {{ t('nav.logout') }}
                </button>
              </div>
            </div>
          </transition>
        </div>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore, useAuthStore, useOnboardingStore } from '@/stores'
import { useAdminSettingsStore } from '@/stores/adminSettings'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import CreditAmount from '@/components/common/CreditAmount.vue'
import SubscriptionProgressMini from '@/components/common/SubscriptionProgressMini.vue'
import AnnouncementBell from '@/components/common/AnnouncementBell.vue'
import Icon from '@/components/icons/Icon.vue'
import AppBrand from './AppBrand.vue'

const router = useRouter()
const route = useRoute()
const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const adminSettingsStore = useAdminSettingsStore()
const onboardingStore = useOnboardingStore()

const user = computed(() => authStore.user)
const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const mobileOpen = computed(() => appStore.mobileOpen)
const isDark = ref(document.documentElement.classList.contains('dark'))
const dropdownOpen = ref(false)
const dropdownRef = ref<HTMLElement | null>(null)
const contactInfo = computed(() => appStore.contactInfo)
const avatarUrl = computed(() => user.value?.avatar_url?.trim() || '')
const availableBalance = computed(() => Number(user.value?.balance || 0))
const frozenBalance = computed(() => Number(user.value?.frozen_balance || 0))
const totalBalance = computed(() => availableBalance.value + frozenBalance.value)
const balanceAvailableText = computed(() => t('common.availableBalance') === 'common.availableBalance' ? '可用余额' : t('common.availableBalance'))
const balanceFrozenText = computed(() => t('common.frozenBalance') === 'common.frozenBalance' ? '冻结金额' : t('common.frozenBalance'))
const balanceTotalText = computed(() => t('common.totalBalance') === 'common.totalBalance' ? '总余额' : t('common.totalBalance'))

// 只在标准模式的管理员下显示新手引导按钮
const showOnboardingButton = computed(() => {
  return !authStore.isSimpleMode && user.value?.role === 'admin'
})

const userInitials = computed(() => {
  if (!user.value) return ''
  // Prefer username, fallback to email
  if (user.value.username) {
    return user.value.username.substring(0, 1).toUpperCase()
  }
  if (user.value.email) {
    // Get the part before @ and take its first character
    const localPart = user.value.email.split('@')[0]
    return localPart.substring(0, 1).toUpperCase()
  }
  return ''
})

const displayName = computed(() => {
  if (!user.value) return ''
  return user.value.username || user.value.email?.split('@')[0] || ''
})

const pageTitle = computed(() => {
  // For custom pages, use the menu item's label instead of generic "自定义页面"
  if (route.name === 'CustomPage') {
    const id = route.params.id as string
    const publicItems = appStore.cachedPublicSettings?.custom_menu_items ?? []
    const menuItem = publicItems.find((item) => item.id === id)
      ?? (authStore.isAdmin ? adminSettingsStore.customMenuItems.find((item) => item.id === id) : undefined)
    if (menuItem?.label) return menuItem.label
  }
  const titleKey = route.meta.titleKey as string
  if (titleKey) {
    return t(titleKey)
  }
  return (route.meta.title as string) || ''
})

function toggleMobileSidebar() {
  if (!mobileOpen.value) {
    appStore.setSidebarCollapsed(false)
  }
  appStore.toggleMobileSidebar()
}

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

function toggleDropdown() {
  dropdownOpen.value = !dropdownOpen.value
}

function closeDropdown() {
  dropdownOpen.value = false
}

async function handleLogout() {
  closeDropdown()
  try {
    await authStore.logout()
  } catch (error) {
    // Ignore logout errors - still redirect to login
    console.error('Logout error:', error)
  }
  await router.push('/login')
}

function handleReplayGuide() {
  closeDropdown()
  onboardingStore.replay()
}

function formatHeaderCredit(value: number) {
  if (!Number.isFinite(value)) return '0.00'
  return value.toFixed(2)
}

function handleClickOutside(event: MouseEvent) {
  if (dropdownRef.value && !dropdownRef.value.contains(event.target as Node)) {
    closeDropdown()
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<style scoped>
.app-header {
  color: rgb(17 24 39);
  font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif;
}

.brand-utility-icon {
  color: rgb(var(--luoxue-blue-rgb));
}

.header-account-avatar {
  background-color: #fce865;
  color: #1c1f23;
}

:global(html.dark .brand-utility-icon) {
  color: rgb(var(--luoxue-blue-light-rgb));
}

.topup-header-surface {
  border-width: 1px 1px 0;
  border-color: rgb(255 255 255 / 0.65) rgb(255 255 255 / 0.36) transparent;
  background: linear-gradient(rgb(248 251 255 / 0.32), rgb(235 242 252 / 0.1));
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 0.65),
    0 2px 8px -2px rgb(15 23 42 / 0.06),
    0 8px 20px -8px rgb(15 23 42 / 0.08);
  backdrop-filter: saturate(1.7) blur(36px);
  -webkit-backdrop-filter: saturate(1.7) blur(36px);
}

.topup-header-surface::before,
.topup-header-surface::after {
  position: absolute;
  pointer-events: none;
  content: '';
}

.topup-header-surface > div {
  transform: translateY(0.5px);
}

.topup-header-surface::before {
  inset: 0;
  z-index: -1;
  border-radius: 1rem;
  background:
    radial-gradient(60% 100% at 10% 0%, rgb(96 165 250 / 0.1) 0%, transparent 70%),
    radial-gradient(60% 100% at 90% 0%, rgb(167 139 250 / 0.1) 0%, transparent 70%);
}

.topup-header-surface::after {
  inset: 0 12% auto;
  height: 1px;
  background: linear-gradient(to right, transparent, rgb(255 255 255 / 0.85), transparent);
}

:global(html.dark .topup-header-surface) {
  border-color: rgb(255 255 255 / 0.12) rgb(255 255 255 / 0.09) transparent;
  background: linear-gradient(rgb(10 12 18 / 0.92), rgb(8 10 16 / 0.82));
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 0.08),
    0 2px 8px -2px rgb(0 0 0 / 0.18),
    0 8px 20px -8px rgb(0 0 0 / 0.28);
  backdrop-filter: saturate(1.6) blur(40px);
  -webkit-backdrop-filter: saturate(1.6) blur(40px);
}

:global(html.dark .topup-header-surface::before) {
  background:
    radial-gradient(60% 100% at 10% 0%, rgb(59 130 246 / 0.08) 0%, transparent 70%),
    radial-gradient(60% 100% at 90% 0%, rgb(139 92 246 / 0.07) 0%, transparent 70%);
}

:global(html.dark .topup-header-surface::after) {
  background: linear-gradient(to right, transparent, rgb(255 255 255 / 0.12), transparent);
}

.dropdown-enter-active,
.dropdown-leave-active {
  transition:
    opacity 0.18s ease-out,
    transform 0.18s ease-out;
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: scale(0.95) translateY(-4px);
}

.header-dropdown {
  border-radius: 0.875rem;
  box-shadow:
    0 18px 40px rgb(15 23 42 / 0.12),
    0 2px 8px rgb(15 23 42 / 0.06);
}

.header-dropdown .dropdown-item {
  min-height: 2.75rem;
  padding: 0.6875rem 0.75rem;
  font-size: 0.875rem;
  line-height: 1.25rem;
}

@media (min-width: 640px) {
  .header-dropdown .dropdown-item {
    min-height: 2.25rem;
    padding: 0.4375rem 0.75rem;
    font-size: 0.8125rem;
  }
}

@media (prefers-reduced-motion: reduce) {
  .dropdown-enter-active,
  .dropdown-leave-active {
    transition-duration: 0.01ms;
  }
}
</style>

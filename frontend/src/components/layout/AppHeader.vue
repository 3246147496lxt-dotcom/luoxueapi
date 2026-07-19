<template>
  <header
    data-testid="app-header"
    class="fixed inset-x-0 top-0 z-50 h-[81px] bg-transparent p-2"
  >
    <div
      data-testid="header-surface"
      class="flex h-[65px] w-full items-center justify-between rounded-2xl border border-white/80 bg-white/95 pr-2 shadow-[inset_0_1px_0_rgba(255,255,255,0.65),0_2px_8px_-2px_rgba(15,23,42,0.06),0_8px_20px_-8px_rgba(15,23,42,0.08)] backdrop-blur-xl dark:border-dark-700/80 dark:bg-dark-900/95 dark:shadow-[inset_0_1px_0_rgba(255,255,255,0.04),0_2px_8px_-2px_rgba(0,0,0,0.25),0_8px_20px_-8px_rgba(0,0,0,0.32)]"
    >
      <!-- Left: brand, sidebar controls, and page context -->
      <div class="flex min-w-0 flex-1 items-center gap-0 lg:gap-2">
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
          class="flex w-10 min-w-0 flex-shrink-0 items-center justify-center transition-[width] duration-300 ease-out motion-reduce:transition-none lg:justify-start"
          :class="
            sidebarCollapsed
              ? 'md:w-[68px]'
              : 'md:w-[196px] lg:w-[184px] min-[1025px]:w-[196px] min-[1281px]:w-[208px]'
          "
        >
          <router-link
            :to="homePath"
            data-testid="header-brand"
            class="flex h-10 w-full min-w-0 items-center justify-center gap-2 rounded-xl transition-colors hover:bg-gray-100/80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 md:h-16 dark:hover:bg-dark-800"
            :class="
              sidebarCollapsed
                ? 'lg:w-[60px]'
                : 'lg:w-44 min-[1025px]:w-[188px] min-[1281px]:w-[200px]'
            "
            :aria-label="siteName"
          >
            <span class="flex h-10 w-10 flex-shrink-0 items-center justify-center overflow-hidden rounded-[10px] bg-white ring-1 ring-gray-200/80 md:h-12 md:w-12 dark:bg-dark-800 dark:ring-dark-700">
              <img :src="siteLogo || '/logo.png'" alt="" class="h-full w-full object-contain">
            </span>
            <span
              class="hidden min-w-0 truncate text-[22px] font-bold tracking-tight text-gray-950 md:block dark:text-white"
              :class="sidebarCollapsed ? 'md:hidden' : 'md:block'"
            >
              {{ siteName }}
            </span>
          </router-link>
        </div>

        <button
          type="button"
          data-testid="header-sidebar-toggle"
          class="relative hidden h-8 w-8 flex-shrink-0 items-center justify-center rounded-[10px] text-gray-500 transition-colors after:absolute after:-inset-1.5 after:content-[''] hover:bg-gray-100 hover:text-gray-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 lg:-ml-px lg:flex dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white dark:focus-visible:ring-offset-dark-900"
          :title="sidebarCollapsed ? t('nav.expand') : t('nav.collapse')"
          :aria-label="sidebarCollapsed ? t('nav.expand') : t('nav.collapse')"
          aria-controls="app-sidebar"
          :aria-expanded="!sidebarCollapsed"
          @click="toggleSidebar"
        >
          <SidebarCollapseIcon
            data-testid="header-sidebar-toggle-icon"
            class="h-5 w-5 transition-transform duration-200 motion-reduce:transition-none"
            :class="{ 'rotate-180': sidebarCollapsed }"
          />
        </button>

        <div class="hidden min-w-0 lg:block">
          <h1 class="truncate text-base font-semibold tracking-tight text-gray-950 dark:text-white">
            {{ pageTitle }}
          </h1>
        </div>
      </div>

      <!-- Right: utility icons + supporting status + account pill -->
      <div class="flex flex-shrink-0 items-center gap-2 md:gap-3">
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
            class="relative flex h-8 w-8 items-center justify-center rounded-full text-gray-600 transition-colors duration-200 after:absolute after:-inset-1.5 after:content-[''] hover:bg-gray-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 dark:text-gray-300 dark:hover:bg-dark-800 dark:focus-visible:ring-offset-dark-900"
            :title="isDark ? t('nav.lightMode') : t('nav.darkMode')"
            :aria-label="isDark ? t('nav.lightMode') : t('nav.darkMode')"
            @click="toggleTheme"
          >
            <Icon
              :name="isDark ? 'sun' : 'moon'"
              size="sm"
              class="!h-[18px] !w-[18px]"
              :class="{ 'text-amber-500': isDark }"
              aria-hidden="true"
            />
          </button>

          <!-- Language Switcher -->
          <LocaleSwitcher compact />
        </div>

        <!-- Docs Link -->
        <a
          v-if="docUrl"
          :href="docUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="hidden h-9 items-center gap-1.5 rounded-lg border border-transparent px-2.5 text-sm font-medium text-gray-600 transition-colors hover:border-gray-200 hover:bg-gray-50 hover:text-gray-950 2xl:flex dark:text-dark-300 dark:hover:border-dark-700 dark:hover:bg-dark-800 dark:hover:text-white"
        >
          <Icon name="book" size="sm" />
          <span>{{ t('nav.docs') }}</span>
        </a>

        <!-- Subscription Progress (for users with active subscriptions) -->
        <div v-if="user" class="hidden 2xl:block">
          <SubscriptionProgressMini />
        </div>

        <!-- Balance Display -->
        <div
          v-if="user"
          class="group relative hidden h-9 items-center gap-2 rounded-lg border border-primary-100 bg-primary-50/80 px-3 dark:border-primary-900/50 dark:bg-primary-900/20 2xl:flex"
        >
          <svg
            class="h-4 w-4 text-primary-600 dark:text-primary-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="1.5"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M2.25 18.75a60.07 60.07 0 0115.797 2.101c.727.198 1.453-.342 1.453-1.096V18.75M3.75 4.5v.75A.75.75 0 013 6h-.75m0 0v-.375c0-.621.504-1.125 1.125-1.125H20.25M2.25 6v9m18-10.5v.75c0 .414.336.75.75.75h.75m-1.5-1.5h.375c.621 0 1.125.504 1.125 1.125v9.75c0 .621-.504 1.125-1.125 1.125h-.375m1.5-1.5H21a.75.75 0 00-.75.75v.75m0 0H3.75m0 0h-.375a1.125 1.125 0 01-1.125-1.125V15m1.5 1.5v-.75A.75.75 0 003 15h-.75M15 10.5a3 3 0 11-6 0 3 3 0 016 0zm3 0h.008v.008H18V10.5zm-12 0h.008v.008H6V10.5z"
            />
          </svg>
          <span class="text-sm font-semibold text-primary-700 dark:text-primary-300">
            {{ formatHeaderMoney(availableBalance) }}
          </span>
          <span
            v-if="frozenBalance > 0"
            class="rounded-full bg-amber-100 px-1.5 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-900/40 dark:text-amber-200"
          >
            {{ balanceFrozenLabel }}
          </span>
          <div
            class="pointer-events-none absolute right-0 top-full mt-2 hidden w-56 rounded-xl border border-gray-200 bg-white p-3 text-xs shadow-[0_12px_32px_rgba(15,23,42,0.10)] group-hover:block dark:border-dark-700 dark:bg-dark-800 dark:shadow-black/30"
          >
            <div class="flex items-center justify-between">
              <span class="text-gray-500 dark:text-dark-400">{{ balanceAvailableText }}</span>
              <span class="font-medium text-gray-900 dark:text-white">{{ formatHeaderMoney(availableBalance) }}</span>
            </div>
            <div class="mt-2 flex items-center justify-between">
              <span class="text-gray-500 dark:text-dark-400">{{ balanceFrozenText }}</span>
              <span class="font-medium text-amber-700 dark:text-amber-200">{{ formatHeaderMoney(frozenBalance) }}</span>
            </div>
            <div class="mt-2 border-t border-gray-100 pt-2 dark:border-dark-700">
              <div class="flex items-center justify-between">
                <span class="text-gray-500 dark:text-dark-400">{{ balanceTotalText }}</span>
                <span class="font-semibold text-gray-900 dark:text-white">{{ formatHeaderMoney(totalBalance) }}</span>
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
            class="relative flex h-8 items-center gap-1.5 rounded-full bg-gray-950/[0.04] p-1 transition-colors after:absolute after:-inset-y-1.5 after:-inset-x-1 after:content-[''] hover:bg-gray-950/[0.08] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 dark:bg-white/[0.06] dark:hover:bg-white/[0.10] dark:focus-visible:ring-offset-dark-900"
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
              class="flex h-6 w-6 flex-shrink-0 items-center justify-center overflow-hidden rounded-full bg-primary-600 text-xs font-semibold text-white ring-1 ring-primary-700/20 dark:bg-primary-500 dark:ring-white/10"
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
                  {{ formatHeaderMoney(availableBalance) }}
                </div>
                <div v-if="frozenBalance > 0" class="mt-1 text-xs text-amber-600 dark:text-amber-300">
                  {{ balanceFrozenText }} {{ formatHeaderMoney(frozenBalance) }}
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
import SubscriptionProgressMini from '@/components/common/SubscriptionProgressMini.vue'
import AnnouncementBell from '@/components/common/AnnouncementBell.vue'
import Icon from '@/components/icons/Icon.vue'
import SidebarCollapseIcon from '@/components/icons/SidebarCollapseIcon.vue'
import { sanitizeUrl } from '@/utils/url'

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
const docUrl = computed(() => sanitizeUrl(appStore.docUrl))
const siteName = computed(() => appStore.siteName || '落雪API')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', {
  allowRelative: true,
  allowDataUrl: true,
}))
const homePath = computed(() => (authStore.isAdmin ? '/admin/dashboard' : '/dashboard'))
const avatarUrl = computed(() => user.value?.avatar_url?.trim() || '')
const availableBalance = computed(() => Number(user.value?.balance || 0))
const frozenBalance = computed(() => Number(user.value?.frozen_balance || 0))
const totalBalance = computed(() => availableBalance.value + frozenBalance.value)
const balanceAvailableText = computed(() => t('common.availableBalance') === 'common.availableBalance' ? '可用余额' : t('common.availableBalance'))
const balanceFrozenText = computed(() => t('common.frozenBalance') === 'common.frozenBalance' ? '冻结金额' : t('common.frozenBalance'))
const balanceTotalText = computed(() => t('common.totalBalance') === 'common.totalBalance' ? '总余额' : t('common.totalBalance'))
const balanceFrozenLabel = computed(() => `${balanceFrozenText.value} ${formatHeaderMoney(frozenBalance.value)}`)

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

function toggleSidebar() {
  appStore.toggleSidebar()
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

function formatHeaderMoney(value: number) {
  if (!Number.isFinite(value)) return '$0.00'
  return `$${value.toFixed(2)}`
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

<template>
  <WorkspaceSidebarOverlayLayer
    :active="personalNarrowViewport"
    :open="narrowSidebarOpen"
    :label="t('nav.workMode')"
    return-focus-id="workspace-sidebar-overlay-trigger"
    @close="closeNarrowSidebar"
  >
    <WorkspaceSidebarFrame
      id="app-sidebar"
      :label="t('nav.workMode')"
      :collapsed="sidebarCollapsed"
      :mobile="mobileViewport"
      :overlay="personalNarrowViewport"
      :placement="personalNarrowViewport ? 'flow' : 'fixed'"
      surface="work"
      :content-mode="isAdminWorkspace ? 'admin' : 'workspace'"
      :aria-hidden="mobileNavigationHidden ? 'true' : undefined"
      :inert="mobileNavigationHidden"
      class="app-sidebar"
      :class="{
        'sidebar-mobile-hidden': mobileViewport && !mobileOpen,
        'sidebar--admin-workspace': isAdminWorkspace,
        'sidebar--personal-work': isPersonalWorkWorkspace,
      }"
    >
    <template #header>
      <WorkspaceSidebarHeader
        ref="sidebarHeaderRef"
        controls="app-sidebar"
        data-testid="sidebar-brand-row"
        :collapsed="sidebarCollapsed"
        :mobile="mobileViewport"
        :overlay="personalNarrowViewport"
        :show-search="false"
        :show-close="personalNarrowViewport"
        :search-expanded="sidebarSearchOpen"
        :search-controls="sidebarSearchPanelId"
        :search-label="t('common.search')"
        :collapse-label="t('nav.collapse')"
        :expand-label="t('nav.expand')"
        :close-label="t('nav.closeNavigation')"
        :collapsed-logo-src="appStore.siteLogo"
        toggle-test-id="sidebar-collapse-toggle"
        toggle-icon-test-id="sidebar-collapse-toggle-icon"
        @search="toggleSidebarSearch"
        @toggle="toggleSidebar"
        @close="closeNarrowSidebar"
      >
        <template #brand>
          <WorkspaceSidebarBrand
            v-if="isPersonalWorkWorkspace"
            home-path="/dashboard"
          />
          <AppBrand
            v-else
            class="workspace-sidebar-brand workspace-sidebar-brand--work"
            placement="sidebar"
            :collapsed="false"
            home-path="/admin/dashboard"
          />
        </template>
      </WorkspaceSidebarHeader>
    </template>

    <form
      v-if="sidebarSearchOpen && !sidebarCollapsed"
      :id="sidebarSearchPanelId"
      class="sidebar-search"
      role="search"
      @submit.prevent
    >
      <Icon name="search" size="sm" aria-hidden="true" />
      <input
        ref="sidebarSearchInputRef"
        v-model="sidebarSearchQuery"
        type="search"
        :placeholder="t('common.searchPlaceholder')"
        :aria-label="t('common.search')"
        @keydown.esc.prevent.stop="closeSidebarSearch"
      />
    </form>

    <!-- Navigation -->
    <nav ref="sidebarNavRef" class="workspace-sidebar-navigation scrollbar-hide">
      <!-- Each frontend entry renders only the navigation it owns. -->
      <template v-if="isAdminWorkspace">
        <div
          v-for="section in displayedAdminNavSections"
          :key="section.id"
          class="sidebar-section"
          :data-testid="`sidebar-admin-${section.id}-section`"
          :role="section.label ? 'group' : undefined"
          :aria-label="section.label"
        >
          <button
            v-if="section.label && !sidebarCollapsed"
            type="button"
            class="sidebar-section-title sidebar-section-toggle"
            :aria-expanded="isAdminSectionExpanded(section.id)"
            :aria-controls="adminSectionPanelId(section.id)"
            :aria-disabled="activeAdminSectionId === section.id ? 'true' : undefined"
            @click="toggleAdminSection(section.id)"
          >
            <span class="sidebar-section-title-text">{{ section.label }}</span>
            <ChevronDownIcon
              class="sidebar-section-chevron h-3.5 w-3.5 flex-shrink-0"
              :class="{ 'rotate-180': isAdminSectionExpanded(section.id) }"
              aria-hidden="true"
            />
          </button>

          <div
            v-show="sidebarCollapsed || isAdminSectionExpanded(section.id)"
            :id="adminSectionPanelId(section.id)"
            class="sidebar-section-items"
            :data-testid="`sidebar-section-panel-${section.id}`"
          >
            <template v-for="item in section.items" :key="item.path">
              <!-- Collapsible group (has children) -->
              <template v-if="item.children?.length">
                <button
                  type="button"
                  class="sidebar-link mb-1 w-full"
                  :class="{
                    'sidebar-link-active': isGroupActive(item) && (sidebarCollapsed || !isGroupExpanded(item)),
                    'sidebar-link-collapsed': sidebarCollapsed
                  }"
                  :title="sidebarCollapsed ? item.label : undefined"
                  :aria-expanded="!sidebarCollapsed && isGroupExpanded(item)"
                  :aria-controls="groupPanelId(item)"
                  @click="handleGroupClick(item)"
                >
                  <component :is="item.icon" class="sidebar-nav-icon h-5 w-5 flex-shrink-0" />
                  <span
                    class="sidebar-label sidebar-label-flex"
                    :class="{ 'sidebar-label-collapsed': sidebarCollapsed }"
                    :aria-hidden="sidebarCollapsed ? 'true' : 'false'"
                  >
                    <span class="min-w-0 truncate">{{ item.label }}</span>
                    <ChevronDownIcon
                      class="h-4 w-4 flex-shrink-0 transition-transform duration-200"
                      :class="isGroupExpanded(item) ? 'rotate-180' : ''"
                    />
                  </span>
                </button>
                <!-- Children -->
                <div
                  v-if="!sidebarCollapsed && isGroupExpanded(item)"
                  :id="groupPanelId(item)"
                  class="sidebar-child-group mb-1 ml-3 border-l pl-2"
                  role="group"
                  :aria-label="item.label"
                >
                  <router-link
                    v-for="child in item.children"
                    :key="child.path"
                    :to="child.path"
                    class="sidebar-link mb-0.5 py-1.5 text-sm"
                    :class="{ 'sidebar-link-active': isChildActive(item, child) }"
                    @click="handleMenuItemClick(child.path)"
                  >
                    <component
                      :is="child.icon"
                      class="sidebar-nav-icon sidebar-nav-icon--child h-4 w-4 flex-shrink-0"
                    />
                    <span>{{ child.label }}</span>
                  </router-link>
                </div>
              </template>
              <!-- Normal item (no children) -->
              <template v-else>
                <a
                  v-if="item.href"
                  :href="item.href"
                  class="sidebar-link mb-1"
                  :class="{ 'sidebar-link-collapsed': sidebarCollapsed }"
                  :title="sidebarCollapsed ? item.label : undefined"
                  target="_blank"
                  rel="noopener noreferrer"
                  @click="handleMenuItemClick(item.path)"
                >
                  <component :is="item.icon" class="sidebar-nav-icon h-5 w-5 flex-shrink-0" />
                  <span class="sidebar-label" :class="{ 'sidebar-label-collapsed': sidebarCollapsed }" :aria-hidden="sidebarCollapsed ? 'true' : 'false'">{{ item.label }}</span>
                </a>
                <router-link
                  v-else
                  :to="item.path"
                  class="sidebar-link mb-1"
                  :class="{ 'sidebar-link-active': isActive(item.path, item.activePaths), 'sidebar-link-collapsed': sidebarCollapsed }"
                  :title="sidebarCollapsed ? item.label : undefined"
                  :id="
                    item.path === '/admin/accounts'
                      ? 'sidebar-channel-manage'
                      : item.path === '/admin/groups'
                        ? 'sidebar-group-manage'
                        : item.path === '/admin/redeem'
                          ? 'sidebar-wallet'
                          : undefined
                  "
                  @click="handleMenuItemClick(item.path)"
                >
                  <span
                    v-if="item.iconSvg"
                    class="sidebar-nav-icon h-5 w-5 flex-shrink-0 sidebar-svg-icon"
                    :class="{ 'sidebar-api-key-icon': item.path === '/keys' }"
                    v-html="sanitizeSvg(item.iconSvg)"
                  ></span>
                  <component v-else :is="item.icon" class="sidebar-nav-icon h-5 w-5 flex-shrink-0" />
                  <span class="sidebar-label flex-1" :class="{ 'sidebar-label-collapsed': sidebarCollapsed }" :aria-hidden="sidebarCollapsed ? 'true' : 'false'">{{ item.label }}</span>
                  <Icon
                    v-if="item.trailingIcon && !sidebarCollapsed"
                    :name="item.trailingIcon"
                    size="sm"
                    :stroke-width="1.7"
                    class="sidebar-nav-trailing-icon flex-shrink-0"
                    data-testid="sidebar-nav-trailing-icon"
                    aria-hidden="true"
                  />
                </router-link>
              </template>
            </template>
          </div>
        </div>
      </template>

      <!-- Regular users and administrators on non-admin routes share the personal workspace. -->
      <template v-else-if="!appStore.backendModeEnabled">
        <div
          v-for="section in displayedUserNavSections"
          :key="section.id"
          class="sidebar-section"
          :data-testid="`sidebar-user-${section.id}-section`"
          :role="section.label ? 'group' : undefined"
          :aria-label="section.label"
        >
          <div
            v-if="section.label"
            class="sidebar-section-title"
            :class="{ 'sidebar-section-title-collapsed': sidebarCollapsed }"
            aria-hidden="true"
          >
            <span
              class="sidebar-section-title-text"
              :class="{ 'sidebar-section-title-text-collapsed': sidebarCollapsed }"
            >
              {{ section.label }}
            </span>
          </div>

          <template v-for="item in section.items" :key="item.path">
            <a
              v-if="item.href"
              :href="item.href"
              class="sidebar-link mb-1"
              :class="{ 'sidebar-link-collapsed': sidebarCollapsed }"
              :title="sidebarCollapsed ? item.label : undefined"
              target="_blank"
              rel="noopener noreferrer"
              @click="handleMenuItemClick(item.path)"
            >
              <component :is="item.icon" class="sidebar-nav-icon h-5 w-5 flex-shrink-0" />
              <span class="sidebar-label" :class="{ 'sidebar-label-collapsed': sidebarCollapsed }" :aria-hidden="sidebarCollapsed ? 'true' : 'false'">{{ item.label }}</span>
            </a>
            <router-link
              v-else
              :to="item.path"
              class="sidebar-link mb-1"
              :class="{ 'sidebar-link-active': isActive(item.path, item.activePaths), 'sidebar-link-collapsed': sidebarCollapsed }"
              :title="sidebarCollapsed ? item.label : undefined"
              :data-tour="item.path === '/keys' ? 'sidebar-my-keys' : undefined"
              @click="handleMenuItemClick(item.path)"
            >
              <span
                v-if="item.iconSvg"
                class="sidebar-nav-icon h-5 w-5 flex-shrink-0 sidebar-svg-icon"
                :class="{ 'sidebar-api-key-icon': item.path === '/keys' }"
                v-html="sanitizeSvg(item.iconSvg)"
              ></span>
              <component v-else :is="item.icon" class="sidebar-nav-icon h-5 w-5 flex-shrink-0" />
              <span class="sidebar-label flex-1" :class="{ 'sidebar-label-collapsed': sidebarCollapsed }" :aria-hidden="sidebarCollapsed ? 'true' : 'false'">{{ item.label }}</span>
              <Icon
                v-if="item.trailingIcon && !sidebarCollapsed"
                :name="item.trailingIcon"
                size="sm"
                :stroke-width="1.7"
                class="sidebar-nav-trailing-icon flex-shrink-0"
                data-testid="sidebar-nav-trailing-icon"
                aria-hidden="true"
              />
            </router-link>
          </template>
        </div>
      </template>

      <div
        v-if="sidebarSearchQuery && !hasSidebarSearchResults"
        class="sidebar-search-empty"
        role="status"
      >
        {{ t('common.noData') }}
      </div>

      <div
        v-if="!isAdminWorkspace && displayedSidebarSupportLinks.length"
        class="sidebar-section sidebar-support-section sidebar-support-section--bottom"
        data-testid="sidebar-support-section"
      >
        <a
          v-for="item in displayedSidebarSupportLinks"
          :key="item.id"
          :href="item.href"
          class="sidebar-link sidebar-support-link mb-1"
          :class="{
            'sidebar-link-collapsed': sidebarCollapsed,
            'sidebar-web-chat-link': item.id === 'webChat',
          }"
          :title="sidebarCollapsed ? item.label : undefined"
          :data-testid="
            item.id === 'webChat'
              ? 'sidebar-web-chat'
              : item.id === 'documentation'
              ? 'sidebar-docs-tutorial'
              : item.id === 'contact'
                ? 'sidebar-contact-us'
                : undefined
          "
          target="_blank"
          rel="noopener noreferrer"
          @click="handleMenuItemClick(item.href)"
        >
          <Icon :name="item.icon" size="md" class="sidebar-nav-icon flex-shrink-0" />
          <span
            class="sidebar-label min-w-0 flex-1 truncate"
            :class="{ 'sidebar-label-collapsed': sidebarCollapsed }"
            :aria-hidden="sidebarCollapsed ? 'true' : 'false'"
          >
            {{ item.label }}
          </span>
          <Icon
            v-if="!sidebarCollapsed"
            name="destinationArrowUpRight"
            size="sm"
            :stroke-width="1.7"
            class="sidebar-nav-trailing-icon flex-shrink-0"
            data-testid="sidebar-nav-trailing-icon"
            aria-hidden="true"
          />
        </a>
      </div>
    </nav>

    <template #footer>
      <UserAccountCard context="work" :collapsed="sidebarCollapsed" />
    </template>
    </WorkspaceSidebarFrame>
  </WorkspaceSidebarOverlayLayer>

  <!-- Mobile Overlay -->
  <transition name="fade">
    <button
      v-if="mobileViewport && mobileOpen"
      type="button"
      class="app-sidebar-backdrop fixed inset-0 z-30 border-0 p-0 backdrop-blur-[1px]"
      :aria-label="t('nav.closeNavigation')"
      @click="closeMobile"
    ></button>
  </transition>
</template>

<script setup lang="ts">
import { computed, h, nextTick, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingStore } from '@/stores/onboarding'
import { resolveDocumentationUrl } from '@/utils/documentationUrl'
import { sanitizeSvg } from '@/utils/sanitize'
import { resolveSupportContactUrl } from '@/utils/supportUrl'
import {
  getShellDestinationSpecs,
  selectVisibleShellDestinations,
  toShellCapabilityState,
  type ShellDestinationSpec,
} from '@/navigation/shellDestinations'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'
import { Icon } from '@/components/icons'
import AppBrand from './AppBrand.vue'
import UserAccountCard from './UserAccountCard.vue'
import WorkspaceSidebarBrand from './WorkspaceSidebarBrand.vue'
import WorkspaceSidebarFrame from './WorkspaceSidebarFrame.vue'
import WorkspaceSidebarHeader from './WorkspaceSidebarHeader.vue'
import WorkspaceSidebarOverlayLayer from './WorkspaceSidebarOverlayLayer.vue'
import { useWorkspaceSidebarCollapse } from './useWorkspaceSidebarCollapse'
import { useWorkspaceResponsiveState } from './workspaceResponsive'
import { buildUserNavigation } from './sidebar/userNavigation'
import { loadAdminNavigation } from './sidebar/adminNavigationLoader'
import type {
  AdminNavigationDefinition,
  AdminNavigationIcons,
  AdminNavSectionId,
  NavItem,
  SidebarSupportIcon,
  SidebarSupportLink,
  UserNavigationIcons,
} from './sidebar/types'

const props = defineProps<{
  adminNavigation?: AdminNavigationDefinition | null
}>()

/**
 * Navigation route ownership moved to the workspace modules. These references
 * keep older source-level contract tests discoverable while they migrate:
 * FeatureFlags.skillMarketplace
 * path: '/admin/skills'
 * path: '/admin/documentation'
 * label: t('nav.documentationManagement')
 */

const { t } = useI18n()

const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()
const onboardingStore = useOnboardingStore()
type AdminSettingsStore = ReturnType<
  (typeof import('@/stores/adminSettings'))['useAdminSettingsStore']
>
const adminSettingsStore = shallowRef<AdminSettingsStore | null>(null)
let adminSettingsRequest: Promise<AdminSettingsStore> | null = null

function ensureAdminSettings(): Promise<AdminSettingsStore> {
  if (adminSettingsStore.value) return Promise.resolve(adminSettingsStore.value)
  if (adminSettingsRequest) return adminSettingsRequest

  adminSettingsRequest = import('@/stores/adminSettings')
    .then(({ useAdminSettingsStore }) => {
      const store = useAdminSettingsStore()
      adminSettingsStore.value = store
      return store
    })
    .finally(() => {
      adminSettingsRequest = null
    })
  return adminSettingsRequest
}
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()
const isAdmin = computed(() => authStore.isAdmin)
const isAdminWorkspace = computed(() => isAdmin.value && route.path.startsWith('/admin'))
const isPersonalWorkWorkspace = computed(() => !isAdminWorkspace.value)
const {
  mobileDrawer: mobileViewport,
  narrowSidebar: narrowViewport,
} = useWorkspaceResponsiveState()
const mobileOpen = computed(() => appStore.mobileOpen)
const personalNarrowViewport = computed(() => (
  isPersonalWorkWorkspace.value && narrowViewport.value
))
const narrowSidebarOpen = computed(() => appStore.workspaceNarrowSidebarOpen)
const documentationUrl = computed(() => resolveDocumentationUrl(
  appStore.cachedPublicSettings?.doc_url || appStore.docUrl,
))
const contactUrl = computed(() => resolveSupportContactUrl(
  appStore.cachedPublicSettings?.contact_info || appStore.contactInfo,
  appStore.cachedPublicSettings?.doc_url || appStore.docUrl,
))
const sidebarNavRef = ref<HTMLElement | null>(null)
const sidebarSearchInputRef = ref<HTMLInputElement | null>(null)
const sidebarSearchOpen = ref(false)
const sidebarSearchQuery = ref('')
const sidebarSearchPanelId = 'app-sidebar-navigation-search'
const {
  collapsed: sidebarCollapsed,
  headerRef: sidebarHeaderRef,
  expand: expandSidebar,
  toggle: toggleSidebar,
} = useWorkspaceSidebarCollapse({
  mobile: mobileViewport,
  overlay: personalNarrowViewport,
  beforeCollapse: () => {
    sidebarSearchOpen.value = false
    sidebarSearchQuery.value = ''
  },
})
const mobileNavigationHidden = computed(() => mobileViewport.value && !mobileOpen.value)

// Track which parent nav groups are expanded
const expandedGroups = ref<Set<string>>(new Set())

// SVG Icon Components
const DashboardIcon = {
  render: () =>
    h(
      'svg',
      { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' },
      [
        h('path', {
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          d: 'M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z'
        })
      ]
    )
}

const BatchImageIcon = {
  render: () =>
    h(
      'svg',
      { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' },
      [
        h('path', {
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          d: 'M6.827 6.175A2.31 2.31 0 015.186 7.23c-.38.054-.757.112-1.134.175C2.999 7.58 2.25 8.507 2.25 9.574V18a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9.574c0-1.067-.75-1.994-1.802-2.169a47.865 47.865 0 00-1.134-.175 2.31 2.31 0 01-1.64-1.055l-.822-1.316a2.25 2.25 0 00-1.906-1.059H9.554a2.25 2.25 0 00-1.906 1.059l-.821 1.316z'
        }),
        h('path', {
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          d: 'M16.5 12.75a4.5 4.5 0 11-9 0 4.5 4.5 0 019 0zM18.75 10.5h.008v.008h-.008V10.5z'
        })
      ]
    )
}

const ChartIcon = {
  render: () =>
    h(
      'svg',
      { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' },
      [
        h('path', {
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          d: 'M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 013 19.875v-6.75zM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V8.625zM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V4.125z'
        })
      ]
    )
}

const OpsChartIcon = {
  render: () => h(Icon, { name: 'chartNoAxesColumn', strokeWidth: 1.7 })
}

const UsageChartIcon = {
  render: () => h(Icon, { name: 'lucideBarChartBig', strokeWidth: 1.7 })
}

const GiftIcon = {
  render: () =>
    h(
      'svg',
      { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' },
      [
        h('path', {
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          d: 'M21 11.25v8.25a1.5 1.5 0 01-1.5 1.5H5.25a1.5 1.5 0 01-1.5-1.5v-8.25M12 4.875A2.625 2.625 0 109.375 7.5H12m0-2.625V7.5m0-2.625A2.625 2.625 0 1114.625 7.5H12m0 0V21m-8.625-9.75h18c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125h-18c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z'
        })
      ]
    )
}

const UserIcon = {
  render: () =>
    h(
      'svg',
      { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' },
      [
        h('path', {
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          d: 'M15.75 6a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0zM4.501 20.118a7.5 7.5 0 0114.998 0A17.933 17.933 0 0112 21.75c-2.676 0-5.216-.584-7.499-1.632z'
        })
      ]
    )
}

const UsersIcon = {
  render: () =>
    h(
      'svg',
      { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' },
      [
        h('path', {
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          d: 'M15 19.128a9.38 9.38 0 002.625.372 9.337 9.337 0 004.121-.952 4.125 4.125 0 00-7.533-2.493M15 19.128v-.003c0-1.113-.285-2.16-.786-3.07M15 19.128v.106A12.318 12.318 0 018.624 21c-2.331 0-4.512-.645-6.374-1.766l-.001-.109a6.375 6.375 0 0111.964-3.07M12 6.375a3.375 3.375 0 11-6.75 0 3.375 3.375 0 016.75 0zm8.25 2.25a2.625 2.625 0 11-5.25 0 2.625 2.625 0 015.25 0z'
        })
      ]
    )
}

const FolderIcon = {
  render: () =>
    h(
      'svg',
      { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' },
      [
        h('path', {
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          d: 'M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z'
        })
      ]
    )
}

const ChannelIcon = {
  render: () =>
    h(
      'svg',
      { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' },
      [
        h('path', {
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          d: 'M6.429 9.75L2.25 12l4.179 2.25m0-4.5l5.571 3 5.571-3m-11.142 0L2.25 7.5 12 2.25l9.75 5.25-4.179 2.25m0 0l4.179 2.25L12 17.25 2.25 12m15.321-2.25l4.179 2.25L12 17.25l-9.75-5.25'
        })
      ]
    )
}

const CreditCardIcon = {
  render: () =>
    h(
      'svg',
      { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' },
      [
        h('path', {
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          d: 'M2.25 8.25h19.5M2.25 9h19.5m-16.5 5.25h6m-6 2.25h3m-3.75 3h15a2.25 2.25 0 002.25-2.25V6.75A2.25 2.25 0 0019.5 4.5h-15a2.25 2.25 0 00-2.25 2.25v10.5A2.25 2.25 0 004.5 19.5z'
        })
      ]
    )
}

const RechargeSubscriptionIcon = {
  render: () =>
    h(
      'svg',
      { fill: 'currentColor', viewBox: '0 0 1024 1024' },
      [
        h('path', {
          d: 'M512 992C247.3 992 32 776.7 32 512S247.3 32 512 32s480 215.3 480 480c0 84.4-22.2 167.4-64.2 240-8.9 15.3-28.4 20.6-43.7 11.7-15.3-8.8-20.5-28.4-11.7-43.7 36.4-62.9 55.6-134.8 55.6-208 0-229.4-186.6-416-416-416S96 282.6 96 512s186.6 416 416 416c17.7 0 32 14.3 32 32s-14.3 32-32 32z'
        }),
        h('path', {
          d: 'M640 512H384c-17.7 0-32-14.3-32-32s14.3-32 32-32h256c17.7 0 32 14.3 32 32s-14.3 32-32 32zM640 640H384c-17.7 0-32-14.3-32-32s14.3-32 32-32h256c17.7 0 32 14.3 32 32s-14.3 32-32 32z'
        }),
        h('path', {
          d: 'M512 480c-8.2 0-16.4-3.1-22.6-9.4l-128-128c-12.5-12.5-12.5-32.8 0-45.3s32.8-12.5 45.3 0l128 128c12.5 12.5 12.5 32.8 0 45.3-6.3 6.3-14.5 9.4-22.7 9.4z'
        }),
        h('path', {
          d: 'M512 480c-8.2 0-16.4-3.1-22.6-9.4-12.5-12.5-12.5-32.8 0-45.3l128-128c12.5-12.5 32.8-12.5 45.3 0s12.5 32.8 0 45.3l-128 128c-6.3 6.3-14.5 9.4-22.7 9.4z'
        }),
        h('path', {
          d: 'M512 736c-17.7 0-32-14.3-32-32V448c0-17.7 14.3-32 32-32s32 14.3 32 32v256c0 17.7-14.3 32-32 32zM896 992H512c-17.7 0-32-14.3-32-32s14.3-32 32-32h306.8l-73.4-73.4c-12.5-12.5-12.5-32.8 0-45.3s32.8-12.5 45.3 0l128 128c9.2 9.2 11.9 22.9 6.9 34.9S908.9 992 896 992z'
        })
      ]
    )
}

const ServerIcon = {
  render: () =>
    h(
      'svg',
      { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' },
      [
        h('path', {
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          d: 'M5.25 14.25h13.5m-13.5 0a3 3 0 01-3-3m3 3a3 3 0 100 6h13.5a3 3 0 100-6m-16.5-3a3 3 0 013-3h13.5a3 3 0 013 3m-19.5 0a4.5 4.5 0 01.9-2.7L5.737 5.1a3.375 3.375 0 012.7-1.35h7.126c1.062 0 2.062.5 2.7 1.35l2.587 3.45a4.5 4.5 0 01.9 2.7m0 0a3 3 0 01-3 3m0 3h.008v.008h-.008v-.008zm0-6h.008v.008h-.008v-.008zm-3 6h.008v.008h-.008v-.008zm0-6h.008v.008h-.008v-.008z'
        })
      ]
    )
}

const TicketIcon = {
  render: () =>
    h(
      'svg',
      { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' },
      [
        h('path', {
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          d: 'M16.5 6v.75m0 3v.75m0 3v.75m0 3V18m-9-5.25h5.25M7.5 15h3M3.375 5.25c-.621 0-1.125.504-1.125 1.125v3.026a2.999 2.999 0 010 5.198v3.026c0 .621.504 1.125 1.125 1.125h17.25c.621 0 1.125-.504 1.125-1.125v-3.026a2.999 2.999 0 010-5.198V6.375c0-.621-.504-1.125-1.125-1.125H3.375z'
        })
      ]
    )
}

const CogIcon = {
  render: () =>
    h(
      'svg',
      { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' },
      [
        h('path', {
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          d: 'M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.324.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 011.37.49l1.296 2.247a1.125 1.125 0 01-.26 1.431l-1.003.827c-.293.24-.438.613-.431.992a6.759 6.759 0 010 .255c-.007.378.138.75.43.99l1.005.828c.424.35.534.954.26 1.43l-1.298 2.247a1.125 1.125 0 01-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.57 6.57 0 01-.22.128c-.331.183-.581.495-.644.869l-.213 1.28c-.09.543-.56.941-1.11.941h-2.594c-.55 0-1.02-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 01-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 01-1.369-.49l-1.297-2.247a1.125 1.125 0 01.26-1.431l1.004-.827c.292-.24.437-.613.43-.992a6.932 6.932 0 010-.255c.007-.378-.138-.75-.43-.99l-1.004-.828a1.125 1.125 0 01-.26-1.43l1.297-2.247a1.125 1.125 0 011.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.087.22-.128.332-.183.582-.495.644-.869l.214-1.281z'
        }),
        h('path', {
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          d: 'M15 12a3 3 0 11-6 0 3 3 0 016 0z'
        })
      ]
    )
}

const OrderIcon = {
  render: () =>
    h(
      'svg',
      { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' },
      [
        h('path', {
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          d: 'M9 12h3.75M9 15h3.75M9 18h3.75m3 .75H18a2.25 2.25 0 002.25-2.25V6.108c0-1.135-.845-2.098-1.976-2.192a48.424 48.424 0 00-1.123-.08m-5.801 0c-.065.21-.1.433-.1.664 0 .414.336.75.75.75h4.5a.75.75 0 00.75-.75 2.25 2.25 0 00-.1-.664m-5.8 0A2.251 2.251 0 0113.5 2.25H15a2.25 2.25 0 012.15 1.586m-5.8 0c-.376.023-.75.05-1.124.08C9.095 4.01 8.25 4.973 8.25 6.108V8.25m0 0H4.875c-.621 0-1.125.504-1.125 1.125v11.25c0 .621.504 1.125 1.125 1.125h9.75c.621 0 1.125-.504 1.125-1.125V9.375c0-.621-.504-1.125-1.125-1.125H8.25zM6.75 12h.008v.008H6.75V12zm0 3h.008v.008H6.75V15zm0 3h.008v.008H6.75V18z'
        })
      ]
    )
}

const SignalIcon = {
  render: () =>
    h(
      'svg',
      { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' },
      [
        h('path', {
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          d: 'M9.348 14.651a3.75 3.75 0 010-5.303m5.304 0a3.75 3.75 0 010 5.303m-7.425 2.122a6.75 6.75 0 010-9.546m9.546 0a6.75 6.75 0 010 9.546M5.106 18.894c-3.808-3.807-3.808-9.98 0-13.788m13.788 0c3.808 3.807 3.808 9.98 0 13.788M12 12h.008v.008H12V12zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0z'
        })
      ]
    )
}

const ShieldIcon = {
  render: () =>
    h(
      'svg',
      { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' },
      [
        h('path', {
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          d: 'M9 12.75L11.25 15 15 9.75m-3-7.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z'
        })
      ]
    )
}

const PriceTagIcon = {
  render: () =>
    h(
      'svg',
      { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' },
      [
        h('path', {
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          d: 'M9.568 3H5.25A2.25 2.25 0 003 5.25v4.318c0 .597.237 1.17.659 1.591l9.581 9.581c.699.699 1.78.872 2.607.33a18.095 18.095 0 005.223-5.223c.542-.827.369-1.908-.33-2.607L11.16 3.66A2.25 2.25 0 009.568 3z'
        }),
        h('path', {
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          d: 'M6 6h.008v.008H6V6z'
        })
      ]
    )
}

const BookIcon = {
  render: () =>
    h(
      'svg',
      { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' },
      [
        h('path', {
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          d: 'M12 6.042A8.967 8.967 0 006 3.75c-1.052 0-2.062.18-3 .512v14.25A8.987 8.987 0 016 18c2.305 0 4.408.867 6 2.292m0-14.25a8.966 8.966 0 016-2.292c1.052 0 2.062.18 3 .512v14.25A8.987 8.987 0 0018 18a8.967 8.967 0 00-6 2.292m0-14.25v14.25',
        }),
      ],
    ),
}

const ChevronDownIcon = {
  render: () =>
    h(
      'svg',
      { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' },
      [
        h('path', {
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          d: 'm19.5 8.25-7.5 7.5-7.5-7.5'
        })
      ]
    )
}

const DiagnosticsIcon = {
  render: () => h(Icon, { name: 'activity', size: 'md', strokeWidth: 1.7 })
}

const QuotaViewerIcon = {
  render: () => h(Icon, { name: 'download', size: 'md', strokeWidth: 1.7 })
}

const SkillMarketIcon = {
  render: () => h(Icon, { name: 'sparkles', size: 'md', strokeWidth: 1.7 })
}

// The user-facing Work entry supplies the uploaded Skill icon as `iconSvg` in
// userNavigation.ts. Keep this fallback and the established admin icon
// separate for consumers that still request an icon component directly.
const AdminSkillMarketIcon = {
  render: () => h(Icon, { name: 'cube', size: 'md', strokeWidth: 1.7 })
}

const AccountAssistantIcon = {
  render: () => h(Icon, { name: 'lightbulb', size: 'md', strokeWidth: 1.7 })
}

const ModelIcon = {
  render: () => h(Icon, { name: 'destinationModels', size: 'md', strokeWidth: 1.7 })
}

const supportIconByDestination: Record<string, SidebarSupportIcon> = {
  webChat: 'chat',
  models: 'destinationModels',
  contact: 'destinationContact',
  documentation: 'destinationDocument',
}

const shellAudience = computed(() => (isAdminWorkspace.value ? 'admin' as const : 'user' as const))
const shellDestinationContext = computed(() => ({
  audience: shellAudience.value,
  simpleMode: authStore.isSimpleMode,
  capabilities: {
    'public-model-catalog': toShellCapabilityState(
      appStore.backendModeEnabled
        ? false
        : appStore.cachedPublicSettings?.public_model_catalog_enabled,
    ),
  },
}))

function resolveSupportHref(spec: ShellDestinationSpec): string | null {
  if (spec.target.kind === 'href') return spec.target.href
  if (spec.target.kind === 'route') return spec.target.path
  if (spec.target.kind === 'configured-href') {
    return spec.target.source === 'documentation'
      ? documentationUrl.value
      : contactUrl.value
  }
  return null
}

const sidebarSupportLinks = computed<SidebarSupportLink[]>(() => {
  const specs = getShellDestinationSpecs(shellAudience.value)
  const visibleSpecs = selectVisibleShellDestinations(
    specs,
    shellDestinationContext.value,
    'support',
  )

  const links = visibleSpecs.flatMap((spec): SidebarSupportLink[] => {
    const href = resolveSupportHref(spec)
    if (!href) return []

    return [{
      id: spec.id,
      label: t(spec.labelKey),
      href,
      icon: supportIconByDestination[spec.id] ?? 'destinationDocument',
    }]
  })

  // Keep the low-frequency support area focused: the web-chat entry is the
  // primary cross-shell handoff when the user-facing shell is available, while
  // documentation remains available in backend-only mode and stays the final
  // link immediately above the account footer.
  return links.filter((link) => (
    link.id === 'documentation'
    || (link.id === 'webChat' && !appStore.backendModeEnabled)
  ))
})

const userNavigationIcons: UserNavigationIcons = {
  dashboard: DashboardIcon,
  model: ModelIcon,
  batchImage: BatchImageIcon,
  chart: ChartIcon,
  channel: ChannelIcon,
  signal: SignalIcon,
  skillMarket: SkillMarketIcon,
  quotaViewer: QuotaViewerIcon,
  rechargeSubscription: RechargeSubscriptionIcon,
  creditCard: CreditCardIcon,
  // Use the checklist/receipt glyph for personal orders so it stays
  // visually distinct from the folded-document glyph used by documentation.
  orderList: OrderIcon,
  users: UsersIcon,
  user: UserIcon,
}

const adminNavigationIcons: AdminNavigationIcons = {
  dashboard: DashboardIcon,
  chart: ChartIcon,
  opsChart: OpsChartIcon,
  usageChart: UsageChartIcon,
  users: UsersIcon,
  folder: FolderIcon,
  server: ServerIcon,
  channel: ChannelIcon,
  priceTag: PriceTagIcon,
  signal: SignalIcon,
  skillMarket: AdminSkillMarketIcon,
  accountAssistant: AccountAssistantIcon,
  creditCard: CreditCardIcon,
  order: OrderIcon,
  ticket: TicketIcon,
  gift: GiftIcon,
  shield: ShieldIcon,
  diagnostics: DiagnosticsIcon,
  book: BookIcon,
  cog: CogIcon,
}

const userNavSections = computed(() => buildUserNavigation({
  t: (key) => t(key),
  simpleMode: authStore.isSimpleMode,
  canUseBatchImage: () => canUseBatchImage.value,
  customMenuItems: appStore.cachedPublicSettings?.custom_menu_items ?? [],
  icons: userNavigationIcons,
}))

const normalizedSidebarSearch = computed(() => sidebarSearchQuery.value.trim().toLocaleLowerCase())

function matchesSidebarSearch(label: string): boolean {
  const query = normalizedSidebarSearch.value
  return !query || label.toLocaleLowerCase().includes(query)
}

function filterSidebarItems(items: NavItem[]): NavItem[] {
  if (!normalizedSidebarSearch.value) return items

  return items.flatMap((item) => {
    if (matchesSidebarSearch(item.label)) return [item]

    const children = item.children ? filterSidebarItems(item.children) : []
    return children.length ? [{ ...item, children }] : []
  })
}

const displayedUserNavSections = computed(() => {
  if (!normalizedSidebarSearch.value) return userNavSections.value

  return userNavSections.value.flatMap((section) => {
    if (section.label && matchesSidebarSearch(section.label)) return [section]
    const items = filterSidebarItems(section.items)
    return items.length ? [{ ...section, items }] : []
  })
})

const displayedSidebarSupportLinks = computed(() => (
  sidebarSupportLinks.value.filter((item) => matchesSidebarSearch(item.label))
))

const adminNavigationDefinition = shallowRef<AdminNavigationDefinition | null>(null)
const expandedAdminSections = ref<Set<AdminNavSectionId>>(new Set())

function activateAdminNavigation(definition: AdminNavigationDefinition | null | undefined): void {
  if (!definition || adminNavigationDefinition.value === definition) return
  adminNavigationDefinition.value = definition
  expandedAdminSections.value = definition.loadExpandedSections()
}

watch(
  () => props.adminNavigation,
  (definition) => activateAdminNavigation(definition),
  { immediate: true, flush: 'sync' },
)

async function ensureAdminNavigationDefinition(): Promise<AdminNavigationDefinition> {
  if (adminNavigationDefinition.value) return adminNavigationDefinition.value
  const definition = await loadAdminNavigation()
  activateAdminNavigation(definition)
  return definition
}

const adminNavItems = computed((): NavItem[] => (
  adminNavigationDefinition.value?.buildItems({
    t: (key) => t(key),
    simpleMode: authStore.isSimpleMode,
    opsMonitoringEnabled: () => adminSettingsStore.value?.opsMonitoringEnabled,
    adminPaymentEnabled: () => adminSettingsStore.value?.paymentEnabled,
    customMenuItems: adminSettingsStore.value?.customMenuItems ?? [],
    icons: adminNavigationIcons,
  }) ?? []
))

const activeAdminSectionId = computed<AdminNavSectionId | null>(() => (
  adminNavigationDefinition.value?.activeSectionId(route.path) ?? null
))

function isAdminSectionExpanded(sectionId: AdminNavSectionId): boolean {
  return activeAdminSectionId.value === sectionId || expandedAdminSections.value.has(sectionId)
}

function toggleAdminSection(sectionId: AdminNavSectionId) {
  if (activeAdminSectionId.value === sectionId) return

  const next = new Set(expandedAdminSections.value)
  if (next.has(sectionId)) {
    next.delete(sectionId)
  } else {
    next.add(sectionId)
  }
  expandedAdminSections.value = next
  adminNavigationDefinition.value?.persistExpandedSections(next)
}

function adminSectionPanelId(sectionId: AdminNavSectionId): string {
  return adminNavigationDefinition.value?.sectionPanelId(sectionId)
    ?? `sidebar-admin-${sectionId}-items`
}

const adminNavSections = computed(() => (
  adminNavigationDefinition.value?.buildSections(
    adminNavItems.value,
    (key) => t(key),
  ) ?? []
))

const displayedAdminNavSections = computed(() => {
  if (!normalizedSidebarSearch.value) return adminNavSections.value

  return adminNavSections.value.flatMap((section) => {
    if (section.label && matchesSidebarSearch(section.label)) return [section]
    const items = filterSidebarItems(section.items)
    return items.length ? [{ ...section, items }] : []
  })
})

const hasSidebarSearchResults = computed(() => {
  if (!normalizedSidebarSearch.value) return true
  if (isAdminWorkspace.value) return displayedAdminNavSections.value.length > 0

  return displayedUserNavSections.value.length > 0
    || displayedSidebarSupportLinks.value.length > 0
})

function closeMobile() {
  appStore.setMobileOpen(false)
}

function closeNarrowSidebar() {
  appStore.setWorkspaceNarrowSidebarOpen(false)
}

async function toggleSidebarSearch() {
  if (sidebarSearchOpen.value) {
    await closeSidebarSearch()
    return
  }

  sidebarSearchOpen.value = true
  await nextTick()
  sidebarSearchInputRef.value?.focus()
}

async function closeSidebarSearch() {
  sidebarSearchQuery.value = ''
  sidebarSearchOpen.value = false
  await nextTick()
  sidebarHeaderRef.value?.focusSearch()
}

function handleMenuItemClick(itemPath: string) {
  if (mobileOpen.value) {
    setTimeout(() => {
      appStore.setMobileOpen(false)
    }, 150)
  }
  if (personalNarrowViewport.value && narrowSidebarOpen.value) {
    appStore.setWorkspaceNarrowSidebarOpen(false)
  }

  // Map paths to tour selectors
  const pathToSelector: Record<string, string> = {
    '/admin/groups': '#sidebar-group-manage',
    '/admin/accounts': '#sidebar-channel-manage',
    '/keys': '[data-tour="sidebar-my-keys"]'
  }

  const selector = pathToSelector[itemPath]
  if (selector && onboardingStore.isCurrentStep(selector)) {
    onboardingStore.nextStep(500)
  }
}

function isActive(path: string, activePaths: readonly string[] = []): boolean {
  // Vue Router exposes query parameters separately, but normalizing here also
  // keeps the selected state correct for lightweight route mocks and direct
  // deep links such as `/pricing?mode=renew`.
  const currentPath = route.path.split(/[?#]/, 1)[0]
  return [path, ...activePaths].some((candidate) => (
    currentPath === candidate || currentPath.startsWith(candidate + '/')
  ))
}

function isChildActive(parent: NavItem, child: NavItem): boolean {
  const currentPath = route.path.split(/[?#]/, 1)[0]
  return child.path === parent.path
    ? currentPath === child.path
    : isActive(child.path, child.activePaths)
}

function isNavItemActive(item: NavItem): boolean {
  return isActive(item.path, item.activePaths)
    || Boolean(item.children?.some(child => isNavItemActive(child)))
}

function isGroupActive(item: NavItem): boolean {
  return Boolean(item.children?.some(child => isNavItemActive(child)))
}

function groupPanelId(item: NavItem): string {
  const key = item.path.replace(/[^a-zA-Z0-9_-]+/g, '-').replace(/^-|-$/g, '')
  return `sidebar-group-${key}`
}

function isGroupExpanded(item: NavItem): boolean {
  return expandedGroups.value.has(item.path) || isGroupActive(item)
}

function toggleGroup(item: NavItem) {
  if (expandedGroups.value.has(item.path)) {
    expandedGroups.value.delete(item.path)
  } else {
    expandedGroups.value.add(item.path)
  }
}

/**
 * Click handler for collapsible parent items.
 * - When sidebar is collapsed: expand it and open the selected group.
 * - When `expandOnly` is true: only toggle expand state.
 * - Otherwise (default, e.g. /admin/orders): navigate to the parent path
 *   (router-link semantics) and ensure the group is expanded.
 */
function handleGroupClick(item: NavItem) {
  if (sidebarCollapsed.value) {
    expandSidebar()
    expandedGroups.value.add(item.path)
    return
  }
  if (item.expandOnly) {
    toggleGroup(item)
    return
  }
  // Push to path and ensure expanded
  if (route.path !== item.path) {
    router.push(item.path)
  }
  if (!expandedGroups.value.has(item.path)) {
    expandedGroups.value.add(item.path)
  }
}

// Fetch admin settings (for feature-gated nav items like Ops).
watch(
  isAdminWorkspace,
  (active) => {
    if (active) {
      void ensureAdminNavigationDefinition()
      void ensureAdminSettings().then((store) => store.fetch())
    }
  },
  { immediate: true },
)

onMounted(() => {
  void refreshBatchImageAccess()
  if (isAdminWorkspace.value) {
    void ensureAdminNavigationDefinition()
    void ensureAdminSettings().then((store) => store.fetch())
  }
  // Restore sidebar scroll position after route change re-mounts the component
  if (appStore.sidebarScrollTop > 0 && sidebarNavRef.value) {
    void nextTick(() => {
      if (sidebarNavRef.value) {
        sidebarNavRef.value.scrollTop = appStore.sidebarScrollTop
      }
    })
  }
})

onBeforeUnmount(() => {
  if (sidebarNavRef.value) {
    appStore.sidebarScrollTop = sidebarNavRef.value.scrollTop
  }
})
</script>

<style scoped>
.app-sidebar {
  isolation: isolate;
  overflow: hidden;
  border-color: var(--app-shell-sidebar-border, var(--workspace-border));
  background: var(--app-shell-sidebar-bg, var(--workspace-sidebar-surface)) !important;
  box-shadow: var(--app-shell-sidebar-shadow, var(--workspace-shadow-surface));
  backdrop-filter: var(--app-shell-sidebar-backdrop, saturate(1.7) blur(36px));
  -webkit-backdrop-filter: var(--app-shell-sidebar-backdrop, saturate(1.7) blur(36px));
}

.app-sidebar::before,
.app-sidebar::after {
  content: '';
  position: absolute;
  pointer-events: none;
}

.app-sidebar::before {
  inset: 0;
  z-index: -1;
  background: var(--app-shell-sidebar-decoration, none);
}

.app-sidebar::after {
  top: 0;
  right: 12%;
  left: 12%;
  height: 1px;
  background: var(--app-shell-sidebar-highlight, var(--workspace-divider));
}

:global(.dark .app-sidebar) {
  border-color: var(--app-shell-sidebar-border, var(--workspace-border));
  background: var(--app-shell-sidebar-bg, var(--workspace-sidebar-surface)) !important;
  box-shadow: var(--app-shell-sidebar-shadow, var(--workspace-shadow-surface));
  backdrop-filter: var(--app-shell-sidebar-backdrop, none);
  -webkit-backdrop-filter: var(--app-shell-sidebar-backdrop, none);
}

:global(.dark .app-sidebar::after) {
  background: var(--app-shell-sidebar-highlight, var(--workspace-divider));
}

.app-sidebar-backdrop {
  display: none;
  background: var(--workspace-overlay-backdrop);
}

.workspace-sidebar-navigation {
  position: relative;
  z-index: 1;
  display: flex;
  min-height: 0;
  flex: 1 1 auto;
  flex-direction: column;
  overflow-y: auto;
  padding: 0;
}

.sidebar-search {
  display: flex;
  min-height: 40px;
  flex: 0 0 auto;
  align-items: center;
  gap: var(--workspace-space-2);
  margin: 0 var(--workspace-space-1) var(--workspace-space-2);
  border: 1px solid var(--workspace-border);
  border-radius: var(--workspace-radius-input);
  padding: 0 var(--workspace-space-2-5);
  color: var(--workspace-text-muted);
  background: var(--workspace-surface);
}

.sidebar-search:focus-within {
  border-color: var(--workspace-border-strong);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--workspace-text) 6%, transparent);
}

.sidebar-search input {
  min-width: 0;
  flex: 1;
  border: 0;
  padding: 0;
  color: var(--workspace-text);
  background: transparent;
  font: inherit;
  font-size: 13px;
  outline: none;
}

.sidebar-search-empty {
  padding: var(--workspace-space-4) var(--workspace-space-3);
  color: var(--workspace-text-muted);
  font-size: 13px;
  text-align: center;
}

.sidebar-child-group {
  border-color: var(--workspace-border);
}

.sidebar-section-toggle:focus-visible {
  outline: 2px solid var(--app-shell-sidebar-focus, var(--lx-clay-accent));
  outline-offset: 2px;
}

.workspace-sidebar-brand--work {
  width: 2.25rem;
  max-width: 2.25rem;
  height: 2.25rem;
  flex: 0 0 2.25rem;
  border-radius: var(--workspace-radius-compact);
}

.workspace-sidebar-brand--work :deep(.app-brand-logo-frame) {
  width: 2.25rem;
  height: 2.25rem;
}

.workspace-sidebar-brand--work :deep(.app-brand-logo-image) {
  width: 1.25rem;
  height: 1.25rem;
}

.workspace-sidebar-brand--work :deep(.app-brand-logo-image-default) {
  transform: scale(1.4);
}

.sidebar-section {
  margin-bottom: 0;
}

.sidebar-support-section {
  margin-top: var(--workspace-space-3);
}

.sidebar-support-section--bottom {
  margin-top: auto;
  padding-top: var(--workspace-space-3);
}

.sidebar-nav-trailing-icon {
  margin-inline-start: auto;
}

:global([dir='rtl']) .sidebar-nav-trailing-icon {
  transform: scaleX(-1);
}

.sidebar-link {
  position: relative;
  width: 100%;
  min-height: 2.25rem;
  gap: 0.625rem;
  padding-top: 0.25rem;
  padding-right: 0.75rem;
  padding-bottom: 0.25rem;
  padding-left: 0.75rem;
  border-radius: 0.625rem;
  color: var(--workspace-text);
  font-size: 0.875rem;
  font-weight: 400;
  line-height: 1.5rem;
  transition:
    width var(--workspace-sidebar-transition-duration) var(--workspace-sidebar-transition-easing),
    color 0.2s ease,
    background-color 0.2s ease,
    box-shadow 0.2s ease;
}

.sidebar-link > :deep(svg),
.sidebar-link > .sidebar-svg-icon {
  color: var(--workspace-text-secondary);
  transition: color 0.2s ease;
}

.sidebar-link:hover {
  color: var(--app-shell-sidebar-hover-color, var(--workspace-text));
  background: var(--app-shell-sidebar-hover-bg, var(--workspace-hover));
  box-shadow: var(--app-shell-sidebar-hover-shadow, none);
}

.sidebar-link:hover > :deep(svg),
.sidebar-link:hover > .sidebar-svg-icon {
  color: var(--app-shell-sidebar-hover-color, var(--workspace-text));
}

.sidebar-link:focus-visible {
  outline: 2px solid var(--app-shell-sidebar-focus, var(--lx-clay-accent));
  outline-offset: 2px;
}

.sidebar-link-active {
  color: var(--app-shell-sidebar-active-color, var(--workspace-text));
  background: var(--app-shell-sidebar-active-bg, var(--workspace-selected));
  box-shadow: none;
}

.sidebar-link-active::before {
  display: var(--app-shell-sidebar-active-marker, none);
  content: '';
  position: absolute;
  top: 50%;
  left: 0;
  width: 2px;
  height: 1rem;
  border-radius: 2px;
  background: var(--app-shell-sidebar-active-marker-color, var(--workspace-text-muted));
  transform: translateY(-50%);
}

.sidebar-link-active > :deep(svg),
.sidebar-link-active > .sidebar-svg-icon {
  color: var(--app-shell-sidebar-active-icon, var(--workspace-text));
}

.sidebar-link-active:hover {
  color: var(--app-shell-sidebar-active-color, var(--workspace-text));
  background: var(--app-shell-sidebar-active-bg, var(--workspace-selected));
}

:global(html:not(.dark) .sidebar--personal-work .sidebar-section + .sidebar-section) {
  margin-top: 0.25rem;
}

/* Keep the documentation link anchored immediately above the footer divider in
 * both themes. The navigation column is flexed so the support link consumes
 * the remaining space without pushing account destinations out of view. */
:global(html:not(.dark) .sidebar--personal-work .sidebar-section.sidebar-support-section),
:global(html.dark .sidebar--personal-work .sidebar-section.sidebar-support-section) {
  margin-top: auto;
}

.sidebar--personal-work .sidebar-link-active {
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
}

.sidebar--personal-work .sidebar-link {
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
}

.sidebar--personal-work .sidebar-search input {
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
}

.sidebar--personal-work .sidebar-search-empty,
.sidebar--personal-work .sidebar-section-title {
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
}

:global(.dark .sidebar-link) {
  color: var(--workspace-text);
}

:global(.dark .sidebar-link > svg),
:global(.dark .sidebar-link > .sidebar-svg-icon) {
  color: var(--workspace-text-secondary);
}

:global(.dark .sidebar-link:hover) {
  color: var(--app-shell-sidebar-hover-color, var(--workspace-text));
  background: var(--app-shell-sidebar-hover-bg, var(--workspace-hover));
  box-shadow: var(--app-shell-sidebar-hover-shadow, none);
}

:global(.dark .sidebar-link-active),
:global(.dark .sidebar-link-active:hover) {
  color: var(--app-shell-sidebar-active-color, var(--workspace-text));
  background: var(--app-shell-sidebar-active-bg, var(--workspace-selected));
}

:global(.dark .sidebar-link-active > svg),
:global(.dark .sidebar-link-active > .sidebar-svg-icon) {
  color: var(--app-shell-sidebar-active-icon, var(--workspace-text));
}

:global(.dark .sidebar-link-active::before) {
  display: var(--app-shell-sidebar-active-marker, none);
}

.sidebar-link-collapsed {
  width: var(--workspace-sidebar-touch-target);
  min-height: 2.75rem;
  justify-content: flex-start;
  gap: 0;
  padding-left: var(--workspace-space-3);
  padding-right: 0;
}

.sidebar-section-title {
  position: relative;
  display: flex;
  align-items: center;
  min-height: 1.5625rem;
  margin: 0;
  padding: 0.1875rem 1rem 0.25rem;
  color: var(--app-shell-sidebar-section-color, var(--workspace-text-muted));
  font-size: 0.75rem;
  font-weight: 400;
  line-height: 1.125rem;
  letter-spacing: 0;
  text-transform: none;
  overflow: hidden;
  white-space: nowrap;
}

.sidebar-section-toggle {
  width: 100%;
  justify-content: space-between;
  gap: 0.5rem;
  cursor: pointer;
  text-align: left;
}

.sidebar-section-toggle:hover {
  color: var(--workspace-text-secondary);
}

.sidebar-section-toggle[aria-disabled='true'] {
  cursor: default;
}

.sidebar-section-chevron {
  transition: transform 0.18s ease;
}

:global(.dark .sidebar-section-toggle:hover) {
  color: var(--workspace-text-secondary);
}

:global(.dark .sidebar-section-title) {
  color: var(--workspace-text-muted);
}

.sidebar-section-title-text {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition:
    opacity 0.16s ease,
    transform 0.16s ease;
}

.sidebar-section-title-text-collapsed {
  opacity: 0;
  transform: translateX(-4px);
}

.sidebar-section-title-collapsed {
  min-height: 0;
  height: 0;
  padding: 0;
}

.sidebar-label {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition:
    max-width 0.2s ease,
    opacity 0.12s ease;
  max-width: 12rem;
}

.sidebar-label-flex {
  display: flex;
  flex: 1 1 auto;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.sidebar-label-collapsed {
  max-width: 0;
  opacity: 0;
  pointer-events: none;
}

/* Custom SVG icon in sidebar: constrain size without overriding uploaded SVG colors */
.sidebar-svg-icon {
  color: currentColor;
}

.sidebar-api-key-icon {
  --key-icon-color: currentColor;
  --key-icon-dot-color: currentColor;
}

.sidebar-svg-icon :deep(svg) {
  display: block;
  width: 1.25rem;
  height: 1.25rem;
}

@media (prefers-reduced-motion: reduce) {
  .app-sidebar,
  .sidebar-label,
  .sidebar-section-title-text,
  .sidebar-section-chevron,
  .sidebar-nav-icon {
    transition-duration: 0.01ms;
  }
}

@media (min-width: 768px) {
  .app-sidebar {
    border-width: 0 1px 0 0;
    border-radius: 0;
    box-shadow: none;
  }

  .sidebar-mobile-hidden {
    transform: translateX(0);
  }
}

@media (max-width: 767px) and (hover: none) and (pointer: coarse) {
  .app-sidebar-backdrop {
    display: block;
  }

  .app-sidebar {
    border-width: 0 1px 0 0;
    border-radius: 0;
    background: var(--app-shell-sidebar-bg, var(--workspace-sidebar-surface)) !important;
    box-shadow: none;
  }

  :global(.dark .app-sidebar) {
    background: var(--app-shell-sidebar-bg, var(--workspace-sidebar-surface)) !important;
  }

  .sidebar-mobile-hidden {
    transform: translateX(-100%);
  }

  .sidebar-link,
  .sidebar-section-toggle {
    min-height: 2.75rem;
  }
}
</style>

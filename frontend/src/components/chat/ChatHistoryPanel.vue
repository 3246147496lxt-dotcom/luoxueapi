<template>
  <WorkspaceSidebarFrame
    :id="resolvedSidebarId"
    :label="t('chat.history.title')"
    :collapsed="sidebarCollapsed"
    :mobile="mobile"
    :overlay="overlay"
    placement="flow"
    surface="chat"
    content-mode="workspace"
    class="chat-history"
    :class="{
      'chat-history--shell': shell,
      'chat-history--collapsed': sidebarCollapsed,
    }"
  >
    <template #header>
      <WorkspaceSidebarHeader
        v-if="shell"
        ref="sidebarHeaderRef"
        :controls="resolvedSidebarId"
        :collapsed="sidebarCollapsed"
        :mobile="mobile"
        :overlay="overlay"
        :show-search="true"
        :search-expanded="searchOpen"
        :search-controls="searchPanelId"
        :search-label="t('chat.history.searchLabel')"
        :collapse-label="t('nav.collapse')"
        :expand-label="t('nav.expand')"
        :close-label="t('chat.actions.closeHistory')"
        :collapsed-logo-src="appStore.siteLogo"
        toggle-test-id="chat-sidebar-collapse-toggle"
        toggle-icon-test-id="chat-sidebar-collapse-toggle-icon"
        @search="toggleSearch"
        @toggle="toggleSidebar"
        @close="$emit('close')"
      >
        <template #brand>
          <WorkspaceSidebarBrand home-path="/chat" />
        </template>
      </WorkspaceSidebarHeader>
    </template>

    <template #mode-switch>
      <AppModeSwitch
        v-if="shell"
        active-mode="chat"
        @change="handleModeChange"
      />
    </template>

    <div
      v-if="!sidebarCollapsed || shell"
      class="chat-history__header"
      :class="{ 'chat-history__header--collapsed': sidebarCollapsed }"
      :data-scrolled-from-top="historyScrolled ? 'true' : undefined"
    >
      <div class="chat-history__primary-nav">
        <button
          type="button"
          class="chat-history__new"
          :class="{
            'chat-history__new--active': activeSection === 'chat' && activeId === null,
            'chat-history__new--collapsed': sidebarCollapsed,
          }"
          :aria-label="t('chat.actions.newChat')"
          :aria-current="activeSection === 'chat' && activeId === null ? 'page' : undefined"
          :title="sidebarCollapsed ? t('chat.actions.newChat') : undefined"
          @click="$emit('new')"
        >
          <Icon name="chatSidebarCompose" size="md" aria-hidden="true" />
          <span :aria-hidden="sidebarCollapsed ? 'true' : undefined">
            {{ t('chat.actions.newChat') }}
          </span>
        </button>
      </div>
      <button
        v-if="!shell"
        ref="searchTriggerRef"
        type="button"
        class="chat-history__search-trigger"
        :aria-label="t('chat.history.searchLabel')"
        :title="t('chat.history.searchLabel')"
        :aria-expanded="searchOpen"
        :aria-controls="searchPanelId"
        @click="toggleSearch"
      >
        <Icon name="search" size="sm" />
      </button>
      <button
        v-if="mobile && !shell"
        type="button"
        class="chat-history__close"
        :aria-label="t('chat.actions.closeHistory')"
        :title="t('chat.actions.closeHistory')"
        @click="$emit('close')"
      >
        <Icon name="x" size="sm" />
      </button>
    </div>

    <nav
      v-if="shell && sidebarCollapsed"
      class="chat-history__collapsed-nav"
      :aria-label="t('nav.chatMode')"
      data-testid="chat-sidebar-collapsed-nav"
    >
      <button
        ref="searchTriggerRef"
        type="button"
        class="chat-history__collapsed-nav-action"
        :aria-label="t('chat.history.searchLabel')"
        :title="t('chat.history.searchLabel')"
        aria-expanded="false"
        :aria-controls="searchPanelId"
        @click="toggleSearch"
      >
        <Icon name="search" size="sm" aria-hidden="true" />
      </button>
      <RouterLink
        to="/chat"
        class="chat-history__collapsed-nav-action chat-history__chat-entry"
        :aria-label="t('nav.chatMode')"
        :title="t('nav.chatMode')"
        :aria-current="activeSection === 'chat' ? 'page' : undefined"
      >
        <Icon name="chat" size="sm" aria-hidden="true" />
      </RouterLink>
      <RouterLink
        to="/library"
        class="chat-history__collapsed-nav-action chat-history__library-entry"
        :aria-label="t('chat.navigation.fileLibrary')"
        :title="t('chat.navigation.fileLibrary')"
        :aria-current="activeSection === 'library' ? 'page' : undefined"
      >
        <Icon name="chatSidebarLibrary" size="sm" aria-hidden="true" />
      </RouterLink>
      <RouterLink
        to="/projects"
        class="chat-history__collapsed-nav-action chat-history__projects-entry"
        :aria-label="t('chat.navigation.projects')"
        :title="t('chat.navigation.projects')"
        :aria-current="activeSection === 'projects' ? 'page' : undefined"
      >
        <Icon name="chatSidebarProjects" size="sm" aria-hidden="true" />
      </RouterLink>
    </nav>

    <form
      v-if="searchOpen && !sidebarCollapsed"
      :id="searchPanelId"
      class="chat-history__search"
      role="search"
      @submit.prevent="$emit('search')"
    >
      <Icon name="search" size="sm" />
      <input
        ref="searchInputRef"
        type="search"
        :value="searchQuery"
        :placeholder="t('chat.history.searchPlaceholder')"
        :aria-label="t('chat.history.searchLabel')"
        @input="updateSearchQuery"
        @keydown.esc.prevent.stop="closeSearch"
      />
      <span v-if="searching" class="chat-history__searching" aria-live="polite">
        {{ t('chat.history.searching') }}
      </span>
    </form>

    <div
      v-show="!sidebarCollapsed"
      class="chat-history__list"
      @scroll.passive="handleHistoryScroll"
    >
      <nav
        v-if="shell && !sidebarCollapsed"
        class="chat-history__primary-nav chat-history__primary-nav--scrolling"
        :aria-label="t('chat.navigation.label')"
      >
        <button
          v-for="item in chatPrimaryItems"
          :key="item.id"
          type="button"
          class="chat-history__primary-action"
          :class="{ 'chat-history__primary-action--active': activeSection === item.id }"
          :aria-disabled="item.available ? undefined : 'true'"
          :aria-current="activeSection === item.id ? 'page' : undefined"
          :aria-label="t(item.labelKey)"
          :title="item.available ? t(item.labelKey) : t('chat.navigation.unavailable', { name: t(item.labelKey) })"
          @click="activatePrimaryNavigation(item)"
        >
          <Icon :name="item.icon" size="md" aria-hidden="true" />
          <span>{{ t(item.labelKey) }}</span>
        </button>
      </nav>

      <section
        v-if="shell && !sidebarCollapsed && projects.length > 0"
        class="chat-history__projects"
        :aria-label="t('chat.navigation.projects')"
      >
        <div class="chat-history__projects-heading">
          <span>{{ t('chat.navigation.projects') }}</span>
          <RouterLink
            to="/projects"
            class="chat-history__projects-add"
            :aria-label="t('projects.newProject')"
            :title="t('projects.newProject')"
          >
            <Icon name="plus" size="sm" aria-hidden="true" />
          </RouterLink>
        </div>
        <nav class="chat-history__project-list">
          <RouterLink
            v-for="project in projects"
            :key="project.id"
            :to="`/projects/${encodeURIComponent(project.id)}`"
            class="chat-history__project-link"
            :class="{ 'chat-history__project-link--active': activeSection === 'projects' && router?.currentRoute.value.path === `/projects/${project.id}` }"
            :aria-label="project.name"
            :title="project.name"
          >
            <span class="chat-history__project-icon" :style="{ '--project-color': projectAccent(project.color) }" aria-hidden="true">{{ project.icon }}</span>
            <span class="chat-history__project-name">{{ project.name }}</span>
            <span class="chat-history__project-count">{{ project.conversationIds.length }}</span>
          </RouterLink>
        </nav>
      </section>

      <div v-if="conversations.length === 0" class="chat-history__empty">
        <Icon name="chat" size="lg" />
        <span>{{ t('chat.history.empty') }}</span>
      </div>

      <section v-if="conversations.length > 0" class="chat-history__group">
        <h2>{{ t('chat.history.recent') }}</h2>
        <ul>
          <li v-for="conversation in conversations" :key="conversation.id">
            <div
              class="chat-history__item"
              :class="{ 'chat-history__item--active': activeSection === 'chat' && conversation.id === activeId }"
            >
              <button
                v-if="renamingId !== conversation.id"
                :ref="(element) => setConversationSelectRef(conversation.id, element)"
                type="button"
                class="chat-history__select"
                :aria-label="conversation.title"
                :aria-current="activeSection === 'chat' && conversation.id === activeId ? 'page' : undefined"
                @click="$emit('select', conversation.id)"
              >
                <strong :title="conversation.title">
                  {{ toChatConversationTitlePreview(conversation.title) }}
                </strong>
              </button>

              <form v-else class="chat-history__rename" @submit.prevent="submitRename(conversation.id)">
                <input
                  :ref="setRenameInput"
                  v-model="renameDraft"
                  type="text"
                  maxlength="80"
                  :aria-label="t('chat.history.renameLabel')"
                  @keydown.esc.prevent.stop="cancelRename(conversation.id)"
                />
                <button type="submit" :aria-label="t('common.save')" :title="t('common.save')">
                  <Icon name="check" size="sm" />
                </button>
              </form>

              <div v-if="renamingId !== conversation.id" class="chat-history__actions">
                <button
                  type="button"
                  class="chat-history__pin-action"
                  aria-disabled="true"
                  :aria-label="t('chat.actions.pinConversation', { title: conversation.title })"
                  @click.stop="showUnavailableConversationAction('chat.actions.pin')"
                >
                  <Icon name="chatHistoryPinSmall" size="sm" aria-hidden="true" />
                </button>
                <button
                  :ref="(element) => setConversationActionRef(conversation.id, element)"
                  type="button"
                  class="chat-history__more-action"
                  aria-haspopup="menu"
                  :aria-expanded="menuConversationId === conversation.id"
                  :aria-controls="menuConversationId === conversation.id
                    ? conversationMenuDomId(conversation.id)
                    : undefined"
                  :aria-label="t('chat.actions.moreForConversation', { title: conversation.title })"
                  @click.stop="toggleConversationMenu(conversation)"
                  @keydown="handleConversationTriggerKeydown($event, conversation)"
                >
                  <Icon name="chatHistoryMore" size="sm" aria-hidden="true" />
                </button>
              </div>

              <div
                v-if="menuConversationId === conversation.id"
                :id="conversationMenuDomId(conversation.id)"
                :ref="setConversationMenu"
                class="chat-history__conversation-menu"
                popover="manual"
                role="menu"
                aria-orientation="vertical"
                :aria-label="t('chat.history.conversationActions', { title: conversation.title })"
                :style="conversationMenuStyle"
                @keydown="handleConversationMenuKeydown"
              >
                <button
                  v-for="action in conversationMenuActions"
                  :key="action.id"
                  type="button"
                  role="menuitem"
                  tabindex="-1"
                  class="chat-history__conversation-menu-item"
                  :class="{
                    'chat-history__conversation-menu-item--danger': action.id === 'delete',
                  }"
                  :aria-disabled="action.available ? undefined : 'true'"
                  :data-action="action.id"
                  @click="activateConversationAction($event, conversation, action.id)"
                >
                  <Icon :name="action.icon" size="md" aria-hidden="true" />
                  <span>{{ t(action.labelKey) }}</span>
                </button>
              </div>
            </div>
          </li>
        </ul>
      </section>

      <button
        v-if="hasMore && !searchQuery"
        type="button"
        class="chat-history__load-more"
        :disabled="loadingMore"
        @click="$emit('loadMore')"
      >
        {{ loadingMore ? t('chat.history.loadingMore') : t('chat.history.loadMore') }}
      </button>
    </div>
    <div v-if="sidebarCollapsed" class="chat-history__collapsed-spacer" aria-hidden="true" />

    <template v-if="shell" #footer>
      <UserAccountCard context="chat" :collapsed="sidebarCollapsed" />
    </template>
  </WorkspaceSidebarFrame>
</template>

<script setup lang="ts">
import { inject, nextTick, onBeforeUnmount, ref, toRef, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { routerKey } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'
import AppModeSwitch from '@/components/layout/AppModeSwitch.vue'
import UserAccountCard from '@/components/layout/UserAccountCard.vue'
import WorkspaceSidebarFrame from '@/components/layout/WorkspaceSidebarFrame.vue'
import WorkspaceSidebarHeader from '@/components/layout/WorkspaceSidebarHeader.vue'
import WorkspaceSidebarBrand from '@/components/layout/WorkspaceSidebarBrand.vue'
import { useWorkspaceSidebarCollapse } from '@/components/layout/useWorkspaceSidebarCollapse'
import { toChatConversationTitlePreview } from '@/features/chat/conversationTitle'
import { useAppStore } from '@/stores/app'
import type { ChatConversation } from '@/types/chat'
import type { Project } from '@/types/projects'

const props = withDefaults(defineProps<{
  conversations: ChatConversation[]
  activeId?: string | null
  mobile?: boolean
  overlay?: boolean
  sidebarId?: string
  searchQuery?: string
  searching?: boolean
  hasMore?: boolean
  loadingMore?: boolean
  shell?: boolean
  activeSection?: 'chat' | 'library' | 'projects'
  projects?: Project[]
}>(), {
  activeId: null,
  mobile: false,
  overlay: false,
  sidebarId: '',
  searchQuery: '',
  searching: false,
  hasMore: false,
  loadingMore: false,
  shell: false,
  activeSection: 'chat',
  projects: () => [],
})

const emit = defineEmits<{
  new: []
  close: []
  select: [id: string]
  rename: [id: string, title: string]
  delete: [id: string]
  clear: []
  'update:searchQuery': [value: string]
  search: []
  loadMore: []
}>()

const { t } = useI18n()
const appStore = useAppStore()
const router = inject(routerKey, null)
const renamingId = ref<string | null>(null)
const renameDraft = ref('')
const renameInputRef = ref<HTMLInputElement | null>(null)
const searchInputRef = ref<HTMLInputElement | null>(null)
const searchTriggerRef = ref<HTMLButtonElement | null>(null)
const conversationMenuRef = ref<HTMLElement | null>(null)
const menuConversationId = ref<string | null>(null)
const conversationMenuStyle = ref({ left: '12px', top: '12px' })
const searchOpen = ref(Boolean(props.searchQuery))
const historyScrolled = ref(false)
const componentId = useId()
const searchPanelId = `${componentId}-history-search`
const resolvedSidebarId = props.sidebarId || `${componentId}-chat-sidebar`
const conversationSelectRefs = new Map<string, HTMLButtonElement>()
const conversationActionRefs = new Map<string, HTMLButtonElement>()
const chatPrimaryItems = [
  {
    id: 'library',
    labelKey: 'chat.navigation.fileLibrary',
    icon: 'chatSidebarLibrary',
    path: '/library',
    available: true,
  },
  {
    id: 'projects',
    labelKey: 'chat.navigation.projects',
    icon: 'chatSidebarProjects',
    path: '/projects',
    available: true,
  },
  {
    id: 'scheduled',
    labelKey: 'chat.navigation.scheduled',
    icon: 'chatSidebarScheduled',
    path: '',
    available: false,
  },
  {
    id: 'plugins',
    labelKey: 'chat.navigation.plugins',
    icon: 'chatSidebarPlugins',
    path: '',
    available: false,
  },
] as const
type ChatPrimaryLabelKey = (typeof chatPrimaryItems)[number]['labelKey']
type ChatPrimaryItem = (typeof chatPrimaryItems)[number]
const conversationMenuActions = [
  {
    id: 'share',
    labelKey: 'chat.actions.share',
    icon: 'chatHistoryShare',
    available: false,
  },
  {
    id: 'rename',
    labelKey: 'chat.actions.rename',
    icon: 'chatHistoryRename',
    available: true,
  },
  {
    id: 'pin',
    labelKey: 'chat.actions.pin',
    icon: 'chatHistoryPin',
    available: false,
  },
  {
    id: 'archive',
    labelKey: 'chat.actions.archive',
    icon: 'chatHistoryArchive',
    available: false,
  },
  {
    id: 'delete',
    labelKey: 'chat.actions.delete',
    icon: 'chatHistoryDelete',
    available: true,
  },
] as const
type ConversationActionId = (typeof conversationMenuActions)[number]['id']
type MenuFocusEdge = 'first' | 'last'
type PopoverElement = HTMLElement & {
  showPopover?: () => void
  hidePopover?: () => void
}

const CONVERSATION_MENU_WIDTH = 144
const CONVERSATION_MENU_HEIGHT = 200
const CONVERSATION_MENU_VIEWPORT_PADDING = 12
const CONVERSATION_MENU_INLINE_OFFSET = -8
const CONVERSATION_MENU_BLOCK_OVERLAP = 4
let conversationMenuListenersAttached = false
const {
  collapsed: sidebarCollapsed,
  headerRef: sidebarHeaderRef,
  expand: expandSidebar,
  toggle: toggleSidebar,
} = useWorkspaceSidebarCollapse({
  enabled: toRef(props, 'shell'),
  mobile: toRef(props, 'mobile'),
  overlay: toRef(props, 'overlay'),
})

function setRenameInput(element: unknown) {
  renameInputRef.value = element instanceof HTMLInputElement ? element : null
}

function updateSearchQuery(event: Event) {
  const target = event.target
  emit('update:searchQuery', target instanceof HTMLInputElement ? target.value : '')
}

function projectAccent(value: string): string {
  if (/^#[\da-f]{3,8}$/i.test(value)) return value
  const palette: Record<string, string> = {
    gray: '#6b7280',
    red: '#ef4444',
    orange: '#f97316',
    yellow: '#eab308',
    green: '#22c55e',
    blue: '#3b82f6',
    purple: '#8b5cf6',
    pink: '#ec4899',
  }
  return palette[value] ?? '#7c3aed'
}

function showUnavailableNavigation(labelKey: ChatPrimaryLabelKey) {
  appStore.showInfo(t('chat.navigation.unavailable', { name: t(labelKey) }))
}

async function activatePrimaryNavigation(item: ChatPrimaryItem) {
  if (!item.available || !item.path) {
    showUnavailableNavigation(item.labelKey)
    return
  }
  if (props.mobile || props.overlay) emit('close')
  await router?.push(item.path)
}

function handleHistoryScroll(event: Event) {
  const target = event.currentTarget
  historyScrolled.value = target instanceof HTMLElement && target.scrollTop > 0
  if (menuConversationId.value) void closeConversationMenu(false)
}

async function toggleSearch() {
  await closeConversationMenu(false)
  const wasCollapsed = sidebarCollapsed.value
  if (wasCollapsed) {
    expandSidebar()
    await nextTick()
    if (searchOpen.value) {
      searchInputRef.value?.focus()
      return
    }
  }
  if (searchOpen.value) {
    await closeSearch()
    return
  }

  searchOpen.value = true
  await nextTick()
  searchInputRef.value?.focus()
}

async function closeSearch() {
  emit('update:searchQuery', '')
  searchOpen.value = false
  await nextTick()
  if (props.shell) sidebarHeaderRef.value?.focusSearch()
  else searchTriggerRef.value?.focus()
}

async function handleModeChange(mode: 'chat' | 'work') {
  if (mode !== 'work') return
  if (props.mobile || props.overlay) emit('close')
  await router?.push('/dashboard')
}

function setConversationSelectRef(id: string, element: unknown) {
  if (element instanceof HTMLButtonElement) conversationSelectRefs.set(id, element)
  else conversationSelectRefs.delete(id)
}

function setConversationActionRef(id: string, element: unknown) {
  if (element instanceof HTMLButtonElement) conversationActionRefs.set(id, element)
  else conversationActionRefs.delete(id)
}

function setConversationMenu(element: unknown) {
  conversationMenuRef.value = element instanceof HTMLElement ? element : null
}

function conversationMenuDomId(id: string) {
  return `${componentId}-conversation-menu-${id}`
}

function getConversationMenuItems() {
  return Array.from(conversationMenuRef.value?.querySelectorAll<HTMLButtonElement>(
    '[role="menuitem"]',
  ) ?? [])
}

function updateConversationMenuPosition() {
  const id = menuConversationId.value
  const trigger = id ? conversationActionRefs.get(id) : null
  if (!trigger?.isConnected) {
    void closeConversationMenu(false)
    return
  }

  const rect = trigger.getBoundingClientRect()
  if (rect.width === 0 && rect.height === 0) {
    conversationMenuStyle.value = {
      left: `${CONVERSATION_MENU_VIEWPORT_PADDING}px`,
      top: `${CONVERSATION_MENU_VIEWPORT_PADDING}px`,
    }
    return
  }

  const viewport = window.visualViewport
  const viewportLeft = viewport?.offsetLeft ?? 0
  const viewportTop = viewport?.offsetTop ?? 0
  const viewportWidth = viewport?.width ?? window.innerWidth
  const viewportHeight = viewport?.height ?? window.innerHeight
  const viewportRight = viewportLeft + viewportWidth
  const viewportBottom = viewportTop + viewportHeight
  const menuWidth = Math.min(
    CONVERSATION_MENU_WIDTH,
    Math.max(0, viewportWidth - (CONVERSATION_MENU_VIEWPORT_PADDING * 2)),
  )
  const menuHeight = Math.min(
    CONVERSATION_MENU_HEIGHT,
    Math.max(0, viewportHeight - (CONVERSATION_MENU_VIEWPORT_PADDING * 2)),
  )

  const preferredLeft = rect.left + CONVERSATION_MENU_INLINE_OFFSET
  const left = Math.min(
    Math.max(preferredLeft, viewportLeft + CONVERSATION_MENU_VIEWPORT_PADDING),
    viewportRight - CONVERSATION_MENU_VIEWPORT_PADDING - menuWidth,
  )
  const belowTop = rect.bottom - CONVERSATION_MENU_BLOCK_OVERLAP
  const preferredTop = belowTop + menuHeight <= viewportBottom - CONVERSATION_MENU_VIEWPORT_PADDING
    ? belowTop
    : rect.top - menuHeight + CONVERSATION_MENU_BLOCK_OVERLAP
  const top = Math.min(
    Math.max(preferredTop, viewportTop + CONVERSATION_MENU_VIEWPORT_PADDING),
    viewportBottom - CONVERSATION_MENU_VIEWPORT_PADDING - menuHeight,
  )

  conversationMenuStyle.value = {
    left: `${Math.round(left * 2) / 2}px`,
    top: `${Math.round(top * 2) / 2}px`,
  }
}

function handleConversationMenuOutsidePointer(event: PointerEvent) {
  const target = event.target
  const id = menuConversationId.value
  const trigger = id ? conversationActionRefs.get(id) : null
  if (!(target instanceof Node)) return
  if (conversationMenuRef.value?.contains(target) || trigger?.contains(target)) return
  void closeConversationMenu(false)
}

function handleConversationMenuOutsideFocus(event: FocusEvent) {
  const target = event.target
  const id = menuConversationId.value
  const trigger = id ? conversationActionRefs.get(id) : null
  if (!(target instanceof Node)) return
  if (conversationMenuRef.value?.contains(target) || trigger?.contains(target)) return
  void closeConversationMenu(false)
}

function handleConversationMenuScroll(event: Event) {
  const target = event.target
  if (target instanceof Node && conversationMenuRef.value?.contains(target)) return
  void closeConversationMenu(false)
}

function attachConversationMenuListeners() {
  if (conversationMenuListenersAttached) return
  conversationMenuListenersAttached = true
  document.addEventListener('pointerdown', handleConversationMenuOutsidePointer, true)
  document.addEventListener('focusin', handleConversationMenuOutsideFocus, true)
  window.addEventListener('resize', updateConversationMenuPosition)
  window.addEventListener('scroll', handleConversationMenuScroll, true)
  window.visualViewport?.addEventListener('resize', updateConversationMenuPosition)
  window.visualViewport?.addEventListener('scroll', updateConversationMenuPosition)
}

function detachConversationMenuListeners() {
  if (!conversationMenuListenersAttached) return
  conversationMenuListenersAttached = false
  document.removeEventListener('pointerdown', handleConversationMenuOutsidePointer, true)
  document.removeEventListener('focusin', handleConversationMenuOutsideFocus, true)
  window.removeEventListener('resize', updateConversationMenuPosition)
  window.removeEventListener('scroll', handleConversationMenuScroll, true)
  window.visualViewport?.removeEventListener('resize', updateConversationMenuPosition)
  window.visualViewport?.removeEventListener('scroll', updateConversationMenuPosition)
}

async function closeConversationMenu(restoreFocus: boolean) {
  const id = menuConversationId.value
  if (!id) return
  const trigger = conversationActionRefs.get(id) ?? null
  const menu = conversationMenuRef.value as PopoverElement | null
  try {
    menu?.hidePopover?.()
  } catch {
    // The native popover may already be closed by the browser.
  }
  detachConversationMenuListeners()
  menuConversationId.value = null
  await nextTick()
  if (restoreFocus && trigger?.isConnected) trigger.focus({ preventScroll: true })
}

async function openConversationMenu(conversation: ChatConversation, focusEdge: MenuFocusEdge) {
  if (menuConversationId.value && menuConversationId.value !== conversation.id) {
    await closeConversationMenu(false)
  }
  menuConversationId.value = conversation.id
  await nextTick()
  updateConversationMenuPosition()
  const menu = conversationMenuRef.value as PopoverElement | null
  try {
    menu?.showPopover?.()
  } catch {
    // The v-if fallback still renders the menu in browsers without Popover API support.
  }
  attachConversationMenuListeners()
  await nextTick()
  const items = getConversationMenuItems()
  ;(focusEdge === 'last' ? items.at(-1) : items[0])?.focus({ preventScroll: true })
}

async function toggleConversationMenu(conversation: ChatConversation) {
  if (menuConversationId.value === conversation.id) {
    await closeConversationMenu(true)
    return
  }
  await openConversationMenu(conversation, 'first')
}

function handleConversationTriggerKeydown(event: KeyboardEvent, conversation: ChatConversation) {
  if (event.key !== 'ArrowDown' && event.key !== 'ArrowUp') return
  event.preventDefault()
  event.stopPropagation()
  void openConversationMenu(conversation, event.key === 'ArrowUp' ? 'last' : 'first')
}

function handleConversationMenuKeydown(event: KeyboardEvent) {
  const items = getConversationMenuItems()
  if (items.length === 0) return

  if (event.key === 'Escape') {
    event.preventDefault()
    event.stopPropagation()
    void closeConversationMenu(true)
    return
  }

  if (event.key === 'Tab') {
    void closeConversationMenu(false)
    return
  }

  const currentIndex = items.indexOf(document.activeElement as HTMLButtonElement)
  let nextIndex: number | null = null
  if (event.key === 'ArrowDown') nextIndex = (currentIndex + 1 + items.length) % items.length
  else if (event.key === 'ArrowUp') nextIndex = (currentIndex - 1 + items.length) % items.length
  else if (event.key === 'Home') nextIndex = 0
  else if (event.key === 'End') nextIndex = items.length - 1
  if (nextIndex === null) return

  event.preventDefault()
  event.stopPropagation()
  items[nextIndex]?.focus({ preventScroll: true })
}

function showUnavailableConversationAction(labelKey: string) {
  appStore.showInfo(t('chat.actions.unavailable', { name: t(labelKey) }))
}

async function activateConversationAction(
  event: MouseEvent,
  conversation: ChatConversation,
  actionId: ConversationActionId,
) {
  const action = conversationMenuActions.find((item) => item.id === actionId)
  if (!action) return
  const restoreFocus = event.detail === 0
  const focusedElement = document.activeElement
  if (
    !restoreFocus
    && focusedElement instanceof HTMLElement
    && conversationMenuRef.value?.contains(focusedElement)
  ) {
    focusedElement.blur()
  }

  if (!action.available) {
    await closeConversationMenu(restoreFocus)
    showUnavailableConversationAction(action.labelKey)
    return
  }

  if (actionId === 'rename') {
    await closeConversationMenu(false)
    await startRename(conversation)
    return
  }

  await closeConversationMenu(restoreFocus)
  emit('delete', conversation.id)
}

async function startRename(conversation: ChatConversation) {
  renamingId.value = conversation.id
  renameDraft.value = conversation.title
  await nextTick()
  renameInputRef.value?.focus()
  renameInputRef.value?.select()
}

async function cancelRename(id: string) {
  renamingId.value = null
  renameDraft.value = ''
  await nextTick()
  ;(conversationActionRefs.get(id) ?? conversationSelectRefs.get(id))?.focus()
}

async function submitRename(id: string) {
  const title = renameDraft.value.trim()
  if (title) emit('rename', id, title)
  await cancelRename(id)
}

watch(() => props.searchQuery, (query) => {
  if (query) {
    if (sidebarCollapsed.value) expandSidebar()
    searchOpen.value = true
  }
})

watch(
  () => props.conversations.map((conversation) => conversation.id),
  (ids) => {
    if (menuConversationId.value && !ids.includes(menuConversationId.value)) {
      void closeConversationMenu(false)
    }
  },
)

watch(sidebarCollapsed, (collapsed) => {
  if (collapsed) void closeConversationMenu(false)
})

onBeforeUnmount(() => {
  const menu = conversationMenuRef.value as PopoverElement | null
  try {
    menu?.hidePopover?.()
  } catch {
    // The browser may already have removed the popover from the top layer.
  }
  detachConversationMenuListeners()
})
</script>

<style scoped>
.chat-history {
  --lx-clay-accent: var(--workspace-text-secondary);
  --chat-history-row-text: var(--workspace-text);
  --chat-history-row-interaction: rgb(0 0 0 / 0.05);
  --chat-history-action-color: #8f8f8f;
  --chat-history-menu-surface: var(--lx-clay-surface-elevated);
  --chat-history-menu-text: var(--workspace-text);
  --chat-history-menu-hover: rgb(0 0 0 / 0.04);
  --chat-history-menu-danger: #ff002a;
  --chat-history-menu-danger-hover: rgb(250 66 62 / 0.16);
  --chat-history-menu-shadow:
    0 0 0 1px rgb(0 0 0 / 0.04),
    0 2px 8px rgb(0 0 0 / 0.04),
    0 4px 80px 8px rgb(0 0 0 / 0.024);
}

:global(html.dark .chat-history) {
  --chat-history-row-text: #ffffff;
  --chat-history-row-interaction: rgb(255 255 255 / 0.1);
  --chat-history-action-color: #afafaf;
  --chat-history-menu-surface: #353535;
  --chat-history-menu-text: #ffffff;
  --chat-history-menu-hover: rgb(255 255 255 / 0.1);
  --chat-history-menu-danger: #fa423e;
  --chat-history-menu-danger-hover: rgb(250 66 62 / 0.16);
  --chat-history-menu-shadow: inset 0 0 1px rgb(255 255 255 / 0.2);
}

.chat-history--collapsed {
  position: relative;
  z-index: 4;
}

.chat-history__collapsed-nav {
  display: flex;
  flex: 0 0 auto;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  padding: 0;
}

.chat-history__collapsed-nav-action {
  display: grid;
  width: 44px;
  height: 44px;
  place-items: center;
  flex: 0 0 44px;
  border: 0;
  border-radius: var(--workspace-radius-button);
  padding: 0;
  color: var(--workspace-text-muted);
  background: transparent;
  text-decoration: none;
  transition: color 150ms ease, background-color 150ms ease;
}

.chat-history__collapsed-nav-action:hover {
  color: var(--workspace-text);
  background: var(--workspace-hover);
}

.chat-history__chat-entry[aria-current='page'],
.chat-history__library-entry[aria-current='page'],
.chat-history__projects-entry[aria-current='page'] {
  color: var(--workspace-text);
  background: var(--workspace-selected);
}

.chat-history__collapsed-spacer {
  min-height: 0;
  flex: 1;
}

.chat-history__header {
  position: relative;
  z-index: 2;
  display: flex;
  align-items: flex-start;
  gap: 8px;
  flex: 0 0 auto;
  padding: 0;
}

.chat-history--shell .chat-history__header:not(.chat-history__header--collapsed)::after {
  position: absolute;
  right: 0;
  top: 100%;
  left: 0;
  height: 6px;
  background: var(--workspace-sidebar-surface);
  box-shadow: 0 1px 0 var(--workspace-footer-divider);
  content: '';
  opacity: 0;
  pointer-events: none;
  transition: opacity 150ms cubic-bezier(0.4, 0, 0.2, 1);
}

.chat-history__primary-nav {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 0;
  margin-inline: -2px;
}

.chat-history__primary-nav--scrolling {
  flex: 0 0 auto;
}

.chat-history__projects {
  margin: 13px 2px 2px;
  padding-top: 12px;
  border-top: 1px solid var(--workspace-footer-divider);
}

.chat-history__projects-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 28px;
  padding: 0 8px;
  color: var(--workspace-sidebar-group-label);
  font-size: var(--workspace-sidebar-group-label-size);
  font-weight: var(--workspace-sidebar-group-label-weight);
}

.chat-history__projects-add {
  display: grid;
  width: 28px;
  height: 28px;
  place-items: center;
  border-radius: 8px;
  color: var(--workspace-text-muted);
}

.chat-history__projects-add:hover {
  color: var(--workspace-text);
  background: var(--workspace-hover);
}

.chat-history__project-list {
  display: grid;
  gap: 2px;
  margin-top: 4px;
}

.chat-history__project-link {
  display: flex;
  align-items: center;
  min-width: 0;
  min-height: 36px;
  gap: 8px;
  padding: 6px 8px;
  border-radius: 9px;
  color: var(--workspace-text-secondary);
  text-decoration: none;
  transition: color 150ms ease, background-color 150ms ease;
}

.chat-history__project-link:hover,
.chat-history__project-link--active {
  color: var(--workspace-text);
  background: var(--workspace-hover);
}

.chat-history__project-icon {
  display: grid;
  width: 22px;
  height: 22px;
  flex: 0 0 22px;
  place-items: center;
  border-radius: 7px;
  background: color-mix(in srgb, var(--project-color, #7c3aed) 14%, transparent);
  font-size: 14px;
  line-height: 1;
}

.chat-history__project-name {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-history__project-count {
  color: var(--workspace-text-muted);
  font-size: 11px;
}

.chat-history__new,
.chat-history__primary-action {
  display: flex;
  width: 100%;
  height: 36px;
  min-height: 36px;
  flex: 0 0 36px;
  align-items: center;
  justify-content: flex-start;
  gap: 6px;
  border: 0;
  border-radius: 10px;
  padding: 6px 10px;
  color: var(--workspace-text);
  background: transparent;
  font-size: 14px;
  font-weight: 400;
  line-height: 20px;
  box-shadow: none;
  overflow: hidden;
  text-align: left;
}

.chat-history__new {
  transition:
    width var(--workspace-sidebar-transition-duration) var(--workspace-sidebar-transition-easing),
    color 150ms ease,
    background-color 150ms ease;
}

.chat-history__primary-action {
  cursor: pointer;
  transition:
    color 150ms cubic-bezier(0.4, 0, 0.2, 1),
    background-color 150ms cubic-bezier(0.4, 0, 0.2, 1);
}

.chat-history__new > svg,
.chat-history__primary-action > svg {
  width: 20px;
  height: 20px;
  flex: 0 0 20px;
}

.chat-history__new > span,
.chat-history__primary-action > span {
  min-width: 0;
  max-width: 12rem;
  overflow: hidden;
  opacity: 1;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-history__new > span {
  transition:
    max-width 0.2s ease,
    opacity 0.12s ease;
}

.chat-history__new--collapsed {
  width: var(--workspace-sidebar-touch-target);
  height: var(--workspace-sidebar-touch-target);
  min-height: var(--workspace-sidebar-touch-target);
  flex: 0 0 var(--workspace-sidebar-touch-target);
  justify-content: center;
  gap: 0;
  padding: 0;
}

.chat-history__new--collapsed > span {
  max-width: 0;
  opacity: 0;
  pointer-events: none;
}

.chat-history__new:hover,
.chat-history__primary-action:hover {
  color: var(--workspace-text);
  background: var(--workspace-hover);
}

.chat-history__new--active,
.chat-history__new--active:hover,
.chat-history__primary-action--active,
.chat-history__primary-action--active:hover {
  color: var(--workspace-text);
  background: var(--workspace-selected);
}

.chat-history__search-trigger,
.chat-history__close {
  display: grid;
  place-items: center;
  width: 36px;
  height: 36px;
  flex: 0 0 36px;
  border: 0;
  border-radius: var(--workspace-radius-compact);
  color: var(--workspace-text-muted);
  background: transparent;
  transition: color 150ms ease, background-color 150ms ease;
}

.chat-history__close {
  grid-column: 2;
  grid-row: 1;
}

.chat-history__search-trigger:hover,
.chat-history__close:hover {
  color: var(--workspace-text);
  background: var(--workspace-hover);
}

.chat-history__list {
  flex: 1;
  min-width: 0;
  min-height: 0;
  overflow-x: hidden;
  overflow-y: auto;
  padding: 0 0 var(--workspace-space-1);
  overscroll-behavior: contain;
}

.chat-history__search {
  position: relative;
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 7px;
  margin: 0 var(--workspace-space-1) var(--workspace-space-2);
  border: 1px solid var(--workspace-border);
  border-radius: var(--workspace-radius-input);
  padding: 0 10px;
  color: var(--workspace-text-muted);
  background: var(--workspace-surface);
  box-shadow: none;
}

.chat-history__search:focus-within {
  border-color: var(--workspace-border-strong);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--workspace-text) 6%, transparent);
}

.chat-history__search input {
  min-width: 0;
  height: 36px;
  flex: 1;
  border: 0;
  padding: 0;
  color: var(--lx-clay-text);
  background: transparent;
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
  outline: none;
}

.chat-history__searching {
  flex: none;
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
}

.chat-history__load-more {
  display: block;
  min-height: 36px;
  width: calc(100% - 16px);
  margin: 10px 8px 0;
  border: 0;
  border-radius: var(--workspace-radius-compact);
  color: var(--workspace-text-secondary);
  background: transparent;
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
}

.chat-history__load-more:hover:not(:disabled) {
  background: var(--workspace-hover);
}

.chat-history__load-more:disabled {
  opacity: 0.6;
}

.chat-history__empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  min-height: 180px;
  padding: 24px;
  color: var(--lx-clay-text-muted);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  text-align: center;
}

.chat-history__group h2 {
  margin: 16px 8px 5px;
  color: var(--workspace-sidebar-group-label);
  font-size: var(--workspace-sidebar-group-label-size);
  font-weight: var(--workspace-sidebar-group-label-weight);
  line-height: var(--workspace-sidebar-group-label-line-height);
  letter-spacing: normal;
}

.chat-history__group {
  margin-right: 6px;
  margin-left: -8px;
}

.chat-history__group ul {
  margin: 0;
  padding: 0;
  list-style: none;
}

.chat-history__item {
  position: relative;
  display: flex;
  align-items: center;
  min-height: 36px;
  margin: 0 6px;
  border-radius: 10px;
  color: var(--chat-history-row-text);
}

.chat-history__item:hover {
  background: var(--chat-history-row-interaction);
}

.chat-history__item--active {
  background: var(--chat-history-row-interaction);
  box-shadow: none;
}

.chat-history__select {
  display: flex;
  align-items: center;
  min-width: 0;
  flex: 1;
  min-height: 36px;
  border: 0;
  padding: 6px 10px;
  color: inherit;
  background: transparent;
  font-size: 14px;
  font-weight: 400;
  line-height: 20px;
  text-align: left;
}

.chat-history__select strong {
  display: block;
  min-width: 0;
  overflow: hidden;
  font-size: inherit;
  font-weight: inherit;
  line-height: inherit;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-history__actions {
  position: absolute;
  right: 0;
  display: flex;
  height: 36px;
  align-items: center;
  gap: 0;
  opacity: 0;
  pointer-events: none;
}

.chat-history__item:hover .chat-history__actions,
.chat-history__item:focus-within .chat-history__actions {
  opacity: 1;
  pointer-events: auto;
}

.chat-history__item:hover .chat-history__select,
.chat-history__item:focus-within .chat-history__select {
  padding-right: 62px;
}

.chat-history__actions button {
  display: flex;
  width: 34px;
  height: 36px;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 0 10px 10px 0;
  padding: 0;
  color: var(--chat-history-action-color);
  background: transparent;
}

.chat-history__rename button {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  border: 0;
  border-radius: 6px;
  color: var(--workspace-text-muted);
  background: transparent;
}

.chat-history__actions button:hover {
  color: var(--chat-history-row-text);
  background: transparent;
}

.chat-history__actions .chat-history__more-action {
  margin-left: -10px;
}

.chat-history__conversation-menu[popover] {
  position: fixed;
  inset: auto;
  z-index: 50;
  width: 144px;
  max-width: calc(100vw - 24px);
  max-height: calc(100dvh - 24px);
  margin: 0;
  overflow-y: auto;
  border: 0;
  border-radius: 20px;
  padding: 10px 0;
  color: var(--chat-history-menu-text);
  background: var(--chat-history-menu-surface);
  box-shadow: var(--chat-history-menu-shadow);
  font-size: 14px;
  font-weight: 400;
  line-height: 20px;
}

.chat-history__conversation-menu::backdrop {
  background: transparent;
}

.chat-history__conversation-menu-item {
  display: flex;
  width: 124px;
  min-height: 36px;
  align-items: center;
  gap: 6px;
  margin: 0 10px;
  border: 0;
  border-radius: 12px;
  padding: 6px 32px 6px 10px;
  color: var(--chat-history-menu-text);
  background: transparent;
  font: inherit;
  white-space: nowrap;
  cursor: pointer;
}

.chat-history__conversation-menu-item > svg {
  width: 20px;
  height: 20px;
  flex: 0 0 20px;
}

.chat-history .chat-history__conversation-menu-item:hover,
.chat-history .chat-history__conversation-menu-item:focus-visible {
  outline: 0;
  background: var(--chat-history-menu-hover);
}

.chat-history__conversation-menu-item[aria-disabled='true'] {
  opacity: 1;
}

.chat-history .chat-history__conversation-menu-item--danger,
.chat-history .chat-history__conversation-menu-item--danger:hover,
.chat-history .chat-history__conversation-menu-item--danger:focus-visible {
  color: var(--chat-history-menu-danger);
}

.chat-history .chat-history__conversation-menu-item--danger:hover,
.chat-history .chat-history__conversation-menu-item--danger:focus-visible {
  background: var(--chat-history-menu-danger-hover);
}

.chat-history__rename {
  display: flex;
  align-items: center;
  gap: 4px;
  width: 100%;
  padding: 7px;
}

.chat-history__rename input {
  min-width: 0;
  flex: 1;
  height: 34px;
  border: 1px solid var(--workspace-border-strong);
  border-radius: 7px;
  padding: 0 8px;
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface);
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
  outline: none;
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--workspace-text) 6%, transparent);
}

.chat-history button:not(.chat-history__conversation-menu-item):focus-visible {
  outline: 2px solid var(--lx-clay-accent);
  outline-offset: 2px;
}

.chat-history__collapsed-nav-action:focus-visible {
  outline: 2px solid var(--workspace-text-secondary);
  outline-offset: 2px;
}

@media (max-width: 767px) {
  .chat-history {
    border-right: 0;
  }

  .chat-history__search-trigger,
  .chat-history__close {
    width: 44px;
    height: 44px;
    flex-basis: 44px;
  }

  .chat-history__new,
  .chat-history__primary-action,
  .chat-history__item,
  .chat-history__select {
    min-height: 44px;
  }

  .chat-history__actions button,
  .chat-history__rename button {
    width: 44px;
    height: 44px;
  }

  .chat-history__actions {
    height: 44px;
  }

  .chat-history__select {
    padding-right: 88px;
  }

  .chat-history__item:hover .chat-history__select,
  .chat-history__item:focus-within .chat-history__select {
    padding-right: 88px;
  }
}

@media (hover: none) and (pointer: coarse) {
  .chat-history__actions {
    opacity: 1;
    pointer-events: auto;
  }
}

@media (min-width: 768px) and (hover: none) and (pointer: coarse) {
  .chat-history__select {
    padding-right: 62px;
  }
}

@media (min-height: 700px) {
  .chat-history--shell
    .chat-history__header[data-scrolled-from-top='true']:not(.chat-history__header--collapsed)::after {
    opacity: 1;
  }
}

@media (prefers-reduced-motion: reduce) {
  .chat-history--shell .chat-history__header::after,
  .chat-history__collapsed-nav-action,
  .chat-history__new,
  .chat-history__primary-action,
  .chat-history__new > span {
    transition: none;
  }
}
</style>

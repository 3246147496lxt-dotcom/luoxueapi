<template>
  <aside class="chat-history" :aria-label="t('chat.history.title')">
    <div class="chat-history__header">
      <button type="button" class="chat-history__new" @click="$emit('new')">
        <Icon name="plus" size="sm" :stroke-width="2" />
        <span>{{ t('chat.actions.newChat') }}</span>
      </button>
      <button
        v-if="mobile"
        type="button"
        class="chat-history__close"
        :aria-label="t('chat.actions.closeHistory')"
        :title="t('chat.actions.closeHistory')"
        @click="$emit('close')"
      >
        <Icon name="x" size="sm" />
      </button>
    </div>

    <form class="chat-history__search" role="search" @submit.prevent="$emit('search')">
      <Icon name="search" size="sm" />
      <input
        type="search"
        :value="searchQuery"
        :placeholder="t('chat.history.searchPlaceholder')"
        :aria-label="t('chat.history.searchLabel')"
        @input="updateSearchQuery"
        @keydown.esc.prevent="$emit('update:searchQuery', '')"
      />
      <span v-if="searching" class="chat-history__searching" aria-live="polite">
        {{ t('chat.history.searching') }}
      </span>
    </form>

    <div class="chat-history__list">
      <div v-if="conversations.length === 0" class="chat-history__empty">
        <Icon name="chat" size="lg" />
        <span>{{ t('chat.history.empty') }}</span>
      </div>

      <section v-for="group in groups" :key="group.key" class="chat-history__group">
        <h2>{{ group.label }}</h2>
        <ul>
          <li v-for="conversation in group.items" :key="conversation.id">
            <div
              class="chat-history__item"
              :class="{ 'chat-history__item--active': conversation.id === activeId }"
            >
              <button
                v-if="renamingId !== conversation.id"
                :ref="(element) => setConversationSelectRef(conversation.id, element)"
                type="button"
                class="chat-history__select"
                :aria-current="conversation.id === activeId ? 'page' : undefined"
                @click="$emit('select', conversation.id)"
              >
                <Icon name="chatBubble" size="sm" />
                <span>
                  <strong>{{ conversation.title }}</strong>
                  <small>{{ formatTime(conversation.updatedAt) }}</small>
                </span>
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
                  :aria-label="t('chat.actions.rename')"
                  :title="t('chat.actions.rename')"
                  @click="startRename(conversation)"
                >
                  <Icon name="edit" size="xs" />
                </button>
                <button
                  type="button"
                  class="chat-history__delete"
                  :aria-label="t('chat.actions.delete')"
                  :title="t('chat.actions.delete')"
                  @click="$emit('delete', conversation.id)"
                >
                  <Icon name="trash" size="xs" />
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

    <div class="chat-history__footer">
      <button
        type="button"
        class="chat-history__clear"
        :disabled="conversations.length === 0"
        @click="$emit('clear')"
      >
        <Icon name="trash" size="sm" />
        <span>{{ t('chat.actions.clearHistory') }}</span>
      </button>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { ChatConversation } from '@/types/chat'

const props = withDefaults(defineProps<{
  conversations: ChatConversation[]
  activeId?: string | null
  mobile?: boolean
  searchQuery?: string
  searching?: boolean
  hasMore?: boolean
  loadingMore?: boolean
}>(), {
  activeId: null,
  mobile: false,
  searchQuery: '',
  searching: false,
  hasMore: false,
  loadingMore: false,
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

const { t, locale } = useI18n()
const renamingId = ref<string | null>(null)
const renameDraft = ref('')
const renameInputRef = ref<HTMLInputElement | null>(null)
const conversationSelectRefs = new Map<string, HTMLButtonElement>()

function setRenameInput(element: unknown) {
  renameInputRef.value = element instanceof HTMLInputElement ? element : null
}

function updateSearchQuery(event: Event) {
  const target = event.target
  emit('update:searchQuery', target instanceof HTMLInputElement ? target.value : '')
}

function setConversationSelectRef(id: string, element: unknown) {
  if (element instanceof HTMLButtonElement) conversationSelectRefs.set(id, element)
  else conversationSelectRefs.delete(id)
}

const groups = computed(() => {
  const now = new Date()
  const todayStart = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime()
  const weekStart = todayStart - 6 * 24 * 60 * 60 * 1000
  const buckets = [
    { key: 'today', label: t('chat.history.today'), items: [] as ChatConversation[] },
    { key: 'week', label: t('chat.history.previousDays'), items: [] as ChatConversation[] },
    { key: 'older', label: t('chat.history.older'), items: [] as ChatConversation[] },
  ]

  for (const conversation of props.conversations) {
    if (conversation.updatedAt >= todayStart) buckets[0].items.push(conversation)
    else if (conversation.updatedAt >= weekStart) buckets[1].items.push(conversation)
    else buckets[2].items.push(conversation)
  }

  return buckets.filter((bucket) => bucket.items.length > 0)
})

function formatTime(timestamp: number) {
  const date = new Date(timestamp)
  const now = new Date()
  if (date.toDateString() === now.toDateString()) {
    return new Intl.DateTimeFormat(locale.value, { hour: '2-digit', minute: '2-digit' }).format(date)
  }
  return new Intl.DateTimeFormat(locale.value, { month: 'short', day: 'numeric' }).format(date)
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
  conversationSelectRefs.get(id)?.focus()
}

async function submitRename(id: string) {
  const title = renameDraft.value.trim()
  if (title) emit('rename', id, title)
  await cancelRename(id)
}
</script>

<style scoped>
.chat-history {
  display: flex;
  flex-direction: column;
  width: 264px;
  min-width: 264px;
  height: 100%;
  overflow: hidden;
  border-right: 1px solid var(--lx-clay-border);
  background: color-mix(in srgb, var(--lx-clay-recessed) 72%, var(--lx-clay-surface));
}

.chat-history__header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 14px 12px 10px;
}

.chat-history__new {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
  min-height: 42px;
  border: 1px solid color-mix(in srgb, var(--lx-clay-accent) 25%, transparent);
  border-radius: 8px;
  color: var(--lx-clay-accent);
  background: var(--lx-clay-surface-soft);
  font-size: 14px;
  font-weight: 800;
  box-shadow: var(--lx-clay-shadow-flat);
  transition: border-color 150ms ease, background-color 150ms ease;
}

.chat-history__new:hover {
  border-color: var(--lx-clay-accent);
  background: var(--lx-clay-accent-soft);
}

.chat-history__close {
  display: grid;
  place-items: center;
  width: 44px;
  height: 44px;
  flex: 0 0 44px;
  border: 0;
  border-radius: 8px;
  color: var(--lx-clay-text-secondary);
  background: transparent;
}

.chat-history__list {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 2px 9px 16px;
  overscroll-behavior: contain;
}

.chat-history__search {
  position: relative;
  display: flex;
  align-items: center;
  gap: 7px;
  margin: 0 12px 8px;
  border: 1px solid var(--lx-clay-border);
  border-radius: 8px;
  padding: 0 10px;
  color: var(--lx-clay-text-muted);
  background: var(--lx-clay-surface-soft);
  box-shadow: var(--lx-clay-shadow-flat);
}

.chat-history__search:focus-within {
  border-color: var(--lx-clay-accent);
}

.chat-history__search input {
  min-width: 0;
  height: 36px;
  flex: 1;
  border: 0;
  padding: 0;
  color: var(--lx-clay-text);
  background: transparent;
  font-size: 12px;
  outline: none;
}

.chat-history__searching {
  flex: none;
  font-size: 10px;
}

.chat-history__load-more {
  display: block;
  min-height: 36px;
  width: calc(100% - 16px);
  margin: 10px 8px 0;
  border: 0;
  border-radius: 8px;
  color: var(--lx-clay-accent);
  background: transparent;
  font-size: 12px;
  font-weight: 700;
}

.chat-history__load-more:hover:not(:disabled) {
  background: var(--lx-clay-accent-soft);
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
  font-size: 13px;
  text-align: center;
}

.chat-history__group h2 {
  margin: 16px 8px 6px;
  color: var(--lx-clay-text-muted);
  font-size: 11px;
  font-weight: 800;
  text-transform: uppercase;
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
  min-height: 54px;
  border-radius: 8px;
  color: var(--lx-clay-text-secondary);
}

.chat-history__item:hover,
.chat-history__item--active {
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface);
}

.chat-history__item--active {
  box-shadow: var(--lx-clay-shadow-flat);
}

.chat-history__item--active .chat-history__select > :first-child {
  color: var(--lx-clay-accent);
}

.chat-history__select {
  display: flex;
  align-items: center;
  gap: 9px;
  min-width: 0;
  flex: 1;
  min-height: 54px;
  border: 0;
  padding: 7px 58px 7px 10px;
  color: inherit;
  background: transparent;
  text-align: left;
}

.chat-history__select > span {
  display: flex;
  min-width: 0;
  flex-direction: column;
}

.chat-history__select strong {
  overflow: hidden;
  font-size: 13px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-history__select small {
  margin-top: 2px;
  color: var(--lx-clay-text-muted);
  font-size: 10px;
}

.chat-history__actions {
  position: absolute;
  right: 5px;
  display: flex;
  gap: 1px;
  opacity: 0;
  pointer-events: none;
}

.chat-history__item:hover .chat-history__actions,
.chat-history__item:focus-within .chat-history__actions,
.chat-history__item--active .chat-history__actions {
  opacity: 1;
  pointer-events: auto;
}

.chat-history__actions button,
.chat-history__rename button {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  border: 0;
  border-radius: 6px;
  color: var(--lx-clay-text-muted);
  background: transparent;
}

.chat-history__actions button:hover {
  color: var(--lx-clay-accent);
  background: var(--lx-clay-accent-soft);
}

.chat-history__actions .chat-history__delete:hover {
  color: var(--lx-clay-danger);
  background: var(--lx-clay-danger-soft);
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
  border: 1px solid var(--lx-clay-accent);
  border-radius: 7px;
  padding: 0 8px;
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface);
  font-size: 12px;
  outline: none;
}

.chat-history__footer {
  border-top: 1px solid var(--lx-clay-border);
  padding: 10px 12px 12px;
}

.chat-history__clear {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  width: 100%;
  min-height: 36px;
  border: 0;
  border-radius: 8px;
  color: var(--lx-clay-text-secondary);
  background: transparent;
  font-size: 12px;
  font-weight: 700;
}

.chat-history__clear:hover:not(:disabled) {
  color: var(--lx-clay-danger);
  background: var(--lx-clay-danger-soft);
}

.chat-history__clear:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}

.chat-history button:focus-visible {
  outline: 2px solid var(--lx-clay-accent);
  outline-offset: 2px;
}

@media (max-width: 1023px) {
  .chat-history {
    width: min(300px, calc(100vw - 48px));
    min-width: min(300px, calc(100vw - 48px));
    border-right: 0;
  }

  .chat-history__actions button,
  .chat-history__rename button {
    width: 44px;
    height: 44px;
  }

  .chat-history__select {
    padding-right: 98px;
  }
}
</style>

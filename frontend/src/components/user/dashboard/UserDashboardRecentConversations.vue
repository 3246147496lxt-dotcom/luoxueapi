<template>
  <section class="dashboard-panel" :aria-label="t('dashboard.workspace.recentConversations')">
    <header class="dashboard-panel-header">
      <div>
        <h2>{{ t('dashboard.workspace.recentConversations') }}</h2>
        <p>{{ t('dashboard.workspace.recentConversationsDescription') }}</p>
      </div>
      <RouterLink to="/chat" class="dashboard-panel-link">
        {{ t('dashboard.workspace.viewAll') }}
        <Icon name="arrowRight" size="xs" aria-hidden="true" />
      </RouterLink>
    </header>

    <div v-if="loading" class="dashboard-conversation-loading" aria-live="polite">
      <div v-for="index in 4" :key="index" class="dashboard-conversation-skeleton">
        <span class="skeleton h-4 w-2/3" />
        <span class="skeleton h-3 w-1/3" />
      </div>
      <span class="sr-only">{{ t('dashboard.workspace.loadingConversations') }}</span>
    </div>

    <div v-else-if="recentConversations.length === 0" class="dashboard-conversation-empty">
      <Icon name="chat" size="lg" aria-hidden="true" />
      <p>{{ t('dashboard.workspace.noConversations') }}</p>
      <RouterLink to="/chat">{{ t('dashboard.workspace.startConversation') }}</RouterLink>
    </div>

    <ol v-else class="dashboard-conversation-list">
      <li v-for="conversation in recentConversations" :key="conversation.id">
        <button
          type="button"
          class="dashboard-conversation"
          :aria-label="t('dashboard.workspace.openConversation', { title: conversation.title })"
          @click="emit('select', conversation.id)"
        >
          <span class="dashboard-conversation__copy">
            <strong :title="conversation.title">{{ conversation.title }}</strong>
            <span>
              {{ conversation.model }}
              <template v-if="conversation.messageCount ?? conversation.messages.length">
                · {{ t('dashboard.workspace.messageCount', {
                  count: conversation.messageCount ?? conversation.messages.length,
                }) }}
              </template>
            </span>
          </span>
          <time :datetime="new Date(conversation.updatedAt).toISOString()">
            {{ formatConversationTime(conversation.updatedAt) }}
          </time>
          <Icon name="chevronRight" size="xs" aria-hidden="true" />
        </button>
      </li>
    </ol>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { ChatConversation } from '@/types/chat'

const props = defineProps<{
  conversations: ChatConversation[]
  loading: boolean
}>()

const emit = defineEmits<{
  select: [id: string]
}>()

const { t, locale } = useI18n()
const recentConversations = computed(() => [...props.conversations]
  .sort((left, right) => right.updatedAt - left.updatedAt)
  .slice(0, 5))

function formatConversationTime(timestamp: number): string {
  const date = new Date(timestamp)
  if (!Number.isFinite(date.getTime())) return ''
  const elapsed = Date.now() - date.getTime()
  const formatter = new Intl.RelativeTimeFormat(locale.value.startsWith('zh') ? 'zh-CN' : 'en-US', {
    numeric: 'auto',
  })
  const minutes = Math.floor(elapsed / 60_000)
  if (Math.abs(minutes) < 60) return formatter.format(-minutes, 'minute')
  const hours = Math.floor(minutes / 60)
  if (Math.abs(hours) < 24) return formatter.format(-hours, 'hour')
  const days = Math.floor(hours / 24)
  if (Math.abs(days) < 7) return formatter.format(-days, 'day')
  return new Intl.DateTimeFormat(locale.value.startsWith('zh') ? 'zh-CN' : 'en-US', {
    month: 'short',
    day: 'numeric',
  }).format(date)
}
</script>

<style scoped>
.dashboard-panel {
  min-width: 0;
  overflow: hidden;
  border: 1px solid var(--workspace-border);
  border-radius: var(--workspace-radius-work-card);
  background: var(--workspace-card-surface);
  box-shadow: var(--workspace-work-shadow-card);
}

.dashboard-panel-header {
  display: flex;
  min-height: 78px;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: var(--workspace-space-4-5) var(--workspace-space-5);
  border-bottom: 1px solid var(--workspace-border);
}

.dashboard-panel-header h2 {
  color: var(--workspace-work-text);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
  line-height: 1.35rem;
}

.dashboard-panel-header p {
  margin-top: 3px;
  color: var(--workspace-work-text-muted);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  line-height: 1rem;
}

.dashboard-panel-link {
  display: inline-flex;
  min-height: 34px;
  flex: 0 0 auto;
  align-items: center;
  gap: 5px;
  border-radius: var(--workspace-radius-compact);
  color: var(--workspace-work-text-secondary);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
}

.dashboard-panel-link:hover {
  color: var(--workspace-work-accent-hover);
  text-decoration: underline;
  text-underline-offset: 3px;
}

.dashboard-panel-link:focus-visible,
.dashboard-conversation:focus-visible,
.dashboard-conversation-empty a:focus-visible {
  outline: 2px solid var(--workspace-work-accent);
  outline-offset: 2px;
}

.dashboard-conversation-list,
.dashboard-conversation-loading {
  padding: 7px;
}

.dashboard-conversation {
  display: grid;
  width: 100%;
  min-height: 64px;
  grid-template-columns: minmax(0, 1fr) auto 16px;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: var(--workspace-radius-button);
  color: var(--workspace-work-text-secondary);
  text-align: left;
  transition: color 140ms ease, background-color 140ms ease;
}

.dashboard-conversation:hover {
  color: var(--workspace-work-accent-hover);
  background: var(--workspace-work-accent-soft);
}

.dashboard-conversation__copy {
  min-width: 0;
}

.dashboard-conversation__copy strong,
.dashboard-conversation__copy span {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-conversation__copy strong {
  color: inherit;
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
  line-height: 1.15rem;
}

.dashboard-conversation__copy span,
.dashboard-conversation time {
  color: var(--workspace-work-text-muted);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  line-height: 1rem;
}

.dashboard-conversation__copy span {
  margin-top: 4px;
}

.dashboard-conversation time {
  white-space: nowrap;
}

.dashboard-conversation > :deep(svg) {
  color: var(--workspace-border-strong);
  transition: color 140ms ease, transform 140ms ease;
}

.dashboard-conversation:hover > :deep(svg) {
  color: var(--workspace-work-accent);
  transform: translateX(2px);
}

.dashboard-conversation-skeleton {
  display: flex;
  min-height: 64px;
  flex-direction: column;
  justify-content: center;
  gap: 8px;
  padding: 10px 12px;
}

.dashboard-conversation-empty {
  display: flex;
  min-height: 272px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 28px;
  color: var(--workspace-work-text-muted);
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
  text-align: center;
}

.dashboard-conversation-empty :deep(svg) {
  color: var(--workspace-work-accent);
}

.dashboard-conversation-empty a {
  min-height: 32px;
  border-radius: var(--workspace-radius-compact);
  color: var(--workspace-work-accent-hover);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
  text-decoration: underline;
  text-underline-offset: 3px;
}

@media (max-width: 479px) {
  .dashboard-panel-link,
  .dashboard-conversation-empty a {
    min-height: 44px;
  }

  .dashboard-conversation-empty a {
    display: inline-flex;
    align-items: center;
  }

  .dashboard-panel-header {
    align-items: flex-start;
    padding: 16px 18px;
  }

  .dashboard-conversation time {
    display: none;
  }
}

@media (prefers-reduced-motion: reduce) {
  .dashboard-conversation {
    transition-duration: 0.01ms;
  }
}

:global(html.dark) .dashboard-panel {
  border-color: var(--workspace-border);
  background: var(--workspace-card-surface);
  box-shadow: none;
}

:global(html.dark) .dashboard-panel-header {
  border-color: var(--workspace-border);
}

:global(html.dark) .dashboard-panel-header h2,
:global(html.dark) .dashboard-conversation {
  color: var(--workspace-dark-text);
}

:global(html.dark) .dashboard-panel-header p,
:global(html.dark) .dashboard-conversation__copy span,
:global(html.dark) .dashboard-conversation time {
  color: var(--workspace-dark-text-muted);
}

:global(html.dark) .dashboard-conversation:hover {
  background: var(--workspace-surface-subtle);
}
</style>

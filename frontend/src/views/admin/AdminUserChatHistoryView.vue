<template>
  <AppLayout variant="home-clay">
    <div
      class="admin-chat-history-page flex min-h-0 flex-col gap-5"
      data-admin-page-kind="table"
    >
      <AdminPageHeader
        :title="t('admin.chatHistory.title')"
        :description="t('admin.chatHistory.description')"
      >
        <template #meta>
          <span class="inline-flex items-center gap-1.5">
            <Icon name="user" size="xs" aria-hidden="true" />
            {{ t('admin.chatHistory.userId', { id: userId }) }}
          </span>
        </template>

        <template #secondary-actions>
          <button
            type="button"
            class="btn btn-secondary min-h-11 lg:min-h-10"
            data-test="back-to-users"
            @click="backToUsers"
          >
            <Icon name="arrowLeft" size="sm" aria-hidden="true" />
            <span>{{ t('admin.chatHistory.backToUsers') }}</span>
          </button>
        </template>
      </AdminPageHeader>

      <section
        class="chat-history-workbench flex min-h-0 flex-1 overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800"
        :aria-label="t('admin.chatHistory.workspaceLabel')"
      >
        <aside
          class="min-h-[30rem] w-full min-w-0 flex-col lg:min-h-0 lg:w-[340px] lg:flex-none"
          :class="selectedConversationId ? 'hidden lg:flex' : 'flex'"
          data-test="conversation-list-pane"
        >
          <div class="flex min-h-[72px] items-center justify-between gap-3 border-b border-gray-200 px-4 py-3 dark:border-dark-700">
            <div class="min-w-0">
              <h2 class="text-sm font-semibold text-gray-950 dark:text-white">
                {{ t('admin.chatHistory.conversationList') }}
              </h2>
              <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-300">
                {{ t('admin.chatHistory.metadataOnly') }}
              </p>
            </div>
            <span
              v-if="!conversationListLoading && conversations.length > 0"
              class="shrink-0 text-xs font-medium tabular-nums text-gray-500 dark:text-dark-300"
            >
              {{ t('admin.chatHistory.pageNumber', { page: listPageNumber }) }}
            </span>
          </div>

          <div class="min-h-0 flex-1 overflow-y-auto" data-test="conversation-list-scroll">
            <div
              v-if="conversationListLoading"
              class="space-y-0"
              aria-live="polite"
              :aria-label="t('admin.chatHistory.loadingConversations')"
            >
              <div
                v-for="index in 6"
                :key="index"
                class="h-[108px] animate-pulse border-b border-gray-100 px-4 py-4 dark:border-dark-700/70"
              >
                <div class="h-4 w-36 rounded bg-gray-200 dark:bg-dark-600"></div>
                <div class="mt-3 h-3 w-52 rounded bg-gray-100 dark:bg-dark-700"></div>
                <div class="mt-3 h-3 w-28 rounded bg-gray-100 dark:bg-dark-700"></div>
              </div>
            </div>

            <div
              v-else-if="conversationListError"
              class="flex min-h-full flex-col items-center justify-center px-6 py-10 text-center"
              role="alert"
              data-test="conversation-list-error"
            >
              <span class="flex h-10 w-10 items-center justify-center rounded-lg bg-red-50 text-red-600 dark:bg-red-950/40 dark:text-red-300">
                <Icon name="exclamationCircle" size="md" aria-hidden="true" />
              </span>
              <h3 class="mt-3 text-sm font-semibold text-gray-950 dark:text-white">
                {{ t('admin.chatHistory.listErrorTitle') }}
              </h3>
              <p class="mt-1 max-w-xs text-xs leading-5 text-gray-500 dark:text-dark-300">
                {{ t('admin.chatHistory.listErrorDescription') }}
              </p>
              <button type="button" class="btn btn-secondary mt-4" @click="loadConversationPage(listPageIndex)">
                <Icon name="refresh" size="sm" aria-hidden="true" />
                {{ t('admin.chatHistory.retry') }}
              </button>
            </div>

            <div
              v-else-if="conversations.length === 0"
              class="flex min-h-full flex-col items-center justify-center px-6 py-10 text-center"
              data-test="conversation-list-empty"
            >
              <span class="flex h-10 w-10 items-center justify-center rounded-lg bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-dark-300">
                <Icon name="chat" size="md" aria-hidden="true" />
              </span>
              <h3 class="mt-3 text-sm font-semibold text-gray-950 dark:text-white">
                {{ t('admin.chatHistory.emptyTitle') }}
              </h3>
              <p class="mt-1 max-w-xs text-xs leading-5 text-gray-500 dark:text-dark-300">
                {{ t('admin.chatHistory.emptyDescription') }}
              </p>
            </div>

            <div v-else>
              <button
                v-for="conversation in conversations"
                :key="conversation.id"
                type="button"
                class="group flex min-h-[108px] w-full min-w-0 flex-col justify-center border-b border-gray-100 px-4 py-3 text-left transition-colors hover:bg-gray-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary-500 dark:border-dark-700/70 dark:hover:bg-dark-700/60"
                :class="conversation.id === selectedConversationId
                  ? 'bg-primary-50/70 ring-1 ring-inset ring-primary-200 dark:bg-primary-950/20 dark:ring-primary-800/60'
                  : ''"
                :aria-pressed="conversation.id === selectedConversationId"
                :data-test="`conversation-${conversation.id}`"
                @click="selectConversation(conversation)"
              >
                <span class="flex min-w-0 items-center justify-between gap-3">
                  <code class="min-w-0 truncate text-xs font-semibold text-gray-900 dark:text-gray-100">
                    {{ conversation.id }}
                  </code>
                  <span class="inline-flex shrink-0 items-center gap-1.5 text-[11px] font-medium text-gray-500 dark:text-dark-300">
                    <span class="h-1.5 w-1.5 rounded-full" :class="statusDotClass(conversation.status)"></span>
                    {{ statusLabel(conversation.status) }}
                  </span>
                </span>
                <span class="mt-2 flex min-w-0 items-center justify-between gap-3">
                  <code class="min-w-0 truncate text-xs text-gray-600 dark:text-gray-300">
                    {{ conversation.model || t('admin.chatHistory.unknownModel') }}
                  </code>
                  <span class="shrink-0 text-[11px] tabular-nums text-gray-500 dark:text-dark-300">
                    {{ t('admin.chatHistory.messageCount', { count: conversation.message_count }) }}
                  </span>
                </span>
                <span class="mt-2 flex items-center gap-1.5 text-[11px] text-gray-500 dark:text-dark-300">
                  <Icon name="clock" size="xs" aria-hidden="true" />
                  {{ formatTimestamp(conversation.updated_at) }}
                </span>
              </button>
            </div>
          </div>

          <div class="flex min-h-[60px] items-center justify-between border-t border-gray-200 px-4 py-2.5 dark:border-dark-700">
            <button
              type="button"
              class="inline-flex h-10 w-10 items-center justify-center rounded-md text-gray-600 transition-colors hover:bg-gray-100 disabled:cursor-not-allowed disabled:opacity-35 dark:text-gray-300 dark:hover:bg-dark-700"
              :disabled="conversationListLoading || listPageIndex === 0"
              :title="t('admin.chatHistory.previousPage')"
              :aria-label="t('admin.chatHistory.previousPage')"
              data-test="previous-conversation-page"
              @click="goToPreviousConversationPage"
            >
              <Icon name="chevronLeft" size="sm" aria-hidden="true" />
            </button>
            <span class="text-xs font-medium tabular-nums text-gray-500 dark:text-dark-300">
              {{ t('admin.chatHistory.pageNumber', { page: listPageNumber }) }}
            </span>
            <button
              type="button"
              class="inline-flex h-10 w-10 items-center justify-center rounded-md text-gray-600 transition-colors hover:bg-gray-100 disabled:cursor-not-allowed disabled:opacity-35 dark:text-gray-300 dark:hover:bg-dark-700"
              :disabled="conversationListLoading || !conversationListHasNextPage"
              :title="t('admin.chatHistory.nextPage')"
              :aria-label="t('admin.chatHistory.nextPage')"
              data-test="next-conversation-page"
              @click="goToNextConversationPage"
            >
              <Icon name="chevronRight" size="sm" aria-hidden="true" />
            </button>
          </div>
        </aside>

        <article
          class="min-h-[30rem] min-w-0 flex-1 flex-col lg:min-h-0 lg:border-l lg:border-gray-200 lg:dark:border-dark-700"
          :class="selectedConversationId ? 'flex' : 'hidden lg:flex'"
          data-test="conversation-detail-pane"
        >
          <div
            v-if="!selectedConversationId"
            class="flex min-h-full flex-1 flex-col items-center justify-center px-8 py-12 text-center"
            data-test="conversation-unselected"
          >
            <span class="flex h-12 w-12 items-center justify-center rounded-lg bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-dark-300">
              <Icon name="eye" size="lg" aria-hidden="true" />
            </span>
            <h2 class="mt-4 text-base font-semibold text-gray-950 dark:text-white">
              {{ t('admin.chatHistory.selectTitle') }}
            </h2>
            <p class="mt-1 max-w-sm text-sm leading-6 text-gray-500 dark:text-dark-300">
              {{ t('admin.chatHistory.selectDescription') }}
            </p>
          </div>

          <template v-else>
            <header class="flex min-h-[76px] flex-none items-center gap-3 border-b border-gray-200 px-4 py-3 sm:px-5 dark:border-dark-700">
              <button
                type="button"
                class="inline-flex h-10 w-10 flex-none items-center justify-center rounded-md text-gray-600 hover:bg-gray-100 lg:hidden dark:text-gray-300 dark:hover:bg-dark-700"
                :title="t('admin.chatHistory.backToList')"
                :aria-label="t('admin.chatHistory.backToList')"
                data-test="back-to-conversation-list"
                @click="clearSelection"
              >
                <Icon name="arrowLeft" size="sm" aria-hidden="true" />
              </button>

              <div class="min-w-0 flex-1">
                <div class="flex min-w-0 items-center gap-2">
                  <h2 class="min-w-0 truncate text-sm font-semibold text-gray-950 sm:text-base dark:text-white">
                    {{ conversationTitle }}
                  </h2>
                  <span
                    v-if="selectedConversationStatus"
                    class="inline-flex shrink-0 items-center gap-1.5 text-xs font-medium text-gray-500 dark:text-dark-300"
                  >
                    <span class="h-1.5 w-1.5 rounded-full" :class="statusDotClass(selectedConversationStatus)"></span>
                    {{ statusLabel(selectedConversationStatus) }}
                  </span>
                </div>
                <div class="mt-1 flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1 text-xs text-gray-500 dark:text-dark-300">
                  <code class="max-w-full truncate">{{ selectedConversationId }}</code>
                  <code v-if="selectedConversationModel" class="max-w-full truncate">
                    {{ selectedConversationModel }}
                  </code>
                </div>
              </div>
            </header>

            <div
              v-if="privacyNoticeVisible"
              class="flex flex-none items-start gap-3 border-b border-amber-200 bg-amber-50 px-4 py-3 text-amber-900 sm:px-5 dark:border-amber-900/60 dark:bg-amber-950/25 dark:text-amber-100"
              data-test="privacy-notice"
              role="note"
            >
              <Icon name="shield" size="sm" class="mt-0.5 flex-none" aria-hidden="true" />
              <p class="min-w-0 flex-1 text-xs leading-5">
                <span class="font-semibold">{{ t('admin.chatHistory.privacyTitle') }}</span>
                {{ t('admin.chatHistory.privacyDescription') }}
              </p>
              <button
                type="button"
                class="inline-flex h-8 w-8 flex-none items-center justify-center rounded-md text-amber-800 hover:bg-amber-100 dark:text-amber-200 dark:hover:bg-amber-900/40"
                :title="t('admin.chatHistory.dismissNotice')"
                :aria-label="t('admin.chatHistory.dismissNotice')"
                @click="privacyNoticeVisible = false"
              >
                <Icon name="x" size="xs" aria-hidden="true" />
              </button>
            </div>

            <div
              ref="messageScrollerRef"
              class="chat-history-message-scroll min-h-0 flex-1 overflow-y-auto"
              data-test="message-scroll"
            >
              <div
                v-if="conversationDetailLoading"
                class="space-y-5 px-5 py-6"
                aria-live="polite"
                :aria-label="t('admin.chatHistory.loadingMessages')"
              >
                <div
                  v-for="index in 4"
                  :key="index"
                  class="animate-pulse rounded-md border border-gray-200 bg-gray-50/70 p-4 dark:border-dark-600 dark:bg-dark-700/30"
                >
                  <div class="h-3 w-24 rounded bg-gray-200 dark:bg-dark-600"></div>
                  <div class="mt-3 h-3 w-full rounded bg-gray-100 dark:bg-dark-700"></div>
                  <div class="mt-2 h-3 w-4/5 rounded bg-gray-100 dark:bg-dark-700"></div>
                </div>
              </div>

              <div
                v-else-if="conversationDetailError"
                class="flex min-h-full flex-col items-center justify-center px-8 py-12 text-center"
                role="alert"
                data-test="conversation-detail-error"
              >
                <span class="flex h-10 w-10 items-center justify-center rounded-lg bg-red-50 text-red-600 dark:bg-red-950/40 dark:text-red-300">
                  <Icon :name="conversationNotFound ? 'search' : 'exclamationCircle'" size="md" aria-hidden="true" />
                </span>
                <h3 class="mt-3 text-sm font-semibold text-gray-950 dark:text-white">
                  {{ conversationNotFound
                    ? t('admin.chatHistory.notFoundTitle')
                    : t('admin.chatHistory.detailErrorTitle') }}
                </h3>
                <p class="mt-1 max-w-sm text-xs leading-5 text-gray-500 dark:text-dark-300">
                  {{ conversationNotFound
                    ? t('admin.chatHistory.notFoundDescription')
                    : t('admin.chatHistory.detailErrorDescription') }}
                </p>
                <button
                  v-if="conversationNotFound"
                  type="button"
                  class="btn btn-secondary mt-4"
                  @click="clearSelection"
                >
                  <Icon name="arrowLeft" size="sm" aria-hidden="true" />
                  {{ t('admin.chatHistory.backToList') }}
                </button>
                <button
                  v-else
                  type="button"
                  class="btn btn-secondary mt-4"
                  @click="reloadSelectedConversation"
                >
                  <Icon name="refresh" size="sm" aria-hidden="true" />
                  {{ t('admin.chatHistory.retry') }}
                </button>
              </div>

              <div
                v-else-if="messages.length === 0"
                class="flex min-h-full flex-col items-center justify-center px-8 py-12 text-center"
                data-test="conversation-messages-empty"
              >
                <span class="flex h-10 w-10 items-center justify-center rounded-lg bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-dark-300">
                  <Icon name="inbox" size="md" aria-hidden="true" />
                </span>
                <h3 class="mt-3 text-sm font-semibold text-gray-950 dark:text-white">
                  {{ t('admin.chatHistory.noMessagesTitle') }}
                </h3>
                <p class="mt-1 max-w-sm text-xs leading-5 text-gray-500 dark:text-dark-300">
                  {{ t('admin.chatHistory.noMessagesDescription') }}
                </p>
              </div>

              <div v-else class="px-4 py-5 sm:px-6">
                <div class="mb-5 flex flex-col items-center gap-2">
                  <button
                    v-if="hasOlderMessages"
                    type="button"
                    class="btn btn-secondary min-h-10"
                    :disabled="olderMessagesLoading || !nextBeforePosition"
                    data-test="load-older-messages"
                    @click="loadOlderMessages"
                  >
                    <Icon
                      name="arrowUp"
                      size="sm"
                      :class="olderMessagesLoading ? 'animate-pulse' : ''"
                      aria-hidden="true"
                    />
                    {{ olderMessagesLoading
                      ? t('admin.chatHistory.loadingOlder')
                      : t('admin.chatHistory.loadOlder') }}
                  </button>
                  <p
                    v-if="olderMessagesError"
                    class="text-center text-xs leading-5 text-red-600 dark:text-red-300"
                    role="alert"
                  >
                    {{ t('admin.chatHistory.olderError') }}
                  </p>
                  <p
                    v-else-if="!hasOlderMessages"
                    class="text-xs text-gray-400 dark:text-dark-400"
                  >
                    {{ t('admin.chatHistory.reachedBeginning') }}
                  </p>
                </div>

                <ol class="divide-y divide-gray-100 dark:divide-dark-700/70" data-test="message-list">
                  <li
                    v-for="message in messages"
                    :key="message.id"
                    class="py-5 first:pt-0 last:pb-0"
                    :data-message-position="message.position"
                  >
                    <div
                      class="rounded-md border p-4 sm:p-5"
                      :class="messageSurfaceClass(message.role)"
                    >
                      <div class="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1">
                        <span class="inline-flex items-center gap-1.5 text-xs font-semibold text-gray-900 dark:text-gray-100">
                          <span class="h-2 w-2 rounded-full" :class="messageDotClass(message.role)"></span>
                          {{ roleLabel(message.role) }}
                        </span>
                        <span class="text-[11px] tabular-nums text-gray-500 dark:text-dark-300">
                          #{{ message.position }}
                        </span>
                        <span class="text-[11px] text-gray-500 dark:text-dark-300">
                          {{ formatTimestamp(message.created_at) }}
                        </span>
                        <code
                          v-if="message.requested_model"
                          class="max-w-full truncate text-[11px] text-gray-500 dark:text-dark-300"
                        >
                          {{ message.requested_model }}
                        </code>
                      </div>

                      <p
                        class="mt-2 whitespace-pre-wrap break-words text-sm leading-6 text-gray-800 dark:text-gray-200"
                        :class="{ 'italic text-gray-400 dark:text-dark-400': !message.content }"
                      >{{ message.content || t('admin.chatHistory.emptyContent') }}</p>

                      <div
                        v-if="message.error_code || message.error_message"
                        class="mt-3 border border-red-200 bg-red-50 px-3 py-2 text-xs leading-5 text-red-700 dark:border-red-800 dark:bg-red-950/25 dark:text-red-200"
                      >
                        <code v-if="message.error_code" class="font-semibold">{{ message.error_code }}</code>
                        <span v-if="message.error_code && message.error_message">: </span>
                        <span>{{ message.error_message }}</span>
                      </div>

                      <div
                        v-if="message.finish_reason || message.receipt_id"
                        class="mt-2 flex min-w-0 flex-wrap gap-x-3 gap-y-1 text-[11px] text-gray-500 dark:text-dark-300"
                      >
                        <span v-if="message.finish_reason">
                          {{ t('admin.chatHistory.finishReason', { reason: message.finish_reason }) }}
                        </span>
                        <code v-if="message.receipt_id" class="max-w-full truncate">
                          {{ t('admin.chatHistory.receiptId', { id: message.receipt_id }) }}
                        </code>
                      </div>
                    </div>
                  </li>
                </ol>
              </div>
            </div>
          </template>
        </article>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import {
  getUserConversation,
  listUserConversations,
  type AdminChatConversationDetail,
  type AdminChatConversationSummary,
  type AdminChatMessage
} from '@/api/admin/chatHistory'
import AppLayout from '@/components/layout/AppLayout.vue'
import AdminPageHeader from '@/components/layout/AdminPageHeader.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatDateTime } from '@/utils/format'

const CONVERSATION_PAGE_SIZE = 30
const MESSAGE_PAGE_SIZE = 100

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const userId = computed(() => String(route.params.userId ?? ''))

const conversations = ref<AdminChatConversationSummary[]>([])
const conversationListLoading = ref(false)
const conversationListError = ref(false)
const listPageIndex = ref(0)
const listPageCursors = ref<Array<string | undefined>>([undefined])
const nextConversationCursor = ref<string>()
const conversationListHasMore = ref(false)

const selectedConversationId = ref<string>()
const conversationDetail = ref<AdminChatConversationDetail>()
const conversationDetailLoading = ref(false)
const conversationDetailError = ref(false)
const conversationNotFound = ref(false)
const messages = ref<AdminChatMessage[]>([])
const hasOlderMessages = ref(false)
const nextBeforePosition = ref<number>()
const olderMessagesLoading = ref(false)
const olderMessagesError = ref(false)
const privacyNoticeVisible = ref(true)
const messageScrollerRef = ref<HTMLElement>()

let conversationListController: AbortController | undefined
let conversationDetailController: AbortController | undefined
let conversationListRequestId = 0
let conversationDetailRequestId = 0

const listPageNumber = computed(() => listPageIndex.value + 1)
const conversationListHasNextPage = computed(() => (
  conversationListHasMore.value && Boolean(nextConversationCursor.value)
))
const selectedConversationSummary = computed(() => (
  conversations.value.find(item => item.id === selectedConversationId.value)
))
const conversationTitle = computed(() => (
  conversationDetail.value?.conversation.title?.trim()
  || (conversationDetailLoading.value
    ? t('admin.chatHistory.loadingConversation')
    : t('admin.chatHistory.untitledConversation'))
))
const selectedConversationStatus = computed(() => (
  conversationDetail.value?.conversation.status
  || selectedConversationSummary.value?.status
  || ''
))
const selectedConversationModel = computed(() => (
  conversationDetail.value?.conversation.model
  || selectedConversationSummary.value?.model
  || ''
))

const isCanceledRequest = (error: unknown): boolean => {
  if (!error || typeof error !== 'object') return false
  const candidate = error as { code?: unknown; name?: unknown }
  return candidate.code === 'ERR_CANCELED' || candidate.name === 'AbortError'
}

const getErrorStatus = (error: unknown): number | undefined => {
  if (!error || typeof error !== 'object') return undefined
  const candidate = error as {
    status?: unknown
    response?: { status?: unknown }
  }
  if (typeof candidate.status === 'number') return candidate.status
  return typeof candidate.response?.status === 'number'
    ? candidate.response.status
    : undefined
}

const resetSelectedConversation = () => {
  conversationDetailController?.abort()
  conversationDetailRequestId += 1
  selectedConversationId.value = undefined
  conversationDetail.value = undefined
  conversationDetailLoading.value = false
  conversationDetailError.value = false
  conversationNotFound.value = false
  messages.value = []
  hasOlderMessages.value = false
  nextBeforePosition.value = undefined
  olderMessagesLoading.value = false
  olderMessagesError.value = false
}

const loadConversationPage = async (targetPageIndex = 0) => {
  const cursor = listPageCursors.value[targetPageIndex]
  const requestId = ++conversationListRequestId
  conversationListController?.abort()
  conversationListController = new AbortController()
  conversationListLoading.value = true
  conversationListError.value = false
  listPageIndex.value = targetPageIndex

  try {
    const page = await listUserConversations(
      userId.value,
      {
        ...(cursor ? { cursor } : {}),
        limit: CONVERSATION_PAGE_SIZE
      },
      { signal: conversationListController.signal }
    )

    if (requestId !== conversationListRequestId) return
    conversations.value = page.items
    nextConversationCursor.value = page.next_cursor
    conversationListHasMore.value = page.has_more
    resetSelectedConversation()
  } catch (error) {
    if (requestId !== conversationListRequestId || isCanceledRequest(error)) return
    resetSelectedConversation()
    conversationListError.value = true
    conversations.value = []
    nextConversationCursor.value = undefined
    conversationListHasMore.value = false
  } finally {
    if (requestId === conversationListRequestId) {
      conversationListLoading.value = false
    }
  }
}

const goToNextConversationPage = async () => {
  if (!conversationListHasNextPage.value || !nextConversationCursor.value) return
  const targetPageIndex = listPageIndex.value + 1
  listPageCursors.value[targetPageIndex] = nextConversationCursor.value
  await loadConversationPage(targetPageIndex)
}

const goToPreviousConversationPage = async () => {
  if (listPageIndex.value === 0) return
  await loadConversationPage(listPageIndex.value - 1)
}

const sortAndDedupeMessages = (items: AdminChatMessage[]): AdminChatMessage[] => {
  const byId = new Map<string, AdminChatMessage>()
  for (const message of items) {
    byId.set(message.id, message)
  }
  return Array.from(byId.values()).sort((left, right) => left.position - right.position)
}

const loadSelectedConversation = async (conversationId: string) => {
  const requestId = ++conversationDetailRequestId
  conversationDetailController?.abort()
  conversationDetailController = new AbortController()
  conversationDetailLoading.value = true
  conversationDetailError.value = false
  conversationNotFound.value = false
  olderMessagesError.value = false

  try {
    const detail = await getUserConversation(
      userId.value,
      conversationId,
      { limit: MESSAGE_PAGE_SIZE },
      { signal: conversationDetailController.signal }
    )

    if (
      requestId !== conversationDetailRequestId
      || selectedConversationId.value !== conversationId
    ) return

    conversationDetail.value = detail
    messages.value = sortAndDedupeMessages(detail.messages)
    hasOlderMessages.value = detail.has_more
    nextBeforePosition.value = detail.next_before_position
  } catch (error) {
    if (requestId !== conversationDetailRequestId || isCanceledRequest(error)) return
    conversationDetailError.value = true
    conversationNotFound.value = getErrorStatus(error) === 404
    conversationDetail.value = undefined
    messages.value = []
    hasOlderMessages.value = false
    nextBeforePosition.value = undefined
  } finally {
    if (requestId === conversationDetailRequestId) {
      conversationDetailLoading.value = false
    }
  }
}

const selectConversation = (conversation: AdminChatConversationSummary) => {
  if (selectedConversationId.value === conversation.id) return
  selectedConversationId.value = conversation.id
  conversationDetail.value = undefined
  messages.value = []
  hasOlderMessages.value = false
  nextBeforePosition.value = undefined
  olderMessagesLoading.value = false
  olderMessagesError.value = false
  if (messageScrollerRef.value) {
    messageScrollerRef.value.scrollTop = 0
  }
  void loadSelectedConversation(conversation.id)
}

const reloadSelectedConversation = () => {
  if (!selectedConversationId.value) return
  void loadSelectedConversation(selectedConversationId.value)
}

const loadOlderMessages = async () => {
  const conversationId = selectedConversationId.value
  const beforePosition = nextBeforePosition.value
  if (!conversationId || !beforePosition || olderMessagesLoading.value) return

  const requestId = ++conversationDetailRequestId
  conversationDetailController?.abort()
  conversationDetailController = new AbortController()
  olderMessagesLoading.value = true
  olderMessagesError.value = false
  const scroller = messageScrollerRef.value
  const previousScrollHeight = scroller?.scrollHeight ?? 0
  const previousScrollTop = scroller?.scrollTop ?? 0

  try {
    const detail = await getUserConversation(
      userId.value,
      conversationId,
      {
        before_position: beforePosition,
        limit: MESSAGE_PAGE_SIZE
      },
      { signal: conversationDetailController.signal }
    )

    if (
      requestId !== conversationDetailRequestId
      || selectedConversationId.value !== conversationId
    ) return

    conversationDetail.value = detail
    messages.value = sortAndDedupeMessages([...detail.messages, ...messages.value])
    hasOlderMessages.value = detail.has_more
    nextBeforePosition.value = detail.next_before_position

    await nextTick()
    if (scroller) {
      scroller.scrollTop = previousScrollTop + scroller.scrollHeight - previousScrollHeight
    }
  } catch (error) {
    if (requestId !== conversationDetailRequestId || isCanceledRequest(error)) return
    olderMessagesError.value = true
  } finally {
    if (requestId === conversationDetailRequestId) {
      olderMessagesLoading.value = false
    }
  }
}

const clearSelection = () => {
  resetSelectedConversation()
}

const backToUsers = () => {
  void router.push({ name: 'AdminUsers' })
}

const formatTimestamp = (value: string): string => formatDateTime(value) || '-'

const statusLabel = (status: string): string => {
  const keyByStatus: Record<string, string> = {
    active: 'active',
    completed: 'completed',
    generating: 'generating',
    pending: 'pending',
    failed: 'failed',
    error: 'failed',
    archived: 'archived'
  }
  const key = keyByStatus[status.toLowerCase()]
  return key
    ? t(`admin.chatHistory.status.${key}`)
    : status || t('admin.chatHistory.status.unknown')
}

const statusDotClass = (status: string): string => {
  switch (status.toLowerCase()) {
    case 'active':
    case 'completed':
      return 'bg-emerald-500'
    case 'generating':
    case 'pending':
      return 'bg-amber-500'
    case 'failed':
    case 'error':
      return 'bg-red-500'
    case 'archived':
      return 'bg-gray-400'
    default:
      return 'bg-gray-400'
  }
}

const roleLabel = (role: string): string => {
  const keyByRole: Record<string, string> = {
    user: 'user',
    assistant: 'assistant',
    system: 'system',
    tool: 'tool'
  }
  const key = keyByRole[role.toLowerCase()]
  return key ? t(`admin.chatHistory.roles.${key}`) : role
}

const messageSurfaceClass = (role: string): string => {
  switch (role.toLowerCase()) {
    case 'user':
      return 'border-blue-100 bg-blue-50/35 dark:border-blue-900/60 dark:bg-blue-950/10'
    case 'assistant':
      return 'border-emerald-100 bg-emerald-50/35 dark:border-emerald-900/60 dark:bg-emerald-950/10'
    case 'system':
      return 'border-amber-100 bg-amber-50/35 dark:border-amber-900/60 dark:bg-amber-950/10'
    case 'tool':
      return 'border-violet-100 bg-violet-50/35 dark:border-violet-900/60 dark:bg-violet-950/10'
    default:
      return 'border-gray-200 bg-gray-50/35 dark:border-dark-600 dark:bg-dark-700/20'
  }
}

const messageDotClass = (role: string): string => {
  switch (role.toLowerCase()) {
    case 'user':
      return 'bg-blue-500'
    case 'assistant':
      return 'bg-emerald-500'
    case 'system':
      return 'bg-amber-500'
    case 'tool':
      return 'bg-violet-500'
    default:
      return 'bg-gray-400'
  }
}

onMounted(() => {
  void loadConversationPage(0)
})

watch(userId, () => {
  listPageCursors.value = [undefined]
  nextConversationCursor.value = undefined
  conversationListHasMore.value = false
  privacyNoticeVisible.value = true
  void loadConversationPage(0)
})

onBeforeUnmount(() => {
  conversationListController?.abort()
  conversationDetailController?.abort()
})
</script>

<style scoped>
.admin-chat-history-page {
  min-height: calc(100dvh - 81px - 62px);
}

.chat-history-message-scroll,
[data-test='conversation-list-scroll'] {
  scrollbar-gutter: stable;
}

@media (min-width: 1024px) {
  .admin-chat-history-page {
    height: calc(100dvh - 81px - 80px);
    min-height: 38rem;
  }
}

@media (prefers-reduced-motion: reduce) {
  .chat-history-workbench *,
  .chat-history-workbench *::before,
  .chat-history-workbench *::after {
    scroll-behavior: auto !important;
    transition-duration: 0.01ms !important;
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
  }
}
</style>

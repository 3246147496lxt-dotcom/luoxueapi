<template>
  <div class="chat-attachment-picker">
    <input
      ref="inputRef"
      class="chat-attachment-picker__input"
      type="file"
      multiple
      :accept="CHAT_ATTACHMENT_ACCEPT"
      :disabled="disabled"
      tabindex="-1"
      aria-hidden="true"
      :aria-label="t('chat.attachments.add')"
      @change="onFileChange"
    />
    <ChatControlTooltip
      v-slot="{ tooltipId }"
      :label="t('chat.attachments.tooltip')"
      :shortcut="['@']"
      :enabled="!menuOpen && !selectionError && !triggerDisabled"
      wide-gap
    >
      <button
        ref="triggerRef"
        type="button"
        class="chat-attachment-picker__trigger"
        :class="{ 'chat-attachment-picker__trigger--open': menuOpen }"
        :disabled="triggerDisabled"
        :aria-label="t('chat.attachments.addMenu')"
        aria-haspopup="menu"
        :aria-expanded="menuOpen"
        :aria-controls="menuId"
        :aria-describedby="!menuOpen && !selectionError && !triggerDisabled ? tooltipId : undefined"
        :aria-keyshortcuts="triggerDisabled ? undefined : '@'"
        data-test="chat-attachment-menu-trigger"
        data-chat-control-anchor
        @pointerdown="menuKeyboardFocus = false"
        @click="toggleMenu"
        @keydown="onTriggerKeydown"
      >
        <Icon name="chatPlus" size="md" aria-hidden="true" />
      </button>
    </ChatControlTooltip>

    <Teleport to="body">
      <Transition name="chat-attachment-menu">
        <div
          v-if="menuOpen"
          ref="menuSurfaceRef"
          class="chat-attachment-menu-popover"
          :class="[
            `chat-attachment-menu-popover--${menuPlacement}`,
            { 'chat-attachment-menu-popover--keyboard': menuKeyboardFocus },
          ]"
          :style="menuStyle"
          data-test="chat-attachment-menu-popover"
          @pointerdown.capture="menuKeyboardFocus = false"
          @keydown="onMenuKeydown"
        >
          <div
            :id="menuId"
            ref="menuRef"
            class="chat-attachment-menu"
            role="menu"
            :aria-label="t('chat.tools.menuLabel')"
            data-test="chat-attachment-menu"
          >
            <button
              v-for="item in filteredMenuItems"
              :key="item.id"
              type="button"
              class="chat-attachment-menu__item"
              :class="{
                'chat-attachment-menu__item--unavailable': !item.available,
              }"
              role="menuitem"
              :aria-disabled="item.available ? undefined : 'true'"
              :aria-label="item.available
                ? `${item.label}, ${item.description}`
                : `${item.label}, ${item.description}, ${t('chat.tools.notSupported')}`"
              :data-test="item.id === 'upload' ? 'chat-attachment-menu-upload' : undefined"
              @click="onMenuItemSelect(item)"
            >
              <span
                class="chat-attachment-menu__icon"
                :class="`chat-attachment-menu__icon--${item.icon}`"
                aria-hidden="true"
              >
                <svg
                  v-if="item.icon === 'upload'"
                  viewBox="0 0 20 20"
                  fill="currentColor"
                  data-chat-tool-icon="paperclip"
                >
                  <path d="M4.335 12.5v-5a.665.665 0 0 1 1.33 0v5a4.335 4.335 0 1 0 8.67 0V5.833a2.668 2.668 0 0 0-5.337 0V12.5a1.002 1.002 0 0 0 2.004 0v-5a.665.665 0 1 1 1.33 0v5a2.332 2.332 0 0 1-4.664 0V5.833a3.999 3.999 0 0 1 7.997 0V12.5a5.665 5.665 0 1 1-11.33 0" />
                </svg>
                <svg
                  v-else-if="item.icon === 'library'"
                  viewBox="0 0 20 20"
                  fill="currentColor"
                  fill-rule="evenodd"
                  clip-rule="evenodd"
                  data-chat-tool-icon="library"
                >
                  <path d="M14.642 2.454a2.33 2.33 0 0 1 2.7 1.89l1.806 10.235a2.33 2.33 0 0 1-1.892 2.701l-1.224.216a2.33 2.33 0 0 1-2.7-1.891l-1.019-5.777v5.092a2.33 2.33 0 0 1-2.332 2.331H8.315c-.65 0-1.238-.268-1.66-.697-.424.43-1.011.697-1.662.697H3.327a2.33 2.33 0 0 1-2.332-2.33v-10a2.333 2.333 0 0 1 2.332-2.333h1.666c.65 0 1.238.266 1.661.695a2.33 2.33 0 0 1 1.661-.695h1.666c.807 0 1.518.41 1.937 1.032.342-.485.87-.84 1.5-.95zm1.39 2.122a1 1 0 0 0-1.16-.812l-1.223.216a1 1 0 0 0-.812 1.16l1.805 10.234c.096.545.615.909 1.16.813l1.222-.216c.545-.096.909-.616.813-1.16zM3.327 3.918c-.553 0-1.002.449-1.002 1.002v10c0 .553.45 1 1.002 1.001h1.666c.516 0 .94-.39.995-.89q-.003-.056-.005-.11v-10q.002-.057.005-.113c-.055-.5-.48-.89-.995-.89zm4.988 0c-.515 0-.94.39-.996.89q.005.056.006.112v10q-.001.056-.006.11c.056.501.48.891.996.891h1.666c.553 0 1.002-.448 1.002-1v-10c0-.554-.449-1.003-1.002-1.003z" />
                </svg>
                <svg
                  v-else-if="item.icon === 'image'"
                  viewBox="0 0 24 24"
                  data-chat-tool-icon="create-image-plugin"
                >
                  <path fill="#43d0fb" d="M7 21.005c-2.211 0-4-1.79-4-4v-10c0-2.211 1.789-4 4-4h10c2.211 0 4 1.789 4 4v10c0 2.21-1.789 4-4 4z" />
                  <path fill="#fff6dd" d="M17.746 9.117a2.845 2.845 0 0 1-2.856 2.845 2.836 2.836 0 0 1-2.844-2.845 2.845 2.845 0 0 1 2.844-2.855 2.855 2.855 0 0 1 2.856 2.855" />
                  <path fill="#ffde83" d="M5.533 12.682c1.367-1.367 3.134-1.367 4.5 0l8.154 8.144a4 4 0 0 1-1.187.179H7c-2.211 0-4-1.79-4-4v-2.072z" />
                </svg>
                <svg
                  v-else-if="item.icon === 'web'"
                  viewBox="0 0 24 24"
                  fill-rule="evenodd"
                  clip-rule="evenodd"
                  data-chat-tool-icon="skill-globe-light"
                >
                  <circle cx="12" cy="12" r="9" fill="#cdf3ff" />
                  <path fill="#41cef9" d="M12 2c5.522 0 10 4.478 10 10s-4.478 10-10 10S2 17.522 2 12 6.478 2 12 2M9.172 13c.146 4.477 1.284 7 2.828 7s2.682-2.523 2.828-7zm-5.108 0a8 8 0 0 0 4.313 6.134C7.686 17.622 7.261 15.549 7.174 13zm12.762 0c-.087 2.55-.512 4.622-1.204 6.134A8 8 0 0 0 19.936 13zm-8.45-8.135A8 8 0 0 0 4.065 11h3.11c.087-2.55.511-4.623 1.203-6.135M12.001 4c-1.544 0-2.682 2.523-2.828 7h5.656C14.682 6.523 13.544 4 12 4m3.623.865C16.314 6.377 16.74 8.45 16.826 11h3.11a8 8 0 0 0-4.314-6.135" />
                </svg>
                <svg
                  v-else-if="item.icon === 'research'"
                  viewBox="0 0 24 24"
                  data-chat-tool-icon="skill-deep-research-light"
                >
                  <path fill="#0c79d8" d="m7.356 20.535 1.947-3.112.49-.73 1.687 1.022-2.424 3.887c-.29.467-.9.611-1.378.322a1.02 1.02 0 0 1-.322-1.389M16.267 20.535l-1.947-3.112-.49-.73-1.687 1.022 2.424 3.887c.289.467.9.611 1.378.322.466-.3.61-.911.322-1.389" />
                  <path fill="#2e9eff" d="M2.611 14.34c.567 1.512 2.1 2.212 3.678 1.7l12.305-4.47-2.657-7.5L3.79 9.13C2.2 9.774 1.5 11.285 2.067 12.807z" />
                  <path fill="#68c4ff" d="M19.578 12.752c-1.722.644-3.044.033-3.678-1.7l-1.867-5.145c-.633-1.733-.022-3.055 1.712-3.688l1.022-.367c1.21-.456 2.155-.011 2.622 1.222l2.445 6.711c.444 1.222-.012 2.156-1.245 2.6z" />
                  <circle cx="11.8" cy="15.304" r="3.056" fill="#68c4ff" />
                  <circle cx="11.8" cy="15.304" r="1.456" fill="#0c79d8" />
                </svg>
                <svg
                  v-else-if="item.icon === 'github'"
                  viewBox="0 0 24 24"
                  fill="currentColor"
                  data-chat-tool-icon="github"
                >
                  <path d="M12 .7a11.5 11.5 0 0 0-3.64 22.41c.58.1.79-.25.79-.56v-2.23c-3.22.7-3.9-1.37-3.9-1.37-.53-1.34-1.29-1.7-1.29-1.7-1.05-.72.08-.7.08-.7 1.16.08 1.77 1.2 1.77 1.2 1.04 1.77 2.71 1.26 3.37.96.1-.75.4-1.26.74-1.55-2.57-.3-5.27-1.29-5.27-5.69 0-1.26.45-2.29 1.19-3.1-.12-.29-.52-1.47.11-3.06 0 0 .97-.31 3.16 1.18A10.9 10.9 0 0 1 12 6.1c.98 0 1.95.13 2.87.39 2.2-1.49 3.16-1.18 3.16-1.18.63 1.59.23 2.77.11 3.06.74.81 1.19 1.84 1.19 3.1 0 4.42-2.7 5.39-5.28 5.68.42.36.79 1.08.79 2.18v3.22c0 .31.21.67.8.56A11.5 11.5 0 0 0 12 .7Z" />
                </svg>
                <svg
                  v-else-if="item.icon === 'figma'"
                  viewBox="0 0 24 24"
                  data-chat-tool-icon="figma"
                >
                  <path fill="#f24e1e" d="M5 2h7v7H8.5A3.5 3.5 0 0 1 5 5.5Z" />
                  <path fill="#ff7262" d="M5 9h7v7H8.5a3.5 3.5 0 1 1 0-7Z" />
                  <path fill="#a259ff" d="M5 16h7v3.5A3.5 3.5 0 1 1 8.5 16Z" />
                  <path fill="#1abcfe" d="M12 2h3.5a3.5 3.5 0 1 1 0 7H12Z" />
                  <circle cx="15.5" cy="12.5" r="3.5" fill="#0acf83" />
                </svg>
                <svg
                  v-else-if="item.icon === 'heygen'"
                  viewBox="0 0 24 24"
                  data-chat-tool-icon="heygen"
                >
                  <defs>
                    <linearGradient id="heygen-green" x1="3" y1="12" x2="12" y2="3">
                      <stop stop-color="#1ed760" />
                      <stop offset="1" stop-color="#12c7b4" />
                    </linearGradient>
                    <linearGradient id="heygen-blue" x1="12" y1="3" x2="21" y2="9">
                      <stop stop-color="#45b9f1" />
                      <stop offset="1" stop-color="#d68bff" />
                    </linearGradient>
                    <linearGradient id="heygen-pink" x1="4" y1="13" x2="13" y2="21">
                      <stop stop-color="#f47ff0" />
                      <stop offset="1" stop-color="#91c7fa" />
                    </linearGradient>
                    <linearGradient id="heygen-cyan" x1="15" y1="11" x2="20" y2="20">
                      <stop stop-color="#32dfc2" />
                      <stop offset="1" stop-color="#05c9ef" />
                    </linearGradient>
                  </defs>
                  <path fill="url(#heygen-green)" d="M2.8 11.8c-.35-.42-.17-.93.18-1.34l6.6-7.63c.48-.55 1.22-.45 1.52.22l1.02 2.48c.58 1.42-.12 3.02-1.55 3.55l-6.5 2.42c-.55.2-.97.57-1.27.3Z" />
                  <path fill="url(#heygen-blue)" d="M11.82 2.87c-.2-.52.3-.85.8-.64 3.3 1.4 7.31 4.34 8.23 6.1.25.48-.12.83-.65.73l-4.85-.93a4.5 4.5 0 0 1-3.12-2.65Z" />
                  <path fill="url(#heygen-pink)" d="m4.08 12.76 5.77-1.98a2.7 2.7 0 0 1 3.58 2.57v6.07c0 1.43-1.55 2.26-2.7 1.42l-6.55-4.8c-1.38-1.02-1.49-2.8-.1-3.28Z" />
                  <path fill="url(#heygen-cyan)" d="M15.53 10.1c1.52-.43 3.57.16 5.03.63.62.2.87.71.68 1.33-.96 3.17-4.2 8.12-5.49 8.98-.58.4-1.3-.02-1.3-.72v-6.87c0-1.47.37-2.94 1.08-3.35Z" />
                </svg>
                <svg
                  v-else
                  viewBox="0 0 24 24"
                  fill="none"
                  data-chat-tool-icon="gmail"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2.6"
                >
                  <path stroke="#4285f4" d="M3 19V6.8" />
                  <path stroke="#34a853" d="M21 19V6.8" />
                  <path stroke="#ea4335" d="m3 6.8 9 7.1 9-7.1" />
                  <path stroke="#fbbc04" d="M3 6.8 6.2 4.5 12 9l5.8-4.5L21 6.8" />
                </svg>
              </span>

              <span class="chat-attachment-menu__copy">
                <span class="chat-attachment-menu__label">{{ item.label }}</span>
                <span class="chat-attachment-menu__description">{{ item.description }}</span>
              </span>

              <span
                v-if="!item.available"
                class="chat-attachment-menu__status"
                :class="{
                  'chat-attachment-menu__status--persistent': item.id === 'gmail',
                }"
                aria-hidden="true"
              >{{ t('chat.tools.notSupported') }}</span>
            </button>

            <div
              v-if="filteredMenuItems.length === 0"
              class="chat-attachment-menu__empty"
              role="status"
            >
              {{ t('chat.tools.noResults') }}
            </div>
          </div>

          <label class="chat-attachment-menu__search">
            <span class="sr-only">{{ t('chat.tools.searchLabel') }}</span>
            <input
              ref="searchInputRef"
              v-model="menuQuery"
              type="search"
              autocomplete="off"
              :placeholder="t('chat.tools.searchPlaceholder')"
              :aria-label="t('chat.tools.searchLabel')"
              data-test="chat-attachment-menu-search"
            />
          </label>
        </div>
      </Transition>
    </Teleport>

    <span
      v-if="selectionError"
      class="chat-attachment-picker__notice"
      role="alert"
    >
      {{ selectionError }}
    </span>
  </div>
</template>

<script setup lang="ts">
import {
  computed,
  nextTick,
  onBeforeUnmount,
  ref,
  useId,
  watch,
  type CSSProperties,
} from 'vue'
import { useI18n } from 'vue-i18n'
import {
  ChatAPIError,
  deleteChatAttachment,
  isAbortError,
  uploadChatAttachment,
} from '@/api/chat'
import ChatControlTooltip from './ChatControlTooltip.vue'
import Icon from '@/components/icons/Icon.vue'
import {
  CHAT_ATTACHMENT_ACCEPT,
  CHAT_ATTACHMENT_MAX_COUNT,
  CHAT_ATTACHMENT_TOTAL_MAX_BYTES,
  attachmentFileKind,
  validateAttachmentFile,
  type ChatAttachmentDraft,
} from './chatAttachmentUi'
import type { ChatAttachment } from '@/types/chat'
import type { LibraryFile } from '@/types/library'

export interface ChatAttachmentReadySelection {
  attachments: ChatAttachment[]
  uploadAttachmentIds: string[]
  libraryAttachments: Array<{ source: 'library'; fileId: string }>
}

type ChatToolIcon = 'upload' | 'library' | 'image' | 'web' | 'research'
  | 'github' | 'figma' | 'heygen' | 'gmail'

interface ChatToolMenuItem {
  id: ChatToolIcon
  icon: ChatToolIcon
  label: string
  description: string
  available: boolean
}

const props = withDefaults(defineProps<{
  disabled?: boolean
  supportsVision?: boolean
}>(), {
  disabled: false,
  supportsVision: false,
})

const emit = defineEmits<{
  change: [items: ChatAttachmentDraft[]]
  'busy-change': [busy: boolean]
  'valid-change': [valid: boolean]
  'select-library': []
}>()

const { t } = useI18n()
const inputRef = ref<HTMLInputElement | null>(null)
const triggerRef = ref<HTMLButtonElement | null>(null)
const menuSurfaceRef = ref<HTMLElement | null>(null)
const menuRef = ref<HTMLElement | null>(null)
const searchInputRef = ref<HTMLInputElement | null>(null)
const menuId = `chat-attachment-menu-${useId()}`
const menuOpen = ref(false)
const menuKeyboardFocus = ref(false)
const menuPlacement = ref<'above' | 'below'>('below')
const menuStyle = ref<CSSProperties>({})
const menuQuery = ref('')
const drafts = ref<ChatAttachmentDraft[]>([])
const selectionError = ref('')
const controllers = new Map<string, AbortController>()
let noticeTimer: ReturnType<typeof setTimeout> | undefined
let draftSequence = 0

const menuItems = computed<ChatToolMenuItem[]>(() => [
  {
    id: 'upload',
    icon: 'upload',
    label: t('chat.tools.upload.label'),
    description: t('chat.tools.upload.description'),
    available: true,
  },
  {
    id: 'library',
    icon: 'library',
    label: t('chat.tools.fileLibrary.label'),
    description: t('chat.tools.fileLibrary.description'),
    available: true,
  },
  {
    id: 'image',
    icon: 'image',
    label: t('chat.tools.createImage.label'),
    description: t('chat.tools.createImage.description'),
    available: false,
  },
  {
    id: 'web',
    icon: 'web',
    label: t('chat.tools.webSearch.label'),
    description: t('chat.tools.webSearch.description'),
    available: false,
  },
  {
    id: 'research',
    icon: 'research',
    label: t('chat.tools.deepResearch.label'),
    description: t('chat.tools.deepResearch.description'),
    available: false,
  },
  {
    id: 'github',
    icon: 'github',
    label: t('chat.tools.integrations.github.label'),
    description: t('chat.tools.integrations.github.description'),
    available: false,
  },
  {
    id: 'figma',
    icon: 'figma',
    label: t('chat.tools.integrations.figma.label'),
    description: t('chat.tools.integrations.figma.description'),
    available: false,
  },
  {
    id: 'heygen',
    icon: 'heygen',
    label: t('chat.tools.integrations.heygen.label'),
    description: t('chat.tools.integrations.heygen.description'),
    available: false,
  },
  {
    id: 'gmail',
    icon: 'gmail',
    label: t('chat.tools.integrations.gmail.label'),
    description: t('chat.tools.integrations.gmail.description'),
    available: false,
  },
])

const filteredMenuItems = computed(() => {
  const query = menuQuery.value.trim().toLocaleLowerCase()
  if (!query) return menuItems.value
  return menuItems.value.filter((item) => (
    `${item.label} ${item.description}`.toLocaleLowerCase().includes(query)
  ))
})

function uploadErrorKey(error: unknown): string {
  if (!(error instanceof ChatAPIError)) return 'chat.attachments.errors.uploadFailed'
  let metadata = ''
  try {
    metadata = JSON.stringify(error.metadata ?? '')
  } catch {
    metadata = String(error.metadata ?? '')
  }
  const signature = [error.code, error.reason, metadata]
    .map((value) => String(value ?? ''))
    .join(' ')
    .toLowerCase()
  if (
    signature.includes('unsafe')
    || signature.includes('malware')
    || signature.includes('virus')
    || signature.includes('zip_bomb')
    || signature.includes('security')
  ) return 'chat.attachments.errors.unsafeDocument'
  if (
    signature.includes('invalid_document')
    || signature.includes('invalid_docx')
    || signature.includes('parse')
    || signature.includes('corrupt')
  ) return 'chat.attachments.errors.documentInvalid'
  return 'chat.attachments.errors.uploadFailed'
}

const busy = computed(() => drafts.value.some(
  ({ state }) => state === 'uploading' || state === 'processing',
))
const triggerDisabled = computed(() => (
  props.disabled || drafts.value.length >= CHAT_ATTACHMENT_MAX_COUNT
))
const valid = computed(() => drafts.value.every((draft) => (
  draft.state === 'ready'
  && !(draft.kind === 'image' && !props.supportsVision)
)))

watch(busy, (value) => emit('busy-change', value), { immediate: true })
watch(valid, (value) => emit('valid-change', value), { immediate: true })
watch(() => props.supportsVision, () => emit('valid-change', valid.value))
watch(menuOpen, (value) => {
  if (value) addMenuListeners()
  else removeMenuListeners()
})
watch(triggerDisabled, (value) => {
  if (value) closeMenu(false)
})

function makeKey(): string {
  try {
    if (typeof globalThis.crypto?.randomUUID === 'function') {
      return `attachment-${globalThis.crypto.randomUUID()}`
    }
  } catch {
    // A local draft key is not security-sensitive.
  }
  draftSequence += 1
  return `attachment-${Date.now().toString(36)}-${draftSequence.toString(36)}`
}

function imagePreview(file: File, kind: ChatAttachmentDraft['kind']): string | undefined {
  if (kind !== 'image' || typeof URL.createObjectURL !== 'function') return undefined
  try {
    return URL.createObjectURL(file)
  } catch {
    return undefined
  }
}

function revokePreview(draft: ChatAttachmentDraft): void {
  if (!draft.previewUrl || typeof URL.revokeObjectURL !== 'function') return
  URL.revokeObjectURL(draft.previewUrl)
}

function publish(): void {
  emit('change', drafts.value.map((draft) => ({ ...draft })))
}

function replaceDraft(key: string, patch: Partial<ChatAttachmentDraft>): void {
  const index = drafts.value.findIndex((draft) => draft.key === key)
  if (index < 0) return
  drafts.value[index] = { ...drafts.value[index]!, ...patch }
  drafts.value = [...drafts.value]
  publish()
}

function attachmentIsUsable(attachment: ChatAttachment): boolean {
  if (attachment.status !== 'ready') return false
  const expiration = Date.parse(attachment.expiresAt)
  return !Number.isFinite(expiration) || expiration > Date.now()
}

function showSelectionError(message: string): void {
  selectionError.value = message
  if (noticeTimer) clearTimeout(noticeTimer)
  noticeTimer = setTimeout(() => {
    selectionError.value = ''
  }, 4200)
}

function updateMenuPosition(): void {
  const trigger = triggerRef.value
  if (!trigger || typeof window === 'undefined') return

  const triggerRect = trigger.getBoundingClientRect()
  const composerRect = trigger.closest<HTMLElement>('.chat-composer')?.getBoundingClientRect()
  const visualViewport = window.visualViewport
  const viewportPadding = 16
  // The trigger sits 8px inside the composer surface. A 16px trigger gap
  // therefore produces the 8px visual gap measured from the composer edge.
  const triggerGap = 16
  const viewportLeft = visualViewport?.offsetLeft ?? 0
  const viewportTop = visualViewport?.offsetTop ?? 0
  const viewportWidth = visualViewport?.width ?? window.innerWidth
  const viewportHeight = visualViewport?.height ?? window.innerHeight
  const viewportRight = viewportLeft + viewportWidth
  const viewportBottom = viewportTop + viewportHeight
  const panelWidth = Math.max(1, Math.min(
    composerRect?.width ?? 768,
    viewportWidth - viewportPadding * 2,
  ))
  const desiredLeft = composerRect?.left ?? triggerRect.left - 8
  const minLeft = viewportLeft + viewportPadding
  const maxLeft = Math.max(minLeft, viewportRight - viewportPadding - panelWidth)
  const left = Math.min(Math.max(desiredLeft, minLeft), maxLeft)
  const measuredHeight = menuSurfaceRef.value?.scrollHeight || 372
  const availableBelow = Math.max(
    0,
    viewportBottom - triggerRect.bottom - triggerGap - viewportPadding,
  )
  const availableAbove = Math.max(
    0,
    triggerRect.top - viewportTop - triggerGap - viewportPadding,
  )
  const opensBelow = availableBelow >= measuredHeight
    || (availableAbove < measuredHeight && availableBelow >= availableAbove)
  const availableHeight = Math.max(1, opensBelow ? availableBelow : availableAbove)
  const visibleHeight = Math.min(measuredHeight, availableHeight)

  menuPlacement.value = opensBelow ? 'below' : 'above'
  menuStyle.value = {
    position: 'fixed',
    left: `${left}px`,
    width: `${panelWidth}px`,
    maxHeight: `${availableHeight}px`,
    top: opensBelow
      ? `${triggerRect.bottom + triggerGap}px`
      : `${Math.max(viewportTop + viewportPadding, triggerRect.top - triggerGap - visibleHeight)}px`,
    bottom: 'auto',
  }
}

function menuControls(): HTMLButtonElement[] {
  return Array.from(
    menuRef.value?.querySelectorAll<HTMLButtonElement>('[role="menuitem"]') ?? [],
  )
}

function closeMenu(restoreFocus = true): void {
  if (!menuOpen.value) return
  menuOpen.value = false
  menuQuery.value = ''
  if (restoreFocus) void nextTick(() => triggerRef.value?.focus({ preventScroll: true }))
}

function openMenu(focus: 'first' | 'last' = 'first'): void {
  if (triggerDisabled.value || menuOpen.value) return
  if (noticeTimer) clearTimeout(noticeTimer)
  selectionError.value = ''
  menuOpen.value = true
  void nextTick(() => {
    updateMenuPosition()
    const controls = menuControls()
    controls[focus === 'first' ? 0 : controls.length - 1]?.focus({ preventScroll: true })
  })
}

function toggleMenu(): void {
  if (menuOpen.value) closeMenu()
  else openMenu()
}

function onMenuItemSelect(item: ChatToolMenuItem): void {
  if (!item.available) {
    closeMenu(false)
    triggerRef.value?.focus({ preventScroll: true })
    showSelectionError(t('chat.tools.notSupportedAction', { name: item.label }))
    return
  }
  closeMenu(false)
  triggerRef.value?.focus({ preventScroll: true })
  if (item.id === 'library') emit('select-library')
  else open()
}

function onTriggerKeydown(event: KeyboardEvent): void {
  const opensFromShortcut = event.key === '@'
    && !event.altKey
    && !event.ctrlKey
    && !event.metaKey
  if (!opensFromShortcut && event.key !== 'ArrowDown' && event.key !== 'ArrowUp') return
  event.preventDefault()
  menuKeyboardFocus.value = true
  openMenu(event.key === 'ArrowUp' ? 'last' : 'first')
}

function onMenuKeydown(event: KeyboardEvent): void {
  menuKeyboardFocus.value = true
  const target = event.target as HTMLElement | null
  if (event.key === 'Escape') {
    event.preventDefault()
    closeMenu()
    return
  }
  if (target === searchInputRef.value) return
  if (event.key === '/') {
    event.preventDefault()
    searchInputRef.value?.focus()
    return
  }
  if (!['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) return

  const controls = menuControls()
  if (controls.length === 0) return
  event.preventDefault()
  if (event.key === 'Home' || event.key === 'End') {
    controls[event.key === 'Home' ? 0 : controls.length - 1]?.focus()
    return
  }
  const currentIndex = target ? controls.indexOf(target as HTMLButtonElement) : -1
  const direction = event.key === 'ArrowDown' ? 1 : -1
  const fallbackIndex = direction > 0 ? 0 : controls.length - 1
  const nextIndex = currentIndex < 0
    ? fallbackIndex
    : (currentIndex + direction + controls.length) % controls.length
  controls[nextIndex]?.focus()
}

function onDocumentPointerDown(event: PointerEvent): void {
  const target = event.target as Node | null
  if (
    !menuOpen.value
    || (target && triggerRef.value?.contains(target))
    || (target && menuSurfaceRef.value?.contains(target))
  ) return
  closeMenu(false)
}

function onDocumentFocusIn(event: FocusEvent): void {
  const target = event.target as Node | null
  if (
    !menuOpen.value
    || (target && triggerRef.value?.contains(target))
    || (target && menuSurfaceRef.value?.contains(target))
  ) return
  closeMenu(false)
}

function addMenuListeners(): void {
  document.addEventListener('pointerdown', onDocumentPointerDown, true)
  document.addEventListener('focusin', onDocumentFocusIn, true)
  window.addEventListener('resize', updateMenuPosition)
  window.addEventListener('scroll', updateMenuPosition, true)
  window.visualViewport?.addEventListener('resize', updateMenuPosition)
  window.visualViewport?.addEventListener('scroll', updateMenuPosition)
}

function removeMenuListeners(): void {
  document.removeEventListener('pointerdown', onDocumentPointerDown, true)
  document.removeEventListener('focusin', onDocumentFocusIn, true)
  window.removeEventListener('resize', updateMenuPosition)
  window.removeEventListener('scroll', updateMenuPosition, true)
  window.visualViewport?.removeEventListener('resize', updateMenuPosition)
  window.visualViewport?.removeEventListener('scroll', updateMenuPosition)
}

function open(): void {
  if (props.disabled || drafts.value.length >= CHAT_ATTACHMENT_MAX_COUNT) return
  closeMenu(false)
  inputRef.value?.click()
}

function onFileChange(event: Event): void {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files ?? [])
  input.value = ''
  if (files.length === 0) return
  addFiles(files)
}

function addFiles(files: File[]): void {
  closeMenu(false)
  if (props.disabled || files.length === 0) return
  for (const file of files) {
    if (drafts.value.length >= CHAT_ATTACHMENT_MAX_COUNT) {
      showSelectionError(t('chat.attachments.errors.tooMany', {
        count: CHAT_ATTACHMENT_MAX_COUNT,
      }))
      break
    }

    const otherTotalBytes = drafts.value.reduce((sum, draft) => sum + draft.file.size, 0)
    const validation = validateAttachmentFile(file, {
      supportsVision: props.supportsVision,
      otherTotalBytes,
      otherDocumentCount: drafts.value.filter((draft) => draft.kind === 'document').length,
    })
    const inferredKind = attachmentFileKind(file) ?? 'document'
    const key = makeKey()
    const previewUrl = imagePreview(file, inferredKind)
    if ('key' in validation) {
      if (validation.key === 'chat.attachments.errors.oneDocument') {
        showSelectionError(t(validation.key))
        continue
      }
      drafts.value = [
        ...drafts.value,
        {
          key,
          file,
          kind: inferredKind,
          state: 'error',
          progress: 0,
          ...(previewUrl ? { previewUrl } : {}),
          errorKey: validation.key,
          ...(validation.args ? { errorArgs: validation.args } : {}),
        },
      ]
      publish()
      continue
    }

    drafts.value = [
      ...drafts.value,
      {
        key,
        file,
        kind: validation.kind,
        source: 'upload',
        state: 'uploading',
        progress: 0,
        ...(previewUrl ? { previewUrl } : {}),
      },
    ]
    publish()
    void startUpload(key)
  }
}

function addLibraryFiles(files: LibraryFile[]): void {
  closeMenu(false)
  if (props.disabled || files.length === 0) return
  for (const file of files) {
    if (drafts.value.some((draft) => draft.libraryFileId === file.id)) continue
    if (drafts.value.length >= CHAT_ATTACHMENT_MAX_COUNT) {
      showSelectionError(t('chat.attachments.errors.tooMany', {
        count: CHAT_ATTACHMENT_MAX_COUNT,
      }))
      break
    }
    if (file.type === 'image' && !props.supportsVision) {
      showSelectionError(t('chat.attachments.errors.visionUnsupported'))
      continue
    }
    const selectedBytes = drafts.value.reduce((total, draft) => total + draft.file.size, 0)
    if (
      !Number.isFinite(file.size)
      || file.size < 0
      || selectedBytes > CHAT_ATTACHMENT_TOTAL_MAX_BYTES - file.size
    ) {
      showSelectionError(t('chat.attachments.errors.totalTooLarge', {
        size: Math.round(CHAT_ATTACHMENT_TOTAL_MAX_BYTES / 1024 / 1024),
      }))
      continue
    }
    const kind = file.type === 'image' ? 'image' : 'document'
    const syntheticFile = new File([], file.name, {
      type: file.mimeType,
      lastModified: Date.parse(file.updatedAt) || Date.now(),
    })
    Object.defineProperty(syntheticFile, 'size', { configurable: true, value: file.size })
    drafts.value = [
      ...drafts.value,
      {
        key: makeKey(),
        file: syntheticFile,
        kind,
        source: 'library',
        libraryFileId: file.id,
        state: 'ready',
        progress: 100,
        attachment: {
          id: file.id,
          name: file.name,
          kind,
          mimeType: file.mimeType,
          size: file.size,
          status: 'ready',
          // The chat attachment shape requires an expiry timestamp. Library
          // lifecycle is actually governed by its durable row, so use the same
          // far-future compatibility value as the backend alias.
          expiresAt: '2126-01-01T00:00:00Z',
          ...(file.pageCount ? { pageCount: file.pageCount } : {}),
          ...(file.width ? { width: file.width } : {}),
          ...(file.height ? { height: file.height } : {}),
        },
      },
    ]
  }
  publish()
}

async function startUpload(key: string): Promise<void> {
  const draft = drafts.value.find((candidate) => candidate.key === key)
  if (!draft) return
  const validation = validateAttachmentFile(draft.file, {
    supportsVision: props.supportsVision,
    otherTotalBytes: drafts.value.reduce(
      (sum, candidate) => sum + (candidate.key === key ? 0 : candidate.file.size),
      0,
    ),
    otherDocumentCount: drafts.value.filter(
      (candidate) => candidate.key !== key && candidate.kind === 'document',
    ).length,
  })
  if ('key' in validation) {
    replaceDraft(key, {
      state: 'error',
      progress: 0,
      errorKey: validation.key,
      errorArgs: validation.args,
    })
    return
  }

  controllers.get(key)?.abort()
  const controller = new AbortController()
  controllers.set(key, controller)
  replaceDraft(key, {
    kind: validation.kind,
    state: 'uploading',
    progress: 0,
    errorKey: undefined,
    errorArgs: undefined,
  })

  try {
    const attachment = await uploadChatAttachment(draft.file, {
      signal: controller.signal,
      onProgress: (progress) => {
        if (controllers.get(key) !== controller || controller.signal.aborted) return
        replaceDraft(key, {
          progress,
          state: progress >= 100 ? 'processing' : 'uploading',
        })
      },
    })
    if (controllers.get(key) !== controller || controller.signal.aborted) {
      void deleteChatAttachment(attachment.id).catch(() => undefined)
      return
    }
    if (!attachmentIsUsable(attachment)) {
      replaceDraft(key, {
        state: 'error',
        progress: 0,
        attachment,
        errorKey: 'chat.attachments.errors.expired',
      })
      return
    }
    replaceDraft(key, {
      state: 'ready',
      progress: 100,
      attachment,
      errorKey: undefined,
      errorArgs: undefined,
    })
  } catch (error) {
    if (controllers.get(key) !== controller) return
    if (controller.signal.aborted || isAbortError(error)) {
      const current = drafts.value.find((candidate) => candidate.key === key)
      if (current && current.state !== 'error') {
        replaceDraft(key, {
          state: 'error',
          progress: 0,
          errorKey: 'chat.attachments.errors.cancelled',
        })
      }
      return
    }
    replaceDraft(key, {
      state: 'error',
      progress: 0,
      errorKey: uploadErrorKey(error),
    })
  } finally {
    if (controllers.get(key) === controller) controllers.delete(key)
  }
}

function cancel(key: string): void {
  const controller = controllers.get(key)
  if (!controller) return
  replaceDraft(key, {
    state: 'error',
    progress: 0,
    errorKey: 'chat.attachments.errors.cancelled',
  })
  controller.abort()
}

function retry(key: string): void {
  if (!drafts.value.some((draft) => draft.key === key && draft.source !== 'library')) return
  void startUpload(key)
}

function remove(key: string): void {
  const draft = drafts.value.find((candidate) => candidate.key === key)
  if (!draft) return
  controllers.get(key)?.abort()
  controllers.delete(key)
  drafts.value = drafts.value.filter((candidate) => candidate.key !== key)
  revokePreview(draft)
  publish()
  if (draft.source !== 'library' && draft.attachment?.id) {
    void deleteChatAttachment(draft.attachment.id).catch(() => undefined)
  }
}

function getReadyAttachments(): ChatAttachment[] {
  const ready: ChatAttachment[] = []
  for (const draft of drafts.value) {
    if (draft.state !== 'ready' || !draft.attachment) continue
    if (!attachmentIsUsable(draft.attachment)) {
      replaceDraft(draft.key, {
        state: 'error',
        progress: 0,
        errorKey: 'chat.attachments.errors.expired',
      })
      continue
    }
    ready.push(draft.attachment)
  }
  return ready
}

function getReadySelection(): ChatAttachmentReadySelection {
  const attachments = getReadyAttachments()
  const readyDrafts = drafts.value.filter((draft) => (
    draft.state === 'ready' && Boolean(draft.attachment)
  ))
  return {
    attachments,
    uploadAttachmentIds: readyDrafts.flatMap((draft) => (
      draft.source === 'library' || !draft.attachment ? [] : [draft.attachment.id]
    )),
    libraryAttachments: readyDrafts.flatMap((draft) => (
      draft.source === 'library' && draft.libraryFileId
        ? [{ source: 'library' as const, fileId: draft.libraryFileId }]
        : []
    )),
  }
}

function commitAll(): void {
  const previous = drafts.value
  drafts.value = []
  for (const draft of previous) {
    controllers.get(draft.key)?.abort()
    controllers.delete(draft.key)
    revokePreview(draft)
  }
  publish()
}

async function discardAll(): Promise<void> {
  const previous = drafts.value
  drafts.value = []
  for (const draft of previous) {
    controllers.get(draft.key)?.abort()
    controllers.delete(draft.key)
    revokePreview(draft)
  }
  publish()
  await Promise.allSettled(previous
    .filter((draft) => draft.source !== 'library')
    .map((draft) => draft.attachment?.id)
    .filter((id): id is string => Boolean(id))
    .map((id) => deleteChatAttachment(id)))
}

onBeforeUnmount(() => {
  removeMenuListeners()
  if (noticeTimer) clearTimeout(noticeTimer)
  const previous = drafts.value
  drafts.value = []
  for (const draft of previous) {
    controllers.get(draft.key)?.abort()
    revokePreview(draft)
    if (draft.source !== 'library' && draft.attachment?.id) {
      void deleteChatAttachment(draft.attachment.id).catch(() => undefined)
    }
  }
  controllers.clear()
})

defineExpose({
  open,
  addFiles,
  addLibraryFiles,
  cancel,
  retry,
  remove,
  getReadyAttachments,
  getReadySelection,
  commitAll,
  discardAll,
})
</script>

<style scoped>
.chat-attachment-picker {
  position: relative;
  display: inline-flex;
  flex: none;
}

.chat-attachment-picker__input {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip: rect(0 0 0 0);
  clip-path: inset(50%);
  white-space: nowrap;
}

.chat-attachment-picker__trigger {
  display: grid;
  place-items: center;
  width: 36px;
  height: 36px;
  flex: none;
  border: 0;
  border-radius: 50%;
  color: var(--chat-composer-secondary-fg, var(--lx-clay-text));
  background: transparent;
  cursor: pointer;
  transition: color 150ms ease, background-color 150ms ease;
}

.chat-attachment-picker__trigger:hover:not(:disabled) {
  color: var(--chat-composer-secondary-fg, var(--lx-clay-text));
  background: var(--chat-composer-secondary-hover, var(--lx-clay-recessed));
}

.chat-attachment-picker__trigger--open {
  color: var(--chat-composer-primary-fg, var(--workspace-text));
  background: var(--chat-composer-secondary-hover, var(--workspace-hover));
}

.chat-attachment-picker__trigger:focus-visible {
  outline: 3px solid var(--lx-clay-accent-soft);
  outline-offset: 1px;
}

.chat-attachment-picker__trigger:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.chat-attachment-menu-popover {
  --chat-tool-menu-surface: var(--workspace-popup-surface);
  --chat-tool-menu-text: var(--workspace-text);
  --chat-tool-menu-description: var(--workspace-text-secondary);
  --chat-tool-menu-muted: var(--workspace-sidebar-group-label);
  --chat-tool-menu-hover: rgb(0 0 0 / 4%);

  z-index: 70;
  display: grid;
  grid-template-rows: minmax(0, 1fr) 36px;
  box-sizing: border-box;
  min-height: 0;
  overflow: hidden;
  border-radius: 20px;
  padding: 10px 10px 2px;
  color: var(--chat-tool-menu-text);
  background: var(--chat-tool-menu-surface);
  box-shadow:
    0 0 0 1px rgb(0 0 0 / 4%),
    0 2px 8px rgb(0 0 0 / 4%),
    0 4px 80px 8px rgb(0 0 0 / 2.4%);
  -webkit-font-smoothing: antialiased;
  transform-origin: left top;
}

.chat-attachment-menu-popover--above {
  transform-origin: left bottom;
}

.chat-attachment-menu {
  min-height: 0;
  overflow-x: hidden;
  overflow-y: auto;
  overscroll-behavior: contain;
  scrollbar-width: thin;
}

.chat-attachment-menu__item {
  position: relative;
  display: grid;
  grid-template-columns: 20px minmax(0, 1fr);
  width: 100%;
  min-height: 36px;
  align-items: center;
  gap: 6px;
  border: 0;
  border-radius: 12px;
  padding: 6px 10px;
  color: inherit;
  background: transparent;
  font: inherit;
  text-align: start;
  cursor: pointer;
  transition: background-color 120ms ease;
}

.chat-attachment-menu__item:hover,
.chat-attachment-menu__item:focus {
  background: var(--chat-tool-menu-hover);
}

.chat-attachment-menu__item:focus-visible {
  outline: none;
}

.chat-attachment-menu__item--unavailable {
  cursor: not-allowed;
}

.chat-attachment-menu__icon {
  display: grid;
  width: 20px;
  height: 20px;
  place-items: center;
  flex: none;
  color: var(--chat-tool-menu-text);
}

.chat-attachment-menu__icon > svg {
  width: 20px;
  height: 20px;
}

.chat-attachment-menu__icon--github,
.chat-attachment-menu__icon--figma,
.chat-attachment-menu__icon--heygen,
.chat-attachment-menu__icon--gmail {
  width: 20px;
  height: 20px;
  overflow: hidden;
  border-radius: 4px;
  color: #050505;
  background: #fff;
  box-shadow:
    inset 0 0 0 0.5px rgb(0 0 0 / 10%),
    0 4px 16px rgb(0 0 0 / 5%);
}

.chat-attachment-menu__icon--github > svg,
.chat-attachment-menu__icon--figma > svg,
.chat-attachment-menu__icon--heygen > svg,
.chat-attachment-menu__icon--gmail > svg {
  width: 20px;
  height: 20px;
}

.chat-attachment-menu__copy {
  display: flex;
  min-width: 0;
  align-items: baseline;
  gap: 12px;
  overflow: hidden;
  line-height: 20px;
  white-space: nowrap;
}

.chat-attachment-menu__label {
  flex: none;
  color: var(--chat-tool-menu-text);
  font-size: 14px;
  font-weight: 400;
  letter-spacing: normal;
}

.chat-attachment-menu__description {
  min-width: 0;
  overflow: hidden;
  color: var(--chat-tool-menu-description);
  font-size: 14px;
  font-weight: 400;
  letter-spacing: normal;
  text-overflow: ellipsis;
}

.chat-attachment-menu__item:hover .chat-attachment-menu__description,
.chat-attachment-menu__item:focus .chat-attachment-menu__description {
  color: var(--chat-tool-menu-text);
}

.chat-attachment-menu__status {
  position: absolute;
  top: 50%;
  right: 10px;
  min-width: 0;
  max-width: 92px;
  overflow: hidden;
  padding-inline-start: 8px;
  color: var(--chat-tool-menu-muted);
  background: var(--chat-tool-menu-hover);
  font-size: 14px;
  font-weight: 400;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
  opacity: 0;
  transform: translateY(-50%);
  transition: opacity 120ms ease;
}

.chat-attachment-menu__item:hover .chat-attachment-menu__status,
.chat-attachment-menu-popover--keyboard
  .chat-attachment-menu__item:focus
  .chat-attachment-menu__status {
  opacity: 1;
}

.chat-attachment-menu__status--persistent {
  background: transparent;
  opacity: 1;
}

.chat-attachment-menu__empty {
  display: grid;
  min-height: 72px;
  place-items: center;
  padding: 12px;
  color: var(--chat-tool-menu-description);
  font-size: 14px;
  line-height: 20px;
  text-align: center;
}

.chat-attachment-menu__search {
  display: flex;
  min-width: 0;
  align-items: center;
}

.chat-attachment-menu__search input {
  width: 100%;
  min-width: 0;
  height: 36px;
  border: 0;
  border-radius: 10px;
  padding: 0 10px;
  color: var(--chat-tool-menu-text);
  background: transparent;
  font: inherit;
  font-size: 14px;
  font-weight: 400;
  line-height: 20px;
  outline: none;
}

.chat-attachment-menu__search input::placeholder {
  color: var(--chat-tool-menu-muted);
  opacity: 1;
}

.chat-attachment-menu__search input:focus {
  background: var(--chat-tool-menu-hover);
}

.chat-attachment-menu__search input::-webkit-search-cancel-button {
  cursor: pointer;
}

.chat-attachment-menu-enter-active,
.chat-attachment-menu-leave-active {
  transition:
    opacity 120ms ease,
    transform 170ms cubic-bezier(0.22, 1, 0.36, 1);
}

.chat-attachment-menu-enter-from,
.chat-attachment-menu-leave-to {
  opacity: 0;
  transform: translateY(-5px) scale(0.99);
}

.chat-attachment-menu-enter-from.chat-attachment-menu-popover--above,
.chat-attachment-menu-leave-to.chat-attachment-menu-popover--above {
  transform: translateY(5px) scale(0.99);
}

:global(html.dark) .chat-attachment-menu-popover {
  --chat-tool-menu-surface: #212121;
  --chat-tool-menu-text: #ffffff;
  --chat-tool-menu-description: #cdcdcd;
  --chat-tool-menu-muted: #afafaf;
  --chat-tool-menu-hover: rgb(255 255 255 / 10%);

  box-shadow:
    inset 0 0 1px rgb(255 255 255 / 20%);
}

:global(html.dark) .chat-attachment-menu__search input::placeholder {
  color: rgb(175 175 175 / 70%);
}

.chat-attachment-picker__notice {
  position: absolute;
  z-index: 12;
  left: 0;
  bottom: calc(100% + 9px);
  width: max-content;
  max-width: min(280px, calc(100vw - 32px));
  border: 1px solid color-mix(in srgb, var(--lx-clay-danger) 35%, transparent);
  border-radius: 8px;
  padding: 8px 10px;
  color: var(--lx-clay-danger);
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-flat);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  line-height: 1.4;
}

@media (max-width: 720px) {
  .chat-attachment-picker__trigger {
    width: 36px;
    height: 36px;
  }

  .chat-attachment-menu-popover {
    grid-template-rows: minmax(0, 1fr) 36px;
    padding: 8px;
  }

  .chat-attachment-menu__item {
    min-height: 44px;
    gap: 10px;
    padding-inline: 10px;
  }

  .chat-attachment-menu__search input {
    height: 36px;
  }

  .chat-attachment-menu__status {
    max-width: 72px;
    font-size: 12px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .chat-attachment-picker__trigger,
  .chat-attachment-menu__item,
  .chat-attachment-menu__status,
  .chat-attachment-menu-enter-active,
  .chat-attachment-menu-leave-active {
    transition: none;
  }
}

@media (forced-colors: active) {
  .chat-attachment-menu__item:focus-visible {
    outline: 1.5px solid CanvasText;
    outline-offset: -2px;
  }
}
</style>

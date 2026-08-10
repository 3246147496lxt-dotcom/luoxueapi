<template>
  <section
    v-if="items.length"
    class="chat-attachment-preview"
    :aria-label="t('chat.attachments.selected')"
  >
    <ul class="chat-attachment-preview__list" role="list">
      <li
        v-for="item in items"
        :key="item.key"
        class="chat-attachment-preview__item"
        :class="{
          'chat-attachment-preview__item--error': item.state === 'error'
            || (item.kind === 'image' && !supportsVision),
        }"
      >
        <div class="chat-attachment-preview__visual" aria-hidden="true">
          <img
            v-if="item.kind === 'image' && item.previewUrl"
            :src="item.previewUrl"
            alt=""
          />
          <Icon v-else :name="item.kind === 'image' ? 'photo' : 'document'" size="md" />
        </div>

        <div class="chat-attachment-preview__content">
          <strong :title="item.file.name">{{ item.file.name }}</strong>
          <span class="chat-attachment-preview__meta">
            {{ formatAttachmentBytes(item.file.size) }}
          </span>
          <span
            class="chat-attachment-preview__status"
            :class="{
              'chat-attachment-preview__status--error': item.state === 'error'
                || (item.kind === 'image' && !supportsVision),
            }"
            aria-live="polite"
          >
            {{ statusText(item) }}
          </span>
          <span
            v-if="isDocx(item) && item.state === 'ready'"
            class="chat-attachment-preview__hint"
          >
            {{ t('chat.attachments.docxTextOnly') }}
          </span>
          <progress
            v-if="item.state === 'uploading'"
            class="chat-attachment-preview__progress"
            max="100"
            :value="item.progress"
            :aria-label="t('chat.attachments.uploadProgress', { progress: item.progress })"
          ></progress>
        </div>

        <div class="chat-attachment-preview__actions">
          <button
            v-if="item.state === 'uploading' || item.state === 'processing'"
            type="button"
            :aria-label="t('chat.attachments.cancel', { name: item.file.name })"
            :title="t('chat.attachments.cancel', { name: item.file.name })"
            @click="$emit('cancel', item.key)"
          >
            <Icon name="x" size="xs" />
          </button>
          <button
            v-if="item.state === 'error'"
            type="button"
            :aria-label="t('chat.attachments.retry', { name: item.file.name })"
            :title="t('chat.attachments.retry', { name: item.file.name })"
            @click="$emit('retry', item.key)"
          >
            <Icon name="refresh" size="xs" />
          </button>
          <button
            v-if="item.state === 'ready' || item.state === 'error'"
            type="button"
            :aria-label="t('chat.attachments.remove', { name: item.file.name })"
            :title="t('chat.attachments.remove', { name: item.file.name })"
            @click="$emit('remove', item.key)"
          >
            <Icon name="x" size="xs" />
          </button>
        </div>
      </li>
    </ul>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import {
  formatAttachmentBytes,
  type ChatAttachmentDraft,
} from './chatAttachmentUi'

const props = withDefaults(defineProps<{
  items: ChatAttachmentDraft[]
  supportsVision?: boolean
}>(), {
  supportsVision: false,
})

defineEmits<{
  cancel: [key: string]
  retry: [key: string]
  remove: [key: string]
}>()

const { t } = useI18n()

function statusText(item: ChatAttachmentDraft): string {
  if (item.kind === 'image' && !props.supportsVision) {
    return t('chat.attachments.errors.visionUnsupported')
  }
  if (item.state === 'uploading') {
    return t('chat.attachments.uploading', { progress: item.progress })
  }
  if (item.state === 'processing') return t('chat.attachments.processing')
  if (item.state === 'ready') return t('chat.attachments.ready')
  return t(item.errorKey ?? 'chat.attachments.errors.uploadFailed', item.errorArgs ?? {})
}

function isDocx(item: ChatAttachmentDraft): boolean {
  return item.file.name.toLowerCase().endsWith('.docx')
}
</script>

<style scoped>
.chat-attachment-preview {
  width: 100%;
  margin: 0;
}

.chat-attachment-preview__list {
  display: flex;
  gap: 8px;
  margin: 0;
  padding: 0 1px 4px;
  overflow-x: auto;
  list-style: none;
  scrollbar-width: thin;
}

.chat-attachment-preview__item {
  display: grid;
  grid-template-columns: 48px minmax(0, 1fr) auto;
  gap: 9px;
  align-items: center;
  width: min(280px, 78vw);
  min-width: 230px;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-control);
  padding: 7px;
  color: var(--lx-clay-text);
  background: var(--lx-clay-recessed);
  box-shadow: none;
}

.chat-attachment-preview__item--error {
  border-color: color-mix(in srgb, var(--lx-clay-danger) 42%, var(--lx-clay-border));
}

.chat-attachment-preview__visual {
  display: grid;
  place-items: center;
  width: 48px;
  height: 48px;
  overflow: hidden;
  border-radius: 7px;
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

.chat-attachment-preview__visual img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.chat-attachment-preview__content {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.chat-attachment-preview__content strong {
  overflow: hidden;
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-attachment-preview__meta,
.chat-attachment-preview__status,
.chat-attachment-preview__hint {
  color: var(--lx-clay-text-muted);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  line-height: 1.3;
}

.chat-attachment-preview__hint {
  white-space: normal;
}

.chat-attachment-preview__status--error {
  color: var(--lx-clay-danger);
}

.chat-attachment-preview__progress {
  width: 100%;
  height: 3px;
  overflow: hidden;
  border: 0;
  border-radius: 999px;
  accent-color: var(--lx-clay-accent);
}

.chat-attachment-preview__actions {
  display: flex;
  align-self: start;
  gap: 2px;
}

.chat-attachment-preview__actions button {
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  border: 0;
  border-radius: 50%;
  color: var(--lx-clay-text-muted);
  background: transparent;
  cursor: pointer;
}

.chat-attachment-preview__actions button:hover {
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

.chat-attachment-preview__actions button:focus-visible {
  outline: 3px solid var(--lx-clay-accent-soft);
}

@media (max-width: 640px) {
  .chat-attachment-preview__list {
    padding-inline: 1px;
    scroll-snap-type: x proximity;
  }

  .chat-attachment-preview__item {
    min-width: min(255px, 82vw);
    scroll-snap-align: start;
  }
}
</style>

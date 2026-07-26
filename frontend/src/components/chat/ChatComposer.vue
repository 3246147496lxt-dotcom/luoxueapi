<template>
  <div class="chat-composer-wrap">
    <div v-if="insufficientBalance" class="chat-composer__balance" role="status">
      <span>
        <Icon name="exclamationCircle" size="sm" />
        {{ t('chat.balance.insufficient') }}
      </span>
      <router-link to="/purchase">{{ t('chat.balance.recharge') }}</router-link>
    </div>

    <form class="chat-composer" @submit.prevent="submit">
      <textarea
        ref="textareaRef"
        v-model="draft"
        rows="1"
        :maxlength="maxLength"
        :placeholder="placeholder"
        :disabled="disabled || insufficientBalance"
        :aria-label="t('chat.composer.label')"
        @input="resize"
        @keydown="onKeydown"
      ></textarea>

      <button
        v-if="streaming"
        type="button"
        class="chat-composer__action chat-composer__action--stop"
        :aria-label="t('chat.actions.stop')"
        :title="t('chat.actions.stop')"
        @click="$emit('stop')"
      >
        <span aria-hidden="true"></span>
      </button>
      <button
        v-else
        type="submit"
        class="chat-composer__action chat-composer__action--send"
        :disabled="!canSubmit"
        :aria-label="t('chat.actions.send')"
        :title="t('chat.actions.send')"
      >
        <Icon name="send" size="sm" :stroke-width="2" />
      </button>
    </form>

    <div class="chat-composer__footer">
      <span class="chat-composer__notice">
        <Icon name="infoCircle" size="xs" />
        <span>{{ t('chat.composer.disclaimer') }}</span>
      </span>
      <span>{{ draft.length }}/{{ maxLength }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const props = withDefaults(defineProps<{
  modelValue?: string
  streaming?: boolean
  disabled?: boolean
  insufficientBalance?: boolean
  maxLength?: number
}>(), {
  modelValue: '',
  streaming: false,
  disabled: false,
  insufficientBalance: false,
  maxLength: 20_000,
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
  send: [value: string]
  stop: []
}>()

const { t } = useI18n()
const textareaRef = ref<HTMLTextAreaElement | null>(null)
const draft = ref(props.modelValue)

const canSubmit = computed(() => (
  draft.value.trim().length > 0
  && !props.disabled
  && !props.insufficientBalance
  && !props.streaming
))

const placeholder = computed(() => (
  props.insufficientBalance
    ? t('chat.composer.rechargePlaceholder')
    : t('chat.composer.placeholder')
))

watch(() => props.modelValue, (value) => {
  if (value === draft.value) return
  draft.value = value
  void nextTick(resize)
})

watch(draft, (value) => emit('update:modelValue', value))

function resize() {
  const textarea = textareaRef.value
  if (!textarea) return
  textarea.style.height = 'auto'
  textarea.style.height = `${Math.min(textarea.scrollHeight, 180)}px`
}

function onKeydown(event: KeyboardEvent) {
  if (event.key !== 'Enter' || event.shiftKey || event.isComposing) return
  event.preventDefault()
  submit()
}

function submit() {
  const value = draft.value.trim()
  if (!canSubmit.value || !value) return
  emit('send', value)
  draft.value = ''
  void nextTick(() => {
    resize()
    textareaRef.value?.focus()
  })
}

function focus() {
  textareaRef.value?.focus()
}

defineExpose({ focus })
</script>

<style scoped>
.chat-composer-wrap {
  width: min(100%, 860px);
  margin: 0 auto;
}

.chat-composer__balance {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 8px;
  border: 1px solid color-mix(in srgb, var(--lx-clay-warning) 32%, transparent);
  border-radius: 8px;
  padding: 9px 12px;
  color: var(--lx-clay-text);
  background: var(--lx-clay-warning-soft);
  font-size: 13px;
}

.chat-composer__balance span {
  display: flex;
  align-items: center;
  gap: 7px;
}

.chat-composer__balance a {
  flex: none;
  color: var(--lx-clay-accent-deep);
  font-weight: 800;
  text-decoration: none;
}

.chat-composer {
  display: flex;
  align-items: flex-end;
  gap: 12px;
  border: 1px solid var(--lx-clay-border-strong);
  border-radius: 8px;
  padding: 10px 10px 10px 18px;
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-form);
  transition: border-color 150ms ease, box-shadow 150ms ease;
}

.chat-composer:focus-within {
  border-color: color-mix(in srgb, var(--lx-clay-accent) 62%, transparent);
  box-shadow: var(--lx-clay-shadow-form), 0 0 0 3px var(--lx-clay-accent-soft);
}

.chat-composer textarea {
  min-width: 0;
  min-height: 36px;
  max-height: 180px;
  flex: 1;
  resize: none;
  border: 0;
  padding: 7px 0 5px;
  color: var(--lx-clay-text);
  background: transparent;
  font-family: var(--lx-clay-font-ui);
  font-size: 15px;
  line-height: 1.5;
  outline: none;
}

.chat-composer textarea::placeholder {
  color: var(--lx-clay-text-muted);
}

.chat-composer textarea:disabled {
  cursor: not-allowed;
}

.chat-composer__action {
  display: grid;
  place-items: center;
  width: 44px;
  height: 44px;
  flex: 0 0 44px;
  border: 0;
  border-radius: 8px;
  color: var(--lx-clay-on-accent);
  background: var(--lx-clay-accent);
  box-shadow: var(--lx-clay-shadow-action);
  transition: transform 150ms ease, opacity 150ms ease;
}

.chat-composer__action:hover:not(:disabled) {
  transform: translateY(-1px);
}

.chat-composer__action:disabled {
  cursor: not-allowed;
  opacity: 0.38;
  box-shadow: none;
}

.chat-composer__action--stop span {
  width: 12px;
  height: 12px;
  border-radius: 2px;
  background: currentColor;
}

.chat-composer__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 22px;
  padding: 8px 3px 0;
  color: var(--lx-clay-text-muted);
  font-size: 11px;
}

.chat-composer__notice {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 6px;
}

.chat-composer__notice > span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 640px) {
  .chat-composer__balance {
    align-items: flex-start;
  }

  .chat-composer__notice > span {
    white-space: normal;
  }
}

@media (prefers-reduced-motion: reduce) {
  .chat-composer,
  .chat-composer__action {
    transition: none;
  }
}
</style>

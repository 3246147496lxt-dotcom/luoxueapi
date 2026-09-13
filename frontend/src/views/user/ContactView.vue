<template>
  <AppLayout>
    <div class="contact-page" data-testid="contact-page">
      <header class="contact-page__header">
        <h1>{{ t('contact.title') }}</h1>
        <p>{{ t('contact.description') }}</p>
      </header>

      <section class="contact-grid" :aria-label="t('contact.title')">
        <article
          v-for="channel in channels"
          :key="channel.id"
          class="contact-card"
          :data-testid="`contact-card-${channel.id}`"
        >
          <div class="contact-card__heading">
            <div class="contact-card__icon" :class="`contact-card__icon--${channel.id}`">
              <Icon :name="channel.icon" size="lg" aria-hidden="true" />
            </div>
            <div>
              <h2>{{ t(channel.titleKey) }}</h2>
              <p>{{ t(channel.subtitleKey) }}</p>
            </div>
          </div>

          <div v-if="channel.id === 'wechat'" class="contact-qr-block">
            <div class="contact-qr-placeholder" aria-hidden="true">
              <Icon name="grid" size="xl" />
              <span>{{ t('contact.qrPlaceholder') }}</span>
            </div>
            <p>{{ t('contact.channels.wechat.helper') }}</p>
          </div>

          <div v-else class="contact-card__details">
            <span class="contact-card__value">{{ channel.value || t(channel.placeholderKey) }}</span>
            <p>{{ t(channel.helperKey) }}</p>
          </div>

          <button
            v-if="channel.action === 'copy'"
            type="button"
            class="contact-card__action contact-card__action--secondary"
            :disabled="!channel.value"
            @click="copyValue(channel.value)"
          >
            <Icon name="copy" size="sm" aria-hidden="true" />
            {{ copiedValue === channel.value ? t('contact.actions.copied') : t('contact.actions.copy') }}
          </button>
          <button
            v-else-if="channel.action === 'saveQr'"
            type="button"
            class="contact-card__action contact-card__action--secondary"
            disabled
          >
            <Icon name="download" size="sm" aria-hidden="true" />
            {{ t('contact.actions.saveQr') }}
          </button>
          <button
            v-else
            type="button"
            class="contact-card__action contact-card__action--primary"
            disabled
          >
            <Icon name="externalLink" size="sm" aria-hidden="true" />
            {{ t(channel.actionKey) }}
          </button>
        </article>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'

type ContactChannel = {
  id: 'qq' | 'qqGroup' | 'wechat' | 'telegram'
  icon: 'chat' | 'users' | 'chatBubble' | 'send'
  titleKey: string
  subtitleKey: string
  helperKey: string
  placeholderKey: string
  action: 'copy' | 'saveQr' | 'join'
  actionKey: string
  value: string
}

const { t } = useI18n()
const copiedValue = ref('')

// Replace these four placeholders when the support accounts are ready.
const channels: ContactChannel[] = [
  {
    id: 'qq',
    icon: 'chat',
    titleKey: 'contact.channels.qq.title',
    subtitleKey: 'contact.channels.qq.subtitle',
    helperKey: 'contact.channels.qq.helper',
    placeholderKey: 'contact.channels.qq.placeholder',
    action: 'copy',
    actionKey: 'contact.actions.copy',
    value: '',
  },
  {
    id: 'qqGroup',
    icon: 'users',
    titleKey: 'contact.channels.qqGroup.title',
    subtitleKey: 'contact.channels.qqGroup.subtitle',
    helperKey: 'contact.channels.qqGroup.helper',
    placeholderKey: 'contact.channels.qqGroup.placeholder',
    action: 'join',
    actionKey: 'contact.actions.joinGroup',
    value: '',
  },
  {
    id: 'wechat',
    icon: 'chatBubble',
    titleKey: 'contact.channels.wechat.title',
    subtitleKey: 'contact.channels.wechat.subtitle',
    helperKey: 'contact.channels.wechat.helper',
    placeholderKey: 'contact.qrPlaceholder',
    action: 'saveQr',
    actionKey: 'contact.actions.saveQr',
    value: '',
  },
  {
    id: 'telegram',
    icon: 'send',
    titleKey: 'contact.channels.telegram.title',
    subtitleKey: 'contact.channels.telegram.subtitle',
    helperKey: 'contact.channels.telegram.helper',
    placeholderKey: 'contact.channels.telegram.placeholder',
    action: 'join',
    actionKey: 'contact.actions.joinGroup',
    value: '',
  },
]

async function copyValue(value: string): Promise<void> {
  if (!value || !navigator.clipboard) return
  await navigator.clipboard.writeText(value)
  copiedValue.value = value
  window.setTimeout(() => {
    if (copiedValue.value === value) copiedValue.value = ''
  }, 1800)
}
</script>

<style scoped>
.contact-page {
  width: min(100%, 920px);
  margin: 0 auto;
  padding: 56px 0 24px;
}

.contact-page__header {
  margin-bottom: 28px;
}

.contact-page__header h1 {
  margin: 0;
  color: var(--workspace-text);
  font-size: 28px;
  font-weight: 600;
  letter-spacing: -0.02em;
  line-height: 1.25;
}

.contact-page__header p {
  margin: 8px 0 0;
  color: var(--workspace-text-secondary);
  font-size: 14px;
  line-height: 1.6;
}

.contact-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.contact-card {
  display: flex;
  min-height: 250px;
  flex-direction: column;
  padding: 24px;
  border: 1px solid var(--workspace-border);
  border-radius: var(--workspace-radius-card);
  background: var(--workspace-surface);
  box-shadow: var(--workspace-light-shadow-card);
}

.contact-card__heading {
  display: flex;
  align-items: center;
  gap: 12px;
}

.contact-card__icon {
  display: inline-flex;
  width: 40px;
  height: 40px;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  border-radius: 12px;
  color: var(--workspace-work-accent);
  background: var(--workspace-work-accent-soft);
}

.contact-card__heading h2 {
  margin: 0;
  color: var(--workspace-text);
  font-size: 16px;
  font-weight: 600;
  line-height: 1.4;
}

.contact-card__heading p,
.contact-card__details p,
.contact-qr-block p {
  margin: 4px 0 0;
  color: var(--workspace-text-muted);
  font-size: 12px;
  line-height: 1.5;
}

.contact-card__details {
  flex: 1;
  padding-top: 28px;
}

.contact-card__value {
  color: var(--workspace-text);
  font-size: 15px;
  font-weight: 550;
}

.contact-card__action {
  display: inline-flex;
  min-height: 44px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
  margin-top: 24px;
  border-radius: var(--workspace-radius-button);
  padding: 0 16px;
  font-size: 14px;
  font-weight: 600;
  transition: background-color 150ms ease, border-color 150ms ease, color 150ms ease;
}

.contact-card__action:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.contact-card__action--primary {
  border: 1px solid var(--workspace-work-accent);
  color: var(--workspace-work-on-accent);
  background: var(--workspace-work-accent);
}

.contact-card__action--secondary {
  border: 1px solid var(--workspace-border);
  color: var(--workspace-text);
  background: var(--workspace-surface);
}

.contact-qr-block {
  display: flex;
  flex: 1;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding-top: 16px;
  text-align: center;
}

.contact-qr-placeholder {
  display: flex;
  width: 112px;
  height: 112px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border: 1px dashed var(--workspace-border-strong);
  border-radius: 12px;
  color: var(--workspace-text-muted);
  background: var(--workspace-surface-subtle);
  font-size: 10px;
}

@media (max-width: 767px) {
  .contact-page {
    padding: 32px 0 16px;
  }

  .contact-grid {
    grid-template-columns: 1fr;
  }
}
</style>

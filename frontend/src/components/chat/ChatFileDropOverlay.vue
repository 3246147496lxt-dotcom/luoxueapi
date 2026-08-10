<template>
  <p class="sr-only" aria-live="polite" aria-atomic="true">
    {{ active ? `${t('chat.attachments.dropHere')} ${t('chat.attachments.dropDescription')}` : '' }}
  </p>

  <Teleport to="body">
    <Transition name="chat-file-drop">
      <div
        v-if="active"
        class="chat-file-drop-overlay"
        data-test="chat-file-drop-overlay"
        aria-hidden="true"
      >
        <svg
          class="chat-file-drop-overlay__art"
          viewBox="0 0 132 108"
          width="132"
          height="108"
          fill="none"
          aria-hidden="true"
          focusable="false"
        >
          <path
            class="chat-file-drop-overlay__card chat-file-drop-overlay__card--code"
            fill-rule="evenodd"
            clip-rule="evenodd"
            d="M25.203 29.351C10.778 33.217 8.515 37.136 11.828 49.5l1.657 6.181c3.312 12.364 7.232 14.627 21.656 10.762l8.243-2.209c14.424-3.865 16.687-7.784 13.374-20.148l-1.656-6.182c-3.313-12.363-7.232-14.626-21.657-10.761l-8.242 2.208Zm-7.009 13.373a2.4 2.4 0 0 1 3.279-.878l5.879 3.394a2.4 2.4 0 0 1 .878 3.279l-3.394 5.878a2.4 2.4 0 1 1-4.157-2.4l2.194-3.8-3.8-2.194a2.4 2.4 0 0 1-.879-3.279Zm11.215 13.66a2.4 2.4 0 0 1 1.697-2.939l9.273-2.485a2.4 2.4 0 1 1 1.242 4.637l-9.273 2.484a2.4 2.4 0 0 1-2.939-1.697Z"
          />
          <path
            class="chat-file-drop-overlay__card chat-file-drop-overlay__card--text"
            fill-rule="evenodd"
            clip-rule="evenodd"
            d="M86.812 13.404c-5.715-1.532-8.245-.139-9.798 5.656l-8.282 30.91c-1.553 5.796-.139 8.245 5.738 9.82l23.02 6.168c5.877 1.575 8.326.16 9.916-5.773l7.987-29.806a1 1 0 0 0-.712-1.219c-1.203-.305-2.246-.54-3.139-.742-5.299-1.193-5.322-1.198-2.093-7.702a1 1 0 0 0-.633-1.416l-22.004-5.896Zm.446 15.027a1.92 1.92 0 0 0-.994 3.709l14.837 3.976a1.92 1.92 0 1 0 .994-3.709l-14.837-3.976Zm-4.339 8.776a1.92 1.92 0 0 1 2.351-1.357l14.837 3.975a1.92 1.92 0 1 1-.994 3.71L84.277 39.56a1.92 1.92 0 0 1-1.358-2.352Zm.364 6.061a1.92 1.92 0 0 0-.994 3.71l7.418 1.987a1.92 1.92 0 1 0 .994-3.709l-7.418-1.988Z"
          />
          <path
            class="chat-file-drop-overlay__card chat-file-drop-overlay__card--image"
            fill-rule="evenodd"
            clip-rule="evenodd"
            d="M40.4 71.843c0-14.629 3.658-18.286 20.724-18.286h9.753c17.066 0 20.723 3.657 20.723 18.286v7.314c0 14.628-3.657 18.286-20.723 18.286h-9.753C44.058 97.443 40.4 93.785 40.4 79.157v-7.314ZM78.8 67.5a4.8 4.8 0 1 1-9.6 0 4.8 4.8 0 0 1 9.6 0Zm-18.08 3.36a2.4 2.4 0 0 0-3.84 0l-9.6 12.8a2.4 2.4 0 1 0 3.84 2.88l7.68-10.24 7.68 10.24a2.4 2.4 0 0 0 3.617.257l4.703-4.703 4.703 4.703a2.4 2.4 0 0 0 3.394-3.394l-6.4-6.4a2.4 2.4 0 0 0-3.394 0l-4.443 4.443-7.94-10.586Z"
          />
        </svg>
        <h3>{{ t('chat.attachments.dropHere') }}</h3>
        <p>{{ t('chat.attachments.dropDescription') }}</p>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

withDefaults(defineProps<{
  active?: boolean
}>(), {
  active: false,
})

const { t } = useI18n()
</script>

<style scoped>
.chat-file-drop-overlay {
  position: fixed;
  z-index: 51;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--lx-clay-text);
  background: color-mix(in srgb, var(--lx-clay-surface) 80%, transparent);
  pointer-events: none;
}

.chat-file-drop-overlay__art {
  --chat-file-card-back: color-mix(
    in srgb,
    var(--lx-clay-light-accent-highlight) 64%,
    var(--lx-clay-light-surface)
  );
  --chat-file-card-middle: var(--lx-clay-light-accent-highlight);
  --chat-file-card-front: var(--lx-clay-light-accent);

  display: block;
  width: 132px;
  height: 108px;
  flex: none;
  overflow: visible;
}

.chat-file-drop-overlay__card--code {
  fill: var(--chat-file-card-back);
}

.chat-file-drop-overlay__card--text {
  fill: var(--chat-file-card-middle);
}

.chat-file-drop-overlay__card--image {
  fill: var(--chat-file-card-front);
}

.chat-file-drop-overlay h3,
.chat-file-drop-overlay p {
  margin: 0;
  text-align: center;
}

.chat-file-drop-overlay h3 {
  font-size: var(--workspace-type-page-title-size);
  font-weight: var(--workspace-type-page-title-weight);
  line-height: 32px;
}

.chat-file-drop-overlay p {
  width: 66.6667%;
  color: var(--lx-clay-text);
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
  line-height: 24px;
}

.chat-file-drop-enter-active {
  transition: opacity 140ms ease-out;
}

.chat-file-drop-leave-active {
  transition: opacity 100ms ease-in;
}

.chat-file-drop-enter-active > * {
  transition:
    opacity 170ms ease-out,
    transform 190ms cubic-bezier(0.2, 0.8, 0.2, 1);
}

.chat-file-drop-enter-from,
.chat-file-drop-leave-to,
.chat-file-drop-enter-from > * {
  opacity: 0;
}

.chat-file-drop-enter-from > * {
  transform: translateY(6px) scale(0.97);
}

@media (max-width: 480px) {
  .chat-file-drop-overlay p {
    width: calc(100% - 48px);
  }
}

@media (prefers-reduced-motion: reduce) {
  .chat-file-drop-enter-active,
  .chat-file-drop-leave-active,
  .chat-file-drop-enter-active > * {
    transition: none;
  }
}
</style>

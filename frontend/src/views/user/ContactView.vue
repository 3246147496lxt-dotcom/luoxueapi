<template>
  <div class="contact-screen" data-testid="contact-page">
    <header class="contact-header">
      <div class="contact-content contact-header__inner">
        <div class="contact-brand">
          <div class="contact-brand__mark" aria-hidden="true">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
              <path d="M12 2v20M4.93 4.93l14.14 14.14M2 12h20M4.93 19.07 19.07 4.93" />
              <path d="m8.5 5.5 3.5 3.5 3.5-3.5M8.5 18.5 12 15l3.5 3.5M5.5 8.5 9 12l-3.5 3.5M18.5 8.5 15 12l3.5 3.5" />
            </svg>
          </div>
          <div class="contact-brand__text">
            <span class="contact-brand__name">落雪API</span>
            <span class="contact-brand__badge">支持中心</span>
          </div>
        </div>

        <a class="contact-back-link" href="/dashboard">
          <Icon name="arrowLeft" size="sm" aria-hidden="true" />
          返回主页
        </a>
      </div>
    </header>

    <main class="contact-main">
      <div class="contact-content">
        <div class="contact-title-row">
          <h1>联系我们</h1>
          <p>遇到问题？我们的支持团队随时准备为您提供帮助。</p>
        </div>

        <section class="contact-grid" aria-label="联系我们">
          <article class="contact-card" data-testid="contact-card-qq">
            <div class="contact-card__heading">
              <div class="contact-card__icon">
                <Icon name="chat" size="lg" aria-hidden="true" />
              </div>
              <h2>QQ客服</h2>
            </div>
            <div class="contact-value-row">
              <span class="contact-value">客服QQ：2456772148</span>
              <CopyButton label="复制QQ号" :copied="copiedChannel === 'qq'" @copy="copyContact('qq', '2456772148')" />
            </div>
          </article>

          <article class="contact-card" data-testid="contact-card-qqGroup">
            <div class="contact-card__heading">
              <div class="contact-card__icon">
                <Icon name="users" size="lg" aria-hidden="true" />
              </div>
              <h2>QQ群</h2>
            </div>
            <div class="contact-value-row">
              <span class="contact-value">群号：1072675573</span>
              <CopyButton label="复制群号" :copied="copiedChannel === 'qqGroup'" @copy="copyContact('qqGroup', '1072675573')" />
            </div>
          </article>

          <article class="contact-card contact-card--wechat" data-testid="contact-card-wechat">
            <div class="contact-card__heading">
              <div class="contact-card__icon">
                <Icon name="chatBubble" size="lg" aria-hidden="true" />
              </div>
              <h2>微信客服</h2>
            </div>
            <div class="contact-wechat-body">
              <div class="contact-qr-frame">
                <img src="/contact-wechat.jpg" alt="微信二维码" />
              </div>
              <div class="contact-wechat-meta">
                <span class="contact-label">微信号</span>
                <div class="contact-wechat-value-row">
                  <span class="contact-value contact-value--wechat">Xxxttt_111</span>
                  <CopyButton label="复制微信号" compact :copied="copiedChannel === 'wechat'" @copy="copyContact('wechat', 'Xxxttt_111')" />
                </div>
              </div>
            </div>
          </article>

          <article class="contact-card" data-testid="contact-card-telegram">
            <div class="contact-card__heading">
              <div class="contact-card__icon">
                <Icon name="send" size="lg" aria-hidden="true" />
              </div>
              <h2>Telegram群</h2>
            </div>
            <div class="contact-telegram-value">链接：<span>https://t.me/+g4eFhwpWMukyNGVl</span></div>
            <a class="contact-telegram-button" href="https://t.me/+g4eFhwpWMukyNGVl" target="_blank" rel="noopener noreferrer">
              <Icon name="externalLink" size="sm" aria-hidden="true" />
              立即加入
            </a>
          </article>
        </section>
      </div>
    </main>

    <footer class="contact-footer">
      <div class="contact-content">© 落雪API · 支持中心</div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { defineComponent, h, onBeforeUnmount, ref } from 'vue'
import Icon from '@/components/icons/Icon.vue'

type CopyChannel = 'qq' | 'qqGroup' | 'wechat'
const copiedChannel = ref<CopyChannel | ''>('')
let copyResetTimer: number | undefined

async function copyContact(channel: CopyChannel, value: string): Promise<void> {
  let copied = false
  try {
    if (navigator.clipboard) {
      await navigator.clipboard.writeText(value)
      copied = true
    }
  } catch {
    copied = false
  }

  if (!copied) {
    const input = document.createElement('textarea')
    input.value = value
    input.setAttribute('readonly', '')
    input.style.position = 'fixed'
    input.style.opacity = '0'
    document.body.appendChild(input)
    input.select()
    copied = document.execCommand('copy')
    input.remove()
  }

  if (!copied) return
  copiedChannel.value = channel
  if (copyResetTimer !== undefined) window.clearTimeout(copyResetTimer)
  copyResetTimer = window.setTimeout(() => {
    if (copiedChannel.value === channel) copiedChannel.value = ''
  }, 900)
}

onBeforeUnmount(() => {
  if (copyResetTimer !== undefined) window.clearTimeout(copyResetTimer)
})

const CopyButton = defineComponent({
  name: 'CopyButton',
  props: {
    label: { type: String, required: true },
    copied: { type: Boolean, default: false },
    compact: { type: Boolean, default: false },
  },
  emits: ['copy'],
  setup(props, { emit }) {
    return () => h('button', {
      type: 'button',
      class: ['contact-copy-button', props.compact && 'contact-copy-button--compact', props.copied && 'contact-copy-button--copied'],
      title: props.label,
      'aria-label': props.label,
      'aria-pressed': props.copied,
      onClick: () => emit('copy'),
    }, [
      h(Icon, { name: props.copied ? 'check' : 'copy', size: props.compact ? 'xs' : 'sm', 'aria-hidden': true }),
      h('span', { class: 'contact-copy-tooltip', 'aria-hidden': true }, '已复制'),
      h('span', { class: 'sr-only', 'aria-live': 'polite' }, props.copied ? '已复制' : ''),
    ])
  },
})
</script>

<style scoped>
.contact-screen {
  --contact-canvas: #f8fafc;
  --contact-card: #fff;
  --contact-border: #e5e7eb;
  --contact-text: #111827;
  --contact-text-secondary: #6b7280;
  --contact-text-muted: #9ca3af;
  --contact-blue: #4f7fe6;
  --contact-blue-hover: #3f6fd8;
  --contact-blue-soft: #eff4ff;
  min-height: 100dvh;
  display: flex;
  flex-direction: column;
  color: var(--contact-text);
  background: var(--contact-canvas);
  font-family: Inter, "PingFang SC", "Microsoft YaHei", sans-serif;
}
.contact-content { width: min(100% - 48px, 920px); margin: 0 auto; }
.contact-header { height: 52px; flex: 0 0 52px; border-bottom: 1px solid var(--contact-border); background: var(--contact-card); }
.contact-header__inner { height: 100%; display: flex; align-items: center; justify-content: space-between; }
.contact-brand, .contact-brand__text, .contact-back-link, .contact-card__heading, .contact-value-row, .contact-wechat-value-row { display: flex; align-items: center; }
.contact-brand { gap: 10px; }
.contact-brand__mark { width: 28px; height: 28px; display: inline-flex; align-items: center; justify-content: center; color: #fff; border-radius: 6px; background: var(--contact-blue); }
.contact-brand__mark svg { width: 18px; height: 18px; }
.contact-brand__text { gap: 8px; }
.contact-brand__name { font-size: 16px; font-weight: 700; letter-spacing: -0.01em; }
.contact-brand__badge { padding: 2px 6px; border-radius: 4px; color: var(--contact-text-muted); background: var(--contact-canvas); font-size: 10px; font-weight: 500; }
.contact-back-link { gap: 6px; color: var(--contact-text-secondary); font-size: 13px; font-weight: 500; text-decoration: none; transition: color 160ms ease; }
.contact-back-link:hover { color: var(--contact-blue); }
.contact-main { flex: 1; display: flex; flex-direction: column; justify-content: center; padding: 24px 0; }
.contact-title-row { margin-bottom: 24px; }
.contact-title-row h1 { margin: 0; color: var(--contact-text); font-size: 32px; font-weight: 700; letter-spacing: -0.03em; line-height: 1.2; }
.contact-title-row p { margin: 6px 0 0; color: var(--contact-text-secondary); font-size: 15px; line-height: 1.5; }
.contact-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 20px; }
.contact-card { height: 256px; display: flex; flex-direction: column; justify-content: space-between; padding: 24px; border: 1px solid var(--contact-border); border-radius: 14px; background: var(--contact-card); box-shadow: 0 1px 2px rgb(0 0 0 / 4%); }
.contact-card__heading { gap: 12px; }
.contact-card__icon { width: 40px; height: 40px; display: inline-flex; align-items: center; justify-content: center; flex: 0 0 auto; border-radius: 12px; color: #2f6bff; background: var(--contact-blue-soft); }
.contact-card__heading h2 { margin: 0; font-size: 16px; font-weight: 600; }
.contact-value-row { gap: 8px; }
.contact-value { color: var(--contact-text-secondary); font-size: 16px; font-weight: 500; }
.contact-copy-button { position: relative; width: 32px; height: 32px; display: inline-flex; align-items: center; justify-content: center; flex: 0 0 auto; padding: 0; border: 1px solid var(--contact-border); border-radius: 8px; color: var(--contact-text-secondary); background: var(--contact-card); cursor: pointer; transition: color 160ms ease, border-color 160ms ease, background-color 160ms ease, transform 160ms ease; }
.contact-copy-button--compact { width: 28px; height: 28px; border-radius: 7px; }
.contact-copy-button:hover, .contact-copy-button--copied { color: #2f6bff; border-color: #2f6bff; background: var(--contact-blue-soft); }
.contact-copy-button:active { transform: scale(0.92); }
.contact-copy-button:focus-visible, .contact-back-link:focus-visible, .contact-telegram-button:focus-visible { outline: 2px solid #2f6bff; outline-offset: 2px; }
.contact-copy-tooltip { position: absolute; top: -32px; left: 50%; z-index: 2; padding: 4px 8px; border: 1px solid #d6e3ff; border-radius: 6px; color: #2f6bff; background: var(--contact-blue-soft); box-shadow: 0 2px 4px rgb(0 0 0 / 5%); font-size: 11px; line-height: 1.2; opacity: 0; pointer-events: none; transform: translate(-50%, 4px); transition: opacity 200ms ease, transform 200ms ease; white-space: nowrap; }
.contact-copy-button--copied .contact-copy-tooltip { opacity: 1; transform: translate(-50%, 0); }
.contact-wechat-body { display: flex; align-items: center; gap: 28px; flex: 1; }
.contact-qr-frame { width: 176px; height: 176px; display: flex; align-items: center; justify-content: center; flex: 0 0 auto; overflow: hidden; border: 1px solid var(--contact-border); border-radius: 12px; background: var(--contact-canvas); }
.contact-qr-frame img { width: 100%; height: 100%; object-fit: contain; }
.contact-wechat-meta { display: flex; flex-direction: column; gap: 3px; min-width: 0; }
.contact-label { color: var(--contact-text-muted); font-size: 10px; font-weight: 700; letter-spacing: 0.02em; }
.contact-wechat-value-row { gap: 8px; }
.contact-value--wechat { color: var(--contact-text); font-weight: 700; white-space: nowrap; }
.contact-telegram-value { flex: 1; display: flex; align-items: center; color: var(--contact-text-secondary); font-size: 13px; font-weight: 500; overflow-wrap: anywhere; }
.contact-telegram-value span { overflow-wrap: anywhere; }
.contact-telegram-button { width: 100%; height: 40px; display: inline-flex; align-items: center; justify-content: center; gap: 8px; border-radius: 10px; color: #fff; background: var(--contact-blue); box-shadow: 0 1px 2px rgb(63 111 216 / 18%); font-size: 13px; font-weight: 600; text-decoration: none; transition: background-color 160ms ease, transform 160ms ease; }
.contact-telegram-button:hover { background: var(--contact-blue-hover); }
.contact-telegram-button:active { transform: scale(0.98); }
.contact-footer { flex: 0 0 auto; padding: 16px 0; border-top: 1px solid var(--contact-border); color: var(--contact-text-muted); background: var(--contact-card); font-size: 11px; }
@media (max-height: 800px) { .contact-main { padding-top: 24px; padding-bottom: 24px; } .contact-card { height: 220px; padding: 20px; } .contact-qr-frame { width: 148px; height: 148px; } }
@media (max-width: 767px) { .contact-content { width: min(100% - 32px, 920px); } .contact-main { justify-content: flex-start; padding: 32px 0 24px; } .contact-grid { grid-template-columns: 1fr; gap: 16px; } .contact-card { height: 220px; } .contact-card--wechat { height: 248px; } .contact-qr-frame { width: 156px; height: 156px; } .contact-wechat-body { gap: 20px; } }
@media (prefers-reduced-motion: reduce) { .contact-copy-button, .contact-copy-tooltip, .contact-back-link, .contact-telegram-button { transition-duration: 0.01ms; } }
</style>

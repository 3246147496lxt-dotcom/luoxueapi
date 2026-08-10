import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

import ChatComposer from '../ChatComposer.vue'
import enChat from '@/i18n/locales/en/chat'
import zhChat from '@/i18n/locales/zh/chat'

const COMPONENT_SOURCE = readFileSync(
  resolve(process.cwd(), 'src/components/chat/ChatComposer.vue'),
  'utf8',
)
const SCOPED_STYLE = COMPONENT_SOURCE.match(/<style scoped>([\s\S]*?)<\/style>/)?.[1] ?? ''

const IconStub = {
  props: ['name'],
  template: '<span :data-icon="name" />',
}

const mountedWrappers: VueWrapper[] = []

function mountComposer(
  props: Record<string, unknown> = {},
  slots: Record<string, string> = {},
) {
  const wrapper = mount(ChatComposer, {
    attachTo: document.body,
    props: {
      modelValue: '已有文字',
      onSend: (_value: string, acknowledge: (accepted: boolean) => void) => acknowledge(true),
      ...props,
    },
    slots,
    global: {
      stubs: {
        Icon: IconStub,
        RouterLink: { template: '<a><slot /></a>' },
      },
    },
  })
  mountedWrappers.push(wrapper)
  return wrapper
}

afterEach(() => {
  mountedWrappers.splice(0).forEach((wrapper) => wrapper.unmount())
  document.body.innerHTML = ''
})

describe('ChatComposer voice transcription integration', () => {
  it('keeps attachment previews inside the unified composer and expands for rich drafts', async () => {
    const wrapper = mountComposer({
      modelValue: '第一行\n第二行',
      hasAttachments: true,
    }, {
      attachments: '<div data-test="attachment-preview">附件</div>',
      leading: '<button type="button" data-test="composer-leading">附件</button>',
      trailing: '<button type="button" data-test="composer-trailing">模型</button>',
    })
    const form = wrapper.get('form')
    const attachment = form.get('[data-test="attachment-preview"]')
    const leading = form.get('.chat-composer__leading')
    const textarea = form.get('textarea')
    const trailing = form.get('.chat-composer__trailing')
    const action = form.get('.chat-composer__action')
    const children = Array.from(form.element.children)

    expect(form.classes()).toContain('chat-composer--expanded')
    expect(form.classes()).toContain('chat-composer--has-attachments')
    expect(children.indexOf(attachment.element.closest('.chat-composer__attachments')!))
      .toBeLessThan(children.indexOf(leading.element))
    expect(children.indexOf(leading.element)).toBeLessThan(children.indexOf(textarea.element))
    expect(children.indexOf(textarea.element)).toBeLessThan(children.indexOf(trailing.element))
    expect(children.indexOf(trailing.element)).toBeLessThan(children.indexOf(action.element))

    await wrapper.setProps({ modelValue: '单行', hasAttachments: false })
    await nextTick()
    expect(form.classes()).not.toContain('chat-composer--expanded')
    expect(form.find('.chat-composer__attachments').exists()).toBe(false)
  })

  it('submits an attachment-only message and blocks invalid attachment drafts', async () => {
    const wrapper = mountComposer({
      modelValue: '',
      hasAttachments: true,
      attachmentsValid: true,
    })

    await wrapper.get('form').trigger('submit')
    expect(wrapper.emitted('send')?.map(([value]) => value)).toEqual([''])

    await wrapper.setProps({ attachmentsValid: false })
    await wrapper.get('form').trigger('submit')
    expect(wrapper.emitted('send')?.map(([value]) => value)).toEqual([''])
  })

  it('shows the voice action only for an empty unsendable draft', async () => {
    const wrapper = mountComposer({ modelValue: '' }, {
      'empty-action': '<button type="button" data-test="voice-action">语音</button>',
    })

    expect(wrapper.find('[data-test="voice-action"]').exists()).toBe(true)
    expect(wrapper.find('.chat-composer__action--send').exists()).toBe(false)

    await wrapper.setProps({ modelValue: '可以发送' })
    await nextTick()

    expect(wrapper.find('[data-test="voice-action"]').exists()).toBe(false)
    const sendButton = wrapper.get('.chat-composer__action--send')
    expect(sendButton.get('[data-icon="chatSend"]').attributes('data-icon')).toBe('chatSend')
  })

  it('matches the target single-line composer geometry', () => {
    expect(SCOPED_STYLE).toContain('width: min(100%, 768px);')
    expect(SCOPED_STYLE).toContain('min-height: 52px;')
    expect(SCOPED_STYLE.match(/border-radius: 28px;/g)).toHaveLength(3)
    expect(SCOPED_STYLE.match(/padding: 8px;/g)).toHaveLength(2)
    expect(SCOPED_STYLE.match(/padding: 6px 8px;/g)).toHaveLength(2)
    expect(SCOPED_STYLE).toContain('column-gap: 0;')
    expect(SCOPED_STYLE).toContain('grid-template-rows: minmax(36px, auto) 36px;')
    expect(SCOPED_STYLE).toContain('padding-block: 5px;')
    expect(SCOPED_STYLE).toContain('padding-inline: 7px 6px;')
    expect(SCOPED_STYLE.match(/padding: 5px 10px;/g)).toHaveLength(2)
    expect(SCOPED_STYLE).toContain('gap: 4px;')
    expect(SCOPED_STYLE.match(/margin-inline-start: 8px;/g)).toHaveLength(2)
    expect(SCOPED_STYLE).toContain('min-height: 84px;')
    expect(SCOPED_STYLE).toContain('width: 36px;')
    expect(SCOPED_STYLE).toContain('height: 36px;')
  })

  it('uses the target copy and typography without conflating placeholder and label', async () => {
    const wrapper = mountComposer({ modelValue: '' })
    const textarea = wrapper.get('textarea')

    expect(textarea.attributes('placeholder')).toBe('chat.composer.placeholder')
    expect(textarea.attributes('aria-label')).toBe('chat.composer.label')
    expect(zhChat.chat.composer).toMatchObject({
      label: '与 ChatGPT 聊天',
      placeholder: '问问 ChatGPT',
    })
    expect(enChat.chat.composer).toMatchObject({
      label: 'Chat with ChatGPT',
      placeholder: 'Ask ChatGPT',
    })

    await wrapper.setProps({ insufficientBalance: true })
    expect(textarea.attributes('placeholder')).toBe('chat.composer.rechargePlaceholder')
    expect(textarea.attributes('aria-label')).toBe('chat.composer.label')

    expect(SCOPED_STYLE).not.toContain('--chat-composer-font')
    expect(SCOPED_STYLE).not.toContain('font-family:')
    expect(SCOPED_STYLE).toMatch(
      /\.chat-composer textarea\s*\{[^}]*font-size: 16px;[^}]*font-weight: 400;/,
    )
    expect(SCOPED_STYLE).toMatch(
      /@media \(max-width: 720px\)[\s\S]*?\.chat-composer textarea\s*\{[^}]*font-size: 16px;/,
    )
    expect(SCOPED_STYLE).toContain('line-height: 26px;')
    expect(SCOPED_STYLE).toContain('--chat-composer-placeholder-fg: #8f8f8f;')
    expect(SCOPED_STYLE).toContain('color: var(--chat-composer-placeholder-fg);')
    expect(SCOPED_STYLE).toMatch(
      /textarea::placeholder\s*\{[^}]*font-size: inherit;[^}]*font-weight: inherit;[^}]*line-height: inherit;[^}]*letter-spacing: inherit;/,
    )
  })

  it('uses a distinct ChatGPT-style composer surface in dark mode', () => {
    expect(SCOPED_STYLE).toContain('--chat-composer-surface: #fff;')
    expect(SCOPED_STYLE).toContain('background: var(--chat-composer-surface);')
    expect(SCOPED_STYLE).toContain('box-shadow: var(--chat-composer-shadow);')
    expect(SCOPED_STYLE).toMatch(
      /:global\(html\.dark \.chat-composer\) \{[\s\S]*?--chat-composer-surface: rgb\(33 33 33 \/ 90%\);/,
    )
    expect(SCOPED_STYLE).toMatch(
      /:global\(html\.dark \.chat-composer\) \{[\s\S]*?--chat-composer-placeholder-fg: #afafaf;/,
    )
    expect(SCOPED_STYLE).toMatch(
      /:global\(html\.dark \.chat-composer\) \{[\s\S]*?--chat-composer-tertiary-fg: #afafaf;/,
    )
    expect(SCOPED_STYLE).toContain(
      '--chat-composer-shadow: inset 0 0 1px rgb(255 255 255 / 20%);',
    )
  })

  it('animates textarea growth without bypassing reduced-motion preferences', () => {
    expect(COMPONENT_SOURCE).toContain("window.matchMedia?.('(prefers-reduced-motion: reduce)')")
    expect(COMPONENT_SOURCE).toContain('requestAnimationFrame(() => {')
    expect(SCOPED_STYLE).toContain(
      'transition: height 180ms cubic-bezier(0.22, 1, 0.36, 1);',
    )
    expect(SCOPED_STYLE).toMatch(
      /@media \(prefers-reduced-motion: reduce\) \{[\s\S]*?\.chat-composer textarea,[\s\S]*?transition: none;/,
    )
  })

  it('blocks submission while transcription is busy without disabling text editing', async () => {
    const wrapper = mountComposer({ submissionBusy: true })
    const textarea = wrapper.get('textarea')

    expect(textarea.attributes('disabled')).toBeUndefined()
    expect(wrapper.get('form').attributes('aria-busy')).toBe('true')
    await textarea.trigger('keydown', { key: 'Enter' })
    await wrapper.get('form').trigger('submit')
    expect(wrapper.emitted('send')).toBeUndefined()

    await wrapper.setProps({ submissionBusy: false })
    await textarea.trigger('keydown', { key: 'Enter' })
    expect(wrapper.emitted('send')?.map(([value]) => value)).toEqual(['已有文字'])
  })

  it('inserts transcription at the current selection and restores the caret', async () => {
    const wrapper = mountComposer()
    const textarea = wrapper.get('textarea').element as HTMLTextAreaElement
    textarea.focus()
    textarea.setSelectionRange(textarea.value.length, textarea.value.length)

    const inserted = (wrapper.vm as unknown as {
      insertText: (value: string) => boolean
    }).insertText('语音结果')
    await nextTick()

    expect(inserted).toBe(true)
    expect(textarea.value).toBe('已有文字\n语音结果')
    expect(textarea.selectionStart).toBe(textarea.value.length)
    expect(document.activeElement).toBe(textarea)
    expect(wrapper.emitted('send')).toBeUndefined()
  })

  it('replaces a selection and enforces the existing maximum draft length', async () => {
    const wrapper = mountComposer({ modelValue: 'abcd', maxLength: 5 })
    const textarea = wrapper.get('textarea').element as HTMLTextAreaElement
    textarea.setSelectionRange(1, 3)
    const api = wrapper.vm as unknown as { insertText: (value: string) => boolean }

    expect(api.insertText('XYZ')).toBe(true)
    await nextTick()
    expect(textarea.value).toBe('aXYZd')

    textarea.setSelectionRange(textarea.value.length, textarea.value.length)
    expect(api.insertText('more')).toBe(false)
    expect(textarea.value).toBe('aXYZd')
  })

  it('keeps the existing draft unchanged when the full transcription does not fit', () => {
    const wrapper = mountComposer({ modelValue: 'abcd', maxLength: 6 })
    const textarea = wrapper.get('textarea').element as HTMLTextAreaElement
    textarea.setSelectionRange(textarea.value.length, textarea.value.length)
    const api = wrapper.vm as unknown as { insertText: (value: string) => boolean }

    expect(api.insertText('XYZ')).toBe(false)
    expect(textarea.value).toBe('abcd')
  })

  it('keeps the draft when the parent atomically rejects submission', async () => {
    const wrapper = mountComposer({
      modelValue: '附件过期也别清空',
      onSend: (_value: string, acknowledge: (accepted: boolean) => void) => acknowledge(false),
    })

    await wrapper.get('form').trigger('submit')

    expect((wrapper.get('textarea').element as HTMLTextAreaElement).value)
      .toBe('附件过期也别清空')
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  })
})

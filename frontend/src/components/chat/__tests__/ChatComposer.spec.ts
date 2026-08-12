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

function installDesktopComposerGeometry(
  wrapper: VueWrapper,
  initial: { compactHeight: number, expandedHeight: number, viewportHeight?: number },
) {
  const form = wrapper.get('form').element as HTMLFormElement
  const input = wrapper.get('.chat-composer__input').element as HTMLTextAreaElement
  const measure = wrapper.get('.chat-composer__measure').element as HTMLTextAreaElement
  const geometry = {
    compactHeight: initial.compactHeight,
    expandedHeight: initial.expandedHeight,
  }

  vi.spyOn(window, 'matchMedia').mockImplementation((query: string) => ({
    matches: query === '(prefers-reduced-motion: reduce)',
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
  vi.spyOn(window, 'innerHeight', 'get').mockReturnValue(initial.viewportHeight ?? 900)
  vi.spyOn(form, 'getBoundingClientRect').mockReturnValue({
    x: 0,
    y: 0,
    top: 0,
    right: 768,
    bottom: 52,
    left: 0,
    width: 768,
    height: 52,
    toJSON: () => ({}),
  })

  Object.defineProperty(input, 'scrollHeight', {
    configurable: true,
    get: () => form.classList.contains('chat-composer--expanded')
      ? geometry.expandedHeight
      : geometry.compactHeight,
  })
  Object.defineProperty(input, 'clientHeight', {
    configurable: true,
    get: () => Number.parseFloat(input.style.height) || 36,
  })
  Object.defineProperty(measure, 'scrollHeight', {
    configurable: true,
    get: () => geometry.compactHeight,
  })

  return { form, input, geometry }
}

afterEach(() => {
  mountedWrappers.splice(0).forEach((wrapper) => wrapper.unmount())
  document.body.innerHTML = ''
  vi.restoreAllMocks()
})

describe('ChatComposer voice transcription integration', () => {
  it('keeps an attachment-only draft in the compact two-row composer', async () => {
    const wrapper = mountComposer({
      modelValue: '',
      hasAttachments: true,
    }, {
      attachments: '<div data-test="attachment-preview">附件</div>',
      leading: '<button type="button" data-test="composer-leading">附件</button>',
      trailing: '<button type="button" data-test="composer-trailing">模型</button>',
    })
    const form = wrapper.get('form')
    const attachment = form.get('[data-test="attachment-preview"]')
    const leading = form.get('.chat-composer__leading')
    const inputShell = form.get('.chat-composer__input-shell')
    const trailing = form.get('.chat-composer__trailing')
    const action = form.get('.chat-composer__action')
    const children = Array.from(form.element.children)

    expect(form.classes()).not.toContain('chat-composer--expanded')
    expect(form.classes()).toContain('chat-composer--has-attachments')
    expect(children.indexOf(attachment.element.closest('.chat-composer__attachments')!))
      .toBeLessThan(children.indexOf(leading.element))
    expect(children.indexOf(leading.element)).toBeLessThan(children.indexOf(inputShell.element))
    expect(children.indexOf(inputShell.element)).toBeLessThan(children.indexOf(trailing.element))
    expect(children.indexOf(trailing.element)).toBeLessThan(children.indexOf(action.element))

    const attachmentOnlyStyle = SCOPED_STYLE.match(
      /\.chat-composer--has-attachments\s*\{([\s\S]*?)\}/,
    )?.[1] ?? ''
    expect(attachmentOnlyStyle).toContain(
      '"composer-attachments composer-attachments composer-attachments composer-attachments"',
    )
    expect(attachmentOnlyStyle).toContain(
      '"composer-leading composer-input composer-trailing composer-action"',
    )
    expect(attachmentOnlyStyle).toContain('grid-template-rows: auto 36px;')
    expect(attachmentOnlyStyle).toContain('row-gap: 18px;')

    await wrapper.setProps({ hasAttachments: false })
    await nextTick()
    await nextTick()
    expect(form.classes()).not.toContain('chat-composer--expanded')
    expect(form.find('.chat-composer__attachments').exists()).toBe(false)
  })

  it('still expands when an attachment draft contains genuinely multiline text', async () => {
    const wrapper = mountComposer({
      modelValue: '第一行\n第二行',
      hasAttachments: true,
    }, {
      attachments: '<div data-test="attachment-preview">附件</div>',
    })
    const form = wrapper.get('form')

    expect(form.classes()).toEqual(expect.arrayContaining([
      'chat-composer--expanded',
      'chat-composer--has-attachments',
    ]))
    expect(wrapper.get('.chat-composer__input').element)
      .toHaveProperty('value', '第一行\n第二行')

    await wrapper.setProps({ modelValue: '单行' })
    await nextTick()
    await nextTick()

    expect(form.classes()).not.toContain('chat-composer--expanded')
    expect(form.classes()).toContain('chat-composer--has-attachments')
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

  it('keeps the reference Chinese sentence fully visible above a stable bottom toolbar', async () => {
    const wrapper = mountComposer({ modelValue: '' }, {
      leading: '<button type="button">附件</button>',
      trailing: '<button type="button">Pro</button><button type="button">语音</button>',
    })
    const { form, input } = installDesktopComposerGeometry(wrapper, {
      compactHeight: 62,
      expandedHeight: 36,
    })
    const referenceSentence = '你觉得充值、订阅套餐、兑换码、我的订阅这些东西应该放在一个功能板块里进行展示吗？站在用户'

    await wrapper.get('.chat-composer__input').setValue(referenceSentence)
    await vi.waitFor(() => {
      expect(form.classList.contains('chat-composer--expanded')).toBe(true)
      expect(input.style.height).toBe('48px')
    })

    expect(input.value).toBe(referenceSentence)
    expect(input.style.overflowY).toBe('hidden')
    expect(form.getAttribute('data-expanded')).toBe('')
    expect(wrapper.find('[data-test="chat-composer-expand"]').exists()).toBe(false)
  })

  it('matches the viewport-based scroll cap and preserves editing state when expanded', async () => {
    const wrapper = mountComposer({ modelValue: '' })
    const { form, input, geometry } = installDesktopComposerGeometry(wrapper, {
      compactHeight: 166,
      expandedHeight: 140,
      viewportHeight: 900,
    })
    const inputWrapper = wrapper.get('.chat-composer__input')
    const longDraft = '测'.repeat(155)

    await inputWrapper.setValue(longDraft)
    await vi.waitFor(() => {
      expect(wrapper.get('[data-test="chat-composer-expand"]').attributes('aria-label'))
        .toBe('chat.composer.expand')
      expect(input.style.height).toBe('140px')
    })

    input.focus()
    input.setSelectionRange(12, 37, 'backward')
    await wrapper.get('[data-test="chat-composer-expand"]').trigger('click')
    await vi.waitFor(() => {
      expect(form.classList.contains('chat-composer--maximized')).toBe(true)
      expect(input.style.height).toBe('621px')
    })

    const collapseButton = wrapper.get('[data-test="chat-composer-expand"]')
    expect(collapseButton.attributes('aria-label')).toBe('chat.composer.collapse')
    expect(collapseButton.attributes('aria-controls')).toBe(input.id)
    expect(collapseButton.attributes('aria-expanded')).toBe('true')
    expect(collapseButton.get('[data-icon="chatComposerCollapse"]').attributes('data-icon'))
      .toBe('chatComposerCollapse')
    expect(document.activeElement).toBe(input)
    expect([input.selectionStart, input.selectionEnd]).toEqual([12, 37])
    expect(input.selectionDirection).toBe('backward')

    await collapseButton.trigger('click')
    await vi.waitFor(() => {
      expect(form.classList.contains('chat-composer--maximized')).toBe(false)
      expect(input.style.height).toBe('140px')
    })

    geometry.compactHeight = 322
    geometry.expandedHeight = 296
    input.setSelectionRange(input.value.length, input.value.length)
    await inputWrapper.trigger('input')
    await vi.waitFor(() => expect(input.style.height).toBe('270px'))
    expect(input.style.overflowY).toBe('auto')
    expect(form.classList.contains('chat-composer--overflowing')).toBe(true)

    geometry.compactHeight = 36
    geometry.expandedHeight = 36
    await inputWrapper.setValue('短文本')
    await vi.waitFor(() => {
      expect(form.classList.contains('chat-composer--expanded')).toBe(false)
      expect(input.style.height).toBe('36px')
      expect(wrapper.find('[data-test="chat-composer-expand"]').exists()).toBe(false)
    })
    expect(input.style.overflowY).toBe('hidden')
  })

  it('offers expansion as soon as a keyboard-shortened viewport reaches its scroll cap', async () => {
    const wrapper = mountComposer({ modelValue: '' })
    const { form, input } = installDesktopComposerGeometry(wrapper, {
      compactHeight: 110,
      expandedHeight: 100,
      viewportHeight: 300,
    })

    await wrapper.get('.chat-composer__input').setValue('测'.repeat(90))
    await vi.waitFor(() => {
      expect(form.classList.contains('chat-composer--expanded')).toBe(true)
      expect(input.style.height).toBe('90px')
      expect(input.style.overflowY).toBe('auto')
      expect(wrapper.get('[data-test="chat-composer-expand"]').attributes('aria-expanded'))
        .toBe('false')
    })

    expect(wrapper.get('[data-test="chat-composer-expand"]').element.parentElement)
      .toBe(wrapper.get('.chat-composer__input-shell').element)
  })

  it('uses the target copy and typography without conflating placeholder and label', async () => {
    const wrapper = mountComposer({ modelValue: '' })
    const textarea = wrapper.get('textarea')

    expect(textarea.attributes('placeholder')).toBe('chat.composer.placeholder')
    expect(textarea.attributes('aria-label')).toBe('chat.composer.label')
    expect(zhChat.chat.composer).toMatchObject({
      label: '与 ChatGPT 聊天',
      placeholder: '问问 ChatGPT',
      expand: '展开输入框',
      collapse: '收起输入框',
    })
    expect(enChat.chat.composer).toMatchObject({
      label: 'Chat with ChatGPT',
      placeholder: 'Ask ChatGPT',
      expand: 'Expand composer',
      collapse: 'Collapse composer',
    })

    await wrapper.setProps({ insufficientBalance: true })
    expect(textarea.attributes('placeholder')).toBe('chat.composer.rechargePlaceholder')
    expect(textarea.attributes('aria-label')).toBe('chat.composer.label')

    expect(SCOPED_STYLE).not.toContain('--chat-composer-font')
    expect(SCOPED_STYLE).not.toContain('font-family:')
    expect(SCOPED_STYLE).toMatch(
      /\.chat-composer__input\s*\{[^}]*font-size: 16px;[^}]*font-weight: 400;/,
    )
    expect(SCOPED_STYLE).toMatch(
      /@media \(max-width: 720px\)[\s\S]*?\.chat-composer__input\s*\{[^}]*font-size: 16px;/,
    )
    expect(SCOPED_STYLE).toContain('line-height: 26px;')
    expect(SCOPED_STYLE).toContain('--chat-composer-placeholder-fg: #8f8f8f;')
    expect(SCOPED_STYLE).toContain('color: var(--chat-composer-placeholder-fg);')
    expect(SCOPED_STYLE).toMatch(
      /\.chat-composer__input::placeholder\s*\{[^}]*font-size: inherit;[^}]*font-weight: inherit;[^}]*line-height: inherit;[^}]*letter-spacing: inherit;/,
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

  it('avoids layout-height transitions while keeping resize work frame-batched', () => {
    expect(COMPONENT_SOURCE).toContain('queuedResizeFrame = requestAnimationFrame(() => {')
    expect(SCOPED_STYLE).not.toContain('transition: height')
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

import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { UserAnnouncement } from '@/types'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')

  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

import AnnouncementBell from '../AnnouncementBell.vue'
import { useAnnouncementStore } from '@/stores/announcements'

function mountAnnouncementBell(compact = false) {
  const pinia = createPinia()
  setActivePinia(pinia)

  return {
    store: useAnnouncementStore(),
    wrapper: mount(AnnouncementBell, {
      props: { compact },
      attachTo: document.body,
      global: { plugins: [pinia] },
    }),
  }
}

describe('AnnouncementBell notification icon', () => {
  afterEach(() => {
    document.body.innerHTML = ''
    document.body.style.overflow = ''
  })

  it('uses the sanitized supplied SVG while preserving the accessible trigger', () => {
    const { wrapper } = mountAnnouncementBell()
    const trigger = wrapper.get('button')
    const icon = trigger.get('svg.announcement-bell-icon')

    expect(trigger.attributes('aria-label')).toBe('announcements.title')
    expect(trigger.classes()).toEqual(expect.arrayContaining(['min-h-11', 'min-w-11']))
    expect(icon.attributes('viewBox')).toBe('0 0 1024 1024')
    expect(icon.attributes('aria-hidden')).toBe('true')
    expect(icon.findAll('path')).toHaveLength(1)
    expect(icon.get('path').attributes('fill')).toBe('currentColor')
    expect(icon.find('script, foreignObject, [href], [xlink\\:href]').exists()).toBe(false)

    wrapper.unmount()
  })

  it('keeps the unread color and indicator behavior', async () => {
    const { store, wrapper } = mountAnnouncementBell()
    const announcement: UserAnnouncement = {
      id: 1,
      title: 'Unread announcement',
      content: 'Content',
      notify_mode: 'silent',
      created_at: '2026-07-18T00:00:00Z',
      updated_at: '2026-07-18T00:00:00Z',
    }

    store.announcements.push(announcement)
    await nextTick()

    const trigger = wrapper.get('button')
    expect(trigger.classes()).toEqual(expect.arrayContaining(['text-blue-600', 'dark:text-blue-400']))
    expect(trigger.find('.bg-red-500').exists()).toBe(true)
    expect(trigger.get('svg.announcement-bell-icon').get('path').attributes('fill')).toBe('currentColor')

    wrapper.unmount()
  })

  it('matches the compact header icon proportions and hover treatment', () => {
    const { wrapper } = mountAnnouncementBell(true)
    const trigger = wrapper.get('button')
    const icon = trigger.get('svg.announcement-bell-icon')

    expect(trigger.classes()).toEqual(expect.arrayContaining([
      'h-8',
      'w-8',
      'text-[#007bff]',
      'hover:bg-[rgba(46,50,56,0.05)]',
    ]))
    expect(trigger.classes()).not.toContain('min-h-11')
    expect(icon.classes()).toEqual(expect.arrayContaining(['h-[18px]', 'w-[18px]']))

    wrapper.unmount()
  })
})

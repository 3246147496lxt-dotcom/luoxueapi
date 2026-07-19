import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { User } from '@/types'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) =>
        key === 'version.updateAvailable' ? '发现新版本' : '当前已是最新版本',
    }),
  }
})

import VersionBadge from '../VersionBadge.vue'
import { useAppStore, useAuthStore } from '@/stores'

const longVersion = 'deploy-20260718-153216-5b5dd209-clean1'

describe('VersionBadge', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('constrains and truncates a long admin version while preserving its accessible label', () => {
    const authStore = useAuthStore()
    const appStore = useAppStore()

    authStore.user = { role: 'admin' } as User
    appStore.currentVersion = longVersion
    appStore.hasUpdate = true
    vi.spyOn(appStore, 'fetchVersion').mockResolvedValue(null)

    const wrapper = mount(VersionBadge, {
      global: {
        stubs: { Icon: true },
      },
    })
    const trigger = wrapper.get('button')
    const versionText = trigger.findAll('span')[0]

    expect(trigger.classes()).toEqual(
      expect.arrayContaining(['min-w-0', 'max-w-full', 'overflow-hidden']),
    )
    expect(versionText.classes()).toEqual(expect.arrayContaining(['min-w-0', 'truncate']))
    expect(trigger.attributes('title')).toContain(`v${longVersion}`)
    expect(trigger.attributes('aria-label')).toContain(`v${longVersion}`)

    wrapper.unmount()
  })
})

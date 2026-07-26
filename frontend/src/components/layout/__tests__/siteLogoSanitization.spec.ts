import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const dir = dirname(fileURLToPath(import.meta.url))
const brandSource = readFileSync(resolve(dir, '../AppBrand.vue'), 'utf8')
const publicSiteLayoutSource = readFileSync(resolve(dir, '../../public/PublicSiteLayout.vue'), 'utf8')
const keyUsageViewSource = readFileSync(resolve(dir, '../../../views/KeyUsageView.vue'), 'utf8')

describe('site_logo sanitization', () => {
  it('AppBrand imports sanitizeUrl and applies it to siteLogo', () => {
    expect(brandSource).toContain("import { sanitizeUrl } from '@/utils/url'")
    expect(brandSource).toContain("sanitizeUrl(appStore.siteLogo || ''")
  })

  it('the shared public site layout applies sanitizeUrl to siteLogo', () => {
    expect(publicSiteLayoutSource).toContain('sanitizeUrl(')
    expect(publicSiteLayoutSource).toContain('appStore.cachedPublicSettings?.site_logo || appStore.siteLogo')
  })

  it('KeyUsageView applies sanitizeUrl to siteLogo', () => {
    expect(keyUsageViewSource).toContain('sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo')
  })

  it('all three pass allowRelative and allowDataUrl options', () => {
    for (const src of [brandSource, publicSiteLayoutSource, keyUsageViewSource]) {
      expect(src).toContain('allowRelative: true')
      expect(src).toContain('allowDataUrl: true')
    }
  })
})

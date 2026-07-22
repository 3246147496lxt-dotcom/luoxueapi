import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const dir = dirname(fileURLToPath(import.meta.url))
const headerSource = readFileSync(resolve(dir, '../AppHeader.vue'), 'utf8')
const sidebarSource = readFileSync(resolve(dir, '../AppSidebar.vue'), 'utf8')
const publicSiteLayoutSource = readFileSync(resolve(dir, '../../public/PublicSiteLayout.vue'), 'utf8')
const homeViewSource = readFileSync(resolve(dir, '../../../views/HomeView.vue'), 'utf8')
const keyUsageViewSource = readFileSync(resolve(dir, '../../../views/KeyUsageView.vue'), 'utf8')

describe('doc_url sanitization', () => {
  it('AppHeader uses the shared environment-aware documentation URL resolver', () => {
    expect(headerSource).toContain("import { resolveDocumentationUrl } from '@/utils/documentationUrl'")
    expect(headerSource).toContain('appStore.cachedPublicSettings?.doc_url || appStore.docUrl')
    expect(headerSource).toContain('resolveDocumentationUrl(configuredDocUrl.value)')
  })

  it('AppSidebar uses the shared environment-aware documentation URL resolver', () => {
    expect(sidebarSource).toContain("import { resolveDocumentationUrl } from '@/utils/documentationUrl'")
    expect(sidebarSource).toContain('appStore.cachedPublicSettings?.doc_url || appStore.docUrl')
    expect(sidebarSource).not.toContain('https://luoxueapi.cc/tutorial-docs/')
  })

  it('HomeView imports sanitizeUrl', () => {
    expect(homeViewSource).toContain("import { sanitizeUrl } from '@/utils/url'")
  })

  it('HomeView applies sanitizeUrl to docUrl', () => {
    expect(homeViewSource).toContain('sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl')
  })

  it('PublicSiteLayout resolves the optional API docs entry and tutorial fallback separately', () => {
    expect(publicSiteLayoutSource).toContain('resolveDocumentationUrl, resolveTutorialUrl')
    expect(publicSiteLayoutSource).toContain('resolveDocumentationUrl(configuredDocUrl.value)')
    expect(publicSiteLayoutSource).toContain('resolveTutorialUrl(configuredDocUrl.value)')
  })

  it('KeyUsageView uses the shared environment-aware documentation URL resolver', () => {
    expect(keyUsageViewSource).toContain("import { resolveDocumentationUrl } from '@/utils/documentationUrl'")
    expect(keyUsageViewSource).toContain('appStore.cachedPublicSettings?.doc_url || appStore.docUrl')
    expect(keyUsageViewSource).toContain('resolveDocumentationUrl(configuredDocUrl.value)')
  })
})

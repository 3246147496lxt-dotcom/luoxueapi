import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const dir = dirname(fileURLToPath(import.meta.url))
const appSidebarSource = readFileSync(resolve(dir, '../AppSidebar.vue'), 'utf8')
const shellDestinationsSource = readFileSync(
  resolve(dir, '../../../navigation/shellDestinations.ts'),
  'utf8',
)
const publicSiteLayoutSource = readFileSync(resolve(dir, '../../public/PublicSiteLayout.vue'), 'utf8')
const homeViewSource = readFileSync(resolve(dir, '../../../views/HomeView.vue'), 'utf8')
const keyUsageViewSource = readFileSync(resolve(dir, '../../../views/KeyUsageView.vue'), 'utf8')

describe('doc_url sanitization', () => {
  it('AppSidebar resolves the configured tutorial URL before rendering it in navigation', () => {
    expect(appSidebarSource).toContain(
      "import { resolveDocumentationUrl } from '@/utils/documentationUrl'",
    )
    expect(appSidebarSource).toContain('appStore.cachedPublicSettings?.doc_url || appStore.docUrl')
    expect(appSidebarSource).toContain('? documentationUrl.value')
    expect(appSidebarSource).toContain('const href = resolveSupportHref(spec)')
  })

  it('declares documentation as a configured support destination', () => {
    expect(shellDestinationsSource).toContain("id: 'documentation'")
    expect(shellDestinationsSource).toContain("kind: 'configured-href'")
    expect(shellDestinationsSource).toContain("source: 'documentation'")
    expect(shellDestinationsSource).not.toContain('https://luoxueapi.cc/tutorial-docs/')
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

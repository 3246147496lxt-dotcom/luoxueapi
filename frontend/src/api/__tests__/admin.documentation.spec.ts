import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post, put } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: { get, post, put },
}))

import {
  getDocumentation,
  listDocumentationRevisions,
  publishDocumentation,
  restoreDocumentationRevision,
  saveDocumentationDraft,
  uploadDocumentationAsset,
  type DocumentationContent,
} from '@/api/admin/documentation'

const content: DocumentationContent = {
  schema_version: 1,
  tutorials: [
    {
      id: 'quick-start',
      tab_label: '新手教程',
      icon: 'key',
      icon_svg: '<svg viewBox="0 0 24 24"><path d="M2 2h20v20H2z"/></svg>',
      description: '完成首次接入',
      steps: [{ title: '创建密钥', description: '打开密钥页面并创建。' }],
    },
  ],
}

describe('admin documentation API', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
    put.mockReset()
  })

  it('normalizes draft and published snapshots', async () => {
    get.mockResolvedValue({
      data: {
        draft: { content, updated_at: '2026-07-19T00:00:00Z' },
        published: { content, version: 3, updated_at: '2026-07-18T00:00:00Z' },
      },
    })

    const result = await getDocumentation()

    expect(get).toHaveBeenCalledWith('/admin/documentation')
    expect(result.draft.content).toEqual(content)
    expect(result.published?.version).toBe(3)
    expect(result.published?.published_at).toBe('2026-07-18T00:00:00Z')
  })

  it('round-trips optional site configuration in draft saves', async () => {
    const configuredContent: DocumentationContent = {
      ...content,
      site_config: { brand_name: '落雪API', support_contact: '客服' },
    }
    put.mockResolvedValue({ data: { content: configuredContent } })

    await saveDocumentationDraft(configuredContent)

    expect(put).toHaveBeenCalledWith('/admin/documentation/draft', {
      content: configuredContent,
    })
  })

  it('supports direct content payloads and both revision list shapes', async () => {
    get.mockResolvedValueOnce({ data: content })
    get.mockResolvedValueOnce({ data: [{ id: 4, version: 4 }] })
    get.mockResolvedValueOnce({ data: { items: [{ id: 5, version: 5 }] } })

    await expect(getDocumentation()).resolves.toMatchObject({ draft: { content } })
    await expect(listDocumentationRevisions()).resolves.toEqual({ items: [{ id: 4, version: 4 }] })
    await expect(listDocumentationRevisions()).resolves.toEqual({ items: [{ id: 5, version: 5 }] })
  })

  it('uses the draft, publish, restore, and multipart asset endpoints', async () => {
    put.mockResolvedValue({ data: { content, updated_at: '2026-07-19T01:00:00Z' } })
    post.mockResolvedValueOnce({ data: { draft: { content }, published: { content, version: 2 } } })
    post.mockResolvedValueOnce({ data: { draft: { content } } })
    post.mockResolvedValueOnce({ data: { url: '/documentation/assets/example.png' } })
    const image = new File(['image'], 'example.png', { type: 'image/png' })

    await saveDocumentationDraft(content)
    await publishDocumentation()
    await restoreDocumentationRevision('revision/2')
    await expect(uploadDocumentationAsset(image)).resolves.toEqual({ url: '/documentation/assets/example.png' })

    expect(put).toHaveBeenCalledWith('/admin/documentation/draft', { content })
    expect(post).toHaveBeenNthCalledWith(1, '/admin/documentation/publish')
    expect(post).toHaveBeenNthCalledWith(2, '/admin/documentation/revisions/revision%2F2/restore')
    expect(post.mock.calls[2][0]).toBe('/admin/documentation/assets')
    expect(post.mock.calls[2][1]).toBeInstanceOf(FormData)
    expect(post.mock.calls[2][2]).toEqual({
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    expect((post.mock.calls[2][1] as FormData).get('file')).toBe(image)
  })
})

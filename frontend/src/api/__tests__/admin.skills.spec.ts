import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post, put } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
}))

vi.mock('@/api/client', () => ({ apiClient: { get, post, put } }))

import {
  activateSkillVersion,
  archiveSkill,
  createSkill,
  getSkillMarketplaceConfig,
  listSkills,
  publishSkill,
  updateSkill,
  updateSkillMarketplaceConfig,
  uploadSkillVersion,
  yankSkillVersion,
  type CreateSkillRequest,
} from '@/api/admin/skills'

const draft: CreateSkillRequest = {
  slug: 'api-doc-writer',
  display_name: 'API 文档生成器',
  summary: '为接口生成接入文档',
  description: '读取项目中的接口定义并生成文档。',
  category: '文档与数据',
  tags: ['Codex', '文档'],
  icon: '',
  example_prompts: ['为这个接口生成接入文档'],
  risk_notes: '只读取当前项目文件。',
  featured: true,
  sort_order: 10,
}

describe('admin Skills API', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
    put.mockReset()
  })

  it('normalizes list responses and omits all-filter sentinels', async () => {
    get.mockResolvedValue({
      data: { items: [{ id: 1 }], total: 1, page: 2, page_size: 10 },
    })

    await expect(listSkills({
      status: 'all',
      featured: 'all',
      search: ' docs ',
      page: 2,
      page_size: 10,
    })).resolves.toEqual({ items: [{ id: 1 }], total: 1, page: 2, page_size: 10 })

    expect(get).toHaveBeenCalledWith('/admin/skills', {
      params: {
        status: undefined,
        search: 'docs',
        featured: undefined,
        page: 2,
        page_size: 10,
      },
    })
  })

  it('creates and fully updates Skill market metadata', async () => {
    post.mockResolvedValue({ data: { id: 4, ...draft } })
    put.mockResolvedValue({ data: { id: 4, ...draft, featured: false } })

    await createSkill(draft)
    await updateSkill(4, { ...draft, featured: false })

    expect(post).toHaveBeenCalledWith('/admin/skills', draft)
    expect(put).toHaveBeenCalledWith('/admin/skills/4', { ...draft, featured: false })
  })

  it('uploads a ZIP as multipart form data', async () => {
    post.mockResolvedValue({ data: { id: 7, version: '1.0.0' } })
    const file = new File(['zip'], 'skill.zip', { type: 'application/zip' })

    await uploadSkillVersion(4, { file, version: '1.0.0', changelog: 'Initial release' })

    const [url, formData, config] = post.mock.calls[0]
    expect(url).toBe('/admin/skills/4/versions')
    expect(formData).toBeInstanceOf(FormData)
    expect((formData as FormData).get('file')).toBe(file)
    expect((formData as FormData).get('version')).toBe('1.0.0')
    expect((formData as FormData).get('changelog')).toBe('Initial release')
    expect(config).toEqual({ headers: { 'Content-Type': 'multipart/form-data' } })
  })

  it('uses explicit lifecycle endpoints and version identifiers', async () => {
    post.mockResolvedValue({ data: { id: 4 } })

    await publishSkill(4, 7)
    await activateSkillVersion(4, 8)
    await yankSkillVersion(4, 7)
    await archiveSkill(4)

    expect(post).toHaveBeenNthCalledWith(1, '/admin/skills/4/publish', { version_id: 7 })
    expect(post).toHaveBeenNthCalledWith(2, '/admin/skills/4/versions/8/activate')
    expect(post).toHaveBeenNthCalledWith(3, '/admin/skills/4/versions/7/yank')
    expect(post).toHaveBeenNthCalledWith(4, '/admin/skills/4/archive')
  })

  it('loads and updates the public marketplace gate', async () => {
    get.mockResolvedValue({ data: { enabled: false } })
    put.mockResolvedValue({ data: { enabled: true } })

    await expect(getSkillMarketplaceConfig()).resolves.toEqual({ enabled: false })
    await expect(updateSkillMarketplaceConfig(true)).resolves.toEqual({ enabled: true })

    expect(get).toHaveBeenCalledWith('/admin/skills/config')
    expect(put).toHaveBeenCalledWith('/admin/skills/config', { enabled: true })
  })
})

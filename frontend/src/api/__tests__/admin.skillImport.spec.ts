import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post, put } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
}))

vi.mock('@/api/client', () => ({ apiClient: { get, post, put } }))

import {
  createSkillImportRun,
  createSkillImportSchedule,
  createSkillImportSource,
  uploadSkillImportRun,
} from '@/api/admin/skillImport'

describe('admin Skill import API', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
    put.mockReset()
    post.mockResolvedValue({ data: { id: 7 } })
  })

  it('sends the versioned run contract without a request_config envelope', async () => {
    const request = {
      source_id: 1,
      mode: 'auto_publish' as const,
      selection: { start_rank: 1, limit: 500 },
      run_config: {
        safe_gate: true,
        auto_publish_gate: {
          require_all_valid: false,
          allow_license_unverified: true,
        },
      },
      publish_policy: 'auto_publish' as const,
      metadata_policy: 'refresh' as const,
    }

    await createSkillImportRun(request, 'run-key')

    expect(post).toHaveBeenCalledWith('/admin/skill-import/runs', request, {
      headers: { 'Idempotency-Key': 'run-key' },
    })
    expect(post.mock.calls[0][1]).not.toHaveProperty('request_config')
  })

  it('uses idempotency keys when creating sources and schedules', async () => {
    const source = {
      name: 'skills.sh',
      adapter: 'skills_sh' as const,
      namespace: 'skills.sh',
      base_url: 'https://skills.sh',
      source_config: {},
      catalog_priority: 10,
      enabled: true,
    }
    const schedule = {
      source_id: 1,
      name: 'Daily top 500',
      enabled: false,
      cron_expression: '0 3 * * *',
      timezone: 'Asia/Shanghai',
      selection: { start_rank: 1, limit: 500 },
      run_config: { safe_gate: true },
      publish_policy: 'auto_publish' as const,
      metadata_policy: 'refresh' as const,
    }

    await createSkillImportSource(source, 'source-key')
    await createSkillImportSchedule(schedule, 'schedule-key')

    expect(post).toHaveBeenNthCalledWith(1, '/admin/skill-import/sources', source, {
      headers: { 'Idempotency-Key': 'source-key' },
    })
    expect(post).toHaveBeenNthCalledWith(2, '/admin/skill-import/schedules', schedule, {
      headers: { 'Idempotency-Key': 'schedule-key' },
    })
  })

  it('serializes manifest uploads with the same run fields', async () => {
    const file = new File(['[]'], 'skills.json', { type: 'application/json' })
    await uploadSkillImportRun({
      file,
      source_id: 1,
      mode: 'review',
      selection: { start_rank: 1, limit: 10 },
      run_config: { safe_gate: true },
      publish_policy: 'review',
      metadata_policy: 'refresh',
    }, 'upload-key')

    const [url, formData, config] = post.mock.calls[0]
    expect(url).toBe('/admin/skill-import/runs/upload')
    expect((formData as FormData).get('file')).toBe(file)
    expect((formData as FormData).get('selection')).toBe('{"start_rank":1,"limit":10}')
    expect((formData as FormData).get('run_config')).toBe('{"safe_gate":true}')
    expect((formData as FormData).get('publish_policy')).toBe('review')
    expect((formData as FormData).get('metadata_policy')).toBe('refresh')
    expect(config).toEqual({
      headers: {
        'Idempotency-Key': 'upload-key',
      },
    })
    expect(config.headers).not.toHaveProperty('Content-Type')
  })
})

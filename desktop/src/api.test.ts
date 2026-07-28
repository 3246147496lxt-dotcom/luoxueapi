import { describe, expect, it } from 'vitest'
import { desktopApi } from './api'

describe('browser desktop adapter', () => {
  it('models the takeover, route and settings flow without exposing credentials', async () => {
    const initial = await desktopApi.snapshot()
    expect(initial.paired).toBe(true)
    expect(JSON.stringify(initial)).not.toMatch(/api[_-]?key|refresh[_-]?token/i)

    const restored = await desktopApi.setTakeover(false)
    expect(restored.takeoverEnabled).toBe(false)
    expect(restored.configStatus).toBe('clean')

    const selected = await desktopApi.selectRoute(18, 'gpt-5.4')
    expect(selected.selectedGroupId).toBe(18)
    expect(selected.selectedModel).toBe('gpt-5.4')

    const updated = await desktopApi.updateSettings({
      ...selected.settings,
      retentionDays: 14,
      theme: 'dark'
    })
    expect(updated.settings.retentionDays).toBe(14)
    expect(updated.settings.theme).toBe('dark')
  })

  it('previews only redacted diagnostic metadata before upload', async () => {
    const summary = await desktopApi.diagnosticSummary()
    const serialized = JSON.stringify(summary)

    expect(summary.requests.recent.length).toBeLessThanOrEqual(20)
    expect(serialized).not.toMatch(/accountEmail|deviceName|authorization|api[_-]?key|refresh[_-]?token|prompt|responseBody|\/Users\//i)

    const receipt = await desktopApi.uploadDiagnostic()
    expect(receipt.id).toMatch(/^diag_/)
  })
})

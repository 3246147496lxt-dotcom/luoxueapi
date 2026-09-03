import { describe, expect, it } from 'vitest'
import {
  appendChatVoiceWaveformSample,
  CHAT_VOICE_WAVEFORM_SAMPLE_INTERVAL_MS,
  CHAT_VOICE_WAVEFORM_STEP_PX,
  chatVoiceWaveformCapacity,
  chatVoiceWaveformHeight,
  chatVoiceWaveformPosition,
  resizeChatVoiceWaveformTrack,
} from '../chatVoiceWaveform'

describe('chatVoiceWaveform', () => {
  it('maps silence and full-scale input into the official visual height range', () => {
    expect(chatVoiceWaveformHeight(0)).toBe(5)
    expect(chatVoiceWaveformHeight(1)).toBe(40)
    expect(chatVoiceWaveformHeight(-1)).toBe(5)
    expect(chatVoiceWaveformHeight(2)).toBe(40)
  })

  it('keeps recorded levels dynamic when they are committed to the track', () => {
    const levels = [0, 0.2, 0.6, 1]
    let track = resizeChatVoiceWaveformTrack([], levels.length)

    for (const level of levels) {
      track = appendChatVoiceWaveformSample(track, level, levels.length)
    }

    const expectedHeights = levels.map(chatVoiceWaveformHeight)
    expect(track).toEqual(expectedHeights)
    expect(new Set(track).size).toBe(levels.length)
    expect(expectedHeights[0]).toBe(5)
    expect(expectedHeights.at(-1)).toBe(40)
  })

  it('moves one shared dot-and-wave track left as new samples enter from the right', () => {
    let history = resizeChatVoiceWaveformTrack([], 3)
    expect(history).toEqual([null, null, null])

    history = appendChatVoiceWaveformSample(history, 0.5, 3)
    const firstWaveBar = history[2]
    expect(history).toEqual([null, null, firstWaveBar])

    history = appendChatVoiceWaveformSample(history, 1, 3)
    expect(history[0]).toBeNull()
    expect(history[1]).toBe(firstWaveBar)
    expect(history[2]).toBe(40)
  })

  it('derives capacity from the six-pixel sample pitch', () => {
    expect(chatVoiceWaveformCapacity(600)).toBe(101)
    expect(chatVoiceWaveformCapacity(5)).toBe(1)
    expect(chatVoiceWaveformCapacity(0)).toBe(0)
  })

  it('matches the reference cadence of roughly six new bars per second', () => {
    expect(CHAT_VOICE_WAVEFORM_SAMPLE_INTERVAL_MS).toBe(166)
  })

  it('keeps every old dot and bar continuous across a sampling boundary', () => {
    const trackStart = 60
    const beforeSecondItem = chatVoiceWaveformPosition(1, 1, trackStart)
    const afterFirstItem = chatVoiceWaveformPosition(0, 0, trackStart)

    expect(afterFirstItem).toBe(beforeSecondItem)
  })

  it('moves dots and bars by the same distance within a sampling interval', () => {
    const trackStart = 60
    const dotDelta = chatVoiceWaveformPosition(1, 0.75, trackStart)
      - chatVoiceWaveformPosition(1, 0.25, trackStart)
    const barDelta = chatVoiceWaveformPosition(8, 0.75, trackStart)
      - chatVoiceWaveformPosition(8, 0.25, trackStart)

    expect(dotDelta).toBe(-3)
    expect(barDelta).toBe(dotDelta)
  })

  it('preserves the dot-wave boundary instead of deleting dots from the middle', () => {
    const before = [null, null, 12, 18]
    const after = appendChatVoiceWaveformSample(before, 1, before.length)

    expect(after).toEqual([null, 12, 18, 40])
  })

  it('keeps newest samples right-aligned while resizing the track', () => {
    const history = [null, 10, 20, 30]

    expect(resizeChatVoiceWaveformTrack(history, 3)).toEqual([10, 20, 30])
    expect(resizeChatVoiceWaveformTrack(history, 6)).toEqual([null, null, null, 10, 20, 30])
  })

  it('places the newest sample at the right edge after the whole track shifts', () => {
    const trackStart = 60
    const capacity = 3
    const waveEnd = trackStart + ((capacity - 1) * CHAT_VOICE_WAVEFORM_STEP_PX)

    expect(chatVoiceWaveformPosition(capacity - 1, 0, trackStart)).toBe(waveEnd)
  })
})

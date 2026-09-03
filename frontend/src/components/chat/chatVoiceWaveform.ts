export const CHAT_VOICE_WAVEFORM_STEP_PX = 6
export const CHAT_VOICE_WAVEFORM_SAMPLE_INTERVAL_MS = 166
export type ChatVoiceWaveformSample = number | null

const MINIMUM_BAR_HEIGHT_PX = 5
const MAXIMUM_BAR_HEIGHT_PX = 40
const LEVEL_FLOOR = 0.05

export function chatVoiceWaveformCapacity(width: number): number {
  if (!Number.isFinite(width) || width <= 0) return 0
  return Math.max(1, Math.floor(width / CHAT_VOICE_WAVEFORM_STEP_PX) + 1)
}

export function chatVoiceWaveformPosition(
  sampleIndex: number,
  sampleProgress: number,
  trackStart: number,
): number {
  const index = Math.max(0, Math.floor(Number.isFinite(sampleIndex) ? sampleIndex : 0))
  const progress = Math.max(0, Math.min(1, Number.isFinite(sampleProgress) ? sampleProgress : 0))
  return trackStart + ((index - progress) * CHAT_VOICE_WAVEFORM_STEP_PX)
}

export function chatVoiceWaveformHeight(level: number): number {
  const normalized = Math.max(0, Math.min(1, Number.isFinite(level) ? level : 0))
  if (normalized <= LEVEL_FLOOR) return MINIMUM_BAR_HEIGHT_PX
  const audible = (normalized - LEVEL_FLOOR) / (1 - LEVEL_FLOOR)
  return MINIMUM_BAR_HEIGHT_PX
    + (Math.pow(audible, 0.72) * (MAXIMUM_BAR_HEIGHT_PX - MINIMUM_BAR_HEIGHT_PX))
}

export function appendChatVoiceWaveformSample(
  history: ChatVoiceWaveformSample[],
  rawLevel: number,
  capacity: number,
): ChatVoiceWaveformSample[] {
  if (capacity <= 0) return []
  const track = resizeChatVoiceWaveformTrack(history, capacity)
  return [...track.slice(1), chatVoiceWaveformHeight(rawLevel)]
}

export function resizeChatVoiceWaveformTrack(
  history: ChatVoiceWaveformSample[],
  capacity: number,
): ChatVoiceWaveformSample[] {
  if (capacity <= 0) return []
  const latest = history.slice(-capacity)
  return [
    ...Array<ChatVoiceWaveformSample>(capacity - latest.length).fill(null),
    ...latest,
  ]
}

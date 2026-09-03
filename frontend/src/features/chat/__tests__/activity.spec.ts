import { describe, expect, it, vi } from 'vitest'
import {
  ChatActivityStateMachine,
  chatActivityPartKey,
  chatActivityPartText,
  normalizeChatActivities,
} from '@/features/chat/activity'
import type { ChatActivity, ChatActivityEvent, ChatActivityEventType } from '@/types/chat'

function activityEvent(
  eventType: ChatActivityEventType,
  payload: Record<string, unknown>,
): ChatActivityEvent {
  return { source: 'openai_responses', eventType, payload }
}

function deterministicClock(): () => number {
  let now = 1_000
  return () => ++now
}

function seedReasoningSummary(
  machine: ChatActivityStateMachine,
  responseId = 'resp-batch',
): void {
  machine.consume(activityEvent('response.created', {
    sequence_number: 0,
    response: { id: responseId },
  }))
  machine.consume(activityEvent('response.output_item.added', {
    sequence_number: 1,
    output_index: 0,
    item: { id: 'reasoning-batch', type: 'reasoning' },
  }))
  machine.consume(activityEvent('response.reasoning_summary_part.added', {
    sequence_number: 2,
    item_id: 'reasoning-batch',
    output_index: 0,
    summary_index: 0,
    part: { type: 'summary_text' },
  }))
}

function consumeSummaryDelta(
  machine: ChatActivityStateMachine,
  sequenceNumber: number,
  delta: string,
): void {
  machine.consume(activityEvent('response.reasoning_summary_text.delta', {
    sequence_number: sequenceNumber,
    item_id: 'reasoning-batch',
    output_index: 0,
    summary_index: 0,
    delta,
  }))
}

describe('ChatActivityStateMachine', () => {
  it('keeps multiple reasoning items and summary parts in canonical order', () => {
    const snapshots: ChatActivity[] = []
    const machine = new ChatActivityStateMachine({
      onActivity: (activity) => snapshots.push(activity),
      reasoningMode: 'pro',
      reasoningEffort: 'medium',
      now: deterministicClock(),
    })

    machine.consume(activityEvent('response.created', {
      sequence_number: 0,
      response: { id: 'resp-1' },
    }))
    expect(snapshots).toEqual([])

    machine.consume(activityEvent('response.output_item.added', {
      sequence_number: 1,
      output_index: 2,
      item: { id: 'reasoning-2', type: 'reasoning' },
    }))
    machine.consume(activityEvent('response.reasoning_summary_part.added', {
      sequence_number: 2,
      item_id: 'reasoning-2',
      output_index: 2,
      summary_index: 1,
      part: { type: 'summary_text', text: '' },
    }))
    machine.consume(activityEvent('response.reasoning_summary_text.delta', {
      sequence_number: 3,
      item_id: 'reasoning-2',
      output_index: 2,
      summary_index: 1,
      delta: 'Second',
    }))
    machine.consume(activityEvent('response.output_item.added', {
      sequence_number: 4,
      output_index: 0,
      item: { id: 'reasoning-1', type: 'reasoning' },
    }))
    machine.consume(activityEvent('response.reasoning_summary_part.added', {
      sequence_number: 5,
      item_id: 'reasoning-1',
      output_index: 0,
      summary_index: 0,
      part: { type: 'summary_text', text: '' },
    }))
    machine.consume(activityEvent('response.reasoning_summary_text.delta', {
      sequence_number: 6,
      item_id: 'reasoning-1',
      output_index: 0,
      summary_index: 0,
      delta: 'Al',
    }))
    machine.consume(activityEvent('response.reasoning_summary_text.done', {
      sequence_number: 7,
      item_id: 'reasoning-1',
      output_index: 0,
      summary_index: 0,
      text: 'Alpha',
    }))
    machine.consume(activityEvent('response.reasoning_summary_part.done', {
      sequence_number: 8,
      item_id: 'reasoning-1',
      output_index: 0,
      summary_index: 0,
      part: { type: 'summary_text', text: 'must not overwrite text.done' },
    }))
    machine.consume(activityEvent('response.reasoning_summary_text.delta', {
      sequence_number: 9,
      item_id: 'reasoning-1',
      output_index: 0,
      summary_index: 0,
      delta: 'must also be ignored after text.done',
    }))
    machine.consume(activityEvent('response.completed', {
      sequence_number: 10,
      response: { id: 'resp-1' },
    }))

    const activity = snapshots.at(-1)!
    expect(activity).toMatchObject({
      key: 'resp-1',
      responseId: 'resp-1',
      status: 'completed',
      reasoningMode: 'pro',
    })
    expect(activity.reasoningEffort).toBeUndefined()
    expect(activity.items.map(({ itemId }) => itemId)).toEqual([
      'reasoning-1',
      'reasoning-2',
    ])
    expect(activity.items[0]?.parts[0]).toMatchObject({
      summaryIndex: 0,
      text: 'Alpha',
      status: 'completed',
    })
    expect(activity.items[1]?.parts[0]?.text).toBe('Second')
  })

  it.each([
    ['response.completed', 'completed'],
    ['response.incomplete', 'incomplete'],
    ['response.failed', 'failed'],
  ] as const)('restores a terminal-only reasoning summary from %s response.output', (
    eventType,
    expectedStatus,
  ) => {
    const snapshots: ChatActivity[] = []
    const machine = new ChatActivityStateMachine({
      onActivity: (activity) => snapshots.push(activity),
      reasoningMode: 'pro',
      reasoningEffort: 'medium',
      now: deterministicClock(),
    })

    machine.consume(activityEvent(eventType, {
      sequence_number: 10,
      response: {
        id: `resp-terminal-only-${expectedStatus}`,
        reasoning: { mode: 'standard', effort: 'high' },
        output: [
          {
            id: 'message-terminal',
            type: 'message',
            content: [{ type: 'output_text', text: 'assistant answer' }],
          },
          {
            id: 'reasoning-terminal',
            type: 'reasoning',
            encrypted_content: 'must-not-leak',
            reasoning_text: 'private chain of thought',
            summary: [
              { type: 'summary_text', text: 'First terminal summary' },
              { type: 'summary_text', summary_index: 2, text: 'Third terminal summary' },
            ],
          },
        ],
      },
    }))

    expect(snapshots).toHaveLength(1)
    expect(snapshots[0]).toMatchObject({
      responseId: `resp-terminal-only-${expectedStatus}`,
      status: expectedStatus,
      reasoningMode: 'pro',
      items: [{
        itemId: 'reasoning-terminal',
        outputIndex: 1,
        status: expectedStatus,
        parts: [
          { summaryIndex: 0, text: 'First terminal summary', status: expectedStatus },
          { summaryIndex: 2, text: 'Third terminal summary', status: expectedStatus },
        ],
      }],
    })
    expect(snapshots[0]?.reasoningEffort).toBeUndefined()
    expect(JSON.stringify(snapshots[0])).not.toContain('assistant answer')
    expect(JSON.stringify(snapshots[0])).not.toContain('must-not-leak')
    expect(JSON.stringify(snapshots[0])).not.toContain('private chain of thought')
    machine.dispose()
  })

  it('ignores terminal output without an official reasoning summary_text item', () => {
    const snapshots: ChatActivity[] = []
    const machine = new ChatActivityStateMachine({
      onActivity: (activity) => snapshots.push(activity),
      now: deterministicClock(),
    })

    machine.consume(activityEvent('response.failed', {
      sequence_number: 1,
      response: {
        id: 'resp-terminal-filtered',
        output: [
          {
            id: 'message-with-fake-summary',
            type: 'message',
            summary: [{ type: 'summary_text', text: 'not reasoning' }],
          },
          {
            id: 'reasoning-without-summary-text',
            type: 'reasoning',
            encrypted_content: 'ciphertext',
            reasoning_text: 'private reasoning',
            summary: [
              { type: 'output_text', text: 'not a summary' },
              { type: 'summary_text', text: { invalid: true } },
            ],
          },
        ],
      },
    }))

    expect(snapshots).toEqual([])
    expect(machine.snapshots()).toEqual([])
    machine.dispose()
  })

  it('deduplicates lifecycle parts and lets terminal output correct their text', () => {
    const snapshots: ChatActivity[] = []
    const machine = new ChatActivityStateMachine({
      onActivity: (activity) => snapshots.push(activity),
      now: deterministicClock(),
    })
    seedReasoningSummary(machine, 'resp-terminal-correction')
    consumeSummaryDelta(machine, 3, 'draft')
    machine.consume(activityEvent('response.reasoning_summary_text.done', {
      sequence_number: 4,
      item_id: 'reasoning-batch',
      output_index: 0,
      summary_index: 0,
      text: 'lifecycle text',
    }))
    machine.consume(activityEvent('response.reasoning_summary_part.done', {
      sequence_number: 5,
      item_id: 'reasoning-batch',
      output_index: 0,
      summary_index: 0,
      status: 'incomplete',
      part: { type: 'summary_text', text: 'must not beat text.done', status: 'incomplete' },
    }))
    machine.consume(activityEvent('response.completed', {
      sequence_number: 6,
      response: {
        id: 'resp-terminal-correction',
        output: [{
          id: 'reasoning-batch',
          type: 'reasoning',
          encrypted_content: 'must-not-leak',
          summary: [{ type: 'summary_text', text: 'terminal canonical text' }],
        }],
      },
    }))

    const activity = snapshots.at(-1)!
    expect(activity).toMatchObject({
      status: 'incomplete',
      items: [{
        itemId: 'reasoning-batch',
        status: 'incomplete',
        parts: [{
          summaryIndex: 0,
          text: 'terminal canonical text',
          status: 'incomplete',
        }],
      }],
    })
    expect(activity.items).toHaveLength(1)
    expect(activity.items[0]?.parts).toHaveLength(1)
    expect(JSON.stringify(activity)).not.toContain('must-not-leak')
    machine.dispose()
  })

  it('deduplicates sequence numbers and reorders nearby deltas', () => {
    const snapshots: ChatActivity[] = []
    const machine = new ChatActivityStateMachine({
      onActivity: (activity) => snapshots.push(activity),
      now: deterministicClock(),
    })
    machine.consume(activityEvent('response.created', {
      sequence_number: 0,
      response: { id: 'resp-order' },
    }))
    machine.consume(activityEvent('response.output_item.added', {
      sequence_number: 1,
      output_index: 0,
      item: { id: 'reasoning-order', type: 'reasoning' },
    }))
    machine.consume(activityEvent('response.reasoning_summary_part.added', {
      sequence_number: 2,
      item_id: 'reasoning-order',
      output_index: 0,
      summary_index: 0,
      part: { type: 'summary_text' },
    }))
    machine.consume(activityEvent('response.reasoning_summary_text.delta', {
      sequence_number: 4,
      item_id: 'reasoning-order',
      output_index: 0,
      summary_index: 0,
      delta: 'B',
    }))
    machine.consume(activityEvent('response.reasoning_summary_text.delta', {
      sequence_number: 3,
      item_id: 'reasoning-order',
      output_index: 0,
      summary_index: 0,
      delta: 'A',
    }))
    machine.consume(activityEvent('response.reasoning_summary_text.delta', {
      sequence_number: 4,
      item_id: 'reasoning-order',
      output_index: 0,
      summary_index: 0,
      delta: 'duplicate',
    }))
    machine.flush()

    expect(chatActivityPartText(snapshots.at(-1)!.items[0]!.parts[0]!)).toBe('AB')
  })

  it('does not expose or persist a reasoning item that never returns a summary part', () => {
    const snapshots: ChatActivity[] = []
    const machine = new ChatActivityStateMachine({
      onActivity: (activity) => snapshots.push(activity),
      now: deterministicClock(),
    })
    machine.consume(activityEvent('response.created', {
      sequence_number: 0,
      response: { id: 'resp-empty' },
    }))
    machine.consume(activityEvent('response.output_item.added', {
      sequence_number: 1,
      output_index: 0,
      item: { id: 'reasoning-empty', type: 'reasoning', summary: [] },
    }))
    machine.consume(activityEvent('response.output_item.done', {
      sequence_number: 2,
      output_index: 0,
      item: { id: 'reasoning-empty', type: 'reasoning', summary: [] },
    }))
    machine.consume(activityEvent('response.completed', {
      sequence_number: 3,
      response: { id: 'resp-empty', status: 'completed', output: [] },
    }))

    expect(snapshots).toEqual([])
    expect(machine.snapshots()).toEqual([])
    machine.dispose()
  })

  it('rejects orphan summary events without a confirmed reasoning output item', () => {
    const machine = new ChatActivityStateMachine({ now: deterministicClock() })
    machine.consume(activityEvent('response.created', {
      sequence_number: 0,
      response: { id: 'resp-orphan-summary' },
    }))
    machine.consume(activityEvent('response.reasoning_summary_text.delta', {
      sequence_number: 1,
      item_id: 'unconfirmed-item',
      output_index: 0,
      summary_index: 0,
      delta: 'must not be exposed',
    }))

    expect(machine.snapshots()).toEqual([])
    machine.dispose()
  })

  it('rejects summary events attached to a non-reasoning output item', () => {
    const machine = new ChatActivityStateMachine({ now: deterministicClock() })
    machine.consume(activityEvent('response.created', {
      sequence_number: 0,
      response: { id: 'resp-message-summary' },
    }))
    machine.consume(activityEvent('response.output_item.added', {
      sequence_number: 1,
      output_index: 0,
      item: { id: 'message-item', type: 'message' },
    }))
    machine.consume(activityEvent('response.reasoning_summary_part.added', {
      sequence_number: 2,
      item_id: 'message-item',
      output_index: 0,
      summary_index: 0,
      part: { type: 'summary_text' },
    }))
    machine.consume(activityEvent('response.reasoning_summary_text.delta', {
      sequence_number: 3,
      item_id: 'message-item',
      output_index: 0,
      summary_index: 0,
      delta: 'must not be exposed',
    }))

    expect(machine.snapshots()).toEqual([])
    machine.dispose()
  })

  it('buffers an out-of-arrival-order summary until its reasoning item is confirmed', () => {
    const snapshots: ChatActivity[] = []
    const machine = new ChatActivityStateMachine({
      onActivity: (activity) => snapshots.push(activity),
      now: deterministicClock(),
    })
    machine.consume(activityEvent('response.created', {
      sequence_number: 0,
      response: { id: 'resp-summary-reorder' },
    }))
    machine.consume(activityEvent('response.reasoning_summary_part.added', {
      sequence_number: 2,
      item_id: 'reasoning-reorder',
      output_index: 0,
      summary_index: 0,
      part: { type: 'summary_text' },
    }))
    expect(machine.snapshots()).toEqual([])
    machine.consume(activityEvent('response.output_item.added', {
      sequence_number: 1,
      output_index: 0,
      item: { id: 'reasoning-reorder', type: 'reasoning' },
    }))
    machine.consume(activityEvent('response.reasoning_summary_text.delta', {
      sequence_number: 3,
      item_id: 'reasoning-reorder',
      output_index: 0,
      summary_index: 0,
      delta: 'confirmed',
    }))
    machine.flush()

    expect(chatActivityPartText(snapshots.at(-1)!.items[0]!.parts[0]!)).toBe('confirmed')
    machine.dispose()
  })

  it('preserves an incomplete status carried by reasoning_summary_part.done', () => {
    const snapshots: ChatActivity[] = []
    const machine = new ChatActivityStateMachine({
      onActivity: (activity) => snapshots.push(activity),
      now: deterministicClock(),
    })
    seedReasoningSummary(machine, 'resp-incomplete-part')
    snapshots.length = 0
    machine.consume(activityEvent('response.reasoning_summary_text.delta', {
      sequence_number: 3,
      item_id: 'reasoning-batch',
      output_index: 0,
      summary_index: 0,
      delta: 'partial',
    }))
    machine.consume(activityEvent('response.reasoning_summary_part.done', {
      sequence_number: 4,
      item_id: 'reasoning-batch',
      output_index: 0,
      summary_index: 0,
      status: 'incomplete',
      part: { type: 'summary_text', text: 'partial', status: 'incomplete' },
    }))

    expect(snapshots.at(-1)?.items[0]?.parts[0]).toMatchObject({
      text: 'partial',
      status: 'incomplete',
    })
    machine.dispose()
  })

  it.each([
    { requestedMode: 'standard' as const, upstreamMode: 'pro' as const },
    { requestedMode: 'pro' as const, upstreamMode: 'standard' as const },
  ])(
    'keeps request reasoning metadata authoritative for $requestedMode versus $upstreamMode',
    ({ requestedMode, upstreamMode }) => {
      const snapshots: ChatActivity[] = []
      const machine = new ChatActivityStateMachine({
        onActivity: (activity) => snapshots.push(activity),
        reasoningMode: requestedMode,
        reasoningEffort: 'medium',
        now: deterministicClock(),
      })
      machine.consume(activityEvent('response.created', {
        sequence_number: 0,
        response: {
          id: `resp-mode-${requestedMode}`,
          reasoning: { mode: upstreamMode, effort: 'high' },
        },
      }))
      machine.consume(activityEvent('response.output_item.added', {
        sequence_number: 1,
        output_index: 0,
        item: { id: 'reasoning-mode', type: 'reasoning' },
      }))
      machine.consume(activityEvent('response.reasoning_summary_part.added', {
        sequence_number: 2,
        item_id: 'reasoning-mode',
        output_index: 0,
        summary_index: 0,
        part: { type: 'summary_text' },
      }))
      machine.consume(activityEvent('response.completed', {
        sequence_number: 3,
        response: {
          id: `resp-mode-${requestedMode}`,
          reasoning: { mode: upstreamMode, effort: 'high' },
        },
      }))

      const activity = snapshots.at(-1)
      expect(activity?.reasoningMode).toBe(requestedMode)
      expect(activity?.reasoningEffort).toBe(requestedMode === 'pro' ? undefined : 'medium')
      machine.dispose()
    },
  )

  it('keeps nested incomplete severity when the response terminal event is completed', () => {
    const snapshots: ChatActivity[] = []
    const machine = new ChatActivityStateMachine({
      onActivity: (activity) => snapshots.push(activity),
      now: deterministicClock(),
    })
    seedReasoningSummary(machine, 'resp-incomplete-completed')
    machine.consume(activityEvent('response.reasoning_summary_text.delta', {
      sequence_number: 3,
      item_id: 'reasoning-batch',
      output_index: 0,
      summary_index: 0,
      delta: 'partial',
    }))
    machine.consume(activityEvent('response.reasoning_summary_part.done', {
      sequence_number: 4,
      item_id: 'reasoning-batch',
      output_index: 0,
      summary_index: 0,
      status: 'incomplete',
      part: { type: 'summary_text', text: 'partial', status: 'incomplete' },
    }))
    machine.consume(activityEvent('response.completed', {
      sequence_number: 5,
      response: { id: 'resp-incomplete-completed', status: 'completed' },
    }))

    expect(snapshots.at(-1)).toMatchObject({
      status: 'incomplete',
      items: [{
        status: 'incomplete',
        parts: [{ status: 'incomplete', text: 'partial' }],
      }],
    })
    machine.dispose()
  })

  it('batches consecutive delta snapshots into one bounded notification', () => {
    vi.useFakeTimers()
    try {
      const snapshots: ChatActivity[] = []
      const machine = new ChatActivityStateMachine({
        onActivity: (activity) => snapshots.push(activity),
        notificationBatchMs: 40,
        now: deterministicClock(),
      })
      seedReasoningSummary(machine)
      snapshots.length = 0

      consumeSummaryDelta(machine, 3, 'A')
      consumeSummaryDelta(machine, 4, 'B')
      consumeSummaryDelta(machine, 5, 'C')

      expect(chatActivityPartText(machine.snapshots().at(-1)!.items[0]!.parts[0]!)).toBe('ABC')
      expect(snapshots).toEqual([])
      vi.advanceTimersByTime(39)
      expect(snapshots).toEqual([])
      vi.advanceTimersByTime(1)
      expect(snapshots).toHaveLength(1)
      expect(chatActivityPartText(snapshots[0]!.items[0]!.parts[0]!)).toBe('ABC')
      machine.dispose()
    } finally {
      vi.useRealTimers()
    }
  })

  it('buffers a long summary as chunks without copying the accumulated text', () => {
    vi.useFakeTimers()
    try {
      const snapshots: ChatActivity[] = []
      const machine = new ChatActivityStateMachine({
        onActivity: (activity) => snapshots.push(activity),
        notificationBatchMs: 40,
        now: deterministicClock(),
      })
      seedReasoningSummary(machine, 'resp-long-summary')
      snapshots.length = 0

      const chunk = '0123456789abcdef'
      for (let index = 0; index < 10_000; index += 1) {
        consumeSummaryDelta(machine, index + 3, chunk)
      }

      expect(snapshots).toEqual([])
      vi.advanceTimersByTime(40)
      expect(snapshots).toHaveLength(1)
      const part = snapshots[0]!.items[0]!.parts[0]!
      expect(part.text).toBe('')
      expect(part.streamingTextChunks).toHaveLength(1)
      expect(chatActivityPartText(part)).toBe(chunk.repeat(10_000))
      machine.dispose()
    } finally {
      vi.useRealTimers()
    }
  })

  it('keeps separate stream segments across notification batches until a terminal event', () => {
    vi.useFakeTimers()
    try {
      const snapshots: ChatActivity[] = []
      const machine = new ChatActivityStateMachine({
        onActivity: (activity) => snapshots.push(activity),
        notificationBatchMs: 40,
        now: deterministicClock(),
      })
      seedReasoningSummary(machine, 'resp-multi-batch-summary')
      snapshots.length = 0

      consumeSummaryDelta(machine, 3, 'first ')
      vi.advanceTimersByTime(40)
      consumeSummaryDelta(machine, 4, 'second ')
      vi.advanceTimersByTime(40)
      consumeSummaryDelta(machine, 5, 'third')
      vi.advanceTimersByTime(40)

      expect(snapshots).toHaveLength(3)
      const streamingPart = snapshots.at(-1)!.items[0]!.parts[0]!
      expect(streamingPart.text).toBe('')
      expect(streamingPart.streamingTextChunks).toEqual(['first ', 'second ', 'third'])
      expect(chatActivityPartText(streamingPart)).toBe('first second third')

      machine.consume(activityEvent('response.reasoning_summary_text.done', {
        sequence_number: 6,
        item_id: 'reasoning-batch',
        output_index: 0,
        summary_index: 0,
        text: 'first second third',
      }))
      const completedPart = snapshots.at(-1)!.items[0]!.parts[0]!
      expect(completedPart.text).toBe('first second third')
      expect(completedPart.streamingTextChunks).toBeUndefined()
      machine.dispose()
    } finally {
      vi.useRealTimers()
    }
  })

  it.each([
    {
      eventType: 'response.reasoning_summary_text.done' as const,
      payload: {
        sequence_number: 6,
        item_id: 'reasoning-batch',
        output_index: 0,
        summary_index: 0,
        text: 'authoritative',
      },
      expectedStatus: 'streaming',
      expectedText: 'authoritative',
    },
    {
      eventType: 'response.reasoning_summary_part.done' as const,
      payload: {
        sequence_number: 6,
        item_id: 'reasoning-batch',
        output_index: 0,
        summary_index: 0,
        part: { type: 'summary_text', text: 'part final' },
      },
      expectedStatus: 'streaming',
      expectedText: 'part final',
    },
    {
      eventType: 'response.completed' as const,
      payload: { sequence_number: 6, response: { id: 'resp-batch' } },
      expectedStatus: 'completed',
      expectedText: 'draft',
    },
    {
      eventType: 'response.incomplete' as const,
      payload: { sequence_number: 6, response: { id: 'resp-batch' } },
      expectedStatus: 'incomplete',
      expectedText: 'draft',
    },
    {
      eventType: 'response.failed' as const,
      payload: { sequence_number: 6, response: { id: 'resp-batch' } },
      expectedStatus: 'failed',
      expectedText: 'draft',
    },
  ])('flushes a sequence gap and notifies $eventType immediately with the latest delta', ({
    eventType,
    payload,
    expectedStatus,
    expectedText,
  }) => {
    vi.useFakeTimers()
    try {
      const snapshots: ChatActivity[] = []
      const machine = new ChatActivityStateMachine({
        onActivity: (activity) => snapshots.push(activity),
        notificationBatchMs: 40,
        now: deterministicClock(),
      })
      seedReasoningSummary(machine)
      snapshots.length = 0
      // Sequences 3 and 5 belong to public stream events which are not copied
      // into the private Activity envelope.
      consumeSummaryDelta(machine, 4, 'draft')
      expect(snapshots).toEqual([])

      machine.consume(activityEvent(eventType, payload))

      expect(snapshots).toHaveLength(1)
      expect(snapshots[0]).toMatchObject({ status: expectedStatus })
      expect(snapshots[0]?.items[0]?.parts[0]?.text).toBe(expectedText)
      vi.advanceTimersByTime(100)
      expect(snapshots).toHaveLength(1)
      machine.dispose()
    } finally {
      vi.useRealTimers()
    }
  })

  it.each([
    ['flush', 'streaming'],
    ['dispose', 'streaming'],
    ['stop', 'stopped'],
    ['disconnect', 'disconnected'],
  ] as const)('%s synchronously flushes a pending delta snapshot', (operation, status) => {
    vi.useFakeTimers()
    try {
      const snapshots: ChatActivity[] = []
      const machine = new ChatActivityStateMachine({
        onActivity: (activity) => snapshots.push(activity),
        notificationBatchMs: 40,
        now: deterministicClock(),
      })
      seedReasoningSummary(machine)
      snapshots.length = 0
      consumeSummaryDelta(machine, 3, 'pending')
      expect(snapshots).toEqual([])

      machine[operation]()

      expect(snapshots).toHaveLength(1)
      expect(snapshots[0]).toMatchObject({ status })
      expect(chatActivityPartText(snapshots[0]!.items[0]!.parts[0]!)).toBe('pending')
      vi.advanceTimersByTime(100)
      expect(snapshots).toHaveLength(1)
      if (operation !== 'dispose') machine.dispose()
    } finally {
      vi.useRealTimers()
    }
  })

  it('flushes a bounded gap instead of waiting forever', () => {
    const snapshots: ChatActivity[] = []
    const machine = new ChatActivityStateMachine({
      onActivity: (activity) => snapshots.push(activity),
      reorderBufferSize: 2,
      now: deterministicClock(),
    })
    machine.consume(activityEvent('response.created', {
      sequence_number: 0,
      response: { id: 'resp-gap' },
    }))
    machine.consume(activityEvent('response.output_item.added', {
      sequence_number: 1,
      output_index: 0,
      item: { id: 'reasoning-gap', type: 'reasoning' },
    }))
    machine.consume(activityEvent('response.reasoning_summary_part.added', {
      sequence_number: 2,
      item_id: 'reasoning-gap',
      output_index: 0,
      summary_index: 0,
      part: { type: 'summary_text' },
    }))
    for (const [sequence, delta] of [[4, 'B'], [6, 'C']] as const) {
      machine.consume(activityEvent('response.reasoning_summary_text.delta', {
        sequence_number: sequence,
        item_id: 'reasoning-gap',
        output_index: 0,
        summary_index: 0,
        delta,
      }))
    }

    expect(chatActivityPartText(machine.snapshots().at(-1)!.items[0]!.parts[0]!)).toBe('BC')
    machine.dispose()
  })

  it('timer-flushes a single missing-sequence gap', () => {
    vi.useFakeTimers()
    try {
      const snapshots: ChatActivity[] = []
      const machine = new ChatActivityStateMachine({
        onActivity: (activity) => snapshots.push(activity),
        reorderWaitMs: 20,
        now: deterministicClock(),
      })
      machine.consume(activityEvent('response.created', {
        sequence_number: 0,
        response: { id: 'resp-timer' },
      }))
      machine.consume(activityEvent('response.output_item.added', {
        sequence_number: 1,
        output_index: 0,
        item: { id: 'reasoning-timer', type: 'reasoning' },
      }))
      machine.consume(activityEvent('response.reasoning_summary_text.delta', {
        sequence_number: 3,
        item_id: 'reasoning-timer',
        output_index: 0,
        summary_index: 0,
        delta: 'eventually visible',
      }))

      expect(snapshots).toEqual([])
      expect(machine.snapshots()).toEqual([])
      vi.advanceTimersByTime(20)
      expect(chatActivityPartText(machine.snapshots().at(-1)!.items[0]!.parts[0]!))
        .toBe('eventually visible')
      machine.dispose()
    } finally {
      vi.useRealTimers()
    }
  })

  it('ignores malformed and forbidden content without inventing a summary', () => {
    const snapshots: ChatActivity[] = []
    const machine = new ChatActivityStateMachine({
      onActivity: (activity) => snapshots.push(activity),
      now: deterministicClock(),
    })
    machine.consume(activityEvent('response.created', {
      sequence_number: 0,
      response: { id: 'resp-safe' },
    }))
    machine.consume(activityEvent('response.output_item.added', {
      sequence_number: 1,
      output_index: 0,
      item: {
        id: 'assistant-output',
        type: 'message',
        content: [{ type: 'output_text', text: 'not a summary' }],
        reasoning_content: 'not a summary either',
      },
    }))
    expect(snapshots).toEqual([])

    machine.consume(activityEvent('response.output_item.added', {
      sequence_number: 2,
      output_index: 1,
      item: {
        id: 'reasoning-safe',
        type: 'reasoning',
        encrypted_content: 'ciphertext',
        summary: [
          { type: 'output_text', text: 'forbidden output' },
          { type: 'summary_text', text: 'official summary' },
        ],
      },
    }))
    machine.consume({
      source: 'openai_responses',
      eventType: 'response.reasoning_text.delta',
      payload: { delta: 'private chain of thought' },
    } as unknown as ChatActivityEvent)
    machine.consume(activityEvent('response.reasoning_summary_text.delta', {
      item_id: 'reasoning-safe',
      output_index: 1,
      summary_index: 'invalid',
      delta: 'malformed',
    }))

    const activity = snapshots.at(-1)!
    expect(activity.items).toHaveLength(1)
    expect(activity.items[0]?.parts).toHaveLength(1)
    expect(activity.items[0]?.parts[0]?.text).toBe('official summary')
    expect(JSON.stringify(activity)).not.toContain('ciphertext')
    expect(JSON.stringify(activity)).not.toContain('private chain of thought')
    expect(JSON.stringify(activity)).not.toContain('forbidden output')
  })
})

describe('normalizeChatActivities', () => {
  it('normalizes snake_case snapshots, deduplicates stable keys, and restores partial text', () => {
    const activities = normalizeChatActivities([{
      response_id: 'resp-server',
      status: 'streaming',
      reasoning_mode: 'pro',
      reasoning_effort: 'high',
      started_at: 1_000,
      updated_at: 1_100,
      items: [{
        item_id: 'reasoning-server',
        output_index: 0,
        status: 'streaming',
        started_at: 1_000,
        updated_at: 1_100,
        summary: [
          { type: 'summary_text', summary_index: 0, text: 'older', updated_at: 1_050 },
          { type: 'summary_text', summary_index: 0, text: 'newer', updated_at: 1_100 },
        ],
      }],
    }], { markDisconnected: true })

    expect(activities).toHaveLength(1)
    expect(activities[0]).toMatchObject({
      key: 'resp-server',
      responseId: 'resp-server',
      status: 'disconnected',
      reasoningMode: 'pro',
    })
    expect(activities[0]?.reasoningEffort).toBeUndefined()
    expect(activities[0]?.items[0]?.parts).toEqual([expect.objectContaining({
      key: chatActivityPartKey('resp-server', 'reasoning-server', 0, 0),
      text: 'newer',
      status: 'disconnected',
    })])
  })

  it('groups the durable flat server schema into response, item, and part snapshots', () => {
    const flatPart = (
      itemId: string,
      outputIndex: number,
      summaryIndex: number,
      text: string,
      status: string,
      sequenceEnd: number,
    ) => ({
      response_id: 'resp-flat',
      source: 'openai_responses',
      activity_type: 'reasoning_summary',
      item_id: itemId,
      output_index: outputIndex,
      summary_index: summaryIndex,
      sort_order: sequenceEnd,
      status,
      text,
      sequence_start: 1,
      sequence_end: sequenceEnd,
      reasoning_mode: 'pro',
      reasoning_effort: 'medium',
      started_at: 100,
      updated_at: 100 + sequenceEnd,
      ...(status === 'completed' ? { completed_at: 100 + sequenceEnd } : {}),
      metadata: { last_event: 'ignored canonical metadata' },
    })
    const activities = normalizeChatActivities([
      flatPart('reasoning-b', 1, 0, 'third', 'completed', 6),
      flatPart('reasoning-a', 0, 1, 'second', 'completed', 5),
      flatPart('reasoning-a', 0, 0, 'first draft', 'in_progress', 3),
      flatPart('reasoning-a', 0, 0, 'first', 'completed', 4),
      { ...flatPart('ignored', 2, 0, 'not allowed', 'completed', 7), source: 'other' },
    ])

    expect(activities).toHaveLength(1)
    expect(activities[0]).toMatchObject({
      responseId: 'resp-flat',
      status: 'completed',
      reasoningMode: 'pro',
    })
    expect(activities[0]?.reasoningEffort).toBeUndefined()
    expect(activities[0]?.items.map(({ itemId }) => itemId)).toEqual([
      'reasoning-a',
      'reasoning-b',
    ])
    expect(activities[0]?.items[0]?.parts.map(({ text }) => text)).toEqual([
      'first',
      'second',
    ])
    expect(JSON.stringify(activities)).not.toContain('canonical metadata')
    expect(JSON.stringify(activities)).not.toContain('not allowed')
  })
})

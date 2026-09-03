import { describe, expect, it } from 'vitest'
import { mergeChatHistoryStates } from '@/features/chat/persistence'

interface MergedState {
  version: number
  clearRevision: number
  deletedConversationIds: string[]
  activeConversationId: string | null
  activeConversationSelectionResolved: boolean
  conversations: Array<Record<string, unknown>>
  serverVersion: number
  outbox: Array<Record<string, unknown>>
  legacyImportDecision: 'pending' | 'accepted' | 'declined' | null
  legacyConversationIds: string[]
}

function conversation(id: string, updatedAt: number, userId = 'user-1'): Record<string, unknown> {
  return {
    id,
    userId,
    title: id,
    model: 'gpt-4.1',
    messages: [],
    createdAt: updatedAt,
    updatedAt,
  }
}

function activityRecord(
  text: string,
  status: 'streaming' | 'completed' | 'incomplete' | 'failed' | 'stopped' | 'disconnected',
  updatedAt: number,
  sequence: number,
): Record<string, unknown> {
  const terminal = status !== 'streaming'
  return {
    response_id: 'resp-persisted',
    status,
    reasoning_mode: 'pro',
    reasoning_effort: 'medium',
    started_at: 90,
    updated_at: updatedAt,
    ...(terminal ? { completed_at: updatedAt } : {}),
    last_sequence_number: sequence,
    ignored: 'must-not-persist',
    items: [{
      item_id: 'reasoning-persisted',
      output_index: 0,
      status,
      started_at: 90,
      updated_at: updatedAt,
      ...(terminal ? { completed_at: updatedAt } : {}),
      last_sequence_number: sequence,
      parts: [{
        type: 'summary_text',
        summary_index: 0,
        text,
        status,
        started_at: 90,
        updated_at: updatedAt,
        ...(terminal ? { completed_at: updatedAt } : {}),
        last_sequence_number: sequence,
        reasoning_content: 'private-content-must-not-persist',
      }],
    }],
  }
}

function state(
  conversations: Array<Record<string, unknown>>,
  options: { clearRevision?: number; deletedConversationIds?: string[]; activeId?: string | null } = {},
): Record<string, unknown> {
  const activeConversationId = Object.prototype.hasOwnProperty.call(options, 'activeId')
    ? (options.activeId ?? null)
    : (String(conversations[0]?.id ?? '') || null)
  return {
    version: 1,
    clearRevision: options.clearRevision ?? 0,
    deletedConversationIds: options.deletedConversationIds ?? [],
    activeConversationId,
    conversations,
  }
}

function version2State(
  conversations: Array<Record<string, unknown>>,
  options: {
    activeId?: string | null
    serverVersion?: number
    outbox?: Array<Record<string, unknown>>
    legacyImportDecision?: 'pending' | 'accepted' | 'declined' | null
    legacyConversationIds?: string[]
    activeConversationSelectionResolved?: boolean
  } = {},
): Record<string, unknown> {
  const activeConversationId = Object.prototype.hasOwnProperty.call(options, 'activeId')
    ? (options.activeId ?? null)
    : (String(conversations[0]?.id ?? '') || null)
  return {
    version: 2,
    clearRevision: 0,
    deletedConversationIds: [],
    activeConversationId,
    ...(options.activeConversationSelectionResolved !== undefined
      ? { activeConversationSelectionResolved: options.activeConversationSelectionResolved }
      : {}),
    conversations,
    serverVersion: options.serverVersion ?? 0,
    outbox: options.outbox ?? [],
    legacyImportDecision: options.legacyImportDecision ?? null,
    legacyConversationIds: options.legacyConversationIds ?? [],
  }
}

function mergedState(value: unknown): MergedState {
  return value as MergedState
}

describe('mergeChatHistoryStates', () => {
  it('preserves an explicit new-chat selection across background history writes', () => {
    const existing = conversation('existing', 10)
    const removed = conversation('removed', 5)
    let stored = mergeChatHistoryStates(
      null,
      version2State([existing, removed], {
        activeId: null,
      }),
      'user-1',
      {
        upsertConversationIds: ['existing', 'removed'],
        createdConversations: [
          { id: 'existing', operationAt: 10 },
          { id: 'removed', operationAt: 5 },
        ],
        activeConversationChanged: true,
      },
    )

    stored = mergeChatHistoryStates(
      stored,
      version2State([conversation('existing', 20), removed], { activeId: 'existing' }),
      'user-1',
      { upsertConversationIds: ['existing'] },
    )

    stored = mergeChatHistoryStates(
      stored,
      version2State([conversation('existing', 20)], { activeId: 'existing' }),
      'user-1',
      { deletedConversationIds: ['removed'] },
    )

    stored = mergeChatHistoryStates(
      stored,
      version2State([conversation('existing', 20)], { activeId: 'existing' }),
      'user-1',
      { initializeActiveConversation: true },
    )

    stored = mergeChatHistoryStates(
      stored,
      version2State([conversation('existing', 20)], { activeId: 'existing' }),
      'user-1',
      { replaceActiveConversationIfId: 'removed' },
    )

    expect(mergedState(stored)).toMatchObject({
      activeConversationId: null,
      activeConversationSelectionResolved: true,
    })
  })

  it('initializes a server-derived selection only while the stored selection is unresolved', () => {
    const existing = conversation('existing', 10)
    const unresolved = version2State([existing], {
      activeId: null,
      activeConversationSelectionResolved: false,
    })
    const selected = version2State([existing], {
      activeId: 'existing',
      activeConversationSelectionResolved: true,
    })

    const merged = mergedState(mergeChatHistoryStates(
      unresolved,
      selected,
      'user-1',
      { initializeActiveConversation: true },
    ))

    expect(merged).toMatchObject({
      activeConversationId: 'existing',
      activeConversationSelectionResolved: true,
    })
  })

  it('replaces a server-deleted active conversation only when the stored ID still matches', () => {
    const removed = conversation('removed', 20)
    const kept = conversation('kept', 10)
    const merged = mergedState(mergeChatHistoryStates(
      version2State([removed, kept], { activeId: 'removed' }),
      version2State([kept], { activeId: 'kept' }),
      'user-1',
      {
        deletedConversationIds: ['removed'],
        replaceActiveConversationIfId: 'removed',
      },
    ))

    expect(merged).toMatchObject({
      activeConversationId: 'kept',
      activeConversationSelectionResolved: true,
    })
  })

  it('preserves conversations created concurrently in different tabs', () => {
    const tabA = conversation('tab-a', 10)
    const tabB = conversation('tab-b', 20)
    let stored = mergeChatHistoryStates(null, state([tabA]), 'user-1', {
      upsertConversationIds: ['tab-a'],
      createdConversations: [{ id: 'tab-a', operationAt: 10 }],
      activeConversationChanged: true,
    })

    stored = mergeChatHistoryStates(stored, state([tabB]), 'user-1', {
      upsertConversationIds: ['tab-b'],
      createdConversations: [{ id: 'tab-b', operationAt: 20 }],
      activeConversationChanged: true,
    })

    const merged = mergedState(stored)
    expect(merged.conversations.map(({ id }) => id)).toEqual(['tab-b', 'tab-a'])
    expect(merged.activeConversationId).toBe('tab-b')
  })

  it('prevents stale updates from restoring conversations after clear', () => {
    const original = conversation('old', 10)
    const stored = mergeChatHistoryStates(null, state([original]), 'user-1', {
      upsertConversationIds: ['old'],
      createdConversations: [{ id: 'old', operationAt: 10 }],
      activeConversationChanged: true,
    })
    const cleared = mergeChatHistoryStates(stored, state([], { clearRevision: 20 }), 'user-1', {
      clear: true,
      activeConversationChanged: true,
    })
    const staleUpdate = conversation('old', 30)
    const afterStaleUpdate = mergedState(mergeChatHistoryStates(
      cleared,
      state([staleUpdate]),
      'user-1',
      { upsertConversationIds: ['old'] },
    ))

    expect(afterStaleUpdate.clearRevision).toBeGreaterThan(0)
    expect(afterStaleUpdate.conversations).toEqual([])
    expect(afterStaleUpdate.activeConversationId).toBeNull()
  })

  it('allows a stale tab to create a new UUID after another tab clears', () => {
    const original = conversation('old', 10)
    const stored = mergeChatHistoryStates(null, state([original]), 'user-1', {
      upsertConversationIds: ['old'],
      createdConversations: [{ id: 'old', operationAt: 10 }],
    })
    const cleared = mergeChatHistoryStates(
      stored,
      state([], { clearRevision: 30 }),
      'user-1',
      { clear: true },
    )
    const replacement = conversation('new-after-clear', 40)
    const recreated = mergedState(mergeChatHistoryStates(
      cleared,
      state([original, replacement], { activeId: 'new-after-clear' }),
      'user-1',
      {
        upsertConversationIds: ['new-after-clear'],
        createdConversations: [{ id: 'new-after-clear', operationAt: 40 }],
        activeConversationChanged: true,
      },
    ))

    expect(recreated.conversations.map(({ id }) => id)).toEqual(['new-after-clear'])
    expect(recreated.activeConversationId).toBe('new-after-clear')
  })

  it('orders delayed create and clear operations by action time, not transaction arrival', () => {
    const cleared = mergeChatHistoryStates(
      null,
      state([], { clearRevision: 100 }),
      'user-1',
      { clear: true },
    )
    const beforeClear = conversation('before-clear', 90)
    const afterDelayedCreate = mergedState(mergeChatHistoryStates(
      cleared,
      state([beforeClear]),
      'user-1',
      {
        upsertConversationIds: ['before-clear'],
        createdConversations: [{ id: 'before-clear', operationAt: 90 }],
      },
    ))
    expect(afterDelayedCreate.conversations).toEqual([])

    const afterClear = conversation('after-clear', 110)
    const createdFirst = mergeChatHistoryStates(
      null,
      state([afterClear]),
      'user-1',
      {
        upsertConversationIds: ['after-clear'],
        createdConversations: [{ id: 'after-clear', operationAt: 110 }],
      },
    )
    const afterDelayedClear = mergedState(mergeChatHistoryStates(
      createdFirst,
      state([], { clearRevision: 100 }),
      'user-1',
      { clear: true },
    ))

    expect(afterDelayedClear.clearRevision).toBe(100)
    expect(afterDelayedClear.conversations.map(({ id }) => id)).toEqual(['after-clear'])
    expect(afterDelayedClear.activeConversationId).toBe('after-clear')
  })

  it('does not let a stale tab resurrect a deleted conversation', () => {
    const deleted = conversation('deleted', 20)
    const kept = conversation('kept', 10)
    const stored = mergeChatHistoryStates(null, state([deleted, kept]), 'user-1', {
      upsertConversationIds: ['deleted', 'kept'],
      createdConversations: [
        { id: 'deleted', operationAt: 20 },
        { id: 'kept', operationAt: 10 },
      ],
    })
    const afterDelete = mergeChatHistoryStates(
      stored,
      state([kept], { activeId: 'kept' }),
      'user-1',
      { deletedConversationIds: ['deleted'], activeConversationChanged: true },
    )
    const staleUpdate = mergedState(mergeChatHistoryStates(
      afterDelete,
      state([conversation('deleted', 50), kept]),
      'user-1',
      { upsertConversationIds: ['deleted'] },
    ))

    expect(staleUpdate.conversations.map(({ id }) => id)).toEqual(['kept'])
    expect(staleUpdate.deletedConversationIds).toEqual([])
  })

  it('canonicalizes stored records before enforcing the top-50 limit', () => {
    const valid = Array.from({ length: 49 }, (_, index) => conversation(`valid-${index}`, index + 1))
    const withSecrets = {
      ...conversation('canonical', Date.now()),
      apiKey: 'must-not-persist',
      messages: [
        {
          id: 'message-1',
          role: 'user',
          content: 'hello',
          createdAt: Date.now(),
          status: 'complete',
          authorization: 'Bearer secret',
        },
      ],
    }
    const farFuture = conversation('future', Date.now() + 10 * 60 * 1000)
    const foreign = conversation('foreign', Date.now(), 'other-user')
    const malformed = { ...conversation('malformed', Date.now()), model: '', extra: 'drop-me' }
    const existing = state([...valid, withSecrets, farFuture, foreign, malformed])
    const merged = mergedState(mergeChatHistoryStates(
      existing,
      state([], { activeId: null }),
      'user-1',
      {},
    ))

    expect(merged.conversations).toHaveLength(50)
    expect(merged.conversations.map(({ id }) => id)).toContain('valid-0')
    expect(merged.conversations.map(({ id }) => id)).not.toEqual(
      expect.arrayContaining(['future', 'foreign', 'malformed']),
    )
    const canonical = merged.conversations.find(({ id }) => id === 'canonical')!
    expect(Object.keys(canonical).sort()).toEqual([
      'createdAt',
      'id',
      'messages',
      'model',
      'title',
      'updatedAt',
      'userId',
    ])
    expect(Object.keys((canonical.messages as Array<Record<string, unknown>>)[0]).sort()).toEqual([
      'content',
      'createdAt',
      'id',
      'role',
      'status',
    ])
    expect(JSON.stringify(merged)).not.toContain('must-not-persist')
    expect(JSON.stringify(merged)).not.toContain('Bearer secret')
  })

  it('canonicalizes chat receipt fields without admitting unknown billing data', () => {
    const withReceipt = {
      ...conversation('receipt-conversation', 20),
      messages: [{
        id: 'assistant-1',
        role: 'assistant',
        content: 'Answer',
        createdAt: 20,
        status: 'complete',
        attemptId: 'attempt-1',
        receiptId: 'receipt-1',
        settlementStatus: 'charged',
        usageLogId: 9,
        requestedModel: 'gpt-5.5',
        actualModel: 'gpt-5.5-2026-07-01',
        inputTokens: 100,
        outputTokens: 20,
        cacheCreationTokens: 5,
        cacheReadTokens: 50,
        totalTokens: 125,
        grossCost: 0.003,
        chargedAmount: 0.002,
        billingType: 0,
        balanceBefore: 3,
        balanceAfter: 2.998,
        receiptCreatedAt: '2026-07-25T01:02:03Z',
        excludedFromContext: true,
        supersededByMessageId: 'assistant-2',
        unexpectedCostAdjustment: 99,
        secret: 'drop-me',
      }],
    }
    const merged = mergedState(mergeChatHistoryStates(
      state([withReceipt]),
      state([]),
      'user-1',
      {},
    ))
    const message = (
      merged.conversations[0].messages as Array<Record<string, unknown>>
    )[0]

    expect(message).toMatchObject({
      attemptId: 'attempt-1',
      receiptId: 'receipt-1',
      settlementStatus: 'charged',
      usageLogId: 9,
      requestedModel: 'gpt-5.5',
      actualModel: 'gpt-5.5-2026-07-01',
      inputTokens: 100,
      outputTokens: 20,
      cacheCreationTokens: 5,
      cacheReadTokens: 50,
      totalTokens: 125,
      grossCost: 0.003,
      chargedAmount: 0.002,
      billingType: 0,
      balanceBefore: 3,
      balanceAfter: 2.998,
      receiptCreatedAt: '2026-07-25T01:02:03Z',
      excludedFromContext: true,
      supersededByMessageId: 'assistant-2',
    })
    expect(message.unexpectedCostAdjustment).toBeUndefined()
    expect(message.secret).toBeUndefined()
  })

  it('canonicalizes and deduplicates persisted reasoning Activities', () => {
    const candidate = {
      ...conversation('activity-canonical', 100),
      messages: [{
        id: 'assistant-activity',
        role: 'assistant',
        content: 'answer',
        createdAt: 80,
        status: 'complete',
        activities: [
          activityRecord('older summary', 'streaming', 95, 2),
          activityRecord('newer summary', 'streaming', 99, 3),
        ],
      }],
    }
    const merged = mergedState(mergeChatHistoryStates(
      null,
      state([candidate]),
      'user-1',
      {
        upsertConversationIds: ['activity-canonical'],
        createdConversations: [{ id: 'activity-canonical', operationAt: 100 }],
      },
    ))
    const message = (
      merged.conversations[0]?.messages as Array<Record<string, unknown>>
    )[0]!
    const activities = message.activities as Array<Record<string, unknown>>
    const items = activities[0]?.items as Array<Record<string, unknown>>
    const parts = items[0]?.parts as Array<Record<string, unknown>>

    expect(activities).toHaveLength(1)
    expect(activities[0]).toMatchObject({
      key: 'resp-persisted',
      responseId: 'resp-persisted',
      reasoningMode: 'pro',
    })
    expect(parts).toHaveLength(1)
    expect(parts[0]).toMatchObject({ text: 'newer summary', lastSequenceNumber: 3 })
    expect(JSON.stringify(activities)).not.toContain('must-not-persist')
    expect(JSON.stringify(activities)).not.toContain('private-content')
  })

  it('preserves live summary chunks without materializing the accumulated text', () => {
    const liveActivity = activityRecord('', 'streaming', 99, 4)
    const items = liveActivity.items as Array<Record<string, unknown>>
    const parts = items[0]?.parts as Array<Record<string, unknown>>
    parts[0]!.streaming_text_chunks = ['first ', 'second', ' third']

    const merged = mergedState(mergeChatHistoryStates(
      null,
      state([{
        ...conversation('activity-stream-chunks', 100),
        messages: [{
          id: 'assistant-activity',
          role: 'assistant',
          content: '',
          createdAt: 80,
          status: 'streaming',
          activities: [liveActivity],
        }],
      }]),
      'user-1',
      {
        upsertConversationIds: ['activity-stream-chunks'],
        createdConversations: [{ id: 'activity-stream-chunks', operationAt: 100 }],
      },
    ))
    const message = (
      merged.conversations[0]?.messages as Array<Record<string, unknown>>
    )[0]!
    const persistedActivities = message.activities as Array<Record<string, unknown>>
    const persistedItems = persistedActivities[0]?.items as Array<Record<string, unknown>>
    const persistedParts = persistedItems[0]?.parts as Array<Record<string, unknown>>

    expect(persistedParts[0]).toMatchObject({
      text: '',
      streamingTextChunks: ['first ', 'second', ' third'],
      status: 'streaming',
    })
  })

  it('keeps newer local Activity progress when a stale peer writes a terminal snapshot', () => {
    const withActivity = (
      updatedAt: number,
      activity: Record<string, unknown>,
    ): Record<string, unknown> => ({
      ...conversation('activity-merge', updatedAt),
      messages: [{
        id: 'assistant-activity',
        role: 'assistant',
        content: 'answer',
        createdAt: 80,
        status: 'complete',
        activities: [activity],
      }],
    })
    let stored = mergeChatHistoryStates(
      null,
      state([withActivity(100, activityRecord('new local stream', 'streaming', 500, 10))]),
      'user-1',
      {
        upsertConversationIds: ['activity-merge'],
        createdConversations: [{ id: 'activity-merge', operationAt: 100 }],
      },
    )
    stored = mergeChatHistoryStates(
      stored,
      state([withActivity(101, activityRecord('old peer stream', 'streaming', 100, 2))]),
      'user-1',
      { upsertConversationIds: ['activity-merge'] },
    )
    let message = (
      mergedState(stored).conversations[0]?.messages as Array<Record<string, unknown>>
    )[0]!
    let activities = message.activities as Array<Record<string, unknown>>
    let items = activities[0]?.items as Array<Record<string, unknown>>
    let parts = items[0]?.parts as Array<Record<string, unknown>>
    expect(parts[0]?.text).toBe('new local stream')
    expect(activities[0]?.status).toBe('streaming')

    stored = mergeChatHistoryStates(
      stored,
      state([withActivity(102, activityRecord('server final', 'completed', 110, 3))]),
      'user-1',
      { upsertConversationIds: ['activity-merge'] },
    )
    message = (
      mergedState(stored).conversations[0]?.messages as Array<Record<string, unknown>>
    )[0]!
    activities = message.activities as Array<Record<string, unknown>>
    items = activities[0]?.items as Array<Record<string, unknown>>
    parts = items[0]?.parts as Array<Record<string, unknown>>
    expect(activities[0]?.status).toBe('streaming')
    expect(parts[0]?.text).toBe('new local stream')
  })

  it.each(['stopped', 'disconnected'] as const)(
    'does not let a stale peer completed snapshot overwrite newer local %s state',
    (newerStatus) => {
      const withActivity = (
        updatedAt: number,
        activity: Record<string, unknown>,
      ): Record<string, unknown> => ({
        ...conversation('activity-terminal-race', updatedAt),
        messages: [{
          id: 'assistant-activity',
          role: 'assistant',
          content: 'partial answer',
          createdAt: 80,
          status: 'stopped',
          activities: [activity],
        }],
      })
      let stored = mergeChatHistoryStates(
        null,
        state([withActivity(
          500,
          activityRecord('newer partial summary', newerStatus, 500, 10),
        )]),
        'user-1',
        {
          upsertConversationIds: ['activity-terminal-race'],
          createdConversations: [{ id: 'activity-terminal-race', operationAt: 500 }],
        },
      )

      stored = mergeChatHistoryStates(
        stored,
        state([withActivity(
          510,
          activityRecord('stale completed summary', 'completed', 110, 3),
        )]),
        'user-1',
        { upsertConversationIds: ['activity-terminal-race'] },
      )

      const message = (
        mergedState(stored).conversations[0]?.messages as Array<Record<string, unknown>>
      )[0]!
      const activities = message.activities as Array<Record<string, unknown>>
      const items = activities[0]?.items as Array<Record<string, unknown>>
      const parts = items[0]?.parts as Array<Record<string, unknown>>
      expect(activities[0]?.status).toBe(newerStatus)
      expect(parts[0]).toMatchObject({
        status: newerStatus,
        text: 'newer partial summary',
        lastSequenceNumber: 10,
      })
    },
  )

  it('normalizes clock-skewed records without letting them crowd out valid history', () => {
    const now = Date.now()
    const valid = conversation('valid', now - 1000)
    const clockSkewed = conversation('clock-skewed', now + 10 * 60 * 1000)
    const merged = mergedState(mergeChatHistoryStates(
      state([clockSkewed, valid]),
      state([]),
      'user-1',
      {},
    ))

    expect(merged.conversations.map(({ id }) => id)).toEqual(['valid', 'clock-skewed'])
    const normalized = merged.conversations[1]
    expect(Number(normalized.createdAt)).toBeLessThanOrEqual(Date.now())
    expect(Number(normalized.updatedAt)).toBeLessThanOrEqual(Date.now())
  })

  it('does not let hydrate sanitation overwrite a newer peer write', () => {
    const staleStreaming = {
      ...conversation('shared', 20),
      messages: [{
        id: 'assistant',
        role: 'assistant',
        content: 'old chunk',
        createdAt: 15,
        status: 'streaming',
      }],
    }
    const sanitizedStale = {
      ...staleStreaming,
      messages: [{
        id: 'assistant',
        role: 'assistant',
        content: 'old chunk',
        createdAt: 15,
        status: 'stopped',
        finishReason: 'interrupted',
      }],
    }
    const peerLatest = {
      ...conversation('shared', 21),
      messages: [{
        id: 'assistant',
        role: 'assistant',
        content: 'old chunk + new chunk',
        createdAt: 15,
        status: 'complete',
        finishReason: 'stop',
      }],
    }
    const merged = mergedState(mergeChatHistoryStates(
      state([peerLatest]),
      state([sanitizedStale]),
      'user-1',
      { sanitizeConversations: [{ id: 'shared', expectedUpdatedAt: 20 }] },
    ))
    const message = (merged.conversations[0].messages as Array<Record<string, unknown>>)[0]

    expect(message.content).toBe('old chunk + new chunk')
    expect(message.status).toBe('complete')
  })

  it('loads old version-1 records without mutation metadata at revision zero', () => {
    const legacy = {
      version: 1,
      activeConversationId: 'legacy',
      conversations: [conversation('legacy', 10)],
    }
    const merged = mergedState(mergeChatHistoryStates(
      legacy,
      state([]),
      'user-1',
      {},
    ))

    expect(merged.clearRevision).toBe(0)
    expect(merged.deletedConversationIds).toEqual([])
    expect(merged.conversations.map(({ id }) => id)).toEqual(['legacy'])
    expect(merged.activeConversationId).toBe('legacy')
  })

  it('migrates a version-1 bucket to version 2 in place with an explicit import decision', () => {
    const legacyConversation = conversation('legacy', 10)
    const legacy = {
      version: 1,
      activeConversationId: 'legacy',
      conversations: [legacyConversation],
    }
    const migrated = mergedState(mergeChatHistoryStates(
      legacy,
      version2State([legacyConversation], {
        activeId: 'legacy',
        legacyImportDecision: 'pending',
        legacyConversationIds: ['legacy'],
      }),
      'user-1',
      {
        sanitizeConversations: [{ id: 'legacy', expectedUpdatedAt: 10 }],
        syncStateChanged: true,
      },
    ))

    expect(migrated).toMatchObject({
      version: 2,
      activeConversationId: 'legacy',
      legacyImportDecision: 'pending',
      legacyConversationIds: ['legacy'],
    })
    expect(migrated.conversations.map(({ id }) => id)).toEqual(['legacy'])
  })

  it('preserves the import decision and legacy IDs during an unrelated tab write', () => {
    const legacy = conversation('legacy', 10)
    const existing = version2State([legacy], {
      legacyImportDecision: 'declined',
      legacyConversationIds: ['legacy'],
    })
    const newConversation = conversation('new-conversation', 20)
    const merged = mergedState(mergeChatHistoryStates(
      existing,
      version2State([newConversation], {
        activeId: 'new-conversation',
      }),
      'user-1',
      {
        upsertConversationIds: ['new-conversation'],
        createdConversations: [{ id: 'new-conversation', operationAt: 20 }],
        activeConversationChanged: true,
      },
    ))

    expect(merged.legacyImportDecision).toBe('declined')
    expect(merged.legacyConversationIds).toEqual(['legacy'])
    expect(merged.conversations.map(({ id }) => id)).toEqual([
      'new-conversation',
      'legacy',
    ])
  })

  it('merges concurrent outbox writes and removes only acknowledged mutation IDs', () => {
    const firstConversation = conversation('first', 10)
    const firstMutation = {
      mutationId: 'create:first',
      type: 'create',
      conversationId: 'first',
      createdAt: 10,
      title: 'first',
      model: 'gpt-5',
    }
    let stored = mergeChatHistoryStates(
      null,
      version2State([firstConversation], { outbox: [firstMutation] }),
      'user-1',
      {
        upsertConversationIds: ['first'],
        createdConversations: [{ id: 'first', operationAt: 10 }],
        enqueueOutboxMutationIds: ['create:first'],
      },
    )

    const secondConversation = conversation('second', 20)
    const secondMutation = {
      mutationId: 'create:second',
      type: 'create',
      conversationId: 'second',
      createdAt: 20,
      title: 'second',
      model: 'gpt-5',
    }
    stored = mergeChatHistoryStates(
      stored,
      version2State([secondConversation], { outbox: [secondMutation] }),
      'user-1',
      {
        upsertConversationIds: ['second'],
        createdConversations: [{ id: 'second', operationAt: 20 }],
        enqueueOutboxMutationIds: ['create:second'],
      },
    )
    const acknowledged = mergedState(mergeChatHistoryStates(
      stored,
      version2State([firstConversation, secondConversation], {
        outbox: [secondMutation],
      }),
      'user-1',
      { acknowledgedOutboxMutationIds: ['create:first'] },
    ))

    expect(acknowledged.outbox).toEqual([expect.objectContaining({
      mutationId: 'create:second',
    })])
  })

  it('keeps the newest payload when tabs enqueue the same mutation ID', () => {
    const sharedConversation = conversation('shared', 10)
    const oldMutation = {
      mutationId: 'patch:shared',
      type: 'patch',
      conversationId: 'shared',
      createdAt: 10,
      revision: 1,
      title: '旧标题',
    }
    const newerMutation = {
      ...oldMutation,
      createdAt: 20,
      title: '新标题',
    }
    const lateOlderMutation = {
      ...oldMutation,
      createdAt: 15,
      title: '迟到旧标题',
    }
    let stored = mergeChatHistoryStates(
      version2State([sharedConversation], { outbox: [oldMutation] }),
      version2State([sharedConversation], { outbox: [newerMutation] }),
      'user-1',
      { enqueueOutboxMutationIds: ['patch:shared'] },
    )
    stored = mergeChatHistoryStates(
      stored,
      version2State([sharedConversation], { outbox: [lateOlderMutation] }),
      'user-1',
      { enqueueOutboxMutationIds: ['patch:shared'] },
    )

    expect(mergedState(stored).outbox).toEqual([expect.objectContaining({
      mutationId: 'patch:shared',
      createdAt: 20,
      title: '新标题',
    })])
  })
})

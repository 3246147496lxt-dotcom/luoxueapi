package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func observeOpenAIResponsesActivityEnvelope(
	t *testing.T,
	observer *deliveredChatStreamObserver,
	eventType string,
	payload string,
) {
	t.Helper()
	frame, err := json.Marshal(struct {
		Source    string          `json:"source"`
		EventType string          `json:"eventType"`
		Payload   json.RawMessage `json:"payload"`
	}{
		Source:    "openai_responses",
		EventType: eventType,
		Payload:   json.RawMessage(payload),
	})
	require.NoError(t, err)
	require.NoError(t, observer.Observe(append(append([]byte("data: "), frame...), []byte("\n\n")...)))
}

func waitForDeliveredChatSnapshotCheckpoint(
	t *testing.T,
	observer *deliveredChatStreamObserver,
) {
	t.Helper()
	require.NoError(t, observer.waitForSnapshotCheckpoints(observer.snapshotCheckpointTarget()))
}

func TestDeliveredChatStreamObserverCapturesOnlyCompleteWrittenEvents(t *testing.T) {
	observer := newDeliveredChatStreamObserver()
	require.NoError(t, observer.Observe([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"hel")))
	require.NoError(t, observer.Observe([]byte("lo\"},\"finish_reason\":\"stop\"}]}\n\n")))
	require.NoError(t, observer.Observe([]byte("data: [DONE]\n\n")))

	snapshot := observer.Snapshot()

	require.Equal(t, "hello", snapshot.Content)
	require.Equal(t, "stop", snapshot.FinishReason)
	require.True(t, snapshot.Done)
}

func TestDeliveredChatStreamObserverDoesNotCompleteOnUnclosedDoneFrame(t *testing.T) {
	observer := newDeliveredChatStreamObserver()
	require.NoError(t, observer.Observe([]byte("data: [DONE]")))

	require.False(t, observer.Snapshot().Done)
}

func TestDeliveredChatStreamObserverCapturesStreamError(t *testing.T) {
	observer := newDeliveredChatStreamObserver()
	require.NoError(t, observer.Observe([]byte(
		"data: {\"error\":{\"code\":\"upstream_error\",\"message\":\"failed\"}}\n\n",
	)))

	snapshot := observer.Snapshot()

	require.Equal(t, "upstream_error", snapshot.ErrorCode)
	require.Equal(t, "failed", snapshot.ErrorMessage)
	require.False(t, snapshot.Done)
}

func TestDeliveredChatStreamObserverErrorTakesPrecedenceOverDone(t *testing.T) {
	observer := newDeliveredChatStreamObserver()
	require.NoError(t, observer.Observe([]byte(
		"data: {\"error\":{\"code\":\"blocked\",\"message\":\"denied\"}}\n\n"+
			"data: [DONE]\n\n",
	)))

	snapshot := observer.Snapshot()

	require.Equal(t, "blocked", snapshot.ErrorCode)
	require.False(t, snapshot.Done)
}

func TestDeliveredChatStreamObserverCheckpointsMonotonicPrefixes(t *testing.T) {
	type checkpoint struct {
		sequence int64
		content  string
	}
	checkpoints := make([]checkpoint, 0, 2)
	observer := newDeliveredChatStreamObserver(func(sequence int64, content string) error {
		checkpoints = append(checkpoints, checkpoint{sequence: sequence, content: content})
		return nil
	})

	require.NoError(t, observer.Observe([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"a\"}}]}\n\n")))
	require.NoError(t, observer.Observe([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"b\"}}]}\n\n")))
	finalSequence, err := observer.ForceCheckpoint()

	require.NoError(t, err)
	require.Equal(t, int64(3), finalSequence)
	require.Equal(t, []checkpoint{
		{sequence: 1, content: "a"},
		{sequence: 2, content: "ab"},
		{sequence: 3, content: "ab"},
	}, checkpoints)
}

func TestDeliveredChatStreamObserverFreezeIgnoresDetachedDrain(t *testing.T) {
	observer := newDeliveredChatStreamObserver()
	require.NoError(t, observer.Observe([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"kept\"}}]}\n\n")))
	observer.Freeze()
	require.NoError(t, observer.Observe([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"dropped\"}}]}\n\n")))

	require.Equal(t, "kept", observer.Snapshot().Content)
}

func TestDeliveredChatStreamObserverCancellationGateIgnoresDetachedDrain(t *testing.T) {
	tests := []struct {
		name  string
		write func(*deliveredChatResponseWriter, string) (int, error)
	}{
		{
			name: "Write",
			write: func(writer *deliveredChatResponseWriter, payload string) (int, error) {
				return writer.Write([]byte(payload))
			},
		},
		{
			name: "WriteString",
			write: func(writer *deliveredChatResponseWriter, payload string) (int, error) {
				return writer.WriteString(payload)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			checkpoints := 0
			observer := newDeliveredChatStreamObserver(func(_ int64, _ string) error {
				checkpoints++
				return nil
			})
			observer.bindCancellation(ctx.Done())
			recorder := httptest.NewRecorder()
			ginContext, _ := gin.CreateTestContext(recorder)
			writer := &deliveredChatResponseWriter{
				ResponseWriter: ginContext.Writer,
				observer:       observer,
			}
			_, err := test.write(
				writer,
				"data: {\"choices\":[{\"delta\":{\"content\":\"kept\"}}]}\n\n",
			)
			require.NoError(t, err)

			cancel()
			_, err = test.write(
				writer,
				"data: {\"choices\":[{\"delta\":{\"content\":\"dropped\"}}]}\n\n"+
					"data: [DONE]\n\n",
			)
			require.NoError(t, err)

			snapshot := observer.Snapshot()
			require.Contains(t, recorder.Body.String(), "dropped")
			require.Equal(t, "kept", snapshot.Content)
			require.False(t, snapshot.Done)
			require.Equal(t, 1, checkpoints)
		})
	}
}

func TestDeliveredChatStreamObserverAggregatesMultipleReasoningItemsAndParts(t *testing.T) {
	observer := newDeliveredChatStreamObserver()
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.created", `{
		"sequence_number":0,
		"response":{"id":"resp_activity_1","reasoning":{"mode":"pro","effort":"medium"}}
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.output_item.added", `{
		"sequence_number":1,"output_index":0,
		"item":{"id":"rs_1","type":"reasoning","status":"in_progress"}
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.delta", `{
		"sequence_number":3,"item_id":"rs_1","output_index":0,"summary_index":0,"delta":"first"
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_part.added", `{
		"sequence_number":2,"item_id":"rs_1","output_index":0,"summary_index":0,
		"part":{"type":"summary_text","text":""}
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_part.added", `{
		"sequence_number":4,"item_id":"rs_1","output_index":0,"summary_index":1,
		"part":{"type":"summary_text"}
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.delta", `{
		"sequence_number":5,"item_id":"rs_1","output_index":0,"summary_index":1,"delta":"second"
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.output_item.added", `{
		"sequence_number":6,"output_index":1,
		"item":{"id":"rs_2","type":"reasoning","status":"in_progress"}
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.delta", `{
		"sequence_number":7,"item_id":"rs_2","output_index":1,"summary_index":0,"delta":"third"
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.completed", `{
		"sequence_number":8,
		"response":{"id":"resp_activity_1","status":"completed","output":[
			{"id":"rs_1","type":"reasoning","summary":[
				{"type":"summary_text","text":"first"},
				{"type":"summary_text","text":"second"}
			]},
			{"id":"rs_2","type":"reasoning","summary":[
				{"type":"summary_text","text":"third"}
			]}
		]}
	}`)

	snapshot := observer.Snapshot()
	require.Equal(t, "resp_activity_1", snapshot.ResponseID)
	require.Equal(t, "pro", snapshot.ReasoningMode)
	require.Empty(t, snapshot.ReasoningEffort)
	require.Len(t, snapshot.Activities, 3)
	require.Equal(t, []string{"first", "second", "third"}, []string{
		snapshot.Activities[0].Text,
		snapshot.Activities[1].Text,
		snapshot.Activities[2].Text,
	})
	require.Equal(t, []int{0, 1, 0}, []int{
		snapshot.Activities[0].SummaryIndex,
		snapshot.Activities[1].SummaryIndex,
		snapshot.Activities[2].SummaryIndex,
	})
	for _, activity := range snapshot.Activities {
		require.Equal(t, "completed", activity.Status)
		require.Equal(t, "resp_activity_1", activity.ResponseID)
		require.Equal(t, "pro", activity.ReasoningMode)
		require.Empty(t, activity.ReasoningEffort)
		require.NotNil(t, activity.CompletedAt)
	}
}

func TestDeliveredChatStreamObserverDeduplicatesAndReordersResponsesSequence(t *testing.T) {
	observer := newDeliveredChatStreamObserver()
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.created", `{"sequence_number":0,"response":{"id":"resp_order"}}`)
	// Deliver 3 before 1 and 2. The observer holds the finite gap and applies the
	// events in sequence order once the missing prefix arrives.
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.delta", `{
		"sequence_number":3,"item_id":"rs_order","output_index":0,"summary_index":0,"delta":"B"
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_part.added", `{
		"sequence_number":2,"item_id":"rs_order","output_index":0,"summary_index":0,"part":{"type":"summary_text"}
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.output_item.added", `{
		"sequence_number":1,"output_index":0,"item":{"id":"rs_order","type":"reasoning"}
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.delta", `{
		"sequence_number":4,"item_id":"rs_order","output_index":0,"summary_index":0,"delta":"C"
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.delta", `{
		"sequence_number":4,"item_id":"rs_order","output_index":0,"summary_index":0,"delta":"duplicate"
	}`)

	activities := observer.Snapshot().Activities
	require.Len(t, activities, 1)
	require.Equal(t, "BC", activities[0].Text)
	require.Equal(t, int64(1), activities[0].SequenceStart)
	require.Equal(t, int64(4), activities[0].SequenceEnd)
}

func TestDeliveredChatStreamObserverTextDoneIsAuthoritative(t *testing.T) {
	observer := newDeliveredChatStreamObserver()
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.created", `{"sequence_number":0,"response":{"id":"resp_done"}}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.delta", `{
		"sequence_number":1,"item_id":"rs_done","output_index":0,"summary_index":0,"delta":"wrng"
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.done", `{
		"sequence_number":2,"item_id":"rs_done","output_index":0,"summary_index":0,"text":"correct"
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_part.done", `{
		"sequence_number":3,"item_id":"rs_done","output_index":0,"summary_index":0,
		"part":{"type":"summary_text","text":"part fallback"}
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.completed", `{
		"sequence_number":4,"response":{"id":"resp_done","output":[
			{"id":"rs_done","type":"reasoning","summary":[{"type":"summary_text","text":"terminal fallback"}]}
		]}
	}`)

	activities := observer.Snapshot().Activities
	require.Len(t, activities, 1)
	require.Equal(t, "correct", activities[0].Text)
	require.Equal(t, "completed", activities[0].Status)
}

func TestDeliveredChatStreamObserverDoesNotPersistEmptyReasoningItems(t *testing.T) {
	observer := newDeliveredChatStreamObserver()
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.created", `{
		"sequence_number":0,"response":{"id":"resp_empty"}
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.output_item.added", `{
		"sequence_number":1,"output_index":0,
		"item":{"id":"rs_empty","type":"reasoning","summary":[]}
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.output_item.done", `{
		"sequence_number":2,"output_index":0,
		"item":{"id":"rs_empty","type":"reasoning","status":"completed","summary":[]}
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.completed", `{
		"sequence_number":3,"response":{"id":"resp_empty","status":"completed","output":[]}
	}`)

	require.Empty(t, observer.Snapshot().Activities)
}

func TestDeliveredChatStreamObserverPreservesIncompletePartDone(t *testing.T) {
	observer := newDeliveredChatStreamObserver()
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.created", `{
		"sequence_number":0,"response":{"id":"resp_part_incomplete"}
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.delta", `{
		"sequence_number":1,"item_id":"rs_part","output_index":0,"summary_index":0,"delta":"partial"
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_part.done", `{
		"sequence_number":2,"item_id":"rs_part","output_index":0,"summary_index":0,
		"status":"incomplete","part":{"type":"summary_text","text":"partial","status":"incomplete"}
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.output_item.done", `{
		"sequence_number":3,"output_index":0,
		"item":{"id":"rs_part","type":"reasoning","status":"completed","summary":[{"type":"summary_text","text":"partial"}]}
	}`)

	activities := observer.Snapshot().Activities
	require.Len(t, activities, 1)
	require.Equal(t, service.ChatMessageActivityStatusIncomplete, activities[0].Status)
}

func TestDeliveredChatStreamObserverPreservesExplicitPartTerminationStatus(t *testing.T) {
	for _, test := range []struct {
		name       string
		upstream   string
		wantStatus string
	}{
		{name: "stopped", upstream: "stopped", wantStatus: service.ChatMessageActivityStatusStopped},
		{name: "disconnected", upstream: "disconnected", wantStatus: service.ChatMessageActivityStatusDisconnected},
		{name: "legacy interrupted", upstream: "interrupted", wantStatus: service.ChatMessageActivityStatusDisconnected},
	} {
		t.Run(test.name, func(t *testing.T) {
			observer := newDeliveredChatStreamObserver()
			observeOpenAIResponsesActivityEnvelope(t, observer, "response.created", `{"sequence_number":0,"response":{"id":"resp_terminal"}}`)
			observeOpenAIResponsesActivityEnvelope(t, observer, "response.output_item.added", `{
				"sequence_number":1,"output_index":0,"item":{"id":"rs_terminal","type":"reasoning"}
			}`)
			observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.delta", `{
				"sequence_number":2,"item_id":"rs_terminal","output_index":0,"summary_index":0,"delta":"kept partial"
			}`)
			payload, err := json.Marshal(map[string]any{
				"sequence_number": 3,
				"item_id":         "rs_terminal",
				"output_index":    0,
				"summary_index":   0,
				"status":          test.upstream,
				"part": map[string]any{
					"type": "summary_text",
					"text": "kept partial",
				},
			})
			require.NoError(t, err)
			observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_part.done", string(payload))

			activities := observer.Snapshot().Activities
			require.Len(t, activities, 1)
			require.Equal(t, "kept partial", activities[0].Text)
			require.Equal(t, test.wantStatus, activities[0].Status)
		})
	}
}

func TestDeliveredChatStreamObserverTimeBoundsMissingSequenceGap(t *testing.T) {
	observer := newDeliveredChatStreamObserver()
	checkpoints := make(chan deliveredChatStreamCheckpoint, 1)
	observer.bindSnapshotCheckpoint(func(checkpoint deliveredChatStreamCheckpoint) error {
		checkpoints <- checkpoint
		return nil
	})
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.created", `{
		"sequence_number":0,"response":{"id":"resp_timed_gap"}
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.output_item.added", `{
		"sequence_number":1,"output_index":0,"item":{"id":"rs_gap","type":"reasoning"}
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.delta", `{
		"sequence_number":3,"item_id":"rs_gap","output_index":0,"summary_index":0,"delta":"B"
	}`)

	select {
	case checkpoint := <-checkpoints:
		require.Len(t, checkpoint.Activities, 1)
		require.Equal(t, "B", checkpoint.Activities[0].Text)
	case <-time.After(time.Second):
		t.Fatal("missing private sequence was not flushed by the reorder timer")
	}

	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.delta", `{
		"sequence_number":4,"item_id":"rs_gap","output_index":0,"summary_index":0,"delta":"C"
	}`)

	activities := observer.Snapshot().Activities
	require.Len(t, activities, 1)
	require.Equal(t, "BC", activities[0].Text)
}

func TestDeliveredChatStreamObserverIgnoresMalformedAndNonSummarySignals(t *testing.T) {
	observer := newDeliveredChatStreamObserver()
	require.NoError(t, observer.Observe([]byte("data: not-json\n\n")))
	require.NoError(t, observer.Observe([]byte(
		`data: {"source":"openai_responses","eventType":"response.reasoning_summary_text.delta","payload":"not-an-object"}`+"\n\n",
	)))
	require.NoError(t, observer.Observe([]byte(
		`data: {"source":"openai_responses","eventType":"response.reasoning_summary_text.delta","payload":[]}`+"\n\n",
	)))
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.output_text.delta", `{"sequence_number":1,"delta":"must not become activity"}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_text.delta", `{"sequence_number":2,"delta":"raw reasoning"}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_part.added", `{
		"sequence_number":3,"item_id":"bad_part","output_index":0,"summary_index":0,"part":{"type":"output_text","text":"no"}
	}`)
	require.NoError(t, observer.Observe([]byte(
		`data: {"choices":[{"delta":{"content":"answer","reasoning_content":"private","reasoning_text":"raw"}}]}`+"\n\n",
	)))

	snapshot := observer.Snapshot()
	require.Equal(t, "answer", snapshot.Content)
	require.Empty(t, snapshot.Activities)
}

func TestDeliveredChatStreamObserverRejectsOrphanAndNonReasoningSummaryEvents(t *testing.T) {
	t.Run("orphan summary", func(t *testing.T) {
		observer := newDeliveredChatStreamObserver()
		observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.delta", `{
			"sequence_number":0,"item_id":"orphan_item","output_index":0,"summary_index":0,
			"delta":"must not persist"
		}`)

		require.Empty(t, observer.Snapshot().Activities)
		require.Equal(t, 1, observer.pendingSummaryCount)
	})

	t.Run("message item", func(t *testing.T) {
		observer := newDeliveredChatStreamObserver()
		observeOpenAIResponsesActivityEnvelope(t, observer, "response.output_item.added", `{
			"sequence_number":0,"output_index":0,
			"item":{"id":"message_item","type":"message"}
		}`)
		observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.delta", `{
			"sequence_number":1,"item_id":"message_item","output_index":0,"summary_index":0,
			"delta":"must not persist"
		}`)

		require.Empty(t, observer.Snapshot().Activities)
		require.Zero(t, observer.pendingSummaryCount)
	})

	t.Run("buffered summary discarded when item is message", func(t *testing.T) {
		observer := newDeliveredChatStreamObserver()
		observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.delta", `{
			"item_id":"late_message_item","output_index":0,"summary_index":0,
			"delta":"must not persist"
		}`)
		observeOpenAIResponsesActivityEnvelope(t, observer, "response.output_item.added", `{
			"output_index":0,"item":{"id":"late_message_item","type":"message"}
		}`)

		require.Empty(t, observer.Snapshot().Activities)
		require.Zero(t, observer.pendingSummaryCount)
	})
}

func TestDeliveredChatStreamObserverReplaysSummaryAfterReasoningItemConfirmation(t *testing.T) {
	observer := newDeliveredChatStreamObserver()
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.delta", `{
		"item_id":"late_reasoning_item","output_index":0,"summary_index":0,"delta":"trusted after confirmation"
	}`)
	require.Empty(t, observer.Snapshot().Activities)
	require.Equal(t, 1, observer.pendingSummaryCount)

	observeOpenAIResponsesActivityEnvelope(t, observer, "response.output_item.added", `{
		"output_index":0,"item":{"id":"late_reasoning_item","type":"reasoning"}
	}`)

	activities := observer.Snapshot().Activities
	require.Len(t, activities, 1)
	require.Equal(t, "late_reasoning_item", activities[0].ItemID)
	require.Equal(t, "trusted after confirmation", activities[0].Text)
	require.Zero(t, observer.pendingSummaryCount)
}

func TestDeliveredChatStreamObserverReplaysOlderSummaryBeforeOutputItemDone(t *testing.T) {
	observer := newDeliveredChatStreamObserver()
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.delta", `{
		"sequence_number":0,"item_id":"late_done_item","output_index":0,"summary_index":0,
		"delta":"draft"
	}`)
	require.Equal(t, 1, observer.pendingSummaryCount)

	observeOpenAIResponsesActivityEnvelope(t, observer, "response.output_item.done", `{
		"sequence_number":1,"output_index":0,
		"item":{"id":"late_done_item","type":"reasoning","status":"completed",
			"summary":[{"type":"summary_text","text":"final"}]}
	}`)

	snapshot := observer.Snapshot()
	require.Len(t, snapshot.Activities, 1)
	require.Equal(t, "final", snapshot.Activities[0].Text)
	require.Equal(t, service.ChatMessageActivityStatusCompleted, snapshot.Activities[0].Status)
	require.JSONEq(t, `{"last_event":"response.output_item.done"}`, string(snapshot.Activities[0].Metadata))
	require.Zero(t, observer.pendingSummaryCount)
}

func TestDeliveredChatStreamObserverFlushesSequencedGapBeforeUnsequencedTerminal(t *testing.T) {
	observer := newDeliveredChatStreamObserver()
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.output_item.added", `{
		"sequence_number":0,"output_index":0,"item":{"id":"unsequenced_terminal","type":"reasoning"}
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.delta", `{
		"sequence_number":2,"item_id":"unsequenced_terminal","output_index":0,"summary_index":0,
		"delta":"draft"
	}`)
	require.Len(t, observer.pendingResponsesEvents, 1)

	observeOpenAIResponsesActivityEnvelope(t, observer, "response.completed", `{
		"response":{"id":"resp_unsequenced_terminal","status":"completed","output":[{
			"id":"unsequenced_terminal","type":"reasoning",
			"summary":[{"type":"summary_text","text":"final"}]
		}]}
	}`)
	_, err := observer.ForceCheckpoint()
	require.NoError(t, err)

	snapshot := observer.Snapshot()
	require.Empty(t, observer.pendingResponsesEvents)
	require.Len(t, snapshot.Activities, 1)
	require.Equal(t, "final", snapshot.Activities[0].Text)
	require.Equal(t, service.ChatMessageActivityStatusCompleted, snapshot.Activities[0].Status)
	require.JSONEq(t, `{"last_event":"response.completed"}`, string(snapshot.Activities[0].Metadata))
}

func TestDeliveredChatStreamObserverTerminalAndInterruptPreservePartialSummary(t *testing.T) {
	tests := []struct {
		name       string
		terminal   string
		wantStatus string
	}{
		{name: "incomplete", terminal: "response.incomplete", wantStatus: "incomplete"},
		{name: "failed", terminal: "response.failed", wantStatus: "failed"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			observer := newDeliveredChatStreamObserver()
			observeOpenAIResponsesActivityEnvelope(t, observer, "response.output_item.added", `{
				"sequence_number":0,"output_index":0,"item":{"id":"rs_partial","type":"reasoning"}
			}`)
			observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.delta", `{
				"sequence_number":1,"item_id":"rs_partial","output_index":0,"summary_index":0,"delta":"partial"
			}`)
			observeOpenAIResponsesActivityEnvelope(t, observer, test.terminal, `{
				"sequence_number":2,"response":{"id":"resp_partial","output":[]}
			}`)
			activity := observer.Snapshot().Activities[0]
			require.Equal(t, "partial", activity.Text)
			require.Equal(t, test.wantStatus, activity.Status)
			require.NotNil(t, activity.CompletedAt)
		})
	}

	observer := newDeliveredChatStreamObserver()
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.output_item.added", `{
		"sequence_number":4,"output_index":0,"item":{"id":"rs_interrupted","type":"reasoning"}
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.delta", `{
		"sequence_number":5,"item_id":"rs_interrupted","output_index":0,"summary_index":0,"delta":"kept"
	}`)
	observer.Freeze()
	activity := observer.Snapshot().Activities[0]
	require.Equal(t, "kept", activity.Text)
	require.Equal(t, service.ChatMessageActivityStatusDisconnected, activity.Status)
	require.JSONEq(t, `{"last_event":"client_disconnected"}`, string(activity.Metadata))
}

func TestDeliveredChatStreamObserverFreezePreservesAppliedResponseTerminal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	observer := newDeliveredChatStreamObserver()
	observer.bindCancellation(ctx.Done())
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.output_item.added", `{
		"sequence_number":0,"output_index":0,"item":{"id":"rs_completed_before_cancel","type":"reasoning"}
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.delta", `{
		"sequence_number":1,"item_id":"rs_completed_before_cancel","output_index":0,"summary_index":0,
		"delta":"finished summary"
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.completed", `{
		"sequence_number":2,"response":{"id":"resp_completed_before_cancel","status":"completed",
			"output":[{"id":"rs_completed_before_cancel","type":"reasoning",
				"summary":[{"type":"summary_text","text":"finished summary"}]}]}
	}`)

	cancel()
	observer.Freeze()

	snapshot := observer.Snapshot()
	require.True(t, snapshot.Done, "response.completed is authoritative even if [DONE] is not delivered")
	require.True(t, observer.responsesTerminalSeen)
	require.Equal(t, service.ChatMessageActivityStatusCompleted, observer.responsesTerminalStatus)
	require.Len(t, snapshot.Activities, 1)
	require.Equal(t, service.ChatMessageActivityStatusCompleted, snapshot.Activities[0].Status)
	require.JSONEq(t, `{"last_event":"response.completed"}`, string(snapshot.Activities[0].Metadata))
}

func TestDeliveredChatStreamObserverBatchesSnapshotCheckpoints(t *testing.T) {
	observer := newDeliveredChatStreamObserver()
	checkpoints := make([]deliveredChatStreamCheckpoint, 0, 4)
	observer.bindSnapshotCheckpoint(func(checkpoint deliveredChatStreamCheckpoint) error {
		checkpoints = append(checkpoints, checkpoint)
		return nil
	})
	observer.setReasoningMetadata(" pro ", " medium ")
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.created", `{"sequence_number":0,"response":{"id":"resp_batch"}}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.output_item.added", `{
		"sequence_number":1,"output_index":0,"item":{"id":"rs_batch","type":"reasoning"}
	}`)
	for sequence := 2; sequence <= 25; sequence++ {
		payload, err := json.Marshal(map[string]any{
			"sequence_number": sequence,
			"item_id":         "rs_batch",
			"output_index":    0,
			"summary_index":   0,
			"delta":           "x",
		})
		require.NoError(t, err)
		observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.delta", string(payload))
		if sequence == 2 {
			waitForDeliveredChatSnapshotCheckpoint(t, observer)
		}
	}
	_, err := observer.ForceCheckpoint()
	require.NoError(t, err)

	require.Len(t, checkpoints, 2, "24 deltas should produce one live snapshot and one final flush")
	require.NotEmpty(t, checkpoints)
	final := checkpoints[len(checkpoints)-1]
	require.Equal(t, "pro", final.ReasoningMode)
	require.Empty(t, final.ReasoningEffort)
	require.Len(t, final.Activities, 1)
	require.Len(t, final.Activities[0].Text, 24)
	for index := 1; index < len(checkpoints); index++ {
		require.Greater(t, checkpoints[index].Sequence, checkpoints[index-1].Sequence)
	}
}

func TestDeliveredChatStreamObserverBuildsLongSummaryIncrementally(t *testing.T) {
	observer := newDeliveredChatStreamObserver()
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.output_item.added", `{
		"sequence_number":0,"output_index":0,"item":{"id":"rs_long","type":"reasoning"}
	}`)

	const delta = "0123456789abcdef"
	const deltaCount = 4096
	for sequence := 1; sequence <= deltaCount; sequence++ {
		payload, err := json.Marshal(map[string]any{
			"sequence_number": sequence,
			"item_id":         "rs_long",
			"output_index":    0,
			"summary_index":   0,
			"delta":           delta,
		})
		require.NoError(t, err)
		observeOpenAIResponsesActivityEnvelope(
			t,
			observer,
			"response.reasoning_summary_text.delta",
			string(payload),
		)
	}

	require.Len(t, observer.activities, 1)
	for _, observed := range observer.activities {
		require.Empty(t, observed.activity.Text, "the mutable builder owns live text until snapshot")
		require.Equal(t, len(delta)*deltaCount, observed.text.Len())
	}
	snapshot := observer.Snapshot()
	require.Len(t, snapshot.Activities, 1)
	require.Equal(t, strings.Repeat(delta, deltaCount), snapshot.Activities[0].Text)
}

func TestDeliveredChatStreamObserverBoundsLongSummaryCheckpointBytes(t *testing.T) {
	observer := newDeliveredChatStreamObserver()
	var checkpointMu sync.Mutex
	checkpointSizes := make([]int, 0, 8)
	observer.bindSnapshotCheckpoint(func(checkpoint deliveredChatStreamCheckpoint) error {
		checkpointMu.Lock()
		defer checkpointMu.Unlock()
		size := 0
		if len(checkpoint.Activities) == 1 {
			size = len(checkpoint.Activities[0].Text)
		}
		checkpointSizes = append(checkpointSizes, size)
		return nil
	})
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.output_item.added", `{
		"sequence_number":0,"output_index":0,"item":{"id":"rs_checkpoint_growth","type":"reasoning"}
	}`)

	const delta = "0123456789abcdef"
	const deltaCount = 2048
	for sequence := 1; sequence <= deltaCount; sequence++ {
		payload, err := json.Marshal(map[string]any{
			"sequence_number": sequence,
			"item_id":         "rs_checkpoint_growth",
			"output_index":    0,
			"summary_index":   0,
			"delta":           delta,
		})
		require.NoError(t, err)
		observeOpenAIResponsesActivityEnvelope(
			t,
			observer,
			"response.reasoning_summary_text.delta",
			string(payload),
		)
		waitForDeliveredChatSnapshotCheckpoint(t, observer)
	}
	_, err := observer.ForceCheckpoint()
	require.NoError(t, err)

	checkpointMu.Lock()
	sizes := append([]int(nil), checkpointSizes...)
	checkpointMu.Unlock()
	require.NotEmpty(t, sizes)
	require.LessOrEqual(t, len(sizes), 8, "geometric growth must bound full-prefix writes")
	finalSize := len(delta) * deltaCount
	require.Equal(t, finalSize, sizes[len(sizes)-1])
	totalBytes := 0
	for _, size := range sizes {
		totalBytes += size
	}
	require.LessOrEqual(t, totalBytes, finalSize*3)
}

func TestDeliveredChatStreamObserverSkipsEmptyPartCheckpoint(t *testing.T) {
	observer := newDeliveredChatStreamObserver()
	checkpoints := make([]deliveredChatStreamCheckpoint, 0, 1)
	observer.setReasoningMetadata("pro", "medium")
	observer.bindSnapshotCheckpoint(func(checkpoint deliveredChatStreamCheckpoint) error {
		checkpoints = append(checkpoints, checkpoint)
		return nil
	})

	observeOpenAIResponsesActivityEnvelope(t, observer, "response.created", `{
		"sequence_number":0,"response":{"id":"resp_empty_checkpoint"}
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.output_item.added", `{
		"sequence_number":1,"output_index":0,
		"item":{"id":"rs_empty_checkpoint","type":"reasoning"}
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_part.added", `{
		"sequence_number":2,"item_id":"rs_empty_checkpoint","output_index":0,"summary_index":0,
		"part":{"type":"summary_text","text":""}
	}`)

	require.Empty(t, checkpoints, "an empty summary part has no durable payload")

	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.delta", `{
		"sequence_number":3,"item_id":"rs_empty_checkpoint","output_index":0,"summary_index":0,
		"delta":"visible"
	}`)
	waitForDeliveredChatSnapshotCheckpoint(t, observer)
	require.Len(t, checkpoints, 1)
	require.Equal(t, "visible", checkpoints[0].Activities[0].Text)
}

func TestDeliveredChatStreamObserverForcesAuthoritativeBoundaryCheckpoints(t *testing.T) {
	observer := newDeliveredChatStreamObserver()
	checkpoints := make([]deliveredChatStreamCheckpoint, 0, 5)
	observer.bindSnapshotCheckpoint(func(checkpoint deliveredChatStreamCheckpoint) error {
		checkpoints = append(checkpoints, checkpoint)
		return nil
	})

	observeOpenAIResponsesActivityEnvelope(t, observer, "response.created", `{
		"sequence_number":0,"response":{"id":"resp_forced_checkpoint"}
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.output_item.added", `{
		"sequence_number":1,"output_index":0,
		"item":{"id":"rs_forced_checkpoint","type":"reasoning"}
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.delta", `{
		"sequence_number":2,"item_id":"rs_forced_checkpoint","output_index":0,"summary_index":0,
		"delta":"draft"
	}`)
	waitForDeliveredChatSnapshotCheckpoint(t, observer)
	require.Len(t, checkpoints, 1)

	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.delta", `{
		"sequence_number":3,"item_id":"rs_forced_checkpoint","output_index":0,"summary_index":0,
		"delta":" stale"
	}`)
	require.Len(t, checkpoints, 1, "ordinary deltas remain batched")

	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.done", `{
		"sequence_number":4,"item_id":"rs_forced_checkpoint","output_index":0,"summary_index":0,
		"text":"correct"
	}`)
	waitForDeliveredChatSnapshotCheckpoint(t, observer)
	require.Len(t, checkpoints, 2)
	require.Equal(t, "correct", checkpoints[1].Activities[0].Text)

	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_part.done", `{
		"sequence_number":5,"item_id":"rs_forced_checkpoint","output_index":0,"summary_index":0,
		"part":{"type":"summary_text","text":"fallback"}
	}`)
	waitForDeliveredChatSnapshotCheckpoint(t, observer)
	require.Len(t, checkpoints, 3)
	require.Equal(t, "correct", checkpoints[2].Activities[0].Text)

	observeOpenAIResponsesActivityEnvelope(t, observer, "response.output_item.done", `{
		"sequence_number":6,"output_index":0,
		"item":{"id":"rs_forced_checkpoint","type":"reasoning","status":"completed",
			"summary":[{"type":"summary_text","text":"fallback"}]}
	}`)
	waitForDeliveredChatSnapshotCheckpoint(t, observer)
	require.Len(t, checkpoints, 4)

	observeOpenAIResponsesActivityEnvelope(t, observer, "response.completed", `{
		"sequence_number":7,"response":{"id":"resp_forced_checkpoint","status":"completed",
			"output":[{"id":"rs_forced_checkpoint","type":"reasoning",
				"summary":[{"type":"summary_text","text":"fallback"}]}]}
	}`)
	waitForDeliveredChatSnapshotCheckpoint(t, observer)
	require.Len(t, checkpoints, 5)

	_, err := observer.ForceCheckpoint()
	require.NoError(t, err)
	require.Len(t, checkpoints, 5, "a clean terminal snapshot must not be written twice")
}

func TestDeliveredChatStreamObserverForcesOnlyAfterReorderedBoundaryIsApplied(t *testing.T) {
	observer := newDeliveredChatStreamObserver()
	checkpoints := make([]deliveredChatStreamCheckpoint, 0, 2)
	observer.bindSnapshotCheckpoint(func(checkpoint deliveredChatStreamCheckpoint) error {
		checkpoints = append(checkpoints, checkpoint)
		return nil
	})

	observeOpenAIResponsesActivityEnvelope(t, observer, "response.created", `{
		"sequence_number":0,"response":{"id":"resp_reordered_checkpoint"}
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.output_item.added", `{
		"sequence_number":1,"output_index":0,
		"item":{"id":"rs_reordered_checkpoint","type":"reasoning"}
	}`)
	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.delta", `{
		"sequence_number":2,"item_id":"rs_reordered_checkpoint","output_index":0,"summary_index":0,
		"delta":"draft"
	}`)
	waitForDeliveredChatSnapshotCheckpoint(t, observer)
	require.Len(t, checkpoints, 1)

	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.done", `{
		"sequence_number":4,"item_id":"rs_reordered_checkpoint","output_index":0,"summary_index":0,
		"text":"authoritative"
	}`)
	require.Len(t, checkpoints, 1, "a buffered done event must not force an older snapshot")

	observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_part.added", `{
		"sequence_number":3,"item_id":"rs_reordered_checkpoint","output_index":0,"summary_index":0,
		"part":{"type":"summary_text"}
	}`)
	waitForDeliveredChatSnapshotCheckpoint(t, observer)
	require.Len(t, checkpoints, 2, "the drain should persist one combined authoritative snapshot")
	require.Equal(t, "authoritative", checkpoints[1].Activities[0].Text)
	require.Equal(t, service.ChatMessageActivityStatusCompleted, checkpoints[1].Activities[0].Status)
}

func TestDeliveredChatStreamObserverForcesEveryResponseTerminalStatus(t *testing.T) {
	for _, test := range []struct {
		eventType  string
		wantStatus string
	}{
		{eventType: "response.completed", wantStatus: service.ChatMessageActivityStatusCompleted},
		{eventType: "response.incomplete", wantStatus: service.ChatMessageActivityStatusIncomplete},
		{eventType: "response.failed", wantStatus: service.ChatMessageActivityStatusFailed},
	} {
		t.Run(test.eventType, func(t *testing.T) {
			observer := newDeliveredChatStreamObserver()
			checkpoints := make([]deliveredChatStreamCheckpoint, 0, 2)
			observer.bindSnapshotCheckpoint(func(checkpoint deliveredChatStreamCheckpoint) error {
				checkpoints = append(checkpoints, checkpoint)
				return nil
			})
			observeOpenAIResponsesActivityEnvelope(t, observer, "response.output_item.added", `{
				"sequence_number":0,"output_index":0,
				"item":{"id":"rs_terminal_checkpoint","type":"reasoning"}
			}`)

			observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.delta", `{
				"sequence_number":1,"item_id":"rs_terminal_checkpoint","output_index":0,"summary_index":0,
				"delta":"partial"
			}`)
			waitForDeliveredChatSnapshotCheckpoint(t, observer)
			require.Len(t, checkpoints, 1)
			observeOpenAIResponsesActivityEnvelope(t, observer, test.eventType, `{
				"sequence_number":2,"response":{"id":"resp_terminal_checkpoint","output":[]}
			}`)
			waitForDeliveredChatSnapshotCheckpoint(t, observer)

			require.Len(t, checkpoints, 2)
			require.Equal(t, test.wantStatus, checkpoints[1].Activities[0].Status)
			require.NotNil(t, checkpoints[1].Activities[0].CompletedAt)
			snapshot := observer.Snapshot()
			require.True(t, snapshot.ResponsesTerminalSeen)
			require.Equal(t, test.wantStatus, snapshot.ResponsesTerminalStatus)
			require.Equal(t, test.eventType == "response.completed", snapshot.Done)

			observeOpenAIResponsesActivityEnvelope(t, observer, test.eventType, `{
				"sequence_number":2,"response":{"id":"resp_terminal_checkpoint","output":[]}
			}`)
			require.Len(t, checkpoints, 2, "a duplicate terminal sequence must not write again")
		})
	}
}

func TestDeliveredChatStreamObserverDoesNotBlockSSEOnSlowSnapshotCheckpoint(t *testing.T) {
	observer := newDeliveredChatStreamObserver()
	started := make(chan struct{})
	release := make(chan struct{})
	var startedOnce sync.Once
	var writes atomic.Int32
	observer.bindSnapshotCheckpoint(func(deliveredChatStreamCheckpoint) error {
		writes.Add(1)
		startedOnce.Do(func() { close(started) })
		<-release
		return nil
	})

	firstObserveDone := make(chan error, 1)
	go func() {
		firstObserveDone <- observer.Observe([]byte(
			`data: {"choices":[{"delta":{"content":"a"}}]}` + "\n\n",
		))
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		close(release)
		t.Fatal("checkpoint callback did not start")
	}
	select {
	case err := <-firstObserveDone:
		require.NoError(t, err)
	case <-time.After(500 * time.Millisecond):
		close(release)
		t.Fatal("initial SSE observation blocked on the checkpoint callback")
	}

	observeDone := make(chan error, 1)
	go func() {
		observeDone <- observer.Observe([]byte(
			`data: {"choices":[{"delta":{"content":"b"}}]}` + "\n\n",
		))
	}()
	select {
	case err := <-observeDone:
		require.NoError(t, err)
	case <-time.After(500 * time.Millisecond):
		close(release)
		t.Fatal("SSE observation blocked on the checkpoint callback")
	}

	forceDone := make(chan error, 1)
	go func() {
		_, err := observer.ForceCheckpoint()
		forceDone <- err
	}()
	close(release)
	select {
	case err := <-forceDone:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("final checkpoint did not drain")
	}
	require.Equal(t, "ab", observer.Snapshot().Content)
	require.LessOrEqual(t, writes.Load(), int32(2), "slow writes must be coalesced")
}

func TestDeliveredChatStreamObserverCheckpointFailureDoesNotTruncateSSE(t *testing.T) {
	observer := newDeliveredChatStreamObserver()
	checkpointErr := errors.New("checkpoint unavailable")
	observer.bindSnapshotCheckpoint(func(deliveredChatStreamCheckpoint) error {
		return checkpointErr
	})
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	writer := &deliveredChatResponseWriter{
		ResponseWriter: ginContext.Writer,
		observer:       observer,
	}

	firstFrame := `data: {"choices":[{"delta":{"content":"kept"}}]}` + "\n\n"
	written, err := writer.WriteString(firstFrame)
	require.NoError(t, err)
	require.Equal(t, len(firstFrame), written)
	target := observer.snapshotCheckpointTarget()
	require.ErrorIs(t, observer.waitForSnapshotCheckpoints(target), checkpointErr)
	secondFrame := `data: {"choices":[{"delta":{"content":" continuing"}}]}` + "\n\n"
	written, err = writer.WriteString(secondFrame)
	require.NoError(t, err, "checkpoint failure must not become a writer error")
	require.Equal(t, len(secondFrame), written)

	_, err = observer.ForceCheckpoint()
	require.ErrorIs(t, err, checkpointErr)
	require.Equal(t, "kept continuing", observer.Snapshot().Content)
	require.Equal(t, firstFrame+secondFrame, recorder.Body.String())
}

func TestDeliveredChatStreamObserverBoundsLongStreamCheckpointWrites(t *testing.T) {
	observer := newDeliveredChatStreamObserver()
	var writes atomic.Int32
	var final deliveredChatStreamCheckpoint
	var finalMu sync.Mutex
	observer.bindSnapshotCheckpoint(func(checkpoint deliveredChatStreamCheckpoint) error {
		writes.Add(1)
		finalMu.Lock()
		final = checkpoint
		finalMu.Unlock()
		return nil
	})

	for index := 0; index < 256; index++ {
		require.NoError(t, observer.Observe([]byte(
			`data: {"choices":[{"delta":{"content":"x"}}]}`+"\n\n",
		)))
		if index == 0 {
			waitForDeliveredChatSnapshotCheckpoint(t, observer)
			observer.mu.Lock()
			observer.lastSnapshotCheckpointAt = time.Now().Add(time.Hour)
			observer.mu.Unlock()
		}
	}
	_, err := observer.ForceCheckpoint()
	require.NoError(t, err)

	finalMu.Lock()
	defer finalMu.Unlock()
	require.Len(t, final.Content, 256)
	require.LessOrEqual(t, writes.Load(), int32(5), "256 deltas must not cause token-level writes")
}

func TestDeliveredChatStreamObserverRequestReasoningMetadataIsAuthoritative(t *testing.T) {
	for _, test := range []struct {
		name           string
		requestMode    string
		requestEffort  string
		upstreamMode   string
		upstreamEffort string
	}{
		{
			name:           "standard request is not promoted to pro",
			requestMode:    "standard",
			requestEffort:  "low",
			upstreamMode:   "pro",
			upstreamEffort: "high",
		},
		{
			name:           "pro request is not silently downgraded",
			requestMode:    "pro",
			requestEffort:  "medium",
			upstreamMode:   "standard",
			upstreamEffort: "low",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			observer := newDeliveredChatStreamObserver()
			observer.setReasoningMetadata(test.requestMode, test.requestEffort)
			created, err := json.Marshal(map[string]any{
				"sequence_number": 0,
				"response": map[string]any{
					"id": "resp_authoritative_metadata",
					"reasoning": map[string]any{
						"mode":   test.upstreamMode,
						"effort": test.upstreamEffort,
					},
				},
			})
			require.NoError(t, err)
			observeOpenAIResponsesActivityEnvelope(t, observer, "response.created", string(created))
			observeOpenAIResponsesActivityEnvelope(t, observer, "response.output_item.added", `{
				"sequence_number":1,"output_index":0,
				"item":{"id":"rs_authoritative_metadata","type":"reasoning"}
			}`)
			observeOpenAIResponsesActivityEnvelope(t, observer, "response.reasoning_summary_text.delta", `{
				"sequence_number":2,"item_id":"rs_authoritative_metadata","output_index":0,"summary_index":0,
				"delta":"summary"
			}`)
			terminal, err := json.Marshal(map[string]any{
				"sequence_number": 3,
				"response": map[string]any{
					"id": "resp_authoritative_metadata",
					"reasoning": map[string]any{
						"mode":   test.upstreamMode,
						"effort": test.upstreamEffort,
					},
					"output": []any{},
				},
			})
			require.NoError(t, err)
			observeOpenAIResponsesActivityEnvelope(t, observer, "response.completed", string(terminal))

			snapshot := observer.Snapshot()
			wantEffort := test.requestEffort
			if test.requestMode == service.WebChatReasoningModePro {
				wantEffort = ""
			}
			require.Equal(t, test.requestMode, snapshot.ReasoningMode)
			require.Equal(t, wantEffort, snapshot.ReasoningEffort)
			require.Len(t, snapshot.Activities, 1)
			require.Equal(t, test.requestMode, snapshot.Activities[0].ReasoningMode)
			require.Equal(t, wantEffort, snapshot.Activities[0].ReasoningEffort)
		})
	}
}

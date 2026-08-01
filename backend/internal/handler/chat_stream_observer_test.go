package handler

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

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

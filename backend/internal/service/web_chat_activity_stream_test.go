package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/stretchr/testify/require"
)

func TestBuildWebChatActivitySSEPreservesOfficialSummaryPayload(t *testing.T) {
	payload := []byte(`{"type":"response.reasoning_summary_text.delta","item_id":"rs_1","output_index":0,"summary_index":0,"delta":"checking","sequence_number":7}`)
	var event apicompat.ResponsesStreamEvent
	require.NoError(t, json.Unmarshal(payload, &event))

	sse, ok := buildWebChatActivitySSE(payload, &event)
	require.True(t, ok)
	require.True(t, strings.HasPrefix(sse, "data: "))

	var envelope webChatActivityEnvelope
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSuffix(strings.TrimPrefix(sse, "data: "), "\n\n")), &envelope))
	require.Equal(t, webChatActivitySource, envelope.Source)
	require.Equal(t, event.Type, envelope.EventType)
	require.JSONEq(t, string(payload), string(envelope.Payload))
}

func TestBuildWebChatActivitySSEFiltersNonSummaryAndNonReasoningItems(t *testing.T) {
	tests := []apicompat.ResponsesStreamEvent{
		{Type: "response.output_text.delta", Delta: "answer"},
		{Type: "response.reasoning_text.delta", Delta: "private reasoning"},
		{Type: "response.output_item.added", Item: &apicompat.ResponsesOutput{Type: "message"}},
	}
	for i := range tests {
		payload, err := json.Marshal(tests[i])
		require.NoError(t, err)
		_, ok := buildWebChatActivitySSE(payload, &tests[i])
		require.False(t, ok)
	}

	reasoning := apicompat.ResponsesStreamEvent{
		Type: "response.output_item.added",
		Item: &apicompat.ResponsesOutput{Type: "reasoning", ID: "rs_1"},
	}
	payload, err := json.Marshal(reasoning)
	require.NoError(t, err)
	_, ok := buildWebChatActivitySSE(payload, &reasoning)
	require.True(t, ok)
}

func TestBuildWebChatActivitySSERedactsReasoningItemSecrets(t *testing.T) {
	payload := []byte(`{
		"type":"response.output_item.done","output_index":2,"sequence_number":9,
		"account_id":"internal-account","error":{"message":"database host"},
		"item":{"id":"rs_private","type":"reasoning","status":"completed",
			"encrypted_content":"ciphertext-secret",
			"content":[{"type":"output_text","text":"raw-reasoning-secret"}],
			"summary":[
				{"type":"summary_text","text":"safe summary","internal":"hidden"},
				{"type":"output_text","text":"answer-secret"}
			]
		}
	}`)
	var event apicompat.ResponsesStreamEvent
	require.NoError(t, json.Unmarshal(payload, &event))

	sse, ok := buildWebChatActivitySSE(payload, &event)
	require.True(t, ok)
	var envelope webChatActivityEnvelope
	require.NoError(t, json.Unmarshal(
		[]byte(strings.TrimSuffix(strings.TrimPrefix(sse, "data: "), "\n\n")),
		&envelope,
	))
	var safe webChatActivityPayload
	require.NoError(t, json.Unmarshal(envelope.Payload, &safe))
	require.NotNil(t, safe.OutputIndex)
	require.Equal(t, 2, *safe.OutputIndex)
	require.NotNil(t, safe.SequenceNumber)
	require.Equal(t, int64(9), *safe.SequenceNumber)
	require.Equal(t, &webChatActivityItem{
		ID:     "rs_private",
		Type:   "reasoning",
		Status: "completed",
		Summary: []webChatActivitySummary{{
			Type: "summary_text",
			Text: "safe summary",
		}},
	}, safe.Item)
	require.NotContains(t, string(envelope.Payload), "ciphertext-secret")
	require.NotContains(t, string(envelope.Payload), "raw-reasoning-secret")
	require.NotContains(t, string(envelope.Payload), "answer-secret")
	require.NotContains(t, string(envelope.Payload), "internal-account")
	require.NotContains(t, string(envelope.Payload), "database host")
}

func TestBuildWebChatActivitySSERedactsTerminalPayloadAndPreservesOutputIndex(t *testing.T) {
	payload := []byte(`{
		"type":"response.failed","sequence_number":12,"internal_route":"account-7",
		"response":{"id":"resp_safe","status":"failed","model":"private-model-route",
			"reasoning":{"mode":"pro","effort":"medium","raw":"hidden"},
			"error":{"code":"upstream_error","message":"postgres://internal-host"},
			"usage":{"input_tokens":100,"output_tokens":20},
			"output":[
				{"id":"msg_1","type":"message","content":[{"type":"output_text","text":"answer-secret"}]},
				{"id":"rs_1","type":"reasoning","encrypted_content":"cipher-secret",
					"summary":[{"type":"summary_text","text":"safe terminal summary"}]}
			]
		}
	}`)
	var event apicompat.ResponsesStreamEvent
	require.NoError(t, json.Unmarshal(payload, &event))

	sse, ok := buildWebChatActivitySSE(payload, &event)
	require.True(t, ok)
	var envelope webChatActivityEnvelope
	require.NoError(t, json.Unmarshal(
		[]byte(strings.TrimSuffix(strings.TrimPrefix(sse, "data: "), "\n\n")),
		&envelope,
	))
	var safe webChatActivityPayload
	require.NoError(t, json.Unmarshal(envelope.Payload, &safe))
	require.NotNil(t, safe.Response)
	require.Equal(t, "resp_safe", safe.Response.ID)
	require.Equal(t, "failed", safe.Response.Status)
	require.Equal(t, &webChatActivityReasoning{Mode: "pro", Effort: "medium"}, safe.Response.Reasoning)
	require.Len(t, safe.Response.Output, 2)
	require.Equal(t, "redacted", safe.Response.Output[0].Type)
	require.Equal(t, "reasoning", safe.Response.Output[1].Type)
	require.Equal(t, "safe terminal summary", safe.Response.Output[1].Summary[0].Text)
	require.NotContains(t, string(envelope.Payload), "answer-secret")
	require.NotContains(t, string(envelope.Payload), "cipher-secret")
	require.NotContains(t, string(envelope.Payload), "private-model-route")
	require.NotContains(t, string(envelope.Payload), "postgres://internal-host")
	require.NotContains(t, string(envelope.Payload), "account-7")
}

func TestWebChatSuppressConvertedReasoningKeepsAnswerContent(t *testing.T) {
	require.True(t, webChatSuppressConvertedReasoning(&apicompat.ResponsesStreamEvent{Type: "response.reasoning_summary_text.delta"}))
	require.True(t, webChatSuppressConvertedReasoning(&apicompat.ResponsesStreamEvent{Type: "response.reasoning_text.delta"}))
	require.False(t, webChatSuppressConvertedReasoning(&apicompat.ResponsesStreamEvent{Type: "response.output_text.delta"}))
}

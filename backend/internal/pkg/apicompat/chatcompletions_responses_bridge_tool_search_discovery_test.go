package apicompat

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// Codex sends tool_search discoveries in two wire forms: older clients use an
// `output` value, while newer clients put the discovered definitions in a
// top-level `tools` array. The Responses→Chat bridge must preserve either form
// as the Chat tool message content so a follow-up turn can load the tools.
func TestResponsesInputToChatMessages_ToolSearchDiscoveryToolsField(t *testing.T) {
	input := json.RawMessage(`[
		{"type":"tool_search_call","call_id":"call_s","arguments":{"query":"github"}},
		{"type":"tool_search_output","call_id":"call_s","status":"completed","execution":"client","tools":[
			{"type":"namespace","name":"github","tools":[
				{"type":"function","name":"issues","parameters":{"type":"object"}}
			]}
		]}
	]`)

	messages, err := responsesInputToChatMessages("", input)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	require.Equal(t, "tool", messages[1].Role)
	require.Equal(t, "call_s", messages[1].ToolCallID)
	var discovery string
	require.NoError(t, json.Unmarshal(messages[1].Content, &discovery))
	require.JSONEq(t, `[{"type":"namespace","name":"github","tools":[{"type":"function","name":"issues","parameters":{"type":"object"}}]}]`, discovery)
}

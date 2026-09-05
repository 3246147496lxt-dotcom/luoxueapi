package service

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat"
	"github.com/tidwall/gjson"
)

// shouldForwardOpenAIResponsesViaRawChatCompletions decides whether an
// OpenAI Responses request must be bridged to the upstream Chat Completions
// endpoint.  The legacy capability marker remains authoritative for generic
// OpenAI API-key accounts; first-class DeepSeek accounts use their explicit
// protocol setting so a newly-created account safely defaults to Chat
// Completions.
func shouldForwardOpenAIResponsesViaRawChatCompletions(account *Account) bool {
	if account == nil || account.Type != AccountTypeAPIKey {
		return false
	}
	if account.IsCNProvider() {
		switch account.GetAPIProtocol() {
		case APIProtocolResponses:
			return false
		case APIProtocolAdaptive:
			// DeepSeek has a native Responses endpoint; providers without one
			// must continue using the Chat bridge in adaptive mode.
			return !account.SupportsNativeCNResponses()
		case APIProtocolAnthropic, APIProtocolChatCompletions:
			return true
		default:
			return true
		}
	}
	return !openai_compat.ShouldUseResponsesAPI(account.Extra)
}

// shouldUseNativeDeepSeekResponses reports whether a DeepSeek Responses
// request should stay on the provider's native /responses endpoint. Compact
// requests are intentionally bridged because the DeepSeek endpoint does not
// implement OpenAI's compaction subresource contract.
func shouldUseNativeDeepSeekResponses(account *Account, compact bool) bool {
	if account == nil || !account.SupportsNativeCNResponses() {
		return false
	}
	return account.UsesNativeCNResponses() && !compact
}

// deepSeekResponsesNeedsChatFallback reports whether a request uses a
// Responses client-tool shape that DeepSeek's native endpoint cannot execute.
// DeepSeek supports the built-in `apply_patch` custom tool, ordinary function
// tools, and web_search; Codex's other custom tools, tool_search, namespaces,
// and their history items need the existing Responses→Chat bridge, which can
// lower and restore those items.  The helper deliberately operates on the raw
// JSON so it also sees tools declared inside `additional_tools` input items.
func deepSeekResponsesNeedsChatFallback(body []byte) bool {
	root := gjson.ParseBytes(body)
	if !root.Exists() || !root.IsObject() {
		return false
	}

	// Collect custom call names first. Output items often carry only call_id and
	// may appear after the corresponding call in the input array; a two-pass
	// scan avoids making the result depend on JSON field order.
	customCallNames := make(map[string]string)
	var collect func(gjson.Result)
	collect = func(value gjson.Result) {
		if value.IsObject() {
			typ := strings.TrimSpace(value.Get("type").String())
			if typ == "custom_tool_call" {
				callID := strings.TrimSpace(value.Get("call_id").String())
				if callID != "" {
					customCallNames[callID] = strings.TrimSpace(value.Get("name").String())
				}
			}
		}
		if value.IsObject() || value.IsArray() {
			value.ForEach(func(_, child gjson.Result) bool {
				collect(child)
				return true
			})
		}
	}
	collect(root)

	needsFallback := false
	// The Responses JSON decoder also accepts the shorthand string form for a
	// custom tool (for example, `tools:["exec"]`).  Walk only arrays attached
	// to tool-bearing keys so ordinary string content in prompts/inputs cannot
	// accidentally force the Chat bridge.  DeepSeek's native endpoint permits
	// only the exact built-in `apply_patch` custom tool in this form.
	var inspectToolStrings func(gjson.Result)
	inspectToolStrings = func(value gjson.Result) {
		if needsFallback {
			return
		}
		if value.IsObject() {
			value.ForEach(func(key, child gjson.Result) bool {
				if key.Type == gjson.String && (key.String() == "tools" || key.String() == "additional_tools") && child.IsArray() {
					child.ForEach(func(_, item gjson.Result) bool {
						if item.Type == gjson.String && item.String() != "apply_patch" {
							needsFallback = true
							return false
						}
						return true
					})
					if needsFallback {
						return false
					}
				}
				inspectToolStrings(child)
				return !needsFallback
			})
			return
		}
		if value.IsArray() {
			value.ForEach(func(_, child gjson.Result) bool {
				inspectToolStrings(child)
				return !needsFallback
			})
		}
	}
	inspectToolStrings(root)
	if needsFallback {
		return true
	}

	var inspect func(gjson.Result)
	inspect = func(value gjson.Result) {
		if needsFallback {
			return
		}
		if value.IsObject() {
			typ := strings.TrimSpace(value.Get("type").String())
			switch typ {
			case "custom":
				// DeepSeek's native Responses contract reserves custom-tool
				// handling for apply_patch. Empty/unknown names are not safe to
				// send natively either.
				name := value.Get("name").String()
				if name != "apply_patch" {
					needsFallback = true
					return
				}
			case "custom_tool_call":
				name := value.Get("name").String()
				if name != "apply_patch" {
					needsFallback = true
					return
				}
			case "custom_tool_call_output":
				callID := strings.TrimSpace(value.Get("call_id").String())
				if customCallNames[callID] != "apply_patch" {
					needsFallback = true
					return
				}
			case "tool_search", "tool_search_call", "tool_search_output", "namespace":
				needsFallback = true
				return
			}
		}
		if value.IsObject() || value.IsArray() {
			value.ForEach(func(_, child gjson.Result) bool {
				inspect(child)
				return !needsFallback
			})
		}
	}
	inspect(root)
	return needsFallback
}

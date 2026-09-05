package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
)

// validateDeepSeekAnthropicCrossChatRequest is retained for existing callers
// and tests. It uses the generic CN-provider validator with the historical
// DeepSeek wording.
func validateDeepSeekAnthropicCrossChatRequest(req *apicompat.ChatCompletionsRequest) error {
	return validateCNAnthropicCrossChatRequest(req, "DeepSeek")
}

// validateCNAnthropicCrossChatRequest keeps the Chat Completions to native
// Anthropic bridge deliberately small. The bridge can faithfully represent
// ordinary function tools, but the Chat→Responses converter drops unknown tool
// kinds. Silently dropping one would change the model contract, so reject it
// before any upstream request is issued.
func validateCNAnthropicCrossChatRequest(req *apicompat.ChatCompletionsRequest, provider string) error {
	provider = normalizeCNAnthropicProviderLabel(provider)
	if req == nil {
		return fmt.Errorf("invalid request: chat completions payload is nil")
	}
	if len(req.Messages) == 0 {
		// A Responses-shaped payload (for example {"input": ...}) can be
		// unmarshaled into ChatCompletionsRequest without an error, but the
		// shared converter would then silently send an empty Anthropic prompt.
		// Reject it explicitly so callers do not accidentally hit a different
		// upstream protocol with lost input.
		return fmt.Errorf("messages is required for %s Anthropic cross-protocol requests", provider)
	}
	for i, tool := range req.Tools {
		if strings.ToLower(strings.TrimSpace(tool.Type)) != "function" {
			return fmt.Errorf("unsupported tool type %q at tools[%d]: %s Anthropic cross-protocol supports only function tools", tool.Type, i, provider)
		}
		if tool.Function == nil || strings.TrimSpace(tool.Function.Name) == "" {
			return fmt.Errorf("invalid function tool at tools[%d]: function.name is required", i)
		}
	}
	return nil
}

// validateDeepSeekAnthropicCrossResponsesRequest is the compatibility wrapper
// for the original DeepSeek-only call sites/tests.
func validateDeepSeekAnthropicCrossResponsesRequest(req *apicompat.ResponsesRequest) error {
	return validateCNAnthropicCrossResponsesRequest(req, "DeepSeek")
}

// validateCNAnthropicCrossResponsesRequest performs the equivalent guard for a
// Responses request. Responses has server/client tool families (web search,
// computer use, custom, namespaces, and newer tool-search variants) that have
// no lossless representation in Anthropic Messages on CN provider facades.
func validateCNAnthropicCrossResponsesRequest(req *apicompat.ResponsesRequest, provider string) error {
	provider = normalizeCNAnthropicProviderLabel(provider)
	if req == nil {
		return fmt.Errorf("invalid request: responses payload is nil")
	}
	for i, tool := range req.Tools {
		if err := validateCNAnthropicResponseTool(tool, fmt.Sprintf("tools[%d]", i), provider); err != nil {
			return err
		}
	}

	// Inspect typed input items for tool declarations that the converter would
	// otherwise ignore.  We intentionally leave ordinary reasoning/message
	// history alone; those items are converted by the shared compatibility
	// layer.  Only unsupported tool families are rejected here.
	raw := bytesTrimSpaceForCrossValidation(req.Input)
	if len(raw) == 0 || string(raw) == "null" || raw[0] != '[' {
		return nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return fmt.Errorf("parse responses input for %s Anthropic validation: %w", provider, err)
	}
	for i, itemRaw := range items {
		var item struct {
			Type  string            `json:"type"`
			Tools []json.RawMessage `json:"tools"`
		}
		if err := json.Unmarshal(itemRaw, &item); err != nil {
			// The normal converter will provide the user-facing parse error. Do
			// not mask it with a duplicate validation error.
			continue
		}
		typ := strings.ToLower(strings.TrimSpace(item.Type))
		switch typ {
		case "additional_tools":
			return fmt.Errorf("unsupported input item type %q at input[%d]: %s Anthropic cross-protocol does not support client tool bundles", item.Type, i, provider)
		case "web_search_call", "computer_call", "computer_call_output", "local_shell_call", "tool_search_call", "custom_tool_call":
			return fmt.Errorf("unsupported input item type %q at input[%d]: %s Anthropic cross-protocol supports only text and function tools", item.Type, i, provider)
		}
	}
	return nil
}

func validateDeepSeekAnthropicResponseTool(tool apicompat.ResponsesTool, path string) error {
	return validateCNAnthropicResponseTool(tool, path, "DeepSeek")
}

func validateCNAnthropicResponseTool(tool apicompat.ResponsesTool, path, provider string) error {
	provider = normalizeCNAnthropicProviderLabel(provider)
	if strings.ToLower(strings.TrimSpace(tool.Type)) != "function" {
		return fmt.Errorf("unsupported tool type %q at %s: %s Anthropic cross-protocol supports only function tools", tool.Type, path, provider)
	}
	if strings.TrimSpace(tool.Name) == "" {
		return fmt.Errorf("invalid function tool at %s: name is required", path)
	}
	return nil
}

func normalizeCNAnthropicProviderLabel(provider string) string {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return "CN provider"
	}
	return provider
}

func cnAnthropicProviderLabel(platform string) string {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case PlatformZhipu:
		return "Zhipu"
	case PlatformDeepseek:
		return "DeepSeek"
	default:
		return "CN provider"
	}
}

// bytesTrimSpaceForCrossValidation is kept local to avoid coupling this guard
// to the unexported byte helpers in apicompat.
func bytesTrimSpaceForCrossValidation(raw json.RawMessage) []byte {
	return []byte(strings.TrimSpace(string(raw)))
}

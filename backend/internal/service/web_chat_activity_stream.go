package service

import (
	"encoding/json"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
)

const webChatActivitySource = "openai_responses"

// webChatActivityEnvelope is private to /api/v1/chat/completions. Public
// /v1/chat/completions clients must continue to receive only compatible Chat
// Completions chunks.
type webChatActivityEnvelope struct {
	Source    string          `json:"source"`
	EventType string          `json:"eventType"`
	Payload   json.RawMessage `json:"payload"`
}

type webChatActivityReasoning struct {
	Mode   string `json:"mode,omitempty"`
	Effort string `json:"effort,omitempty"`
}

type webChatActivitySummary struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type webChatActivityPart struct {
	Type   string `json:"type"`
	Text   string `json:"text,omitempty"`
	Status string `json:"status,omitempty"`
}

type webChatActivityItem struct {
	ID      string                   `json:"id,omitempty"`
	Type    string                   `json:"type"`
	Status  string                   `json:"status,omitempty"`
	Summary []webChatActivitySummary `json:"summary,omitempty"`
}

type webChatActivityResponse struct {
	ID        string                    `json:"id,omitempty"`
	Status    string                    `json:"status,omitempty"`
	Reasoning *webChatActivityReasoning `json:"reasoning,omitempty"`
	Output    []webChatActivityItem     `json:"output,omitempty"`
}

// webChatActivityPayload deliberately models only fields required by the
// private Activity protocol. json.Unmarshal ignores every other upstream
// field, including encrypted/raw reasoning, answer content, usage and internal
// error details.
type webChatActivityPayload struct {
	Type            string                    `json:"type"`
	ResponseID      string                    `json:"response_id,omitempty"`
	ItemID          string                    `json:"item_id,omitempty"`
	OutputIndex     *int                      `json:"output_index,omitempty"`
	SummaryIndex    *int                      `json:"summary_index,omitempty"`
	SequenceNumber  *int64                    `json:"sequence_number,omitempty"`
	Delta           string                    `json:"delta,omitempty"`
	Text            string                    `json:"text,omitempty"`
	Status          string                    `json:"status,omitempty"`
	Part            *webChatActivityPart      `json:"part,omitempty"`
	Item            *webChatActivityItem      `json:"item,omitempty"`
	Response        *webChatActivityResponse  `json:"response,omitempty"`
	Reasoning       *webChatActivityReasoning `json:"reasoning,omitempty"`
	ReasoningMode   string                    `json:"reasoning_mode,omitempty"`
	ReasoningEffort string                    `json:"reasoning_effort,omitempty"`
}

// buildWebChatActivitySSE preserves official event names and Activity fields
// inside a small private envelope. It never forwards the complete upstream
// object to the browser.
func buildWebChatActivitySSE(
	payload []byte,
	event *apicompat.ResponsesStreamEvent,
) (string, bool) {
	if event == nil || !json.Valid(payload) || !isWebChatActivityEvent(event) {
		return "", false
	}
	safePayload, ok := sanitizeWebChatActivityPayload(payload, strings.TrimSpace(event.Type))
	if !ok {
		return "", false
	}
	envelope, err := json.Marshal(webChatActivityEnvelope{
		Source:    webChatActivitySource,
		EventType: strings.TrimSpace(event.Type),
		Payload:   safePayload,
	})
	if err != nil {
		return "", false
	}
	return "data: " + string(envelope) + "\n\n", true
}

func sanitizeWebChatActivityPayload(payload []byte, eventType string) (json.RawMessage, bool) {
	var decoded webChatActivityPayload
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, false
	}
	safe := webChatActivityPayload{
		Type:            eventType,
		ResponseID:      decoded.ResponseID,
		SequenceNumber:  validWebChatActivityIndex64(decoded.SequenceNumber),
		Reasoning:       sanitizeWebChatActivityReasoning(decoded.Reasoning),
		ReasoningMode:   decoded.ReasoningMode,
		ReasoningEffort: decoded.ReasoningEffort,
	}

	switch eventType {
	case "response.created":
		safe.Response = sanitizeWebChatActivityResponse(decoded.Response, false)
	case "response.output_item.added", "response.output_item.done":
		safe.OutputIndex = validWebChatActivityIndex(decoded.OutputIndex)
		safe.Status = decoded.Status
		safe.Item = sanitizeWebChatActivityItem(decoded.Item)
		if safe.Item == nil {
			return nil, false
		}
	case "response.reasoning_summary_part.added", "response.reasoning_summary_part.done":
		copyWebChatSummaryCoordinates(&safe, &decoded)
		safe.Status = decoded.Status
		safe.Part = sanitizeWebChatActivityPart(decoded.Part)
		if safe.Part == nil {
			return nil, false
		}
	case "response.reasoning_summary_text.delta":
		copyWebChatSummaryCoordinates(&safe, &decoded)
		safe.Delta = decoded.Delta
		safe.Status = decoded.Status
	case "response.reasoning_summary_text.done":
		copyWebChatSummaryCoordinates(&safe, &decoded)
		safe.Text = decoded.Text
		safe.Status = decoded.Status
	case "response.completed", "response.incomplete", "response.failed":
		safe.Status = decoded.Status
		safe.Response = sanitizeWebChatActivityResponse(decoded.Response, true)
	default:
		return nil, false
	}

	marshaled, err := json.Marshal(safe)
	if err != nil {
		return nil, false
	}
	return json.RawMessage(marshaled), true
}

func copyWebChatSummaryCoordinates(
	target *webChatActivityPayload,
	source *webChatActivityPayload,
) {
	if target == nil || source == nil {
		return
	}
	target.ItemID = source.ItemID
	target.OutputIndex = validWebChatActivityIndex(source.OutputIndex)
	target.SummaryIndex = validWebChatActivityIndex(source.SummaryIndex)
}

func validWebChatActivityIndex(value *int) *int {
	if value == nil || *value < 0 {
		return nil
	}
	copy := *value
	return &copy
}

func validWebChatActivityIndex64(value *int64) *int64 {
	if value == nil || *value < 0 {
		return nil
	}
	copy := *value
	return &copy
}

func sanitizeWebChatActivityReasoning(
	reasoning *webChatActivityReasoning,
) *webChatActivityReasoning {
	if reasoning == nil {
		return nil
	}
	return &webChatActivityReasoning{
		Mode:   reasoning.Mode,
		Effort: reasoning.Effort,
	}
}

func sanitizeWebChatActivityPart(part *webChatActivityPart) *webChatActivityPart {
	if part == nil || strings.TrimSpace(part.Type) != "summary_text" {
		return nil
	}
	return &webChatActivityPart{
		Type:   "summary_text",
		Text:   part.Text,
		Status: part.Status,
	}
}

func sanitizeWebChatActivityItem(item *webChatActivityItem) *webChatActivityItem {
	if item == nil || strings.TrimSpace(item.Type) != "reasoning" {
		return nil
	}
	safe := &webChatActivityItem{
		ID:     item.ID,
		Type:   "reasoning",
		Status: item.Status,
	}
	for _, summary := range item.Summary {
		if strings.TrimSpace(summary.Type) != "summary_text" {
			continue
		}
		safe.Summary = append(safe.Summary, webChatActivitySummary{
			Type: "summary_text",
			Text: summary.Text,
		})
	}
	return safe
}

func sanitizeWebChatActivityResponse(
	response *webChatActivityResponse,
	includeOutput bool,
) *webChatActivityResponse {
	if response == nil {
		return nil
	}
	safe := &webChatActivityResponse{
		ID:        response.ID,
		Status:    response.Status,
		Reasoning: sanitizeWebChatActivityReasoning(response.Reasoning),
	}
	if includeOutput {
		for i := range response.Output {
			item := sanitizeWebChatActivityItem(&response.Output[i])
			if item == nil {
				// Preserve the official output array position without exposing the
				// answer/tool payload. Both parsers use this position as output_index.
				safe.Output = append(safe.Output, webChatActivityItem{Type: "redacted"})
				continue
			}
			safe.Output = append(safe.Output, *item)
		}
	}
	return safe
}

func isWebChatActivityEvent(event *apicompat.ResponsesStreamEvent) bool {
	switch strings.TrimSpace(event.Type) {
	case "response.created",
		"response.reasoning_summary_text.delta",
		"response.reasoning_summary_text.done",
		"response.completed",
		"response.incomplete",
		"response.failed":
		return true
	case "response.reasoning_summary_part.added", "response.reasoning_summary_part.done":
		return event.Part != nil && strings.TrimSpace(event.Part.Type) == "summary_text"
	case "response.output_item.added", "response.output_item.done":
		return event.Item != nil && strings.TrimSpace(event.Item.Type) == "reasoning"
	default:
		return false
	}
}

func webChatSuppressConvertedReasoning(event *apicompat.ResponsesStreamEvent) bool {
	if event == nil {
		return false
	}
	switch strings.TrimSpace(event.Type) {
	case "response.reasoning_summary_text.delta", "response.reasoning_text.delta":
		return true
	default:
		return false
	}
}

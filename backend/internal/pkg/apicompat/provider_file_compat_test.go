package apicompat

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func inlineProviderFile(mediaType string, data []byte) string {
	return "data:" + mediaType + ";base64," + base64.StdEncoding.EncodeToString(data)
}

func responsesRequestWithFile(t *testing.T, filename, mediaType string, data []byte) *ResponsesRequest {
	t.Helper()
	content, err := json.Marshal([]ResponsesContentPart{{
		Type: "input_file", Filename: filename,
		FileData: inlineProviderFile(mediaType, data),
	}})
	require.NoError(t, err)
	input, err := json.Marshal([]ResponsesInputItem{{Role: "user", Content: content}})
	require.NoError(t, err)
	return &ResponsesRequest{Model: "test-model", Input: input}
}

func TestChatCompletionsToResponses_PreservesSupportedLibraryFilesForOpenAI(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		mediaType string
	}{
		{name: "pdf", filename: "report.pdf", mediaType: "application/pdf"},
		{name: "docx", filename: "report.docx", mediaType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{name: "xlsx", filename: "report.xlsx", mediaType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{name: "pptx", filename: "report.pptx", mediaType: "application/vnd.openxmlformats-officedocument.presentationml.presentation"},
		{name: "txt", filename: "notes.txt", mediaType: "text/plain"},
		{name: "markdown", filename: "notes.md", mediaType: "text/markdown"},
		{name: "csv", filename: "data.csv", mediaType: "text/csv"},
		{name: "json", filename: "data.json", mediaType: "application/json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileData := inlineProviderFile(tt.mediaType, []byte("test payload"))
			content, err := json.Marshal([]ChatContentPart{{
				Type: "file",
				File: &ChatFile{Filename: tt.filename, FileData: fileData},
			}})
			require.NoError(t, err)

			out, err := ChatCompletionsToResponses(&ChatCompletionsRequest{
				Model:    "gpt-5.6",
				Messages: []ChatMessage{{Role: "user", Content: content}},
			})
			require.NoError(t, err)

			var items []ResponsesInputItem
			require.NoError(t, json.Unmarshal(out.Input, &items))
			require.Len(t, items, 1)
			var parts []ResponsesContentPart
			require.NoError(t, json.Unmarshal(items[0].Content, &parts))
			require.Equal(t, []ResponsesContentPart{{
				Type: "input_file", Filename: tt.filename, FileData: fileData,
			}}, parts)
		})
	}
}

func TestResponsesToAnthropicRequest_ConvertsPDFAndTextLibraryFiles(t *testing.T) {
	tests := []struct {
		name           string
		filename       string
		mediaType      string
		data           []byte
		wantSourceType string
		wantMediaType  string
		wantData       string
	}{
		{
			name: "pdf", filename: "report.pdf", mediaType: "application/pdf",
			data: []byte("%PDF-test"), wantSourceType: "base64",
			wantMediaType: "application/pdf", wantData: base64.StdEncoding.EncodeToString([]byte("%PDF-test")),
		},
		{name: "txt", filename: "notes.txt", mediaType: "text/plain", data: []byte("plain"), wantSourceType: "text", wantMediaType: "text/plain", wantData: "plain"},
		{name: "markdown", filename: "notes.md", mediaType: "text/markdown", data: []byte("# heading"), wantSourceType: "text", wantMediaType: "text/plain", wantData: "# heading"},
		{name: "csv", filename: "data.csv", mediaType: "text/csv", data: []byte("a,b\n1,2"), wantSourceType: "text", wantMediaType: "text/plain", wantData: "a,b\n1,2"},
		{name: "json", filename: "data.json", mediaType: "application/json", data: []byte(`{"ok":true}`), wantSourceType: "text", wantMediaType: "text/plain", wantData: `{"ok":true}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := ResponsesToAnthropicRequest(responsesRequestWithFile(t, tt.filename, tt.mediaType, tt.data))
			require.NoError(t, err)
			require.Len(t, out.Messages, 1)

			var blocks []AnthropicContentBlock
			require.NoError(t, json.Unmarshal(out.Messages[0].Content, &blocks))
			require.Len(t, blocks, 1)
			require.Equal(t, "document", blocks[0].Type)
			require.Equal(t, tt.filename, blocks[0].Title)
			require.NotNil(t, blocks[0].Source)
			require.Equal(t, tt.wantSourceType, blocks[0].Source.Type)
			require.Equal(t, tt.wantMediaType, blocks[0].Source.MediaType)
			require.Equal(t, tt.wantData, blocks[0].Source.Data)
		})
	}
}

func TestResponsesToAnthropicRequest_RejectsOfficeFilesWithoutDroppingThem(t *testing.T) {
	tests := []struct {
		filename  string
		mediaType string
	}{
		{filename: "report.docx", mediaType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{filename: "report.xlsx", mediaType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{filename: "report.pptx", mediaType: "application/vnd.openxmlformats-officedocument.presentationml.presentation"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			out, err := ResponsesToAnthropicRequest(responsesRequestWithFile(t, tt.filename, tt.mediaType, []byte("PK")))
			require.Nil(t, out)
			require.Error(t, err)
			require.True(t, IsProviderFileCompatibilityError(err))
			require.True(t, strings.HasPrefix(err.Error(), ProviderFileUnsupportedCode+":"), err.Error())
			require.Contains(t, err.Error(), tt.filename)
			require.Contains(t, err.Error(), "convert the file to PDF or UTF-8 text")
			if tt.filename == "report.docx" {
				require.Equal(t,
					`provider_file_unsupported: selected provider route cannot accept file "report.docx": DOCX, XLSX, and PPTX cannot be represented on this provider route; convert the file to PDF or UTF-8 text before retrying`,
					err.Error(),
				)
			}
		})
	}
}

func TestResponsesToAnthropicRequest_RejectsProviderFileIDAndMalformedData(t *testing.T) {
	t.Run("provider file id", func(t *testing.T) {
		content, err := json.Marshal([]ResponsesContentPart{{
			Type: "input_file", Filename: "report.pdf", FileID: "file_openai_123",
		}})
		require.NoError(t, err)
		input, err := json.Marshal([]ResponsesInputItem{{Role: "user", Content: content}})
		require.NoError(t, err)

		_, err = ResponsesToAnthropicRequest(&ResponsesRequest{Model: "test", Input: input})
		require.Error(t, err)
		require.True(t, IsProviderFileCompatibilityError(err))
		require.Contains(t, err.Error(), ProviderFileUnsupportedCode)
		require.Contains(t, err.Error(), "file_id")
	})

	t.Run("malformed inline data", func(t *testing.T) {
		content, err := json.Marshal([]ResponsesContentPart{{
			Type: "input_file", Filename: "report.pdf", FileData: "not-a-data-uri",
		}})
		require.NoError(t, err)
		input, err := json.Marshal([]ResponsesInputItem{{Role: "user", Content: content}})
		require.NoError(t, err)

		_, err = ResponsesToAnthropicRequest(&ResponsesRequest{Model: "test", Input: input})
		require.Error(t, err)
		require.True(t, IsProviderFileCompatibilityError(err))
		require.Contains(t, err.Error(), ProviderFileInvalidCode)
		require.Contains(t, err.Error(), "base64 data URI")
	})
}

func TestResponsesToAnthropicRequest_TopLevelInputFileIsNotDropped(t *testing.T) {
	input, err := json.Marshal([]ResponsesInputItem{{
		Type: "input_file", Filename: "notes.txt",
		FileData: inlineProviderFile("text/plain", []byte("hello")),
	}})
	require.NoError(t, err)

	out, err := ResponsesToAnthropicRequest(&ResponsesRequest{Model: "test", Input: input})
	require.NoError(t, err)
	require.Len(t, out.Messages, 1)
	var blocks []AnthropicContentBlock
	require.NoError(t, json.Unmarshal(out.Messages[0].Content, &blocks))
	require.Len(t, blocks, 1)
	require.Equal(t, "document", blocks[0].Type)
	require.Equal(t, "hello", blocks[0].Source.Data)
}

func TestResponsesToAnthropicRequest_FileInNonUserRoleIsRejected(t *testing.T) {
	for _, role := range []string{"system", "developer", "assistant"} {
		t.Run(role, func(t *testing.T) {
			content, err := json.Marshal([]ResponsesContentPart{{
				Type: "input_file", Filename: "notes.txt",
				FileData: inlineProviderFile("text/plain", []byte("hello")),
			}})
			require.NoError(t, err)
			input, err := json.Marshal([]ResponsesInputItem{{Role: role, Content: content}})
			require.NoError(t, err)

			out, err := ResponsesToAnthropicRequest(&ResponsesRequest{Model: "test", Input: input})
			require.Nil(t, out)
			require.Error(t, err)
			require.True(t, IsProviderFileCompatibilityError(err))
			require.Contains(t, err.Error(), "only be represented in user messages")
		})
	}
}

func TestAnthropicToResponses_DocumentIsConvertedToInputFile(t *testing.T) {
	content, err := json.Marshal([]AnthropicContentBlock{
		{
			Type: "document", Title: "report.pdf",
			Source: &AnthropicContentSource{
				Type: "base64", MediaType: "application/pdf", Data: "JVBERi0=",
			},
		},
		{
			Type: "document", Title: "notes.md",
			Source: &AnthropicContentSource{
				Type: "text", MediaType: "text/plain", Data: "# heading",
			},
		},
	})
	require.NoError(t, err)

	out, err := AnthropicToResponses(&AnthropicRequest{
		Model: "gpt-5.6", MaxTokens: 1024,
		Messages: []AnthropicMessage{{Role: "user", Content: content}},
	})
	require.NoError(t, err)

	var items []ResponsesInputItem
	require.NoError(t, json.Unmarshal(out.Input, &items))
	require.Len(t, items, 1)
	var parts []ResponsesContentPart
	require.NoError(t, json.Unmarshal(items[0].Content, &parts))
	require.Len(t, parts, 2)
	require.Equal(t, "input_file", parts[0].Type)
	require.Equal(t, "report.pdf", parts[0].Filename)
	require.Equal(t, "data:application/pdf;base64,JVBERi0=", parts[0].FileData)
	require.Equal(t, "input_file", parts[1].Type)
	require.Equal(t, "notes.md", parts[1].Filename)
	require.Equal(t, inlineProviderFile("text/plain", []byte("# heading")), parts[1].FileData)
}

func TestAnthropicToResponses_ProviderFileIDIsRejectedInsteadOfDropped(t *testing.T) {
	content, err := json.Marshal([]AnthropicContentBlock{{
		Type: "document", Title: "report.pdf",
		Source: &AnthropicContentSource{Type: "file", FileID: "file_anthropic_123"},
	}})
	require.NoError(t, err)

	out, err := AnthropicToResponses(&AnthropicRequest{
		Model: "gpt-5.6", MaxTokens: 1024,
		Messages: []AnthropicMessage{{Role: "user", Content: content}},
	})
	require.Nil(t, out)
	require.Error(t, err)
	require.True(t, IsProviderFileCompatibilityError(err))
	require.Contains(t, err.Error(), ProviderFileUnsupportedCode)
	require.Contains(t, err.Error(), "file_id")
}

func TestAnthropicToResponses_ToolResultDocumentIsLiftedIntoUserInput(t *testing.T) {
	toolResultContent, err := json.Marshal([]AnthropicContentBlock{
		{Type: "text", Text: "tool summary"},
		{
			Type: "document", Title: "report.pdf",
			Source: &AnthropicContentSource{
				Type: "base64", MediaType: "application/pdf", Data: "JVBERi0=",
			},
		},
	})
	require.NoError(t, err)
	messageContent, err := json.Marshal([]AnthropicContentBlock{{
		Type: "tool_result", ToolUseID: "toolu_123", Content: toolResultContent,
	}})
	require.NoError(t, err)

	out, err := AnthropicToResponses(&AnthropicRequest{
		Model: "gpt-5.6", MaxTokens: 1024,
		Messages: []AnthropicMessage{{Role: "user", Content: messageContent}},
	})
	require.NoError(t, err)

	var items []ResponsesInputItem
	require.NoError(t, json.Unmarshal(out.Input, &items))
	require.Len(t, items, 2)
	require.Equal(t, "function_call_output", items[0].Type)
	require.Equal(t, "tool summary", items[0].Output)
	require.Equal(t, "message", items[1].Type)
	require.Equal(t, "user", items[1].Role)
	var parts []ResponsesContentPart
	require.NoError(t, json.Unmarshal(items[1].Content, &parts))
	require.Equal(t, []ResponsesContentPart{{
		Type: "input_file", Filename: "report.pdf",
		FileData: "data:application/pdf;base64,JVBERi0=",
	}}, parts)
}

func TestAnthropicToResponses_DocumentInNonUserRoleIsRejected(t *testing.T) {
	document := AnthropicContentBlock{
		Type: "document", Title: "report.pdf",
		Source: &AnthropicContentSource{
			Type: "base64", MediaType: "application/pdf", Data: "JVBERi0=",
		},
	}
	content, err := json.Marshal([]AnthropicContentBlock{document})
	require.NoError(t, err)

	t.Run("system", func(t *testing.T) {
		out, err := AnthropicToResponses(&AnthropicRequest{
			Model: "gpt-5.6", MaxTokens: 1024, System: content,
		})
		require.Nil(t, out)
		require.Error(t, err)
		require.True(t, IsProviderFileCompatibilityError(err))
		require.Contains(t, err.Error(), "system role")
	})

	t.Run("assistant", func(t *testing.T) {
		out, err := AnthropicToResponses(&AnthropicRequest{
			Model: "gpt-5.6", MaxTokens: 1024,
			Messages: []AnthropicMessage{{Role: "assistant", Content: content}},
		})
		require.Nil(t, out)
		require.Error(t, err)
		require.True(t, IsProviderFileCompatibilityError(err))
		require.Contains(t, err.Error(), "assistant role")
	})
}

func TestAnthropicToChatCompletions_DocumentIsNotDroppedOnDirectFallback(t *testing.T) {
	content, err := json.Marshal([]AnthropicContentBlock{{
		Type: "document", Title: "report.pdf",
		Source: &AnthropicContentSource{
			Type: "base64", MediaType: "application/pdf", Data: "JVBERi0=",
		},
	}})
	require.NoError(t, err)

	out, err := AnthropicToChatCompletionsRequest(&AnthropicRequest{
		Model: "openai-compatible", MaxTokens: 1024,
		Messages: []AnthropicMessage{{Role: "user", Content: content}},
	})
	require.NoError(t, err)
	require.Len(t, out.Messages, 1)
	var parts []ChatContentPart
	require.NoError(t, json.Unmarshal(out.Messages[0].Content, &parts))
	require.Len(t, parts, 1)
	require.Equal(t, "file", parts[0].Type)
	require.NotNil(t, parts[0].File)
	require.Equal(t, "report.pdf", parts[0].File.Filename)
	require.Equal(t, "data:application/pdf;base64,JVBERi0=", parts[0].File.FileData)
}

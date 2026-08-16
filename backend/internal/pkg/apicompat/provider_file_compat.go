package apicompat

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/url"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	// ProviderFileUnsupportedCode identifies a valid file that cannot be
	// represented on the selected provider protocol.
	ProviderFileUnsupportedCode = "provider_file_unsupported"
	// ProviderFileInvalidCode identifies a malformed or incomplete inline file.
	ProviderFileInvalidCode = "provider_file_invalid"
)

// ProviderFileCompatibilityError is returned before an upstream request is
// sent whenever converting a file would otherwise lose it or change its
// meaning. Its code and string shape are part of the client-facing contract.
type ProviderFileCompatibilityError struct {
	Code     string
	Provider string
	Filename string
	Reason   string
}

func (e *ProviderFileCompatibilityError) Error() string {
	if e == nil {
		return ProviderFileUnsupportedCode + ": file cannot be represented on the selected provider route"
	}
	provider := strings.TrimSpace(e.Provider)
	if provider == "" {
		provider = "selected provider route"
	}
	filename := strings.TrimSpace(e.Filename)
	if filename == "" {
		filename = "unnamed file"
	}
	return fmt.Sprintf("%s: %s cannot accept file %q: %s", e.Code, provider, filename, e.Reason)
}

// IsProviderFileCompatibilityError reports whether err is a stable file
// compatibility failure suitable for returning to the client as HTTP 400.
func IsProviderFileCompatibilityError(err error) bool {
	var target *ProviderFileCompatibilityError
	return errors.As(err, &target)
}

// NewProviderFileUnsupportedError creates a stable unsupported-file error.
func NewProviderFileUnsupportedError(provider, filename, reason string) error {
	return &ProviderFileCompatibilityError{
		Code: ProviderFileUnsupportedCode, Provider: provider,
		Filename: filename, Reason: reason,
	}
}

// NewProviderFileInvalidError creates a stable malformed-file error.
func NewProviderFileInvalidError(provider, filename, reason string) error {
	return &ProviderFileCompatibilityError{
		Code: ProviderFileInvalidCode, Provider: provider,
		Filename: filename, Reason: reason,
	}
}

type providerFileKind int

const (
	providerFileUnknown providerFileKind = iota
	providerFilePDF
	providerFileText
	providerFileOffice
)

var providerFileMIMEs = map[string]providerFileKind{
	"application/pdf": providerFilePDF,

	"text/plain":       providerFileText,
	"text/markdown":    providerFileText,
	"text/x-markdown":  providerFileText,
	"text/csv":         providerFileText,
	"application/csv":  providerFileText,
	"application/json": providerFileText,
	"text/json":        providerFileText,

	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   providerFileOffice,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         providerFileOffice,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": providerFileOffice,
}

func providerFileKindFor(filename, mediaType string) (providerFileKind, error) {
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	kind := providerFileMIMEs[mediaType]

	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
	var extKind providerFileKind
	switch ext {
	case ".pdf":
		extKind = providerFilePDF
	case ".txt", ".md", ".csv", ".json":
		extKind = providerFileText
	case ".docx", ".xlsx", ".pptx":
		extKind = providerFileOffice
	}

	if kind != providerFileUnknown && extKind != providerFileUnknown && kind != extKind {
		return providerFileUnknown, fmt.Errorf("filename extension %s does not match media type %s", ext, mediaType)
	}
	if kind != providerFileUnknown {
		return kind, nil
	}
	return extKind, nil
}

func firstResponsesFilePart(raw json.RawMessage) (ResponsesContentPart, bool) {
	var rawParts []json.RawMessage
	if err := json.Unmarshal(raw, &rawParts); err != nil {
		return ResponsesContentPart{}, false
	}
	for _, rawPart := range rawParts {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(rawPart, &fields); err != nil {
			continue
		}
		partType := rawString(fields["type"])
		if partType == "input_file" || partType == "file" {
			return ResponsesContentPart{
				Type: partType, Filename: rawString(fields["filename"]),
				FileData: rawString(fields["file_data"]), FileID: rawString(fields["file_id"]),
				FileURL: rawString(fields["file_url"]),
			}, true
		}
	}
	return ResponsesContentPart{}, false
}

// responsesFileToAnthropicBlock converts an OpenAI input_file into an
// Anthropic document block. Anthropic can natively consume PDF and inline
// plain-text documents. OOXML must be converted by the caller first.
func responsesFileToAnthropicBlock(part ResponsesContentPart) (AnthropicContentBlock, error) {
	const provider = "selected provider route"
	filename := normalizedProviderFilename(part.Filename)

	if strings.TrimSpace(part.FileID) != "" {
		return AnthropicContentBlock{}, NewProviderFileUnsupportedError(
			provider, filename,
			"provider-specific OpenAI file_id references cannot be resolved; send inline file_data instead",
		)
	}

	if fileURL := strings.TrimSpace(part.FileURL); fileURL != "" {
		parsed, err := url.Parse(fileURL)
		if err != nil || !strings.EqualFold(parsed.Scheme, "https") || parsed.Host == "" {
			return AnthropicContentBlock{}, NewProviderFileInvalidError(provider, filename, "file_url must be an absolute HTTPS URL")
		}
		urlName := filename
		if urlName == "" {
			urlName = normalizedProviderFilename(filepath.Base(parsed.Path))
		}
		kind, kindErr := providerFileKindFor(urlName, "")
		if kindErr != nil {
			return AnthropicContentBlock{}, NewProviderFileInvalidError(provider, filename, kindErr.Error())
		}
		if kind != providerFilePDF {
			return AnthropicContentBlock{}, NewProviderFileUnsupportedError(provider, urlName, "only PDF file_url inputs can be represented on this provider route")
		}
		return AnthropicContentBlock{
			Type: "document", Title: providerDocumentTitle(urlName, ".pdf"),
			Source: &AnthropicContentSource{Type: "url", URL: fileURL},
		}, nil
	}

	mediaType, payload, err := parseProviderFileDataURI(part.FileData)
	if err != nil {
		return AnthropicContentBlock{}, NewProviderFileInvalidError(provider, filename, err.Error())
	}
	kind, err := providerFileKindFor(filename, mediaType)
	if err != nil {
		return AnthropicContentBlock{}, NewProviderFileInvalidError(provider, filename, err.Error())
	}

	switch kind {
	case providerFilePDF:
		if mediaType != "application/pdf" {
			return AnthropicContentBlock{}, NewProviderFileInvalidError(provider, filename, "PDF file_data must use application/pdf")
		}
		if !validProviderBase64(payload) {
			return AnthropicContentBlock{}, NewProviderFileInvalidError(provider, filename, "inline file_data contains invalid base64 data")
		}
		return AnthropicContentBlock{
			Type: "document", Title: providerDocumentTitle(filename, ".pdf"),
			Source: &AnthropicContentSource{
				Type: "base64", MediaType: "application/pdf",
				Data: payload,
			},
		}, nil
	case providerFileText:
		data, decodeErr := decodeProviderBase64(payload)
		if decodeErr != nil || len(data) == 0 {
			return AnthropicContentBlock{}, NewProviderFileInvalidError(provider, filename, "inline file_data contains invalid or empty base64 data")
		}
		if !utf8.Valid(data) {
			return AnthropicContentBlock{}, NewProviderFileInvalidError(provider, filename, "text file_data must be valid UTF-8")
		}
		return AnthropicContentBlock{
			Type: "document", Title: providerDocumentTitle(filename, ".txt"),
			Source: &AnthropicContentSource{
				Type: "text", MediaType: "text/plain", Data: string(data),
			},
		}, nil
	case providerFileOffice:
		return AnthropicContentBlock{}, NewProviderFileUnsupportedError(
			provider, filename,
			"DOCX, XLSX, and PPTX cannot be represented on this provider route; convert the file to PDF or UTF-8 text before retrying",
		)
	default:
		return AnthropicContentBlock{}, NewProviderFileUnsupportedError(
			provider, filename,
			fmt.Sprintf("media type %q is not supported; use PDF, TXT, Markdown, CSV, or JSON", mediaType),
		)
	}
}

// anthropicDocumentToResponsesPart converts an Anthropic document block into
// an OpenAI input_file. Provider-owned file IDs are deliberately rejected:
// forwarding such an ID to another provider would reference the wrong object.
func anthropicDocumentToResponsesPart(block AnthropicContentBlock) (ResponsesContentPart, error) {
	const provider = "openai"
	filename := normalizedProviderFilename(block.Title)
	if block.Source == nil {
		return ResponsesContentPart{}, NewProviderFileInvalidError(provider, filename, "document source is required")
	}

	sourceType := strings.ToLower(strings.TrimSpace(block.Source.Type))
	mediaType := strings.ToLower(strings.TrimSpace(block.Source.MediaType))
	switch sourceType {
	case "base64":
		if mediaType == "" {
			return ResponsesContentPart{}, NewProviderFileInvalidError(provider, filename, "base64 document media_type is required")
		}
		data := strings.TrimSpace(block.Source.Data)
		if data == "" {
			return ResponsesContentPart{}, NewProviderFileInvalidError(provider, filename, "document data is empty")
		}
		if !validProviderBase64(data) {
			return ResponsesContentPart{}, NewProviderFileInvalidError(provider, filename, "document data is not valid base64")
		}
		kind, kindErr := providerFileKindFor(filename, mediaType)
		if kindErr != nil {
			return ResponsesContentPart{}, NewProviderFileInvalidError(provider, filename, kindErr.Error())
		}
		if kind == providerFileUnknown {
			return ResponsesContentPart{}, NewProviderFileUnsupportedError(provider, filename, fmt.Sprintf("document media type %q is not supported", mediaType))
		}
		if filename == "" {
			filename = providerFilenameForMediaType(mediaType)
		}
		return ResponsesContentPart{
			Type: "input_file", Filename: filename,
			FileData: "data:" + mediaType + ";base64," + data,
		}, nil

	case "text":
		if mediaType != "" && mediaType != "text/plain" {
			return ResponsesContentPart{}, NewProviderFileUnsupportedError(provider, filename, fmt.Sprintf("text document media type %q is not supported", mediaType))
		}
		if block.Source.Data == "" {
			return ResponsesContentPart{}, NewProviderFileInvalidError(provider, filename, "text document data is empty")
		}
		if filename == "" {
			filename = "document.txt"
		}
		return ResponsesContentPart{
			Type: "input_file", Filename: filename,
			FileData: inlineProviderTextDataURI(block.Source.Data),
		}, nil

	case "url":
		fileURL := strings.TrimSpace(block.Source.URL)
		parsed, err := url.Parse(fileURL)
		if err != nil || !strings.EqualFold(parsed.Scheme, "https") || parsed.Host == "" {
			return ResponsesContentPart{}, NewProviderFileInvalidError(provider, filename, "document URL must be an absolute HTTPS URL")
		}
		return ResponsesContentPart{Type: "input_file", Filename: filename, FileURL: fileURL}, nil

	case "file":
		return ResponsesContentPart{}, NewProviderFileUnsupportedError(
			provider, filename,
			"provider-specific Anthropic file_id references cannot be resolved; send inline document data instead",
		)
	default:
		return ResponsesContentPart{}, NewProviderFileUnsupportedError(provider, filename, fmt.Sprintf("document source type %q is not supported", sourceType))
	}
}

func inlineProviderTextDataURI(text string) string {
	return "data:text/plain;base64," + base64.StdEncoding.EncodeToString([]byte(text))
}

func providerFilenameForMediaType(mediaType string) string {
	switch strings.ToLower(strings.TrimSpace(mediaType)) {
	case "application/pdf":
		return "document.pdf"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return "document.docx"
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return "document.xlsx"
	case "application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return "document.pptx"
	case "text/csv", "application/csv":
		return "document.csv"
	case "application/json", "text/json":
		return "document.json"
	case "text/markdown", "text/x-markdown":
		return "document.md"
	default:
		return "document.txt"
	}
}

func parseProviderFileDataURI(raw string) (string, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", errors.New("inline file_data is required")
	}
	if !strings.HasPrefix(strings.ToLower(raw), "data:") {
		return "", "", errors.New("inline file_data must be a base64 data URI")
	}
	comma := strings.IndexByte(raw, ',')
	if comma < 0 {
		return "", "", errors.New("inline file_data is not a valid data URI")
	}
	header := raw[len("data:"):comma]
	if !strings.HasSuffix(strings.ToLower(header), ";base64") {
		return "", "", errors.New("inline file_data must use base64 encoding")
	}
	header = header[:len(header)-len(";base64")]
	mediaType, _, err := mime.ParseMediaType(header)
	if err != nil || strings.TrimSpace(mediaType) == "" {
		return "", "", errors.New("inline file_data has an invalid media type")
	}

	payload := strings.TrimSpace(raw[comma+1:])
	if payload == "" {
		return "", "", errors.New("inline file_data is empty")
	}
	return strings.ToLower(mediaType), payload, nil
}

func decodeProviderBase64(payload string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err == nil {
		return decoded, nil
	}
	return base64.RawStdEncoding.DecodeString(payload)
}

func validProviderBase64(payload string) bool {
	validate := func(encoding *base64.Encoding) bool {
		reader := base64.NewDecoder(encoding, strings.NewReader(payload))
		written, err := io.Copy(io.Discard, reader)
		return err == nil && written > 0
	}
	return validate(base64.StdEncoding) || validate(base64.RawStdEncoding)
}

func normalizedProviderFilename(name string) string {
	name = strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))
	name = filepath.Base(name)
	if name == "." || name == "/" {
		return ""
	}
	return name
}

func providerDocumentTitle(filename, fallbackExt string) string {
	filename = normalizedProviderFilename(filename)
	if filename == "" {
		return "document" + fallbackExt
	}
	return filename
}

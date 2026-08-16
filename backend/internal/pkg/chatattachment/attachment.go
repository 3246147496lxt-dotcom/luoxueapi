package chatattachment

import (
	"bytes"
	"errors"
	"mime"
	"path/filepath"
	"strings"
)

const (
	MaxImageDimension                = 8192
	MaxImagePixels             int64 = 25_000_000
	MaxExtractedTextCharacters       = 50_000
	MaxDOCXEntries                   = 2048
	MaxDOCXUncompressedBytes         = 50 << 20
)

type Kind string

const (
	KindImage Kind = "image"
	KindFile  Kind = "file"
	KindDOCX  Kind = "docx"
)

type ErrorCode string

const (
	CodeEmptyFile                 ErrorCode = "EMPTY_FILE"
	CodeUnsupportedType           ErrorCode = "UNSUPPORTED_TYPE"
	CodeTypeMismatch              ErrorCode = "TYPE_MISMATCH"
	CodeInvalidImage              ErrorCode = "INVALID_IMAGE"
	CodeImageDimensionsExceeded   ErrorCode = "IMAGE_DIMENSIONS_EXCEEDED"
	CodeTextLimitExceeded         ErrorCode = "TEXT_LIMIT_EXCEEDED"
	CodeInvalidDOCX               ErrorCode = "INVALID_DOCX"
	CodeDOCXMacroForbidden        ErrorCode = "DOCX_MACRO_FORBIDDEN"
	CodeDOCXExternalRelationship  ErrorCode = "DOCX_EXTERNAL_RELATIONSHIP"
	CodeDOCXArchiveLimitExceeded  ErrorCode = "DOCX_ARCHIVE_LIMIT_EXCEEDED"
	CodeInvalidPDF                ErrorCode = "INVALID_PDF"
	CodeInvalidText               ErrorCode = "INVALID_TEXT"
	CodeInvalidJSON               ErrorCode = "INVALID_JSON"
	CodeInvalidOOXML              ErrorCode = "INVALID_OOXML"
	CodeOOXMLMacroForbidden       ErrorCode = "OOXML_MACRO_FORBIDDEN"
	CodeOOXMLExternalRelationship ErrorCode = "OOXML_EXTERNAL_RELATIONSHIP"
	CodeOOXMLArchiveLimitExceeded ErrorCode = "OOXML_ARCHIVE_LIMIT_EXCEEDED"
)

type Error struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func CodeOf(err error) ErrorCode {
	var attachmentErr *Error
	if errors.As(err, &attachmentErr) {
		return attachmentErr.Code
	}
	return ""
}

type Input struct {
	Filename     string
	DeclaredMIME string
	Data         []byte
}

type Result struct {
	Kind          Kind
	MIMEType      string
	Extension     string
	SanitizedData []byte
	Text          string
	Width         int
	Height        int
	PageCount     int
}

type supportedType struct {
	kind      Kind
	mimeType  string
	extension string
	signature func([]byte) bool
}

func Process(input Input) (*Result, error) {
	if len(input.Data) == 0 {
		return nil, attachmentError(CodeEmptyFile, "Attachment file is empty", nil)
	}

	fileType, err := classifyInput(input.Filename, input.DeclaredMIME, input.Data)
	if err != nil {
		return nil, err
	}

	switch fileType.kind {
	case KindImage:
		return processImage(input.Data, fileType)
	case KindDOCX:
		return processDOCX(input.Data)
	default:
		return nil, attachmentError(CodeUnsupportedType, "Attachment type is not supported", nil)
	}
}

func classifyInput(filename, declaredMIME string, data []byte) (supportedType, error) {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
	var fileType supportedType
	switch ext {
	case ".jpg", ".jpeg":
		fileType = supportedType{KindImage, "image/jpeg", ".jpg", hasJPEGSignature}
	case ".png":
		fileType = supportedType{KindImage, "image/png", ".png", hasPNGSignature}
	case ".webp":
		fileType = supportedType{KindImage, "image/webp", ".webp", hasWebPSignature}
	case ".docx":
		fileType = supportedType{
			KindDOCX,
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			".docx",
			hasZIPSignature,
		}
	default:
		return supportedType{}, attachmentError(CodeUnsupportedType, "Attachment type is not supported", nil)
	}

	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(declaredMIME))
	if err != nil || !strings.EqualFold(strings.TrimSpace(mediaType), fileType.mimeType) {
		return supportedType{}, attachmentError(CodeTypeMismatch, "Attachment filename and declared MIME type do not match", err)
	}
	if !fileType.signature(data) {
		return supportedType{}, attachmentError(CodeTypeMismatch, "Attachment content does not match its declared type", nil)
	}
	return fileType, nil
}

func attachmentError(code ErrorCode, message string, cause error) *Error {
	return &Error{Code: code, Message: message, Cause: cause}
}

func hasJPEGSignature(data []byte) bool {
	return len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff
}

func hasPNGSignature(data []byte) bool {
	return len(data) >= 8 && bytes.Equal(data[:8], []byte("\x89PNG\r\n\x1a\n"))
}

func hasWebPSignature(data []byte) bool {
	return len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP"
}

func hasZIPSignature(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	return bytes.Equal(data[:4], []byte("PK\x03\x04")) ||
		bytes.Equal(data[:4], []byte("PK\x05\x06")) ||
		bytes.Equal(data[:4], []byte("PK\x07\x08"))
}

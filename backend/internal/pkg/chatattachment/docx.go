package chatattachment

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"encoding/xml"
	"fmt"
	"hash/crc32"
	"io"
	"math"
	"net/url"
	"path"
	"strings"
)

const (
	docxContentTypesName = "[Content_Types].xml"
	docxDocumentName     = "word/document.xml"

	docxMIME = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	docxMain = "application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"

	wordprocessingMLNamespace       = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
	strictWordprocessingMLNamespace = "http://purl.oclc.org/ooxml/wordprocessingml/main"
	maxDOCXFieldInstructionBytes    = 64 << 10
)

type docxParts struct {
	contentTypes  []byte
	document      []byte
	relationships map[string][]byte
}

func processDOCX(data []byte) (result *Result, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			result = nil
			err = attachmentError(
				CodeInvalidDOCX,
				"DOCX cannot be parsed safely",
				fmt.Errorf("DOCX parser panic: %v", recovered),
			)
		}
	}()

	if len(data) > MaxDOCXUncompressedBytes {
		return nil, attachmentError(CodeDOCXArchiveLimitExceeded, "DOCX archive exceeds the 50 MiB limit", nil)
	}

	archive, zipErr := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if zipErr != nil {
		return nil, attachmentError(CodeInvalidDOCX, "DOCX is not a valid ZIP archive", zipErr)
	}
	if len(archive.File) == 0 {
		return nil, attachmentError(CodeInvalidDOCX, "DOCX archive is empty", nil)
	}
	if len(archive.File) > MaxDOCXEntries {
		return nil, attachmentError(CodeDOCXArchiveLimitExceeded, "DOCX archive contains too many entries", nil)
	}

	files, metadataTotal, validationErr := validateDOCXEntries(archive.File)
	if validationErr != nil {
		return nil, validationErr
	}
	if metadataTotal > MaxDOCXUncompressedBytes {
		return nil, attachmentError(CodeDOCXArchiveLimitExceeded, "DOCX uncompressed data exceeds the 50 MiB limit", nil)
	}

	parts, readErr := readDOCXParts(files)
	if readErr != nil {
		return nil, readErr
	}
	if parts.contentTypes == nil || parts.document == nil {
		return nil, attachmentError(CodeInvalidDOCX, "DOCX is missing required package parts", nil)
	}

	if err := validateDOCXContentTypes(parts.contentTypes); err != nil {
		return nil, err
	}
	for name, relationshipXML := range parts.relationships {
		if err := validateDOCXRelationships(name, relationshipXML); err != nil {
			return nil, err
		}
	}

	extracted, extractionErr := extractDOCXText(parts.document)
	if extractionErr != nil {
		return nil, extractionErr
	}
	if extracted.empty() {
		return nil, attachmentError(CodeInvalidDOCX, "DOCX does not contain extractable text", nil)
	}

	return &Result{
		Kind:      KindDOCX,
		MIMEType:  docxMIME,
		Extension: ".docx",
		Text:      extracted.string(),
	}, nil
}

func validateDOCXEntries(entries []*zip.File) (map[string]*zip.File, uint64, error) {
	files := make(map[string]*zip.File, len(entries))
	var total uint64
	for _, file := range entries {
		name, err := validateDOCXEntryName(file.Name, file.FileInfo().IsDir())
		if err != nil {
			return nil, 0, err
		}
		if file.Flags&0x1 != 0 {
			return nil, 0, attachmentError(CodeInvalidDOCX, "Encrypted DOCX ZIP entries are not supported", nil)
		}
		mode := file.Mode()
		if !file.FileInfo().IsDir() && !mode.IsRegular() {
			return nil, 0, attachmentError(CodeInvalidDOCX, "DOCX contains a non-regular ZIP entry", nil)
		}

		lookupName := strings.ToLower(name)
		if _, duplicate := files[lookupName]; duplicate {
			return nil, 0, attachmentError(CodeInvalidDOCX, "DOCX contains duplicate package part names", nil)
		}
		files[lookupName] = file

		if isDOCXMacroPart(lookupName) {
			return nil, 0, attachmentError(CodeDOCXMacroForbidden, "Macro-enabled DOCX content is not supported", nil)
		}

		if file.UncompressedSize64 > MaxDOCXUncompressedBytes || total > math.MaxUint64-file.UncompressedSize64 {
			return nil, 0, attachmentError(CodeDOCXArchiveLimitExceeded, "DOCX uncompressed data exceeds the 50 MiB limit", nil)
		}
		total += file.UncompressedSize64
		if total > MaxDOCXUncompressedBytes {
			return nil, 0, attachmentError(CodeDOCXArchiveLimitExceeded, "DOCX uncompressed data exceeds the 50 MiB limit", nil)
		}
	}
	return files, total, nil
}

func validateDOCXEntryName(name string, directory bool) (string, error) {
	if name == "" || strings.ContainsRune(name, '\x00') || strings.Contains(name, "\\") || strings.HasPrefix(name, "/") {
		return "", attachmentError(CodeInvalidDOCX, "DOCX contains an invalid package part name", nil)
	}
	normalized := name
	if directory {
		normalized = strings.TrimSuffix(normalized, "/")
	}
	if normalized == "" || normalized == "." || path.Clean(normalized) != normalized || strings.HasPrefix(normalized, "../") {
		return "", attachmentError(CodeInvalidDOCX, "DOCX contains an unsafe package part path", nil)
	}
	return normalized, nil
}

func isDOCXMacroPart(lowerName string) bool {
	base := path.Base(lowerName)
	return base == "vbaproject.bin" ||
		strings.Contains(lowerName, "/vba/") ||
		strings.HasSuffix(base, ".vba")
}

func readDOCXParts(files map[string]*zip.File) (docxParts, error) {
	parts := docxParts{relationships: make(map[string][]byte)}
	var actualTotal int64
	for lowerName, file := range files {
		if file.FileInfo().IsDir() {
			continue
		}
		capture := lowerName == strings.ToLower(docxContentTypesName) ||
			lowerName == docxDocumentName || strings.HasSuffix(lowerName, ".rels")
		content, count, err := readDOCXEntry(file, MaxDOCXUncompressedBytes-actualTotal, capture)
		if err != nil {
			return docxParts{}, err
		}
		actualTotal += count
		if actualTotal > MaxDOCXUncompressedBytes {
			return docxParts{}, attachmentError(CodeDOCXArchiveLimitExceeded, "DOCX uncompressed data exceeds the 50 MiB limit", nil)
		}
		if !capture {
			continue
		}
		switch {
		case lowerName == strings.ToLower(docxContentTypesName):
			parts.contentTypes = content
		case lowerName == docxDocumentName:
			parts.document = content
		case strings.HasSuffix(lowerName, ".rels"):
			parts.relationships[lowerName] = content
		}
	}
	return parts, nil
}

func readDOCXEntry(file *zip.File, remaining int64, capture bool) ([]byte, int64, error) {
	if remaining < 0 {
		return nil, 0, attachmentError(CodeDOCXArchiveLimitExceeded, "DOCX uncompressed data exceeds the 50 MiB limit", nil)
	}
	rawReader, err := file.OpenRaw()
	if err != nil {
		return nil, 0, attachmentError(CodeInvalidDOCX, "DOCX ZIP entry cannot be opened", err)
	}
	var reader io.ReadCloser
	switch file.Method {
	case zip.Store:
		reader = io.NopCloser(rawReader)
	case zip.Deflate:
		reader = flate.NewReader(rawReader)
	default:
		return nil, 0, attachmentError(CodeInvalidDOCX, "DOCX uses an unsupported ZIP compression method", nil)
	}

	limited := io.LimitReader(reader, remaining+1)
	checksum := crc32.NewIEEE()
	var destination io.Writer = checksum
	var buffer bytes.Buffer
	if capture {
		destination = io.MultiWriter(&buffer, checksum)
	}
	count, copyErr := io.Copy(destination, limited)
	if count > remaining {
		_ = reader.Close()
		return nil, count, attachmentError(CodeDOCXArchiveLimitExceeded, "DOCX uncompressed data exceeds the 50 MiB limit", nil)
	}
	if copyErr != nil {
		_ = reader.Close()
		return nil, count, attachmentError(CodeInvalidDOCX, "DOCX ZIP entry cannot be decompressed safely", copyErr)
	}
	if closeErr := reader.Close(); closeErr != nil {
		return nil, count, attachmentError(CodeInvalidDOCX, "DOCX ZIP entry failed integrity validation", closeErr)
	}
	if uint64(count) != file.UncompressedSize64 || checksum.Sum32() != file.CRC32 {
		return nil, count, attachmentError(CodeInvalidDOCX, "DOCX ZIP entry failed size or checksum validation", nil)
	}
	return buffer.Bytes(), count, nil
}

func validateDOCXContentTypes(data []byte) error {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Strict = true
	foundDocument := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return attachmentError(CodeInvalidDOCX, "DOCX content types XML is malformed", err)
		}
		switch value := token.(type) {
		case xml.Directive:
			return attachmentError(CodeInvalidDOCX, "DOCX XML directives are not supported", nil)
		case xml.StartElement:
			if value.Name.Local != "Default" && value.Name.Local != "Override" {
				continue
			}
			contentType := xmlAttribute(value.Attr, "ContentType")
			lowerContentType := strings.ToLower(contentType)
			if strings.Contains(lowerContentType, "macroenabled") || strings.Contains(lowerContentType, "vba") {
				return attachmentError(CodeDOCXMacroForbidden, "Macro-enabled DOCX content is not supported", nil)
			}
			if value.Name.Local == "Override" && strings.EqualFold(xmlAttribute(value.Attr, "PartName"), "/"+docxDocumentName) {
				if !strings.EqualFold(contentType, docxMain) {
					return attachmentError(CodeInvalidDOCX, "DOCX main document has an unsupported content type", nil)
				}
				foundDocument = true
			}
		}
	}
	if !foundDocument {
		return attachmentError(CodeInvalidDOCX, "DOCX content types do not declare the main document", nil)
	}
	return nil
}

func validateDOCXRelationships(name string, data []byte) error {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Strict = true
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return attachmentError(CodeInvalidDOCX, "DOCX relationships XML is malformed", fmt.Errorf("%s: %w", name, err))
		}
		switch value := token.(type) {
		case xml.Directive:
			return attachmentError(CodeInvalidDOCX, "DOCX XML directives are not supported", nil)
		case xml.StartElement:
			if value.Name.Local != "Relationship" {
				continue
			}
			targetMode := strings.TrimSpace(xmlAttribute(value.Attr, "TargetMode"))
			target := strings.TrimSpace(xmlAttribute(value.Attr, "Target"))
			if strings.EqualFold(targetMode, "External") || isExternalRelationshipTarget(target) {
				return attachmentError(CodeDOCXExternalRelationship, "DOCX external relationships are not supported", nil)
			}
		}
	}
}

func isExternalRelationshipTarget(target string) bool {
	if target == "" {
		return false
	}
	lowerTarget := strings.ToLower(target)
	if strings.HasPrefix(lowerTarget, "//") || strings.HasPrefix(lowerTarget, "\\\\") || strings.HasPrefix(lowerTarget, "file:") {
		return true
	}
	parsed, err := url.Parse(target)
	return err == nil && parsed.IsAbs()
}

func extractDOCXText(data []byte) (*boundedText, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Strict = true
	extracted := newBoundedText(MaxExtractedTextCharacters)
	inText := 0
	inInstruction := 0
	var instruction strings.Builder

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			if hasExternalFieldInstruction(instruction.String()) {
				return nil, attachmentError(CodeDOCXExternalRelationship, "DOCX external field links are not supported", nil)
			}
			return extracted, nil
		}
		if err != nil {
			return nil, attachmentError(CodeInvalidDOCX, "DOCX document XML is malformed", err)
		}
		switch value := token.(type) {
		case xml.Directive:
			return nil, attachmentError(CodeInvalidDOCX, "DOCX XML directives are not supported", nil)
		case xml.StartElement:
			if !isWordprocessingMLElement(value.Name) {
				continue
			}
			switch value.Name.Local {
			case "t":
				inText++
			case "instrText":
				inInstruction++
			case "tab":
				if !extracted.append("\t") {
					return nil, docxTextLimitError()
				}
			case "br", "cr":
				if !extracted.append("\n") {
					return nil, docxTextLimitError()
				}
			case "fldSimple":
				if hasExternalFieldInstruction(xmlAttribute(value.Attr, "instr")) {
					return nil, attachmentError(CodeDOCXExternalRelationship, "DOCX external field links are not supported", nil)
				}
			case "altChunk":
				return nil, attachmentError(CodeDOCXExternalRelationship, "DOCX linked alternate content is not supported", nil)
			}
		case xml.CharData:
			if inInstruction > 0 {
				if instruction.Len()+len(value) > maxDOCXFieldInstructionBytes {
					return nil, attachmentError(CodeInvalidDOCX, "DOCX field instruction is too large", nil)
				}
				_, _ = instruction.Write([]byte(value))
			}
			if inText > 0 && !extracted.append(string(value)) {
				return nil, docxTextLimitError()
			}
		case xml.EndElement:
			if !isWordprocessingMLElement(value.Name) {
				continue
			}
			switch value.Name.Local {
			case "t":
				if inText > 0 {
					inText--
				}
			case "instrText":
				if inInstruction > 0 {
					inInstruction--
				}
			case "p":
				if hasExternalFieldInstruction(instruction.String()) {
					return nil, attachmentError(CodeDOCXExternalRelationship, "DOCX external field links are not supported", nil)
				}
				instruction.Reset()
				if !extracted.append("\n") {
					return nil, docxTextLimitError()
				}
			case "tc":
				if !extracted.append("\t") {
					return nil, docxTextLimitError()
				}
			}
		}
	}
}

func hasExternalFieldInstruction(instruction string) bool {
	lower := strings.ToLower(strings.TrimSpace(instruction))
	if lower == "" {
		return false
	}
	if strings.Contains(lower, "http:") ||
		strings.Contains(lower, "https:") ||
		strings.Contains(lower, "ftp:") ||
		strings.Contains(lower, "mailto:") ||
		strings.Contains(lower, "file:") ||
		strings.Contains(lower, "\\\\") {
		return true
	}
	for _, command := range []string{"includepicture", "includetext", "link", "dde", "ddeauto", "database"} {
		if strings.HasPrefix(lower, command+" ") || lower == command {
			return true
		}
	}
	if hyperlink := strings.Index(lower, "hyperlink"); hyperlink >= 0 {
		target := strings.TrimSpace(lower[hyperlink+len("hyperlink"):])
		// A field whose only target is a local bookmark is not an external link.
		return target != "" && !strings.HasPrefix(target, `\l`)
	}
	return false
}

func isWordprocessingMLElement(name xml.Name) bool {
	return name.Space == wordprocessingMLNamespace || name.Space == strictWordprocessingMLNamespace
}

func xmlAttribute(attributes []xml.Attr, localName string) string {
	for _, attribute := range attributes {
		if attribute.Name.Local == localName {
			return attribute.Value
		}
	}
	return ""
}

func docxTextLimitError() error {
	return attachmentError(CodeTextLimitExceeded, "Extracted document text exceeds 50000 characters", nil)
}

package chatattachment

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"hash/crc32"
	"io"
	"math"
	"mime"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	MaxOOXMLEntries           = 2048
	MaxOOXMLUncompressedBytes = 50 << 20
	MaxOOXMLXMLDepth          = 256

	ooxmlContentTypesNamespace    = "http://schemas.openxmlformats.org/package/2006/content-types"
	strictContentTypesNamespace   = "http://purl.oclc.org/ooxml/package/content-types"
	ooxmlPackageRelationshipsNS   = "http://schemas.openxmlformats.org/package/2006/relationships"
	strictPackageRelationshipsNS  = "http://purl.oclc.org/ooxml/package/relationships"
	ooxmlOfficeDocumentRelation   = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument"
	strictOfficeDocumentRelation  = "http://purl.oclc.org/ooxml/officeDocument/relationships/officeDocument"
	spreadsheetMLNamespace        = "http://schemas.openxmlformats.org/spreadsheetml/2006/main"
	strictSpreadsheetMLNamespace  = "http://purl.oclc.org/ooxml/spreadsheetml/main"
	presentationMLNamespace       = "http://schemas.openxmlformats.org/presentationml/2006/main"
	strictPresentationMLNamespace = "http://purl.oclc.org/ooxml/presentationml/main"
)

type LibraryFormat string

const (
	LibraryFormatJPEG LibraryFormat = "jpeg"
	LibraryFormatPNG  LibraryFormat = "png"
	LibraryFormatWebP LibraryFormat = "webp"
	LibraryFormatPDF  LibraryFormat = "pdf"
	LibraryFormatDOCX LibraryFormat = "docx"
	LibraryFormatXLSX LibraryFormat = "xlsx"
	LibraryFormatPPTX LibraryFormat = "pptx"
	LibraryFormatTXT  LibraryFormat = "txt"
	LibraryFormatMD   LibraryFormat = "md"
	LibraryFormatCSV  LibraryFormat = "csv"
	LibraryFormatJSON LibraryFormat = "json"
)

type LibraryCategory string

const (
	LibraryCategoryImage LibraryCategory = "image"
	LibraryCategoryFile  LibraryCategory = "file"
)

type LibraryType string

const (
	LibraryTypeImage        LibraryType = "image"
	LibraryTypePDF          LibraryType = "pdf"
	LibraryTypeDocument     LibraryType = "document"
	LibraryTypeSpreadsheet  LibraryType = "spreadsheet"
	LibraryTypePresentation LibraryType = "presentation"
	LibraryTypeOther        LibraryType = "other"
)

// LibraryResult describes a validated library upload. Data aliases the original
// input and must be retained for downloads. Images additionally expose a
// metadata-free SanitizedData rendition suitable for thumbnails and previews.
type LibraryResult struct {
	Kind               Kind
	Format             LibraryFormat
	Category           LibraryCategory
	Type               LibraryType
	MIMEType           string
	Extension          string
	Data               []byte
	SanitizedData      []byte
	SanitizedMIMEType  string
	SanitizedExtension string
	ExtractedText      string
	Width              int
	Height             int
	PageCount          int
}

type librarySupportedType struct {
	format       LibraryFormat
	category     LibraryCategory
	fileType     LibraryType
	mimeType     string
	mimeAliases  []string
	extension    string
	image        bool
	ooxml        *ooxmlPackageType
	validateData func([]byte) error
}

type ooxmlPackageType struct {
	format          LibraryFormat
	mainPart        string
	mainContentType string
	mainRootLocal   string
	mainNamespaces  []string
}

var ooxmlPackageTypes = map[LibraryFormat]ooxmlPackageType{
	LibraryFormatDOCX: {
		format:          LibraryFormatDOCX,
		mainPart:        "word/document.xml",
		mainContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml",
		mainRootLocal:   "document",
		mainNamespaces:  []string{wordprocessingMLNamespace, strictWordprocessingMLNamespace},
	},
	LibraryFormatXLSX: {
		format:          LibraryFormatXLSX,
		mainPart:        "xl/workbook.xml",
		mainContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml",
		mainRootLocal:   "workbook",
		mainNamespaces:  []string{spreadsheetMLNamespace, strictSpreadsheetMLNamespace},
	},
	LibraryFormatPPTX: {
		format:          LibraryFormatPPTX,
		mainPart:        "ppt/presentation.xml",
		mainContentType: "application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml",
		mainRootLocal:   "presentation",
		mainNamespaces:  []string{presentationMLNamespace, strictPresentationMLNamespace},
	},
}

// ProcessLibrary validates and classifies a library upload without extracting
// document content. Upload size and account quota limits remain the caller's
// responsibility; the OOXML decompression limits below are archive-safety bounds.
func ProcessLibrary(input Input) (*LibraryResult, error) {
	if len(input.Data) == 0 {
		return nil, attachmentError(CodeEmptyFile, "Library file is empty", nil)
	}

	fileType, err := classifyLibraryInput(input.Filename, input.DeclaredMIME)
	if err != nil {
		return nil, err
	}

	result := &LibraryResult{
		Kind:      KindFile,
		Format:    fileType.format,
		Category:  fileType.category,
		Type:      fileType.fileType,
		MIMEType:  fileType.mimeType,
		Extension: fileType.extension,
		Data:      input.Data,
	}

	if fileType.image {
		if !libraryImageSignatureMatches(fileType.format, input.Data) {
			return nil, attachmentError(CodeTypeMismatch, "Library file content does not match its declared image type", nil)
		}
		if envelopeErr := validateLibraryImageEnvelope(fileType.format, input.Data); envelopeErr != nil {
			return nil, envelopeErr
		}
		processed, processErr := processImage(input.Data, supportedType{
			kind:      KindImage,
			mimeType:  fileType.mimeType,
			extension: fileType.extension,
		})
		if processErr != nil {
			return nil, processErr
		}
		result.Kind = KindImage
		result.SanitizedData = processed.SanitizedData
		result.SanitizedMIMEType = processed.MIMEType
		result.SanitizedExtension = processed.Extension
		result.Width = processed.Width
		result.Height = processed.Height
		return result, nil
	}

	if fileType.ooxml != nil {
		if err := validateOOXMLPackage(input.Data, *fileType.ooxml); err != nil {
			return nil, err
		}
		return result, nil
	}
	if fileType.validateData != nil {
		if err := fileType.validateData(input.Data); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func classifyLibraryInput(filename, declaredMIME string) (librarySupportedType, error) {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
	fileType, ok := libraryTypeForExtension(ext)
	if !ok {
		return librarySupportedType{}, attachmentError(CodeUnsupportedType, "Library file type is not supported", nil)
	}

	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(declaredMIME))
	if err != nil || !matchesLibraryMIME(mediaType, fileType) {
		return librarySupportedType{}, attachmentError(CodeTypeMismatch, "Library filename and declared MIME type do not match", err)
	}
	return fileType, nil
}

func libraryTypeForExtension(ext string) (librarySupportedType, bool) {
	switch ext {
	case ".jpg", ".jpeg":
		return librarySupportedType{format: LibraryFormatJPEG, category: LibraryCategoryImage, fileType: LibraryTypeImage, mimeType: "image/jpeg", extension: ".jpg", image: true}, true
	case ".png":
		return librarySupportedType{format: LibraryFormatPNG, category: LibraryCategoryImage, fileType: LibraryTypeImage, mimeType: "image/png", extension: ".png", image: true}, true
	case ".webp":
		return librarySupportedType{format: LibraryFormatWebP, category: LibraryCategoryImage, fileType: LibraryTypeImage, mimeType: "image/webp", extension: ".webp", image: true}, true
	case ".pdf":
		return librarySupportedType{format: LibraryFormatPDF, category: LibraryCategoryFile, fileType: LibraryTypePDF, mimeType: "application/pdf", extension: ".pdf", validateData: validatePDF}, true
	case ".docx":
		packageType := ooxmlPackageTypes[LibraryFormatDOCX]
		return librarySupportedType{format: LibraryFormatDOCX, category: LibraryCategoryFile, fileType: LibraryTypeDocument, mimeType: docxMIME, extension: ".docx", ooxml: &packageType}, true
	case ".xlsx":
		packageType := ooxmlPackageTypes[LibraryFormatXLSX]
		return librarySupportedType{format: LibraryFormatXLSX, category: LibraryCategoryFile, fileType: LibraryTypeSpreadsheet, mimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", extension: ".xlsx", ooxml: &packageType}, true
	case ".pptx":
		packageType := ooxmlPackageTypes[LibraryFormatPPTX]
		return librarySupportedType{format: LibraryFormatPPTX, category: LibraryCategoryFile, fileType: LibraryTypePresentation, mimeType: "application/vnd.openxmlformats-officedocument.presentationml.presentation", extension: ".pptx", ooxml: &packageType}, true
	case ".txt":
		return librarySupportedType{format: LibraryFormatTXT, category: LibraryCategoryFile, fileType: LibraryTypeDocument, mimeType: "text/plain", extension: ".txt", validateData: validateUTF8Text}, true
	case ".md":
		return librarySupportedType{format: LibraryFormatMD, category: LibraryCategoryFile, fileType: LibraryTypeDocument, mimeType: "text/markdown", mimeAliases: []string{"text/x-markdown"}, extension: ".md", validateData: validateUTF8Text}, true
	case ".csv":
		return librarySupportedType{format: LibraryFormatCSV, category: LibraryCategoryFile, fileType: LibraryTypeSpreadsheet, mimeType: "text/csv", mimeAliases: []string{"application/csv"}, extension: ".csv", validateData: validateUTF8Text}, true
	case ".json":
		return librarySupportedType{format: LibraryFormatJSON, category: LibraryCategoryFile, fileType: LibraryTypeOther, mimeType: "application/json", mimeAliases: []string{"text/json"}, extension: ".json", validateData: validateJSON}, true
	default:
		return librarySupportedType{}, false
	}
}

func matchesLibraryMIME(mediaType string, fileType librarySupportedType) bool {
	if strings.EqualFold(strings.TrimSpace(mediaType), fileType.mimeType) {
		return true
	}
	for _, alias := range fileType.mimeAliases {
		if strings.EqualFold(strings.TrimSpace(mediaType), alias) {
			return true
		}
	}
	return false
}

func libraryImageSignatureMatches(format LibraryFormat, data []byte) bool {
	switch format {
	case LibraryFormatJPEG:
		return hasJPEGSignature(data)
	case LibraryFormatPNG:
		return hasPNGSignature(data)
	case LibraryFormatWebP:
		return hasWebPSignature(data)
	default:
		return false
	}
}

// validateLibraryImageEnvelope rejects image/polyglot files whose decodable
// image is followed by an unrelated payload. The original bytes are retained
// for downloads, so checking the complete container is separate from the safe
// re-encode used for model input and previews.
func validateLibraryImageEnvelope(format LibraryFormat, data []byte) error {
	var valid bool
	switch format {
	case LibraryFormatJPEG:
		valid = validJPEGEnvelope(data)
	case LibraryFormatPNG:
		valid = validPNGEnvelope(data)
	case LibraryFormatWebP:
		valid = validWebPEnvelope(data)
	}
	if !valid {
		return attachmentError(CodeInvalidImage, "Image container has invalid or trailing data", nil)
	}
	return nil
}

func validJPEGEnvelope(data []byte) bool {
	if !hasJPEGSignature(data) || len(data) < 4 {
		return false
	}
	position := 2
	for position < len(data) {
		if data[position] != 0xff {
			return false
		}
		for position < len(data) && data[position] == 0xff {
			position++
		}
		if position >= len(data) {
			return false
		}
		marker := data[position]
		position++
		switch {
		case marker == 0xd9:
			return len(bytes.Trim(data[position:], " \t\r\n\f")) == 0
		case marker == 0xd8 || marker == 0x00:
			return false
		case marker == 0x01 || marker >= 0xd0 && marker <= 0xd7:
			continue
		}
		if position+2 > len(data) {
			return false
		}
		segmentLength := int(binary.BigEndian.Uint16(data[position : position+2]))
		if segmentLength < 2 || position+segmentLength > len(data) {
			return false
		}
		position += segmentLength
		if marker != 0xda {
			continue
		}

		// Entropy-coded scan data uses FF 00 byte stuffing and may include
		// restart markers. Any other marker ends the scan and is parsed by
		// the outer loop, allowing progressive JPEGs with multiple scans.
		for position < len(data) {
			if data[position] != 0xff {
				position++
				continue
			}
			markerStart := position
			position++
			for position < len(data) && data[position] == 0xff {
				position++
			}
			if position >= len(data) {
				return false
			}
			scanMarker := data[position]
			if scanMarker == 0x00 || scanMarker >= 0xd0 && scanMarker <= 0xd7 {
				position++
				continue
			}
			position = markerStart
			break
		}
	}
	return false
}

func validPNGEnvelope(data []byte) bool {
	if !hasPNGSignature(data) {
		return false
	}
	position := 8
	seenHeader := false
	for position < len(data) {
		if position+12 > len(data) {
			return false
		}
		chunkLength := uint64(binary.BigEndian.Uint32(data[position : position+4]))
		chunkEnd := uint64(position) + 12 + chunkLength
		if chunkEnd > uint64(len(data)) {
			return false
		}
		chunkType := data[position+4 : position+8]
		chunkDataEnd := position + 8 + int(chunkLength)
		checksum := crc32.NewIEEE()
		_, _ = checksum.Write(chunkType)
		_, _ = checksum.Write(data[position+8 : chunkDataEnd])
		if checksum.Sum32() != binary.BigEndian.Uint32(data[chunkDataEnd:chunkDataEnd+4]) {
			return false
		}
		if !seenHeader {
			if string(chunkType) != "IHDR" || chunkLength != 13 {
				return false
			}
			seenHeader = true
		} else if string(chunkType) == "IHDR" {
			return false
		}
		position = int(chunkEnd)
		if string(chunkType) == "IEND" {
			return chunkLength == 0 && position == len(data)
		}
	}
	return false
}

func validWebPEnvelope(data []byte) bool {
	if !hasWebPSignature(data) || len(data) < 12 {
		return false
	}
	declaredSize := uint64(binary.LittleEndian.Uint32(data[4:8]))
	return declaredSize >= 4 && declaredSize+8 == uint64(len(data))
}

func validatePDF(data []byte) error {
	if len(data) < 14 || !bytes.HasPrefix(data, []byte("%PDF-")) ||
		!validPDFVersion(data[5], data[7]) || data[6] != '.' ||
		(data[8] != '\r' && data[8] != '\n') {
		return attachmentError(CodeInvalidPDF, "PDF header is invalid", nil)
	}
	trimmed := bytes.TrimRight(data, " \t\r\n\f")
	if !bytes.HasSuffix(trimmed, []byte("%%EOF")) {
		return attachmentError(CodeInvalidPDF, "PDF end marker is missing or has trailing data", nil)
	}
	return nil
}

func validPDFVersion(major, minor byte) bool {
	return (major == '1' && minor >= '0' && minor <= '7') ||
		(major == '2' && minor == '0')
}

func validateUTF8Text(data []byte) error {
	if !utf8.Valid(data) {
		return attachmentError(CodeInvalidText, "Text file is not valid UTF-8", nil)
	}
	for _, value := range data {
		if value < 0x20 && value != '\t' && value != '\n' && value != '\r' && value != '\f' || value == 0x7f {
			return attachmentError(CodeInvalidText, "Text file contains binary control data", nil)
		}
	}
	return nil
}

func validateJSON(data []byte) error {
	if err := validateUTF8Text(data); err != nil {
		return err
	}
	if !json.Valid(data) {
		return attachmentError(CodeInvalidJSON, "JSON file contains invalid syntax", nil)
	}
	return nil
}

type ooxmlValidationState struct {
	contentTypes             []byte
	rootRelationships        []byte
	foundMainPart            bool
	validatedUncompressedSum int64
}

func validateOOXMLPackage(data []byte, expected ooxmlPackageType) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = attachmentError(CodeInvalidOOXML, "Office document cannot be parsed safely", fmt.Errorf("OOXML parser panic: %v", recovered))
		}
	}()

	if !hasZIPSignature(data) {
		return attachmentError(CodeInvalidOOXML, "Office document does not start with a ZIP signature", nil)
	}
	if !hasExactZIPEndRecord(data) {
		return attachmentError(CodeInvalidOOXML, "Office document ZIP container has invalid trailing data", nil)
	}
	archive, zipErr := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if zipErr != nil {
		return attachmentError(CodeInvalidOOXML, "Office document is not a valid ZIP archive", zipErr)
	}
	if len(archive.File) == 0 {
		return attachmentError(CodeInvalidOOXML, "Office document archive is empty", nil)
	}
	if len(archive.File) > MaxOOXMLEntries {
		return attachmentError(CodeOOXMLArchiveLimitExceeded, "Office document contains too many ZIP entries", nil)
	}

	files, _, validationErr := validateOOXMLEntries(archive.File)
	if validationErr != nil {
		return validationErr
	}
	state := ooxmlValidationState{}
	for lowerName, file := range files {
		if file.FileInfo().IsDir() {
			continue
		}
		capture := lowerName == strings.ToLower(docxContentTypesName) ||
			lowerName == "_rels/.rels" || strings.HasSuffix(lowerName, ".rels") ||
			strings.HasSuffix(lowerName, ".xml")
		content, count, readErr := readOOXMLEntry(file, MaxOOXMLUncompressedBytes-state.validatedUncompressedSum, capture)
		if readErr != nil {
			return readErr
		}
		state.validatedUncompressedSum += count
		if state.validatedUncompressedSum > MaxOOXMLUncompressedBytes {
			return attachmentError(CodeOOXMLArchiveLimitExceeded, "Office document uncompressed data exceeds the archive safety limit", nil)
		}

		switch {
		case lowerName == strings.ToLower(docxContentTypesName):
			state.contentTypes = content
		case lowerName == "_rels/.rels":
			state.rootRelationships = content
			if _, relationshipErr := validateOOXMLRelationships(lowerName, content, expected); relationshipErr != nil {
				return relationshipErr
			}
		case strings.HasSuffix(lowerName, ".rels"):
			if _, relationshipErr := validateOOXMLRelationships(lowerName, content, expected); relationshipErr != nil {
				return relationshipErr
			}
		case strings.HasSuffix(lowerName, ".xml"):
			inspectWordFields := expected.format == LibraryFormatDOCX && strings.HasPrefix(lowerName, "word/")
			var expectedMain *ooxmlPackageType
			if lowerName == strings.ToLower(expected.mainPart) {
				expectedMain = &expected
			}
			if xmlErr := validateOOXMLPartXML(lowerName, content, inspectWordFields, expectedMain); xmlErr != nil {
				return xmlErr
			}
		}
		if lowerName == strings.ToLower(expected.mainPart) {
			state.foundMainPart = true
		}
	}

	if state.contentTypes == nil || state.rootRelationships == nil || !state.foundMainPart {
		return attachmentError(CodeInvalidOOXML, "Office document is missing required package parts", nil)
	}
	if err := validateOOXMLContentTypes(state.contentTypes, expected); err != nil {
		return err
	}
	rootTargetFound, err := validateOOXMLRelationships("_rels/.rels", state.rootRelationships, expected)
	if err != nil {
		return err
	}
	if !rootTargetFound {
		return attachmentError(CodeInvalidOOXML, "Office document root relationship does not identify the expected main part", nil)
	}
	return nil
}

func hasExactZIPEndRecord(data []byte) bool {
	const (
		endRecordLength  = 22
		maxCommentLength = 1<<16 - 1
	)
	if len(data) < endRecordLength {
		return false
	}
	start := len(data) - endRecordLength - maxCommentLength
	if start < 0 {
		start = 0
	}
	for offset := len(data) - endRecordLength; offset >= start; offset-- {
		if !bytes.Equal(data[offset:offset+4], []byte("PK\x05\x06")) {
			continue
		}
		commentLength := int(binary.LittleEndian.Uint16(data[offset+20 : offset+22]))
		if offset+endRecordLength+commentLength == len(data) {
			return true
		}
	}
	return false
}

func validateOOXMLEntries(entries []*zip.File) (map[string]*zip.File, uint64, error) {
	files := make(map[string]*zip.File, len(entries))
	var total uint64
	for _, file := range entries {
		name, err := validateOOXMLEntryName(file.Name, file.FileInfo().IsDir())
		if err != nil {
			return nil, 0, err
		}
		if file.Flags&0x1 != 0 {
			return nil, 0, attachmentError(CodeInvalidOOXML, "Encrypted Office ZIP entries are not supported", nil)
		}
		if !file.FileInfo().IsDir() && !file.Mode().IsRegular() {
			return nil, 0, attachmentError(CodeInvalidOOXML, "Office document contains a non-regular ZIP entry", nil)
		}
		lowerName := strings.ToLower(name)
		if _, duplicate := files[lowerName]; duplicate {
			return nil, 0, attachmentError(CodeInvalidOOXML, "Office document contains duplicate package part names", nil)
		}
		files[lowerName] = file
		if isOOXMLMacroPart(lowerName) {
			return nil, 0, attachmentError(CodeOOXMLMacroForbidden, "Macro-enabled Office content is not supported", nil)
		}
		if isOOXMLActiveContentPart(lowerName) {
			return nil, 0, attachmentError(CodeOOXMLMacroForbidden, "Embedded executable Office content is not supported", nil)
		}
		if file.UncompressedSize64 > MaxOOXMLUncompressedBytes || total > math.MaxUint64-file.UncompressedSize64 {
			return nil, 0, attachmentError(CodeOOXMLArchiveLimitExceeded, "Office document uncompressed data exceeds the archive safety limit", nil)
		}
		total += file.UncompressedSize64
		if total > MaxOOXMLUncompressedBytes {
			return nil, 0, attachmentError(CodeOOXMLArchiveLimitExceeded, "Office document uncompressed data exceeds the archive safety limit", nil)
		}
	}
	return files, total, nil
}

func validateOOXMLEntryName(name string, directory bool) (string, error) {
	decodedName, decodeErr := url.PathUnescape(name)
	if decodeErr != nil {
		return "", attachmentError(CodeInvalidOOXML, "Office document contains an invalid escaped package part name", decodeErr)
	}
	if decodedName == "" || strings.ContainsRune(decodedName, '\x00') || strings.Contains(decodedName, "\\") || strings.HasPrefix(decodedName, "/") {
		return "", attachmentError(CodeInvalidOOXML, "Office document contains an invalid package part name", nil)
	}
	normalized := decodedName
	if directory {
		normalized = strings.TrimSuffix(normalized, "/")
	}
	if normalized == "" || normalized == "." || path.Clean(normalized) != normalized || strings.HasPrefix(normalized, "../") {
		return "", attachmentError(CodeInvalidOOXML, "Office document contains an unsafe package part path", nil)
	}
	return normalized, nil
}

func isOOXMLMacroPart(lowerName string) bool {
	base := path.Base(lowerName)
	return base == "vbaproject.bin" || base == "vbadata.xml" ||
		strings.Contains(lowerName, "/vba/") || strings.HasSuffix(base, ".vba") ||
		strings.Contains(lowerName, "/macrosheets/")
}

func isOOXMLActiveContentPart(lowerName string) bool {
	wrappedName := "/" + strings.Trim(lowerName, "/") + "/"
	if strings.Contains(wrappedName, "/embeddings/") || strings.Contains(wrappedName, "/activex/") {
		return true
	}
	switch strings.ToLower(path.Ext(lowerName)) {
	case ".exe", ".dll", ".com", ".scr", ".msi", ".bat", ".cmd", ".ps1",
		".vbs", ".vbe", ".js", ".jse", ".wsf", ".wsh", ".hta", ".lnk",
		".reg", ".jar", ".class", ".so", ".dylib", ".sh":
		return true
	default:
		return false
	}
}

func readOOXMLEntry(file *zip.File, remaining int64, capture bool) ([]byte, int64, error) {
	if remaining < 0 {
		return nil, 0, attachmentError(CodeOOXMLArchiveLimitExceeded, "Office document uncompressed data exceeds the archive safety limit", nil)
	}
	rawReader, err := file.OpenRaw()
	if err != nil {
		return nil, 0, attachmentError(CodeInvalidOOXML, "Office ZIP entry cannot be opened", err)
	}
	var reader io.ReadCloser
	switch file.Method {
	case zip.Store:
		reader = io.NopCloser(rawReader)
	case zip.Deflate:
		reader = flate.NewReader(rawReader)
	default:
		return nil, 0, attachmentError(CodeInvalidOOXML, "Office document uses an unsupported ZIP compression method", nil)
	}

	limited := io.LimitReader(reader, remaining+1)
	checksum := crc32.NewIEEE()
	var output bytes.Buffer
	var destination io.Writer = checksum
	if capture {
		destination = io.MultiWriter(&output, checksum)
	}
	count, copyErr := io.Copy(destination, limited)
	if count > remaining {
		_ = reader.Close()
		return nil, count, attachmentError(CodeOOXMLArchiveLimitExceeded, "Office document uncompressed data exceeds the archive safety limit", nil)
	}
	if copyErr != nil {
		_ = reader.Close()
		return nil, count, attachmentError(CodeInvalidOOXML, "Office ZIP entry cannot be decompressed safely", copyErr)
	}
	if closeErr := reader.Close(); closeErr != nil {
		return nil, count, attachmentError(CodeInvalidOOXML, "Office ZIP entry failed integrity validation", closeErr)
	}
	if uint64(count) != file.UncompressedSize64 || checksum.Sum32() != file.CRC32 {
		return nil, count, attachmentError(CodeInvalidOOXML, "Office ZIP entry failed size or checksum validation", nil)
	}
	return output.Bytes(), count, nil
}

func validateOOXMLContentTypes(data []byte, expected ooxmlPackageType) error {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Strict = true
	rootCount := 0
	depth := 0
	foundExpectedMain := 0
	defaultExtensions := make(map[string]struct{})
	overrideParts := make(map[string]struct{})
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return attachmentError(CodeInvalidOOXML, "Office content types XML is malformed", err)
		}
		switch value := token.(type) {
		case xml.Directive:
			return attachmentError(CodeInvalidOOXML, "Office XML directives are not supported", nil)
		case xml.ProcInst:
			if !strings.EqualFold(value.Target, "xml") {
				return attachmentError(CodeInvalidOOXML, "Office XML processing instructions are not supported", nil)
			}
		case xml.StartElement:
			if depth == 0 {
				rootCount++
				if rootCount != 1 || value.Name.Local != "Types" ||
					!isOOXMLNamespace(value.Name.Space, ooxmlContentTypesNamespace, strictContentTypesNamespace) {
					return attachmentError(CodeInvalidOOXML, "Office content types XML has an invalid root", nil)
				}
			}
			depth++
			if depth > MaxOOXMLXMLDepth {
				return attachmentError(CodeInvalidOOXML, "Office content types XML is nested too deeply", nil)
			}
			if value.Name.Local != "Default" && value.Name.Local != "Override" {
				continue
			}
			if !isOOXMLNamespace(value.Name.Space, ooxmlContentTypesNamespace, strictContentTypesNamespace) {
				return attachmentError(CodeInvalidOOXML, "Office content type declaration has an invalid namespace", nil)
			}
			contentType := strings.TrimSpace(xmlAttribute(value.Attr, "ContentType"))
			if contentType == "" {
				return attachmentError(CodeInvalidOOXML, "Office content type declaration is incomplete", nil)
			}
			lowerContentType := strings.ToLower(contentType)
			if strings.Contains(lowerContentType, "macro") || strings.Contains(lowerContentType, "vba") ||
				strings.Contains(lowerContentType, "activex") || strings.Contains(lowerContentType, "oleobject") {
				return attachmentError(CodeOOXMLMacroForbidden, "Macro-enabled Office content is not supported", nil)
			}
			if value.Name.Local == "Default" {
				extension := strings.ToLower(strings.TrimSpace(xmlAttribute(value.Attr, "Extension")))
				if extension == "" {
					return attachmentError(CodeInvalidOOXML, "Office default content type has no extension", nil)
				}
				if _, duplicate := defaultExtensions[extension]; duplicate {
					return attachmentError(CodeInvalidOOXML, "Office content types contain a duplicate extension", nil)
				}
				defaultExtensions[extension] = struct{}{}
				continue
			}
			rawPartName := strings.TrimSpace(xmlAttribute(value.Attr, "PartName"))
			if !strings.HasPrefix(rawPartName, "/") {
				return attachmentError(CodeInvalidOOXML, "Office override content type has an invalid part name", nil)
			}
			decodedPartName, decodeErr := url.PathUnescape(strings.TrimPrefix(rawPartName, "/"))
			if decodeErr != nil || decodedPartName == "" || strings.HasPrefix(decodedPartName, "/") ||
				strings.ContainsRune(decodedPartName, '\x00') || strings.Contains(decodedPartName, "\\") ||
				path.Clean(decodedPartName) != decodedPartName || strings.HasPrefix(decodedPartName, "../") {
				return attachmentError(CodeInvalidOOXML, "Office override content type has an unsafe part name", decodeErr)
			}
			partName := strings.ToLower(decodedPartName)
			if _, duplicate := overrideParts[partName]; duplicate {
				return attachmentError(CodeInvalidOOXML, "Office content types contain a duplicate part override", nil)
			}
			overrideParts[partName] = struct{}{}
			if partName == strings.ToLower(expected.mainPart) {
				if !strings.EqualFold(contentType, expected.mainContentType) {
					return attachmentError(CodeInvalidOOXML, "Office main part has an unexpected content type", nil)
				}
				foundExpectedMain++
				continue
			}
			for _, other := range ooxmlPackageTypes {
				if other.format != expected.format && partName == strings.ToLower(other.mainPart) {
					return attachmentError(CodeTypeMismatch, "Office package type does not match its filename and MIME type", nil)
				}
			}
		case xml.EndElement:
			depth--
		}
	}
	if rootCount != 1 || depth != 0 || foundExpectedMain != 1 {
		return attachmentError(CodeInvalidOOXML, "Office content types do not declare exactly one expected main part", nil)
	}
	return nil
}

func validateOOXMLRelationships(name string, data []byte, expected ooxmlPackageType) (bool, error) {
	if !isOOXMLRelationshipPartName(name) {
		return false, attachmentError(CodeInvalidOOXML, "Office relationship part has an invalid package path", nil)
	}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Strict = true
	foundExpectedRootTarget := false
	rootCount := 0
	depth := 0
	relationshipIDs := make(map[string]struct{})
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			if rootCount != 1 || depth != 0 {
				return false, attachmentError(CodeInvalidOOXML, "Office relationships XML has an invalid root", nil)
			}
			return foundExpectedRootTarget, nil
		}
		if err != nil {
			return false, attachmentError(CodeInvalidOOXML, "Office relationships XML is malformed", fmt.Errorf("%s: %w", name, err))
		}
		switch value := token.(type) {
		case xml.Directive:
			return false, attachmentError(CodeInvalidOOXML, "Office XML directives are not supported", nil)
		case xml.ProcInst:
			if !strings.EqualFold(value.Target, "xml") {
				return false, attachmentError(CodeInvalidOOXML, "Office XML processing instructions are not supported", nil)
			}
		case xml.StartElement:
			if depth == 0 {
				rootCount++
				if rootCount != 1 || value.Name.Local != "Relationships" ||
					!isOOXMLNamespace(value.Name.Space, ooxmlPackageRelationshipsNS, strictPackageRelationshipsNS) {
					return false, attachmentError(CodeInvalidOOXML, "Office relationships XML has an invalid root", nil)
				}
			}
			depth++
			if depth > MaxOOXMLXMLDepth {
				return false, attachmentError(CodeInvalidOOXML, "Office relationships XML is nested too deeply", nil)
			}
			if value.Name.Local != "Relationship" {
				continue
			}
			if !isOOXMLNamespace(value.Name.Space, ooxmlPackageRelationshipsNS, strictPackageRelationshipsNS) {
				return false, attachmentError(CodeInvalidOOXML, "Office relationship declaration has an invalid namespace", nil)
			}
			id := strings.TrimSpace(xmlAttribute(value.Attr, "Id"))
			relationshipType := strings.TrimSpace(xmlAttribute(value.Attr, "Type"))
			targetMode := strings.TrimSpace(xmlAttribute(value.Attr, "TargetMode"))
			target := strings.TrimSpace(xmlAttribute(value.Attr, "Target"))
			if id == "" || relationshipType == "" || target == "" {
				return false, attachmentError(CodeInvalidOOXML, "Office relationship declaration is incomplete", nil)
			}
			if targetMode != "" && !strings.EqualFold(targetMode, "Internal") && !strings.EqualFold(targetMode, "External") {
				return false, attachmentError(CodeInvalidOOXML, "Office relationship target mode is invalid", nil)
			}
			if _, duplicate := relationshipIDs[id]; duplicate {
				return false, attachmentError(CodeInvalidOOXML, "Office relationships contain a duplicate identifier", nil)
			}
			relationshipIDs[id] = struct{}{}
			if strings.EqualFold(targetMode, "External") || isExternalOOXMLTarget(target) || isUnsafeOOXMLInternalTarget(name, target) {
				return false, attachmentError(CodeOOXMLExternalRelationship, "Office external relationships are not supported", nil)
			}
			if name == "_rels/.rels" && isOOXMLOfficeDocumentRelationship(relationshipType) {
				normalizedTarget := strings.TrimPrefix(strings.ToLower(target), "/")
				if normalizedTarget != strings.ToLower(expected.mainPart) {
					return false, attachmentError(CodeTypeMismatch, "Office root relationship points to a different package type", nil)
				}
				if foundExpectedRootTarget {
					return false, attachmentError(CodeInvalidOOXML, "Office package has duplicate main-document relationships", nil)
				}
				foundExpectedRootTarget = true
			}
		case xml.EndElement:
			depth--
		}
	}
}

func isOOXMLOfficeDocumentRelationship(relationshipType string) bool {
	return strings.EqualFold(strings.TrimSpace(relationshipType), ooxmlOfficeDocumentRelation) ||
		strings.EqualFold(strings.TrimSpace(relationshipType), strictOfficeDocumentRelation)
}

func isOOXMLRelationshipPartName(name string) bool {
	if name == "_rels/.rels" {
		return true
	}
	directory := path.Dir(name)
	base := path.Base(name)
	return path.Base(directory) == "_rels" && strings.HasSuffix(base, ".rels") && strings.TrimSuffix(base, ".rels") != ""
}

func isOOXMLNamespace(actual string, allowed ...string) bool {
	for _, namespace := range allowed {
		if actual == namespace {
			return true
		}
	}
	return false
}

func isExternalOOXMLTarget(target string) bool {
	if target == "" {
		return false
	}
	if isExternalRelationshipTarget(target) {
		return true
	}
	unescaped, err := url.PathUnescape(target)
	return err == nil && unescaped != target && isExternalRelationshipTarget(unescaped)
}

func isUnsafeOOXMLInternalTarget(relationshipName, target string) bool {
	if target == "" || strings.HasPrefix(target, "#") {
		return false
	}
	unescaped, err := url.PathUnescape(target)
	if err != nil || strings.ContainsRune(unescaped, '\x00') || strings.Contains(unescaped, "\\") {
		return true
	}
	parsed, err := url.Parse(unescaped)
	if err != nil {
		return true
	}
	targetPath := parsed.Path
	if targetPath == "" {
		return false
	}
	if strings.HasPrefix(targetPath, "/") {
		cleaned := path.Clean(strings.TrimPrefix(targetPath, "/"))
		return cleaned == ".." || strings.HasPrefix(cleaned, "../")
	}

	sourceDirectory := ""
	if relationshipName != "_rels/.rels" {
		relationshipDirectory := path.Dir(relationshipName)
		if path.Base(relationshipDirectory) != "_rels" || !strings.HasSuffix(path.Base(relationshipName), ".rels") {
			return true
		}
		sourceName := strings.TrimSuffix(path.Base(relationshipName), ".rels")
		sourceDirectory = path.Dir(path.Join(path.Dir(relationshipDirectory), sourceName))
	}
	resolved := path.Clean(path.Join(sourceDirectory, targetPath))
	return resolved == ".." || strings.HasPrefix(resolved, "../")
}

func validateOOXMLPartXML(name string, data []byte, inspectWordFields bool, expectedMain *ooxmlPackageType) error {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Strict = true
	inInstruction := 0
	var instruction strings.Builder
	rootCount := 0
	depth := 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			if rootCount != 1 || depth != 0 {
				return attachmentError(CodeInvalidOOXML, "Office package XML has an invalid root", nil)
			}
			if inspectWordFields && hasExternalFieldInstruction(instruction.String()) {
				return attachmentError(CodeOOXMLExternalRelationship, "Office document contains an external field link", nil)
			}
			return nil
		}
		if err != nil {
			return attachmentError(CodeInvalidOOXML, "Office package XML is malformed", fmt.Errorf("%s: %w", name, err))
		}
		switch value := token.(type) {
		case xml.Directive:
			return attachmentError(CodeInvalidOOXML, "Office XML directives are not supported", nil)
		case xml.ProcInst:
			if !strings.EqualFold(value.Target, "xml") {
				return attachmentError(CodeInvalidOOXML, "Office XML processing instructions are not supported", nil)
			}
		case xml.StartElement:
			if depth == 0 {
				rootCount++
				if rootCount != 1 {
					return attachmentError(CodeInvalidOOXML, "Office package XML has multiple roots", nil)
				}
				if expectedMain != nil && (value.Name.Local != expectedMain.mainRootLocal ||
					!isOOXMLNamespace(value.Name.Space, expectedMain.mainNamespaces...)) {
					return attachmentError(CodeTypeMismatch, "Office main part XML does not match the declared package type", nil)
				}
			}
			depth++
			if depth > MaxOOXMLXMLDepth {
				return attachmentError(CodeInvalidOOXML, "Office package XML is nested too deeply", nil)
			}
			if !inspectWordFields || !isWordprocessingMLElement(value.Name) {
				continue
			}
			switch value.Name.Local {
			case "instrText":
				inInstruction++
			case "fldSimple":
				if hasExternalFieldInstruction(xmlAttribute(value.Attr, "instr")) {
					return attachmentError(CodeOOXMLExternalRelationship, "Office document contains an external field link", nil)
				}
			case "altChunk":
				return attachmentError(CodeOOXMLExternalRelationship, "Office linked alternate content is not supported", nil)
			}
		case xml.CharData:
			if inspectWordFields && inInstruction > 0 {
				if instruction.Len()+len(value) > maxDOCXFieldInstructionBytes {
					return attachmentError(CodeInvalidOOXML, "Office field instruction is too large", nil)
				}
				_, _ = instruction.Write([]byte(value))
			}
		case xml.EndElement:
			depth--
			if !inspectWordFields || !isWordprocessingMLElement(value.Name) {
				continue
			}
			switch value.Name.Local {
			case "instrText":
				if inInstruction > 0 {
					inInstruction--
				}
			case "p":
				if hasExternalFieldInstruction(instruction.String()) {
					return attachmentError(CodeOOXMLExternalRelationship, "Office document contains an external field link", nil)
				}
				instruction.Reset()
			}
		}
	}
}

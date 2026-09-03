package chatattachment

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"sort"
	"strings"
	"testing"
)

func TestProcessLibrarySupportsSafeFileTypes(t *testing.T) {
	t.Parallel()

	jpegData := injectJPEGAPP1(t, encodeTestJPEG(t, 3, 2), []byte("Exif\x00\x00PRIVATE-METADATA"))
	webpData, err := base64.StdEncoding.DecodeString("UklGRiIAAABXRUJQVlA4IBYAAAAwAQCdASoBAAEADsD+JaQAA3AAAAAA")
	if err != nil {
		t.Fatal(err)
	}
	testCases := []struct {
		name      string
		filename  string
		mimeType  string
		data      []byte
		format    LibraryFormat
		category  LibraryCategory
		fileType  LibraryType
		extension string
	}{
		{"jpeg", "photo.JPEG", "image/jpeg", jpegData, LibraryFormatJPEG, LibraryCategoryImage, LibraryTypeImage, ".jpg"},
		{"png", "photo.png", "image/png; charset=binary", encodeTestPNG(t, 4, 5), LibraryFormatPNG, LibraryCategoryImage, LibraryTypeImage, ".png"},
		{"webp", "photo.webp", "image/webp", webpData, LibraryFormatWebP, LibraryCategoryImage, LibraryTypeImage, ".webp"},
		{"pdf", "report.pdf", "application/pdf", []byte("%PDF-1.7\n1 0 obj\n<<>>\nendobj\n%%EOF\n"), LibraryFormatPDF, LibraryCategoryFile, LibraryTypePDF, ".pdf"},
		{"docx", "notes.docx", docxMIME, buildLibraryOOXML(t, LibraryFormatDOCX, nil), LibraryFormatDOCX, LibraryCategoryFile, LibraryTypeDocument, ".docx"},
		{"xlsx", "table.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buildLibraryOOXML(t, LibraryFormatXLSX, nil), LibraryFormatXLSX, LibraryCategoryFile, LibraryTypeSpreadsheet, ".xlsx"},
		{"pptx", "slides.pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation", buildLibraryOOXML(t, LibraryFormatPPTX, nil), LibraryFormatPPTX, LibraryCategoryFile, LibraryTypePresentation, ".pptx"},
		{"txt", "notes.txt", "text/plain; charset=utf-8", []byte("plain 世界"), LibraryFormatTXT, LibraryCategoryFile, LibraryTypeDocument, ".txt"},
		{"markdown alias", "readme.md", "text/x-markdown", []byte("# 标题\n"), LibraryFormatMD, LibraryCategoryFile, LibraryTypeDocument, ".md"},
		{"csv without parsing", "data.csv", "application/csv", []byte("unclosed,\"quote\n"), LibraryFormatCSV, LibraryCategoryFile, LibraryTypeSpreadsheet, ".csv"},
		{"json alias", "data.json", "text/json; charset=utf-8", []byte(`{"ok":true}`), LibraryFormatJSON, LibraryCategoryFile, LibraryTypeOther, ".json"},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			result, processErr := ProcessLibrary(Input{
				Filename:     testCase.filename,
				DeclaredMIME: testCase.mimeType,
				Data:         testCase.data,
			})
			if processErr != nil {
				t.Fatalf("ProcessLibrary() error = %v (code %q)", processErr, CodeOf(processErr))
			}
			if result.Format != testCase.format || result.Category != testCase.category ||
				result.Type != testCase.fileType || result.Extension != testCase.extension {
				t.Fatalf("ProcessLibrary() metadata = %+v", result)
			}
			if !bytes.Equal(result.Data, testCase.data) {
				t.Fatal("ProcessLibrary() did not retain the original download bytes")
			}
			if result.ExtractedText != "" {
				t.Fatalf("ProcessLibrary() extracted document content = %q", result.ExtractedText)
			}
			if testCase.category == LibraryCategoryImage {
				if result.Kind != KindImage || len(result.SanitizedData) == 0 || result.Width <= 0 || result.Height <= 0 {
					t.Fatalf("ProcessLibrary() image result = %+v", result)
				}
				if _, _, decodeErr := image.Decode(bytes.NewReader(result.SanitizedData)); decodeErr != nil {
					t.Fatalf("sanitized image cannot be decoded: %v", decodeErr)
				}
			} else if result.Kind != KindFile || result.SanitizedData != nil {
				t.Fatalf("ProcessLibrary() file result = %+v", result)
			}
		})
	}
}

func TestProcessLibrarySanitizesImagesWithoutReplacingOriginals(t *testing.T) {
	t.Parallel()

	original := injectJPEGAPP1(t, encodeTestJPEG(t, 2, 3), []byte("Exif\x00\x00PRIVATE-METADATA"))
	result, err := ProcessLibrary(Input{Filename: "photo.jpg", DeclaredMIME: "image/jpeg", Data: original})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(result.Data, []byte("PRIVATE-METADATA")) {
		t.Fatal("original image bytes were unexpectedly replaced")
	}
	if bytes.Contains(result.SanitizedData, []byte("Exif")) || bytes.Contains(result.SanitizedData, []byte("PRIVATE-METADATA")) {
		t.Fatal("sanitized image retained metadata")
	}
	if result.SanitizedMIMEType != "image/jpeg" || result.SanitizedExtension != ".jpg" {
		t.Fatalf("sanitized image metadata = %q %q", result.SanitizedMIMEType, result.SanitizedExtension)
	}

	webpData, decodeErr := base64.StdEncoding.DecodeString("UklGRiIAAABXRUJQVlA4IBYAAAAwAQCdASoBAAEADsD+JaQAA3AAAAAA")
	if decodeErr != nil {
		t.Fatal(decodeErr)
	}
	webp, err := ProcessLibrary(Input{Filename: "photo.webp", DeclaredMIME: "image/webp", Data: webpData})
	if err != nil {
		t.Fatal(err)
	}
	if webp.MIMEType != "image/webp" || webp.Extension != ".webp" ||
		webp.SanitizedMIMEType != "image/png" || webp.SanitizedExtension != ".png" {
		t.Fatalf("WebP original/sanitized metadata = %+v", webp)
	}
}

func TestProcessLibraryRejectsUnsupportedMismatchedAndMalformedFiles(t *testing.T) {
	t.Parallel()

	assertLibraryError(t, Input{}, CodeEmptyFile)
	assertLibraryError(t, Input{Filename: "archive.zip", DeclaredMIME: "application/zip", Data: []byte("PK\x03\x04")}, CodeUnsupportedType)
	assertLibraryError(t, Input{Filename: "photo.png", DeclaredMIME: "image/jpeg", Data: encodeTestPNG(t, 1, 1)}, CodeTypeMismatch)
	assertLibraryError(t, Input{Filename: "photo.jpg", DeclaredMIME: "image/jpeg", Data: encodeTestPNG(t, 1, 1)}, CodeTypeMismatch)
	assertLibraryError(t, Input{Filename: "report.pdf", DeclaredMIME: "application/pdf", Data: []byte("%PDF-1.7\nmissing EOF")}, CodeInvalidPDF)
	assertLibraryError(t, Input{Filename: "report.pdf", DeclaredMIME: "application/pdf", Data: []byte("%PDF-1.7\n%%EOF\n<html>")}, CodeInvalidPDF)
	assertLibraryError(t, Input{Filename: "bad.txt", DeclaredMIME: "text/plain", Data: []byte{0xff, 0xfe}}, CodeInvalidText)
	assertLibraryError(t, Input{Filename: "bad.md", DeclaredMIME: "text/markdown", Data: []byte{0xc3, 0x28}}, CodeInvalidText)
	assertLibraryError(t, Input{Filename: "bad.csv", DeclaredMIME: "text/csv", Data: []byte{0xff}}, CodeInvalidText)
	assertLibraryError(t, Input{Filename: "binary.txt", DeclaredMIME: "text/plain", Data: []byte("safe\x00payload")}, CodeInvalidText)
	assertLibraryError(t, Input{Filename: "bad.json", DeclaredMIME: "application/json", Data: []byte(`{"missing":}`)}, CodeInvalidJSON)
	assertLibraryError(t, Input{Filename: "polyglot.png", DeclaredMIME: "image/png", Data: append(encodeTestPNG(t, 1, 1), []byte("<script>")...)}, CodeInvalidImage)
}

func TestProcessLibraryValidatesOOXMLPackageIdentityAndActiveContent(t *testing.T) {
	t.Parallel()

	t.Run("package type mismatch", func(t *testing.T) {
		t.Parallel()
		assertLibraryError(t, Input{
			Filename:     "fake.docx",
			DeclaredMIME: docxMIME,
			Data:         buildLibraryOOXML(t, LibraryFormatXLSX, nil),
		}, CodeTypeMismatch)
	})

	t.Run("macro part", func(t *testing.T) {
		t.Parallel()
		assertLibraryOOXMLError(t, LibraryFormatXLSX, map[string][]byte{"xl/vbaProject.bin": []byte("macro")}, CodeOOXMLMacroForbidden)
	})

	t.Run("macro content type", func(t *testing.T) {
		t.Parallel()
		files := libraryOOXMLFiles(LibraryFormatDOCX)
		files[docxContentTypesName] = []byte(strings.Replace(
			string(files[docxContentTypesName]),
			ooxmlPackageTypes[LibraryFormatDOCX].mainContentType,
			"application/vnd.ms-word.document.macroEnabled.main+xml",
			1,
		))
		assertLibraryError(t, libraryInput(LibraryFormatDOCX, buildLibraryZIP(t, files)), CodeOOXMLMacroForbidden)
	})

	t.Run("external relationship", func(t *testing.T) {
		t.Parallel()
		assertLibraryOOXMLError(t, LibraryFormatPPTX, map[string][]byte{
			"ppt/_rels/presentation.xml.rels": []byte(`<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" Target="https://example.com" TargetMode="External"/></Relationships>`),
		}, CodeOOXMLExternalRelationship)
	})

	t.Run("encoded external relationship", func(t *testing.T) {
		t.Parallel()
		assertLibraryOOXMLError(t, LibraryFormatXLSX, map[string][]byte{
			"xl/_rels/workbook.xml.rels": []byte(`<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId2" Type="link" Target="https%3A%2F%2Fexample.com"/></Relationships>`),
		}, CodeOOXMLExternalRelationship)
	})

	t.Run("escaping internal relationship", func(t *testing.T) {
		t.Parallel()
		assertLibraryOOXMLError(t, LibraryFormatXLSX, map[string][]byte{
			"xl/_rels/workbook.xml.rels": []byte(`<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId2" Type="link" Target="../../outside.xml"/></Relationships>`),
		}, CodeOOXMLExternalRelationship)
	})

	t.Run("safe parent relationship within package", func(t *testing.T) {
		t.Parallel()
		data := buildLibraryOOXML(t, LibraryFormatDOCX, map[string][]byte{
			"word/_rels/document.xml.rels": []byte(`<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId2" Type="custom" Target="../customXml/item1.xml"/></Relationships>`),
			"customXml/item1.xml":          []byte(`<?xml version="1.0"?><item/>`),
		})
		if _, err := ProcessLibrary(libraryInput(LibraryFormatDOCX, data)); err != nil {
			t.Fatalf("ProcessLibrary() rejected a safe package-relative relationship: %v", err)
		}
	})

	t.Run("external Word field", func(t *testing.T) {
		t.Parallel()
		files := libraryOOXMLFiles(LibraryFormatDOCX)
		files["word/document.xml"] = []byte(`<?xml version="1.0"?><w:document xmlns:w="` + wordprocessingMLNamespace + `"><w:body><w:p><w:r><w:instrText>HYPERLINK &quot;https://example.com&quot;</w:instrText></w:r></w:p></w:body></w:document>`)
		assertLibraryError(t, libraryInput(LibraryFormatDOCX, buildLibraryZIP(t, files)), CodeOOXMLExternalRelationship)
	})

	t.Run("unsafe archive path", func(t *testing.T) {
		t.Parallel()
		assertLibraryOOXMLError(t, LibraryFormatDOCX, map[string][]byte{"../escape": []byte("x")}, CodeInvalidOOXML)
	})

	t.Run("encoded unsafe archive path", func(t *testing.T) {
		t.Parallel()
		assertLibraryOOXMLError(t, LibraryFormatDOCX, map[string][]byte{"word/%2e%2e/%2e%2e/escape": []byte("x")}, CodeInvalidOOXML)
	})

	t.Run("wrong main XML root", func(t *testing.T) {
		t.Parallel()
		files := libraryOOXMLFiles(LibraryFormatXLSX)
		files["xl/workbook.xml"] = []byte(`<?xml version="1.0"?><w:document xmlns:w="` + wordprocessingMLNamespace + `"/>`)
		assertLibraryError(t, libraryInput(LibraryFormatXLSX, buildLibraryZIP(t, files)), CodeTypeMismatch)
	})

	t.Run("prefixed ZIP polyglot", func(t *testing.T) {
		t.Parallel()
		data := append([]byte("not-a-zip-prefix"), buildLibraryOOXML(t, LibraryFormatPPTX, nil)...)
		assertLibraryError(t, libraryInput(LibraryFormatPPTX, data), CodeInvalidOOXML)
	})

	t.Run("trailing ZIP polyglot", func(t *testing.T) {
		t.Parallel()
		data := append(buildLibraryOOXML(t, LibraryFormatPPTX, nil), []byte("<script>")...)
		assertLibraryError(t, libraryInput(LibraryFormatPPTX, data), CodeInvalidOOXML)
	})

	t.Run("embedded executable", func(t *testing.T) {
		t.Parallel()
		assertLibraryOOXMLError(t, LibraryFormatDOCX, map[string][]byte{
			"word/embeddings/payload.exe": []byte("MZ"),
		}, CodeOOXMLMacroForbidden)
	})

	t.Run("malformed package XML", func(t *testing.T) {
		t.Parallel()
		assertLibraryOOXMLError(t, LibraryFormatXLSX, map[string][]byte{"xl/styles.xml": []byte(`<styleSheet>`)}, CodeInvalidOOXML)
	})

	t.Run("missing root relationship", func(t *testing.T) {
		t.Parallel()
		files := libraryOOXMLFiles(LibraryFormatPPTX)
		delete(files, "_rels/.rels")
		assertLibraryError(t, libraryInput(LibraryFormatPPTX, buildLibraryZIP(t, files)), CodeInvalidOOXML)
	})

	t.Run("archive metadata limit", func(t *testing.T) {
		files := libraryOOXMLFiles(LibraryFormatDOCX)
		files["word/media/filler.dat"] = []byte("x")
		data := buildLibraryZIP(t, files)
		mutateZIPCentralUncompressedSize(t, data, "word/media/filler.dat", MaxOOXMLUncompressedBytes+1)
		assertLibraryError(t, libraryInput(LibraryFormatDOCX, data), CodeOOXMLArchiveLimitExceeded)
	})
}

func TestProcessLibraryKeepsChatProcessingContractUnchanged(t *testing.T) {
	t.Parallel()

	files := baseDOCXFiles(`<w:p><w:r/></w:p>`)
	data := buildTestDOCX(t, files)
	if _, err := ProcessLibrary(Input{Filename: "empty.docx", DeclaredMIME: docxMIME, Data: data}); err != nil {
		t.Fatalf("ProcessLibrary() rejected an empty but valid DOCX: %v", err)
	}
	assertProcessError(t, Input{Filename: "empty.docx", DeclaredMIME: docxMIME, Data: data}, CodeInvalidDOCX)
	assertProcessError(t, Input{Filename: "report.pdf", DeclaredMIME: "application/pdf", Data: []byte("%PDF-1.7\n%%EOF")}, CodeUnsupportedType)
}

func assertLibraryOOXMLError(t *testing.T, format LibraryFormat, additions map[string][]byte, want ErrorCode) {
	t.Helper()
	assertLibraryError(t, libraryInput(format, buildLibraryOOXML(t, format, additions)), want)
}

func assertLibraryError(t *testing.T, input Input, want ErrorCode) {
	t.Helper()
	result, err := ProcessLibrary(input)
	if err == nil {
		t.Fatalf("ProcessLibrary() result = %+v, want error %q", result, want)
	}
	if result != nil {
		t.Fatalf("ProcessLibrary() result = %+v with error, want nil", result)
	}
	if got := CodeOf(err); got != want {
		t.Fatalf("ProcessLibrary() error = %v (code %q), want %q", err, got, want)
	}
}

func libraryInput(format LibraryFormat, data []byte) Input {
	switch format {
	case LibraryFormatDOCX:
		return Input{Filename: "document.docx", DeclaredMIME: docxMIME, Data: data}
	case LibraryFormatXLSX:
		return Input{Filename: "workbook.xlsx", DeclaredMIME: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", Data: data}
	case LibraryFormatPPTX:
		return Input{Filename: "presentation.pptx", DeclaredMIME: "application/vnd.openxmlformats-officedocument.presentationml.presentation", Data: data}
	default:
		panic(fmt.Sprintf("unsupported test OOXML format %q", format))
	}
}

func buildLibraryOOXML(t *testing.T, format LibraryFormat, additions map[string][]byte) []byte {
	t.Helper()
	files := libraryOOXMLFiles(format)
	for name, content := range additions {
		files[name] = content
	}
	return buildLibraryZIP(t, files)
}

func libraryOOXMLFiles(format LibraryFormat) map[string][]byte {
	packageType, ok := ooxmlPackageTypes[format]
	if !ok {
		panic(fmt.Sprintf("unsupported test OOXML format %q", format))
	}
	mainXML := ""
	switch format {
	case LibraryFormatDOCX:
		mainXML = `<?xml version="1.0" encoding="UTF-8"?><w:document xmlns:w="` + wordprocessingMLNamespace + `"><w:body/></w:document>`
	case LibraryFormatXLSX:
		mainXML = `<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="` + spreadsheetMLNamespace + `"/>`
	case LibraryFormatPPTX:
		mainXML = `<?xml version="1.0" encoding="UTF-8"?><p:presentation xmlns:p="` + presentationMLNamespace + `"/>`
	}
	return map[string][]byte{
		docxContentTypesName: []byte(`<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/` + packageType.mainPart + `" ContentType="` + packageType.mainContentType + `"/></Types>`),
		"_rels/.rels":        []byte(`<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="` + packageType.mainPart + `"/></Relationships>`),
		packageType.mainPart: []byte(mainXML),
	}
}

func buildLibraryZIP(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)

	var output bytes.Buffer
	archive := zip.NewWriter(&output)
	for _, name := range names {
		entry, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(files[name]); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

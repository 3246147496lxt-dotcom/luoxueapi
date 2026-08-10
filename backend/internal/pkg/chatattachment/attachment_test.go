package chatattachment

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"sort"
	"strings"
	"sync"
	"testing"
)

func TestProcessRejectsEmptyAndUnsupportedAttachments(t *testing.T) {
	t.Parallel()

	assertProcessError(t, Input{Filename: "empty.png", DeclaredMIME: "image/png"}, CodeEmptyFile)
	for _, testCase := range []struct {
		filename string
		mimeType string
		data     []byte
	}{
		{"legacy.doc", "application/msword", []byte("legacy")},
		{"vector.svg", "image/svg+xml", []byte("<svg/>")},
		{"animated.gif", "image/gif", []byte("GIF89a")},
		{"macro.docm", "application/vnd.ms-word.document.macroEnabled.12", []byte("PK\x03\x04")},
	} {
		testCase := testCase
		t.Run(testCase.filename, func(t *testing.T) {
			t.Parallel()
			assertProcessError(t, Input{
				Filename:     testCase.filename,
				DeclaredMIME: testCase.mimeType,
				Data:         testCase.data,
			}, CodeUnsupportedType)
		})
	}
}

func TestProcessRejectsTypeMismatches(t *testing.T) {
	t.Parallel()

	pngData := encodeTestPNG(t, 2, 2)
	assertProcessError(t, Input{
		Filename:     "photo.png",
		DeclaredMIME: "image/jpeg",
		Data:         pngData,
	}, CodeTypeMismatch)
	assertProcessError(t, Input{
		Filename:     "photo.jpg",
		DeclaredMIME: "image/jpeg",
		Data:         pngData,
	}, CodeTypeMismatch)
	assertProcessError(t, Input{
		Filename:     "photo.png",
		DeclaredMIME: "not a MIME type",
		Data:         pngData,
	}, CodeTypeMismatch)
}

func TestProcessJPEGStripsMetadataByReencoding(t *testing.T) {
	t.Parallel()

	jpegData := encodeTestJPEG(t, 3, 2)
	metadata := []byte("Exif\x00\x00GPS-SECRET-LOCATION")
	jpegData = injectJPEGAPP1(t, jpegData, metadata)

	result, err := Process(Input{
		Filename:     "photo.jpeg",
		DeclaredMIME: "image/jpeg",
		Data:         jpegData,
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if result.Kind != KindImage || result.MIMEType != "image/jpeg" || result.Extension != ".jpg" {
		t.Fatalf("unexpected result metadata: %+v", result)
	}
	if result.Width != 3 || result.Height != 2 {
		t.Fatalf("dimensions = %dx%d, want 3x2", result.Width, result.Height)
	}
	if bytes.Contains(result.SanitizedData, []byte("Exif")) || bytes.Contains(result.SanitizedData, []byte("GPS-SECRET")) {
		t.Fatal("sanitized JPEG retained injected EXIF data")
	}
	if _, format, decodeErr := image.Decode(bytes.NewReader(result.SanitizedData)); decodeErr != nil || format != "jpeg" {
		t.Fatalf("sanitized image decode = %q, %v", format, decodeErr)
	}
}

func TestProcessJPEGAppliesEXIFOrientationsBeforeStrippingMetadata(t *testing.T) {
	t.Parallel()

	const sourceWidth, sourceHeight = 48, 32
	topLeft := color.NRGBA{R: 240, G: 20, B: 20, A: 255}
	topRight := color.NRGBA{R: 20, G: 220, B: 20, A: 255}
	bottomLeft := color.NRGBA{R: 20, G: 20, B: 240, A: 255}
	bottomRight := color.NRGBA{R: 240, G: 220, B: 20, A: 255}
	source := encodeCornerJPEG(t, sourceWidth, sourceHeight, [4]color.NRGBA{
		topLeft, topRight, bottomLeft, bottomRight,
	})

	testCases := []struct {
		name         string
		orientation  uint16
		littleEndian bool
		width        int
		height       int
		corners      [4]color.NRGBA
	}{
		{"mirror-horizontal", 2, true, sourceWidth, sourceHeight, [4]color.NRGBA{topRight, topLeft, bottomRight, bottomLeft}},
		{"rotate-180", 3, false, sourceWidth, sourceHeight, [4]color.NRGBA{bottomRight, bottomLeft, topRight, topLeft}},
		{"mirror-vertical", 4, true, sourceWidth, sourceHeight, [4]color.NRGBA{bottomLeft, bottomRight, topLeft, topRight}},
		{"transpose", 5, false, sourceHeight, sourceWidth, [4]color.NRGBA{topLeft, bottomLeft, topRight, bottomRight}},
		{"rotate-90-clockwise", 6, true, sourceHeight, sourceWidth, [4]color.NRGBA{bottomLeft, topLeft, bottomRight, topRight}},
		{"transverse", 7, false, sourceHeight, sourceWidth, [4]color.NRGBA{bottomRight, topRight, bottomLeft, topLeft}},
		{"rotate-90-counterclockwise", 8, true, sourceHeight, sourceWidth, [4]color.NRGBA{topRight, bottomRight, topLeft, bottomLeft}},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			withEXIF := injectJPEGAPP1(t, source, exifOrientationPayload(testCase.orientation, testCase.littleEndian))
			result, err := Process(Input{
				Filename:     "oriented.jpg",
				DeclaredMIME: "image/jpeg",
				Data:         withEXIF,
			})
			if err != nil {
				t.Fatalf("Process() error = %v", err)
			}
			if result.Width != testCase.width || result.Height != testCase.height {
				t.Fatalf("orientation %d dimensions = %dx%d, want %dx%d", testCase.orientation, result.Width, result.Height, testCase.width, testCase.height)
			}
			if bytes.Contains(result.SanitizedData, []byte("Exif\x00\x00")) {
				t.Fatalf("orientation %d output retained EXIF metadata", testCase.orientation)
			}
			decoded, format, decodeErr := image.Decode(bytes.NewReader(result.SanitizedData))
			if decodeErr != nil || format != "jpeg" {
				t.Fatalf("decode sanitized orientation %d JPEG = %q, %v", testCase.orientation, format, decodeErr)
			}
			assertImageCornerColors(t, decoded, testCase.corners)
		})
	}
}

func TestProcessPNGReencodesImage(t *testing.T) {
	t.Parallel()

	result, err := Process(Input{
		Filename:     "image.PNG",
		DeclaredMIME: "image/png; charset=binary",
		Data:         encodeTestPNG(t, 4, 5),
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if result.MIMEType != "image/png" || result.Extension != ".png" || result.Width != 4 || result.Height != 5 {
		t.Fatalf("unexpected PNG result: %+v", result)
	}
	if !hasPNGSignature(result.SanitizedData) {
		t.Fatal("sanitized PNG has no PNG signature")
	}
}

func TestProcessWebPOutputsMetadataFreePNG(t *testing.T) {
	t.Parallel()

	// A constructed 1x1 lossy WebP image.
	const onePixelWebP = "UklGRiIAAABXRUJQVlA4IBYAAAAwAQCdASoBAAEADsD+JaQAA3AAAAAA"
	data, err := base64.StdEncoding.DecodeString(onePixelWebP)
	if err != nil {
		t.Fatal(err)
	}
	result, processErr := Process(Input{
		Filename:     "pixel.webp",
		DeclaredMIME: "image/webp",
		Data:         data,
	})
	if processErr != nil {
		t.Fatalf("Process() error = %v", processErr)
	}
	if result.MIMEType != "image/png" || result.Extension != ".png" || !hasPNGSignature(result.SanitizedData) {
		t.Fatalf("WebP was not normalized to PNG: %+v", result)
	}
}

func TestProcessImageLimitsAreCheckedBeforeFullDecode(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name   string
		width  uint32
		height uint32
	}{
		{"side", MaxImageDimension + 1, 1},
		{"pixels", 6000, 5000},
	} {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assertProcessError(t, Input{
				Filename:     "oversized.png",
				DeclaredMIME: "image/png",
				Data:         pngHeaderOnly(testCase.width, testCase.height),
			}, CodeImageDimensionsExceeded)
		})
	}
}

func TestProcessRejectsTruncatedImage(t *testing.T) {
	t.Parallel()
	assertProcessError(t, Input{
		Filename:     "bad.png",
		DeclaredMIME: "image/png",
		Data:         []byte("\x89PNG\r\n\x1a\n"),
	}, CodeInvalidImage)
}

func TestProcessPDFExtractsText(t *testing.T) {
	t.Parallel()

	result, err := Process(Input{
		Filename:     "report.pdf",
		DeclaredMIME: "application/pdf",
		Data:         buildTestPDF(t, []string{"First page", "Second page"}, false),
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if result.Kind != KindPDF || result.PageCount != 2 || result.SanitizedData != nil {
		t.Fatalf("unexpected PDF result: %+v", result)
	}
	if !strings.Contains(result.Text, "First page") || !strings.Contains(result.Text, "Second page") {
		t.Fatalf("extracted PDF text = %q", result.Text)
	}
}

func TestProcessPDFRejectsPageLimit(t *testing.T) {
	t.Parallel()
	pages := make([]string, MaxPDFPages+1)
	for index := range pages {
		pages[index] = fmt.Sprintf("Page %d", index+1)
	}
	assertProcessError(t, Input{
		Filename:     "long.pdf",
		DeclaredMIME: "application/pdf",
		Data:         buildTestPDF(t, pages, false),
	}, CodePDFPageLimitExceeded)
}

func TestProcessPDFAcceptsFiftyPageBoundary(t *testing.T) {
	t.Parallel()
	pages := make([]string, MaxPDFPages)
	for index := range pages {
		pages[index] = fmt.Sprintf("Page %d", index+1)
	}
	result, err := Process(Input{
		Filename:     "fifty.pdf",
		DeclaredMIME: "application/pdf",
		Data:         buildTestPDF(t, pages, false),
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if result.PageCount != MaxPDFPages {
		t.Fatalf("PageCount = %d, want %d", result.PageCount, MaxPDFPages)
	}
}

func TestProcessPDFRejectsMissingTextLayer(t *testing.T) {
	t.Parallel()
	assertProcessError(t, Input{
		Filename:     "scan.pdf",
		DeclaredMIME: "application/pdf",
		Data:         buildTestPDF(t, []string{""}, false),
	}, CodePDFNoText)
}

func TestProcessPDFRejectsTextLimit(t *testing.T) {
	t.Parallel()
	assertProcessError(t, Input{
		Filename:     "huge-text.pdf",
		DeclaredMIME: "application/pdf",
		Data:         buildTestPDF(t, []string{strings.Repeat("A", MaxExtractedTextCharacters+1)}, false),
	}, CodeTextLimitExceeded)
}

func TestProcessRejectsEncryptedAndMalformedPDF(t *testing.T) {
	t.Parallel()

	assertProcessError(t, Input{
		Filename:     "secret.pdf",
		DeclaredMIME: "application/pdf",
		Data:         buildTestPDF(t, []string{"Secret"}, true),
	}, CodePDFEncrypted)
	assertProcessError(t, Input{
		Filename:     "broken.pdf",
		DeclaredMIME: "application/pdf",
		Data:         append([]byte("%PDF-1.7\n"), make([]byte, 128)...),
	}, CodeInvalidPDF)
}

func TestProcessDOCXExtractsText(t *testing.T) {
	t.Parallel()
	files := baseDOCXFiles(`<w:p><w:r><w:t>Hello</w:t><w:tab/><w:t>World</w:t></w:r></w:p><w:p><w:r><w:t>Second line</w:t></w:r></w:p>`)
	result, err := Process(Input{
		Filename:     "notes.docx",
		DeclaredMIME: docxMIME,
		Data:         buildTestDOCX(t, files),
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if result.Kind != KindDOCX || result.MIMEType != docxMIME || result.Extension != ".docx" || result.SanitizedData != nil {
		t.Fatalf("unexpected DOCX result: %+v", result)
	}
	if result.Text != "Hello\tWorld\nSecond line" {
		t.Fatalf("extracted DOCX text = %q", result.Text)
	}
}

func TestProcessDOCXRejectsMacros(t *testing.T) {
	t.Parallel()

	t.Run("vba part", func(t *testing.T) {
		t.Parallel()
		files := baseDOCXFiles(`<w:p><w:r><w:t>Hello</w:t></w:r></w:p>`)
		files["word/vbaProject.bin"] = []byte("macro")
		assertProcessError(t, docxInput(buildTestDOCX(t, files)), CodeDOCXMacroForbidden)
	})

	t.Run("macro content type", func(t *testing.T) {
		t.Parallel()
		files := baseDOCXFiles(`<w:p><w:r><w:t>Hello</w:t></w:r></w:p>`)
		files[docxContentTypesName] = []byte(strings.ReplaceAll(
			string(files[docxContentTypesName]), docxMain,
			"application/vnd.ms-word.document.macroEnabled.main+xml",
		))
		assertProcessError(t, docxInput(buildTestDOCX(t, files)), CodeDOCXMacroForbidden)
	})
}

func TestProcessDOCXRejectsExternalRelationshipsAndFields(t *testing.T) {
	t.Parallel()

	t.Run("relationship", func(t *testing.T) {
		t.Parallel()
		files := baseDOCXFiles(`<w:p><w:hyperlink r:id="rId5"><w:r><w:t>Link</w:t></w:r></w:hyperlink></w:p>`)
		files["word/_rels/document.xml.rels"] = []byte(`<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId5" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" Target="https://example.com" TargetMode="External"/></Relationships>`)
		assertProcessError(t, docxInput(buildTestDOCX(t, files)), CodeDOCXExternalRelationship)
	})

	t.Run("field", func(t *testing.T) {
		t.Parallel()
		files := baseDOCXFiles(`<w:p><w:r><w:instrText>HYPERLINK &quot;https://example.com&quot;</w:instrText><w:t>Link</w:t></w:r></w:p>`)
		assertProcessError(t, docxInput(buildTestDOCX(t, files)), CodeDOCXExternalRelationship)
	})

	t.Run("split field without URI scheme", func(t *testing.T) {
		t.Parallel()
		files := baseDOCXFiles(`<w:p><w:r><w:instrText>HYPER</w:instrText><w:instrText>LINK &quot;www.example.com&quot;</w:instrText><w:t>Link</w:t></w:r></w:p>`)
		assertProcessError(t, docxInput(buildTestDOCX(t, files)), CodeDOCXExternalRelationship)
	})
}

func TestProcessDOCXRejectsArchiveLimits(t *testing.T) {
	t.Parallel()

	t.Run("entry count", func(t *testing.T) {
		files := baseDOCXFiles(`<w:p><w:r><w:t>Hello</w:t></w:r></w:p>`)
		for index := 0; len(files) <= MaxDOCXEntries; index++ {
			files[fmt.Sprintf("word/media/%04d.dat", index)] = nil
		}
		assertProcessError(t, docxInput(buildTestDOCX(t, files)), CodeDOCXArchiveLimitExceeded)
	})

	t.Run("declared uncompressed size", func(t *testing.T) {
		files := baseDOCXFiles(`<w:p><w:r><w:t>Hello</w:t></w:r></w:p>`)
		files["word/media/filler.dat"] = []byte("x")
		data := buildTestDOCX(t, files)
		mutateZIPCentralUncompressedSize(t, data, "word/media/filler.dat", MaxDOCXUncompressedBytes+1)
		assertProcessError(t, docxInput(data), CodeDOCXArchiveLimitExceeded)
	})

	t.Run("compressed zip bomb with false size", func(t *testing.T) {
		data := buildTestDOCXZipBomb(t)
		mutateZIPCentralUncompressedSize(t, data, "word/media/bomb.bin", 1)
		assertProcessError(t, docxInput(data), CodeDOCXArchiveLimitExceeded)
	})
}

func TestProcessDOCXRejectsUnsafeOrMalformedPackages(t *testing.T) {
	t.Parallel()

	t.Run("path traversal", func(t *testing.T) {
		files := baseDOCXFiles(`<w:p><w:r><w:t>Hello</w:t></w:r></w:p>`)
		files["../escape"] = []byte("x")
		assertProcessError(t, docxInput(buildTestDOCX(t, files)), CodeInvalidDOCX)
	})

	t.Run("missing main part", func(t *testing.T) {
		files := baseDOCXFiles(`<w:p><w:r><w:t>Hello</w:t></w:r></w:p>`)
		delete(files, docxDocumentName)
		assertProcessError(t, docxInput(buildTestDOCX(t, files)), CodeInvalidDOCX)
	})

	t.Run("malformed relationships", func(t *testing.T) {
		files := baseDOCXFiles(`<w:p><w:r><w:t>Hello</w:t></w:r></w:p>`)
		files["word/_rels/document.xml.rels"] = []byte(`<Relationships><Relationship`)
		assertProcessError(t, docxInput(buildTestDOCX(t, files)), CodeInvalidDOCX)
	})

	t.Run("empty text", func(t *testing.T) {
		files := baseDOCXFiles(`<w:p><w:r/></w:p>`)
		assertProcessError(t, docxInput(buildTestDOCX(t, files)), CodeInvalidDOCX)
	})
}

func TestProcessDOCXRejectsTextLimit(t *testing.T) {
	t.Parallel()
	files := baseDOCXFiles(`<w:p><w:r><w:t>` + strings.Repeat("界", MaxExtractedTextCharacters+1) + `</w:t></w:r></w:p>`)
	assertProcessError(t, docxInput(buildTestDOCX(t, files)), CodeTextLimitExceeded)
}

func TestCodeOfFindsWrappedAttachmentError(t *testing.T) {
	t.Parallel()
	err := fmt.Errorf("outer: %w", attachmentError(CodeInvalidPDF, "bad PDF", nil))
	if got := CodeOf(err); got != CodeInvalidPDF {
		t.Fatalf("CodeOf() = %q, want %q", got, CodeInvalidPDF)
	}
	if got := CodeOf(fmt.Errorf("ordinary")); got != "" {
		t.Fatalf("CodeOf(ordinary) = %q, want empty", got)
	}
}

func TestProcessMalformedInputsDoNotPanicConcurrently(t *testing.T) {
	t.Parallel()
	inputs := []Input{
		{Filename: "bad.jpg", DeclaredMIME: "image/jpeg", Data: []byte("\xff\xd8\xff")},
		{Filename: "bad.png", DeclaredMIME: "image/png", Data: []byte("\x89PNG\r\n\x1a\n")},
		{Filename: "bad.webp", DeclaredMIME: "image/webp", Data: []byte("RIFF\x01\x00\x00\x00WEBP")},
		{Filename: "bad.pdf", DeclaredMIME: "application/pdf", Data: append([]byte("%PDF-1.7\n"), make([]byte, 128)...)},
		{Filename: "bad.docx", DeclaredMIME: docxMIME, Data: []byte("PK\x03\x04broken")},
	}

	panicValues := make(chan any, len(inputs)*8)
	unexpectedSuccesses := make(chan string, len(inputs)*8)
	var group sync.WaitGroup
	for iteration := 0; iteration < 8; iteration++ {
		for _, input := range inputs {
			input := input
			group.Add(1)
			go func() {
				defer group.Done()
				defer func() {
					if recovered := recover(); recovered != nil {
						panicValues <- recovered
					}
				}()
				if _, err := Process(input); err == nil {
					unexpectedSuccesses <- input.Filename
				}
			}()
		}
	}
	group.Wait()
	close(panicValues)
	close(unexpectedSuccesses)
	for recovered := range panicValues {
		t.Fatalf("Process() panicked under concurrent malformed input: %v", recovered)
	}
	for filename := range unexpectedSuccesses {
		t.Fatalf("Process() unexpectedly accepted malformed %s", filename)
	}
}

func assertProcessError(t *testing.T, input Input, want ErrorCode) {
	t.Helper()
	result, err := Process(input)
	if err == nil {
		t.Fatalf("Process() result = %+v, want error %q", result, want)
	}
	if result != nil {
		t.Fatalf("Process() result = %+v with error, want nil", result)
	}
	if got := CodeOf(err); got != want {
		t.Fatalf("Process() error = %v (code %q), want code %q", err, got, want)
	}
}

func encodeTestPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	var output bytes.Buffer
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 23), G: uint8(y * 31), B: 90, A: 255})
		}
	}
	if err := png.Encode(&output, img); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func encodeTestJPEG(t *testing.T, width, height int) []byte {
	t.Helper()
	var output bytes.Buffer
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(20 + x*30), G: uint8(40 + y*20), B: 100, A: 255})
		}
	}
	if err := jpeg.Encode(&output, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func encodeCornerJPEG(t *testing.T, width, height int, corners [4]color.NRGBA) []byte {
	t.Helper()
	var output bytes.Buffer
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			corner := 0
			if x >= width/2 {
				corner++
			}
			if y >= height/2 {
				corner += 2
			}
			img.SetNRGBA(x, y, corners[corner])
		}
	}
	if err := jpeg.Encode(&output, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func exifOrientationPayload(orientation uint16, littleEndian bool) []byte {
	tiff := make([]byte, 8+2+12+4)
	var byteOrder binary.ByteOrder = binary.BigEndian
	copy(tiff[:2], "MM")
	if littleEndian {
		byteOrder = binary.LittleEndian
		copy(tiff[:2], "II")
	}
	byteOrder.PutUint16(tiff[2:4], 42)
	byteOrder.PutUint32(tiff[4:8], 8)
	byteOrder.PutUint16(tiff[8:10], 1)
	byteOrder.PutUint16(tiff[10:12], 0x0112)
	byteOrder.PutUint16(tiff[12:14], 3)
	byteOrder.PutUint32(tiff[14:18], 1)
	byteOrder.PutUint16(tiff[18:20], orientation)
	return append([]byte("Exif\x00\x00"), tiff...)
}

func assertImageCornerColors(t *testing.T, img image.Image, want [4]color.NRGBA) {
	t.Helper()
	bounds := img.Bounds()
	const inset = 3
	points := [4]image.Point{
		{X: bounds.Min.X + inset, Y: bounds.Min.Y + inset},
		{X: bounds.Max.X - 1 - inset, Y: bounds.Min.Y + inset},
		{X: bounds.Min.X + inset, Y: bounds.Max.Y - 1 - inset},
		{X: bounds.Max.X - 1 - inset, Y: bounds.Max.Y - 1 - inset},
	}
	for index, point := range points {
		red, green, blue, _ := img.At(point.X, point.Y).RGBA()
		got := color.NRGBA{R: uint8(red >> 8), G: uint8(green >> 8), B: uint8(blue >> 8), A: 255}
		if colorChannelDifference(got.R, want[index].R) > 60 ||
			colorChannelDifference(got.G, want[index].G) > 60 ||
			colorChannelDifference(got.B, want[index].B) > 60 {
			t.Fatalf("corner %d at %v = %#v, want approximately %#v", index, point, got, want[index])
		}
	}
}

func colorChannelDifference(left, right uint8) int {
	difference := int(left) - int(right)
	if difference < 0 {
		return -difference
	}
	return difference
}

func injectJPEGAPP1(t *testing.T, input, payload []byte) []byte {
	t.Helper()
	if len(input) < 2 || !bytes.Equal(input[:2], []byte{0xff, 0xd8}) || len(payload)+2 > 0xffff {
		t.Fatal("invalid JPEG test fixture")
	}
	output := make([]byte, 0, len(input)+len(payload)+4)
	output = append(output, input[:2]...)
	output = append(output, 0xff, 0xe1, byte((len(payload)+2)>>8), byte(len(payload)+2))
	output = append(output, payload...)
	output = append(output, input[2:]...)
	return output
}

func pngHeaderOnly(width, height uint32) []byte {
	output := append([]byte(nil), []byte("\x89PNG\r\n\x1a\n")...)
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:4], width)
	binary.BigEndian.PutUint32(ihdr[4:8], height)
	ihdr[8] = 8
	ihdr[9] = 2
	output = appendPNGChunk(output, "IHDR", ihdr)
	return appendPNGChunk(output, "IEND", nil)
}

func appendPNGChunk(output []byte, chunkType string, data []byte) []byte {
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(len(data)))
	output = append(output, length...)
	chunkStart := len(output)
	output = append(output, chunkType...)
	output = append(output, data...)
	checksum := make([]byte, 4)
	binary.BigEndian.PutUint32(checksum, crc32.ChecksumIEEE(output[chunkStart:]))
	return append(output, checksum...)
}

func buildTestPDF(t *testing.T, pageTexts []string, encrypted bool) []byte {
	t.Helper()
	if len(pageTexts) == 0 {
		t.Fatal("PDF fixture needs at least one page")
	}

	fontObject := 3 + len(pageTexts)*2
	encryptionObject := 0
	objectCount := fontObject
	if encrypted {
		encryptionObject = fontObject + 1
		objectCount++
	}
	objects := make([]string, objectCount+1)
	objects[1] = "<< /Type /Catalog /Pages 2 0 R >>"

	kids := make([]string, 0, len(pageTexts))
	for index, text := range pageTexts {
		pageObject := 3 + index*2
		contentObject := pageObject + 1
		kids = append(kids, fmt.Sprintf("%d 0 R", pageObject))
		objects[pageObject] = fmt.Sprintf(
			"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 %d 0 R >> >> /Contents %d 0 R >>",
			fontObject, contentObject,
		)
		content := "q Q"
		if text != "" {
			content = fmt.Sprintf("BT /F1 12 Tf 72 720 Td (%s) Tj ET", escapePDFString(text))
		}
		objects[contentObject] = fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content)
	}
	objects[2] = fmt.Sprintf("<< /Type /Pages /Count %d /Kids [%s] >>", len(pageTexts), strings.Join(kids, " "))
	widths := strings.TrimSpace(strings.Repeat("600 ", 256))
	objects[fontObject] = fmt.Sprintf("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding /FirstChar 0 /LastChar 255 /Widths [%s] >>", widths)
	if encrypted {
		owner := strings.Repeat("O", 32)
		user := strings.Repeat("U", 32)
		objects[encryptionObject] = fmt.Sprintf("<< /Filter /Standard /V 1 /R 2 /O (%s) /U (%s) /P -4 >>", owner, user)
	}

	var output bytes.Buffer
	output.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")
	offsets := make([]int, objectCount+1)
	for objectNumber := 1; objectNumber <= objectCount; objectNumber++ {
		offsets[objectNumber] = output.Len()
		fmt.Fprintf(&output, "%d 0 obj\n%s\nendobj\n", objectNumber, objects[objectNumber])
	}
	xrefOffset := output.Len()
	fmt.Fprintf(&output, "xref\n0 %d\n0000000000 65535 f \n", objectCount+1)
	for objectNumber := 1; objectNumber <= objectCount; objectNumber++ {
		fmt.Fprintf(&output, "%010d 00000 n \n", offsets[objectNumber])
	}
	fmt.Fprintf(&output, "trailer\n<< /Size %d /Root 1 0 R", objectCount+1)
	if encrypted {
		fmt.Fprintf(&output, " /Encrypt %d 0 R", encryptionObject)
	}
	fmt.Fprintf(&output, " >>\nstartxref\n%d\n%%%%EOF\n", xrefOffset)
	return output.Bytes()
}

func escapePDFString(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "(", "\\(")
	return strings.ReplaceAll(value, ")", "\\)")
}

func baseDOCXFiles(body string) map[string][]byte {
	return map[string][]byte{
		docxContentTypesName: []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="` + docxMain + `"/></Types>`),
		"_rels/.rels":        []byte(`<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`),
		docxDocumentName:     []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:document xmlns:w="` + wordprocessingMLNamespace + `" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><w:body>` + body + `</w:body></w:document>`),
	}
}

func buildTestDOCX(t *testing.T, files map[string][]byte) []byte {
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
			t.Fatalf("create ZIP entry %q: %v", name, err)
		}
		if _, err := entry.Write(files[name]); err != nil {
			t.Fatalf("write ZIP entry %q: %v", name, err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatalf("close ZIP fixture: %v", err)
	}
	return output.Bytes()
}

func buildTestDOCXZipBomb(t *testing.T) []byte {
	t.Helper()
	files := baseDOCXFiles(`<w:p><w:r><w:t>Hello</w:t></w:r></w:p>`)
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
	bomb, err := archive.Create("word/media/bomb.bin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.CopyN(bomb, zeroReader{}, MaxDOCXUncompressedBytes+1); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

type zeroReader struct{}

func (zeroReader) Read(buffer []byte) (int, error) {
	clear(buffer)
	return len(buffer), nil
}

func docxInput(data []byte) Input {
	return Input{Filename: "document.docx", DeclaredMIME: docxMIME, Data: data}
}

func mutateZIPCentralUncompressedSize(t *testing.T, data []byte, targetName string, size uint32) {
	t.Helper()
	const centralHeaderLength = 46
	for offset := 0; ; {
		relative := bytes.Index(data[offset:], []byte("PK\x01\x02"))
		if relative < 0 {
			break
		}
		offset += relative
		if offset+centralHeaderLength > len(data) {
			t.Fatal("truncated central directory in fixture")
		}
		nameLength := int(binary.LittleEndian.Uint16(data[offset+28 : offset+30]))
		extraLength := int(binary.LittleEndian.Uint16(data[offset+30 : offset+32]))
		commentLength := int(binary.LittleEndian.Uint16(data[offset+32 : offset+34]))
		end := offset + centralHeaderLength + nameLength + extraLength + commentLength
		if end > len(data) {
			t.Fatal("invalid central directory fixture")
		}
		name := string(data[offset+centralHeaderLength : offset+centralHeaderLength+nameLength])
		if name == targetName {
			binary.LittleEndian.PutUint32(data[offset+24:offset+28], size)
			return
		}
		offset = end
	}
	t.Fatalf("central directory entry %q not found", targetName)
}

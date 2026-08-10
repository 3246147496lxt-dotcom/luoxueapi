package chatattachment

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"rsc.io/pdf"
)

func processPDF(data []byte) (result *Result, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			result = nil
			err = attachmentError(
				CodeInvalidPDF,
				"PDF cannot be parsed safely",
				fmt.Errorf("PDF parser panic: %v", recovered),
			)
		}
	}()

	// rsc.io/pdf reads the final 100 bytes while locating startxref. Rejecting
	// smaller inputs here also turns truncated inputs into a stable typed error.
	if len(data) < 100 {
		return nil, attachmentError(CodeInvalidPDF, "PDF is truncated or malformed", nil)
	}

	reader, parseErr := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if parseErr != nil {
		if errors.Is(parseErr, pdf.ErrInvalidPassword) ||
			strings.Contains(strings.ToLower(parseErr.Error()), "encrypted pdf") ||
			bytes.Contains(data, []byte("/Encrypt")) {
			return nil, attachmentError(CodePDFEncrypted, "Encrypted or password-protected PDFs are not supported", parseErr)
		}
		return nil, attachmentError(CodeInvalidPDF, "PDF cannot be parsed", parseErr)
	}
	if !reader.Trailer().Key("Encrypt").IsNull() {
		return nil, attachmentError(CodePDFEncrypted, "Encrypted or password-protected PDFs are not supported", nil)
	}

	pageCount := reader.NumPage()
	if pageCount <= 0 {
		return nil, attachmentError(CodeInvalidPDF, "PDF does not contain a valid page tree", nil)
	}
	if pageCount > MaxPDFPages {
		return nil, attachmentError(CodePDFPageLimitExceeded, "PDF exceeds the 50-page limit", nil)
	}

	extracted := newBoundedText(MaxExtractedTextCharacters)
	for pageNumber := 1; pageNumber <= pageCount; pageNumber++ {
		page := reader.Page(pageNumber)
		if page.V.IsNull() {
			return nil, attachmentError(CodeInvalidPDF, "PDF contains an invalid page", nil)
		}
		pageText := appendPDFPageText(extracted, page.Content().Text)
		if pageText == textLimitExceeded {
			return nil, attachmentError(CodeTextLimitExceeded, "Extracted document text exceeds 50000 characters", nil)
		}
	}

	if extracted.empty() {
		return nil, attachmentError(CodePDFNoText, "PDF does not contain an extractable text layer", nil)
	}
	return &Result{
		Kind:      KindPDF,
		MIMEType:  "application/pdf",
		Extension: ".pdf",
		Text:      extracted.string(),
		PageCount: pageCount,
	}, nil
}

type textAppendStatus uint8

const (
	textAppendOK textAppendStatus = iota
	textLimitExceeded
)

func appendPDFPageText(extracted *boundedText, fragments []pdf.Text) textAppendStatus {
	if len(fragments) == 0 {
		return textAppendOK
	}
	sort.Stable(pdf.TextVertical(fragments))
	if !extracted.empty() && !extracted.append("\n\n") {
		return textLimitExceeded
	}

	first := true
	var lineY, previousRight, previousFontSize float64
	for _, fragment := range fragments {
		if fragment.S == "" {
			continue
		}
		if first {
			lineY = fragment.Y
			first = false
		} else {
			fontSize := math.Max(math.Abs(fragment.FontSize), math.Abs(previousFontSize))
			lineTolerance := math.Max(1, fontSize*0.45)
			switch {
			case math.Abs(fragment.Y-lineY) > lineTolerance:
				if !extracted.append("\n") {
					return textLimitExceeded
				}
				lineY = fragment.Y
			case fragment.X-previousRight > math.Max(1, fontSize*0.1):
				if !extracted.append(" ") {
					return textLimitExceeded
				}
			}
		}
		if !extracted.append(fragment.S) {
			return textLimitExceeded
		}
		previousRight = fragment.X + math.Max(0, fragment.W)
		previousFontSize = fragment.FontSize
	}
	return textAppendOK
}

package chatattachment

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"

	_ "golang.org/x/image/webp"
)

func processImage(data []byte, expected supportedType) (result *Result, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			result = nil
			err = attachmentError(
				CodeInvalidImage,
				"Image cannot be decoded safely",
				fmt.Errorf("image decoder panic: %v", recovered),
			)
		}
	}()

	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, attachmentError(CodeInvalidImage, "Image cannot be decoded", err)
	}
	if !imageFormatMatches(format, expected.mimeType) {
		return nil, attachmentError(CodeTypeMismatch, "Decoded image format does not match its declared type", nil)
	}
	if err := validateImageDimensions(config.Width, config.Height); err != nil {
		return nil, err
	}

	decoded, decodedFormat, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, attachmentError(CodeInvalidImage, "Image cannot be decoded", err)
	}
	if decodedFormat != format {
		return nil, attachmentError(CodeTypeMismatch, "Image decoders disagreed about the attachment format", nil)
	}
	bounds := decoded.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if err := validateImageDimensions(width, height); err != nil {
		return nil, err
	}
	if width != config.Width || height != config.Height {
		return nil, attachmentError(CodeInvalidImage, "Image dimensions changed while decoding", nil)
	}
	if format == "jpeg" {
		decoded = applyEXIFOrientation(decoded, jpegEXIFOrientation(data))
		bounds = decoded.Bounds()
		width, height = bounds.Dx(), bounds.Dy()
		if err := validateImageDimensions(width, height); err != nil {
			return nil, err
		}
	}

	var output bytes.Buffer
	outputMIME := expected.mimeType
	outputExtension := expected.extension
	switch format {
	case "jpeg":
		if err := jpeg.Encode(&output, decoded, &jpeg.Options{Quality: 90}); err != nil {
			return nil, attachmentError(CodeInvalidImage, "Image cannot be safely re-encoded", err)
		}
	case "png", "webp":
		encoder := png.Encoder{CompressionLevel: png.DefaultCompression}
		if err := encoder.Encode(&output, decoded); err != nil {
			return nil, attachmentError(CodeInvalidImage, "Image cannot be safely re-encoded", err)
		}
		outputMIME = "image/png"
		outputExtension = ".png"
	default:
		return nil, attachmentError(CodeUnsupportedType, "Image type is not supported", nil)
	}

	return &Result{
		Kind:          KindImage,
		MIMEType:      outputMIME,
		Extension:     outputExtension,
		SanitizedData: output.Bytes(),
		Width:         width,
		Height:        height,
	}, nil
}

func validateImageDimensions(width, height int) error {
	if width <= 0 || height <= 0 {
		return attachmentError(CodeInvalidImage, "Image dimensions are invalid", nil)
	}
	if width > MaxImageDimension || height > MaxImageDimension || int64(width) > MaxImagePixels/int64(height) {
		return attachmentError(
			CodeImageDimensionsExceeded,
			fmt.Sprintf("Image must not exceed %d pixels on one side or %d total pixels", MaxImageDimension, MaxImagePixels),
			nil,
		)
	}
	return nil
}

func imageFormatMatches(format, mimeType string) bool {
	switch format {
	case "jpeg":
		return mimeType == "image/jpeg"
	case "png":
		return mimeType == "image/png"
	case "webp":
		return mimeType == "image/webp"
	default:
		return false
	}
}

package chatattachment

import (
	"bytes"
	"encoding/binary"
	"image"
)

const (
	maxJPEGMetadataScanBytes = 1 << 20
	maxJPEGMetadataSegments  = 256
	maxExifIFDEntries        = 1024
)

// jpegEXIFOrientation reads only the bounded JPEG metadata area and the TIFF
// IFD0 orientation entry. Any malformed or unusually large metadata safely
// falls back to orientation 1; image decoding and re-encoding still proceed.
func jpegEXIFOrientation(data []byte) uint16 {
	if len(data) < 4 || data[0] != 0xff || data[1] != 0xd8 {
		return 1
	}
	scanLimit := len(data)
	if scanLimit > maxJPEGMetadataScanBytes {
		scanLimit = maxJPEGMetadataScanBytes
	}

	offset := 2
	for segments := 0; segments < maxJPEGMetadataSegments && offset < scanLimit; segments++ {
		if data[offset] != 0xff {
			return 1
		}
		for offset < scanLimit && data[offset] == 0xff {
			offset++
		}
		if offset >= scanLimit {
			return 1
		}
		marker := data[offset]
		offset++
		switch {
		case marker == 0x00:
			return 1
		case marker == 0xd9 || marker == 0xda:
			return 1
		case marker == 0x01 || marker == 0xd8 || marker >= 0xd0 && marker <= 0xd7:
			continue
		}

		if offset+2 > scanLimit {
			return 1
		}
		segmentLength := int(binary.BigEndian.Uint16(data[offset : offset+2]))
		if segmentLength < 2 || segmentLength > scanLimit-offset {
			return 1
		}
		segmentEnd := offset + segmentLength
		payload := data[offset+2 : segmentEnd]
		if marker == 0xe1 && bytes.HasPrefix(payload, []byte("Exif\x00\x00")) {
			return tiffOrientation(payload[6:])
		}
		offset = segmentEnd
	}
	return 1
}

func tiffOrientation(tiff []byte) uint16 {
	if len(tiff) < 8 {
		return 1
	}
	var byteOrder binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		byteOrder = binary.LittleEndian
	case "MM":
		byteOrder = binary.BigEndian
	default:
		return 1
	}
	if byteOrder.Uint16(tiff[2:4]) != 42 {
		return 1
	}

	ifdOffset := uint64(byteOrder.Uint32(tiff[4:8]))
	if ifdOffset > uint64(len(tiff)-2) {
		return 1
	}
	entryCount := uint64(byteOrder.Uint16(tiff[ifdOffset : ifdOffset+2]))
	if entryCount > maxExifIFDEntries {
		return 1
	}
	entriesOffset := ifdOffset + 2
	if entryCount > uint64(len(tiff))/12 || entriesOffset > uint64(len(tiff))-entryCount*12 {
		return 1
	}

	for index := uint64(0); index < entryCount; index++ {
		entryOffset := entriesOffset + index*12
		entry := tiff[entryOffset : entryOffset+12]
		if byteOrder.Uint16(entry[0:2]) != 0x0112 {
			continue
		}
		if byteOrder.Uint16(entry[2:4]) != 3 || byteOrder.Uint32(entry[4:8]) != 1 {
			return 1
		}
		orientation := byteOrder.Uint16(entry[8:10])
		if orientation < 1 || orientation > 8 {
			return 1
		}
		return orientation
	}
	return 1
}

func applyEXIFOrientation(source image.Image, orientation uint16) image.Image {
	if orientation <= 1 || orientation > 8 {
		return source
	}
	sourceBounds := source.Bounds()
	sourceWidth, sourceHeight := sourceBounds.Dx(), sourceBounds.Dy()
	outputWidth, outputHeight := sourceWidth, sourceHeight
	if orientation >= 5 {
		outputWidth, outputHeight = sourceHeight, sourceWidth
	}

	output := image.NewNRGBA(image.Rect(0, 0, outputWidth, outputHeight))
	for outputY := 0; outputY < outputHeight; outputY++ {
		for outputX := 0; outputX < outputWidth; outputX++ {
			var sourceX, sourceY int
			switch orientation {
			case 2: // Mirror horizontally.
				sourceX, sourceY = sourceWidth-1-outputX, outputY
			case 3: // Rotate 180 degrees.
				sourceX, sourceY = sourceWidth-1-outputX, sourceHeight-1-outputY
			case 4: // Mirror vertically.
				sourceX, sourceY = outputX, sourceHeight-1-outputY
			case 5: // Transpose across the top-left/bottom-right diagonal.
				sourceX, sourceY = outputY, outputX
			case 6: // Rotate 90 degrees clockwise.
				sourceX, sourceY = outputY, sourceHeight-1-outputX
			case 7: // Transpose across the top-right/bottom-left diagonal.
				sourceX, sourceY = sourceWidth-1-outputY, sourceHeight-1-outputX
			case 8: // Rotate 90 degrees counterclockwise.
				sourceX, sourceY = sourceWidth-1-outputY, outputX
			}
			output.Set(outputX, outputY, source.At(sourceBounds.Min.X+sourceX, sourceBounds.Min.Y+sourceY))
		}
	}
	return output
}

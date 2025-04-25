// Copied from https://github.com/disintegration/imaging/blob/v1.6.2/io.go#L285-L422
// The MIT License (MIT)
// Copyright (c) 2012 Grigory Dryapak

package orientation

import (
	"encoding/binary"
	"image"
	"io"

	"github.com/disintegration/imaging"
)

// Orientation is an EXIF flag that specifies the transformation
// that should be applied to image to display it correctly.
type Orientation int

const (
	Unspecified Orientation = 0
	Normal      Orientation = 1
	FlipH       Orientation = 2
	Rotate180   Orientation = 3
	FlipV       Orientation = 4
	Transpose   Orientation = 5
	Rotate270   Orientation = 6
	Transverse  Orientation = 7
	Rotate90    Orientation = 8
)

func (o Orientation) ApplyToDimensions(w, h int) (int, int) {
	switch o {
	case Unspecified, Normal, FlipH, FlipV, Rotate180:
		return w, h
	case Rotate90, Rotate270, Transpose, Transverse:
		return h, w
	default:
		return w, h
	}
}

func (o Orientation) Fix(img image.Image) image.Image {
	switch o {
	case Normal:
	case FlipH:
		img = imaging.FlipH(img)
	case FlipV:
		img = imaging.FlipV(img)
	case Rotate90:
		img = imaging.Rotate90(img)
	case Rotate180:
		img = imaging.Rotate180(img)
	case Rotate270:
		img = imaging.Rotate270(img)
	case Transpose:
		img = imaging.Transpose(img)
	case Transverse:
		img = imaging.Transverse(img)
	}
	return img
}

// Read tries to read the orientation EXIF flag from image data in r.
// If the EXIF data block is not found or the orientation flag is not found
// or any other error occures while reading the data, it returns the
// Unspecified (0) value.
func Read(r io.Reader) Orientation {
	const (
		markerSOI      = 0xffd8
		markerAPP1     = 0xffe1
		exifHeader     = 0x45786966
		byteOrderBE    = 0x4d4d
		byteOrderLE    = 0x4949
		orientationTag = 0x0112
	)

	// Check if JPEG SOI marker is present.
	var soi uint16
	if err := binary.Read(r, binary.BigEndian, &soi); err != nil {
		return Unspecified
	}
	if soi != markerSOI {
		return Unspecified // Missing JPEG SOI marker.
	}

	// Find JPEG APP1 marker.
	for {
		var marker, size uint16
		if err := binary.Read(r, binary.BigEndian, &marker); err != nil {
			return Unspecified
		}
		if err := binary.Read(r, binary.BigEndian, &size); err != nil {
			return Unspecified
		}
		if marker>>8 != 0xff {
			return Unspecified // Invalid JPEG marker.
		}
		if marker == markerAPP1 {
			break
		}
		if size < 2 {
			return Unspecified // Invalid block size.
		}
		if _, err := io.CopyN(io.Discard, r, int64(size-2)); err != nil {
			return Unspecified
		}
	}

	// Check if EXIF header is present.
	var header uint32
	if err := binary.Read(r, binary.BigEndian, &header); err != nil {
		return Unspecified
	}
	if header != exifHeader {
		return Unspecified
	}
	if _, err := io.CopyN(io.Discard, r, 2); err != nil {
		return Unspecified
	}

	// Read byte order information.
	var (
		byteOrderTag uint16
		byteOrder    binary.ByteOrder
	)
	if err := binary.Read(r, binary.BigEndian, &byteOrderTag); err != nil {
		return Unspecified
	}
	switch byteOrderTag {
	case byteOrderBE:
		byteOrder = binary.BigEndian
	case byteOrderLE:
		byteOrder = binary.LittleEndian
	default:
		return Unspecified // Invalid byte order flag.
	}
	if _, err := io.CopyN(io.Discard, r, 2); err != nil {
		return Unspecified
	}

	// Skip the EXIF offset.
	var offset uint32
	if err := binary.Read(r, byteOrder, &offset); err != nil {
		return Unspecified
	}
	if offset < 8 {
		return Unspecified // Invalid offset value.
	}
	if _, err := io.CopyN(io.Discard, r, int64(offset-8)); err != nil {
		return Unspecified
	}

	// Read the number of tags.
	var numTags uint16
	if err := binary.Read(r, byteOrder, &numTags); err != nil {
		return Unspecified
	}

	// Find the orientation tag.
	for i := 0; i < int(numTags); i++ {
		var tag uint16
		if err := binary.Read(r, byteOrder, &tag); err != nil {
			return Unspecified
		}
		if tag != orientationTag {
			if _, err := io.CopyN(io.Discard, r, 10); err != nil {
				return Unspecified
			}
			continue
		}
		if _, err := io.CopyN(io.Discard, r, 6); err != nil {
			return Unspecified
		}
		var val uint16
		if err := binary.Read(r, byteOrder, &val); err != nil {
			return Unspecified
		}
		if val < 1 || val > 8 {
			return Unspecified // Invalid tag value.
		}
		return Orientation(val)
	}
	return Unspecified // Missing orientation tag.
}

//       ___  _____  ____
//      / _ \/  _/ |/_/ /____ ______ _
//     / ___// /_>  </ __/ -_) __/  ' \
//    /_/  /___/_/|_|\__/\__/_/ /_/_/_/
//
//    Copyright 2017 Eliuk Blau
//
//    This Source Code Form is subject to the terms of the Mozilla Public
//    License, v. 2.0. If a copy of the MPL was not distributed with this
//    file, You can obtain one at http://mozilla.org/MPL/2.0/.

package ansimage

import (
	"errors"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"  // initialize decoder
	_ "image/jpeg" // initialize decoder
	_ "image/png"  // initialize decoder
	"io"
	"os"

	"github.com/disintegration/imaging"
	_ "golang.org/x/image/bmp"  // initialize decoder
	_ "golang.org/x/image/tiff" // initialize decoder
	_ "golang.org/x/image/webp" // initialize decoder

	"go.mau.fi/tcell"

	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/ui/messages/tstring"
)

var (
	// ErrHeightNonMoT happens when ANSImage height is not a Multiple of Two value.
	ErrHeightNonMoT = errors.New("ANSImage: height must be a Multiple of Two value")

	// ErrInvalidBoundsMoT happens when ANSImage height or width are invalid values (Multiple of Two).
	ErrInvalidBoundsMoT = errors.New("ANSImage: height or width must be >=2")

	// ErrOutOfBounds happens when ANSI-pixel coordinates are out of ANSImage bounds.
	ErrOutOfBounds = errors.New("ANSImage: out of bounds")
)

// ANSIpixel represents a pixel of an ANSImage.
type ANSIpixel struct {
	Brightness uint8
	R, G, B    uint8
	upper      bool
	source     *ANSImage
}

// ANSImage represents an image encoded in ANSI escape codes.
type ANSImage struct {
	h, w     int
	maxprocs int
	bgR      uint8
	bgG      uint8
	bgB      uint8
	pixmap   [][]*ANSIpixel
}

func (ai *ANSImage) Pixmap() [][]*ANSIpixel {
	return ai.pixmap
}

// Height gets total rows of ANSImage.
func (ai *ANSImage) Height() int {
	return ai.h
}

// Width gets total columns of ANSImage.
func (ai *ANSImage) Width() int {
	return ai.w
}

// SetMaxProcs sets the maximum number of parallel goroutines to render the ANSImage
// (user should manually sets `runtime.GOMAXPROCS(max)` before to this change takes effect).
func (ai *ANSImage) SetMaxProcs(max int) {
	ai.maxprocs = max
}

// GetMaxProcs gets the maximum number of parallels goroutines to render the ANSImage.
func (ai *ANSImage) GetMaxProcs() int {
	return ai.maxprocs
}

// SetAt sets ANSI-pixel color (RBG) and brightness in coordinates (y,x).
func (ai *ANSImage) SetAt(y, x int, r, g, b, brightness uint8) error {
	if y >= 0 && y < ai.h && x >= 0 && x < ai.w {
		ai.pixmap[y][x].R = r
		ai.pixmap[y][x].G = g
		ai.pixmap[y][x].B = b
		ai.pixmap[y][x].Brightness = brightness
		ai.pixmap[y][x].upper = y%2 == 0
		return nil
	}
	return ErrOutOfBounds
}

// GetAt gets ANSI-pixel in coordinates (y,x).
func (ai *ANSImage) GetAt(y, x int) (*ANSIpixel, error) {
	if y >= 0 && y < ai.h && x >= 0 && x < ai.w {
		return &ANSIpixel{
				R:          ai.pixmap[y][x].R,
				G:          ai.pixmap[y][x].G,
				B:          ai.pixmap[y][x].B,
				Brightness: ai.pixmap[y][x].Brightness,
				upper:      ai.pixmap[y][x].upper,
				source:     ai.pixmap[y][x].source,
			},
			nil
	}
	return nil, ErrOutOfBounds
}

// Render returns the ANSI-compatible string form of ANSImage.
// (Nice info for ANSI True Colour - https://gist.github.com/XVilka/8346728)
func (ai *ANSImage) Render() []tstring.TString {
	type renderData struct {
		row    int
		render tstring.TString
	}

	rows := make([]tstring.TString, ai.h/2)
	for y := 0; y < ai.h; y += ai.maxprocs {
		ch := make(chan renderData, ai.maxprocs)
		for n, row := 0, y; (n <= ai.maxprocs) && (2*row+1 < ai.h); n, row = n+1, y+n {
			go func(row, y int) {
				defer func() {
					err := recover()
					if err != nil {
						debug.Print("Panic rendering ANSImage:", err)
						ch <- renderData{row: row, render: tstring.NewColorTString("ERROR", tcell.ColorRed)}
					}
				}()
				str := make(tstring.TString, ai.w)
				for x := 0; x < ai.w; x++ {
					topPixel := ai.pixmap[y][x]
					topColor := tcell.NewRGBColor(int32(topPixel.R), int32(topPixel.G), int32(topPixel.B))

					bottomPixel := ai.pixmap[y+1][x]
					bottomColor := tcell.NewRGBColor(int32(bottomPixel.R), int32(bottomPixel.G), int32(bottomPixel.B))

					str[x] = tstring.Cell{
						Char:  '▄',
						Style: tcell.StyleDefault.Background(topColor).Foreground(bottomColor),
					}
				}
				ch <- renderData{row: row, render: str}
			}(row, 2*row)
		}
		for n, row := 0, y; (n <= ai.maxprocs) && (2*row+1 < ai.h); n, row = n+1, y+n {
			data := <-ch
			rows[data.row] = data.render
		}
	}
	return rows
}

// New creates a new empty ANSImage ready to draw on it.
func New(h, w int, bg color.Color) (*ANSImage, error) {
	if h%2 != 0 {
		return nil, ErrHeightNonMoT
	}

	if h < 2 || w < 2 {
		return nil, ErrInvalidBoundsMoT
	}

	r, g, b, _ := bg.RGBA()
	ansimage := &ANSImage{
		h: h, w: w,
		maxprocs: 1,
		bgR:      uint8(r),
		bgG:      uint8(g),
		bgB:      uint8(b),
		pixmap:   nil,
	}

	ansimage.pixmap = func() [][]*ANSIpixel {
		v := make([][]*ANSIpixel, h)
		for y := 0; y < h; y++ {
			v[y] = make([]*ANSIpixel, w)
			for x := 0; x < w; x++ {
				v[y][x] = &ANSIpixel{
					R:          0,
					G:          0,
					B:          0,
					Brightness: 0,
					source:     ansimage,
					upper:      y%2 == 0,
				}
			}
		}
		return v
	}()

	return ansimage, nil
}

// NewFromReader creates a new ANSImage from an io.Reader.
// Background color is used to fill when image has transparency or dithering mode is enabled
// Dithering mode is used to specify the way that ANSImage render ANSI-pixels (char/block elements).
func NewFromReader(reader io.Reader, bg color.Color) (*ANSImage, error) {
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, err
	}

	return createANSImage(img, bg)
}

// NewScaledFromReader creates a new scaled ANSImage from an io.Reader.
// Background color is used to fill when image has transparency or dithering mode is enabled
// Dithering mode is used to specify the way that ANSImage render ANSI-pixels (char/block elements).
func NewScaledFromReader(reader io.Reader, y, x int, bg color.Color) (*ANSImage, error) {
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, err
	}

	img = imaging.Resize(img, x, y, imaging.Lanczos)

	return createANSImage(img, bg)
}

// NewFromFile creates a new ANSImage from a file.
// Background color is used to fill when image has transparency or dithering mode is enabled
// Dithering mode is used to specify the way that ANSImage render ANSI-pixels (char/block elements).
func NewFromFile(name string, bg color.Color) (*ANSImage, error) {
	reader, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return NewFromReader(reader, bg)
}

// NewScaledFromFile creates a new scaled ANSImage from a file.
// Background color is used to fill when image has transparency or dithering mode is enabled
// Dithering mode is used to specify the way that ANSImage render ANSI-pixels (char/block elements).
func NewScaledFromFile(name string, y, x int, bg color.Color) (*ANSImage, error) {
	reader, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return NewScaledFromReader(reader, y, x, bg)
}

// createANSImage loads data from an image and returns an ANSImage.
// Background color is used to fill when image has transparency or dithering mode is enabled
// Dithering mode is used to specify the way that ANSImage render ANSI-pixels (char/block elements).
func createANSImage(img image.Image, bg color.Color) (*ANSImage, error) {
	var rgbaOut *image.RGBA
	bounds := img.Bounds()

	// do compositing only if background color has no transparency (thank you @disq for the idea!)
	// (info - http://stackoverflow.com/questions/36595687/transparent-pixel-color-go-lang-image)
	if _, _, _, a := bg.RGBA(); a >= 0xffff {
		rgbaOut = image.NewRGBA(bounds)
		draw.Draw(rgbaOut, bounds, image.NewUniform(bg), image.ZP, draw.Src)
		draw.Draw(rgbaOut, bounds, img, image.ZP, draw.Over)
	} else {
		if v, ok := img.(*image.RGBA); ok {
			rgbaOut = v
		} else {
			rgbaOut = image.NewRGBA(bounds)
			draw.Draw(rgbaOut, bounds, img, image.ZP, draw.Src)
		}
	}

	yMin, xMin := bounds.Min.Y, bounds.Min.X
	yMax, xMax := bounds.Max.Y, bounds.Max.X

	// always sets an even number of ANSIPixel rows...
	yMax = yMax - yMax%2 // one for upper pixel and another for lower pixel --> without dithering

	ansimage, err := New(yMax, xMax, bg)
	if err != nil {
		return nil, err
	}

	for y := yMin; y < yMax; y++ {
		for x := xMin; x < xMax; x++ {
			v := rgbaOut.RGBAAt(x, y)
			if err := ansimage.SetAt(y, x, v.R, v.G, v.B, 0); err != nil {
				return nil, err
			}
		}
	}

	return ansimage, nil
}

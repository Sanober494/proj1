// Package png allows for loading png images and applying
// image filtering effects on them.
package png

import (
	"image"
	"image/draw"
	"image/png"
	"os"
)

// bytesPerPixel is the size of one pixel in an image.RGBA64 buffer:
// four channels (R, G, B, A) with two bytes each, stored big-endian.
const bytesPerPixel = 8

// Image represents a PNG image that effects can be applied to.
//
// Every effect reads from cur and writes into next. Once all workers have
// finished writing, Swap exchanges the two pointers so that cur always holds
// the most recent result. This makes chaining effects free of pixel copies.
type Image struct {
	cur    *image.RGBA64   // The pixels after the most recently applied effect
	next   *image.RGBA64   // Scratch buffer the next effect writes into
	Bounds image.Rectangle // The size of the image
}

// Load returns an Image that was loaded from filePath.
func Load(filePath string) (*Image, error) {
	inReader, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer inReader.Close()

	decoded, err := png.Decode(inReader)
	if err != nil {
		return nil, err
	}

	bounds := decoded.Bounds()
	cur := image.NewRGBA64(bounds)
	draw.Draw(cur, bounds, decoded, bounds.Min, draw.Src)

	return &Image{cur: cur, next: image.NewRGBA64(bounds), Bounds: bounds}, nil
}

// Save writes the current state of the image (i.e., the result of all effects
// applied so far) to filePath.
func (img *Image) Save(filePath string) error {
	outWriter, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer outWriter.Close()

	// Speed matters more than file size here: encoding is a sequential part
	// of the program, so a cheaper compression level helps overall speedup.
	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	return encoder.Encode(outWriter, img.cur)
}

// Height returns the number of pixel rows in the image. Effects are applied
// to row ranges [0, Height()).
func (img *Image) Height() int {
	return img.Bounds.Dy()
}

// Swap makes the output of the last effect the input of the next one. It must
// only be called after every worker has finished its rows for the effect.
func (img *Image) Swap() {
	img.cur, img.next = img.next, img.cur
}

package png

import (
	"encoding/binary"
	"fmt"
	"math"
)

// kernelSize is the width and height of all convolution kernels.
const kernelSize = 3

// Convolution kernels, stored as flat row-major arrays.
var (
	sharpenKernel = [9]float64{0, -1, 0, -1, 5, -1, 0, -1, 0}
	edgeKernel    = [9]float64{-1, -1, -1, -1, 8, -1, -1, -1, -1}
	blurKernel    = [9]float64{1 / 9.0, 1 / 9.0, 1 / 9.0, 1 / 9.0, 1 / 9.0, 1 / 9.0, 1 / 9.0, 1 / 9.0, 1 / 9.0}
)

// ApplyEffect applies one effect ("S", "E", "B" or "G") to the pixel rows
// [yStart, yEnd) of the image, reading from cur and writing to next.
//
// Different goroutines may call ApplyEffect concurrently as long as their row
// ranges do not overlap. Once all rows are done the caller must call Swap
// before applying the next effect.
func (img *Image) ApplyEffect(effect string, yStart, yEnd int) {
	switch effect {
	case "G":
		img.grayscale(yStart, yEnd)
	case "S":
		img.convolve(&sharpenKernel, yStart, yEnd)
	case "E":
		img.convolve(&edgeKernel, yStart, yEnd)
	case "B":
		img.convolve(&blurKernel, yStart, yEnd)
	default:
		panic(fmt.Sprintf("png: unknown effect %q", effect))
	}
}

// Grayscale applies the grayscale effect to the whole image.
func (img *Image) Grayscale() {
	img.ApplyEffect("G", 0, img.Height())
	img.Swap()
}

// grayscale replaces R, G and B of every pixel in the rows by their average.
func (img *Image) grayscale(yStart, yEnd int) {
	width := img.Bounds.Dx()
	for y := yStart; y < yEnd; y++ {
		in := img.cur.Pix[y*img.cur.Stride:]
		out := img.next.Pix[y*img.next.Stride:]
		for x := 0; x < width; x++ {
			p := x * bytesPerPixel
			r := uint32(binary.BigEndian.Uint16(in[p:]))
			g := uint32(binary.BigEndian.Uint16(in[p+2:]))
			b := uint32(binary.BigEndian.Uint16(in[p+4:]))
			gray := clamp(float64(r+g+b) / 3)
			binary.BigEndian.PutUint16(out[p:], gray)
			binary.BigEndian.PutUint16(out[p+2:], gray)
			binary.BigEndian.PutUint16(out[p+4:], gray)
			copy(out[p+6:p+8], in[p+6:p+8]) // alpha is left untouched
		}
	}
}

// convolve applies a 3x3 kernel to the rows [yStart, yEnd). Pixels outside the
// image are treated as zero, which is the same as skipping those kernel taps.
func (img *Image) convolve(kernel *[9]float64, yStart, yEnd int) {
	width, height := img.Bounds.Dx(), img.Bounds.Dy()
	radius := kernelSize / 2

	for y := yStart; y < yEnd; y++ {
		out := img.next.Pix[y*img.next.Stride:]
		dyMin, dyMax := maxInt(-radius, -y), minInt(radius, height-1-y)

		for x := 0; x < width; x++ {
			dxMin, dxMax := maxInt(-radius, -x), minInt(radius, width-1-x)
			var r, g, b float64

			for dy := dyMin; dy <= dyMax; dy++ {
				in := img.cur.Pix[(y+dy)*img.cur.Stride:]
				for dx := dxMin; dx <= dxMax; dx++ {
					k := kernel[(dy+radius)*kernelSize+dx+radius]
					p := (x + dx) * bytesPerPixel
					r += k * float64(binary.BigEndian.Uint16(in[p:]))
					g += k * float64(binary.BigEndian.Uint16(in[p+2:]))
					b += k * float64(binary.BigEndian.Uint16(in[p+4:]))
				}
			}

			p := x * bytesPerPixel
			binary.BigEndian.PutUint16(out[p:], clamp(r))
			binary.BigEndian.PutUint16(out[p+2:], clamp(g))
			binary.BigEndian.PutUint16(out[p+4:], clamp(b))
			copy(out[p+6:p+8], img.cur.Pix[y*img.cur.Stride+p+6:]) // alpha
		}
	}
}

// clamp limits comp to the range [0, 65535].
func clamp(comp float64) uint16 {
	return uint16(math.Min(65535, math.Max(0, comp)))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

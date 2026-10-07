package app

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/lkarlslund/jetkvm-desktop/pkg/logging"
)

// yuvShaderSrc turns a 4:2:0 picture packed by packI420 into RGB on the GPU.
// The packed image holds four bytes per texel: the luma rows come first, then
// one row per chroma row with Cb in its first half and Cr in its second. The
// conversion is the JFIF one image/draw uses, so colours match the CPU path.
var yuvShaderSrc = []byte(`//kage:unit pixels

package main

// LumaSize is the picture's width and height in pixels.
var LumaSize vec2

func byteAt(index float, row float) float {
	texel := imageSrc0At(imageSrc0Origin() + vec2(floor(index/4)+0.5, row+0.5))
	ch := mod(index, 4)
	if ch < 0.5 {
		return texel.r
	}
	if ch < 1.5 {
		return texel.g
	}
	if ch < 2.5 {
		return texel.b
	}
	return texel.a
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	p := floor(dstPos.xy - imageDstOrigin())
	c := floor(p / 2)
	y := byteAt(p.x, p.y)
	cb := byteAt(c.x, LumaSize.y+c.y) - 128.0/255.0
	cr := byteAt(LumaSize.x/2+c.x, LumaSize.y+c.y) - 128.0/255.0
	rgb := vec3(y+1.402*cr, y-0.344136*cb-0.714136*cr, y+1.772*cb)
	return vec4(clamp(rgb, 0, 1), 1)
}
`)

// yuvConverter uploads decoded video as raw YCbCr planes and converts them to
// RGB on the GPU, which saves converting every frame on the CPU and allocating
// an RGBA copy of it.
type yuvConverter struct {
	shader *ebiten.Shader
	failed bool
	planes *ebiten.Image
	buf    []byte
}

// canConvert reports whether img can take the GPU path. The packing needs
// whole texels per luma row and whole chroma samples per picture.
func (c *yuvConverter) canConvert(img *image.YCbCr) bool {
	if c.failed || img.SubsampleRatio != image.YCbCrSubsampleRatio420 {
		return false
	}
	b := img.Rect
	return b.Dx() > 0 && b.Dx()%4 == 0 && b.Dy() > 0 && b.Dy()%2 == 0 && b.Min.X%2 == 0 && b.Min.Y%2 == 0
}

// draw renders img into dst, which must be img's size. It reports false when the
// shader is unavailable, leaving the caller to convert on the CPU.
func (c *yuvConverter) draw(dst *ebiten.Image, img *image.YCbCr) bool {
	if c.shader == nil {
		shader, err := ebiten.NewShader(yuvShaderSrc)
		if err != nil {
			log := logging.Subsystem("video")
			log.Warn().Err(err).Msg("YCbCr shader unavailable, converting on the CPU")
			c.failed = true
			return false
		}
		c.shader = shader
	}

	w, h := img.Rect.Dx(), img.Rect.Dy()
	pw, ph := w/4, h+h/2
	if c.planes == nil || c.planes.Bounds().Dx() != pw || c.planes.Bounds().Dy() != ph {
		if c.planes != nil {
			c.planes.Deallocate()
		}
		c.planes = ebiten.NewImage(pw, ph)
		c.buf = make([]byte, w*ph)
	}
	packI420(c.buf, img)
	c.planes.WritePixels(c.buf)

	fw, fh := float32(w), float32(h)
	vertices := []ebiten.Vertex{
		{DstX: 0, DstY: 0, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: fw, DstY: 0, SrcX: float32(pw), SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: 0, DstY: fh, SrcX: 0, SrcY: float32(ph), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: fw, DstY: fh, SrcX: float32(pw), SrcY: float32(ph), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	}
	op := &ebiten.DrawTrianglesShaderOptions{
		Uniforms: map[string]any{"LumaSize": []float32{fw, fh}},
		Images:   [4]*ebiten.Image{c.planes},
		Blend:    ebiten.BlendCopy,
	}
	dst.DrawTrianglesShader(vertices, []uint16{0, 1, 2, 1, 2, 3}, c.shader, op)
	return true
}

// packI420 lays img out for yuvShaderSrc: h luma rows of w bytes, then h/2 rows
// each holding a Cb row followed by a Cr row of w/2 bytes.
func packI420(dst []byte, img *image.YCbCr) {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	x0, y0 := img.Rect.Min.X, img.Rect.Min.Y
	for r := 0; r < h; r++ {
		off := img.YOffset(x0, y0+r)
		copy(dst[r*w:(r+1)*w], img.Y[off:off+w])
	}
	cw := w / 2
	for r := 0; r < h/2; r++ {
		off := img.COffset(x0, y0+2*r)
		row := dst[(h+r)*w : (h+r+1)*w]
		copy(row[:cw], img.Cb[off:off+cw])
		copy(row[cw:], img.Cr[off:off+cw])
	}
}

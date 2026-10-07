package app

import (
	"image"
	"testing"
)

func TestPackI420(t *testing.T) {
	// A sub-image, so the planes have strides wider than the picture.
	full := image.NewYCbCr(image.Rect(0, 0, 12, 6), image.YCbCrSubsampleRatio420)
	for i := range full.Y {
		full.Y[i] = byte(i)
	}
	for i := range full.Cb {
		full.Cb[i] = byte(100 + i)
		full.Cr[i] = byte(200 + i)
	}
	img := full.SubImage(image.Rect(2, 2, 10, 6)).(*image.YCbCr)

	var c yuvConverter
	if !c.canConvert(img) {
		t.Fatal("canConvert rejected an 8x4 4:2:0 picture")
	}
	w, h := 8, 4
	dst := make([]byte, w*(h+h/2))
	packI420(dst, img)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if got, want := dst[y*w+x], img.YCbCrAt(2+x, 2+y).Y; got != want {
				t.Fatalf("Y(%d,%d) = %d, want %d", x, y, got, want)
			}
		}
	}
	for cy := 0; cy < h/2; cy++ {
		row := dst[(h+cy)*w : (h+cy+1)*w]
		for cx := 0; cx < w/2; cx++ {
			want := img.YCbCrAt(2+2*cx, 2+2*cy)
			if row[cx] != want.Cb || row[w/2+cx] != want.Cr {
				t.Fatalf("chroma(%d,%d) = %d/%d, want %d/%d", cx, cy, row[cx], row[w/2+cx], want.Cb, want.Cr)
			}
		}
	}
}

func TestCanConvertRejectsUnpackableSizes(t *testing.T) {
	var c yuvConverter
	for _, img := range []*image.YCbCr{
		image.NewYCbCr(image.Rect(0, 0, 6, 4), image.YCbCrSubsampleRatio420),
		image.NewYCbCr(image.Rect(0, 0, 8, 3), image.YCbCrSubsampleRatio420),
		image.NewYCbCr(image.Rect(0, 0, 8, 4), image.YCbCrSubsampleRatio444),
	} {
		if c.canConvert(img) {
			t.Errorf("canConvert accepted %v %v", img.Rect, img.SubsampleRatio)
		}
	}
}

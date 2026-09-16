//go:build darwin && cgo && !ios

package video

import (
	"image"
	"testing"

	openh264 "github.com/Azunyan1111/openh264-go"
)

// TestVideoToolboxDecodesOpenH264Stream round-trips the emulator's test pattern
// through the openh264 encoder and the VideoToolbox decoder.
func TestVideoToolboxDecodesOpenH264Stream(t *testing.T) {
	// 540 is not a multiple of 16, so this also covers frame cropping. Like the
	// emulator, force a keyframe every 30 frames so the decoder has to handle
	// new parameter sets mid-stream.
	const width, height, frames, keyframeInterval = 960, 540, 90, 30

	params := openh264.NewEncoderParams()
	params.Width = width
	params.Height = height
	params.BitRate = width * height * 4
	params.MaxFrameRate = 30
	params.UsageType = openh264.ScreenContentRealTime
	params.IntraPeriod = 60
	encoder, err := openh264.NewEncoder(params)
	if err != nil {
		t.Fatal(err)
	}
	defer closeEncoder(encoder)

	decoder, err := newVideoToolboxDecoder()
	if err != nil {
		t.Fatal(err)
	}
	defer decoder.Close()

	decoded := 0
	for i := 0; i < frames; i++ {
		if i%keyframeInterval == 0 {
			_ = encoder.ForceKeyFrame()
		}
		src := newPatternFrame(width, height, i)
		payload, err := encoder.Encode(src)
		if err != nil {
			t.Fatal(err)
		}
		if len(payload) == 0 {
			continue
		}
		img, err := decoder.Decode(payload)
		if err != nil {
			t.Fatalf("frame %d: %v", i, err)
		}
		if img == nil {
			continue
		}
		decoded++
		if got := img.Bounds().Size(); got.X != width || got.Y != height {
			t.Fatalf("frame %d: decoded size %v, want %dx%d", i, got, width, height)
		}
		if diff := meanAbsDiff(src.Y, img.Y, src.YStride, img.YStride, width, height); diff > 8 {
			t.Fatalf("frame %d: mean luma difference %.2f too large", i, diff)
		}
		if diff := meanAbsDiff(src.Cb, img.Cb, src.CStride, img.CStride, width/2, height/2); diff > 8 {
			t.Fatalf("frame %d: mean Cb difference %.2f too large", i, diff)
		}
	}
	if decoded < frames-1 {
		t.Fatalf("decoded %d of %d frames", decoded, frames)
	}
}

func meanAbsDiff(a, b []byte, strideA, strideB, width, height int) float64 {
	var total int
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			d := int(a[y*strideA+x]) - int(b[y*strideB+x])
			if d < 0 {
				d = -d
			}
			total += d
		}
	}
	return float64(total) / float64(width*height)
}

// TestVideoToolboxPausedKeepsLatestFrame checks what a hidden window relies on:
// while paused the decoder produces no images, but it keeps decoding, so the
// picture it hands back afterwards is the current one and not the one from
// before the pause.
func TestVideoToolboxPausedKeepsLatestFrame(t *testing.T) {
	const width, height, frames = 320, 176, 20

	params := openh264.NewEncoderParams()
	params.Width = width
	params.Height = height
	params.BitRate = width * height * 4
	params.MaxFrameRate = 30
	params.UsageType = openh264.ScreenContentRealTime
	params.IntraPeriod = 60
	encoder, err := openh264.NewEncoder(params)
	if err != nil {
		t.Fatal(err)
	}
	defer closeEncoder(encoder)

	decoder, err := newVideoToolboxDecoder()
	if err != nil {
		t.Fatal(err)
	}
	defer decoder.Close()
	defer SetPaused(false)

	var last *image.YCbCr
	for i := 0; i < frames; i++ {
		if i == 0 {
			_ = encoder.ForceKeyFrame()
		}
		// Hide the window halfway, after the decoder has a picture to hold on to.
		SetPaused(i >= frames/2)
		src := newPatternFrame(width, height, i)
		payload, err := encoder.Encode(src)
		if err != nil {
			t.Fatal(err)
		}
		if len(payload) == 0 {
			continue
		}
		img, err := decoder.Decode(payload)
		if err != nil {
			t.Fatalf("frame %d: %v", i, err)
		}
		if paused.Load() && img != nil {
			t.Fatalf("frame %d: decoded an image while paused", i)
		}
		last = src
	}

	// Coming back must show the last frame that arrived while hidden.
	img, err := decoder.LatestFrame()
	if err != nil {
		t.Fatal(err)
	}
	if img == nil {
		t.Fatal("no frame held after pause")
	}
	if diff := meanAbsDiff(last.Y, img.Y, last.YStride, img.YStride, width, height); diff > 8 {
		t.Fatalf("held frame is not the last one decoded: mean luma difference %.2f", diff)
	}
}

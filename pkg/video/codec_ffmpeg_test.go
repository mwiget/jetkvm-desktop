//go:build ffmpeg && !ios

package video

import (
	"bytes"
	"image"
	"testing"

	openh264 "github.com/Azunyan1111/openh264-go"
)

// TestFFmpegMatchesOpenH264 decodes the emulator's test pattern with FFmpeg, in
// software and (where the machine has it) on the GPU, and with openh264. H.264
// decoding is exact, so every picture must come out the same.
func TestFFmpegMatchesOpenH264(t *testing.T) {
	// 540 is not a multiple of 16, so this also covers frame cropping.
	const width, height, frames, keyframeInterval = 960, 540, 60, 30

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

	var stream [][]byte
	for i := 0; i < frames; i++ {
		if i%keyframeInterval == 0 {
			_ = encoder.ForceKeyFrame()
		}
		payload, err := encoder.Encode(newPatternFrame(width, height, i))
		if err != nil {
			t.Fatal(err)
		}
		if len(payload) > 0 {
			stream = append(stream, append([]byte(nil), payload...))
		}
	}

	reference, err := openh264.NewDecoder(bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}
	defer reference.Close()
	want := decodeAll(t, reference, stream)

	for _, hardware := range []bool{false, true} {
		decoder, name, err := newFFmpegDecoder(hardware)
		if err != nil {
			t.Fatal(err)
		}
		if hardware && name != "ffmpeg-vaapi" {
			t.Logf("no VA-API device; skipping the GPU decoder")
			decoder.Close()
			continue
		}
		got := decodeAll(t, decoder, stream)
		decoder.Close()
		if len(got) != len(want) {
			t.Fatalf("%s decoded %d pictures, openh264 %d", name, len(got), len(want))
		}
		for i := range want {
			if got[i].Rect != want[i].Rect {
				t.Fatalf("%s picture %d is %v, want %v", name, i, got[i].Rect, want[i].Rect)
			}
			if !bytes.Equal(got[i].Y, want[i].Y) || !bytes.Equal(got[i].Cb, want[i].Cb) || !bytes.Equal(got[i].Cr, want[i].Cr) {
				t.Fatalf("%s picture %d differs from openh264", name, i)
			}
		}
		t.Logf("%s decoded %d pictures identical to openh264", name, len(got))
	}
}

func decodeAll(t *testing.T, decoder h264Decoder, stream [][]byte) []*image.YCbCr {
	t.Helper()
	var out []*image.YCbCr
	for i, payload := range stream {
		img, err := decoder.Decode(payload)
		if err != nil {
			t.Fatalf("access unit %d: %v", i, err)
		}
		if img != nil {
			out = append(out, img)
		}
	}
	return out
}

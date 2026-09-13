//go:build ios

package video

import (
	"context"
	"errors"
	"image"

	"github.com/pion/webrtc/v4"
)

// The bundled openh264 static libraries are macOS-only. Until an iOS decoder
// exists, frames are dropped so the rest of the session still works.
type discardDecoder struct{}

func (discardDecoder) Decode([]byte) (*image.YCbCr, error) { return nil, nil }

func (discardDecoder) Close() error { return nil }

func newH264Decoder() (h264Decoder, error) {
	return discardDecoder{}, nil
}

func StartTestPattern(context.Context, int, int, int, *webrtc.TrackLocalStaticSample) error {
	return errors.New("video test pattern is not supported on iOS")
}

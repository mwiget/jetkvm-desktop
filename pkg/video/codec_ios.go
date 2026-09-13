//go:build ios

package video

import (
	"context"
	"errors"

	"github.com/pion/webrtc/v4"
)

func newH264Decoder() (h264Decoder, error) {
	return newVideoToolboxDecoder()
}

func StartTestPattern(context.Context, int, int, int, *webrtc.TrackLocalStaticSample) error {
	return errors.New("video test pattern is not supported on iOS")
}

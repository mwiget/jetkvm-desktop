//go:build !ios

package video

import (
	"bytes"
	"context"
	"os"
	"runtime"
	"strings"
	"time"

	openh264 "github.com/Azunyan1111/openh264-go"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"

	"github.com/lkarlslund/jetkvm-desktop/pkg/logging"
)

// decoderEnv picks the H.264 decoder: "openh264", "software" (FFmpeg without
// the GPU) or, by default, the fastest one the build and machine offer.
const decoderEnv = "JETKVM_DESKTOP_DECODER"

// preferredDecoder opens a faster decoder than openh264 when the build has
// one (see codec_ffmpeg.go). With hardware it may decode on the GPU. It also
// returns a name for the log.
var preferredDecoder func(hardware bool) (h264Decoder, string, error)

// Desktop decoders run on the CPU, so decoding is put off while the window
// can't be seen; see deferDecodeWhileIdle in stream.go.
func init() {
	deferDecodeWhileIdle = true
}

func newH264Decoder() (h264Decoder, error) {
	log := logging.Subsystem("video")
	choice := strings.ToLower(strings.TrimSpace(os.Getenv(decoderEnv)))
	if preferredDecoder != nil && choice != "openh264" {
		decoder, name, err := preferredDecoder(choice != "software")
		if err == nil {
			log.Info().Str("decoder", name).Msg("video decoder")
			return decoder, nil
		}
		log.Warn().Err(err).Msg("falling back to openh264")
	}
	log.Info().Str("decoder", "openh264").Msg("video decoder")
	return openh264.NewDecoder(bytes.NewReader(nil))
}

func StartTestPattern(ctx context.Context, width, height, fps int, track *webrtc.TrackLocalStaticSample) error {
	params := openh264.NewEncoderParams()
	params.Width = width
	params.Height = height
	params.BitRate = width * height * 4
	params.MaxFrameRate = float32(fps)
	params.UsageType = openh264.ScreenContentRealTime
	params.EnableFrameSkip = false
	params.IntraPeriod = uint(fps * 2)

	encoder, err := openh264.NewEncoder(params)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(time.Second / time.Duration(fps))
	go func() {
		defer ticker.Stop()
		defer closeEncoder(encoder)

		frameIndex := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if frameIndex%(fps*2) == 0 {
					_ = encoder.ForceKeyFrame()
				}
				img := newPatternFrame(width, height, frameIndex)
				payload, err := encoder.Encode(img)
				if err != nil {
					return
				}
				if len(payload) == 0 {
					frameIndex++
					continue
				}
				if err := track.WriteSample(media.Sample{
					Data:     payload,
					Duration: time.Second / time.Duration(fps),
				}); err != nil {
					return
				}
				frameIndex++
			}
		}
	}()

	return nil
}

func closeEncoder(encoder *openh264.Encoder) {
	// openh264-go encoder teardown can block indefinitely on Windows in CI.
	// The process is short-lived in tests and the OS will reclaim resources.
	if runtime.GOOS == "windows" {
		return
	}
	_ = encoder.Close()
}

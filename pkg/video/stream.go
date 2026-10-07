package video

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media/samplebuilder"

	"github.com/lkarlslund/jetkvm-desktop/pkg/logging"
)

const decodeStatsInterval = 5 * time.Second

// paused stops turning the incoming video into images while the app's window is
// hidden. Decoding itself carries on, because the decoder's reference frames
// have to stay current for the picture to be right when the window comes back;
// what is skipped is copying every frame into Go memory for a window that draws
// nothing. It is process-wide because it describes the window rather than one
// stream, so it also applies to a stream that reconnects while still hidden.
var paused atomic.Bool

// SetPaused hides or shows the app's window. Pair it with Stream.Refresh on the
// way back, to put the picture the screen changed to while hidden on screen.
func SetPaused(hidden bool) {
	paused.Store(hidden)
}

// deferDecodeWhileIdle is set by platforms that decode on the CPU, where
// decoding video for a window nobody can see is most of what a connected client
// costs. See windowIdle.
var deferDecodeWhileIdle bool

// maxDeferredBytes bounds the access units held back while the window is idle.
// A device that sends no keyframes would otherwise grow the backlog forever, so
// past this it is decoded and dropped, as if the window were showing.
const maxDeferredBytes = 16 << 20

// The game loop ticks at the display rate while its window is on screen, and
// falls to about once a second when the compositor stops asking it to draw (for
// a window on another workspace, say). drawTickLast is when it last ticked, and
// drawTickFast when it last ticked soon after the previous tick.
var drawTickLast, drawTickFast atomic.Int64

const (
	drawTickFastGap = 250 * time.Millisecond
	drawIdleAfter   = 500 * time.Millisecond
)

// NoteDrawTick is called on every game loop tick that shows video.
func NoteDrawTick() {
	now := time.Now().UnixNano()
	if prev := drawTickLast.Swap(now); prev != 0 && now-prev < int64(drawTickFastGap) {
		drawTickFast.Store(now)
	}
}

// windowIdle reports whether the game loop has stopped ticking at a drawing
// rate. Until it has ticked fast once it is not idle, so a stream that nothing
// draws (in tests, say) decodes as usual.
func windowIdle(now time.Time) bool {
	fast := drawTickFast.Load()
	return fast != 0 && now.UnixNano()-fast > int64(drawIdleAfter)
}

// containsIDR reports whether an Annex B access unit holds an IDR slice, which
// decoding can start from.
func containsIDR(data []byte) bool {
	for _, nal := range splitAnnexB(data) {
		if nalType(nal) == nalTypeIDR {
			return true
		}
	}
	return false
}

// frameHolder is implemented by decoders that keep the last decoded picture.
type frameHolder interface {
	LatestFrame() (*image.YCbCr, error)
}

// h264Decoder turns Annex B access units into frames. The implementation is
// platform specific; see codec_openh264.go and codec_ios.go.
type h264Decoder interface {
	Decode(data []byte) (*image.YCbCr, error)
	Close() error
}

type Frame struct {
	Image image.Image
	At    time.Time
}

type Stream struct {
	mu         sync.RWMutex
	latest     *Frame
	lastErr    error
	frameCh    chan Frame
	closeOnce  sync.Once
	cancelFunc context.CancelFunc
	decoder    h264Decoder
	onClose    func() error
}

func NewStream() *Stream {
	return &Stream{frameCh: make(chan Frame, 4)}
}

func (s *Stream) Latest() *Frame {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.latest
}

func (s *Stream) Err() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastErr
}

func (s *Stream) publish(frame Frame) {
	s.mu.Lock()
	s.latest = &frame
	s.mu.Unlock()

	select {
	case s.frameCh <- frame:
	default:
	}
}

func (s *Stream) setError(err error) {
	s.mu.Lock()
	s.lastErr = err
	s.mu.Unlock()
}

func (s *Stream) Frames() <-chan Frame {
	return s.frameCh
}

// Refresh publishes the picture the decoder is holding. Nothing was published
// while the window was hidden, and a static screen sends nothing more, so
// without this the window would come back showing what it last drew.
func (s *Stream) Refresh() {
	holder, ok := s.decoder.(frameHolder)
	if !ok {
		return
	}
	img, err := holder.LatestFrame()
	if err != nil || img == nil {
		return
	}
	s.publish(Frame{Image: img, At: time.Now()})
}

func (s *Stream) Close() {
	s.closeOnce.Do(func() {
		if s.cancelFunc != nil {
			s.cancelFunc()
		}
		if s.onClose != nil {
			_ = s.onClose()
		}
	})
}

func AttachRemoteTrack(parent context.Context, track *webrtc.TrackRemote) (*Stream, error) {
	if track.Codec().MimeType != webrtc.MimeTypeH264 {
		return nil, fmt.Errorf("unsupported codec %s", track.Codec().MimeType)
	}

	ctx, cancel := context.WithCancel(parent)
	stream := NewStream()
	stream.cancelFunc = cancel

	decoder, err := newH264Decoder()
	if err != nil {
		cancel()
		return nil, err
	}
	stream.decoder = decoder
	stream.onClose = decoder.Close

	go func() {
		defer stream.Close()

		log := logging.Subsystem("video")
		var samples, frames, empty, failures, deferred, keyframes int
		lastReport := time.Now()

		// While the window is idle, access units wait here from the latest
		// keyframe on instead of being decoded; decoding them once it shows
		// again brings the picture up to date.
		var backlog [][]byte
		var backlogBytes int
		decode := func(payload []byte) (*image.YCbCr, bool) {
			img, err := decoder.Decode(payload)
			if err != nil {
				failures++
				if prev := stream.Err(); prev == nil || prev.Error() != err.Error() {
					log.Debug().Err(err).Int("sample", samples).Msg("video decode error")
				}
				stream.setError(err)
				return nil, false
			}
			return img, true
		}
		drainBacklog := func() {
			for _, payload := range backlog {
				decode(payload)
			}
			backlog, backlogBytes = backlog[:0], 0
		}

		// Screen-content H.264 keyframes can span well over hundreds of RTP packets,
		// especially on real 1080p devices. A too-small samplebuilder buffer drops
		// fragmented access units before the decoder ever sees a complete frame.
		sb := samplebuilder.New(
			4096,
			&codecs.H264Packet{},
			track.Codec().ClockRate,
			samplebuilder.WithMaxTimeDelay(33*time.Millisecond),
		)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			pkt, _, readErr := track.ReadRTP()
			if readErr != nil {
				return
			}
			sb.Push(pkt)
			for {
				sample := sb.Pop()
				if sample == nil {
					break
				}
				payload := ensureAnnexB(sample.Data)
				if len(payload) == 0 {
					continue
				}
				samples++
				idr := containsIDR(payload)
				if idr {
					keyframes++
				}
				if deferDecodeWhileIdle && windowIdle(time.Now()) {
					if idr {
						backlog, backlogBytes = backlog[:0], 0
					}
					backlog = append(backlog, payload)
					backlogBytes += len(payload)
					deferred++
					if backlogBytes > maxDeferredBytes {
						drainBacklog()
					}
				} else {
					if len(backlog) > 0 {
						drainBacklog()
					}
					img, ok := decode(payload)
					switch {
					case !ok:
					case img == nil || paused.Load():
						// Nothing decoded, or a hidden window that draws nothing.
						empty++
					default:
						frames++
						stream.publish(Frame{Image: img, At: time.Now()})
					}
				}
				if now := time.Now(); now.Sub(lastReport) >= decodeStatsInterval {
					log.Debug().
						Int("samples", samples).
						Int("frames", frames).
						Int("empty", empty).
						Int("errors", failures).
						Int("deferred", deferred).
						Int("keyframes", keyframes).
						AnErr("last_error", stream.Err()).
						Msg("video decode stats")
					lastReport = now
				}
			}
		}
	}()

	return stream, nil
}

func newPatternFrame(width, height, frameIndex int) *image.YCbCr {
	img := image.NewYCbCr(image.Rect(0, 0, width, height), image.YCbCrSubsampleRatio420)
	phase := frameIndex % 255

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			yy := uint8((x + y + phase) % 256)
			img.Y[y*img.YStride+x] = yy
		}
	}

	for y := 0; y < height/2; y++ {
		for x := 0; x < width/2; x++ {
			img.Cb[y*img.CStride+x] = uint8((x*2 + phase*3) % 256)
			img.Cr[y*img.CStride+x] = uint8((y*2 + phase*5) % 256)
		}
	}

	drawBox(img, 20+(frameIndex*7)%(max(1, width-120)), 20+(frameIndex*5)%(max(1, height-80)), 100, 60, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	return img
}

func drawBox(img *image.YCbCr, x0, y0, w, h int, c color.Color) {
	ycbcr := color.YCbCrModel.Convert(c).(color.YCbCr)
	for y := max(0, y0); y < minInt(img.Rect.Dy(), y0+h); y++ {
		for x := max(0, x0); x < minInt(img.Rect.Dx(), x0+w); x++ {
			img.Y[y*img.YStride+x] = ycbcr.Y
			img.Cb[(y/2)*img.CStride+(x/2)] = ycbcr.Cb
			img.Cr[(y/2)*img.CStride+(x/2)] = ycbcr.Cr
		}
	}
}

func ensureAnnexB(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	if bytes.HasPrefix(data, []byte{0x00, 0x00, 0x00, 0x01}) || bytes.HasPrefix(data, []byte{0x00, 0x00, 0x01}) {
		return data
	}
	return append([]byte{0x00, 0x00, 0x00, 0x01}, data...)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

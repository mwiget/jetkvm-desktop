//go:build ffmpeg && !ios

package video

/*
#cgo pkg-config: libavcodec libavutil
#include <stdlib.h>
#include <string.h>
#include <libavcodec/avcodec.h>
#include <libavutil/hwcontext.h>
#include <libavutil/pixdesc.h>

typedef struct {
	AVCodecContext *ctx;
	AVPacket *pkt;
	AVFrame *frame;
	AVFrame *out;
	AVFrame *sw;
	uint8_t *buf;
	int buf_cap;
} ffdec;

// ffdec_get_format takes a VA-API surface when the decoder offers one, and
// otherwise the first format in system memory.
static enum AVPixelFormat ffdec_get_format(AVCodecContext *ctx, const enum AVPixelFormat *fmts) {
	const enum AVPixelFormat *p;
	for (p = fmts; *p != AV_PIX_FMT_NONE; p++) {
		if (*p == AV_PIX_FMT_VAAPI) {
			return *p;
		}
	}
	for (p = fmts; *p != AV_PIX_FMT_NONE; p++) {
		if (!(av_pix_fmt_desc_get(*p)->flags & AV_PIX_FMT_FLAG_HWACCEL)) {
			return *p;
		}
	}
	return AV_PIX_FMT_NONE;
}

static void ffdec_free(ffdec *d) {
	if (d == NULL) {
		return;
	}
	avcodec_free_context(&d->ctx);
	av_packet_free(&d->pkt);
	av_frame_free(&d->frame);
	av_frame_free(&d->out);
	av_frame_free(&d->sw);
	free(d->buf);
	free(d);
}

// ffdec_new opens an H.264 decoder. With want_hw it decodes on the GPU through
// VA-API when a device opens, and reports that in *hw.
static ffdec *ffdec_new(int want_hw, int *hw, int *err) {
	*hw = 0;
	const AVCodec *codec = avcodec_find_decoder(AV_CODEC_ID_H264);
	if (codec == NULL) {
		*err = AVERROR_DECODER_NOT_FOUND;
		return NULL;
	}
	ffdec *d = calloc(1, sizeof(ffdec));
	if (d == NULL) {
		*err = AVERROR(ENOMEM);
		return NULL;
	}
	d->ctx = avcodec_alloc_context3(codec);
	d->pkt = av_packet_alloc();
	d->frame = av_frame_alloc();
	d->out = av_frame_alloc();
	d->sw = av_frame_alloc();
	if (d->ctx == NULL || d->pkt == NULL || d->frame == NULL || d->out == NULL || d->sw == NULL) {
		ffdec_free(d);
		*err = AVERROR(ENOMEM);
		return NULL;
	}
	// Every access unit comes out as soon as it is decoded: frame threading
	// would hold pictures back by a frame per thread.
	d->ctx->flags |= AV_CODEC_FLAG_LOW_DELAY;
	d->ctx->thread_type = FF_THREAD_SLICE;
	d->ctx->thread_count = 0;
	if (want_hw) {
		AVBufferRef *dev = NULL;
		if (av_hwdevice_ctx_create(&dev, AV_HWDEVICE_TYPE_VAAPI, NULL, NULL, 0) == 0) {
			d->ctx->hw_device_ctx = dev;
			d->ctx->get_format = ffdec_get_format;
			*hw = 1;
		}
	}
	int rv = avcodec_open2(d->ctx, codec, NULL);
	if (rv < 0) {
		ffdec_free(d);
		*err = rv;
		return NULL;
	}
	return d;
}

// ffdec_decode feeds one access unit. It returns 1 when that completed a
// picture, 0 when not, and a negative AVERROR on failure.
static int ffdec_decode(ffdec *d, const uint8_t *data, int len) {
	int need = len + AV_INPUT_BUFFER_PADDING_SIZE;
	if (d->buf_cap < need) {
		uint8_t *buf = realloc(d->buf, need);
		if (buf == NULL) {
			return AVERROR(ENOMEM);
		}
		d->buf = buf;
		d->buf_cap = need;
	}
	memcpy(d->buf, data, len);
	memset(d->buf + len, 0, AV_INPUT_BUFFER_PADDING_SIZE);
	d->pkt->data = d->buf;
	d->pkt->size = len;

	int rv = avcodec_send_packet(d->ctx, d->pkt);
	if (rv < 0 && rv != AVERROR(EAGAIN)) {
		return rv;
	}
	int got = 0;
	for (;;) {
		rv = avcodec_receive_frame(d->ctx, d->frame);
		if (rv == AVERROR(EAGAIN) || rv == AVERROR_EOF) {
			break;
		}
		if (rv < 0) {
			return rv;
		}
		av_frame_unref(d->out);
		av_frame_move_ref(d->out, d->frame);
		got = 1;
	}
	return got;
}

// ffdec_picture returns the latest picture in system memory, copying it off
// the GPU first when it was decoded there.
static AVFrame *ffdec_picture(ffdec *d, int *err) {
	if (d->out->format != AV_PIX_FMT_VAAPI) {
		return d->out;
	}
	av_frame_unref(d->sw);
	int rv = av_hwframe_transfer_data(d->sw, d->out, 0);
	if (rv < 0) {
		*err = rv;
		return NULL;
	}
	return d->sw;
}

static int ffdec_copyable(const AVFrame *f) {
	return f->format == AV_PIX_FMT_YUV420P || f->format == AV_PIX_FMT_YUVJ420P || f->format == AV_PIX_FMT_NV12;
}

// ffdec_copy writes f, which ffdec_copyable accepted, into 4:2:0 planes.
static void ffdec_copy(const AVFrame *f, uint8_t *y, int ys, uint8_t *cb, uint8_t *cr, int cs) {
	int w = f->width, h = f->height;
	int cw = (w + 1) / 2, ch = (h + 1) / 2;
	for (int r = 0; r < h; r++) {
		memcpy(y + r * ys, f->data[0] + r * f->linesize[0], w);
	}
	if (f->format == AV_PIX_FMT_NV12) {
		for (int r = 0; r < ch; r++) {
			const uint8_t *s = f->data[1] + r * f->linesize[1];
			uint8_t *u = cb + r * cs, *v = cr + r * cs;
			for (int x = 0; x < cw; x++) {
				u[x] = s[2 * x];
				v[x] = s[2 * x + 1];
			}
		}
		return;
	}
	for (int r = 0; r < ch; r++) {
		memcpy(cb + r * cs, f->data[1] + r * f->linesize[1], cw);
		memcpy(cr + r * cs, f->data[2] + r * f->linesize[2], cw);
	}
}

static const char *ffdec_format_name(const AVFrame *f) {
	const char *name = av_get_pix_fmt_name(f->format);
	return name ? name : "unknown";
}
*/
import "C"

import (
	"fmt"
	"image"
	"io"
	"sync"
	"unsafe"
)

func init() {
	preferredDecoder = newFFmpegDecoder
}

// ffmpegDecoder decodes with libavcodec, on the GPU through VA-API where the
// machine has it. It is built only with the ffmpeg tag, because it links the
// system's FFmpeg libraries.
type ffmpegDecoder struct {
	mu sync.Mutex
	d  *C.ffdec
	hw bool
}

func newFFmpegDecoder(hardware bool) (h264Decoder, string, error) {
	var hw, cerr C.int
	want := C.int(0)
	if hardware {
		want = 1
	}
	d := C.ffdec_new(want, &hw, &cerr)
	if d == nil {
		return nil, "", fmt.Errorf("open FFmpeg H.264 decoder: %w", averror(cerr))
	}
	name := "ffmpeg"
	if hw != 0 {
		name = "ffmpeg-vaapi"
	}
	return &ffmpegDecoder{d: d, hw: hw != 0}, name, nil
}

func (d *ffmpegDecoder) Decode(data []byte) (*image.YCbCr, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.d == nil {
		return nil, io.EOF
	}
	if len(data) == 0 {
		return nil, nil
	}
	rv := C.ffdec_decode(d.d, (*C.uint8_t)(unsafe.Pointer(&data[0])), C.int(len(data)))
	if rv < 0 {
		return nil, fmt.Errorf("decode failed: %w", averror(rv))
	}
	if rv == 0 {
		return nil, nil
	}
	var cerr C.int
	f := C.ffdec_picture(d.d, &cerr)
	if f == nil {
		return nil, fmt.Errorf("read decoded picture: %w", averror(cerr))
	}
	if C.ffdec_copyable(f) == 0 {
		return nil, fmt.Errorf("unsupported decoded pixel format %s", C.GoString(C.ffdec_format_name(f)))
	}
	img := image.NewYCbCr(image.Rect(0, 0, int(f.width), int(f.height)), image.YCbCrSubsampleRatio420)
	C.ffdec_copy(f,
		(*C.uint8_t)(unsafe.Pointer(&img.Y[0])), C.int(img.YStride),
		(*C.uint8_t)(unsafe.Pointer(&img.Cb[0])), (*C.uint8_t)(unsafe.Pointer(&img.Cr[0])), C.int(img.CStride))
	return img, nil
}

func (d *ffmpegDecoder) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	C.ffdec_free(d.d)
	d.d = nil
	return nil
}

func averror(code C.int) error {
	buf := make([]C.char, 128)
	C.av_strerror(code, &buf[0], C.size_t(len(buf)))
	return fmt.Errorf("%s (%d)", C.GoString(&buf[0]), int(code))
}

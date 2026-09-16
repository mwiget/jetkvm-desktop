//go:build darwin && cgo

package video

/*
#cgo LDFLAGS: -framework VideoToolbox -framework CoreMedia -framework CoreVideo -framework CoreFoundation

#include <stdlib.h>
#include <string.h>
#include <CoreFoundation/CoreFoundation.h>
#include <CoreMedia/CoreMedia.h>
#include <CoreVideo/CoreVideo.h>
#include <VideoToolbox/VideoToolbox.h>

typedef struct {
	VTDecompressionSessionRef session;
	CMVideoFormatDescriptionRef format;
	CVPixelBufferRef frame;
	OSStatus status;
} vt_decoder;

static void vt_output(void *refcon, void *frameRefcon, OSStatus status, VTDecodeInfoFlags flags,
                      CVImageBufferRef image, CMTime pts, CMTime duration) {
	vt_decoder *d = (vt_decoder *)refcon;
	d->status = status;
	if (status != noErr || image == NULL) {
		return;
	}
	if (d->frame != NULL) {
		CVPixelBufferRelease(d->frame);
	}
	d->frame = CVPixelBufferRetain(image);
}

static vt_decoder *vt_new(void) {
	return (vt_decoder *)calloc(1, sizeof(vt_decoder));
}

// cgo maps CoreFoundation references to integers, so nil checks live in C.
static int vt_has_session(vt_decoder *d) {
	return d->session != NULL;
}

static void vt_destroy_session(vt_decoder *d) {
	if (d->session == NULL) {
		return;
	}
	VTDecompressionSessionInvalidate(d->session);
	CFRelease(d->session);
	d->session = NULL;
}

static OSStatus vt_create_session(vt_decoder *d) {
	vt_destroy_session(d);

	CFMutableDictionaryRef attrs = CFDictionaryCreateMutable(kCFAllocatorDefault, 1,
		&kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
	// Video-range NV12 keeps the decoded samples untouched, matching openh264.
	int32_t pixelFormat = kCVPixelFormatType_420YpCbCr8BiPlanarVideoRange;
	CFNumberRef pixelFormatNumber = CFNumberCreate(kCFAllocatorDefault, kCFNumberSInt32Type, &pixelFormat);
	CFDictionarySetValue(attrs, kCVPixelBufferPixelFormatTypeKey, pixelFormatNumber);
	CFRelease(pixelFormatNumber);

	VTDecompressionOutputCallbackRecord callback = {vt_output, d};
	OSStatus status = VTDecompressionSessionCreate(kCFAllocatorDefault, d->format, NULL, attrs, &callback, &d->session);
	CFRelease(attrs);
	if (status == noErr) {
		VTSessionSetProperty(d->session, kVTDecompressionPropertyKey_RealTime, kCFBooleanTrue);
	}
	return status;
}

static OSStatus vt_configure(vt_decoder *d, const uint8_t *sps, size_t spsLen, const uint8_t *pps, size_t ppsLen) {
	const uint8_t *sets[2] = {sps, pps};
	size_t sizes[2] = {spsLen, ppsLen};
	CMVideoFormatDescriptionRef format = NULL;
	OSStatus status = CMVideoFormatDescriptionCreateFromH264ParameterSets(kCFAllocatorDefault, 2, sets, sizes, 4, &format);
	if (status != noErr) {
		return status;
	}
	// Always start a new session. Swapping the format into a session that
	// claims to accept it works on macOS, but the iOS simulator's software
	// decoder then rejects every following frame with kVTVideoDecoderBadDataErr.
	if (d->format != NULL) {
		CFRelease(d->format);
	}
	d->format = format;
	return vt_create_session(d);
}

// vt_decode decodes one access unit of length-prefixed (AVCC) NAL units.
static OSStatus vt_decode(vt_decoder *d, const uint8_t *data, size_t len, int *gotFrame) {
	*gotFrame = 0;
	CMBlockBufferRef block = NULL;
	OSStatus status = CMBlockBufferCreateWithMemoryBlock(kCFAllocatorDefault, NULL, len, kCFAllocatorDefault,
		NULL, 0, len, kCMBlockBufferAssureMemoryNowFlag, &block);
	if (status != noErr) {
		return status;
	}
	status = CMBlockBufferReplaceDataBytes(data, block, 0, len);
	if (status != noErr) {
		CFRelease(block);
		return status;
	}
	CMSampleBufferRef sample = NULL;
	size_t sampleSize = len;
	status = CMSampleBufferCreateReady(kCFAllocatorDefault, block, d->format, 1, 0, NULL, 1, &sampleSize, &sample);
	CFRelease(block);
	if (status != noErr) {
		return status;
	}

	if (d->frame != NULL) {
		CVPixelBufferRelease(d->frame);
		d->frame = NULL;
	}
	d->status = noErr;
	VTDecodeInfoFlags info = 0;
	// No asynchronous flag: the output callback runs before this returns.
	status = VTDecompressionSessionDecodeFrame(d->session, sample, 0, NULL, &info);
	CFRelease(sample);
	if (status != noErr) {
		return status;
	}
	if (d->status != noErr) {
		return d->status;
	}
	*gotFrame = d->frame != NULL;
	return noErr;
}

static int vt_has_frame(vt_decoder *d) {
	return d->frame != NULL;
}

static void vt_frame_size(vt_decoder *d, int *width, int *height) {
	*width = (int)CVPixelBufferGetWidth(d->frame);
	*height = (int)CVPixelBufferGetHeight(d->frame);
}

static void vt_copy_frame(vt_decoder *d, uint8_t *dstY, int dstYStride, uint8_t *dstCb, uint8_t *dstCr, int dstCStride) {
	CVPixelBufferRef frame = d->frame;
	CVPixelBufferLockBaseAddress(frame, kCVPixelBufferLock_ReadOnly);

	const uint8_t *srcY = CVPixelBufferGetBaseAddressOfPlane(frame, 0);
	size_t yStride = CVPixelBufferGetBytesPerRowOfPlane(frame, 0);
	size_t yWidth = CVPixelBufferGetWidthOfPlane(frame, 0);
	size_t yHeight = CVPixelBufferGetHeightOfPlane(frame, 0);
	for (size_t row = 0; row < yHeight; row++) {
		memcpy(dstY + row * dstYStride, srcY + row * yStride, yWidth);
	}

	const uint8_t *srcCbCr = CVPixelBufferGetBaseAddressOfPlane(frame, 1);
	size_t cStride = CVPixelBufferGetBytesPerRowOfPlane(frame, 1);
	size_t cWidth = CVPixelBufferGetWidthOfPlane(frame, 1);
	size_t cHeight = CVPixelBufferGetHeightOfPlane(frame, 1);
	for (size_t row = 0; row < cHeight; row++) {
		const uint8_t *src = srcCbCr + row * cStride;
		uint8_t *cb = dstCb + row * dstCStride;
		uint8_t *cr = dstCr + row * dstCStride;
		for (size_t col = 0; col < cWidth; col++) {
			cb[col] = src[2 * col];
			cr[col] = src[2 * col + 1];
		}
	}

	CVPixelBufferUnlockBaseAddress(frame, kCVPixelBufferLock_ReadOnly);
}

static void vt_free(vt_decoder *d) {
	vt_destroy_session(d);
	if (d->format != NULL) {
		CFRelease(d->format);
	}
	if (d->frame != NULL) {
		CVPixelBufferRelease(d->frame);
	}
	free(d);
}
*/
import "C"

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"sync"
	"unsafe"
)

// kVTInvalidSessionErr is returned after the system tears down decoder
// sessions, e.g. when an iOS app returns from the background.
const kVTInvalidSessionErr = -12903

// videoToolboxDecoder decodes Annex B H.264 access units with VideoToolbox,
// using the hardware decoder where available.
type videoToolboxDecoder struct {
	mu     sync.Mutex
	d      *C.vt_decoder
	sps    []byte
	pps    []byte
	sample []byte
}

func newVideoToolboxDecoder() (*videoToolboxDecoder, error) {
	d := C.vt_new()
	if d == nil {
		return nil, errors.New("videotoolbox: allocate decoder")
	}
	return &videoToolboxDecoder{d: d}, nil
}

func (v *videoToolboxDecoder) Decode(data []byte) (*image.YCbCr, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.d == nil {
		return nil, errors.New("videotoolbox: decoder closed")
	}

	sample := v.sample[:0]
	var sps, pps []byte
	var types []byte
	for _, nal := range splitAnnexB(data) {
		types = append(types, nalType(nal))
		switch nalType(nal) {
		case nalTypeSPS:
			sps = nal
		case nalTypePPS:
			pps = nal
		case nalTypeSlice, nalTypeIDR:
			sample = binary.BigEndian.AppendUint32(sample, uint32(len(nal)))
			sample = append(sample, nal...)
		}
	}
	v.sample = sample

	paramsChanged, err := v.updateParameterSets(sps, pps)
	if err != nil {
		return nil, err
	}
	if len(sample) == 0 || C.vt_has_session(v.d) == 0 {
		// Nothing to show, or still waiting for the first keyframe's parameter sets.
		return nil, nil
	}

	gotFrame, status := v.decodeSample(sample)
	if status == kVTInvalidSessionErr {
		if status := C.vt_create_session(v.d); status != 0 {
			return nil, fmt.Errorf("videotoolbox: recreate session: OSStatus %d", status)
		}
		gotFrame, status = v.decodeSample(sample)
	}
	if status != 0 {
		return nil, fmt.Errorf("videotoolbox: decode: OSStatus %d (NAL types %v, parameter sets changed: %t)", status, types, paramsChanged)
	}
	if !gotFrame || paused.Load() {
		// VideoToolbox holds on to the decoded picture, so a hidden window can
		// skip the copy into Go memory -- the bulk of the per-frame cost --
		// and still get the current picture from LatestFrame when it returns.
		return nil, nil
	}
	return v.copyFrame(), nil
}

// LatestFrame copies the picture the decoder is holding, or returns nil when it
// has not decoded one yet.
func (v *videoToolboxDecoder) LatestFrame() (*image.YCbCr, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.d == nil {
		return nil, errors.New("videotoolbox: decoder closed")
	}
	if C.vt_has_frame(v.d) == 0 {
		return nil, nil
	}
	return v.copyFrame(), nil
}

// copyFrame copies the held picture into a new image. The caller holds v.mu and
// has checked that there is a frame.
func (v *videoToolboxDecoder) copyFrame() *image.YCbCr {
	var width, height C.int
	C.vt_frame_size(v.d, &width, &height)
	img := image.NewYCbCr(image.Rect(0, 0, int(width), int(height)), image.YCbCrSubsampleRatio420)
	C.vt_copy_frame(v.d,
		(*C.uint8_t)(unsafe.Pointer(&img.Y[0])), C.int(img.YStride),
		(*C.uint8_t)(unsafe.Pointer(&img.Cb[0])), (*C.uint8_t)(unsafe.Pointer(&img.Cr[0])), C.int(img.CStride))
	return img
}

// updateParameterSets reconfigures the decoder when new SPS or PPS units
// arrive and reports whether it did.
func (v *videoToolboxDecoder) updateParameterSets(sps, pps []byte) (bool, error) {
	if sps == nil && pps == nil {
		return false, nil
	}
	if sps == nil {
		sps = v.sps
	}
	if pps == nil {
		pps = v.pps
	}
	if sps == nil || pps == nil {
		return false, nil
	}
	if C.vt_has_session(v.d) != 0 && bytes.Equal(sps, v.sps) && bytes.Equal(pps, v.pps) {
		return false, nil
	}
	status := C.vt_configure(v.d,
		(*C.uint8_t)(unsafe.Pointer(&sps[0])), C.size_t(len(sps)),
		(*C.uint8_t)(unsafe.Pointer(&pps[0])), C.size_t(len(pps)))
	if status != 0 {
		return false, fmt.Errorf("videotoolbox: configure: OSStatus %d", status)
	}
	v.sps = bytes.Clone(sps)
	v.pps = bytes.Clone(pps)
	return true, nil
}

func (v *videoToolboxDecoder) decodeSample(sample []byte) (bool, C.OSStatus) {
	var gotFrame C.int
	status := C.vt_decode(v.d, (*C.uint8_t)(unsafe.Pointer(&sample[0])), C.size_t(len(sample)), &gotFrame)
	return gotFrame != 0, status
}

func (v *videoToolboxDecoder) Close() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.d != nil {
		C.vt_free(v.d)
		v.d = nil
	}
	return nil
}

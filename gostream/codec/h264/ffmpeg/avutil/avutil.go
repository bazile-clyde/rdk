package avutil

//#cgo CFLAGS: -Wno-deprecated-declarations -I${SRCDIR}/../ffmpeg/include
//#cgo LDFLAGS: -L${SRCDIR}/../ffmpeg/lib -lavformat -lavcodec -lavutil -lavdevice -lavfilter -lswresample -lswscale -lm  -lvdpau -lva-drm -lva-x11 -lva
//#include <libswresample/swresample.h>
//#include <libavutil/error.h>
//#include <stdlib.h>
//static const char *error2string(int code) { return av_err2str(code); }
import "C"
import (
	"github.com/pkg/errors"
	"image"
	"reflect"
	"unsafe"
)

type Frame C.struct_AVFrame

// FrameAlloc Allocate an AVFrame and set its fields to default values. The resulting
// struct must be freed using av_frame_free().
//
// @return An AVFrame filled with default values or NULL on failure.
//
// @note this only allocates the AVFrame itself, not the data buffers. Those
// must be allocated through other means, e.g. with av_frame_get_buffer() or
// manually.
func FrameAlloc() *Frame {
	return (*Frame)(unsafe.Pointer(C.av_frame_alloc()))
}

func SetFrame(f *Frame, w int, h int, pixFmt int) error {
	f.width = C.int(w)
	f.height = C.int(h)
	f.format = C.int(pixFmt)
	if ret := C.av_frame_get_buffer((*C.struct_AVFrame)(unsafe.Pointer(f)), 0 /*alignment*/); ret < 0 {
		return errors.Errorf("error allocating avframe buffer: return value %d", int(ret))
	}
	return nil
}

// FrameMakeWritable Ensure that the frame data is writable, avoiding data copy if possible.
//
// Do nothing if the frame is writable, allocate new buffers and copy the data
// if it is not. Non-refcounted frames behave as non-writable, i.e. a copy
// is always made.
//
// @return 0 on success, a negative AVERROR on error.
//
// @see av_frame_is_writable(), av_buffer_is_writable(),
// av_buffer_make_writable()
func FrameMakeWritable(f *Frame) int {
	return int(C.av_frame_make_writable((*C.struct_AVFrame)(unsafe.Pointer(f))))
}

// FrameUnref Unreference all the buffers referenced by frame and reset the frame fields.
func FrameUnref(f *Frame) {
	C.av_frame_unref((*C.struct_AVFrame)(unsafe.Pointer(f)))
}

func ptr(buf []byte) *C.uint8_t {
	h := (*reflect.SliceHeader)(unsafe.Pointer(&buf))
	return (*C.uint8_t)(unsafe.Pointer(h.Data))
}

func (f *Frame) SetFrameFromImg(img *image.YCbCr) {
	f.data[0] = ptr(img.Y)
	f.data[1] = ptr(img.Cb)
	f.data[2] = ptr(img.Cr)

	w := C.int(img.Bounds().Dx())
	f.linesize[0] = w
	f.linesize[1] = w / 2
	f.linesize[2] = w / 2
}

func (f *Frame) SetFramePTS(pts int64) {
	f.pts = C.int64_t(pts)
}

const (
	ErrorEAGAIN = -11
	Success     = 0
)

func ErrorFromCode(code int) error {
	if code >= 0 {
		return nil
	}

	return errors.New(C.GoString(C.error2string(C.int(code))))
}

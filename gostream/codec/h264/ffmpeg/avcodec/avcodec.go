package avcodec

//#cgo CFLAGS: -I${SRCDIR}/include/libavutil
//#cgo LDFLAGS: ${SRCDIR}/lib/libavutil.a
//#include <libavutil/pixfmt.h>
import "C"

const AV_PIX_FMT_YUV420P = C.AV_PIX_FMT_YUV420P

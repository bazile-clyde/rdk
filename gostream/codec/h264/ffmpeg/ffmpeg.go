package ffmpeg

//#cgo CFLAGS: -I${SRCDIR}/include/libavcodec
//#cgo CFLAGS: -I${SRCDIR}/include/libavdevice
//#cgo CFLAGS: -I${SRCDIR}/include/libavfilter
//#cgo CFLAGS: -I${SRCDIR}/include/libavformat
//#cgo CFLAGS: -I${SRCDIR}/include/libavutil
//#cgo CFLAGS: -I${SRCDIR}/include/libswresample
//#cgo CFLAGS: -I${SRCDIR}/include/libswscale
//#cgo linux LDFLAGS: ${SRCDIR}/lib/libavcodec.a -lm
//#cgo linux LDFLAGS: ${SRCDIR}/lib/libavdevice.a -lm
//#cgo linux LDFLAGS: ${SRCDIR}/lib/libavfilter.a -lm
//#cgo linux LDFLAGS: ${SRCDIR}/lib/libavformat.a -lm
//#cgo linux LDFLAGS: ${SRCDIR}/lib/libavutil.a -lm
//#cgo linux LDFLAGS: ${SRCDIR}/lib/libswresample.a -lm
//#cgo linux LDFLAGS: ${SRCDIR}/lib/libswscale.a -lm
import "C"

package h264

import (
	"github.com/edaniels/golog"
	"go.viam.com/rdk/gostream/codec/h264/ffmpeg/avcodec"

	"go.viam.com/rdk/gostream"
	"go.viam.com/rdk/gostream/codec"
)

var DefaultStreamConfig gostream.StreamConfig

func init() {
	avcodec.RegisterAll()
	DefaultStreamConfig.VideoEncoderFactory = NewEncoderFactory()
}

func NewEncoderFactory() codec.VideoEncoderFactory {
	return &factory{}
}

type factory struct{}

func (f *factory) New(width, height, keyFrameInterval int, logger golog.Logger) (codec.VideoEncoder, error) {
	return NewEncoder(width, height, keyFrameInterval, logger)
}

func (f *factory) MIMEType() string {
	return "video/H264"
}

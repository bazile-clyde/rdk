//go:build linux || darwin

package server

import (
	"github.com/viamrobotics/gostream"
	"github.com/viamrobotics/gostream/codec/h264"
	"github.com/viamrobotics/gostream/codec/opus"
)

func makeStreamConfig() gostream.StreamConfig {
	var streamConfig gostream.StreamConfig
	streamConfig.AudioEncoderFactory = opus.NewEncoderFactory()
	streamConfig.VideoEncoderFactory = h264.NewEncoderFactory()
	return streamConfig
}

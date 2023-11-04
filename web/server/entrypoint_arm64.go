//go:build cgo && !android

package server

import (
	"go.viam.com/rdk/gostream"
	"go.viam.com/rdk/gostream/codec/h264"
	"go.viam.com/rdk/gostream/codec/opus"
	"go.viam.com/rdk/gostream/codec/x264"
	"go.viam.com/rdk/utils"
	"strings"
)

var onRaspberryPi = false

func init() {
	if osInfo, err := utils.DetectOSInformation(); err == nil && strings.Contains(osInfo.Device, "Raspberry Pi") {
		onRaspberryPi = true
	}
}

func makeStreamConfig() gostream.StreamConfig {
	var streamConfig gostream.StreamConfig
	streamConfig.AudioEncoderFactory = opus.NewEncoderFactory()
	if onRaspberryPi {
		streamConfig.VideoEncoderFactory = h264.NewEncoderFactory()
	} else {
		streamConfig.VideoEncoderFactory = x264.NewEncoderFactory()
	}
	return streamConfig
}

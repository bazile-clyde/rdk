package fake

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	"go.viam.com/test"

	"go.viam.com/rdk/components/camera"
)

func TestFakeCamera(t *testing.T) {
	camOri := &Camera{Name: "test", Model: fakeModel}
	cam, err := camera.NewFromReader(context.Background(), camOri, fakeModel, camera.ColorStream)
	test.That(t, err, test.ShouldBeNil)
	stream, err := cam.Stream(context.Background())
	test.That(t, err, test.ShouldBeNil)
	img, _, err := stream.Next(context.Background())
	test.That(t, err, test.ShouldBeNil)
	test.That(t, img.Bounds().Dx(), test.ShouldEqual, 1280)
	test.That(t, img.Bounds().Dy(), test.ShouldEqual, 720)
	pc, err := cam.NextPointCloud(context.Background())
	test.That(t, err, test.ShouldBeNil)
	test.That(t, pc.Size(), test.ShouldEqual, 812050)
	prop, err := cam.Properties(context.Background())
	test.That(t, err, test.ShouldBeNil)
	test.That(t, prop.IntrinsicParams, test.ShouldResemble, fakeIntrinsics)
	test.That(t, prop.DistortionParams, test.ShouldResemble, fakeDistortion)
	err = cam.Close(context.Background())
	test.That(t, err, test.ShouldBeNil)
}

type imageSource struct {
	Images []image.Image
	idx    int
}

// Returns the next image or nil if there are no more images left. This should never error.
func (is *imageSource) Read(_ context.Context) (image.Image, func(), error) {
	if is.idx >= len(is.Images) {
		return nil, func() {}, nil
	}
	img := is.Images[is.idx]
	is.idx++
	return img, func() {}, nil
}

func (is *imageSource) Close(_ context.Context) error {
	return nil
}

func pngToImage(t *testing.T, path string) image.Image {
	t.Helper()
	openBytes, err := os.ReadFile(path)
	test.That(t, err, test.ShouldBeNil)
	img, err := png.Decode(bytes.NewReader(openBytes))
	test.That(t, err, test.ShouldBeNil)
	return img
}

func imageToColor(t *testing.T, img image.Image) string {
	t.Helper()
	r, g, b, a := img.At(0, 0).RGBA()
	rgba := color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)}
	switch rgba {
	case color.RGBA{R: 255, B: 0, G: 0, A: 255}:
		return "red"
	case color.RGBA{R: 0, B: 255, G: 0, A: 255}:
		return "green"
	case color.RGBA{R: 0, B: 0, G: 255, A: 255}:
		return "blue"
	case color.RGBA{R: 255, B: 255, G: 0, A: 255}:
		return "yellow"
	case color.RGBA{R: 255, B: 0, G: 255, A: 255}:
		return "fuchsia"
	case color.RGBA{R: 0, B: 255, G: 255, A: 255}:
		return "cyan"
	default:
		t.Errorf("rgba=%v undefined", rgba)
		return ""
	}
}

func TestCameraStream(t *testing.T) {
	colors := []image.Image{
		pngToImage(t, "data/red.png"),     // 0xff0000
		pngToImage(t, "data/green.png"),   // 0x00ff00
		pngToImage(t, "data/blue.png"),    // 0x0000ff
		pngToImage(t, "data/yellow.png"),  // 0xffff00
		pngToImage(t, "data/fuchsia.png"), // 0xff00ff
		pngToImage(t, "data/cyan.png"),    // 0x00ffff
	}

	imgSource := &imageSource{Images: colors}
	cam, err := camera.NewFromReader(context.Background(), imgSource, fakeModel, camera.ColorStream)
	test.That(t, err, test.ShouldBeNil)

	stream, err := cam.Stream(context.Background())
	test.That(t, err, test.ShouldBeNil)

	// Test all images are returned in order.
	for _, expected := range colors {
		actual, _, err := stream.Next(context.Background())
		test.That(t, err, test.ShouldBeNil)
		test.That(t, actual, test.ShouldNotBeNil)

		actualColor := imageToColor(t, actual)
		expectedColor := imageToColor(t, expected)
		test.That(t, actualColor, test.ShouldEqual, expectedColor)
	}

	// Sanity-check: Test image comparison can fail if two images are not the same
	imgSource.Images = []image.Image{pngToImage(t, "data/red.png")}
	cam, err = camera.NewFromReader(context.Background(), imgSource, fakeModel, camera.ColorStream)
	test.That(t, err, test.ShouldBeNil)

	stream, err = cam.Stream(context.Background())
	test.That(t, err, test.ShouldBeNil)

	expected := pngToImage(t, "data/blue.png")
	actual, _, err := stream.Next(context.Background())
	test.That(t, err, test.ShouldBeNil)

	actualColor := imageToColor(t, actual)
	expectedColor := imageToColor(t, expected)
	test.That(t, actualColor, test.ShouldNotEqual, expectedColor)
}

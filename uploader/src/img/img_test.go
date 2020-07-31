// +build gm

package img

import (
	"config"
	"testing"
	"utils"

	"github.com/openalpr/openalpr"
)

var filename = "test_data/test_image.jpg"

func init() {
	config.Init([]string{})
	config.Opts.FrameDir = "test_output/"
	config.Opts.PlateDir = "test_output/"
}

func TestVersion(t *testing.T) {
	t.Log(GetImageLib())
}

func TestCreatePlateImageGood(t *testing.T) {
	points := []openalpr.AlprCoordinate{
		{
			X: 615,
			Y: 360,
		},
		{
			X: 770,
			Y: 380,
		},
		{
			X: 765,
			Y: 415,
		},
		{
			X: 616,
			Y: 390,
		},
	}
	filebytes, err := CreatePlateImage(filename, points)
	if err != nil {
		t.Fatal(err)
	}
	plate := utils.GetPlateFilename(filename)
	utils.SaveBuffer(plate, filebytes)
}

func TestCreatePlateImageInverted(t *testing.T) {
	// The points are inverted.
	points := []openalpr.AlprCoordinate{
		{
			X: 765,
			Y: 415,
		},
		{
			X: 616,
			Y: 390,
		},
		{
			X: 615,
			Y: 360,
		},
		{
			X: 770,
			Y: 380,
		},
	}
	filebytes, err := CreatePlateImage(filename, points)
	if err != nil {
		t.Fatal(err)
	}
	plate := utils.GetPlateFilename(filename)
	utils.SaveBuffer(plate, filebytes)
}

func TestCreatePlateImageZeros(t *testing.T) {
	points := []openalpr.AlprCoordinate{
		{
			X: 0,
			Y: 0,
		},
		{
			X: 0,
			Y: 0,
		},
		{
			X: 0,
			Y: 0,
		},
		{
			X: 0,
			Y: 0,
		},
	}
	_, err := CreatePlateImage(filename, points)
	if err == nil {
		t.Fatal("Expected error for zero values")
	}
}

func TestCreateFrameThumbnail(t *testing.T) {
	filebytes, err := CreateFrameThumbnail(filename)
	if err != nil {
		t.Fatal(err)
	}
	frame := utils.GetFrameFilename(filename)
	utils.SaveBuffer(frame, filebytes)
}

// +build gm

package img

import (
	"bytes"
	"config"
	"errors"

	"github.com/openalpr/openalpr"
	"github.com/rainycape/magick"
)

// GetImageLib outputs which library we build with: GraphicsMagick or ImageMagick
func GetImageLib() string {
	return magick.Backend()
}

// CreatePlateImage makes a grayscale thumbnail and returns a pointer to the bytes.
func CreatePlateImage(filename string, points []openalpr.AlprCoordinate) (*bytes.Buffer, error) {
	img, err := magick.DecodeFile(filename)
	if err != nil {
		// File doesn't exist?
		return nil, err
	}

	// The four points are not a rectangle, we need to find the bounding rect
	// given as X,Y,Width,Height. Start with arbritrary large int for mins.
	minX := 500000
	minY := 500000
	maxX := 0
	maxY := 0
	for i := 0; i < 4; i++ {
		if points[i].X < minX {
			minX = points[i].X
		}
		if points[i].Y < minY {
			minY = points[i].Y
		}
		if points[i].X > maxX {
			maxX = points[i].X
		}
		if points[i].Y > maxY {
			maxY = points[i].Y
		}
	}
	// Sanity check for max/min
	if maxX-minX <= 0 || maxY-minY <= 0 {
		return nil, errors.New("Min was greater than Max cropping plate out of image")
	}

	var cropRegion = magick.Rect{
		X:      minX,
		Y:      minY,
		Width:  uint(maxX - minX),
		Height: uint(maxY - minY),
	}

	// Create the cropped thumbnail
	croppedPlate, err := img.Crop(cropRegion)
	if err != nil {
		return nil, err
	}

	// Resize the thumb
	resizedCropped, err := croppedPlate.Resize(-1, config.Opts.PlateImageHeight, magick.FLanczos)
	if err != nil {
		return nil, err
	}

	// In-memory bytes of the result.
	var plateBytes bytes.Buffer

	// Set output options
	info := magick.NewInfo()
	info.SetQuality(config.Opts.PlateImageQuality)
	info.SetColorspace(magick.GRAY)

	// Encode the final image.
	err = resizedCropped.Encode(&plateBytes, info)
	if err != nil {
		return nil, err
	}

	return &plateBytes, nil
}

// CreateFrameThumbnail makes a thumbnail and saves it, and returns a pointer to the bytes.
func CreateFrameThumbnail(filename string) (*bytes.Buffer, error) {
	img, err := magick.DecodeFile(filename)
	if err != nil {
		// File doesn't exist?
		return nil, err
	}

	// Resize it.
	resized, err := img.Resize(-1, config.Opts.FrameImageHeight, magick.FLanczos)
	if err != nil {
		return nil, err
	}

	// In-memory bytes of the result.
	var plateBytes bytes.Buffer

	// Set output options
	info := magick.NewInfo()
	info.SetQuality(config.Opts.PlateImageQuality)

	// Encode the final image.
	err = resized.Encode(&plateBytes, info)
	if err != nil {
		return nil, err
	}

	return &plateBytes, nil
}

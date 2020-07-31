package utils

import (
	"bytes"
	"config"
	"errors"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// ExtractTime returns the UTC time from the filepath
func ExtractTime(file string) (*time.Time, error) {
	// This must match the motion.conf format.
	const shortForm = "20060102150405"

	_, filename := filepath.Split(file)

	exploded := strings.Split(filename, "-")

	if len(exploded) != 3 || len(exploded[1]) != 14 {
		now := time.Now().UTC()
		return &now, errors.New("Invalid file name, defaulting to UTC timestamp")
	}

	t, err := time.Parse(shortForm, exploded[1])
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// GetPlateFilename returns the full path to the plate image file's location.
func GetPlateFilename(filename string) string {
	_, file := filepath.Split(filename)
	file = strings.TrimSuffix(file, filepath.Ext(file))
	plateFileName := path.Join(config.Opts.PlateDir, file+".plate.jpg")
	return plateFileName
}

// GetFrameFilename returns the full path to the frame thumbnail image file's location.
func GetFrameFilename(filename string) string {
	_, file := filepath.Split(filename)
	file = strings.TrimSuffix(file, filepath.Ext(file))
	frameFileName := path.Join(config.Opts.FrameDir, file+".frame.jpg")
	return frameFileName
}

// SaveBuffer will save the buffer to a file on disk.
func SaveBuffer(filename string, filebytes *bytes.Buffer) error {
	err := ioutil.WriteFile(filename, filebytes.Bytes(), 0644)
	if err != nil {
		return err
	}
	return nil
}

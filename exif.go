package main

import (
	"os"
	"github.com/rwcarlsen/goexif/exif"
)

// GetImgDate pulls the shutter-click date from the RAW metadata
func GetImgDate(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Decode reads the EXIF metadata from the file stream
	x, err := exif.Decode(f)
	if err != nil {
		// Fallback: If EXIF is missing/corrupt, we don't want to crash.
		// We return a "Misc" folder name so the process continues.
		return "Unknown/Unknown-Date", nil
	}

	tm, err := x.DateTime()
	if err != nil {
		return "Unknown/Unknown-Date", nil
	}

	// Go's unique date formatting: 2006/01/02 is the reference template.
	// This will result in: "2026/2026-02-28"
	return tm.Format("2006/2006-01-02"), nil
}
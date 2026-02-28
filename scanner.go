package main

import (
	"os"
	"path/filepath"
	"strings"
)

// WorkOrder represents a single file that needs to be ingested
type WorkOrder struct {
	SourcePath string
	Filename   string
	Extension  string
}

// ScanSource walks the directory and finds files matching our whitelist
func ScanSource(sourceDir string, allowedTypes []string) ([]WorkOrder, error) {
	var orders []WorkOrder

	// WalkDir is efficient and modern (Go 1.16+)
	err := filepath.WalkDir(sourceDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories, we only want the files inside them
		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if isAllowed(ext, allowedTypes) {
			orders = append(orders, WorkOrder{
				SourcePath: path,
				Filename:   d.Name(),
				Extension:  ext,
			})
		}
		return nil
	})

	return orders, err
}

func isAllowed(ext string, allowed []string) bool {
	for _, a := range allowed {
		if ext == a {
			return true
		}
	}
	return false
}
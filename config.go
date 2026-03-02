package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ... the rest of your struct and functions ...
// Config represents the merged state of Global and Import settings
type Config struct {
	EnginePath      string         `json:"engine_path"`
	SourceDir       string         `json:"source_dir"`
	Destinations    []string       `json:"destinations"`
	FolderTemplate  string         `json:"folder_template"`
	VerifyChecksums bool           `json:"verify_checksums"`
	ExtractPreviews bool           `json:"extract_previews"`
	AllowedTypes    []string       `json:"allowed_types"`
	DeleteRecipes   bool           `json:"delete_recipes"`   // <-- NEW
	DeleteOriginals bool           `json:"delete_originals"` // <-- NEW
	OrientationMap  map[string]int `json:"orientation_map"`
	Verbose         bool
}

// LoadAndMerge reads the global config, then the local config, and merges them.
func LoadAndMerge(globalPath, localPath string) (*Config, error) {
	// Start with sane defaults
	cfg := &Config{
		EnginePath:      "dim-engine",      // Fallback if not in JSON
		FolderTemplate:  "2006/2006-01-02", // Go's quirky standard date format
		VerifyChecksums: true,
		//ExtractPreviews: true,
		AllowedTypes:    []string{".arw", ".nef", ".cr3", ".dng"},
		DeleteRecipes:   true,  // Safe to default true: it's just temp data
		DeleteOriginals: false, // ALWAYS default false: protect the user's RAWs!
		Verbose:         false,
	}

	// 1. Load Global Config (e.g., ~/.dimdesk/global.json)
	if globalData, err := os.ReadFile(globalPath); err == nil {
		json.Unmarshal(globalData, cfg)
		fmt.Println("📖 Loaded Global Config")
	}

	// 2. Load Local Import Config (e.g., ./import_wedding.json)
	// Any fields present here will overwrite the global/default fields
	if localData, err := os.ReadFile(localPath); err == nil {
		json.Unmarshal(localData, cfg)
		fmt.Println("📖 Loaded Local Import Config (Overrides active)")
	}

	// Clean up the paths immediately
	cfg.SourceDir = ExpandPath(cfg.SourceDir)
	cfg.EnginePath = ExpandPath(cfg.EnginePath)
	for i, dest := range cfg.Destinations {
		cfg.Destinations[i] = ExpandPath(dest)
	}

	return cfg, nil
}

func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path // Fallback to original if we can't find Home
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

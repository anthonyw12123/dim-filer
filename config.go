package main

import (
	"encoding/json"
	"os"
)

// Config represents the merged state of Global and Import settings
type Config struct {
	EnginePath      string   `json:"engine_path"`
	SourceDir       string   `json:"source_dir"`
	Destinations    []string `json:"destinations"`
	FolderTemplate  string   `json:"folder_template"`
	VerifyChecksums bool     `json:"verify_checksums"`
	ExtractPreviews bool     `json:"extract_previews"`
	AllowedTypes    []string `json:"allowed_types"`
}

// LoadAndMerge reads the global config, then the local config, and merges them.
func LoadAndMerge(globalPath, localPath string) (*Config, error) {
	// Start with sane defaults
	cfg := &Config{
		FolderTemplate:  "2006/2006-01-02", // Go's quirky standard date format
		VerifyChecksums: true,
		ExtractPreviews: true,
		AllowedTypes:    []string{".arw", ".nef", ".cr3", ".dng"},
	}

	// 1. Load Global Config (e.g., ~/.dimdesk/global.json)
	if globalData, err := os.ReadFile(globalPath); err == nil {
		json.Unmarshal(globalData, cfg)
	}

	// 2. Load Local Import Config (e.g., ./import_wedding.json)
	// Any fields present here will overwrite the global/default fields
	if localData, err := os.ReadFile(localPath); err == nil {
		json.Unmarshal(localData, cfg)
	}

	return cfg, nil
}
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// IngestJob represents one file's journey from SD to SSD
type IngestJob struct {
	Source    string
	DestRoot  string
	SubFolder string // e.g., "2026/2026-02-28"
	Filename  string
	Config    *Config
}

// EngineRecipe matches the Rust `Recipe` struct.
// We only populate the fields we need for the Fast-Path extraction.
type EngineRecipe struct {
	ExtractJpg     *bool          `json:"extract_jpg"`
	OutputPath     string         `json:"output_path"`
	OrientationMap map[string]int `json:"orientation_map"`
}

type AssetMeta struct {
	OriginalName string `json:"original_filename"`
	SHA256       string `json:"sha256"`
	IngestDate   string `json:"ingest_date"`
}

// Worker is a single concurrent processor
func Worker(id int, jobs <-chan IngestJob, results chan<- string, dryRun bool) {
	for job := range jobs {
		destDir := filepath.Join(job.DestRoot, job.SubFolder)
		previewDir := filepath.Join(destDir, ".previews")
		finalPath := filepath.Join(destDir, job.Filename)

		if dryRun {
			results <- fmt.Sprintf("Worker %d: [DRY RUN] Would ingest %s and generate recipe", id, job.Filename)
			continue
		}

		// 1. Create Directories
		os.MkdirAll(destDir, 0755)
		if job.Config.ExtractPreviews {
			os.MkdirAll(previewDir, 0755)
		}

		// 2. Perform Copy (SD -> SSD)
		if err := copyFile(job.Source, finalPath); err != nil {
			results <- fmt.Sprintf("Worker %d: ❌ Copy Failed %s: %v", id, job.Filename, err)
			continue
		}

		// 3. Verify Hash & Capture it
		var finalHash string
		if job.Config.VerifyChecksums {
			match, hashStr, err := verifyChecksum(job.Source, finalPath)
			if err != nil || !match {
				os.Remove(finalPath)
				results <- fmt.Sprintf("Worker %d: ❌ Checksum Failed (Corrupted Copy) %s", id, job.Filename)
				continue
			}
			finalHash = hashStr

			// The Unix way: Only print if Verbose is true
			if job.Config.Verbose {
				fmt.Printf("Worker %d: 🔒 Checksum verified for %s\n", id, job.Filename)
			}
		} else {
			// Even if verify is off, the Indexer STILL needs a hash!
			hashBytes, _ := hashFile(finalPath)
			finalHash = hex.EncodeToString(hashBytes)
		}

		// 3.5 Write the Immutable Metadata Sidecar (.meta)
		metaDir := filepath.Join(destDir, ".meta")
		os.MkdirAll(metaDir, 0755)

		meta := AssetMeta{
			OriginalName: job.Filename,
			SHA256:       finalHash,
			IngestDate:   time.Now().UTC().Format(time.RFC3339),
		}

		metaBytes, _ := json.MarshalIndent(meta, "", "  ")
		metaName := strings.TrimSuffix(job.Filename, filepath.Ext(job.Filename)) + ".json"
		os.WriteFile(filepath.Join(metaDir, metaName), metaBytes, 0644)

		// 4. Generate JSON Recipe & Trigger Rust
		if job.Config.ExtractPreviews {
			previewPath := filepath.Join(previewDir, job.Filename+".jpg")
			recipePath := filepath.Join(previewDir, job.Filename+".json")

			extractFlag := true
			recipe := EngineRecipe{
				ExtractJpg:     &extractFlag,
				OutputPath:     previewPath,
				OrientationMap: job.Config.OrientationMap,
			}

			recipeData, _ := json.MarshalIndent(recipe, "", "  ")
			os.WriteFile(recipePath, recipeData, 0644)

			cmd := exec.Command(job.Config.EnginePath, finalPath, recipePath)
			output, err := cmd.CombinedOutput()
			if err != nil {
				results <- fmt.Sprintf("Worker %d: ⚠️  Preview Failed for %s: %v\n%s", id, job.Filename, err, string(output))
			} else if job.Config.Verbose {
				// Only print success if verbose
				fmt.Printf("Worker %d: 🖼️  Preview extracted for %s\n", id, job.Filename)
			}

			if job.Config.DeleteRecipes {
				os.Remove(recipePath)
			}
		}

		// --- Cleanup the Original SD Card File ---
		if job.Config.DeleteOriginals {
			if err := os.Remove(job.Source); err != nil {
				results <- fmt.Sprintf("Worker %d: ⚠️  Could not delete original %s: %v", id, job.Filename, err)
			} else if job.Config.Verbose {
				fmt.Printf("Worker %d: 🗑️  Deleted original %s\n", id, job.Filename)
			}
		}

		// Only send success to the results channel if we actually want to log it in main
		results <- fmt.Sprintf("Worker %d: ✅ Success %s", id, job.Filename)
	}
}

// copyFile is a simple, efficient stream-based copy
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// verifyChecksum compares hashes and returns (match, dstHashString, error)
func verifyChecksum(src, dst string) (bool, string, error) {
	srcHash, err := hashFile(src)
	if err != nil {
		return false, "", err
	}

	dstHash, err := hashFile(dst)
	if err != nil {
		return false, "", err
	}

	// hex.EncodeToString converts the raw bytes to the standard "e3b0c442..." format
	return bytes.Equal(srcHash, dstHash), hex.EncodeToString(dstHash), nil
}

// hashFile calculates the SHA256 hash of a file without loading it all into RAM
func hashFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

// ABOUTME: Discovers sample infrastructure JSON files
// ABOUTME: Looks in frontend/public/samples or DIEGO_SAMPLES_PATH

package samples

import (
	"os"
	"path/filepath"
	"strings"
)

// SampleFile represents a discovered sample file
type SampleFile struct {
	Name string // Filename (e.g., "large-foundation-16-hosts.json")
	Path string // Full path to the file
}

// Discover finds all JSON files in the given directory
func Discover(dir string) ([]SampleFile, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return []SampleFile{}, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []SampleFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.ToLower(filepath.Ext(entry.Name())) != ".json" {
			continue
		}
		files = append(files, SampleFile{
			Name: entry.Name(),
			Path: filepath.Join(dir, entry.Name()),
		})
	}

	return files, nil
}

// FindSamplesDir locates the samples directory
// Checks in order:
// 1. DIEGO_SAMPLES_PATH environment variable
// 2. ./frontend/public/samples/ relative to given base path
func FindSamplesDir(basePath string) string {
	// Check environment variable first
	if envPath := os.Getenv("DIEGO_SAMPLES_PATH"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}
	}

	// Check relative to base path
	samplesDir := filepath.Join(basePath, "frontend", "public", "samples")
	if _, err := os.Stat(samplesDir); err == nil {
		return samplesDir
	}

	return ""
}

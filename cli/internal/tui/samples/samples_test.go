// ABOUTME: Tests for sample files discovery
// ABOUTME: Validates finding JSON files in samples directory

package samples

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscover(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some sample files
	os.WriteFile(filepath.Join(tmpDir, "sample1.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "sample2.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("ignore"), 0644)

	files, err := Discover(tmpDir)
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("expected 2 JSON files, got %d", len(files))
	}

	// Should only include .json files
	for _, f := range files {
		if filepath.Ext(f.Path) != ".json" {
			t.Errorf("unexpected non-JSON file: %s", f.Path)
		}
	}
}

func TestDiscoverMissingDir(t *testing.T) {
	files, err := Discover("/nonexistent/path")
	if err != nil {
		t.Fatalf("Discover() should not error for missing dir, got: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected empty list for missing dir, got %d", len(files))
	}
}

func TestDiscoverEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	files, err := Discover(tmpDir)
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected empty list for empty dir, got %d", len(files))
	}
}

func TestSampleFileInfo(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a sample file with descriptive name
	samplePath := filepath.Join(tmpDir, "large-foundation-16-hosts.json")
	os.WriteFile(samplePath, []byte("{}"), 0644)

	files, err := Discover(tmpDir)
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	if files[0].Name != "large-foundation-16-hosts.json" {
		t.Errorf("expected name 'large-foundation-16-hosts.json', got '%s'", files[0].Name)
	}
	if files[0].Path != samplePath {
		t.Errorf("expected path %s, got %s", samplePath, files[0].Path)
	}
}

func TestFindSamplesDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create the expected directory structure
	samplesDir := filepath.Join(tmpDir, "frontend", "public", "samples")
	os.MkdirAll(samplesDir, 0755)
	os.WriteFile(filepath.Join(samplesDir, "test.json"), []byte("{}"), 0644)

	// Test finding from repo root
	found := FindSamplesDir(tmpDir)
	if found != samplesDir {
		t.Errorf("expected %s, got %s", samplesDir, found)
	}
}

func TestFindSamplesDirNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	found := FindSamplesDir(tmpDir)
	if found != "" {
		t.Errorf("expected empty string for missing samples dir, got %s", found)
	}
}

func TestFindSamplesDirFromEnv(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "env-sample.json"), []byte("{}"), 0644)

	// Set environment variable
	os.Setenv("DIEGO_SAMPLES_PATH", tmpDir)
	defer os.Unsetenv("DIEGO_SAMPLES_PATH")

	found := FindSamplesDir("/some/other/path")
	if found != tmpDir {
		t.Errorf("expected %s from env, got %s", tmpDir, found)
	}
}

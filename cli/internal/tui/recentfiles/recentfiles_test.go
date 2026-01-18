// ABOUTME: Tests for recent files management
// ABOUTME: Validates XDG config storage, max limit, and path deduplication

package recentfiles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	rf := New(tmpDir)

	if rf == nil {
		t.Fatal("New() returned nil")
	}
	if rf.configDir != tmpDir {
		t.Errorf("expected configDir %s, got %s", tmpDir, rf.configDir)
	}
}

func TestLoadEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	rf := New(tmpDir)

	files, err := rf.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected empty list, got %d files", len(files))
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	rf := New(tmpDir)

	// Create real files for testing
	file1 := filepath.Join(tmpDir, "file1.json")
	file2 := filepath.Join(tmpDir, "file2.json")
	os.WriteFile(file1, []byte("{}"), 0644)
	os.WriteFile(file2, []byte("{}"), 0644)

	paths := []string{file1, file2}
	if err := rf.Save(paths); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := rf.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 files, got %d", len(loaded))
	}
	if loaded[0] != paths[0] {
		t.Errorf("expected %s, got %s", paths[0], loaded[0])
	}
}

func TestAddMoveToFront(t *testing.T) {
	tmpDir := t.TempDir()
	rf := New(tmpDir)

	// Create real files
	file1 := filepath.Join(tmpDir, "file1.json")
	file2 := filepath.Join(tmpDir, "file2.json")
	os.WriteFile(file1, []byte("{}"), 0644)
	os.WriteFile(file2, []byte("{}"), 0644)

	// Add first file
	rf.Add(file1)
	// Add second file
	rf.Add(file2)

	files, _ := rf.Load()
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	// Most recent should be first
	if files[0] != file2 {
		t.Errorf("expected file2 first, got %s", files[0])
	}

	// Add file1 again - should move to front
	rf.Add(file1)
	files, _ = rf.Load()
	if len(files) != 2 {
		t.Fatalf("expected 2 files after re-add, got %d", len(files))
	}
	if files[0] != file1 {
		t.Errorf("expected file1 first after re-add, got %s", files[0])
	}
}

func TestMaxLimit(t *testing.T) {
	tmpDir := t.TempDir()
	rf := New(tmpDir)

	// Create 7 real files
	var lastFile string
	for i := 1; i <= 7; i++ {
		f := filepath.Join(tmpDir, "file"+string(rune('0'+i))+".json")
		os.WriteFile(f, []byte("{}"), 0644)
		rf.Add(f)
		lastFile = f
	}

	files, _ := rf.Load()
	if len(files) != MaxRecentFiles {
		t.Errorf("expected %d files max, got %d", MaxRecentFiles, len(files))
	}
	// Most recent (file7) should be first
	if files[0] != lastFile {
		t.Errorf("expected %s first, got %s", lastFile, files[0])
	}
}

func TestLoadRemovesStaleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	rf := New(tmpDir)

	// Create a real file
	realFile := filepath.Join(tmpDir, "real.json")
	if err := os.WriteFile(realFile, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Save paths including a non-existent file
	paths := []string{"/nonexistent/file.json", realFile}
	rf.Save(paths)

	// Load should filter out non-existent files
	loaded, err := rf.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(loaded) != 1 {
		t.Errorf("expected 1 file after filtering, got %d", len(loaded))
	}
	if loaded[0] != realFile {
		t.Errorf("expected %s, got %s", realFile, loaded[0])
	}
}

func TestCreatesConfigDir(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "diego-capacity")
	rf := New(configDir)

	// Directory shouldn't exist yet
	if _, err := os.Stat(configDir); !os.IsNotExist(err) {
		t.Fatal("config dir should not exist yet")
	}

	// Save should create it
	rf.Add("/path/to/file.json")

	// Now it should exist
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Error("config dir should have been created")
	}
}

func TestDefaultConfigDir(t *testing.T) {
	// Test that DefaultConfigDir returns something reasonable
	dir := DefaultConfigDir()
	if dir == "" {
		t.Error("DefaultConfigDir() returned empty string")
	}
}

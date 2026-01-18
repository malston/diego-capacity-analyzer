// ABOUTME: Manages recent files list for the TUI file picker
// ABOUTME: Stores recent JSON file paths in XDG config directory

package recentfiles

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// MaxRecentFiles is the maximum number of recent files to keep
const MaxRecentFiles = 5

// RecentFiles manages the list of recently used JSON files
type RecentFiles struct {
	configDir string
	files     []string
}

type recentData struct {
	Files []string `json:"files"`
}

// New creates a new RecentFiles manager with the given config directory
func New(configDir string) *RecentFiles {
	return &RecentFiles{
		configDir: configDir,
		files:     nil,
	}
}

// DefaultConfigDir returns the default config directory following XDG spec
func DefaultConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "diego-capacity")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "diego-capacity")
}

// configFile returns the path to the recent files JSON
func (rf *RecentFiles) configFile() string {
	return filepath.Join(rf.configDir, "recent.json")
}

// Load reads the recent files list from disk
// Filters out files that no longer exist
func (rf *RecentFiles) Load() ([]string, error) {
	data, err := os.ReadFile(rf.configFile())
	if os.IsNotExist(err) {
		rf.files = []string{}
		return rf.files, nil
	}
	if err != nil {
		return nil, err
	}

	var recent recentData
	if err := json.Unmarshal(data, &recent); err != nil {
		// Invalid JSON, start fresh
		rf.files = []string{}
		return rf.files, nil
	}

	// Filter out files that no longer exist
	rf.files = make([]string, 0, len(recent.Files))
	for _, path := range recent.Files {
		if _, err := os.Stat(path); err == nil {
			rf.files = append(rf.files, path)
		}
	}

	return rf.files, nil
}

// Save writes the recent files list to disk
func (rf *RecentFiles) Save(files []string) error {
	// Ensure directory exists
	if err := os.MkdirAll(rf.configDir, 0755); err != nil {
		return err
	}

	// Trim to max
	if len(files) > MaxRecentFiles {
		files = files[:MaxRecentFiles]
	}

	rf.files = files

	data, err := json.MarshalIndent(recentData{Files: files}, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(rf.configFile(), data, 0644)
}

// Add adds a file path to the recent list (moves to front if exists)
func (rf *RecentFiles) Add(path string) error {
	// Load current list if not loaded
	if rf.files == nil {
		if _, err := rf.Load(); err != nil {
			rf.files = []string{}
		}
	}

	// Remove if already exists
	newFiles := make([]string, 0, len(rf.files)+1)
	newFiles = append(newFiles, path)
	for _, f := range rf.files {
		if f != path {
			newFiles = append(newFiles, f)
		}
	}

	return rf.Save(newFiles)
}

// List returns the current list of recent files
func (rf *RecentFiles) List() []string {
	if rf.files == nil {
		rf.Load()
	}
	return rf.files
}

// ABOUTME: Build constraint file to pin tool dependencies in go.mod.
// ABOUTME: This ensures Charm TUI libraries remain available.

//go:build tools

package tools

import (
	_ "github.com/charmbracelet/bubbles"
	_ "github.com/charmbracelet/bubbletea"
	_ "github.com/charmbracelet/huh"
	_ "github.com/charmbracelet/lipgloss"
)

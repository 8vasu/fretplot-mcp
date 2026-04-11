package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// userDataDir returns the OS-appropriate directory for application data:
//   - Linux:   $XDG_DATA_HOME or ~/.local/share
//   - macOS:   ~/Library/Application Support
//   - Windows: %APPDATA%
func userDataDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		dir := os.Getenv("APPDATA")
		if dir == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
		return dir, nil
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support"), nil
	default: // Linux and other Unix-like systems
		if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
			return dir, nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share"), nil
	}
}

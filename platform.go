// fretplot-mcp - MCP server for the fretplot LuaTeX package.
// Copyright (C) 2026 Soumendra Ganguly
// 
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
// 
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
// 
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Returns the OS-appropriate directory for application data:
//   - Linux:   "$XDG_DATA_HOME" or "${HOME}/.local/share"
//   - macOS:   "${HOME}/Library/Application Support"
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
	default:
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

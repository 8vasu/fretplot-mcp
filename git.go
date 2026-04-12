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
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// Runs a git(1) command, forwarding git(1)'s stderr to the server's stderr.
func git(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// Clones repoURL into cloneDir using a non-cone sparse checkout of sparsePaths,
// or pulls if repoURL is already cloned.
func syncRepo(cloneDir, repoURL string, sparsePaths []string) error {
	// Check for an existing clone by looking for .git.
	if _, err := os.Stat(filepath.Join(cloneDir, ".git")); os.IsNotExist(err) {
		log.Printf("Cloning %s (sparse) into %s", repoURL, cloneDir)

		// Create the parent directory of cloneDir if it does not exist.
		if err := os.MkdirAll(filepath.Dir(cloneDir), 0755); err != nil {
			return fmt.Errorf("creating data dir: %w", err)
		}

		// Clone without downloading any blobs and without checking out any files.
		if err := git("clone", "--filter=blob:none", "--no-checkout", repoURL, cloneDir); err != nil {
			return fmt.Errorf("git clone: %w", err)
		}

		// Configure a non-cone sparse checkout to fetch only the paths specified in sparsePaths.
		args := append([]string{"-C", cloneDir, "sparse-checkout", "set", "--no-cone"}, sparsePaths...)
		if err := git(args...); err != nil {
			return fmt.Errorf("git sparse-checkout set: %w", err)
		}

		// Materialize the files selected by the sparse checkout.
		if err := git("-C", cloneDir, "checkout"); err != nil {
			return fmt.Errorf("git checkout: %w", err)
		}

		return nil
	}

	// Already cloned: bring it up to date.
	log.Printf("Pulling %s in %s", repoURL, cloneDir)

	return git("-C", cloneDir, "pull")
}

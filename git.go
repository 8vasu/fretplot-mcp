package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// git runs a git command, forwarding stderr to the server's stderr.
func git(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// syncRepo clones repo into dir using a non-cone sparse checkout of sparsePaths,
// or pulls if the repo is already cloned.
func syncRepo(dir, repo string, sparsePaths []string) error {
	if _, err := os.Stat(filepath.Join(dir, ".git")); os.IsNotExist(err) {
		log.Printf("Cloning %s (sparse) into %s", repo, dir)
		if err := os.MkdirAll(filepath.Dir(dir), 0755); err != nil {
			return fmt.Errorf("creating data dir: %w", err)
		}
		if err := git("clone", "--filter=blob:none", "--no-checkout", repo, dir); err != nil {
			return fmt.Errorf("git clone: %w", err)
		}
		args := append([]string{"-C", dir, "sparse-checkout", "set", "--no-cone"}, sparsePaths...)
		if err := git(args...); err != nil {
			return fmt.Errorf("git sparse-checkout set: %w", err)
		}
		if err := git("-C", dir, "checkout"); err != nil {
			return fmt.Errorf("git checkout: %w", err)
		}
		return nil
	}
	log.Printf("Pulling %s in %s", repo, dir)
	return git("-C", dir, "pull")
}

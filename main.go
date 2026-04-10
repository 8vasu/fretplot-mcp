package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const fretplotRepo = "https://github.com/8vasu/fretplot"

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

func fretplotDir() (string, error) {
	dataDir, err := userDataDir()
	if err != nil {
		return "", fmt.Errorf("could not determine user data dir: %w", err)
	}
	return filepath.Join(dataDir, "fretplot-mcp", "fretplot"), nil
}

// run git, forwarding stderr to the server's stderr.
func git(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// syncFretplot clones or pulls the fretplot repo using a sparse checkout:
// only doc_fretplot.tex (a root file, included automatically in cone mode)
// and the include/ directory are fetched.
func syncFretplot() error {
	dir, err := fretplotDir()
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); os.IsNotExist(err) {
		log.Printf("Cloning fretplot (sparse) into %s", dir)
		if err := os.MkdirAll(filepath.Dir(dir), 0755); err != nil {
			return fmt.Errorf("creating data dir: %w", err)
		}
		// Partial clone without checkout; non-cone sparse checkout lets us
		// specify exact files/dirs to check out, excluding other root-level files.
		if err := git("clone", "--filter=blob:none", "--no-checkout", fretplotRepo, dir); err != nil {
			return fmt.Errorf("git clone: %w", err)
		}
		// Check out only doc_fretplot.tex and the include/ directory.
		if err := git("-C", dir, "sparse-checkout", "set", "--no-cone", "/doc_fretplot.tex", "/include/"); err != nil {
			return fmt.Errorf("git sparse-checkout set: %w", err)
		}
		if err := git("-C", dir, "checkout"); err != nil {
			return fmt.Errorf("git checkout: %w", err)
		}
		return nil
	}
	log.Printf("Pulling fretplot in %s", dir)
	return git("-C", dir, "pull")
}

func main() {
	if err := syncFretplot(); err != nil {
		log.Printf("Warning: fretplot sync failed: %v", err)
	}

	dir, err := fretplotDir()
	if err != nil {
		log.Fatalf("fretplot dir: %v", err)
	}
	docPath := filepath.Join(dir, "doc_fretplot.tex")

	sections, err := ParseDocSections(docPath)
	if err != nil {
		log.Printf("Warning: could not parse doc_fretplot.tex: %v", err)
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "fretplot-mcp", Version: "v0.1.0"}, nil)

	addResources(server, sections)
	addTools(server, sections)
	addPrompts(server)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Printf("Server error: %v", err)
	}
}

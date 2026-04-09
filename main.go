package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

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

// run executes a command, forwarding stderr to the server's stderr.
func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
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
		// Partial clone with cone-mode sparse checkout.
		// Root files (including doc_fretplot.tex) are checked out automatically.
		if err := run("git", "clone", "--filter=blob:none", "--sparse", fretplotRepo, dir); err != nil {
			return fmt.Errorf("git clone: %w", err)
		}
		// Add include/ so example files referenced by doc_fretplot.tex are also present.
		if err := run("git", "-C", dir, "sparse-checkout", "add", "include"); err != nil {
			return fmt.Errorf("git sparse-checkout add: %w", err)
		}
		return nil
	}
	log.Printf("Pulling fretplot in %s", dir)
	return run("git", "-C", dir, "pull")
}

func listScales(_ context.Context, _ *mcp.CallToolRequest, input snippetInput) (*mcp.CallToolResult, any, error) {
	dir, err := fretplotDir()
	if err != nil {
		return nil, nil, fmt.Errorf("fretplot dir: %w", err)
	}
	entries, err := ParseScaleMacros(filepath.Join(dir, "doc_fretplot.tex"))
	if err != nil {
		return nil, nil, fmt.Errorf("parsing scale macros from doc_fretplot.tex: %w", err)
	}
	var sb strings.Builder
	sb.WriteString("Built-in fretplot scale/arpeggio macros (interval formulas in semitones):\n\n")
	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("%-24s  %-44s  %s\n", e.macro, e.name, e.formula))
	}
	if input.Query != "" {
		sb.WriteString(fmt.Sprintf("\nQuery: %s\n", input.Query))
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: sb.String()}},
	}, nil, nil
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

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_scales",
		Description: "List all built-in fretplot scale and arpeggio macros with their interval formulas (semitones). Optionally takes a query to answer a specific question about a scale or arpeggio.",
	}, listScales)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Printf("Server error: %v", err)
	}
}

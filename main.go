package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

func syncFretplot() error {
	dir, err := fretplotDir()
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); os.IsNotExist(err) {
		log.Printf("Cloning fretplot into %s", dir)
		if err := os.MkdirAll(filepath.Dir(dir), 0755); err != nil {
			return fmt.Errorf("could not create data dir: %w", err)
		}
		cmd := exec.Command("git", "clone", fretplotRepo, dir)
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	log.Printf("Pulling fretplot in %s", dir)
	cmd := exec.Command("git", "-C", dir, "pull")
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type scaleEntry struct {
	macro   string
	name    string
	formula string
}

func parseScales(styPath string) ([]scaleEntry, error) {
	f, err := os.Open(styPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Matches only zero-argument \newcommand{\fp<name>}{<formula>}
	macroRe := regexp.MustCompile(`^\\newcommand\{\\(fp\w+)\}\{([^}]+)\}`)

	var entries []scaleEntry
	var lastComment string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "% "):
			lastComment = strings.TrimPrefix(line, "% ")
		case macroRe.MatchString(line):
			m := macroRe.FindStringSubmatch(line)
			entries = append(entries, scaleEntry{
				macro:   `\` + m[1],
				name:    lastComment,
				formula: m[2],
			})
			lastComment = ""
		case line != "" && !strings.HasPrefix(line, "%"):
			lastComment = ""
		}
	}
	return entries, scanner.Err()
}

func listScales(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	dir, err := fretplotDir()
	if err != nil {
		return nil, nil, fmt.Errorf("fretplot dir: %w", err)
	}
	entries, err := parseScales(filepath.Join(dir, "fretplot.sty"))
	if err != nil {
		return nil, nil, fmt.Errorf("parsing fretplot.sty: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("Built-in fretplot scale/arpeggio macros (interval formulas in semitones):\n\n")
	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("%-24s  %-44s  %s\n", e.macro, e.name, e.formula))
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: sb.String()}},
	}, nil, nil
}

func main() {
	if err := syncFretplot(); err != nil {
		log.Printf("Warning: fretplot sync failed: %v", err)
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "fretplot-mcp", Version: "v0.1.0"}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_scales",
		Description: "List all built-in fretplot scale and arpeggio macros with their interval formulas (semitones).",
	}, listScales)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Printf("Server error: %v", err)
	}
}

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
	"context"
	"log"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- MCP Server ---

const fretplotMCPServerName = "fretplot-mcp"
const fretplotMCPServerVersion = "v0.1.0"

// --- fretplot repository ---

const fretplotRepoName = "fretplot"
const fretplotRepoURL = "https://github.com/8vasu/" + fretplotRepoName

// Names of documentation file and include directory within the repo.
const fretplotDocFileName = "doc_fretplot.tex"
const fretplotDocIncludeDirName = "include"

// Paths passed to git sparse-checkout: only the doc file and include directory are fetched.
var fretplotSparsePaths = []string{
	"/" + fretplotDocFileName,
	"/" + fretplotDocIncludeDirName + "/",
}

// In the fretplot documentation, \fpdocexample includes the following files:
//   - include/<exampleDirName>/src.fp
//   - include/<exampleDirName>/full.tex
var fpdocexampleFileNames = []string{
	"src.fp",
	"full.tex",
}

// Computed at startup by init().
var (
	// Local path to the cloned repo;
	// e.g. `~/.local/share/fretplot-mcp/fretplot`.
	fretplotRepoDirPath string
	// Local path to the fretplot documentation file;
	// e.g. `~/.local/share/fretplot-mcp/fretplot/doc_fretplot.tex`.
	fretplotDocFilePath string
)

// --- fretplot documentation structure ---

// Types for better readability.
type fretplotDocSectionTitle string   // Type for title of a section in the fretplot documentation.
type fretplotDocSectionContent string // Type for content of a section in the fretplot documentation.

// --- MCP tools ---

type fretplotMCPToolName string
type fretplotMCPToolInfo struct {
	description      string                    // MCP tool description.
	docSectionTitles []fretplotDocSectionTitle // Titles of relevant sections (\section) in the fretplot LaTeX documentation.
}

// JSON input schema for all MCP tools.
type fretplotMCPToolInput struct {
	Query string `json:"query"`
}

// Handler type for MCP tools.
type fretplotMCPToolHandler = func(
	context.Context,
	*mcp.CallToolRequest,
	fretplotMCPToolInput,
) (*mcp.CallToolResult, any, error)

// Maps MCP tool names to corresponding tool info.
var fretplotMCPToolTable = map[fretplotMCPToolName]fretplotMCPToolInfo{
	"fp": {
		"Generate .fp code based on provided query and the relevant .fp format documentation.",
		[]fretplotDocSectionTitle{"The fretplot file format"},
	},
	"fps": {
		"Generate .fps code based on provided query and the relevant .fp format documentation.",
		[]fretplotDocSectionTitle{"The fretplot scale style file format"},
	},
	"fptex": {
		"Generate LaTeX code involving the fretplot macros \\fptotikz, \\fptemplate, \\fpstemplate, \\fpscale, and built-in scale/arpeggio macros based on provided query and the relevant macro documentation.",
		[]fretplotDocSectionTitle{"Introduction", `\LaTeX\ macros`},
	},
}

// --- Startup initialization ---

// Resolve local fretplot repo dir path and fretplot documentation file path
// from the OS-appropriate data directory.
func init() {
	dataDir, err := userDataDir()
	if err != nil {
		log.Fatal(err)
	}

	fretplotRepoDirPath = filepath.Join(dataDir, fretplotMCPServerName, fretplotRepoName)
	fretplotDocFilePath = filepath.Join(fretplotRepoDirPath, fretplotDocFileName)
}

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
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Returns a handler for an MCP tool that injects the combination (fullContext)
// of the following to the LLM's context:
//   - The user's query (input.Query), and
//   - Sections of the fretplot documentation relevant to the tool (docContext).
func makeToolHandler(docContext string) fretplotMCPToolHandler {
	return func(
		_ context.Context,
		_ *mcp.CallToolRequest,
		input fretplotMCPToolInput,
	) (*mcp.CallToolResult, any, error) {
		fullContext := fmt.Sprintf(
			"Query: %s\n\nRelevant documentation:\n\n%s",
			input.Query,
			docContext,
		)

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fullContext}},
		}, nil, nil
	}
}

// Concatenates the contents of the sections in fretplot's
// LaTeX documentation which have titles in targetDocSectionTitles,
// in the order of those titles.
//
// We will use this function to concatenate the contents of relevant fretplot
// documentation sections that a given MCP tool needs to inject into the LLM's
// context.
func concatTargetDocSections(
	docSectionTable map[fretplotDocSectionTitle]fretplotDocSectionContent,
	targetDocSectionTitles []fretplotDocSectionTitle,
) string {
	parts := make([]string, 0, len(targetDocSectionTitles))
	for _, t := range targetDocSectionTitles {
		parts = append(parts, string(docSectionTable[t]))
	}

	return strings.Join(parts, "\n\n")
}

// Parses the documentation and registers all tools.
func addTools(server *mcp.Server) error {
	// Parse fretplot documentation to get a map of section titles
	// to section contents.
	docSectionTable, err := ParseDocSections()
	if err != nil {
		return err
	}

	// Register all tools.
	for toolName, toolInfo := range fretplotMCPToolTable {
		mcp.AddTool(server, &mcp.Tool{
			Name:        string(toolName),
			Description: toolInfo.description,
		}, makeToolHandler(concatTargetDocSections(docSectionTable, toolInfo.docSectionTitles)))
	}

	return nil
}

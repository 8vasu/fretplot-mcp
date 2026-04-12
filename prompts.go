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

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Returns a handler for an MCP prompt that directs the prompt
// query argument to the correct MCP tool.
//
// In case of this MCP server, there is a one-to-one correspondence
// between MCP tools and MCP prompts.
func makePromptHandler(toolName fretplotMCPToolName) mcp.PromptHandler {
	return func(
		_ context.Context,
		req *mcp.GetPromptRequest,
	) (*mcp.GetPromptResult, error) {
		query := req.Params.Arguments["query"]

		return &mcp.GetPromptResult{
			Messages: []*mcp.PromptMessage{
				{
					Role: "user",
					Content: &mcp.TextContent{
						Text: fmt.Sprintf(
							"Use the %s tool to answer: %s",
							toolName,
							query,
						),
					},
				},
			},
		}, nil
	}
}

// Registers one prompt per tool, each accepting a single query argument.
func addPrompts(server *mcp.Server) {
	queryArg := []*mcp.PromptArgument{{Name: "query", Required: true}}

	for toolName, toolInfo := range fretplotMCPToolTable {
		server.AddPrompt(&mcp.Prompt{
			Name:        string(toolName),
			Description: toolInfo.description,
			Arguments:   queryArg,
		}, makePromptHandler(toolName))
	}
}

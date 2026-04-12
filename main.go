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

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	// Perform sparse checkout of documentation files from the fretplot repo.
	if err := syncRepo(fretplotRepoDirPath, fretplotRepoURL, fretplotSparsePaths); err != nil {
		log.Printf("Warning: %s repo sync failed: %v", fretplotRepoName, err)
	}

	// Create and run the MCP server.
	server := mcp.NewServer(&mcp.Implementation{Name: fretplotMCPServerName,
		Version: fretplotMCPServerVersion}, nil)

	// Add tools and prompts to the server.
	if err := addTools(server); err != nil {
		log.Fatal(err)
	}
	addPrompts(server)

	// Run the server with stdio transport.
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Printf("Server error: %v", err)
	}
}

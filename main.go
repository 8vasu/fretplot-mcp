package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	server := mcp.NewServer(&mcp.Implementation{Name: fretplotMCPServerName,
		Version: fretplotMCPServerVersion}, nil)

	if err := addTools(server); err != nil {
		log.Fatal(err)
	}
	addPrompts(server)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Printf("Server error: %v", err)
	}
}

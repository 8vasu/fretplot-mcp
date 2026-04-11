package main

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func makePromptHandler(toolName string) mcp.PromptHandler {
	return func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		query := req.Params.Arguments["query"]
		return &mcp.GetPromptResult{
			Messages: []*mcp.PromptMessage{
				{
					Role:    "user",
					Content: &mcp.TextContent{Text: fmt.Sprintf("Use the %s tool to answer: %s", toolName, query)},
				},
			},
		}, nil
	}
}

func addPrompts(server *mcp.Server) {
	arg := []*mcp.PromptArgument{{Name: "query", Required: true}}
	for name, cfg := range tools {
		server.AddPrompt(&mcp.Prompt{
			Name:        name,
			Description: cfg.description,
			Arguments:   arg,
		}, makePromptHandler(name))
	}
}

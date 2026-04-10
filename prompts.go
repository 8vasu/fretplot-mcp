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

	prompts := []struct {
		name, description string
	}{
		{
			"list_scales",
			"Look up built-in fretplot scale and arpeggio macros.",
		},
		{
			"fp_snippet",
			"Generate .fp code for any fretplot diagram property or effect.",
		},
		{
			"fps_snippet",
			"Generate .fps code for any fretplot scale style customization.",
		},
		{
			"tex_snippet",
			"Generate fretplot LaTeX code: macros, preamble, complete documents.",
		},
	}

	for _, p := range prompts {
		server.AddPrompt(&mcp.Prompt{
			Name:        p.name,
			Description: p.description,
			Arguments:   arg,
		}, makePromptHandler(p.name))
	}
}

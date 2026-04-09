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

	server.AddPrompt(&mcp.Prompt{
		Name:        "list_scales",
		Description: "Look up built-in fretplot scale and arpeggio macros.",
		Arguments:   arg,
	}, makePromptHandler("list_scales"))

	server.AddPrompt(&mcp.Prompt{
		Name:        "fp_snippet",
		Description: "Generate .fp code for any fretplot diagram property or effect.",
		Arguments:   arg,
	}, makePromptHandler("fp_snippet"))

	server.AddPrompt(&mcp.Prompt{
		Name:        "fps_snippet",
		Description: "Generate .fps code for any fretplot scale style customization.",
		Arguments:   arg,
	}, makePromptHandler("fps_snippet"))

	server.AddPrompt(&mcp.Prompt{
		Name:        "tex_snippet",
		Description: "Generate fretplot LaTeX code: macros, preamble, complete documents.",
		Arguments:   arg,
	}, makePromptHandler("tex_snippet"))
}

package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type snippetInput struct {
	Query string `json:"query"`
}

func makeSnippetHandler(docs string) func(context.Context, *mcp.CallToolRequest, snippetInput) (*mcp.CallToolResult, any, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input snippetInput) (*mcp.CallToolResult, any, error) {
		text := fmt.Sprintf("Query: %s\n\nRelevant documentation:\n\n%s", input.Query, docs)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
		}, nil, nil
	}
}

func sectionDocs(doc map[string]string, titles []string) string {
	parts := make([]string, 0, len(titles))
	for _, t := range titles {
		parts = append(parts, doc[t])
	}
	return strings.Join(parts, "\n\n")
}

func addTools(server *mcp.Server) error {
	doc, err := ParseDocSections()
	if err != nil {
		return err
	}

	for name, cfg := range tools {
		mcp.AddTool(server, &mcp.Tool{Name: name, Description: cfg.description},
			makeSnippetHandler(sectionDocs(doc, cfg.docSectionTitles)))
	}
	return nil
}

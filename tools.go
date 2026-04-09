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

func sectionsWithPrefix(sections []DocSection, prefix string) []DocSection {
	var result []DocSection
	for _, s := range sections {
		if strings.HasPrefix(s.URI, prefix) {
			result = append(result, s)
		}
	}
	return result
}

func formatSections(sections []DocSection) string {
	var sb strings.Builder
	for _, s := range sections {
		fmt.Fprintf(&sb, "=== %s (%s) ===\n%s\n\n", s.Name, s.URI, s.Content)
	}
	return strings.TrimSpace(sb.String())
}

func makeSnippetHandler(docs string) func(context.Context, *mcp.CallToolRequest, snippetInput) (*mcp.CallToolResult, any, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input snippetInput) (*mcp.CallToolResult, any, error) {
		text := fmt.Sprintf("Query: %s\n\nRelevant documentation:\n\n%s", input.Query, docs)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
		}, nil, nil
	}
}

func addTools(server *mcp.Server, sections []DocSection) {
	fpDocs := formatSections(sectionsWithPrefix(sections, "fretplot://doc/fp/"))
	fpsDocs := formatSections(sectionsWithPrefix(sections, "fretplot://doc/fps/"))

	var texSections []DocSection
	for _, prefix := range []string{"fretplot://doc/introduction/", "fretplot://doc/macros/"} {
		texSections = append(texSections, sectionsWithPrefix(sections, prefix)...)
	}
	texDocs := formatSections(texSections)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fp_snippet",
		Description: "Generate .fp code for any fretplot diagram property or effect (rotation, string tuning, fret markers, capo, layout, etc.). Provide the query describing what you need; the tool returns the relevant .fp format documentation for you to generate the correct snippet.",
	}, makeSnippetHandler(fpDocs))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fps_snippet",
		Description: "Generate .fps code for any fretplot scale style customization (note colors, shapes, labels, finger numbers, etc.). Provide the query describing what you need; the tool returns the relevant .fps format documentation for you to generate the correct snippet.",
	}, makeSnippetHandler(fpsDocs))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "tex_snippet",
		Description: "Generate fretplot LaTeX code for any macro usage: \\fpscale, \\fpfret, \\fpchord, document preamble setup, package options, TikZ integration, and complete compilable documents. Provide the query describing what you need; the tool returns the relevant documentation for you to generate the correct snippet.",
	}, makeSnippetHandler(texDocs))
}

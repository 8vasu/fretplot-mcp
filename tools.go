package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type snippetInput struct {
	Query string `json:"query"`
}

func listScales(_ context.Context, _ *mcp.CallToolRequest, input snippetInput) (*mcp.CallToolResult, any, error) {
	dir, err := fretplotDir()
	if err != nil {
		return nil, nil, fmt.Errorf("fretplot dir: %w", err)
	}
	entries, err := ParseScaleMacros(filepath.Join(dir, "doc_fretplot.tex"))
	if err != nil {
		return nil, nil, fmt.Errorf("parsing scale macros from doc_fretplot.tex: %w", err)
	}
	var sb strings.Builder
	sb.WriteString("Built-in fretplot scale/arpeggio macros (interval formulas in semitones):\n\n")
	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("%-24s  %-44s  %s\n", e.macro, e.name, e.formula))
	}
	if input.Query != "" {
		sb.WriteString(fmt.Sprintf("\nQuery: %s\n", input.Query))
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: sb.String()}},
	}, nil, nil
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
	fpDocs := formatSections(sectionsWithPrefix(sections, "fretplot://fp/"))
	fpsDocs := formatSections(sectionsWithPrefix(sections, "fretplot://fps/"))

	var texSections []DocSection
	for _, prefix := range []string{"fretplot://introduction/", "fretplot://macros/"} {
		texSections = append(texSections, sectionsWithPrefix(sections, prefix)...)
	}
	texDocs := formatSections(texSections)

	type toolDef struct {
		name, description string
		handler           func(context.Context, *mcp.CallToolRequest, snippetInput) (*mcp.CallToolResult, any, error)
	}
	tools := []toolDef{
		{
			"list_scales",
			"List all built-in fretplot scale and arpeggio macros with their interval formulas (semitones). Optionally takes a query to answer a specific question about a scale or arpeggio.",
			listScales,
		},
		{
			"fp_snippet",
			"Generate .fp code for any fretplot diagram property or effect (rotation, string tuning, fret markers, capo, layout, etc.). Provide the query describing what you need; the tool returns the relevant .fp format documentation for you to generate the correct snippet.",
			makeSnippetHandler(fpDocs),
		},
		{
			"fps_snippet",
			"Generate .fps code for any fretplot scale style customization (note colors, shapes, labels, finger numbers, etc.). Provide the query describing what you need; the tool returns the relevant .fps format documentation for you to generate the correct snippet.",
			makeSnippetHandler(fpsDocs),
		},
		{
			"tex_snippet",
			"Generate fretplot LaTeX code for any macro usage: \\fpscale, \\fpfret, \\fpchord, document preamble setup, package options, TikZ integration, and complete compilable documents. Provide the query describing what you need; the tool returns the relevant documentation for you to generate the correct snippet.",
			makeSnippetHandler(texDocs),
		},
	}
	for _, t := range tools {
		mcp.AddTool(server, &mcp.Tool{Name: t.name, Description: t.description}, t.handler)
	}
}

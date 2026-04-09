package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// addResources registers one MCP resource per parsed DocSection.
func addResources(server *mcp.Server, sections []DocSection) {
	for _, s := range sections {
		content := s.Content
		uri := s.URI
		name := s.Name
		server.AddResource(&mcp.Resource{
			Name:     name,
			URI:      uri,
			MIMEType: "text/plain",
		}, func(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{
					{URI: req.Params.URI, MIMEType: "text/plain", Text: content},
				},
			}, nil
		})
	}
	log.Printf("Registered %d resources from doc_fretplot.tex", len(sections))
}

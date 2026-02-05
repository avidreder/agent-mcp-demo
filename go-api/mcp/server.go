// Package mcp provides MCP (Model Context Protocol) server implementation
// for AI agent discovery of x402 payment resources.
package mcp

import (
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server wraps the MCP server implementation for x402 discovery.
type Server struct {
	mcpServer *mcp.Server
	resources []X402DiscoveryResource
}

// NewServer creates a new MCP server instance with x402 discovery capabilities.
func NewServer() (*Server, error) {
	resources, err := loadDiscoveryResources()
	if err != nil {
		return nil, err
	}
	mcpServer := mcp.NewServer(
		&mcp.Implementation{
			Name:    "x402-discovery",
			Version: "1.0.0",
		},
		&mcp.ServerOptions{},
	)

	s := &Server{
		mcpServer: mcpServer,
		resources: resources,
	}

	s.registerTools()

	return s, nil
}

// Handler returns an http.Handler for the MCP streamable HTTP transport.
// This handler should be mounted at /discovery/mcp.
func (s *Server) Handler() http.Handler {
	return mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return s.mcpServer
	}, nil)
}

// HandlerWithOptions returns an http.Handler for the MCP streamable HTTP transport
// with custom StreamableHTTPOptions.
func (s *Server) HandlerWithOptions(opts *mcp.StreamableHTTPOptions) http.Handler {
	return mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return s.mcpServer
	}, opts)
}

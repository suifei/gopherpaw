// Package mcp provides MCP (Model Context Protocol) client for connecting to external tool servers.
package mcp

import "context"

// Transport abstracts MCP communication layer.
// Implementations include StdioTransport, HTTPTransport, and SSETransport.
type Transport interface {
	// Start initializes the transport connection.
	Start(ctx context.Context) error

	// Stop closes the transport connection.
	Stop() error

	// Call sends a JSON-RPC request and returns the response.
	Call(ctx context.Context, req jsonRPCRequest, result interface{}) error

	// WriteNotification sends a notification (no response expected).
	WriteNotification(msg map[string]any) error

	// IsRunning returns true if transport is active.
	IsRunning() bool
}

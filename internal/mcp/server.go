package mcp

import (
	"encoding/json"
	"fmt"
	"strings"

	mcps "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"xtop/internal/collector"
	"xtop/internal/version"
)

// newMCPServer builds a configured MCPServer backed by the collector.
func newMCPServer(c *collector.Collector) *mcpserver.MCPServer {
	s := mcpserver.NewMCPServer(
		"xtop",
		version.Version,
		mcpserver.WithResourceCapabilities(true, true),
		mcpserver.WithToolCapabilities(true),
		mcpserver.WithRecovery(),
	)

	registerResources(s, c)
	registerTools(s, c)
	return s
}

// serveStdio builds and runs the stdio MCP server backed by the collector.
func serveStdio(c *collector.Collector) error {
	return mcpserver.ServeStdio(newMCPServer(c))
}

// jsonResourceContents marshals v as a JSON text resource contents for uri.
func jsonResourceContents(uri string, v any) ([]mcps.ResourceContents, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return []mcps.ResourceContents{
		mcps.TextResourceContents{
			URI:      uri,
			MIMEType: "application/json",
			Text:     string(b),
		},
	}, nil
}

// snapshotPart selects the requested subsystem snapshot from a full Snapshot.
func snapshotPart(part string, snap collector.Snapshot) any {
	switch part {
	case "cpu":
		return snap.CPU
	case "mem":
		return snap.Mem
	case "disk":
		return snap.Disk
	case "gpu":
		return snap.GPU
	case "net":
		return snap.Net
	case "proc":
		return snap.Proc
	default:
		return snap
	}
}

func snapshotURI(part string) string {
	if part == "" {
		return "xtop://snapshot/latest"
	}
	return fmt.Sprintf("xtop://snapshot/%s", part)
}

func partFromURI(uri string) string {
	const prefix = "xtop://snapshot/"
	if !strings.HasPrefix(uri, prefix) {
		return ""
	}
	part := strings.TrimPrefix(uri, prefix)
	if part == "latest" {
		return ""
	}
	return part
}

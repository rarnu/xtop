package mcp

import (
	"context"
	"fmt"

	mcps "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"xtop/internal/collector"
)

// registerResources adds all read-only xtop resources to the MCP server.
func registerResources(s *mcpserver.MCPServer, c *collector.Collector) {
	parts := []struct {
		part        string
		displayName string
		desc        string
	}{
		{"", "Latest Snapshot", "The most recent complete system monitoring snapshot."},
		{"cpu", "CPU Snapshot", "Current CPU utilisation (overall and per-core)."},
		{"mem", "Memory Snapshot", "Current physical memory usage."},
		{"disk", "Disk Snapshot", "Per-mount disk usage and IO rates."},
		{"gpu", "GPU Snapshot", "Current GPU utilisation and memory."},
		{"net", "Network Snapshot", "Current network throughput and cumulative counters."},
		{"proc", "Process Snapshot", "Current process statistics and top-N lists."},
	}

	for _, p := range parts {
		uri := snapshotURI(p.part)
		resource := mcps.NewResource(
			uri,
			p.displayName,
			mcps.WithResourceDescription(p.desc),
			mcps.WithMIMEType("application/json"),
		)
		s.AddResource(resource, resourceHandler(c, p.part))
	}
}

func resourceHandler(c *collector.Collector, part string) func(context.Context, mcps.ReadResourceRequest) ([]mcps.ResourceContents, error) {
	return func(ctx context.Context, request mcps.ReadResourceRequest) ([]mcps.ResourceContents, error) {
		uri := request.Params.URI
		if part == "" {
			// latest resource must match exactly
			if uri != snapshotURI("") {
				return nil, fmt.Errorf("unknown resource URI: %s", uri)
			}
		} else {
			if partFromURI(uri) != part {
				return nil, fmt.Errorf("unknown resource URI: %s", uri)
			}
		}
		snap := c.Snapshot()
		return jsonResourceContents(uri, snapshotPart(part, snap))
	}
}

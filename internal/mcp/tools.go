package mcp

import (
	"context"
	"fmt"

	mcps "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"xtop/internal/collector"
)

// readOnlyTool creates a tool annotated as read-only, non-destructive and idempotent.
func readOnlyTool(name, description string, opts ...mcps.ToolOption) mcps.Tool {
	all := append([]mcps.ToolOption{
		mcps.WithDescription(description),
		mcps.WithToolAnnotation(mcps.ToolAnnotation{
			ReadOnlyHint:    mcps.ToBoolPtr(true),
			DestructiveHint: mcps.ToBoolPtr(false),
			IdempotentHint:  mcps.ToBoolPtr(true),
		}),
	}, opts...)
	return mcps.NewTool(name, all...)
}

// registerTools adds all read-only xtop query tools to the MCP server.
func registerTools(s *mcpserver.MCPServer, c *collector.Collector) {
	tools := []struct {
		name    string
		builder func() mcps.Tool
		handler func(context.Context, mcps.CallToolRequest) (*mcps.CallToolResult, error)
	}{
		{
			name: "get_system_summary",
			builder: func() mcps.Tool {
				return readOnlyTool("get_system_summary", "Get a concise, human-readable summary of the current system status including CPU, memory, disk, GPU, network and top processes.")
			},
			handler: makeSummaryHandler(c),
		},
		{
			name: "get_cpu_info",
			builder: func() mcps.Tool {
				return readOnlyTool("get_cpu_info", "Get current CPU utilisation: overall and per-core.")
			},
			handler: makePartHandler(c, "cpu", formatCPU),
		},
		{
			name: "get_memory_info",
			builder: func() mcps.Tool {
				return readOnlyTool("get_memory_info", "Get current physical memory usage: total, used, cached and free.")
			},
			handler: makePartHandler(c, "mem", formatMem),
		},
		{
			name: "get_disk_info",
			builder: func() mcps.Tool {
				return readOnlyTool("get_disk_info", "Get per-mount disk usage and read/write rates.")
			},
			handler: makePartHandler(c, "disk", formatDisk),
		},
		{
			name: "get_gpu_info",
			builder: func() mcps.Tool {
				return readOnlyTool("get_gpu_info", "Get GPU status: name, load, video memory and power/temperature.")
			},
			handler: makePartHandler(c, "gpu", formatGPU),
		},
		{
			name: "get_network_info",
			builder: func() mcps.Tool {
				return readOnlyTool("get_network_info", "Get current network upload/download throughput and cumulative traffic.")
			},
			handler: makePartHandler(c, "net", formatNet),
		},
		{
			name:    "get_process_info",
			builder: processToolBuilder,
			handler: makeProcessHandler(c),
		},
	}

	for _, t := range tools {
		s.AddTool(t.builder(), t.handler)
	}
}

func makeSummaryHandler(c *collector.Collector) func(context.Context, mcps.CallToolRequest) (*mcps.CallToolResult, error) {
	return func(ctx context.Context, request mcps.CallToolRequest) (*mcps.CallToolResult, error) {
		snap := c.Snapshot()
		text := formatSummary(snap)
		return mcps.NewToolResultText(text), nil
	}
}

func makePartHandler(c *collector.Collector, part string, formatter func(collector.Snapshot) string) func(context.Context, mcps.CallToolRequest) (*mcps.CallToolResult, error) {
	return func(ctx context.Context, request mcps.CallToolRequest) (*mcps.CallToolResult, error) {
		snap := c.Snapshot()
		return mcps.NewToolResultText(formatter(snap)), nil
	}
}

func processToolBuilder() mcps.Tool {
	return readOnlyTool("get_process_info",
		"Get top processes ordered by CPU, memory, disk or GPU memory. The full process list is never returned; only the requested top-N slice is serialized.",
		mcps.WithString("top_by",
			mcps.Description("Dimension to sort by: cpu, mem, disk or gpu"),
			mcps.Enum("cpu", "mem", "disk", "gpu"),
		),
		mcps.WithNumber("limit",
			mcps.Description("Maximum number of processes to return (1-20, default 10)"),
		),
	)
}

func makeProcessHandler(c *collector.Collector) func(context.Context, mcps.CallToolRequest) (*mcps.CallToolResult, error) {
	return func(ctx context.Context, request mcps.CallToolRequest) (*mcps.CallToolResult, error) {
		topBy := request.GetString("top_by", "cpu")
		limit := request.GetInt("limit", 10)
		if limit < 1 {
			limit = 1
		}
		if limit > 20 {
			limit = 20
		}
		// Avoid collecting a full snapshot when only process info is needed.
		ps := c.CollectProc()
		return mcps.NewToolResultText(formatProc(ps, topBy, limit)), nil
	}
}

// toolResultError returns a tool result containing an error message.
func toolResultError(format string, args ...any) *mcps.CallToolResult {
	return mcps.NewToolResultError(fmt.Sprintf(format, args...))
}

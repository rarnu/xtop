package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	mcps "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"xtop/internal/collector"
)

func newTestServer(t *testing.T) *mcpserver.MCPServer {
	t.Helper()
	c := collector.New()
	c.StartProcLoop()
	waitForFirstCaches(c, 500*time.Millisecond)

	s := mcpserver.NewMCPServer(
		"xtop-test",
		"0.0.0",
		mcpserver.WithResourceCapabilities(true, true),
		mcpserver.WithToolCapabilities(true),
	)
	registerResources(s, c)
	registerTools(s, c)
	return s
}

func send(t *testing.T, s *mcpserver.MCPServer, message string) mcps.JSONRPCMessage {
	t.Helper()
	resp := s.HandleMessage(context.Background(), []byte(message))
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	return resp
}

func TestMCPServer_Initialize(t *testing.T) {
	s := newTestServer(t)
	resp := send(t, s, `{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "initialize"
	}`)
	r := resp.(mcps.JSONRPCResponse)
	result, ok := r.Result.(mcps.InitializeResult)
	if !ok {
		t.Fatalf("expected InitializeResult, got %T", r.Result)
	}
	if result.ServerInfo.Name != "xtop-test" {
		t.Fatalf("unexpected server name: %s", result.ServerInfo.Name)
	}
}

func TestMCPServer_ListTools(t *testing.T) {
	s := newTestServer(t)
	resp := send(t, s, `{
		"jsonrpc": "2.0",
		"id": 2,
		"method": "tools/list"
	}`)
	r := resp.(mcps.JSONRPCResponse)
	result, ok := r.Result.(mcps.ListToolsResult)
	if !ok {
		t.Fatalf("expected ListToolsResult, got %T", r.Result)
	}
	if len(result.Tools) != 7 {
		t.Fatalf("expected 7 tools, got %d", len(result.Tools))
	}
}

func TestMCPServer_ListResources(t *testing.T) {
	s := newTestServer(t)
	resp := send(t, s, `{
		"jsonrpc": "2.0",
		"id": 3,
		"method": "resources/list"
	}`)
	r := resp.(mcps.JSONRPCResponse)
	result, ok := r.Result.(mcps.ListResourcesResult)
	if !ok {
		t.Fatalf("expected ListResourcesResult, got %T", r.Result)
	}
	if len(result.Resources) != 7 {
		t.Fatalf("expected 7 resources, got %d", len(result.Resources))
	}
}

func TestMCPServer_CallTool(t *testing.T) {
	s := newTestServer(t)
	resp := send(t, s, `{
		"jsonrpc": "2.0",
		"id": 4,
		"method": "tools/call",
		"params": {
			"name": "get_system_summary",
			"arguments": {}
		}
	}`)
	r := resp.(mcps.JSONRPCResponse)
	result, ok := r.Result.(*mcps.CallToolResult)
	if !ok {
		t.Fatalf("expected *CallToolResult, got %T", r.Result)
	}
	if len(result.Content) == 0 {
		t.Fatalf("expected non-empty tool result content")
	}
}

func TestMCPServer_ReadResource(t *testing.T) {
	s := newTestServer(t)
	resp := send(t, s, `{
		"jsonrpc": "2.0",
		"id": 5,
		"method": "resources/read",
		"params": {
			"uri": "xtop://snapshot/latest"
		}
	}`)
	r := resp.(mcps.JSONRPCResponse)
	result, ok := r.Result.(mcps.ReadResourceResult)
	if !ok {
		t.Fatalf("expected ReadResourceResult, got %T", r.Result)
	}
	if len(result.Contents) == 0 {
		t.Fatalf("expected non-empty resource contents")
	}
	// Verify the content is valid JSON.
	content := result.Contents[0].(mcps.TextResourceContents)
	var v any
	if err := json.Unmarshal([]byte(content.Text), &v); err != nil {
		t.Fatalf("resource content is not valid JSON: %v", err)
	}
}

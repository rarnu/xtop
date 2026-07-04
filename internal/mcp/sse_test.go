package mcp

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	mcpserver "github.com/mark3labs/mcp-go/server"

	"xtop/internal/collector"
)

func TestSSEServerEndpoints(t *testing.T) {
	c := collector.New()
	c.StartProcLoop()
	waitForFirstCaches(c, 500*time.Millisecond)

	sse := mcpserver.NewSSEServer(newMCPServer(c))

	// Start on a system-assigned port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	go func() {
		if err := http.Serve(ln, sse); err != nil && !strings.Contains(err.Error(), "closed") {
			t.Logf("sse serve error: %v", err)
		}
	}()

	baseURL := "http://" + ln.Addr().String()

	// Wait for the server to be ready.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if resp, err := http.Get(baseURL + "/sse"); err == nil {
			_ = resp.Body.Close()
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	resp, err := http.Get(baseURL + "/sse")
	if err != nil {
		t.Fatalf("get /sse: %v", err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("expected text/event-stream, got %s", ct)
	}

	// Read the first event containing the message endpoint.
	scanner := bufio.NewScanner(resp.Body)
	var messageEndpoint string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: endpoint") {
			// Next line should be the data line.
			if scanner.Scan() {
				data := scanner.Text()
				if strings.HasPrefix(data, "data: ") {
					messageEndpoint = strings.TrimPrefix(data, "data: ")
					break
				}
			}
		}
	}
	if messageEndpoint == "" {
		t.Fatal("did not receive message endpoint from /sse")
	}

	// Verify the message endpoint responds to OPTIONS (CORS preflight).
	req, _ := http.NewRequest(http.MethodOptions, baseURL+messageEndpoint, nil)
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Origin", "http://example.com")
	optionsResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("options message endpoint: %v", err)
	}
	_ = optionsResp.Body.Close()

	// Shutdown cleanly.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = sse.Shutdown(ctx)
}

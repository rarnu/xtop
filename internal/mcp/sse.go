package mcp

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	mcpserver "github.com/mark3labs/mcp-go/server"

	"xtop/internal/collector"
)

// serveSSE starts the MCP server using Server-Sent Events transport.
func serveSSE(c *collector.Collector, cfg Config) error {
	sse := mcpserver.NewSSEServer(newMCPServer(c))

	errCh := make(chan error, 1)
	go func() {
		fmt.Fprintf(os.Stderr, "xtop mcp: SSE server listening on http://%s/sse\n", cfg.Addr())
		errCh <- sse.Start(cfg.Addr())
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-sigCh:
		fmt.Fprintf(os.Stderr, "xtop mcp: received %s, shutting down...\n", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return sse.Shutdown(ctx)
	}
}

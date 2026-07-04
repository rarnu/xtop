// Package mcp implements the Model Context Protocol server for xtop.
package mcp

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"time"

	"xtop/internal/collector"
)

// Config holds the parsed `xtop mcp` subcommand flags.
type Config struct {
	Transport string
	Host      string
	Port      int
	Help      bool
}

// Addr returns the host:port listen address for SSE mode.
func (c Config) Addr() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

const usage = `xtop mcp - Model Context Protocol server

Exposes current system monitoring data to AI assistants via MCP.

Usage:
  xtop mcp [flags]

Flags:
  --help              Show this help message and exit
  --transport <type>  Transport type: stdio (default) or sse
  --host <addr>       Host address for SSE mode (default: 127.0.0.1)
  --port <n>          Port for SSE mode (default: 3001)

stdio mode (default):
  xtop mcp

  Configure your MCP client with:

    {
      "mcpServers": {
        "xtop": {
          "command": "xtop",
          "args": ["mcp"]
        }
      }
    }

SSE mode:
  xtop mcp --transport sse --port 8080

  Configure your MCP client with the SSE URL, e.g.:

    {
      "mcpServers": {
        "xtop": {
          "url": "http://127.0.0.1:8080/sse"
        }
      }
    }
`

// ParseArgs parses the `xtop mcp` subcommand arguments.
func ParseArgs(args []string) (Config, error) {
	fs := flag.NewFlagSet("xtop mcp", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var cfg Config
	fs.StringVar(&cfg.Transport, "transport", "stdio", "transport type: stdio or sse")
	fs.StringVar(&cfg.Host, "host", "127.0.0.1", "host address for SSE mode")
	fs.IntVar(&cfg.Port, "port", 3001, "port for SSE mode")
	fs.BoolVar(&cfg.Help, "help", false, "show help")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	if cfg.Transport != "stdio" && cfg.Transport != "sse" {
		return Config{}, fmt.Errorf("--transport must be 'stdio' or 'sse', got %q", cfg.Transport)
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return Config{}, fmt.Errorf("--port must be between 1 and 65535, got %d", cfg.Port)
	}

	return cfg, nil
}

// Run parses MCP subcommand arguments and starts the selected MCP server.
func Run(args []string) error {
	cfg, err := ParseArgs(args)
	if err != nil {
		return err
	}
	if cfg.Help {
		fmt.Fprint(os.Stdout, usage)
		return nil
	}

	c := collector.New()
	c.StartProcLoop()
	c.StartNetProcLoop()
	waitForFirstCaches(c, 3*time.Second)

	switch cfg.Transport {
	case "sse":
		return serveSSE(c, cfg)
	default:
		return serveStdio(c)
	}
}

// waitForFirstCaches blocks briefly until the process and network caches have
// been populated at least once, or the timeout expires.
func waitForFirstCaches(c *collector.Collector, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if ps := c.ProcCache(); len(ps.All) == 0 {
		select {
		case <-c.ProcUpdate():
		case <-ctx.Done():
		}
	}

	if !c.NetProcSupported() {
		select {
		case <-c.NetProcUpdate():
		case <-ctx.Done():
		}
	}
}

// PrintUsage writes the MCP usage message to w.
func PrintUsage(w *os.File) {
	_, _ = fmt.Fprint(w, usage)
}

// IsHelpError reports whether err is the flag.ErrHelp sentinel.
func IsHelpError(err error) bool {
	return errors.Is(err, flag.ErrHelp)
}

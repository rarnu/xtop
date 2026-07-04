// Package cli implements the non-interactive command-line mode of xtop.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

// Config holds the parsed command-line flags.
type Config struct {
	All     bool
	CPU     bool
	Mem     bool
	Disk    bool
	GPU     bool
	Net     bool
	Proc    bool
	JSON    bool
	Stream  int
	Help    bool
	Version bool
}

// HasContent reports whether the user requested at least one subsystem.
func (c Config) HasContent() bool {
	return c.All || c.CPU || c.Mem || c.Disk || c.GPU || c.Net || c.Proc
}

const usage = `xtop - a terminal system monitor

Usage:
  xtop                      Run the interactive TUI (default)
  xtop [flags]              Run command-line mode

Flags:
  --help                    Show this help message and exit
  --version                 Show version information and exit
  --all                     Output all information (CPU, memory, disk, GPU, network, processes)
  --cpu                     Output CPU usage
  --mem                     Output memory usage
  --disk                    Output disk usage
  --gpu                     Output GPU usage
  --net                     Output network usage
  --proc                    Output current process information
  --json                    Output in JSON format (default: plain text)
  --stream <x>              Stream output every x seconds (x must be an integer >= 1)

Examples:
  xtop --all --json --stream 5
  xtop --cpu --mem
  xtop --cpu --mem --json
  xtop --cpu --mem --json --stream 5
  xtop --cpu --mem --json --stream 5 --proc
  xtop --cpu --mem --proc --gpu

When no content flag (--cpu, --mem, etc.) is given, --all is assumed.
`

// ParseArgs parses the command-line arguments and returns a Config.
// It returns an error for unknown flags or invalid --stream values.
func ParseArgs(args []string) (Config, error) {
	fs := flag.NewFlagSet("xtop", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // silence default error output; we handle it ourselves

	var cfg Config
	fs.BoolVar(&cfg.All, "all", false, "output all information")
	fs.BoolVar(&cfg.CPU, "cpu", false, "output CPU usage")
	fs.BoolVar(&cfg.Mem, "mem", false, "output memory usage")
	fs.BoolVar(&cfg.Disk, "disk", false, "output disk usage")
	fs.BoolVar(&cfg.GPU, "gpu", false, "output GPU usage")
	fs.BoolVar(&cfg.Net, "net", false, "output network usage")
	fs.BoolVar(&cfg.Proc, "proc", false, "output process information")
	fs.BoolVar(&cfg.JSON, "json", false, "output in JSON format")
	fs.IntVar(&cfg.Stream, "stream", 0, "stream output every x seconds")
	fs.BoolVar(&cfg.Help, "help", false, "show help")
	fs.BoolVar(&cfg.Version, "version", false, "show version")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			cfg.Help = true
			return cfg, nil
		}
		return Config{}, err
	}

	// Detect whether --stream was explicitly supplied.
	streamSet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "stream" {
			streamSet = true
		}
	})
	if streamSet && cfg.Stream < 1 {
		return Config{}, fmt.Errorf("--stream requires an integer >= 1")
	}

	// If no content flag was given, default to --all.
	if !cfg.HasContent() {
		cfg.All = true
	}

	return cfg, nil
}

// PrintUsage writes the usage message to the given writer.
func PrintUsage(w *os.File) {
	_, _ = fmt.Fprint(w, usage)
}

// PrintHelp prints the usage message to stdout and exits with status 0.
func PrintHelp() {
	PrintUsage(os.Stdout)
	os.Exit(0)
}

// PrintError prints an error message to stderr, optionally followed by usage.
func PrintError(err error, showUsage bool) {
	msg := err.Error()
	if !strings.HasPrefix(msg, "xtop: ") {
		msg = "xtop: " + msg
	}
	_, _ = fmt.Fprintln(os.Stderr, msg)
	if showUsage {
		_, _ = fmt.Fprintln(os.Stderr)
		PrintUsage(os.Stderr)
	}
}

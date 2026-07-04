// Command xtop is a Matrix-green terminal system monitor (a top replacement).
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"xtop/internal/cli"
	"xtop/internal/tui"
)

func main() {
	// No arguments: launch the interactive TUI as before.
	if len(os.Args) == 1 {
		p := tea.NewProgram(
			tui.New(),
			tea.WithAltScreen(),
			tea.WithMouseCellMotion(),
		)
		if _, err := p.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "xtop:", err)
			os.Exit(1)
		}
		return
	}

	// With arguments: run command-line mode.
	cfg, err := cli.ParseArgs(os.Args[1:])
	if err != nil {
		cli.PrintError(err, true)
		os.Exit(2)
	}

	if cfg.Help {
		cli.PrintHelp()
	}
	if cfg.Version {
		cli.PrintVersion()
		os.Exit(0)
	}

	if err := cli.Run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "xtop:", err)
		os.Exit(1)
	}
}

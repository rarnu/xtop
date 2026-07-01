// Command xtop is a Matrix-green terminal system monitor (a top replacement).
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"xtop/internal/tui"
)

func main() {
	p := tea.NewProgram(
		tui.New(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "xtop:", err)
		os.Exit(1)
	}
}

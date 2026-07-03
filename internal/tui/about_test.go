package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestAboutDialogShownOnAKey(t *testing.T) {
	m := New()
	m.width, m.height = 120, 40
	m.recompute()

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	mm := m2.(*model)
	if !mm.about.active {
		t.Fatal("about dialog should be active after pressing 'a'")
	}

	v := mm.View()
	if !strings.Contains(stripANSI(v), "XTOP") {
		t.Fatal("about dialog should contain 'XTOP'")
	}
	if !strings.Contains(stripANSI(v), T("about.github")) {
		t.Fatal("about dialog should contain GitHub label")
	}
}

func TestAboutDialogClosesOnEsc(t *testing.T) {
	m := New()
	m.width, m.height = 120, 40
	m.recompute()
	m.about.active = true

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mm := m2.(*model)
	if mm.about.active {
		t.Fatal("about dialog should close on ESC")
	}
}

func TestAboutDialogClosesOnClickOutside(t *testing.T) {
	m := New()
	m.width, m.height = 120, 40
	m.recompute()
	m.about.active = true

	m2, _ := m.Update(tea.MouseMsg{X: 1, Y: 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	mm := m2.(*model)
	if mm.about.active {
		t.Fatal("about dialog should close on click outside")
	}
}

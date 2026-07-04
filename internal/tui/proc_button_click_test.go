package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestProcManagerButtonClick verifies that clicking the "open process manager"
// button in the process card opens the process manager modal, regardless of
// the display locale.
func TestProcManagerButtonClick(t *testing.T) {
	origDict := dict
	defer func() { dict = origDict }()

	locales := []string{"en_US", "zh_CN"}
	for _, locale := range locales {
		t.Run(locale, func(t *testing.T) {
			d := loadLocaleDict(t, locale)
			if d == nil {
				t.Fatalf("failed to load locale %s", locale)
			}
			dict = d

			m := New()
			m.width, m.height = 120, 40
			m.recompute()

			if !m.btnFound {
				t.Fatalf("button not found for locale %s", locale)
			}

			x := (m.btnX0 + m.btnX1) / 2
			y := m.btnLine
			m2, _ := m.Update(tea.MouseMsg{
				X:      x,
				Y:      y,
				Button: tea.MouseButtonLeft,
				Action: tea.MouseActionPress,
			})
			mm := m2.(*model)
			if !mm.modal {
				t.Fatalf("clicking the process manager button did not open the modal for locale %s (btnLine=%d btnX0=%d btnX1=%d click=%d,%d)",
					locale, m.btnLine, m.btnX0, m.btnX1, x, y)
			}
			mm.stopLoops()
		})
	}
}

func loadLocaleDict(t *testing.T, locale string) map[string]string {
	t.Helper()
	p := filepath.Join("..", "..", "lang", locale+".json")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read language file %s: %v", p, err)
	}
	var out map[string]string
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal language file %s: %v", p, err)
	}
	return out
}

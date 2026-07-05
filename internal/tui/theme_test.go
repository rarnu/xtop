package tui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetThemeFallsBackToDarkForMissingThirdParty(t *testing.T) {
	SetTheme(ThemeDark)
	SetTheme("nonexistent-theme")
	if CurrentThemeName() != ThemeDark {
		t.Fatalf("expected fallback to dark, got %s", CurrentThemeName())
	}
}

func TestLoadThirdPartyTheme(t *testing.T) {
	origHome := os.Getenv("HOME")
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	if err := os.MkdirAll(UserThemesDir(), 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(UserThemesDir(), "blue.json")
	if err := os.WriteFile(path, []byte(`{
  "primary": "#1976D2",
  "primaryHi": "#0D47A1",
  "primaryDim": "#64B5F6",
  "text": "#1565C0",
  "faint": "#42A5F5",
  "gray": "#90CAF9",
  "grayLite": "#BBDEFB",
  "track": "#E3F2FD",
  "red": "#D32F2F",
  "yellow": "#FBC02D",
  "orange": "#F57C00",
  "blue": "#0D47A1",
  "rowButtonBg": "#E3F2FD",
  "rowButtonDangerBg": "#FFEBEE",
  "selRowBg": "#BBDEFB",
  "selRowFg": "#0D47A1"
}`), 0644); err != nil {
		t.Fatal(err)
	}

	SetTheme("blue")
	if CurrentThemeName() != "blue" {
		t.Fatalf("expected blue theme, got %s", CurrentThemeName())
	}

	// Delete the file and verify fallback.
	_ = os.Remove(path)
	SetTheme("blue")
	if CurrentThemeName() != ThemeDark {
		t.Fatalf("expected fallback to dark after deletion, got %s", CurrentThemeName())
	}
}

package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeLocale(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"zh_CN.UTF-8", "zh_CN"},
		{"en_US.utf8", "en_US"},
		{"zh_CN", "zh_CN"},
		{"C", "en_US"},
		{"POSIX", "en_US"},
	}
	for _, c := range cases {
		got := normalizeLocale(c.in)
		if got != c.want {
			t.Errorf("normalizeLocale(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestTFFallback(t *testing.T) {
	// With no language files loaded, T should return the builtin English value.
	if got := T("card.mem"); got != "MEM" {
		t.Errorf("T(card.mem) = %q, want MEM", got)
	}
	if got := Tf("proc.modal.count", 5); got != "5 processes" {
		t.Errorf("Tf(proc.modal.count, 5) = %q, want '5 processes'", got)
	}
}

func TestLoadLangFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zh_CN.json")
	if err := os.WriteFile(path, []byte(`{"card.cpu":"CPU","card.mem":"内存"}`), 0644); err != nil {
		t.Fatal(err)
	}
	d, ok := loadLangFile(path)
	if !ok {
		t.Fatal("loadLangFile returned false")
	}
	if d["card.mem"] != "内存" {
		t.Errorf("got %q, want 内存", d["card.mem"])
	}
}

func TestLangFilePath(t *testing.T) {
	// Ensure the language file path is inside ~/.xtop/lang.
	path := langFilePath("zh_CN")
	if filepath.Base(path) != "zh_CN.json" {
		t.Errorf("basename = %q", filepath.Base(path))
	}
	if !contains(path, ".xtop/lang") {
		t.Errorf("path %q does not contain '.xtop/lang'", path)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

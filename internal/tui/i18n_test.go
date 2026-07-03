package tui

import (
	"os"
	"path/filepath"
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

func TestLangFilePaths(t *testing.T) {
	// Ensure search order is /etc/xtop/lang, ~/.xtop/lang, ./lang.
	paths := langFilePaths("zh_CN")
	if len(paths) != 3 {
		t.Fatalf("expected 3 paths, got %d", len(paths))
	}
	if paths[0] != "/etc/xtop/lang/zh_CN.json" {
		t.Errorf("first path = %q", paths[0])
	}
	if filepath.Base(paths[2]) != "zh_CN.json" {
		t.Errorf("last path basename = %q", filepath.Base(paths[2]))
	}
}

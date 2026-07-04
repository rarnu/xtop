package cli

import (
	"testing"
)

func TestParseArgsDefaults(t *testing.T) {
	cfg, err := ParseArgs([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Help || cfg.Version || cfg.JSON || cfg.Stream != 0 {
		t.Fatalf("expected zero flags for empty args, got %+v", cfg)
	}
	// No content flags were given, so --all is assumed. (main bypasses CLI mode
	// when there are no arguments at all and starts the TUI instead.)
	if !cfg.All {
		t.Fatalf("expected All=true when no content flags are present")
	}
}

func TestParseArgsNoContentDefaultsToAll(t *testing.T) {
	for _, args := range [][]string{
		{"--json"},
		{"--stream", "5"},
		{"--json", "--stream", "2"},
	} {
		cfg, err := ParseArgs(args)
		if err != nil {
			t.Fatalf("args %v: unexpected error: %v", args, err)
		}
		if !cfg.All {
			t.Fatalf("args %v: expected All to default to true, got %+v", args, cfg)
		}
	}
}

func TestParseArgsContentFlags(t *testing.T) {
	cfg, err := ParseArgs([]string{"--cpu", "--mem", "--json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.CPU || !cfg.Mem || !cfg.JSON {
		t.Fatalf("expected cpu, mem and json, got %+v", cfg)
	}
	if cfg.All {
		t.Fatalf("expected All=false when content flags are present")
	}
}

func TestParseArgsStream(t *testing.T) {
	cfg, err := ParseArgs([]string{"--all", "--stream", "5"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Stream != 5 {
		t.Fatalf("expected Stream=5, got %d", cfg.Stream)
	}

	if _, err := ParseArgs([]string{"--stream", "0"}); err == nil {
		t.Fatalf("expected error for --stream 0")
	}
	if _, err := ParseArgs([]string{"--stream", "-1"}); err == nil {
		t.Fatalf("expected error for --stream -1")
	}
	if _, err := ParseArgs([]string{"--stream", "abc"}); err == nil {
		t.Fatalf("expected error for --stream abc")
	}
}

func TestParseArgsHelp(t *testing.T) {
	cfg, err := ParseArgs([]string{"--help"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Help {
		t.Fatalf("expected Help=true")
	}
}

func TestParseArgsVersion(t *testing.T) {
	cfg, err := ParseArgs([]string{"--version"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Version {
		t.Fatalf("expected Version=true")
	}
}

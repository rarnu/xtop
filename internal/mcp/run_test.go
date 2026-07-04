package mcp

import (
	"testing"
)

func TestParseArgsDefaults(t *testing.T) {
	cfg, err := ParseArgs(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Transport != "stdio" {
		t.Fatalf("expected default transport stdio, got %s", cfg.Transport)
	}
	if cfg.Host != "127.0.0.1" {
		t.Fatalf("expected default host 127.0.0.1, got %s", cfg.Host)
	}
	if cfg.Port != 3001 {
		t.Fatalf("expected default port 3001, got %d", cfg.Port)
	}
	if cfg.Help {
		t.Fatal("expected Help=false")
	}
}

func TestParseArgsSSE(t *testing.T) {
	cfg, err := ParseArgs([]string{"--transport", "sse", "--port", "8080", "--host", "0.0.0.0"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Transport != "sse" {
		t.Fatalf("expected transport sse, got %s", cfg.Transport)
	}
	if cfg.Port != 8080 {
		t.Fatalf("expected port 8080, got %d", cfg.Port)
	}
	if cfg.Host != "0.0.0.0" {
		t.Fatalf("expected host 0.0.0.0, got %s", cfg.Host)
	}
	if cfg.Addr() != "0.0.0.0:8080" {
		t.Fatalf("expected address 0.0.0.0:8080, got %s", cfg.Addr())
	}
}

func TestParseArgsInvalid(t *testing.T) {
	if _, err := ParseArgs([]string{"--transport", "foo"}); err == nil {
		t.Fatal("expected error for invalid transport")
	}
	if _, err := ParseArgs([]string{"--port", "0"}); err == nil {
		t.Fatal("expected error for port 0")
	}
	if _, err := ParseArgs([]string{"--port", "99999"}); err == nil {
		t.Fatal("expected error for port 99999")
	}
}

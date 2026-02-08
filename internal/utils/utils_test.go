package utils

import (
	"os"
	"testing"
)

func TestStringPtr(t *testing.T) {
	s := "test"
	ptr := StringPtr(s)
	if ptr == nil {
		t.Fatal("expected pointer, got nil")
	}
	if *ptr != s {
		t.Errorf("expected %q, got %q", s, *ptr)
	}
}

func TestDefaultString(t *testing.T) {
	tests := []struct {
		name     string
		s        *string
		def      string
		expected string
	}{
		{"nil", nil, "default", "default"},
		{"value", StringPtr("test"), "default", "test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DefaultString(tt.s, tt.def); got != tt.expected {
				t.Errorf("DefaultString() = %q, expected %q", got, tt.expected)
			}
		})
	}
}

func TestBoolPtr(t *testing.T) {
	b := true
	ptr := BoolPtr(b)
	if ptr == nil {
		t.Fatal("expected pointer, got nil")
	}
	if *ptr != b {
		t.Errorf("expected %v, got %v", b, *ptr)
	}
}

func TestDefaultBool(t *testing.T) {
	tests := []struct {
		name     string
		b        *bool
		def      bool
		expected bool
	}{
		{"nil", nil, true, true},
		{"value true", BoolPtr(true), false, true},
		{"value false", BoolPtr(false), true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DefaultBool(tt.b, tt.def); got != tt.expected {
				t.Errorf("DefaultBool() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestShellQuote(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected string
	}{
		{"empty", "", "''"},
		{"simple", "hello", "hello"},
		{"space", "hello world", "'hello world'"},
		{"single quote", "don't", "'don'\"'\"'t'"},
		{"double quote", `say "hello"`, `'say "hello"'`},
		{"variable", "$HOME", "'$HOME'"},
		{"complex", `foo 'bar' "baz"`, `'foo '"'"'bar'"'"' "baz"'`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShellQuote(tt.s); got != tt.expected {
				t.Errorf("ShellQuote() = %q, expected %q", got, tt.expected)
			}
		})
	}
}

func TestRemoveWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected *string
	}{
		{"empty", "", nil},
		{"spaces", "   ", nil},
		{"value", " test ", StringPtr(" test ")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemoveWhitespace(tt.s)
			if tt.expected == nil {
				if got != nil {
					t.Errorf("RemoveWhitespace() = %q, expected nil", *got)
				}
			} else {
				if got == nil {
					t.Fatal("expected pointer, got nil")
				}
				if *got != *tt.expected {
					t.Errorf("RemoveWhitespace() = %q, expected %q", *got, *tt.expected)
				}
			}
		})
	}
}

func TestNonce(t *testing.T) {
	n1 := Nonce()
	n2 := Nonce()
	if len(n1) != 16 {
		t.Errorf("expected length 16, got %d", len(n1))
	}
	if n1 == n2 {
		t.Error("expected different nonces")
	}
}

func TestGetShell(t *testing.T) {
	originalShell := os.Getenv("SHELL")
	defer os.Setenv("SHELL", originalShell)

	tests := []struct {
		name     string
		setEnv   bool
		envVal   string
		expected string
	}{
		{"set", true, "/bin/zsh", "/bin/zsh"},
		{"unset", false, "", "/bin/sh"},
		{"empty", true, "", "/bin/sh"},
		{"whitespace", true, "   ", "/bin/sh"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				os.Setenv("SHELL", tt.envVal)
			} else {
				os.Unsetenv("SHELL")
			}
			if got := GetShell(); got != tt.expected {
				t.Errorf("GetShell() = %q, expected %q", got, tt.expected)
			}
		})
	}
}

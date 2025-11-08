package utils

import (
	"math/rand"
	"os"
	"strings"
)

func StringPtr(s string) *string {
	return &s
}

func DerefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func DefaultString(s *string, defaultValue string) string {
	if s == nil {
		return defaultValue
	}
	return *s
}

func DefaultBool(b *bool, defaultValue bool) bool {
	if b == nil {
		return defaultValue
	}
	return *b
}

func ShellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n\"'\\$`") {
		return s
	}
	// Replace every ' with '\'' (close, escape single quote, reopen)
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func RemoveWhitespace(str string) *string {
	if strings.TrimSpace(str) == "" {
		return nil
	}
	return &str
}

func Nonce() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 16)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func GetShell() string {
	shell, exists := os.LookupEnv("SHELL")
	if !exists || RemoveWhitespace(shell) == nil {
		shell = "/bin/sh"
	}
	return shell
}

package utils

import (
	"strings"
)

func StringPtr(s string) *string {
	return &s
}

func DefaultString(s *string, defaultValue string) string {
	if s == nil {
		return defaultValue
	}
	return *s
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

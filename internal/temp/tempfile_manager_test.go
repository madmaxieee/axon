package temp

import (
	"os"
	"testing"
)

func TestTempFileManager(t *testing.T) {
	manager := NewManager("")
	defer manager.Cleanup()

	handleID := "test-handle"

	// Test GetTempFile
	handle, err := manager.GetTempFile(handleID)
	if err != nil {
		t.Fatalf("Failed to get temp file: %v", err)
	}

	if handle.Path() == "" {
		t.Fatal("Expected non-empty path")
	}

	// Test WriteText
	err = handle.WriteText("hello ")
	if err != nil {
		t.Fatalf("Failed to write text: %v", err)
	}

	// Test AppendText
	err = handle.AppendText("world")
	if err != nil {
		t.Fatalf("Failed to append text: %v", err)
	}

	// Verify content
	content, err := os.ReadFile(handle.Path())
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	expected := "hello world"
	if string(content) != expected {
		t.Errorf("Expected %q, got %q", expected, string(content))
	}

	// Test getting the same handle returns the exact same underlying path
	handle2, err := manager.GetTempFile(handleID)
	if err != nil {
		t.Fatalf("Failed to get existing temp file: %v", err)
	}

	if handle.Path() != handle2.Path() {
		t.Errorf("Expected paths to be equal, got %q and %q", handle.Path(), handle2.Path())
	}
}

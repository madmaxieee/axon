package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetPromptByName(t *testing.T) {
	dir := t.TempDir()

	cfg := Config{
		ConfigFile: &ConfigFile{
			General: GeneralConfig{
				PromptPath: []string{dir},
			},
		},
		Prompts: make(map[string]Prompt),
	}

	// 1. Not found initially, triggers scan, but still not found
	_, err := cfg.GetPromptByName("missing")
	if err == nil || err.Error() != "prompt missing not found" {
		t.Errorf("expected missing error, got %v", err)
	}

	// 2. Found in map (loaded)
	content := "preloaded"
	cfg.Prompts["loaded"] = Prompt{
		Name:   "loaded",
		System: &content,
		loaded: true,
	}

	p, err := cfg.GetPromptByName("loaded")
	if err != nil || p == nil {
		t.Errorf("unexpected error: %v", err)
	}
	if p.System == nil || *p.System != "preloaded" {
		t.Errorf("expected preloaded content, got %v", p.System)
	}

	// 3. Not in map, scan finds it on disk
	filePath := filepath.Join(dir, "diskprompt.md")
	os.WriteFile(filePath, []byte("disk content"), 0644)

	p2, err := cfg.GetPromptByName("diskprompt")
	if err != nil || p2 == nil {
		t.Fatalf("expected to find diskprompt, got %v", err)
	}
	if p2.System == nil || *p2.System != "disk content" {
		t.Errorf("expected disk content, got %v", p2.System)
	}
}

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrompt_LoadContent_Directory(t *testing.T) {
	dir := t.TempDir()
	promptDir := filepath.Join(dir, "myprompt")
	err := os.Mkdir(promptDir, 0755)
	if err != nil {
		t.Fatalf("failed to create prompt dir: %v", err)
	}

	sysPath := filepath.Join(promptDir, "system.md")
	userPath := filepath.Join(promptDir, "user.md")

	os.WriteFile(sysPath, []byte("system content"), 0644)
	os.WriteFile(userPath, []byte("user content"), 0644)

	promptPath := promptDir
	p := Prompt{
		Name: "myprompt",
		Path: &promptPath,
	}

	loaded, err := p.LoadContent()
	if err != nil || !loaded {
		t.Fatalf("failed to load content: %v", err)
	}

	if *p.System != "system content" {
		t.Errorf("expected 'system content', got %q", *p.System)
	}
	if *p.User != "user content" {
		t.Errorf("expected 'user content', got %q", *p.User)
	}
}

func TestPrompt_LoadContent_File(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "single.md")

	os.WriteFile(filePath, []byte("single prompt content"), 0644)

	p := Prompt{
		Name: "single",
		Path: &filePath,
	}

	loaded, err := p.LoadContent()
	if err != nil || !loaded {
		t.Fatalf("failed to load content: %v", err)
	}

	if *p.System != "single prompt content" {
		t.Errorf("expected 'single prompt content', got %v", *p.System)
	}
	if p.User != nil {
		t.Errorf("expected user to be nil, got %v", *p.User)
	}
}

func TestScanPromptPath(t *testing.T) {
	dir := t.TempDir()

	// Create a dummy config where prompt path is our temp dir
	cfg := Config{
		ConfigFile: &ConfigFile{
			General: GeneralConfig{
				PromptPath: []string{dir},
			},
		},
		Prompts: make(map[string]Prompt),
	}

	// Create a prompt dir
	promptDir := filepath.Join(dir, "dir-prompt")
	os.Mkdir(promptDir, 0755)
	os.WriteFile(filepath.Join(promptDir, "system.md"), []byte("system"), 0644)

	// Create a prompt file
	os.WriteFile(filepath.Join(dir, "file-prompt.md"), []byte("file"), 0644)

	// Create something to ignore
	os.WriteFile(filepath.Join(dir, "ignore.txt"), []byte("ignore"), 0644)

	cfg.scanPromptPath()

	if len(cfg.Prompts) != 2 {
		t.Fatalf("expected 2 prompts, got %d", len(cfg.Prompts))
	}

	if _, ok := cfg.Prompts["dir-prompt"]; !ok {
		t.Errorf("expected dir-prompt to be found")
	}

	if _, ok := cfg.Prompts["file-prompt"]; !ok {
		t.Errorf("expected file-prompt to be found")
	}

	// Test GetPromptByName which uses scanPromptPath under the hood if not loaded
	p, err := cfg.GetPromptByName("dir-prompt")
	if err != nil || p == nil {
		t.Fatalf("expected dir-prompt, got err=%v", err)
	}

	// Because LoadContent modifies the prompt inline, and the map stores Prompt by value... wait.
	// Check if GetPromptByName actually modifies the map entry correctly?
	// GetPromptByName returns `&prompt` where `prompt` is a copy from the map, but it calls `prompt.LoadContent()`.
	// Wait, since `prompt` is a local variable, `prompt.LoadContent()` will only load the content on that local copy! Let's check.
	// If it doesn't update the map, it's fine as long as `p` has the content.
	if p.System == nil || *p.System != "system" {
		t.Errorf("expected system content 'system', got %v", p.System)
	}
}

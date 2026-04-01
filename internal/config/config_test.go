package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(cfgPath, []byte(`
provider: claude
providers:
  claude:
    auth: api_key
    model: claude-sonnet-4-6-20250514
brain:
  path: ./brain
  max_context_files: 10
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Provider != "claude" {
		t.Errorf("Provider = %q, want %q", cfg.Provider, "claude")
	}
	if cfg.Brain.Path != "./brain" {
		t.Errorf("Brain.Path = %q, want %q", cfg.Brain.Path, "./brain")
	}
	if cfg.Brain.MaxContextFiles != 10 {
		t.Errorf("Brain.MaxContextFiles = %d, want %d", cfg.Brain.MaxContextFiles, 10)
	}
	if cfg.Providers["claude"].Model != "claude-sonnet-4-6-20250514" {
		t.Errorf("Providers[claude].Model = %q, want %q", cfg.Providers["claude"].Model, "claude-sonnet-4-6-20250514")
	}
}

func TestLoadConfig_EnvVarExpansion(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	t.Setenv("TEST_API_KEY", "sk-test-123")
	err := os.WriteFile(cfgPath, []byte(`
provider: claude
providers:
  claude:
    auth: api_key
    api_key: ${TEST_API_KEY}
    model: claude-sonnet-4-6-20250514
brain:
  path: ./brain
  max_context_files: 10
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Providers["claude"].APIKey != "sk-test-123" {
		t.Errorf("APIKey = %q, want %q", cfg.Providers["claude"].APIKey, "sk-test-123")
	}
}

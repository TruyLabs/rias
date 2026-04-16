package importer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TruyLabs/rias/internal/importer"
)

func TestLoadManifestMissingFile(t *testing.T) {
	m, err := importer.LoadManifest("/tmp/rias_nonexistent_manifest.json")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if len(m.Processed) != 0 {
		t.Errorf("expected empty manifest, got %d entries", len(m.Processed))
	}
}

func TestSaveAndLoadManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")

	m := &importer.ImportManifest{
		Processed: map[string]string{"conv-1": "abc123", "conv-2": "def456"},
	}
	if err := importer.SaveManifest(path, m); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}

	loaded, err := importer.LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if loaded.Processed["conv-1"] != "abc123" {
		t.Errorf("expected conv-1=abc123, got %q", loaded.Processed["conv-1"])
	}
	if loaded.Processed["conv-2"] != "def456" {
		t.Errorf("expected conv-2=def456, got %q", loaded.Processed["conv-2"])
	}
}

func TestConversationHash(t *testing.T) {
	c := importer.Conversation{
		ID:    "x",
		Title: "test",
		Messages: []importer.Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there"},
		},
	}
	h1 := importer.ConversationHash(c)
	h2 := importer.ConversationHash(c)
	if h1 != h2 {
		t.Error("ConversationHash is not deterministic")
	}
	if len(h1) != 16 {
		t.Errorf("expected 16-char hash, got %d chars: %q", len(h1), h1)
	}

	// Different content → different hash
	c.Messages[0].Content = "Different"
	h3 := importer.ConversationHash(c)
	if h1 == h3 {
		t.Error("expected different hash for different content")
	}
}

func TestManifestFileIsCreatedByParent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")

	m := &importer.ImportManifest{Processed: map[string]string{}}
	if err := importer.SaveManifest(path, m); err != nil {
		t.Fatalf("SaveManifest should not fail when dir exists: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("manifest file not created: %v", err)
	}
}

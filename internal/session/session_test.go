package session

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/tinhvqbk/kai/internal/provider"
)

func TestSaveAndLoadSession(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)

	s := &Session{
		ID:        "2026-03-25T11-30-00",
		StartedAt: time.Date(2026, 3, 25, 11, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 3, 25, 11, 45, 0, 0, time.UTC),
		Provider:  "claude",
		Messages: []SessionMessage{
			{
				Message:   provider.Message{Role: "user", Content: "hello"},
				Timestamp: time.Date(2026, 3, 25, 11, 30, 0, 0, time.UTC),
			},
			{
				Message:        provider.Message{Role: "assistant", Content: "Hi Kyle!"},
				Timestamp:      time.Date(2026, 3, 25, 11, 30, 5, 0, time.UTC),
				BrainFilesUsed: []string{"identity/profile.md"},
				Confidence:     "high",
			},
		},
		LearningsExtracted: 0,
	}

	if err := mgr.Save(s); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify file exists
	expectedPath := filepath.Join(dir, "2026-03-25T11-30-00.json")
	loaded, err := mgr.Load(expectedPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if loaded.ID != s.ID {
		t.Errorf("ID = %q, want %q", loaded.ID, s.ID)
	}
	if len(loaded.Messages) != 2 {
		t.Errorf("Messages count = %d, want 2", len(loaded.Messages))
	}
	if loaded.Messages[1].Confidence != "high" {
		t.Errorf("Confidence = %q, want %q", loaded.Messages[1].Confidence, "high")
	}
}

func TestNewSession(t *testing.T) {
	mgr := NewManager(t.TempDir())
	s := mgr.New("claude")

	if s.ID == "" {
		t.Error("ID should not be empty")
	}
	if s.Provider != "claude" {
		t.Errorf("Provider = %q, want %q", s.Provider, "claude")
	}
	if len(s.Messages) != 0 {
		t.Errorf("Messages should be empty, got %d", len(s.Messages))
	}
}

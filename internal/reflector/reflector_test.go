package reflector_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TruyLabs/rias/internal/provider"
	"github.com/TruyLabs/rias/internal/reflector"
	"github.com/TruyLabs/rias/internal/session"
)

func TestParseSinceDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"7d", 7 * 24 * time.Hour, false},
		{"30d", 30 * 24 * time.Hour, false},
		{"1w", 7 * 24 * time.Hour, false},
		{"2w", 14 * 24 * time.Hour, false},
		{"invalid", 0, true},
		{"0d", 0, true},
		{"0w", 0, true},
		{"", 0, true},
	}
	for _, tt := range tests {
		d, err := reflector.ParseSinceDuration(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseSinceDuration(%q): expected error, got nil", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseSinceDuration(%q): unexpected error: %v", tt.input, err)
			continue
		}
		if d != tt.expected {
			t.Errorf("ParseSinceDuration(%q): expected %v, got %v", tt.input, tt.expected, d)
		}
	}
}

// writeTestSession writes a session JSON file to dir.
func writeTestSession(t *testing.T, dir string, s *session.Session) {
	t.Helper()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, s.ID+".json"), data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadRecentSessionsAllSessions(t *testing.T) {
	dir := t.TempDir()
	s := &session.Session{
		ID:        "2026-01-01T12-00-00-abcd",
		StartedAt: time.Now(),
		Messages: []session.SessionMessage{
			{Message: provider.Message{Role: "user", Content: "Hello"}},
		},
	}
	writeTestSession(t, dir, s)

	sessions, err := reflector.LoadRecentSessions(dir, time.Time{})
	if err != nil {
		t.Fatalf("LoadRecentSessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].ID != s.ID {
		t.Errorf("expected session ID %q, got %q", s.ID, sessions[0].ID)
	}
}

func TestLoadRecentSessionsMissingDir(t *testing.T) {
	sessions, err := reflector.LoadRecentSessions("/tmp/rias_nonexistent_sessions_dir_xyz", time.Time{})
	if err != nil {
		t.Fatalf("expected no error for missing dir, got: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions for missing dir, got %d", len(sessions))
	}
}

func TestLoadRecentSessionsFiltersByCutoff(t *testing.T) {
	dir := t.TempDir()

	old := &session.Session{
		ID:        "2026-01-01T00-00-00-aaaa",
		StartedAt: time.Now().Add(-30 * 24 * time.Hour),
		Messages:  []session.SessionMessage{},
	}
	writeTestSession(t, dir, old)

	// Set the old session file's modification time to 30 days ago
	oldPath := filepath.Join(dir, old.ID+".json")
	oldTime := time.Now().Add(-30 * 24 * time.Hour)
	if err := os.Chtimes(oldPath, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	new := &session.Session{
		ID:        "2026-04-16T00-00-00-bbbb",
		StartedAt: time.Now(),
		Messages:  []session.SessionMessage{},
	}
	writeTestSession(t, dir, new)

	// Cutoff: 1 day ago — old session should be filtered out
	cutoff := time.Now().Add(-24 * time.Hour)
	sessions, err := reflector.LoadRecentSessions(dir, cutoff)
	if err != nil {
		t.Fatalf("LoadRecentSessions: %v", err)
	}
	for _, s := range sessions {
		if s.ID == old.ID {
			t.Errorf("old session should have been filtered out")
		}
	}
}

func TestSessionsToMessages(t *testing.T) {
	sessions := []*session.Session{
		{
			ID: "s1",
			Messages: []session.SessionMessage{
				{Message: provider.Message{Role: "user", Content: "What is Go?"}},
				{Message: provider.Message{Role: "assistant", Content: "Go is a language."}},
			},
		},
		{
			ID: "s2",
			Messages: []session.SessionMessage{
				{Message: provider.Message{Role: "user", Content: "Tell me more."}},
			},
		},
	}
	msgs := reflector.SessionsToMessages(sessions)
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "What is Go?" {
		t.Errorf("unexpected first message: %+v", msgs[0])
	}
	if msgs[2].Content != "Tell me more." {
		t.Errorf("unexpected third message: %+v", msgs[2])
	}
}

func TestSessionsToMessagesSkipsEmpty(t *testing.T) {
	sessions := []*session.Session{
		{
			ID: "s1",
			Messages: []session.SessionMessage{
				{Message: provider.Message{Role: "user", Content: ""}}, // empty — skip
				{Message: provider.Message{Role: "user", Content: "Real message"}},
			},
		},
	}
	msgs := reflector.SessionsToMessages(sessions)
	if len(msgs) != 1 {
		t.Errorf("expected empty content skipped, got %d messages", len(msgs))
	}
	if msgs[0].Content != "Real message" {
		t.Errorf("expected 'Real message', got %q", msgs[0].Content)
	}
}

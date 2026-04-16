package importer_test

import (
	"testing"

	"github.com/TruyLabs/rias/internal/importer"
)

const claudeExportJSON = `[
  {
    "uuid": "conv-1",
    "name": "Test conversation",
    "chat_messages": [
      {"uuid": "m1", "sender": "human", "text": "What is Go?"},
      {"uuid": "m2", "sender": "assistant", "text": "Go is a compiled language."},
      {"uuid": "m3", "sender": "human", "text": "Who made it?"},
      {"uuid": "m4", "sender": "assistant", "text": "Google created Go."}
    ]
  },
  {
    "uuid": "conv-2",
    "name": "Empty convo",
    "chat_messages": []
  }
]`

func TestParseClaude(t *testing.T) {
	convs, err := importer.ParseClaude([]byte(claudeExportJSON))
	if err != nil {
		t.Fatalf("ParseClaude: %v", err)
	}
	if len(convs) != 2 {
		t.Fatalf("expected 2 conversations, got %d", len(convs))
	}

	c := convs[0]
	if c.ID != "conv-1" {
		t.Errorf("expected ID=conv-1, got %q", c.ID)
	}
	if c.Title != "Test conversation" {
		t.Errorf("expected Title=Test conversation, got %q", c.Title)
	}
	if len(c.Messages) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(c.Messages))
	}
	if c.Messages[0].Role != "user" || c.Messages[0].Content != "What is Go?" {
		t.Errorf("unexpected first message: %+v", c.Messages[0])
	}
	if c.Messages[1].Role != "assistant" {
		t.Errorf("expected assistant role, got %q", c.Messages[1].Role)
	}

	// Empty conversation is included but has no messages
	if convs[1].ID != "conv-2" {
		t.Errorf("expected conv-2, got %q", convs[1].ID)
	}
	if len(convs[1].Messages) != 0 {
		t.Errorf("expected 0 messages for empty convo, got %d", len(convs[1].Messages))
	}
}

func TestParseClaudeInvalidJSON(t *testing.T) {
	_, err := importer.ParseClaude([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

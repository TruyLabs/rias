package prompt

import (
	"strings"
	"testing"
	"time"

	"github.com/TruyLabs/rias/internal/brain"
	"github.com/TruyLabs/rias/internal/provider"
)

func TestBuildSystemPrompt(t *testing.T) {
	brainFiles := []*brain.BrainFile{
		{
			Path:       "identity/profile.md",
			Tags:       []string{"identity"},
			Confidence: "high",
			Updated:    brain.DateOnly{Time: time.Now()},
			Content:    "\nKyle is a software engineer.\n",
		},
		{
			Path:       "opinions/golang.md",
			Tags:       []string{"go", "languages"},
			Confidence: "high",
			Updated:    brain.DateOnly{Time: time.Now()},
			Content:    "\nKyle prefers Go for CLI tools.\n",
		},
	}

	b := NewBuilder("kai", "Kyle")
	prompt := b.BuildSystemPrompt(brainFiles)

	if !strings.Contains(prompt, "Kyle is a software engineer") {
		t.Error("prompt missing identity content")
	}
	if !strings.Contains(prompt, "Kyle prefers Go") {
		t.Error("prompt missing opinion content")
	}
	if !strings.Contains(prompt, "digital twin") {
		t.Error("prompt missing base personality instructions")
	}
}

func TestBuildMessages(t *testing.T) {
	b := NewBuilder("kai", "Kyle")
	history := []provider.Message{
		{Role: "user", Content: "what do I think about Go?"},
		{Role: "assistant", Content: "You love Go."},
	}
	userInput := "and what about Python?"

	msgs := b.BuildMessages(history, userInput)

	if len(msgs) != 3 { // 2 history + new user
		t.Errorf("got %d messages, want 3", len(msgs))
	}
	if msgs[2].Role != "user" || msgs[2].Content != "and what about Python?" {
		t.Errorf("last message should be new user input")
	}
}

func TestBuildReflectPrompt(t *testing.T) {
	b := NewBuilder("rias", "User")
	msgs := []provider.Message{
		{Role: "user", Content: "how do I structure a Go service?"},
		{Role: "assistant", Content: "Here are some patterns..."},
		{Role: "user", Content: "I prefer clean architecture"},
	}

	p := b.BuildReflectPrompt(nil, msgs)
	if p == "" {
		t.Error("expected non-empty prompt")
	}
	if !strings.Contains(p, "patterns") {
		t.Error("expected prompt to mention patterns analysis")
	}
	if !strings.Contains(p, "how do I structure a Go service") {
		t.Error("expected prompt to include session messages")
	}

	// With brain context files
	files := []*brain.BrainFile{
		{
			Path:    "style/writing.md",
			Content: "\nUser writes concisely.\n",
		},
	}
	p2 := b.BuildReflectPrompt(files, msgs)
	if !strings.Contains(p2, "User writes concisely") {
		t.Error("expected prompt to include brain context")
	}
}

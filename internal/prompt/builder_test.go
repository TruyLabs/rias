package prompt

import (
	"strings"
	"testing"
	"time"

	"github.com/norenis/kai/internal/brain"
	"github.com/norenis/kai/internal/provider"
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

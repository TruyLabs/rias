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

func TestBuildSystemPromptVocabularyMirror(t *testing.T) {
	files := []*brain.BrainFile{
		{Path: "style/writing.md", Content: "\nUses short sentences. Avoids jargon.\n"},
		{Path: "identity/profile.md", Content: "\nSoftware engineer.\n"},
	}
	b := NewBuilder("rias", "Kyle")
	p := b.BuildSystemPrompt(files)

	styleIdx := strings.Index(p, "Uses short sentences")
	identityIdx := strings.Index(p, "Software engineer")
	if styleIdx == -1 {
		t.Error("style content missing from prompt")
	}
	if identityIdx == -1 {
		t.Error("identity content missing from prompt")
	}
	if styleIdx > identityIdx {
		t.Error("style section should appear before other brain files")
	}
	if !strings.Contains(p, "Mirror") {
		t.Error("expected 'Mirror' directive in style section")
	}
}

func TestBuildSystemPromptNoStyleFiles(t *testing.T) {
	files := []*brain.BrainFile{
		{Path: "identity/profile.md", Content: "\nSoftware engineer.\n"},
	}
	b := NewBuilder("rias", "Kyle")
	p := b.BuildSystemPrompt(files)
	if strings.Contains(p, "Mirror") {
		t.Error("should not have style mirror section when no style files present")
	}
	if !strings.Contains(p, "Software engineer") {
		t.Error("identity content should still appear")
	}
}

func TestBuildSystemPromptStyleOnly(t *testing.T) {
	files := []*brain.BrainFile{
		{Path: "style/voice.md", Content: "\nDirect and terse.\n"},
	}
	b := NewBuilder("rias", "Kyle")
	p := b.BuildSystemPrompt(files)
	if !strings.Contains(p, "Direct and terse") {
		t.Error("style content missing")
	}
	if strings.Contains(p, "### style/voice.md") {
		t.Error("style files must not emit a ### heading")
	}
	if !strings.Contains(p, "Mirror") {
		t.Error("Mirror directive missing")
	}
}

func TestBuildSystemPromptMultipleStyleFiles(t *testing.T) {
	files := []*brain.BrainFile{
		{Path: "style/voice.md", Content: "\nDirect.\n"},
		{Path: "style/format.md", Content: "\nBullets preferred.\n"},
	}
	b := NewBuilder("rias", "Kyle")
	p := b.BuildSystemPrompt(files)
	if !strings.Contains(p, "Direct") || !strings.Contains(p, "Bullets preferred") {
		t.Error("both style files must appear in output")
	}
	if strings.Count(p, "Mirror") != 1 {
		t.Errorf("expected exactly one Mirror directive, got %d", strings.Count(p, "Mirror"))
	}
}

func TestBuildSystemPromptProactiveRecall(t *testing.T) {
	b := NewBuilder("rias", "Kyle")
	b.SetProactiveRecall(true)
	p := b.BuildSystemPrompt(nil)
	if !strings.Contains(p, "proactively") {
		t.Error("expected proactive recall directive in prompt when enabled")
	}
}

func TestBuildSystemPromptProactiveRecallDisabled(t *testing.T) {
	b := NewBuilder("rias", "Kyle")
	// proactiveRecall defaults false — do NOT call SetProactiveRecall
	p := b.BuildSystemPrompt(nil)
	if strings.Contains(p, "proactively") {
		t.Error("proactive recall directive should not appear when disabled")
	}
}

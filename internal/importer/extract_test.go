package importer_test

import (
	"context"
	"testing"

	"github.com/TruyLabs/rias/internal/importer"
	"github.com/TruyLabs/rias/internal/prompt"
	"github.com/TruyLabs/rias/internal/provider"
)

func TestBuildExtractionPromptContainsMessages(t *testing.T) {
	pb := prompt.NewBuilder("rias", "User")
	conv := importer.Conversation{
		ID:    "c1",
		Title: "Test",
		Messages: []importer.Message{
			{Role: "user", Content: "I prefer TDD"},
			{Role: "assistant", Content: "Got it, noted."},
		},
	}

	p := importer.BuildExtractionPrompt(pb, conv)
	if p == "" {
		t.Error("expected non-empty prompt")
	}
	if len(p) < 20 {
		t.Errorf("prompt suspiciously short: %q", p)
	}
}

// stubProvider is a minimal provider.Provider for testing.
type stubProvider struct {
	response string
}

func (s *stubProvider) Chat(_ context.Context, _ string, _ []provider.Message, _ ...provider.Option) (*provider.Response, error) {
	return &provider.Response{Content: s.response}, nil
}

func (s *stubProvider) Stream(_ context.Context, _ string, _ []provider.Message, _ ...provider.Option) (<-chan provider.Chunk, error) {
	ch := make(chan provider.Chunk, 1)
	ch <- provider.Chunk{Content: s.response, Done: true}
	close(ch)
	return ch, nil
}

func TestExtractLearningsReturnsParsedResults(t *testing.T) {
	pb := prompt.NewBuilder("rias", "User")
	stub := &stubProvider{response: `[{"category":"opinions","topic":"tdd-preference","tags":["tdd"],"content":"Prefers TDD","confidence":"high","action":"create"}]`}

	conv := importer.Conversation{
		ID: "c1",
		Messages: []importer.Message{
			{Role: "user", Content: "I prefer TDD for all code"},
		},
	}

	learnings, err := importer.ExtractLearnings(context.Background(), conv, stub, pb)
	if err != nil {
		t.Fatalf("ExtractLearnings: %v", err)
	}
	if len(learnings) != 1 {
		t.Fatalf("expected 1 learning, got %d", len(learnings))
	}
	if learnings[0].Category != "opinions" {
		t.Errorf("expected category=opinions, got %q", learnings[0].Category)
	}
}

func TestExtractLearningsEmptyConversation(t *testing.T) {
	pb := prompt.NewBuilder("rias", "User")
	stub := &stubProvider{response: `[]`}
	conv := importer.Conversation{ID: "c1"}

	learnings, err := importer.ExtractLearnings(context.Background(), conv, stub, pb)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(learnings) != 0 {
		t.Errorf("expected 0 learnings, got %d", len(learnings))
	}
}

package router

import (
	"context"
	"testing"

	"github.com/tinhvqbk/kai/internal/brain"
	"github.com/tinhvqbk/kai/internal/prompt"
	"github.com/tinhvqbk/kai/internal/provider"
	"github.com/tinhvqbk/kai/internal/retriever"
	"github.com/tinhvqbk/kai/internal/session"
)

type mockProvider struct{}

func (m *mockProvider) Chat(ctx context.Context, systemPrompt string, messages []provider.Message, opts ...provider.Option) (*provider.Response, error) {
	// Check if this is a learning extraction call
	for _, msg := range messages {
		if msg.Role == "user" && len(msg.Content) > 0 && msg.Content[0] == 'G' {
			return &provider.Response{Content: "[]"}, nil
		}
	}
	return &provider.Response{Content: "I think Go is great for CLI tools."}, nil
}

func (m *mockProvider) Stream(ctx context.Context, systemPrompt string, messages []provider.Message, opts ...provider.Option) (<-chan provider.Chunk, error) {
	ch := make(chan provider.Chunk, 1)
	ch <- provider.Chunk{Content: "I think Go is great.", Done: true}
	close(ch)
	return ch, nil
}

func TestRouterChat(t *testing.T) {
	dir := t.TempDir()
	b := brain.New(dir)

	// Create a minimal brain
	b.Save(&brain.BrainFile{
		Path:       "identity/profile.md",
		Tags:       []string{"identity"},
		Confidence: "high",
		Source:     "direct",
		Content:    "\nKyle is a software engineer.\n",
	})
	b.RebuildIndex()

	sessDir := t.TempDir()
	sessMgr := session.NewManager(sessDir)

	r := New(
		b,
		retriever.New(b, 10),
		prompt.NewBuilder(),
		&mockProvider{},
		sessMgr,
	)

	sess := sessMgr.New("mock")
	result, err := r.Chat(context.Background(), sess, "what do I think about Go?")
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}

	if result.Response == "" {
		t.Error("expected non-empty response")
	}
	if len(sess.Messages) != 2 { // user + assistant
		t.Errorf("session has %d messages, want 2", len(sess.Messages))
	}
}

package router

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tinhvqbk/kai/internal/brain"
	"github.com/tinhvqbk/kai/internal/prompt"
	"github.com/tinhvqbk/kai/internal/provider"
	"github.com/tinhvqbk/kai/internal/retriever"
	"github.com/tinhvqbk/kai/internal/session"
)

type learningMockProvider struct{}

func (m *learningMockProvider) Chat(ctx context.Context, systemPrompt string, messages []provider.Message, opts ...provider.Option) (*provider.Response, error) {
	// Check if this is a learning extraction call
	lastMsg := messages[len(messages)-1]
	if len(lastMsg.Content) > 50 && lastMsg.Content[:5] == "Given" {
		learnings := []brain.Learning{
			{
				Category:   "opinions",
				Topic:      "testing",
				Tags:       []string{"testing", "tdd"},
				Content:    "Kyle is a big fan of TDD.",
				Confidence: "high",
				Action:     "create",
			},
		}
		data, _ := json.Marshal(learnings)
		return &provider.Response{Content: string(data)}, nil
	}

	return &provider.Response{Content: "Based on what I know about you, you prefer TDD for business logic."}, nil
}

func (m *learningMockProvider) Stream(ctx context.Context, systemPrompt string, messages []provider.Message, opts ...provider.Option) (<-chan provider.Chunk, error) {
	ch := make(chan provider.Chunk, 1)
	ch <- provider.Chunk{Content: "mock", Done: true}
	close(ch)
	return ch, nil
}

func TestFullPipeline(t *testing.T) {
	// Setup brain with some knowledge
	brainDir := t.TempDir()
	b := brain.New(brainDir)

	os.MkdirAll(filepath.Join(brainDir, "identity"), 0755)
	b.Save(&brain.BrainFile{
		Path:       "identity/profile.md",
		Tags:       []string{"identity"},
		Confidence: "high",
		Source:     "direct",
		Updated:    brain.DateOnly{Time: time.Now()},
		Content:    "\nKyle is a software engineer who loves Go.\n",
	})
	b.RebuildIndex()

	sessDir := t.TempDir()
	sessMgr := session.NewManager(sessDir)
	ret := retriever.New(b, 10)

	r := New(b, ret, prompt.NewBuilder("kai", "TestUser"), &learningMockProvider{}, sessMgr)

	// Run a chat
	sess := sessMgr.New("mock")
	result, err := r.Chat(context.Background(), sess, "What do I think about testing?")
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}

	// Verify response
	if result.Response == "" {
		t.Error("expected non-empty response")
	}

	// Verify session has messages
	if len(sess.Messages) != 2 {
		t.Errorf("session has %d messages, want 2", len(sess.Messages))
	}

	// Verify brain files were used
	if len(result.BrainFilesUsed) == 0 {
		t.Error("expected brain files to be used")
	}

	// Verify learning was extracted and saved
	_, err = b.Load("opinions/testing.md")
	if err != nil {
		t.Errorf("expected opinions/testing.md to be created by learning extraction, got: %v", err)
	}
}

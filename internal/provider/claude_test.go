package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClaudeChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			t.Error("missing or wrong API key header")
		}
		if r.Header.Get("anthropic-version") == "" {
			t.Error("missing anthropic-version header")
		}

		resp := map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": "Hello from Claude"},
			},
			"role":        "assistant",
			"stop_reason": "end_turn",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClaude("test-key", "claude-sonnet-4-6-20250514", server.URL, 0)
	resp, err := c.Chat(context.Background(), "You are a helpful assistant.", []Message{
		{Role: "user", Content: "hello"},
	})
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if resp.Content != "Hello from Claude" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello from Claude")
	}
}

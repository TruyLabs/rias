package provider

import (
	"context"
	"testing"
)

type mockProvider struct {
	name string
}

func (m *mockProvider) Chat(ctx context.Context, systemPrompt string, messages []Message, opts ...Option) (*Response, error) {
	return &Response{Content: "mock response from " + m.name}, nil
}

func (m *mockProvider) Stream(ctx context.Context, systemPrompt string, messages []Message, opts ...Option) (<-chan Chunk, error) {
	ch := make(chan Chunk, 1)
	ch <- Chunk{Content: "mock stream"}
	close(ch)
	return ch, nil
}

func TestRegistryRegisterAndGet(t *testing.T) {
	r := NewRegistry()
	mock := &mockProvider{name: "test"}

	r.Register("test", mock)

	p, err := r.Get("test")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	resp, err := p.Chat(context.Background(), "", nil)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if resp.Content != "mock response from test" {
		t.Errorf("Content = %q, want %q", resp.Content, "mock response from test")
	}
}

func TestRegistryGetUnknown(t *testing.T) {
	r := NewRegistry()
	_, err := r.Get("nonexistent")
	if err == nil {
		t.Error("Get() should error for unknown provider")
	}
}

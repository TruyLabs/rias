package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultClaudeURL   = "https://api.anthropic.com"
	anthropicVersion   = "2023-06-01"
	claudeMessagesPath = "/v1/messages"
)

// Claude implements the Provider interface for Anthropic's API.
type Claude struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewClaude creates a Claude provider.
// Pass empty baseURL to use the default. Pass 0 timeout for the default (120s).
func NewClaude(apiKey, model, baseURL string, timeout time.Duration) *Claude {
	if baseURL == "" {
		baseURL = defaultClaudeURL
	}
	if timeout == 0 {
		timeout = time.Duration(defaultProviderTimeoutSec) * time.Second
	}
	return &Claude{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{Timeout: timeout},
	}
}

const defaultProviderTimeoutSec = 120

type claudeRequest struct {
	Model     string          `json:"model"`
	System    string          `json:"system,omitempty"`
	Messages  []claudeMessage `json:"messages"`
	MaxTokens int             `json:"max_tokens"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
}

func (c *Claude) Chat(ctx context.Context, systemPrompt string, messages []Message, opts ...Option) (*Response, error) {
	co := ApplyOptions(opts)
	model := c.model
	if co.Model != "" {
		model = co.Model
	}

	cmsgs := make([]claudeMessage, len(messages))
	for i, m := range messages {
		cmsgs[i] = claudeMessage{Role: m.Role, Content: m.Content}
	}

	reqBody := claudeRequest{
		Model:     model,
		System:    systemPrompt,
		Messages:  cmsgs,
		MaxTokens: co.MaxTokens,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+claudeMessagesPath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("claude API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var cr claudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	var content string
	for _, block := range cr.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return &Response{Content: content}, nil
}

func (c *Claude) Stream(ctx context.Context, systemPrompt string, messages []Message, opts ...Option) (<-chan Chunk, error) {
	resp, err := c.Chat(ctx, systemPrompt, messages, opts...)
	if err != nil {
		return nil, err
	}
	ch := make(chan Chunk, 1)
	ch <- Chunk{Content: resp.Content, Done: true}
	close(ch)
	return ch, nil
}

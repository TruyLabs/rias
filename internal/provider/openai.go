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
	defaultOpenAIURL      = "https://api.openai.com"
	openaiCompletionsPath = "/v1/chat/completions"
)

// OpenAI implements the Provider interface for OpenAI's API.
type OpenAI struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewOpenAI creates an OpenAI provider.
// Pass empty baseURL to use the default. Pass 0 timeout for the default (120s).
func NewOpenAI(apiKey, model, baseURL string, timeout time.Duration) *OpenAI {
	if baseURL == "" {
		baseURL = defaultOpenAIURL
	}
	if timeout == 0 {
		timeout = time.Duration(defaultProviderTimeoutSec) * time.Second
	}
	return &OpenAI{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{Timeout: timeout},
	}
}

type openaiRequest struct {
	Model    string          `json:"model"`
	Messages []openaiMessage `json:"messages"`
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

func (o *OpenAI) Chat(ctx context.Context, systemPrompt string, messages []Message, opts ...Option) (*Response, error) {
	co := ApplyOptions(opts)
	model := o.model
	if co.Model != "" {
		model = co.Model
	}

	omsgs := make([]openaiMessage, 0, len(messages)+1)
	if systemPrompt != "" {
		omsgs = append(omsgs, openaiMessage{Role: "system", Content: systemPrompt})
	}
	for _, m := range messages {
		omsgs = append(omsgs, openaiMessage{Role: m.Role, Content: m.Content})
	}

	reqBody := openaiRequest{
		Model:    model,
		Messages: omsgs,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+openaiCompletionsPath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var or openaiResponse
	if err := json.NewDecoder(resp.Body).Decode(&or); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(or.Choices) == 0 {
		return nil, fmt.Errorf("openai returned no choices")
	}

	return &Response{Content: or.Choices[0].Message.Content}, nil
}

func (o *OpenAI) Stream(ctx context.Context, systemPrompt string, messages []Message, opts ...Option) (<-chan Chunk, error) {
	resp, err := o.Chat(ctx, systemPrompt, messages, opts...)
	if err != nil {
		return nil, err
	}
	ch := make(chan Chunk, 1)
	ch <- Chunk{Content: resp.Content, Done: true}
	close(ch)
	return ch, nil
}

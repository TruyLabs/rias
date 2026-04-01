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
	defaultGeminiURL   = "https://generativelanguage.googleapis.com"
	defaultGeminiModel = "gemini-2.0-flash"
)

// Gemini implements the Provider interface for Google's Gemini API.
type Gemini struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewGemini creates a Gemini provider.
// Pass empty baseURL to use the default. Pass 0 timeout for the default (120s).
func NewGemini(apiKey, model, baseURL string, timeout time.Duration) *Gemini {
	if baseURL == "" {
		baseURL = defaultGeminiURL
	}
	if model == "" {
		model = defaultGeminiModel
	}
	if timeout == 0 {
		timeout = time.Duration(defaultProviderTimeoutSec) * time.Second
	}
	return &Gemini{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{Timeout: timeout},
	}
}

// Gemini API request/response types.

type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	SystemInstruction *geminiContent        `json:"systemInstruction,omitempty"`
	GenerationConfig *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	Temperature     float64 `json:"temperature,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
			Role string `json:"role"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

func (g *Gemini) Chat(ctx context.Context, systemPrompt string, messages []Message, opts ...Option) (*Response, error) {
	co := ApplyOptions(opts)
	model := g.model
	if co.Model != "" {
		model = co.Model
	}

	req := geminiRequest{
		GenerationConfig: &geminiGenerationConfig{
			MaxOutputTokens: co.MaxTokens,
			Temperature:     co.Temperature,
		},
	}

	// System prompt goes into systemInstruction.
	if systemPrompt != "" {
		req.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: systemPrompt}},
		}
	}

	// Convert messages to Gemini format.
	for _, m := range messages {
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		req.Contents = append(req.Contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: m.Content}},
		})
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// POST https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent?key={apiKey}
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", g.baseURL, model, g.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var gr geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if gr.Error != nil {
		return nil, fmt.Errorf("gemini error: %s", gr.Error.Message)
	}

	if len(gr.Candidates) == 0 || len(gr.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini returned no content")
	}

	// Concatenate all text parts.
	var text string
	for _, part := range gr.Candidates[0].Content.Parts {
		text += part.Text
	}

	return &Response{Content: text}, nil
}

func (g *Gemini) Stream(ctx context.Context, systemPrompt string, messages []Message, opts ...Option) (<-chan Chunk, error) {
	resp, err := g.Chat(ctx, systemPrompt, messages, opts...)
	if err != nil {
		return nil, err
	}
	ch := make(chan Chunk, 1)
	ch <- Chunk{Content: resp.Content, Done: true}
	close(ch)
	return ch, nil
}

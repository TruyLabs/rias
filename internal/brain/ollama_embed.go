package brain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Default Ollama embedding settings.
const (
	DefaultOllamaURL   = "http://localhost:11434"
	DefaultEmbedModel  = "nomic-embed-text"
	OllamaTimeout      = 30 * time.Second
)

// OllamaEmbedder generates embeddings via a local Ollama instance.
type OllamaEmbedder struct {
	baseURL string
	model   string
	client  *http.Client
	dims    int // cached after first call
}

// OllamaEmbedConfig holds configuration for the Ollama embedder.
type OllamaEmbedConfig struct {
	URL   string // Ollama API base URL (default: http://localhost:11434)
	Model string // Embedding model name (default: nomic-embed-text)
}

// NewOllamaEmbedder creates an Ollama-based embedder.
func NewOllamaEmbedder(cfg OllamaEmbedConfig) *OllamaEmbedder {
	if cfg.URL == "" {
		cfg.URL = DefaultOllamaURL
	}
	if cfg.Model == "" {
		cfg.Model = DefaultEmbedModel
	}
	return &OllamaEmbedder{
		baseURL: cfg.URL,
		model:   cfg.Model,
		client:  &http.Client{Timeout: OllamaTimeout},
	}
}

// ollamaEmbedRequest is the request body for /api/embed.
type ollamaEmbedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

// ollamaEmbedResponse is the response from /api/embed.
type ollamaEmbedResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
}

// Embed generates an embedding for the given text.
func (e *OllamaEmbedder) Embed(text string) ([]float32, error) {
	reqBody := ollamaEmbedRequest{
		Model: e.model,
		Input: text,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal embed request: %w", err)
	}

	resp, err := e.client.Post(e.baseURL+"/api/embed", "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("ollama embed request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama embed: status %d: %s", resp.StatusCode, string(body))
	}

	var embedResp ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("decode embed response: %w", err)
	}

	if len(embedResp.Embeddings) == 0 {
		return nil, fmt.Errorf("ollama returned no embeddings")
	}

	vec64 := embedResp.Embeddings[0]
	vec := make([]float32, len(vec64))
	for i, v := range vec64 {
		vec[i] = float32(v)
	}

	e.dims = len(vec)
	return vec, nil
}

// Dims returns the embedding dimensionality (known after first Embed call).
func (e *OllamaEmbedder) Dims() int {
	return e.dims
}

// Available checks if the Ollama service is reachable and the model exists.
func (e *OllamaEmbedder) Available() bool {
	// Quick test: embed a single word.
	_, err := e.Embed("test")
	return err == nil
}

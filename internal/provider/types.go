package provider

import "context"

// Default call options.
const (
	DefaultMaxTokens   = 4096
	DefaultTemperature = 0.7
)

// Provider is the interface for LLM providers.
type Provider interface {
	Chat(ctx context.Context, systemPrompt string, messages []Message, opts ...Option) (*Response, error)
	Stream(ctx context.Context, systemPrompt string, messages []Message, opts ...Option) (<-chan Chunk, error)
}

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Response is the LLM response.
type Response struct {
	Content string
}

// Chunk is a streaming response fragment.
type Chunk struct {
	Content string
	Done    bool
}

// Option configures a provider call.
type Option func(*CallOptions)

// CallOptions holds per-call configuration.
type CallOptions struct {
	MaxTokens   int
	Temperature float64
	Model       string
}

// WithMaxTokens sets the max tokens for a call.
func WithMaxTokens(n int) Option {
	return func(o *CallOptions) { o.MaxTokens = n }
}

// WithTemperature sets the temperature for a call.
func WithTemperature(t float64) Option {
	return func(o *CallOptions) { o.Temperature = t }
}

// WithModel overrides the model for a call.
func WithModel(m string) Option {
	return func(o *CallOptions) { o.Model = m }
}

// ApplyOptions builds CallOptions from variadic options.
func ApplyOptions(opts []Option) CallOptions {
	co := CallOptions{MaxTokens: DefaultMaxTokens, Temperature: DefaultTemperature}
	for _, o := range opts {
		o(&co)
	}
	return co
}

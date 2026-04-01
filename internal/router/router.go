package router

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/tinhvqbk/kai/internal/brain"
	"github.com/tinhvqbk/kai/internal/prompt"
	"github.com/tinhvqbk/kai/internal/provider"
	"github.com/tinhvqbk/kai/internal/retriever"
	"github.com/tinhvqbk/kai/internal/session"
)

// Confidence thresholds based on number of matching brain files.
const (
	HighConfidenceThreshold   = 3
	MediumConfidenceThreshold = 1
	DefaultRetrievalLimit     = 10
	MinMessagesForLearning    = 2
)

// Confidence level labels.
const (
	ConfidenceHigh   = "high"
	ConfidenceMedium = "medium"
	ConfidenceLow    = "low"
)

// ChatResult holds the result of a chat interaction.
type ChatResult struct {
	Response       string
	BrainFilesUsed []string
	Confidence     string
}

// Router orchestrates the conversation pipeline.
type Router struct {
	brain     *brain.FileBrain
	retriever *retriever.FileRetriever
	builder   *prompt.Builder
	provider  provider.Provider
	sessions  *session.Manager
}

// New creates a Router.
func New(
	b *brain.FileBrain,
	r *retriever.FileRetriever,
	pb *prompt.Builder,
	p provider.Provider,
	sm *session.Manager,
) *Router {
	return &Router{
		brain:     b,
		retriever: r,
		builder:   pb,
		provider:  p,
		sessions:  sm,
	}
}

// Chat processes a user message through the full pipeline.
func (r *Router) Chat(ctx context.Context, sess *session.Session, userInput string) (*ChatResult, error) {
	// 1. Retrieve relevant brain context
	brainFiles, err := r.retriever.Retrieve(ctx, userInput, DefaultRetrievalLimit)
	if err != nil {
		return nil, fmt.Errorf("retrieve: %w", err)
	}

	var brainPaths []string
	for _, bf := range brainFiles {
		brainPaths = append(brainPaths, bf.Path)
	}

	// 2. Build system prompt
	systemPrompt := r.builder.BuildSystemPrompt(brainFiles)

	// 3. Build messages with history
	var history []provider.Message
	for _, m := range sess.Messages {
		history = append(history, m.Message)
	}
	messages := r.builder.BuildMessages(history, userInput)

	// 4. Call LLM
	resp, err := r.provider.Chat(ctx, systemPrompt, messages)
	if err != nil {
		return nil, fmt.Errorf("llm chat: %w", err)
	}

	// Determine confidence based on brain files count
	confidence := ConfidenceLow
	if len(brainFiles) > HighConfidenceThreshold {
		confidence = ConfidenceHigh
	} else if len(brainFiles) > MediumConfidenceThreshold {
		confidence = ConfidenceMedium
	}

	// 5. Add messages to session
	now := time.Now().UTC()
	sess.AddMessage(session.SessionMessage{
		Message:   provider.Message{Role: "user", Content: userInput},
		Timestamp: now,
	})
	sess.AddMessage(session.SessionMessage{
		Message:        provider.Message{Role: "assistant", Content: resp.Content},
		Timestamp:      now,
		BrainFilesUsed: brainPaths,
		Confidence:     confidence,
	})

	// 6. Extract learnings (non-fatal on error)
	r.extractLearnings(ctx, brainPaths, sess)

	return &ChatResult{
		Response:       resp.Content,
		BrainFilesUsed: brainPaths,
		Confidence:     confidence,
	}, nil
}

func (r *Router) extractLearnings(ctx context.Context, brainPaths []string, sess *session.Session) {
	if len(sess.Messages) < MinMessagesForLearning {
		return
	}

	// Get last exchange
	lastTwo := sess.Messages[len(sess.Messages)-2:]
	exchange := []provider.Message{
		lastTwo[0].Message,
		lastTwo[1].Message,
	}

	learningPrompt := r.builder.BuildLearningPrompt(brainPaths, exchange)
	resp, err := r.provider.Chat(ctx, "", []provider.Message{
		{Role: "user", Content: learningPrompt},
	})
	if err != nil {
		log.Printf("learning extraction failed: %v", err)
		return
	}

	learnings, err := brain.ParseLearnings(resp.Content)
	if err != nil {
		log.Printf("parse learnings failed: %v", err)
		return
	}

	if len(learnings) == 0 {
		return
	}

	if err := r.brain.Learn(learnings); err != nil {
		log.Printf("brain learn failed: %v", err)
		return
	}

	if err := r.brain.RebuildIndex(); err != nil {
		log.Printf("rebuild index failed: %v", err)
	}

	sess.LearningsExtracted += len(learnings)
}

package importer

import (
	"context"
	"fmt"

	"github.com/TruyLabs/rias/internal/brain"
	"github.com/TruyLabs/rias/internal/prompt"
	"github.com/TruyLabs/rias/internal/provider"
)

// maxMessagesPerExtraction limits how many messages are sent to the LLM per conversation.
// Very long conversations are truncated to the last N messages to fit context windows.
const maxMessagesPerExtraction = 40

// BuildExtractionPrompt builds the LLM prompt for extracting learnings from a conversation.
func BuildExtractionPrompt(pb *prompt.Builder, conv Conversation) string {
	msgs := conversationToProviderMessages(conv)
	return pb.BuildLearningPrompt(nil, msgs)
}

// ExtractLearnings sends a conversation to the LLM and returns parsed learnings.
// Returns an empty slice (not an error) when the LLM finds nothing new.
func ExtractLearnings(ctx context.Context, conv Conversation, p provider.Provider, pb *prompt.Builder) ([]brain.Learning, error) {
	if len(conv.Messages) == 0 {
		return nil, nil
	}

	extractPrompt := BuildExtractionPrompt(pb, conv)
	resp, err := p.Chat(ctx, "", []provider.Message{
		{Role: "user", Content: extractPrompt},
	})
	if err != nil {
		return nil, fmt.Errorf("LLM extraction for %q: %w", conv.ID, err)
	}

	return brain.ParseLearnings(resp.Content)
}

// conversationToProviderMessages converts importer Messages to provider Messages,
// truncating to maxMessagesPerExtraction if necessary (last N messages kept).
func conversationToProviderMessages(conv Conversation) []provider.Message {
	msgs := conv.Messages
	if len(msgs) > maxMessagesPerExtraction {
		msgs = msgs[len(msgs)-maxMessagesPerExtraction:]
	}
	out := make([]provider.Message, len(msgs))
	for i, m := range msgs {
		out[i] = provider.Message{Role: m.Role, Content: m.Content}
	}
	return out
}

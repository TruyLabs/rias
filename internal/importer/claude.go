package importer

import (
	"encoding/json"
	"fmt"
)

type claudeExport []claudeConversation

type claudeConversation struct {
	UUID         string         `json:"uuid"`
	Name         string         `json:"name"`
	ChatMessages []claudeMessage `json:"chat_messages"`
}

type claudeMessage struct {
	UUID   string `json:"uuid"`
	Sender string `json:"sender"` // "human" | "assistant"
	Text   string `json:"text"`
}

// ParseClaude parses the JSON bytes from a claude.ai data export.
// Returns one Conversation per entry, preserving message order.
func ParseClaude(data []byte) ([]Conversation, error) {
	var raw claudeExport
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse claude export: %w", err)
	}

	convs := make([]Conversation, 0, len(raw))
	for _, rc := range raw {
		c := Conversation{
			ID:    rc.UUID,
			Title: rc.Name,
		}
		for _, m := range rc.ChatMessages {
			role := "assistant"
			if m.Sender == "human" {
				role = "user"
			}
			if m.Text == "" {
				continue
			}
			c.Messages = append(c.Messages, Message{Role: role, Content: m.Text})
		}
		convs = append(convs, c)
	}
	return convs, nil
}

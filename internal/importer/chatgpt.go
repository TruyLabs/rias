package importer

import (
	"encoding/json"
	"fmt"
	"strings"
)

type gptExport []gptConversation

type gptConversation struct {
	ID      string             `json:"id"`
	Title   string             `json:"title"`
	Mapping map[string]gptNode `json:"mapping"`
}

type gptNode struct {
	ID       string   `json:"id"`
	Message  *gptMsg  `json:"message"`
	Parent   *string  `json:"parent"`
	Children []string `json:"children"`
}

type gptMsg struct {
	Author  gptAuthor  `json:"author"`
	Content gptContent `json:"content"`
}

type gptAuthor struct {
	Role string `json:"role"`
}

type gptContent struct {
	ContentType string            `json:"content_type"`
	Parts       []json.RawMessage `json:"parts"`
}

// stringParts returns only the string entries from parts, skipping object parts
// (image asset pointers, audio, etc.) that appear in multimodal_text messages.
func (c gptContent) stringParts() []string {
	out := make([]string, 0, len(c.Parts))
	for _, p := range c.Parts {
		var s string
		if err := json.Unmarshal(p, &s); err == nil {
			out = append(out, s)
		}
	}
	return out
}

// ParseChatGPT parses the JSON bytes from a ChatGPT data export (conversations.json).
// It reconstructs the linear conversation by walking the tree from root to leaf.
func ParseChatGPT(data []byte) ([]Conversation, error) {
	var raw gptExport
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse chatgpt export: %w", err)
	}

	convs := make([]Conversation, 0, len(raw))
	for _, gc := range raw {
		msgs := walkGPTTree(gc.Mapping)
		convs = append(convs, Conversation{
			ID:       gc.ID,
			Title:    gc.Title,
			Messages: msgs,
		})
	}
	return convs, nil
}

// walkGPTTree reconstructs the ordered message list from the mapping tree.
// Starts at the root node (no parent) and follows the first child at each step.
func walkGPTTree(mapping map[string]gptNode) []Message {
	// Find root: node whose parent is empty or not in the mapping.
	rootID := ""
	for id, node := range mapping {
		if node.Parent == nil {
			rootID = id
			break
		}
		if _, ok := mapping[*node.Parent]; !ok {
			rootID = id
			break
		}
	}
	if rootID == "" {
		return nil
	}

	var msgs []Message
	visited := make(map[string]bool)
	current := rootID

	for current != "" && !visited[current] {
		visited[current] = true
		node := mapping[current]

		if node.Message != nil {
			role := node.Message.Author.Role
			if role == "user" || role == "assistant" {
				content := strings.Join(node.Message.Content.stringParts(), "")
				content = strings.TrimSpace(content)
				if content != "" {
					msgs = append(msgs, Message{Role: role, Content: content})
				}
			}
		}

		if len(node.Children) > 0 {
			current = node.Children[0]
		} else {
			current = ""
		}
	}
	return msgs
}

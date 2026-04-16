package importer_test

import (
	"testing"

	"github.com/TruyLabs/rias/internal/importer"
)

// Minimal ChatGPT conversations.json with 2-turn conversation.
// The mapping is a tree: root → node-a (user) → node-b (assistant)
const chatgptExportJSON = `[
  {
    "id": "gpt-conv-1",
    "title": "Go programming",
    "mapping": {
      "root": {
        "id": "root",
        "message": null,
        "parent": null,
        "children": ["node-a"]
      },
      "node-a": {
        "id": "node-a",
        "message": {
          "id": "msg-a",
          "author": {"role": "user"},
          "content": {"content_type": "text", "parts": ["Tell me about Go"]},
          "create_time": 1700000000.0
        },
        "parent": "root",
        "children": ["node-b"]
      },
      "node-b": {
        "id": "node-b",
        "message": {
          "id": "msg-b",
          "author": {"role": "assistant"},
          "content": {"content_type": "text", "parts": ["Go is great!"]},
          "create_time": 1700000001.0
        },
        "parent": "node-a",
        "children": []
      }
    }
  }
]`

func TestParseChatGPT(t *testing.T) {
	convs, err := importer.ParseChatGPT([]byte(chatgptExportJSON))
	if err != nil {
		t.Fatalf("ParseChatGPT: %v", err)
	}
	if len(convs) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(convs))
	}

	c := convs[0]
	if c.ID != "gpt-conv-1" {
		t.Errorf("expected ID=gpt-conv-1, got %q", c.ID)
	}
	if c.Title != "Go programming" {
		t.Errorf("expected Title=Go programming, got %q", c.Title)
	}
	if len(c.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(c.Messages))
	}
	if c.Messages[0].Role != "user" || c.Messages[0].Content != "Tell me about Go" {
		t.Errorf("unexpected first message: %+v", c.Messages[0])
	}
	if c.Messages[1].Role != "assistant" || c.Messages[1].Content != "Go is great!" {
		t.Errorf("unexpected second message: %+v", c.Messages[1])
	}
}

func TestParseChatGPTInvalidJSON(t *testing.T) {
	_, err := importer.ParseChatGPT([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseChatGPTSkipsSystemMessages(t *testing.T) {
	const withSystem = `[{
		"id": "c1", "title": "t",
		"mapping": {
			"root": {"id":"root","message":null,"parent":null,"children":["sys","usr"]},
			"sys": {"id":"sys","message":{"id":"s","author":{"role":"system"},"content":{"content_type":"text","parts":["You are..."]},"create_time":1.0},"parent":"root","children":["usr"]},
			"usr": {"id":"usr","message":{"id":"u","author":{"role":"user"},"content":{"content_type":"text","parts":["Hello"]},"create_time":2.0},"parent":"sys","children":[]}
		}
	}]`
	convs, err := importer.ParseChatGPT([]byte(withSystem))
	if err != nil {
		t.Fatalf("ParseChatGPT: %v", err)
	}
	if len(convs[0].Messages) != 1 {
		t.Errorf("expected 1 message (system filtered), got %d", len(convs[0].Messages))
	}
	if convs[0].Messages[0].Role != "user" {
		t.Errorf("expected user message, got %q", convs[0].Messages[0].Role)
	}
}

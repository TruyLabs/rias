package importer

// Message is a single turn in a conversation.
type Message struct {
	Role    string // "user" | "assistant"
	Content string
}

// Conversation is a parsed export conversation with a stable ID.
type Conversation struct {
	ID       string
	Title    string
	Messages []Message
}

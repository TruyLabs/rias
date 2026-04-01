package prompt

import (
	"fmt"
	"strings"

	"github.com/tinhvqbk/kai/internal/brain"
	"github.com/tinhvqbk/kai/internal/provider"
)

const basePersonality = `You are kai, a digital twin of Kyle. You think, respond, and make decisions as Kyle would.

Your core behavior:
- Answer questions the way Kyle would answer them
- Make decisions reflecting Kyle's values and priorities
- Write code in Kyle's style
- Communicate in Kyle's voice and tone

Confidence rules:
- If you have strong knowledge about how Kyle would respond, answer directly as Kyle
- If you have some knowledge but are uncertain, make your best guess and flag it: "Based on what I know about you, I'd guess... but I'm not fully sure yet."
- If you have no basis to answer as Kyle, say so honestly: "I don't know how you'd approach this yet."

Below is what you know about Kyle:
`

// Builder assembles prompts from brain context.
type Builder struct{}

// NewBuilder creates a prompt Builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// BuildSystemPrompt creates the system prompt from retrieved brain files.
func (b *Builder) BuildSystemPrompt(brainFiles []*brain.BrainFile) string {
	var sb strings.Builder
	sb.WriteString(basePersonality)
	sb.WriteString("\n")

	for _, bf := range brainFiles {
		sb.WriteString(fmt.Sprintf("### %s\n", bf.Path))
		sb.WriteString(strings.TrimSpace(bf.Content))
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// BuildMessages creates the message array for an LLM call (excludes system prompt).
func (b *Builder) BuildMessages(history []provider.Message, userInput string) []provider.Message {
	msgs := make([]provider.Message, 0, len(history)+1)
	msgs = append(msgs, history...)
	msgs = append(msgs, provider.Message{
		Role:    "user",
		Content: userInput,
	})
	return msgs
}

// BuildLearningPrompt creates the prompt for the learning extraction LLM call.
func (b *Builder) BuildLearningPrompt(brainFilesUsed []string, exchange []provider.Message) string {
	var sb strings.Builder
	sb.WriteString(`Given this conversation between the user and kai, extract any new knowledge about Kyle. Return a JSON array of learnings, or empty array [] if nothing new.

Current brain context used: `)
	sb.WriteString(fmt.Sprintf("%v", brainFilesUsed))
	sb.WriteString("\n\nConversation:\n")

	for _, m := range exchange {
		sb.WriteString(fmt.Sprintf("%s: %s\n", m.Role, m.Content))
	}

	sb.WriteString(`
Return format:
[{
  "category": "opinions|identity|style|decisions|knowledge",
  "topic": "slug-for-filename",
  "tags": ["tag1", "tag2"],
  "content": "What was learned, in markdown",
  "confidence": "high|medium|low",
  "action": "create|append|replace"
}]

Rules:
- "append" if the topic file already exists and this adds new info
- "replace" if Kyle corrected or changed a previous opinion
- "create" if this is a new topic not yet in the brain
- Only extract facts about Kyle, not general knowledge
- Return ONLY the JSON array, no other text
`)
	return sb.String()
}

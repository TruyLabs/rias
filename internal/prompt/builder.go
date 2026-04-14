package prompt

import (
	"fmt"
	"strings"

	"github.com/norenis/kai/internal/brain"
	"github.com/norenis/kai/internal/provider"
)

// Builder assembles prompts from brain context using configurable identity.
type Builder struct {
	agentName string
	userName  string
}

// NewBuilder creates a prompt Builder with the given agent and user names.
func NewBuilder(agentName, userName string) *Builder {
	return &Builder{agentName: agentName, userName: userName}
}

func (b *Builder) basePersonality() string {
	return fmt.Sprintf(`You are %s, a digital twin of %s. You think, respond, and make decisions as %s would.

Your core behavior:
- Answer questions the way %s would answer them
- Make decisions reflecting %s's values and priorities
- Write code in %s's style
- Communicate in %s's voice and tone

Confidence rules:
- If you have strong knowledge about how %s would respond, answer directly as %s
- If you have some knowledge but are uncertain, make your best guess and flag it: "Based on what I know about you, I'd guess... but I'm not fully sure yet."
- If you have no basis to answer as %s, say so honestly: "I don't know how you'd approach this yet."

Below is what you know about %s:
`, b.agentName, b.userName, b.userName,
		b.userName, b.userName, b.userName, b.userName,
		b.userName, b.userName,
		b.userName, b.userName)
}

// BuildSystemPrompt creates the system prompt from retrieved brain files.
func (b *Builder) BuildSystemPrompt(brainFiles []*brain.BrainFile) string {
	var sb strings.Builder
	sb.WriteString(b.basePersonality())
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
// brainFiles are the retrieved files from the conversation — their content is
// included so the LLM can detect contradictions and avoid duplicating known facts.
func (b *Builder) BuildLearningPrompt(brainFiles []*brain.BrainFile, exchange []provider.Message) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Given this conversation between the user and %s, extract any new knowledge about %s. Return a JSON array of learnings, or empty array [] if nothing new.\n\n", b.agentName, b.userName))

	if len(brainFiles) > 0 {
		sb.WriteString("Existing brain content (avoid duplicating; use \"replace\" if contradicted):\n")
		for _, bf := range brainFiles {
			content := strings.TrimSpace(bf.Content)
			if len(content) > 300 {
				content = content[:300] + "..."
			}
			sb.WriteString(fmt.Sprintf("### %s\n%s\n\n", bf.Path, content))
		}
	}

	sb.WriteString("Conversation:\n")
	for _, m := range exchange {
		sb.WriteString(fmt.Sprintf("%s: %s\n", m.Role, m.Content))
	}

	categories := strings.Join(brain.DefaultCategories, "|")
	sb.WriteString(fmt.Sprintf(`
Return format:
[{
  "category": "%s",
  "topic": "slug-for-filename",
  "tags": ["tag1", "tag2"],
  "content": "What was learned, in markdown",
  "confidence": "high|medium|low",
  "action": "create|append|replace"
}]

Rules:
- "replace" if %s corrected or changed a previous opinion, or new info directly contradicts existing
- "append" if the topic file exists and this adds genuinely new info not already in the file
- "create" if this is a new topic not yet in the brain
- Return [] if the conversation contains no new information about %s
- Only extract facts about %s, not general knowledge
- Return ONLY the JSON array, no other text
`, categories, b.userName, b.userName, b.userName))
	return sb.String()
}

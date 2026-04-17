package prompt

import (
	"fmt"
	"strings"

	"github.com/TruyLabs/rias/internal/brain"
	"github.com/TruyLabs/rias/internal/provider"
)

// Builder assembles prompts from brain context using configurable identity.
type Builder struct {
	agentName       string
	userName        string
	proactiveRecall bool
}

// NewBuilder creates a prompt Builder with the given agent and user names.
func NewBuilder(agentName, userName string) *Builder {
	return &Builder{agentName: agentName, userName: userName}
}

// SetProactiveRecall enables or disables the proactive recall directive in system prompts.
func (b *Builder) SetProactiveRecall(enabled bool) {
	b.proactiveRecall = enabled
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
// Style brain files (style/*.md) are injected first with a "mirror communication style"
// directive, before other knowledge/identity files.
func (b *Builder) BuildSystemPrompt(brainFiles []*brain.BrainFile) string {
	var sb strings.Builder
	sb.WriteString(b.basePersonality())
	if b.proactiveRecall {
		sb.WriteString(fmt.Sprintf("\nWhen you have information about %s that is directly relevant to their question — even if not explicitly asked — proactively mention it.\n", b.userName))
	}
	sb.WriteString("\n")

	var styleFiles, otherFiles []*brain.BrainFile
	for _, bf := range brainFiles {
		if strings.HasPrefix(bf.Path, "style/") {
			styleFiles = append(styleFiles, bf)
		} else {
			otherFiles = append(otherFiles, bf)
		}
	}

	if len(styleFiles) > 0 {
		sb.WriteString(fmt.Sprintf("## Mirror %s's communication style exactly:\n\n", b.userName))
		// Style files are injected as raw content (no "### path" heading) so
		// the LLM treats them as voice/tone rules rather than factual entries.
		for _, bf := range styleFiles {
			sb.WriteString(strings.TrimSpace(bf.Content))
			sb.WriteString("\n\n")
		}
	}

	for _, bf := range otherFiles {
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

// BuildContradictionPrompt creates a prompt to detect contradictions between brain files in a category.
// The LLM returns a JSON array of contradicting file pairs with descriptions and suggestions.
func (b *Builder) BuildContradictionPrompt(category string, files []*brain.BrainFile) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(
		"Analyze these brain files from the %q category for %s's personal knowledge base.\n"+
			"Identify any direct contradictions or conflicts between them — cases where one file's content "+
			"directly opposes or is incompatible with another file's content.\n\n",
		category, b.userName,
	))
	for _, bf := range files {
		content := strings.TrimSpace(bf.Content)
		if len(content) > 400 {
			content = content[:400] + "..."
		}
		sb.WriteString(fmt.Sprintf("### %s\n%s\n\n", bf.Path, content))
	}
	sb.WriteString(`Return a JSON array of contradictions:
[{
  "file_a": "category/file1.md",
  "file_b": "category/file2.md",
  "description": "File A says X but File B says Y",
  "suggestion": "How to resolve (e.g. keep the more recent view, or merge into a nuanced position)"
}]

Return [] if no contradictions are found.
Return ONLY the JSON array, no other text.`)
	return sb.String()
}

// BuildExpertisePrompt creates a prompt that synthesizes all brain files into a structured
// expertise map in markdown format. The date string is embedded in the output.
func (b *Builder) BuildExpertisePrompt(brainFiles []*brain.BrainFile, date string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(
		"Analyze these personal knowledge brain files for %s and create a concise expertise map.\n\n",
		b.userName,
	))
	for _, bf := range brainFiles {
		content := strings.TrimSpace(bf.Content)
		if len(content) > 300 {
			content = content[:300] + "..."
		}
		sb.WriteString(fmt.Sprintf("### %s\n%s\n\n", bf.Path, content))
	}
	sb.WriteString(fmt.Sprintf(`Return a markdown expertise map with this exact structure:

# Expertise Map

*Generated: %s*

## Technical Skills

| Skill/Domain | Level | Key Evidence |
|---|---|---|
| example | expert | brief evidence from brain |

## Domain Knowledge

| Area | Depth | Notes |
|---|---|---|
| example | deep | brief note |

## Notable Strengths

- Key strength 1
- Key strength 2

Rules:
- Only include skills directly evidenced in the brain files
- Levels: expert, proficient, familiar
- Keep each row to one concise line
- Return ONLY the markdown, no other text
`, date))
	return sb.String()
}

// BuildReflectPrompt creates a prompt for analyzing patterns across multiple sessions.
// It asks the LLM to extract communication style, vocabulary, and recurring interests,
// returning the same JSON Learning format as BuildLearningPrompt.
func (b *Builder) BuildReflectPrompt(brainFiles []*brain.BrainFile, sessionMessages []provider.Message) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(
		"Analyze these conversations between %s and an AI assistant to extract behavioral patterns. "+
			"Look for: how %s phrases questions, vocabulary they repeatedly use, topics they return to, "+
			"and values or preferences expressed implicitly. "+
			"Return a JSON array of learnings with categories 'style', 'opinions', or 'knowledge'. "+
			"Focus on patterns that appear multiple times, not one-off mentions. "+
			"Return [] if no clear patterns emerge.\n\n",
		b.userName, b.userName,
	))

	if len(brainFiles) > 0 {
		sb.WriteString("Existing brain entries (skip duplicates; only extract new patterns):\n")
		for _, bf := range brainFiles {
			content := strings.TrimSpace(bf.Content)
			if len(content) > 300 {
				content = content[:300] + "..."
			}
			sb.WriteString(fmt.Sprintf("### %s\n%s\n\n", bf.Path, content))
		}
	}

	sb.WriteString("Sessions to analyze:\n")
	for _, m := range sessionMessages {
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

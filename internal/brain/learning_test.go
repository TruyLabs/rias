package brain

import "testing"

func TestParseLearnings(t *testing.T) {
	raw := `[
		{
			"category": "opinions",
			"topic": "golang",
			"tags": ["go", "languages"],
			"content": "Kyle prefers Go for its simplicity.",
			"confidence": "high",
			"action": "append"
		},
		{
			"category": "style",
			"topic": "communication",
			"tags": ["communication", "writing"],
			"content": "Kyle is direct and concise.",
			"confidence": "medium",
			"action": "create"
		}
	]`

	learnings, err := ParseLearnings(raw)
	if err != nil {
		t.Fatalf("ParseLearnings() error: %v", err)
	}
	if len(learnings) != 2 {
		t.Fatalf("got %d learnings, want 2", len(learnings))
	}
	if learnings[0].Topic != "golang" {
		t.Errorf("Topic = %q, want %q", learnings[0].Topic, "golang")
	}
	if learnings[1].Action != "create" {
		t.Errorf("Action = %q, want %q", learnings[1].Action, "create")
	}
}

func TestParseLearnings_Empty(t *testing.T) {
	learnings, err := ParseLearnings("[]")
	if err != nil {
		t.Fatalf("ParseLearnings() error: %v", err)
	}
	if len(learnings) != 0 {
		t.Errorf("got %d learnings, want 0", len(learnings))
	}
}

func TestParseLearnings_ExtractJSON(t *testing.T) {
	// LLM might wrap JSON in markdown code block
	raw := "Here are the learnings:\n```json\n[{\"category\":\"opinions\",\"topic\":\"testing\",\"tags\":[\"testing\"],\"content\":\"Kyle likes TDD.\",\"confidence\":\"high\",\"action\":\"create\"}]\n```"

	learnings, err := ParseLearnings(raw)
	if err != nil {
		t.Fatalf("ParseLearnings() error: %v", err)
	}
	if len(learnings) != 1 {
		t.Errorf("got %d learnings, want 1", len(learnings))
	}
}

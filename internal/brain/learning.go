package brain

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseLearnings parses the LLM's JSON response into Learning structs.
// Handles cases where the LLM wraps JSON in markdown code blocks.
func ParseLearnings(raw string) ([]Learning, error) {
	cleaned := extractJSON(raw)

	var learnings []Learning
	if err := json.Unmarshal([]byte(cleaned), &learnings); err != nil {
		return nil, fmt.Errorf("parse learnings JSON: %w", err)
	}

	return learnings, nil
}

// extractJSON pulls JSON array from potentially messy LLM output.
// Uses bracket depth tracking to find the correct outermost array.
func extractJSON(raw string) string {
	s := strings.TrimSpace(raw)

	// Find the first '[' and track depth to find its matching ']'.
	start := strings.Index(s, "[")
	if start < 0 {
		return s
	}

	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' && inString {
			escaped = true
			continue
		}
		if c == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if c == '[' {
			depth++
		} else if c == ']' {
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}

	return s
}

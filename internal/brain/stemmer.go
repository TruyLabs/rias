package brain

import (
	"strings"
	"unicode"
)

// stem applies simple suffix-stripping to an English word.
// No external dependencies; handles common suffixes only.
func stem(word string) string {
	w := strings.ToLower(word)
	// Strip trailing punctuation
	w = strings.TrimRightFunc(w, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	if len(w) < 3 {
		return w
	}

	// Order matters: try longer suffixes first.
	suffixes := []struct {
		suffix string
		minLen int // minimum remaining length after stripping
	}{
		{"tion", 3},
		{"ment", 3},
		{"ness", 3},
		{"able", 3},
		{"ible", 3},
		{"ies", 3},  // companies -> compan
		{"ing", 3},  // running -> runn (good enough for matching)
		{"est", 3},
		{"ed", 3},
		{"er", 3},
		{"ly", 3},
		{"s", 3},
	}

	for _, sf := range suffixes {
		if strings.HasSuffix(w, sf.suffix) && len(w)-len(sf.suffix) >= sf.minLen {
			return w[:len(w)-len(sf.suffix)]
		}
	}
	return w
}

// tokenize splits text into lowercase, punctuation-stripped tokens.
func tokenize(text string) []string {
	words := strings.Fields(strings.ToLower(text))
	tokens := make([]string, 0, len(words))
	for _, w := range words {
		w = strings.TrimFunc(w, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		})
		if w != "" {
			tokens = append(tokens, w)
		}
	}
	return tokens
}

// tokenizeAndStem tokenizes text and applies stemming to each token.
func tokenizeAndStem(text string) []string {
	tokens := tokenize(text)
	stemmed := make([]string, len(tokens))
	for i, t := range tokens {
		stemmed[i] = stem(t)
	}
	return stemmed
}

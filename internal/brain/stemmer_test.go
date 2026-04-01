package brain

import "testing"

func TestStem(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"running", "runn"},
		{"creation", "crea"},
		{"slowly", "slow"},
		{"played", "play"},
		{"player", "play"},
		{"fastest", "fast"},
		{"happiness", "happi"},
		{"movement", "move"},
		{"readable", "read"},
		{"flexible", "flex"},
		{"companies", "compan"},
		{"tools", "tool"},
		{"go", "go"},           // too short to stem
		{"a", "a"},             // single char
		{"CLI", "cli"},         // lowercased
		{"testing!", "test"},   // punctuation stripped, then -ing
	}

	for _, tt := range tests {
		got := stem(tt.input)
		if got != tt.want {
			t.Errorf("stem(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTokenize(t *testing.T) {
	tokens := tokenize("Kyle prefers Go for CLI tools!")
	expected := []string{"kyle", "prefers", "go", "for", "cli", "tools"}
	if len(tokens) != len(expected) {
		t.Fatalf("tokenize length = %d, want %d", len(tokens), len(expected))
	}
	for i, tok := range tokens {
		if tok != expected[i] {
			t.Errorf("token[%d] = %q, want %q", i, tok, expected[i])
		}
	}
}

func TestTokenizeAndStem(t *testing.T) {
	tokens := tokenizeAndStem("running tools testing")
	expected := []string{"runn", "tool", "test"}
	if len(tokens) != len(expected) {
		t.Fatalf("tokenizeAndStem length = %d, want %d", len(tokens), len(expected))
	}
	for i, tok := range tokens {
		if tok != expected[i] {
			t.Errorf("token[%d] = %q, want %q", i, tok, expected[i])
		}
	}
}

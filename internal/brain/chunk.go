package brain

import (
	"strings"
)

// ChunkTargetWords is the target size for a single chunk in words.
const ChunkTargetWords = 200

// Chunk represents a segment of a brain file's content.
type Chunk struct {
	Text      string
	Offset    int // character offset into the original content
	Length    int // character length
	WordCount int
}

// chunkContent splits content into chunks of approximately targetWords.
// It prefers splitting on paragraph boundaries (double-newline), falling
// back to word boundaries for very long paragraphs. Files shorter than
// targetWords produce a single chunk.
func chunkContent(content string, targetWords int) []Chunk {
	if targetWords <= 0 {
		targetWords = ChunkTargetWords
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}

	paragraphs := splitParagraphs(content)

	var chunks []Chunk
	var curText strings.Builder
	curOffset := 0
	curWords := 0

	flush := func() {
		text := curText.String()
		if strings.TrimSpace(text) == "" {
			return
		}
		chunks = append(chunks, Chunk{
			Text:      text,
			Offset:    curOffset,
			Length:     len(text),
			WordCount: curWords,
		})
		curOffset += len(text)
		curText.Reset()
		curWords = 0
	}

	for _, para := range paragraphs {
		paraWords := len(strings.Fields(para.text))

		// If a single paragraph exceeds target, split it at word boundaries.
		if paraWords > targetWords {
			// Flush anything accumulated so far.
			flush()

			words := strings.Fields(para.text)
			var sub strings.Builder
			subWords := 0
			for _, w := range words {
				if subWords > 0 && subWords+1 > targetWords {
					chunks = append(chunks, Chunk{
						Text:      sub.String(),
						Offset:    curOffset,
						Length:     sub.Len(),
						WordCount: subWords,
					})
					curOffset += sub.Len()
					sub.Reset()
					subWords = 0
				}
				if sub.Len() > 0 {
					sub.WriteByte(' ')
				}
				sub.WriteString(w)
				subWords++
			}
			if sub.Len() > 0 {
				chunks = append(chunks, Chunk{
					Text:      sub.String(),
					Offset:    curOffset,
					Length:     sub.Len(),
					WordCount: subWords,
				})
				curOffset += sub.Len()
			}
			continue
		}

		// Adding this paragraph would exceed target — flush first.
		if curWords > 0 && curWords+paraWords > targetWords {
			flush()
		}

		if curText.Len() > 0 {
			curText.WriteString("\n\n")
		}
		curText.WriteString(para.text)
		curWords += paraWords
	}

	flush()
	return chunks
}

type paragraph struct {
	text string
}

// splitParagraphs splits content on double-newline boundaries.
func splitParagraphs(content string) []paragraph {
	raw := strings.Split(content, "\n\n")
	var out []paragraph
	for _, r := range raw {
		text := strings.TrimSpace(r)
		if text != "" {
			out = append(out, paragraph{text: text})
		}
	}
	return out
}

// ExtractTags automatically extracts meaningful tags from content.
// It identifies common keywords and phrases that would be useful for tagging.
func ExtractTags(content string) []string {
	// Common stop words to exclude
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "up": true, "about": true, "is": true,
		"are": true, "was": true, "were": true, "be": true, "been": true, "being": true,
		"have": true, "has": true, "had": true, "do": true, "does": true, "did": true,
		"will": true, "would": true, "could": true, "should": true, "may": true, "might": true,
		"must": true, "can": true, "that": true, "this": true, "it": true, "as": true,
		"if": true, "because": true, "while": true, "when": true, "where": true,
	}

	// Extract potential tags: mostly from headers and first few words
	tagMap := make(map[string]int)

	// Extract from headers (# ## ###)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			// Remove # symbols and markdown formatting
			header := strings.TrimLeft(line, "# ")
			header = strings.TrimSpace(header)
			// Extract words from header
			words := strings.Fields(strings.ToLower(header))
			for _, word := range words {
				word = strings.Trim(word, ".,!?;:")
				if len(word) > 2 && !stopWords[word] {
					tagMap[word]++
				}
			}
		}
	}

	// Extract common multi-word phrases (from first paragraph mostly)
	firstPara := ""
	for _, line := range lines {
		if line := strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "#") {
			firstPara = line
			break
		}
	}

	if firstPara != "" {
		words := strings.Fields(strings.ToLower(firstPara))
		// Look for 2-word phrases
		for i := 0; i < len(words)-1; i++ {
			w1 := strings.Trim(words[i], ".,!?;:")
			w2 := strings.Trim(words[i+1], ".,!?;:")
			if len(w1) > 2 && len(w2) > 2 && !stopWords[w1] && !stopWords[w2] {
				phrase := w1 + "-" + w2
				tagMap[phrase]++
			}
		}
	}

	// Build result, sorted by frequency
	type tagScore struct {
		tag   string
		score int
	}
	var tags []tagScore
	for tag, score := range tagMap {
		tags = append(tags, tagScore{tag, score})
	}

	// Sort by frequency (descending)
	for i := 0; i < len(tags); i++ {
		for j := i + 1; j < len(tags); j++ {
			if tags[j].score > tags[i].score {
				tags[i], tags[j] = tags[j], tags[i]
			}
		}
	}

	// Return top 5 tags
	result := make([]string, 0, 5)
	for i := 0; i < len(tags) && i < 5; i++ {
		result = append(result, tags[i].tag)
	}

	return result
}

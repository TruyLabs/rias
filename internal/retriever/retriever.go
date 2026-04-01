package retriever

import (
	"context"
	"sort"
	"strings"

	"github.com/tinhvqbk/kai/internal/brain"
)

var stopWords = map[string]bool{
	"the": true, "is": true, "a": true, "an": true, "and": true,
	"or": true, "but": true, "in": true, "on": true, "at": true,
	"to": true, "for": true, "of": true, "with": true, "by": true,
	"it": true, "this": true, "that": true, "are": true, "was": true,
	"be": true, "have": true, "has": true, "had": true, "do": true,
	"does": true, "did": true, "will": true, "would": true, "could": true,
	"should": true, "may": true, "might": true, "can": true, "i": true,
	"me": true, "my": true, "we": true, "you": true, "your": true,
	"what": true, "how": true, "why": true, "when": true, "where": true,
	"which": true, "who": true, "about": true, "think": true,
}

// FileRetriever retrieves brain files by keyword/tag matching.
type FileRetriever struct {
	brain           *brain.FileBrain
	maxContextFiles int
}

// New creates a FileRetriever.
func New(b *brain.FileBrain, maxContextFiles int) *FileRetriever {
	return &FileRetriever{brain: b, maxContextFiles: maxContextFiles}
}

// Retrieve finds brain files relevant to the query.
// Always includes identity/profile.md as baseline context.
// Uses full-text TF-IDF search with relevance scoring, falling back to tag-based search.
func (r *FileRetriever) Retrieve(ctx context.Context, query string, limit int) ([]*brain.BrainFile, error) {
	if limit > r.maxContextFiles {
		limit = r.maxContextFiles
	}

	// Always include profile as first result
	var results []*brain.BrainFile
	profile, err := r.brain.Load("identity/profile.md")
	if err == nil {
		results = append(results, profile)
	}

	seen := map[string]bool{"identity/profile.md": true}

	// Try full-text TF-IDF search first
	type scoredPath struct {
		path  string
		score float64
	}
	var candidates []scoredPath

	fullResults := r.brain.QueryFullIndex(query)
	if len(fullResults) > 0 {
		for _, fr := range fullResults {
			if seen[fr.Path] {
				continue
			}
			candidates = append(candidates, scoredPath{path: fr.Path, score: fr.Score})
		}
	} else {
		// Fall back to tag-based search
		keywords := extractKeywords(query)
		tagResults := r.brain.QueryIndex(keywords)
		for _, tr := range tagResults {
			if seen[tr.Path] {
				continue
			}
			candidates = append(candidates, scoredPath{path: tr.Path, score: float64(tr.Score)})
		}
	}

	// Load files and apply relevance scoring
	type loadedCandidate struct {
		bf    *brain.BrainFile
		score float64
	}
	var loaded []loadedCandidate
	for _, c := range candidates {
		bf, err := r.brain.Load(c.path)
		if err != nil {
			continue
		}
		finalScore := brain.RelevanceScore(bf, c.score)
		loaded = append(loaded, loadedCandidate{bf: bf, score: finalScore})
	}

	// Sort by final relevance score descending
	sort.Slice(loaded, func(i, j int) bool {
		return loaded[i].score > loaded[j].score
	})

	// Collect top results
	for _, lc := range loaded {
		if len(results) >= limit {
			break
		}
		if seen[lc.bf.Path] {
			continue
		}
		results = append(results, lc.bf)
		seen[lc.bf.Path] = true
	}

	// Touch each retrieved file to track access (best-effort, don't fail retrieval)
	for _, bf := range results {
		_ = r.brain.Touch(bf.Path)
	}

	return results, nil
}

func extractKeywords(query string) []string {
	words := strings.Fields(strings.ToLower(query))
	var keywords []string
	for _, w := range words {
		// Strip punctuation
		w = strings.Trim(w, "?!.,;:'\"")
		if w == "" || stopWords[w] {
			continue
		}
		keywords = append(keywords, w)
	}
	return keywords
}

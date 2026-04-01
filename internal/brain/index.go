package brain

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Index file names.
const (
	IndexFileName     = "index.json"
	FullIndexFileName = "index_full.json"
)

// TF-IDF field boost weights.
const (
	FieldBoostTag     = 3.0
	FieldBoostPath    = 2.0
	FieldBoostContent = 1.0
)

// RebuildIndex scans all brain files and rebuilds index.json.
func (b *FileBrain) RebuildIndex() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.rebuildIndex()
}

// rebuildIndex is the internal unlocked implementation of RebuildIndex.
func (b *FileBrain) rebuildIndex() error {
	idx := &Index{Tags: make(map[string][]string)}

	files, err := b.listAll()
	if err != nil {
		return fmt.Errorf("list brain files: %w", err)
	}

	for _, f := range files {
		bf, err := b.load(f)
		if err != nil {
			continue // skip malformed files
		}
		for _, tag := range bf.Tags {
			idx.Tags[tag] = append(idx.Tags[tag], f)
		}
	}

	if err := b.saveIndex(idx); err != nil {
		return err
	}
	return b.rebuildFullIndex()
}

// LoadIndex reads the index.json file.
func (b *FileBrain) LoadIndex() (*Index, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.loadIndex()
}

// loadIndex is the internal unlocked implementation of LoadIndex.
func (b *FileBrain) loadIndex() (*Index, error) {
	path := filepath.Join(b.root, IndexFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read index: %w", err)
	}

	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parse index: %w", err)
	}
	return &idx, nil
}

func (b *FileBrain) saveIndex(idx *Index) error {
	path := filepath.Join(b.root, IndexFileName)
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal index: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write index tmp: %w", err)
	}
	return os.Rename(tmpPath, path)
}

// IndexResult holds a brain file path and its match score.
type IndexResult struct {
	Path  string
	Score int
}

// QueryIndex returns brain file paths ranked by number of matching tags.
func (b *FileBrain) QueryIndex(keywords []string) []IndexResult {
	b.mu.RLock()
	defer b.mu.RUnlock()

	idx, err := b.loadIndex()
	if err != nil {
		return nil
	}

	scores := make(map[string]int)
	for _, kw := range keywords {
		if paths, ok := idx.Tags[kw]; ok {
			for _, p := range paths {
				scores[p]++
			}
		}
	}

	results := make([]IndexResult, 0, len(scores))
	for path, score := range scores {
		results = append(results, IndexResult{Path: path, Score: score})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

// RebuildFullIndex builds the full-text inverted index (public, write-locked).
func (b *FileBrain) RebuildFullIndex() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.rebuildFullIndex()
}

// rebuildFullIndex is the internal unlocked implementation.
func (b *FileBrain) rebuildFullIndex() error {
	files, err := b.listAll()
	if err != nil {
		return fmt.Errorf("list brain files: %w", err)
	}

	idx := &FullIndex{
		Documents:     make(map[string]DocEntry),
		InvertedIndex: make(map[string][]Posting),
		TotalDocs:     0,
	}

	for _, f := range files {
		bf, err := b.load(f)
		if err != nil {
			continue
		}

		contentTokens := tokenizeAndStem(bf.Content)
		idx.Documents[f] = DocEntry{
			Path:        f,
			WordCount:   len(contentTokens),
			Tags:        bf.Tags,
			Updated:     bf.Updated.Time.Format(DateFormat),
			AccessCount: bf.AccessCount,
		}
		idx.TotalDocs++

		// Index content terms
		contentFreq := make(map[string]int)
		for _, t := range contentTokens {
			contentFreq[t]++
		}
		for term, freq := range contentFreq {
			idx.InvertedIndex[term] = append(idx.InvertedIndex[term], Posting{
				Path:      f,
				Frequency: freq,
				Field:     "content",
			})
		}

		// Index tag terms (stem each tag)
		tagFreq := make(map[string]int)
		for _, tag := range bf.Tags {
			for _, t := range tokenizeAndStem(tag) {
				tagFreq[t]++
			}
		}
		for term, freq := range tagFreq {
			idx.InvertedIndex[term] = append(idx.InvertedIndex[term], Posting{
				Path:      f,
				Frequency: freq,
				Field:     "tag",
			})
		}

		// Index path segments
		pathSegments := strings.Split(strings.TrimSuffix(f, ".md"), string(filepath.Separator))
		pathFreq := make(map[string]int)
		for _, seg := range pathSegments {
			for _, t := range tokenizeAndStem(seg) {
				pathFreq[t]++
			}
		}
		for term, freq := range pathFreq {
			idx.InvertedIndex[term] = append(idx.InvertedIndex[term], Posting{
				Path:      f,
				Frequency: freq,
				Field:     "path",
			})
		}
	}

	return b.saveFullIndex(idx)
}

// LoadFullIndex reads the index_full.json file (public, read-locked).
func (b *FileBrain) LoadFullIndex() (*FullIndex, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.loadFullIndex()
}

// loadFullIndex is the internal unlocked implementation.
func (b *FileBrain) loadFullIndex() (*FullIndex, error) {
	path := filepath.Join(b.root, FullIndexFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read full index: %w", err)
	}
	var idx FullIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parse full index: %w", err)
	}
	return &idx, nil
}

func (b *FileBrain) saveFullIndex(idx *FullIndex) error {
	path := filepath.Join(b.root, FullIndexFileName)
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal full index: %w", err)
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write full index tmp: %w", err)
	}
	return os.Rename(tmpPath, path)
}

// FullIndexResult holds a brain file path and its TF-IDF score.
type FullIndexResult struct {
	Path  string
	Score float64
}

// QueryFullIndex searches the full-text index using TF-IDF with field boosts (public, read-locked).
func (b *FileBrain) QueryFullIndex(query string) []FullIndexResult {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.queryFullIndex(query)
}

// queryFullIndex is the internal unlocked implementation.
func (b *FileBrain) queryFullIndex(query string) []FullIndexResult {
	idx, err := b.loadFullIndex()
	if err != nil {
		return nil
	}

	terms := tokenizeAndStem(query)
	if len(terms) == 0 {
		return nil
	}

	fieldBoost := map[string]float64{
		"tag":     FieldBoostTag,
		"path":    FieldBoostPath,
		"content": FieldBoostContent,
	}

	scores := make(map[string]float64)

	for _, term := range terms {
		postings, ok := idx.InvertedIndex[term]
		if !ok {
			continue
		}

		// docFreq = number of unique documents this term appears in
		docSet := make(map[string]bool)
		for _, p := range postings {
			docSet[p.Path] = true
		}
		docFreq := len(docSet)
		idf := math.Log(float64(idx.TotalDocs)/float64(docFreq)) + 1.0

		for _, p := range postings {
			tf := float64(p.Frequency)
			boost := fieldBoost[p.Field]
			scores[p.Path] += tf * idf * boost
		}
	}

	results := make([]FullIndexResult, 0, len(scores))
	for path, score := range scores {
		results = append(results, FullIndexResult{Path: path, Score: score})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

package brain

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Index file names.
const (
	IndexFileName     = "index.json"
	FullIndexFileName = "index_full.json.gz"
)

// BM25 parameters.
const (
	BM25K1 = 1.5 // Term frequency saturation. Higher = more weight to repeated terms.
	BM25B  = 0.5 // Length normalization. Lower than standard 0.75 — brain files vary
	             // widely in length and short concise files should rank fairly against
	             // long consolidated ones.
)

// Field boost weights applied on top of BM25 scores.
const (
	FieldBoostTag     = 3.0
	FieldBoostPath    = 2.0
	FieldBoostContent = 1.0
)

// PRF (Pseudo-Relevance Feedback) parameters.
const (
	PRFTopK        = 3 // Number of top chunks used for feedback.
	PRFExpandTerms = 5 // Number of expansion terms to add.
	PRFMinScore    = 1.0 // Minimum top-chunk score to trigger PRF.
)

// RebuildIndex scans all brain files and rebuilds both the tag index and the
// full BM25+vector index. Use this at startup or when a full reindex is needed.
func (b *FileBrain) RebuildIndex() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.rebuildIndex()
}

// RebuildTagIndex rebuilds only the lightweight tag index (index.json).
// Use this after writes — it is fast and does not call Ollama.
func (b *FileBrain) RebuildTagIndex() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.rebuildTagIndex()
}

// rebuildIndex is the internal unlocked implementation of RebuildIndex.
func (b *FileBrain) rebuildIndex() error {
	// Prime a shared file cache so each brain file is read from disk only once
	// across the tag, BM25, and vector index phases.
	b.fileCache = make(map[string]*BrainFile)
	defer func() { b.fileCache = nil }()

	if err := b.rebuildTagIndex(); err != nil {
		return err
	}
	return b.rebuildFullIndex()
}

// rebuildTagIndex rebuilds only index.json (tag-based lookup). Fast, no Ollama.
func (b *FileBrain) rebuildTagIndex() error {
	idx := &Index{Tags: make(map[string][]string)}

	files, err := b.listAll()
	if err != nil {
		return fmt.Errorf("list brain files: %w", err)
	}
	slog.Debug("rebuilding tag index", "files", len(files))

	for _, f := range files {
		bf, err := b.loadCached(f)
		if err != nil {
			slog.Debug("tag index: skipping unreadable file", "path", f, "err", err)
			continue
		}
		for _, tag := range bf.Tags {
			idx.Tags[tag] = append(idx.Tags[tag], f)
		}
	}

	slog.Debug("tag index rebuilt", "files", len(files), "unique_tags", len(idx.Tags))
	return b.saveIndex(idx)
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
	slog.Debug("rebuilding full BM25 index", "files", len(files))

	idx := &FullIndex{
		Documents:     make(map[string]DocEntry),
		Chunks:        make(map[string]ChunkEntry),
		InvertedIndex: make(map[string][]Posting),
	}

	var totalDocWords, totalChunkWords int

	for _, f := range files {
		bf, err := b.loadCached(f)
		if err != nil {
			continue
		}

		content := strings.TrimSpace(bf.Content)
		contentTokens := tokenizeAndStem(content)
		docWordCount := len(contentTokens)
		totalDocWords += docWordCount

		// Chunk the content.
		chunks := chunkContent(content, ChunkTargetWords)
		if len(chunks) == 0 {
			chunks = []Chunk{{Text: content, Offset: 0, Length: len(content), WordCount: docWordCount}}
		}

		idx.Documents[f] = DocEntry{
			Path:        f,
			WordCount:   docWordCount,
			Tags:        bf.Tags,
			Updated:     bf.Updated.Time.Format(DateFormat),
			AccessCount: bf.AccessCount,
			ChunkCount:  len(chunks),
		}
		idx.TotalDocs++

		// Index each chunk's content terms.
		for ci, ch := range chunks {
			chunkKey := fmt.Sprintf("%s#%d", f, ci)
			chunkTokens := tokenizeAndStem(ch.Text)
			chunkWordCount := len(chunkTokens)
			totalChunkWords += chunkWordCount

			idx.Chunks[chunkKey] = ChunkEntry{
				DocPath:   f,
				ChunkID:   ci,
				WordCount: chunkWordCount,
				Offset:    ch.Offset,
				Length:    ch.Length,
			}
			idx.TotalChunks++

			freq := make(map[string]int)
			for _, t := range chunkTokens {
				freq[t]++
			}
			for term, count := range freq {
				idx.InvertedIndex[term] = append(idx.InvertedIndex[term], Posting{
					Path:      f,
					Frequency: count,
					Field:     "content",
					ChunkKey:  chunkKey,
				})
			}
		}

		// Index tag terms (document-level, no ChunkKey).
		tagFreq := make(map[string]int)
		for _, tag := range bf.Tags {
			for _, t := range tokenizeAndStem(tag) {
				tagFreq[t]++
			}
		}
		for term, count := range tagFreq {
			idx.InvertedIndex[term] = append(idx.InvertedIndex[term], Posting{
				Path:      f,
				Frequency: count,
				Field:     "tag",
			})
		}

		// Index path segments (document-level, no ChunkKey).
		pathSegments := strings.Split(strings.TrimSuffix(f, ".md"), string(filepath.Separator))
		pathFreq := make(map[string]int)
		for _, seg := range pathSegments {
			for _, t := range tokenizeAndStem(seg) {
				pathFreq[t]++
			}
		}
		for term, count := range pathFreq {
			idx.InvertedIndex[term] = append(idx.InvertedIndex[term], Posting{
				Path:      f,
				Frequency: count,
				Field:     "path",
			})
		}
	}

	if idx.TotalDocs > 0 {
		idx.AvgDocLength = float64(totalDocWords) / float64(idx.TotalDocs)
	}
	if idx.TotalChunks > 0 {
		idx.AvgChunkLength = float64(totalChunkWords) / float64(idx.TotalChunks)
	}

	slog.Debug("full BM25 index built", "docs", idx.TotalDocs, "chunks", idx.TotalChunks)
	if err := b.saveFullIndex(idx); err != nil {
		return err
	}

	// Build and save vector index for hybrid search.
	slog.Debug("building vector index", "provider", b.embed.Provider)
	vi := b.buildVecIndex(idx)
	if vi != nil {
		slog.Debug("saving vector index", "chunks", len(vi.ChunkVecs))
		if err := b.saveVecIndex(vi); err != nil {
			return fmt.Errorf("save vec index: %w", err)
		}
	} else {
		slog.Debug("vector index skipped (no embedder available)")
	}

	return nil
}

// buildVecIndex selects the embedding backend and builds the vector index.
func (b *FileBrain) buildVecIndex(idx *FullIndex) *VecIndex {
	switch b.embed.Provider {
	case EmbedOllama:
		return b.tryOllamaEmbed(idx)

	case EmbedLSI:
		return buildVecIndexLSI(idx)

	default: // Auto: try Ollama first, fall back to LSI.
		if vi := b.tryOllamaEmbed(idx); vi != nil {
			return vi
		}
		return buildVecIndexLSI(idx)
	}
}

// tryOllamaEmbed attempts to build the vector index using Ollama.
// Loads the existing vector index for incremental embedding.
func (b *FileBrain) tryOllamaEmbed(idx *FullIndex) *VecIndex {
	embedder := NewOllamaEmbedder(OllamaEmbedConfig{
		URL:   b.embed.OllamaURL,
		Model: b.embed.OllamaModel,
	})
	slog.Debug("checking ollama embedder availability", "url", b.embed.OllamaURL, "model", b.embed.OllamaModel)
	if !embedder.Available() {
		slog.Debug("ollama embedder not available, skipping vector index")
		return nil
	}
	slog.Debug("ollama embedder available, embedding chunks", "chunks", len(idx.Chunks))

	// Load existing index for incremental rebuild.
	existing, _ := b.loadVecIndex()
	return buildVecIndexOllama(b, idx, embedder, existing)
}

// LoadFullIndex reads the index_full.json file (public, read-locked).
func (b *FileBrain) LoadFullIndex() (*FullIndex, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.loadFullIndex()
}

// loadFullIndex is the internal unlocked implementation.
func (b *FileBrain) loadFullIndex() (*FullIndex, error) {
	return b.loadFullIndexCompact()
}

func (b *FileBrain) saveFullIndex(idx *FullIndex) error {
	return b.saveFullIndexCompact(idx)
}

// FullIndexResult holds a brain file path and its BM25 score.
type FullIndexResult struct {
	Path  string
	Score float64
}

// QueryFullIndex searches using BM25 and returns document-level results.
// Internally aggregates chunk scores (max per doc). Backward-compatible.
func (b *FileBrain) QueryFullIndex(query string) []FullIndexResult {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.queryFullIndex(query)
}

func (b *FileBrain) queryFullIndex(query string) []FullIndexResult {
	chunks := b.queryFullIndexChunks(query)
	if len(chunks) == 0 {
		return nil
	}

	// Aggregate: keep max chunk score per document.
	docMax := make(map[string]float64)
	for _, c := range chunks {
		if c.Score > docMax[c.DocPath] {
			docMax[c.DocPath] = c.Score
		}
	}

	results := make([]FullIndexResult, 0, len(docMax))
	for path, score := range docMax {
		results = append(results, FullIndexResult{Path: path, Score: score})
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	return results
}

// QueryFullIndexChunks searches using BM25 and returns chunk-level results.
func (b *FileBrain) QueryFullIndexChunks(query string) []ChunkResult {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.queryFullIndexChunks(query)
}

func (b *FileBrain) queryFullIndexChunks(query string) []ChunkResult {
	idx, err := b.loadFullIndex()
	if err != nil {
		return nil
	}
	return queryChunksBM25(idx, tokenizeAndStem(query))
}

// queryChunksBM25 scores chunks against the given stemmed query terms.
func queryChunksBM25(idx *FullIndex, terms []string) []ChunkResult {
	if len(terms) == 0 || idx.TotalChunks == 0 {
		return nil
	}

	avgdl := idx.AvgChunkLength
	if avgdl == 0 {
		avgdl = idx.AvgDocLength
	}
	if avgdl == 0 {
		avgdl = 1
	}
	N := float64(idx.TotalDocs)

	scores := make(map[string]float64) // chunk key → score

	for _, term := range terms {
		postings, ok := idx.InvertedIndex[term]
		if !ok {
			continue
		}

		// Count unique documents for IDF.
		docSet := make(map[string]bool)
		for _, p := range postings {
			docSet[p.Path] = true
		}
		n := float64(len(docSet))
		idf := math.Log((N-n+0.5)/(n+0.5) + 1.0)

		for _, p := range postings {
			tf := float64(p.Frequency)
			boost := fieldBoost(p.Field)

			if p.ChunkKey != "" {
				// Content posting — score at chunk level.
				dl := float64(idx.Chunks[p.ChunkKey].WordCount)
				num := tf * (BM25K1 + 1)
				denom := tf + BM25K1*(1-BM25B+BM25B*dl/avgdl)
				scores[p.ChunkKey] += idf * (num / denom) * boost
			} else {
				// Tag/path posting — distribute to all chunks of this document,
				// divided by chunk count to avoid inflating multi-chunk files.
				doc := idx.Documents[p.Path]
				chunkCount := doc.ChunkCount
				if chunkCount == 0 {
					chunkCount = 1
				}
				dl := float64(doc.WordCount)
				num := tf * (BM25K1 + 1)
				denom := tf + BM25K1*(1-BM25B+BM25B*dl/avgdl)
				perChunk := idf * (num / denom) * boost / float64(chunkCount)
				for ci := 0; ci < doc.ChunkCount; ci++ {
					ck := fmt.Sprintf("%s#%d", p.Path, ci)
					scores[ck] += perChunk
				}
			}
		}
	}

	results := make([]ChunkResult, 0, len(scores))
	for ck, score := range scores {
		ce, ok := idx.Chunks[ck]
		if !ok {
			continue
		}
		results = append(results, ChunkResult{
			DocPath: ce.DocPath,
			ChunkID: ce.ChunkID,
			Score:   score,
			Offset:  ce.Offset,
			Length:   ce.Length,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	return results
}

// QueryWithPRF runs a two-pass search: BM25 → extract top-chunk terms → re-query.
func (b *FileBrain) QueryWithPRF(query string) []ChunkResult {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.queryWithPRF(query)
}

func (b *FileBrain) queryWithPRF(query string) []ChunkResult {
	idx, err := b.loadFullIndex()
	if err != nil {
		return nil
	}

	originalTerms := tokenizeAndStem(query)
	if len(originalTerms) == 0 {
		return nil
	}

	// First pass.
	firstPass := queryChunksBM25(idx, originalTerms)
	if len(firstPass) == 0 {
		return nil
	}

	// Skip PRF if top result is too weak.
	if firstPass[0].Score < PRFMinScore {
		return firstPass
	}

	// Collect text from top-K chunks.
	topK := PRFTopK
	if topK > len(firstPass) {
		topK = len(firstPass)
	}
	var feedbackText strings.Builder
	for _, cr := range firstPass[:topK] {
		ce, ok := idx.Chunks[fmt.Sprintf("%s#%d", cr.DocPath, cr.ChunkID)]
		if !ok {
			continue
		}
		// Load the chunk text from the inverted index isn't possible, so we
		// collect the terms that appear in this chunk from the inverted index.
		// Instead, we extract the top-IDF terms that co-occur in these chunks.
		_ = ce
	}
	_ = feedbackText

	// Build a set of original terms for dedup.
	origSet := make(map[string]bool, len(originalTerms))
	for _, t := range originalTerms {
		origSet[t] = true
	}

	// Collect candidate expansion terms: terms that appear in the top-K chunks
	// with their IDF scores. We iterate the inverted index and check which terms
	// have postings in our top chunk keys.
	topChunkKeys := make(map[string]bool, topK)
	for _, cr := range firstPass[:topK] {
		topChunkKeys[fmt.Sprintf("%s#%d", cr.DocPath, cr.ChunkID)] = true
	}

	type termScore struct {
		term string
		idf  float64
	}
	var candidates []termScore
	N := float64(idx.TotalDocs)

	for term, postings := range idx.InvertedIndex {
		if origSet[term] {
			continue
		}
		// Check if this term appears in any top chunk.
		inTopChunk := false
		for _, p := range postings {
			if p.ChunkKey != "" && topChunkKeys[p.ChunkKey] {
				inTopChunk = true
				break
			}
		}
		if !inTopChunk {
			continue
		}
		// Compute IDF.
		docSet := make(map[string]bool)
		for _, p := range postings {
			docSet[p.Path] = true
		}
		n := float64(len(docSet))
		idf := math.Log((N-n+0.5)/(n+0.5) + 1.0)
		candidates = append(candidates, termScore{term: term, idf: idf})
	}

	// Pick the highest-IDF expansion terms.
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].idf > candidates[j].idf
	})
	expandCount := PRFExpandTerms
	if expandCount > len(candidates) {
		expandCount = len(candidates)
	}

	expandedTerms := make([]string, len(originalTerms), len(originalTerms)+expandCount)
	copy(expandedTerms, originalTerms)
	for _, c := range candidates[:expandCount] {
		expandedTerms = append(expandedTerms, c.term)
	}

	// Second pass with expanded query.
	return queryChunksBM25(idx, expandedTerms)
}

// fieldBoost returns the weight multiplier for the given field.
func fieldBoost(field string) float64 {
	switch field {
	case "tag":
		return FieldBoostTag
	case "path":
		return FieldBoostPath
	default:
		return FieldBoostContent
	}
}

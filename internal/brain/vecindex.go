package brain

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// VecIndexFileName is the on-disk file for the vector index.
const VecIndexFileName = "vectors.bin.gz"

// Concurrent embedding worker count.
const EmbedWorkers = 4

// VecIndex stores chunk embeddings and metadata for query embedding.
type VecIndex struct {
	Dims      int                  `json:"dims"`
	Provider  string               `json:"provider"`         // "ollama" or "lsi"
	TermVecs  map[string][]float32 `json:"tv,omitempty"`     // stem → embedding (LSI only)
	IDF       map[string]float64   `json:"idf,omitempty"`    // stem → IDF weight (LSI only)
	ChunkVecs map[string][]float32 `json:"cv"`               // chunkKey → embedding
	Hashes    map[string]string    `json:"h,omitempty"`      // chunkKey → content hash (for incremental)
}

// VecResult holds a chunk key and its cosine similarity score.
type VecResult struct {
	ChunkKey string
	Score    float64
}

// contentHash returns a short SHA-256 hex hash of the text.
func contentHash(text string) string {
	h := sha256.Sum256([]byte(text))
	return fmt.Sprintf("%x", h[:12]) // 24 hex chars, collision-safe for 5K chunks
}

// buildVecIndexLSI builds the vector index using LSI embeddings.
func buildVecIndexLSI(idx *FullIndex) *VecIndex {
	model := BuildLSI(idx)
	if model == nil {
		return nil
	}

	chunkVecs := make(map[string][]float32, idx.TotalChunks)

	// Build chunk term frequencies from the inverted index.
	chunkTerms := make(map[string]map[string]int)
	for term, postings := range idx.InvertedIndex {
		for _, p := range postings {
			if p.Field != "content" || p.ChunkKey == "" {
				continue
			}
			if _, ok := chunkTerms[p.ChunkKey]; !ok {
				chunkTerms[p.ChunkKey] = make(map[string]int)
			}
			chunkTerms[p.ChunkKey][term] = p.Frequency
		}
	}

	for ck, terms := range chunkTerms {
		vec := make([]float64, model.Dims)
		var totalWeight float64

		for term, freq := range terms {
			tv, ok := model.TermVecs[term]
			if !ok {
				continue
			}
			w := math.Log(1.0+float64(freq)) * model.IDF[term]
			for i, v := range tv {
				vec[i] += float64(v) * w
			}
			totalWeight += w
		}

		if totalWeight == 0 {
			continue
		}

		embedding := make([]float32, model.Dims)
		for i := range embedding {
			embedding[i] = float32(vec[i] / totalWeight)
		}
		normalizeF32(embedding)
		chunkVecs[ck] = embedding
	}

	return &VecIndex{
		Dims:      model.Dims,
		Provider:  "lsi",
		TermVecs:  model.TermVecs,
		IDF:       model.IDF,
		ChunkVecs: chunkVecs,
	}
}

// embedJob is a unit of work for the concurrent embedding pool.
type embedJob struct {
	chunkKey string
	text     string
}

// embedResult is the output of a single embedding job.
type embedResult struct {
	chunkKey string
	vec      []float32
	hash     string
}

// buildVecIndexOllama builds the vector index using Ollama embeddings.
// Supports incremental mode: reuses vectors for unchanged chunks from existing index.
// Uses concurrent workers for parallel Ollama API calls.
func buildVecIndexOllama(b *FileBrain, idx *FullIndex, embedder *OllamaEmbedder, existing *VecIndex) *VecIndex {
	// Prepare chunk text and detect changes.
	type chunkInfo struct {
		key  string
		text string
		hash string
	}

	var toEmbed []embedJob
	reused := make(map[string][]float32)
	hashes := make(map[string]string)
	var dims int

	// If we have an existing Ollama index, reuse unchanged vectors.
	existingHashes := make(map[string]string)
	existingVecs := make(map[string][]float32)
	if existing != nil && existing.Provider == "ollama" && existing.Hashes != nil {
		existingHashes = existing.Hashes
		existingVecs = existing.ChunkVecs
		dims = existing.Dims
	}

	for chunkKey, ce := range idx.Chunks {
		bf, err := b.load(ce.DocPath)
		if err != nil {
			continue
		}
		content := strings.TrimSpace(bf.Content)
		chunkText := safeSubstring(content, ce.Offset, ce.Length)
		if chunkText == "" {
			continue
		}

		// Prepend tags for richer context.
		doc := idx.Documents[ce.DocPath]
		prefix := ""
		if len(doc.Tags) > 0 {
			prefix = "Tags: " + strings.Join(doc.Tags, ", ") + ". "
		}
		fullText := prefix + chunkText
		hash := contentHash(fullText)
		hashes[chunkKey] = hash

		// Check if unchanged — reuse existing vector.
		if oldHash, ok := existingHashes[chunkKey]; ok && oldHash == hash {
			if vec, ok := existingVecs[chunkKey]; ok {
				reused[chunkKey] = vec
				continue
			}
		}

		toEmbed = append(toEmbed, embedJob{chunkKey: chunkKey, text: fullText})
	}

	// Embed new/changed chunks concurrently.
	var newVecs []embedResult
	if len(toEmbed) > 0 {
		newVecs = embedConcurrent(embedder, toEmbed, hashes)
	}

	// Merge reused + new vectors.
	chunkVecs := make(map[string][]float32, len(reused)+len(newVecs))
	for k, v := range reused {
		chunkVecs[k] = v
	}
	for _, r := range newVecs {
		chunkVecs[r.chunkKey] = r.vec
		if dims == 0 {
			dims = len(r.vec)
		}
	}

	if len(chunkVecs) == 0 {
		return nil
	}

	return &VecIndex{
		Dims:      dims,
		Provider:  "ollama",
		ChunkVecs: chunkVecs,
		Hashes:    hashes,
	}
}

// embedConcurrent runs embedding jobs across multiple workers.
func embedConcurrent(embedder *OllamaEmbedder, jobs []embedJob, hashes map[string]string) []embedResult {
	workers := EmbedWorkers
	if workers > len(jobs) {
		workers = len(jobs)
	}

	jobCh := make(chan embedJob, len(jobs))
	resultCh := make(chan embedResult, len(jobs))

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobCh {
				vec, err := embedder.Embed(job.text)
				if err != nil {
					continue
				}
				normalizeF32(vec)
				resultCh <- embedResult{
					chunkKey: job.chunkKey,
					vec:      vec,
					hash:     hashes[job.chunkKey],
				}
			}
		}()
	}

	for _, job := range jobs {
		jobCh <- job
	}
	close(jobCh)

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	var results []embedResult
	for r := range resultCh {
		results = append(results, r)
	}
	return results
}

// safeSubstring extracts a substring safely by offset and length.
func safeSubstring(s string, offset, length int) string {
	if offset >= len(s) {
		return s
	}
	end := offset + length
	if end > len(s) {
		end = len(s)
	}
	return strings.TrimSpace(s[offset:end])
}

// QueryVec embeds a query and returns chunks ranked by cosine similarity.
// For LSI provider, uses stored term vectors. For Ollama, needs an embedder.
func (vi *VecIndex) QueryVec(query string) []VecResult {
	if vi == nil || len(vi.ChunkVecs) == 0 {
		return nil
	}

	var qVec []float32

	switch vi.Provider {
	case "ollama":
		embedder := NewOllamaEmbedder(OllamaEmbedConfig{})
		vec, err := embedder.Embed(query)
		if err != nil {
			return nil
		}
		normalizeF32(vec)
		qVec = vec

	default: // "lsi"
		qVec = vi.embedQueryLSI(query)
	}

	if qVec == nil {
		return nil
	}

	return searchBruteForce(vi.ChunkVecs, qVec)
}

// QueryVecWithEmbedder queries using a specific Ollama embedder instance.
func (vi *VecIndex) QueryVecWithEmbedder(query string, embedder *OllamaEmbedder) []VecResult {
	if vi == nil || len(vi.ChunkVecs) == 0 || embedder == nil {
		return nil
	}

	vec, err := embedder.Embed(query)
	if err != nil {
		return nil
	}
	normalizeF32(vec)

	return searchBruteForce(vi.ChunkVecs, vec)
}

// searchBruteForce scans all vectors and returns sorted results.
func searchBruteForce(chunkVecs map[string][]float32, qVec []float32) []VecResult {
	results := make([]VecResult, 0, len(chunkVecs))
	for ck, cv := range chunkVecs {
		if len(cv) != len(qVec) {
			continue
		}
		score := cosineF32(qVec, cv)
		if score > 0 {
			results = append(results, VecResult{ChunkKey: ck, Score: score})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

// embedQueryLSI embeds a query using stored LSI term vectors.
func (vi *VecIndex) embedQueryLSI(query string) []float32 {
	terms := tokenizeAndStem(query)
	if len(terms) == 0 {
		return nil
	}

	qVec := make([]float64, vi.Dims)
	var totalWeight float64

	termFreq := make(map[string]int)
	for _, t := range terms {
		termFreq[t]++
	}

	for term, freq := range termFreq {
		tv, ok := vi.TermVecs[term]
		if !ok {
			continue
		}
		w := math.Log(1.0+float64(freq)) * vi.IDF[term]
		for i, v := range tv {
			qVec[i] += float64(v) * w
		}
		totalWeight += w
	}

	if totalWeight == 0 {
		return nil
	}

	result := make([]float32, vi.Dims)
	for i := range result {
		result[i] = float32(qVec[i] / totalWeight)
	}
	normalizeF32(result)
	return result
}

// saveVecIndex writes the vector index to disk as gzip-compressed JSON.
func (b *FileBrain) saveVecIndex(vi *VecIndex) error {
	if vi == nil {
		return nil
	}

	path := filepath.Join(b.root, VecIndexFileName)
	data, err := json.Marshal(vi)
	if err != nil {
		return fmt.Errorf("marshal vec index: %w", err)
	}

	tmpPath := path + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create vec index tmp: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewWriterLevel(f, gzip.BestCompression)
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("create gzip writer: %w", err)
	}

	if _, err := gz.Write(data); err != nil {
		gz.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write vec index: %w", err)
	}
	if err := gz.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close gzip writer: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close vec index file: %w", err)
	}

	return os.Rename(tmpPath, path)
}

// loadVecIndex reads the vector index from disk.
func (b *FileBrain) loadVecIndex() (*VecIndex, error) {
	path := filepath.Join(b.root, VecIndexFileName)
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open vec index: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("create gzip reader: %w", err)
	}
	defer gz.Close()

	var vi VecIndex
	if err := json.NewDecoder(gz).Decode(&vi); err != nil {
		return nil, fmt.Errorf("decode vec index: %w", err)
	}

	return &vi, nil
}

// LoadVecIndex reads the vector index (public, read-locked).
func (b *FileBrain) LoadVecIndex() (*VecIndex, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.loadVecIndex()
}

// normalizeF32 normalizes a float32 vector to unit length in-place.
func normalizeF32(v []float32) {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	norm := math.Sqrt(sum)
	if norm < 1e-10 {
		return
	}
	for i := range v {
		v[i] = float32(float64(v[i]) / norm)
	}
}

// cosineF32 computes cosine similarity between two unit vectors.
func cosineF32(a, b []float32) float64 {
	var sum float64
	for i := range a {
		sum += float64(a[i]) * float64(b[i])
	}
	return sum
}

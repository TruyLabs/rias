package brain

import (
	"math"
	"sort"
)

// LSI embedding parameters.
const (
	LSIDims    = 50 // Target embedding dimensions.
	LSIMinDocs = 5  // Minimum documents needed for meaningful LSI.
	LSIMinDF   = 2  // Minimum document frequency for a term to be included.
	LSIMaxIter = 100 // Power iteration convergence iterations.
)

// LSIModel holds the learned LSI embeddings and IDF weights.
// Used to embed queries at search time.
type LSIModel struct {
	Dims     int
	TermVecs map[string][]float32 // stem → embedding
	IDF      map[string]float64   // stem → IDF weight
}

// BuildLSI computes an LSI model from the full index.
// Returns nil if there aren't enough documents.
func BuildLSI(idx *FullIndex) *LSIModel {
	if idx.TotalDocs < LSIMinDocs {
		return nil
	}

	dims := LSIDims
	if dims >= idx.TotalDocs {
		dims = idx.TotalDocs - 1
	}
	if dims < 2 {
		return nil
	}

	// Collect document paths (sorted for determinism).
	docPaths := make([]string, 0, len(idx.Documents))
	for p := range idx.Documents {
		docPaths = append(docPaths, p)
	}
	sort.Strings(docPaths)
	docIdx := make(map[string]int, len(docPaths))
	for i, p := range docPaths {
		docIdx[p] = i
	}

	nDocs := len(docPaths)
	N := float64(nDocs)

	// Collect content terms that appear in at least LSIMinDF documents.
	termDF := make(map[string]int) // stem → document frequency
	for term, postings := range idx.InvertedIndex {
		docs := make(map[string]bool)
		for _, p := range postings {
			if p.Field == "content" {
				docs[p.Path] = true
			}
		}
		if len(docs) >= LSIMinDF {
			termDF[term] = len(docs)
		}
	}

	// Sort terms for deterministic matrix construction.
	terms := make([]string, 0, len(termDF))
	for t := range termDF {
		terms = append(terms, t)
	}
	sort.Strings(terms)
	termIdx := make(map[string]int, len(terms))
	for i, t := range terms {
		termIdx[t] = i
	}

	nTerms := len(terms)
	if nTerms < dims {
		dims = nTerms
	}
	if dims < 2 {
		return nil
	}

	// Compute IDF weights.
	idf := make(map[string]float64, nTerms)
	for _, t := range terms {
		df := float64(termDF[t])
		idf[t] = math.Log(N / df)
	}

	// Build TF-IDF matrix A (nTerms × nDocs).
	// TF = log(1 + raw_count), IDF = log(N/df).
	// We aggregate term frequency across all chunks of a document.
	A := make([][]float64, nTerms)
	for i := range A {
		A[i] = make([]float64, nDocs)
	}

	for term, postings := range idx.InvertedIndex {
		ti, ok := termIdx[term]
		if !ok {
			continue
		}
		// Aggregate frequency per document for content postings.
		docFreq := make(map[string]int)
		for _, p := range postings {
			if p.Field == "content" {
				docFreq[p.Path] += p.Frequency
			}
		}
		for path, freq := range docFreq {
			di, ok := docIdx[path]
			if !ok {
				continue
			}
			A[ti][di] = math.Log(1.0+float64(freq)) * idf[term]
		}
	}

	// Compute SVD: A ≈ U * Σ * V^T
	// Since nDocs << nTerms, compute via A^T*A (nDocs × nDocs).
	U, _ := truncatedSVD(A, nTerms, nDocs, dims)

	// Build term embedding map.
	termVecs := make(map[string][]float32, nTerms)
	for i, t := range terms {
		vec := make([]float32, dims)
		for j := 0; j < dims; j++ {
			vec[j] = float32(U[i][j])
		}
		termVecs[t] = vec
	}

	return &LSIModel{
		Dims:     dims,
		TermVecs: termVecs,
		IDF:      idf,
	}
}

// EmbedTerms computes a weighted average embedding for a list of stemmed terms.
func (m *LSIModel) EmbedTerms(terms []string) []float32 {
	if m == nil || len(terms) == 0 {
		return nil
	}

	vec := make([]float64, m.Dims)
	var totalWeight float64

	for _, t := range terms {
		tv, ok := m.TermVecs[t]
		if !ok {
			continue
		}
		w := m.IDF[t]
		if w == 0 {
			w = 1
		}
		for i, v := range tv {
			vec[i] += float64(v) * w
		}
		totalWeight += w
	}

	if totalWeight == 0 {
		return nil
	}

	result := make([]float32, m.Dims)
	for i := range result {
		result[i] = float32(vec[i] / totalWeight)
	}
	return result
}

// truncatedSVD computes the top-k left singular vectors of the m×n matrix A
// via eigendecomposition of A^T*A (efficient when n << m).
// Returns U (m × k) and singular values σ (k).
func truncatedSVD(A [][]float64, m, n, k int) ([][]float64, []float64) {
	// C = A^T * A (n × n symmetric matrix).
	C := make([][]float64, n)
	for i := range C {
		C[i] = make([]float64, n)
		for j := 0; j <= i; j++ {
			var sum float64
			for r := 0; r < m; r++ {
				sum += A[r][i] * A[r][j]
			}
			C[i][j] = sum
			C[j][i] = sum
		}
	}

	// Eigendecomposition via power iteration with deflation.
	eigenVecs := make([][]float64, k)
	eigenVals := make([]float64, k)

	for ki := 0; ki < k; ki++ {
		// Deterministic initial vector (different per ki).
		v := make([]float64, n)
		for i := range v {
			// Simple deterministic seed based on ki and index.
			v[i] = math.Sin(float64(i*17+ki*31+1)) + 0.1
		}
		vecNormalize(v)

		for iter := 0; iter < LSIMaxIter; iter++ {
			// v_new = C * v
			newV := make([]float64, n)
			for i := 0; i < n; i++ {
				var sum float64
				for j := 0; j < n; j++ {
					sum += C[i][j] * v[j]
				}
				newV[i] = sum
			}

			// Deflate: remove components of previous eigenvectors.
			for pi := 0; pi < ki; pi++ {
				d := vecDot(newV, eigenVecs[pi])
				for j := 0; j < n; j++ {
					newV[j] -= d * eigenVecs[pi][j]
				}
			}

			norm := vecNorm2(newV)
			if norm < 1e-12 {
				break
			}
			for j := range newV {
				newV[j] /= norm
			}
			v = newV
		}

		eigenVecs[ki] = v

		// Eigenvalue = v^T * C * v.
		Cv := make([]float64, n)
		for i := 0; i < n; i++ {
			var sum float64
			for j := 0; j < n; j++ {
				sum += C[i][j] * v[j]
			}
			Cv[i] = sum
		}
		eigenVals[ki] = vecDot(v, Cv)
	}

	// Compute singular values.
	sigma := make([]float64, k)
	for i := range sigma {
		if eigenVals[i] > 0 {
			sigma[i] = math.Sqrt(eigenVals[i])
		}
	}

	// Compute left singular vectors: U = A * V * Σ^(-1).
	U := make([][]float64, m)
	for i := 0; i < m; i++ {
		U[i] = make([]float64, k)
		for j := 0; j < k; j++ {
			if sigma[j] < 1e-10 {
				continue
			}
			var sum float64
			for l := 0; l < n; l++ {
				sum += A[i][l] * eigenVecs[j][l]
			}
			U[i][j] = sum / sigma[j]
		}
	}

	return U, sigma
}

// vecDot computes the dot product of two vectors.
func vecDot(a, b []float64) float64 {
	var sum float64
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}

// vecNorm2 computes the L2 norm of a vector.
func vecNorm2(v []float64) float64 {
	return math.Sqrt(vecDot(v, v))
}

// vecNormalize normalizes a vector in-place to unit length.
func vecNormalize(v []float64) {
	n := vecNorm2(v)
	if n < 1e-12 {
		return
	}
	for i := range v {
		v[i] /= n
	}
}

package dashboard

import (
	"math"
	"math/rand"
	"net/http"
	"sort"
	"strings"
)

// maxVecNodes limits nodes in the 3D graph for performance.
const maxVecNodes = 500

// topNeighbors is the max neighbors per node shown as links.
const topNeighbors = 4

// minSimilarity is the cosine similarity threshold for drawing a link.
const minSimilarity = 0.5

type vecNode struct {
	ID    string  `json:"id"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Z     float64 `json:"z"`
	Doc   string  `json:"doc"`
	Group string  `json:"group"`
	Label string  `json:"label"`
}

type vecLink struct {
	Source string  `json:"source"`
	Target string  `json:"target"`
	Value  float64 `json:"value"`
}

type vecEntry struct {
	key string
	vec []float32
}

func (s *Server) handleVectorsBuild(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := s.brain.RebuildFullIndex(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleVectors(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Nodes    []vecNode `json:"nodes"`
		Links    []vecLink `json:"links"`
		Provider string    `json:"provider"`
		Dims     int       `json:"dims"`
	}

	vi, err := s.brain.LoadVecIndex()
	if err != nil || vi == nil || len(vi.ChunkVecs) == 0 {
		writeJSON(w, response{Nodes: []vecNode{}, Links: []vecLink{}})
		return
	}

	entries := make([]vecEntry, 0, len(vi.ChunkVecs))
	for k, v := range vi.ChunkVecs {
		entries = append(entries, vecEntry{key: k, vec: v})
	}

	// Random sample when exceeding limit.
	rand.Shuffle(len(entries), func(i, j int) { entries[i], entries[j] = entries[j], entries[i] })
	if len(entries) > maxVecNodes {
		entries = entries[:maxVecNodes]
	}

	n := len(entries)
	dims := vi.Dims
	if dims == 0 && n > 0 {
		dims = len(entries[0].vec)
	}

	// Build float64 matrix for PCA.
	X := make([][]float64, n)
	for i, e := range entries {
		X[i] = make([]float64, dims)
		for j, f := range e.vec {
			if j < dims {
				X[i][j] = float64(f)
			}
		}
	}

	coords := pca3D(X, dims)
	scaleCoords(coords)

	nodes := make([]vecNode, n)
	for i, e := range entries {
		doc := chunkKeyDoc(e.key)
		nodes[i] = vecNode{
			ID:    e.key,
			X:     coords[i][0],
			Y:     coords[i][1],
			Z:     coords[i][2],
			Doc:   doc,
			Group: firstSegment(doc),
			Label: chunkKeyLabel(e.key),
		}
	}

	links := buildVecLinks(entries, topNeighbors, minSimilarity)

	writeJSON(w, response{
		Nodes:    nodes,
		Links:    links,
		Provider: vi.Provider,
		Dims:     dims,
	})
}

// pca3D reduces an n×dims matrix to n×3 using power iteration PCA.
func pca3D(X [][]float64, dims int) [][3]float64 {
	n := len(X)
	if n == 0 || dims == 0 {
		return make([][3]float64, n)
	}

	// Center the data.
	mean := make([]float64, dims)
	for _, row := range X {
		for j, v := range row {
			if j < dims {
				mean[j] += v
			}
		}
	}
	for j := range mean {
		mean[j] /= float64(n)
	}
	Xc := make([][]float64, n)
	for i, row := range X {
		Xc[i] = make([]float64, dims)
		for j := range Xc[i] {
			if j < len(row) {
				Xc[i][j] = row[j] - mean[j]
			}
		}
	}

	// Extract top 3 principal components via power iteration.
	k := 3
	if dims < 3 {
		k = dims
	}
	components := make([][]float64, k)
	for c := 0; c < k; c++ {
		v := make([]float64, dims)
		for j := range v {
			v[j] = math.Sin(float64(j+1) * math.Pi / float64(dims+1) * float64(c+1))
		}
		normVec64(v)

		for iter := 0; iter < 60; iter++ {
			Xv := make([]float64, n)
			for i, row := range Xc {
				for j, val := range row {
					Xv[i] += val * v[j]
				}
			}
			newV := make([]float64, dims)
			for i, row := range Xc {
				for j, val := range row {
					newV[j] += val * Xv[i]
				}
			}
			// Gram-Schmidt deflation against previous components.
			for pc := 0; pc < c; pc++ {
				d := dot64(newV, components[pc])
				for j := range newV {
					newV[j] -= d * components[pc][j]
				}
			}
			normVec64(newV)
			v = newV
		}
		components[c] = v
	}

	// Project all points onto the 3 components.
	result := make([][3]float64, n)
	for i, row := range Xc {
		for c := 0; c < k; c++ {
			for j, val := range row {
				result[i][c] += val * components[c][j]
			}
		}
	}
	return result
}

func scaleCoords(coords [][3]float64) {
	if len(coords) == 0 {
		return
	}
	var mn, mx [3]float64
	for i := range mn {
		mn[i] = math.MaxFloat64
		mx[i] = -math.MaxFloat64
	}
	for _, c := range coords {
		for i := range c {
			if c[i] < mn[i] {
				mn[i] = c[i]
			}
			if c[i] > mx[i] {
				mx[i] = c[i]
			}
		}
	}
	const target = 400.0
	for i := range coords {
		for j := range coords[i] {
			span := mx[j] - mn[j]
			if span < 1e-10 {
				coords[i][j] = 0
			} else {
				coords[i][j] = (coords[i][j]-mn[j])/span*target*2 - target
			}
		}
	}
}

func buildVecLinks(entries []vecEntry, topK int, minSim float64) []vecLink {
	n := len(entries)
	var links []vecLink
	type scored struct {
		j     int
		score float64
	}
	for i := 0; i < n; i++ {
		var sims []scored
		for j := i + 1; j < n; j++ {
			sc := cosSim32(entries[i].vec, entries[j].vec)
			if sc >= minSim {
				sims = append(sims, scored{j, sc})
			}
		}
		sort.Slice(sims, func(a, b int) bool { return sims[a].score > sims[b].score })
		if len(sims) > topK {
			sims = sims[:topK]
		}
		for _, s := range sims {
			links = append(links, vecLink{
				Source: entries[i].key,
				Target: entries[s.j].key,
				Value:  math.Round(s.score*1000) / 1000,
			})
		}
	}
	return links
}

// cosSim32 computes cosine similarity between two float32 unit vectors.
func cosSim32(a, b []float32) float64 {
	n := len(a)
	if n > len(b) {
		n = len(b)
	}
	var sum float64
	for i := 0; i < n; i++ {
		sum += float64(a[i]) * float64(b[i])
	}
	return sum
}

func normVec64(v []float64) {
	var sum float64
	for _, x := range v {
		sum += x * x
	}
	norm := math.Sqrt(sum)
	if norm < 1e-10 {
		return
	}
	for i := range v {
		v[i] /= norm
	}
}

func dot64(a, b []float64) float64 {
	var sum float64
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}

// chunkKeyDoc extracts the document path from a chunk key ("doc/path#chunkID").
func chunkKeyDoc(key string) string {
	if i := strings.LastIndex(key, "#"); i >= 0 {
		return key[:i]
	}
	return key
}

// chunkKeyLabel returns a short display label for a chunk key.
func chunkKeyLabel(key string) string {
	doc := chunkKeyDoc(key)
	parts := strings.Split(doc, "/")
	name := parts[len(parts)-1]
	name = strings.TrimSuffix(name, ".md")
	if i := strings.LastIndex(key, "#"); i >= 0 {
		return name + " #" + key[i+1:]
	}
	return name
}

// firstSegment returns the first path segment (category).
func firstSegment(path string) string {
	if i := strings.Index(path, "/"); i >= 0 {
		return path[:i]
	}
	return path
}

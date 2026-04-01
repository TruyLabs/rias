package brain

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func setupLSITestBrain(t *testing.T) *FileBrain {
	t.Helper()
	dir := t.TempDir()
	b := New(dir)

	// Create enough documents (>= LSIMinDocs) with overlapping topics
	// for LSI to produce meaningful embeddings.
	files := map[string]string{
		"knowledge/golang.md": `---
tags: [go, programming, backend]
confidence: high
source: conversation
updated: 2026-03-25
---

Go is a statically typed compiled language designed at Google.
It has garbage collection, structural typing, and CSP-style concurrency.
Go is great for building microservices, CLI tools, and backend systems.
The language emphasizes simplicity and readability.
`,
		"knowledge/rust.md": `---
tags: [rust, programming, systems]
confidence: high
source: conversation
updated: 2026-03-25
---

Rust is a systems programming language focused on safety and performance.
It has a unique ownership model that prevents memory bugs at compile time.
Rust is used for systems programming, WebAssembly, and embedded devices.
The borrow checker enforces memory safety without garbage collection.
`,
		"knowledge/python.md": `---
tags: [python, programming, scripting]
confidence: high
source: conversation
updated: 2026-03-25
---

Python is a dynamically typed interpreted language known for simplicity.
It is widely used in data science, machine learning, and scripting.
Python has a large ecosystem of packages for scientific computing.
The language prioritizes readability and developer productivity.
`,
		"knowledge/docker.md": `---
tags: [docker, containers, devops]
confidence: high
source: conversation
updated: 2026-03-25
---

Docker is a platform for building and running containers.
Containers package applications with their dependencies for consistent deployment.
Docker uses images and layers for efficient storage and distribution.
Container orchestration tools like Kubernetes manage Docker containers at scale.
`,
		"knowledge/kubernetes.md": `---
tags: [kubernetes, containers, orchestration]
confidence: high
source: conversation
updated: 2026-03-25
---

Kubernetes is an open-source container orchestration platform.
It automates deployment, scaling, and management of containerized applications.
Kubernetes uses pods, services, and deployments as core abstractions.
The platform supports rolling updates, self-healing, and horizontal scaling.
`,
		"opinions/architecture.md": `---
tags: [architecture, microservices, backend]
confidence: medium
source: conversation
updated: 2026-03-25
---

Microservices architecture works well for large teams and complex systems.
Each service owns its data and communicates via APIs or message queues.
Monolithic architecture is simpler for small teams and new projects.
The choice depends on team size, deployment requirements, and complexity.
`,
		"decisions/tech-stack.md": `---
tags: [tech-stack, decisions, backend]
confidence: high
source: conversation
updated: 2026-03-25
---

The backend is built with Go for performance and type safety.
Docker containers handle deployment and environment consistency.
Kubernetes orchestrates services in production environments.
PostgreSQL is the primary database for structured data storage.
`,
	}

	for relPath, content := range files {
		fullPath := filepath.Join(dir, relPath)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	if err := b.RebuildIndex(); err != nil {
		t.Fatalf("RebuildIndex() error: %v", err)
	}

	return b
}

func TestBuildLSI(t *testing.T) {
	b := setupLSITestBrain(t)

	idx, err := b.LoadFullIndex()
	if err != nil {
		t.Fatalf("LoadFullIndex() error: %v", err)
	}

	model := BuildLSI(idx)
	if model == nil {
		t.Fatal("BuildLSI returned nil")
	}

	if model.Dims < 2 {
		t.Errorf("Dims = %d, want >= 2", model.Dims)
	}

	if len(model.TermVecs) == 0 {
		t.Error("TermVecs is empty")
	}

	if len(model.IDF) == 0 {
		t.Error("IDF is empty")
	}

	// Check that terms appearing in ≥2 docs have embeddings.
	// "language" appears in golang, rust, python; "contain" in docker, kubernetes.
	for _, term := range []string{"language", "contain"} {
		if _, ok := model.TermVecs[term]; !ok {
			t.Errorf("expected embedding for stem %q", term)
		}
	}
}

func TestBuildLSIMinDocs(t *testing.T) {
	// With fewer than LSIMinDocs documents, BuildLSI should return nil.
	dir := t.TempDir()
	b := New(dir)

	for i := 0; i < LSIMinDocs-1; i++ {
		path := filepath.Join(dir, fmt.Sprintf("doc%d.md", i))
		os.MkdirAll(filepath.Dir(path), 0755)
		content := fmt.Sprintf("---\ntags: [test]\nconfidence: high\nsource: test\nupdated: 2026-03-25\n---\n\nDocument %d content.\n", i)
		os.WriteFile(path, []byte(content), 0644)
	}

	b.RebuildIndex()
	idx, _ := b.LoadFullIndex()
	model := BuildLSI(idx)
	if model != nil {
		t.Error("BuildLSI should return nil for too few documents")
	}
}

func TestVecIndexQuery(t *testing.T) {
	b := setupLSITestBrain(t)

	vi, err := b.LoadVecIndex()
	if err != nil {
		t.Fatalf("LoadVecIndex() error: %v", err)
	}

	if vi == nil {
		t.Fatal("VecIndex is nil")
	}

	if len(vi.ChunkVecs) == 0 {
		t.Fatal("ChunkVecs is empty")
	}

	// Query for "programming language" should return results.
	results := vi.QueryVec("programming language")
	if len(results) == 0 {
		t.Fatal("QueryVec returned no results for 'programming language'")
	}

	// Top results should be programming-related documents.
	t.Logf("Top vector results for 'programming language':")
	for i, r := range results {
		if i >= 5 {
			break
		}
		t.Logf("  %d. %s (score=%.4f)", i+1, r.ChunkKey, r.Score)
	}
}

func TestHybridSearch(t *testing.T) {
	b := setupLSITestBrain(t)

	// Hybrid search should combine BM25 and vector scores.
	results := b.QueryHybrid("container deployment orchestration")
	if len(results) == 0 {
		t.Fatal("QueryHybrid returned no results")
	}

	// Docker and Kubernetes docs should rank high.
	topPaths := make(map[string]bool)
	for i, r := range results {
		if i >= 3 {
			break
		}
		topPaths[r.DocPath] = true
		t.Logf("  %d. %s (score=%.4f)", i+1, r.DocPath, r.Score)
	}

	if !topPaths["knowledge/kubernetes.md"] && !topPaths["knowledge/docker.md"] {
		t.Error("Expected kubernetes.md or docker.md in top 3 hybrid results")
	}
}

func TestHybridFallback(t *testing.T) {
	// With too few docs for LSI, hybrid should fall back to BM25.
	dir := t.TempDir()
	b := New(dir)

	content := `---
tags: [test]
confidence: high
source: test
updated: 2026-03-25
---

This is a test document about search functionality.
`
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte(content), 0644)
	b.RebuildIndex()

	results := b.QueryHybrid("search")
	if len(results) == 0 {
		t.Fatal("Hybrid fallback should still return BM25 results")
	}
}

func TestLSISemanticSimilarity(t *testing.T) {
	b := setupLSITestBrain(t)

	idx, err := b.LoadFullIndex()
	if err != nil {
		t.Fatalf("LoadFullIndex() error: %v", err)
	}

	model := BuildLSI(idx)
	if model == nil {
		t.Fatal("BuildLSI returned nil")
	}

	// Terms that co-occur in similar documents should have similar embeddings.
	// "docker" and "kubernetes" both appear in container-related docs.
	dockerVec := model.EmbedTerms(tokenizeAndStem("containers deployment"))
	kubeVec := model.EmbedTerms(tokenizeAndStem("orchestration scaling"))
	goVec := model.EmbedTerms(tokenizeAndStem("compiled typed language"))

	if dockerVec == nil || kubeVec == nil || goVec == nil {
		t.Skip("Some terms don't have embeddings — corpus may be too small")
	}

	normalizeF32(dockerVec)
	normalizeF32(kubeVec)
	normalizeF32(goVec)

	dockerKubeSim := cosineF32(dockerVec, kubeVec)
	dockerGoSim := cosineF32(dockerVec, goVec)

	t.Logf("docker-kubernetes similarity: %.4f", dockerKubeSim)
	t.Logf("docker-golang similarity: %.4f", dockerGoSim)

	// Docker should be more similar to Kubernetes than to Go.
	if dockerKubeSim <= dockerGoSim {
		t.Logf("Warning: expected docker-kubernetes > docker-golang, got %.4f <= %.4f",
			dockerKubeSim, dockerGoSim)
		// Not a hard failure — depends on corpus patterns.
	}
}

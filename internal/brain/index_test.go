package brain

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestBrain(t *testing.T) (string, *FileBrain) {
	t.Helper()
	dir := t.TempDir()
	b := New(dir)

	files := map[string]string{
		"opinions/golang.md": `---
tags: [go, languages, backend]
confidence: high
source: conversation
updated: 2026-03-25
---

Kyle prefers Go for CLI tools.
`,
		"opinions/architecture.md": `---
tags: [architecture, microservices, backend]
confidence: medium
source: conversation
updated: 2026-03-25
---

Kyle prefers monorepos.
`,
		"identity/profile.md": `---
tags: [identity, background]
confidence: high
source: direct
updated: 2026-03-25
---

Kyle is a software engineer.
`,
	}

	for relPath, content := range files {
		fullPath := filepath.Join(dir, relPath)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	return dir, b
}

func TestRebuildIndex(t *testing.T) {
	_, b := setupTestBrain(t)

	if err := b.RebuildIndex(); err != nil {
		t.Fatalf("RebuildIndex() error: %v", err)
	}

	idx, err := b.LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex() error: %v", err)
	}

	// "go" tag should map to opinions/golang.md
	paths, ok := idx.Tags["go"]
	if !ok || len(paths) != 1 || paths[0] != "opinions/golang.md" {
		t.Errorf("Tags[go] = %v, want [opinions/golang.md]", paths)
	}

	// "backend" should map to both files
	paths = idx.Tags["backend"]
	if len(paths) != 2 {
		t.Errorf("Tags[backend] has %d entries, want 2", len(paths))
	}
}

func TestQueryIndex(t *testing.T) {
	_, b := setupTestBrain(t)
	b.RebuildIndex()

	results := b.QueryIndex([]string{"go", "backend"})

	// golang.md matches both tags (score 2), architecture.md matches one (score 1)
	if len(results) < 1 {
		t.Fatal("QueryIndex returned no results")
	}
	if results[0].Path != "opinions/golang.md" {
		t.Errorf("Top result = %q, want %q", results[0].Path, "opinions/golang.md")
	}
}

func TestRebuildFullIndex(t *testing.T) {
	_, b := setupTestBrain(t)

	if err := b.RebuildIndex(); err != nil {
		t.Fatalf("RebuildIndex() error: %v", err)
	}

	idx, err := b.LoadFullIndex()
	if err != nil {
		t.Fatalf("LoadFullIndex() error: %v", err)
	}

	if idx.TotalDocs != 3 {
		t.Errorf("TotalDocs = %d, want 3", idx.TotalDocs)
	}

	if _, ok := idx.Documents["opinions/golang.md"]; !ok {
		t.Error("expected opinions/golang.md in Documents")
	}
}

func TestQueryFullIndexContentSearch(t *testing.T) {
	_, b := setupTestBrain(t)
	b.RebuildIndex()

	// Search for content that exists in golang.md ("CLI tools")
	results := b.QueryFullIndex("CLI tools")
	if len(results) == 0 {
		t.Fatal("QueryFullIndex returned no results for 'CLI tools'")
	}
	if results[0].Path != "opinions/golang.md" {
		t.Errorf("Top result = %q, want %q", results[0].Path, "opinions/golang.md")
	}
}

func TestQueryFullIndexFieldBoosts(t *testing.T) {
	_, b := setupTestBrain(t)
	b.RebuildIndex()

	// "go" appears as a tag on golang.md (3x boost) and in content of profile.md (1x).
	// golang.md should rank higher due to tag boost.
	results := b.QueryFullIndex("go")
	if len(results) == 0 {
		t.Fatal("QueryFullIndex returned no results for 'go'")
	}
	if results[0].Path != "opinions/golang.md" {
		t.Errorf("Top result = %q, want %q (tag boost should rank it first)", results[0].Path, "opinions/golang.md")
	}
}

func TestQueryFullIndexMonorepos(t *testing.T) {
	_, b := setupTestBrain(t)
	b.RebuildIndex()

	// "monorepos" only appears in architecture.md content
	results := b.QueryFullIndex("monorepos")
	if len(results) == 0 {
		t.Fatal("QueryFullIndex returned no results for 'monorepos'")
	}
	if results[0].Path != "opinions/architecture.md" {
		t.Errorf("Top result = %q, want %q", results[0].Path, "opinions/architecture.md")
	}
}

func TestQueryFullIndexEmpty(t *testing.T) {
	_, b := setupTestBrain(t)
	b.RebuildIndex()

	results := b.QueryFullIndex("")
	if len(results) != 0 {
		t.Errorf("Empty query should return no results, got %d", len(results))
	}
}

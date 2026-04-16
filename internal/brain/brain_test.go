package brain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadBrainFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "golang.md")
	content := `---
tags: [go, languages, preferences]
confidence: high
source: conversation
updated: 2026-03-25
---

Kyle prefers Go for CLI tools and backend services.
`
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	b := New(dir)
	bf, err := b.Load("golang.md")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if bf.Path != "golang.md" {
		t.Errorf("Path = %q, want %q", bf.Path, "golang.md")
	}
	if len(bf.Tags) != 3 || bf.Tags[0] != "go" {
		t.Errorf("Tags = %v, want [go languages preferences]", bf.Tags)
	}
	if bf.Confidence != "high" {
		t.Errorf("Confidence = %q, want %q", bf.Confidence, "high")
	}
	if bf.Source != "conversation" {
		t.Errorf("Source = %q, want %q", bf.Source, "conversation")
	}
	expected := "\nKyle prefers Go for CLI tools and backend services.\n"
	if bf.Content != expected {
		t.Errorf("Content = %q, want %q", bf.Content, expected)
	}
}

func TestSaveBrainFile(t *testing.T) {
	dir := t.TempDir()
	b := New(dir)

	bf := &BrainFile{
		Path:       "opinions/testing.md",
		Tags:       []string{"testing", "tdd"},
		Confidence: "medium",
		Source:     "conversation",
		Updated:    DateOnly{time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)},
		Content:    "\nKyle likes TDD for business logic.\n",
	}

	if err := b.Save(bf); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	savedPath := filepath.Join(dir, "opinions", "testing.md")
	data, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "testing") || !strings.Contains(content, "tdd") {
		t.Errorf("saved file missing tags, got:\n%s", content)
	}
	if !strings.Contains(content, "Kyle likes TDD") {
		t.Errorf("saved file missing content, got:\n%s", content)
	}
}

func TestDefaultCategoriesIncludePersonalizationCategories(t *testing.T) {
	required := []string{"expertise", "goals"}
	for _, cat := range required {
		found := false
		for _, c := range DefaultCategories {
			if c == cat {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("DefaultCategories missing required category: %q", cat)
		}
	}
}

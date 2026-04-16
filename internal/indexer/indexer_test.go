package indexer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TruyLabs/rias/internal/indexer"
)

func TestIndexRepo(t *testing.T) {
	// Create a temporary fake repo with one Go file
	repoDir := t.TempDir()
	brainDir := t.TempDir()

	goSrc := `package service

type Order struct {
	ID string
}

func CreateOrder(id string) *Order {
	return &Order{ID: id}
}
`
	srcFile := filepath.Join(repoDir, "order.go")
	if err := os.WriteFile(srcFile, []byte(goSrc), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	result, err := indexer.IndexRepo(repoDir, brainDir)
	if err != nil {
		t.Fatalf("IndexRepo: %v", err)
	}
	if result.Indexed == 0 {
		t.Error("expected at least one file indexed")
	}

	// Brain file should exist at brain/knowledge/repos/<reponame>/order.md
	repoName := filepath.Base(repoDir)
	brainFile := filepath.Join(brainDir, "knowledge", "repos", repoName, "order.md")
	data, err := os.ReadFile(brainFile)
	if err != nil {
		t.Fatalf("expected brain file at %s: %v", brainFile, err)
	}

	content := string(data)
	if !strings.Contains(content, "Order") {
		t.Errorf("expected brain file to contain 'Order', got: %s", content)
	}
	if !strings.Contains(content, "CreateOrder") {
		t.Errorf("expected brain file to contain 'CreateOrder', got: %s", content)
	}
}

func TestIndexRepoSkipsUnchangedFiles(t *testing.T) {
	repoDir := t.TempDir()
	brainDir := t.TempDir()

	goSrc := `package main
func Hello() string { return "hi" }
`
	srcFile := filepath.Join(repoDir, "main.go")
	if err := os.WriteFile(srcFile, []byte(goSrc), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	r1, err := indexer.IndexRepo(repoDir, brainDir)
	if err != nil {
		t.Fatalf("first IndexRepo: %v", err)
	}
	if r1.Indexed != 1 {
		t.Errorf("expected 1 indexed on first run, got %d", r1.Indexed)
	}

	r2, err := indexer.IndexRepo(repoDir, brainDir)
	if err != nil {
		t.Fatalf("second IndexRepo: %v", err)
	}
	if r2.Indexed != 0 {
		t.Errorf("expected 0 indexed on second run (unchanged), got %d", r2.Indexed)
	}
	if r2.Skipped != 1 {
		t.Errorf("expected 1 skipped on second run, got %d", r2.Skipped)
	}
}

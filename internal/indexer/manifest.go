package indexer

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

// RepoManifest tracks file hashes for a single indexed repository.
// Stored at brain/knowledge/repos/<name>/.manifest.json.
type RepoManifest struct {
	Files map[string]string `json:"files"` // relative file path → SHA-256 hash
}

// LoadRepoManifest reads a manifest from disk.
// Returns an empty manifest (not an error) when the file is missing.
func LoadRepoManifest(path string) (*RepoManifest, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &RepoManifest{Files: make(map[string]string)}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read repo manifest: %w", err)
	}
	var m RepoManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return &RepoManifest{Files: make(map[string]string)}, nil
	}
	if m.Files == nil {
		m.Files = make(map[string]string)
	}
	return &m, nil
}

// SaveRepoManifest writes a manifest to disk atomically.
func SaveRepoManifest(path string, m *RepoManifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal repo manifest: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("write repo manifest: %w", err)
	}
	return os.Rename(tmp, path)
}

// FileHash returns a short SHA-256 hex hash of the file at path.
func FileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash %s: %w", path, err)
	}
	return fmt.Sprintf("%x", h.Sum(nil)[:12]), nil
}

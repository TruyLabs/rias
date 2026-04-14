package brain

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ManifestFileName is the on-disk file for the incremental index manifest.
const ManifestFileName = "index_manifest.json.gz"

// FileState records the last-indexed state of a single brain file.
type FileState struct {
	Hash   string `json:"h"` // SHA-256 (24 hex chars) of the full file content
	Chunks int    `json:"c"` // Number of chunks indexed
}

// IndexManifest maps relative file paths to their last-indexed state.
// Used to detect which files changed since the last index build.
type IndexManifest struct {
	Files map[string]FileState `json:"files"`
}

// fileContentHash returns a short SHA-256 hex hash of the file at absPath.
func fileContentHash(absPath string) (string, error) {
	f, err := os.Open(absPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash %s: %w", absPath, err)
	}
	return fmt.Sprintf("%x", h.Sum(nil)[:12]), nil // 24 hex chars
}

// loadManifest reads the index manifest from disk.
// Returns an empty manifest (not an error) when the file is missing or corrupt —
// this triggers a full rebuild on the first incremental run.
func (b *FileBrain) loadManifest() *IndexManifest {
	path := filepath.Join(b.root, ManifestFileName)
	f, err := os.Open(path)
	if err != nil {
		return &IndexManifest{Files: make(map[string]FileState)}
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return &IndexManifest{Files: make(map[string]FileState)}
	}
	defer gz.Close()

	var m IndexManifest
	if err := json.NewDecoder(gz).Decode(&m); err != nil {
		return &IndexManifest{Files: make(map[string]FileState)}
	}
	if m.Files == nil {
		m.Files = make(map[string]FileState)
	}
	return &m
}

// saveManifest writes the index manifest to disk atomically.
func (b *FileBrain) saveManifest(m *IndexManifest) error {
	path := filepath.Join(b.root, ManifestFileName)
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	tmpPath := path + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create manifest tmp: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewWriterLevel(f, gzip.DefaultCompression)
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("create gzip writer: %w", err)
	}
	if _, err := gz.Write(data); err != nil {
		gz.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write manifest: %w", err)
	}
	if err := gz.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close manifest gzip: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close manifest file: %w", err)
	}
	return os.Rename(tmpPath, path)
}

// buildAndSaveManifest hashes all listed files and saves a fresh manifest.
// Called after a full rebuild so the next run can use incremental mode.
func (b *FileBrain) buildAndSaveManifest(files []string, chunkCounts map[string]int) error {
	m := &IndexManifest{Files: make(map[string]FileState, len(files))}
	for _, f := range files {
		absPath := filepath.Join(b.root, f)
		hash, err := fileContentHash(absPath)
		if err != nil {
			continue
		}
		m.Files[f] = FileState{Hash: hash, Chunks: chunkCounts[f]}
	}
	return b.saveManifest(m)
}

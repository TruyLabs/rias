package importer

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ImportManifest tracks which conversation IDs have been processed.
// Stored as JSON at ~/.rias/import_manifest.json.
type ImportManifest struct {
	Processed map[string]string `json:"processed"` // ID → content hash
}

// LoadManifest reads the manifest from disk.
// Returns an empty manifest (not an error) when the file is missing.
func LoadManifest(path string) (*ImportManifest, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &ImportManifest{Processed: make(map[string]string)}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read import manifest: %w", err)
	}
	var m ImportManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return &ImportManifest{Processed: make(map[string]string)}, nil
	}
	if m.Processed == nil {
		m.Processed = make(map[string]string)
	}
	return &m, nil
}

// SaveManifest writes the manifest to disk atomically.
func SaveManifest(path string, m *ImportManifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal import manifest: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("write import manifest: %w", err)
	}
	return os.Rename(tmp, path)
}

// ConversationHash returns a short SHA-256 hash of a conversation's content.
// Used to detect whether a previously-processed conversation has changed.
func ConversationHash(c Conversation) string {
	var sb strings.Builder
	for _, m := range c.Messages {
		sb.WriteString(m.Role)
		sb.WriteString(":")
		sb.WriteString(m.Content)
		sb.WriteString("\n")
	}
	sum := sha256.Sum256([]byte(sb.String()))
	return fmt.Sprintf("%x", sum[:8])
}

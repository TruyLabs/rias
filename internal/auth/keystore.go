package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	credentialsDirPerm  = 0700
	credentialsFilePerm = 0600
)

// Keystore manages API key storage on disk.
type Keystore struct {
	path string
	mu   sync.Mutex
}

// NewKeystore creates a Keystore at the given path.
func NewKeystore(path string) *Keystore {
	return &Keystore{path: path}
}

// SaveKey stores an API key for a provider.
func (ks *Keystore) SaveKey(provider, key string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	creds, err := ks.loadAll()
	if err != nil {
		return fmt.Errorf("load existing credentials: %w", err)
	}
	creds[provider] = key

	if err := os.MkdirAll(filepath.Dir(ks.path), credentialsDirPerm); err != nil {
		return fmt.Errorf("create credentials dir: %w", err)
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}

	tmpPath := ks.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, credentialsFilePerm); err != nil {
		return fmt.Errorf("write credentials: %w", err)
	}
	return os.Rename(tmpPath, ks.path)
}

// GetKey retrieves an API key for a provider.
func (ks *Keystore) GetKey(provider string) (string, error) {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	creds, err := ks.loadAll()
	if err != nil {
		return "", fmt.Errorf("load credentials: %w", err)
	}
	key, ok := creds[provider]
	if !ok {
		return "", fmt.Errorf("no credentials for provider: %s", provider)
	}
	return key, nil
}

func (ks *Keystore) loadAll() (map[string]string, error) {
	data, err := os.ReadFile(ks.path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("read credentials file: %w", err)
	}
	var creds map[string]string
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("parse credentials file (may be corrupted): %w", err)
	}
	return creds, nil
}

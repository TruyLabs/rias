package auth

import (
	"path/filepath"
	"testing"
)

func TestSaveAndLoadAPIKey(t *testing.T) {
	dir := t.TempDir()
	ks := NewKeystore(filepath.Join(dir, "credentials.json"))

	if err := ks.SaveKey("claude", "sk-ant-test-123"); err != nil {
		t.Fatalf("SaveKey() error: %v", err)
	}

	key, err := ks.GetKey("claude")
	if err != nil {
		t.Fatalf("GetKey() error: %v", err)
	}
	if key != "sk-ant-test-123" {
		t.Errorf("key = %q, want %q", key, "sk-ant-test-123")
	}
}

func TestGetKeyNotFound(t *testing.T) {
	dir := t.TempDir()
	ks := NewKeystore(filepath.Join(dir, "credentials.json"))

	_, err := ks.GetKey("nonexistent")
	if err == nil {
		t.Error("GetKey() should error for unknown provider")
	}
}

func TestManagerGetCredential(t *testing.T) {
	dir := t.TempDir()
	ks := NewKeystore(filepath.Join(dir, "credentials.json"))
	ks.SaveKey("claude", "sk-from-keystore")

	m := NewManager(ks)

	key, err := m.GetCredential("claude")
	if err != nil {
		t.Fatalf("GetCredential() error: %v", err)
	}
	if key != "sk-from-keystore" {
		t.Errorf("key = %q, want %q", key, "sk-from-keystore")
	}
}

package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tinhvqbk/kai/internal/provider"
)

const (
	sessionIDTimeFormat = "2006-01-02T15-04-05"
	sessionIDSuffixLen  = 4
	sessionDirPerm      = 0755
	sessionFilePerm     = 0644
)

// Session represents a conversation session.
type Session struct {
	ID                 string           `json:"id"`
	StartedAt          time.Time        `json:"started_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
	Provider           string           `json:"provider"`
	Messages           []SessionMessage `json:"messages"`
	LearningsExtracted int              `json:"learnings_extracted"`
}

// SessionMessage wraps a message with metadata.
type SessionMessage struct {
	provider.Message
	Timestamp      time.Time `json:"timestamp"`
	BrainFilesUsed []string  `json:"brain_files_used,omitempty"`
	Confidence     string    `json:"confidence,omitempty"`
}

// Manager handles session persistence.
type Manager struct {
	dir string
}

// NewManager creates a session Manager.
func NewManager(dir string) *Manager {
	return &Manager{dir: dir}
}

// New creates a new session.
func (m *Manager) New(providerName string) *Session {
	now := time.Now().UTC()
	suffix := make([]byte, sessionIDSuffixLen)
	rand.Read(suffix)
	return &Session{
		ID:        now.Format(sessionIDTimeFormat) + "-" + hex.EncodeToString(suffix),
		StartedAt: now,
		UpdatedAt: now,
		Provider:  providerName,
		Messages:  []SessionMessage{},
	}
}

// Save writes a session to disk as JSON.
func (m *Manager) Save(s *Session) error {
	if err := os.MkdirAll(m.dir, sessionDirPerm); err != nil {
		return fmt.Errorf("create sessions dir: %w", err)
	}

	s.UpdatedAt = time.Now().UTC()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	path := filepath.Join(m.dir, s.ID+".json")
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, sessionFilePerm); err != nil {
		return fmt.Errorf("write session: %w", err)
	}
	return os.Rename(tmpPath, path)
}

// Load reads a session from disk.
func (m *Manager) Load(path string) (*Session, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read session: %w", err)
	}

	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse session: %w", err)
	}
	return &s, nil
}

// AddMessage appends a message to the session.
func (s *Session) AddMessage(msg SessionMessage) {
	s.Messages = append(s.Messages, msg)
}

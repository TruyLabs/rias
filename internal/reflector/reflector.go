package reflector

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/TruyLabs/rias/internal/provider"
	"github.com/TruyLabs/rias/internal/session"
)

// ParseSinceDuration parses a "Nd" or "Nw" duration string.
// Supported: "7d" (7 days), "30d" (30 days), "1w" (1 week), "2w" (2 weeks).
func ParseSinceDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, fmt.Errorf("invalid duration %q: expected Nd or Nw (e.g. 7d, 1w)", s)
	}
	if strings.HasSuffix(s, "w") {
		n, err := strconv.Atoi(s[:len(s)-1])
		if err != nil || n <= 0 {
			return 0, fmt.Errorf("invalid duration %q: expected positive Nw (e.g. 1w, 2w)", s)
		}
		return time.Duration(n) * 7 * 24 * time.Hour, nil
	}
	if strings.HasSuffix(s, "d") {
		n, err := strconv.Atoi(s[:len(s)-1])
		if err != nil || n <= 0 {
			return 0, fmt.Errorf("invalid duration %q: expected positive Nd (e.g. 7d, 30d)", s)
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}
	return 0, fmt.Errorf("invalid duration %q: expected Nd or Nw (e.g. 7d, 1w)", s)
}

// LoadRecentSessions reads session JSON files from sessionsPath.
// If cutoff is zero, all sessions are returned.
// Sessions are returned sorted oldest-first by session ID.
// Missing sessionsPath returns an empty slice, not an error.
func LoadRecentSessions(sessionsPath string, cutoff time.Time) ([]*session.Session, error) {
	entries, err := os.ReadDir(sessionsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read sessions dir: %w", err)
	}

	mgr := session.NewManager(sessionsPath)
	var sessions []*session.Session

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		if !cutoff.IsZero() {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.ModTime().Before(cutoff) {
				continue
			}
		}
		s, err := mgr.Load(filepath.Join(sessionsPath, entry.Name()))
		if err != nil {
			continue // skip malformed session files silently
		}
		sessions = append(sessions, s)
	}

	// Sort oldest-first by session ID (IDs start with timestamp "2006-01-02T...")
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].ID < sessions[j].ID
	})

	return sessions, nil
}

// SessionsToMessages flattens multiple sessions into a single ordered provider.Message slice.
// Empty-content messages are skipped.
func SessionsToMessages(sessions []*session.Session) []provider.Message {
	var msgs []provider.Message
	for _, s := range sessions {
		for _, sm := range s.Messages {
			if sm.Content == "" {
				continue
			}
			msgs = append(msgs, provider.Message{
				Role:    sm.Role,
				Content: sm.Content,
			})
		}
	}
	return msgs
}

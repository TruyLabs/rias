package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TruyLabs/rias/internal/brain"
	"github.com/TruyLabs/rias/internal/config"
	"github.com/TruyLabs/rias/internal/session"
	"github.com/spf13/cobra"
)

const feedbackFilePath = "feedback/feedback.md"

func newFeedbackCmd() *cobra.Command {
	var note string

	cmd := &cobra.Command{
		Use:   "feedback [good|bad]",
		Short: "Rate the last response to improve future recall",
		Args:  cobra.ExactArgs(1),
		Example: `  rias feedback good
  rias feedback bad --note "missed the point about error handling"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			rating := strings.ToLower(args[0])
			if rating != "good" && rating != "bad" {
				return fmt.Errorf("rating must be 'good' or 'bad', got %q", args[0])
			}
			return runFeedback(rating, note)
		},
	}
	cmd.Flags().StringVar(&note, "note", "", "Optional note explaining the rating")
	return cmd
}

func runFeedback(rating, note string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	sessPath := config.ExpandPath(cfg.SessionsPath)
	entries, err := os.ReadDir(sessPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("No sessions found. Start chatting with 'rias' first.")
			return nil
		}
		return fmt.Errorf("read sessions dir: %w", err)
	}

	// Find most recent session file (session IDs are timestamp-prefixed, so max == most recent)
	var lastFile string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") && e.Name() > lastFile {
			lastFile = e.Name()
		}
	}
	if lastFile == "" {
		fmt.Println("No sessions found. Start chatting with 'rias' first.")
		return nil
	}

	mgr := session.NewManager(sessPath)
	sess, err := mgr.Load(filepath.Join(sessPath, lastFile))
	if err != nil {
		return fmt.Errorf("load session: %w", err)
	}

	// Build feedback entry
	date := time.Now().Format("2006-01-02")
	var entry strings.Builder
	entry.WriteString(fmt.Sprintf("\n## %s — %s (session: %s)\n\n", date, rating, sess.ID))
	if note != "" {
		entry.WriteString(fmt.Sprintf("*%s*\n\n", note))
	}
	msgs := sess.Messages
	if len(msgs) >= 2 {
		last := msgs[len(msgs)-2:]
		entry.WriteString("Last exchange:\n")
		for _, m := range last {
			content := strings.TrimSpace(m.Content)
			if len(content) > 200 {
				content = content[:200] + "..."
			}
			entry.WriteString(fmt.Sprintf("**%s:** %s\n", m.Role, content))
		}
	}

	brainPath := getBrainPath()
	b := brain.New(brainPath)

	bf, err := b.Load(feedbackFilePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("load feedback file: %w", err)
		}
		bf = &brain.BrainFile{
			Path:       feedbackFilePath,
			Tags:       []string{"feedback"},
			Confidence: brain.ConfidenceHigh,
			Source:     "feedback-cmd",
			Updated:    brain.DateOnly{Time: time.Now()},
			Content:    "",
		}
	}
	bf.Content = strings.TrimRight(bf.Content, "\n") + entry.String() + "\n"
	bf.Updated = brain.DateOnly{Time: time.Now()}

	if err := b.Save(bf); err != nil {
		return fmt.Errorf("save feedback: %w", err)
	}
	if err := b.RebuildTagIndex(); err != nil {
		fmt.Printf("  warning: tag index rebuild: %v\n", err)
	}
	if err := b.RebuildIndex(); err != nil {
		fmt.Printf("  warning: index rebuild: %v\n", err)
	}

	symbol := "+"
	if rating == "bad" {
		symbol = "-"
	}
	fmt.Printf("[%s] Feedback recorded for session %s\n", symbol, sess.ID)
	return nil
}

package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/TruyLabs/rias/internal/brain"
	"github.com/TruyLabs/rias/internal/prompt"
	"github.com/TruyLabs/rias/internal/provider"
	"github.com/TruyLabs/rias/internal/reflector"
	"github.com/spf13/cobra"
)

// maxReflectMessages is the maximum number of messages sent to the LLM for reflection.
// Large session histories are truncated to the most recent messages.
const maxReflectMessages = 80

func newReflectCmd() *cobra.Command {
	var sinceFlag string

	cmd := &cobra.Command{
		Use:   "reflect",
		Short: "Analyze session history to extract behavioral patterns into brain",
		Example: `  rias reflect              # analyze all sessions
  rias reflect --since 7d   # only last 7 days
  rias reflect --since 2w   # only last 2 weeks`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReflect(sinceFlag)
		},
	}
	cmd.Flags().StringVar(&sinceFlag, "since", "", "Only analyze sessions from the last N days/weeks (e.g. 7d, 2w)")
	return cmd
}

func runReflect(since string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Determine cutoff time from --since flag
	var cutoff time.Time
	if since != "" {
		d, err := reflector.ParseSinceDuration(since)
		if err != nil {
			return fmt.Errorf("invalid --since value: %w", err)
		}
		cutoff = time.Now().Add(-d)
	}

	sessions, err := reflector.LoadRecentSessions(cfg.SessionsPath, cutoff)
	if err != nil {
		return fmt.Errorf("load sessions: %w", err)
	}
	if len(sessions) == 0 {
		fmt.Println("No sessions found. Start chatting with 'rias ask' to build session history.")
		return nil
	}

	msgs := reflector.SessionsToMessages(sessions)
	if len(msgs) == 0 {
		fmt.Println("Sessions contain no messages to analyze.")
		return nil
	}

	// Truncate to most recent messages to stay within LLM context
	if len(msgs) > maxReflectMessages {
		msgs = msgs[len(msgs)-maxReflectMessages:]
	}

	fmt.Printf("Analyzing %d sessions (%d messages)...\n", len(sessions), len(msgs))

	// Load existing style/ and opinions/ to avoid duplicating known entries
	brainPath := getBrainPath()
	b := brain.New(brainPath)
	var contextFiles []*brain.BrainFile
	for _, dir := range []string{"style", "opinions"} {
		dirPath := filepath.Join(brainPath, dir)
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			continue // directory may not exist yet
		}
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
				continue
			}
			bf, err := b.Load(dir + "/" + entry.Name())
			if err == nil {
				contextFiles = append(contextFiles, bf)
			}
		}
	}

	// Build LLM provider
	_, _, p, _, err := buildRouter(cfg)
	if err != nil {
		return fmt.Errorf("build router: %w", err)
	}
	pb := prompt.NewBuilder(cfg.AgentName(), cfg.UserName())

	reflectPrompt := pb.BuildReflectPrompt(contextFiles, msgs)
	resp, err := p.Chat(context.Background(), "", []provider.Message{
		{Role: "user", Content: reflectPrompt},
	})
	if err != nil {
		return fmt.Errorf("LLM reflection: %w", err)
	}

	learnings, err := brain.ParseLearnings(resp.Content)
	if err != nil {
		return fmt.Errorf("parse learnings: %w", err)
	}

	if len(learnings) == 0 {
		fmt.Println("No new patterns found in sessions.")
		return nil
	}

	for i := range learnings {
		learnings[i].Source = "reflect"
	}

	if err := b.Learn(learnings); err != nil {
		return fmt.Errorf("write learnings to brain: %w", err)
	}
	if err := b.RebuildTagIndex(); err != nil {
		fmt.Printf("  Warning: tag index rebuild failed: %v\n", err)
	}
	if err := b.RebuildIndex(); err != nil {
		fmt.Printf("  Warning: index rebuild failed: %v\n", err)
	}

	fmt.Printf("Extracted %d patterns from %d sessions.\n", len(learnings), len(sessions))
	for _, l := range learnings {
		fmt.Printf("  Saved to brain/%s/%s.md\n", l.Category, l.Topic)
	}
	return nil
}

package cli

import (
	"context"
	"fmt"

	"github.com/norenis/kai/internal/brain"
	"github.com/norenis/kai/internal/config"
	bsync "github.com/norenis/kai/internal/sync"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync brain to git",
	}

	cmd.AddCommand(newSyncInitCmd())
	cmd.AddCommand(newSyncPushCmd())
	cmd.AddCommand(newSyncPullCmd())
	cmd.AddCommand(newSyncStatusCmd())

	return cmd
}

func newSyncInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize git sync",
		RunE: func(cmd *cobra.Command, args []string) error {
			syncer, err := buildSyncer()
			if err != nil {
				return err
			}
			if !syncer.HasBackends() {
				fmt.Println("No sync backends configured. Enable git or cloud sync in config.yaml.")
				return nil
			}
			return syncer.Init(context.Background())
		},
	}
}

func newSyncPushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push brain to git",
		RunE: func(cmd *cobra.Command, args []string) error {
			syncer, err := buildSyncer()
			if err != nil {
				return err
			}
			if !syncer.HasBackends() {
				fmt.Println("No sync backends configured.")
				return nil
			}
			return syncer.Push(context.Background(), false, false)
		},
	}
	return cmd
}

func newSyncPullCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull brain from git",
		RunE: func(cmd *cobra.Command, args []string) error {
			syncer, err := buildSyncer()
			if err != nil {
				return err
			}
			if !syncer.HasBackends() {
				fmt.Println("No sync backends configured.")
				return nil
			}
			if err := syncer.Pull(context.Background(), false, false); err != nil {
				return err
			}

			// Rebuild brain index after pull.
			b := brain.New(getBrainPath())
			if err := b.RebuildIndex(); err != nil {
				fmt.Printf("Warning: failed to rebuild index: %v\n", err)
			} else {
				fmt.Println("Brain index rebuilt.")
			}
			return nil
		},
	}
	return cmd
}

func newSyncStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show sync status (local vs remote diff)",
		RunE: func(cmd *cobra.Command, args []string) error {
			syncer, err := buildSyncer()
			if err != nil {
				return err
			}
			if !syncer.HasBackends() {
				fmt.Println("No sync backends configured.")
				return nil
			}

			gitStatus, cloudStatus, err := syncer.StatusAll(context.Background())
			if err != nil {
				return err
			}

			if gitStatus != nil {
				printStatus("Git", gitStatus)
			}
			if cloudStatus != nil {
				printStatus("Cloud", cloudStatus)
			}
			if gitStatus == nil && cloudStatus == nil {
				fmt.Println("No backends available for status.")
			}
			return nil
		},
	}
}

func printStatus(name string, s *bsync.Status) {
	fmt.Printf("\n%s sync status:\n", name)
	if len(s.LocalOnly) == 0 && len(s.RemoteOnly) == 0 && len(s.Modified) == 0 {
		fmt.Println("  Everything in sync.")
		return
	}
	for _, f := range s.LocalOnly {
		fmt.Printf("  + %s (local only)\n", f)
	}
	for _, f := range s.RemoteOnly {
		fmt.Printf("  - %s (remote only)\n", f)
	}
	for _, f := range s.Modified {
		fmt.Printf("  ~ %s (modified)\n", f)
	}
}

func buildSyncer() (*bsync.Syncer, error) {
	cfg, err := loadConfig()
	if err != nil {
		// Use defaults if no config.
		cfg = config.Defaults()
	}

	brainPath := config.ExpandPath(cfg.Brain.Path)
	if brainPath == "" {
		brainPath = config.ExpandPath(config.DefaultBrainPath)
	}

	var gitBackend bsync.Backend
	if cfg.Brain.Sync.Git.Enabled {
		gitBackend = bsync.NewGitBackend(
			brainPath,
			cfg.Brain.Sync.Git.Remote,
			cfg.Brain.Sync.Git.Branch,
		)
	}

	return bsync.NewSyncer(brainPath, gitBackend, nil), nil
}

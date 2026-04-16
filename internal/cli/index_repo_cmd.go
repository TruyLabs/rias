package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/TruyLabs/rias/internal/brain"
	"github.com/TruyLabs/rias/internal/indexer"
	"github.com/spf13/cobra"
)

func newIndexRepoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "index-repo <path>",
		Short: "Index a code repository into brain/knowledge/repos/",
		Args:  cobra.ExactArgs(1),
		Example: `  rias index-repo ~/Documents/code/personal/super-app
  rias index-repo ~/Documents/code/personal/badminton-club-app`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runIndexRepo(args[0])
		},
	}
}

func runIndexRepo(repoPath string) error {
	// Resolve to absolute path
	absRepo, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("resolve repo path: %w", err)
	}
	if _, err := os.Stat(absRepo); err != nil {
		return fmt.Errorf("repo path not found: %w", err)
	}

	brainPath := getBrainPath()

	// Ensure repos directory exists
	reposDir := filepath.Join(brainPath, "knowledge", "repos")
	if err := os.MkdirAll(reposDir, brain.DirPermissions); err != nil {
		return fmt.Errorf("create repos dir: %w", err)
	}

	repoName := filepath.Base(absRepo)
	fmt.Printf("Indexing %s into brain/knowledge/repos/%s/...\n", absRepo, repoName)

	result, err := indexer.IndexRepo(absRepo, brainPath)
	if err != nil {
		return fmt.Errorf("index repo: %w", err)
	}

	fmt.Printf("Done. Indexed: %d | Skipped (unchanged): %d\n", result.Indexed, result.Skipped)

	if result.Indexed > 0 {
		b := brain.New(brainPath)
		if err := b.RebuildIndex(); err != nil {
			fmt.Printf("  ⚠ Index rebuild failed: %v\n", err)
		} else {
			fmt.Println("  ✓ Brain index rebuilt.")
		}
	}

	return nil
}

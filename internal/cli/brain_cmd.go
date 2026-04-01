package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tinhvqbk/kai/internal/brain"
	"github.com/tinhvqbk/kai/internal/config"
	"github.com/spf13/cobra"
)

const defaultEditor = "vi"

func newBrainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "brain",
		Short: "Browse and manage brain knowledge",
		RunE: func(cmd *cobra.Command, args []string) error {
			b := brain.New(getBrainPath())
			files, err := b.ListAll()
			if err != nil {
				return fmt.Errorf("list brain files: %w", err)
			}

			if len(files) == 0 {
				fmt.Println("Brain is empty. Use 'kai teach' to start teaching.")
				return nil
			}

			fmt.Printf("Brain contains %d files:\n\n", len(files))
			for _, f := range files {
				bf, err := b.Load(f)
				if err != nil {
					fmt.Printf("  %s (error loading)\n", f)
					continue
				}
				tags := strings.Join(bf.Tags, ", ")
				fmt.Printf("  %s [%s] (%s)\n", f, tags, bf.Confidence)
			}
			return nil
		},
	}

	cmd.AddCommand(newBrainSearchCmd())
	cmd.AddCommand(newBrainEditCmd())
	cmd.AddCommand(newBrainReorganizeCmd())
	cmd.AddCommand(newSyncCmd())

	return cmd
}

func newBrainSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search [query]",
		Short: "Search brain by keyword/tag with full-text relevance scoring",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			b := brain.New(getBrainPath())

			type result struct {
				path    string
				score   float64
				preview string
			}

			// Full-text TF-IDF search with fallback to tag-based
			type candidate struct {
				path  string
				score float64
			}
			var candidates []candidate

			fullResults := b.QueryFullIndex(query)
			if len(fullResults) > 0 {
				for _, fr := range fullResults {
					candidates = append(candidates, candidate{path: fr.Path, score: fr.Score})
				}
			} else {
				keywords := strings.Fields(strings.ToLower(query))
				tagResults := b.QueryIndex(keywords)
				for _, tr := range tagResults {
					candidates = append(candidates, candidate{path: tr.Path, score: float64(tr.Score)})
				}
			}

			if len(candidates) == 0 {
				fmt.Println("No matching brain files found.")
				return nil
			}

			var results []result
			for _, c := range candidates {
				bf, err := b.Load(c.path)
				if err != nil {
					continue
				}
				finalScore := brain.RelevanceScore(bf, c.score)
				preview := ""
				content := strings.TrimSpace(bf.Content)
				lines := strings.SplitN(content, "\n", 2)
				if len(lines) > 0 && lines[0] != "" {
					preview = lines[0]
				}
				results = append(results, result{path: c.path, score: finalScore, preview: preview})
			}

			sort.Slice(results, func(i, j int) bool {
				return results[i].score > results[j].score
			})

			fmt.Printf("Found %d matching files:\n\n", len(results))
			for _, r := range results {
				fmt.Printf("  %s (score: %.2f)\n", r.path, r.score)
				if r.preview != "" {
					fmt.Printf("    %s\n", r.preview)
				}
				fmt.Println()
			}
			return nil
		},
	}
}

func newBrainEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit [file]",
		Short: "Open a brain file in $EDITOR",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = defaultEditor
			}

			relPath := args[0]
			if strings.Contains(relPath, "..") || filepath.IsAbs(relPath) {
				return fmt.Errorf("invalid path: must be relative within brain directory")
			}
			path := filepath.Join(getBrainPath(), relPath)
			c := exec.Command(editor, path)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		},
	}
}

func newBrainReorganizeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reorganize",
		Short: "Analyze and reorganize brain: deduplicate, recategorize, and consolidate",
		Long: `Analyzes brain files for:
  - Duplicates: files with highly similar content (proposes merge)
  - Miscategorizations: files that fit better in another category (proposes move)
  - Consolidation: small related files that should be combined (proposes consolidate)

By default runs in dry-run mode. Use --apply to execute the plan.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReorganize(cmd, brain.ModeAll)
		},
	}

	addReorgFlags(cmd, true, true)
	cmd.AddCommand(newBrainReorgDedupCmd())
	cmd.AddCommand(newBrainReorgRecategorizeCmd())
	cmd.AddCommand(newBrainReorgConsolidateCmd())

	return cmd
}

func newBrainReorgDedupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dedup",
		Short: "Find and merge duplicate brain files",
		RunE:  func(cmd *cobra.Command, args []string) error { return runReorganize(cmd, brain.ModeDedup) },
	}
	addReorgFlags(cmd, true, false)
	return cmd
}

func newBrainReorgRecategorizeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recategorize",
		Short: "Find brain files that belong in a different category",
		RunE:  func(cmd *cobra.Command, args []string) error { return runReorganize(cmd, brain.ModeRecategorize) },
	}
	addReorgFlags(cmd, false, false)
	return cmd
}

func newBrainReorgConsolidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "consolidate",
		Short: "Find small related brain files and merge them",
		RunE:  func(cmd *cobra.Command, args []string) error { return runReorganize(cmd, brain.ModeConsolidate) },
	}
	addReorgFlags(cmd, false, true)
	return cmd
}

// addReorgFlags registers the common reorganize flags. withSimilarity and
// withSmallFile control which optional flags are added for that subcommand.
func addReorgFlags(cmd *cobra.Command, withSimilarity, withSmallFile bool) {
	cmd.Flags().Bool("apply", false, "Execute the plan (default: dry-run)")
	if withSimilarity {
		cmd.Flags().Float64("similarity", 0.7, "Similarity threshold for duplicate detection (0.0-1.0)")
	}
	if withSmallFile {
		cmd.Flags().Int("small-file", 50, "Word count threshold for small files eligible for consolidation")
	}
}

// runReorganize is the shared runner for all reorganize subcommands.
// It builds opts from the command flags, calls Reorganize with the given mode,
// and prints the result. Find and apply happen under a single lock inside
// Reorganize, avoiding any TOCTOU race between analysis and execution.
func runReorganize(cmd *cobra.Command, mode string) error {
	apply, _ := cmd.Flags().GetBool("apply")

	opts := brain.ReorgOptions{
		Mode:   mode,
		DryRun: !apply,
	}
	if f := cmd.Flags().Lookup("similarity"); f != nil {
		opts.SimilarityThreshold, _ = cmd.Flags().GetFloat64("similarity")
	}
	if f := cmd.Flags().Lookup("small-file"); f != nil {
		opts.SmallFileThreshold, _ = cmd.Flags().GetInt("small-file")
	}

	b := brain.New(getBrainPath())
	plan, err := b.Reorganize(opts)
	if err != nil {
		return fmt.Errorf("reorganize: %w", err)
	}

	printReorgPlan(plan, apply)
	return nil
}

func printReorgPlan(plan *brain.ReorgPlan, applied bool) {
	if len(plan.Actions) == 0 {
		fmt.Println("Brain looks clean — no reorganization needed.")
		return
	}

	status := "Proposed"
	if applied {
		status = "Applied"
	}
	fmt.Printf("%s %d reorganization action(s):\n\n", status, len(plan.Actions))

	for i, a := range plan.Actions {
		switch a.Type {
		case brain.ActionMerge:
			fmt.Printf("  %d. MERGE → %s\n", i+1, a.TargetPath)
			for _, src := range a.SourcePaths {
				fmt.Printf("       absorb: %s\n", src)
			}
		case brain.ActionMove:
			fmt.Printf("  %d. MOVE %s → %s\n", i+1, a.SourcePaths[0], a.TargetPath)
		case brain.ActionConsolidate:
			fmt.Printf("  %d. CONSOLIDATE → %s\n", i+1, a.TargetPath)
			for _, src := range a.SourcePaths {
				fmt.Printf("       include: %s\n", src)
			}
		}
		fmt.Printf("       reason: %s\n\n", a.Reason)
	}

	if !applied {
		fmt.Println("Run with --apply to execute.")
	} else {
		fmt.Printf("Removed files have been moved to %s/.trash/ and can be recovered if needed.\n", getBrainPath())
	}
}

func getBrainPath() string {
	cfg, err := loadConfig()
	if err != nil {
		return config.DefaultBrainPath
	}
	if cfg.Brain.Path != "" {
		return cfg.Brain.Path
	}
	return config.DefaultBrainPath
}

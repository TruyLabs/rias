package cli

import (
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/norenis/kai/internal/brain"
	"github.com/norenis/kai/internal/config"
	"github.com/spf13/cobra"
	"github.com/xuri/excelize/v2"
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
				fmt.Printf("Brain is empty. Use '%s teach' to start teaching.\n", config.DefaultAgentName)
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
	cmd.AddCommand(newBrainImportCmd())
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

func newBrainImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import [files...]",
		Short: "Import .md, .csv, or .xlsx files into brain",
		Long: `Import markdown, CSV, or Excel files into the brain as knowledge files.
CSV and Excel files are automatically converted to markdown table format.
You can optionally specify brain subdirectory and tags.

Auto-tagging extracts meaningful keywords from file content.
Auto-chunking breaks large files into semantic chunks for better search.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			category, _ := cmd.Flags().GetString("category")
			tagsStr, _ := cmd.Flags().GetString("tags")
			confidence, _ := cmd.Flags().GetString("confidence")
			autoTag, _ := cmd.Flags().GetBool("auto-tag")
			autoChunk, _ := cmd.Flags().GetBool("auto-chunk")

			b := brain.New(getBrainPath())
			var tags []string
			if tagsStr != "" {
				tags = strings.Split(tagsStr, ",")
				for i := range tags {
					tags[i] = strings.TrimSpace(tags[i])
				}
			}

			imported := 0
			for _, filePath := range args {
				bf, err := importFile(filePath, category, tags, confidence, autoTag, autoChunk)
				if err != nil {
					fmt.Printf("Error importing %s: %v\n", filePath, err)
					continue
				}

				if err := b.Save(bf); err != nil {
					fmt.Printf("Error saving %s: %v\n", bf.Path, err)
					continue
				}

				fmt.Printf("Imported: %s\n", bf.Path)
				if autoTag && len(bf.Tags) > 0 {
					fmt.Printf("  Tags: %s\n", strings.Join(bf.Tags, ", "))
				}
				imported++
			}

			if err := b.RebuildIndex(); err != nil {
				return fmt.Errorf("rebuild index: %w", err)
			}

			fmt.Printf("\nSuccessfully imported %d file(s).\n", imported)
			return nil
		},
	}

	cmd.Flags().StringP("category", "c", "knowledge", "Brain subdirectory category")
	cmd.Flags().StringP("tags", "t", "", "Comma-separated tags")
	cmd.Flags().StringP("confidence", "C", "medium", "Confidence level: high, medium, or low")
	cmd.Flags().BoolP("auto-tag", "a", false, "Auto-extract tags from content")
	cmd.Flags().BoolP("auto-chunk", "k", false, "Auto-chunk large files for better search")

	return cmd
}


func importFile(filePath string, category string, tags []string, confidence string, autoTag bool, autoChunk bool) (*brain.BrainFile, error) {
	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	baseFileName := strings.TrimSuffix(filepath.Base(filePath), ext)

	var content string
	switch ext {
	case ".md":
		content = string(data)

	case ".csv":
		// Parse CSV and convert to markdown table
		records, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
		if err != nil {
			return nil, fmt.Errorf("parse CSV: %w", err)
		}

		if len(records) == 0 {
			return nil, fmt.Errorf("CSV is empty")
		}

		// Build markdown table with proper escaping
		content = brain.BuildMarkdownTable(records)

	case ".xlsx":
		// Parse Excel file and convert to markdown tables
		f, err := excelize.OpenFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("open Excel file: %w", err)
		}
		defer f.Close()

		sheetNames := f.GetSheetList()
		if len(sheetNames) == 0 {
			return nil, fmt.Errorf("Excel file has no sheets")
		}

		// Get all rows from all sheets
		var buf strings.Builder
		for i, sheetName := range sheetNames {
			if i > 0 {
				buf.WriteString("\n\n")
			}
			if len(sheetNames) > 1 {
				buf.WriteString("## " + sheetName + "\n\n")
			}

			rows, err := f.GetRows(sheetName)
			if err != nil {
				return nil, fmt.Errorf("read sheet %s: %w", sheetName, err)
			}

			if len(rows) == 0 {
				continue
			}

			// Build markdown table with proper escaping
			if len(rows[0]) > 0 {
				buf.WriteString(brain.BuildMarkdownTable(rows))
			}
		}
		content = buf.String()

	default:
		return nil, fmt.Errorf("unsupported file type: %s (only .md, .csv, and .xlsx supported)", ext)
	}

	// Construct brain file path
	brainPath := filepath.Join(category, baseFileName+".md")
	brainPath = filepath.Clean(strings.ReplaceAll(brainPath, "\\", "/"))

	// Auto-extract tags if enabled
	finalTags := tags
	if autoTag {
		extractedTags := brain.ExtractTags(content)
		// Merge extracted tags with provided tags
		seen := make(map[string]bool)
		for _, t := range tags {
			seen[t] = true
		}
		for _, t := range extractedTags {
			if !seen[t] {
				finalTags = append(finalTags, t)
				seen[t] = true
			}
		}
	}

	// Note: Auto-chunking is handled by the indexing system (RebuildIndex)
	// when it analyzes the content. Files are automatically chunked during search indexing.

	return &brain.BrainFile{
		Path:       brainPath,
		Content:    "\n" + strings.TrimSpace(content) + "\n",
		Tags:       finalTags,
		Confidence: confidence,
		Source:     "imported:" + filepath.Base(filePath),
		Updated:    brain.DateOnly{Time: time.Now()},
	}, nil
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

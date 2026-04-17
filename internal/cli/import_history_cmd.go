package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/TruyLabs/rias/internal/importer"
	"github.com/TruyLabs/rias/internal/prompt"
	"github.com/spf13/cobra"
)

func newImportHistoryCmd() *cobra.Command {
	var providerFlag string
	var fileFlag string

	cmd := &cobra.Command{
		Use:   "import-history",
		Short: "Import conversation history from Claude or ChatGPT into brain",
		Example: `  rias import-history --provider claude --file ~/Downloads/claude-export.json
  rias import-history --provider chatgpt --file ~/Downloads/conversations.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runImportHistory(providerFlag, fileFlag)
		},
	}

	cmd.Flags().StringVar(&providerFlag, "provider", "", "Export provider: claude or chatgpt (required)")
	cmd.Flags().StringVar(&fileFlag, "file", "", "Path to the export JSON file (required)")
	_ = cmd.MarkFlagRequired("provider")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}

func runImportHistory(providerName, filePath string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Read export file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read export file: %w", err)
	}

	// Parse conversations
	var convs []importer.Conversation
	switch providerName {
	case "claude":
		convs, err = importer.ParseClaude(data)
	case "chatgpt":
		convs, err = importer.ParseChatGPT(data)
	default:
		return fmt.Errorf("unknown provider %q — use claude or chatgpt", providerName)
	}
	if err != nil {
		return fmt.Errorf("parse %s export: %w", providerName, err)
	}

	fmt.Printf("Parsed %d conversations from %s export.\n", len(convs), providerName)

	// Load import manifest
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	manifestPath := filepath.Join(home, "."+cfg.AgentName(), "import_manifest.json")
	manifest, err := importer.LoadManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("load import manifest: %w", err)
	}

	// Build LLM provider + brain
	_, b, p, _, err := buildRouter(cfg)
	if err != nil {
		return fmt.Errorf("build router: %w", err)
	}
	// SetProactiveRecall is intentionally omitted: BuildLearningPrompt does not use it.
	pb := prompt.NewBuilder(cfg.AgentName(), cfg.UserName())

	processed, skipped, failed := 0, 0, 0
	for _, conv := range convs {
		if len(conv.Messages) == 0 {
			skipped++
			continue
		}
		hash := importer.ConversationHash(conv)
		if manifest.Processed[conv.ID] == hash {
			skipped++
			continue
		}

		learnings, err := importer.ExtractLearnings(context.Background(), conv, p, pb)
		if err != nil {
			fmt.Printf("  ⚠ Skipping %q: %v\n", conv.Title, err)
			failed++
			continue
		}

		if len(learnings) > 0 {
			if err := b.Learn(learnings); err != nil {
				fmt.Printf("  ⚠ Brain write failed for %q: %v\n", conv.Title, err)
				failed++
				continue
			}
		}

		manifest.Processed[conv.ID] = hash
		processed++
		fmt.Printf("  ✓ %s (%d learnings)\n", conv.Title, len(learnings))
	}

	// Save updated manifest
	if err := importer.SaveManifest(manifestPath, manifest); err != nil {
		return fmt.Errorf("save import manifest: %w", err)
	}

	// Rebuild brain indexes so new content is searchable
	if processed > 0 {
		if err := b.RebuildTagIndex(); err != nil {
			fmt.Printf("  ⚠ Tag index rebuild failed: %v\n", err)
		}
		if err := b.RebuildIndex(); err != nil {
			fmt.Printf("  ⚠ Full index rebuild failed: %v\n", err)
		}
	}

	fmt.Printf("\nDone. Processed: %d | Skipped: %d | Failed: %d\n", processed, skipped, failed)
	return nil
}

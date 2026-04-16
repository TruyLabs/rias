package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TruyLabs/rias/internal/auth"
	"github.com/TruyLabs/rias/internal/brain"
	"github.com/TruyLabs/rias/internal/config"
	"github.com/TruyLabs/rias/internal/prompt"
	"github.com/TruyLabs/rias/internal/provider"
	"github.com/spf13/cobra"
)

func newTeachCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "teach",
		Short: "Enter teaching mode — tell your digital twin about yourself",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			_, _, _, _, err = buildRouter(cfg)
			if err != nil {
				return err
			}

			return runTeachMode(cfg)
		},
	}
}

func runTeachMode(cfg *config.Config) error {
	b := brain.New(cfg.Brain.Path)
	applyEmbedConfig(b, cfg)
	builder := prompt.NewBuilder(cfg.AgentName(), cfg.UserName())
	reader := bufio.NewReader(os.Stdin)
	ctx := context.Background()

	// Get provider for learning extraction
	provCfg := cfg.Providers[cfg.Provider]
	apiKey := provCfg.APIKey
	if apiKey == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		ks := auth.NewKeystore(filepath.Join(home, credentialsDir, credentialsFile))
		mgr := auth.NewManager(ks)
		key, err := mgr.GetCredential(cfg.Provider)
		if err != nil {
			return fmt.Errorf("no API key configured")
		}
		apiKey = key
	}

	timeout := time.Duration(provCfg.TimeoutSec) * time.Second

	var prov provider.Provider
	switch cfg.Provider {
	case "claude":
		prov = provider.NewClaude(apiKey, provCfg.Model, provCfg.BaseURL, timeout)
	case "openai":
		prov = provider.NewOpenAI(apiKey, provCfg.Model, provCfg.BaseURL, timeout)
	case "gemini":
		prov = provider.NewGemini(apiKey, provCfg.Model, provCfg.BaseURL, timeout)
	default:
		return fmt.Errorf("unsupported provider: %s", cfg.Provider)
	}

	fmt.Println("Teaching mode. Tell me about yourself and I'll remember.")
	fmt.Println("Type /done when finished.")
	fmt.Println()

	learned := 0

	for {
		fmt.Print("teach> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}
		if input == "/done" {
			break
		}

		// Use LLM to extract learning from direct teaching
		teachPrompt := builder.BuildLearningPrompt(
			nil,
			[]provider.Message{
				{Role: "user", Content: "I want to teach you about myself: " + input},
			},
		)

		resp, err := prov.Chat(ctx, "", []provider.Message{
			{Role: "user", Content: teachPrompt},
		})
		if err != nil {
			fmt.Printf("  Error extracting learning: %v\n", err)
			continue
		}

		learnings, err := brain.ParseLearnings(resp.Content)
		if err != nil || len(learnings) == 0 {
			fmt.Println("  Hmm, I couldn't extract anything specific from that. Try being more concrete.")
			continue
		}

		for i := range learnings {
			learnings[i].Source = "direct"
		}

		if err := b.Learn(learnings); err != nil {
			fmt.Printf("  Error saving: %v\n", err)
			continue
		}
		if err := b.RebuildIndex(); err != nil {
			fmt.Printf("  Warning: index rebuild failed: %v\n", err)
		}

		for _, l := range learnings {
			fmt.Printf("  Saved to brain/%s/%s.md\n", l.Category, l.Topic)
		}
		learned += len(learnings)
	}

	fmt.Printf("\nLearned %d new things. Brain updated.\n", learned)
	return nil
}

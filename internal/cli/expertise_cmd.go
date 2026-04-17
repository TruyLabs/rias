package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/TruyLabs/rias/internal/brain"
	"github.com/TruyLabs/rias/internal/prompt"
	"github.com/TruyLabs/rias/internal/provider"
	"github.com/spf13/cobra"
)

const expertiseFilePath = "expertise/map.md"
const maxExpertiseFiles = 60

func newExpertiseCmd() *cobra.Command {
	var updateFlag bool

	cmd := &cobra.Command{
		Use:   "expertise",
		Short: "Show or regenerate your expertise map",
		Example: `  rias expertise            # show current expertise map
  rias expertise --update   # regenerate from all brain files`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExpertise(updateFlag)
		},
	}
	cmd.Flags().BoolVar(&updateFlag, "update", false, "Regenerate expertise map from brain files")
	return cmd
}

func runExpertise(update bool) error {
	brainPath := getBrainPath()
	b := brain.New(brainPath)

	if !update {
		bf, err := b.Load(expertiseFilePath)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("load expertise map: %w", err)
			}
			fmt.Println("No expertise map yet. Run with --update to generate one.")
			return nil
		}
		fmt.Println(strings.TrimSpace(bf.Content))
		return nil
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	allPaths, err := b.ListAll()
	if err != nil {
		return fmt.Errorf("list brain: %w", err)
	}

	var files []*brain.BrainFile
	for _, path := range allPaths {
		if path == expertiseFilePath {
			continue // skip the expertise map itself to avoid circular input
		}
		bf, err := b.Load(path)
		if err != nil {
			continue
		}
		files = append(files, bf)
		if len(files) >= maxExpertiseFiles {
			break
		}
	}

	if len(files) == 0 {
		fmt.Println("No brain files found. Start chatting with 'rias' to build your brain.")
		return nil
	}

	fmt.Printf("Analyzing %d brain files...\n", len(files))

	prov, err := getProvider(cfg)
	if err != nil {
		return fmt.Errorf("get provider: %w", err)
	}

	pb := prompt.NewBuilder(cfg.AgentName(), cfg.UserName())
	date := time.Now().Format("2006-01-02")
	expertisePrompt := pb.BuildExpertisePrompt(files, date)

	resp, err := prov.Chat(context.Background(), "", []provider.Message{
		{Role: "user", Content: expertisePrompt},
	})
	if err != nil {
		return fmt.Errorf("LLM expertise analysis: %w", err)
	}

	bf := &brain.BrainFile{
		Path:       expertiseFilePath,
		Tags:       []string{"expertise", "map"},
		Confidence: brain.ConfidenceMedium,
		Source:     "expertise-cmd",
		Updated:    brain.DateOnly{Time: time.Now()},
		Content:    "\n" + strings.TrimSpace(resp.Content) + "\n",
	}
	if err := b.Save(bf); err != nil {
		return fmt.Errorf("save expertise map: %w", err)
	}

	fmt.Printf("\n%s\n", strings.TrimSpace(resp.Content))
	fmt.Printf("\nSaved to brain/%s\n", expertiseFilePath)
	return nil
}

package cli

import (
	"fmt"
	"os"
	"strings"

	kai "github.com/TruyLabs/rias"
	"github.com/TruyLabs/rias/internal/config"
	"github.com/spf13/cobra"
)

var cfgFile string

// NewRootCmd creates the root cobra command.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   config.DefaultAgentName,
		Short: "Your digital twin — an AI agent that thinks like you",
		Long:  config.DefaultAgentName + " is a CLI-based AI agent that acts as your digital twin. It learns about you through conversations and can answer questions, make decisions, and write code the way you would.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChat(cmd, args)
		},
	}

	root.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ./config.yaml)")

	root.AddCommand(newVersionCmd())
	root.AddCommand(newAskCmd())
	root.AddCommand(newAuthCmd())
	root.AddCommand(newBrainCmd())
	root.AddCommand(newTeachCmd())
	root.AddCommand(newMcpCmd())
	root.AddCommand(newSetupCmd())
	root.AddCommand(newDashboardCmd())
	root.AddCommand(newSyncCmd())
	root.AddCommand(newReindexCmd())
	root.AddCommand(newModuleCmd())
	root.AddCommand(newTaskCmd())
	root.AddCommand(newImportHistoryCmd())
	root.AddCommand(newIndexRepoCmd())
	root.AddCommand(newGoalCmd())
	root.AddCommand(newReflectCmd())
	root.AddCommand(newExpertiseCmd())

	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s %s (commit: %s, built: %s)\n", config.DefaultAgentName, kai.Version, kai.Commit, kai.BuildDate)
			go checkForUpdate()
		},
	}
}

func newAskCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ask [question]",
		Short: "Ask a one-shot question",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			r, _, _, sessMgr, err := buildRouter(cfg)
			if err != nil {
				return err
			}
			question := strings.Join(args, " ")
			return runOneShotAsk(r, sessMgr, cfg, question)
		},
	}
}

func runChat(cmd *cobra.Command, args []string) error {
	go checkForUpdate()
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	r, b, _, sessMgr, err := buildRouter(cfg)
	if err != nil {
		return err
	}
	return runInteractiveChat(r, b, sessMgr, cfg)
}

// Execute runs the root command.
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

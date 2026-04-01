package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/norenis/kai/internal/auth"
	"github.com/spf13/cobra"
)

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication for LLM providers",
	}

	cmd.AddCommand(newAuthSetKeyCmd())
	cmd.AddCommand(newAuthStatusCmd())

	return cmd
}

func newAuthSetKeyCmd() *cobra.Command {
	var providerName string

	cmd := &cobra.Command{
		Use:   "set-key",
		Short: "Set an API key for a provider",
		RunE: func(cmd *cobra.Command, args []string) error {
			if providerName == "" {
				return fmt.Errorf("--provider is required")
			}

			fmt.Printf("Enter API key for %s: ", providerName)
			reader := bufio.NewReader(os.Stdin)
			key, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("read input: %w", err)
			}
			key = strings.TrimSpace(key)

			if key == "" {
				return fmt.Errorf("API key cannot be empty")
			}

			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("cannot determine home directory: %w", err)
			}
			ks := auth.NewKeystore(filepath.Join(home, credentialsDir, credentialsFile))
			mgr := auth.NewManager(ks)

			if err := mgr.SetKey(providerName, key); err != nil {
				return fmt.Errorf("save key: %w", err)
			}

			fmt.Printf("API key saved for %s\n", providerName)
			return nil
		},
	}

	cmd.Flags().StringVar(&providerName, "provider", "", "Provider name (e.g., claude, openai)")
	cmd.MarkFlagRequired("provider")

	return cmd
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show authentication status for all providers",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("cannot determine home directory: %w", err)
			}
			ks := auth.NewKeystore(filepath.Join(home, credentialsDir, credentialsFile))
			mgr := auth.NewManager(ks)

			providers := []string{"claude", "openai"}
			for _, p := range providers {
				status := "not configured"
				if mgr.HasCredential(p) {
					status = "configured"
				}
				fmt.Printf("  %s: %s\n", p, status)
			}
			return nil
		},
	}
}

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/tinhvqbk/kai/internal/brain"
	"github.com/spf13/cobra"
)

const (
	mcpConfigName = "kai"
	moduleURL     = "github.com/tinhvqbk/kai/cmd/kai@latest"
)

func newSetupCmd() *cobra.Command {
	var skipInstall bool

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Install kai and configure as MCP server for Claude Code",
		Long: `One-time setup that:
  1. Installs kai binary via 'go install'
  2. Creates default brain directory and config
  3. Registers kai as an MCP server for Claude Code`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetup(skipInstall)
		},
	}

	cmd.Flags().BoolVar(&skipInstall, "skip-install", false, "Skip 'go install' (use existing binary)")

	return cmd
}

func runSetup(skipInstall bool) error {
	// Step 1: Install binary.
	if !skipInstall {
		fmt.Println("Installing kai...")
		install := exec.Command("go", "install", moduleURL)
		install.Stdout = os.Stdout
		install.Stderr = os.Stderr
		if err := install.Run(); err != nil {
			return fmt.Errorf("go install failed: %w\n  Run with --skip-install if kai is already on PATH", err)
		}
		fmt.Println("  Installed kai binary.")
	}

	// Verify kai is on PATH.
	kaiPath, err := exec.LookPath("kai")
	if err != nil {
		return fmt.Errorf("kai not found on PATH — ensure $GOPATH/bin is in your PATH")
	}
	fmt.Printf("  Binary: %s\n", kaiPath)

	// Step 2: Create brain directory.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	brainPath := filepath.Join(homeDir, ".kai", "brain")
	for _, cat := range brain.DefaultCategories {
		dir := filepath.Join(brainPath, cat)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create brain dir: %w", err)
		}
	}
	fmt.Printf("  Brain: %s\n", brainPath)

	// Step 3: Create default config if not exists.
	configPath := filepath.Join(homeDir, ".kai", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConfig := fmt.Sprintf(`# kai configuration
brain:
  path: %s
  max_context_files: 10
sessions_path: %s
`, brainPath, filepath.Join(homeDir, ".kai", "sessions"))
		if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
			return fmt.Errorf("write config: %w", err)
		}
		fmt.Printf("  Config: %s\n", configPath)
	} else {
		fmt.Printf("  Config: %s (already exists)\n", configPath)
	}

	// Step 4: Register as Claude Code MCP server.
	if err := registerMCPServer(homeDir, kaiPath, configPath); err != nil {
		return fmt.Errorf("register MCP server: %w", err)
	}

	fmt.Println("\nSetup complete! kai is ready as an MCP server for Claude Code.")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Teach kai about yourself:  kai teach --config " + configPath)
	fmt.Println("  2. Restart Claude Code to pick up the MCP server")
	return nil
}

func registerMCPServer(homeDir, kaiPath, configPath string) error {
	// Write .mcp.json in the user's home directory for global MCP access.
	mcpFile := filepath.Join(homeDir, ".mcp.json")

	var mcpConfig map[string]interface{}

	// Read existing .mcp.json if it exists.
	if data, err := os.ReadFile(mcpFile); err == nil {
		if err := json.Unmarshal(data, &mcpConfig); err != nil {
			mcpConfig = make(map[string]interface{})
		}
	} else {
		mcpConfig = make(map[string]interface{})
	}

	// Ensure mcpServers key exists.
	servers, ok := mcpConfig["mcpServers"].(map[string]interface{})
	if !ok {
		servers = make(map[string]interface{})
		mcpConfig["mcpServers"] = servers
	}

	// Add kai server entry.
	servers[mcpConfigName] = map[string]interface{}{
		"type":    "stdio",
		"command": kaiPath,
		"args":    []string{"mcp", "--config", configPath},
		"env":     map[string]string{},
	}

	data, err := json.MarshalIndent(mcpConfig, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(mcpFile, data, 0644); err != nil {
		return err
	}
	fmt.Printf("  MCP config: %s\n", mcpFile)

	// Platform hint.
	if runtime.GOOS == "darwin" {
		fmt.Println("  Registered kai as global MCP server for Claude Code.")
	}

	return nil
}

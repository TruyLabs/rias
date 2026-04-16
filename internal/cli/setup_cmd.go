package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	kai "github.com/TruyLabs/rias"
	"github.com/TruyLabs/rias/internal/brain"
	"github.com/TruyLabs/rias/internal/config"
	"github.com/spf13/cobra"
)

const moduleURL = "github.com/TruyLabs/rias/cmd/kai@latest"

func newSetupCmd() *cobra.Command {
	var useLocal bool

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Configure as MCP server for Claude Code",
		Long: `One-time setup that:
  1. Creates default brain directory and config
  2. Registers as an MCP server for Claude Code

By default uses 'go run' from the module (no install needed).
Use --local to register the current binary instead.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetup(useLocal)
		},
	}

	cmd.Flags().BoolVar(&useLocal, "local", false, "Use the current binary path instead of 'go run'")

	return cmd
}

func runSetup(useLocal bool) error {
	agentName := config.DefaultAgentName

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	kaiDir := filepath.Join(homeDir, "."+agentName)

	// Step 1: Create brain directory.
	brainPath := filepath.Join(kaiDir, "brain")
	for _, cat := range brain.DefaultCategories {
		dir := filepath.Join(brainPath, cat)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create brain dir: %w", err)
		}
	}
	fmt.Printf("  Brain:    %s\n", brainPath)

	// Step 1b: Create sessions directory.
	sessionsPath := filepath.Join(kaiDir, "sessions")
	if err := os.MkdirAll(sessionsPath, 0755); err != nil {
		return fmt.Errorf("create sessions dir: %w", err)
	}
	fmt.Printf("  Sessions: %s\n", sessionsPath)

	// Step 2: Create default config if not exists.
	configPath := filepath.Join(homeDir, "."+agentName, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConfig := fmt.Sprintf(`# %s configuration
agent:
  name: %s
  user_name: %s
brain:
  path: %s
  max_context_files: 5
sessions_path: %s
`, agentName, agentName, config.DefaultUserName, brainPath, sessionsPath)
		if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
			return fmt.Errorf("write config: %w", err)
		}
		fmt.Printf("  Config: %s\n", configPath)
	} else {
		fmt.Printf("  Config: %s (already exists)\n", configPath)
	}

	// Step 3: Register as Claude Code MCP server.
	if err := registerMCPServer(homeDir, configPath, useLocal, agentName); err != nil {
		return fmt.Errorf("register MCP server: %w", err)
	}

	// Step 4: Install Claude Code slash commands.
	if err := installClaudeCommands(homeDir, agentName); err != nil {
		fmt.Printf("  ⚠ Could not install slash commands: %v\n", err)
	}

	fmt.Printf("\nSetup complete! %s is ready as an MCP server for Claude Code.\n", agentName)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Restart Claude Code to pick up the MCP server")
	fmt.Printf("  2. Use /%s:ask, /%s:teach, /%s:brain-search, etc.\n", agentName, agentName, agentName)
	fmt.Printf("  3. Ask Claude Code to teach %s about you\n", agentName)
	return nil
}

func registerMCPServer(homeDir, configPath string, useLocal bool, agentName string) error {
	mcpFile := filepath.Join(homeDir, ".mcp.json")

	var mcpConfig map[string]interface{}

	if data, err := os.ReadFile(mcpFile); err == nil {
		if err := json.Unmarshal(data, &mcpConfig); err != nil {
			mcpConfig = make(map[string]interface{})
		}
	} else {
		mcpConfig = make(map[string]interface{})
	}

	servers, ok := mcpConfig["mcpServers"].(map[string]interface{})
	if !ok {
		servers = make(map[string]interface{})
		mcpConfig["mcpServers"] = servers
	}

	var entry map[string]interface{}

	if useLocal {
		// Use the current binary directly.
		kaiPath, err := exec.LookPath(agentName)
		if err != nil {
			// Fall back to the current executable.
			kaiPath, err = os.Executable()
			if err != nil {
				return fmt.Errorf("%s not found on PATH and cannot determine current executable", agentName)
			}
		}
		entry = map[string]interface{}{
			"type":    "stdio",
			"command": kaiPath,
			"args":    []string{"mcp", "--config", configPath},
			"env":     map[string]string{},
		}
		fmt.Printf("  Mode: local binary (%s)\n", kaiPath)
	} else {
		// Use 'go run' from module — no install needed, always runs latest.
		entry = map[string]interface{}{
			"type":    "stdio",
			"command": "go",
			"args":    []string{"run", moduleURL, "mcp", "--config", configPath},
			"env":     map[string]string{},
		}
		fmt.Printf("  Mode: go run %s\n", moduleURL)
	}

	servers[agentName] = entry

	data, err := json.MarshalIndent(mcpConfig, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(mcpFile, data, 0644); err != nil {
		return err
	}
	fmt.Printf("  MCP config: %s\n", mcpFile)
	return nil
}

// installClaudeCommands writes the slash commands to ~/.claude/commands/<agentName>/.
func installClaudeCommands(homeDir, agentName string) error {
	cmdDir := filepath.Join(homeDir, ".claude", "commands", agentName)
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		return err
	}

	cmds := kai.ClaudeCommands(agentName)
	for name, content := range cmds {
		path := filepath.Join(cmdDir, name+".md")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}

	fmt.Printf("  Commands: %s (%d slash commands)\n", cmdDir, len(cmds))
	return nil
}

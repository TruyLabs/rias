package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tinhvqbk/kai/internal/brain"
	"github.com/tinhvqbk/kai/internal/config"
	kaimcp "github.com/tinhvqbk/kai/internal/mcp"
	"github.com/tinhvqbk/kai/internal/session"
	"github.com/spf13/cobra"
)

const mcpTokenEnvVar = "KAI_MCP_TOKEN"

func newMcpCmd() *cobra.Command {
	var transport string
	var addr string

	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start MCP server",
		Long:  "Start a Model Context Protocol server. Supports stdio (default) and HTTP transports. HTTP transport requires KAI_MCP_TOKEN for bearer auth. Brain tools work without an LLM provider; ask/teach require one.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				cfg = config.Defaults()
			}

			b := brain.New(cfg.Brain.Path)
			for _, dir := range brain.DefaultCategories {
				os.MkdirAll(filepath.Join(cfg.Brain.Path, dir), brain.DirPermissions)
			}
			if _, err := b.LoadIndex(); err != nil {
				if rebuildErr := b.RebuildIndex(); rebuildErr != nil {
					fmt.Fprintf(os.Stderr, "WARNING: failed to rebuild brain index: %v\n", rebuildErr)
				}
			}

			sessPath := cfg.SessionsPath
			if sessPath == "" {
				sessPath = config.DefaultSessionsPath
			}
			sessMgr := session.NewManager(sessPath)

			// Provider is optional — brain tools work without it
			r, _, prov, _, err := buildRouter(cfg)
			if err != nil {
				// Non-fatal: brain tools still work
				r = nil
				prov = nil
			}

			srv := kaimcp.NewServer(r, b, sessMgr, prov, cfg)

			switch transport {
			case "stdio":
				return srv.Serve()
			case "http":
				token := os.Getenv(mcpTokenEnvVar)
				if token == "" {
					return fmt.Errorf("%s must be set for HTTP transport", mcpTokenEnvVar)
				}
				return srv.ServeHTTP(addr, token)
			default:
				return fmt.Errorf("unknown transport %q (use stdio or http)", transport)
			}
		},
	}

	cmd.Flags().StringVar(&transport, "transport", "stdio", "Transport type: stdio or http")
	cmd.Flags().StringVar(&addr, "addr", config.DefaultListenAddr, "Listen address for HTTP transport")

	return cmd
}

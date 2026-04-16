package cli

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/TruyLabs/rias/internal/brain"
	"github.com/TruyLabs/rias/internal/config"
	kaimcp "github.com/TruyLabs/rias/internal/mcp"
	"github.com/TruyLabs/rias/internal/session"
	"github.com/spf13/cobra"
)

const mcpTokenEnvVar = "KAI_MCP_TOKEN"

func newMcpCmd() *cobra.Command {
	var transport string
	var addr string
	var logLevel string

	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start MCP server",
		Long:  "Start a Model Context Protocol server. Supports stdio (default) and HTTP transports. HTTP transport requires KAI_MCP_TOKEN for bearer auth. Brain tools work without an LLM provider; ask/teach require one.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Log level: flag > env var > default (info).
			if logLevel == "" {
				logLevel = os.Getenv("KAI_LOG_LEVEL")
			}
			setupLogger(logLevel)

			cfg, err := loadConfig()
			if err != nil {
				cfg = config.Defaults()
			}

			b := brain.New(cfg.Brain.Path)
			applyEmbedConfig(b, cfg)
			for _, dir := range brain.DefaultCategories {
				os.MkdirAll(filepath.Join(cfg.Brain.Path, dir), brain.DirPermissions)
			}
			if _, err := b.LoadIndex(); err != nil {
				if rebuildErr := b.RebuildIndex(); rebuildErr != nil {
					slog.Warn("failed to rebuild brain index", "err", rebuildErr)
				}
			}

			sessPath := config.ExpandPath(cfg.SessionsPath)
			if sessPath == "" {
				sessPath = config.ExpandPath(config.DefaultSessionsPath)
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
			srv.TriggerReindex() // kick off full BM25+vector rebuild in background

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
	cmd.Flags().StringVar(&logLevel, "log-level", "", "Log level: debug, info, warn, error (default: info, env: KAI_LOG_LEVEL)")

	return cmd
}

// setupLogger configures the default slog logger (and thereby log.Printf) at the given level.
func setupLogger(level string) {
	var l slog.Level
	switch strings.ToLower(level) {
	case "debug":
		l = slog.LevelDebug
	case "warn", "warning":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: l})))
}

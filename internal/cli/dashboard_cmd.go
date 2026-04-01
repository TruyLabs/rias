package cli

import (
	"fmt"
	"log/slog"
	"net"
	"os/exec"
	"runtime"

	"github.com/tinhvqbk/kai/internal/brain"
	"github.com/tinhvqbk/kai/internal/config"
	"github.com/tinhvqbk/kai/internal/dashboard"
	bsync "github.com/tinhvqbk/kai/internal/sync"
	"github.com/spf13/cobra"
)

func newDashboardCmd() *cobra.Command {
	var port int
	var noOpen bool

	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Launch the brain dashboard web UI",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			brainPath := cfg.Brain.Path
			if brainPath == "" {
				brainPath = config.DefaultBrainPath
			}

			// Pick a random available port if not specified.
			if !cmd.Flags().Changed("port") {
				p, err := randomPort()
				if err != nil {
					return fmt.Errorf("find available port: %w", err)
				}
				port = p
			}

			b := brain.New(brainPath)
			srv := dashboard.New(b, brainPath, cfg)

			// Initialize syncer if configured.
			syncer := dashboardSyncer(cfg, brainPath)
			if syncer != nil {
				srv.SetSyncer(syncer)
				slog.Info("sync backends available for dashboard")
			}

			addr := fmt.Sprintf("localhost:%d", port)
			url := fmt.Sprintf("http://%s", addr)
			fmt.Printf("%s dashboard: %s\n", cfg.AgentName(), url)

			if !noOpen {
				openBrowser(url)
			}

			return srv.ListenAndServe(addr)
		},
	}

	cmd.Flags().IntVar(&port, "port", 0, "Port to serve on (default: random available)")
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "Don't auto-open the browser")

	return cmd
}

func randomPort() (int, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port, nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}
	cmd.Start()
}

func dashboardSyncer(cfg *config.Config, brainPath string) *bsync.Syncer {
	var gitBackend bsync.Backend
	if cfg.Brain.Sync.Git.Enabled {
		gitBackend = bsync.NewGitBackend(brainPath, cfg.Brain.Sync.Git.Remote, cfg.Brain.Sync.Git.Branch)
	}

	if gitBackend == nil {
		return nil
	}
	return bsync.NewSyncer(brainPath, gitBackend, nil)
}

package cli

import (
	"context"
	"fmt"
	"sort"

	"github.com/TruyLabs/rias/internal/brain"
	"github.com/TruyLabs/rias/internal/config"
	"github.com/TruyLabs/rias/internal/module"
	"github.com/spf13/cobra"
)

func newModuleCmd() *cobra.Command {
	var runAll bool

	cmd := &cobra.Command{
		Use:   "module",
		Short: "Manage and run kai modules",
		Long:  "Modules pull data from external sources (GitHub, Google Sheets, etc.) into the brain.\nConfigure modules in config.yaml under the 'modules:' key.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !runAll {
				return cmd.Help()
			}
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			return runEnabledModules(cfg, brain.New(cfg.Brain.Path))
		},
	}

	cmd.Flags().BoolVar(&runAll, "all", false, "Run all enabled modules")

	cmd.AddCommand(newModuleListCmd())

	// Dynamically register each module in the registry as its own subcommand.
	reg := module.Default()
	for _, name := range reg.Available() {
		name := name // capture loop var
		cmd.AddCommand(&cobra.Command{
			Use:   name,
			Short: reg.Description(name),
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := loadConfig()
				if err != nil {
					return err
				}
				return execModule(cfg, brain.New(cfg.Brain.Path), name)
			},
		})
	}

	return cmd
}

func newModuleListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available modules and their status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			reg := module.Default()
			available := reg.Available()
			sort.Strings(available)

			enabled := enabledModuleSet(cfg)

			for _, name := range available {
				tag := ""
				if enabled[name] {
					tag = " [enabled]"
				}
				fmt.Printf("%-22s%-10s %s\n", name, tag, reg.Description(name))
			}
			return nil
		},
	}
}

// runEnabledModules runs every module that has enabled: true in config.
func runEnabledModules(cfg *config.Config, b *brain.FileBrain) error {
	ran := 0
	for _, mc := range cfg.Modules {
		if !mc.Enabled {
			continue
		}
		if err := execModule(cfg, b, mc.Name); err != nil {
			fmt.Printf("%s: %v\n", mc.Name, err)
		}
		ran++
	}
	if ran == 0 {
		fmt.Println("no modules enabled — set 'enabled: true' in config.yaml")
	}
	return nil
}

// execModule builds, runs, and saves the results of a single module.
func execModule(cfg *config.Config, b *brain.FileBrain, name string) error {
	modCfg := moduleConfig(cfg, name)

	reg := module.Default()
	mod, err := reg.Build(name, modCfg)
	if err != nil {
		return err
	}

	learnings, err := mod.Fetch(context.Background())
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}

	if len(learnings) == 0 {
		fmt.Printf("%s: nothing fetched\n", name)
		return nil
	}

	if err := b.Learn(learnings); err != nil {
		return fmt.Errorf("save to brain: %w", err)
	}
	if err := b.RebuildTagIndex(); err != nil {
		return fmt.Errorf("rebuild index: %w", err)
	}

	fmt.Printf("%s → %d item(s) imported\n", name, len(learnings))
	return nil
}

// moduleConfig returns the raw config map for a named module, or nil if not configured.
func moduleConfig(cfg *config.Config, name string) map[string]interface{} {
	for _, mc := range cfg.Modules {
		if mc.Name == name {
			return mc.Config
		}
	}
	return nil
}

// enabledModuleSet returns a set of module names that have enabled: true.
func enabledModuleSet(cfg *config.Config) map[string]bool {
	set := make(map[string]bool, len(cfg.Modules))
	for _, mc := range cfg.Modules {
		if mc.Enabled {
			set[mc.Name] = true
		}
	}
	return set
}

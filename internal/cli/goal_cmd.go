package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/TruyLabs/rias/internal/brain"
	"github.com/spf13/cobra"
)

func newGoalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "goal",
		Short: "Manage your goals",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGoalList()
		},
	}
	cmd.AddCommand(newGoalListCmd())
	cmd.AddCommand(newGoalAddCmd())
	cmd.AddCommand(newGoalDoneCmd())
	return cmd
}

func newGoalListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all goals",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGoalList()
		},
	}
}

func runGoalList() error {
	b := brain.New(getBrainPath())
	bf, err := b.Load(brain.GoalFilePath())
	if err != nil {
		fmt.Println("No goals yet. Use 'rias goal add' to create one.")
		return nil
	}
	goals := brain.ParseGoals(bf.Content)
	if len(goals) == 0 {
		fmt.Println("No goals yet. Use 'rias goal add' to create one.")
		return nil
	}
	done := 0
	for _, g := range goals {
		if g.Done {
			done++
		}
	}
	fmt.Printf("Goals (%d/%d done):\n\n", done, len(goals))
	for i, g := range goals {
		mark := "[ ]"
		if g.Done {
			mark = "[x]"
		}
		fmt.Printf("  [%d] %s [%s] %s\n", i, mark, g.Horizon, g.Text)
	}
	return nil
}

func newGoalAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [text]",
		Short: "Add a goal",
		Args:  cobra.MinimumNArgs(1),
		Example: `  rias goal add "learn distributed systems" --horizon medium
  rias goal add "ship rias Phase 1" --horizon short`,
		RunE: func(cmd *cobra.Command, args []string) error {
			horizon, _ := cmd.Flags().GetString("horizon")
			text := strings.Join(args, " ")
			b := brain.New(getBrainPath())
			bf, err := b.Load(brain.GoalFilePath())
			if err != nil {
				bf = brain.NewGoalFile()
			}
			bf.Content = brain.AppendGoal(bf.Content, text, horizon)
			bf.Updated = brain.DateOnly{Time: time.Now()}
			if err := b.Save(bf); err != nil {
				return fmt.Errorf("save goals: %w", err)
			}
			if horizon == "" {
				horizon = "medium"
			}
			fmt.Printf("Added [%s]: %s\n", horizon, text)
			return nil
		},
	}
	cmd.Flags().StringP("horizon", "H", "", "Goal horizon: short, medium, or long (default: medium)")
	return cmd
}

func newGoalDoneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "done [index]",
		Short: "Mark a goal as done",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var idx int
			if _, err := fmt.Sscanf(args[0], "%d", &idx); err != nil {
				return fmt.Errorf("invalid index: %s", args[0])
			}
			b := brain.New(getBrainPath())
			bf, err := b.Load(brain.GoalFilePath())
			if err != nil {
				return fmt.Errorf("no goals found")
			}
			newContent, err := brain.ToggleGoalDone(bf.Content, idx, true)
			if err != nil {
				return err
			}
			bf.Content = newContent
			bf.Updated = brain.DateOnly{Time: time.Now()}
			if err := b.Save(bf); err != nil {
				return fmt.Errorf("save goals: %w", err)
			}
			fmt.Printf("Done: goal [%d]\n", idx)
			return nil
		},
	}
}

package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/TruyLabs/rias/internal/brain"
	"github.com/spf13/cobra"
)

func newTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage today's tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTaskList()
		},
	}
	cmd.AddCommand(newTaskListCmd())
	cmd.AddCommand(newTaskAddCmd())
	cmd.AddCommand(newTaskDoneCmd())
	cmd.AddCommand(newTaskUndoneCmd())
	cmd.AddCommand(newTaskRmCmd())
	return cmd
}

func newTaskListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List today's tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTaskList()
		},
	}
}

func runTaskList() error {
	b := brain.New(getBrainPath())
	path := brain.TaskFilePath(time.Now())
	bf, err := b.Load(path)
	if err != nil {
		fmt.Println("No tasks for today. Use 'kai task add' to create one.")
		return nil
	}
	items := brain.ParseTasks(bf.Content)
	if len(items) == 0 {
		fmt.Println("No tasks for today. Use 'kai task add' to create one.")
		return nil
	}
	done := 0
	for _, t := range items {
		if t.Done {
			done++
		}
	}
	fmt.Printf("Tasks (%d/%d done):\n\n", done, len(items))
	for i, t := range items {
		mark := "○"
		if t.Done {
			mark = "✓"
		}
		pri := ""
		switch t.Priority {
		case "high":
			pri = " 🔴"
		case "medium":
			pri = " 🟡"
		case "low":
			pri = " 🟢"
		}
		fmt.Printf("  [%d] %s %s%s\n", i, mark, t.Text, pri)
	}
	return nil
}

func newTaskAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [text]",
		Short: "Add a task for today",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			priority, _ := cmd.Flags().GetString("priority")
			text := strings.Join(args, " ")
			b := brain.New(getBrainPath())
			path := brain.TaskFilePath(time.Now())
			bf, err := b.Load(path)
			if err != nil {
				bf = brain.NewTaskFile(time.Now())
			}
			bf.Content = brain.AppendTask(bf.Content, text, priority)
			bf.Updated = brain.DateOnly{Time: time.Now()}
			if err := b.Save(bf); err != nil {
				return fmt.Errorf("save tasks: %w", err)
			}
			fmt.Printf("Added: %s\n", text)
			return nil
		},
	}
	cmd.Flags().StringP("priority", "p", "", "Priority: high, medium, or low")
	return cmd
}

func newTaskDoneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "done [index]",
		Short: "Mark a task as done",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setTaskDone(args[0], true)
		},
	}
}

func newTaskUndoneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "undone [index]",
		Short: "Mark a task as not done",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setTaskDone(args[0], false)
		},
	}
}

func setTaskDone(indexStr string, done bool) error {
	var idx int
	if _, err := fmt.Sscanf(indexStr, "%d", &idx); err != nil {
		return fmt.Errorf("invalid index: %s", indexStr)
	}
	b := brain.New(getBrainPath())
	path := brain.TaskFilePath(time.Now())
	bf, err := b.Load(path)
	if err != nil {
		return fmt.Errorf("no tasks for today")
	}
	newContent, err := brain.ToggleTask(bf.Content, idx, done)
	if err != nil {
		return err
	}
	bf.Content = newContent
	bf.Updated = brain.DateOnly{Time: time.Now()}
	if err := b.Save(bf); err != nil {
		return fmt.Errorf("save tasks: %w", err)
	}
	verb := "Done"
	if !done {
		verb = "Undone"
	}
	fmt.Printf("%s: task [%d]\n", verb, idx)
	return nil
}

func newTaskRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm [index]",
		Short: "Remove a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var idx int
			if _, err := fmt.Sscanf(args[0], "%d", &idx); err != nil {
				return fmt.Errorf("invalid index: %s", args[0])
			}
			b := brain.New(getBrainPath())
			path := brain.TaskFilePath(time.Now())
			bf, err := b.Load(path)
			if err != nil {
				return fmt.Errorf("no tasks for today")
			}
			newContent, err := brain.RemoveTask(bf.Content, idx)
			if err != nil {
				return err
			}
			bf.Content = newContent
			bf.Updated = brain.DateOnly{Time: time.Now()}
			if err := b.Save(bf); err != nil {
				return fmt.Errorf("save tasks: %w", err)
			}
			fmt.Printf("Removed task [%d]\n", idx)
			return nil
		},
	}
}

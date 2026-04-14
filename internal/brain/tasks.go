package brain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// TaskItem represents a single task line from a brain tasks file.
type TaskItem struct {
	Text     string `json:"text"`
	Done     bool   `json:"done"`
	Priority string `json:"priority,omitempty"`
}

var taskLineRe = regexp.MustCompile(`^- \[([ x~])\] (.+)$`)
var taskMarkerRe = regexp.MustCompile(`\[[ x~]\]`)

// TaskFilePath returns the relative brain path for a given date's task file.
func TaskFilePath(date time.Time) string {
	return "tasks/" + date.Format(DateFormat) + ".md"
}

// ParseTasks parses GFM task list lines from brain file content.
func ParseTasks(content string) []TaskItem {
	var items []TaskItem
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		m := taskLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		items = append(items, TaskItem{
			Text:     m[2],
			Done:     m[1] == "x",
			Priority: extractPriority(m[2]),
		})
	}
	return items
}

// extractPriority detects priority from emoji markers in task text.
func extractPriority(text string) string {
	switch {
	case strings.Contains(text, "🔴"):
		return "high"
	case strings.Contains(text, "🟡"):
		return "medium"
	case strings.Contains(text, "🟢"):
		return "low"
	default:
		return ""
	}
}

// ToggleTask flips the done state of the task at index idx in markdown content.
func ToggleTask(content string, idx int, done bool) (string, error) {
	lines := strings.Split(content, "\n")
	taskIdx := 0
	for i, line := range lines {
		if taskLineRe.MatchString(strings.TrimSpace(line)) {
			if taskIdx == idx {
				marker := "[ ]"
				if done {
					marker = "[x]"
				}
				lines[i] = taskMarkerRe.ReplaceAllString(line, marker)
				return strings.Join(lines, "\n"), nil
			}
			taskIdx++
		}
	}
	return content, fmt.Errorf("task index %d out of range (%d tasks)", idx, taskIdx)
}

// AppendTask adds a new task line to content. Priority emoji is prepended if set.
func AppendTask(content, text, priority string) string {
	var prefix string
	switch priority {
	case "high":
		prefix = "🔴 "
	case "medium":
		prefix = "🟡 "
	case "low":
		prefix = "🟢 "
	}
	line := "- [ ] " + prefix + text
	trimmed := strings.TrimRight(content, " \t")
	if strings.HasSuffix(trimmed, "\n") {
		return content + line + "\n"
	}
	return content + "\n" + line + "\n"
}

// RemoveTask removes the task at index idx from content.
func RemoveTask(content string, idx int) (string, error) {
	lines := strings.Split(content, "\n")
	taskIdx := 0
	for i, line := range lines {
		if taskLineRe.MatchString(strings.TrimSpace(line)) {
			if taskIdx == idx {
				lines = append(lines[:i], lines[i+1:]...)
				return strings.Join(lines, "\n"), nil
			}
			taskIdx++
		}
	}
	return content, fmt.Errorf("task index %d out of range (%d tasks)", idx, taskIdx)
}

// NewTaskFile creates a new BrainFile for the given date's tasks.
func NewTaskFile(date time.Time) *BrainFile {
	return &BrainFile{
		Path:       TaskFilePath(date),
		Tags:       []string{"tasks", "daily"},
		Confidence: ConfidenceHigh,
		Source:     "cli",
		Updated:    DateOnly{Time: date},
		Content:    "\n",
	}
}

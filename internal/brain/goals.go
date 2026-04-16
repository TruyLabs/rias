package brain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// GoalItem represents a single goal entry parsed from a goals brain file.
type GoalItem struct {
	Text    string // goal description
	Horizon string // "short" | "medium" | "long"
	Done    bool
}

var goalLineRe = regexp.MustCompile(`^- \[([ x])\] \[(\w+)\] (.+)$`)
var goalMarkerRe = regexp.MustCompile(`\[[ x]\]`)

// GoalFilePath returns the brain-relative path for the goals file.
func GoalFilePath() string {
	return "goals/goals.md"
}

// ParseGoals parses goal list lines from brain file content.
func ParseGoals(content string) []GoalItem {
	var items []GoalItem
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		m := goalLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		items = append(items, GoalItem{
			Done:    m[1] == "x",
			Horizon: m[2],
			Text:    m[3],
		})
	}
	return items
}

// AppendGoal adds a new goal line to content.
// If horizon is empty, it defaults to "medium".
func AppendGoal(content, text, horizon string) string {
	if horizon == "" {
		horizon = "medium"
	}
	line := fmt.Sprintf("- [ ] [%s] %s", horizon, text)
	trimmed := strings.TrimRight(content, " \t")
	if strings.HasSuffix(trimmed, "\n") {
		return content + line + "\n"
	}
	return content + "\n" + line + "\n"
}

// ToggleGoalDone flips the done state of the goal at index idx in markdown content.
func ToggleGoalDone(content string, idx int, done bool) (string, error) {
	lines := strings.Split(content, "\n")
	goalIdx := 0
	for i, line := range lines {
		if goalLineRe.MatchString(strings.TrimSpace(line)) {
			if goalIdx == idx {
				marker := "[ ]"
				if done {
					marker = "[x]"
				}
				lines[i] = goalMarkerRe.ReplaceAllString(line, marker)
				return strings.Join(lines, "\n"), nil
			}
			goalIdx++
		}
	}
	return content, fmt.Errorf("goal index %d out of range (%d goals)", idx, goalIdx)
}

// NewGoalFile creates a new BrainFile for goals.
func NewGoalFile() *BrainFile {
	return &BrainFile{
		Path:       GoalFilePath(),
		Tags:       []string{"goals"},
		Confidence: ConfidenceHigh,
		Source:     "cli",
		Updated:    DateOnly{Time: time.Now()},
		Content:    "\n",
	}
}

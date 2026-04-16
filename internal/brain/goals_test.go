package brain_test

import (
	"testing"

	"github.com/TruyLabs/rias/internal/brain"
)

func TestParseGoals(t *testing.T) {
	content := `
- [ ] [short] Ship rias Phase 1
- [x] [medium] Learn distributed systems
- [ ] [long] Build a successful startup
`
	goals := brain.ParseGoals(content)
	if len(goals) != 3 {
		t.Fatalf("expected 3 goals, got %d", len(goals))
	}
	if goals[0].Text != "Ship rias Phase 1" {
		t.Errorf("expected 'Ship rias Phase 1', got %q", goals[0].Text)
	}
	if goals[0].Horizon != "short" {
		t.Errorf("expected horizon=short, got %q", goals[0].Horizon)
	}
	if goals[0].Done {
		t.Error("expected goal[0] to be not done")
	}
	if !goals[1].Done {
		t.Error("expected goal[1] to be done")
	}
	if goals[2].Horizon != "long" {
		t.Errorf("expected horizon=long, got %q", goals[2].Horizon)
	}
}

func TestAppendGoal(t *testing.T) {
	content := "- [ ] [short] Existing goal\n"
	result := brain.AppendGoal(content, "New goal", "medium")
	goals := brain.ParseGoals(result)
	if len(goals) != 2 {
		t.Fatalf("expected 2 goals, got %d", len(goals))
	}
	if goals[1].Text != "New goal" {
		t.Errorf("expected 'New goal', got %q", goals[1].Text)
	}
	if goals[1].Horizon != "medium" {
		t.Errorf("expected horizon=medium, got %q", goals[1].Horizon)
	}
}

func TestAppendGoalDefaultHorizon(t *testing.T) {
	result := brain.AppendGoal("\n", "My goal", "")
	goals := brain.ParseGoals(result)
	if len(goals) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(goals))
	}
	if goals[0].Horizon != "medium" {
		t.Errorf("expected default horizon=medium, got %q", goals[0].Horizon)
	}
}

func TestToggleGoalDone(t *testing.T) {
	content := "- [ ] [short] Goal one\n- [ ] [medium] Goal two\n"
	result, err := brain.ToggleGoalDone(content, 1, true)
	if err != nil {
		t.Fatalf("ToggleGoalDone: %v", err)
	}
	goals := brain.ParseGoals(result)
	if goals[0].Done {
		t.Error("expected goal[0] to still be not done")
	}
	if !goals[1].Done {
		t.Error("expected goal[1] to be done")
	}
}

func TestToggleGoalDoneOutOfRange(t *testing.T) {
	content := "- [ ] [short] Only one\n"
	_, err := brain.ToggleGoalDone(content, 5, true)
	if err == nil {
		t.Error("expected error for out-of-range index")
	}
}

func TestGoalFilePath(t *testing.T) {
	p := brain.GoalFilePath()
	if p != "goals/goals.md" {
		t.Errorf("expected goals/goals.md, got %q", p)
	}
}

func TestToggleGoalDonePreservesHorizon(t *testing.T) {
	// horizon "x" is the pathological case: goalMarkerRe must not replace it
	content := "- [ ] [x] Explore something\n"
	result, err := brain.ToggleGoalDone(content, 0, true)
	if err != nil {
		t.Fatal(err)
	}
	goals := brain.ParseGoals(result)
	if len(goals) != 1 {
		t.Fatalf("expected 1 goal, got %d (horizon tag may have been corrupted)", len(goals))
	}
	if goals[0].Horizon != "x" {
		t.Errorf("horizon corrupted: want %q, got %q", "x", goals[0].Horizon)
	}
	if !goals[0].Done {
		t.Error("expected goal to be marked done")
	}
}

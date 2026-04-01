package brain

import (
	"math"
	"testing"
	"time"
)

func TestRelevanceScoreRecency(t *testing.T) {
	// A file updated today should score higher than one updated 180 days ago.
	recent := &BrainFile{
		Updated:    DateOnly{time.Now()},
		Confidence: "medium",
	}
	old := &BrainFile{
		Updated:    DateOnly{time.Now().AddDate(0, 0, -180)},
		Confidence: "medium",
	}

	recentScore := RelevanceScore(recent, 1.0)
	oldScore := RelevanceScore(old, 1.0)

	if recentScore <= oldScore {
		t.Errorf("recent score (%.4f) should be > old score (%.4f)", recentScore, oldScore)
	}
}

func TestRelevanceScoreAccess(t *testing.T) {
	// A file accessed more often should score higher.
	lowAccess := &BrainFile{
		Updated:     DateOnly{time.Now()},
		Confidence:  "medium",
		AccessCount: 1,
	}
	highAccess := &BrainFile{
		Updated:     DateOnly{time.Now()},
		Confidence:  "medium",
		AccessCount: 100,
	}

	lowScore := RelevanceScore(lowAccess, 1.0)
	highScore := RelevanceScore(highAccess, 1.0)

	if highScore <= lowScore {
		t.Errorf("high access score (%.4f) should be > low access score (%.4f)", highScore, lowScore)
	}
}

func TestRelevanceScoreConfidence(t *testing.T) {
	base := 1.0
	high := &BrainFile{Updated: DateOnly{time.Now()}, Confidence: "high"}
	low := &BrainFile{Updated: DateOnly{time.Now()}, Confidence: "low"}

	highScore := RelevanceScore(high, base)
	lowScore := RelevanceScore(low, base)

	if highScore <= lowScore {
		t.Errorf("high confidence score (%.4f) should be > low confidence score (%.4f)", highScore, lowScore)
	}
}

func TestRelevanceScoreBaseZero(t *testing.T) {
	bf := &BrainFile{Updated: DateOnly{time.Now()}, Confidence: "high"}
	score := RelevanceScore(bf, 0.0)
	if math.Abs(score) > 1e-10 {
		t.Errorf("score with baseScore=0 should be ~0, got %.4f", score)
	}
}

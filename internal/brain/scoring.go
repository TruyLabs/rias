package brain

import (
	"math"
	"time"
)

// Scoring weights for relevance calculation.
const (
	RecencyDecayDays       = 90.0 // exponential decay half-life in days
	AccessFrequencyDivisor = 10.0
	ConfidenceBoostHigh    = 0.3
	ConfidenceBoostMedium  = 0.2
	ConfidenceBoostLow     = 0.1
)

// RelevanceScore adjusts a base TF-IDF score with recency, access frequency,
// and confidence factors.
func RelevanceScore(bf *BrainFile, baseScore float64) float64 {
	// Recency: exponential decay over ~3 months
	daysSinceUpdate := time.Since(bf.Updated.Time).Hours() / 24
	if daysSinceUpdate < 0 {
		daysSinceUpdate = 0
	}
	recency := math.Exp(-daysSinceUpdate / RecencyDecayDays)

	// Access frequency: diminishing returns via log
	access := math.Log(1.0+float64(bf.AccessCount)) / AccessFrequencyDivisor

	// Confidence boost
	confidenceBoosts := [...]float64{ConfidenceBoostLow, ConfidenceBoostLow, ConfidenceBoostMedium, ConfidenceBoostHigh}
	confidence := confidenceBoosts[confidenceRank(bf.Confidence)]

	return baseScore * (1.0 + recency + access + confidence)
}

package brain

import (
	"time"
)

// DateFormat is the standard date format used across brain files.
const DateFormat = "2006-01-02"

// File system permissions for brain data.
const (
	DirPermissions  = 0755
	FilePermissions = 0644
)

// DefaultCategories are the standard brain knowledge categories.
var DefaultCategories = []string{"identity", "opinions", "style", "decisions", "knowledge"}

// DateOnly is a time.Time that marshals/unmarshals as "2006-01-02".
type DateOnly struct {
	time.Time
}

func (d *DateOnly) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	t, err := time.Parse(DateFormat, s)
	if err != nil {
		// Fall back to full RFC3339
		t, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return err
		}
	}
	d.Time = t
	return nil
}

func (d DateOnly) MarshalYAML() (interface{}, error) {
	return d.Time.Format(DateFormat), nil
}

// BrainFile represents a loaded brain file with parsed frontmatter.
type BrainFile struct {
	Path         string   `yaml:"-"`
	Tags         []string `yaml:"tags"`
	Confidence   string   `yaml:"confidence"`
	Source       string   `yaml:"source"`
	Updated      DateOnly `yaml:"updated"`
	AccessCount  int      `yaml:"access_count,omitempty"`
	LastAccessed DateOnly `yaml:"last_accessed,omitempty"`
	Content      string   `yaml:"-"`
}

// Learning represents a piece of knowledge extracted from conversation.
type Learning struct {
	Category   string   `json:"category"`
	Topic      string   `json:"topic"`
	Tags       []string `json:"tags"`
	Content    string   `json:"content"`
	Confidence string   `json:"confidence"`
	Action     string   `json:"action"`
	Source     string   `json:"source,omitempty"`
}

// Index maps tags to file paths for retrieval.
type Index struct {
	Tags map[string][]string `json:"tags"`
}

// FullIndex stores an inverted index for full-text TF-IDF search.
type FullIndex struct {
	Documents     map[string]DocEntry  `json:"documents"`
	InvertedIndex map[string][]Posting `json:"inverted_index"`
	TotalDocs     int                  `json:"total_docs"`
}

// Posting records a term occurrence in a document field.
type Posting struct {
	Path      string `json:"path"`
	Frequency int    `json:"frequency"`
	Field     string `json:"field"` // "tag", "content", or "path"
}

// DocEntry stores per-document metadata for the full index.
type DocEntry struct {
	Path        string   `json:"path"`
	WordCount   int      `json:"word_count"`
	Tags        []string `json:"tags"`
	Updated     string   `json:"updated"`
	AccessCount int      `json:"access_count"`
}

// Reorganization action type constants.
const (
	ActionMerge       = "merge"
	ActionMove        = "move"
	ActionConsolidate = "consolidate"
)

// Reorganization mode constants.
const (
	ModeAll          = "all"
	ModeDedup        = "dedup"
	ModeRecategorize = "recategorize"
	ModeConsolidate  = "consolidate"
)

// ReorgAction describes a single proposed reorganization operation.
type ReorgAction struct {
	Type        string   `json:"type"`
	SourcePaths []string `json:"source_paths"`
	TargetPath  string   `json:"target_path"`
	Reason      string   `json:"reason"`
	Similarity  float64  `json:"similarity,omitempty"`
}

// ReorgPlan is a set of proposed reorganization actions.
type ReorgPlan struct {
	Actions []ReorgAction `json:"actions"`
}

// ReorgOptions controls reorganization behavior.
type ReorgOptions struct {
	Mode                string  // ModeAll, ModeDedup, ModeRecategorize, ModeConsolidate
	SimilarityThreshold float64 // 0.0-1.0, default 0.7 for dedup
	SmallFileThreshold  int     // word count below which a file is "small", default 50
	DryRun              bool    // if true, return plan without executing
}

// DefaultReorgOptions returns sensible defaults for reorganization.
func DefaultReorgOptions() ReorgOptions {
	return ReorgOptions{
		Mode:                ModeAll,
		SimilarityThreshold: 0.7,
		SmallFileThreshold:  50,
		DryRun:              true,
	}
}

package module

import (
	"context"

	"github.com/norenis/kai/internal/brain"
)

// Module is the interface all kai modules must implement.
// Modules fetch data from external sources and return learnings to be saved to the brain.
type Module interface {
	Name() string
	Description() string
	Fetch(ctx context.Context) ([]brain.Learning, error)
}

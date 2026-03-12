package agent

import (
	"context"

	"github.com/GrayFlash/kirkup-cli/models"
)

// Adapter is implemented by each AI agent source.
type Adapter interface {
	// Name returns the agent identifier (e.g. "gemini-cli", "cursor").
	Name() string

	// Detect reports whether this agent is installed on the system.
	Detect() bool

	// WatchGlobs returns file glob patterns the collector should watch.
	WatchGlobs() []string

	// Events reads all prompt events from the given file path.
	// The collector is responsible for deduplication.
	Events(ctx context.Context, path string) ([]models.PromptEvent, error)
}

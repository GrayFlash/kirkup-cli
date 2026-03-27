package classifier

import (
	"context"

	"github.com/GrayFlash/kirkup-cli/models"
)

// Classifier assigns categories to prompt events.
type Classifier interface {
	// Name identifies this classifier (e.g. "rules-v1").
	Name() string

	// Classify returns a Classification for each provided event.
	// Events that cannot be classified are omitted from the result.
	Classify(ctx context.Context, events []models.PromptEvent) ([]models.Classification, error)
}

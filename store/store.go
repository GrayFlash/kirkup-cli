package store

import (
	"context"
	"time"

	"github.com/GrayFlash/kirkup-cli/models"
)

type Store interface {
	InsertPromptEvent(ctx context.Context, e *models.PromptEvent) error
	QueryPromptEvents(ctx context.Context, f EventFilter) ([]models.PromptEvent, error)
	InsertClassification(ctx context.Context, c *models.Classification) error
	Migrate(ctx context.Context) error
	Close() error
}

type EventFilter struct {
	Since *time.Time
	Until *time.Time
	Agent string
	Limit int
}

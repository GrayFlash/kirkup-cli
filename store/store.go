package store

import (
	"context"
	"time"

	"github.com/GrayFlash/kirkup-cli/models"
)

type Store interface {
	// Prompt events
	InsertPromptEvent(ctx context.Context, e *models.PromptEvent) error
	QueryPromptEvents(ctx context.Context, f EventFilter) ([]models.PromptEvent, error)

	// Classifications
	InsertClassification(ctx context.Context, c *models.Classification) error
	GetUnclassified(ctx context.Context, limit int) ([]models.PromptEvent, error)
	QueryClassifications(ctx context.Context, eventIDs []string) ([]models.Classification, error)

	// Sessions
	UpsertSession(ctx context.Context, s *models.Session) error
	QuerySessions(ctx context.Context, f SessionFilter) ([]models.Session, error)

	// Projects
	UpsertProject(ctx context.Context, p *models.Project) error
	ListProjects(ctx context.Context) ([]models.Project, error)

	// Lifecycle
	Migrate(ctx context.Context) error
	Close() error
}

type EventFilter struct {
	Since   *time.Time
	Until   *time.Time
	Agent   string
	Project string
	Limit   int
}

type SessionFilter struct {
	Since   *time.Time
	Until   *time.Time
	Project string
	Agent   string
}

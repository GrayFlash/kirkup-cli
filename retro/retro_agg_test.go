package retro

import (
	"context"
	"testing"
	"time"

	"github.com/GrayFlash/kirkup-cli/models"
	"github.com/GrayFlash/kirkup-cli/store"
)

type mockStore struct {
	events []models.PromptEvent
}

func (m *mockStore) QueryPromptEvents(ctx context.Context, f store.EventFilter) ([]models.PromptEvent, error) {
	return m.events, nil
}

// Stubs for other interface methods
func (m *mockStore) InsertPromptEvent(ctx context.Context, e *models.PromptEvent) error { return nil }
func (m *mockStore) InsertClassification(ctx context.Context, c *models.Classification) error { return nil }
func (m *mockStore) QueryClassifications(ctx context.Context, ids []string) ([]models.Classification, error) { return nil, nil }
func (m *mockStore) GetUnclassified(ctx context.Context, limit int) ([]models.PromptEvent, error) { return nil, nil }
func (m *mockStore) UpsertSession(ctx context.Context, s *models.Session) error { return nil }
func (m *mockStore) QuerySessions(ctx context.Context, f store.SessionFilter) ([]models.Session, error) { return nil, nil }
func (m *mockStore) UpsertProject(ctx context.Context, p *models.Project) error { return nil }
func (m *mockStore) ListProjects(ctx context.Context) ([]models.Project, error) { return nil, nil }
func (m *mockStore) Close() error { return nil }
func (m *mockStore) Migrate(ctx context.Context) error { return nil }

func TestAggregate_Empty(t *testing.T) {
	store := &mockStore{events: []models.PromptEvent{}}
	
	from := time.Now()
	to := time.Now()
	
	summary, err := Aggregate(context.Background(), store, from, to, "", 30)
	if err != nil {
		t.Fatalf("Aggregate error: %v", err)
	}
	
	if summary.TotalPrompts != 0 {
		t.Errorf("expected 0 prompts, got %d", summary.TotalPrompts)
	}
}

func TestAggregate_CalculatesSummary(t *testing.T) {
	now := time.Now()
	
	events := []models.PromptEvent{
		{ID: "1", Timestamp: now, Agent: "cursor", Project: "projA"},
		{ID: "2", Timestamp: now.Add(5 * time.Minute), Agent: "cursor", Project: "projA"},
		{ID: "3", Timestamp: now.Add(1 * time.Hour), Agent: "gemini", Project: "projB"},
	}
	store := &mockStore{events: events}
	
	summary, err := Aggregate(context.Background(), store, now.Add(-time.Hour), now.Add(2*time.Hour), "", 30)
	if err != nil {
		t.Fatalf("Aggregate error: %v", err)
	}
	
	if summary.TotalPrompts != 3 {
		t.Errorf("expected 3 prompts, got %d", summary.TotalPrompts)
	}
	
	if summary.TotalSessions != 2 {
		t.Errorf("expected 2 sessions, got %d", summary.TotalSessions)
	}
}

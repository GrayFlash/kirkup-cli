package collector

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GrayFlash/kirkup-cli/agent"
	"github.com/GrayFlash/kirkup-cli/config"
	"github.com/GrayFlash/kirkup-cli/models"
	"github.com/GrayFlash/kirkup-cli/store"
)

type mockAdapter struct {
	name   string
	events []models.PromptEvent
}

func (a *mockAdapter) Name() string { return a.name }
func (a *mockAdapter) Detect() bool { return true }
func (a *mockAdapter) WatchGlobs() []string { return []string{"*.log"} }
func (a *mockAdapter) Events(ctx context.Context, path string) ([]models.PromptEvent, error) {
	return a.events, nil
}

type mockFullStore struct {
	events []models.PromptEvent
}

func (m *mockFullStore) InsertPromptEvent(ctx context.Context, e *models.PromptEvent) error {
	m.events = append(m.events, *e)
	return nil
}
func (m *mockFullStore) QueryPromptEvents(ctx context.Context, f store.EventFilter) ([]models.PromptEvent, error) {
	return m.events, nil
}
func (m *mockFullStore) ListEventIDs(ctx context.Context) ([]string, error) {
	var ids []string
	for _, e := range m.events {
		ids = append(ids, e.ID)
	}
	return ids, nil
}
func (m *mockFullStore) InsertClassification(ctx context.Context, c *models.Classification) error { return nil }
func (m *mockFullStore) GetUnclassified(ctx context.Context, limit int) ([]models.PromptEvent, error) { return nil, nil }
func (m *mockFullStore) QueryClassifications(ctx context.Context, ids []string) ([]models.Classification, error) { return nil, nil }
func (m *mockFullStore) UpsertSession(ctx context.Context, s *models.Session) error { return nil }
func (m *mockFullStore) QuerySessions(ctx context.Context, f store.SessionFilter) ([]models.Session, error) { return nil, nil }
func (m *mockFullStore) UpsertProject(ctx context.Context, p *models.Project) error { return nil }
func (m *mockFullStore) ListProjects(ctx context.Context) ([]models.Project, error) { return nil, nil }
func (m *mockFullStore) Close() error { return nil }
func (m *mockFullStore) Migrate(ctx context.Context) error { return nil }

func TestCollector_Scan(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")
	if err := os.WriteFile(logPath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	events := []models.PromptEvent{
		{Timestamp: time.Now(), Prompt: "p1", Agent: "mock"},
		{Timestamp: time.Now(), Prompt: "p2", Agent: "mock"},
	}
	
	adapter := &mockAdapter{name: "mock", events: events}
	registry := agent.NewRegistry(adapter)
	
	// Create a dummy config where the adapter pattern matches our temp file
	cfg := &config.Config{
		Agents: map[string]config.AgentConfig{
			"mock": {LogPaths: []string{filepath.Join(tmpDir, "*.log")}},
		},
	}
	
	s := &mockFullStore{}
	c := New(registry, s, cfg, nil)
	
	processed, newCount := c.Scan(context.Background())
	
	if processed != 2 {
		t.Errorf("expected 2 processed, got %d", processed)
	}
	if newCount != 2 {
		t.Errorf("expected 2 new, got %d", newCount)
	}
	
	// Run again, should be 0 new due to memory 'seen' map
	_, newCount = c.Scan(context.Background())
	if newCount != 0 {
		t.Errorf("expected 0 new on second run, got %d", newCount)
	}
}

func TestCollector_Redact(t *testing.T) {
	cfg := &config.Config{
		Privacy: config.PrivacyConfig{
			Redact:   true,
			Patterns: []string{`sk-[a-zA-Z0-9]{10}`},
		},
	}
	
	c := New(nil, nil, cfg, nil)
	
	prompt := "my key is sk-1234567890 and it is secret"
	expected := "my key is [REDACTED] and it is secret"
	
	result := c.redact(prompt)
	if result != expected {
		t.Errorf("redact() = %q, want %q", result, expected)
	}
}

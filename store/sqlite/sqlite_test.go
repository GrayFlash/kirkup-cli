package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/GrayFlash/kirkup-cli/models"
	"github.com/GrayFlash/kirkup-cli/store"
	"github.com/GrayFlash/kirkup-cli/store/sqlite"
)

func openTestStore(t *testing.T) *sqlite.Store {
	t.Helper()
	s, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func mustTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

// -- PromptEvent --

func TestInsertAndQueryPromptEvents(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	events := []models.PromptEvent{
		{
			Timestamp:  mustTime("2026-03-10T10:00:00Z"),
			Agent:      "gemini-cli",
			Prompt:     "implement the parser",
			Project:    "project-alpha",
			GitBranch:  "feature/parser",
			GitRemote:  "github.com/example/project-alpha",
			WorkingDir: "/home/user/project-alpha",
		},
		{
			Timestamp: mustTime("2026-03-10T11:00:00Z"),
			Agent:     "cursor",
			Prompt:    "fix the Docker build",
			Project:   "project-beta",
		},
		{
			Timestamp: mustTime("2026-03-09T09:00:00Z"),
			Agent:     "gemini-cli",
			Prompt:    "refactor the calculator",
			Project:   "project-alpha",
		},
	}

	for i := range events {
		if err := s.InsertPromptEvent(ctx, &events[i]); err != nil {
			t.Fatalf("insert event %d: %v", i, err)
		}
	}

	t.Run("all events returned unfiltered", func(t *testing.T) {
		got, err := s.QueryPromptEvents(ctx, store.EventFilter{})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 3 {
			t.Errorf("want 3 events, got %d", len(got))
		}
	})

	t.Run("filter by agent", func(t *testing.T) {
		got, err := s.QueryPromptEvents(ctx, store.EventFilter{Agent: "gemini-cli"})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Errorf("want 2 events, got %d", len(got))
		}
	})

	t.Run("filter by project", func(t *testing.T) {
		got, err := s.QueryPromptEvents(ctx, store.EventFilter{Project: "project-beta"})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Errorf("want 1 event, got %d", len(got))
		}
	})

	t.Run("filter by time range", func(t *testing.T) {
		since := mustTime("2026-03-10T00:00:00Z")
		got, err := s.QueryPromptEvents(ctx, store.EventFilter{Since: &since})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Errorf("want 2 events, got %d", len(got))
		}
	})

	t.Run("limit applied", func(t *testing.T) {
		got, err := s.QueryPromptEvents(ctx, store.EventFilter{Limit: 1})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Errorf("want 1 event, got %d", len(got))
		}
	})

	t.Run("all fields persisted", func(t *testing.T) {
		got, err := s.QueryPromptEvents(ctx, store.EventFilter{Agent: "gemini-cli", Limit: 1})
		if err != nil {
			t.Fatal(err)
		}
		e := got[0]
		if e.Project != "project-alpha" {
			t.Errorf("project: want project-alpha, got %q", e.Project)
		}
		if e.GitBranch != "feature/parser" {
			t.Errorf("git_branch: want feature/parser, got %q", e.GitBranch)
		}
		if e.GitRemote != "github.com/example/project-alpha" {
			t.Errorf("git_remote: want github.com/example/project-alpha, got %q", e.GitRemote)
		}
		if e.WorkingDir != "/home/user/project-alpha" {
			t.Errorf("working_dir: want /home/user/project-alpha, got %q", e.WorkingDir)
		}
	})
}

// -- Classification --

func TestInsertClassificationAndGetUnclassified(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	e1 := &models.PromptEvent{Timestamp: mustTime("2026-03-10T10:00:00Z"), Agent: "gemini-cli", Prompt: "implement foo"}
	e2 := &models.PromptEvent{Timestamp: mustTime("2026-03-10T11:00:00Z"), Agent: "gemini-cli", Prompt: "fix bar"}
	for _, e := range []*models.PromptEvent{e1, e2} {
		if err := s.InsertPromptEvent(ctx, e); err != nil {
			t.Fatalf("insert event: %v", err)
		}
	}

	t.Run("both unclassified initially", func(t *testing.T) {
		got, err := s.GetUnclassified(ctx, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Errorf("want 2 unclassified, got %d", len(got))
		}
	})

	if err := s.InsertClassification(ctx, &models.Classification{
		PromptEventID: e1.ID,
		Category:      "coding",
		Confidence:    1.0,
		Classifier:    "rules-v1",
	}); err != nil {
		t.Fatalf("insert classification: %v", err)
	}

	t.Run("one classified, one remaining", func(t *testing.T) {
		got, err := s.GetUnclassified(ctx, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Errorf("want 1 unclassified, got %d", len(got))
		}
		if got[0].ID != e2.ID {
			t.Errorf("want unclassified event to be e2")
		}
	})

	t.Run("limit on unclassified", func(t *testing.T) {
		e3 := &models.PromptEvent{Timestamp: mustTime("2026-03-10T12:00:00Z"), Agent: "cursor", Prompt: "refactor baz"}
		_ = s.InsertPromptEvent(ctx, e3)

		got, err := s.GetUnclassified(ctx, 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Errorf("want 1 with limit, got %d", len(got))
		}
	})
}

// -- Sessions --

func TestUpsertAndQuerySessions(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	sess := &models.Session{
		Project:             "project-alpha",
		Agent:               "gemini-cli",
		StartedAt:           mustTime("2026-03-10T09:00:00Z"),
		EndedAt:             mustTime("2026-03-10T09:35:00Z"),
		PromptCount:         4,
		GapThresholdMinutes: 30,
	}

	if err := s.UpsertSession(ctx, sess); err != nil {
		t.Fatalf("upsert session: %v", err)
	}

	t.Run("session retrieved", func(t *testing.T) {
		got, err := s.QuerySessions(ctx, store.SessionFilter{})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Fatalf("want 1 session, got %d", len(got))
		}
		if got[0].PromptCount != 4 {
			t.Errorf("prompt_count: want 4, got %d", got[0].PromptCount)
		}
	})

	t.Run("upsert updates existing session", func(t *testing.T) {
		sess.PromptCount = 7
		sess.EndedAt = mustTime("2026-03-10T10:00:00Z")
		if err := s.UpsertSession(ctx, sess); err != nil {
			t.Fatalf("upsert: %v", err)
		}
		got, err := s.QuerySessions(ctx, store.SessionFilter{})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Fatalf("want 1 session after upsert, got %d", len(got))
		}
		if got[0].PromptCount != 7 {
			t.Errorf("prompt_count after upsert: want 7, got %d", got[0].PromptCount)
		}
	})

	t.Run("filter by project", func(t *testing.T) {
		other := &models.Session{
			Project:   "project-beta",
			Agent:     "cursor",
			StartedAt: mustTime("2026-03-10T14:00:00Z"),
			EndedAt:   mustTime("2026-03-10T14:30:00Z"),
		}
		_ = s.UpsertSession(ctx, other)

		got, err := s.QuerySessions(ctx, store.SessionFilter{Project: "project-alpha"})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Errorf("want 1 session for project, got %d", len(got))
		}
	})

	t.Run("filter by time range", func(t *testing.T) {
		since := mustTime("2026-03-10T13:00:00Z")
		got, err := s.QuerySessions(ctx, store.SessionFilter{Since: &since})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Errorf("want 1 session in range, got %d", len(got))
		}
		if got[0].Project != "project-beta" {
			t.Errorf("want project-beta session, got %q", got[0].Project)
		}
	})
}

// -- Projects --

func TestUpsertAndListProjects(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	p := &models.Project{
		Name:        "project-alpha",
		DisplayName: "Project Alpha",
		GitRemotes:  []string{"github.com/example/project-alpha"},
		Paths:       []string{"/home/user/project-alpha", "/home/user/work/alpha"},
	}

	if err := s.UpsertProject(ctx, p); err != nil {
		t.Fatalf("upsert project: %v", err)
	}

	t.Run("project listed", func(t *testing.T) {
		got, err := s.ListProjects(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Fatalf("want 1 project, got %d", len(got))
		}
		if got[0].DisplayName != "Project Alpha" {
			t.Errorf("display_name: want Project Alpha, got %q", got[0].DisplayName)
		}
		if len(got[0].GitRemotes) != 1 || got[0].GitRemotes[0] != "github.com/example/project-alpha" {
			t.Errorf("git_remotes: unexpected value %v", got[0].GitRemotes)
		}
		if len(got[0].Paths) != 2 {
			t.Errorf("paths: want 2, got %d", len(got[0].Paths))
		}
	})

	t.Run("upsert updates existing project", func(t *testing.T) {
		p.DisplayName = "Project Alpha (updated)"
		p.GitRemotes = []string{"github.com/example/project-alpha", "github.com/example/alpha-mirror"}
		if err := s.UpsertProject(ctx, p); err != nil {
			t.Fatalf("upsert: %v", err)
		}
		got, err := s.ListProjects(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Fatalf("want 1 project after upsert, got %d", len(got))
		}
		if got[0].DisplayName != "Project Alpha (updated)" {
			t.Errorf("display_name after upsert: want Project Alpha (updated), got %q", got[0].DisplayName)
		}
		if len(got[0].GitRemotes) != 2 {
			t.Errorf("git_remotes after upsert: want 2, got %d", len(got[0].GitRemotes))
		}
	})

	t.Run("multiple projects listed in order", func(t *testing.T) {
		_ = s.UpsertProject(ctx, &models.Project{Name: "project-beta", DisplayName: "Project Beta"})
		got, err := s.ListProjects(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Fatalf("want 2 projects, got %d", len(got))
		}
		if got[0].Name != "project-alpha" {
			t.Errorf("expected alphabetical order, first should be project-alpha, got %q", got[0].Name)
		}
	})
}

package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/GrayFlash/kirkup-cli/models"
	"github.com/GrayFlash/kirkup-cli/store"
)

const schema = `
CREATE TABLE IF NOT EXISTS prompt_events (
	id          TEXT PRIMARY KEY,
	timestamp   DATETIME NOT NULL,
	agent       TEXT NOT NULL,
	session_id  TEXT,
	prompt      TEXT NOT NULL,
	project     TEXT,
	git_branch  TEXT,
	git_remote  TEXT,
	working_dir TEXT,
	raw_source  TEXT,
	created_at  DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_prompt_events_timestamp ON prompt_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_prompt_events_agent     ON prompt_events(agent);
CREATE INDEX IF NOT EXISTS idx_prompt_events_project   ON prompt_events(project);

CREATE TABLE IF NOT EXISTS classifications (
	id              TEXT PRIMARY KEY,
	prompt_event_id TEXT NOT NULL REFERENCES prompt_events(id),
	category        TEXT NOT NULL,
	confidence      REAL NOT NULL DEFAULT 1.0,
	classifier      TEXT NOT NULL DEFAULT 'rules-v1',
	created_at      DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_classifications_event    ON classifications(prompt_event_id);
CREATE INDEX IF NOT EXISTS idx_classifications_category ON classifications(category);

CREATE TABLE IF NOT EXISTS sessions (
	id                     TEXT PRIMARY KEY,
	project                TEXT,
	agent                  TEXT,
	started_at             DATETIME NOT NULL,
	ended_at               DATETIME NOT NULL,
	prompt_count           INTEGER NOT NULL DEFAULT 0,
	gap_threshold_minutes  INTEGER NOT NULL DEFAULT 30
);

CREATE INDEX IF NOT EXISTS idx_sessions_project ON sessions(project);
CREATE INDEX IF NOT EXISTS idx_sessions_started ON sessions(started_at);

CREATE TABLE IF NOT EXISTS projects (
	name         TEXT PRIMARY KEY,
	display_name TEXT,
	git_remotes  TEXT,
	paths        TEXT,
	created_at   DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
`

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Migrate(_ context.Context) error {
	_, err := s.db.Exec(schema)
	return err
}

func (s *Store) Close() error {
	return s.db.Close()
}

// -- Prompt events --

func (s *Store) InsertPromptEvent(ctx context.Context, e *models.PromptEvent) error {
	if e.ID == "" {
		e.ID = newID()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO prompt_events
		 (id, timestamp, agent, session_id, prompt, project, git_branch, git_remote, working_dir, raw_source, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.Timestamp.UTC(), e.Agent, e.SessionID, e.Prompt,
		e.Project, e.GitBranch, e.GitRemote, e.WorkingDir, e.RawSource, e.CreatedAt.UTC(),
	)
	return err
}

func (s *Store) QueryPromptEvents(ctx context.Context, f store.EventFilter) ([]models.PromptEvent, error) {
	query := `SELECT id, timestamp, agent, session_id, prompt, project, git_branch, git_remote, working_dir, raw_source, created_at
	          FROM prompt_events WHERE 1=1`
	var args []any

	if f.Since != nil {
		query += " AND timestamp >= ?"
		args = append(args, f.Since.UTC())
	}
	if f.Until != nil {
		query += " AND timestamp <= ?"
		args = append(args, f.Until.UTC())
	}
	if f.Agent != "" {
		query += " AND agent = ?"
		args = append(args, f.Agent)
	}
	if f.Project != "" {
		query += " AND project = ?"
		args = append(args, f.Project)
	}
	query += " ORDER BY timestamp DESC"
	if f.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, f.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var events []models.PromptEvent
	for rows.Next() {
		var e models.PromptEvent
		if err := rows.Scan(
			&e.ID, &e.Timestamp, &e.Agent, &e.SessionID, &e.Prompt,
			&e.Project, &e.GitBranch, &e.GitRemote, &e.WorkingDir, &e.RawSource, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// -- Classifications --

func (s *Store) InsertClassification(ctx context.Context, c *models.Classification) error {
	if c.ID == "" {
		c.ID = newID()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO classifications (id, prompt_event_id, category, confidence, classifier, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		c.ID, c.PromptEventID, c.Category, c.Confidence, c.Classifier, c.CreatedAt.UTC(),
	)
	return err
}

func (s *Store) GetUnclassified(ctx context.Context, limit int) ([]models.PromptEvent, error) {
	query := `SELECT id, timestamp, agent, session_id, prompt, project, git_branch, git_remote, working_dir, raw_source, created_at
	          FROM prompt_events
	          WHERE id NOT IN (SELECT DISTINCT prompt_event_id FROM classifications)
	          ORDER BY timestamp DESC`
	var args []any
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var events []models.PromptEvent
	for rows.Next() {
		var e models.PromptEvent
		if err := rows.Scan(
			&e.ID, &e.Timestamp, &e.Agent, &e.SessionID, &e.Prompt,
			&e.Project, &e.GitBranch, &e.GitRemote, &e.WorkingDir, &e.RawSource, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// -- Sessions --

func (s *Store) UpsertSession(ctx context.Context, sess *models.Session) error {
	if sess.ID == "" {
		sess.ID = newID()
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, project, agent, started_at, ended_at, prompt_count, gap_threshold_minutes)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   ended_at            = excluded.ended_at,
		   prompt_count        = excluded.prompt_count,
		   gap_threshold_minutes = excluded.gap_threshold_minutes`,
		sess.ID, sess.Project, sess.Agent,
		sess.StartedAt.UTC(), sess.EndedAt.UTC(),
		sess.PromptCount, sess.GapThresholdMinutes,
	)
	return err
}

func (s *Store) QuerySessions(ctx context.Context, f store.SessionFilter) ([]models.Session, error) {
	query := `SELECT id, project, agent, started_at, ended_at, prompt_count, gap_threshold_minutes
	          FROM sessions WHERE 1=1`
	var args []any

	if f.Since != nil {
		query += " AND started_at >= ?"
		args = append(args, f.Since.UTC())
	}
	if f.Until != nil {
		query += " AND ended_at <= ?"
		args = append(args, f.Until.UTC())
	}
	if f.Project != "" {
		query += " AND project = ?"
		args = append(args, f.Project)
	}
	if f.Agent != "" {
		query += " AND agent = ?"
		args = append(args, f.Agent)
	}
	query += " ORDER BY started_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var sessions []models.Session
	for rows.Next() {
		var sess models.Session
		if err := rows.Scan(
			&sess.ID, &sess.Project, &sess.Agent,
			&sess.StartedAt, &sess.EndedAt,
			&sess.PromptCount, &sess.GapThresholdMinutes,
		); err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

// -- Projects --

func (s *Store) UpsertProject(ctx context.Context, p *models.Project) error {
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO projects (name, display_name, git_remotes, paths, created_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(name) DO UPDATE SET
		   display_name = excluded.display_name,
		   git_remotes  = excluded.git_remotes,
		   paths        = excluded.paths`,
		p.Name, p.DisplayName,
		joinStrings(p.GitRemotes), joinStrings(p.Paths),
		p.CreatedAt.UTC(),
	)
	return err
}

func (s *Store) ListProjects(ctx context.Context) ([]models.Project, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT name, display_name, git_remotes, paths, created_at FROM projects ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var projects []models.Project
	for rows.Next() {
		var p models.Project
		var gitRemotes, paths string
		if err := rows.Scan(&p.Name, &p.DisplayName, &gitRemotes, &paths, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.GitRemotes = splitStrings(gitRemotes)
		p.Paths = splitStrings(paths)
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// -- helpers --

func newID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// joinStrings serialises a string slice as a newline-delimited string for storage.
func joinStrings(ss []string) string {
	var b strings.Builder
	for i, s := range ss {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(s)
	}
	return b.String()
}

// splitStrings deserialises a newline-delimited string back into a slice.
func splitStrings(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

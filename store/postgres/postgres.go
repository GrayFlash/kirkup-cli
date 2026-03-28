package postgres

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/GrayFlash/kirkup-cli/models"
	"github.com/GrayFlash/kirkup-cli/store"
)

type Store struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS prompt_events (
	id          TEXT PRIMARY KEY,
	timestamp   TIMESTAMP NOT NULL,
	agent       TEXT NOT NULL,
	session_id  TEXT,
	prompt      TEXT NOT NULL,
	project     TEXT,
	git_branch  TEXT,
	git_remote  TEXT,
	working_dir TEXT,
	raw_source  TEXT,
	created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_prompt_events_timestamp ON prompt_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_prompt_events_agent     ON prompt_events(agent);
CREATE INDEX IF NOT EXISTS idx_prompt_events_project   ON prompt_events(project);

CREATE TABLE IF NOT EXISTS classifications (
	id              TEXT PRIMARY KEY,
	prompt_event_id TEXT NOT NULL UNIQUE REFERENCES prompt_events(id),
	category        TEXT NOT NULL,
	confidence      REAL NOT NULL DEFAULT 1.0,
	classifier      TEXT NOT NULL DEFAULT 'rules-v1',
	created_at      TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_classifications_event    ON classifications(prompt_event_id);
CREATE INDEX IF NOT EXISTS idx_classifications_category ON classifications(category);

CREATE TABLE IF NOT EXISTS sessions (
	id                     TEXT PRIMARY KEY,
	project                TEXT,
	agent                  TEXT,
	started_at             TIMESTAMP NOT NULL,
	ended_at               TIMESTAMP NOT NULL,
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
	created_at   TIMESTAMP NOT NULL DEFAULT NOW()
);
`

func Open(dsn string) (*Store, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

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
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 ON CONFLICT DO NOTHING`,
		e.ID, e.Timestamp, e.Agent, e.SessionID, e.Prompt, e.Project, e.GitBranch, e.GitRemote, e.WorkingDir, e.RawSource, e.CreatedAt,
	)
	return err
}

func (s *Store) QueryPromptEvents(ctx context.Context, f store.EventFilter) ([]models.PromptEvent, error) {
	query := `SELECT id, timestamp, agent, session_id, prompt, project, git_branch, git_remote, working_dir, raw_source, created_at 
	          FROM prompt_events WHERE 1=1`
	var args []any
	i := 1

	if f.Since != nil {
		query += fmt.Sprintf(" AND timestamp >= $%d", i)
		args = append(args, *f.Since)
		i++
	}
	if f.Until != nil {
		query += fmt.Sprintf(" AND timestamp <= $%d", i)
		args = append(args, *f.Until)
		i++
	}
	if f.Agent != "" {
		query += fmt.Sprintf(" AND agent = $%d", i)
		args = append(args, f.Agent)
		i++
	}
	if f.Project != "" {
		query += fmt.Sprintf(" AND project = $%d", i)
		args = append(args, f.Project)
		i++
	}

	query += " ORDER BY timestamp DESC"
	if f.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", i)
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
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Agent, &e.SessionID, &e.Prompt, &e.Project, &e.GitBranch, &e.GitRemote, &e.WorkingDir, &e.RawSource, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (s *Store) ListEventIDs(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id FROM prompt_events")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *Store) InsertClassification(ctx context.Context, c *models.Classification) error {
	if c.ID == "" {
		c.ID = newID()
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO classifications (id, prompt_event_id, category, confidence, classifier, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (prompt_event_id) DO UPDATE SET
		 category = EXCLUDED.category, confidence = EXCLUDED.confidence, 
		 classifier = EXCLUDED.classifier, created_at = EXCLUDED.created_at`,
		c.ID, c.PromptEventID, c.Category, c.Confidence, c.Classifier, c.CreatedAt,
	)
	return err
}

func (s *Store) QueryClassifications(ctx context.Context, eventIDs []string) ([]models.Classification, error) {
	if len(eventIDs) == 0 {
		return nil, nil
	}

	var all []models.Classification
	for i := 0; i < len(eventIDs); i += 100 {
		end := i + 100
		if end > len(eventIDs) {
			end = len(eventIDs)
		}
		batch := eventIDs[i:end]
		placeholders := make([]string, len(batch))
		args := make([]any, len(batch))
		for j, id := range batch {
			placeholders[j] = fmt.Sprintf("$%d", j+1)
			args[j] = id
		}
		query := fmt.Sprintf(`SELECT id, prompt_event_id, category, confidence, classifier, created_at
		                      FROM classifications WHERE prompt_event_id IN (%s)`, strings.Join(placeholders, ","))
		
		rows, err := s.db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var c models.Classification
			if err := rows.Scan(&c.ID, &c.PromptEventID, &c.Category, &c.Confidence, &c.Classifier, &c.CreatedAt); err != nil {
				_ = rows.Close()
				return nil, err
				}
				all = append(all, c)
				}
				_ = rows.Close()
				}
				return all, nil}

func (s *Store) GetUnclassified(ctx context.Context, limit int) ([]models.PromptEvent, error) {
	query := `SELECT e.id, e.timestamp, e.agent, e.session_id, e.prompt, e.project, e.git_branch, e.git_remote, e.working_dir, e.raw_source, e.created_at
	          FROM prompt_events e
	          LEFT JOIN classifications c ON e.id = c.prompt_event_id
	          WHERE c.id IS NULL
	          ORDER BY e.timestamp DESC`
	var args []any
	if limit > 0 {
		query += " LIMIT $1"
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
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Agent, &e.SessionID, &e.Prompt, &e.Project, &e.GitBranch, &e.GitRemote, &e.WorkingDir, &e.RawSource, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (s *Store) UpsertSession(ctx context.Context, sess *models.Session) error {
	if sess.ID == "" {
		sess.ID = newID()
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, project, agent, started_at, ended_at, prompt_count, gap_threshold_minutes)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (id) DO UPDATE SET
		 project = EXCLUDED.project, agent = EXCLUDED.agent, 
		 started_at = EXCLUDED.started_at, ended_at = EXCLUDED.ended_at, 
		 prompt_count = EXCLUDED.prompt_count, gap_threshold_minutes = EXCLUDED.gap_threshold_minutes`,
		sess.ID, sess.Project, sess.Agent, sess.StartedAt, sess.EndedAt, sess.PromptCount, sess.GapThresholdMinutes,
	)
	return err
}

func (s *Store) QuerySessions(ctx context.Context, f store.SessionFilter) ([]models.Session, error) {
	query := `SELECT id, project, agent, started_at, ended_at, prompt_count, gap_threshold_minutes FROM sessions WHERE 1=1`
	var args []any
	i := 1

	if f.Since != nil {
		query += fmt.Sprintf(" AND started_at >= $%d", i)
		args = append(args, *f.Since)
		i++
	}
	if f.Until != nil {
		query += fmt.Sprintf(" AND ended_at <= $%d", i)
		args = append(args, *f.Until)
		i++
	}
	if f.Project != "" {
		query += fmt.Sprintf(" AND project = $%d", i)
		args = append(args, f.Project)
		i++
	}
	if f.Agent != "" {
		query += fmt.Sprintf(" AND agent = $%d", i)
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
		if err := rows.Scan(&sess.ID, &sess.Project, &sess.Agent, &sess.StartedAt, &sess.EndedAt, &sess.PromptCount, &sess.GapThresholdMinutes); err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

func (s *Store) UpsertProject(ctx context.Context, p *models.Project) error {
	gitRemotes := strings.Join(p.GitRemotes, "\n")
	paths := strings.Join(p.Paths, "\n")
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO projects (name, display_name, git_remotes, paths)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (name) DO UPDATE SET
		 display_name = EXCLUDED.display_name, git_remotes = EXCLUDED.git_remotes, paths = EXCLUDED.paths`,
		p.Name, p.DisplayName, gitRemotes, paths,
	)
	return err
}

func (s *Store) ListProjects(ctx context.Context) ([]models.Project, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT name, display_name, git_remotes, paths, created_at FROM projects ORDER BY name ASC")
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

func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func splitStrings(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

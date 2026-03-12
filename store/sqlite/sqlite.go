package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

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
	working_dir TEXT
);

CREATE TABLE IF NOT EXISTS classifications (
	id              TEXT PRIMARY KEY,
	prompt_event_id TEXT NOT NULL REFERENCES prompt_events(id),
	category        TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_prompt_events_timestamp ON prompt_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_prompt_events_agent     ON prompt_events(agent);
CREATE INDEX IF NOT EXISTS idx_classifications_event   ON classifications(prompt_event_id);
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

func (s *Store) InsertPromptEvent(ctx context.Context, e *models.PromptEvent) error {
	if e.ID == "" {
		e.ID = newID()
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO prompt_events (id, timestamp, agent, session_id, prompt, working_dir)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		e.ID, e.Timestamp, e.Agent, e.SessionID, e.Prompt, e.WorkingDir,
	)
	return err
}

func (s *Store) QueryPromptEvents(ctx context.Context, f store.EventFilter) ([]models.PromptEvent, error) {
	query := `SELECT id, timestamp, agent, session_id, prompt, working_dir
	          FROM prompt_events WHERE 1=1`
	args := []any{}

	if f.Since != nil {
		query += " AND timestamp >= ?"
		args = append(args, f.Since)
	}
	if f.Until != nil {
		query += " AND timestamp <= ?"
		args = append(args, f.Until)
	}
	if f.Agent != "" {
		query += " AND agent = ?"
		args = append(args, f.Agent)
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
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Agent, &e.SessionID, &e.Prompt, &e.WorkingDir); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (s *Store) InsertClassification(ctx context.Context, c *models.Classification) error {
	if c.ID == "" {
		c.ID = newID()
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO classifications (id, prompt_event_id, category) VALUES (?, ?, ?)`,
		c.ID, c.PromptEventID, c.Category,
	)
	return err
}

func (s *Store) Close() error {
	return s.db.Close()
}

func newID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

# Store

Package: `store/` — interface + SQLite implementation.

## Interface

```go
type Store interface {
    InsertPromptEvent(ctx, *PromptEvent) error
    QueryPromptEvents(ctx, EventFilter) ([]PromptEvent, error)
    InsertClassification(ctx, *Classification) error
    Migrate(ctx) error
    Close() error
}
```

## EventFilter

```go
type EventFilter struct {
    Since *time.Time
    Until *time.Time
    Agent string
    Limit int
}
```

## SQLite Implementation

Package: `store/sqlite/`
Driver: `modernc.org/sqlite` (pure Go, no CGo)

- `Open(path)` — creates the DB directory if missing, opens the connection
- `Migrate()` — runs schema inline (no migration files for now)
- IDs are generated using `crypto/rand` (no external UUID dep)
- Schema lives as a `const` string in `sqlite.go`

### Schema

```sql
CREATE TABLE prompt_events (
    id, timestamp, agent, session_id, prompt, working_dir
);
CREATE TABLE classifications (
    id, prompt_event_id, category
);
```

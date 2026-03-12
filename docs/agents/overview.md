# Agent Adapters

Package: `agent/`

## Interface

```go
type Adapter interface {
    Name() string
    Detect() bool
    WatchGlobs() []string
    Events(ctx context.Context, path string) ([]models.PromptEvent, error)
}
```

- `WatchGlobs()` — glob patterns the collector watches for file changes
- `Events(ctx, path)` — reads all prompt events from a specific file; collector handles deduplication
- `Detect()` — returns true if the agent is installed on the system

## Registry

`agent.NewRegistry(adapters...)` holds all adapters. `Detected()` filters to installed ones.

## Implemented Adapters

### gemini-cli (`agent/gemini/`)

| Property | Value |
|----------|-------|
| Watch | `~/.gemini/tmp/*/logs.json` |
| Format | JSON array |
| SessionID | `sessionId` field |
| WorkingDir | Sibling `.project_root` file |

`logs.json` is written automatically when `sessionRetention.enabled: true` in `~/.gemini/settings.json`. No telemetry opt-in required.

Filters to entries where `type == "user"`.

### cursor (`agent/cursor/`)

| Property | Value |
|----------|-------|
| Watch | `~/.config/Cursor/User/workspaceStorage/*/state.vscdb` |
| Format | SQLite (`ItemTable` → key `aiService.generations`) |
| SessionID | `generationUUID` |
| WorkingDir | Sibling `workspace.json` → `folder` field (strips `file://` prefix) |

## Adding a New Adapter

1. Create `agent/<name>/` package
2. Implement the `agent.Adapter` interface
3. Register it in `main.go` via `agent.NewRegistry(...)`

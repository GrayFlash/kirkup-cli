# Architecture Overview

Flat, plugin-driven. Package by feature, not by layer.

## Package Structure

```
kirkup/
├── main.go          # Entry point, wires dependencies
├── cmd/             # Cobra CLI commands (thin — no business logic)
├── agent/           # Adapter interface + per-agent implementations
├── store/           # Store interface + SQLite/Postgres implementations
├── classifier/      # Classifier interface + rule-based/LLM implementations
├── collector/       # Daemon lifecycle + fsnotify file watchers
├── retro/           # Aggregation, summary logic, terminal rendering
├── config/          # YAML config loading + defaults
└── models/          # Shared domain types (stdlib only, no external deps)
```

## Key Conventions

- Interfaces live next to their implementations, not in a separate ports layer
- `models/` is the only shared package — keep it dependency-free
- Adding a new agent = one new sub-package implementing `agent.Adapter`
- Adding a new store = one new sub-package implementing `store.Store`

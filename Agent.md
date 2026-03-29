# Agent Instructions

## Documentation

- Always load documentation for context at startup.
- Project spec managed in `issues/kirkup-spec.md`
- Project Status managed in `issues/v0.1-scope.md`
- Project related public docs wrriten to `docs`.
- Always confirm with user before writing to documentations.
- Keep code comments for documentation to minimal.

## Code Pattern

Golang based, Flat structured, plugin-driven.
Package by feature, not by layer.
Each package owns its interface, types, and implementations together.

## Conventions

- Interfaces live next to implementations, not in a separate ports layer
- `models/` is the only shared package — keep it dependency-free
- Adding a new agent or store = one new sub-package implementing the interface
- Technology: Go, Cobra, fsnotify, SQLite + Postgres, YAML config

## Directory structure

```md
kirkup-cli/
├── main.go                        # Entry point, wires dependencies
├── cmd/kirkup/                    # Cobra CLI commands (thin — no business logic)
├── agent/                         # AgentAdapter interface + implementations
│   ├── agent.go                   # Interface + shared types
│   ├── registry.go                # Agent discovery & registration
│   ├── gemini/                    # Gemini CLI log parser
│   └── cursor/                    # Cursor log parser
├── store/                         # Store interface + implementations
│   ├── store.go                   # Interface + query filter types
│   ├── migrations/
│   ├── sqlite/
│   └── postgres/
├── classifier/                    # Classifier interface + implementations
│   ├── classifier.go              # Interface + category types
│   ├── rules.go                   # Rule-based classifier
│   └── llm.go                     # LLM-based classifier (V2)
├── collector/                     # Daemon lifecycle + fsnotify file watchers
├── retro/                         # Aggregation, summary logic, terminal rendering
├── config/                        # YAML config parsing
├── models/                        # Shared domain types (stdlib only, no deps)
├── config/defaults/default.yaml           # Default config template
├── go.mod
└── .goreleaser.yaml
```

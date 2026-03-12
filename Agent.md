# Agent Instructions

Full spec: `issues/kirkup-spec.md`

## Architecture

Flat, plugin-driven. Package by feature, not by layer. Each package owns its interface, types, and implementations together.

```md
kirkup/
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
├── configs/default.yaml           # Default config template
├── go.mod
└── .goreleaser.yaml
```

## Conventions

- Interfaces live next to implementations, not in a separate ports layer
- `models/` is the only shared package — keep it dependency-free
- Adding a new agent or store = one new sub-package implementing the interface
- Go, Cobra, fsnotify, SQLite + Postgres, YAML config

# Data Models

Package: `models/` — stdlib only, no external dependencies.

## PromptEvent

The atomic unit of collected data. All fields are directly extractable from agent log files — no enrichment or derivation.

| Field | Type | Source |
|-------|------|--------|
| `ID` | string | Generated at insert time |
| `Timestamp` | time.Time | From agent log |
| `Agent` | string | Agent identifier (e.g. `gemini-cli`, `cursor`) |
| `SessionID` | string | From agent log (empty if unavailable) |
| `Prompt` | string | The user's prompt text |
| `WorkingDir` | string | Resolved from agent log metadata |

Fields intentionally omitted:
- `GitBranch` — not available from any agent log without extra git calls
- `GitRemote` / `Project` — derived concepts, added later during enrichment

## Classification

A category tag applied to a `PromptEvent`. One event can have many classifications.

| Field | Type | Notes |
|-------|------|-------|
| `ID` | string | Generated at insert time |
| `PromptEventID` | string | FK to `PromptEvent.ID` |
| `Category` | string | e.g. `coding`, `debugging`, `refactoring` |

`Category` is a plain string for now. Typed constants will be added when the classifier is built.

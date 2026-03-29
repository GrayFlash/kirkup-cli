# kirkup

Reads local log files from your AI coding agents and gives you a weekly retrospective from the terminal.

Named after 記録 (*kiroku*) — Japanese for "record."

> All data stays local — nothing leaves your machine.

---

## How it works

kirkup watches the log files written by your AI agents in the background. When you run a prompt in Gemini CLI, Cursor, or Claude Code, kirkup captures it along with git context (branch, remote) and the project it belongs to. No shell wrappers. No manual logging.

At the end of the week, run `kirkup retro` to see what you actually worked on.

---

## Supported agents

| Agent       | Status |
|-------------|--------|
| Gemini CLI  | ✅     |
| Cursor      | ✅     |
| Claude Code | ✅     |

---

## Install

**Homebrew**
```sh
brew tap GrayFlash/taps
brew install kirkup
```

**Go**
```sh
go install github.com/GrayFlash/kirkup-cli@latest
```

---

## Quick start

```sh
kirkup init       # create config, init DB, detect agents
kirkup start      # start the background collector
kirkup classify   # tag events by category (rule-based, LLM support coming soon)
kirkup retro      # view this week's summary
```

---

## Commands

| Command              | Description                                      |
|----------------------|--------------------------------------------------|
| `kirkup init`        | Set up config and database                       |
| `kirkup start`       | Start the collector daemon                       |
| `kirkup stop`        | Stop the collector daemon                        |
| `kirkup status`      | Show daemon status and today's activity          |
| `kirkup retro`       | Weekly summary (use `--month`, `--from/--to`)    |
| `kirkup events`      | View raw events (`--today`, `--tail`)            |
| `kirkup classify`    | Run classifier on unclassified events            |
| `kirkup export`      | Export data as JSON                              |
| `kirkup agents`      | List supported agents and detection status       |
| `kirkup tui`         | Open the terminal dashboard                      |

---

## Configuration

Config lives at `~/.kirkup/config.yaml` after `kirkup init`.

```yaml
store:
  driver: sqlite
  sqlite:
    path: ~/.kirkup/kirkup.db

agents:
  gemini-cli:
    enabled: true
  cursor:
    enabled: true
  claude-code:
    enabled: true

classifier:
  mode: rules   # rules | llm | both

sessions:
  gap_threshold_minutes: 30
```

---

## License

MIT

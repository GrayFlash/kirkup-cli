package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/GrayFlash/kirkup-cli/models"
)

// Adapter collects prompt events from Claude Code JSONL session logs.
type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) Name() string { return "claude-code" }

func (a *Adapter) Detect() bool {
	base, ok := claudeBase()
	if !ok {
		return false
	}
	_, err := os.Stat(filepath.Join(base, "projects"))
	return err == nil
}

func (a *Adapter) WatchGlobs() []string {
	base, ok := claudeBase()
	if !ok {
		return nil
	}
	return []string{filepath.Join(base, "projects", "*", "*.jsonl")}
}

// logEntry represents a single line in a Claude Code JSONL session file.
type logEntry struct {
	Type        string    `json:"type"`
	UUID        string    `json:"uuid"`
	Timestamp   time.Time `json:"timestamp"`
	CWD         string    `json:"cwd"`
	SessionID   string    `json:"sessionId"`
	GitBranch   string    `json:"gitBranch"`
	IsSidechain bool      `json:"isSidechain"`
	Message     struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

func (a *Adapter) Events(_ context.Context, path string) ([]models.PromptEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var events []models.PromptEvent
	scanner := bufio.NewScanner(f)
	// Claude JSONL lines can be long; increase buffer to 1 MB.
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry logEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		if entry.Type != "user" || entry.IsSidechain {
			continue
		}

		prompt := extractPrompt(entry.Message.Content)
		if prompt == "" {
			continue
		}

		events = append(events, models.PromptEvent{
			Timestamp:  entry.Timestamp,
			Agent:      a.Name(),
			SessionID:  entry.SessionID,
			Prompt:     prompt,
			WorkingDir: entry.CWD,
			GitBranch:  entry.GitBranch,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

// extractPrompt pulls the human-readable text from a message content field.
// Content may be a plain string or a JSON array of content blocks.
// Returns "" when the content contains only tool_result blocks (no real prompt).
func extractPrompt(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// Try plain string first.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}

	// Try array of content blocks.
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}

	for _, b := range blocks {
		if b.Type == "text" && b.Text != "" {
			return b.Text
		}
	}
	return ""
}

// claudeBase returns the ~/.claude directory.
func claudeBase() (string, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	return filepath.Join(home, ".claude"), true
}

package gemini

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GrayFlash/kirkup-cli/models"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) Name() string { return "gemini-cli" }

func (a *Adapter) Detect() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(home, ".gemini"))
	return err == nil
}

func (a *Adapter) WatchGlobs() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{filepath.Join(home, ".gemini", "tmp", "*", "logs.json")}
}

func (a *Adapter) Events(ctx context.Context, path string) ([]models.PromptEvent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var log struct {
		Entries []logEntry `json:"entries"`
	}
	if err := json.Unmarshal(data, &log); err != nil {
		return nil, err
	}

	cwd := readProjectRoot(path)

	var events []models.PromptEvent
	for _, e := range log.Entries {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if e.Role != "user" {
			continue
		}
		events = append(events, models.PromptEvent{
			Agent:      "gemini-cli",
			Timestamp:  e.timestamp(),
			Prompt:     e.Content,
			WorkingDir: cwd,
			RawSource:  e.SessionID,
		})
	}
	return events, nil
}

type logEntry struct {
	SessionID string `json:"sessionId"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	Time      string `json:"time"` // RFC3339
}

func (e logEntry) timestamp() time.Time {
	t, _ := time.Parse(time.RFC3339, e.Time)
	return t
}

func readProjectRoot(logsPath string) string {
	data, err := os.ReadFile(filepath.Join(filepath.Dir(logsPath), ".project_root"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

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

type logEntry struct {
	SessionID string `json:"sessionId"`
	Type      string `json:"type"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

func (a *Adapter) Events(_ context.Context, path string) ([]models.PromptEvent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var entries []logEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}

	cwd := readProjectRoot(path)

	var events []models.PromptEvent
	for _, e := range entries {
		if e.Type != "user" || e.Message == "" {
			continue
		}
		ts, err := time.Parse(time.RFC3339, e.Timestamp)
		if err != nil {
			continue
		}
		events = append(events, models.PromptEvent{
			Timestamp:  ts,
			Agent:      a.Name(),
			SessionID:  e.SessionID,
			Prompt:     e.Message,
			WorkingDir: cwd,
		})
	}
	return events, nil
}

// readProjectRoot reads the cwd from the .project_root file
// that Gemini CLI writes alongside logs.json.
func readProjectRoot(logsPath string) string {
	data, err := os.ReadFile(filepath.Join(filepath.Dir(logsPath), ".project_root"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

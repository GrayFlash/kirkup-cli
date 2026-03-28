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

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) Name() string { return "claude-code" }

func (a *Adapter) Detect() bool {
	base, ok := claudeBase()
	if !ok {
		return false
	}
	_, err := os.Stat(base)
	return err == nil
}

func (a *Adapter) WatchGlobs() []string {
	base, ok := claudeBase()
	if !ok {
		return nil
	}
	// Claude Code stores logs in projects folders
	return []string{filepath.Join(base, "projects", "*", "conversation.jsonl")}
}

func (a *Adapter) Events(ctx context.Context, path string) ([]models.PromptEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var events []models.PromptEvent
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		var msg chatMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}
		if msg.Role != "user" {
			continue
		}
		events = append(events, models.PromptEvent{
			Agent:     "claude-code",
			Timestamp: msg.timestamp(),
			Prompt:    msg.Content,
			RawSource: msg.ProjectID,
		})
	}
	return events, scanner.Err()
}

type chatMessage struct {
	ProjectID string `json:"projectId"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

func (m chatMessage) timestamp() time.Time {
	t, _ := time.Parse(time.RFC3339, m.CreatedAt)
	return t
}

func claudeBase() (string, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	return filepath.Join(home, ".claude"), true
}

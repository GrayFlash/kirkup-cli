package generic

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/GrayFlash/kirkup-cli/config"
	"github.com/GrayFlash/kirkup-cli/models"
)

type Adapter struct {
	name string
	cfg  config.AgentConfig
}

func New(name string, cfg config.AgentConfig) *Adapter {
	return &Adapter{name: name, cfg: cfg}
}

func (a *Adapter) Name() string { return a.name }
func (a *Adapter) Detect() bool { return true }
func (a *Adapter) WatchGlobs() []string { return a.cfg.LogPaths }

func (a *Adapter) Events(ctx context.Context, path string) ([]models.PromptEvent, error) {
	if a.cfg.Format == "json" {
		return a.parseJSON(ctx, path)
	}
	return a.parseJSONL(ctx, path)
}

func (a *Adapter) parseJSON(ctx context.Context, path string) ([]models.PromptEvent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	// Handle both single object and array of objects.
	switch v := raw.(type) {
	case map[string]any:
		e, ok := a.mapEvent(v)
		if !ok {
			return nil, nil
		}
		return []models.PromptEvent{e}, nil
	case []any:
		var events []models.PromptEvent
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				if e, ok := a.mapEvent(m); ok {
					events = append(events, e)
				}
			}
		}
		return events, nil
	default:
		return nil, nil
	}
}

func (a *Adapter) parseJSONL(ctx context.Context, path string) ([]models.PromptEvent, error) {
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
		var m map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &m); err != nil {
			continue
		}
		if e, ok := a.mapEvent(m); ok {
			events = append(events, e)
		}
	}
	return events, scanner.Err()
}

func (a *Adapter) mapEvent(m map[string]any) (models.PromptEvent, bool) {
	// Role check
	if a.cfg.RoleField != "" && a.cfg.UserRoleValue != "" {
		role, _ := m[a.cfg.RoleField].(string)
		if role != a.cfg.UserRoleValue {
			return models.PromptEvent{}, false
		}
	}

	prompt, _ := m[a.cfg.PromptField].(string)
	if prompt == "" {
		return models.PromptEvent{}, false
	}

	e := models.PromptEvent{
		Agent:  a.name,
		Prompt: prompt,
	}

	if a.cfg.SessionIDField != "" {
		if sid, ok := m[a.cfg.SessionIDField]; ok {
			e.SessionID = fmt.Sprint(sid)
			e.RawSource = fmt.Sprint(sid)
		}
	}

	if a.cfg.TimestampField != "" {
		if val, ok := m[a.cfg.TimestampField]; ok {
			e.Timestamp = a.parseTime(val)
		}
	}

	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}

	return e, true
}

func (a *Adapter) parseTime(val any) time.Time {
	switch v := val.(type) {
	case string:
		// Try common formats
		for _, layout := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02 15:04:05", "2006-01-02"} {
			if t, err := time.Parse(layout, v); err == nil {
				return t
			}
		}
	case float64:
		// Assume Unix timestamp (seconds or milliseconds)
		if v > 1e12 { // Milliseconds
			return time.Unix(int64(v)/1000, (int64(v)%1000)*1000000)
		}
		return time.Unix(int64(v), 0)
	}
	return time.Time{}
}

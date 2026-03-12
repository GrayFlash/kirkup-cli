package cursor

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/GrayFlash/kirkup-cli/models"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) Name() string { return "cursor" }

func (a *Adapter) Detect() bool {
	home, _ := os.UserHomeDir()
	_, err := os.Stat(filepath.Join(home, ".config", "Cursor"))
	return err == nil
}

func (a *Adapter) WatchGlobs() []string {
	home, _ := os.UserHomeDir()
	return []string{
		filepath.Join(home, ".config", "Cursor", "User", "workspaceStorage", "*", "state.vscdb"),
	}
}

type generation struct {
	UnixMs          int64  `json:"unixMs"`
	GenerationUUID  string `json:"generationUUID"`
	TextDescription string `json:"textDescription"`
}

func (a *Adapter) Events(_ context.Context, path string) ([]models.PromptEvent, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()

	var raw string
	err = db.QueryRow(`SELECT value FROM ItemTable WHERE key = 'aiService.generations'`).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var generations []generation
	if err := json.Unmarshal([]byte(raw), &generations); err != nil {
		return nil, err
	}

	cwd := readWorkspaceFolder(path)

	var events []models.PromptEvent
	for _, g := range generations {
		if g.TextDescription == "" {
			continue
		}
		events = append(events, models.PromptEvent{
			Timestamp:  time.UnixMilli(g.UnixMs),
			Agent:      a.Name(),
			SessionID:  g.GenerationUUID,
			Prompt:     g.TextDescription,
			WorkingDir: cwd,
		})
	}
	return events, nil
}

// readWorkspaceFolder resolves the project directory from the
// workspace.json file that Cursor writes alongside state.vscdb.
func readWorkspaceFolder(dbPath string) string {
	data, err := os.ReadFile(filepath.Join(filepath.Dir(dbPath), "workspace.json"))
	if err != nil {
		return ""
	}
	var ws struct {
		Folder string `json:"folder"`
	}
	if err := json.Unmarshal(data, &ws); err != nil {
		return ""
	}
	// folder is a file URI: "file:///path/to/project"
	return strings.TrimPrefix(ws.Folder, "file://")
}

package cursor

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
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
	base, ok := cursorBase()
	if !ok {
		return false
	}
	_, err := os.Stat(filepath.Join(base, "chats"))
	return err == nil
}

func (a *Adapter) WatchGlobs() []string {
	base, ok := cursorBase()
	if !ok {
		return nil
	}
	return []string{
		filepath.Join(base, "chats", "*", "*", "store.db"),
	}
}

// chatMeta mirrors the JSON stored in the meta table under key "0".
type chatMeta struct {
	AgentID       string `json:"agentId"`
	Name          string `json:"name"`
	CreatedAt     int64  `json:"createdAt"`
	Mode          string `json:"mode"`
	LastUsedModel string `json:"lastUsedModel"`
}

type chatMessage struct {
	Role            string          `json:"role"`
	Content         json.RawMessage `json:"content"`
	ProviderOptions json.RawMessage `json:"providerOptions,omitempty"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type providerOpts struct {
	Cursor struct {
		RequestID string `json:"requestId"`
	} `json:"cursor"`
}

func (a *Adapter) Events(_ context.Context, path string) ([]models.PromptEvent, error) {
	db, err := sql.Open("sqlite", path+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()

	meta, err := readChatMeta(db)
	if err != nil {
		return nil, err
	}

	cwd := resolveWorkspace(path)

	rows, err := db.Query("SELECT data FROM blobs")
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	var events []models.PromptEvent
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			continue
		}

		var msg chatMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		if msg.Role != "user" {
			continue
		}

		rawText := extractPromptText(msg.Content)
		if rawText == "" {
			continue
		}

		// If workspace resolution via MD5 failed, try to extract
		// Workspace Path from Cursor's <user_info> preamble.
		eventCwd := cwd
		if eventCwd == "" {
			eventCwd = extractWorkspacePath(rawText)
		}

		prompt := cleanPrompt(rawText)
		if prompt == "" {
			continue
		}

		sessionID := meta.AgentID
		if rid := extractRequestID(msg.ProviderOptions); rid != "" {
			sessionID = meta.AgentID + "/" + rid
		}

		events = append(events, models.PromptEvent{
			Timestamp:  time.UnixMilli(meta.CreatedAt),
			Agent:      a.Name(),
			SessionID:  meta.AgentID,
			Prompt:     prompt,
			WorkingDir: eventCwd,
			RawSource:  sessionID,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func readChatMeta(db *sql.DB) (chatMeta, error) {
	var raw []byte
	if err := db.QueryRow("SELECT value FROM meta WHERE key = '0'").Scan(&raw); err != nil {
		return chatMeta{}, err
	}

	var meta chatMeta
	if err := json.Unmarshal(raw, &meta); err == nil {
		return meta, nil
	}
	// Fallback: the value may be stored as hex-encoded bytes.
	decoded, err := hex.DecodeString(string(raw))
	if err != nil {
		return chatMeta{}, err
	}
	if err := json.Unmarshal(decoded, &meta); err != nil {
		return chatMeta{}, err
	}
	return meta, nil
}

func extractPromptText(content json.RawMessage) string {
	var blocks []contentBlock
	if err := json.Unmarshal(content, &blocks); err == nil {
		var texts []string
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				texts = append(texts, b.Text)
			}
		}
		return strings.Join(texts, "\n")
	}
	var text string
	if err := json.Unmarshal(content, &text); err == nil {
		return text
	}
	return ""
}

// extractWorkspacePath pulls the Workspace Path from Cursor's <user_info>
// preamble when the MD5-based workspace resolution fails.
func extractWorkspacePath(raw string) string {
	const marker = "Workspace Path: "
	idx := strings.Index(raw, marker)
	if idx < 0 {
		return ""
	}
	rest := raw[idx+len(marker):]
	if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
		rest = rest[:nl]
	}
	return strings.TrimSpace(rest)
}

// cleanPrompt strips Cursor's system wrapper tags from user messages.
// Cursor injects preamble blocks like <user_info>, <git_status>, <rules>,
// etc. If <user_query> tags are present, we extract only that section.
func cleanPrompt(s string) string {
	s = strings.TrimSpace(s)

	if start := strings.Index(s, "<user_query>"); start >= 0 {
		inner := s[start+len("<user_query>"):]
		if end := strings.Index(inner, "</user_query>"); end >= 0 {
			return strings.TrimSpace(inner[:end])
		}
		return strings.TrimSpace(inner)
	}

	// Strip known system preamble blocks.
	for _, tag := range []string{"user_info", "git_status", "rules", "agent_transcripts", "attached_files", "system_reminder", "agent_skills", "task_notification"} {
		s = stripXMLBlock(s, tag)
	}
	return strings.TrimSpace(s)
}

func stripXMLBlock(s, tag string) string {
	open := "<" + tag + ">"
	close := "</" + tag + ">"
	for {
		start := strings.Index(s, open)
		if start < 0 {
			// Also handle tags with attributes: <tag ...>
			start = strings.Index(s, "<"+tag+" ")
			if start < 0 {
				return s
			}
			// Find the closing > of the opening tag.
			gtPos := strings.Index(s[start:], ">")
			if gtPos < 0 {
				return s
			}
			open = s[start : start+gtPos+1]
		}
		end := strings.Index(s, close)
		if end < 0 {
			return s
		}
		s = s[:start] + s[end+len(close):]
	}
}

func extractRequestID(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var opts providerOpts
	if err := json.Unmarshal(raw, &opts); err != nil {
		return ""
	}
	return opts.Cursor.RequestID
}

// resolveWorkspace maps a store.db path back to the original workspace
// directory. Cursor stores chats under ~/.cursor/chats/<md5(workspace)>/…,
// so we scan ~/.cursor/projects/ to find a project whose decoded path
// produces the matching MD5.
func resolveWorkspace(dbPath string) string {
	wsHash := extractWorkspaceHash(dbPath)
	if wsHash == "" {
		return ""
	}

	base, ok := cursorBase()
	if !ok {
		return ""
	}

	entries, err := os.ReadDir(filepath.Join(base, "projects"))
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		wsPath := decodeDirName(entry.Name())
		if wsPath == "" {
			continue
		}
		h := md5.Sum([]byte(wsPath))
		if hex.EncodeToString(h[:]) == wsHash {
			return wsPath
		}
	}
	return ""
}

// extractWorkspaceHash pulls the workspace hash from a store.db path.
// Expected layout: …/chats/<wsHash>/<chatUUID>/store.db
func extractWorkspaceHash(dbPath string) string {
	dir := filepath.Dir(filepath.Dir(dbPath)) // …/chats/<wsHash>
	return filepath.Base(dir)
}

// decodeDirName converts a Cursor project directory name
// Dashes are ambiguous (original dash vs path separator), so we walk the
// filesystem to pick only splits that correspond to real directories.
func decodeDirName(encoded string) string {
	return tryDecodePath(encoded, "/")
}

func tryDecodePath(remaining, prefix string) string {
	if remaining == "" {
		return prefix
	}
	for i := range remaining {
		if remaining[i] != '-' {
			continue
		}
		seg := remaining[:i]
		if seg == "" {
			continue
		}
		candidate := filepath.Join(prefix, seg)
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			if result := tryDecodePath(remaining[i+1:], candidate); result != "" {
				return result
			}
		}
	}
	// Treat everything remaining as the final path component.
	candidate := filepath.Join(prefix, remaining)
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
}

func cursorBase() (string, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	dotCursor := filepath.Join(home, ".cursor")
	if _, err := os.Stat(dotCursor); err == nil {
		return dotCursor, true
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", false
	}
	return filepath.Join(dir, "Cursor"), true
}

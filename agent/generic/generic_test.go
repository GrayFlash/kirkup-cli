package generic

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/GrayFlash/kirkup-cli/config"
)

func TestGenericAdapter_JSONL(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "generic.jsonl")
	
	jsonl := `{"msg": "hello", "role": "user", "ts": "2026-01-01T10:00:00Z", "sid": "s1"}
{"msg": "bot reply", "role": "assistant", "ts": "2026-01-01T10:01:00Z", "sid": "s1"}
{"msg": "world", "role": "user", "ts": 1704016800, "sid": "s2"}`
	
	if err := os.WriteFile(logPath, []byte(jsonl), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.AgentConfig{
		Format:         "jsonl",
		PromptField:    "msg",
		RoleField:      "role",
		UserRoleValue:  "user",
		TimestampField: "ts",
		SessionIDField: "sid",
	}
	
	a := New("test-agent", cfg)
	events, err := a.Events(context.Background(), logPath)
	if err != nil {
		t.Fatal(err)
	}
	
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}
	
	if events[0].Prompt != "hello" {
		t.Errorf("expected prompt 'hello', got %q", events[0].Prompt)
	}
	if events[0].RawSource != "s1" {
		t.Errorf("expected session 's1', got %q", events[0].RawSource)
	}
	
	// Test unix timestamp parsing
	if events[1].Prompt != "world" {
		t.Errorf("expected prompt 'world', got %q", events[1].Prompt)
	}
}

func TestGenericAdapter_JSON(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "generic.json")
	
	jsonData := `[
		{"text": "cmd1", "user": true},
		{"text": "cmd2", "user": true}
	]`
	
	if err := os.WriteFile(logPath, []byte(jsonData), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.AgentConfig{
		Format:      "json",
		PromptField: "text",
		// Missing role field means skip role check
	}
	
	a := New("json-agent", cfg)
	events, err := a.Events(context.Background(), logPath)
	if err != nil {
		t.Fatal(err)
	}
	
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}
	
	if events[0].Prompt != "cmd1" {
		t.Errorf("expected 'cmd1', got %q", events[0].Prompt)
	}
}

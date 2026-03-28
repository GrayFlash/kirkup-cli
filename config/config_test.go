package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfig_Validation(t *testing.T) {
	// Create a temporary YAML file with invalid 0 and negative durations
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.yaml")
	
	yamlContent := []byte(`
daemon:
  poll_interval_seconds: 0
sessions:
  gap_threshold_minutes: -5
`)
	if err := os.WriteFile(path, yamlContent, 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Should clamp to a minimum of 1 to prevent time.NewTicker(0) panic
	if cfg.Daemon.PollIntervalSeconds <= 0 {
		t.Errorf("expected PollIntervalSeconds to be > 0, got %d", cfg.Daemon.PollIntervalSeconds)
	}
	if cfg.Sessions.GapThresholdMinutes <= 0 {
		t.Errorf("expected GapThresholdMinutes to be > 0, got %d", cfg.Sessions.GapThresholdMinutes)
	}
}

func TestConfig_MergeDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.yaml")
	
	// Write an empty config file
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify defaults were applied
	if cfg.Store.Driver != "sqlite" {
		t.Errorf("expected default driver sqlite, got %s", cfg.Store.Driver)
	}
	if cfg.Daemon.PollIntervalSeconds != 5 {
		t.Errorf("expected default poll interval 5, got %d", cfg.Daemon.PollIntervalSeconds)
	}
}

func TestConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.yaml")
	
	// Write malformed YAML
	if err := os.WriteFile(path, []byte("invalid:\n  - yaml\n    bad: indentation"), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("UserHomeDir not available")
	}

	path := "~/test/dir"
	expected := filepath.Join(home, "test/dir")
	
	result := ExpandHome(path)
	if result != expected {
		t.Errorf("ExpandHome() = %v, want %v", result, expected)
	}

	// Non-tilde paths should be unchanged
	path2 := "/absolute/path"
	result2 := ExpandHome(path2)
	if result2 != path2 {
		t.Errorf("ExpandHome() = %v, want %v", result2, path2)
	}
}
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

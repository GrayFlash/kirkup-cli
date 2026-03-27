package cursor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDetect(t *testing.T) {
	a := New()
	if !a.Detect() {
		t.Skip("Cursor not installed; skipping integration test")
	}
	t.Log("Detect: true")
}

func TestWatchGlobs(t *testing.T) {
	a := New()
	globs := a.WatchGlobs()
	if len(globs) == 0 {
		t.Fatal("expected at least one glob pattern")
	}
	for _, g := range globs {
		matches, err := filepath.Glob(g)
		if err != nil {
			t.Fatalf("glob error: %v", err)
		}
		t.Logf("pattern %s matched %d files", g, len(matches))
	}
}

func TestEventsIntegration(t *testing.T) {
	a := New()
	if !a.Detect() {
		t.Skip("Cursor not installed; skipping")
	}

	globs := a.WatchGlobs()
	if len(globs) == 0 {
		t.Fatal("no globs")
	}

	var totalEvents int
	for _, g := range globs {
		matches, _ := filepath.Glob(g)
		for _, m := range matches {
			events, err := a.Events(context.Background(), m)
			if err != nil {
				t.Logf("WARN: %s: %v", m, err)
				continue
			}
			for _, e := range events {
				truncated := e.Prompt
				if len(truncated) > 80 {
					truncated = truncated[:80] + "…"
				}
				t.Logf("[%s] session=%s cwd=%s prompt=%s",
					e.Timestamp.Format("2006-01-02"), e.SessionID, e.WorkingDir, truncated)
				totalEvents++
			}
		}
	}
	t.Logf("total events extracted: %d", totalEvents)
}

func TestDecodeDirName(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}
	encoded := "Users-" + filepath.Base(home)
	result := decodeDirName(encoded)
	if result != home {
		t.Errorf("decodeDirName(%q) = %q, want %q", encoded, result, home)
	}
}

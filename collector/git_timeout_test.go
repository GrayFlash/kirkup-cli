package collector

import (
	"os"
	"testing"
	"time"
)

func TestGitContext_Timeout(t *testing.T) {
	// A bit hacky: if we create a fake "git" that sleeps for 10s, 
	// the test should pass in ~10s total (5s + 5s) if the timeout works.
	
	tmp := t.TempDir()
	fakeGit := tmp + "/git"
	
	script := "#!/bin/sh\nsleep 10\n"
	if err := os.WriteFile(fakeGit, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write fake git: %v", err)
	}
	
	// Prepend our fake git to PATH
	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", tmp+":"+oldPath); err != nil {
		t.Fatalf("failed to set PATH: %v", err)
	}
	defer func() {
		if err := os.Setenv("PATH", oldPath); err != nil {
			t.Logf("failed to restore PATH: %v", err)
		}
	}()
	
	start := time.Now()
	
	// This will call gitRemote and gitBranch, which both should time out.
	info := GitContext(".")
	
	elapsed := time.Since(start)
	
	if elapsed > 15*time.Second {
		t.Fatalf("GitContext took %v, expected ~10s from timeouts", elapsed)
	}
	
	if info.Remote != "" || info.Branch != "" {
		t.Errorf("expected empty info due to timeout, got %+v", info)
	}
}

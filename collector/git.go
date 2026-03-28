package collector

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// GitInfo holds the git context resolved from a working directory.
type GitInfo struct {
	Remote string
	Branch string
}

// GitContext runs git commands in dir to resolve the origin remote URL
// and current branch. Returns zero-value GitInfo if dir is not a git repo
// or git is not available.
func GitContext(dir string) GitInfo {
	if dir == "" {
		return GitInfo{}
	}
	return GitInfo{
		Remote: gitRemote(dir),
		Branch: gitBranch(dir),
	}
}

func gitRemote(dir string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "remote", "get-url", "origin")
	cmd.WaitDelay = 100 * time.Millisecond
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	remote := strings.TrimSpace(string(out))
	return normaliseRemote(remote)
}

func gitBranch(dir string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "branch", "--show-current")
	cmd.WaitDelay = 100 * time.Millisecond
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// normaliseRemote strips protocol prefixes and .git suffix so remotes can be
// compared uniformly regardless of whether they use SSH or HTTPS.
//
//	git@github.com:example/repo.git → github.com/example/repo
//	https://github.com/example/repo.git → github.com/example/repo
func normaliseRemote(raw string) string {
	// SSH: git@github.com:example/repo.git
	if idx := strings.Index(raw, "@"); idx != -1 {
		raw = raw[idx+1:]
		raw = strings.Replace(raw, ":", "/", 1)
	}
	// HTTPS: https://github.com/...
	for _, prefix := range []string{"https://", "http://"} {
		raw = strings.TrimPrefix(raw, prefix)
	}
	raw = strings.TrimSuffix(raw, ".git")
	return raw
}

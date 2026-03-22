package collector

import (
	"testing"

	"github.com/GrayFlash/kirkup-cli/config"
)

var testProjects = []config.ProjectConfig{
	{
		Name:        "project-alpha",
		DisplayName: "Project Alpha",
		Match: config.ProjectMatch{
			GitRemote: "github.com/example/project-alpha",
			Paths:     []string{"/home/user/work/project-alpha"},
		},
	},
	{
		Name:        "project-beta",
		DisplayName: "Project Beta",
		Match: config.ProjectMatch{
			GitRemote: "github.com/example/project-beta",
			Paths:     []string{"/home/user/work/project-beta"},
		},
	},
}

func TestResolveProject_ByGitRemote(t *testing.T) {
	got := ResolveProject(testProjects, "github.com/example/project-alpha", "")
	if got != "project-alpha" {
		t.Errorf("want project-alpha, got %q", got)
	}
}

func TestResolveProject_ByGitRemoteNormalisedSSH(t *testing.T) {
	// SSH remote should normalise to match the config value
	got := ResolveProject(testProjects, "github.com/example/project-beta", "")
	if got != "project-beta" {
		t.Errorf("want project-beta, got %q", got)
	}
}

func TestResolveProject_ByWorkingDir(t *testing.T) {
	// No git remote, but working dir matches
	got := ResolveProject(testProjects, "", "/home/user/work/project-alpha/subdir")
	if got != "project-alpha" {
		t.Errorf("want project-alpha, got %q", got)
	}
}

func TestResolveProject_GitRemoteTakesPrecedence(t *testing.T) {
	// Remote matches beta, working dir would match alpha — remote wins
	got := ResolveProject(testProjects, "github.com/example/project-beta", "/home/user/work/project-alpha")
	if got != "project-beta" {
		t.Errorf("want project-beta (remote wins), got %q", got)
	}
}

func TestResolveProject_FallbackToDirName(t *testing.T) {
	got := ResolveProject(testProjects, "", "/home/user/work/unknown-project")
	if got != "unknown-project" {
		t.Errorf("want unknown-project (fallback), got %q", got)
	}
}

func TestResolveProject_EmptyInputs(t *testing.T) {
	got := ResolveProject(testProjects, "", "")
	if got != "" {
		t.Errorf("want empty string, got %q", got)
	}
}

func TestResolveProject_NoProjectsConfigured(t *testing.T) {
	got := ResolveProject(nil, "github.com/example/anything", "/home/user/work/myproject")
	if got != "myproject" {
		t.Errorf("want myproject (fallback), got %q", got)
	}
}

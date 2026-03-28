package collector

import (
	"path/filepath"
	"strings"

	"github.com/GrayFlash/kirkup-cli/config"
)

// ResolveProject returns the project name for a given git remote and working
// directory by matching against the configured project list.
//
// Resolution order:
//  1. Git remote match (exact, normalised)
//  2. Working directory prefix match
//  3. Fallback: base name of working directory
func ResolveProject(projects []config.ProjectConfig, gitRemote, workingDir string) string {
	// 1. Match by git remote
	if gitRemote != "" {
		for _, p := range projects {
			if normaliseRemote(p.Match.GitRemote) == gitRemote {
				return p.Name
			}
		}
	}

	// 2. Match by working directory prefix
	if workingDir != "" {
		for _, p := range projects {
			for _, path := range p.Match.Paths {
				expanded := config.ExpandHome(path)
				if strings.HasPrefix(workingDir, expanded) {
					return p.Name
				}
			}
		}
	}

	// 3. Fallback: directory base name
	if workingDir != "" {
		return filepath.Base(workingDir)
	}
	return ""
}


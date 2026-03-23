package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/GrayFlash/kirkup-cli/store"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List projects with activity stats",
	RunE:  runProjects,
}

func init() {
	rootCmd.AddCommand(projectsCmd)
}

func runProjects(_ *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	s, err := openStore(cfg)
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()

	ctx := context.Background()

	projects, err := s.ListProjects(ctx)
	if err != nil {
		return err
	}

	// Also discover projects from event history that aren't in the registry.
	events, err := s.QueryPromptEvents(ctx, store.EventFilter{})
	if err != nil {
		return err
	}
	seen := make(map[string]struct{})
	for _, p := range projects {
		seen[p.Name] = struct{}{}
	}
	for _, e := range events {
		if e.Project != "" {
			seen[e.Project] = struct{}{}
		}
	}

	if len(seen) == 0 {
		fmt.Println("no projects found")
		return nil
	}

	// Count prompts and last-seen per project.
	type stat struct {
		prompts  int
		lastSeen time.Time
	}
	stats := make(map[string]*stat)
	for _, e := range events {
		if e.Project == "" {
			continue
		}
		if stats[e.Project] == nil {
			stats[e.Project] = &stat{}
		}
		stats[e.Project].prompts++
		if e.Timestamp.After(stats[e.Project].lastSeen) {
			stats[e.Project].lastSeen = e.Timestamp
		}
	}

	fmt.Printf("%-30s  %7s  %s\n", "Project", "Prompts", "Last Active")
	fmt.Printf("%-30s  %7s  %s\n", "───────────────────────────", "───────", "───────────")
	for name := range seen {
		st := stats[name]
		prompts := 0
		lastSeen := "-"
		if st != nil {
			prompts = st.prompts
			lastSeen = st.lastSeen.Local().Format("2006-01-02 15:04")
		}
		fmt.Printf("%-30s  %7d  %s\n", name, prompts, lastSeen)
	}
	return nil
}

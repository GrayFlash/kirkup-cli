package cmd

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"

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
	_, s, cleanup, err := openApp()
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := context.Background()

	projects, err := s.ListProjects(ctx)
	if err != nil {
		return err
	}

	pStats, err := s.ProjectStats(ctx)
	if err != nil {
		return err
	}

	seen := make(map[string]struct{})
	for _, p := range projects {
		seen[p.Name] = struct{}{}
	}
	for _, st := range pStats {
		seen[st.Name] = struct{}{}
	}

	if len(seen) == 0 {
		fmt.Println("no projects found")
		return nil
	}

	type stat struct {
		prompts  int
		lastSeen time.Time
	}
	stats := make(map[string]*stat)
	for _, st := range pStats {
		stats[st.Name] = &stat{
			prompts:  st.Prompts,
			lastSeen: st.LastSeen,
		}
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		si, sj := stats[names[i]], stats[names[j]]
		pi, pj := 0, 0
		if si != nil {
			pi = si.prompts
		}
		if sj != nil {
			pj = sj.prompts
		}
		if pi != pj {
			return pi > pj
		}
		return names[i] < names[j]
	})

	fmt.Printf("%-30s  %7s  %s\n", "Project", "Prompts", "Last Active")
	fmt.Printf("%-30s  %7s  %s\n", "───────────────────────────", "───────", "───────────")
	for _, name := range names {
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

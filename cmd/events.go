package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/GrayFlash/kirkup-cli/store"
)

var (
	eventsToday   bool
	eventsTail    bool
	eventsProject string
	eventsLimit   int
)

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "List collected prompt events",
	RunE:  runEvents,
}

func init() {
	eventsCmd.Flags().BoolVar(&eventsToday, "today", false, "Show events from today only")
	eventsCmd.Flags().BoolVar(&eventsTail, "tail", false, "Stream new events as they are collected")
	eventsCmd.Flags().StringVar(&eventsProject, "project", "", "Filter by project name")
	eventsCmd.Flags().IntVar(&eventsLimit, "limit", 50, "Maximum number of events to show (0 = unlimited)")
	rootCmd.AddCommand(eventsCmd)
}

func runEvents(_ *cobra.Command, _ []string) error {
	_, s, cleanup, err := openApp()
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := context.Background()

	if eventsTail {
		return tailEvents(ctx, s)
	}

	f := store.EventFilter{
		Project: eventsProject,
		Limit:   eventsLimit,
	}
	if eventsToday {
		t := today()
		f.Since = &t
	}

	events, err := s.QueryPromptEvents(ctx, f)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		fmt.Println("no events found")
		return nil
	}

	for _, e := range events {
		printEvent(e.Timestamp, e.Agent, e.Project, e.Prompt)
	}
	return nil
}

// tailEvents polls the store every few seconds and prints new events.
func tailEvents(ctx context.Context, s store.Store) error {
	fmt.Println("tailing events (ctrl+c to stop)...")

	last := time.Now()
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case t := <-ticker.C:
			events, err := s.QueryPromptEvents(ctx, store.EventFilter{Since: &last})
			if err != nil {
				return err
			}
			// Events are returned newest-first; print oldest-first.
			for i := len(events) - 1; i >= 0; i-- {
				printEvent(events[i].Timestamp, events[i].Agent, events[i].Project, events[i].Prompt)
			}
			last = t
		}
	}
}

func printEvent(ts time.Time, agent, project, prompt string) {
	proj := project
	if proj == "" {
		proj = "-"
	}
	truncated := truncateStr(prompt, 80)
	fmt.Printf("%s  %-12s  %-16s  %s\n",
		ts.Local().Format("2006-01-02 15:04:05"),
		agent, proj, truncated,
	)
}
